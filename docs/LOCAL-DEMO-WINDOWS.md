# Nodal 本机演示部署记录（Windows / SQLite）

目标：在 Windows 本机把 `https://git.loongfans.cn/tsuibin/nodal` 跑起来，浏览器能访问、能登录、能画布推演。
仓库本身面向 Linux 部署（`scripts/*.sh` + Docker Compose），本文记录的是**不依赖 WSL/Docker**的本机跑法。

## 1. 结论

| 项 | 值 |
| --- | --- |
| 前端（Vite dev） | http://127.0.0.1:5173 |
| 后端（Go） | http://127.0.0.1:8080 ，健康检查 `/healthz` |
| 数据库 | SQLite，`backend/data/nodal.db`（启动时 AutoMigrate） |
| 演示账号 | `demo@nodal.local` / `Demo12345!`（昵称「大壮」，user_id 1） |
| 演示数据 | 注册时自动生成 2 张画布：「Demo｜AI 著作权辩论推演」「Demo｜缓存穿透方案对比」 |
| 启动 | `powershell .\scripts\start-local.ps1` |
| 停止 | `powershell .\scripts\stop-local.ps1` |

验证过的链路：首页渲染 → 登录（拿到 JWT）→ 资源库列出 2 张画布 → 画布页 React Flow 渲染节点/连线 → 推演接口
`POST /api/canvas/:id/run-infer`（`mock/basic` 模拟模型）产出 3 个节点 + 2 条边的 SSE 事件。

## 2. 工具链（本机无 Go / Node）

本机 PATH 里没有 `go`、`node`、`npm`、`docker`，因此使用解压在 `..\_runtime` 下的便携运行时（不污染系统 PATH）：

| 组件 | 路径 | 版本 |
| --- | --- | --- |
| Go | `..\_runtime\go\bin\go.exe` | go1.23.12 windows/amd64 |
| Node | `..\_runtime\node-v22.23.4-win-x64\node.exe` | v22.23.4 / npm 10.9.9 |
| 后端二进制 | `..\_runtime\nodal-server.exe` | 由 `backend` 源码构建 |

相关环境变量（编译与下载依赖时）：

```powershell
$env:GOPROXY      = "https://goproxy.cn,direct"
$env:GOTOOLCHAIN  = "local"
$env:GOPATH       = "..\_runtime\gopath"
$env:GOMODCACHE   = "..\_runtime\gopath\pkg\mod"
```

前端依赖安装：`npm install --registry=https://registry.npmmirror.com --no-audit --no-fund`（253 个包）。

## 3. Windows 兼容改动（本地工作区，未提交、未推送）

仓库的 REPL 沙箱只支持 Linux（依赖 `unshare` / 用户命名空间 / tmpfs），且 `session.go` 里用负 pid 给整个进程组发
`SIGKILL`，而 Windows 没有这套语义，直接编译会报：

```
backend\replsandbox\session.go:221:16: undefined: syscall.Kill
```

改法是把「杀进程组」抽成按平台分文件的函数，行为在 Linux 上完全不变：

| 文件 | 变更 |
| --- | --- |
| `backend/replsandbox/session.go` | `syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)` 一行替换为 `terminateGroup(cmd)`，其后的 `cmd.Process.Kill()` 兜底保留 |
| `backend/replsandbox/proc_unix.go`（新增） | `//go:build !windows`，`terminateGroup` 就是原来的负 pid `syscall.Kill` |
| `backend/replsandbox/proc_windows.go`（新增） | `//go:build windows`，`terminateGroup` 退化为 `cmd.Process.Kill()` |

除这一处，后端没有其它 Linux 专有符号（`creack/pty` 在 Windows 上走 ConPTY）。因此本机的 REPL 能力用
`NODAL_REPL_ENABLED=false` 显式关闭——**这是平台限制，不是 bug**；要体验 REPL 需要 Linux 环境。

## 4. 后端启动参数

```
NODAL_ENV=dev
NODAL_HTTP_ADDR=:8080
NODAL_DB_DRIVER=sqlite
NODAL_DB_DSN=data/nodal.db          # 相对 backend 工作目录
NODAL_STORAGE_DIR=data/uploads
NODAL_EMBED_MODEL=mock/embed        # 本地确定性向量，免 API Key
NODAL_REPL_ENABLED=false
NODAL_JWT_SECRET=nodal-local-demo-secret-please-change-32
NODAL_CORS_ORIGINS=http://localhost:5173,http://127.0.0.1:5173
```

工作目录必须是 `backend`（相对路径按它解析）。日志：`backend/data/server.out.log`、`server.err.log`。

前端：`node node_modules\vite\bin\vite.js --host 127.0.0.1 --port 5173`，工作目录 `frontend`，
`VITE_DEV_API_TARGET=http://127.0.0.1:8080` 决定 `/api` 反代目标。日志：`backend/data/frontend.*.log`。
两个进程号写在 `backend/data/local-pids.json`，由停止脚本回收。

## 5. 模型与 AI 推演

模型目录里只有 `mock/basic`（`requires_key: false, configured: true`）开箱可用，其余 provider
（OpenAI / Anthropic / DeepSeek / 智谱 / 月之暗面 / 通义）都需要填 API Key。

