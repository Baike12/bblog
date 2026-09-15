package web

import (
	"bytes"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	assetCacheControl = "public, max-age=2592000, immutable"
	mediaCacheControl = "public, max-age=604800"
)

// embeddedAssetDirs 把 URL 前缀映射到内嵌静态资源目录。
var embeddedAssetDirs = map[string]string{
	"/katex/":  "static/katex",
	"/chroma/": "static/chroma",
}

// serveEmbeddedFile 从内嵌文件系统读取并返回文件。
func serveEmbeddedFile(c *gin.Context, fsys fs.FS, name, cacheControl string) bool {
	name = strings.TrimPrefix(path.Clean("/"+name), "/")
	if name == "" || name == "." {
		return false
	}
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return false
	}
	if ct := mime.TypeByExtension(path.Ext(name)); ct != "" {
		c.Header("Content-Type", ct)
	}
	if cacheControl != "" {
		c.Header("Cache-Control", cacheControl)
	}
	http.ServeContent(c.Writer, c.Request, path.Base(name), time.Time{}, bytes.NewReader(data))
	return true
}

// handleAsset 提供 /assets/** 下的资源：
// 先查前端构建产物（dist/assets/**），再回退到内嵌静态资源（katex、chroma）。
// gin 不允许 /assets/*filepath 与 /assets/katex/*filepath 共存，因此在这里分流。
func (s *Server) handleAsset(c *gin.Context) {
	rel := c.Param("filepath")
	// Vite 的产物位于 dist/assets/ 下，URL 前缀已由路由剥掉，这里补回目录
	if serveEmbeddedFile(c, s.dist, path.Join("assets", rel), assetCacheControl) {
		return
	}
	for prefix, dir := range embeddedAssetDirs {
		if !strings.HasPrefix(rel, prefix) {
			continue
		}
		sub, err := fs.Sub(staticFS, dir)
		if err != nil {
			continue
		}
		if serveEmbeddedFile(c, sub, strings.TrimPrefix(rel, prefix), assetCacheControl) {
			return
		}
	}
	s.renderNotFound(c)
}

// handleMedia 提供内容目录下的图片等静态资源，路径固定为 content/media/**。
func (s *Server) handleMedia(c *gin.Context) {
	rel := c.Param("filepath")
	cleaned := path.Clean("/" + rel)
	if strings.Contains(cleaned, "..") {
		c.Status(http.StatusBadRequest)
		return
	}
	full := filepath.Join(s.contentDir, "media", filepath.FromSlash(strings.TrimPrefix(cleaned, "/")))
	info, err := os.Stat(full)
	if err != nil || info.IsDir() {
		s.renderNotFound(c)
		return
	}
	f, err := os.Open(full)
	if err != nil {
		s.renderNotFound(c)
		return
	}
	defer f.Close()
	if ct := mime.TypeByExtension(filepath.Ext(full)); ct != "" {
		c.Header("Content-Type", ct)
	}
	c.Header("Cache-Control", mediaCacheControl)
	http.ServeContent(c.Writer, c.Request, info.Name(), info.ModTime(), f)
}

// serveDistPath 尝试从内嵌前端产物中返回指定文件，用于 favicon 等根级文件。
func (s *Server) serveDistPath(c *gin.Context, urlPath string) bool {
	return serveEmbeddedFile(c, s.dist, urlPath, assetCacheControl)
}
