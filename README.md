# bblog

一个以 Markdown 文件为唯一真相的个人博客：Go 服务端直出 HTML，React 负责列表与交互，内容用 rsync 发布，整体以单容器部署在火山服务器上。

- 线上地址：http://101.96.243.197/blog/
- 代码仓库：https://github.com/Baike12/bblog

## 架构

```
浏览器
  │  http://101.96.243.197/blog/...
  ▼
宿主 nginx :80（sites-available/iteach.conf，server_name _）
  ├── location /blog/  → proxy_pass http://127.0.0.1:8090
  └── location /       → /opt/iteach/frontend（另一个站点，保持不变）
                              │
                              ▼
                     容器 bblog（alpine + 静态 Go 二进制，:8090）
                       ├── /app/bblog            二进制（内嵌前端产物 + KaTeX + chroma 样式）
                       ├── /app/content   ←bind→ /opt/bblog/content   文章与图片
                       ├── /app/data      ←bind→ /opt/bblog/data      index.json + 渲染缓存
                       └── /app/bblog.yaml ←ro→  /opt/bblog/bblog.yaml
```

| 路径 | 处理方式 |
|---|---|
| `/blog/posts/YYYY/MM/{slug}/` | Go 直出完整 HTML（正文、标签、上下篇、giscus） |
| `/blog/pages/{slug}/` | Go 直出独立页面（如关于页） |
| `/blog/`、`/blog/tags/`、`/blog/archive/`、`/blog/search/` | SPA 外壳 + React 客户端路由 |
| `/blog/api/**` | JSON 接口 |
| `/blog/rss.xml`、`/blog/sitemap.xml` | 服务端生成 |
| `/blog/media/**` | 内容目录里的图片 |
| `/blog/assets/**` | 前端产物与 KaTeX、chroma 样式 |
| `/blog/healthz` | 健康检查 |

## 目录结构

```
cmd/bblog/            入口（serve / check / version）
internal/config/      YAML 配置 + 环境变量覆盖
internal/content/     frontmatter 解析、目录扫描、slug 与 URL、内容校验
internal/render/      goldmark 渲染、chroma 高亮、goja + KaTeX 服务端公式渲染、渲染缓存
internal/index/       索引构建、原子落盘、增量复用、搜索打分
internal/watch/       fsnotify 监听 + 去抖
internal/web/         gin 路由、直出模板、API、RSS/sitemap、内嵌资源
web/                  Vite + React + TypeScript + Tailwind v4
content/              文章与图片（唯一真相）
scripts/              build.sh / publish.sh / deploy.sh
deploy/               Dockerfile、server_setup.sh、remote_run.sh、nginx 片段
tools/chroma-css/     生成深浅色代码高亮样式表
_references/          6 个参考博客项目（不入库，Go 会忽略 _ 前缀目录）
```

## 本地开发

```bash
scripts/build.sh          # 生成 chroma 样式 → 构建前端 → 编译后端
./dist/bblog serve        # 启动，默认 http://127.0.0.1:8090/blog/
go run ./cmd/bblog check  # 只校验内容（YAML、日期、slug 冲突、图片引用）
go test ./...             # 单元测试
```

前端热更新：`cd web && npm run dev`（开发服务器默认 5173，接口仍走 8090，需要自行配置代理）。

## 写文章

文章放 `content/` 下，Markdown 与原生 HTML 都支持。

### frontmatter

Markdown 用 `---` 包裹 YAML；`.html` 用 `<!-- ... -->` 包裹同一份 YAML。字段都可省略。

| 字段 | 类型 | 缺省行为 |
|---|---|---|
| `title` | string | 取文件名（去掉 `YYYY-MM-DD-` 前缀） |
| `date` | `YYYY-MM-DD[ HH:MM[:SS]]` 或 RFC3339 | 取文件修改时间 |
| `updated` | 同上 | 不显示"更新于" |
| `tags` | 列表或逗号分隔字符串 | 空 |
| `summary` | string | 取正文纯文本前 120 字 |
| `slug` | string | 取文件名（去掉日期前缀） |
| `draft` | bool | `false` |
| `private` | bool | `false` |
| `lang` | string | `zh` |

### 目录语义

