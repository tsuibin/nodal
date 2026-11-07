/**
 * 资源库：
 * - 我的画布：游客也可用，列表 + 搜索 + 新建 + 删除；
 * - 知识库：仅注册用户，上传/删除；
 * - Agent 模板：仅注册用户，展示可用 Agent 与模型。
 *
 * 三个 Tab 的数据按需加载，避免进入页面就把所有接口打一遍。
 */
import { useEffect, useRef, useState, type ChangeEvent } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { CanvasCard } from '../components/CanvasCard'
import { EmptyState } from '../components/EmptyState'
import { LoadingGrid } from '../components/LoadingGrid'
import { useLoginModal } from '../components/LoginModalProvider'
import { BookIcon, BotIcon, NodeIcon, PlusIcon, SearchIcon } from '../components/icons'
import { useAgentList } from '../hooks/useAgentList'
import { useAuth } from '../hooks/useAuth'
import { useCanvasList } from '../hooks/useCanvasList'
import { useKnowledgeFiles } from '../hooks/useKnowledge'
import { formatRelativeTime } from '../lib/format'
import { demoApi } from '../services/demo'
import { ApiError } from '../services/http'
import { useToast } from '../components/ToastProvider'

type TabKey = 'canvas' | 'knowledge' | 'agent'

const TABS: Array<{ key: TabKey; label: string; icon: typeof NodeIcon }> = [
  { key: 'canvas', label: '我的画布', icon: NodeIcon },
  { key: 'knowledge', label: '知识库', icon: BookIcon },
  { key: 'agent', label: 'Agent 模板', icon: BotIcon },
]

function readableError(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.needsLogin) return '该功能需要登录后使用'
    if (err.code === 4002) return '文件类型不支持，请上传 PDF / Markdown / TXT / DOCX'
    if (err.code === 4003) return '文件解析失败，请检查内容或换一个文件'
    if (err.code === 4005) return '向量化失败，请检查模型配置后重试'
    return err.message || '操作失败'
  }
  return err instanceof Error ? err.message : '网络异常'
}

