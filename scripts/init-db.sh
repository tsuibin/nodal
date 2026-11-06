#!/usr/bin/env bash
# 初始化 Nodal 的 PostgreSQL：开启 pgvector 扩展并建好基础 schema。
# 用法：bash scripts/init-db.sh
set -euo pipefail

POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-nodal-postgres}"
DB_USER="${POSTGRES_USER:-nodal}"
DB_NAME="${POSTGRES_DB:-nodal}"

echo "==> 等待 PostgreSQL 就绪..."
for i in $(seq 1 30); do
  if docker exec "$POSTGRES_CONTAINER" pg_isready -U "$DB_USER" -d "$DB_NAME" >/dev/null 2>&1; then
    echo "    PostgreSQL 已就绪"
    break
  fi
  if [ "$i" -eq 30 ]; then
    echo "!! PostgreSQL 启动超时，请检查 docker compose ps" >&2
    exit 1
  fi
  sleep 1
done

echo "==> 开启 pgvector 扩展..."
docker exec -i "$POSTGRES_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" <<'SQL'
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS pgcrypto;
SELECT extname, extversion FROM pg_extension WHERE extname IN ('vector', 'pgcrypto');
SQL

echo "==> 完成。后续表结构由 backend/migrate/ 下的迁移脚本负责。"
