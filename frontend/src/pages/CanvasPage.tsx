/**
 * 模块 9：核心画布页 /canvas/:id
 *
 * 结构：
 *   CanvasPage        负责加载画布、错误页、ReactFlowProvider
 *   CanvasWorkspace   负责受控图、历史栈、保存、推演流、面板与弹窗
 *
 * 交互约定：
 *   - 左键拖拽框选，右键/中键平移，滚轮缩放；
 *   - 拖拽节点右侧 Handle 到另一个节点，先选关系再落边；
 *   - 选中节点后底部浮出「推演」，AI 只新增节点/边；
 *   - 所有修改进入撤销栈，Ctrl+Z / Ctrl+Shift+Z；
 *   - 失焦 1.4 秒自动保存，Ctrl+S 立即保存。
 */
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  Background,
  BackgroundVariant,
  ConnectionMode,
  Controls,
  MiniMap,
  ReactFlow,
  ReactFlowProvider,
  useReactFlow,
  type Connection,
  type OnSelectionChangeParams,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'

import { useAuth } from '../hooks/useAuth'
import { useAgentList } from '../hooks/useAgentList'
import { useModelCatalog } from '../hooks/useModelCatalog'
import { useCanvasGraph } from '../hooks/useCanvasGraph'
import { useInferStream } from '../hooks/useInferStream'
import { useLoginModal } from '../components/LoginModalProvider'
import { useToast } from '../components/ToastProvider'
import { ApiError, tokenStore } from '../services/http'
import { canvasApi } from '../services/canvas'
import type { CanvasView } from '../types/canvas'
import type { RFEdge, RFNode } from '../types/infer'
import {
  createEdge,
  createNode,
  findFreePosition,
  normalizeEdge,
  normalizeNode,
  toPersistedEdge,
  toPersistedNode,
} from '../lib/graph'
import {
  type NodalEdge,
  type NodalNode,
  type NodalNodeData,
  type NodalNodeKind,
  type RelationType,
} from '../types/canvas-node'
import { CanvasToolbar } from '../components/canvas/CanvasToolbar'
import { CanvasSidePanel } from '../components/canvas/CanvasSidePanel'
import { CanvasStatusBar } from '../components/canvas/CanvasStatusBar'
import { CanvasEmptyState } from '../components/canvas/CanvasEmptyState'
import { InferDialog } from '../components/canvas/InferDialog'
import { NodeEditorDialog } from '../components/canvas/NodeEditorDialog'
import { RelationEdge } from '../components/canvas/RelationEdge'
import { RelationPicker } from '../components/canvas/RelationPicker'
import { SelectionBar } from '../components/canvas/SelectionBar'
import { ShareDialog } from '../components/canvas/ShareDialog'
import { NodalNodeCard } from '../components/canvas/NodalNode'
import { EmptyState } from '../components/EmptyState'
import { NodeIcon } from '../components/icons'

const nodeTypes = { nodal: NodalNodeCard }
const edgeTypes = { relation: RelationEdge }

export function CanvasPage() {
  const { id } = useParams()
  const canvasID = Number(id)
  const navigate = useNavigate()
  const auth = useAuth()

  const [canvas, setCanvas] = useState<CanvasView | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<ApiError | Error | null>(null)
  const [reloadKey, setReloadKey] = useState(0)

  useEffect(() => {
    if (!Number.isFinite(canvasID) || canvasID <= 0) {
      setError(new Error('画布 id 不合法'))
      setLoading(false)
      return
    }
    let cancelled = false
    const load = async () => {
      setLoading(true)
      setError(null)
      let token = tokenStore.getActive().token
      if (!token) {
        // 没登录也没有游客身份时，先领一个临时身份，才能读自己的临时画布
        await auth.ensureGuest()
        token = tokenStore.getActive().token
      }
      if (!token) {
        if (!cancelled) {
          setError(new ApiError({ code: 1001, message: '请先登录或领取游客身份' }))
          setLoading(false)
        }
        return
      }
      try {
        const view = await canvasApi.get(canvasID)
        if (!cancelled) {
          setCanvas(view)
          setLoading(false)
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err : new Error(String(err)))
          setLoading(false)
        }
      }
    }
    void load()
    return () => {
      cancelled = true
    }
    // auth 变化只在首次领取游客时用到，避免登录后重复请求画布
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [canvasID, reloadKey])

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-[var(--nodal-bg)] text-[var(--nodal-text)]">
        <div className="text-center">
          <span className="nodal-spinner mx-auto block" />
          <p className="mt-3 text-sm text-[var(--nodal-muted)]">正在打开画布…</p>
        </div>
      </div>
    )
  }

  if (!canvas || error) {
    return (
      <div className="mx-auto max-w-2xl px-4 py-24">
        <EmptyState
          icon={<NodeIcon />}
          title={error instanceof ApiError && error.code === 1003 ? '需要登录才能查看' : '打不开这张画布'}
          description={error?.message || '画布不存在，或者已经被删除。'}
          action={
            <div className="flex flex-wrap justify-center gap-2">
              <button
                type="button"
                className="nodal-btn nodal-btn-primary"
                onClick={() => setReloadKey((key) => key + 1)}
              >
                重试
              </button>
              <button type="button" className="nodal-btn nodal-btn-ghost" onClick={() => navigate('/library')}>
                返回资源库
              </button>
            </div>
          }
        />
      </div>
    )
  }

  return (
    <ReactFlowProvider>
      <CanvasWorkspace
        key={canvas.id}
        initialCanvas={canvas}
        onBack={() => navigate('/library')}
        onNeedLibrary={() => navigate('/library')}
      />
    </ReactFlowProvider>
  )
}

