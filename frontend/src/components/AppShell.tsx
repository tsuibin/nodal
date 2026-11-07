/**
 * 全站外壳：顶部导航 + 登录态 + 主题切换 + 页脚。
 * 所有页面通过 <Outlet /> 渲染，保证导航和主题状态不随路由重建。
 */
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { useAuth } from '../hooks/useAuth'
import { useLoginModal } from './LoginModalProvider'
import { LogoMark, PlusIcon } from './icons'
import { ThemeToggle } from './ThemeToggle'

const NAV_ITEMS = [
  { to: '/', label: '首页', end: true },
  { to: '/library', label: '资源库', end: false },
  { to: '/square', label: '广场', end: false },
]

export function AppShell() {
  const navigate = useNavigate()
  const { status, user, isGuest, logout } = useAuth()
  const { openLogin } = useLoginModal()

  const displayName = user?.nickname || (isGuest ? '游客' : '')
  const avatarLetter = displayName.slice(0, 1).toUpperCase() || 'N'

  return (
    <div className="flex min-h-screen flex-col bg-[var(--nodal-bg)] text-[var(--nodal-text)]">
      <header className="sticky top-0 z-40 border-b border-[var(--nodal-border)] bg-[var(--nodal-bg)]/82 backdrop-blur-xl">
        <div className="mx-auto flex h-16 w-full max-w-7xl items-center gap-4 px-4 sm:px-6 lg:px-8">
          <button
            type="button"
            onClick={() => navigate('/')}
            className="flex shrink-0 items-center gap-2 rounded-lg py-1 pr-2 text-left"
            aria-label="Nodal 首页"
          >
            <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-brand-soft dark:bg-brand/10">
              <LogoMark />
            </span>
            <span className="hidden sm:block">
              <span className="block text-[17px] font-bold leading-5 tracking-tight text-[var(--nodal-text)]">Nodal</span>
              <span className="block text-[10px] leading-4 text-[var(--nodal-muted)]">思维不再是一条直线</span>
            </span>
          </button>

          <nav className="ml-2 flex items-center gap-1 sm:ml-6" aria-label="主导航">
            {NAV_ITEMS.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.end}
                className={({ isActive }) =>
                  `rounded-full px-3 py-1.5 text-[13px] font-medium transition sm:px-4 ${
                    isActive
                      ? 'bg-brand-soft text-brand dark:bg-brand/12'
                      : 'text-[var(--nodal-muted)] hover:bg-[var(--nodal-surface-soft)] hover:text-[var(--nodal-text)]'
                  }`
                }
              >
                {item.label}
              </NavLink>
            ))}
          </nav>

          <div className="ml-auto flex items-center gap-2">
            <button
              type="button"
              onClick={() => navigate('/library?new=1')}
              className="hidden items-center gap-1.5 rounded-full bg-brand px-3.5 py-2 text-[12px] font-semibold text-white shadow-[0_6px_16px_rgb(64_96_255/0.24)] transition hover:bg-brand-strong sm:inline-flex"
            >
              <PlusIcon width={15} height={15} />
              新建画布
            </button>

            <ThemeToggle />

            {status === 'loading' ? (
              <span className="h-9 w-20 animate-pulse rounded-full bg-[var(--nodal-surface-soft)]" />
            ) : status === 'authenticated' && user ? (
              <div className="flex items-center gap-2">
                <span className="flex h-8 w-8 items-center justify-center rounded-full bg-brand text-xs font-bold text-white" title={user.email}>
                  {avatarLetter}
                </span>
                <span className="hidden max-w-[92px] truncate text-xs font-medium text-[var(--nodal-text)] md:block">
                  {displayName}
                </span>
                <button
                  type="button"
                  onClick={() => void logout()}
                  className="rounded-full px-2 py-1 text-[11px] text-[var(--nodal-muted)] transition hover:text-[var(--nodal-text)]"
                >
                  退出
                </button>
              </div>
            ) : (
              <div className="flex items-center gap-1.5">
                <span className="hidden rounded-full border border-[var(--nodal-border)] px-2 py-1 text-[10px] text-[var(--nodal-muted)] sm:inline">
                  {isGuest ? '游客模式' : '未登录'}
                </span>
                <button
                  type="button"
                  onClick={openLogin}
                  className="rounded-full border border-brand/30 bg-[var(--nodal-surface)] px-3.5 py-1.5 text-[12px] font-semibold text-brand transition hover:bg-brand-soft dark:hover:bg-brand/10"
                >
                  登录
                </button>
              </div>
            )}
          </div>
        </div>
      </header>

      <main className="flex-1">
        <Outlet />
      </main>

      <footer className="border-t border-[var(--nodal-border)]">
        <div className="mx-auto flex w-full max-w-7xl flex-col gap-2 px-4 py-6 text-[11px] text-[var(--nodal-muted)] sm:flex-row sm:items-center sm:justify-between sm:px-6 lg:px-8">
          <p>Nodal · 无限画布式 AI 思维推演引擎</p>
          <p className="flex flex-wrap items-center gap-x-3 gap-y-1">
            <span>React 18 + ReactFlow</span>
            <span aria-hidden="true">·</span>
            <span>Go + PostgreSQL</span>
            <span aria-hidden="true">·</span>
            <span>你的数据只属于你</span>
          </p>
        </div>
      </footer>
    </div>
  )
}
