/**
 * 统一 HTTP 客户端。
 *
 * 约定（与后端 response.Envelope 对齐）：
 *   { code: 0, message: "ok", data, details, request_id }
 *
 * - code === 0：resolve，直接拿到 data；
 * - code !== 0：reject 一个 ApiError，页面按 code 分支，不解析文案；
 * - 1001/1002（未登录/过期）：自动清掉用户 token 并广播事件，让 AuthProvider 回到匿名态。
 *
 * 游客：后端在 X-Guest-Token 响应头下发临时身份，这里自动落库并在后续请求中带上，
 * 保证刷新页面后仍是同一个临时画布。
 */
import axios, { type AxiosError, type AxiosRequestConfig } from 'axios'

export interface ApiEnvelope<T> {
  code: number
  message: string
  data?: T
  details?: unknown
  request_id?: string
}

export interface FieldErrors {
  [field: string]: string
}

export class ApiError extends Error {
  readonly code: number
  readonly status?: number
  readonly details?: unknown
  readonly requestId?: string
  readonly cause?: unknown

  constructor(params: {
    code: number
    message: string
    status?: number
    details?: unknown
    requestId?: string
    cause?: unknown
  }) {
    super(params.message)
    this.name = 'ApiError'
    this.code = params.code
    this.status = params.status
    this.details = params.details
    this.requestId = params.requestId
    this.cause = params.cause
  }

  /** 未登录 / 登录过期 */
  get isUnauthorized(): boolean {
    return this.code === 1001 || this.code === 1002
  }

  /** 需要登录才能用的功能（游客被拒） */
  get needsLogin(): boolean {
    return this.code === 1003
  }

  /** 参数校验错误，details 是 { 字段: 文案 } */
  get fieldErrors(): FieldErrors | undefined {
    if (this.code !== 1000 || !this.details || typeof this.details !== 'object') return undefined
    return this.details as FieldErrors
  }
}

export const AUTH_CODES = {
  ok: 0,
  invalidParam: 1000,
  unauthorized: 1001,
  tokenExpired: 1002,
  forbidden: 1003,
  notFound: 1004,
  emailTaken: 2001,
  userNotFound: 2002,
  badCredentials: 2003,
  accountDisabled: 2004,
  guestDisabled: 2005,
  guestExpired: 2006,
  // 模块 4：Agent / 模型
  agentNotFound: 3001,
  agentNameTaken: 3002,
  modelNotSupported: 3003,
  modelKeyMissing: 3004,
  modelCallFailed: 3005,
  emptyCompletion: 3006,
  inferRunning: 3007,
  // 模块 5：知识库 / RAG
  knowledgeNotFound: 4001,
  unsupportedFile: 4002,
  fileParseFailed: 4003,
  knowledgeNotMounted: 4004,
  embeddingFailed: 4005,
  // 模块 6：广场 / 分享 / 互动
  shareInvalid: 6001,
  shareExpired: 6002,
  shareForbidden: 6003,
  canvasNotPublished: 6004,
  commentNotFound: 6005,
  // 模块 7：REPL 沙箱
  replUnavailable: 7001,
  replNotFound: 7002,
  replTooMany: 7003,
  replAttached: 7004,
  replLanguage: 7005,
  replCodeTooLarge: 7006,
  internal: 5000,
  network: -1,
} as const

// ---------- token 存储 ----------

const USER_TOKEN_KEY = 'nodal.auth.token'
const GUEST_TOKEN_KEY = 'nodal.guest.token'
const UNAUTHORIZED_EVENT = 'nodal:auth:unauthorized'

type TokenKind = 'user' | 'guest' | 'none'
type TokenListener = (token: string | null, kind: TokenKind) => void

const listeners = new Set<TokenListener>()

function safeGet(key: string): string | null {
  try {
    return window.localStorage.getItem(key)
  } catch {
    return null // 隐私模式 / SSR
  }
}

function safeSet(key: string, value: string | null): void {
  try {
    if (value) window.localStorage.setItem(key, value)
    else window.localStorage.removeItem(key)
  } catch {
    /* ignore */
  }
}

function emit(token: string | null, kind: TokenKind): void {
  listeners.forEach((fn) => fn(token, kind))
}

