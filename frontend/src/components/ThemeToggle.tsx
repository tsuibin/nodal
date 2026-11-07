/** 全局主题切换：按钮显示当前解析后的主题，点击在 light/dark 间切换。 */
import { MoonIcon, SunIcon } from './icons'
import { useTheme } from '../hooks/useTheme'

export function ThemeToggle({ className = '' }: { className?: string }) {
  const { resolved, toggle } = useTheme()
  const isDark = resolved === 'dark'

  return (
    <button
      type="button"
      onClick={toggle}
      aria-label={isDark ? '切换到浅色主题' : '切换到深色主题'}
      title={isDark ? '浅色模式' : '深色模式'}
      className={`inline-flex h-9 w-9 items-center justify-center rounded-full border border-[var(--nodal-border)] bg-[var(--nodal-surface)] text-[var(--nodal-muted)] transition hover:text-brand hover:border-brand/40 ${className}`}
    >
      {isDark ? <SunIcon /> : <MoonIcon />}
    </button>
  )
}
