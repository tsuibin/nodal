# backend/migrate

`0001_init.sql` 是 PostgreSQL 16 + pgvector 的空库初始化参考脚本。

## 使用方式

```bash
createdb nodal
psql "postgres://nodal@127.0.0.1:5432/nodal" -f backend/migrate/0001_init.sql
```

运行后端时仍会执行 GORM `AutoMigrate`（`backend/db/db.go`）作为最终兜底；
`vector.EnsurePGVectorSchema` 会补齐 `knowledge_chunks.embedding` 与 HNSW 索引。

## 边界

- 脚本按默认 `NODAL_EMBED_DIM=1536` 建 `vector(1536)`；改维度时必须同步修改本文件并重建索引；
- 这是首次初始化脚本，不是增量迁移框架；后续字段变更请新增 `0002_xxx.sql`，不要改动已上线的 `0001`；
- SQLite/MySQL 不使用本脚本，继续依赖 AutoMigrate；
- 本机没有 PostgreSQL，脚本未在真库执行验证，上线前请在一个空库跑一次并核对索引。
