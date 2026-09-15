package web

import (
	"encoding/json"
	"io/fs"
	"path"
	"sort"
)

// assets 是服务端直出页面需要引用的前端资源路径。
type assets struct {
	Style  string
	Script string
}

// defaultAssets 在没有 Vite manifest（未构建前端）时使用。
var defaultAssets = assets{Style: "/assets/app.css", Script: "/assets/app.js"}

// manifestEntry 对应 Vite build.manifest 生成的条目结构。
type manifestEntry struct {
	File    string   `json:"file"`
	CSS     []string `json:"css"`
	IsEntry bool     `json:"isEntry"`
}

// loadAssets 读取 dist/.vite/manifest.json，取出入口的 JS 与 CSS 路径。
func loadAssets(dist fs.FS) assets {
	raw, err := fs.ReadFile(dist, ".vite/manifest.json")
	if err != nil {
		return defaultAssets
	}
	var manifest map[string]manifestEntry
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return defaultAssets
	}
	keys := make([]string, 0, len(manifest))
	for key := range manifest {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		entry := manifest[key]
		if !entry.IsEntry {
			continue
		}
		out := assets{Script: "/assets/" + path.Base(entry.File), Style: defaultAssets.Style}
		if len(entry.CSS) > 0 {
			out.Style = "/assets/" + path.Base(entry.CSS[0])
		}
		return out
	}
	return defaultAssets
}
