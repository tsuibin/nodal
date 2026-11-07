# AI_BUILD_GUIDE

本文件约定 AI 在本仓库中的开发、分支、提交与交付方式。

## 1. 仓库与分支

- 主分支：`main`，始终保持可构建、可运行。
- 每个模块从 `main` 切出独立分支：`feature/module-<编号>-<短名>`，例如 `feature/module-7-repl`。
- 模块代码、测试、文档全部在该分支提交，自测通过后合并回 `main`。
- 合并使用 `git merge --no-ff`，保留模块边界；不使用 `push --force`，不改写已合并历史。

当前状态：

```text
main
└── merge: 模块7 REPL 隔离沙箱
    └── feature/module-7-repl
        └── feat: 完成模块7 REPL 隔离沙箱（含既有模块基线导入）
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

当前可运行结构（模块 1–7 已落盘）：

```text
nodal/
├── server/              # Go 后端（module github.com/nodal/nodal-server）
│   ├── cmd/server/      # 入口
│   └── internal/
│       ├── handler/     # HTTP 适配层
│       ├── service/     # 业务编排
│       ├── model/       # GORM 模型
│       ├── middleware/  # JWT / 错误出口
│       ├── repl/        # 模块7 沙箱（目标树中的 replsandbox）
│       ├── router/      # 路由分组
│       └── ...
├── web/                 # React + TS 前端
│   └── src/
│       ├── api/         # 请求封装（目标树中的 services）
│       ├── hooks/       # 数据/流式 hook
│       ├── types/       # 接口 TS 类型
│       └── components/  # UI 组件
└── README.md
```

目标产品树（`backend/`、`frontend/`、`docs/`、`scripts/`、`docker-compose.yml`）会在模块 8–10 逐步补齐。
**不建议在模块开发中途整体改目录**：Go import 路径、前端相对路径、README 与部署命令都会同时变化。
如确需严格迁移，单独开 `refactor/migrate-layout` 分支，先完成构建/测试再合并。

映射关系：

| 目标树 | 当前位置 | 说明 |
|---|---|---|
| `backend/handler` | `server/internal/handler` | 保持 internal 包边界 |
| `backend/service` | `server/internal/service` | 同上 |
| `backend/model` | `server/internal/model` | 同上 |
| `backend/middleware` | `server/internal/middleware` | 同上 |
| `backend/migrate` | `server/internal/db` | `AutoMigrate` 入口 |
| `backend/replsandbox` | `server/internal/repl` | 模块7 沙箱 |
| `frontend/src/services` | `web/src/api` | HTTP/SSE/WS 封装 |
| `frontend/src/pages` | 模块 8/9 新建 | 首页/资源库/广场/画布 |
| `frontend/src/components` | `web/src/components` | 复用组件 |
| `frontend/src/hooks` | `web/src/hooks` | 数据 hook |
| `frontend/src/store` | 模块 9 新建 | 画布撤销重做/选择状态 |

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
| 8 | 前端首页 / 资源库 / 广场 | 待开始 | - |
| 9 | 画布页 / React Flow / 推演交互 | 待开始 | - |
| 10 | Demo 画布 / 联调 / 部署文档 | 待开始 | - |

## 5. 本地校验

```bash
# 后端
cd server
gofmt -w internal cmd
go vet ./...
go test ./...

# 前端
cd ../web
npm install
npx tsc --noEmit
```

模块 7 的 `internal/repl` 集成测试会真实启动 `unshare` + PTY；在不支持用户命名空间的 CI 环境会自动 `t.Skip`。

## 6. 交付约定

- 每个模块结束交付两份文件：`README.md`（说明、接口、配置、自测、边界）+ `nodal-moduleN.zip`（完整代码）。
- 先删除旧 zip，再打包，避免重复交付。
- 打包排除：`web/node_modules`、`server/data`、`.env`、`dist`、`*.db`、`.git`。
- 无法真机验证的能力（Firecracker、pgvector、真实模型 Key）必须在 README 的「已知边界」中如实标注，不得宣称已验证。
