/**
 * useRepl：代码节点「运行」按钮的完整生命周期。
 *
 *   const repl = useRepl()
 *   <button onClick={() => repl.run(code, 'python')}>运行</button>
 *   <pre>{repl.output}</pre>
 *
 * - 创建会话时把代码交给后端，后端在 PTY 就绪后自动写入，输出不会丢；
 * - 输出按 WS output 帧追加，超过 MAX_OUTPUT 截断头部，避免前端内存爆掉；
 * - 30s 后端硬超时，同时前端显示倒计时；exit 帧带回退出码与原因。
 */
import { useCallback, useEffect, useRef, useState } from 'react'
import { replApi, openReplSocket } from '../api/repl'
import { ApiError } from '../api/http'
import type { ReplLanguage, ReplServerMessage, ReplSession } from '../types/repl'

const MAX_OUTPUT = 200 * 1024

export type ReplStatus = 'idle' | 'creating' | 'connecting' | 'running' | 'closed' | 'error'

export interface UseReplOptions {
  onOutput?: (chunk: string) => void
}

export interface UseReplResult {
  status: ReplStatus
  output: string
  error: string | null
  exitCode: number | null
  remaining: number
  session: ReplSession | null
  run: (code: string, language?: ReplLanguage) => Promise<void>
  send: (data: string) => void
  resize: (cols: number, rows: number) => void
  stop: () => void
  reset: () => void
}

export function useRepl(options: UseReplOptions = {}): UseReplResult {
  const [status, setStatus] = useState<ReplStatus>('idle')
  const [output, setOutput] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [exitCode, setExitCode] = useState<number | null>(null)
  const [remaining, setRemaining] = useState(0)
  const [session, setSession] = useState<ReplSession | null>(null)

  const socketRef = useRef<WebSocket | null>(null)
  const timerRef = useRef<number | null>(null)
  const optionsRef = useRef(options)
  optionsRef.current = options

  const clearTimer = useCallback(() => {
    if (timerRef.current !== null) {
      window.clearInterval(timerRef.current)
      timerRef.current = null
    }
  }, [])

  const closeSocket = useCallback(() => {
    const socket = socketRef.current
    socketRef.current = null
    if (socket && socket.readyState <= WebSocket.OPEN) {
      socket.close()
    }
  }, [])

  const reset = useCallback(() => {
    closeSocket()
    clearTimer()
    setStatus('idle')
    setOutput('')
    setError(null)
    setExitCode(null)
    setRemaining(0)
    setSession(null)
  }, [closeSocket, clearTimer])

  const stop = useCallback(() => {
    const socket = socketRef.current
    if (socket && socket.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify({ type: 'close' }))
    }
    closeSocket()
    clearTimer()
    setStatus('closed')
  }, [closeSocket, clearTimer])

  const append = useCallback((chunk: string) => {
    optionsRef.current.onOutput?.(chunk)
    setOutput((prev) => {
      const next = prev + chunk
      return next.length > MAX_OUTPUT ? next.slice(next.length - MAX_OUTPUT) : next
    })
  }, [])

  const run = useCallback(
    async (code: string, language: ReplLanguage = 'python') => {
      reset()
      setStatus('creating')

      let created: ReplSession
      try {
        created = await replApi.create({ language, code })
      } catch (err) {
        setStatus('error')
        setError(err instanceof ApiError ? err.message : err instanceof Error ? err.message : '创建 REPL 失败')
        return
      }

      setSession(created)
      setRemaining(Math.max(0, Math.round((new Date(created.expires_at).getTime() - Date.now()) / 1000)))
      setStatus('connecting')

      let socket: WebSocket
      try {
        socket = openReplSocket(created.session_id)
      } catch (err) {
        setStatus('error')
        setError(err instanceof Error ? err.message : '连接 REPL 失败')
        return
      }
      socketRef.current = socket

      timerRef.current = window.setInterval(() => {
        const left = Math.max(0, Math.round((new Date(created.expires_at).getTime() - Date.now()) / 1000))
        setRemaining(left)
        if (left <= 0) clearTimer()
      }, 500)

      socket.onmessage = (event) => {
        let msg: ReplServerMessage
        try {
          msg = JSON.parse(String(event.data)) as ReplServerMessage
        } catch {
          return
        }
        switch (msg.type) {
          case 'ready':
            setStatus('running')
            break
          case 'output':
            if (msg.data) append(msg.data)
            break
          case 'exit':
            setStatus('closed')
            setExitCode(msg.code ?? null)
            clearTimer()
            setRemaining(0)
            break
          case 'error':
            setStatus('error')
            setError(msg.message ?? 'REPL 执行失败')
            break
        }
      }

      socket.onerror = () => {
        setStatus((prev) => (prev === 'closed' ? prev : 'error'))
        setError((prev) => prev ?? '终端连接异常')
      }

      socket.onclose = () => {
        clearTimer()
        socketRef.current = null
        setStatus((prev) => (prev === 'closed' || prev === 'error' ? prev : 'closed'))
      }
    },
    [append, clearTimer, reset],
  )

  const send = useCallback((data: string) => {
    const socket = socketRef.current
    if (socket && socket.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify({ type: 'input', data }))
    }
  }, [])

  const resize = useCallback((cols: number, rows: number) => {
    const socket = socketRef.current
    if (socket && socket.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify({ type: 'resize', cols, rows }))
    }
  }, [])

  useEffect(() => {
    return () => {
      closeSocket()
      clearTimer()
    }
  }, [closeSocket, clearTimer])

  return { status, output, error, exitCode, remaining, session, run, send, resize, stop, reset }
}
