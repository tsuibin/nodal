/**
 * useCanvasGraph：画布页的受控图状态 + 撤销/重做 + dirty 标记。
 *
 * ReactFlow 的节点拖动会以高频 changes 到达，如果每帧都入历史栈，
 * Ctrl+Z 会退化成「退一像素」。这里只在 dragStart 记快照、dragStop 入栈，
 * 其他结构性操作（新增、删除、连线、编辑）显式调用 commit。
 */
import { useCallback, useEffect, useRef, useState } from 'react'
import {
  type EdgeChange,
  type NodeChange,
  useEdgesState,
  useNodesState,
} from '@xyflow/react'
import { CanvasHistory, type GraphSnapshot } from '../store/canvasHistory'
import type { NodalEdge, NodalNode, NodalNodeData } from '../types/canvas-node'

export interface UseCanvasGraphResult {
  nodes: NodalNode[]
  edges: NodalEdge[]
  dirty: boolean
  canUndo: boolean
  canRedo: boolean
  setGraph: (nodes: NodalNode[], edges: NodalEdge[]) => void
  onNodesChange: (changes: NodeChange<NodalNode>[]) => void
  onEdgesChange: (changes: EdgeChange<NodalEdge>[]) => void
  onNodeDragStart: () => void
  onNodeDragStop: () => void
  commit: (nodes: NodalNode[], edges?: NodalEdge[]) => void
  updateNodeData: (id: string, patch: Partial<NodalNodeData>) => void
  deleteNode: (id: string) => void
  deleteSelection: (nodeIDs: string[], edgeIDs: string[]) => void
  undo: () => void
  redo: () => void
  appendGeneratedNode: (node: NodalNode) => void
  appendGeneratedEdge: (edge: NodalEdge) => void
  markSaved: () => void
  markDirty: () => void
}

