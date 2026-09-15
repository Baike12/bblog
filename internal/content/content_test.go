package content

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseFrontmatterMarkdown(t *testing.T) {
	raw := []byte("---\ntitle: 标题\ndate: 2026-09-15 21:30:00\ntags: [go, 部署]\ndraft: true\n---\n\n正文\n")
	meta, body, err := ParseFrontmatter(raw, FormatMarkdown)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if meta.Title != "标题" {
		t.Errorf("title = %q", meta.Title)
	}
	if len(meta.Tags) != 2 || meta.Tags[1] != "部署" {
		t.Errorf("tags = %v", meta.Tags)
	}
	if !meta.Draft {
		t.Error("draft 应为 true")
	}
	if body != "\n正文\n" {
		t.Errorf("body = %q", body)
	}
}

func TestParseFrontmatterTagsAsString(t *testing.T) {
	raw := []byte("---\ntags: go, 部署 , go\n---\nbody")
	meta, _, err := ParseFrontmatter(raw, FormatMarkdown)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(meta.Tags) != 2 {
		t.Fatalf("应去重后剩 2 个标签，实际 %v", meta.Tags)
	}
}

func TestParseFrontmatterHTML(t *testing.T) {
	raw := []byte("<!--\ntitle: HTML 文章\ndate: 2026-08-20\n-->\n<p>正文</p>")
	meta, body, err := ParseFrontmatter(raw, FormatHTML)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if meta.Title != "HTML 文章" {
		t.Errorf("title = %q", meta.Title)
	}
	if body != "\n<p>正文</p>" {
		t.Errorf("body = %q", body)
	}
}

func TestParseFrontmatterUnclosedHTMLComment(t *testing.T) {
	raw := []byte("<!--\ntitle: 未闭合\n<p>正文</p>")
	if _, _, err := ParseFrontmatter(raw, FormatHTML); err == nil {
		t.Fatal("未闭合注释应返回错误")
	}
}

func TestParseFrontmatterWithoutHeader(t *testing.T) {
	raw := []byte("没有 frontmatter 的正文\n")
	meta, body, err := ParseFrontmatter(raw, FormatMarkdown)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if meta.Title != "" {
		t.Errorf("title 应为空，实际 %q", meta.Title)
	}
	if body != string(raw) {
		t.Errorf("body = %q", body)
	}
}

func TestParseDateLayouts(t *testing.T) {
	cases := map[string]string{
		"2026-09-15":          "2026-09-15",
		"2026-09-15 21:30":    "2026-09-15",
		"2026-09-15 21:30:00": "2026-09-15",
		"2026-09-15T21:30:00+08:00": "2026-09-15",
	}
	for input, want := range cases {
		got, err := ParseDate(input)
		if err != nil {
			t.Fatalf("%q 解析失败: %v", input, err)
		}
		if got.In(Location).Format("2006-01-02") != want {
			t.Errorf("%q → %v", input, got)
		}
	}
	if _, err := ParseDate("昨天"); err == nil {
		t.Error("非法日期应返回错误")
	}
}

