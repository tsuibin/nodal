#!/usr/bin/env bash
# 同时启动后端与前端；Ctrl+C 结束时回收两个子进程。
# 用法：./scripts/dev.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

cleanup() {
  kill "${BACK_PID:-}" "${FRONT_PID:-}" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

(
  cd backend
  GOTOOLCHAIN=local \
  NODAL_HTTP_ADDR=:8080 \
  NODAL_DB_DRIVER=sqlite \
  NODAL_DB_DSN=data/nodal.db \
  NODAL_JWT_SECRET=dev-secret-at-least-32-characters \
  NODAL_STORAGE_DIR=data/uploads \
  NODAL_EMBED_MODEL=mock/embed \
  go run ./cmd/server
) &
BACK_PID=$!

(
  cd frontend
  VITE_DEV_API_TARGET=http://127.0.0.1:8080 npm run dev
) &
FRONT_PID=$!

wait
