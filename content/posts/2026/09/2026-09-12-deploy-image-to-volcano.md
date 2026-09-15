---
title: 把镜像传到火山服务器
date: 2026-09-12 10:05:00
updated: 2026-09-13 09:20:00
tags: [部署, docker, 火山引擎]
summary: 服务器访问不了 Docker Hub，所以镜像在本机交叉构建后 docker save、gzip、ssh 流式传输、docker load，全程不需要镜像仓库。
---

## 约束

服务器实测 `registry-1.docker.io` 超时不可达，而本机 Docker Hub 可达。因此：

1. 基础镜像只在本机拉取；
2. 服务器只做 `docker load`，不执行任何 `docker pull`。

## 传输命令

```bash
docker save "bblog:$TAG" | gzip -6 \
  | ssh root@101.96.243.197 'gunzip | docker load'
```

管道方式不落中间文件，本机与服务器都不占额外磁盘。

## 架构必须显式指定

本机是 arm64，服务器是 x86_64，构建时必须写 `--platform linux/amd64`，否则容器启动会报 `exec format error`：

```bash
docker buildx build --platform linux/amd64 -f deploy/Dockerfile -t bblog:$TAG --load .
```

## 网络与端口

容器内监听 `0.0.0.0:8090`（宿主端口映射时 docker-proxy 从容器 IP 转发，绑定 127.0.0.1 会连不上），宿主绑定 `127.0.0.1:8090`，由 nginx 的 `/blog/` 反代对外。

| 项 | 值 |
|---|---|
| 容器端口 | 8090 |
| 宿主绑定 | 127.0.0.1:8090 |
| 对外路径 | http://101.96.243.197/blog/ |
