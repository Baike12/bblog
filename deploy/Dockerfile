# 运行镜像：只拷贝本机交叉编译好的静态二进制，镜像内不需要 Go 工具链与 Node。
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -u 10001 bblog

WORKDIR /app
COPY dist/bblog-linux-amd64 /app/bblog

USER 10001
ENV TZ=Asia/Shanghai

# 容器内必须监听 0.0.0.0：宿主 -p 127.0.0.1:8090:8090 时由 docker-proxy 从容器 IP 转发
ENV BBLOG_LISTEN=0.0.0.0:8090
EXPOSE 8090

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s \
  CMD wget -qO- http://127.0.0.1:8090/blog/healthz || exit 1

ENTRYPOINT ["/app/bblog", "serve", "-config", "/app/bblog.yaml"]
