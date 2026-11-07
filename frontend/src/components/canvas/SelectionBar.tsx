/**
 * 选中节点/边后的浮动操作条：
 * 推演、编辑、标记结论、删除。
 */
import type { NodalNode } from '../../types/canvas-node'
import { CheckIcon, PencilIcon, SparkIcon, TrashIcon } from '../icons'

export interface SelectionBarProps {
  nodes: NodalNode[]
  edgeCount: number
  canEdit: boolean
  authenticated: boolean
  inferRunning: boolean
  onInfer: () => void
  onEdit: () => void
  onDelete: () => void
  onConclusion: () => void
}

export function SelectionBar({
  nodes,
  edgeCount,
  canEdit,
  authenticated,
  inferRunning,
  onInfer,
  onEdit,
  onDelete,
  onConclusion,
}: SelectionBarProps) {
  if (!nodes.length && !edgeCount) return null
  const oneNode = nodes.length === 1 && edgeCount === 0

  return (
    <div className="nodal-selection-bar">
      <span className="nodal-selection-count">
        {nodes.length ? `${nodes.length} 个节点` : ''}
        {nodes.length && edgeCount ? ' · ' : ''}
        {edgeCount ? `${edgeCount} 条连线` : ''}
      </span>
      {nodes.length > 0 ? (
        <button
          type="button"
          className="nodal-selection-primary"
          disabled={inferRunning}
          onClick={onInfer}
          title={authenticated ? '选中节点继续推演' : '登录后可用 AI 推演'}
        >
          <SparkIcon width={15} height={15} />
          {inferRunning ? '推演中…' : '推演'}
        </button>
      ) : null}
      {oneNode && canEdit ? (
        <button type="button" className="nodal-selection-btn" onClick={onEdit}>
          <PencilIcon width={14} height={14} />
          编辑
        </button>
      ) : null}
      {oneNode && canEdit ? (
        <button type="button" className="nodal-selection-btn" onClick={onConclusion}>
          <CheckIcon width={14} height={14} />
          {nodes[0]?.data.status === 'conclusion' ? '取消结论' : '标记结论'}
        </button>
      ) : null}
      {canEdit ? (
        <button type="button" className="nodal-selection-btn nodal-selection-danger" onClick={onDelete}>
          <TrashIcon width={14} height={14} />
          删除
        </button>
      ) : null}
      {!authenticated ? <span className="text-[10px] text-[var(--nodal-muted)]">登录后可 AI 推演</span> : null}
    </div>
  )
}
