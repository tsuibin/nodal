/**
 * 画布图的纯函数：服务端 JSON ⇄ React Flow Node/Edge、关系元数据、
 * 新节点 id、空闲位置查找。
 *
 * 这些函数不依赖 React，方便在 hook、测试和导入 Demo 时复用。
 */
import { MarkerType } from '@xyflow/react'
import type { RFEdge, RFNode } from '../types/infer'
import type {
  NodalEdge,
  NodalNode,
  NodalNodeData,
  NodalNodeKind,
  RelationType,
} from '../types/canvas-node'
import { RELATION_META, RELATION_ORDER } from '../types/canvas-node'

const NODE_KINDS: NodalNodeKind[] = ['text', 'code', 'formula', 'image', 'file']
const DEFAULT_NODE_WIDTH = 248
const DEFAULT_NODE_HEIGHT = 154
const NODE_GAP = 28

export function isNodalNodeKind(value: unknown): value is NodalNodeKind {
  return typeof value === 'string' && NODE_KINDS.includes(value as NodalNodeKind)
}

export function isRelationType(value: unknown): value is RelationType {
  return typeof value === 'string' && RELATION_ORDER.includes(value as RelationType)
}

export function relationColor(relation: RelationType): string {
  return RELATION_META[relation].color
}

/** 后端节点 → 前端 Render Node。未知 type 一律按文本节点处理。 */
export function normalizeNode(raw: RFNode, index = 0): NodalNode {
  const source = (raw.data ?? {}) as Record<string, unknown>
  const rawType = typeof raw.type === 'string' ? raw.type : ''
  const kind = isNodalNodeKind(source.kind)
    ? (source.kind as NodalNodeKind)
    : isNodalNodeKind(rawType)
      ? (rawType as NodalNodeKind)
      : inferKindFromType(rawType)

  const ai = source.ai_generated === true
  const agentID = numberOrUndefined(source.ai_agent_id ?? source.agent_id)
  const label =
    stringOr(source.label) ||
    stringOr(source.title) ||
    (kind === 'formula' ? '公式节点' : kind === 'image' ? '图片节点' : `节点 ${index + 1}`)

  const data: NodalNodeData = {
    ...source,
    kind,
    label,
    source: ai || source.source === 'agent' ? 'agent' : 'user',
    agent_id: agentID,
    agent_name: stringOr(source.agent_name) || undefined,
    ai_generated: ai || source.ai_generated === true,
    status: source.status === 'conclusion' ? 'conclusion' : 'normal',
    content: stringOr(source.content ?? source.text) || undefined,
    code: stringOr(source.code) || undefined,
    language: stringOr(source.language) || (kind === 'code' ? 'python' : undefined),
    formula: stringOr(source.formula) || undefined,
    imageUrl: stringOr(source.imageUrl ?? source.image_url) || undefined,
    fileName: stringOr(source.fileName ?? source.file_name) || undefined,
    fileType: stringOr(source.fileType ?? source.file_type) || undefined,
    fileSize: numberOrUndefined(source.fileSize ?? source.file_size),
    created_at: stringOr(source.created_at) || undefined,
  }

  return {
    ...raw,
    id: String(raw.id ?? `n_${Date.now()}_${index}`),
    type: 'nodal',
    position: {
      x: Number(raw.position?.x) || 0,
      y: Number(raw.position?.y) || 0,
    },
    data,
  }
}

function inferKindFromType(type: string): NodalNodeKind {
  switch (type) {
    case 'code':
    case 'formula':
    case 'image':
    case 'file':
      return type
    default:
      return 'text'
  }
}

/** 后端边 → 前端 Render Edge；没有 relation 的历史数据按「推导自」处理。 */
export function normalizeEdge(raw: RFEdge): NodalEdge {
  const data = (raw.data ?? {}) as Record<string, unknown>
  const relation = isRelationType(data.relation) ? (data.relation as RelationType) : 'derive'
  const color = relationColor(relation)
  return {
    ...raw,
    id: String(raw.id),
    source: String(raw.source),
    target: String(raw.target),
    type: 'relation',
    data: {
      ...data,
      relation,
      ai_generated: data.ai_generated === true,
      agent_id: numberOrUndefined(data.agent_id ?? data.ai_agent_id),
      created_at: stringOr(data.created_at) || undefined,
    },
    markerEnd: {
      type: MarkerType.ArrowClosed,
      width: 14,
      height: 14,
      color,
    },
    style: { ...(raw.style ?? {}), stroke: color, strokeWidth: 1.8 },
    animated: false,
  }
}

