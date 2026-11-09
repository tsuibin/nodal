/**
 * 代码节点的最小运行面板：运行按钮 → 创建 REPL → WebSocket 回显。
 *
 * PTY 输出包含 ANSI 光标控制与颜色序列，必须交给终端模拟器渲染；
 * 直接用 <pre> 打印会把 \x1b[?25l、\x1b[4D 之类的控制码显示成乱码。
 */
import { useEffect, useRef } from 'react'
import type { CSSProperties } from 'react'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'

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

  const terminalHostRef = useRef<HTMLDivElement | null>(null)
  const terminalRef = useRef<Terminal | null>(null)
  const writtenRef = useRef('')
  const sendRef = useRef(repl.send)
  const resizeRef = useRef(repl.resize)
  sendRef.current = repl.send
  resizeRef.current = repl.resize

  useEffect(() => {
    const host = terminalHostRef.current
    if (!host) return

    const term = new Terminal({
      convertEol: true,
      cursorBlink: true,
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Consolas, monospace',
      fontSize: 12,
      lineHeight: 1.35,
      scrollback: 2000,
      theme: {
        background: '#0f172a',
        foreground: '#e2e8f0',
        cursor: '#93c5fd',
        selectionBackground: '#334155',
      },
    })
    const fitAddon = new FitAddon()
    term.loadAddon(fitAddon)
    term.open(host)
    terminalRef.current = term

    const fit = () => {
      try {
        fitAddon.fit()
        resizeRef.current(term.cols, term.rows)
      } catch {
        // 容器尚未完成布局时 fit 可能失败，下一次 resize 会重试。
      }
    }
    const dataDisposable = term.onData((data) => sendRef.current(data))
    const observer = new ResizeObserver(fit)
    observer.observe(host)
    requestAnimationFrame(fit)

    return () => {
      observer.disconnect()
      dataDisposable.dispose()
      terminalRef.current = null
      term.dispose()
    }
  }, [])

  useEffect(() => {
    const term = terminalRef.current
    if (!term) return

    // 新一轮运行会把 output 清空；此时终端也清屏，避免上一轮内容残留。
    if (writtenRef.current && !repl.output.startsWith(writtenRef.current)) {
      term.clear()
      writtenRef.current = ''
    }
    if (repl.output.length > writtenRef.current.length) {
      const delta = repl.output.slice(writtenRef.current.length)
      term.write(delta)
      writtenRef.current = repl.output
    }
  }, [repl.output])

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
      <div style={terminalWrapStyle}>
        <div ref={terminalHostRef} style={terminalHostStyle} />
        {!repl.output ? <div style={placeholderStyle}>输出会显示在这里</div> : null}
      </div>
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
const terminalWrapStyle: CSSProperties = {
  position: 'relative',
  height: 260,
  background: '#0f172a',
}
const terminalHostStyle: CSSProperties = {
  position: 'absolute',
  inset: 10,
}
const placeholderStyle: CSSProperties = {
  position: 'absolute',
  inset: 10,
  display: 'flex',
  alignItems: 'flex-start',
  color: '#94a3b8',
  fontSize: 12,
  pointerEvents: 'none',
  fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Consolas, monospace',
}
const footerStyle: CSSProperties = {
  padding: '4px 10px',
  fontSize: 11,
  color: 'var(--nodal-muted, #6b7280)',
  borderTop: '1px solid var(--nodal-border, #e5e7eb)',
}
