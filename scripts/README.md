# scripts

| 脚本 | 用途 |
|---|---|
| `dev.sh` | 同时起 SQLite 后端（`:8080`）与 Vite 前端（`5173`） |
| `test.sh` | `gofmt` + `go vet` + `go test` + `typecheck` + `build` |
| `db-init.sh` | 对 `NODAL_DB_DSN` 指向的 PostgreSQL 执行 `backend/migrate/0001_init.sql` |
| `smoke.sh` | 健康检查、注册、校验新账号自动有 2 张 Demo |

## 示例

```bash
./scripts/dev.sh
./scripts/test.sh
NODAL_DB_DSN='postgres://nodal:pass@127.0.0.1:5432/nodal' ./scripts/db-init.sh
BASE=http://127.0.0.1:8080 ./scripts/smoke.sh
```
