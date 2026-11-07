/**
 * 自定义关系边：平滑贝塞尔曲线 + 关系色 + 关系标签。
 * 颜色只跟 data.relation 走，不跟节点走；AI 生成的边加虚线。
 */
import {
  BaseEdge,
  EdgeLabelRenderer,
  getBezierPath,
  type EdgeProps,
} from '@xyflow/react'
import { RELATION_META } from '../../types/canvas-node'
import type { NodalEdge } from '../../types/canvas-node'

export function RelationEdge({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  data,
  selected,
  markerEnd,
}: EdgeProps<NodalEdge>) {
  const relation = data?.relation ?? 'derive'
  const meta = RELATION_META[relation]
  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
    curvature: 0.28,
  })

  return (
    <>
      <BaseEdge
        id={id}
        path={edgePath}
        markerEnd={markerEnd}
        style={{
          stroke: meta.color,
          strokeWidth: selected ? 3 : data?.ai_generated ? 2 : 1.8,
          strokeDasharray: data?.ai_generated ? '6 4' : undefined,
          opacity: selected ? 1 : 0.86,
        }}
      />
      <EdgeLabelRenderer>
        <div
          className="nodal-edge-label"
          style={{
            transform: `translate(-50%, -50%) translate(${labelX}px, ${labelY}px)`,
            color: meta.color,
            borderColor: `${meta.color}55`,
            background: `${meta.color}12`,
          }}
        >
          {meta.label}
        </div>
      </EdgeLabelRenderer>
    </>
  )
}
