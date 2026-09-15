package index

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Baike12/bblog/internal/content"
	"github.com/Baike12/bblog/internal/render"
)

func newTestBuilder(t *testing.T, root string) *Builder {
	t.Helper()
	mathRenderer, err := render.NewMathRenderer([]byte(`var katex = { version: "fake", renderToString: function (tex) { return '<span class="katex">' + tex + '</span>'; } };`))
	if err != nil {
		t.Fatalf("初始化 KaTeX 失败: %v", err)
	}
	return &Builder{
		ContentDir: root,
		IndexPath:  filepath.Join(root, "..", "data", "index.json"),
		BasePath:   "/blog",
		Renderer:   render.NewRenderer(mathRenderer),
		Cache:      render.NewCache(filepath.Join(root, "..", "data", "cache")),
	}
}

func write(t *testing.T, root, rel, body string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestBuildFiltersDraftAndPrivate(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "content")
	write(t, root, "posts/2026/09/2026-09-15-pub.md", "---\ntitle: 已发布\ndate: 2026-09-15\ntags: [go]\n---\n正文")
	write(t, root, "draft/2026-09-14-d.md", "---\ntitle: 草稿\ndate: 2026-09-14\n---\n草稿")
	write(t, root, "private/2026-09-13-p.md", "---\ntitle: 私密\ndate: 2026-09-13\n---\n私密")
	write(t, root, "trash/2026-01-01-x.md", "---\ntitle: 忽略\ndate: 2026-01-01\n---\n忽略")

	b := newTestBuilder(t, root)
	idx, errs, err := b.Build(nil)
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("不应有错误: %v", errs)
	}
	if got := len(idx.PublishedPosts()); got != 1 {
		t.Fatalf("已发布文章数 = %d，期望 1", got)
	}
	if got := len(idx.Posts()); got != 3 {
		t.Fatalf("全部文章数 = %d，期望 3（含草稿与私密）", got)
	}
	if _, ok := idx.FindPost("2026", "09", "d"); ok {
		t.Error("草稿不应能被直链命中")
	}
	if _, ok := idx.FindPost("2026", "09", "p"); !ok {
		t.Error("私密文章应能被直链命中")
	}
	if len(idx.Tags) != 1 || idx.Tags[0].Name != "go" {
		t.Errorf("标签统计 = %+v", idx.Tags)
	}
}

func TestBuildReusesCacheForUnchangedFiles(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "content")
	write(t, root, "posts/2026/09/2026-09-15-a.md", "---\ntitle: A\ndate: 2026-09-15\n---\n正文 A")

	b := newTestBuilder(t, root)
	idx, _, err := b.Build(nil)
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}
	cacheRel := idx.Posts()[0].HTMLCache

	// 写入哨兵值：若未变化文件被复用，哨兵不会被覆盖
	sentinel := "<!--SENTINEL-->"
	if err := b.Cache.Write(cacheRel, sentinel); err != nil {
		t.Fatal(err)
	}
	if _, _, err := b.Build(idx); err != nil {
		t.Fatalf("重建失败: %v", err)
	}
	got, ok := b.Cache.Read(cacheRel)
	if !ok || got != sentinel {
		t.Errorf("未变化的文件应复用缓存，实际缓存内容 = %q", got)
	}

	// 修改源文件后应重新渲染，哨兵被覆盖
	write(t, root, "posts/2026/09/2026-09-15-a.md", "---\ntitle: A2\ndate: 2026-09-15\n---\n正文 A 已修改")
	idx2, _, err := b.Build(idx)
	if err != nil {
		t.Fatalf("重建失败: %v", err)
	}
	got2, _ := b.Cache.Read(cacheRel)
	if strings.Contains(got2, "SENTINEL") {
		t.Error("源文件变化后应重新渲染并覆盖缓存")
	}
	if idx2.Posts()[0].Title != "A2" {
		t.Errorf("标题未更新: %q", idx2.Posts()[0].Title)
	}
}

func TestBuildRemovesDeletedEntries(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "content")
	write(t, root, "posts/2026/09/2026-09-15-a.md", "---\ntitle: A\ndate: 2026-09-15\n---\nA")
	write(t, root, "posts/2026/09/2026-09-16-b.md", "---\ntitle: B\ndate: 2026-09-16\n---\nB")

	b := newTestBuilder(t, root)
	idx, _, err := b.Build(nil)
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}
	if len(idx.PublishedPosts()) != 2 {
		t.Fatalf("应有 2 篇，实际 %d", len(idx.PublishedPosts()))
	}
	if err := os.Remove(filepath.Join(root, "posts/2026/09/2026-09-15-a.md")); err != nil {
		t.Fatal(err)
	}
	idx2, _, err := b.Build(idx)
	if err != nil {
		t.Fatalf("重建失败: %v", err)
	}
	if len(idx2.PublishedPosts()) != 1 {
		t.Fatalf("删除后应剩 1 篇，实际 %d", len(idx2.PublishedPosts()))
	}
	if _, ok := idx2.FindPost("2026", "09", "a"); ok {
		t.Error("被删除的文章不应仍在索引里")
	}
	if got := idx2.Search("A", 10); len(got) != 0 {
		t.Error("被删除的文章不应出现在搜索结果里")
	}
}

