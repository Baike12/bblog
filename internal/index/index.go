package index

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Baike12/bblog/internal/content"
)

// Version 是索引结构版本，结构变更时递增会让旧索引被判定为失效。
const Version = 1

// RenderVersion 是渲染管线版本，渲染逻辑变更时递增会让缓存 HTML 全部重算。
const RenderVersion = 1

// Entry 是索引中的一条内容（文章或独立页面）。
type Entry struct {
	Kind   content.Kind   `json:"kind"`
	Format content.Format `json:"format"`

	Slug    string    `json:"slug"`
	URL     string    `json:"url"`
	Title   string    `json:"title"`
	Date    time.Time `json:"date"`
	Updated time.Time `json:"updated,omitempty"`
	Tags    []string  `json:"tags,omitempty"`
	Summary string    `json:"summary"`

	Draft   bool `json:"draft,omitempty"`
	Private bool `json:"private,omitempty"`

	SourcePath  string `json:"source_path"`
	SourceMtime int64  `json:"source_mtime"`
	SourceSize  int64  `json:"source_size"`

	WordCount      int    `json:"word_count"`
	ReadingMinutes int    `json:"reading_minutes"`
	HTMLCache      string `json:"html_cache"`
	HasMath        bool   `json:"has_math,omitempty"`

	RenderVersion int                `json:"render_version"`
	Headings      []content.Heading  `json:"headings,omitempty"`
}

// Published 表示该条目是否应出现在列表、标签、归档、RSS、搜索中。
func (e Entry) Published() bool {
	return e.Kind == content.KindPost && !e.Draft && !e.Private
}

// Tag 是标签及其文章数。
type Tag struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// Index 是全部内容的索引，可原子落盘到 JSON 文件。
type Index struct {
	Version      int               `json:"version"`
	GeneratedAt  time.Time         `json:"generated_at"`
	SourceDigest string            `json:"source_digest"`
	Entries      []Entry           `json:"entries"`
	Tags         []Tag             `json:"tags"`
	SearchText   map[string]string `json:"search_text"`

	mu sync.RWMutex
}

func newIndex() *Index {
	return &Index{Version: Version, Tags: []Tag{}, Entries: []Entry{}, SearchText: map[string]string{}}
}

// Posts 返回全部文章（含草稿与私密），按时间倒序。
func (i *Index) Posts() []Entry {
	i.mu.RLock()
	defer i.mu.RUnlock()
	out := make([]Entry, 0, len(i.Entries))
	for _, e := range i.Entries {
		if e.Kind == content.KindPost {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].Date.Equal(out[b].Date) {
			return out[a].Slug < out[b].Slug
		}
		return out[a].Date.After(out[b].Date)
	})
	return out
}

// PublishedPosts 返回已发布文章，按时间倒序。
func (i *Index) PublishedPosts() []Entry {
	out := make([]Entry, 0, len(i.Entries))
	for _, e := range i.Posts() {
		if e.Published() {
			out = append(out, e)
		}
	}
	return out
}

// Pages 返回独立页面。
func (i *Index) Pages() []Entry {
	i.mu.RLock()
	defer i.mu.RUnlock()
	out := make([]Entry, 0, len(i.Entries))
	for _, e := range i.Entries {
		if e.Kind == content.KindPage {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Slug < out[b].Slug })
	return out
}

// FindPost 按年月与 slug 定位文章。
// 草稿一律不可访问（404）；私密文章可由直链访问，由上层附加 noindex 响应头。
func (i *Index) FindPost(year, month, slug string) (Entry, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	for _, e := range i.Entries {
		if e.Kind != content.KindPost || e.Slug != slug || e.Draft {
			continue
		}
		if e.Date.Format("2006") == year && e.Date.Format("01") == month {
			return e, true
		}
	}
	return Entry{}, false
}

// FindPage 按 slug 定位独立页面。
func (i *Index) FindPage(slug string) (Entry, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	for _, e := range i.Entries {
		if e.Kind == content.KindPage && e.Slug == slug {
			return e, true
		}
	}
	return Entry{}, false
}