| 目录 | 语义 |
|---|---|
| `content/posts/**` | 已发布文章，参与列表/标签/归档/RSS/搜索 |
| `content/draft/**` | 草稿，等同 `draft: true`，直链也返回 404 |
| `content/private/**` | 私密文章，不进任何列表，直链可访问并带 `X-Robots-Tag: noindex` |
| `content/trash/**` | 完全忽略 |
| `content/pages/**` | 独立页面 |
| `content/media/**` | 图片，映射到 `/blog/media/**` |

### 正文写法

- URL 由 `date` 决定：`/blog/posts/{YYYY}/{MM}/{slug}/`。
- 代码块用 ``` 围栏，服务端用 chroma 高亮（class 模式，深浅色各一套配色）。
- 公式支持 `$...$`、`$$...$$`、`\(...\)`、`\[...\]`，由 goja 执行 KaTeX 在**服务端**渲染成 HTML，单公式超过 200ms 会中断并降级为纯文本。
- 原始 HTML 原样输出（写入通道只有 SSH/rsync，作者可信）。
- 中文标题的锚点保留中文，例如 `## 架构` → `id="架构"`。
- 图片写 `/blog/media/xxx.png`，或写相对路径（相对于文章所在目录）。

## 发布文章

```bash
scripts/publish.sh
```

三步：`bblog check` 校验内容 → `rsync -az --delete content/ root@服务器:/opt/bblog/content/` → 调用 `/blog/api/admin/reload` 兜底重载。

不重建镜像、不重启容器。容器内 fsnotify 会在 500ms 去抖后自动重建索引，重载接口只是兜底。

首次使用需要把服务器上的发布令牌取回本地：

```bash
ssh root@101.96.243.197 'grep PUBLISH_TOKEN /opt/bblog/.env' > /tmp/token
mkdir -p .secrets && cat /tmp/token | cut -d= -f2 > .secrets/publish_token
```

## 部署

```bash
scripts/deploy.sh                  # 完整部署
scripts/deploy.sh --skip-frontend  # 只改了后端时更快
scripts/deploy.sh --rollback 20260915-1430   # 回滚到指定标签
```

### 镜像传输设计

服务器实测访问不了 Docker Hub（`registry-1.docker.io` 超时），而本机可达，因此流程是：

1. 本机 `docker buildx build --platform linux/amd64` 构建镜像（本机是 arm64，必须显式指定平台，否则服务器启动报 `exec format error`）；
2. `docker save ${IMAGE} bblog:latest | gzip -6 | ssh root@服务器 'gunzip | docker load'` 流式传输，不落中间文件。两个标签必须一起点名：`docker save` 只导出被指定的标签，只写 `${IMAGE}` 会让服务器上没有 `bblog:latest`，与本地状态不一致；
3. 服务器只做 `docker load`，不执行任何 `docker pull`，因此不需要配置镜像加速器。

镜像里只有 alpine 基础层与一个静态 Go 二进制（前端产物、KaTeX、chroma 样式都已 `go:embed`），不含 Go 工具链与 Node。

容器参数：`-p 127.0.0.1:8090:8090`（只在本机可达）、`--restart unless-stopped`、挂载 `content`/`data`/`bblog.yaml`、日志限制 10m×3。容器内监听 `0.0.0.0:8090`——宿主端口映射时由 docker-proxy 从容器 IP 转发，绑定容器内 127.0.0.1 会连不上。

服务器上保留最近 3 个镜像标签用于回滚。

### 服务器初始化

`deploy/server_setup.sh`（幂等，由 deploy.sh 调用）：

1. `apt-get install -y docker.io`（源是 `mirrors.ivolces.com`）；
2. 建 `/opt/bblog/{content,data}`，`data` 归属 UID 10001（容器内运行用户）；
3. 写 `/opt/bblog/bblog.yaml` 与 `.env`（含随机 `PUBLISH_TOKEN`），已存在则不覆盖；
4. 安装 nginx 片段 `/etc/nginx/snippets/bblog.conf`，并在 `iteach.conf` 的 server 块内插入一行 `include`（改动前自动备份，插入失败会报错退出）；
5. `nginx -t` 校验后 `systemctl reload nginx`。

不新建独立 server 块的原因：现有 iteach 站点是 `server_name _` 的 catch-all，再加一个 catch-all 会冲突。

### 部署验收

