/**
 * 标签组件。品牌色为主色，关系色按语义区分：
 * - counter 反例质疑 #ff6b6b
 * - evidence 补充佐证 #36c9a5
 * - hypothesis 并行假设 #ffc857
 */
import { cn } from '../lib/format'

export type TagTone = 'brand' | 'counter' | 'evidence' | 'hypothesis' | 'neutral'

const TONES: Record<TagTone, string> = {
  brand: 'bg-brand-soft text-brand border-brand/20 dark:bg-brand/10 dark:text-[#9aa9ff] dark:border-brand/30',
  counter: 'bg-counter-soft text-counter border-counter/20 dark:bg-counter/10 dark:text-[#ff9a9a] dark:border-counter/30',
  evidence: 'bg-evidence-soft text-evidence border-evidence/20 dark:bg-evidence/10 dark:text-[#7fe0c9] dark:border-evidence/30',
  hypothesis: 'bg-hypothesis-soft text-[#9a6b00] border-hypothesis/25 dark:bg-hypothesis/10 dark:text-hypothesis dark:border-hypothesis/30',
  neutral: 'bg-[var(--nodal-surface-soft)] text-[var(--nodal-muted)] border-[var(--nodal-border)]',
}

interface TagChipProps {
  label: string
  tone?: TagTone
  className?: string
  onClick?: () => void
}

export function TagChip({ label, tone = 'brand', className, onClick }: TagChipProps) {
  const shared = cn(
    'inline-flex items-center rounded-full border px-2.5 py-1 text-[12px] leading-none font-medium',
    TONES[tone],
    onClick && 'cursor-pointer transition hover:brightness-95',
    className,
  )
  if (onClick) {
    return (
      <button type="button" className={shared} onClick={onClick}>
        {label}
      </button>
    )
  }
  return <span className={shared}>{label}</span>
}
