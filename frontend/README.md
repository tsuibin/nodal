# Nodal Frontend

Nodal 的前端工作台：React 18 + TypeScript + Vite 5 + Tailwind CSS v4。
它同时承担三件事：品牌首页、资源库/广场、以及产品核心的无限画布 `/canvas/:id`。

> 后端接口契约、错误码、SSE/WS 协议见 [`../docs/API.md`](../docs/API.md)；
> 部署与反向代理见 [`../docs/DEPLOYMENT.md`](../docs/DEPLOYMENT.md)。

## 快速开始

```bash
cd frontend
npm install
npm run dev
```

默认 `http://127.0.0.1:5173`，`/api` 由 Vite 代理到 `http://127.0.0.1:8080`。
后端换了端口时，不要改 `vite.config.ts`，用环境变量覆盖：

```bash
VITE_DEV_API_TARGET=http://127.0.0.1:18088 npm run dev
```

生产环境默认同源部署：前端静态文件与 Go 服务由同一个域名提供，`API_BASE_URL` 保持 `/api`。

## 脚本

| 命令 | 作用 |
|---|---|
| `npm run dev` | Vite 开发服务器，HMR + `/api` 代理 |
| `npm run typecheck` | `tsc --noEmit`，严格类型检查 |
| `npm run build` | 类型检查 + 生产构建（`NODE_OPTIONS=--max-old-space-size=4096`） |
| `npm run preview` | 预览 `dist/` |

`vite build` 输出 `dist/`，并生成 sourcemap；生产建议只保留 `dist` 静态文件，sourcemap 可按需关闭。

## 环境变量

| 变量 | 默认 | 说明 |
|---|---|---|
| `VITE_API_BASE_URL` | `/api` | 前端请求前缀；同源部署保持默认 |
| `VITE_DEV_API_TARGET` | `http://127.0.0.1:8080` | **仅 dev**，Vite 代理目标 |

注意：`VITE_*` 会在构建时写进产物，不要放任何密钥。

## 路由

| 路径 | 页面 | 说明 |
|---|---|---|
| `/` | `HomePage` | 品牌主张、关系类型、最新广场画布 |
| `/library` | `LibraryPage` | 我的画布 / 知识库 / Agent 模板 |
| `/square` | `SquarePage` | 公开广场、标签筛选、点赞、评论 |
| `/canvas/:id` | `CanvasPage` | 无限画布工作台（不在 `AppShell` 内） |
| `/share/:token` | `CanvasPlaceholderPage` | 只读分享页占位，当前没有独立阅读器 |

## 目录

```text
src/
├─ main.tsx                 # ThemeProvider → ToastProvider → ErrorBoundary → AuthProvider → Router
├─ App.tsx                  # 路由表；/canvas/:id 用 React.lazy 懒加载
├─ index.css                # Tailwind v4 + 品牌令牌 + 画布/toast/错误页样式
├─ pages/                   # 首页、资源库、广场、画布、占位页
├─ components/
│  ├─ AppShell.tsx          # 顶栏、导航、登录态、页脚
│  ├─ LoginModal*.tsx       # 登录/注册弹窗与 Provider
│  ├─ CanvasCard.tsx        # 画布卡片
│  ├─ ToastProvider.tsx     # 全局 toast + 未捕获异常监听
│  ├─ ErrorBoundary.tsx     # 渲染异常兜底
│  └─ canvas/               # 节点、边、编辑器、侧面板、推演/分享弹窗
├─ hooks/                   # useAuth、useCanvasList、useCanvasGraph、useInferStream 等
├─ services/                # 每个后端模块一个请求封装；http.ts 统一 axios 实例
├─ types/                   # 与后端 DTO 一一对应的 TS 类型
├─ lib/                     # graph、format、monaco、shiki、katex、notify
└─ store/                   # 画布撤销/重做历史栈
```

## 鉴权与游客

- 用户 token：`localStorage['nodal.auth.token']`；
- 游客 token：`localStorage['nodal.guest.token']`，后端也会通过 `X-Guest-Token` 响应头下发；
- 请求层优先带用户 token，其次游客 token；
- `1001/1002` 会清掉用户 token 并广播 `nodal:auth:unauthorized`，由 `AuthProvider` 回到匿名态；
- 游客访问仅用户能力（知识库、Agent、推演、REPL、分享链接）时，后端返回 `1003`，前端唤起登录弹窗。

## 画布页架构

- `CanvasPage`：加载画布、错误/权限页、`ReactFlowProvider` 与工作台状态；
- `useCanvasGraph`：受控 `nodes/edges`、历史栈、`dirty`；推演追加时同步改 ref，避免流式更新丢快照；
- `lib/graph.ts`：ReactFlow ↔ 后端 JSON 的归一化/序列化、`findFreePosition` 避让；
- `store/`：`CanvasHistory`，上限 60，拖拽只在 `dragStop` 记录真实变化；
- 自动保存：编辑后 1.4s debounce，`Ctrl+S` 立即保存；保存期间有新修改不会误清 `dirty`；
- 推演：`POST /api/canvas/:id/run-infer` 返回 SSE，前端只 append `node`/`edge`，不修改已有内容；
- 快捷键：`Ctrl+Z` / `Ctrl+Shift+Z` / `Delete` / `Esc`，在输入框与 Monaco 内不劫持。

### Monaco 离线打包

`src/lib/monaco.ts` 使用：

```ts
import * as monaco from 'monaco-editor/editor/editor.api'
```

并按需注册 `python / shell / javascript / typescript / go / markdown`。
`?worker` 导入必须写 `monaco-editor/editor/editor.worker?worker`；
直接 `import * as monaco from 'monaco-editor'` 会把 60+ 语言打进构建并导致 Node OOM。

## Demo 画布

- 后端注册成功后自动为 `u:<id>` 创建 2 张 Demo（辩论推演、缓存方案对比）；
- 老账号/游客第一次打开资源库且画布数为 0 时，`LibraryPage` 调用 `POST /api/demo/seed` 兜底；
- 接口幂等：该身份已有画布时 `created=0`；
- Demo 是一等公民画布，可以编辑、推演、Fork、分享，不是前端假数据。

## 全局异常处理

- `ErrorBoundary`：捕获 React 渲染异常，展示刷新/重试/返回资源库；
- `ToastProvider`：右上角 toast，最多 3 条，2 秒内同文案去重；
- `services/http.ts`：网络错误、HTTP 5xx、后端 `5000` 统一发 `nodal:toast`；
- `unhandledrejection` / `window.error`：兜底提示，详细栈写 console；
- 业务参数错误、字段校验仍留在表单内联，避免每个错误都弹 toast。

## 类型约定

- 所有请求走 `request<T>()`，禁止在页面里 `as any`；
- 信封：`ApiEnvelope<T>`；删除接口统一 `DeleteResult`；错误码常量集中在 `types/api.ts`；
- 画布节点 `NodalNodeData.kind`：`text | code | formula | image | file`；
- 关系 `derive | counter | evidence | hypothesis`；AI 节点 `source='agent'`，服务端同时写 `ai_generated` 与 agent 信息。

## 已知边界

- 画布 chunk 约 3.6MB（gzip 约 980KB），靠路由懒加载隔离；后续可把 Monaco 再推迟到打开编辑器时加载；
- `/share/:token` 仍是占位页；
- 知识库挂载是服务端进程内临时关联，重启失效；
- 移动端只做了自适应布局，未做真机触控回归；
- 本机 npm 源为 `https://registry.npmmirror.com`，CI 可按需切回官方源。