interface CanvasWorkspaceProps {
  initialCanvas: CanvasView
  onBack: () => void
  onNeedLibrary: () => void
}

function CanvasWorkspace({ initialCanvas, onBack, onNeedLibrary }: CanvasWorkspaceProps) {
  const auth = useAuth()
  const navigate = useNavigate()
  const { openLogin } = useLoginModal()
  const toast = useToast()
  const flow = useReactFlow()
  const authenticated = auth.status === 'authenticated'
  const canEdit = initialCanvas.permissions.can_edit
  const graph = useCanvasGraph()
  const agents = useAgentList(authenticated)
  const models = useModelCatalog(authenticated)

  const [canvas, setCanvas] = useState<CanvasView>(initialCanvas)
  const [title, setTitle] = useState(initialCanvas.title)
  const [description, setDescription] = useState(initialCanvas.description)
  const [tags, setTags] = useState(initialCanvas.tags)
  const [panelOpen, setPanelOpen] = useState(true)
  const [saving, setSaving] = useState(false)
  const [savedAt, setSavedAt] = useState('')
  const [notice, setNotice] = useState('')
  const [selectedAgentID, setSelectedAgentID] = useState<number | null>(null)
  const [selected, setSelected] = useState<{ nodes: string[]; edges: string[] }>({ nodes: [], edges: [] })
  const [editorNode, setEditorNode] = useState<NodalNode | null>(null)
  const [pendingConnection, setPendingConnection] = useState<Connection | null>(null)
  const [inferOpen, setInferOpen] = useState(false)
  const [shareOpen, setShareOpen] = useState(false)
  const [forking, setForking] = useState(false)
  const [zoom, setZoom] = useState(initialCanvas.viewport.zoom || 1)

  const nodesRef = useRef<NodalNode[]>(graph.nodes)
  const edgesRef = useRef<NodalEdge[]>(graph.edges)
  const viewportRef = useRef(initialCanvas.viewport)
  const metaRef = useRef({ title: initialCanvas.title, description: initialCanvas.description, tags: initialCanvas.tags })
  const startRef = useRef<{ agent_name?: string; agent_id?: number } | null>(null)
  const saveLockRef = useRef(false)
  const mountedRef = useRef(true)

  useEffect(() => {
    nodesRef.current = graph.nodes
  }, [graph.nodes])
  useEffect(() => {
    edgesRef.current = graph.edges
  }, [graph.edges])
  useEffect(() => {
    metaRef.current = { title, description, tags }
  }, [title, description, tags])
  useEffect(() => {
    mountedRef.current = true
    return () => {
      mountedRef.current = false
    }
  }, [])

  // 服务端 JSON → React Flow 渲染对象；载入时清空历史栈
  useEffect(() => {
    const nextNodes = initialCanvas.nodes.map((node, index) => normalizeNode(node, index))
    const nextEdges = initialCanvas.edges.map((edge) => normalizeEdge(edge))
    graph.setGraph(nextNodes, nextEdges)
    applyViewport(flow, initialCanvas.viewport)
    // 只应在画布 id 变化时执行；graph.setGraph 在 hook 内稳定
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [initialCanvas.id])

  useEffect(() => {
    if (agents.items.length === 0) return
    if (selectedAgentID === null || !agents.items.some((agent) => agent.id === selectedAgentID)) {
      setSelectedAgentID(agents.items[0]?.id ?? null)
    }
  }, [agents.items, selectedAgentID])

  // ---------- 保存 ----------

  const save = useCallback(async () => {
    if (!canEdit || saveLockRef.current) return
    saveLockRef.current = true
    setSaving(true)
    try {
      const payload = {
        title: metaRef.current.title.trim() || '未命名画布',
        description: metaRef.current.description,
        tags: metaRef.current.tags,
        nodes: nodesRef.current.map(toPersistedNode),
        edges: edgesRef.current.map(toPersistedEdge),
        viewport: viewportRef.current,
      }
      const signature = graphSignature(payload.nodes, payload.edges, payload.title, payload.description, payload.tags)
      const view = await canvasApi.update(initialCanvas.id, payload)
      if (!mountedRef.current) return
      setCanvas((current) => ({
        ...current,
        ...view,
        // count 用本地实时值，避免保存期间新增节点导致数字回退
        node_count: nodesRef.current.length,
        edge_count: edgesRef.current.length,
      }))
      const currentSignature = graphSignature(
        nodesRef.current.map(toPersistedNode),
        edgesRef.current.map(toPersistedEdge),
        metaRef.current.title,
        metaRef.current.description,
        metaRef.current.tags,
      )
      if (currentSignature === signature) graph.markSaved()
      setSavedAt(formatTime(new Date()))
    } catch (err) {
      if (mountedRef.current) setNotice(err instanceof Error ? err.message : String(err))
    } finally {
      saveLockRef.current = false
      if (mountedRef.current) setSaving(false)
    }
  }, [canEdit, graph, initialCanvas.id])

  // 编辑后 1.4 秒自动保存；推演高频追加也会被 debounce 合并
  useEffect(() => {
    if (!graph.dirty || !canEdit || saving) return
    const timer = window.setTimeout(() => void save(), 1400)
    return () => window.clearTimeout(timer)
  }, [canEdit, graph.dirty, save, saving])

  // ---------- 推演流 ----------

  const infer = useInferStream({
    onStart: (start) => {
      startRef.current = { agent_name: start.agent_name, agent_id: start.agent_id }
    },
    onNode: ({ node }) => {
      const normalized = normalizeNode(node as RFNode)
      const desired = findFreePosition(normalized.position, nodesRef.current)
      const withAgent: NodalNode = {
        ...normalized,
        position: desired,
        data: {
          ...normalized.data,
          source: 'agent',
          agent_id: startRef.current?.agent_id ?? normalized.data.agent_id,
          agent_name: startRef.current?.agent_name,
          ai_generated: true,
        },
      }
      nodesRef.current = [...nodesRef.current, withAgent]
      graph.appendGeneratedNode(withAgent)
    },
    onEdge: ({ edge }) => {
      const normalized = normalizeEdge(edge as RFEdge)
      const aiEdge: NodalEdge = {
        ...normalized,
        data: {
          ...(normalized.data ?? {}),
          relation: normalized.data?.relation ?? 'derive',
          ai_generated: true,
          agent_id: startRef.current?.agent_id,
        },
      }
      edgesRef.current = [...edgesRef.current, aiEdge]
      graph.appendGeneratedEdge(aiEdge)
    },
    onDone: (done) => {
      const message = `推演完成：新增 ${done.node_count} 个节点、${done.edge_count} 条连线${
        done.skipped ? `，忽略 ${done.skipped} 条非法输出` : ''
      }。`
      setNotice(message)
      toast.success(message)
      // 让新增节点进入视野，但不打断用户当前手动缩放太多
      window.setTimeout(() => flow.fitView({ padding: 0.22, duration: 500 }), 80)
    },
    onError: (err) => {
      setNotice(err.message)
      toast.error(err.message)
    },
  })

  const startInfer = (agentID: number, modelID: string) => {
    if (!authenticated) {
      openLogin()
      return
    }
    setNotice('')
    void infer.run(initialCanvas.id, {
      selected_node_ids: selected.nodes,
      agent_id: agentID,
      model_id: modelID,
    })
  }

  // ---------- 图操作 ----------

  const selectedNodes = graph.nodes.filter((node) => node.selected)
  const selectedEdges = graph.edges.filter((edge) => edge.selected)

  const onSelectionChange = useCallback(({ nodes, edges }: OnSelectionChangeParams) => {
    setSelected({ nodes: nodes.map((node) => node.id), edges: edges.map((edge) => edge.id) })
  }, [])

  const addNode = (kind: NodalNodeKind) => {
    if (!canEdit) return
    const center = flow.screenToFlowPosition({
      x: window.innerWidth / 2,
      y: window.innerHeight / 2,
    })
    const candidate = findFreePosition({ x: center.x - 124, y: center.y - 70 }, nodesRef.current)
    const node = { ...createNode(kind, candidate), selected: true }
    const nextNodes = nodesRef.current.map((item) => ({ ...item, selected: false })).concat(node)
    graph.commit(nextNodes, edgesRef.current)
    setSelected({ nodes: [node.id], edges: [] })
    flow.setCenter(candidate.x + 124, candidate.y + 70, { zoom: 1, duration: 300 })
  }

  const onConnect = useCallback((connection: Connection) => {
    if (!canEdit) return
    if (!connection.source || !connection.target || connection.source === connection.target) return
    setPendingConnection(connection)
  }, [canEdit])

  const pickRelation = (relation: RelationType) => {
    if (!pendingConnection) return
    const edge = createEdge(
      pendingConnection.source,
      pendingConnection.target,
      relation,
      pendingConnection.sourceHandle,
      pendingConnection.targetHandle,
    )
    graph.commit(nodesRef.current, [...edgesRef.current, edge])
    setPendingConnection(null)
  }

  const deleteSelected = () => {
    if (!canEdit) return
    graph.deleteSelection(selected.nodes, selected.edges)
    setSelected({ nodes: [], edges: [] })
  }

  const updateNodeData = (id: string, patch: Partial<NodalNodeData>) => {
    if (!canEdit) return
    graph.updateNodeData(id, patch)
  }

  const toggleConclusion = () => {
    if (selectedNodes.length !== 1 || !canEdit) return
    const node = selectedNodes[0]
    graph.updateNodeData(node.id, {
      status: node.data.status === 'conclusion' ? 'normal' : 'conclusion',
    })
  }

  const onMetaChange = (patch: { title?: string; description?: string; tags?: string[] }) => {
    if (!canEdit) return
    if (patch.title !== undefined) setTitle(patch.title)
    if (patch.description !== undefined) setDescription(patch.description)
    if (patch.tags !== undefined) setTags(patch.tags)
    setCanvas((current) => ({
      ...current,
      title: patch.title ?? current.title,
      description: patch.description ?? current.description,
      tags: patch.tags ?? current.tags,
    }))
    graph.markDirty()
  }

  const fork = async () => {
    if (!authenticated) {
      openLogin()
      return
    }
    setForking(true)
    setNotice('')
    try {
      const copy = await canvasApi.fork(initialCanvas.id)
      navigate(`/canvas/${copy.id}`)
    } catch (err) {
      setNotice(err instanceof Error ? err.message : String(err))
    } finally {
      setForking(false)
    }
  }

  // 键盘：保存、撤销、重做、删除；输入框/Monaco 内不劫持
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null
      const typing =
        target?.tagName === 'INPUT' ||
        target?.tagName === 'TEXTAREA' ||
        target?.tagName === 'SELECT' ||
        Boolean(target?.closest('.monaco-editor'))
      if (typing) return

      const mod = event.metaKey || event.ctrlKey
      if (mod && event.key.toLowerCase() === 's') {
        event.preventDefault()
        void save()
        return
      }
      if (canEdit && mod && event.key.toLowerCase() === 'z') {
        event.preventDefault()
        if (event.shiftKey) graph.redo()
        else graph.undo()
        return
      }
      if (canEdit && (event.key === 'Delete' || event.key === 'Backspace') && (selected.nodes.length || selected.edges.length)) {
        event.preventDefault()
        deleteSelected()
      }
      if (event.key === 'Escape') {
        setEditorNode(null)
        setInferOpen(false)
        setShareOpen(false)
        setPendingConnection(null)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
    // deleteSelected 是闭包，依赖 selected；save 稳定
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [canEdit, graph, save, selected.nodes, selected.edges])

  // 离开页面前提醒未保存
  useEffect(() => {
    const onBeforeUnload = (event: BeforeUnloadEvent) => {
      if (!graph.dirty) return
      event.preventDefault()
      event.returnValue = ''
    }
    window.addEventListener('beforeunload', onBeforeUnload)
    return () => window.removeEventListener('beforeunload', onBeforeUnload)
  }, [graph.dirty])

  const openInfer = () => {
    if (!authenticated) {
      openLogin()
      return
    }
    if (!selected.nodes.length) {
      setNotice('先选中至少一个节点，再发起推演。')
      return
    }
    setInferOpen(true)
  }

  const share = () => {
    if (!authenticated) {
      openLogin()
      return
    }
    if (!initialCanvas.permissions.can_share) {
      setNotice('当前画布没有分享权限。')
      return
    }
    setShareOpen(true)
  }

  const editorSelected = useMemo(
    () => graph.nodes.find((node) => node.id === editorNode?.id) ?? null,
    [editorNode?.id, graph.nodes],
  )

  return (
    <div className="nodal-canvas-page">
      <CanvasToolbar
        title={title}
        dirty={graph.dirty}
        saving={saving}
        savedAt={savedAt}
        canEdit={canEdit}
        // 游客也允许点分享/Fork：先唤起登录，而不是把按钮直接置灰
        canShare={authenticated ? initialCanvas.permissions.can_share : true}
        canFork={authenticated ? initialCanvas.permissions.can_fork : true}
        canUndo={graph.canUndo}
        canRedo={graph.canRedo}
        panelOpen={panelOpen}
        onBack={onBack}
        onTitleChange={(value) => onMetaChange({ title: value })}
        onCommitTitle={() => {
          if (title.trim() === canvas.title) return
          graph.markDirty()
        }}
        onSave={() => void save()}
        onUndo={graph.undo}
        onRedo={graph.redo}
        onFit={() => flow.fitView({ padding: 0.2, duration: 450 })}
        onAddNode={addNode}
        onShare={share}
        onFork={() => void fork()}
        onTogglePanel={() => setPanelOpen((open) => !open)}
      />

      {notice ? (
        <div className="nodal-canvas-notice">
          <span>{notice}</span>
          <button type="button" onClick={() => setNotice('')} aria-label="关闭提示">
            ×
          </button>
        </div>
      ) : null}

      {infer.running ? (
        <div className="nodal-infer-banner">
          <span className="nodal-pulse-dot" />
          <span>
            {infer.start?.agent_name ?? 'Agent'} 正在推演
            {infer.generatedNodes.length ? ` · 已生成 ${infer.generatedNodes.length} 个节点` : ''}
          </span>
          <button type="button" onClick={infer.cancel}>
            停止
          </button>
        </div>
      ) : null}

      {!canEdit ? <div className="nodal-readonly-banner mx-4 mt-3">只读画布：你可以查看、Fork，但不能编辑。</div> : null}

      <div className="nodal-canvas-stage">
        <div className="nodal-canvas-flow">
          <ReactFlow
            nodes={graph.nodes}
            edges={graph.edges}
            nodeTypes={nodeTypes}
            edgeTypes={edgeTypes}
            onNodesChange={graph.onNodesChange}
            onEdgesChange={graph.onEdgesChange}
            onNodeDragStart={graph.onNodeDragStart}
            onNodeDragStop={graph.onNodeDragStop}
            onConnect={onConnect}
            onNodeDoubleClick={(_, node) => {
              if (node.type !== 'nodal') return
              setEditorNode(node as NodalNode)
            }}
            onSelectionChange={onSelectionChange}
            onMoveEnd={(_, viewport) => {
              const previous = viewportRef.current
              viewportRef.current = viewport
              setZoom(viewport.zoom)
              // 初次 applyViewport 也会触发 onMoveEnd，只有真正发生了变化才算编辑
              const moved =
                Math.abs(previous.x - viewport.x) > 0.5 ||
                Math.abs(previous.y - viewport.y) > 0.5 ||
                Math.abs(previous.zoom - viewport.zoom) > 0.001
              if (canEdit && moved) graph.markDirty()
            }}
            nodesDraggable={canEdit}
            nodesConnectable={canEdit}
            edgesReconnectable={canEdit}
            elementsSelectable
            deleteKeyCode={null}
            selectionOnDrag
            panOnDrag={[1, 2]}
            multiSelectionKeyCode={['Shift', 'Meta', 'Control']}
            connectionMode={ConnectionMode.Strict}
            snapToGrid
            snapGrid={[24, 24]}
            minZoom={0.2}
            maxZoom={2.2}
            zoomOnScroll
            zoomOnPinch
            fitView={false}
            defaultViewport={initialCanvas.viewport}
            connectionLineStyle={{
              stroke: '#4060FF',
              strokeWidth: 2,
              strokeDasharray: '5 4',
            }}
            className="nodal-reactflow"
          >
            <Background variant={BackgroundVariant.Dots} gap={24} size={1.2} color="var(--nodal-grid)" />
            <Controls showInteractive={false} position="bottom-left" />
            <MiniMap
              pannable
              zoomable
              position="bottom-right"
              nodeColor={(node) => {
                const data = (node.data ?? {}) as NodalNodeData
                if (data.status === 'conclusion') return '#ffc857'
                if (data.source === 'agent') return '#4060FF'
                return '#9aa3b5'
              }}
              maskColor="var(--nodal-minimap-mask)"
            />
          </ReactFlow>

          {graph.nodes.length === 0 ? (
            <CanvasEmptyState canEdit={canEdit} onAdd={addNode} onOpenLibrary={onNeedLibrary} />
          ) : null}

          <CanvasStatusBar
            nodeCount={graph.nodes.length}
            edgeCount={graph.edges.length}
            selectedNodeCount={selectedNodes.length}
            zoom={zoom}
            readOnly={!canEdit}
            authenticated={authenticated}
            guest={auth.status === 'guest'}
          />

          <SelectionBar
            nodes={selectedNodes}
            edgeCount={selectedEdges.length}
            canEdit={canEdit}
            authenticated={authenticated}
            inferRunning={infer.running}
            onInfer={openInfer}
            onEdit={() => selectedNodes[0] && setEditorNode(selectedNodes[0])}
            onDelete={deleteSelected}
            onConclusion={toggleConclusion}
          />
        </div>

        {panelOpen ? (
          <CanvasSidePanel
            canvas={canvas}
            nodes={graph.nodes}
            edges={graph.edges}
            authenticated={authenticated}
            canEdit={canEdit}
            agents={agents.items}
            agentsLoading={agents.loading}
            selectedAgentID={selectedAgentID}
            onSelectAgent={setSelectedAgentID}
            onNeedLogin={openLogin}
            onNeedAgent={onNeedLibrary}
            onMetaChange={onMetaChange}
          />
        ) : null}
      </div>

      <NodeEditorDialog
        node={editorSelected}
        canEdit={canEdit}
        onClose={() => setEditorNode(null)}
        onApply={updateNodeData}
        onDelete={(id) => {
          graph.deleteNode(id)
          setSelected({ nodes: [], edges: [] })
        }}
      />

      <RelationPicker
        open={Boolean(pendingConnection)}
        onPick={pickRelation}
        onCancel={() => setPendingConnection(null)}
      />

      <InferDialog
        open={inferOpen}
        selectedCount={selected.nodes.length}
        agents={agents.items}
        loadingAgents={agents.loading}
        models={models.items}
        loadingModels={models.loading}
        running={infer.running}
        error={infer.error}
        done={infer.done}
        onClose={() => setInferOpen(false)}
        onRun={startInfer}
        onNeedLogin={openLogin}
        onNeedAgent={onNeedLibrary}
      />

      {shareOpen ? <ShareDialog canvasId={initialCanvas.id} onClose={() => setShareOpen(false)} /> : null}

      {forking ? <div className="nodal-saving-mask">正在 Fork…</div> : null}
    </div>
  )
}

function applyViewport(
  flow: ReturnType<typeof useReactFlow>,
  viewport: { x: number; y: number; zoom: number },
) {
  window.requestAnimationFrame(() => {
    try {
      flow.setViewport(viewport, { duration: 0 })
    } catch {
      /* 画布还未挂载时忽略 */
    }
  })
}

function graphSignature(
  nodes: RFNode[],
  edges: RFEdge[],
  title: string,
  description: string,
  tags: string[],
): string {
  return JSON.stringify({ nodes, edges, title, description, tags })
}

function formatTime(date: Date): string {
  return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', hour12: false })
}
