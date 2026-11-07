/**
 * 登录 / 注册弹窗。
 *
 * - 与 AuthProvider 对接，成功后写入 JWT 并广播登录态；
 * - 错误按语义码给出可执行文案（邮箱已注册、密码错误等），不直接透传后端堆栈；
 * - 支持 Esc / 遮罩关闭，焦点落在第一个输入框。
 */
import { useEffect, useRef, useState, type FormEvent } from 'react'
import { ApiError } from '../services/http'
import { useAuth } from '../hooks/useAuth'
import { CloseIcon, LogoMark } from './icons'

type Mode = 'login' | 'register'

interface LoginModalProps {
  open: boolean
  onClose: () => void
  initialMode?: Mode
}

function errorText(err: unknown): string {
  if (err instanceof ApiError) {
    switch (err.code) {
      case 1000:
        return '请输入有效的邮箱和至少 8 位密码'
      case 2001:
        return '该邮箱已注册，请直接登录'
      case 2003:
        return '邮箱或密码不正确'
      case 2004:
        return '账号已被停用，请联系管理员'
      default:
        return err.message || '请求失败，请稍后重试'
    }
  }
  return err instanceof Error ? err.message : '网络异常，请稍后重试'
}

export function LoginModal({ open, onClose, initialMode = 'login' }: LoginModalProps) {
  const { login, register } = useAuth()
  const [mode, setMode] = useState<Mode>(initialMode)
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [nickname, setNickname] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const emailRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (!open) return
    setMode(initialMode)
    setError(null)
    setSubmitting(false)
    const timer = window.setTimeout(() => emailRef.current?.focus(), 120)
    return () => window.clearTimeout(timer)
  }, [open, initialMode])

  useEffect(() => {
    if (!open) return
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', onKey)
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', onKey)
      document.body.style.overflow = ''
    }
  }, [open, onClose])

  if (!open) return null

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (submitting) return
    if (!email.trim() || password.length < 8) {
      setError('请输入有效邮箱，密码至少 8 位')
      return
    }
    setSubmitting(true)
    setError(null)
    try {
      if (mode === 'login') {
        await login({ email: email.trim(), password })
      } else {
        await register({
          email: email.trim(),
          password,
          ...(nickname.trim() ? { nickname: nickname.trim() } : {}),
        })
      }
      onClose()
    } catch (err) {
      setError(errorText(err))
    } finally {
      setSubmitting(false)
    }
  }

  const inputClass =
    'w-full rounded-xl border border-[var(--nodal-border)] bg-[var(--nodal-bg)] px-3.5 py-2.5 text-sm text-[var(--nodal-text)] outline-none transition placeholder:text-[var(--nodal-muted)] focus:border-brand focus:ring-2 focus:ring-brand/15'

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/45 p-4 backdrop-blur-sm nodal-fade-in"
      role="dialog"
      aria-modal="true"
      aria-label="登录或注册 Nodal"
      onMouseDown={onClose}
    >
      <div
        className="w-full max-w-md rounded-2xl border border-[var(--nodal-border)] bg-[var(--nodal-surface)] p-6 shadow-2xl nodal-rise-in sm:p-7"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <div className="mb-5 flex items-start justify-between">
          <div className="flex items-center gap-2.5">
            <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-brand-soft">
              <LogoMark />
            </span>
            <div>
              <p className="text-[15px] font-semibold text-[var(--nodal-text)]">进入 Nodal</p>
              <p className="mt-0.5 text-xs text-[var(--nodal-muted)]">登录后保存画布、挂载知识库、使用 Agent</p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-full p-1.5 text-[var(--nodal-muted)] transition hover:bg-[var(--nodal-surface-soft)] hover:text-[var(--nodal-text)]"
            aria-label="关闭"
          >
            <CloseIcon />
          </button>
        </div>

        <div className="mb-5 grid grid-cols-2 rounded-xl bg-[var(--nodal-surface-soft)] p-1">
          {(['login', 'register'] as const).map((item) => (
            <button
              key={item}
              type="button"
              onClick={() => {
                setMode(item)
                setError(null)
              }}
              className={`rounded-lg py-2 text-sm font-medium transition ${
                mode === item
                  ? 'bg-[var(--nodal-surface)] text-brand shadow-sm'
                  : 'text-[var(--nodal-muted)] hover:text-[var(--nodal-text)]'
              }`}
            >
              {item === 'login' ? '登录' : '注册'}
            </button>
          ))}
        </div>

        <form className="space-y-3.5" onSubmit={submit}>
          <label className="block">
            <span className="mb-1.5 block text-xs font-medium text-[var(--nodal-muted)]">邮箱</span>
            <input
              ref={emailRef}
              type="email"
              autoComplete="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              placeholder="you@example.com"
              className={inputClass}
            />
          </label>
          {mode === 'register' && (
            <label className="block">
              <span className="mb-1.5 block text-xs font-medium text-[var(--nodal-muted)]">昵称（可选）</span>
              <input
                type="text"
                autoComplete="nickname"
                maxLength={20}
                value={nickname}
                onChange={(event) => setNickname(event.target.value)}
                placeholder="别人在广场看到的名字"
                className={inputClass}
              />
            </label>
          )}
          <label className="block">
            <span className="mb-1.5 block text-xs font-medium text-[var(--nodal-muted)]">密码</span>
            <input
              type="password"
              autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
              minLength={8}
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              placeholder="至少 8 位"
              className={inputClass}
            />
          </label>

          {error && (
            <p role="alert" className="rounded-lg bg-counter-soft px-3 py-2 text-xs text-counter dark:bg-counter/10">
              {error}
            </p>
          )}

          <button
            type="submit"
            disabled={submitting}
            className="mt-1 flex w-full items-center justify-center rounded-xl bg-brand px-4 py-2.5 text-sm font-semibold text-white shadow-[0_8px_20px_rgb(64_96_255/0.28)] transition hover:bg-brand-strong disabled:cursor-not-allowed disabled:opacity-60"
          >
            {submitting ? '处理中…' : mode === 'login' ? '登录' : '创建账号'}
          </button>
        </form>

        <p className="mt-4 text-center text-[11px] leading-relaxed text-[var(--nodal-muted)]">
          不登录也可以先进入临时画布；临时身份 7 天后失效，登录后画布归属你的账号。
        </p>
      </div>
    </div>
  )
}
