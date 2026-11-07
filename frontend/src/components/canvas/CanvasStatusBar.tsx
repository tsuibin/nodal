/**
 * 画布左下角状态条：节点/连线数量、选中数、缩放比例、编辑权限。
 */
export function CanvasStatusBar({
  nodeCount,
  edgeCount,
  selectedNodeCount,
  zoom,
  readOnly,
  authenticated,
  guest,
}: {
  nodeCount: number
  edgeCount: number
  selectedNodeCount: number
  zoom: number
  readOnly: boolean
  authenticated: boolean
  guest: boolean
}) {
  return (
    <div className="nodal-status-bar">
      <span>{nodeCount} 节点</span>
      <span>{edgeCount} 连线</span>
      {selectedNodeCount ? <span className="text-brand">已选 {selectedNodeCount}</span> : null}
      <span>{Math.round(zoom * 100)}%</span>
      <span className="nodal-status-divider" />
      <span>{readOnly ? '只读' : guest ? '游客临时画布' : authenticated ? '已登录' : '未登录'}</span>
      <span className="hidden sm:inline">左键框选 · 右键平移 · 滚轮缩放</span>
    </div>
  )
}
