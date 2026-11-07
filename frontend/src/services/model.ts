/**
 * 模型目录与 API Key 管理：
 *
 * GET    /api/model/catalog                 模型目录（含每个模型是否已配 Key）
 * GET    /api/model/credentials             各 provider 的配置状态（只返回尾号）
 * PUT    /api/model/credentials/:provider  保存/更新 Key（后端加密落库）
 * DELETE /api/model/credentials/:provider  删除 Key
 *
 * 注意：保存后在页面上只能看到 key_hint（如 "…7890"），永远拿不到明文。
 */
import { request } from './http'
import type {
  ModelCatalogItem,
  ModelCredentialListPayload,
  ModelCredentialView,
  SaveCredentialPayload,
} from '../types/agent'

export interface ModelCatalogPayload {
  items: ModelCatalogItem[]
}

export const modelApi = {
  catalog(): Promise<ModelCatalogPayload> {
    return request<ModelCatalogPayload>({ url: '/model/catalog', method: 'GET' })
  },

  listCredentials(): Promise<ModelCredentialListPayload> {
    return request<ModelCredentialListPayload>({ url: '/model/credentials', method: 'GET' })
  },

  saveCredential(provider: string, payload: SaveCredentialPayload): Promise<ModelCredentialView> {
    return request<ModelCredentialView>({
      url: `/model/credentials/${provider}`,
      method: 'PUT',
      data: payload,
    })
  },

  removeCredential(provider: string): Promise<{ provider: string; deleted: boolean }> {
    return request<{ provider: string; deleted: boolean }>({
      url: `/model/credentials/${provider}`,
      method: 'DELETE',
    })
  },
}
