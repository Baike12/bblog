package content

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// IssueLevel 区分错误与警告。
type IssueLevel string

const (
	LevelError   IssueLevel = "error"
	LevelWarning IssueLevel = "warning"
)

// Issue 是一条内容校验结果。
type Issue struct {
	Level   IssueLevel
	Path    string
	Message string
}

func (i Issue) String() string {
	prefix := "ERROR"
	if i.Level == LevelWarning {
		prefix = "WARN "
	}
	if i.Path == "" {
		return fmt.Sprintf("%s %s", prefix, i.Message)
	}
	return fmt.Sprintf("%s %s: %s", prefix, i.Path, i.Message)
}

// HasError 表示校验结果中存在错误级别的问题。
func HasError(issues []Issue) bool {
	for _, i := range issues {
		if i.Level == LevelError {
			return true
		}
	}
	return false
}

// Validate 校验内容目录：解析错误、slug 冲突、图片引用缺失。
func Validate(contentDir string, posts, pages []Item, errs []FileError) []Issue {
	var issues []Issue
	for _, e := range errs {
		issues = append(issues, Issue{Level: LevelError, Path: e.Path, Message: e.Err.Error()})
	}

	issues = append(issues, checkDuplicateSlugs(posts, pages)...)

	for _, item := range append(append([]Item{}, posts...), pages...) {
		for _, ref := range mediaRefs(item) {
			if issue, ok := checkMediaRef(contentDir, item, ref); ok {
				issues = append(issues, issue)
			}
		}
	}
	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].Path == issues[j].Path {
			return issues[i].Message < issues[j].Message
		}
		return issues[i].Path < issues[j].Path
	})
	return issues
}

func checkDuplicateSlugs(posts, pages []Item) []Issue {
	var issues []Issue
	byKey := map[string][]Item{}
	for _, p := range posts {
		key := p.Date.Format("2006/01") + "/" + p.Slug
		byKey[key] = append(byKey[key], p)
	}
	keys := make([]string, 0, len(byKey))
	for k := range byKey {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		group := byKey[k]
		if len(group) < 2 {
			continue
		}
		paths := make([]string, 0, len(group))
		for _, g := range group {
			paths = append(paths, g.SourcePath)
		}
		issues = append(issues, Issue{
			Level:   LevelError,
			Path:    strings.Join(paths, ", "),
			Message: fmt.Sprintf("slug 冲突: %s 在同一月份出现 %d 次", k, len(group)),
		})
	}

	byPageSlug := map[string][]Item{}
	for _, p := range pages {
		byPageSlug[p.Slug] = append(byPageSlug[p.Slug], p)
	}
	pageKeys := make([]string, 0, len(byPageSlug))
	for k := range byPageSlug {
		pageKeys = append(pageKeys, k)
	}
	sort.Strings(pageKeys)
	for _, k := range pageKeys {
		group := byPageSlug[k]
		if len(group) < 2 {
			continue
		}
		paths := make([]string, 0, len(group))
		for _, g := range group {
			paths = append(paths, g.SourcePath)
		}
		issues = append(issues, Issue{
			Level:   LevelError,
			Path:    strings.Join(paths, ", "),
			Message: fmt.Sprintf("页面 slug 冲突: %s 出现 %d 次", k, len(group)),
		})
	}
	return issues
}

var (
	mdImageRe  = regexp.MustCompile(`!?\[[^\]]*\]\(([^)\s]+)`)
	htmlAttrRe = regexp.MustCompile(`(?i)(?:src|href)\s*=\s*["']([^"']+)["']`)
)

func mediaRefs(item Item) []string {
	var refs []string
	for _, m := range mdImageRe.FindAllStringSubmatch(item.Body, -1) {
		refs = append(refs, m[1])
	}
	for _, m := range htmlAttrRe.FindAllStringSubmatch(item.Body, -1) {
		refs = append(refs, m[1])
	}
	out := make([]string, 0, len(refs))
	for _, r := range refs {
		if r == "" || strings.HasPrefix(r, "#") || strings.HasPrefix(r, "data:") {
			continue
		}
		out = append(out, r)
	}
	return out
}

func checkMediaRef(contentDir string, item Item, ref string) (Issue, bool) {
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") || strings.HasPrefix(ref, "//") {
		return Issue{}, false
	}
	clean := ref
	if i := strings.IndexAny(clean, "?#"); i >= 0 {
		clean = clean[:i]
	}
	if decoded, err := url.PathUnescape(clean); err == nil {
		clean = decoded
	}

	var candidates []string
	if strings.HasPrefix(clean, "/") {
		idx := strings.Index(clean, "/media/")
		if idx < 0 {
			return Issue{}, false
		}
		candidates = append(candidates, filepath.Join(contentDir, "media", filepath.FromSlash(strings.TrimPrefix(clean[idx+len("/media/"):], "/"))))
	} else {
		candidates = append(candidates,
			filepath.Join(contentDir, filepath.Dir(filepath.FromSlash(item.SourcePath)), filepath.FromSlash(clean)),
			filepath.Join(contentDir, "media", filepath.FromSlash(path.Clean(clean))),
		)
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return Issue{}, false
		}
	}
	return Issue{
		Level:   LevelError,
		Path:    item.SourcePath,
		Message: fmt.Sprintf("引用的图片不存在: %s", ref),
	}, true
}
