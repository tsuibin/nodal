/**
 * 模块 10：渲染异常兜底。
 *
 * React 的渲染错误不会冒泡到 window.error，必须由 ErrorBoundary 捕获；
 * 捕获后展示可恢复页面，并把错误写入 console 便于排查。
 */
import { Component, type ErrorInfo, type ReactNode } from 'react'
import { notifyToast } from '../lib/notify'

interface Props {
  children: ReactNode
}

interface State {
  error: Error | null
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    console.error('[Nodal] uncaught render error', error, info.componentStack)
    notifyToast({ kind: 'error', message: '页面发生异常，已记录日志' })
  }

  private reset = () => {
    this.setState({ error: null })
  }

  render() {
    const { error } = this.state
    if (!error) return this.props.children

    return (
      <main className="nodal-error-page">
        <p className="nodal-error-eyebrow">SOMETHING BROKE</p>
        <h1>这一步走岔了</h1>
        <p className="nodal-error-desc">
          页面渲染遇到异常。你的画布数据保存在服务端，刷新后通常可以继续。
        </p>
        <pre className="nodal-error-detail">{error.message}</pre>
        <div className="nodal-error-actions">
          <button type="button" className="nodal-error-primary" onClick={() => window.location.reload()}>
            刷新页面
          </button>
          <button type="button" className="nodal-error-secondary" onClick={this.reset}>
            重试渲染
          </button>
          <a className="nodal-error-secondary" href="/library">
            返回资源库
          </a>
        </div>
      </main>
    )
  }
}
