/**
 * 模块 5：私有知识库 / 画布临时挂载 / RAG 检索的类型。
 *
 * 与后端字段一一对应；上传是同步接口，返回时 status 已经是 ready/failed。
 */

export type KnowledgeFileStatus = 'processing' | 'ready' | 'failed'

export interface KnowledgeFile {
  id: number
  name: string
  ext: string
  mime: string
  size: number
  status: KnowledgeFileStatus
  chunk_count: number
  error?: string
  created_at: string
}

export interface KnowledgeListPayload {
  items: KnowledgeFile[]
  total: number
}

export interface MountedFile {
  id: number
  name: string
  chunk_count: number
  status: KnowledgeFileStatus
}

/** 挂载是临时关联：只存在服务端内存，重启/刷新失效 */
export interface MountView {
  canvas_id: number
  file_ids: number[]
  files: MountedFile[]
  updated_at?: string
}

export interface RetrievePayload {
      canvas_id: number
  query: string
  /** 1-20，缺省用服务端 NODAL_RAG_TOP_K */
  top_k?: number
}

export interface RetrievedChunk {
  file_id: number
  file_name: string
  chunk_index: number
  content: string
  preview: string
  score: number
}

export interface RetrieveResult {
  query: string
  top_k: number
  /** json = sqlite/mysql 兜底，pgvector = Postgres */
  backend: string
  items: RetrievedChunk[]
}
