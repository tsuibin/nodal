/**
 * Agent 模板 / 模型目录 / 模型凭证的类型，与后端 service 出参一一对应。
 */

export interface AgentTemplate {
  id: number
  name: string
  description: string
  system_prompt: string
  model_id: string
  model_label: string
  created_at: string
  updated_at: string
}

export interface AgentListPayload {
  items: AgentTemplate[]
  total: number
}

export interface CreateAgentPayload {
  name: string
  description?: string
  system_prompt: string
  model_id: string
}

/** 更新接口支持部分字段，后端用指针字段区分「未传」与「置空」 */
export interface UpdateAgentPayload {
  name?: string
  description?: string
  system_prompt?: string
  model_id?: string
}

export interface ModelCatalogItem {
  id: string
  label: string
  provider: string
  max_tokens: number
  requires_key: boolean
  /** 该 provider 是否已配置 API Key（mock 等免密钥模型恒为 true） */
  configured: boolean
}

export interface ModelCredentialView {
  provider: string
  label: string
  configured: boolean
  /** 只有尾号，如 "…7890"；任何接口都不会返回明文 */
  key_hint: string
  base_url: string
  updated_at?: string
}

export interface ModelCredentialListPayload {
  items: ModelCredentialView[]
}

export interface SaveCredentialPayload {
  api_key: string
  /** 可选：自建代理/网关，留空用内置默认地址 */
  base_url?: string
}

/** 删除某个 provider 的自有密钥（不返回明文） */
export interface CredentialDeleteResult {
  provider: string
  deleted: boolean
}
