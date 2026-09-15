package index

import (
	"sort"
	"strings"
)

// Hit 是一条搜索结果。
type Hit struct {
	Entry
	Score   int    `json:"score"`
	Excerpt string `json:"excerpt"`
}

const (
	weightTitle   = 3
	weightTags    = 2
	weightSummary = 2
	weightBody    = 1

	excerptRadius = 40
)

// Search 在已发布文章内做大小写不敏感的子串匹配。
// 中文不做分词，因此"镜像 部署"这类跨词边界查询不会命中。
func (i *Index) Search(q string, limit int) []Hit {
	q = strings.TrimSpace(q)
	if q == "" || limit <= 0 {
		return []Hit{}
	}
	needle := strings.ToLower(q)
	texts := i.searchTextByPath()

	var hits []Hit
	for _, e := range i.PublishedPosts() {
		score := strings.Count(strings.ToLower(e.Title), needle) * weightTitle
		for _, t := range e.Tags {
			score += strings.Count(strings.ToLower(t), needle) * weightTags
		}
		score += strings.Count(strings.ToLower(e.Summary), needle) * weightSummary
		body := texts[e.SourcePath]
		score += strings.Count(strings.ToLower(body), needle) * weightBody
		if score == 0 {
			continue
		}
		hits = append(hits, Hit{Entry: e, Score: score, Excerpt: excerpt(body, e.Summary, needle)})
	}

	sort.Slice(hits, func(a, b int) bool {
		if hits[a].Score == hits[b].Score {
			return hits[a].Date.After(hits[b].Date)
		}
		return hits[a].Score > hits[b].Score
	})
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits
}

func excerpt(body, summary, needle string) string {
	source := body
	if source == "" {
		source = summary
	}
	idx := strings.Index(strings.ToLower(source), needle)
	if idx < 0 {
		return truncate(source, excerptRadius*2)
	}
	runes := []rune(source)
	// 把字节下标换算成字符下标，避免中文被截断
	runeIdx := len([]rune(source[:idx]))
	start := runeIdx - excerptRadius
	if start < 0 {
		start = 0
	}
	end := runeIdx + len([]rune(needle)) + excerptRadius
	if end > len(runes) {
		end = len(runes)
	}
	out := strings.TrimSpace(string(runes[start:end]))
	if start > 0 {
		out = "…" + out
	}
	if end < len(runes) {
		out = out + "…"
	}
	return out
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
