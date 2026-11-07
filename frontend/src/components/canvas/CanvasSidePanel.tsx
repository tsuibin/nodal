/**
 * 画布右侧面板：画布信息 / Agent 列表 / 知识库挂载。
 *
 * 知识库挂载是服务端内存临时关联，刷新页面会失效；面板每次打开都
 * 通过 currentMount 回显，并在保存后提示「重启失效」。
 */
import { useEffect, useState, type ChangeEvent, type ReactNode } from 'react'
import { useKnowledgeFiles } from '../../hooks/useKnowledge'
import { useKnowledgeMount } from '../../hooks/useKnowledgeMount'
import type { AgentTemplate } from '../../types/agent'
import type { CanvasView } from '../../types/canvas'
import type { NodalEdge, NodalNode } from '../../types/canvas-node'
import { BotIcon, BookIcon, CheckIcon, NodeIcon, UploadIcon } from '../icons'

export type SidePanelTab = 'info' | 'agents' | 'knowledge'

export interface CanvasSidePanelProps {
  canvas: CanvasView
  nodes: NodalNode[]
  edges: NodalEdge[]
  authenticated: boolean
  canEdit: boolean
  agents: AgentTemplate[]
  agentsLoading: boolean
  selectedAgentID: number | null
  onSelectAgent: (id: number) => void
  onNeedLogin: () => void
  onNeedAgent: () => void
  onMetaChange: (patch: { title?: string; description?: string; tags?: string[] }) => void
}

