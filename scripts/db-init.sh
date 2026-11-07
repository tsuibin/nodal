#!/usr/bin/env bash
# 初始化 PostgreSQL：创建扩展、执行 backend/migrate/0001_init.sql。
# 用法：NODAL_DB_DSN='postgres://nodal:pass@127.0.0.1:5432/nodal' ./scripts/db-init.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DSN="${NODAL_DB_DSN:?请设置 NODAL_DB_DSN}"
psql "$DSN" -v ON_ERROR_STOP=1 -f "$ROOT/backend/migrate/0001_init.sql"
