#!/usr/bin/env bash
# 本地构建：生成 chroma 样式表 → 构建前端 → 编译当前平台后端。
# 服务器部署请用 scripts/deploy.sh。
set -euo pipefail
cd "$(dirname "$0")/.."

VERSION="$(date -u +%Y%m%d-%H%M)"

echo ">>> [1/3] 生成 chroma 代码高亮样式表"
go run ./tools/chroma-css

echo ">>> [2/3] 构建前端"
(
  cd web
  if [ -f package-lock.json ]; then
    npm ci
  else
    npm install
  fi
  npm run build
)

echo ">>> [3/3] 编译后端（当前平台）"
mkdir -p dist
go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o dist/bblog ./cmd/bblog
ls -l dist/bblog
echo "构建完成：版本 ${VERSION}"
