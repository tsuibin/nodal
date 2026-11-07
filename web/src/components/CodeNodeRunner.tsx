/**
 * 代码节点的最小运行面板：运行按钮 → 创建 REPL → WebSocket 回显。
 * 模块 9 的画布页可以直接复用，或把 useRepl 接到 Monaco 的工具栏上。
 */
import type { CSSProperties } from 'react'
import { useRepl } from '../hooks/useRepl'
import type { ReplLanguage } from '../types/repl'

export interface CodeNodeRunnerProps {
  code: string
  language?: ReplLanguage
  title?: string
}

export function CodeNodeRunner({ code, language = 'python', title = '代码运行' }: CodeNodeRunnerProps) {
  const repl = useRepl()
  const busy = repl.status === 'creating' || repl.status === 'connecting' || repl.status === 'running'

  return (
    <div style={panelStyle}>
      <div style={toolbarStyle}>
        <span style={titleStyle}>{title}</span>
        <span style={statusStyle}>{statusText(repl.status, repl.remaining)}</span>
        <button
          type="button"
          style={primaryButtonStyle}
          disabled={busy || !code.trim()}
          onClick={() => void repl.run(code, language)}
        >
          {busy ? '运行中…' : '运行'}
        </button>
        <button type="button" style={buttonStyle} disabled={!busy} onClick={repl.stop}>
          停止
        </button>
      </div>
      {repl.error ? <div style={errorStyle}>{repl.error}</div> : null}
      <pre style={outputStyle}>{repl.output || '输出会显示在这里'}</pre>
      {repl.status === 'closed' ? (
        <div style={footerStyle}>进程已结束{repl.exitCode === null ? '' : `（exit ${repl.exitCode}）`}</div>
      ) : null}
    </div>
  )
}

function statusText(status: string, remaining: number): string {
  switch (status) {
    case 'creating':
      return '创建沙箱…'
    case 'connecting':
      return '连接终端…'
    case 'running':
      return `运行中 · 剩余 ${remaining}s`
    case 'closed':
      return '已结束'
    case 'error':
      return '出错'
    default:
      return '待运行'
  }
}

const panelStyle: CSSProperties = {
  border: '1px solid var(--nodal-border, #e5e7eb)',
  borderRadius: 10,
  background: 'var(--nodal-surface, #ffffff)',
  overflow: 'hidden',
}
const toolbarStyle: CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 8,
  padding: '8px 10px',
  borderBottom: '1px solid var(--nodal-border, #e5e7eb)',
}
const titleStyle: CSSProperties = { fontSize: 13, fontWeight: 600, marginRight: 'auto' }
const statusStyle: CSSProperties = { fontSize: 12, color: 'var(--nodal-muted, #6b7280)' }
const primaryButtonStyle: CSSProperties = {
  padding: '4px 12px',
  borderRadius: 6,
  border: 'none',
  background: 'var(--nodal-primary, #2563eb)',
  color: '#fff',
  cursor: 'pointer',
  fontSize: 12,
}
const buttonStyle: CSSProperties = {
  padding: '4px 10px',
  borderRadius: 6,
  border: '1px solid var(--nodal-border, #d1d5db)',
  background: 'transparent',
  cursor: 'pointer',
  fontSize: 12,
}
const errorStyle: CSSProperties = { padding: '8px 10px', color: 'var(--nodal-danger, #dc2626)', fontSize: 12 }
const outputStyle: CSSProperties = {
  margin: 0,
  padding: 10,
  minHeight: 96,
  maxHeight: 320,
  overflow: 'auto',
  background: 'var(--nodal-code-bg, #0f172a)',
  color: 'var(--nodal-code-fg, #e2e8f0)',
  fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
  fontSize: 12,
  lineHeight: 1.5,
  whiteSpace: 'pre-wrap',
}
const footerStyle: CSSProperties = {
  padding: '4px 10px',
  fontSize: 11,
  color: 'var(--nodal-muted, #6b7280)',
  borderTop: '1px solid var(--nodal-border, #e5e7eb)',
}
