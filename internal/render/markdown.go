package render

import (
	"bytes"
	"fmt"
	"html"
	"log"
	"strings"
	"unicode"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"

	"github.com/Baike12/bblog/internal/content"
)

// Renderer 把正文渲染成 HTML。
type Renderer struct {
	md   goldmark.Markdown
	math *MathRenderer
}

// Result 是一次渲染的产物。
type Result struct {
	HTML     string
	Headings []content.Heading
	HasMath  bool
}

// NewRenderer 构造渲染器；math 为 nil 时公式降级为纯文本。
func NewRenderer(math *MathRenderer) *Renderer {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Footnote,
			// 用 class 模式而非行内样式，深浅色主题由 chroma.css 切换
			highlighting.NewHighlighting(
				highlighting.WithStyle("github"),
				highlighting.WithFormatOptions(chromahtml.WithClasses(true)),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		// 内容写入通道只有 SSH/rsync，作者完全可信，因此原样输出原始 HTML
		goldmark.WithRendererOptions(gmhtml.WithUnsafe()),
	)
	return &Renderer{md: md, math: math}
}

// Render 渲染正文。format 为 html 时原文直接输出，只做公式回填。
func (r *Renderer) Render(body string, format content.Format) (Result, error) {
	extracted := ExtractMath(body)
	result := Result{HasMath: len(extracted.Items) > 0}

	if format == content.FormatHTML {
		result.HTML = r.fillMath(extracted, extracted.Masked)
		return result, nil
	}

	src := []byte(extracted.Masked)
	// 每次解析都用新的 IDs 实例，避免多次渲染之间互相污染标题锚点
	ctx := parser.NewContext(parser.WithIDs(newCJKIDs()))
	doc := r.md.Parser().Parse(text.NewReader(src), parser.WithContext(ctx))

	var buf bytes.Buffer
	if err := r.md.Renderer().Render(&buf, src, doc); err != nil {
		return Result{}, fmt.Errorf("渲染正文失败: %w", err)
	}
	result.HTML = r.fillMath(extracted, buf.String())
	result.Headings = collectHeadings(doc, src)
	return result, nil
}

func (r *Renderer) fillMath(ex Extracted, s string) string {
	for i, item := range ex.Items {
		rendered := mathFallbackHTML(item.Tex)
		if r.math != nil {
			out, err := r.math.Render(item.Tex, item.Display)
			if err != nil {
				log.Printf("warn: 公式渲染失败，降级为纯文本: %v", err)
			} else {
				rendered = out
			}
		}
		s = strings.ReplaceAll(s, mathPlaceholder(i), rendered)
	}
	return s
}

func mathFallbackHTML(tex string) string {
	return `<code class="math-fallback">` + html.EscapeString(tex) + `</code>`
}

func collectHeadings(doc ast.Node, src []byte) []content.Heading {
	var out []content.Heading
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}
		id := ""
		if v, ok := h.AttributeString("id"); ok {
			switch t := v.(type) {
			case []byte:
				id = string(t)
			case string:
				id = t
			}
		}
		out = append(out, content.Heading{Level: h.Level, ID: id, Text: nodeText(h, src)})
		return ast.WalkContinue, nil
	})
	return out
}

func nodeText(n ast.Node, src []byte) string {
	var b strings.Builder
	_ = ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch t := c.(type) {
		case *ast.Text:
			b.Write(t.Segment.Value(src))
			if t.SoftLineBreak() {
				b.WriteByte(' ')
			}
		case *ast.String:
			b.Write(t.Value)
		}
		return ast.WalkContinue, nil
	})
	return strings.TrimSpace(b.String())
}

// cjkIDs 是保留中日韩字符的标题锚点生成器。
// goldmark 默认实现只保留 ASCII 字母数字，中文标题会全部退化成 id="heading"。
type cjkIDs struct {
	used map[string]bool
}

func newCJKIDs() *cjkIDs { return &cjkIDs{used: map[string]bool{}} }

func (s *cjkIDs) Generate(value []byte, kind ast.NodeKind) []byte {
	var b strings.Builder
	for _, r := range string(value) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
		case unicode.IsSpace(r) || r == '-' || r == '_':
			b.WriteByte('-')
		}
	}
	id := collapseDashes(strings.Trim(b.String(), "-"))
	if id == "" {
		if kind == ast.KindHeading {
			id = "heading"
		} else {
			id = "id"
		}
	}
	if !s.used[id] {
		s.used[id] = true
		return []byte(id)
	}
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s-%d", id, i)
		if !s.used[candidate] {
			s.used[candidate] = true
			return []byte(candidate)
		}
	}
}

func (s *cjkIDs) Put(value []byte) { s.used[string(value)] = true }

func collapseDashes(s string) string {
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return s
}
