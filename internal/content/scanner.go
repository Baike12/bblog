package content

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

// Options 控制扫描行为。
type Options struct {
	// BasePath 形如 /blog，会拼进 Item.URL。
	BasePath string
}

// FileMeta 是一次 stat 得到的内容文件信息，不需要读取文件内容。
type FileMeta struct {
	RelPath string
	Top     string
	Format  Format
	Mtime   int64
	Size    int64
}

// FileError 记录单个文件的错误，不影响其它文件。
type FileError struct {
	Path string
	Err  error
}

func (e FileError) Error() string { return fmt.Sprintf("%s: %v", e.Path, e.Err) }

// List 只做目录遍历与 stat，供增量构建判断哪些文件发生了变化。
// 目录语义：posts 已发布、draft 草稿、private 私密、trash 忽略、pages 独立页面、media 静态资源。
func List(root string, opts Options) (files []FileMeta, errs []FileError, err error) {
	root = filepath.Clean(root)
	info, statErr := os.Stat(root)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("读取内容目录 %s: %w", root, statErr)
	}
	if !info.IsDir() {
		return nil, nil, fmt.Errorf("内容目录不是目录: %s", root)
	}

	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			errs = append(errs, FileError{Path: relOrAbs(root, path), Err: walkErr})
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if path != root && (strings.HasPrefix(name, ".") || name == "node_modules") {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") {
			return nil
		}
		rel := relOrAbs(root, path)
		top := topSegment(rel)
		switch top {
		case "trash", "media":
			return nil
		case "posts", "draft", "private", "pages":
		default:
			return nil
		}
		format, ok := formatOf(name)
		if !ok {
			return nil
		}
		fi, infoErr := d.Info()
		if infoErr != nil {
			errs = append(errs, FileError{Path: rel, Err: infoErr})
			return nil
		}
		files = append(files, FileMeta{RelPath: rel, Top: top, Format: format, Mtime: fi.ModTime().Unix(), Size: fi.Size()})
		return nil
	})
	if walkErr != nil {
		return nil, errs, walkErr
	}
	sort.Slice(files, func(i, j int) bool { return files[i].RelPath < files[j].RelPath })
	return files, errs, nil
}

// Load 读取并解析单个内容文件。
func Load(root string, fm FileMeta, opts Options) (Item, error) {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(fm.RelPath)))
	if err != nil {
		return Item{}, err
	}
	return buildItem(fm, raw, opts)
}

// Scan 读取全部内容文件，返回文章与独立页面。
func Scan(root string, opts Options) (posts []Item, pages []Item, errs []FileError, err error) {
	files, errs, err := List(root, opts)
	if err != nil {
		return nil, nil, errs, err
	}
	for _, fm := range files {
		item, loadErr := Load(root, fm, opts)
		if loadErr != nil {
			errs = append(errs, FileError{Path: fm.RelPath, Err: loadErr})
			continue
		}
		if item.Kind == KindPage {
			pages = append(pages, item)
		} else {
			posts = append(posts, item)
		}
	}
	SortPosts(posts)
	sort.Slice(pages, func(i, j int) bool { return pages[i].Slug < pages[j].Slug })
	return posts, pages, errs, nil
}

// SortPosts 按发布时间倒序排列，时间相同时按 slug 排序。
func SortPosts(posts []Item) {
	sort.Slice(posts, func(i, j int) bool {
		if posts[i].Date.Equal(posts[j].Date) {
			return posts[i].Slug < posts[j].Slug
		}
		return posts[i].Date.After(posts[j].Date)
	})
}

