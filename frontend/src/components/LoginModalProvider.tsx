/**
 * 全局登录弹窗。页面通过 useLoginModal().openLogin() 唤起，
 * 不依赖某个具体按钮，避免资源库/广场各自维护一份弹窗状态。
 */
import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react'
import { LoginModal } from './LoginModal'

interface LoginModalContextValue {
  openLogin: () => void
  closeLogin: () => void
}

const LoginModalContext = createContext<LoginModalContextValue | null>(null)

export function LoginModalProvider({ children }: { children: ReactNode }) {
  const [open, setOpen] = useState(false)
  const openLogin = useCallback(() => setOpen(true), [])
  const closeLogin = useCallback(() => setOpen(false), [])
  const value = useMemo(() => ({ openLogin, closeLogin }), [openLogin, closeLogin])

  return (
    <LoginModalContext.Provider value={value}>
      {children}
      <LoginModal open={open} onClose={closeLogin} />
    </LoginModalContext.Provider>
  )
}

export function useLoginModal(): LoginModalContextValue {
  const ctx = useContext(LoginModalContext)
  if (!ctx) throw new Error('useLoginModal 必须在 <LoginModalProvider> 内使用')
  return ctx
}
