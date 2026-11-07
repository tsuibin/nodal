/**
 * KaTeX 渲染。公式来自用户输入，throwOnError=false 保证非法表达式只显示
 * 红色原文，而不是让整个画布崩溃；trust=false 禁止 \href 等危险命令。
 */
import katex from 'katex'

export function renderFormula(expression: string, displayMode = true): string {
  const source = expression.trim()
  if (!source) {
    return '<span class="nodal-formula-empty">输入公式后会在这里实时预览</span>'
  }
  try {
    return katex.renderToString(source, {
      displayMode,
      throwOnError: false,
      strict: false,
      trust: false,
      output: 'html',
    })
  } catch {
    return `<span class="nodal-formula-error">${escapeHTML(source)}</span>`
  }
}

function escapeHTML(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}