func buildItem(fm FileMeta, raw []byte, opts Options) (Item, error) {
	rel := fm.RelPath
	meta, body, err := ParseFrontmatter(raw, fm.Format)
	if err != nil {
		return Item{}, err
	}
	base := strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
	_, namePart := splitDatePrefix(base)

	slug := strings.TrimSpace(meta.Slug)
	if slug == "" {
		slug = namePart
	}
	if slug == "" {
		return Item{}, fmt.Errorf("无法从文件名推导 slug: %s", filepath.Base(rel))
	}

	date, err := ParseDate(meta.Date)
	if err != nil {
		return Item{}, err
	}
	if date.IsZero() {
		date = time.Unix(fm.Mtime, 0).In(Location)
	}
	updated, err := ParseDate(meta.Updated)
	if err != nil {
		return Item{}, err
	}

	title := strings.TrimSpace(meta.Title)
	if title == "" {
		title = namePart
	}

	item := Item{
		Kind:        KindPost,
		Format:      fm.Format,
		Slug:        slug,
		Title:       title,
		Date:        date,
		Updated:     updated,
		Tags:        meta.Tags,
		Summary:     strings.TrimSpace(meta.Summary),
		Draft:       meta.Draft || fm.Top == "draft",
		Private:     meta.Private || fm.Top == "private",
		Lang:        meta.Lang,
		SourcePath:  rel,
		SourceMtime: fm.Mtime,
		SourceSize:  fm.Size,
		Body:        body,
	}
	if item.Lang == "" {
		item.Lang = "zh"
	}
	if fm.Top == "pages" {
		item.Kind = KindPage
		item.URL = joinPath(opts.BasePath, "pages", slug, "")
	} else {
		item.URL = joinPath(opts.BasePath, "posts", date.Format("2006"), date.Format("01"), slug, "")
	}
	plain := PlainText(body, fm.Format)
	if item.Summary == "" {
		item.Summary = truncateRunes(plain, 120)
	}
	item.WordCount, item.ReadingMinutes = countWords(plain)
	return item, nil
}

func formatOf(name string) (Format, bool) {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".md", ".markdown":
		return FormatMarkdown, true
	case ".html", ".htm":
		return FormatHTML, true
	}
	return "", false
}

func topSegment(rel string) string {
	rel = filepath.ToSlash(rel)
	if i := strings.IndexByte(rel, '/'); i >= 0 {
		return rel[:i]
	}
	return ""
}

// splitDatePrefix 拆分文件名中的 YYYY-MM-DD- 前缀。
func splitDatePrefix(base string) (datePart, name string) {
	if len(base) < 11 {
		return "", base
	}
	candidate := base[:10]
	if _, err := time.ParseInLocation("2006-01-02", candidate, Location); err != nil {
		return "", base
	}
	sep := base[10]
	if sep != '-' && sep != '_' {
		return "", base
	}
	return candidate, base[11:]
}

func joinPath(base string, parts ...string) string {
	segments := make([]string, 0, len(parts)+1)
	if trimmed := strings.Trim(base, "/"); trimmed != "" {
		segments = append(segments, trimmed)
	}
	for _, p := range parts {
		if trimmed := strings.Trim(p, "/"); trimmed != "" {
			segments = append(segments, trimmed)
		}
	}
	return "/" + strings.Join(segments, "/") + "/"
}

func relOrAbs(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

var (
	fenceRe    = regexp.MustCompile("(?s)```.*?```")
	tildesRe   = regexp.MustCompile("(?s)~~~.*?~~~")
	htmlTagRe  = regexp.MustCompile(`(?s)<[^>]*>`)
	mdLinkRe   = regexp.MustCompile(`!?\[([^\]]*)\]\([^)]*\)`)
	mdMarkRe   = regexp.MustCompile("[#>*`_~\\[\\]|]+")
	spaceRe    = regexp.MustCompile(`\s+`)
	htmlScript = regexp.MustCompile(`(?s)<(script|style)[^>]*>.*?</(script|style)>`)
)

// PlainText 粗略地把正文转成纯文本，用于摘要与搜索索引。
func PlainText(body string, format Format) string {
	s := body
	if format == FormatHTML {
		s = htmlScript.ReplaceAllString(s, " ")
		s = htmlTagRe.ReplaceAllString(s, " ")
	} else {
		s = fenceRe.ReplaceAllString(s, " ")
		s = tildesRe.ReplaceAllString(s, " ")
		s = htmlScript.ReplaceAllString(s, " ")
		s = htmlTagRe.ReplaceAllString(s, " ")
		s = mdLinkRe.ReplaceAllString(s, "$1")
		s = mdMarkRe.ReplaceAllString(s, " ")
	}
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	return strings.TrimSpace(spaceRe.ReplaceAllString(s, " "))
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// countWords 把 CJK 字符按字计数、拉丁文本按词计数，用于估算阅读时长。
func countWords(s string) (words, minutes int) {
	inWord := false
	for _, r := range s {
		switch {
		case isCJK(r):
			words++
			inWord = false
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if !inWord {
				words++
				inWord = true
			}
		default:
			inWord = false
		}
	}
	if words == 0 {
		return 0, 0
	}
	minutes = (words + 299) / 300
	if minutes < 1 {
		minutes = 1
	}
	return words, minutes
}

func isCJK(r rune) bool {
	return (r >= 0x2E80 && r <= 0x9FFF) ||
		(r >= 0xF900 && r <= 0xFAFF) ||
		(r >= 0xFF00 && r <= 0xFFEF)
}
