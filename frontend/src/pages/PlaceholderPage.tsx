/**
 * 模块 8 只交付静态页面；画布页与分享页在模块 9/10 接入 ReactFlow。
 * 这里给一个可返回的占位页，避免从卡片点进去出现 404。
 */
import { useNavigate, useParams } from 'react-router-dom'
import { EmptyState } from '../components/EmptyState'
import { NodeIcon, SparkIcon } from '../components/icons'

export function CanvasPlaceholderPage() {
  const navigate = useNavigate()
  const { id, token } = useParams()
  const isShare = Boolean(token)
  return (
    <div className="mx-auto w-full max-w-4xl px-4 py-16 sm:px-6">
      <EmptyState
        icon={isShare ? <SparkIcon /> : <NodeIcon />}
        title={isShare ? '分享画布即将接入' : `画布 #${id ?? ''} 即将接入`}
        description="模块 8 交付首页、资源库与广场；ReactFlow 无限画布、节点编辑器与推演交互将在模块 9 实现。"
        action={
          <button type="button" onClick={() => navigate('/library')} className="rounded-full bg-brand px-4 py-2 text-xs font-semibold text-white hover:bg-brand-strong">
            返回资源库
          </button>
        }
      />
    </div>
  )
}

export function NotFoundPage() {
  const navigate = useNavigate()
  return (
    <div className="mx-auto w-full max-w-2xl px-4 py-20 sm:px-6">
      <EmptyState
        icon={<SparkIcon />}
        title="这个节点不存在"
        description="链接可能已失效，或者画布已经被删除。"
        action={
          <button type="button" onClick={() => navigate('/')} className="rounded-full bg-brand px-4 py-2 text-xs font-semibold text-white hover:bg-brand-strong">
            回到首页
          </button>
        }
      />
    </div>
  )
}
