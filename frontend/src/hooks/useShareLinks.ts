/**
 * useShareLinks：画布分享链接的创建/列出/停用。
 */
import { useCallback, useEffect, useRef, useState } from 'react'
import { ApiError } from '../services/http'
import { shareApi } from '../services/share'
import type { ShareLink, SharePermission } from '../types/square'

export interface UseShareLinksResult {
  links: ShareLink[]
  loading: boolean
  error: ApiError | Error | null
  refresh: () => Promise<ShareLink[] | null>
  create: (permission: SharePermission, expiresHours?: number) => Promise<ShareLink | null>
  revoke: (linkId: number) => Promise<boolean>
}

export function useShareLinks(canvasId: number | null): UseShareLinksResult {
  const [links, setLinks] = useState<ShareLink[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<ApiError | Error | null>(null)

  const mountedRef = useRef(true)
  useEffect(() => {
    mountedRef.current = true
    return () => {
      mountedRef.current = false
    }
  }, [])

  const refresh = useCallback(async (): Promise<ShareLink[] | null> => {
    if (!canvasId) return null
    setLoading(true)
    setError(null)
    try {
      const data = await shareApi.listLinks(canvasId)
      if (mountedRef.current) setLinks(data.items)
      return data.items
    } catch (err) {
      if (mountedRef.current) setError(err instanceof Error ? err : new Error(String(err)))
      return null
    } finally {
      if (mountedRef.current) setLoading(false)
    }
  }, [canvasId])

  const create = useCallback(
    async (permission: SharePermission, expiresHours?: number): Promise<ShareLink | null> => {
      if (!canvasId) return null
      setLoading(true)
      setError(null)
      try {
        const link = await shareApi.createLink(canvasId, permission, expiresHours)
        if (mountedRef.current) {
          setLinks((prev) => (prev.some((item) => item.id === link.id) ? prev : [link, ...prev]))
        }
        return link
      } catch (err) {
        if (mountedRef.current) setError(err instanceof Error ? err : new Error(String(err)))
        return null
      } finally {
        if (mountedRef.current) setLoading(false)
      }
    },
    [canvasId],
  )

  const revoke = useCallback(
    async (linkId: number): Promise<boolean> => {
      if (!canvasId) return false
      try {
        await shareApi.revokeLink(canvasId, linkId)
        if (mountedRef.current) {
          setLinks((prev) => prev.map((item) => (item.id === linkId ? { ...item, enabled: false } : item)))
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
    else setLinks([])
  }, [canvasId, refresh])

  return { links, loading, error, refresh, create, revoke }
}
