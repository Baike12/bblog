package index

import (
	"fmt"
	"log"

	"github.com/Baike12/bblog/internal/content"
	"github.com/Baike12/bblog/internal/render"
)

// Builder 负责从内容目录构建索引。
// 未变化的源文件会直接复用上一次的索引条目与渲染缓存，因此重建成本与变更量成正比。
type Builder struct {
	ContentDir string
	IndexPath  string
	BasePath   string
	Renderer   *render.Renderer
	Cache      *render.Cache
}

// Build 构建索引。prev 可为 nil（首次构建）；解析失败的文件会被跳过并记录。
func (b *Builder) Build(prev *Index) (*Index, []content.FileError, error) {
	opts := content.Options{BasePath: b.BasePath}
	files, errs, err := content.List(b.ContentDir, opts)
	if err != nil {
		return nil, errs, err
	}

	idx := newIndex()
	var prevByPath map[string]Entry
	var prevText map[string]string
	if prev != nil {
		prevByPath = prev.entriesByPath()
		prevText = prev.searchTextByPath()
	}
	keep := make(map[string]bool, len(files))

	for _, fm := range files {
		if cached, ok := prevByPath[fm.RelPath]; ok &&
			cached.SourceMtime == fm.Mtime &&
			cached.SourceSize == fm.Size &&
			cached.RenderVersion == RenderVersion {
			if _, ok := b.Cache.Read(cached.HTMLCache); ok {
				idx.add(cached, prevText[fm.RelPath])
				keep[cached.HTMLCache] = true
				continue
			}
		}

		item, loadErr := content.Load(b.ContentDir, fm, opts)
		if loadErr != nil {
			errs = append(errs, content.FileError{Path: fm.RelPath, Err: loadErr})
			continue
		}
		result, renderErr := b.Renderer.Render(item.Body, item.Format)
		if renderErr != nil {
			errs = append(errs, content.FileError{Path: fm.RelPath, Err: renderErr})
			continue
		}
		cacheRel := item.RelCachePath()
		if writeErr := b.Cache.Write(cacheRel, result.HTML); writeErr != nil {
			errs = append(errs, content.FileError{Path: fm.RelPath, Err: fmt.Errorf("写渲染缓存失败: %w", writeErr)})
		}
		idx.add(entryFromItem(item, result, cacheRel), content.PlainText(item.Body, item.Format))
		keep[cacheRel] = true
	}

	idx.finalize(b.ContentDir)

	if pruneErr := b.Cache.Prune(keep); pruneErr != nil {
		log.Printf("warn: 清理渲染缓存失败: %v", pruneErr)
	}
	if b.IndexPath != "" {
		if saveErr := idx.Save(b.IndexPath); saveErr != nil {
			// 索引落盘失败不影响服务，内存索引仍然可用
			log.Printf("warn: 索引落盘失败（内存索引继续服务）: %v", saveErr)
		}
	}
	return idx, errs, nil
}

func entryFromItem(item content.Item, result render.Result, cacheRel string) Entry {
	return Entry{
		Kind:           item.Kind,
		Format:         item.Format,
		Slug:           item.Slug,
		URL:            item.URL,
		Title:          item.Title,
		Date:           item.Date,
		Updated:        item.Updated,
		Tags:           item.Tags,
		Summary:        item.Summary,
		Draft:          item.Draft,
		Private:        item.Private,
		SourcePath:     item.SourcePath,
		SourceMtime:    item.SourceMtime,
		SourceSize:     item.SourceSize,
		WordCount:      item.WordCount,
		ReadingMinutes: item.ReadingMinutes,
		HTMLCache:      cacheRel,
		HasMath:        result.HasMath,
		RenderVersion:  RenderVersion,
		Headings:       result.Headings,
	}
}
