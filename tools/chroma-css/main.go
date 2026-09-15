// 生成 chroma 代码高亮样式表：浅色用 github，深色用 github-dark 并作用域限定在 [data-theme="dark"] 下。
// 用法：go run ./tools/chroma-css
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

const outputPath = "internal/web/static/chroma/chroma.css"

func main() {
	light := styles.Get("github")
	dark := styles.Get("github-dark")
	if light == nil || dark == nil {
		log.Fatal("chroma 样式缺失：github / github-dark")
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
	out.WriteString(lightCSS.String())
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
