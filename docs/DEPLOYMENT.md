# Nodal 部署

Nodal 是 monorepo：Go 后端在 `backend/`，React 前端在 `frontend/`。
支持三种运行方式：本地 SQLite（开发/演示）、本地 PostgreSQL + pgvector（预发）、Docker Compose（参考编排）。

> Go module 名为 `github.com/nodal/nodal-server`，代码位于 `backend/`；部署命令以本文为准。

## 1. 环境要求

| 组件 | 版本 | 说明 |
|---|---|---|
| Go | 1.23+ | 本机工具链 1.24.1 也可，构建时加 `GOTOOLCHAIN=local` |
| Node.js | 22+ / npm 10+ | 前端构建 |
| 数据库 | SQLite（默认）或 PostgreSQL 16 + pgvector | 生产推荐 PostgreSQL |
| 反向代理 | Nginx / Caddy | 同源托管前端并代理 `/api`，SSE/WS 必须关闭缓冲 |
| REPL | Linux + 用户命名空间 | 容器内通常受限；无隔离能力时关闭 REPL |

Redis 当前**没有接入**：没有缓存层依赖，部署不需要 Redis。

## 2. 本地开发（SQLite，最快跑通）

```bash
# 终端 1：后端
cd backend
GOTOOLCHAIN=local \
NODAL_HTTP_ADDR=:8080 \
NODAL_DB_DRIVER=sqlite \
NODAL_DB_DSN=data/nodal.db \
NODAL_JWT_SECRET=dev-secret-at-least-32-characters \
NODAL_STORAGE_DIR=data/uploads \
NODAL_EMBED_MODEL=mock/embed \
go run ./cmd/server

# 终端 2：前端
cd frontend
npm install
VITE_DEV_API_TARGET=http://127.0.0.1:8080 npm run dev
```

打开 `http://127.0.0.1:5173`。SQLite 模式会自动 `AutoMigrate` 建表，无需手工迁移。
注册新账号后会自动创建两张 Demo 画布。

## 3. 生产配置

### 3.1 必填环境变量

| 变量 | 示例 | 说明 |
|---|---|---|
| `NODAL_ENV` | `prod` | 设为 prod 后强制校验密钥长度、关闭 gin debug |
| `NODAL_HTTP_ADDR` | `:8080` | 后端监听地址 |
| `NODAL_DB_DRIVER` | `postgres` | `sqlite` / `mysql` / `postgres` |
| `NODAL_DB_DSN` | `host=db user=nodal password=... dbname=nodal port=5432 sslmode=disable` | PostgreSQL 连接串 |
| `NODAL_JWT_SECRET` | 随机 ≥32 字符 | 生产必须显式设置，不能用默认值 |
| `NODAL_ENC_KEY` | 32 字节 base64 或 64 位 hex | 用户 API Key 的 AES-256-GCM 密钥，生产必须独立配置 |
| `NODAL_STORAGE_DIR` | `/app/data/uploads` | 知识库上传文件目录，需持久化卷 |
| `NODAL_CORS_ORIGINS` | `https://nodal.example.com` | 浏览器 Origin 白名单；同源部署也要写实际域名 |

生成密钥：

```bash
openssl rand -base64 32     # NODAL_JWT_SECRET（也可作 ENC_KEY 的原值）
openssl rand -base64 32     # NODAL_ENC_KEY
```

`NODAL_ENC_KEY` 一经上线不要更换，否则已保存的用户 API Key 全部无法解密。

### 3.2 可选配置

