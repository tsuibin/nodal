/**
 * 路由表：
 * /          首页
 * /library   资源库（我的画布 / 知识库 / Agent 模板）
 * /square    公共广场
 * /canvas/:id 画布页（模块 9）
 * /share/:token 分享页占位，模块 10 接入
 */
import { lazy, Suspense } from 'react'
import { Route, Routes } from 'react-router-dom'
import { AppShell } from './components/AppShell'
import { LoginModalProvider } from './components/LoginModalProvider'
import { HomePage } from './pages/HomePage'
import { LibraryPage } from './pages/LibraryPage'
import { SquarePage } from './pages/SquarePage'
import { CanvasPlaceholderPage, NotFoundPage } from './pages/PlaceholderPage'
// 画布页带着 ReactFlow / Monaco / Shiki / KaTeX，按需加载，避免首页首屏就下载 3MB+
const CanvasPage = lazy(() => import('./pages/CanvasPage').then((module) => ({ default: module.CanvasPage })))

export default function App() {
  return (
    <LoginModalProvider>
      <Routes>
        {/* 画布页是沉浸式全屏工作台，不套首页/页脚外壳 */}
        <Route
          path="canvas/:id"
          element={
            <Suspense
              fallback={
                <div className="flex min-h-screen items-center justify-center bg-[var(--nodal-bg)]">
                  <span className="nodal-spinner" />
                </div>
              }
            >
              <CanvasPage />
            </Suspense>
          }
        />
        <Route element={<AppShell />}>
          <Route index element={<HomePage />} />
          <Route path="library" element={<LibraryPage />} />
          <Route path="square" element={<SquarePage />} />
          <Route path="share/:token" element={<CanvasPlaceholderPage />} />
          <Route path="*" element={<NotFoundPage />} />
        </Route>
      </Routes>
    </LoginModalProvider>
  )
}
