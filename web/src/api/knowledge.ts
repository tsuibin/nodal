/**
 * 知识库接口（都要求注册用户，游客返回 1003）：
 *
 * POST   /api/knowledge/upload                 上传并同步解析/切片/向量化
 * GET    /api/knowledge/list                   我的文件列表
 * DELETE /api/knowledge/:id                    删除（连同切片/向量/磁盘文件）
 * POST   /api/canvas/:id/mount-knowledge       覆盖式挂载（空数组=解除）
 * GET    /api/canvas/:id/mount-knowledge       当前挂载
 * DELETE /api/canvas/:id/mount-knowledge       解除挂载
 */
import { request } from './http'
import type { KnowledgeFile, KnowledgeListPayload, MountView } from '../types/knowledge'

export const knowledgeApi = {
  upload(file: File): Promise<KnowledgeFile> {
    const form = new FormData()
    form.append('file', file)
    return request<KnowledgeFile>({
      url: '/knowledge/upload',
      method: 'POST',
      data: form,
      // 不要手动设 Content-Type：浏览器需要自己补 multipart boundary
      timeout: 120_000,
    })
  },

  list(): Promise<KnowledgeListPayload> {
    return request<KnowledgeListPayload>({ url: '/knowledge/list', method: 'GET' })
  },

  remove(id: number): Promise<{ id: number; deleted: boolean }> {
    return request<{ id: number; deleted: boolean }>({ url: `/knowledge/${id}`, method: 'DELETE' })
  },

  mount(canvasId: number, fileIds: number[]): Promise<MountView> {
    return request<MountView>({
      url: `/canvas/${canvasId}/mount-knowledge`,
      method: 'POST',
      data: { file_ids: fileIds },
    })
  },

  currentMount(canvasId: number): Promise<MountView> {
    return request<MountView>({ url: `/canvas/${canvasId}/mount-knowledge`, method: 'GET' })
  },

  unmount(canvasId: number): Promise<MountView> {
    return request<MountView>({ url: `/canvas/${canvasId}/mount-knowledge`, method: 'DELETE' })
  },
}