| 变量 | 默认 | 说明 |
|---|---|---|
| `NODAL_JWT_ISSUER` | `nodal` | JWT issuer |
| `NODAL_USER_TOKEN_TTL` | `24h` | 用户 token 有效期 |
| `NODAL_GUEST_TOKEN_TTL` | `72h` | 游客会话有效期 |
| `NODAL_GUEST_ENABLED` | `true` | 是否允许游客临时画布 |
| `NODAL_LLM_TIMEOUT` | `2m` | 单次推演总超时 |
| `NODAL_INFER_MAX_NODES` | `50` | 单次推演新增节点上限（1–500） |
| `NODAL_UPLOAD_MAX_BYTES` | `10485760` | 单文件 10MB |
| `NODAL_CHUNK_RUNES` / `NODAL_CHUNK_OVERLAP` | `800` / `100` | 文本切片粒度 |
| `NODAL_EMBED_MODEL` | `mock/embed` | 生产应换成真实 embedding 模型 |
| `NODAL_EMBED_DIM` | `1536` | 必须与 pgvector 维度一致；改维度需要重建向量列 |
| `NODAL_RAG_TOP_K` | `5` | 默认召回片段数 |
| `NODAL_MOUNT_MAX_FILES` | `5` | 单画布挂载文件上限 |
| `NODAL_REPL_ENABLED` | `true` | 生产建议按宿主机隔离能力决定 |
| `NODAL_REPL_RUNTIME` | `local` | `local` 或 `firecracker` |
| `NODAL_REPL_TIMEOUT` | `30s` | 5–30s，硬超时 |
| `NODAL_REPL_MEMORY_MB` | `256` | 64–1024 |
| `NODAL_REPL_MAX_SESSIONS` / `NODAL_REPL_MAX_PER_USER` | `20` / `2` | 并发配额 |
| `NODAL_REPL_DISK_MB` | `32` | 会话 tmpfs |
| `NODAL_REPL_FIRECRACKER_LAUNCHER` | 空 | `runtime=firecracker` 时必填 |

完整定义见 `backend/config/config.go`；不在表里的变量不要臆造。

## 4. PostgreSQL + pgvector

```sql
CREATE DATABASE nodal;
\c nodal
CREATE EXTENSION IF NOT EXISTS vector;
```

可以直接执行仓库内的初始化脚本（推荐空库）：

```bash
NODAL_DB_DSN='postgres://nodal:pass@127.0.0.1:5432/nodal' ./scripts/db-init.sh
```

等价 SQL：`psql "$NODAL_DB_DSN" -f backend/migrate/0001_init.sql`。

后端启动时仍会 `AutoMigrate`，并在 `vector.EnsurePGVectorSchema` 中补齐 `vector(N)` 列与 HNSW 索引。
`NODAL_EMBED_DIM` 必须与模型输出维度一致；默认脚本按 1536 建列，改维度时要同步改 SQL 并重建索引。

生产库建议：

- 独立账号，最小权限；
- 每日 `pg_dump` + 上传目录快照；
- 连接池默认 50（`db.Open` 中设置），按实例规格调整。

## 5. 构建与运行

```bash
# 后端
cd backend
CGO_ENABLED=0 GOTOOLCHAIN=local go build -trimpath -ldflags="-s -w" -o bin/nodal ./cmd/server

# 前端
cd frontend
npm ci
npm run build          # 产物在 dist/
```

### 5.1 systemd 示例

```ini
# /etc/systemd/system/nodal.service
[Unit]
Description=Nodal backend
After=network.target postgresql.service

[Service]
Type=simple
User=nodal
WorkingDirectory=/opt/nodal/backend
EnvironmentFile=/etc/nodal/nodal.env
ExecStart=/opt/nodal/backend/bin/nodal
Restart=on-failure
RestartSec=5
LimitNOFILE=65536
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now nodal
```

如果启用 REPL `local` 运行时，`NoNewPrivileges=true` 会阻止 `unshare`；
此时需要评估隔离方案，或设置 `NODAL_REPL_ENABLED=false`。

### 5.2 Nginx 反向代理

前端静态文件与 `/api` 同域，避免额外 CORS：

