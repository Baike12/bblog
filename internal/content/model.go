package content

import "time"

// Kind 区分文章与独立页面。
type Kind string

const (
	KindPost Kind = "post"
	KindPage Kind = "page"
)

// Format 是正文的源格式。
type Format string

const (
	FormatMarkdown Format = "markdown"
	FormatHTML     Format = "html"
)

// Item 是扫描出的一个内容单元（文章或页面）。
type Item struct {
	Kind   Kind
	Format Format

	Slug  string
	Title string
	Date  time.Time
	// Updated 为零值表示未设置。
	Updated time.Time
	Tags    []string
	Summary string
	Draft   bool
	Private bool
	Lang    string

	// SourcePath 是相对内容目录的路径，例如 posts/2026/09/hello.md。
	SourcePath  string
	SourceMtime int64
	SourceSize  int64

	// Body 是去掉 frontmatter 后的原始正文。
	Body string

	// URL 形如 /blog/posts/2026/09/hello/ 或 /blog/pages/about/。
	URL string

	WordCount      int
	ReadingMinutes int
}

// Heading 是正文中的一个标题，用于目录与锚点。
type Heading struct {
	Level int    `json:"level"`
	ID    string `json:"id"`
	Text  string `json:"text"`
}

// Published 表示该内容是否应出现在列表、标签、归档、RSS、搜索中。
func (i *Item) Published() bool { return !i.Draft && !i.Private }

// RelCachePath 返回渲染缓存的相对路径，例如 2026/09/hello.html。
func (i *Item) RelCachePath() string {
	if i.Kind == KindPage {
		return "pages/" + i.Slug + ".html"
	}
	return i.Date.Format("2006/01") + "/" + i.Slug + ".html"
}
