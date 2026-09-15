package render

import (
	"fmt"
	"strings"
)

// MathItem 是一段待渲染的公式。
type MathItem struct {
	Tex     string
	Display bool
}

// Extracted 是公式抽取的结果：Masked 用占位符替换了公式，Items 保存公式原文。
type Extracted struct {
	Masked string
	Items  []MathItem
}

func mathPlaceholder(i int) string { return fmt.Sprintf("@@BBLOG-MATH-%d@@", i) }

// ExtractMath 抽取 $...$、$$...$$、\(...\)、\[...\] 四种写法的公式，
// 并跳过围栏代码块与行内代码，避免代码里的 $ 被误判。
func ExtractMath(src string) Extracted {
	var b strings.Builder
	b.Grow(len(src))
	var items []MathItem
	lineStart := true

	for i := 0; i < len(src); {
		if lineStart {
			if markerLen := fenceMarkerAt(src, i); markerLen > 0 {
				end := fenceBlockEnd(src, i, markerLen)
				b.WriteString(src[i:end])
				i = end
				lineStart = true
				continue
			}
		}
		c := src[i]
		switch {
		case c == '\n':
			b.WriteByte(c)
			i++
			lineStart = true
		case c == '`':
			run := runLen(src[i:], '`')
			if end := closingRun(src, i+run, '`', run); end > 0 {
				b.WriteString(src[i:end])
				i = end
			} else {
				b.WriteString(src[i : i+run])
				i += run
			}
			lineStart = false
		case c == '$' && i+1 < len(src) && src[i+1] == '$':
			if end := strings.Index(src[i+2:], "$$"); end >= 0 {
				items = append(items, MathItem{Tex: src[i+2 : i+2+end], Display: true})
				b.WriteString(mathPlaceholder(len(items) - 1))
				i = i + 2 + end + 2
			} else {
				b.WriteByte(c)
				i++
			}
			lineStart = false
		case c == '$':
			if end := inlineDollarEnd(src, i); end > 0 {
				items = append(items, MathItem{Tex: src[i+1 : end], Display: false})
				b.WriteString(mathPlaceholder(len(items) - 1))
				i = end + 1
			} else {
				b.WriteByte(c)
				i++
			}
			lineStart = false
		case c == '\\' && i+1 < len(src) && src[i+1] == '[':
			if end := strings.Index(src[i+2:], "\\]"); end >= 0 {
				items = append(items, MathItem{Tex: src[i+2 : i+2+end], Display: true})
				b.WriteString(mathPlaceholder(len(items) - 1))
				i = i + 2 + end + 2
			} else {
				b.WriteByte(c)
				i++
			}
			lineStart = false
		case c == '\\' && i+1 < len(src) && src[i+1] == '(':
			if end := strings.Index(src[i+2:], "\\)"); end >= 0 {
				items = append(items, MathItem{Tex: src[i+2 : i+2+end], Display: false})
				b.WriteString(mathPlaceholder(len(items) - 1))
				i = i + 2 + end + 2
			} else {
				b.WriteByte(c)
				i++
			}
			lineStart = false
		default:
			b.WriteByte(c)
			i++
			lineStart = false
		}
	}
	return Extracted{Masked: b.String(), Items: items}
}

// inlineDollarEnd 返回同一行内闭合 $ 的下标，不满足行内公式条件时返回 -1。
func inlineDollarEnd(src string, start int) int {
	rest := src[start+1:]
	if nl := strings.IndexByte(rest, '\n'); nl >= 0 {
		rest = rest[:nl]
	}
	end := strings.IndexByte(rest, '$')
	if end <= 0 {
		return -1
	}
	tex := rest[:end]
	if tex != strings.TrimSpace(tex) {
		return -1
	}
	// 形如 "$5 和 $10" 的货币写法不当作公式
	after := start + 1 + end + 1
	if after < len(src) && src[after] >= '0' && src[after] <= '9' {
		return -1
	}
	return start + 1 + end
}

func runLen(s string, ch byte) int {
	n := 0
	for n < len(s) && s[n] == ch {
		n++
	}
	return n
}

// closingRun 返回与起始反引号等长的闭合串之后的偏移，找不到时返回 -1。
func closingRun(src string, from int, ch byte, n int) int {
	for i := from; i < len(src); {
		if src[i] != ch {
			i++
			continue
		}
		run := runLen(src[i:], ch)
		if run == n {
			return i + run
		}
		i += run
	}
	return -1
}

// fenceMarkerAt 判断 src[i:] 是否是行首的 ``` 或 ~~~ 围栏，返回标记长度。
func fenceMarkerAt(src string, i int) int {
	if i >= len(src) {
		return 0
	}
	ch := src[i]
	if ch != '`' && ch != '~' {
		return 0
	}
	n := runLen(src[i:], ch)
	if n < 3 {
		return 0
	}
	// 同一行内不允许再出现反引号（避免把行内代码当成围栏）
	lineEnd := strings.IndexByte(src[i:], '\n')
	line := src[i:]
	if lineEnd >= 0 {
		line = src[i : i+lineEnd]
	}
	if ch == '`' && strings.Contains(line[n:], "`") {
		return 0
	}
	return n
}

// fenceBlockEnd 返回围栏代码块（含闭合行）之后的偏移。
func fenceBlockEnd(src string, start, markerLen int) int {
	marker := src[start]
	pos := start
	for {
		nl := strings.IndexByte(src[pos:], '\n')
		if nl < 0 {
			return len(src)
		}
		lineStart := pos + nl + 1
		if lineStart >= len(src) {
			return len(src)
		}
		lineEndRel := strings.IndexByte(src[lineStart:], '\n')
		if lineEndRel < 0 {
			if isClosingFence(src[lineStart:], marker, markerLen) {
				return len(src)
			}
			return len(src)
		}
		if isClosingFence(src[lineStart:lineStart+lineEndRel], marker, markerLen) {
			return lineStart + lineEndRel + 1
		}
		pos = lineStart
	}
}

func isClosingFence(line string, marker byte, minLen int) bool {
	t := strings.TrimRight(line, " \t\r")
	n := 0
	for n < len(t) && t[n] == marker {
		n++
	}
	return n >= minLen && strings.TrimSpace(t[n:]) == ""
}
