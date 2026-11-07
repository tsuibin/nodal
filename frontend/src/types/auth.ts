/**
 * 与后端 service 层出参一一对应的类型。
 * 时间字段均为 RFC3339 字符串（服务端统一 UTC），展示前用 new Date() 本地化。
 */

export interface AuthUser {
  id: number
  email: string
  nickname: string
  avatar_url: string
  created_at: string
}

export interface GuestInfo {
  id: string
  expires_at: string
}

/** POST /api/auth/register、/login、/guest 的返回体 */
export interface SessionPayload {
  token: string
  token_type: 'Bearer'
  expires_at: string
  is_guest: boolean
  user?: AuthUser
  guest?: GuestInfo
}

/** GET /api/auth/me、PUT /api/auth/profile 的返回体 */
export interface ProfilePayload {
  is_guest: boolean
  user?: AuthUser
  guest?: GuestInfo
}

export interface RegisterPayload {
  email: string
  password: string
  nickname?: string
}

export interface LoginPayload {
  email: string
  password: string
}

export interface UpdateProfilePayload {
  nickname?: string
  avatar_url?: string
}
