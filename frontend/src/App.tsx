/**
 * 路由表：
 * /          首页
 * /library   资源库（我的画布 / 知识库 / Agent 模板）
 * /square    公共广场
 * /canvas/:id 画布页占位，模块 9 接入 ReactFlow
 * /share/:token 分享页占位，模块 10 接入
 */
import { Route, Routes } from 'react-router-dom'
import { AppShell } from './components/AppShell'
import { LoginModalProvider } from './components/LoginModalProvider'
import { HomePage } from './pages/HomePage'
import { LibraryPage } from './pages/LibraryPage'
import { SquarePage } from './pages/SquarePage'
import { CanvasPlaceholderPage, NotFoundPage } from './pages/PlaceholderPage'

export default function App() {
  return (
    <LoginModalProvider>
      <Routes>
        <Route element={<AppShell />}>
          <Route index element={<HomePage />} />
          <Route path="library" element={<LibraryPage />} />
          <Route path="square" element={<SquarePage />} />
          <Route path="canvas/:id" element={<CanvasPlaceholderPage />} />
          <Route path="share/:token" element={<CanvasPlaceholderPage />} />
          <Route path="*" element={<NotFoundPage />} />
        </Route>
      </Routes>
    </LoginModalProvider>
  )
}
