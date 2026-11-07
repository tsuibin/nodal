/**
 * 认证接口封装，与后端路由一一对应：
 *
 * POST /api/auth/register   注册并直接返回登录态
 * POST /api/auth/login      登录
 * GET  /api/auth/me         恢复会话（用户或游客）
 * PUT  /api/auth/profile    修改昵称/头像（仅注册用户）
 * POST /api/auth/guest      领取游客临时会话（临时画布入口）
 */
import { request, tokenStore } from './http'
import type {
  LoginPayload,
  ProfilePayload,
  RegisterPayload,
  SessionPayload,
  UpdateProfilePayload,
} from '../types/auth'

export const authApi = {
  register(payload: RegisterPayload): Promise<SessionPayload> {
    return request<SessionPayload>({
      url: '/auth/register',
      method: 'POST',
      data: payload,
    })
  },

  login(payload: LoginPayload): Promise<SessionPayload> {
    return request<SessionPayload>({
      url: '/auth/login',
      method: 'POST',
      data: payload,
    })
  },

  me(): Promise<ProfilePayload> {
    return request<ProfilePayload>({ url: '/auth/me', method: 'GET' })
  },

  updateProfile(payload: UpdateProfilePayload): Promise<ProfilePayload> {
    return request<ProfilePayload>({
      url: '/auth/profile',
      method: 'PUT',
      data: payload,
    })
  },

  /** 领取游客临时会话；后端同时会把 token 写进 X-Guest-Token 响应头 */
  guest(): Promise<SessionPayload> {
    return request<SessionPayload>({ url: '/auth/guest', method: 'POST' })
  },

  /** 登出是纯客户端行为：JWT 无状态，服务端未做黑名单 */
  logout(): void {
    tokenStore.clear()
  },
}

export type { LoginPayload, ProfilePayload, RegisterPayload, SessionPayload, UpdateProfilePayload }
