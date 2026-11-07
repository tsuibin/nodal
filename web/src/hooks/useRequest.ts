/**
 * 通用请求状态 hook：把 loading / error / 防竞态 收在一处，
 * 页面里不用为每个接口手写一套 useState。
 *
 * const { data, loading, error, run } = useRequest(authApi.me, { immediate: false })
 * await run()
 */
import { useCallback, useEffect, useRef, useState } from 'react'
import { ApiError } from '../api/http'

export interface UseRequestOptions {
  /** 挂载后立即执行，默认 false */
  immediate?: boolean
}

export interface UseRequestResult<TArgs extends unknown[], TData> {
  data: TData | null
  error: ApiError | Error | null
  loading: boolean
  /** 手动触发；返回 null 表示被更新的请求取代，不再更新 state */
  run: (...args: TArgs) => Promise<TData | null>
  reset: () => void
}

export function useRequest<TArgs extends unknown[], TData>(
  fn: (...args: TArgs) => Promise<TData>,
  options: UseRequestOptions = {},
): UseRequestResult<TArgs, TData> {
  const [data, setData] = useState<TData | null>(null)
  const [error, setError] = useState<ApiError | Error | null>(null)
  const [loading, setLoading] = useState(false)

  // 只保留最后一次请求的结果，避免快速切换页面时旧响应覆盖新状态
  const seqRef = useRef(0)
  const mountedRef = useRef(true)
  const fnRef = useRef(fn)
  fnRef.current = fn

  useEffect(() => {
    mountedRef.current = true
    return () => {
      mountedRef.current = false
    }
  }, [])

  const run = useCallback(async (...args: TArgs): Promise<TData | null> => {
    const seq = ++seqRef.current
    setLoading(true)
    setError(null)
    try {
      const result = await fnRef.current(...args)
      if (mountedRef.current && seq === seqRef.current) {
        setData(result)
      }
      return result
    } catch (err) {
      if (mountedRef.current && seq === seqRef.current) {
        setError(err instanceof Error ? err : new Error(String(err)))
      }
      return null
    } finally {
      if (mountedRef.current && seq === seqRef.current) {
        setLoading(false)
      }
    }
  }, [])

  const reset = useCallback(() => {
    seqRef.current += 1
    setData(null)
    setError(null)
    setLoading(false)
  }, [])

  useEffect(() => {
    if (options.immediate) {
      void run(...([] as unknown as TArgs))
    }
    // 只在挂载时决定是否立即请求
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return { data, error, loading, run, reset }
}
