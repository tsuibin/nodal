/** 模块 7：REPL 沙箱的前端类型。 */

export type ReplLanguage = 'python' | 'shell'

export interface ReplCreatePayload {
  language: ReplLanguage
  /** 创建后自动送入终端的代码；不传则进入交互模式 */
  code?: string
}

export interface ReplSession {
  session_id: string
  language: ReplLanguage
  ws_path: string
  timeout_seconds: number
  memory_mb: number
  expires_at: string
  runtime: string
}

/** 服务端 WebSocket 帧：ready / output / exit / error。 */
export interface ReplServerMessage {
  type: 'ready' | 'output' | 'exit' | 'error'
  session_id?: string
  language?: ReplLanguage
  timeout_seconds?: number
  expires_at?: string
  data?: string
  code?: number
  reason?: string
  error_code?: number
  message?: string
}

/** 客户端 WebSocket 帧。 */
export interface ReplClientMessage {
  type: 'input' | 'resize' | 'close'
  data?: string
  cols?: number
  rows?: number
}
