/**
 * Shiki 高亮单例。
 *
 * 只注册画布代码节点用得到的语言和双主题，避免把全量语言包打进首屏；
 * 高亮结果按 lang + theme + code 缓存，节点拖动时不会反复初始化。
 */
import { createHighlighter, type Highlighter } from 'shiki'

const LANGS = ['python', 'bash', 'javascript', 'typescript', 'json', 'go', 'shellscript'] as const
const THEMES = ['github-light', 'github-dark'] as const

let highlighterPromise: Promise<Highlighter> | null = null
const cache = new Map<string, string>()

export type HighlightTheme = (typeof THEMES)[number]

function getHighlighter(): Promise<Highlighter> {
  if (!highlighterPromise) {
    highlighterPromise = createHighlighter({
      themes: [...THEMES],
      langs: [...LANGS],
    })
  }
  return highlighterPromise
}

export function normalizeLang(lang: string | undefined): string {
  const value = (lang ?? 'python').toLowerCase()
  if (value === 'sh' || value === 'shell' || value === 'zsh') return 'bash'
  if (value === 'py') return 'python'
  if (value === 'js') return 'javascript'
  if (value === 'ts') return 'typescript'
  return LANGS.includes(value as (typeof LANGS)[number]) ? value : 'text'
}

export async function highlightCode(
  code: string,
  lang: string | undefined,
  theme: HighlightTheme,
): Promise<string> {
  const language = normalizeLang(lang)
  const key = `${theme}:${language}:${code}`
  const cached = cache.get(key)
  if (cached) return cached

  try {
    const highlighter = await getHighlighter()
    const html = highlighter.codeToHtml(code, {
      lang: language,
      theme,
    })
    // 只缓存最近 30 条，防止长时间编辑把内存刷高
    if (cache.size > 30) cache.clear()
    cache.set(key, html)
    return html
  } catch {
    return `<pre class="shiki-fallback"><code>${escapeHTML(code)}</code></pre>`
  }
}

function escapeHTML(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}
