package web

import "embed"

// 前端构建产物（Vite 输出，不入库）。
// 用 all: 前缀是为了让 dist/.gitkeep 被识别，保证全新克隆下 go build 也能通过。
//
//go:embed all:dist
var distFS embed.FS

// 未构建前端时展示的兜底页（入库，不会被 Vite 覆盖）。
//
//go:embed fallback
var fallbackFS embed.FS

//go:embed templates/*.html
var templateFS embed.FS

// 内嵌静态资源：KaTeX（公式样式与字体）与 chroma 代码高亮样式。
// 全部内嵌，不依赖任何外部 CDN。
//
//go:embed static
var staticFS embed.FS
