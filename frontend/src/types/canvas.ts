/**
 * 画布相关类型（与后端 CanvasView / CanvasSummary 对齐）。
 * 节点/边沿用模块 4 的结构化类型，可直接喂给 React Flow。
 */
import type { RFEdge, RFNode } from './infer'

export interface CanvasOwner {
  is_guest: boolean
  user_id?: number
}

export interface CanvasPermissions {
  is_owner: boolean
  can_edit: boolean
  can_delete: boolean
  can_fork: boolean
  can_share: boolean
}

export interface CanvasView {
  id: number
  title: string
  description: string
  nodes: RFNode[]
  edges: RFEdge[]
  viewport: { x: number; y: number; zoom: number }
  tags: string[]
  is_public: boolean
  share_mode: 'private' | 'link' | 'public'
  forked_from?: number
  node_count: number
  edge_count: number
  like_count: number
  comment_count: number
  published_at?: string
  owner: CanvasOwner
  permissions: CanvasPermissions
  created_at: string
  updated_at: string
}

/** 列表项（GET /api/canvas/list），不包含 nodes/edges，避免列表传输过大 */
export interface CanvasSummary {
  id: number
  title: string
  description: string
  tags: string[]
  is_public: boolean
  share_mode: 'private' | 'link' | 'public'
  forked_from?: number
  node_count: number
  edge_count: number
  like_count: number
  comment_count: number
  published_at?: string
  owner: CanvasOwner
  permissions: CanvasPermissions
  created_at: string
  updated_at: string
}

export interface CanvasList {
  items: CanvasSummary[]
  total: number
  page: number
  page_size: number
  has_more: boolean
}

export interface CanvasListQuery {
  scope?: 'mine' | 'public'
  keyword?: string
  tag?: string
  page?: number
  page_size?: number
}

export interface CreateCanvasPayload {
  title?: string
  description?: string
  nodes?: RFNode[]
  edges?: RFEdge[]
  viewport?: { x: number; y: number; zoom: number }
  tags?: string[]
  is_public?: boolean
  share_mode?: 'private' | 'link' | 'public'
}

export interface UpdateCanvasPayload extends CreateCanvasPayload {
  nodes?: RFNode[]
  edges?: RFEdge[]
}