```nginx
server {
    listen 443 ssl http2;
    server_name nodal.example.com;

    root /opt/nodal/frontend/dist;
    index index.html;
    client_max_body_size 12m;

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_buffering off;      # SSE
        proxy_cache off;
        proxy_read_timeout 180s;
    }

    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

Caddy 等价配置：

```caddy
nodal.example.com {
    root * /opt/nodal/frontend/dist
    encode gzip zstd
    try_files {path} /index.html
    handle_path /api/* {
        reverse_proxy 127.0.0.1:8080 {
            flush_interval -1
        }
    }
    file_server
}
```

要点：SSE 必须关闭 `proxy_buffering`；WebSocket 必须转发 `Upgrade`/`Connection`。

## 6. Docker Compose（参考，未真机验证）

仓库提供：

- `docker-compose.yml`：PostgreSQL 16 + pgvector、后端、前端/Nginx；
- `deploy/Dockerfile.backend`：多阶段 Go 构建；
- `deploy/Dockerfile.frontend`：Node 构建 + Nginx；
- `deploy/nginx.conf`：静态托管 + `/api` 反代（SSE/WS）。

```bash
# 先把 docker-compose.yml 里的三处占位值替换为真实密钥
docker compose config
docker compose up -d --build
docker compose logs -f backend
```

访问 `http://localhost:8080`。

**重要边界**：本机没有 Docker，以上镜像与编排**没有做容器真机验证**。
首次上线前必须：

1. 把 `docker-compose.yml` 的 `POSTGRES_PASSWORD`、`NODAL_JWT_SECRET`、`NODAL_ENC_KEY` 换成真实值；
2. 确认 `pgvector/pgvector:pg16` 镜像可拉取，且 `GetExtension` 正常；
3. 确认前端构建内存足够（Monaco 打包需要约 4GB，脚本已设 `--max-old-space-size=4096`）；
4. 默认 `NODAL_REPL_ENABLED=false`；容器内开放 REPL 需要额外内核/权限配置，不建议直接 privileged；
5. 用 `docker compose config` 与一次完整注册→建画布→推演→上传知识库冒烟。

## 7. REPL 运行边界

- `local` 运行时：Linux 用户命名空间 + PTY + tmpfs，已在本机 5.x 内核真机验证断网、家目录只读、并发配额与 30s 超时；
- Firecracker：代码只实现了 launcher 合约，**未在本机真机验证**；生产需要 KVM、Firecracker 与独立 launcher；
- 容器/受限内核里若 `unshare` 失败，REPL 会报 `7001`，应关闭该能力或改用受支持的运行时；
- 无论哪种运行时，都不要把沙箱进程与主库放在同一权限域。

## 8. 上线检查清单

- [ ] `NODAL_ENV=prod`，JWT/ENC 密钥为真实随机值且长度合格；
- [ ] `NODAL_CORS_ORIGINS` 只包含真实域名，不是 `*`；
- [ ] HTTPS 终止在 Nginx/Caddy，后端不直接暴露公网；
- [ ] `/app/data/uploads` 与 PostgreSQL 数据目录已挂载持久化卷；
- [ ] `NODAL_EMBED_DIM` 与 embedding 模型一致，pgvector 扩展已安装；
- [ ] SSE 代理关闭缓冲，WS Upgrade 头已转发；
- [ ] 注册新账号能看到 2 张 Demo 画布；
- [ ] 备份策略覆盖数据库与上传文件；
- [ ] REPL 若开启，已确认隔离能力与配额；否则显式关闭。

## 9. 健康检查与冒烟

```bash
curl -fsS http://127.0.0.1:8080/healthz
```

最小冒烟链路：

1. 注册 → 资源库出现 2 张 Demo；
2. 打开 Demo 画布 → 编辑节点 → 自动保存；
3. 选中节点 → 推演（需配置 mock 或真实模型 Key）；
4. 上传一个 Markdown → 挂载 → RAG 检索；
5. 发布到广场 → 另开无痕窗口读取评论；
6. 代码节点运行（仅当 REPL 开启且隔离可用）。

## 10. 已知差异

- `backend/migrate/0001_init.sql` 是 PostgreSQL 初始化参考脚本，运行期仍以 GORM AutoMigrate 为准；
- 画布数据仍分 `nodes/edges/viewport/tags` 多列存储，未切到单列 `content_json`；
- Redis 缓存未实现；分享只读页前端仍是占位。
