/**
 * 我的画布列表：分页 + 关键词筛选 + 创建。
 *
 * 仅用于资源库「我的画布」；广场列表由 useSquareList 负责。
 */
import { useCallback, useEffect, useRef, useState } from 'react'
import { ApiError } from '../services/http'
import { canvasApi } from '../services/canvas'
import type { CanvasListQuery, CanvasSummary, CanvasView } from '../types/canvas'

export interface UseCanvasListResult {
  items: CanvasSummary[]
  total: number
  page: number
  hasMore: boolean
  loading: boolean
  /** 首次列表请求是否已结束（无论成功/失败） */
  loaded: boolean
  error: ApiError | Error | null
  refresh: () => Promise<void>
  loadMore: () => Promise<void>
  setFilters: (next: Pick<CanvasListQuery, 'keyword' | 'tag'>) => void
  create: (title?: string) => Promise<CanvasView | null>
  remove: (id: number) => Promise<boolean>
}

export function useCanvasList(options: { immediate?: boolean } = {}): UseCanvasListResult {
  const { immediate = true } = options
  const [filters, setFilterState] = useState<Pick<CanvasListQuery, 'keyword' | 'tag'>>({})
  const [items, setItems] = useState<CanvasSummary[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)
  const [loaded, setLoaded] = useState(false)
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
        const data = await canvasApi.list({ ...filters, scope: 'mine', page: targetPage, page_size: 12 })
        if (!mountedRef.current || seq !== seqRef.current) return
        setItems((prev) => (append ? [...prev, ...data.items] : data.items))
        setTotal(data.total)
        setPage(data.page)
      } catch (err) {
        if (mountedRef.current && seq === seqRef.current) {
          setError(err instanceof Error ? err : new Error(String(err)))
        }
      } finally {
        if (mountedRef.current && seq === seqRef.current) {
          setLoading(false)
          setLoaded(true)
        }
      }
    },
    [filters],
  )

  const refresh = useCallback(() => load(1, false), [load])
  const loadMore = useCallback(() => load(page + 1, true), [load, page])

  const setFilters = useCallback((next: Pick<CanvasListQuery, 'keyword' | 'tag'>) => {
    setFilterState((prev) => ({ ...prev, ...next }))
    setPage(1)
    setItems([])
  }, [])

  const create = useCallback(async (title?: string): Promise<CanvasView | null> => {
    setError(null)
    try {
      const created = await canvasApi.create({
        title: title?.trim() || '未命名画布',
        nodes: [],
        edges: [],
        viewport: { x: 0, y: 0, zoom: 1 },
      })
      await refresh()
      return created
    } catch (err) {
      if (mountedRef.current) setError(err instanceof Error ? err : new Error(String(err)))
      return null
    }
  }, [refresh])

  const remove = useCallback(async (id: number): Promise<boolean> => {
    setError(null)
    try {
      await canvasApi.remove(id)
      if (mountedRef.current) {
        setItems((prev) => prev.filter((item) => item.id !== id))
        setTotal((prev) => Math.max(0, prev - 1))
      }
      return true
    } catch (err) {
      if (mountedRef.current) setError(err instanceof Error ? err : new Error(String(err)))
      return false
    }
  }, [])

  // filters 变化时自动回到第 1 页；load 依赖 filters，所以这里等价于监听筛选条件
  useEffect(() => {
    if (immediate) void load(1, false)
  }, [immediate, load])

  return { items, total, page, hasMore: items.length < total, loading, loaded, error, refresh, loadMore, setFilters, create, remove }
}