export function useCanvasGraph(): UseCanvasGraphResult {
  const [nodes, setNodes, baseOnNodesChange] = useNodesState<NodalNode>([])
  const [edges, setEdges, baseOnEdgesChange] = useEdgesState<NodalEdge>([])
  const [dirty, setDirty] = useState(false)
  const [, setHistoryTick] = useState(0)
  const historyRef = useRef(new CanvasHistory())
  const nodesRef = useRef<NodalNode[]>(nodes)
  const edgesRef = useRef<NodalEdge[]>(edges)
  const dragSnapshotRef = useRef<GraphSnapshot | null>(null)

  useEffect(() => {
    nodesRef.current = nodes
  }, [nodes])
  useEffect(() => {
    edgesRef.current = edges
  }, [edges])

  const currentGraph = useCallback(
    (): GraphSnapshot => ({
      nodes: nodesRef.current,
      edges: edgesRef.current,
    }),
    [],
  )

  const record = useCallback(() => {
    historyRef.current.push(currentGraph())
    setHistoryTick((tick) => tick + 1)
  }, [currentGraph])

  const setGraph = useCallback((nextNodes: NodalNode[], nextEdges: NodalEdge[]) => {
    historyRef.current.clear()
    setHistoryTick((tick) => tick + 1)
    nodesRef.current = nextNodes
    edgesRef.current = nextEdges
    setNodes(nextNodes)
    setEdges(nextEdges)
    setDirty(false)
  }, [setEdges, setNodes])

  const markDirty = useCallback(() => setDirty(true), [])

  const onNodesChange = useCallback(
    (changes: NodeChange<NodalNode>[]) => {
      baseOnNodesChange(changes)
      // 选中、尺寸测量不算内容变更；位置/删除/新增才算
      if (changes.some((change) => change.type !== 'select' && change.type !== 'dimensions')) {
        setDirty(true)
      }
    },
    [baseOnNodesChange],
  )

  const onEdgesChange = useCallback(
    (changes: EdgeChange<NodalEdge>[]) => {
      baseOnEdgesChange(changes)
      if (changes.some((change) => change.type !== 'select')) setDirty(true)
    },
    [baseOnEdgesChange],
  )

  const onNodeDragStart = useCallback(() => {
    dragSnapshotRef.current = currentGraph()
  }, [currentGraph])

  const onNodeDragStop = useCallback(() => {
    if (!dragSnapshotRef.current) return
    // 位置真的变了才入栈，避免单击节点也产生一条历史
    const before = dragSnapshotRef.current
    const after = currentGraph()
    const changed =
      before.nodes.some((node, index) => {
        const next = after.nodes[index]
        return !next || node.position.x !== next.position.x || node.position.y !== next.position.y
      }) || before.nodes.length !== after.nodes.length
    if (changed) {
      historyRef.current.push(before)
      setHistoryTick((tick) => tick + 1)
    }
    dragSnapshotRef.current = null
  }, [currentGraph])

  const commit = useCallback(
    (nextNodes: NodalNode[], nextEdges?: NodalEdge[]) => {
      record()
      nodesRef.current = nextNodes
      if (nextEdges) edgesRef.current = nextEdges
      setNodes(nextNodes)
      if (nextEdges) setEdges(nextEdges)
      setDirty(true)
    },
    [record, setEdges, setNodes],
  )

  const updateNodeData = useCallback(
    (id: string, patch: Partial<NodalNodeData>) => {
      record()
      setNodes((current) => {
        const next = current.map((node) =>
          node.id === id ? { ...node, data: { ...node.data, ...patch } } : node,
        )
        nodesRef.current = next
        return next
      })
      setDirty(true)
    },
    [record, setNodes],
  )

  const deleteNode = useCallback(
    (id: string) => {
      record()
      const nextNodes = nodesRef.current.filter((node) => node.id !== id)
      const nextEdges = edgesRef.current.filter((edge) => edge.source !== id && edge.target !== id)
      nodesRef.current = nextNodes
      edgesRef.current = nextEdges
      setNodes(nextNodes)
      setEdges(nextEdges)
      setDirty(true)
    },
    [record, setEdges, setNodes],
  )

  const deleteSelection = useCallback(
    (nodeIDs: string[], edgeIDs: string[]) => {
      if (!nodeIDs.length && !edgeIDs.length) return
      record()
      const nextNodes = nodesRef.current.filter((node) => !nodeIDs.includes(node.id))
      const nextEdges = edgesRef.current.filter(
        (edge) =>
          !edgeIDs.includes(edge.id) && !nodeIDs.includes(edge.source) && !nodeIDs.includes(edge.target),
      )
      nodesRef.current = nextNodes
      edgesRef.current = nextEdges
      setNodes(nextNodes)
      setEdges(nextEdges)
      setDirty(true)
    },
    [record, setEdges, setNodes],
  )

  const applySnapshot = useCallback(
    (snapshot: GraphSnapshot) => {
      nodesRef.current = snapshot.nodes
      edgesRef.current = snapshot.edges
      setNodes(snapshot.nodes)
      setEdges(snapshot.edges)
      setDirty(true)
    },
    [setEdges, setNodes],
  )

  const undo = useCallback(() => {
    const snapshot = historyRef.current.undo(currentGraph())
    if (!snapshot) return
    applySnapshot(snapshot)
    setHistoryTick((tick) => tick + 1)
  }, [applySnapshot, currentGraph])

  const redo = useCallback(() => {
    const snapshot = historyRef.current.redo(currentGraph())
    if (!snapshot) return
    applySnapshot(snapshot)
    setHistoryTick((tick) => tick + 1)
  }, [applySnapshot, currentGraph])

  // 流式推演可能在一帧内连续追加多个节点/边，不能依赖 setState 回调
  // 才更新 ref，否则历史快照会漏掉前一个刚生成的节点。这里同步改 ref 后入栈。
  const appendGeneratedNode = useCallback((node: NodalNode) => {
    historyRef.current.push({ nodes: nodesRef.current, edges: edgesRef.current })
    const next = [...nodesRef.current, node]
    nodesRef.current = next
    setNodes(next)
    setHistoryTick((tick) => tick + 1)
    setDirty(true)
  }, [setNodes])

  const appendGeneratedEdge = useCallback((edge: NodalEdge) => {
    historyRef.current.push({ nodes: nodesRef.current, edges: edgesRef.current })
    const next = [...edgesRef.current, edge]
    edgesRef.current = next
    setEdges(next)
    setHistoryTick((tick) => tick + 1)
    setDirty(true)
  }, [setEdges])

  return {
    nodes,
    edges,
    dirty,
    canUndo: historyRef.current.canUndo,
    canRedo: historyRef.current.canRedo,
    setGraph,
    onNodesChange,
    onEdgesChange,
    onNodeDragStart,
    onNodeDragStop,
    commit,
    updateNodeData,
    deleteNode,
    deleteSelection,
    undo,
    redo,
    appendGeneratedNode,
    appendGeneratedEdge,
    markSaved: () => setDirty(false),
    markDirty,
  }
}