export const tokenStore = {
  getUser(): string | null {
    return safeGet(USER_TOKEN_KEY)
  },

  getGuest(): string | null {
    return safeGet(GUEST_TOKEN_KEY)
  },

  /** 用户 token 优先；没有登录时才用游客 token */
  getActive(): { token: string | null; kind: TokenKind } {
    const user = safeGet(USER_TOKEN_KEY)
    if (user) return { token: user, kind: 'user' }
    const guest = safeGet(GUEST_TOKEN_KEY)
    if (guest) return { token: guest, kind: 'guest' }
    return { token: null, kind: 'none' }
  },

  setUser(token: string | null): void {
    safeSet(USER_TOKEN_KEY, token)
    if (token) {
      // 登录后游客身份作废，避免两个 token 同时存在导致归属混乱
      safeSet(GUEST_TOKEN_KEY, null)
    }
    emit(token, token ? 'user' : 'none')
  },

  setGuest(token: string | null): void {
    safeSet(GUEST_TOKEN_KEY, token)
    if (token) emit(token, 'guest')
  },

  clear(): void {
    safeSet(USER_TOKEN_KEY, null)
    safeSet(GUEST_TOKEN_KEY, null)
    emit(null, 'none')
  },

  /** 只清用户 token（登录失效，但保留当前游客会话） */
  clearUser(): void {
    safeSet(USER_TOKEN_KEY, null)
    emit(null, 'none')
  },
}

/** 订阅 token 变化（多标签页同步、登录态失效等） */
export function onTokenChange(fn: TokenListener): () => void {
  listeners.add(fn)
  return () => listeners.delete(fn)
}

/** 订阅「登录失效」事件 */
export function onUnauthorized(fn: () => void): () => void {
  const handler = () => fn()
  window.addEventListener(UNAUTHORIZED_EVENT, handler)
  return () => window.removeEventListener(UNAUTHORIZED_EVENT, handler)
}

function broadcastUnauthorized(): void {
  tokenStore.clearUser()
  window.dispatchEvent(new Event(UNAUTHORIZED_EVENT))
}

// ---------- axios 实例 ----------

// import.meta.env 在 Vite 内一定存在；加 ?. 是为了让同一份代码也能在 Node/测试环境跑。
export const API_BASE_URL = import.meta.env?.VITE_API_BASE_URL ?? '/api'

export const http = axios.create({
  baseURL: API_BASE_URL,
  timeout: 15_000,
  headers: { 'Content-Type': 'application/json' },
})

http.interceptors.request.use((config) => {
  const { token } = tokenStore.getActive()
  if (token) {
    config.headers = config.headers ?? {}
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

http.interceptors.response.use(
  (res) => {
    // 保存后端即时下发的游客 token
    const guestToken = res.headers['x-guest-token']
    if (typeof guestToken === 'string' && guestToken.length > 0) {
      tokenStore.setGuest(guestToken)
    }

    const body = res.data as ApiEnvelope<unknown> | undefined
    if (body && typeof body.code === 'number' && body.code !== AUTH_CODES.ok) {
      const error = new ApiError({
        code: body.code,
        message: body.message || '请求失败',
        status: res.status,
        details: body.details,
        requestId: body.request_id,
      })
      if (error.isUnauthorized) broadcastUnauthorized()
      throw error
    }
    return res
  },
  (error: AxiosError<ApiEnvelope<unknown>>) => {
    if (axios.isCancel(error)) {
      return Promise.reject(
        new ApiError({ code: AUTH_CODES.network, message: '请求已取消', cause: error }),
      )
    }

    const response = error.response
    const body = response?.data

    if (body && typeof body.code === 'number') {
      const apiError = new ApiError({
        code: body.code,
        message: body.message || '请求失败',
        status: response?.status,
        details: body.details,
        requestId: body.request_id,
        cause: error,
      })
      if (apiError.isUnauthorized) broadcastUnauthorized()
      return Promise.reject(apiError)
    }

    const message =
      error.code === 'ECONNABORTED'
        ? '请求超时，请稍后重试'
        : response
          ? `服务异常（HTTP ${response.status}）`
          : '网络异常，请检查网络连接'

    return Promise.reject(
      new ApiError({
        code: AUTH_CODES.network,
        message,
        status: response?.status,
        cause: error,
      }),
    )
  },
)

/** 发起请求并直接返回 data 字段 */
export async function request<T>(config: AxiosRequestConfig): Promise<T> {
  const res = await http.request<ApiEnvelope<T>>(config)
  const body = res.data as ApiEnvelope<T>
  return body.data as T
}