export function CanvasSidePanel({
  canvas,
  nodes,
  edges,
  authenticated,
  canEdit,
  agents,
  agentsLoading,
  selectedAgentID,
  onSelectAgent,
  onNeedLogin,
  onNeedAgent,
  onMetaChange,
}: CanvasSidePanelProps) {
  const [tab, setTab] = useState<SidePanelTab>('info')
  const [title, setTitle] = useState(canvas.title)
  const [description, setDescription] = useState(canvas.description)
  const [tagText, setTagText] = useState(canvas.tags.join(', '))

  useEffect(() => {
    setTitle(canvas.title)
    setDescription(canvas.description)
    setTagText(canvas.tags.join(', '))
  }, [canvas.id, canvas.title, canvas.description, canvas.tags])

  return (
    <aside className="nodal-side-panel">
      <div className="nodal-side-tabs">
        <TabButton active={tab === 'info'} onClick={() => setTab('info')} icon={<NodeIcon />} label="画布" />
        <TabButton active={tab === 'agents'} onClick={() => setTab('agents')} icon={<BotIcon />} label="Agent" />
        <TabButton
          active={tab === 'knowledge'}
          onClick={() => setTab('knowledge')}
          icon={<BookIcon />}
          label="知识库"
        />
      </div>

      {tab === 'info' ? (
        <div className="nodal-side-body">
          <div className="grid grid-cols-3 gap-2">
            <Stat label="节点" value={nodes.length} />
            <Stat label="连线" value={edges.length} />
            <Stat label="AI 节点" value={nodes.filter((node) => node.data.source === 'agent').length} />
          </div>

          <label className="nodal-field">
            <span>标题</span>
            <input
              className="nodal-input"
              value={title}
              disabled={!canEdit}
              onChange={(event) => setTitle(event.target.value)}
              onBlur={() => onMetaChange({ title: title.trim() || '未命名画布' })}
            />
          </label>

          <label className="nodal-field">
            <span>描述</span>
            <textarea
              className="nodal-textarea min-h-[88px]"
              value={description}
              disabled={!canEdit}
              onChange={(event) => setDescription(event.target.value)}
              onBlur={() => onMetaChange({ description })}
            />
          </label>

          <label className="nodal-field">
            <span>标签（逗号分隔，最多 10 个）</span>
            <input
              className="nodal-input"
              value={tagText}
              disabled={!canEdit}
              onChange={(event) => setTagText(event.target.value)}
              onBlur={() =>
                onMetaChange({
                  tags: tagText
                    .split(/[,，]/)
                    .map((tag) => tag.trim())
                    .filter(Boolean)
                    .slice(0, 10),
                })
              }
            />
          </label>

          <div className="nodal-side-meta">
            <MetaRow label="权限" value={canvas.permissions.can_edit ? '可编辑' : '只读'} />
            <MetaRow label="所有者" value={canvas.owner.is_guest ? '游客临时画布' : '注册用户'} />
            <MetaRow label="公开" value={canvas.is_public ? '已发布到广场' : '未公开'} />
            <MetaRow label="更新时间" value={formatDate(canvas.updated_at)} />
            {canvas.forked_from ? <MetaRow label="Fork 自" value={`#${canvas.forked_from}`} /> : null}
          </div>
        </div>
      ) : null}

      {tab === 'agents' ? (
        <div className="nodal-side-body">
          {!authenticated ? (
            <AuthGate onNeedLogin={onNeedLogin} what="Agent 模板与推演" />
          ) : (
            <>
              <div className="flex items-center justify-between">
                <p className="text-xs font-semibold">选择推演 Agent</p>
                <button type="button" className="text-[11px] text-brand hover:underline" onClick={onNeedAgent}>
                  + 新建
                </button>
              </div>
              {agentsLoading ? <p className="text-xs text-[var(--nodal-muted)]">加载中…</p> : null}
              {!agentsLoading && agents.length === 0 ? (
                <p className="rounded-xl border border-dashed border-[var(--nodal-border)] p-3 text-xs leading-5 text-[var(--nodal-muted)]">
                  还没有 Agent 模板。先创建 system prompt 和模型，才能推演。
                </p>
              ) : null}
              <div className="space-y-2">
                {agents.map((agent) => (
                  <button
                    key={agent.id}
                    type="button"
                    onClick={() => onSelectAgent(agent.id)}
                    className={`nodal-agent-row ${selectedAgentID === agent.id ? 'nodal-agent-row-active' : ''}`}
                  >
                    <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-brand/10 text-brand">
                      <BotIcon width={16} height={16} />
                    </span>
                    <span className="min-w-0 flex-1 text-left">
                      <span className="block truncate text-xs font-semibold">{agent.name}</span>
                      <span className="block truncate text-[10px] text-[var(--nodal-muted)]">
                        {agent.model_label || agent.model_id}
                      </span>
                    </span>
                    {selectedAgentID === agent.id ? <CheckIcon width={15} height={15} className="text-brand" /> : null}
                  </button>
                ))}
              </div>
              <p className="text-[10px] leading-4 text-[var(--nodal-muted)]">
                选中节点后点画布底部「推演」，AI 新增的内容会实时出现在空闲位置。
              </p>
            </>
          )}
        </div>
      ) : null}

      {tab === 'knowledge' ? (
        <KnowledgeTab
          canvasId={canvas.id}
          authenticated={authenticated}
          onNeedLogin={onNeedLogin}
        />
      ) : null}
    </aside>
  )
}

