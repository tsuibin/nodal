/**
 * 画布撤销/重做栈。
 *
 * 只保存「用户可见」的 nodes/edges 快照，不保存选中态、拖拽态和视口；
 * 视口变化频繁且不属于内容编辑，独立由页面保存。
 */
import type { NodalEdge, NodalNode } from '../types/canvas-node'

export interface GraphSnapshot {
  nodes: NodalNode[]
  edges: NodalEdge[]
}

export class CanvasHistory {
  private past: GraphSnapshot[] = []
  private future: GraphSnapshot[] = []

  constructor(private readonly limit = 60) {}

  get canUndo(): boolean {
    return this.past.length > 0
  }

  get canRedo(): boolean {
    return this.future.length > 0
  }

  push(snapshot: GraphSnapshot): void {
    this.past.push(copyGraph(snapshot))
    if (this.past.length > this.limit) this.past.shift()
    this.future = []
  }

  undo(current: GraphSnapshot): GraphSnapshot | null {
    const previous = this.past.pop()
    if (!previous) return null
    this.future.push(copyGraph(current))
    return previous
  }

  redo(current: GraphSnapshot): GraphSnapshot | null {
    const next = this.future.pop()
    if (!next) return null
    this.past.push(copyGraph(current))
    return next
  }

  clear(): void {
    this.past = []
    this.future = []
  }
}

export function copyGraph(graph: GraphSnapshot): GraphSnapshot {
  if (typeof structuredClone === 'function') {
    return structuredClone(graph)
  }
  return JSON.parse(JSON.stringify(graph)) as GraphSnapshot
}