2026-09-15 首次部署的实测结果（`scripts/deploy.sh` 全流程 exit 0，耗时约 30s，其中传输+加载 16s）：

| 检查项 | 命令 | 实测结果 |
|---|---|---|
| 镜像架构 | `docker inspect bblog:latest --format '{{.Architecture}}'` | `amd64`（`Os` 为 `linux`） |
| 部署的二进制与本机一致 | 比对 `sha256sum /app/bblog` 与 `dist/bblog-linux-amd64` | 均为 `604ae367234472c62751b4c881d3d12a602aa642fac813439f1d51325480ed9e` |
| 端口未对外暴露 | `ss -lntp \| grep 8090` | 仅 `127.0.0.1:8090`（docker-proxy） |
| 容器自愈 | `docker inspect bblog --format '{{.HostConfig.RestartPolicy.Name}}'` | `unless-stopped`，容器报告 `healthy` |
| 挂载 | `docker inspect bblog --format '{{range .Mounts}}...'` | `content`/`data` 读写、`bblog.yaml` 只读 |
| 健康检查 | `curl http://101.96.243.197/blog/healthz` | 200，`posts:5 published:3 pages:1 parse_errors:0` |
| 未破坏既有站点 | `curl http://101.96.243.197/` | 200（iteach 页面） |
| 镜像体积 | `docker image inspect bblog:latest --format '{{.Size}}'` | 41,658,380 字节（39 MB），gzip 后 15,445,756 字节 |
| 服务器磁盘 | `df -h /` | 40G 中已用 4.9G，可用 33G |

服务器上保留 `bblog:latest` 加最近 3 个带时间戳的标签（`deploy/remote_run.sh` 里的保留逻辑按创建时间倒序删掉第 4 个之后的时间戳标签），用于 `--rollback`。`deploy/server_setup.sh` 重复执行时跳过已装好的 Docker、保留既有 `bblog.yaml` 与 `.env`、跳过已插入的 nginx include。

## 配置

`bblog.yaml`（服务器上位于 `/opt/bblog/bblog.yaml`，改动后需重启容器）：

```yaml
site:
  title: 我的博客
  author: Baike
  description: 记录技术与部署的博客
  base_url: http://101.96.243.197
  base_path: /blog
  language: zh-CN
server:
  listen: 0.0.0.0:8090
content:
  dir: content      # 相对路径以配置文件所在目录为基准
data:
  dir: data
giscus:
  repo: Baike12/bblog
  repo_id: R_kgDOUcDEAA
  category: Announcements
  category_id: DIC_kwDOUcDEAM4DFqu7
  mapping: pathname
  lang: zh-CN
```

环境变量覆盖：`BBLOG_LISTEN`、`BBLOG_CONTENT_DIR`、`BBLOG_DATA_DIR`、`PUBLISH_TOKEN`。

## API

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/blog/api/posts?page=&size=&tag=&year=` | 已发布文章分页列表 |
| GET | `/blog/api/tags` | 标签与计数 |
| GET | `/blog/api/archive` | 按年月分组 |
| GET | `/blog/api/search?q=&limit=` | 站内搜索 |
| POST | `/blog/api/admin/reload` | 强制重建索引，需要 `X-Publish-Token` 头 |
| GET | `/blog/healthz` | 状态、文章数、索引年龄 |

错误统一为 `{"error":{"code":"...","message":"..."}}`。

## 已知限制

- **giscus 需要安装 GitHub App**：仓库 `Baike12/bblog` 已开启 Discussions，但还需在 https://github.com/apps/giscus 上为该仓库安装 giscus 应用，否则评论区会显示 "giscus is not installed on this repository"。
- 中文搜索不做分词，只做子串匹配：`部署镜像` 能命中，`镜像 部署` 不能命中。
- 私密文章只保证"不进列表、不被收录"，知道直链即可访问；需要口令保护要另行实现。
- 未构建前端时（全新克隆直接 `go build`），SPA 页面会显示编译期兜底页；文章页、RSS 与 API 不受影响。
- `internal/web/dist/` 不入库，仓库只保留 `dist/.gitkeep`。构建前会清空该目录的产物但保留 `.gitkeep`（`web/scripts/clean-dist.mjs`），因此 vite 的 `emptyOutDir` 是关闭的。
