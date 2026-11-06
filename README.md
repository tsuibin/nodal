# Nodal

**思维不再是一条直线。**

Nodal 是一个无限画布式 AI 思维推演引擎。它不做"更快的回答"，而是把思考过程本身变成可以摆开、分叉、辩论、复用的空间：

- **Canvas** 无限画布，思路自由摆放
- **Nodes** 文本 / 代码 / 公式 / 文件 / 图片，都是一等公民节点
- **Edges** 带关系标签的连线：推导自、反例质疑、补充佐证、并行假设
- **Multi-Agent** 探索者 / 校验者 / 总结者 / 构建者 四类内置角色同台协作
- **Isolation** 画布默认隔离，知识库必须手动挂载
- **Fork** 任意节点或整张画布可分叉，公共广场可二次创作

> 本仓库按"模块化开发、按模块提交"的方式组织，每个模块对应一次独立 commit。

## 仓库结构

```
nodal/
├── frontend/            # React 18 + TS + Vite + ReactFlow
├── backend/             # Go 1.23 + Gin + GORM
├── docs/                # PRD / 视觉规范 / 演示稿大纲
├── scripts/             # 初始化脚本
├── .github/workflows/   # CI
├── docker-compose.yml   # postgres(pgvector) + redis
├── AI_BUILD_GUIDE.md    # AI 开发总引导词（模块清单与规则）
└── Makefile
```

## 技术栈

| 层 | 选型 |
|---|---|
| 前端 | React 18 / TypeScript / Vite / ReactFlow / Monaco / KaTeX / Tailwind v4 |
| 后端 | Go 1.23 / Gin / GORM |
| 数据库 | PostgreSQL 16 + pgvector |
| 缓存 | Redis 7 |
| 沙箱 | Firecracker microVM + ttyd（受限 Python / Shell） |
| AI 层 | 统一模型适配器（OpenAI / Anthropic / 通义 / Qwen / 本地模型） |

## 本地启动

### 1. 起依赖服务

```bash
cp .env.example .env
docker compose up -d          # postgres + redis
bash scripts/init-db.sh       # 开启 pgvector 扩展
```

### 2. 建库（两种方式，任选其一）

```bash
cd backend
go run ./cmd/migrate --mode=sql     # 推荐：按 backend/migrate/*.up.sql 建库并记录 schema_migrations
go run ./cmd/migrate --mode=auto    # 本地快速迭代：GORM AutoMigrate，自动补齐复合外键/部分索引/HNSW
go run ./cmd/migrate --mode=sql --down   # 回滚最新一版
```

两种方式最终结构一致（9 张表 + 4 个内置 Agent 种子）；`--mode=sql` 支持版本记录与回滚，线上用这一条。

### 3. 起后端

```bash
cd backend
go mod download
go run ./cmd/server           # 默认 :8080
curl http://localhost:8080/healthz
```

### 4. 起前端

```bash
cd frontend
npm install
npm run dev                   # 默认 :5173，已代理 /api → :8080
```

## 模块进度

| # | 模块 | 状态 |
|---|---|---|
| 0 | 脚手架 & 基础依赖 | ✅ |
| 1 | 数据库迁移脚本 | ✅ |
| 2 | 用户 & 鉴权 | ⬜ |
| 3 | 画布 CRUD / 节点边 | ⬜ |
| 4 | Agent 模板 & 模型调度 | ⬜ |
| 5 | 知识库 & RAG | ⬜ |
| 6 | 广场 / 分享 / Fork | ⬜ |
| 7 | REPL 沙箱 | ⬜ |
| 8 | 前端 首页 / 资源库 / 广场 | ⬜ |
| 9 | 核心画布页 | ⬜ |
| 10 | 主题系统 & 联调 & Demo | ⬜ |

## 文档

- [产品 PRD](docs/PRD.md)
- [视觉规范](docs/visual-spec.md)
- [演示稿大纲](docs/ppt-outline.md)
- [AI 开发引导词](AI_BUILD_GUIDE.md)
