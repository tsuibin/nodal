/**
 * Agent 模板接口，与后端路由一一对应（都要求注册用户）：
 *
 * POST   /api/agent        创建
 * GET    /api/agent/list   列表
 * GET    /api/agent/:id    详情
 * PUT    /api/agent/:id    局部更新
 * DELETE /api/agent/:id    删除（物理删除）
 */
import { request } from './http'
import type {
  AgentListPayload,
  AgentTemplate,
  CreateAgentPayload,
  UpdateAgentPayload,
} from '../types/agent'

export const agentApi = {
  create(payload: CreateAgentPayload): Promise<AgentTemplate> {
    return request<AgentTemplate>({ url: '/agent', method: 'POST', data: payload })
  },

  list(): Promise<AgentListPayload> {
    return request<AgentListPayload>({ url: '/agent/list', method: 'GET' })
  },

  get(id: number): Promise<AgentTemplate> {
    return request<AgentTemplate>({ url: `/agent/${id}`, method: 'GET' })
  },

  update(id: number, payload: UpdateAgentPayload): Promise<AgentTemplate> {
    return request<AgentTemplate>({ url: `/agent/${id}`, method: 'PUT', data: payload })
  },

  remove(id: number): Promise<{ id: number; deleted: boolean }> {
    return request<{ id: number; deleted: boolean }>({ url: `/agent/${id}`, method: 'DELETE' })
  },
}
