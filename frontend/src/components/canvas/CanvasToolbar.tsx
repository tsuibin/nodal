/**
 * 画布顶部工具栏：返回、标题、保存、撤销重做、适配视图、添加节点、
 * 分享/Fork、右侧面板与主题切换。
 */
import { useEffect, useRef, useState } from 'react'
import { ThemeToggle } from '../ThemeToggle'
import type { NodalNodeKind } from '../../types/canvas-node'
import { NODE_KIND_META } from '../../types/canvas-node'
import {
  ArrowLeftIcon,
  ChevronDownIcon,
  CodeIcon,
  FileIcon,
  FitIcon,
  FormulaIcon,
  ForkIcon,
  ImageIcon,
  PanelIcon,
  PlusIcon,
  RedoIcon,
  SaveIcon,
  ShareIcon,
  TextIcon,
  UndoIcon,
} from '../icons'

const KIND_ICON = {
  text: TextIcon,
  code: CodeIcon,
  formula: FormulaIcon,
  image: ImageIcon,
  file: FileIcon,
} as const

export interface CanvasToolbarProps {
  title: string
  dirty: boolean
  saving: boolean
  savedAt: string
  canEdit: boolean
  canShare: boolean
  canFork: boolean
  canUndo: boolean
  canRedo: boolean
  panelOpen: boolean
  onBack: () => void
  onTitleChange: (title: string) => void
  onCommitTitle: () => void
  onSave: () => void
  onUndo: () => void
  onRedo: () => void
  onFit: () => void
  onAddNode: (kind: NodalNodeKind) => void
  onShare: () => void
  onFork: () => void
  onTogglePanel: () => void
}

export function CanvasToolbar({
  title,
  dirty,
  saving,
  savedAt,
  canEdit,
  canShare,
  canFork,
  canUndo,
  canRedo,
  panelOpen,
  onBack,
  onTitleChange,
  onCommitTitle,
  onSave,
  onUndo,
  onRedo,
  onFit,
  onAddNode,
  onShare,
  onFork,
  onTogglePanel,
}: CanvasToolbarProps) {
  const [addOpen, setAddOpen] = useState(false)
  const addRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!addOpen) return
    const onDown = (event: MouseEvent) => {
      if (!addRef.current?.contains(event.target as Node)) setAddOpen(false)
    }
    window.addEventListener('mousedown', onDown)
    return () => window.removeEventListener('mousedown', onDown)
  }, [addOpen])

  return (
    <header className="nodal-canvas-toolbar">
      <button type="button" className="nodal-icon-btn" onClick={onBack} aria-label="返回资源库">
        <ArrowLeftIcon />
      </button>

      <div className="min-w-0">
        <input
          className="nodal-canvas-title"
          value={title}
          disabled={!canEdit}
          onChange={(event) => onTitleChange(event.target.value)}
          onBlur={onCommitTitle}
          placeholder="未命名画布"
          aria-label="画布标题"
        />
        <p className="nodal-canvas-save-state">
          {saving ? '保存中…' : dirty ? '有未保存修改' : savedAt ? `已保存 · ${savedAt}` : '已保存'}
        </p>
      </div>

      <div className="ml-2 flex items-center gap-1">
        <button type="button" className="nodal-icon-btn" onClick={onUndo} disabled={!canUndo} title="撤销 Ctrl+Z">
          <UndoIcon />
        </button>
        <button type="button" className="nodal-icon-btn" onClick={onRedo} disabled={!canRedo} title="重做 Ctrl+Shift+Z">
          <RedoIcon />
        </button>
        <button type="button" className="nodal-icon-btn" onClick={onFit} title="适配视图">
          <FitIcon />
        </button>
      </div>

      <div className="relative ml-1" ref={addRef}>
        <button
          type="button"
          className="nodal-toolbar-btn"
          disabled={!canEdit}
          onClick={() => setAddOpen((open) => !open)}
        >
          <PlusIcon width={15} height={15} />
          添加节点
          <ChevronDownIcon width={13} height={13} />
        </button>
        {addOpen ? (
          <div className="nodal-add-menu">
            {(Object.keys(NODE_KIND_META) as NodalNodeKind[]).map((kind) => {
              const Icon = KIND_ICON[kind]
              return (
                <button
                  key={kind}
                  type="button"
                  className="nodal-add-menu-item"
                  onClick={() => {
                    onAddNode(kind)
                    setAddOpen(false)
                  }}
                >
                  <span className="flex h-7 w-7 items-center justify-center rounded-lg bg-brand/10 text-brand">
                    <Icon width={15} height={15} />
                  </span>
                  <span>
                    <span className="block text-xs font-semibold">{NODE_KIND_META[kind].label}</span>
                    <span className="block text-[10px] text-[var(--nodal-muted)]">
                      {NODE_KIND_META[kind].hint}
                    </span>
                  </span>
                </button>
              )
            })}
          </div>
        ) : null}
      </div>

      <div className="ml-auto flex items-center gap-1">
        <button type="button" className="nodal-toolbar-btn" onClick={onSave} disabled={!canEdit || saving}>
          <SaveIcon width={15} height={15} />
          <span className="hidden sm:inline">保存</span>
        </button>
        <button type="button" className="nodal-toolbar-btn" onClick={onShare} disabled={!canShare} title="分享">
          <ShareIcon width={15} height={15} />
          <span className="hidden sm:inline">分享</span>
        </button>
        <button type="button" className="nodal-toolbar-btn" onClick={onFork} disabled={!canFork} title="Fork 画布">
          <ForkIcon width={15} height={15} />
          <span className="hidden md:inline">Fork</span>
        </button>
        <button
          type="button"
          className={`nodal-icon-btn ${panelOpen ? 'nodal-icon-btn-active' : ''}`}
          onClick={onTogglePanel}
          title="右侧面板"
        >
          <PanelIcon />
        </button>
        <ThemeToggle />
      </div>
    </header>
  )
}
