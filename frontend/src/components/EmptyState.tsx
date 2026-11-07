/** 空状态：图标 + 标题 + 说明 + 可选操作，页面统一视觉。 */
import type { ReactNode } from 'react'

interface EmptyStateProps {
  icon?: ReactNode
  title: string
  description?: string
  action?: ReactNode
  compact?: boolean
}

export function EmptyState({ icon, title, description, action, compact }: EmptyStateProps) {
  return (
    <div
      className={`flex flex-col items-center justify-center rounded-2xl border border-dashed border-[var(--nodal-border)] bg-[var(--nodal-surface)] text-center ${
        compact ? 'px-5 py-8' : 'px-6 py-14'
      }`}
    >
      {icon && (
        <span className="mb-3 flex h-11 w-11 items-center justify-center rounded-full bg-brand-soft text-brand dark:bg-brand/10">
          {icon}
        </span>
      )}
      <p className="text-sm font-semibold text-[var(--nodal-text)]">{title}</p>
      {description && <p className="mt-1.5 max-w-sm text-xs leading-relaxed text-[var(--nodal-muted)]">{description}</p>}
      {action && <div className="mt-4">{action}</div>}
    </div>
  )
}
