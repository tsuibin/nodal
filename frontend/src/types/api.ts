/**
 * 模块 10：接口公共类型。
 *
 * 所有接口响应都是同一个信封；业务 data 由各模块 types/*.ts 定义。
 * 删除类接口统一返回 DeleteResult，避免每个 service 各写一遍内联对象。
 */

export interface ApiEnvelope<T> {
  code: number
  message: string
  data?: T
  details?: unknown
  request_id?: string
}

/** 语义化错误码（与后端 internal/apperr/codes.go 对齐） */
export const API_CODES = {
  ok: 0,
  invalidParam: 1000,
  unauthorized: 1001,
  tokenExpired: 1002,
  loginRequired: 1003,
  notFound: 1004,
  forbidden: 1005,
  emailTaken: 2001,
  userNotFound: 2002,
  badCredentials: 2003,
  accountDisabled: 2004,
  guestDisabled: 2005,
  guestExpired: 2006,
  agentNotFound: 3001,
  agentNameTaken: 3002,
  modelUnsupported: 3003,
  modelKeyMissing: 3004,
  modelCallFailed: 3005,
  emptyCompletion: 3006,
  inferRunning: 3007,
  knowledgeNotFound: 4001,
  uploadType: 4002,
  parseFailed: 4003,
  notMounted: 4004,
  embedFailed: 4005,
  internal: 5000,
  shareInvalid: 6001,
  shareExpired: 6002,
  shareForbidden: 6003,
  canvasNotPublished: 6004,
  commentNotFound: 6005,
  replUnavailable: 7001,
  replNotFound: 7002,
  replTooMany: 7003,
  replAttached: 7004,
  replLanguage: 7005,
  replCodeTooLarge: 7006,
  network: -1,
} as const

export type ApiCode = (typeof API_CODES)[keyof typeof API_CODES]

/** 删除接口统一返回体 */
export interface DeleteResult {
  id: number
  deleted: boolean
}

/** POST /api/demo/seed */
export interface DemoSeedResult {
  /** created=0 表示该身份已有画布，未重复创建 */
  created: number
  titles: string[]
}
