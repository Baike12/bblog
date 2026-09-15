#!/usr/bin/env bash
# 在服务器上执行：用指定标签的镜像（重新）启动 bblog 容器。
# 由 scripts/deploy.sh 通过 ssh 传入 TAG 后执行。
set -euo pipefail

: "${TAG:?需要 TAG 环境变量（镜像标签）}"
IMAGE="bblog:${TAG}"

if ! docker image inspect "${IMAGE}" >/dev/null 2>&1; then
  echo "服务器上没有镜像 ${IMAGE}" >&2
  docker images bblog
  exit 1
fi

echo ">>> 停止并移除旧容器"
docker rm -f bblog >/dev/null 2>&1 || true

echo ">>> 启动容器（镜像 ${IMAGE}）"
docker run -d --name bblog --restart unless-stopped \
  -p 127.0.0.1:8090:8090 \
  -v /opt/bblog/content:/app/content \
  -v /opt/bblog/data:/app/data \
  -v /opt/bblog/bblog.yaml:/app/bblog.yaml:ro \
  --env-file /opt/bblog/.env \
  --log-driver json-file --log-opt max-size=10m --log-opt max-file=3 \
  "${IMAGE}" >/dev/null

echo ">>> 只保留最近 3 个镜像（回滚依赖这些标签）"
docker images --format '{{.Repository}}:{{.Tag}} {{.CreatedAt}}' bblog \
  | grep -v ':latest ' \
  | sort -k2 -r \
  | awk 'NR>3 {print $1}' \
  | xargs -r docker rmi -f >/dev/null 2>&1 || true

sleep 1
docker ps --filter name=bblog --format '容器状态: {{.Status}} | 镜像: {{.Image}}'
