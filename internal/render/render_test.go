package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Baike12/bblog/internal/content"
)

func loadKatexBundle(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "web", "static", "katex", "katex.min.js")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 KaTeX 包失败: %v", err)
	}
	return raw
}

func TestExtractMathSkipsCode(t *testing.T) {
	src := "行内 $a_1$ 结束\n\n```sh\necho $HOME\n```\n\n行内代码 `$x$` 不算\n\n$$\n\\int_0^1 x\\,dx\n$$\n"
	ex := ExtractMath(src)
	if len(ex.Items) != 2 {
		t.Fatalf("应抽取 2 个公式，实际 %d: %+v", len(ex.Items), ex.Items)
	}
	if ex.Items[0].Tex != "a_1" || ex.Items[0].Display {
		t.Errorf("第一个公式 = %+v", ex.Items[0])
	}
	if !ex.Items[1].Display || !strings.Contains(ex.Items[1].Tex, "\\int_0^1") {
		t.Errorf("第二个公式 = %+v", ex.Items[1])
	}
	if !strings.Contains(ex.Masked, "$HOME") {
		t.Error("代码块内容应保持原样")
	}
	if !strings.Contains(ex.Masked, "`$x$`") {
		t.Error("行内代码应保持原样")
	}
}

func TestExtractMathBackslashDelimiters(t *testing.T) {
	src := `行内 \(a+b\) 与块级 \[c+d\] 都要抽取`
	ex := ExtractMath(src)
	if len(ex.Items) != 2 {
		t.Fatalf("应抽取 2 个公式，实际 %d", len(ex.Items))
	}
	if ex.Items[0].Tex != "a+b" || ex.Items[0].Display {
		t.Errorf("行内公式 = %+v", ex.Items[0])
	}
	if ex.Items[1].Tex != "c+d" || !ex.Items[1].Display {
		t.Errorf("块级公式 = %+v", ex.Items[1])
	}
}

func TestExtractMathIgnoresCurrency(t *testing.T) {
	ex := ExtractMath("价格是 $5 到 $10 之间")
	if len(ex.Items) != 0 {
		t.Fatalf("货币写法不应被当作公式，实际 %+v", ex.Items)
	}
}

func TestRenderMarkdownFeatures(t *testing.T) {
	mathRenderer, err := NewMathRenderer(loadKatexBundle(t))
	if err != nil {
		t.Fatalf("初始化 KaTeX 失败: %v", err)
	}
	r := NewRenderer(mathRenderer)
	body := "# 标题\n\n| a | b |\n|---|---|\n| 1 | 2 |\n\n```go\nfunc main() {}\n```\n\n<iframe src=\"https://example.com/embed\"></iframe>\n\n<div class=\"custom-widget\">原始 HTML</div>\n\n行内 $a_1$ 公式\n"
	result, err := r.Render(body, content.FormatMarkdown)
	if err != nil {
		t.Fatalf("渲染失败: %v", err)
	}
	checks := map[string]string{
		"中文标题锚点":  `id="标题"`,
		"表格":      "<table>",
		"chroma 容器": `class="chroma"`,
		"chroma 关键字": `<span class="kd">func</span>`,
		"原始 iframe": "<iframe",
		"原始 div":   `<div class="custom-widget">`,
		"KaTeX 输出": `class="katex"`,
	}
	for name, want := range checks {
		if !strings.Contains(result.HTML, want) {
			t.Errorf("%s 缺失，期望包含 %q", name, want)
		}
	}
	if !result.HasMath {
		t.Error("HasMath 应为 true")
	}
	if len(result.Headings) == 0 || result.Headings[0].ID != "标题" {
		t.Errorf("headings = %+v", result.Headings)
	}
}

func TestRenderDuplicateHeadingIDs(t *testing.T) {
	mathRenderer, err := NewMathRenderer(loadKatexBundle(t))
	if err != nil {
		t.Fatalf("初始化 KaTeX 失败: %v", err)
	}
	r := NewRenderer(mathRenderer)
	result, err := r.Render("## 重复\n\n## 重复\n\n## 重复\n", content.FormatMarkdown)
	if err != nil {
		t.Fatalf("渲染失败: %v", err)
	}
	for _, want := range []string{`id="重复"`, `id="重复-1"`, `id="重复-2"`} {
		if !strings.Contains(result.HTML, want) {
			t.Errorf("缺少锚点 %s", want)
		}
	}
}

func TestRenderHTMLFormatKeepsRawBody(t *testing.T) {
	r := NewRenderer(nil)
	result, err := r.Render("<p>原文</p>\n\n<div>块</div>", content.FormatHTML)
	if err != nil {
		t.Fatalf("渲染失败: %v", err)
	}
	if !strings.Contains(result.HTML, "<p>原文</p>") || !strings.Contains(result.HTML, "<div>块</div>") {
		t.Errorf("HTML 正文应原样输出，实际 %q", result.HTML)
	}
}

func TestRenderMathFallbackOnInvalidFormula(t *testing.T) {
	mathRenderer, err := NewMathRenderer(loadKatexBundle(t))
	if err != nil {
		t.Fatalf("初始化 KaTeX 失败: %v", err)
	}
	r := NewRenderer(mathRenderer)
	result, err := r.Render("公式 $\\frac{$ 有语法错误\n", content.FormatMarkdown)
	if err != nil {
		t.Fatalf("渲染失败: %v", err)
	}
	if !strings.Contains(result.HTML, "math-fallback") {
		t.Errorf("非法公式应降级输出，实际 %q", result.HTML)
	}
}

// TestMathTimeoutInterrupt 用死循环的假 KaTeX 验证超时中断与中断后恢复。
func TestMathTimeoutInterrupt(t *testing.T) {
	fake := []byte(`var katex = {
	  version: "fake",
	  renderToString: function (tex) {
	    if (tex === "boom") { while (true) {} }
	    return '<span class="katex">' + tex + '</span>';
	  }
	};`)
	renderer, err := NewMathRenderer(fake)
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}
	if _, err := renderer.Render("boom", false); err == nil {
		t.Fatal("死循环公式应被超时中断并返回错误")
	}
	out, err := renderer.Render("ok", false)
	if err != nil {
		t.Fatalf("中断后应能继续渲染: %v", err)
	}
	if !strings.Contains(out, "katex") {
		t.Errorf("恢复渲染输出异常: %q", out)
	}
}

func TestCacheRoundTripAndPrune(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(dir)
	if err := c.Write("2026/09/a.html", "<p>A</p>"); err != nil {
		t.Fatalf("写入失败: %v", err)
	}
	if err := c.Write("2026/09/b.html", "<p>B</p>"); err != nil {
		t.Fatalf("写入失败: %v", err)
	}
	got, ok := c.Read("2026/09/a.html")
	if !ok || got != "<p>A</p>" {
		t.Fatalf("读取失败: %q %v", got, ok)
	}
	if err := c.Prune(map[string]bool{"2026/09/a.html": true}); err != nil {
		t.Fatalf("清理失败: %v", err)
	}
	if _, ok := c.Read("2026/09/b.html"); ok {
		t.Error("未被保留的缓存应被删除")
	}
	if _, ok := c.Read("2026/09/a.html"); !ok {
		t.Error("被保留的缓存不应被删除")
	}
}
