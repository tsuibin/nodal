/**
 * useKnowledgeFiles：知识库列表 + 上传 + 删除。
 *
 * const kb = useKnowledgeFiles()
 * await kb.refresh()
 * await kb.upload(file)
 */
import { useCallback, useEffect, useRef, useState } from 'react'
import { ApiError } from '../api/http'
import { knowledgeApi } from '../api/knowledge'
import type { KnowledgeFile } from '../types/knowledge'

export interface UseKnowledgeFilesResult {
  files: KnowledgeFile[]
  loading: boolean
  uploading: boolean
  error: ApiError | Error | null
  refresh: () => Promise<KnowledgeFile[] | null>
  upload: (file: File) => Promise<KnowledgeFile | null>
  remove: (id: number) => Promise<boolean>
}

export function useKnowledgeFiles(options: { immediate?: boolean } = {}): UseKnowledgeFilesResult {
  const [files, setFiles] = useState<KnowledgeFile[]>([])
  const [loading, setLoading] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [error, setError] = useState<ApiError | Error | null>(null)

  const mountedRef = useRef(true)
  useEffect(() => {
    mountedRef.current = true
    return () => {
      mountedRef.current = false
    }
  }, [])

  const refresh = useCallback(async (): Promise<KnowledgeFile[] | null> => {
    setLoading(true)
    setError(null)
    try {
      const data = await knowledgeApi.list()
      if (mountedRef.current) setFiles(data.items)
      return data.items
    } catch (err) {
      if (mountedRef.current) setError(err instanceof Error ? err : new Error(String(err)))
      return null
    } finally {
      if (mountedRef.current) setLoading(false)
    }
  }, [])

  const upload = useCallback(async (file: File): Promise<KnowledgeFile | null> => {
    setUploading(true)
    setError(null)
    try {
      const created = await knowledgeApi.upload(file)
      if (mountedRef.current) setFiles((prev) => [created, ...prev])
      return created
    } catch (err) {
      if (mountedRef.current) setError(err instanceof Error ? err : new Error(String(err)))
      return null
    } finally {
      if (mountedRef.current) setUploading(false)
    }
  }, [])

  const remove = useCallback(async (id: number): Promise<boolean> => {
    setError(null)
    try {
      await knowledgeApi.remove(id)
      if (mountedRef.current) setFiles((prev) => prev.filter((item) => item.id !== id))
      return true
    } catch (err) {
      if (mountedRef.current) setError(err instanceof Error ? err : new Error(String(err)))
      return false
    }
  }, [])

  useEffect(() => {
    if (options.immediate) void refresh()
  }, [options.immediate, refresh])

  return { files, loading, uploading, error, refresh, upload, remove }
}