function KnowledgeTab({
  canvasId,
  authenticated,
  onNeedLogin,
}: {
  canvasId: number
  authenticated: boolean
  onNeedLogin: () => void
}) {
  const files = useKnowledgeFiles({ immediate: authenticated })
  const mount = useKnowledgeMount(authenticated ? canvasId : null)
  const [selected, setSelected] = useState<number[]>([])
  const [notice, setNotice] = useState('')

  useEffect(() => {
    setSelected(mount.mount?.file_ids ?? [])
  }, [mount.mount])

  if (!authenticated) {
    return (
      <div className="nodal-side-body">
        <AuthGate onNeedLogin={onNeedLogin} what="知识库挂载与 RAG 召回" />
      </div>
    )
  }

  const toggle = (id: number) => {
    setSelected((current) =>
      current.includes(id) ? current.filter((item) => item !== id) : [...current, id],
    )
  }

  const save = async () => {
    setNotice('')
    const result = await mount.setMount(selected)
    setNotice(result ? `已挂载 ${result.files.length} 个文件（临时关联，重启失效）` : '挂载失败')
  }

  return (
    <div className="nodal-side-body">
      <p className="text-xs font-semibold">挂载到当前画布</p>
      <p className="text-[10px] leading-4 text-[var(--nodal-muted)]">
        只有勾选的文件会参与下一次 RAG 召回；挂载是进程内临时关联，刷新/重启后需要重新勾选。
      </p>

      <label className="nodal-file-pick">
        <span className="flex items-center gap-1.5">
          <UploadIcon width={14} height={14} />
          上传新文件
        </span>
        <input
          type="file"
          accept=".txt,.md,.pdf,.csv,.json"
          disabled={files.uploading}
          onChange={(event) => void uploadFile(event, files.upload)}
        />
      </label>
      {files.uploading ? <p className="text-[11px] text-brand">解析、切片、向量化中…</p> : null}
      {files.error ? <p className="text-[11px] text-counter">{files.error.message}</p> : null}

      <div className="space-y-2">
        {files.files.map((file) => {
          const disabled = file.status !== 'ready'
          const checked = selected.includes(file.id)
          return (
            <label
              key={file.id}
              className={`nodal-knowledge-row ${checked ? 'nodal-knowledge-row-active' : ''} ${disabled ? 'opacity-50' : ''}`}
            >
              <input
                type="checkbox"
                checked={checked}
                disabled={disabled}
                onChange={() => toggle(file.id)}
                className="accent-brand"
              />
              <span className="min-w-0 flex-1">
                <span className="block truncate text-xs font-medium">{file.name}</span>
                <span className="block text-[10px] text-[var(--nodal-muted)]">
                  {file.status === 'ready' ? `${file.chunk_count} 个切片` : file.status} · {formatBytes(file.size)}
                </span>
              </span>
            </label>
          )
        })}
        {files.loading ? <p className="text-xs text-[var(--nodal-muted)]">加载中…</p> : null}
        {!files.loading && files.files.length === 0 ? (
          <p className="rounded-xl border border-dashed border-[var(--nodal-border)] p-3 text-xs leading-5 text-[var(--nodal-muted)]">
            还没有知识文件。上传 txt / md / pdf / csv / json 后，系统会同步解析并向量化。
          </p>
        ) : null}
      </div>

      <div className="flex items-center gap-2">
        <button type="button" className="nodal-btn nodal-btn-primary" disabled={mount.loading} onClick={() => void save()}>
          {mount.loading ? '保存中…' : '保存挂载'}
        </button>
        <button
          type="button"
          className="nodal-btn nodal-btn-ghost"
          onClick={() => void mount.unmount().then(() => setSelected([]))}
        >
          全部解除
        </button>
      </div>
      {notice ? <p className="text-[11px] text-evidence">{notice}</p> : null}
      {mount.error ? <p className="text-[11px] text-counter">{mount.error.message}</p> : null}
    </div>
  )
}

function TabButton({
  active,
  onClick,
  icon,
  label,
}: {
  active: boolean
  onClick: () => void
  icon: ReactNode
  label: string
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={active ? 'nodal-side-tab nodal-side-tab-active' : 'nodal-side-tab'}
    >
      {icon}
      {label}
    </button>
  )
}

function Stat({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-xl bg-[var(--nodal-surface-soft)] p-2 text-center">
      <p className="font-mono text-lg font-semibold leading-6">{value}</p>
      <p className="text-[10px] text-[var(--nodal-muted)]">{label}</p>
    </div>
  )
}

function MetaRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <span>{label}</span>
      <span className="truncate text-[var(--nodal-text)]">{value}</span>
    </div>
  )
}

function AuthGate({ onNeedLogin, what }: { onNeedLogin: () => void; what: string }) {
  return (
    <div className="rounded-xl border border-dashed border-[var(--nodal-border)] p-4 text-center">
      <p className="text-xs font-medium">登录后可用</p>
      <p className="mt-1 text-[11px] leading-5 text-[var(--nodal-muted)]">{what}需要注册账号。</p>
      <button type="button" className="nodal-btn nodal-btn-primary mt-3" onClick={onNeedLogin}>
        登录 / 注册
      </button>
    </div>
  )
}

async function uploadFile(
  event: ChangeEvent<HTMLInputElement>,
  upload: (file: File) => Promise<unknown>,
) {
  const file = event.target.files?.[0]
  if (!file) return
  await upload(file)
  event.target.value = ''
}

function formatDate(value: string): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', { hour12: false })
}

function formatBytes(size: number): string {
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / (1024 * 1024)).toFixed(1)} MB`
}
