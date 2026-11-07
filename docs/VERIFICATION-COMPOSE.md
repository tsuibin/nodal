# Docker Compose 真机验证记录（受限沙箱）

日期：2026-11-07 | 环境：Ubuntu 22.04 / uid 1234 / cgroup2 ro / 无 CAP_SYS_ADMIN / rootless 单 UID 映射

## 结论

三服务已在 Compose 中真实启动并互通：

- `db`：`pgvector/pgvector:pg16`，PostgreSQL **16.15**
- `backend`：Go 静态二进制，`:8080`
- `frontend`：Nginx 静态页 + `/api` 反代

实测证据：

```text
backend:8080/healthz -> {"code":0,"message":"ok","data":{"status":"ok"}}
frontend:80/          -> <!doctype html> ... Nodal 页面
frontend:80/api/canvas/list -> {"code":0,...,"items":[],"total":0}
```

数据库内确认：

```text
vector 扩展版本：0.8.7
knowledge_chunks.embedding：vector
HNSW 索引：CREATE INDEX idx_knowledge_chunks_embedding ON public.knowledge_chunks
           USING hnsw (embedding vector_cosine_ops)
```

后端启动日志：

```text
pgvector schema ready dim=1536
database migrated
nodal server started addr=:8080 env=prod db=postgres
```

## 复现命令

```bash
DOCKER_HOST=unix:///tmp/d20-net.sock \
docker compose -f docker-compose.yml -f /tmp/compose-sandbox.yml up -d --no-build
```

服务发现与反代验证：

```bash
docker run --rm --network nodal_default --entrypoint sh nodal-frontend:latest \
  -c 'wget -qO- http://backend:8080/healthz; wget -qO- http://frontend/api/canvas/list'
```

## 沙箱偏差（必须知悉）

1. 官方镜像层在 rootless 单 UID 下解包触发 `lchown ... invalid argument`，因此镜像由 `skopeo` 拉取 + 扁平化 + `docker import` 导入；`nodal-backend` / `nodal-frontend` 不是用 `Dockerfile` 多阶段构建生成，但运行的是同一份后端 Go 源码编译产物与同一份前端 `dist` + `deploy/nginx.conf`。
2. `docker-compose.yml` 的 `db` 健康检查通过 `docker exec` 执行，而容器无 `/proc`，`exec` 报 `open /proc/self/setgroups`。沙箱覆盖文件禁用该健康检查，改为 `service_started` 依赖；生产环境应恢复 `service_healthy`。
3. 沙箱覆盖文件把默认桥命名为 `docker0`，并移除了 `frontend` 的 `8080:80` 端口映射；容器网络与 Nginx 反代已验证，但宿主机端口映射未验证。
4. PG 容器使用 uid 垫片绕过 `postgres` 用户切换，并手工追加了 `pg_hba.conf` 远程认证行、手工创建 `nodal` 库；这些是当前沙箱限制下的绕过，不应进入生产镜像。

## 尚未验证

- `docker-compose up --build` 的 Dockerfile 真实构建链路（受 rootless layer 权限限制）。
- 宿主机访问 `localhost:8080` 的端口映射。
- 生产 `service_healthy` 依赖链（健康检查 exec 在无 `/proc` 容器中不可用）。
