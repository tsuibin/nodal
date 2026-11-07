#!/usr/bin/env bash
# 最小冒烟：健康检查 + 注册 + 我的画布列表（应看到 2 张 Demo）。
# 用法：BASE=http://127.0.0.1:8080 ./scripts/smoke.sh
set -euo pipefail

BASE="${BASE:-http://127.0.0.1:8080}"
EMAIL="smoke-$(date +%s)@test.com"

curl -fsS "$BASE/healthz" >/dev/null
TOKEN="$(curl -fsS -X POST "$BASE/api/auth/register" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"password123\",\"nickname\":\"冒烟\"}" \
  | python3 -c 'import json,sys; print(json.load(sys.stdin)["data"]["token"])')"

curl -fsS "$BASE/api/canvas/list?scope=mine" \
  -H "Authorization: Bearer $TOKEN" \
  | python3 -c 'import json,sys; d=json.load(sys.stdin)["data"]; print("canvases:", d["total"], [i["title"] for i in d["items"]]); assert d["total"] >= 2'