// Neighbors 返回已发布文章中的上一篇与下一篇（按时间倒序相邻）。
func (i *Index) Neighbors(e Entry) (prev, next Entry, hasPrev, hasNext bool) {
	posts := i.PublishedPosts()
	for idx, p := range posts {
		if p.SourcePath != e.SourcePath {
			continue
		}
		if idx > 0 {
			next, hasNext = posts[idx-1], true
		}
		if idx+1 < len(posts) {
			prev, hasPrev = posts[idx+1], true
		}
		break
	}
	return prev, next, hasPrev, hasNext
}

// Counts 返回文章总数与已发布数，用于健康检查与日志。
func (i *Index) Counts() (posts, published, pages int) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	for _, e := range i.Entries {
		switch e.Kind {
		case content.KindPost:
			posts++
			if e.Published() {
				published++
			}
		case content.KindPage:
			pages++
		}
	}
	return posts, published, pages
}

// GeneratedAtTime 返回索引生成时间。
func (i *Index) GeneratedAtTime() time.Time {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.GeneratedAt
}

func (i *Index) entriesByPath() map[string]Entry {
	i.mu.RLock()
	defer i.mu.RUnlock()
	out := make(map[string]Entry, len(i.Entries))
	for _, e := range i.Entries {
		out[e.SourcePath] = e
	}
	return out
}

func (i *Index) searchTextByPath() map[string]string {
	i.mu.RLock()
	defer i.mu.RUnlock()
	out := make(map[string]string, len(i.SearchText))
	for k, v := range i.SearchText {
		out[k] = v
	}
	return out
}

func (i *Index) add(e Entry, searchText string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.Entries = append(i.Entries, e)
	i.SearchText[e.SourcePath] = searchText
}

// finalize 计算标签统计与源目录指纹。
func (i *Index) finalize(contentDir string) {
	i.mu.Lock()
	tagCount := map[string]int{}
	for _, e := range i.Entries {
		if !e.Published() {
			continue
		}
		for _, t := range e.Tags {
			tagCount[t]++
		}
	}
	tags := make([]Tag, 0, len(tagCount))
	for name, count := range tagCount {
		tags = append(tags, Tag{Name: name, Count: count})
	}
	sort.Slice(tags, func(a, b int) bool {
		if tags[a].Count == tags[b].Count {
			return tags[a].Name < tags[b].Name
		}
		return tags[a].Count > tags[b].Count
	})
	i.Tags = tags
	i.GeneratedAt = time.Now()
	i.mu.Unlock()

	i.SourceDigest = Digest(contentDir)
}

// Digest 计算内容目录与渲染管线的指纹（每个文件的路径+大小+修改时间，以及渲染版本）。
// 渲染版本参与计算，保证渲染逻辑或索引结构变更后旧索引会失效。
func Digest(contentDir string) string {
	files, _, err := content.List(contentDir, content.Options{})
	if err != nil {
		return ""
	}
	h := sha256.New()
	fmt.Fprintf(h, "render=%d index=%d\n", RenderVersion, Version)
	for _, f := range files {
		fmt.Fprintf(h, "%s|%d|%d\n", f.RelPath, f.Size, f.Mtime)
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// Load 读取索引文件；文件不存在或结构版本不匹配时返回 nil。
func Load(path string) (*Index, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	idx := &Index{}
	if err := json.Unmarshal(raw, idx); err != nil {
		return nil, fmt.Errorf("索引文件解析失败（将重建）: %w", err)
	}
	if idx.Version != Version {
		return nil, nil
	}
	if idx.SearchText == nil {
		idx.SearchText = map[string]string{}
	}
	return idx, nil
}

// Save 原子写入索引文件。
func (i *Index) Save(path string) error {
	i.mu.RLock()
	snapshot := struct {
		Version      int               `json:"version"`
		GeneratedAt  time.Time         `json:"generated_at"`
		SourceDigest string            `json:"source_digest"`
		Entries      []Entry           `json:"entries"`
		Tags         []Tag             `json:"tags"`
		SearchText   map[string]string `json:"search_text"`
	}{i.Version, i.GeneratedAt, i.SourceDigest, i.Entries, i.Tags, i.SearchText}
	i.mu.RUnlock()

	raw, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// TrimBasePath 去掉 URL 前缀，用于反查 source path 等场景。
func TrimBasePath(url, basePath string) string {
	return strings.TrimPrefix(url, basePath)
}