func writeFile(t *testing.T, root, rel, body string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanClassifiesDirectories(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "posts/2026/09/2026-09-15-hello.md", "---\ntitle: 你好\ndate: 2026-09-15 10:00:00\ntags: [go]\n---\n正文")
	writeFile(t, root, "draft/2026-09-14-draft.md", "---\ntitle: 草稿\ndate: 2026-09-14\n---\n草稿正文")
	writeFile(t, root, "private/2026-09-10-note.md", "---\ntitle: 私密\ndate: 2026-09-10\n---\n私密正文")
	writeFile(t, root, "trash/2026-01-01-deleted.md", "---\ntitle: 删除\ndate: 2026-01-01\n---\n删除正文")
	writeFile(t, root, "pages/about.md", "---\ntitle: 关于\n---\n关于正文")
	writeFile(t, root, "media/pic.png", "not-a-real-png")

	posts, pages, errs, err := Scan(root, Options{BasePath: "/blog"})
	if err != nil {
		t.Fatalf("扫描失败: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("不应有错误: %v", errs)
	}
	if len(posts) != 3 {
		t.Fatalf("应有 3 篇文章（trash 被忽略），实际 %d", len(posts))
	}
	if len(pages) != 1 {
		t.Fatalf("应有 1 个页面，实际 %d", len(pages))
	}

	bySlug := map[string]Item{}
	for _, p := range posts {
		bySlug[p.Slug] = p
	}
	if !bySlug["draft"].Draft {
		t.Error("draft 目录下的文章应为草稿")
	}
	if !bySlug["note"].Private {
		t.Error("private 目录下的文章应为私密")
	}
	if bySlug["hello"].URL != "/blog/posts/2026/09/hello/" {
		t.Errorf("URL = %q", bySlug["hello"].URL)
	}
	if pages[0].URL != "/blog/pages/about/" {
		t.Errorf("页面 URL = %q", pages[0].URL)
	}
	if bySlug["hello"].Date.Format("2006-01-02") != "2026-09-15" {
		t.Errorf("date = %v", bySlug["hello"].Date)
	}
}

func TestScanFallsBackToFileNameAndMtime(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "posts/2026/09/2026-09-15-no-frontmatter.md", "只有正文")
	posts, _, _, err := Scan(root, Options{BasePath: ""})
	if err != nil {
		t.Fatalf("扫描失败: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("应有 1 篇，实际 %d", len(posts))
	}
	if posts[0].Title != "no-frontmatter" {
		t.Errorf("title 应取文件名，实际 %q", posts[0].Title)
	}
	if posts[0].Slug != "no-frontmatter" {
		t.Errorf("slug = %q", posts[0].Slug)
	}
	if posts[0].URL != "/posts/2026/09/no-frontmatter/" {
		t.Errorf("URL = %q（应使用文件 mtime 的年月）", posts[0].URL)
	}
}

func TestScanSummaryAndWordCount(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "posts/2026/09/2026-09-15-sum.md", "---\ntitle: 摘要\ndate: 2026-09-15\n---\n这是一段足够长的中文正文，用来验证摘要会自动截取，并且字数统计把中文按字计数。")
	posts, _, _, err := Scan(root, Options{})
	if err != nil {
		t.Fatalf("扫描失败: %v", err)
	}
	if posts[0].WordCount == 0 {
		t.Error("word_count 不应为 0")
	}
	if posts[0].ReadingMinutes < 1 {
		t.Error("reading_minutes 至少为 1")
	}
	if posts[0].Summary == "" {
		t.Error("summary 不应为空")
	}
}

func TestValidateDuplicateSlug(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "posts/2026/09/2026-09-15-dup.md", "---\ntitle: A\ndate: 2026-09-15\n---\nA")
	writeFile(t, root, "posts/2026/09/2026-09-20-dup.md", "---\ntitle: B\ndate: 2026-09-20\n---\nB")
	posts, pages, errs, err := Scan(root, Options{})
	if err != nil {
		t.Fatalf("扫描失败: %v", err)
	}
	issues := Validate(root, posts, pages, errs)
	if !HasError(issues) {
		t.Fatal("重复 slug 应判定为错误")
	}
}

func TestValidateMissingMedia(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "posts/2026/09/2026-09-15-img.md", "---\ntitle: 图\ndate: 2026-09-15\n---\n![x](/blog/media/nope.png)")
	writeFile(t, root, "posts/2026/09/2026-09-16-ok.md", "---\ntitle: 好\ndate: 2026-09-16\n---\n![x](/blog/media/yes.png)")
	writeFile(t, root, "media/yes.png", "png")
	posts, pages, errs, err := Scan(root, Options{BasePath: "/blog"})
	if err != nil {
		t.Fatalf("扫描失败: %v", err)
	}
	issues := Validate(root, posts, pages, errs)
	if len(issues) != 1 {
		t.Fatalf("应只有 1 条问题，实际 %d: %v", len(issues), issues)
	}
	if issues[0].Level != LevelError {
		t.Errorf("级别 = %s", issues[0].Level)
	}
}

func TestDigestChangesWithMtime(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "posts/2026/09/2026-09-15-a.md", "---\ntitle: A\ndate: 2026-09-15\n---\nA")
	files, _, err := List(root, Options{})
	if err != nil {
		t.Fatalf("列举失败: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("应有 1 个文件，实际 %d", len(files))
	}
	if files[0].Mtime == 0 || files[0].Size == 0 {
		t.Error("Mtime 与 Size 都应被填充")
	}
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(filepath.Join(root, "posts/2026/09/2026-09-15-a.md"), future, future); err != nil {
		t.Fatal(err)
	}
	after, _, err := List(root, Options{})
	if err != nil {
		t.Fatalf("列举失败: %v", err)
	}
	if after[0].Mtime == files[0].Mtime {
		t.Error("修改时间变化后 Mtime 应不同")
	}
}
