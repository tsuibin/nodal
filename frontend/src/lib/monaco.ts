/**
 * Monaco 本地加载配置。
 *
 * 默认 @monaco-editor/react 会从 jsDelivr 拉编辑器，公网不可用时节点编辑器
 * 会一直转圈。这里直接 import monaco-editor，并把 editor worker 用 Vite 的
 * ?worker 方式打到本地包，离线环境也能编辑。
 */
// editor.api 只带编辑器核心，不带 monaco-editor 全量语言包；
// 需要的语言在下面按需引入，避免 Vite 打包时把 60+ 种语言全塞进主 chunk。
import * as monaco from 'monaco-editor/editor/editor.api'
import 'monaco-editor/languages/definitions/python/register'
import 'monaco-editor/languages/definitions/shell/register'
import 'monaco-editor/languages/definitions/javascript/register'
import 'monaco-editor/languages/definitions/typescript/register'
import 'monaco-editor/languages/definitions/go/register'
import 'monaco-editor/languages/definitions/markdown/register'
import { loader } from '@monaco-editor/react'
import EditorWorker from 'monaco-editor/editor/editor.worker?worker'

const globalScope = globalThis as unknown as {
  MonacoEnvironment?: { getWorker: () => Worker }
}

globalScope.MonacoEnvironment = {
  getWorker: () => new EditorWorker(),
}

loader.config({ monaco })

export { monaco }
