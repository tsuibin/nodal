import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'

import App from './App'
import { ErrorBoundary } from './components/ErrorBoundary'
import { ToastProvider } from './components/ToastProvider'
import { AuthProvider } from './hooks/useAuth'
import { ThemeProvider } from './hooks/useTheme'
import './index.css'
import 'katex/dist/katex.min.css'

/**
 * 应用入口：
 * - ThemeProvider：双主题，读写 localStorage + 跟随系统；
 * - ToastProvider：全局 toast + 未捕获异常监听；
 * - ErrorBoundary：渲染异常兜底；
 * - AuthProvider：JWT/游客会话恢复；
 * - BrowserRouter：/、/library、/square 等页面路由。
 */
ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <ThemeProvider>
      <ToastProvider>
        <ErrorBoundary>
          <AuthProvider>
            <BrowserRouter>
              <App />
            </BrowserRouter>
          </AuthProvider>
        </ErrorBoundary>
      </ToastProvider>
    </ThemeProvider>
  </React.StrictMode>,
)
