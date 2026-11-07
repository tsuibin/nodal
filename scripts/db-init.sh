#!/usr/bin/env bash
# 初始化 PostgreSQL：创建扩展、执行 backend/migrate/0001_init.sql。
# 用法：NODAL_DB_DSN='postgres://nodal:pass@127.0.0.1:5432/nodal' ./scripts/db-init.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DSN="${NODAL_DB_DSN:?请设置 NODAL_DB_DSN}"

# 扩展需要超级用户；已由 DBA 安装过可跳过。提供 SUPER_DSN 时自动执行。
if [ -n "${NODAL_PG_SUPER_DSN:-}" ]; then
  psql "$NODAL_PG_SUPER_DSN" -v ON_ERROR_STOP=1 -f "$ROOT/backend/migrate/0000_extensions.sql"
else
  echo "提示：pgvector 扩展需由超级用户执行 backend/migrate/0000_extensions.sql"
fi

# 表结构由应用账号执行，确保 owner 与运行账号一致。
psql "$DSN" -v ON_ERROR_STOP=1 -f "$ROOT/backend/migrate/0001_init.sql"
