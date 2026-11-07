/**
 * 认证状态 hooks。
 *
 * <AuthProvider autoGuest> 包在应用根部：
 *   - 启动时用本地 token 调 /auth/me 恢复会话；
 *   - 用户 token 失效（1001/1002）自动回到匿名/游客态；
 *   - 跨标签页登录态变化通过 storage 事件同步。
 *
 * 页面里：
 *   const { status, user, isGuest, login, logout, ensureGuest } = useAuth()
 */
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'

import { authApi } from '../services/auth'
import { ApiError, onTokenChange, onUnauthorized, tokenStore } from '../services/http'
import type {
  AuthUser,
  GuestInfo,
  LoginPayload,
  ProfilePayload,
  RegisterPayload,
  SessionPayload,
  UpdateProfilePayload,
} from '../types/auth'

export type AuthStatus = 'loading' | 'anonymous' | 'guest' | 'authenticated'

interface AuthState {
  status: AuthStatus
  user: AuthUser | null
  guest: GuestInfo | null
}

export interface AuthContextValue extends AuthState {
  isAuthenticated: boolean
  isGuest: boolean
  /** 最近一次认证操作失败的错误，页面可直接展示 */
  error: ApiError | Error | null
  login: (payload: LoginPayload) => Promise<SessionPayload>
  register: (payload: RegisterPayload) => Promise<SessionPayload>
  logout: (options?: { keepGuest?: boolean }) => Promise<void>
  /** 重新拉取 /auth/me（例如从后台切回前台） */
  refresh: () => Promise<ProfilePayload | null>
  updateProfile: (payload: UpdateProfilePayload) => Promise<AuthUser>
  /** 为当前匿名访客领取游客临时会话（进入临时画布时调用） */
  ensureGuest: () => Promise<SessionPayload | null>
  clearError: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

const INITIAL_STATE: AuthState = { status: 'loading', user: null, guest: null }

interface AuthProviderProps {
  children: ReactNode
  /** 首次进入且没有任何 token 时，是否自动领取游客会话，默认 false（进入画布时再调用 ensureGuest） */
  autoGuest?: boolean
}

export function AuthProvider({ children, autoGuest = false }: AuthProviderProps) {
  const [state, setState] = useState<AuthState>(INITIAL_STATE)
  const [error, setError] = useState<ApiError | Error | null>(null)

  const applyProfile = useCallback((profile: ProfilePayload) => {
    if (profile.is_guest) {
      setState({
        status: 'guest',
        user: null,
        guest: profile.guest ?? null,
      })
      return
    }
    setState({
      status: profile.user ? 'authenticated' : 'anonymous',
      user: profile.user ?? null,
      guest: null,
    })
  }, [])

  const applySession = useCallback((session: SessionPayload) => {
    if (session.is_guest) {
      setState({
        status: 'guest',
        user: null,
        guest: session.guest ?? null,
      })
      return
    }
    setState({
      status: session.user ? 'authenticated' : 'anonymous',
      user: session.user ?? null,
      guest: null,
    })
  }, [])

  const ensureGuest = useCallback(async (): Promise<SessionPayload | null> => {
    try {
      const session = await authApi.guest()
      tokenStore.setGuest(session.token)
      applySession(session)
      return session
    } catch (err) {
      // 后端关闭游客模式（2005）不算错误，静默保持匿名
      if (err instanceof ApiError && err.code === 2005) {
        setState({ status: 'anonymous', user: null, guest: null })
        return null
      }
      setError(err instanceof Error ? err : new Error(String(err)))
      return null
    }
  }, [applySession])

  const refresh = useCallback(async (): Promise<ProfilePayload | null> => {
    const { token } = tokenStore.getActive()
    if (!token) {
      setState({ status: 'anonymous', user: null, guest: null })
      return null
    }
    try {
      const profile = await authApi.me()
      applyProfile(profile)
      return profile
    } catch (err) {
      if (err instanceof ApiError && (err.isUnauthorized || err.code === 2006)) {
        tokenStore.clear()
        setState({ status: 'anonymous', user: null, guest: null })
        return null
      }
      setError(err instanceof Error ? err : new Error(String(err)))
      return null
    }
  }, [applyProfile])

  // 启动恢复会话
  useEffect(() => {
    let cancelled = false

    const bootstrap = async () => {
      const { token } = tokenStore.getActive()
      if (!token) {
        if (autoGuest) {
          await ensureGuest()
        }
        if (!cancelled && !autoGuest) {
          setState((prev) => (prev.status === 'loading' ? { status: 'anonymous', user: null, guest: null } : prev))
        }
        return
      }
      await refresh()
    }

    void bootstrap()
    return () => {
      cancelled = true
    }
  }, [autoGuest, ensureGuest, refresh])

  // 登录失效广播（另一个页面/接口 401）
  useEffect(() => {
    return onUnauthorized(() => {
      setState({ status: 'anonymous', user: null, guest: null })
    })
  }, [])

  // 跨标签页同步：storage 变化后重新拉取
  useEffect(() => {
    const handler = (event: StorageEvent) => {
      if (event.key && event.key !== 'nodal.auth.token' && event.key !== 'nodal.guest.token') return
      void refresh()
    }
    window.addEventListener('storage', handler)
    return () => window.removeEventListener('storage', handler)
  }, [refresh])

  // 供非 React 场景（如 axios 拦截器外部）感知 token 变化
  useEffect(() => {
    return onTokenChange(() => {
      /* 由各 action 自行更新 state，这里只保留扩展点 */
    })
  }, [])

  const login = useCallback(
    async (payload: LoginPayload): Promise<SessionPayload> => {
      setError(null)
      try {
        const session = await authApi.login(payload)
        tokenStore.setUser(session.token)
        applySession(session)
        return session
      } catch (err) {
        setError(err instanceof Error ? err : new Error(String(err)))
        throw err
      }
    },
    [applySession],
  )

  const register = useCallback(
    async (payload: RegisterPayload): Promise<SessionPayload> => {
      setError(null)
      try {
        const session = await authApi.register(payload)
        tokenStore.setUser(session.token)
        applySession(session)
        return session
      } catch (err) {
        setError(err instanceof Error ? err : new Error(String(err)))
        throw err
      }
    },
    [applySession],
  )

  const logout = useCallback(
    async (options?: { keepGuest?: boolean }): Promise<void> => {
      const keepGuest = options?.keepGuest ?? true
      authApi.logout()
      setState({ status: 'anonymous', user: null, guest: null })
      setError(null)
      if (keepGuest) {
        // 退出后仍可浏览临时画布，重新领一个游客身份
        await ensureGuest()
      }
    },
    [ensureGuest],
  )

  const updateProfile = useCallback(
    async (payload: UpdateProfilePayload): Promise<AuthUser> => {
      setError(null)
      try {
        const profile = await authApi.updateProfile(payload)
        applyProfile(profile)
        return profile.user as AuthUser
      } catch (err) {
        setError(err instanceof Error ? err : new Error(String(err)))
        throw err
      }
    },
    [applyProfile],
  )

  const clearError = useCallback(() => setError(null), [])

  const value = useMemo<AuthContextValue>(
    () => ({
      ...state,
      isAuthenticated: state.status === 'authenticated',
      isGuest: state.status === 'guest',
      error,
      login,
      register,
      logout,
      refresh,
      updateProfile,
      ensureGuest,
      clearError,
    }),
    [state, error, login, register, logout, refresh, updateProfile, ensureGuest, clearError],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error('useAuth 必须在 <AuthProvider> 内使用')
  }
  return ctx
}

/** 便捷选择器：需要登录才能操作的地方用这个判断 */
export function useRequireUser(): { ready: boolean; isGuest: boolean; hint: string | null } {
  const { status, isGuest, isAuthenticated } = useAuth()
  return {
    ready: status !== 'loading' && isAuthenticated,
    isGuest,
    hint: isGuest ? '登录后可保存画布到你的账号' : null,
  }
}
