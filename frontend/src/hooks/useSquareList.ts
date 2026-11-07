/**
 * useSquareList：广场列表的分页/标签筛选状态。
 * 换筛选条件会重置到第 1 页；loadMore 追加下一页。
 */
import { useCallback, useEffect, useRef, useState } from 'react'
import { ApiError } from '../services/http'
import { squareApi, type SquareListQuery } from '../services/square'
import type { SquareItem } from '../types/square'

export interface UseSquareListOptions extends SquareListQuery {
  immediate?: boolean
}

export interface UseSquareListResult {
  items: SquareItem[]
  total: number
  page: number
  hasMore: boolean
  loading: boolean
  error: ApiError | Error | null
  refresh: () => Promise<void>
  loadMore: () => Promise<void>
  setFilters: (next: SquareListQuery) => void
}

export function useSquareList(options: UseSquareListOptions = {}): UseSquareListResult {
  const { immediate = true, ...initial } = options
  const [filters, setFilters] = useState<SquareListQuery>({
    tag: initial.tag,
    keyword: initial.keyword,
    sort: initial.sort ?? 'latest',
    page_size: initial.page_size ?? 20,
  })
  const [items, setItems] = useState<SquareItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<ApiError | Error | null>(null)

  const mountedRef = useRef(true)
  const seqRef = useRef(0)
  useEffect(() => {
    mountedRef.current = true
    return () => {
      mountedRef.current = false
    }
  }, [])

  const load = useCallback(
    async (targetPage: number, append: boolean) => {
      const seq = ++seqRef.current
      setLoading(true)
      setError(null)
      try {
        const data = await squareApi.list({ ...filters, page: targetPage })
        if (!mountedRef.current || seq !== seqRef.current) return
        setItems((prev) => (append ? [...prev, ...data.items] : data.items))
        setTotal(data.total)
        setPage(data.page)
      } catch (err) {
        if (mountedRef.current && seq === seqRef.current) {
          setError(err instanceof Error ? err : new Error(String(err)))
        }
      } finally {
        if (mountedRef.current && seq === seqRef.current) setLoading(false)
      }
    },
    [filters],
  )

  const refresh = useCallback(() => load(1, false), [load])
  const loadMore = useCallback(() => load(page + 1, true), [load, page])

  const updateFilters = useCallback((next: SquareListQuery) => {
    setFilters((prev) => ({ ...prev, ...next }))
    setPage(1)
    setItems([])
  }, [])

  useEffect(() => {
    if (immediate) void refresh()
  }, [immediate, refresh])

  return { items, total, page, hasMore: items.length < total, loading, error, refresh, loadMore, setFilters: updateFilters }
}
