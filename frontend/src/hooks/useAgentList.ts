/**
 * Agent 模板列表。游客调用会拿到 1003，页面据此展示登录引导。
 */
import { useCallback, useEffect, useRef, useState } from 'react'
import { ApiError } from '../services/http'
import { agentApi } from '../services/agent'
import type { AgentTemplate } from '../types/agent'

export interface UseAgentListResult {
  items: AgentTemplate[]
  loading: boolean
  error: ApiError | Error | null
  refresh: () => Promise<void>
}

export function useAgentList(immediate = true): UseAgentListResult {
  const [items, setItems] = useState<AgentTemplate[]>([])
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
      const data = await agentApi.list()
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
