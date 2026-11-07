/**
 * 节点编辑器。
 *
 * 文本：Markdown 编辑 / 预览切换；
 * 代码：Monaco + 语言选择 + REPL 运行；
 * 公式：KaTeX 实时预览；
 * 图片：URL 或本地文件（转 data URL，限制 1.5MB）；
 * 文件：记录文件名、类型、大小（文件内容不入库）。
 */
import { useEffect, useMemo, useState, type ChangeEvent } from 'react'
import Editor from '@monaco-editor/react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import '../../lib/monaco'
import { useTheme } from '../../hooks/useTheme'
import { renderFormula } from '../../lib/katex'
import { CodeNodeRunner } from '../CodeNodeRunner'
import { NODE_KIND_META, type NodalNode, type NodalNodeData } from '../../types/canvas-node'
import { CheckIcon, CloseIcon, TrashIcon } from '../icons'
import { ShikiCode } from './ShikiCode'

export interface NodeEditorDialogProps {
  node: NodalNode | null
  canEdit: boolean
  onClose: () => void
  onApply: (id: string, patch: Partial<NodalNodeData>) => void
  onDelete: (id: string) => void
}

export function NodeEditorDialog({
  node,
  canEdit,
  onClose,
  onApply,
  onDelete,
}: NodeEditorDialogProps) {
  const { resolved } = useTheme()
  const [draft, setDraft] = useState<NodalNodeData | null>(null)
  const [textMode, setTextMode] = useState<'edit' | 'preview'>('edit')

  useEffect(() => {
    setDraft(node ? { ...node.data } : null)
    setTextMode('edit')
  }, [node])

  useEffect(() => {
    if (!node) return
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [node, onClose])

  const kind = draft?.kind ?? 'text'
  const kindMeta = NODE_KIND_META[kind]
  const canRun =
    kind === 'code' &&
    Boolean(draft?.code?.trim()) &&
    canEdit &&
    (draft?.language === 'bash' || draft?.language === 'python' || draft?.language === undefined)
  const formulaHTML = useMemo(() => renderFormula(draft?.formula ?? '', true), [draft?.formula])

  if (!node || !draft) return null

  const patch = (value: Partial<NodalNodeData>) => setDraft((current) => (current ? { ...current, ...value } : current))

  const apply = () => {
    if (!canEdit) return
    onApply(node.id, draft)
    onClose()
  }

  return (
    <div className="nodal-dialog-backdrop" role="presentation" onMouseDown={onClose}>
      <div
        className="nodal-dialog nodal-node-editor"
        role="dialog"
        aria-modal="true"
        aria-label="编辑节点"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <header className="nodal-dialog-head">
          <div>
            <p className="nodal-kicker">NODE EDITOR</p>
            <h2 className="nodal-dialog-title">
              {draft?.label?.trim() || `${kindMeta.label}节点`}
              <span className="ml-2 text-[11px] font-normal text-[var(--nodal-muted)]">
                {kindMeta.label} · {node.id}
              </span>
            </h2>
          </div>
          <button type="button" className="nodal-icon-btn" onClick={onClose} aria-label="关闭">
            <CloseIcon />
          </button>
        </header>

        {!canEdit ? (
          <div className="nodal-readonly-banner">只读模式：当前身份没有编辑权限，内容仅供查看。</div>
        ) : null}

        <div className="nodal-node-editor-body">
          <label className="nodal-field">
            <span>标题</span>
            <input
              value={draft.label}
              disabled={!canEdit}
              onChange={(event) => patch({ label: event.target.value })}
              className="nodal-input"
              placeholder="给这个节点一个短标题"
            />
          </label>

          {kind === 'text' ? (
            <div className="nodal-editor-section">
              <div className="nodal-tabs">
                <button
                  type="button"
                  className={textMode === 'edit' ? 'nodal-tab nodal-tab-active' : 'nodal-tab'}
                  onClick={() => setTextMode('edit')}
                >
                  编辑
                </button>
                <button
                  type="button"
                  className={textMode === 'preview' ? 'nodal-tab nodal-tab-active' : 'nodal-tab'}
                  onClick={() => setTextMode('preview')}
                >
                  预览
                </button>
              </div>
              {textMode === 'edit' ? (
                <textarea
                  value={draft.content ?? ''}
                  disabled={!canEdit}
                  onChange={(event) => patch({ content: event.target.value })}
                  className="nodal-textarea min-h-[300px] font-mono text-[13px]"
                  placeholder="支持 Markdown：标题、列表、引用、表格、代码块…"
                />
              ) : (
                <div className="nodal-markdown-preview">
                  <ReactMarkdown remarkPlugins={[remarkGfm]}>{draft.content ?? ''}</ReactMarkdown>
                </div>
              )}
            </div>
          ) : null}

          {kind === 'code' ? (
            <div className="nodal-editor-section">
              <div className="flex items-center gap-2">
                <span className="text-xs text-[var(--nodal-muted)]">语言</span>
                <select
                  value={draft.language ?? 'python'}
                  disabled={!canEdit}
                  onChange={(event) => patch({ language: event.target.value })}
                  className="nodal-select"
                >
                  <option value="python">Python</option>
                  <option value="bash">Shell / Bash</option>
                  <option value="javascript">JavaScript（仅高亮）</option>
                  <option value="typescript">TypeScript（仅高亮）</option>
                </select>
                <span className="ml-auto text-[11px] text-[var(--nodal-muted)]">沙箱支持 Python / Shell</span>
              </div>
              <div className="nodal-monaco-wrap">
                <Editor
                  height="320px"
                  language={draft.language === 'bash' ? 'shell' : draft.language ?? 'python'}
                  value={draft.code ?? ''}
                  theme={resolved === 'dark' ? 'vs-dark' : 'light'}
                  options={{
                    readOnly: !canEdit,
                    minimap: { enabled: false },
                    fontSize: 13,
                    lineHeight: 20,
                    scrollBeyondLastLine: false,
                    padding: { top: 10, bottom: 10 },
                    wordWrap: 'on',
                    automaticLayout: true,
                  }}
                  onChange={(value) => patch({ code: value ?? '' })}
                />
              </div>
              {canRun ? (
                <CodeNodeRunner
                  code={draft.code ?? ''}
                  language={draft.language === 'bash' ? 'shell' : 'python'}
                  title="代码沙箱"
                />
              ) : (
                <p className="text-[11px] text-[var(--nodal-muted)]">
                  代码语言只支持 Python / Shell；其他语言仅做语法高亮，不进入沙箱。
                </p>
              )}
            </div>
          ) : null}

          {kind === 'formula' ? (
            <div className="nodal-editor-section">
              <label className="nodal-field">
                <span>KaTeX 表达式</span>
                <textarea
                  value={draft.formula ?? ''}
                  disabled={!canEdit}
                  onChange={(event) => patch({ formula: event.target.value })}
                  className="nodal-textarea min-h-[120px] font-mono text-[13px]"
                  placeholder={'例如：\\frac{-b \\pm \\sqrt{b^2-4ac}}{2a}'}
                />
              </label>
              <div
                className="nodal-formula-preview"
                dangerouslySetInnerHTML={{ __html: formulaHTML }}
              />
            </div>
          ) : null}

          {kind === 'image' ? (
            <div className="nodal-editor-section">
              <label className="nodal-field">
                <span>图片 URL</span>
                <input
                  value={draft.imageUrl ?? ''}
                  disabled={!canEdit}
                  onChange={(event) => patch({ imageUrl: event.target.value })}
                  className="nodal-input"
                  placeholder="https://… 或上传本地图片"
                />
              </label>
              {canEdit ? (
                <label className="nodal-file-pick">
                  <span>上传图片（≤1.5MB，会转成 data URL 存进画布）</span>
                  <input
                    type="file"
                    accept="image/*"
                    onChange={(event) => void readImage(event, (url, size) => patch({ imageUrl: url, fileSize: size }))}
                  />
                </label>
              ) : null}
              {draft.imageUrl ? (
                <img src={draft.imageUrl} alt={draft.label} className="nodal-image-preview" />
              ) : null}
            </div>
          ) : null}

          {kind === 'file' ? (
            <div className="nodal-editor-section">
              {canEdit ? (
                <label className="nodal-file-pick">
                  <span>选择文件，只记录元信息（文件名、类型、大小），不上传内容</span>
                  <input
                    type="file"
                    onChange={(event) => {
                      const file = event.target.files?.[0]
                      if (!file) return
                      patch({
                        fileName: file.name,
                        fileType: file.type || '未知类型',
                        fileSize: file.size,
                        label: draft.label || file.name,
                      })
                    }}
                  />
                </label>
              ) : null}
              <div className="nodal-node-file nodal-file-inline">
                <div className="min-w-0">
                  <p className="truncate font-medium">{draft.fileName || '未选择文件'}</p>
                  <p className="text-[11px] text-[var(--nodal-muted)]">
                    {draft.fileType || '未知类型'}
                    {draft.fileSize ? ` · ${formatBytes(draft.fileSize)}` : ''}
                  </p>
                </div>
              </div>
            </div>
          ) : null}

          {kind === 'code' && draft.code ? (
            <details className="nodal-code-details">
              <summary>查看当前高亮预览</summary>
              <ShikiCode code={draft.code} language={draft.language} />
            </details>
          ) : null}
        </div>

        <footer className="nodal-dialog-foot">
          {canEdit ? (
            <button
              type="button"
              className="nodal-btn nodal-btn-danger"
              onClick={() => {
                onDelete(node.id)
                onClose()
              }}
            >
              <TrashIcon width={15} height={15} />
              删除节点
            </button>
          ) : null}
          <button
            type="button"
            className="nodal-btn nodal-btn-ghost"
            onClick={() => patch({ status: draft.status === 'conclusion' ? 'normal' : 'conclusion' })}
            disabled={!canEdit}
          >
            <CheckIcon width={15} height={15} />
            {draft.status === 'conclusion' ? '取消结论' : '标记结论'}
          </button>
          <div className="ml-auto flex items-center gap-2">
            <button type="button" className="nodal-btn nodal-btn-ghost" onClick={onClose}>
              取消
            </button>
            {canEdit ? (
              <button type="button" className="nodal-btn nodal-btn-primary" onClick={apply}>
                应用修改
              </button>
            ) : null}
          </div>
        </footer>
      </div>
    </div>
  )
}

function readImage(event: ChangeEvent<HTMLInputElement>, done: (url: string, size: number) => void) {
  const file = event.target.files?.[0]
  if (!file) return
  if (file.size > 1.5 * 1024 * 1024) {
    window.alert('图片超过 1.5MB，请压缩后再上传，或改用图片 URL。')
    return
  }
  const reader = new FileReader()
  reader.onload = () => {
    if (typeof reader.result === 'string') done(reader.result, file.size)
  }
  reader.readAsDataURL(file)
}

function formatBytes(size: number): string {
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / (1024 * 1024)).toFixed(1)} MB`
}
