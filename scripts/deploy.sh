#!/usr/bin/env bash
# 部署镜像到火山服务器。
#
# 服务器实测访问不了 Docker Hub（registry-1.docker.io 超时），因此流程是：
#   本机构建 linux/amd64 镜像 → docker save 流式传输 → 服务器 docker load
# 全程不需要镜像仓库，服务器也不执行任何 docker pull。
#
# 用法：
#   scripts/deploy.sh                 完整部署（构建前端 + 编译 + 打镜像 + 传输 + 重启）
#   scripts/deploy.sh --skip-frontend  跳过前端构建（只改后端时更快）
#   scripts/deploy.sh --rollback TAG   回滚到服务器上已有的某个镜像标签
set -euo pipefail
cd "$(dirname "$0")/.."

SERVER_HOST="${BBLOG_HOST:-root@101.96.243.197}"
SITE_URL="${BBLOG_SITE_URL:-http://101.96.243.197/blog}"
PLATFORM="linux/amd64"
SKIP_FRONTEND=0
ROLLBACK_TAG=""

while [ $# -gt 0 ]; do
  case "$1" in
    --skip-frontend) SKIP_FRONTEND=1 ;;
    --rollback)
      shift
      ROLLBACK_TAG="${1:-}"
      [ -n "${ROLLBACK_TAG}" ] || { echo "错误：--rollback 需要一个镜像标签" >&2; exit 2; }
      ;;
    -h|--help)
      sed -n '2,14p' "$0"
      exit 0
      ;;
    *) echo "未知参数: $1" >&2; exit 2 ;;
  esac
  shift
done

if [ -n "${ROLLBACK_TAG}" ]; then
  echo ">>> 回滚到 bblog:${ROLLBACK_TAG}"
  ssh "${SERVER_HOST}" "TAG=${ROLLBACK_TAG} bash -s" < deploy/remote_run.sh
  echo ">>> 健康检查"
  for _ in $(seq 1 30); do
    if curl -fsS "${SITE_URL}/healthz" >/dev/null 2>&1; then
      curl -fsS "${SITE_URL}/healthz"; echo
      echo "回滚完成"
      exit 0
    fi
    sleep 2
  done
  echo "回滚后健康检查未通过" >&2
  exit 1
fi

TAG="${BBLOG_TAG:-$(date -u +%Y%m%d-%H%M)}"
IMAGE="bblog:${TAG}"

echo ">>> [1/8] 生成 chroma 代码高亮样式表"
go run ./tools/chroma-css

if [ "${SKIP_FRONTEND}" -eq 0 ]; then
  echo ">>> [2/8] 构建前端"
  (
    cd web
    if [ -f package-lock.json ]; then npm ci; else npm install; fi
    npm run build
  )
else
  echo ">>> [2/8] 跳过前端构建（--skip-frontend）"
fi

echo ">>> [3/8] 交叉编译 linux/amd64 二进制"
mkdir -p dist
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -ldflags "-s -w -X main.version=${TAG}" \
  -o dist/bblog-linux-amd64 ./cmd/bblog
file dist/bblog-linux-amd64

echo ">>> [4/8] 构建镜像 ${IMAGE}（平台 ${PLATFORM}）"
docker buildx build --platform "${PLATFORM}" -f deploy/Dockerfile \
  -t "${IMAGE}" -t bblog:latest --load .
IMAGE_BYTES="$(docker image inspect "${IMAGE}" --format '{{.Size}}')"
echo "    镜像体积: $(( IMAGE_BYTES / 1024 / 1024 )) MB (${IMAGE_BYTES} 字节)"

echo ">>> [5/8] 服务器初始化（安装 Docker、建目录、配置 nginx，幂等）"
ssh "${SERVER_HOST}" "mkdir -p /opt/bblog/deploy"
rsync -az -e ssh deploy/ "${SERVER_HOST}:/opt/bblog/deploy/"
rsync -az -e ssh bblog.yaml "${SERVER_HOST}:/opt/bblog/deploy/bblog.yaml"
rsync -az --ignore-existing -e ssh bblog.yaml "${SERVER_HOST}:/opt/bblog/bblog.yaml"
ssh "${SERVER_HOST}" 'bash /opt/bblog/deploy/server_setup.sh'

echo ">>> [6/8] 传输到 ${SERVER_HOST} 并加载（流式，不落中间文件）"
# 两个标签一起保存：docker save 只导出被点名的标签，
# 只写 ${IMAGE} 会让服务器上没有 bblog:latest，与本地状态不一致。
started=$(date +%s)
docker save "${IMAGE}" bblog:latest | gzip -6 | ssh "${SERVER_HOST}" 'gunzip | docker load'
echo "    传输+加载耗时: $(( $(date +%s) - started ))s"

echo ">>> [7/8] 启动容器"
ssh "${SERVER_HOST}" "TAG=${TAG} bash -s" < deploy/remote_run.sh

echo ">>> [8/8] 健康检查"
for _ in $(seq 1 30); do
  if curl -fsS "${SITE_URL}/healthz" >/dev/null 2>&1; then
    curl -fsS "${SITE_URL}/healthz"; echo
    echo "部署完成：${SITE_URL}/ （镜像标签 ${TAG}）"
    exit 0
  fi
  sleep 2
done

echo "健康检查未通过，最近日志：" >&2
ssh "${SERVER_HOST}" 'docker logs --tail 40 bblog' >&2 || true
exit 1
