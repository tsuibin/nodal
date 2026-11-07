/**
 * 模块 6：公共广场 / 点赞 / 评论 / 分享链接。
 */
import type { CanvasView } from './canvas'

export interface SquareAuthor {
  user_id?: number
  nickname: string
  avatar_url?: string
  is_guest: boolean
}

export interface SquareItem {
  id: number
  title: string
  description: string
  tags: string[]
  forked_from?: number
  node_count: number
  edge_count: number
  like_count: number
  comment_count: number
  liked: boolean
  author: SquareAuthor
  permissions: CanvasView['permissions']
  published_at?: string
  updated_at: string
}

export interface SquareList {
  items: SquareItem[]
  total: number
  page: number
  page_size: number
  has_more: boolean
}

export interface LikeResult {
  canvas_id: number
  liked: boolean
  like_count: number
}

export interface CommentAuthor {
  user_id: number
  nickname: string
  avatar_url?: string
}

export interface CanvasComment {
  id: number
  canvas_id: number
  content: string
  author: CommentAuthor
  created_at: string
  mine: boolean
}

export interface CommentList {
  items: CanvasComment[]
  total: number
  page: number
  page_size: number
  has_more: boolean
}

export type SharePermission = 'read' | 'fork'

export interface ShareLink {
  id: number
  canvas_id: number
  token: string
  permission: SharePermission
  enabled: boolean
  expired: boolean
  expires_at?: string
  created_at: string
  /** 前端路由路径，例如 /share/<token> */
  share_path: string
}

export interface ShareResolveResult {
  token: string
  permission: SharePermission
  can_read: boolean
  can_fork: boolean
  expires_at?: string
  canvas: CanvasView
}
