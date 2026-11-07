/**
 * RAG 检索：只检索「当前画布已挂载」的文件。
 * 未挂载时返回 4004（knowledgeNotMounted），前端可提示先去挂载。
 */
import { request } from './http'
import type { RetrievePayload, RetrieveResult } from '../types/knowledge'

export const ragApi = {
  retrieve(payload: RetrievePayload): Promise<RetrieveResult> {
    return request<RetrieveResult>({ url: '/rag/retrieve', method: 'POST', data: payload })
  },
}
