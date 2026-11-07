/**
 * useCanvasComments：评论列表 + 发表 + 删除。
 * 未登录（游客）发表评论会收到 1003，页面据此弹登录框。
 */
import { useCallback, useEffect, useRef, useState } from 'react'
import { ApiError } from '../services/http'
import { squareApi } from '../services/square'
import type { CanvasComment } from '../types/square'

export interface UseCanvasCommentsResult {
  comments: CanvasComment[]
  total: number
  page: number
  hasMore: boolean
  loading: boolean
  submitting: boolean
  error: ApiError | Error | null
  refresh: () => Promise<void>
  loadMore: () => Promise<void>
  submit: (content: string) => Promise<CanvasComment | null>
  remove: (commentId: number) => Promise<boolean>
}

export function useCanvasComments(canvasId: number | null, pageSize = 20): UseCanvasCommentsResult {
  const [comments, setComments] = useState<CanvasComment[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<ApiError | Error | null>(null)

  const mountedRef = useRef(true)
  useEffect(() => {
    mountedRef.current = true
    return () => {
      mountedRef.current = false
    }
  }, [])

  const fetchPage = useCallback(
    async (targetPage: number, append: boolean) => {
      if (!canvasId) return
      setLoading(true)
      setError(null)
      try {
        const data = await squareApi.comments(canvasId, targetPage, pageSize)
        if (!mountedRef.current) return
        setComments((prev) => (append ? [...prev, ...data.items] : data.items))
        setTotal(data.total)
        setPage(data.page)
      } catch (err) {
        if (mountedRef.current) setError(err instanceof Error ? err : new Error(String(err)))
      } finally {
        if (mountedRef.current) setLoading(false)
      }
    },
    [canvasId, pageSize],
  )

  const refresh = useCallback(() => fetchPage(1, false), [fetchPage])
  const loadMore = useCallback(() => fetchPage(page + 1, true), [fetchPage, page])

  const submit = useCallback(
    async (content: string): Promise<CanvasComment | null> => {
      if (!canvasId) return null
      setSubmitting(true)
      setError(null)
      try {
        const created = await squareApi.comment(canvasId, content)
        if (mountedRef.current) {
          setComments((prev) => [created, ...prev])
          setTotal((prev) => prev + 1)
        }
        return created
      } catch (err) {
        if (mountedRef.current) setError(err instanceof Error ? err : new Error(String(err)))
        return null
      } finally {
        if (mountedRef.current) setSubmitting(false)
      }
    },
    [canvasId],
  )

  const remove = useCallback(
    async (commentId: number): Promise<boolean> => {
      if (!canvasId) return false
      try {
        await squareApi.removeComment(canvasId, commentId)
        if (mountedRef.current) {
          setComments((prev) => prev.filter((item) => item.id !== commentId))
          setTotal((prev) => Math.max(0, prev - 1))
        }
        return true
      } catch (err) {
        if (mountedRef.current) setError(err instanceof Error ? err : new Error(String(err)))
        return false
      }
    },
    [canvasId],
  )

  useEffect(() => {
    if (canvasId) void refresh()
    else {
      setComments([])
      setTotal(0)
      setPage(1)
    }
  }, [canvasId, refresh])

  return { comments, total, page, hasMore: comments.length < total, loading, submitting, error, refresh, loadMore, submit, remove }
}
