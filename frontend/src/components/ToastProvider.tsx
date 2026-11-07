/**
 * 模块 10：全局 toast。
 *
 * - useToast() 供页面主动提示成功/失败；
 * - 监听 toast 总线，接收 service/http 抛出的系统级错误；
 * - 监听 unhandledrejection / window.error，兜住未捕获异常；
 * - 最多同时显示 3 条，2 秒内相同文案去重。
 */
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import { createPortal } from 'react-dom'
import { TOAST_EVENT, notifyToast, type ToastDetail, type ToastKind } from '../lib/notify'

interface ToastItem extends ToastDetail {
  id: number
}

interface ToastContextValue {
  show: (detail: ToastDetail) => void
  success: (message: string) => void
  error: (message: string) => void
  info: (message: string) => void
}

const ToastContext = createContext<ToastContextValue | null>(null)

const DEFAULT_DURATION = 3200
const MAX_VISIBLE = 3

export function ToastProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<ToastItem[]>([])
  const seqRef = useRef(0)
  const recentRef = useRef<Map<string, number>>(new Map())

  const dismiss = useCallback((id: number) => {
    setItems((prev) => prev.filter((item) => item.id !== id))
  }, [])

  const show = useCallback(
    (detail: ToastDetail) => {
      const message = detail.message.trim()
      if (!message) return

      const now = Date.now()
      const last = recentRef.current.get(message) ?? 0
      if (now - last < 2000) return
      recentRef.current.set(message, now)

      const id = ++seqRef.current
      setItems((prev) => [...prev, { ...detail, message, id }].slice(-MAX_VISIBLE))

      const duration = detail.duration ?? DEFAULT_DURATION
      window.setTimeout(() => dismiss(id), Math.max(1200, duration))
    },
    [dismiss],
  )

  useEffect(() => {
    const onToast = (event: Event) => {
      const custom = event as CustomEvent<ToastDetail>
      if (custom.detail) show(custom.detail)
    }
    const onRejection = (event: PromiseRejectionEvent) => {
      const reason = event.reason
      if (reason instanceof Error) show({ kind: 'error', message: reason.message || '操作失败' })
      else show({ kind: 'error', message: String(reason || '未知错误') })
    }
    const onError = (event: ErrorEvent) => {
      show({ kind: 'error', message: event.message || '页面发生异常' })
    }

    window.addEventListener(TOAST_EVENT, onToast)
    window.addEventListener('unhandledrejection', onRejection)
    window.addEventListener('error', onError)
    return () => {
      window.removeEventListener(TOAST_EVENT, onToast)
      window.removeEventListener('unhandledrejection', onRejection)
      window.removeEventListener('error', onError)
    }
  }, [show])

  const value = useMemo<ToastContextValue>(
    () => ({
      show,
      success: (message) => show({ kind: 'success', message }),
      error: (message) => show({ kind: 'error', message }),
      info: (message) => show({ kind: 'info', message }),
    }),
    [show],
  )

  return (
    <ToastContext.Provider value={value}>
      {children}
      {createPortal(
        <div className="nodal-toast-viewport" aria-live="polite" aria-atomic="false">
          {items.map((item) => (
            <div key={item.id} className={`nodal-toast nodal-toast-${item.kind}`} role="status">
              <span className="nodal-toast-mark" aria-hidden="true">
                {item.kind === 'success' ? '✓' : item.kind === 'error' ? '!' : 'i'}
              </span>
              <span className="nodal-toast-message">{item.message}</span>
              <button
                type="button"
                className="nodal-toast-close"
                onClick={() => dismiss(item.id)}
                aria-label="关闭提示"
              >
                ×
              </button>
            </div>
          ))}
        </div>,
        document.body,
      )}
    </ToastContext.Provider>
  )
}

export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext)
  if (!ctx) throw new Error('useToast 必须在 ToastProvider 内使用')
  return ctx
}

export { notifyToast, type ToastKind }
