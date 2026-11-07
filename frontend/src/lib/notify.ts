/**
 * 模块 10：跨层通知总线。
 *
 * service/http 不需要认识 React，只把错误/提示丢到 window 事件；
 * ToastProvider 监听后统一渲染，避免每个请求各自弹一次。
 */

export type ToastKind = 'success' | 'error' | 'info'

export interface ToastDetail {
  kind: ToastKind
  message: string
  duration?: number
}

/** 组件、service、全局异常统一使用的事件名 */
export const TOAST_EVENT = 'nodal:toast'

export function notifyToast(detail: ToastDetail): void {
  if (typeof window === 'undefined') return
  window.dispatchEvent(new CustomEvent<ToastDetail>(TOAST_EVENT, { detail }))
}

interface ApiErrorLike {
  code: number
  message: string
  status?: number
}

/** 401 类交给登录弹窗，参数校验交给表单内联；只把系统级故障提升为 toast。 */
export function notifyApiError(error: ApiErrorLike): void {
  if (error.code === 1000 || error.code === 1001 || error.code === 1002 || error.code === 1003) return
  if (error.message === '请求已取消') return
  if (error.code !== 5000 && error.code !== -1 && (error.status ?? 0) < 500) return
  notifyToast({ kind: 'error', message: error.message || '服务异常，请稍后重试' })
}
