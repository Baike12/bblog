#!/usr/bin/env bash
# 发布内容：校验 → rsync 到服务器 → 触发索引重载。
# 不重建镜像、不重启容器；日常发文章用这个脚本。
set -euo pipefail
cd "$(dirname "$0")/.."

SERVER_HOST="${BBLOG_HOST:-root@101.96.243.197}"
REMOTE_CONTENT="${BBLOG_REMOTE_CONTENT:-/opt/bblog/content}"
SITE_URL="${BBLOG_SITE_URL:-http://101.96.243.197/blog}"
TOKEN_FILE="${BBLOG_TOKEN_FILE:-.secrets/publish_token}"

echo ">>> [1/4] 校验内容"
go run ./cmd/bblog check

echo ">>> [2/4] 同步内容到 ${SERVER_HOST}:${REMOTE_CONTENT}"
started=$(date +%s)
rsync -az --delete \
  --exclude '.DS_Store' \
  --exclude 'trash/' \
  -e ssh \
  content/ "${SERVER_HOST}:${REMOTE_CONTENT}/"
echo "    rsync 完成，耗时 $(( $(date +%s) - started ))s"

echo ">>> [3/4] 触发服务端重建索引"
if [ -f "${TOKEN_FILE}" ]; then
  curl -fsS -X POST \
    -H "X-Publish-Token: $(cat "${TOKEN_FILE}")" \
    "${SITE_URL}/api/admin/reload"
  echo
else
  echo "    未找到 ${TOKEN_FILE}，跳过（容器内 fsnotify 会自动重载）"
fi

echo ">>> [4/4] 健康检查"
curl -fsS "${SITE_URL}/healthz"
echo
echo "发布完成：${SITE_URL}/"
