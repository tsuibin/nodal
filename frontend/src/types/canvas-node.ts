/**
 * 画布节点的业务类型。
 *
 * 存储层与 React Flow 完全对齐：Node / Edge 的字段（id、position、data、
 * sourceHandle…）原样落库；`data` 里放 Nodal 自己的语义字段。
 *
 * 来源标记（PRD 规则 8）：
 *   source: 'user' | 'agent'
 *   agent_id: AI 节点所属 Agent 模板 id
 * 后端推演流回传的字段是 `ai_generated` / `ai_agent_id`，这里在
 * normalizeNode 里统一映射成上面的业务字段。
 */
import type { Edge, Node } from '@xyflow/react'

export type NodalNodeKind = 'text' | 'code' | 'formula' | 'image' | 'file'

export type NodeSource = 'user' | 'agent'

/** 节点语义状态：普通节点 / 结论节点 */
export type NodalNodeStatus = 'normal' | 'conclusion'

export type RelationType = 'derive' | 'counter' | 'evidence' | 'hypothesis'

export interface NodalNodeData extends Record<string, unknown> {
  kind: NodalNodeKind
  /** 节点标题，画布上始终展示 */
  label: string
  /** 文本节点的 Markdown 正文 */
  content?: string
  /** 代码节点的源码 */
  code?: string
  /** 代码语言，如 python / bash / javascript */
  language?: string
  /** 公式节点的 KaTeX 表达式（不带 $$） */
  formula?: string
  /** 图片节点的 URL 或 data URL */
  imageUrl?: string
  /** 文件节点的元信息（文件本身不落库，只保留描述） */
  fileName?: string
  fileSize?: number
  fileType?: string
  source: NodeSource
  /** AI 节点所属 Agent */
  agent_id?: number
  agent_name?: string
  ai_generated?: boolean
  status?: NodalNodeStatus
  /** 节点创建时间，便于排序/审计 */
  created_at?: string
  /** 保留后端阿里的其他 React Flow data 字段 */
  [key: string]: unknown
}

export interface NodalEdgeData extends Record<string, unknown> {
  relation: RelationType
  ai_generated?: boolean
  agent_id?: number
  /** 关系选择时间 */
  created_at?: string
}

export type NodalNode = Node<NodalNodeData, 'nodal'>
export type NodalEdge = Edge<NodalEdgeData, 'relation'>

/** 四种关系的视觉与文案，严格对齐品牌色 */
export const RELATION_META: Record<
  RelationType,
  { label: string; color: string; soft: string; description: string }
> = {
  derive: {
    label: '推导自',
    color: '#4060FF',
    soft: '#eef1ff',
    description: '从一个判断继续向前，得到更具体的结论。',
  },
  counter: {
    label: '反例质疑',
    color: '#ff6b6b',
    soft: '#fff0f0',
    description: '主动寻找反例，让结论在压力下站稳。',
  },
  evidence: {
    label: '补充佐证',
    color: '#36c9a5',
    soft: '#e9fbf6',
    description: '引入证据、数据与引用，补齐论证链。',
  },
  hypothesis: {
    label: '并行假设',
    color: '#ffc857',
    soft: '#fff8e6',
    description: '同时保留多个可能，比较后再收敛。',
  },
}

export const RELATION_ORDER: RelationType[] = ['derive', 'counter', 'evidence', 'hypothesis']

export const NODE_KIND_META: Record<NodalNodeKind, { label: string; hint: string }> = {
  text: { label: '文本', hint: 'Markdown 观点、判断、摘录' },
  code: { label: '代码', hint: 'Monaco 编辑 + 沙箱运行' },
  formula: { label: '公式', hint: 'KaTeX 实时渲染' },
  image: { label: '图片', hint: '粘贴图片 URL 或上传' },
  file: { label: '文件', hint: '记录文件名、类型、大小' },
}