- 想直接用真模型：在页面「模型凭证」里保存对应 provider 的 Key（密钥用 `NODAL_ENC_KEY` 加密落库；dev 下未配置时从 JWT secret 派生，**生产必须独立配置**）。
- Agent 模板不预置，需要自己建一个（画布推演的 `run-infer` 需要 `agent_id`）；本机演示已建「演示 Agent」，绑定 `mock/basic`。

## 6. 联调中发现并修复的缺陷：知识库上传 DOCX 报「请求参数不正确」

现象：资源库 → 知识库 → 选择文件上传 DOCX，提示「请求参数不正确」，后端日志
`POST /api/knowledge/upload status=400 err="biz error code=1000 msg=请求参数不正确"`，且响应耗时 0ms。

根因在 `frontend/src/services/http.ts`：axios 实例设了全局默认头
`headers: { 'Content-Type': 'application/json' }`。axios 1.21 的 `transformRequest` 里有这段逻辑：

```js
const hasJSONContentType = contentType.indexOf('application/json') > -1;
const isFormData = utils.isFormData(data);
if (isFormData) {
  return hasJSONContentType ? JSON.stringify(formDataToJSON(data)) : data;  // ← 命中这里
}
```

也就是说，默认头是 JSON 时，`knowledgeApi.upload()` 里的 `FormData` 会被序列化成 JSON 字符串
（文件对象的属性全丢），请求根本没有 multipart 体，后端 `c.FormFile("file")` 拿不到字段，
`bindError` 走默认分支返回 `code=1000 请求参数不正确`。

修复：删掉 axios 实例上的全局 `Content-Type` 默认头，注释说明原因。
普通对象请求 axios 会自行补 `application/json`，`FormData` 由浏览器补 multipart boundary。

验证（用一份自制 DOCX，含 6 段中文）：

1. 知识库页上传 → 列表出现 `DOCX / test-knowledge.docx / 1 KB · 1 个切片 / 已就绪`；
2. `POST /api/canvas/1/mount-knowledge` 挂载成功，`POST /api/rag/retrieve`（query「布隆过滤器怎么防缓存穿透」）
   召回该文件的切片，中文抽取正常（`backend/service/knowledge_extract.go` 走 zip + `word/document.xml` 正则）；
3. 回归 JSON 请求：画布标题 PUT 仍 `200`「已保存」，说明去掉默认头没影响普通接口。

## 7. 代码节点「运行」为什么提示「服务开小差了，请稍后重试」

两件事叠在一起，且第二件是仓库真正的缺陷。

**（1）本机没有 REPL 沙箱（平台限制）**
启动脚本设了 `NODAL_REPL_ENABLED=false`：`LocalRuntime` 依赖 Linux 用户命名空间
（`unshare --user --map-root-user --net --mount --pid --fork` + tmpfs + `ulimit`），
Windows 上不存在等价能力，所以部署时显式关闭。此时 `Manager.Create` 的第一道校验
`!Enabled || Runtime==nil` 直接返回 `CodeReplUnavailable(7001)`。

**（2）响应层把 503 折叠成了 5000（缺陷）**
`backend/response/response.go` 原来这样写：

```go
if appErr.Status() >= http.StatusInternalServerError {
    appErr = apperr.New(apperr.CodeInternal, "")   // ← 连 503 一起折叠
}
```

`CodeReplUnavailable` 在 `apperr.go` 里映射的是 **503 +「REPL 沙箱暂不可用」**，
但它 ≥500，于是被折叠成 `5000 服务开小差了，请稍后重试`，真正原因被吃掉：
后端日志明明记着 `code=7001 msg=REPL 沙箱未启用`，用户界面上却只看到一句通用文案。

修复：只折叠「未知错误」（`appErr.Code == CodeInternal`，即 `apperr.From` 归一化出来的那种），
已知业务码（7001、5001 等）原样返回自己的状态与文案。同时把 `Manager.Create` 的文案改成可操作的
「代码执行未启用：需要 NODAL_REPL_ENABLED=true，且沙箱依赖 Linux 用户命名空间（Windows 等无隔离环境不支持）」。

验证：`POST /api/repl/create` 由 `500 +「服务开小差了」` 变为
`503 +「代码执行未启用：需要 NODAL_REPL_ENABLED=true…」`；画布 2 的代码节点双击 → 点「运行」，
运行面板红字显示同一句可操作提示（截图 `data\shot-repl-error.png`）。

**（3）顺带修了启动脚本的一个坑**
`scripts/start-local.ps1 -Rebuild` 原来在项目根目录直接执行 `go build ./cmd/server`，
`./cmd/server` 是相对 cwd 的包路径，导致 `-Rebuild` 必然报「后端编译失败」。
改为 `Push-Location backend` 后再编译。

## 8. 已知限制

1. 只有 dev 双端口（5173 前端 + 8080 后端），没有做生产同源单端口（把 `frontend/dist` 交给后端或加反向代理）。
2. REPL 代码执行关闭（见第 3 节）。
3. `mock/basic` 的输出是固定 JSONL 样例（3 节点 2 边），不是真实推理结果。
4. 生产口径还需要：`NODAL_ENV=prod`、独立 `NODAL_ENC_KEY`、真实随机 `NODAL_JWT_SECRET`、PostgreSQL/MySQL。
5. 端口 8080 与本地参考站（vBulletin 复刻站）冲突，两者不能同时跑；需要并存时用 `-BackendPort` 改端口（脚本会把前端反代一起改）。
