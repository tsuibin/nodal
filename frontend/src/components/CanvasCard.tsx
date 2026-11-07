/**
 * 画布卡片：资源库「我的画布」和广场复用同一个视觉组件。
 *
 * 两种数据源字段略有差异：
 * - CanvasSummary：owner / is_public / permissions
 * - SquareItem：author / liked
 * 用 'author' in item 做类型收窄，避免为展示层复制一套 DTO。
 */
import type { KeyboardEvent } from 'react'
import type { CanvasSummary } from '../types/canvas'
import type { SquareItem } from '../types/square'
import { formatCount, formatRelativeTime } from '../lib/format'
import { CommentIcon, HeartIcon, NodeIcon } from './icons'
import { TagChip } from './TagChip'

interface CanvasCardProps {
  item: CanvasSummary | SquareItem
  variant: 'mine' | 'square'
  onOpen?: () => void
  onLike?: () => void
  onDelete?: () => void
}

export function CanvasCard({ item, variant, onOpen, onLike, onDelete }: CanvasCardProps) {
  const isSquare = 'author' in item
  const authorName = isSquare ? item.author.nickname : item.owner.is_guest ? '游客' : '我的画布'
  const updatedAt = isSquare ? item.updated_at : item.updated_at
  const liked = isSquare ? item.liked : false

  const triggerOpen = () => onOpen?.()

  const onKeyDown = (event: KeyboardEvent<HTMLElement>) => {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault()
      triggerOpen()
    }
  }

  return (
    <article
      role="button"
      tabIndex={0}
      onClick={triggerOpen}
      onKeyDown={onKeyDown}
      className="nodal-card group flex cursor-pointer flex-col overflow-hidden transition duration-200 hover:-translate-y-0.5 hover:border-brand/35 hover:shadow-[var(--shadow-card-hover)]"
    >
      <div className="relative h-28 overflow-hidden border-b border-[var(--nodal-border)] bg-[var(--nodal-surface-soft)] nodal-grid-bg">
        <div className="absolute inset-0 bg-gradient-to-br from-brand/12 via-transparent to-evidence/10" />
        <div className="absolute left-5 top-4 flex items-center gap-1.5">
          <span className="h-2.5 w-2.5 rounded-full bg-brand" />
          <span className="h-2 w-2 rounded-full bg-evidence" />
          <span className="h-1.5 w-1.5 rounded-full bg-hypothesis" />
        </div>
        <div className="absolute right-4 top-3 rounded-full border border-[var(--nodal-border)] bg-[var(--nodal-surface)]/85 px-2 py-1 text-[10px] font-medium text-[var(--nodal-muted)] backdrop-blur">
          {item.node_count} 节点 · {item.edge_count} 连线
        </div>
        <p className="absolute bottom-3 left-5 right-5 line-clamp-2 text-[13px] leading-5 text-[var(--nodal-text)]/80">
          {item.description || '还没有描述，打开画布继续推演。'}
        </p>
      </div>

      <div className="flex flex-1 flex-col p-4">
        <div className="mb-2 flex items-start justify-between gap-2">
          <h3 className="line-clamp-1 text-[15px] font-semibold text-[var(--nodal-text)] transition group-hover:text-brand">
            {item.title || '未命名画布'}
          </h3>
          {variant === 'mine' && (
            <span className="shrink-0 rounded-full border border-[var(--nodal-border)] px-2 py-0.5 text-[10px] text-[var(--nodal-muted)]">
              {!isSquare && item.is_public ? '公开' : '私有'}
            </span>
          )}
        </div>

        {item.tags.length > 0 && (
          <div className="mb-3 flex flex-wrap gap-1.5">
            {item.tags.slice(0, 3).map((tag) => (
              <TagChip key={tag} label={tag} tone="neutral" />
            ))}
          </div>
        )}

        <div className="mt-auto flex items-center justify-between gap-2 border-t border-[var(--nodal-border)] pt-3">
          <div className="min-w-0">
            <p className="truncate text-xs font-medium text-[var(--nodal-text)]">{authorName}</p>
            <p className="mt-0.5 text-[11px] text-[var(--nodal-muted)]">更新于 {formatRelativeTime(updatedAt)}</p>
          </div>

          <div className="flex shrink-0 items-center gap-1 text-[11px] text-[var(--nodal-muted)]">
            <NodeIcon width={14} height={14} />
            <span>{formatCount(item.node_count)}</span>
            <span className="mx-0.5">·</span>
            <CommentIcon width={14} height={14} />
            <span>{formatCount(item.comment_count)}</span>
            {variant === 'square' ? (
              <button
                type="button"
                onClick={(event) => {
                  event.stopPropagation()
                  onLike?.()
                }}
                aria-label={liked ? '取消点赞' : '点赞'}
                className={`ml-1 inline-flex items-center gap-1 rounded-full px-1.5 py-1 transition ${
                  liked ? 'text-counter' : 'hover:text-counter'
                }`}
              >
                <HeartIcon width={14} height={14} filled={liked} />
                <span>{formatCount(item.like_count)}</span>
              </button>
            ) : (
              <span className="ml-1 inline-flex items-center gap-1">
                <HeartIcon width={14} height={14} />
                {formatCount(item.like_count)}
              </span>
            )}
            {variant === 'mine' && onDelete && (
              <button
                type="button"
                onClick={(event) => {
                  event.stopPropagation()
                  onDelete()
                }}
                className="ml-1 rounded px-1.5 py-1 text-[11px] text-[var(--nodal-muted)] transition hover:bg-counter-soft hover:text-counter dark:hover:bg-counter/10"
              >
                删除
              </button>
            )}
          </div>
        </div>
      </div>
    </article>
  )
}
