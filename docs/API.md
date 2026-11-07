# Nodal API

本文是 Nodal 后端接口契约总表，与 `backend/router/router.go` 一一对应。
所有示例省略了外层信封里的 `request_id`。

## 1. 通用约定

- Base URL：`/api`（开发 `http://127.0.0.1:8080`）；
- 请求/响应：`application/json; charset=utf-8`，上传接口为 `multipart/form-data`；
- 时间：RFC3339 / UTC 字符串；
- 成功信封：

```json
{ "code": 0, "message": "ok", "data": {} }
```

- 失败信封：

```json
{
  "code": 3004,
  "message": "该模型需要先配置 API Key",
  "details": { "model_id": "openai/gpt-4o-mini" },
  "request_id": "..."
}
```

`code` 是稳定判据，前端不要匹配 `message` 文案。`details` 只在参数校验（1000）或需要定位字段时出现。

### 鉴权

```http
Authorization: Bearer <JWT>
```

- 用户 token：`role=user`，`uid` 指向注册用户；
- 游客 token：`role=guest`，`gid` 指向临时会话，有效期默认 72h；
- 未带 token 的 `/api/**` 业务请求由中间件自动分配游客身份，并在响应头返回：

```http
X-Guest-Token: <JWT>
```

- `POST /api/auth/register|login|guest` 不需要 token；
- 标记「用户」的接口在全局鉴权之上再要求 `role=user`，游客返回 `1003`。

## 2. 公开接口

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/healthz` | 健康检查，返回 `{"status":"ok"}`，不在 `/api` 下 |
| POST | `/api/auth/register` | 注册；成功后返回 JWT，并自动创建 2 张 Demo 画布 |
| POST | `/api/auth/login` | 登录 |
| POST | `/api/auth/guest` | 显式领取游客会话；token 同时写入 `X-Guest-Token` |
| GET | `/api/square/list` | 广场列表；带 token 时返回 `liked` 状态 |
| GET | `/api/canvas/:id/comments` | 评论列表；公开画布可匿名读 |
| GET | `/api/share/:token` | 校验分享链接并返回画布 |
| GET | `/api/repl/ws/:sessionId` | REPL WebSocket；token 走 `?token=` |

### 2.1 注册 / 登录 / 游客

`POST /api/auth/register`

```json
{ "email": "a@b.com", "password": "password123", "nickname": "林深" }
```

`POST /api/auth/login`

```json
{ "email": "a@b.com", "password": "password123" }
```

`POST /api/auth/guest` 无 body，可带 `X-Request-ID`。

返回 `SessionPayload`：

```json
{
  "token": "<JWT>",
  "token_type": "Bearer",
  "expires_at": "2026-11-08T06:55:45Z",
  "is_guest": false,
  "user": { "id": 1, "email": "a@b.com", "nickname": "林深", "avatar_url": "", "created_at": "..." }
}
```

游客会话返回 `is_guest=true` 与 `guest: {id, expires_at}`，没有 `user`。

常见错误：`1000` 参数、`2001` 邮箱已注册、`2003` 账号或密码错误、`2004` 停用、`2005` 游客关闭、`2006` 游客过期。

### 2.2 广场列表

`GET /api/square/list?tag=示例&keyword=缓存&sort=latest&page=1&page_size=12`

- `sort`：`latest`（默认）/ `hot`；
- 返回 `SquareList`，`items[].liked` 对匿名恒为 `false`。

### 2.3 分享解析

`GET /api/share/:token` 返回：

```json
{
  "token": "...",
  "permission": "read",
  "can_read": true,
  "can_fork": false,
  "canvas": { "id": 1, "title": "...", "nodes": [], "edges": [], "viewport": {} }
}
```

错误：`6001` 无效、`6002` 过期、`6003` 权限不足。

## 3. 全局鉴权接口（用户 + 游客）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/auth/me` | 当前用户/游客资料 |
| POST | `/api/demo/seed` | 幂等创建 2 张 Demo 画布 |
| POST | `/api/canvas` | 创建画布 |
| GET | `/api/canvas/list` | 画布列表 |
| GET | `/api/canvas/:id` | 画布详情 |
| PUT | `/api/canvas/:id` | 全量更新 nodes/edges（仅所有者） |
| DELETE | `/api/canvas/:id` | 删除（仅所有者） |
| POST | `/api/canvas/:id/fork` | Fork（可读即可） |
| POST | `/api/share/:token/fork` | 按 fork 权限分叉分享画布 |

### 3.1 当前身份

`GET /api/auth/me` 返回 `ProfilePayload`：

```json
{ "is_guest": false, "user": { "id": 1, "nickname": "林深", "..." : "..." } }
```

游客返回 `{ "is_guest": true, "guest": { "id": "g_...", "expires_at": "..." } }`。

### 3.2 Demo 种子

`POST /api/demo/seed`：

```json
{ "created": 2, "titles": ["Demo｜AI 著作权辩论推演", "Demo｜缓存穿透方案对比"] }
```

