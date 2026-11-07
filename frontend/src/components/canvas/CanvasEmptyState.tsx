/**
 * 空画布引导：没有节点时居中显示，提供五种节点的快捷创建。
 */
import type { NodalNodeKind } from '../../types/canvas-node'
import { NODE_KIND_META } from '../../types/canvas-node'
import { CodeIcon, FileIcon, FormulaIcon, ImageIcon, TextIcon } from '../icons'

const ICONS = {
  text: TextIcon,
  code: CodeIcon,
  formula: FormulaIcon,
  image: ImageIcon,
  file: FileIcon,
} as const

export function CanvasEmptyState({
  canEdit,
  onAdd,
  onOpenLibrary,
}: {
  canEdit: boolean
  onAdd: (kind: NodalNodeKind) => void
  onOpenLibrary: () => void
}) {
  return (
    <div className="nodal-empty-canvas">
      <div className="nodal-empty-card">
        <p className="nodal-kicker">EMPTY CANVAS</p>
        <h2 className="mt-1 text-xl font-bold tracking-tight">先把一个判断放上来</h2>
        <p className="mt-2 text-xs leading-6 text-[var(--nodal-muted)]">
          Nodal 的推演从一个节点开始。选一种节点类型，写下你的问题、证据或代码，
          再拖一条关系线，让思维长出分支。
        </p>
        {canEdit ? (
          <div className="mt-5 grid grid-cols-2 gap-2 sm:grid-cols-5">
            {(Object.keys(NODE_KIND_META) as NodalNodeKind[]).map((kind) => {
              const Icon = ICONS[kind]
              return (
                <button key={kind} type="button" className="nodal-empty-action" onClick={() => onAdd(kind)}>
                  <Icon width={17} height={17} className="text-brand" />
                  {NODE_KIND_META[kind].label}
                </button>
              )
            })}
          </div>
        ) : (
          <button type="button" className="nodal-btn nodal-btn-primary mt-5" onClick={onOpenLibrary}>
            返回资源库
          </button>
        )}
      </div>
    </div>
  )
}
