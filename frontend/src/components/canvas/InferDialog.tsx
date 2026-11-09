/**
 * 推演弹窗：选 Agent → 发起 SSE 流式推演。
 *
 * 流本身由页面持有（关闭弹窗后仍要继续增量渲染），弹窗只负责选择与状态展示。
 */
import { useEffect, useState } from 'react'
import type { AgentTemplate, ModelCatalogItem } from '../../types/agent'
import type { ApiError } from '../../services/http'
import type { InferDonePayload } from '../../types/infer'
import { BotIcon, CloseIcon, PlayIcon } from '../icons'

export interface InferDialogProps {
  open: boolean
  selectedCount: number
  agents: AgentTemplate[]
  loadingAgents: boolean
  models: ModelCatalogItem[]
  loadingModels: boolean
  running: boolean
  error: ApiError | Error | null
  done: InferDonePayload | null
  onClose: () => void
  onRun: (agentId: number, modelId: string) => void
  onNeedLogin: () => void
  onNeedAgent: () => void
}

export function InferDialog({
  open,
  selectedCount,
  agents,
  loadingAgents,
  models,
  loadingModels,
  running,
  error,
  done,
  onClose,
  onRun,
  onNeedLogin,
  onNeedAgent,
}: InferDialogProps) {
  const [agentID, setAgentID] = useState<number | null>(null)
  const [modelID, setModelID] = useState<string>('')

  useEffect(() => {
    if (!open) return
    if (agentID === null && agents[0]) setAgentID(agents[0].id)
    if (agentID !== null && !agents.some((agent) => agent.id === agentID)) {
      setAgentID(agents[0]?.id ?? null)
    }
  }, [agents, agentID, open])

  // 切换 Agent 时，模型默认回到该 Agent 模板绑定的模型。
  useEffect(() => {
    const agent = agents.find((item) => item.id === agentID)
    if (agent) setModelID(agent.model_id)
    // 只在 Agent 变化时重置；不要覆盖用户手动选择的模型
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [agentID])

  // 模型目录加载完成或当前模型失效时，回落到第一个已配置 Key 的模型。
  useEffect(() => {
    if (models.length === 0) return
    if (modelID && models.some((model) => model.id === modelID)) return
    const configured = models.find((model) => model.configured)
    setModelID(configured?.id ?? models[0]?.id ?? '')
  }, [models, modelID])

  if (!open) return null

  const selected = agents.find((agent) => agent.id === agentID) ?? null

  return (
    <div className="nodal-dialog-backdrop" role="presentation" onMouseDown={running ? undefined : onClose}>
      <div
        className="nodal-dialog max-w-xl"
        role="dialog"
        aria-modal="true"
        aria-label="AI 推演"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <header className="nodal-dialog-head">
          <div>
            <p className="nodal-kicker">INFERENCE</p>
            <h2 className="nodal-dialog-title">让 Agent 接着你的节点往下推</h2>
          </div>
          <button type="button" className="nodal-icon-btn" onClick={onClose} aria-label="关闭">
            <CloseIcon />
          </button>
        </header>

        <div className="space-y-4 p-5">
          <div className="nodal-infer-summary">
            <span className="nodal-infer-count">{selectedCount}</span>
            <div>
              <p className="text-sm font-semibold">已选中 {selectedCount} 个节点</p>
              <p className="text-[11px] text-[var(--nodal-muted)]">
                AI 只会新增节点与连线，不会删除或修改你已有的内容；服务端会强制校验。
              </p>
            </div>
          </div>

          {agents.length === 0 ? (
            <div className="rounded-xl border border-dashed border-[var(--nodal-border)] p-4 text-center">
              <BotIcon width={22} height={22} className="mx-auto text-brand" />
              <p className="mt-2 text-sm font-medium">还没有 Agent 模板</p>
              <p className="mt-1 text-[11px] text-[var(--nodal-muted)]">
                先去资源库创建一个 Agent，填好 system prompt 和模型，再回来推演。
              </p>
              <button type="button" className="nodal-btn nodal-btn-primary mt-3" onClick={onNeedAgent}>
                去创建 Agent
              </button>
            </div>
          ) : (
            <label className="nodal-field">
              <span>选择 Agent{loadingAgents ? '（加载中…）' : ''}</span>
              <select
                className="nodal-select w-full"
                value={agentID ?? ''}
                disabled={running || loadingAgents}
                onChange={(event) => setAgentID(Number(event.target.value))}
              >
                {agents.map((agent) => (
                  <option key={agent.id} value={agent.id}>
                    {agent.name} · {agent.model_label || agent.model_id}
                  </option>
                ))}
              </select>
            </label>
          )}

          {models.length > 0 ? (
            <label className="nodal-field">
              <span>选择模型{loadingModels ? '（加载中…）' : ''}</span>
              <select
                className="nodal-select w-full"
                value={modelID}
                disabled={running || loadingModels}
                onChange={(event) => setModelID(event.target.value)}
              >
                {models.map((model) => (
                  <option key={model.id} value={model.id} disabled={model.requires_key && !model.configured}>
                    {model.label}
                    {model.requires_key && !model.configured ? '（未配置 Key）' : ''}
                  </option>
                ))}
              </select>
              {models.every((model) => model.requires_key && !model.configured) ? (
                <span className="text-[11px] text-[var(--nodal-muted)]">
                  还没有可用模型。先到模型设置里保存至少一个 API Key，或使用本地模拟模型。
                </span>
              ) : null}
            </label>
          ) : loadingModels ? (
            <p className="text-[11px] text-[var(--nodal-muted)]">正在加载模型目录…</p>
          ) : null}

          {selected ? (
            <div className="rounded-xl bg-[var(--nodal-surface-soft)] p-3">
              <p className="text-[11px] font-semibold text-[var(--nodal-muted)]">SYSTEM PROMPT</p>
              <p className="mt-1 line-clamp-3 text-xs leading-5">{selected.system_prompt || '未填写'}</p>
            </div>
          ) : null}

          {running ? (
            <div className="flex items-center gap-2 rounded-xl border border-brand/25 bg-brand/8 px-3 py-2 text-xs text-brand">
              <span className="nodal-pulse-dot" />
              正在流式生成，新的节点会直接出现在画布上…
            </div>
          ) : null}

          {done ? (
            <div className="rounded-xl border border-evidence/30 bg-evidence/8 px-3 py-2 text-xs text-[#1d9c7d] dark:text-[#6fe0c1]">
              完成：新增 {done.node_count} 个节点、{done.edge_count} 条连线
              {done.skipped ? `，跳过 ${done.skipped} 条非法输出` : ''}。
            </div>
          ) : null}

          {error ? (
            <div className="rounded-xl border border-counter/30 bg-counter/8 px-3 py-2 text-xs text-counter">
              {error.message}
              {isLoginError(error) ? (
                <button type="button" className="ml-2 underline" onClick={onNeedLogin}>
                  去登录
                </button>
              ) : null}
            </div>
          ) : null}
        </div>

        <footer className="nodal-dialog-foot">
          <span className="text-[11px] text-[var(--nodal-muted)]">
            推演过程可关闭弹窗，画布会继续增量渲染。
          </span>
          <div className="ml-auto flex gap-2">
            <button type="button" className="nodal-btn nodal-btn-ghost" onClick={onClose}>
              {running ? '后台继续' : '取消'}
            </button>
            <button
              type="button"
              className="nodal-btn nodal-btn-primary"
              disabled={running || !agentID || !modelID || selectedCount === 0}
              onClick={() => agentID && modelID && onRun(agentID, modelID)}
            >
              <PlayIcon width={15} height={15} />
              {running ? '推演中…' : '开始推演'}
            </button>
          </div>
        </footer>
      </div>
    </div>
  )
}

function isLoginError(error: ApiError | Error): boolean {
  return 'code' in error && (error as ApiError).needsLogin
}