func TestSearchScoringAndOrder(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "content")
	write(t, root, "posts/2026/09/2026-09-15-t.md", "---\ntitle: 部署镜像\ndate: 2026-09-15\n---\n正文不含关键词")
	write(t, root, "posts/2026/09/2026-09-14-b.md", "---\ntitle: 别的标题\ndate: 2026-09-14\nsummary: 这里没有关键词\n---\n正文里提到镜像一次")
	write(t, root, "posts/2026/09/2026-09-13-n.md", "---\ntitle: 无关\ndate: 2026-09-13\n---\n完全无关")

	b := newTestBuilder(t, root)
	idx, _, err := b.Build(nil)
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}
	hits := idx.Search("镜像", 10)
	if len(hits) != 2 {
		t.Fatalf("应命中 2 篇，实际 %d", len(hits))
	}
	if hits[0].Title != "部署镜像" {
		t.Errorf("标题命中应排在前面，实际 %q", hits[0].Title)
	}
	if hits[0].Score <= hits[1].Score {
		t.Errorf("分数应递减: %d vs %d", hits[0].Score, hits[1].Score)
	}
	if hits[0].Excerpt == "" {
		t.Error("命中项应带摘要片段")
	}
	if got := idx.Search("", 10); len(got) != 0 {
		t.Error("空关键词应返回空结果")
	}
	if got := idx.Search("完全不存在", 10); len(got) != 0 {
		t.Error("未命中应返回空结果")
	}
}

func TestSaveAndLoad(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "content")
	write(t, root, "posts/2026/09/2026-09-15-a.md", "---\ntitle: A\ndate: 2026-09-15\n---\n正文")

	b := newTestBuilder(t, root)
	idx, _, err := b.Build(nil)
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}
	loaded, err := Load(b.IndexPath)
	if err != nil {
		t.Fatalf("读取失败: %v", err)
	}
	if loaded == nil {
		t.Fatal("索引应能从磁盘读回")
	}
	if loaded.SourceDigest != idx.SourceDigest {
		t.Errorf("指纹不一致: %q vs %q", loaded.SourceDigest, idx.SourceDigest)
	}
	if Digest(root) != idx.SourceDigest {
		t.Error("Digest 应可复现")
	}
	if len(loaded.PublishedPosts()) != 1 {
		t.Errorf("读回的已发布文章数 = %d", len(loaded.PublishedPosts()))
	}
}

func TestNeighborsSkipUnpublished(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "content")
	write(t, root, "posts/2026/09/2026-09-15-new.md", "---\ntitle: 最新\ndate: 2026-09-15\n---\nA")
	write(t, root, "draft/2026-09-14-mid.md", "---\ntitle: 草稿\ndate: 2026-09-14\n---\nB")
	write(t, root, "posts/2026/09/2026-09-13-old.md", "---\ntitle: 更早\ndate: 2026-09-13\n---\nC")

	b := newTestBuilder(t, root)
	idx, _, err := b.Build(nil)
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}
	newest, _ := idx.FindPost("2026", "09", "new")
	prev, next, hasPrev, hasNext := idx.Neighbors(newest)
	if hasNext {
		t.Errorf("最新一篇不应有下一篇，实际 %+v", next)
	}
	if !hasPrev || prev.Slug != "old" {
		t.Errorf("上一篇应跳过草稿直接是 old，实际 %+v", prev)
	}
}

func TestEntryRoundTripKeepsKind(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "content")
	write(t, root, "posts/2026/09/2026-09-15-a.md", "---\ntitle: A\ndate: 2026-09-15\n---\nA")
	write(t, root, "pages/about.md", "---\ntitle: 关于\n---\n关于")

	b := newTestBuilder(t, root)
	idx, _, err := b.Build(nil)
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}
	if len(idx.Pages()) != 1 || idx.Pages()[0].Kind != content.KindPage {
		t.Fatalf("页面条目异常: %+v", idx.Pages())
	}
	if idx.Pages()[0].URL != "/blog/pages/about/" {
		t.Errorf("页面 URL = %q", idx.Pages()[0].URL)
	}
}