幂等规则：该 owner 已有任意画布时返回 `{ "created": 0, "titles": null }`。
注册成功会在服务端直接调用同一逻辑，前端只在「老账号/游客首次进入资源库且 0 张画布」时兜底调用。

### 3.3 画布

`POST /api/canvas`

```json
{
  "title": "远程办公与长期创新",
  "description": "从核心命题分出正反方",
  "nodes": [{ "id": "n1", "type": "nodal", "position": {"x": 80, "y": 120}, "data": {"kind": "text", "label": "核心命题"} }],
  "edges": [],
  "viewport": { "x": 0, "y": 0, "zoom": 1 },
  "tags": ["组织", "远程办公"],
  "is_public": false,
  "share_mode": "private"
}
```

`PUT /api/canvas/:id` 支持部分字段；`nodes` 与 `edges` 必须成对提交（整图替换）。
更新/删除仅 owner；可读但非 owner 的角色会得到 `1005`。

`GET /api/canvas/list?scope=mine|public&keyword=&tag=&page=1&page_size=12`

返回 `CanvasList`；列表项不含 `nodes/edges`（避免大 payload）。

节点/边约束（`normalizeDocument`）：

- 节点 `id` 唯一，`position.x/y` 必填；
- 边 `source/target` 必须指向本次提交的节点；
- 单画布上限：5MB、2000 节点、5000 边、10 个标签、标签 20 字、标题 120 字。

### 3.4 Fork

`POST /api/canvas/:id/fork`

```json
{ "title": "我的分叉版本" }
```

返回新的 `CanvasView`，默认私有；源画布必须对调用者可读。

`POST /api/share/:token/fork` 无需 body，按分享链接的 `fork` 权限决定是否允许（`6003` 拒绝）。

## 4. 仅注册用户接口

### 4.1 资料

`PUT /api/auth/profile`

```json
{ "nickname": "林深", "avatar_url": "https://..." }
```

字段均可选；空昵称返回 `1000`。

### 4.2 Agent 模板

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/agent` | 创建 |
| GET | `/api/agent/list` | 列表 |
| GET | `/api/agent/:id` | 详情 |
| PUT | `/api/agent/:id` | 部分更新 |
| DELETE | `/api/agent/:id` | 删除 |

创建 body：

```json
{ "name": "反方辩手", "description": "", "system_prompt": "你是反方辩手…", "model_id": "mock/basic" }
```

`name` 重名返回 `3002`；`model_id` 不在目录返回 `3003`。

### 4.3 模型目录 / 自有 Key

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/model/catalog` | 模型目录 + 是否已配置 Key |
| GET | `/api/model/credentials` | 各 provider 配置状态（仅尾号） |
| PUT | `/api/model/credentials/:provider` | 保存/更新 Key，AES-256-GCM 加密落库 |
| DELETE | `/api/model/credentials/:provider` | 删除 Key |

保存 body：`{ "api_key": "sk-..." }`。任何读接口都不会返回明文。

### 4.4 推演（SSE）

`POST /api/canvas/:id/run-infer`

```json
{ "selected_node_ids": ["n1", "n2"], "agent_id": 1 }
```

响应 `text/event-stream`，事件如下（`data` 均为 JSON）：

```text
event: start
data: {"run_id":"...","canvas_id":1,"canvas_title":"...","agent_id":1,"agent_name":"反方辩手","model_id":"mock/basic","model_label":"Mock"}

event: sources
data: {"query":"...","items":[{"file_id":1,"file_name":"a.md","chunk_index":0,"score":0.8,"preview":"..."}],"warning":""}

event: node
data: {"node":{"id":"mock-n1","type":"nodal","position":{"x":1,"y":2},"data":{"source":"agent","ai_generated":true}},"index":0}

event: edge
data: {"edge":{"id":"mock-e1","source":"n1","target":"mock-n1"},"index":0}

event: warning
data: {"message":"模型尝试删除节点，已忽略","skipped":1}

event: done
data: {"run_id":"...","node_count":3,"edge_count":2,"skipped":1}
```

错误时（开流前）返回普通 JSON 信封；已开流后由 `error` 事件返回 `{code,message}`。
硬约束：AI 只能新增 node/edge，`delete/update` 一律丢弃并计入 `skipped`。
同一画布并发推演返回 `3007`；缺 Key `3004`；模型调用失败 `3005`。

