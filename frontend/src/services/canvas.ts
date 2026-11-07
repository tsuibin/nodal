/**
 * 画布 CRUD / 分叉接口，对应后端：
 *
 * POST   /api/canvas              创建
 * GET    /api/canvas/list         我的画布 / 公开画布列表
 * GET    /api/canvas/:id          详情（nodes/edges/viewport 可直接喂 ReactFlow）
 * PUT    /api/canvas/:id          全量更新 nodes + edges
 * DELETE /api/canvas/:id          删除
 * POST   /api/canvas/:id/fork     分叉
 *
 * 鉴权：全部走全局 JWT 中间件；游客也能创建/读取自己的临时画布，
 * 但知识库、Agent 模板等能力游客调用会返回 1003。
 */
import { request } from './http'
import type { DeleteResult } from '../types/api'
import type {
  CanvasList,
  CanvasListQuery,
  CanvasView,
  CreateCanvasPayload,
  UpdateCanvasPayload,
} from '../types/canvas'

export const canvasApi = {
  list(query: CanvasListQuery = {}): Promise<CanvasList> {
    return request<CanvasList>({
      url: '/canvas/list',
      method: 'GET',
      params: { scope: query.scope ?? 'mine', ...query },
    })
  },

  create(payload: CreateCanvasPayload = {}): Promise<CanvasView> {
    return request<CanvasView>({ url: '/canvas', method: 'POST', data: payload })
  },

  get(id: number): Promise<CanvasView> {
    return request<CanvasView>({ url: `/canvas/${id}`, method: 'GET' })
  },

  update(id: number, payload: UpdateCanvasPayload): Promise<CanvasView> {
    return request<CanvasView>({ url: `/canvas/${id}`, method: 'PUT', data: payload })
  },

  remove(id: number): Promise<DeleteResult> {
    return request<DeleteResult>({ url: `/canvas/${id}`, method: 'DELETE' })
  },

  fork(id: number, title?: string): Promise<CanvasView> {
    return request<CanvasView>({
      url: `/canvas/${id}/fork`,
      method: 'POST',
      data: title ? { title } : {},
    })
  },
}
