package web

import (
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

// Router 注册全部路由。
// 直出路由：文章页、独立页面、RSS、sitemap；其余前端路由返回 SPA 外壳。
func (s *Server) Router() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(rewriteHeadToGet())

	base := s.cfg.Site.BasePath
	g := r.Group(base)

	g.GET("/", s.handleSPA("home", true))
	g.GET("/tags/", s.handleSPA("tags", false))
	g.GET("/tags/:tag/", s.handleSPA("tag", false))
	g.GET("/archive/", s.handleSPA("archive", false))
	g.GET("/search/", s.handleSPA("search", false))

	g.GET("/posts/:year/:month/:slug/", s.handlePost)
	g.GET("/pages/:slug/", s.handlePage)

	g.GET("/api/posts", s.handleAPIPosts)
	g.GET("/api/tags", s.handleAPITags)
	g.GET("/api/archive", s.handleAPIArchive)
	g.GET("/api/search", s.handleAPISearch)
	g.POST("/api/admin/reload", s.handleAPIReload)

	g.GET("/rss.xml", s.handleRSS)
	g.GET("/sitemap.xml", s.handleSitemap)
	g.GET("/healthz", s.handleHealthz)

	g.GET("/media/*filepath", s.handleMedia)
	g.GET("/assets/*filepath", s.handleAsset)

	r.NoRoute(func(c *gin.Context) {
		urlPath := c.Request.URL.Path
		if base != "" && !strings.HasPrefix(urlPath, base) {
			s.notFoundJSON(c)
			return
		}
		rel := strings.TrimPrefix(urlPath, base)
		if rel == "" {
			rel = "/"
		}
		if strings.HasPrefix(rel, "/api/") {
			s.notFoundJSON(c)
			return
		}
		if !strings.HasSuffix(rel, "/") {
			// 目录型前端路由（/blog/tags）统一补斜杠，避免前端路由判定为文件
			if !strings.Contains(path.Base(rel), ".") {
				c.Redirect(http.StatusMovedPermanently, base+rel+"/")
				return
			}
			if s.serveDistPath(c, rel) {
				return
			}
			s.notFoundJSON(c)
			return
		}
		s.handleSPA("spa", false)(c)
	})

	return r
}

func (s *Server) notFoundJSON(c *gin.Context) {
	apiError(c, http.StatusNotFound, "not_found", "资源不存在")
}

// rewriteHeadToGet 让 HEAD 请求走 GET 处理；net/http 会自动丢弃响应体。
func rewriteHeadToGet() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodHead {
			c.Request.Method = http.MethodGet
		}
		c.Next()
	}
}