export function LibraryPage() {
  const navigate = useNavigate()
  const [params, setParams] = useSearchParams()
  const { isAuthenticated, status } = useAuth()
  const { openLogin } = useLoginModal()
  const [tab, setTab] = useState<TabKey>('canvas')
  const [keyword, setKeyword] = useState('')
  const [notice, setNotice] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)
  const autoCreateRef = useRef(false)
  const demoSeedRef = useRef(false)

  const canvases = useCanvasList()
  const knowledge = useKnowledgeFiles({ immediate: false })
  const agents = useAgentList(false)
  const toast = useToast()

  // 顶部「新建画布」通过 ?new=1 进来，只自动创建一次
  useEffect(() => {
    if (params.get('new') !== '1' || autoCreateRef.current || status === 'loading') return
    autoCreateRef.current = true
    void (async () => {
      const created = await canvases.create('未命名画布')
      setParams({}, { replace: true })
      if (created) navigate(`/canvas/${created.id}`)
    })()
  }, [params, setParams, status, canvases, navigate])

  // 新用户兜底：注册时后端已经种了 Demo；老账号/游客第一次打开资源库且 0 张画布时补种。
  // 只在无关键词、首屏加载完成后触发，避免把「搜索无结果」误判为空账号。
  useEffect(() => {
    if (status === 'loading' || status === 'anonymous') return
    if (!canvases.loaded || canvases.loading || demoSeedRef.current) return
    if (keyword.trim() || canvases.total !== 0) return
    demoSeedRef.current = true
    void demoApi
      .seed()
      .then((result) => {
        if (result.created > 0) {
          toast.success(`已为你生成 ${result.created} 张 Demo 画布`)
          void canvases.refresh()
        }
      })
      .catch(() => {
        // 失败由全局接口错误 toast 兜底，这里不重复提示
      })
  }, [canvases.loaded, canvases.loading, canvases.total, canvases.refresh, keyword, status, toast])

  // 画布列表：关键词防抖后重新拉取
  useEffect(() => {
    const timer = window.setTimeout(() => {
      canvases.setFilters({ keyword: keyword.trim() || undefined })
    }, 280)
    return () => window.clearTimeout(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [keyword])

  // 知识库 / Agent 需要登录；切换 Tab 时按需加载
  useEffect(() => {
    if (status === 'loading') return
    if (!isAuthenticated) return
    if (tab === 'knowledge' && knowledge.files.length === 0) void knowledge.refresh()
    if (tab === 'agent' && agents.items.length === 0) void agents.refresh()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tab, isAuthenticated, status])

  const createCanvas = async () => {
    if (creating) return
    setCreating(true)
    setNotice(null)
    try {
      const created = await canvases.create('未命名画布')
      if (created) {
        toast.success('画布已创建')
        navigate(`/canvas/${created.id}`)
      } else setNotice(readableError(canvases.error))
    } finally {
      setCreating(false)
    }
  }

  const switchTab = (next: TabKey) => {
    setTab(next)
    setNotice(null)
    if (next !== 'canvas' && !isAuthenticated) openLogin()
  }

  const removeCanvas = async (id: number) => {
    if (!window.confirm('删除后不可恢复，确认删除这张画布？')) return
    const removed = await canvases.remove(id)
    if (removed) toast.success('画布已删除')
    else setNotice(readableError(canvases.error))
  }

  const uploadFile = async (event: ChangeEvent<HTMLInputElement>) => {

    const file = event.target.files?.[0]
    event.target.value = ''
    if (!file) return
    setNotice(null)
    const created = await knowledge.upload(file)
    if (!created) setNotice(readableError(knowledge.error))
  }

  return (
    <div className="mx-auto w-full max-w-7xl px-4 py-8 sm:px-6 lg:px-8 lg:py-10">
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-brand">Library</p>
          <h1 className="mt-1.5 text-2xl font-bold tracking-tight text-[var(--nodal-text)] sm:text-3xl">你的思考资产</h1>
          <p className="mt-2 text-xs leading-5 text-[var(--nodal-muted)]">画布、知识库与 Agent 模板集中在这里；游客画布在登录后仍会保留当前会话。</p>
        </div>
        {tab === 'canvas' && (
          <button
            type="button"
            onClick={() => void createCanvas()}
            disabled={creating}
            className="inline-flex items-center justify-center gap-1.5 self-start rounded-full bg-brand px-4 py-2.5 text-xs font-semibold text-white shadow-[0_8px_20px_rgb(64_96_255/0.25)] transition hover:bg-brand-strong disabled:opacity-60"
          >
            <PlusIcon width={15} height={15} />
            {creating ? '创建中…' : '新建画布'}
          </button>
        )}
      </div>

      <div className="mb-5 flex flex-wrap items-center gap-2 border-b border-[var(--nodal-border)] pb-3">
        {TABS.map((item) => {
          const Icon = item.icon
          const active = tab === item.key
          return (
            <button
              key={item.key}
              type="button"
              onClick={() => switchTab(item.key)}
              className={`inline-flex items-center gap-1.5 rounded-full px-3.5 py-2 text-[12px] font-semibold transition ${
                active ? 'bg-brand-soft text-brand dark:bg-brand/12' : 'text-[var(--nodal-muted)] hover:text-[var(--nodal-text)]'
              }`}
            >
              <Icon width={15} height={15} />
              {item.label}
            </button>
          )
        })}

        {tab === 'canvas' && (
          <label className="ml-auto flex w-full items-center gap-1.5 rounded-full border border-[var(--nodal-border)] bg-[var(--nodal-surface)] px-3 py-1.5 sm:w-64">
            <SearchIcon width={14} height={14} className="text-[var(--nodal-muted)]" />
            <input
              value={keyword}
              onChange={(event) => setKeyword(event.target.value)}
              placeholder="搜索我的画布…"
              className="w-full bg-transparent text-xs text-[var(--nodal-text)] outline-none placeholder:text-[var(--nodal-muted)]"
            />
          </label>
        )}
      </div>

      {notice && (
        <div className="mb-4 rounded-xl border border-counter/25 bg-counter-soft px-3.5 py-2.5 text-xs text-counter dark:bg-counter/10">
          {notice}
        </div>
      )}

      {tab === 'canvas' && (
        <CanvasTab
          items={canvases.items}
          loading={canvases.loading}
          error={canvases.error}
          hasMore={canvases.hasMore}
          onLoadMore={() => void canvases.loadMore()}
          onRetry={() => void canvases.refresh()}
          onOpen={(id) => navigate(`/canvas/${id}`)}
          onCreate={() => void createCanvas()}
          onDelete={(id) => void removeCanvas(id)}
        />
      )}

      {tab === 'knowledge' && (
        <KnowledgeTab
          authenticated={isAuthenticated}
          files={knowledge.files}
          loading={knowledge.loading}
          uploading={knowledge.uploading}
          error={knowledge.error}
          onUpload={uploadFile}
          onDelete={(id) => {
            if (window.confirm('删除文件会同时删除切片与向量，确认？')) void knowledge.remove(id)
          }}
        />
      )}

      {tab === 'agent' && (
        <AgentTab authenticated={isAuthenticated} items={agents.items} loading={agents.loading} error={agents.error} />
      )}
    </div>
  )
}

function CanvasTab(props: {
  items: ReturnType<typeof useCanvasList>['items']
  loading: boolean
  error: Error | null
  hasMore: boolean
  onLoadMore: () => void
  onRetry: () => void
  onOpen: (id: number) => void
  onCreate: () => void
  onDelete: (id: number) => void
}) {
  const { items, loading, error, hasMore, onLoadMore, onRetry, onOpen, onCreate, onDelete } = props
  if (loading && items.length === 0) return <LoadingGrid />
  if (error && items.length === 0) return <EmptyState title="画布加载失败" description={readableError(error)} action={<RetryButton onClick={onRetry} />} />
  if (items.length === 0) {
    return (
      <EmptyState
        icon={<NodeIcon />}
        title="还没有画布"
        description="从一张空白画布开始：先写下核心判断，再用连线展开推导、反例与佐证。"
        action={
          <button type="button" onClick={onCreate} className="rounded-full bg-brand px-4 py-2 text-xs font-semibold text-white hover:bg-brand-strong">
            创建第一张画布
          </button>
        }
      />
    )
  }
  return (
    <>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
        {items.map((item) => (
          <CanvasCard key={item.id} item={item} variant="mine" onOpen={() => onOpen(item.id)} onDelete={() => onDelete(item.id)} />
        ))}
      </div>
      {hasMore && (
        <div className="mt-6 flex justify-center">
          <button type="button" onClick={onLoadMore} disabled={loading} className="rounded-full border border-[var(--nodal-border)] px-4 py-2 text-xs text-[var(--nodal-muted)] hover:text-brand disabled:opacity-60">
            {loading ? '加载中…' : '加载更多'}
          </button>
        </div>
      )}
    </>
  )
}

function KnowledgeTab(props: {
  authenticated: boolean
  files: ReturnType<typeof useKnowledgeFiles>['files']
  loading: boolean
  uploading: boolean
  error: Error | null
  onUpload: (event: ChangeEvent<HTMLInputElement>) => void
  onDelete: (id: number) => void
}) {
  const { authenticated, files, loading, uploading, error, onUpload, onDelete } = props
  if (!authenticated) return <LoginGate feature="知识库" description="登录后可上传资料，推演时按需召回片段。" />
  return (
    <div>
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-dashed border-brand/30 bg-brand-soft/60 px-4 py-3.5 dark:bg-brand/8">
        <div>
          <p className="text-sm font-semibold text-[var(--nodal-text)]">上传知识文件</p>
          <p className="mt-1 text-[11px] text-[var(--nodal-muted)]">支持 PDF / Markdown / TXT / DOCX，上传后同步解析、切片并向量化。</p>
        </div>
        <label className="inline-flex cursor-pointer items-center gap-1.5 rounded-full bg-brand px-4 py-2 text-xs font-semibold text-white transition hover:bg-brand-strong">
          <PlusIcon width={14} height={14} />
          {uploading ? '处理中…' : '选择文件'}
          <input type="file" accept=".pdf,.md,.markdown,.txt,.docx" className="hidden" onChange={onUpload} disabled={uploading} />
        </label>
      </div>

      {error && <div className="mb-4 rounded-xl bg-counter-soft px-3 py-2 text-xs text-counter dark:bg-counter/10">{readableError(error)}</div>}
      {loading && files.length === 0 ? (
        <LoadingGrid count={3} />
      ) : files.length === 0 ? (
        <EmptyState icon={<BookIcon />} title="知识库还是空的" description="上传一份资料，画布挂载后推演时会自动召回相关片段。" />
      ) : (
        <div className="space-y-2.5">
          {files.map((file) => (
            <div key={file.id} className="nodal-card flex flex-wrap items-center gap-3 px-4 py-3">
              <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-brand-soft text-[11px] font-bold uppercase text-brand dark:bg-brand/10">
                {file.ext.replace('.', '') || 'FILE'}
              </span>
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium text-[var(--nodal-text)]">{file.name}</p>
                <p className="mt-0.5 text-[11px] text-[var(--nodal-muted)]">
                  {(file.size / 1024).toFixed(0)} KB · {file.chunk_count} 个切片 · {formatRelativeTime(file.created_at)}
                </p>
              </div>
              <span
                className={`rounded-full px-2 py-0.5 text-[10px] ${
                  file.status === 'ready'
                    ? 'bg-evidence-soft text-evidence dark:bg-evidence/10'
                    : file.status === 'failed'
                      ? 'bg-counter-soft text-counter dark:bg-counter/10'
                      : 'bg-hypothesis-soft text-[#9a6b00]'
                }`}
              >
                {file.status === 'ready' ? '已就绪' : file.status === 'failed' ? '失败' : '处理中'}
              </span>
              <button type="button" onClick={() => onDelete(file.id)} className="text-[11px] text-[var(--nodal-muted)] hover:text-counter">
                删除
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

function AgentTab(props: {
  authenticated: boolean
  items: ReturnType<typeof useAgentList>['items']
  loading: boolean
  error: Error | null
}) {
  const { authenticated, items, loading, error } = props
  if (!authenticated) return <LoginGate feature="Agent 模板" description="登录后可创建自己的 Agent、绑定模型与系统提示词。" />
  if (loading && items.length === 0) return <LoadingGrid count={3} />
  if (error && items.length === 0) return <EmptyState title="Agent 加载失败" description={readableError(error)} />
  if (items.length === 0) {
    return <EmptyState icon={<BotIcon />} title="还没有 Agent 模板" description="在画布页创建 Agent：给它一个角色、系统提示词和模型，就能参与推演。" />
  }
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
      {items.map((agent) => (
        <div key={agent.id} className="nodal-card p-5">
          <div className="flex items-start justify-between gap-2">
            <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-brand-soft text-brand dark:bg-brand/10">
              <BotIcon />
            </span>
            <span className="rounded-full border border-[var(--nodal-border)] px-2 py-0.5 text-[10px] text-[var(--nodal-muted)]">{agent.model_label}</span>
          </div>
          <h3 className="mt-3 text-sm font-semibold text-[var(--nodal-text)]">{agent.name}</h3>
          <p className="mt-1 line-clamp-2 text-xs leading-5 text-[var(--nodal-muted)]">{agent.description || '未填写描述'}</p>
          <p className="mt-3 line-clamp-2 rounded-lg bg-[var(--nodal-surface-soft)] p-2 font-mono text-[10px] leading-4 text-[var(--nodal-muted)]">{agent.system_prompt}</p>
        </div>
      ))}
    </div>
  )
}

function LoginGate({ feature, description }: { feature: string; description: string }) {
  const { openLogin } = useLoginModal()
  return (
    <EmptyState
      icon={<BotIcon />}
      title={`登录后使用${feature}`}
      description={description}
      action={
        <button type="button" onClick={openLogin} className="rounded-full bg-brand px-4 py-2 text-xs font-semibold text-white hover:bg-brand-strong">
          登录 / 注册
        </button>
      }
    />
  )
}

function RetryButton({ onClick }: { onClick: () => void }) {
  return (
    <button type="button" onClick={onClick} className="rounded-full border border-[var(--nodal-border)] px-4 py-2 text-xs text-[var(--nodal-muted)] hover:text-brand">
      重新加载
    </button>
  )
}
