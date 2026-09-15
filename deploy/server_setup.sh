#!/usr/bin/env bash
# 服务器端一次性初始化（幂等，可重复执行）：
#   1. 安装 Docker（apt 源 mirrors.ivolces.com）
#   2. 创建 /opt/bblog/{content,data} 并设置权限
#   3. 写入配置与发布令牌（已存在则不覆盖）
#   4. 安装 nginx 片段并在 iteach 站点里 include 一行
#   5. nginx -t 校验后 reload
#
# 不需要配置镜像加速器：部署走 docker load，服务器不执行 docker pull。
set -euo pipefail

APP_DIR="/opt/bblog"
NGINX_SNIPPET="/etc/nginx/snippets/bblog.conf"
ITEACH_CONF="/etc/nginx/sites-available/iteach.conf"
CONTAINER_UID=10001

echo ">>> [1/5] 安装 Docker"
if ! command -v docker >/dev/null 2>&1; then
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -qq
  apt-get install -y -qq docker.io
fi
systemctl enable --now docker >/dev/null 2>&1 || true
docker --version

echo ">>> [2/5] 创建目录"
mkdir -p "${APP_DIR}/content" "${APP_DIR}/data" "${APP_DIR}/deploy"
# 容器以 UID 10001 运行，索引与渲染缓存要能写
chown -R "${CONTAINER_UID}:${CONTAINER_UID}" "${APP_DIR}/data"
chmod 755 "${APP_DIR}/content" "${APP_DIR}/data"

echo ">>> [3/5] 配置与密钥（已存在则保留）"
if [ ! -f "${APP_DIR}/bblog.yaml" ]; then
  install -m 644 "${APP_DIR}/deploy/bblog.yaml" "${APP_DIR}/bblog.yaml"
  echo "    已写入 ${APP_DIR}/bblog.yaml"
else
  echo "    保留现有 ${APP_DIR}/bblog.yaml"
fi
if [ ! -f "${APP_DIR}/.env" ]; then
  printf 'PUBLISH_TOKEN=%s\n' "$(openssl rand -hex 24)" > "${APP_DIR}/.env"
  chmod 600 "${APP_DIR}/.env"
  echo "    已生成 ${APP_DIR}/.env（发布令牌）"
else
  echo "    保留现有 ${APP_DIR}/.env"
fi

echo ">>> [4/5] 安装 nginx 片段"
mkdir -p /etc/nginx/snippets
install -m 644 "${APP_DIR}/deploy/nginx/bblog.conf" "${NGINX_SNIPPET}"
if grep -q 'snippets/bblog.conf' "${ITEACH_CONF}"; then
  echo "    ${ITEACH_CONF} 已包含该片段，跳过"
else
  cp "${ITEACH_CONF}" "${ITEACH_CONF}.bak.$(date +%Y%m%d%H%M%S)"
  # 在 server 块的最后一个 } 之前插入 include，iteach 站点其它配置保持不变
  awk '
    { lines[NR] = $0 }
    END {
      last = NR
      while (last > 0 && lines[last] ~ /^[[:space:]]*$/) last--
      for (i = 1; i <= NR; i++) {
        if (i == last && lines[i] ~ /^[[:space:]]*}/) {
          print "    include /etc/nginx/snippets/bblog.conf;"
        }
        print lines[i]
      }
    }
  ' "${ITEACH_CONF}" > "${ITEACH_CONF}.tmp"
  mv "${ITEACH_CONF}.tmp" "${ITEACH_CONF}"
  if ! grep -q 'snippets/bblog.conf' "${ITEACH_CONF}"; then
    echo "错误：未能在 ${ITEACH_CONF} 中插入 include，请手工添加：" >&2
    echo "    include /etc/nginx/snippets/bblog.conf;   # 放在 server { } 块内" >&2
    exit 1
  fi
  echo "    已在 ${ITEACH_CONF} 中插入 include（原文件已备份）"
fi

echo ">>> [5/5] 校验并重载 nginx"
nginx -t
systemctl reload nginx

echo
echo "初始化完成。发布令牌位于 ${APP_DIR}/.env，取回本地后写入 .secrets/publish_token："
echo "  ssh root@<服务器> 'grep PUBLISH_TOKEN ${APP_DIR}/.env'"
