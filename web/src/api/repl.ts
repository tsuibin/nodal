import { API_BASE_URL, http, request, tokenStore } from './http'
import type { ReplCreatePayload, ReplSession } from '../types/repl'

export const replApi = {
  /** 创建隔离会话；会话从创建时刻开始计时。 */
  create(payload: ReplCreatePayload): Promise<ReplSession> {
    return request<ReplSession>({
      method: 'POST',
      url: '/repl/create',
      data: payload,
    })
  },
}

/**
 * 浏览器 WebSocket 不能带 Authorization 头，因此 token 走查询参数。
 * 后端访问日志只记录 Path，不会落 token；生产建议再套一层同源反向代理。
 */
export function buildReplWebSocketUrl(sessionId: string, token?: string | null): string {
  const rawToken = token ?? tokenStore.getActive().token
  if (!rawToken) throw new Error('需要登录后才能运行代码')

  const base = API_BASE_URL.startsWith('http')
    ? API_BASE_URL
    : new URL(API_BASE_URL, window.location.href).toString()

  const url = new URL(`${base.replace(/\/$/, '')}/repl/ws/${encodeURIComponent(sessionId)}`)
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
  url.searchParams.set('token', rawToken)
  return url.toString()
}

export function openReplSocket(sessionId: string): WebSocket {
  const { token, kind } = tokenStore.getActive()
  if (kind !== 'user' || !token) throw new Error('REPL 仅对注册用户开放，请先登录')
  return new WebSocket(buildReplWebSocketUrl(sessionId, token))
}

export { http }
