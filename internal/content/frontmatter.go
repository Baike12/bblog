package content

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Location 是解析与展示日期使用的时区。
var Location = loadLocation()

func loadLocation() *time.Location {
	if loc, err := time.LoadLocation("Asia/Shanghai"); err == nil {
		return loc
	}
	return time.FixedZone("CST", 8*3600)
}

// Tags 同时接受 YAML 列表与逗号分隔字符串两种写法。
type Tags []string

func (t *Tags) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.SequenceNode:
		var out []string
		if err := node.Decode(&out); err != nil {
			return err
		}
		*t = cleanTags(out)
	case yaml.ScalarNode:
		var s string
		if err := node.Decode(&s); err != nil {
			return err
		}
		*t = cleanTags(strings.Split(s, ","))
	default:
		return fmt.Errorf("tags 字段类型不支持: %s", node.Tag)
	}
	return nil
}

func cleanTags(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]bool{}
	for _, t := range in {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

// Meta 是 frontmatter 的字段集合，所有字段都可缺省。
type Meta struct {
	Title   string `yaml:"title"`
	Date    string `yaml:"date"`
	Updated string `yaml:"updated"`
	Tags    Tags   `yaml:"tags"`
	Summary string `yaml:"summary"`
	Slug    string `yaml:"slug"`
	Draft   bool   `yaml:"draft"`
	Private bool   `yaml:"private"`
	Lang    string `yaml:"lang"`
}

// ParseFrontmatter 拆分 frontmatter 与正文。
// Markdown 使用首行 --- 包裹的 YAML；HTML 使用首个 <!-- --> 注释包裹的 YAML。
func ParseFrontmatter(raw []byte, format Format) (Meta, string, error) {
	raw = bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf"))
	var fm, body []byte
	var err error
	switch format {
	case FormatHTML:
		fm, body, err = splitHTMLComment(raw)
	default:
		fm, body = splitDashes(raw)
	}
	if err != nil {
		return Meta{}, "", err
	}
	meta := Meta{}
	if len(bytes.TrimSpace(fm)) > 0 {
		if err := yaml.Unmarshal(fm, &meta); err != nil {
			return Meta{}, "", fmt.Errorf("frontmatter 解析失败: %w", err)
		}
	}
	return meta, string(body), nil
}

// splitDashes 识别以 --- 单独成行包裹的 YAML 头部。
func splitDashes(raw []byte) (fm, body []byte) {
	first := bytes.IndexByte(raw, '\n')
	if first < 0 || !isFenceLine(raw[:first], "---") {
		return nil, raw
	}
	fmStart := first + 1
	for pos := fmStart; pos <= len(raw); {
		rel := bytes.IndexByte(raw[pos:], '\n')
		if rel < 0 {
			// 最后一行没有换行符，说明缺少结束的 ---
			return nil, raw
		}
		lineEnd := pos + rel
		if isFenceLine(raw[pos:lineEnd], "---") {
			return raw[fmStart:pos], raw[lineEnd+1:]
		}
		pos = lineEnd + 1
	}
	return nil, raw
}

// splitHTMLComment 识别 <!-- ... --> 包裹的 YAML 头部。
func splitHTMLComment(raw []byte) (fm, body []byte, err error) {
	trimmed := bytes.TrimLeft(raw, " \t\r\n")
	if !bytes.HasPrefix(trimmed, []byte("<!--")) {
		return nil, raw, nil
	}
	rest := trimmed[len("<!--"):]
	end := bytes.Index(rest, []byte("-->"))
	if end < 0 {
		return nil, nil, fmt.Errorf("HTML frontmatter 注释未闭合（缺少 -->）")
	}
	inner := rest[:end]
	body = rest[end+len("-->"):]
	inner = bytes.TrimSpace(inner)
	// 允许 <!-- --- ... --- --> 的写法
	lines := bytes.Split(inner, []byte("\n"))
	if len(lines) >= 2 && isFenceLine(lines[0], "---") && isFenceLine(lines[len(lines)-1], "---") {
		inner = bytes.Join(lines[1:len(lines)-1], []byte("\n"))
	}
	return inner, body, nil
}

func isFenceLine(line []byte, fence string) bool {
	return strings.TrimSpace(string(line)) == fence
}

var dateLayouts = []string{
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02 15:04",
	"2006-01-02T15:04",
	"2006-01-02",
	"2006/01/02 15:04:05",
	"2006/01/02",
}

// ParseDate 解析 frontmatter 中的日期字符串，按 Asia/Shanghai 解释无时区的写法。
func ParseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, nil
	}
	for _, layout := range dateLayouts {
		if strings.Contains(layout, "Z07:00") {
			if t, err := time.Parse(layout, s); err == nil {
				return t.In(Location), nil
			}
			continue
		}
		if t, err := time.ParseInLocation(layout, s, Location); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("日期格式无法识别: %q（支持 2006-01-02、2006-01-02 15:04、2006-01-02 15:04:05、RFC3339）", s)
}
