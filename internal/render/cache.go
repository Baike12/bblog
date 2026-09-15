package render

import (
	"io/fs"
	"os"
	"path/filepath"
)

// Cache 把渲染好的正文 HTML 落到磁盘，未变化的源文件在重建时直接复用。
type Cache struct {
	dir string
}

func NewCache(dir string) *Cache { return &Cache{dir: dir} }

func (c *Cache) Path(rel string) string {
	return filepath.Join(c.dir, filepath.FromSlash(rel))
}

func (c *Cache) Read(rel string) (string, bool) {
	b, err := os.ReadFile(c.Path(rel))
	if err != nil {
		return "", false
	}
	return string(b), true
}

// Write 原子写入：先写临时文件再 rename，避免读到半个文件。
func (c *Cache) Write(rel, body string) error {
	p := c.Path(rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, []byte(body), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// Prune 删除不在 keep 集合中的缓存文件。
func (c *Cache) Prune(keep map[string]bool) error {
	root := c.dir
	if _, err := os.Stat(root); err != nil {
		return nil
	}
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		if !keep[filepath.ToSlash(rel)] {
			os.Remove(path)
		}
		return nil
	})
}
