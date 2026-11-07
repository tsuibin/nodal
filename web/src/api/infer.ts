/**
 * 推演接口的流式封装。
 *
 * 为什么不用 EventSource：
 *   EventSource 只能发 GET，无法带 Authorization 头、也无法发 JSON body；
 *   所以这里用 fetch + ReadableStream 自己解析 SSE 帧。
 *
 * 事件协议（与后端 handler/sse.go 对齐）：
 *   start   {run_id, canvas_id, canvas_title, agent_id, agent_name, model_id, model_label}
 *   sources {query, items[], warning}   ← 仅当画布挂载了知识库时出现
 *   node    {node, index}
 *   edge    {edge, index}
 *   warning {message, skipped}
 *   done    {run_id, node_count, edge_count, skipped}
 *   error   {code, message}
 *
 * 关键约定：流里只会出现「新增」的 node/edge，已有节点不会被修改或删除；
 * 前端收到事件后直接 append 到画布即可，最后是否保存由用户决定。
 */
import { API_BASE_URL, ApiError, tokenStore } from './http'
import type {
  InferDonePayload,
  InferEdgeEvent,
  InferNodeEvent,
  InferSourcesEvent,
  InferStartEvent,
  InferStreamHandlers,
  InferWarningEvent,
  RunInferPayload,
} from '../types/infer'

interface ParsedFrame {
  event: string
  data: string
}

interface ErrorEnvelope {
  code?: number
  message?: string
  details?: unknown
  request_id?: string
}

function parseFrame(frame: string): ParsedFrame | null {
  let event = 'message'
  const dataLines: string[] = []

  for (const rawLine of frame.split('\n')) {
    if (rawLine.startsWith(':')) continue // 心跳注释
    const sep = rawLine.indexOf(':')
    const field = sep >= 0 ? rawLine.slice(0, sep) : rawLine
    const value = sep >= 0 ? rawLine.slice(sep + 1).replace(/^ /, '') : ''

    if (field === 'event') event = value
    else if (field === 'data') dataLines.push(value)
  }

  if (dataLines.length === 0) return null
  return { event, data: dataLines.join('\n') }
}

/**
 * 发起推演并按事件回调增量数据；resolve 时返回 done 汇总。
 * 调用方可用 AbortSignal 取消（例如用户关闭面板/离开页面）。
 */
export async function streamRunInfer(
  canvasId: number,
  payload: RunInferPayload,
  handlers: InferStreamHandlers = {},
  signal?: AbortSignal,
): Promise<InferDonePayload> {
  const { token } = tokenStore.getActive()

  const resp = await fetch(`${API_BASE_URL}/canvas/${canvasId}/run-infer`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'text/event-stream',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify(payload),
    signal,
  })

  const contentType = resp.headers.get('content-type') ?? ''

  // 鉴权/参数/缺 Key 等错误在开流之前返回，后端走的是普通 JSON 出口
  if (!resp.ok || contentType.includes('application/json')) {
    let body: ErrorEnvelope | null = null
    try {
      body = (await resp.json()) as ErrorEnvelope
    } catch {
      body = null
    }
    throw new ApiError({
      code: body?.code ?? 5000,
      message: body?.message ?? `推演请求失败（HTTP ${resp.status}）`,
      status: resp.status,
      details: body?.details,
      requestId: body?.request_id,
    })
  }

  if (!resp.body) {
    throw new ApiError({ code: 5000, message: '当前浏览器不支持流式响应' })
  }

  const reader = resp.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let donePayload: InferDonePayload | null = null

  const handleFrame = (frame: string): void => {
    const parsed = parseFrame(frame)
    if (!parsed) return

    let data: unknown
    try {
      data = JSON.parse(parsed.data)
    } catch {
      return // 坏帧直接跳过，不打断整条流
    }

    switch (parsed.event) {
      case 'start':
        handlers.onStart?.(data as InferStartEvent)
        break
      case 'sources':
        handlers.onSources?.(data as InferSourcesEvent)
        break
      case 'node':
        handlers.onNode?.(data as InferNodeEvent)
        break
      case 'edge':
        handlers.onEdge?.(data as InferEdgeEvent)
        break
      case 'warning':
        handlers.onWarning?.(data as InferWarningEvent)
        break
      case 'done':
        donePayload = data as InferDonePayload
        break
      case 'error': {
        const err = data as { code?: number; message?: string }
        throw new ApiError({
          code: err.code ?? 5000,
          message: err.message || '推演失败',
          status: resp.status,
        })
      }
      default:
        break
    }
  }

  const drainFrames = (): void => {
    buffer = buffer.replace(/\r\n/g, '\n')
    let sep = buffer.indexOf('\n\n')
    while (sep >= 0) {
      const frame = buffer.slice(0, sep)
      buffer = buffer.slice(sep + 2)
      handleFrame(frame)
      sep = buffer.indexOf('\n\n')
    }
  }

  try {
    while (!donePayload) {
      const { value, done } = await reader.read()
      if (done) break
      buffer += decoder.decode(value, { stream: true })
      drainFrames()
    }

    // 流结束前可能还有没被空行收尾的最后一帧
    buffer += decoder.decode()
    if (!donePayload && buffer.trim()) {
      handleFrame(buffer)
      buffer = ''
    }
  } finally {
    await reader.cancel().catch(() => undefined)
  }

  if (!donePayload) {
    throw new ApiError({ code: 5000, message: '推演流意外结束（未收到 done 事件）' })
  }
  return donePayload
}
