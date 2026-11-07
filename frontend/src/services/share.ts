/**
 * 分享链接：
 *
 * POST   /api/canvas/:id/share-link          创建/复用链接（read | fork）
 * GET    /api/canvas/:id/share-links          列出链接
 * DELETE /api/canvas/:id/share-links/:linkId  停用链接
 * GET    /api/share/:token                    校验并读取（公开）
 * POST   /api/share/:token/fork               按 fork 权限分叉
 */
import { request } from './http'
import type { CanvasView } from '../types/canvas'
import type { ShareLink, SharePermission, ShareResolveResult } from '../types/square'

export const shareApi = {
  createLink(
    canvasId: number,
    permission: SharePermission,
    expiresHours?: number,
  ): Promise<ShareLink> {
    return request<ShareLink>({
      url: `/canvas/${canvasId}/share-link`,
      method: 'POST',
      data: { permission, expires_hours: expiresHours },
    })
  },

  listLinks(canvasId: number): Promise<{ items: ShareLink[]; total: number }> {
    return request<{ items: ShareLink[]; total: number }>({
      url: `/canvas/${canvasId}/share-links`,
      method: 'GET',
    })
  },

  revokeLink(canvasId: number, linkId: number): Promise<{ id: number; enabled: boolean }> {
    return request<{ id: number; enabled: boolean }>({
      url: `/canvas/${canvasId}/share-links/${linkId}`,
      method: 'DELETE',
    })
  },

  resolve(token: string): Promise<ShareResolveResult> {
    return request<ShareResolveResult>({ url: `/share/${token}`, method: 'GET' })
  },

  fork(token: string, title?: string): Promise<CanvasView> {
    return request<CanvasView>({
      url: `/share/${token}/fork`,
      method: 'POST',
      data: title ? { title } : {},
    })
  },
}
