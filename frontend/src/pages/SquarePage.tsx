/**
 * 公共广场：分页、标签筛选、关键词搜索、最新/最热排序、点赞。
 * 列表数据来自 GET /api/square/list（公开接口，可选身份），点赞即时更新本地状态。
 */
import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { CanvasCard } from '../components/CanvasCard'
import { EmptyState } from '../components/EmptyState'
import { LoadingGrid } from '../components/LoadingGrid'
import { SearchIcon, SparkIcon } from '../components/icons'
import { TagChip } from '../components/TagChip'
import { useSquareList } from '../hooks/useSquareList'
import { squareApi } from '../services/square'

export function SquarePage() {
  const navigate = useNavigate()
  const square = useSquareList({ immediate: true, sort: 'latest' })
  const [keyword, setKeyword] = useState('')
  const [activeTag, setActiveTag] = useState<string | undefined>()
  const [sort, setSort] = useState<'latest' | 'hot'>('latest')
  const [liking, setLiking] = useState<number | null>(null)
  const [likeState, setLikeState] = useState<Record<number, { liked: boolean; count: number }>>({})
  const [notice, setNotice] = useState<string | null>(null)

  // 关键词防抖：避免每敲一个字都请求
  useEffect(() => {
    const timer = window.setTimeout(() => {
      square.setFilters({ keyword: keyword.trim() || undefined })
    }, 320)
    return () => window.clearTimeout(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [keyword])

  const tags = useMemo(() => {
    const set = new Set<string>()
    square.items.forEach((item) => item.tags.forEach((tag) => set.add(tag)))
    if (activeTag) set.add(activeTag)
    return Array.from(set).slice(0, 10)
  }, [square.items, activeTag])

  const selectTag = (tag?: string) => {
    setActiveTag(tag)
    square.setFilters({ tag })
  }

  const like = async (id: number) => {
    if (liking === id) return
    setLiking(id)
    setNotice(null)
    try {
      const result = await squareApi.like(id)
      setLikeState((prev) => ({ ...prev, [id]: { liked: result.liked, count: result.like_count } }))
    } catch {
      setNotice('点赞失败，请稍后重试')
    } finally {
      setLiking(null)
    }
  }

  return (
    <div className="mx-auto w-full max-w-7xl px-4 py-8 sm:px-6 lg:px-8 lg:py-10">
      <div className="mb-6">
        <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-brand">Square</p>
        <h1 className="mt-1.5 text-2xl font-bold tracking-tight text-[var(--nodal-text)] sm:text-3xl">公共广场</h1>
        <p className="mt-2 max-w-2xl text-xs leading-5 text-[var(--nodal-muted)]">
          公开的思维图谱。读一条别人的论证路径，Fork 成自己的版本继续推演；也可以把你的画布发布到这里。
        </p>
      </div>

      <div className="mb-5 flex flex-col gap-3 rounded-2xl border border-[var(--nodal-border)] bg-[var(--nodal-surface)] p-3 sm:flex-row sm:items-center">
        <label className="flex flex-1 items-center gap-2 rounded-full border border-[var(--nodal-border)] bg-[var(--nodal-bg)] px-3 py-2">
          <SearchIcon width={15} height={15} className="text-[var(--nodal-muted)]" />
          <input
            value={keyword}
            onChange={(event) => setKeyword(event.target.value)}
            placeholder="搜索标题、描述或标签…"
            className="w-full bg-transparent text-xs text-[var(--nodal-text)] outline-none placeholder:text-[var(--nodal-muted)]"
          />
        </label>
        <div className="flex items-center gap-1 rounded-full bg-[var(--nodal-surface-soft)] p-1">
          {(
            [
              { key: 'latest' as const, label: '最新' },
              { key: 'hot' as const, label: '最热' },
            ]
          ).map((item) => (
            <button
              key={item.key}
              type="button"
              onClick={() => {
                setSort(item.key)
                square.setFilters({ sort: item.key })
              }}
              className={`rounded-full px-3 py-1.5 text-[11px] font-semibold transition ${
                sort === item.key
                  ? 'bg-[var(--nodal-surface)] text-brand shadow-sm'
                  : 'text-[var(--nodal-muted)] hover:text-[var(--nodal-text)]'
              }`}
            >
              {item.label}
            </button>
          ))}
        </div>
      </div>

      {tags.length > 0 && (
        <div className="mb-5 flex flex-wrap items-center gap-1.5">
          <TagChip label="全部" tone={activeTag ? 'neutral' : 'brand'} onClick={() => selectTag(undefined)} />
          {tags.map((tag) => (
            <TagChip key={tag} label={tag} tone={activeTag === tag ? 'brand' : 'neutral'} onClick={() => selectTag(tag)} />
          ))}
        </div>
      )}

      {notice && <div className="mb-4 rounded-xl bg-counter-soft px-3 py-2 text-xs text-counter dark:bg-counter/10">{notice}</div>}

      {square.loading && square.items.length === 0 ? (
        <LoadingGrid />
      ) : square.error && square.items.length === 0 ? (
        <EmptyState title="广场加载失败" description="请检查后端服务是否启动，或稍后刷新重试。" />
      ) : square.items.length === 0 ? (
        <EmptyState
          icon={<SparkIcon />}
          title={activeTag || keyword ? '没有匹配的画布' : '广场还没有内容'}
          description={activeTag || keyword ? '换一个关键词或标签试试。' : '发布一张公开画布，成为第一个分享思考路径的人。'}
        />
      ) : (
        <>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
            {square.items.map((item) => {
              const override = likeState[item.id]
              const display = override ? { ...item, liked: override.liked, like_count: override.count } : item
              return (
                <CanvasCard
                  key={item.id}
                  item={display}
                  variant="square"
                  onOpen={() => navigate(`/canvas/${item.id}`)}
                  onLike={() => void like(item.id)}
                />
              )
            })}
          </div>

          {square.hasMore && (
            <div className="mt-7 flex justify-center">
              <button
                type="button"
                onClick={() => void square.loadMore()}
                disabled={square.loading}
                className="rounded-full border border-[var(--nodal-border)] px-5 py-2 text-xs font-medium text-[var(--nodal-muted)] transition hover:border-brand/40 hover:text-brand disabled:opacity-60"
              >
                {square.loading ? '加载中…' : '加载更多'}
              </button>
            </div>
          )}
        </>
      )}
    </div>
  )
}
