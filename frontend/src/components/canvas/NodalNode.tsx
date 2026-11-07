/**
 * 画布节点的统一渲染组件。
 *
 * 所有节点都用同一个 React Flow type = "nodal"，通过 data.kind 区分
 * 文本/代码/公式/图片/文件；来源、结论状态叠加在样式上：
 *   - 用户节点：中性卡片
 *   - AI 节点：品牌色虚线边框 + AI 徽标
 *   - 结论节点：加粗边框 + 「结论」徽标
 */
import { useEffect, useState } from 'react'
import { Handle, Position, type NodeProps } from '@xyflow/react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { useTheme } from '../../hooks/useTheme'
import { highlightCode } from '../../lib/shiki'
import { renderFormula } from '../../lib/katex'
import { RELATION_META } from '../../types/canvas-node'
import type { NodalNode } from '../../types/canvas-node'
import {
  BotIcon,
  CodeIcon,
  FileIcon,
  FormulaIcon,
  ImageIcon,
  TextIcon,
} from '../icons'

const KIND_ICON = {
  text: TextIcon,
  code: CodeIcon,
  formula: FormulaIcon,
  image: ImageIcon,
  file: FileIcon,
} as const

export function NodalNodeCard({ data, selected }: NodeProps<NodalNode>) {
  const { resolved } = useTheme()
  const isAgent = data.source === 'agent'
  const isConclusion = data.status === 'conclusion'
  const KindIcon = KIND_ICON[data.kind] ?? TextIcon

  return (
    <div
      className={[
        'nodal-node',
        selected ? 'nodal-node-selected' : '',
        isAgent ? 'nodal-node-agent' : '',
        isConclusion ? 'nodal-node-conclusion' : '',
      ].join(' ')}
      data-kind={data.kind}
    >
      <Handle type="target" position={Position.Left} className="nodal-handle nodal-handle-target" />
      <div className="nodal-node-head">
        <span className="nodal-node-kind">
          <KindIcon width={14} height={14} />
        </span>
        <span className="nodal-node-label" title={data.label}>
          {data.label}
        </span>
        {isConclusion ? <span className="nodal-badge nodal-badge-conclusion">结论</span> : null}
        {isAgent ? (
          <span className="nodal-badge nodal-badge-agent">
            <BotIcon width={12} height={12} />
            AI
          </span>
        ) : null}
      </div>

      <div className="nodal-node-body">
        <NodeBody data={data} dark={resolved === 'dark'} />
      </div>

      {isAgent && data.agent_id ? (
        <div className="nodal-node-foot">
          <span>Agent #{data.agent_id}</span>
          {data.agent_name ? <span className="truncate">{data.agent_name}</span> : null}
        </div>
      ) : null}

      <Handle type="source" position={Position.Right} className="nodal-handle nodal-handle-source" />
    </div>
  )
}

function NodeBody({ data, dark }: { data: NodalNode['data']; dark: boolean }) {
  switch (data.kind) {
    case 'code':
      return <CodePreview code={data.code ?? ''} language={data.language} dark={dark} />
    case 'formula':
      return (
        <div
          className="nodal-node-formula"
          // KaTeX 输出由 library 内部生成，trust=false，不会执行用户脚本
          dangerouslySetInnerHTML={{ __html: renderFormula(data.formula ?? '', true) }}
        />
      )
    case 'image':
      return data.imageUrl ? (
        <div className="nodal-node-image">
          <img src={data.imageUrl} alt={data.label} loading="lazy" />
        </div>
      ) : (
        <NodeEmpty text="双击节点，填入图片 URL 或上传图片" />
      )
    case 'file':
      return data.fileName ? (
        <div className="nodal-node-file">
          <FileIcon width={20} height={20} />
          <div className="min-w-0">
            <p className="truncate font-medium">{data.fileName}</p>
            <p className="text-[10px] text-[var(--nodal-muted)]">
              {data.fileType || '未知类型'}
              {data.fileSize ? ` · ${formatBytes(data.fileSize)}` : ''}
            </p>
          </div>
        </div>
      ) : (
        <NodeEmpty text="双击节点，记录文件名与类型" />
      )
    default:
      return data.content ? (
        <div className="nodal-node-markdown">
          <ReactMarkdown remarkPlugins={[remarkGfm]}>{data.content}</ReactMarkdown>
        </div>
      ) : (
        <NodeEmpty text="双击节点，写下判断或粘贴材料" />
      )
  }
}

function NodeEmpty({ text }: { text: string }) {
  return <p className="nodal-node-empty">{text}</p>
}

function CodePreview({ code, language, dark }: { code: string; language?: string; dark: boolean }) {
  const [html, setHtml] = useState<string>('')
  const theme = dark ? 'github-dark' : 'github-light'

  useEffect(() => {
    let cancelled = false
    if (!code.trim()) {
      setHtml('')
      return
    }
    void highlightCode(code, language, theme).then((result) => {
      if (!cancelled) setHtml(result)
    })
    return () => {
      cancelled = true
    }
  }, [code, language, theme])

  if (!code.trim()) return <NodeEmpty text="双击节点，写一段可运行代码" />
  if (!html) {
    return (
      <pre className="nodal-node-code">
        <code>{code.slice(0, 420)}</code>
      </pre>
    )
  }
  return (
    <div
      className="nodal-node-code shiki-preview"
      // Shiki 输出是转义后的 HTML，不含用户可执行脚本
      dangerouslySetInnerHTML={{ __html: html }}
    />
  )
}

function formatBytes(size: number): string {
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / (1024 * 1024)).toFixed(1)} MB`
}

/** 关系标签的展示文案，供节点未使用，导出给面板/边组件共用。 */
export function relationLabel(relation: keyof typeof RELATION_META): string {
  return RELATION_META[relation].label
}

/** 将节点尺寸调整为内容自适应后仍保持稳定的最小宽度。 */
export const NODAL_NODE_MIN_WIDTH = 248
