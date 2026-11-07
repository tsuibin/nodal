/**
 * 推演（run-infer）相关的类型。
 *
 * React Flow 节点/边用结构化类型而非直接依赖 @xyflow/react，
 * 这样本包在没有安装 react-flow 的环境里也能通过类型检查；
 * 接入真实 React Flow 时可直接把 RFNode 赋给 Node，字段是超集。
 */

export type RFNode = {
  id: string
  position: { x: number; y: number }
  type?: string
  data?: Record<string, unknown>
} & Record<string, unknown>

export type RFEdge = {
  id: string
  source: string
  target: string
  type?: string
} & Record<string, unknown>

/** POST /api/canvas/:id/run-infer 请求体 */
export interface RunInferPayload {
  /** 画布上被选中的节点 id（1-50 个） */
  selected_node_ids: string[]
  agent_id: number
}

/** SSE event: start */
export interface InferStartEvent {
  run_id: string
  canvas_id: number
  canvas_title: string
  agent_id: number
  agent_name: string
  model_id: string
  model_label: string
}

/** SSE event: node —— 前端直接 [...nodes, node] 增量渲染 */
export interface InferNodeEvent {
  node: RFNode
  index: number
}

/** SSE event: edge —— 前端直接 [...edges, edge] 增量渲染 */
export interface InferEdgeEvent {
  edge: RFEdge
  index: number
}

/** SSE event: sources —— 本次推演使用的知识库片段（画布已挂载的文件） */
export interface InferSourceItem {
  file_id: number
  file_name: string
  chunk_index: number
  score: number
  preview: string
}

export interface InferSourcesEvent {
  /** 用于检索的查询串（由 Agent 名 + 选中节点标题组成） */
  query: string
  items: InferSourceItem[]
  /** 检索失败时的降级提示；为空表示正常 */
  warning: string
}

/** SSE event: warning —— 模型输出了被服务端忽略的内容（如 delete/update） */
export interface InferWarningEvent {
  message: string
  skipped: number
}

/** SSE event: done */
export interface InferDonePayload {
  run_id: string
  node_count: number
  edge_count: number
  skipped: number
}

export interface InferStreamHandlers {
  onStart?: (payload: InferStartEvent) => void
  onSources?: (payload: InferSourcesEvent) => void
  onNode?: (payload: InferNodeEvent) => void
  onEdge?: (payload: InferEdgeEvent) => void
  onWarning?: (payload: InferWarningEvent) => void
}

export interface UseInferStreamOptions extends InferStreamHandlers {
  onDone?: (payload: InferDonePayload) => void
  onError?: (error: Error) => void
}
