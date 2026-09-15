package web

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Baike12/bblog/internal/config"
	"github.com/Baike12/bblog/internal/index"
)

const (
	defaultPageSize = 10
	maxPageSize     = 50
	maxSearchLimit  = 100
)

type postsResponse struct {
	Items      []entryView `json:"items"`
	Page       int         `json:"page"`
	Size       int         `json:"size"`
	Total      int         `json:"total"`
	TotalPages int         `json:"total_pages"`
}

type tagsResponse struct {
	Tags []index.Tag `json:"tags"`
}

type archiveMonth struct {
	Month     int         `json:"month"`
	MonthText string      `json:"month_text"`
	Count     int         `json:"count"`
	Items     []entryView `json:"items"`
}

type archiveYear struct {
	Year   int            `json:"year"`
	Count  int            `json:"count"`
	Months []archiveMonth `json:"months"`
}

type archiveResponse struct {
	Groups []archiveYear `json:"groups"`
	Total  int           `json:"total"`
}

type searchItem struct {
	entryView
	Score   int    `json:"score"`
	Excerpt string `json:"excerpt"`
}

type searchResponse struct {
	Q     string       `json:"q"`
	Count int          `json:"count"`
	Items []searchItem `json:"items"`
}

type healthResponse struct {
	Status     string `json:"status"`
	Version    string `json:"version"`
	Posts      int    `json:"posts"`
	Published  int    `json:"published"`
	Pages      int    `json:"pages"`
	IndexAgeS  int64  `json:"index_age_s"`
	ParseErrs  int    `json:"parse_errors"`
	ListenAddr string `json:"listen"`
}

func apiError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}

// postsResponse 构造文章列表响应；tag 与 year 为空表示不过滤。
func (s *Server) postsResponse(page, size int, tag, year string) postsResponse {
	all := s.Index().PublishedPosts()
	filtered := make([]index.Entry, 0, len(all))
	for _, e := range all {
		if tag != "" && !containsString(e.Tags, tag) {
			continue
		}
		if year != "" && e.Date.Format("2006") != year {
			continue
		}
		filtered = append(filtered, e)
	}

	total := len(filtered)
	totalPages := (total + size - 1) / size
	if totalPages == 0 {
		totalPages = 0
	}
	start := (page - 1) * size
	if start > total {
		start = total
	}
	end := start + size
	if end > total {
		end = total
	}
	items := make([]entryView, 0, end-start)
	for _, e := range filtered[start:end] {
		items = append(items, toEntryView(e))
	}
	return postsResponse{Items: items, Page: page, Size: size, Total: total, TotalPages: totalPages}
}

func (s *Server) handleAPIPosts(c *gin.Context) {
	page, err := parsePositiveInt(c.Query("page"), 1)
	if err != nil {
		apiError(c, http.StatusBadRequest, "invalid_param", "page 必须是正整数")
		return
	}
	size, err := parsePositiveInt(c.Query("size"), defaultPageSize)
	if err != nil {
		apiError(c, http.StatusBadRequest, "invalid_param", "size 必须是正整数")
		return
	}
	if size > maxPageSize {
		size = maxPageSize
	}
	c.JSON(http.StatusOK, s.postsResponse(page, size, c.Query("tag"), c.Query("year")))
}

func (s *Server) handleAPITags(c *gin.Context) {
	idx := s.Index()
	tags := idx.Tags
	if tags == nil {
		tags = []index.Tag{}
	}
	c.JSON(http.StatusOK, tagsResponse{Tags: tags})
}

func (s *Server) handleAPIArchive(c *gin.Context) {
	posts := s.Index().PublishedPosts()
	years := make([]archiveYear, 0)
	var current *archiveYear
	var currentMonth *archiveMonth
	for _, e := range posts {
		year := e.Date.Year()
		if current == nil || current.Year != year {
			years = append(years, archiveYear{Year: year})
			current = &years[len(years)-1]
			currentMonth = nil
		}
		month := int(e.Date.Month())
		if currentMonth == nil || currentMonth.Month != month {
			current.Months = append(current.Months, archiveMonth{
				Month:     month,
				MonthText: e.Date.Format("01月"),
			})
			currentMonth = &current.Months[len(current.Months)-1]
		}
		currentMonth.Items = append(currentMonth.Items, toEntryView(e))
		currentMonth.Count++
		current.Count++
	}
	c.JSON(http.StatusOK, archiveResponse{Groups: years, Total: len(posts)})
}

func (s *Server) handleAPISearch(c *gin.Context) {
	q := c.Query("q")
	limit, err := parsePositiveInt(c.Query("limit"), 20)
	if err != nil {
		apiError(c, http.StatusBadRequest, "invalid_param", "limit 必须是正整数")
		return
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}
	hits := s.Index().Search(q, limit)
	items := make([]searchItem, 0, len(hits))
	for _, h := range hits {
		items = append(items, searchItem{entryView: toEntryView(h.Entry), Score: h.Score, Excerpt: h.Excerpt})
	}
	c.JSON(http.StatusOK, searchResponse{Q: q, Count: len(items), Items: items})
}

func (s *Server) handleAPIReload(c *gin.Context) {
	expected := config.PublishToken()
	if expected == "" {
		apiError(c, http.StatusUnauthorized, "unauthorized", "服务端未配置 PUBLISH_TOKEN，拒绝重载")
		return
	}
	if c.GetHeader("X-Publish-Token") != expected {
		apiError(c, http.StatusUnauthorized, "unauthorized", "X-Publish-Token 不正确")
		return
	}
	started := time.Now()
	posts, pages, err := s.Reload()
	if err != nil {
		apiError(c, http.StatusInternalServerError, "reload_failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"reloaded": true,
		"posts":    posts,
		"pages":    pages,
		"ms":       time.Since(started).Milliseconds(),
	})
}

func (s *Server) handleHealthz(c *gin.Context) {
	idx := s.Index()
	posts, published, pages := idx.Counts()
	c.JSON(http.StatusOK, healthResponse{
		Status:     "ok",
		Version:    s.version,
		Posts:      posts,
		Published:  published,
		Pages:      pages,
		IndexAgeS:  int64(time.Since(idx.GeneratedAtTime()).Seconds()),
		ParseErrs:  len(s.FileErrors()),
		ListenAddr: s.cfg.Server.Listen,
	})
}

func parsePositiveInt(raw string, fallback int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 1 {
		return 0, errInvalidInt
	}
	return v, nil
}

var errInvalidInt = &invalidIntError{}

type invalidIntError struct{}

func (e *invalidIntError) Error() string { return "必须是正整数" }

func containsString(list []string, target string) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}