### 4.5 知识库 / RAG

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/knowledge/upload` | `multipart/form-data`，字段名 `file`；同步解析/切片/向量化 |
| GET | `/api/knowledge/list` | 我的文件 |
| DELETE | `/api/knowledge/:id` | 删除文件、切片、向量与磁盘文件 |
| POST | `/api/canvas/:id/mount-knowledge` | 覆盖式挂载；`{ "file_ids": [1,2] }`，空数组=解除 |
| GET | `/api/canvas/:id/mount-knowledge` | 当前挂载 |
| DELETE | `/api/canvas/:id/mount-knowledge` | 解除挂载 |
| POST | `/api/rag/retrieve` | `{ "canvas_id": 1, "query": "...", "top_k": 5 }` |

挂载是**进程内临时关联**，服务重启即失效；只有画布已挂载的文件参与推演召回。
错误：`4001` 文件不存在、`4002` 类型不支持、`4003` 解析失败、`4004` 未挂载、`4005` 向量化失败。

### 4.6 广场 / 分享

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/canvas/:id/publish` | 发布到广场（owner） |
| DELETE | `/api/canvas/:id/publish` | 下架 |
| POST | `/api/canvas/:id/like` | 切换点赞 |
| POST | `/api/canvas/:id/comment` | `{ "content": "..." }` |
| DELETE | `/api/canvas/:id/comments/:commentId` | 删除评论（作者/owner） |
| POST | `/api/canvas/:id/share-link` | `{ "permission": "read|fork", "expires_hours": 1..720 }` |
| GET | `/api/canvas/:id/share-links` | 列出链接 |
| DELETE | `/api/canvas/:id/share-links/:linkId` | 停用链接 |

分享链接创建会复用「同画布 + 同权限 + 未过期」的既有链接，不会重复新建。

### 4.7 REPL 沙箱

`POST /api/repl/create`

```json
{ "language": "python", "code": "print('hi')" }
```

返回：

```json
{
  "session_id": "...",
  "language": "python",
  "ws_path": "/api/repl/ws/...",
  "timeout_seconds": 30,
  "memory_mb": 256,
  "expires_at": "...",
  "runtime": "local"
}
```

`GET /api/repl/ws/:sessionId?token=<JWT>` 升级 WebSocket：

- 服务端帧 `ready` / `output` / `exit` / `error`；
- 客户端帧 `input` / `resize` / `close`；
- 硬限制：单会话 30s、内存上限、禁网络出站、禁止持久化；全局/单用户并发配额。

错误：`7001` 不可用、`7002` 会话不存在、`7003` 并发过多、`7004` 已附加、`7005` 语言不支持、`7006` 代码过长。

## 5. 错误码

| 段 | 码 | 含义 |
|---|---|---|
| 通用 | 0 | 成功 |
| | 1000 | 参数校验失败（`details` 为字段错误） |
| | 1001 | 未登录 / token 无效 |
| | 1002 | token 过期 |
| | 1003 | 需要注册用户（游客被拒） |
| | 1004 | 资源不存在 |
| | 1005 | 无权限 |
| 账号 | 2001 | 邮箱已注册 |
| | 2002 | 用户不存在 |
| | 2003 | 账号或密码错误 |
| | 2004 | 账号停用 |
| | 2005 | 游客已关闭 |
| | 2006 | 游客会话过期 |
| Agent/模型 | 3001 | Agent 不存在 |
| | 3002 | Agent 重名 |
| | 3003 | 模型不支持 |
| | 3004 | 缺少模型 Key |
| | 3005 | 模型调用失败 |
| | 3006 | 空补全 |
| | 3007 | 推演进行中 |
| 知识库 | 4001 | 文件不存在 |
| | 4002 | 文件类型不支持 |
| | 4003 | 解析失败 |
| | 4004 | 知识库未挂载 |
| | 4005 | 向量化失败 |
| 系统 | 5000 | 服务端异常 |
| 分享/互动 | 6001 | 分享无效 |
| | 6002 | 分享过期 |
| | 6003 | 分享无权限 |
| | 6004 | 画布未发布 |
| | 6005 | 评论不存在 |
| REPL | 7001 | 沙箱不可用 |
| | 7002 | 会话不存在 |
| | 7003 | 并发过多 |
| | 7004 | 会话已附加 |
| | 7005 | 语言不支持 |
| | 7006 | 代码过长 |

前端网络层用 `-1` 表示网络/超时错误，不对应后端业务码。

错误码到 HTTP 状态的映射在 `backend/apperr/apperr.go` 的 `codeTable` 里，例如 `1003` → 403、
`7001` → 503、`7003` → 429、`5000` → 500。

**折叠约定**（`backend/response/response.go`）：只有「未知错误」——即非 `*apperr.Error`、
由 `apperr.From` 归一化出来的 `5000`——才会被折叠成 `code=5000 message=服务开小差了，请稍后重试`，
避免内部细节外泄。已在 `codeTable` 里登记的已知业务码（7001、7005、4005 等）**原样返回**自己的
HTTP 状态与文案，因为它们的文案本身是可操作的（例如「代码执行未启用：需要 NODAL_REPL_ENABLED=true…」）。
新增业务码时务必同时登记 `codeTable`，否则会退化成 400 且丢失语义。

## 6. 已知契约边界

- 当前画布文档以 `Nodes / Edges / Viewport` 多列存储，尚未切到单列 `content_json`；接口层已经按「整图 JSON」语义暴露；
- 分享只读页 `/share/:token` 前端仍是占位，`GET /api/share/:token` 已可用；
- 知识库挂载不持久化；
- Redis 尚未接入，列表/热门排序直接查库；
- 本文是 Markdown 契约，Go handler 尚未逐接口补 OpenAPI 注解；字段以 `backend/service/*.go` 的 json tag 为准。
