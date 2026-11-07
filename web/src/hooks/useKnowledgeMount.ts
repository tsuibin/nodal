/**
 * useKnowledgeMount：画布右侧面板「知识库挂载」。
 *
 * 挂载是临时关联（服务端内存），刷新页面后需要重新挂载；
 * hook 在 canvasId 变化时自动拉取当前挂载用于回显。
 */
import { useCallback, useEffect, useRef, useState } from 'react'
import { ApiError } from '../api/http'
import { knowledgeApi } from '../api/knowledge'
import { ragApi } from '../api/rag'
import type { MountView, RetrieveResult } from '../types/knowledge'

export interface UseKnowledgeMountResult {
  mount: MountView | null
  loading: boolean
  error: ApiError | Error | null
  refresh: () => Promise<MountView | null>
  setMount: (fileIds: number[]) => Promise<MountView | null>
  unmount: () => Promise<MountView | null>
  retrieve: (query: string, topK?: number) => Promise<RetrieveResult | null>
}

export function useKnowledgeMount(canvasId: number | null): UseKnowledgeMountResult {
  const [mount, setMountState] = useState<MountView | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<ApiError | Error | null>(null)

  const mountedRef = useRef(true)
  useEffect(() => {
    mountedRef.current = true
    return () => {
      mountedRef.current = false
    }
  }, [])

  const refresh = useCallback(async (): Promise<MountView | null> => {
    if (!canvasId) return null
    setLoading(true)
    setError(null)
    try {
      const view = await knowledgeApi.currentMount(canvasId)
      if (mountedRef.current) setMountState(view)
      return view
    } catch (err) {
      if (mountedRef.current) setError(err instanceof Error ? err : new Error(String(err)))
      return null
    } finally {
      if (mountedRef.current) setLoading(false)
    }
  }, [canvasId])

  const apply = useCallback(async (fn: () => Promise<MountView>): Promise<MountView | null> => {
    setLoading(true)
    setError(null)
    try {
      const view = await fn()
      if (mountedRef.current) setMountState(view)
      return view
    } catch (err) {
      if (mountedRef.current) setError(err instanceof Error ? err : new Error(String(err)))
      return null
    } finally {
      if (mountedRef.current) setLoading(false)
    }
  }, [])

  const setMount = useCallback(
    (fileIds: number[]) => {
      if (!canvasId) return Promise.resolve(null)
      return apply(() => knowledgeApi.mount(canvasId, fileIds))
    },
    [canvasId, apply],
  )

  const unmount = useCallback(() => {
    if (!canvasId) return Promise.resolve(null)
    return apply(() => knowledgeApi.unmount(canvasId))
  }, [canvasId, apply])

  const retrieve = useCallback(
    async (query: string, topK?: number): Promise<RetrieveResult | null> => {
      if (!canvasId) return null
      setError(null)
      try {
        return await ragApi.retrieve({ canvas_id: canvasId, query, top_k: topK })
      } catch (err) {
        if (mountedRef.current) setError(err instanceof Error ? err : new Error(String(err)))
        return null
      }
    },
    [canvasId],
  )

  useEffect(() => {
    if (canvasId) void refresh()
    else setMountState(null)
  }, [canvasId, refresh])

  return { mount, loading, error, refresh, setMount, unmount, retrieve }
}