/** 保存到服务端前去掉只属于渲染层的字段，减少画布 JSON 体积。 */
export function toPersistedNode(node: NodalNode): RFNode {
  const { selected: _selected, dragging: _dragging, measured: _measured, ...rest } = node
  void _selected
  void _dragging
  void _measured
  return rest as RFNode
}

export function toPersistedEdge(edge: NodalEdge): RFEdge {
  const { selected: _selected, ...rest } = edge
  void _selected
  return rest as RFEdge
}

export function createNode(
  kind: NodalNodeKind,
  position: { x: number; y: number },
  seed?: Partial<NodalNodeData>,
): NodalNode {
  const id = newNodeID('n')
  const defaults: Record<NodalNodeKind, Partial<NodalNodeData>> = {
    text: {
      label: '新观点',
      content: '## 新观点\n\n在这里写下一个判断，或者把一段材料粘进来。',
    },
    code: {
      label: '代码实验',
      code: 'print("hello, nodal")',
      language: 'python',
    },
    formula: {
      label: '公式',
      formula: '\\Delta = b^2 - 4ac',
    },
    image: {
      label: '图片',
      imageUrl: '',
    },
    file: {
      label: '文件',
      fileName: '',
    },
  }
  return {
    id,
    type: 'nodal',
    position,
    data: {
      kind,
      label: defaults[kind].label ?? '新节点',
      source: 'user',
      status: 'normal',
      ai_generated: false,
      created_at: new Date().toISOString(),
      ...defaults[kind],
      ...seed,
    },
  }
}

export function createEdge(
  source: string,
  target: string,
  relation: RelationType,
  sourceHandle?: string | null,
  targetHandle?: string | null,
): NodalEdge {
  const color = relationColor(relation)
  return {
    id: newNodeID('e'),
    source,
    target,
    sourceHandle: sourceHandle ?? undefined,
    targetHandle: targetHandle ?? undefined,
    type: 'relation',
    data: {
      relation,
      ai_generated: false,
      created_at: new Date().toISOString(),
    },
    markerEnd: {
      type: MarkerType.ArrowClosed,
      width: 14,
      height: 14,
      color,
    },
    style: { stroke: color, strokeWidth: 1.8 },
  }
}

export function newNodeID(prefix: 'n' | 'e' | 'ai'): string {
  const random = Math.random().toString(36).slice(2, 7)
  return `${prefix}_${Date.now().toString(36)}_${random}`
}

/**
 * 找一个不覆盖已有节点的位置。
 * 画布节点没有固定宽高时用 248×154 的保守估计，按 3 列网格向外扩。
 */
export function findFreePosition(
  desired: { x: number; y: number },
  nodes: NodalNode[],
  width = DEFAULT_NODE_WIDTH,
  height = DEFAULT_NODE_HEIGHT,
): { x: number; y: number } {
  const boxes = nodes.map((node) => nodeBox(node))
  const start = { x: desired.x, y: desired.y }
  for (let ring = 0; ring < 12; ring += 1) {
    const stepX = width + NODE_GAP
    const stepY = height + NODE_GAP
    // 先试当前环的中心/右侧，再逐步向下，避免新节点全部堆在原点
    const candidates: Array<{ x: number; y: number }> = []
    if (ring === 0) candidates.push(start)
    for (let col = 0; col <= ring; col += 1) {
      candidates.push({ x: start.x + col * stepX, y: start.y + ring * stepY })
    }
    for (let row = 0; row < ring; row += 1) {
      candidates.push({ x: start.x + ring * stepX, y: start.y + row * stepY })
    }
    for (const candidate of candidates) {
      const rect = { x: candidate.x, y: candidate.y, width, height }
      if (!boxes.some((box) => intersects(box, rect))) return candidate
    }
  }
  return { x: start.x + 320, y: start.y + 220 }
}

function nodeBox(node: NodalNode) {
  const width = Number(node.width) || Number(node.style?.width) || DEFAULT_NODE_WIDTH
  const height = Number(node.height) || DEFAULT_NODE_HEIGHT
  return { x: node.position.x, y: node.position.y, width, height }
}

function intersects(
  a: { x: number; y: number; width: number; height: number },
  b: { x: number; y: number; width: number; height: number },
): boolean {
  return !(
    a.x + a.width + NODE_GAP < b.x ||
    b.x + b.width + NODE_GAP < a.x ||
    a.y + a.height + NODE_GAP < b.y ||
    b.y + b.height + NODE_GAP < a.y
  )
}

function stringOr(value: unknown): string {
  return typeof value === 'string' ? value : ''
}

function numberOrUndefined(value: unknown): number | undefined {
  const num = typeof value === 'number' ? value : Number(value)
  return Number.isFinite(num) ? num : undefined
}
