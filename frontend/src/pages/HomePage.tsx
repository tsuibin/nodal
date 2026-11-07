/**
 * 首页：
 * - 品牌主张 + 主 CTA（直接开一张空白画布，未登录先领游客身份）；
 * - 关系类型/核心能力说明；
 * - 最新广场画布，验证广场接口并给新用户内容入口。
 */
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { CanvasCard } from '../components/CanvasCard'
import { EmptyState } from '../components/EmptyState'
import { useLoginModal } from '../components/LoginModalProvider'
import { ArrowRightIcon, BookIcon, BotIcon, NodeIcon, SparkIcon } from '../components/icons'
import { useAuth } from '../hooks/useAuth'
import { useSquareList } from '../hooks/useSquareList'
import { canvasApi } from '../services/canvas'
import { ApiError } from '../services/http'

const RELATIONS = [
  { name: '推导自', color: 'border-l-brand', dot: 'bg-brand', text: '从一个判断继续向前，得到更具体的结论。' },
  { name: '反例质疑', color: 'border-l-counter', dot: 'bg-counter', text: '主动寻找反例，让结论在压力下站稳。' },
  { name: '补充佐证', color: 'border-l-evidence', dot: 'bg-evidence', text: '引入证据、数据与引用，补齐论证链。' },
  { name: '并行假设', color: 'border-l-hypothesis', dot: 'bg-hypothesis', text: '同时保留多个可能，比较后再收敛。' },
]

const FEATURES = [
  { icon: <NodeIcon />, title: '无限画布', desc: '不被消息流截断，任何分支都能继续生长。' },
  { icon: <BotIcon />, title: '多 Agent 推演', desc: '选择节点与 Agent，AI 只新增节点，不改你的结构。' },
  { icon: <BookIcon />, title: '知识库挂载', desc: '上传资料，按画布临时挂载，推演时按需召回。' },
  { icon: <SparkIcon />, title: 'Fork 与广场', desc: '把一张画布分叉成你的版本，也可以发布给所有人。' },
]

