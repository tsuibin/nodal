# AI_BUILD_GUIDE

本文件约定 AI 在本仓库中的开发、分支、提交与交付方式。

## 1. 仓库与分支

- 主分支：`main`，始终保持可构建、可运行。
- 每个模块从 `main` 切出独立分支：`feature/module-<编号>-<短名>`，例如 `feature/module-7-repl`。
- 模块代码、测试、文档全部在该分支提交，自测通过后合并回 `main`。
- 合并使用 `git merge --no-ff`，保留模块边界；不使用 `push --force`，不改写已合并历史。

当前状态：

```text
main：模块 1–10 已完成；目录已对齐目标 monorepo 结构
历史模块分支：feature/module-7-repl … feature/module-10-wrap-up
结构重构分支：refactor/final-layout（backend/ + replsandbox + scripts + CI + migrate）
```

仓库在模块 7 阶段初始化，因此首个功能提交同时导入了模块 1–6 的既有代码；之后每个模块严格按上述流程执行。

## 2. 提交信息

格式：

```text
feat: 完成模块7 REPL 隔离沙箱
fix: 修复 REPL 会话超时后进程组残留
docs: 补充模块7 Firecracker launcher 合约
test: 增加 REPL 网络隔离集成测试
chore: 更新依赖与构建脚本
```

一个提交只做一件事；模块级提交必须包含可运行代码、测试与文档。

## 3. 目录约定

当前可运行结构（模块 1–10 已落盘，目录已对齐目标树）：

```text
nodal/
├── backend/                 # Go 后端（module github.com/nodal/nodal-server）
│   ├── cmd/server/          # 入口
│   ├── handler/             # HTTP 适配层
│   ├── service/             # 业务编排
│   ├── model/               # GORM 模型
│   ├── middleware/          # JWT / 错误出口
│   ├── replsandbox/         # 模块7 沙箱（local / firecracker）
│   ├── router/              # 路由分组
│   ├── migrate/             # PostgreSQL 初始化 SQL + 用法说明
│   ├── config/ db/ embed/ llm/ vector/ token/ response/ apperr/ secretbox/
│   └── go.mod
├── frontend/                # React + TS 前端
│   ├── README.md
│   ├── package.json
│   └── src/
│       ├── pages/           # 首页 / 资源库 / 广场 / 画布
│       ├── components/      # 通用组件 + canvas/
│       ├── hooks/           # 数据 / 流式 / 鉴权 hook
│       ├── services/        # HTTP / SSE / WS 封装
│       └── store/           # 画布撤销重做
├── docs/
│   ├── PRD.md               # 产品需求（当前实现的契约化整理）
│   ├── visual-spec.md       # 视觉规范
│   ├── ppt-outline.md       # 汇报大纲
│   ├── API.md               # 接口契约
│   └── DEPLOYMENT.md        # 部署文档
├── scripts/                 # 本地启动 / 测试 / 数据库初始化
├── deploy/                  # Dockerfile.backend / Dockerfile.frontend / nginx.conf
├── .github/workflows/       # CI：后端 vet/test + 前端 typecheck/build
├── docker-compose.yml       # 参考编排（未真机验证）
├── AI_BUILD_GUIDE.md
└── README.md
```

说明：Go module 名仍为 `github.com/nodal/nodal-server`，只改物理目录，不改 import 前缀；
`backend/` 下各包已提升到根级（不再有 `internal/`），沙箱目录与包名统一为 `replsandbox`。

## 4. 模块状态

| 模块 | 内容 | 状态 | 分支 |
|---|---|---|---|
| 1 | 项目骨架 / 配置 / 响应封装 | 已完成 | 基线 |
| 2 | 用户、注册登录、JWT、游客 | 已完成 | 基线 |
| 3 | 画布 CRUD / Fork / 权限 | 已完成 | 基线 |
| 4 | Agent 模板 / 模型适配 / SSE 推演 | 已完成 | 基线 |
| 5 | 知识库 / 切片 / 向量 / RAG 挂载 | 已完成 | 基线 |
| 6 | 广场 / 点赞评论 / 分享链接 | 已完成 | 基线 |
| 7 | REPL 隔离沙箱 / WS 终端 / 前端运行面板 | 已完成 | `feature/module-7-repl` |
| 8 | 前端首页 / 资源库 / 广场 | 已完成 | `feature/module-8-frontend` |
| 9 | 画布页 / React Flow / 推演交互 | 已完成 | `feature/module-9-canvas` |
| 10 | Demo 画布 / Toast / 类型 / 文档 / 部署 | 已完成 | `feature/module-10-wrap-up` |

## 5. 本地校验

```bash
# 后端
cd backend
gofmt -w internal cmd
go vet ./...
go test ./...

# 前端
cd ../frontend
npm install
npm run typecheck
```

模块 7 的 `internal/repl` 集成测试会真实启动 `unshare` + PTY；在不支持用户命名空间的 CI 环境会自动 `t.Skip`。

## 6. 交付约定

- 每个模块结束交付两份文件：`README.md`（说明、接口、配置、自测、边界）+ `nodal-moduleN.zip`（完整代码）。
- 先删除旧 zip，再打包，避免重复交付。
- 打包排除：`frontend/node_modules`、`backend/data`、`.env`、`dist`、`*.db`、`.git`。
- 无法真机验证的能力（Firecracker、pgvector、真实模型 Key）必须在 README 的「已知边界」中如实标注，不得宣称已验证。
