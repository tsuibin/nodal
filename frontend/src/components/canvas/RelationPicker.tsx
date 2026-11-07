/**
 * 拖拽连线后先选关系类型，再落库。
 * 关系色与卡片色严格复用 RELATION_META，不在组件里另写色值。
 */
import { RELATION_META, RELATION_ORDER, type RelationType } from '../../types/canvas-node'
import { CloseIcon } from '../icons'

export interface RelationPickerProps {
  open: boolean
  onPick: (relation: RelationType) => void
  onCancel: () => void
}

export function RelationPicker({ open, onPick, onCancel }: RelationPickerProps) {
  if (!open) return null
  return (
    <div className="nodal-dialog-backdrop" role="presentation" onMouseDown={onCancel}>
      <div
        className="nodal-dialog max-w-lg"
        role="dialog"
        aria-modal="true"
        aria-label="选择关系类型"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <header className="nodal-dialog-head">
          <div>
            <p className="nodal-kicker">RELATION</p>
            <h2 className="nodal-dialog-title">这条线表达什么关系？</h2>
          </div>
          <button type="button" className="nodal-icon-btn" onClick={onCancel} aria-label="关闭">
            <CloseIcon />
          </button>
        </header>
        <div className="grid grid-cols-1 gap-2 p-5 sm:grid-cols-2">
          {RELATION_ORDER.map((relation) => {
            const meta = RELATION_META[relation]
            return (
              <button
                key={relation}
                type="button"
                onClick={() => onPick(relation)}
                className="group rounded-xl border p-3 text-left transition hover:-translate-y-0.5"
                style={{ borderColor: `${meta.color}55`, background: `${meta.color}0d` }}
              >
                <span className="flex items-center gap-2 text-sm font-semibold" style={{ color: meta.color }}>
                  <span className="h-2.5 w-2.5 rounded-full" style={{ background: meta.color }} />
                  {meta.label}
                </span>
                <span className="mt-1 block text-[11px] leading-5 text-[var(--nodal-muted)]">
                  {meta.description}
                </span>
              </button>
            )
          })}
        </div>
        <footer className="nodal-dialog-foot">
          <span className="text-[11px] text-[var(--nodal-muted)]">连线颜色会跟随关系类型变化，之后可在右侧面板修改。</span>
          <button type="button" className="nodal-btn nodal-btn-ghost ml-auto" onClick={onCancel}>
            取消
          </button>
        </footer>
      </div>
    </div>
  )
}