export function HomePage() {
  const navigate = useNavigate()
  const { isAuthenticated, ensureGuest } = useAuth()
  const { openLogin } = useLoginModal()
  const [creating, setCreating] = useState(false)
  const square = useSquareList({ page_size: 3, sort: 'latest' })

  const startCanvas = async () => {
    if (creating) return
    setCreating(true)
    try {
      if (!isAuthenticated) await ensureGuest()
      const created = await canvasApi.create({
        title: '未命名画布',
        nodes: [],
        edges: [],
        viewport: { x: 0, y: 0, zoom: 1 },
      })
      navigate(`/canvas/${created.id}`)
    } catch (err) {
      if (err instanceof ApiError && (err.isUnauthorized || err.needsLogin)) openLogin()
    } finally {
      setCreating(false)
    }
  }

  return (
    <div>
      <section className="relative overflow-hidden border-b border-[var(--nodal-border)] nodal-grid-bg">
        <div className="pointer-events-none absolute -left-24 top-10 h-64 w-64 rounded-full bg-brand/10 blur-3xl" />
        <div className="pointer-events-none absolute -right-16 bottom-0 h-72 w-72 rounded-full bg-evidence/10 blur-3xl" />

        <div className="mx-auto grid w-full max-w-7xl items-center gap-10 px-4 py-16 sm:px-6 lg:grid-cols-[1.05fr_0.95fr] lg:px-8 lg:py-24">
          <div className="nodal-rise-in">
            <span className="inline-flex items-center gap-1.5 rounded-full border border-brand/20 bg-brand-soft px-3 py-1 text-[11px] font-semibold text-brand dark:bg-brand/10">
              <SparkIcon width={13} height={13} />
              无限画布 · AI 思维推演
            </span>

            <h1 className="mt-6 text-[40px] font-bold leading-[1.12] tracking-[-0.03em] text-[var(--nodal-text)] sm:text-5xl lg:text-[58px]">
              让思维，
              <br />
              长成<span className="text-brand">一张图。</span>
            </h1>
            <p className="mt-5 max-w-xl text-[15px] leading-7 text-[var(--nodal-muted)] sm:text-base">
              Nodal 把线性对话拆成节点与有向连线。你可以从任意一个判断继续推演、插入反例、补充证据，
              也可以让多个 Agent 在画布上并行辩论。
            </p>

            <div className="mt-7 flex flex-wrap items-center gap-3">
              <button
                type="button"
                onClick={() => void startCanvas()}
                disabled={creating}
                className="inline-flex items-center gap-2 rounded-full bg-brand px-6 py-3 text-sm font-semibold text-white shadow-[0_10px_28px_rgb(64_96_255/0.3)] transition hover:bg-brand-strong disabled:opacity-60"
              >
                {creating ? '正在创建…' : '开始推演'}
                <ArrowRightIcon width={16} height={16} />
              </button>
              <button
                type="button"
                onClick={() => navigate('/square')}
                className="inline-flex items-center gap-2 rounded-full border border-[var(--nodal-border)] bg-[var(--nodal-surface)] px-6 py-3 text-sm font-semibold text-[var(--nodal-text)] transition hover:border-brand/40 hover:text-brand"
              >
                逛逛广场
              </button>
            </div>

            <div className="mt-8 flex flex-wrap items-center gap-x-6 gap-y-2 text-[11px] text-[var(--nodal-muted)]">
              <span className="inline-flex items-center gap-1.5">
                <span className="h-1.5 w-1.5 rounded-full bg-brand" />4 种关系类型
              </span>
              <span className="inline-flex items-center gap-1.5">
                <span className="h-1.5 w-1.5 rounded-full bg-evidence" />AI 只新增，不改你的节点
              </span>
              <span className="inline-flex items-center gap-1.5">
                <span className="h-1.5 w-1.5 rounded-full bg-hypothesis" />游客可先体验
              </span>
            </div>
          </div>

          <HeroPreview />
        </div>
      </section>

      <section className="mx-auto w-full max-w-7xl px-4 py-14 sm:px-6 lg:px-8">
        <div className="mb-7 flex flex-wrap items-end justify-between gap-3">
          <div>
            <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-brand">Relations</p>
            <h2 className="mt-1.5 text-2xl font-bold tracking-tight text-[var(--nodal-text)]">四种关系，让论证有方向</h2>
          </div>
          <p className="max-w-md text-xs leading-5 text-[var(--nodal-muted)]">连线不是装饰：颜色和标签一起说明这段推理是支持、反驳，还是并行的另一种可能。</p>
        </div>

        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
          {RELATIONS.map((relation) => (
            <div
              key={relation.name}
              className={`nodal-card border-l-4 p-4 transition hover:-translate-y-0.5 ${relation.color}`}
            >
              <div className="flex items-center gap-2">
                <span className={`h-2 w-2 rounded-full ${relation.dot}`} />
                <p className="text-sm font-semibold text-[var(--nodal-text)]">{relation.name}</p>
              </div>
              <p className="mt-2 text-xs leading-5 text-[var(--nodal-muted)]">{relation.text}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="border-y border-[var(--nodal-border)] bg-[var(--nodal-surface)]/50">
        <div className="mx-auto w-full max-w-7xl px-4 py-14 sm:px-6 lg:px-8">
          <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-brand">Capabilities</p>
          <h2 className="mt-1.5 text-2xl font-bold tracking-tight text-[var(--nodal-text)]">为复杂思考准备的工作台</h2>
          <div className="mt-7 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {FEATURES.map((feature) => (
              <div key={feature.title} className="nodal-card p-5">
                <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-brand-soft text-brand dark:bg-brand/10">
                  {feature.icon}
                </span>
                <h3 className="mt-4 text-sm font-semibold text-[var(--nodal-text)]">{feature.title}</h3>
                <p className="mt-1.5 text-xs leading-5 text-[var(--nodal-muted)]">{feature.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="mx-auto w-full max-w-7xl px-4 py-14 sm:px-6 lg:px-8">
        <div className="mb-6 flex items-end justify-between gap-3">
          <div>
            <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-brand">Square</p>
            <h2 className="mt-1.5 text-2xl font-bold tracking-tight text-[var(--nodal-text)]">看看别人怎么想</h2>
          </div>
          <button type="button" onClick={() => navigate('/square')} className="text-xs font-semibold text-brand hover:underline">
            全部广场 →
          </button>
        </div>

        {square.loading && square.items.length === 0 ? (
          <div className="grid gap-4 sm:grid-cols-3">
            {Array.from({ length: 3 }).map((_, index) => (
              <div key={index} className="nodal-card h-56 animate-pulse bg-[var(--nodal-surface-soft)]" />
            ))}
          </div>
        ) : square.items.length === 0 ? (
          <EmptyState
            compact
            icon={<SparkIcon />}
            title="广场还没有公开画布"
            description="发布第一张画布，让更多人看到你的思考路径。"
          />
        ) : (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {square.items.map((item) => (
              <CanvasCard
                key={item.id}
                item={item}
                variant="square"
                onOpen={() => navigate(`/canvas/${item.id}`)}
              />
            ))}
          </div>
        )}
      </section>
    </div>
  )
}

/** 首页右侧的静态画布预览，用 SVG 表达节点、关系色和 AI 新增节点。 */
function HeroPreview() {
  return (
    <div className="relative mx-auto w-full max-w-[520px] nodal-rise-in lg:max-w-none">
      <div className="nodal-card overflow-hidden p-3">
        <div className="relative h-[330px] overflow-hidden rounded-xl border border-[var(--nodal-border)] bg-[var(--nodal-surface-soft)] nodal-grid-bg">
          <svg viewBox="0 0 520 330" className="absolute inset-0 h-full w-full" role="img" aria-label="思维画布示意图">
            <path d="M128 72 C 190 72, 210 108, 258 128" stroke="#4060FF" strokeWidth="2" fill="none" />
            <path d="M128 72 C 190 72, 208 196, 250 220" stroke="#ff6b6b" strokeWidth="2" fill="none" />
            <path d="M292 142 C 336 142, 344 70, 396 62" stroke="#36c9a5" strokeWidth="2" fill="none" />
            <path d="M292 230 C 338 230, 342 292, 398 292" stroke="#ffc857" strokeWidth="2" fill="none" />

            <NodeBox x={22} y={42} title="核心判断" body="AI 会改变知识工作的协作方式" tone="brand" />
            <NodeBox x={250} y={110} title="推导自" body="重复性整理会被压缩" tone="brand" />
            <NodeBox x={250} y={202} title="反例质疑" body="高责任场景仍需人类签字" tone="counter" />
            <NodeBox x={392} y={40} title="补充佐证" body="团队采用 AI 后交付周期下降" tone="evidence" />
            <NodeBox x={394} y={270} title="并行假设" body="新岗位：思维编排者" tone="hypothesis" />
          </svg>

          <div className="absolute left-3 top-3 rounded-full bg-[var(--nodal-surface)]/90 px-2.5 py-1 text-[10px] font-medium text-[var(--nodal-muted)] backdrop-blur">
            推演中 · Agent「反方」已加入
          </div>
          <div className="absolute bottom-3 right-3 flex items-center gap-1.5 rounded-full bg-[var(--nodal-surface)]/90 px-2.5 py-1 text-[10px] text-[var(--nodal-muted)] backdrop-blur">
            <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-evidence" />
            AI 只新增节点
          </div>
        </div>
      </div>
      <div className="absolute -bottom-3 -left-3 hidden rounded-2xl border border-[var(--nodal-border)] bg-[var(--nodal-surface)] px-3 py-2 shadow-lg sm:block">
        <p className="text-[10px] text-[var(--nodal-muted)]">关系颜色</p>
        <div className="mt-1 flex gap-2 text-[10px]">
          <span className="text-brand">● 推导</span>
          <span className="text-counter">● 反例</span>
          <span className="text-evidence">● 佐证</span>
          <span className="text-[#9a6b00]">● 假设</span>
        </div>
      </div>
    </div>
  )
}

interface NodeBoxProps {
  x: number
  y: number
  title: string
  body: string
  tone: 'brand' | 'counter' | 'evidence' | 'hypothesis'
}

const TONE_STYLE: Record<NodeBoxProps['tone'], { border: string; fill: string; text: string }> = {
  brand: { border: '#4060FF', fill: '#EEF1FF', text: '#2F4BE0' },
  counter: { border: '#ff6b6b', fill: '#FFF0F0', text: '#D64545' },
  evidence: { border: '#36c9a5', fill: '#E9FBF6', text: '#16876D' },
  hypothesis: { border: '#ffc857', fill: '#FFF8E6', text: '#9A6B00' },
}

function NodeBox({ x, y, title, body, tone }: NodeBoxProps) {
  const style = TONE_STYLE[tone]
  return (
    <g>
      <rect x={x} y={y} width="138" height="62" rx="9" fill={style.fill} stroke={style.border} strokeWidth="1.2" />
      <rect x={x} y={y} width="4" height="62" rx="2" fill={style.border} />
      <text x={x + 14} y={y + 20} fontSize="10" fontWeight="700" fill={style.text}>
        {title}
      </text>
      <foreignObject x={x + 12} y={y + 27} width="116" height="30">
        <div style={{ fontSize: '9px', lineHeight: '13px', color: '#4b5565' }}>{body}</div>
      </foreignObject>
    </g>
  )
}
