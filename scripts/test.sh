#!/usr/bin/env bash
# 交付前统一校验：后端 vet/test，前端 typecheck/build。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT/backend"
gofmt -w .
GOTOOLCHAIN=local go vet ./...
GOTOOLCHAIN=local go test ./...

cd "$ROOT/frontend"
npm run typecheck
npm run build
