package web

import (
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Baike12/bblog/internal/content"
	"github.com/Baike12/bblog/internal/index"
)

type siteView struct {
	Title       string `json:"title"`
	Author      string `json:"author"`
	Description string `json:"description"`
	BasePath    string `json:"base_path"`
	BaseURL     string `json:"base_url"`
	Language    string `json:"language"`
}

type entryView struct {
	Slug           string   `json:"slug"`
	URL            string   `json:"url"`
	Title          string   `json:"title"`
	Summary        string   `json:"summary"`
	Date           string   `json:"date"`
	DateText       string   `json:"date_text"`
	Updated        string   `json:"updated,omitempty"`
	UpdatedText    string   `json:"updated_text,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	ReadingMinutes int      `json:"reading_minutes"`
	WordCount      int      `json:"word_count"`
	Draft          bool     `json:"draft,omitempty"`
	Private        bool     `json:"private,omitempty"`
}

type postView struct {
	entryView
	Body     template.HTML
	Headings []content.Heading
	HasMath  bool
	Prev     *entryView
	Next     *entryView
}

type pageContentView struct {
	Title     string
	Body      template.HTML
	Headings  []content.Heading
	Updated   string
	UpdatedTx string
}

type giscusView struct {
	Repo       string
	RepoID     string
	Category   string
	CategoryID string
	Mapping    string
	Lang       string
}

// pageCommon 是所有直出页面共用的字段，layout.html 依赖这一组字段。
type pageCommon struct {
	Site        siteView
	Assets      assets
	Nav         []entryView
	Title       string
	Description string
	Canonical   string
	OGType      string
	HasMath     bool
	NoIndex     bool
}

type postPageData struct {
	pageCommon
	Post   postView
	Giscus giscusView
}

type pagePageData struct {
	pageCommon
	Page pageContentView
}

type spaPageData struct {
	Site      siteView
	Assets    assets
	Nav       []entryView
	Title     string
	Bootstrap template.JS
	Skeleton  []entryView
}

type notFoundData struct {
	pageCommon
}

// absolute 把带基础路径的站点内路径拼成含域名的绝对地址。
func (s *Server) absolute(p string) string { return s.cfg.Site.BaseURL + p }

func (s *Server) siteView() siteView {	return siteView{
		Title:       s.cfg.Site.Title,
		Author:      s.cfg.Site.Author,
		Description: s.cfg.Site.Description,
		BasePath:    s.cfg.Site.BasePath,
		BaseURL:     s.cfg.Site.BaseURL,
		Language:    s.cfg.Site.Language,
	}
}

func (s *Server) giscusView() giscusView {
	return giscusView{
		Repo:       s.cfg.Giscus.Repo,
		RepoID:     s.cfg.Giscus.RepoID,
		Category:   s.cfg.Giscus.Category,
		CategoryID: s.cfg.Giscus.CategoryID,
		Mapping:    s.cfg.Giscus.Mapping,
		Lang:       s.cfg.Giscus.Lang,
	}
}

func (s *Server) navEntries() []entryView {
	pages := s.Index().Pages()
	out := make([]entryView, 0, len(pages))
	for _, p := range pages {
		out = append(out, toEntryView(p))
	}
	return out
}

func toEntryView(e index.Entry) entryView {
	v := entryView{
		Slug:           e.Slug,
		URL:            e.URL,
		Title:          e.Title,
		Summary:        e.Summary,
		Date:           e.Date.Format("2006-01-02"),
		DateText:       e.Date.Format("2006年01月02日"),
		Tags:           e.Tags,
		ReadingMinutes: e.ReadingMinutes,
		WordCount:      e.WordCount,
		Draft:          e.Draft,
		Private:        e.Private,
	}
	if !e.Updated.IsZero() {
		v.Updated = e.Updated.Format("2006-01-02")
		v.UpdatedText = e.Updated.Format("2006年01月02日")
	}
	return v
}

func (s *Server) handlePost(c *gin.Context) {
	year, month, slug := c.Param("year"), c.Param("month"), c.Param("slug")
	entry, ok := s.Index().FindPost(year, month, slug)
	if !ok {
		s.renderNotFound(c)
		return
	}
	body, ok := s.cache.Read(entry.HTMLCache)
	if !ok {
		body = "<p>渲染缓存缺失，请触发一次重建。</p>"
	}
	view := postView{
		entryView: toEntryView(entry),
		Body:      template.HTML(body),
		Headings:  entry.Headings,
		HasMath:   entry.HasMath,
	}
	prev, next, hasPrev, hasNext := s.Index().Neighbors(entry)
	if hasPrev {
		v := toEntryView(prev)
		view.Prev = &v
	}
	if hasNext {
		v := toEntryView(next)
		view.Next = &v
	}

	if entry.Private {
		c.Header("X-Robots-Tag", "noindex, nofollow")
	}
	data := postPageData{
		pageCommon: pageCommon{
			Site:        s.siteView(),
			Assets:      s.assets,
			Nav:         s.navEntries(),
			Title:       entry.Title + " · " + s.cfg.Site.Title,
			Description: entry.Summary,
			Canonical:   s.absolute(entry.URL),
			OGType:      "article",
			HasMath:     entry.HasMath,
			NoIndex:     entry.Private,
		},
		Post:   view,
		Giscus: s.giscusView(),
	}
	c.Status(http.StatusOK)
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := s.tmplPost.ExecuteTemplate(c.Writer, "layout", data); err != nil {
		c.Error(err)
	}
}

func (s *Server) handlePage(c *gin.Context) {
	slug := c.Param("slug")
	entry, ok := s.Index().FindPage(slug)
	if !ok {
		s.renderNotFound(c)
		return
	}
	body, ok := s.cache.Read(entry.HTMLCache)
	if !ok {
		body = "<p>渲染缓存缺失，请触发一次重建。</p>"
	}
	pageContent := pageContentView{
		Title:    entry.Title,
		Body:     template.HTML(body),
		Headings: entry.Headings,
	}
	// 未设置 updated 时保持为空，避免把零值时间格式化成 0001年01月01日
	if !entry.Updated.IsZero() {
		pageContent.Updated = entry.Updated.Format("2006-01-02")
		pageContent.UpdatedTx = entry.Updated.Format("2006年01月02日")
	}
	data := pagePageData{
		pageCommon: pageCommon{
			Site:        s.siteView(),
			Assets:      s.assets,
			Nav:         s.navEntries(),
			Title:       entry.Title + " · " + s.cfg.Site.Title,
			Description: entry.Summary,
			Canonical:   s.absolute(entry.URL),
			OGType:      "website",
			HasMath:     entry.HasMath,
		},
		Page: pageContent,
	}
	c.Status(http.StatusOK)
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := s.tmplPage.ExecuteTemplate(c.Writer, "layout", data); err != nil {
		c.Error(err)
	}
}

// handleSPA 返回 React 应用外壳；withInitial 为 true 时注入首页首屏数据。
func (s *Server) handleSPA(route string, withInitial bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !s.hasFrontend {
			s.serveSPAFallback(c)
			return
		}
		recent := s.Index().PublishedPosts()
		skeletonLimit := 10
		if len(recent) > skeletonLimit {
			recent = recent[:skeletonLimit]
		}
		skeleton := make([]entryView, 0, len(recent))
		for _, e := range recent {
			skeleton = append(skeleton, toEntryView(e))
		}

		var initial any
		if withInitial {
			initial = s.postsResponse(1, defaultPageSize, "", "")
		}
		data := spaPageData{
			Site:      s.siteView(),
			Assets:    s.assets,
			Nav:       s.navEntries(),
			Title:     s.cfg.Site.Title,
			Bootstrap: s.BootstrapJSON(route, initial),
			Skeleton:  skeleton,
		}
		c.Status(http.StatusOK)
		c.Header("Content-Type", "text/html; charset=utf-8")
		if err := s.tmplSPA.ExecuteTemplate(c.Writer, "spa", data); err != nil {
			c.Error(err)
		}
	}
}

func (s *Server) serveSPAFallback(c *gin.Context) {
	if len(s.spaFallback) == 0 {
		c.String(http.StatusOK, "前端未构建：请在 web/ 目录执行 npm ci && npm run build 后重新编译后端。")
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", s.spaFallback)
}

func (s *Server) renderNotFound(c *gin.Context) {
	data := notFoundData{
		pageCommon: pageCommon{
			Site:   s.siteView(),
			Assets: s.assets,
			Nav:    s.navEntries(),
			Title:  "页面不存在 · " + s.cfg.Site.Title,
		},
	}
	c.Status(http.StatusNotFound)
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := s.tmplNotFound.ExecuteTemplate(c.Writer, "layout", data); err != nil {
		c.Error(err)
	}
}
