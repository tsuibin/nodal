/**
 * Shiki 代码块。节点预览、编辑器预览和运行面板共用。
 */
import { useEffect, useState } from 'react'
import { useTheme } from '../../hooks/useTheme'
import { highlightCode } from '../../lib/shiki'

export function ShikiCode({
  code,
  language,
  className = '',
  empty = '暂无代码',
}: {
  code: string
  language?: string
  className?: string
  empty?: string
}) {
  const { resolved } = useTheme()
  const [html, setHtml] = useState('')
  const theme = resolved === 'dark' ? 'github-dark' : 'github-light'

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

  if (!code.trim()) return <p className="nodal-node-empty">{empty}</p>
  if (!html) {
    return (
      <pre className={`nodal-code-fallback ${className}`.trim()}>
        <code>{code}</code>
      </pre>
    )
  }
  return (
    <div
      className={`shiki-preview ${className}`.trim()}
      // Shiki 输出是转义后的 HTML
      dangerouslySetInnerHTML={{ __html: html }}
    />
  )
}
