package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Baike12/bblog/internal/config"
	"github.com/Baike12/bblog/internal/content"
	"github.com/Baike12/bblog/internal/index"
	"github.com/Baike12/bblog/internal/render"
)

// Server 持有配置、索引与模板，索引通过原子指针热替换。
type Server struct {
	cfg     *config.Config
	version string
	started time.Time

	builder  *index.Builder
	cache    *render.Cache
	renderer *render.Renderer

	idx atomic.Pointer[index.Index]

	tmplPost     *template.Template
	tmplPage     *template.Template
	tmplSPA      *template.Template
	tmplNotFound *template.Template

	dist          fs.FS
	assets        assets
	hasFrontend   bool
	spaFallback   []byte
	contentDir    string
	indexPath     string
	reloadMu      sync.Mutex
	fileErrsMu    sync.RWMutex
	fileErrs      []content.FileError
	lastReloadErr atomic.Pointer[string]
}

// NewServer 加载渲染器、索引与模板。
func NewServer(cfg *config.Config, version string) (*Server, error) {
	katexJS, err := fs.ReadFile(staticFS, "static/katex/katex.min.js")
	if err != nil {
		return nil, fmt.Errorf("读取内嵌 KaTeX 失败: %w", err)
	}
	mathRenderer, err := render.NewMathRenderer(katexJS)
	if err != nil {
		return nil, err
	}

	dist, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, fmt.Errorf("读取内嵌前端产物失败: %w", err)
	}

	s := &Server{
		cfg:      cfg,
		version:  version,
		started:  time.Now(),
		renderer: render.NewRenderer(mathRenderer),
		dist:     dist,
	}

	s.cache = render.NewCache(cfg.DataDir() + "/cache")
	s.contentDir = cfg.ContentDir()
	s.indexPath = cfg.DataDir() + "/index.json"
	s.builder = &index.Builder{
		ContentDir: s.contentDir,
		IndexPath:  s.indexPath,
		BasePath:   cfg.Site.BasePath,
		Renderer:   s.renderer,
		Cache:      s.cache,
	}

	if s.tmplPost, err = template.ParseFS(templateFS, "templates/layout.html", "templates/post.html"); err != nil {
		return nil, fmt.Errorf("解析文章模板失败: %w", err)
	}
	if s.tmplPage, err = template.ParseFS(templateFS, "templates/layout.html", "templates/page.html"); err != nil {
		return nil, fmt.Errorf("解析页面模板失败: %w", err)
	}
	if s.tmplSPA, err = template.ParseFS(templateFS, "templates/spa.html"); err != nil {
		return nil, fmt.Errorf("解析 SPA 模板失败: %w", err)
	}
	if s.tmplNotFound, err = template.ParseFS(templateFS, "templates/layout.html", "templates/404.html"); err != nil {
		return nil, fmt.Errorf("解析 404 模板失败: %w", err)
	}

	s.assets = loadAssets(dist)
	if s.assets.Style != defaultAssets.Style || s.assets.Script != defaultAssets.Script {
		s.hasFrontend = true
	}
	if raw, readErr := fs.ReadFile(fallbackFS, "fallback/index.html"); readErr == nil {
		s.spaFallback = raw
	}

	if err := s.Load(); err != nil {
		return nil, err
	}
	return s, nil
}

// Load 读取磁盘索引；缺失或失效时全量重建。
func (s *Server) Load() error {
	prev, err := index.Load(s.indexPath)
	if err != nil {
		log.Printf("warn: %v", err)
		prev = nil
	}
	if prev != nil && prev.SourceDigest == index.Digest(s.contentDir) {
		s.idx.Store(prev)
		posts, published, pages := prev.Counts()
		log.Printf("索引已载入: 文章 %d（已发布 %d），页面 %d", posts, published, pages)
		return nil
	}
	_, _, err = s.Reload()
	return err
}

// Reload 全量重建索引并原子替换。未变化的源文件直接复用渲染缓存。
func (s *Server) Reload() (posts int, pages int, err error) {
	s.reloadMu.Lock()
	defer s.reloadMu.Unlock()

	prev := s.idx.Load()
	started := time.Now()
	idx, fileErrs, err := s.builder.Build(prev)
	if err != nil {
		msg := err.Error()
		s.lastReloadErr.Store(&msg)
		return 0, 0, err
	}
	s.idx.Store(idx)
	s.lastReloadErr.Store(nil)

	errList := fileErrs
	s.setFileErrors(errList)

	postCount, published, pageCount := idx.Counts()
	log.Printf("索引重建完成: 文章 %d（已发布 %d），页面 %d，耗时 %dms",
		postCount, published, pageCount, time.Since(started).Milliseconds())
	for _, fe := range fileErrs {
		log.Printf("warn: %s", fe.Error())
	}
	return postCount, pageCount, nil
}

func (s *Server) setFileErrors(errs []content.FileError) {
	// 保留在内存里，供健康检查与日志使用
	s.fileErrsMu.Lock()
	s.fileErrs = errs
	s.fileErrsMu.Unlock()
}

// FileErrors 返回最近一次重建时跳过的文件。
func (s *Server) FileErrors() []content.FileError {
	s.fileErrsMu.RLock()
	defer s.fileErrsMu.RUnlock()
	out := make([]content.FileError, len(s.fileErrs))
	copy(out, s.fileErrs)
	return out
}

// Index 返回当前索引。
func (s *Server) Index() *index.Index { return s.idx.Load() }

// BootstrapJSON 把站点信息、导航与首屏数据序列化成 JSON，供前端首屏直接使用。
func (s *Server) BootstrapJSON(route string, initial any) template.JS {
	payload := map[string]any{
		"site":  s.siteView(),
		"nav":   s.navEntries(),
		"route": route,
	}
	if initial != nil {
		payload["initial"] = initial
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return template.JS("{}")
	}
	return template.JS(raw)
}
