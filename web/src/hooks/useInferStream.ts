/**
 * useInferStream：把 SSE 推演流接到 React Flow 的增量渲染。
 *
 * 用法（节点/边直接 append，绝不覆盖已有内容）：
 *
 *   const infer = useInferStream({
 *     onNode: (node) => setNodes((prev) => [...prev, node as Node]),
 *     onEdge: (edge) => setEdges((prev) => [...prev, edge as Edge]),
 *   })
 *   await infer.run(canvasId, { selected_node_ids: ['n1'], agent_id: 3 })
 *
 * hook 本身也维护了一份 generatedNodes / generatedEdges，
 * 不接 React Flow 也可以先渲染到一个独立预览层。
 */
import { useCallback, useEffect, useRef, useState } from 'react'
import { streamRunInfer } from '../api/infer'
import { ApiError } from '../api/http'
import type {
  InferDonePayload,
  InferSourcesEvent,
  InferStartEvent,
  InferWarningEvent,
  RFEdge,
  RFNode,
  RunInferPayload,
  UseInferStreamOptions,
} from '../types/infer'

export interface UseInferStreamResult {
  running: boolean
  error: ApiError | Error | null
  /** 本次推演新增的节点（只增不改） */
  generatedNodes: RFNode[]
  /** 本次推演新增的连线 */
  generatedEdges: RFEdge[]
  warnings: InferWarningEvent[]
  /** 本次推演召回到的知识库片段；没有挂载知识库时为 null */
  sources: InferSourcesEvent | null
  start: InferStartEvent | null
  done: InferDonePayload | null
  run: (canvasId: number, payload: RunInferPayload) => Promise<InferDonePayload | null>
  cancel: () => void
  reset: () => void
}

export function useInferStream(options: UseInferStreamOptions = {}): UseInferStreamResult {
  const [running, setRunning] = useState(false)
  const [error, setError] = useState<ApiError | Error | null>(null)
  const [generatedNodes, setGeneratedNodes] = useState<RFNode[]>([])
  const [generatedEdges, setGeneratedEdges] = useState<RFEdge[]>([])
  const [warnings, setWarnings] = useState<InferWarningEvent[]>([])
  const [sources, setSources] = useState<InferSourcesEvent | null>(null)
  const [start, setStart] = useState<InferStartEvent | null>(null)
  const [done, setDone] = useState<InferDonePayload | null>(null)

  const optionsRef = useRef(options)
  optionsRef.current = options

  const abortRef = useRef<AbortController | null>(null)
  const mountedRef = useRef(true)

  useEffect(() => {
    mountedRef.current = true
    return () => {
      mountedRef.current = false
      abortRef.current?.abort()
    }
  }, [])

  const cancel = useCallback(() => {
    abortRef.current?.abort()
    abortRef.current = null
    if (mountedRef.current) setRunning(false)
  }, [])

  const reset = useCallback(() => {
    setGeneratedNodes([])
    setGeneratedEdges([])
    setWarnings([])
    setSources(null)
    setStart(null)
    setDone(null)
    setError(null)
  }, [])

  const run = useCallback(
    async (canvasId: number, payload: RunInferPayload): Promise<InferDonePayload | null> => {
      // 同一时刻只保留一条流，避免两条推演把同一片区域刷乱
      abortRef.current?.abort()
      const controller = new AbortController()
      abortRef.current = controller

      reset()
      setRunning(true)

      const isActive = () => !controller.signal.aborted && abortRef.current === controller
      const opts = optionsRef.current

      try {
        const result = await streamRunInfer(
          canvasId,
          payload,
          {
            onStart: (payload) => {
              if (!isActive()) return
              setStart(payload)
              opts.onStart?.(payload)
            },
            onSources: (payload) => {
              if (!isActive()) return
              setSources(payload)
              opts.onSources?.(payload)
            },
            onNode: (event) => {
              if (!isActive()) return
              setGeneratedNodes((prev) => [...prev, event.node])
              opts.onNode?.(event)
            },
            onEdge: (event) => {
              if (!isActive()) return
              setGeneratedEdges((prev) => [...prev, event.edge])
              opts.onEdge?.(event)
            },
            onWarning: (payload) => {
              if (!isActive()) return
              // 只保留最近 20 条，避免异常模型把内存刷爆
              setWarnings((prev) => [...prev.slice(-19), payload])
              opts.onWarning?.(payload)
            },
          },
          controller.signal,
        )

        if (!isActive()) return null
        setDone(result)
        opts.onDone?.(result)
        return result
      } catch (err) {
        // 主动取消不算错误
        if (controller.signal.aborted) return null
        const normalized = err instanceof Error ? err : new Error(String(err))
        if (mountedRef.current && abortRef.current === controller) {
          setError(normalized)
          opts.onError?.(normalized)
        }
        return null
      } finally {
        if (abortRef.current === controller) {
          abortRef.current = null
          if (mountedRef.current) setRunning(false)
        }
      }
    },
    [reset],
  )

  return {
    running,
    error,
    generatedNodes,
    generatedEdges,
    warnings,
    sources,
    start,
    done,
    run,
    cancel,
    reset,
  }
}
