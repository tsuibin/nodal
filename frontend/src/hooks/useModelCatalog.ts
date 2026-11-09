/**
 * 模型目录（/api/model/catalog）。推演弹窗用它渲染模型下拉框。
 * 未配置 Key 的模型仍然列出，但标记为不可选，避免用户选中后才收到 3004。
 */
import { useCallback, useEffect, useRef, useState } from 'react'
import { ApiError } from '../services/http'
import { modelApi } from '../services/model'
import type { ModelCatalogItem } from '../types/agent'

export interface UseModelCatalogResult {
  items: ModelCatalogItem[]
  loading: boolean
  error: ApiError | Error | null
  refresh: () => Promise<void>
}

export function useModelCatalog(immediate = true): UseModelCatalogResult {
  const [items, setItems] = useState<ModelCatalogItem[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<ApiError | Error | null>(null)
  const mountedRef = useRef(true)

  useEffect(() => {
    mountedRef.current = true
    return () => {
      mountedRef.current = false
    }
  }, [])

  const refresh = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await modelApi.catalog()
      if (mountedRef.current) setItems(data.items)
    } catch (err) {
      if (mountedRef.current) setError(err instanceof Error ? err : new Error(String(err)))
    } finally {
      if (mountedRef.current) setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (immediate) void refresh()
  }, [immediate, refresh])

  return { items, loading, error, refresh }
}
