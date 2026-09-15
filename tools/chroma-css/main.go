// 生成 chroma 代码高亮样式表：浅色用 gruvbox-light，深色用 gruvbox。
// 用法：go run ./tools/chroma-css
//
// 两套样式分别包在 :root 与 [data-theme="dark"] 里（CSS 嵌套）。
// 作用域隔离是必须的：有些 token 只在其中一套里有定义（例如 github-dark 不含
// NameOther），若浅色不加作用域，这些规则会在深色模式下继续命中，
// 使该 token 呈现浅色主题的文字色，在深色底上不可见。
package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
)

const (
	outputPath = "internal/web/static/chroma/chroma.css"
	lightStyle = "gruvbox-light"
	darkStyle  = "gruvbox"
)

func main() {
	light := styles.Get(lightStyle)
	dark := styles.Get(darkStyle)
	if light == nil || dark == nil {
		log.Fatalf("chroma 样式缺失：%s / %s", lightStyle, darkStyle)
	}

	formatter := html.New(html.WithClasses(true))

	var lightCSS, darkCSS bytes.Buffer
	if err := formatter.WriteCSS(&lightCSS, light); err != nil {
		log.Fatalf("生成浅色样式失败: %v", err)
	}
	if err := formatter.WriteCSS(&darkCSS, dark); err != nil {
		log.Fatalf("生成深色样式失败: %v", err)
	}

	var out bytes.Buffer
	out.WriteString("/* 由 tools/chroma-css 生成，请勿手工修改 */\n")
	out.WriteString("/* 浅色主题：CSS 嵌套，展开后选择器为 :root .chroma ... */\n")
	out.WriteString(":root {\n")
	out.WriteString(lightCSS.String())
	out.WriteString("}\n")
	out.WriteString("\n/* 深色主题：CSS 嵌套，展开后选择器为 [data-theme=\"dark\"] .chroma ... */\n")
	out.WriteString("[data-theme=\"dark\"] {\n")
	out.WriteString(darkCSS.String())
	out.WriteString("}\n")

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(outputPath, out.Bytes(), 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("已生成 %s（%d 字节）\n", outputPath, out.Len())
}
