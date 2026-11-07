/**
 * 公共广场接口：
 *
 * POST   /api/canvas/:id/publish       发布到广场（仅所有者）
 * DELETE /api/canvas/:id/publish       下架
 * GET    /api/square/list              广场列表（公开，分页 + 标签筛选 + 排序）
 * POST   /api/canvas/:id/like          切换点赞
 * POST   /api/canvas/:id/comment       发表评论
 * GET    /api/canvas/:id/comments      评论列表
 * DELETE /api/canvas/:id/comments/:id  删除评论（作者或画布所有者）
 */
import { request } from './http'
import type { CanvasView } from '../types/canvas'
import type { CanvasComment, CommentList, LikeResult, SquareList } from '../types/square'

export interface SquareListQuery {
  tag?: string
  keyword?: string
  sort?: 'latest' | 'hot'
  page?: number
  page_size?: number
}

export const squareApi = {
  publish(canvasId: number): Promise<CanvasView> {
    return request<CanvasView>({ url: `/canvas/${canvasId}/publish`, method: 'POST' })
  },

  unpublish(canvasId: number): Promise<CanvasView> {
    return request<CanvasView>({ url: `/canvas/${canvasId}/publish`, method: 'DELETE' })
  },

  list(query: SquareListQuery = {}): Promise<SquareList> {
    return request<SquareList>({ url: '/square/list', method: 'GET', params: query })
  },

  like(canvasId: number): Promise<LikeResult> {
    return request<LikeResult>({ url: `/canvas/${canvasId}/like`, method: 'POST' })
  },

  comment(canvasId: number, content: string): Promise<CanvasComment> {
    return request<CanvasComment>({ url: `/canvas/${canvasId}/comment`, method: 'POST', data: { content } })
  },

  comments(canvasId: number, page = 1, pageSize = 20): Promise<CommentList> {
    return request<CommentList>({
      url: `/canvas/${canvasId}/comments`,
      method: 'GET',
      params: { page, page_size: pageSize },
    })
  },

  removeComment(canvasId: number, commentId: number): Promise<{ id: number; deleted: boolean }> {
    return request<{ id: number; deleted: boolean }>({
      url: `/canvas/${canvasId}/comments/${commentId}`,
      method: 'DELETE',
    })
  },
}
