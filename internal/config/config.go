package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Site    Site    `yaml:"site"`
	Server  Server  `yaml:"server"`
	Content Content `yaml:"content"`
	Data    Data    `yaml:"data"`
	Giscus  Giscus  `yaml:"giscus"`

	// ConfigDir 是本配置文件所在目录，相对路径均以它为基准解析。
	ConfigDir string `yaml:"-"`
}

type Site struct {
	Title       string `yaml:"title"`
	Author      string `yaml:"author"`
	Description string `yaml:"description"`
	BaseURL     string `yaml:"base_url"`
	BasePath    string `yaml:"base_path"`
	Language    string `yaml:"language"`
}

type Server struct {
	Listen string `yaml:"listen"`
}

type Content struct {
	Dir string `yaml:"dir"`
}

type Data struct {
	Dir string `yaml:"dir"`
}

type Giscus struct {
	Repo       string `yaml:"repo"`
	RepoID     string `yaml:"repo_id"`
	Category   string `yaml:"category"`
	CategoryID string `yaml:"category_id"`
	Mapping    string `yaml:"mapping"`
	Lang       string `yaml:"lang"`
}

// Load 读取配置文件并应用环境变量覆盖。
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置 %s: %w", path, err)
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(raw, cfg); err != nil {
		return nil, fmt.Errorf("解析配置 %s: %w", path, err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	cfg.ConfigDir = filepath.Dir(abs)

	cfg.applyDefaults()
	cfg.applyEnv()
	return cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Site.Title == "" {
		c.Site.Title = "我的博客"
	}
	if c.Site.BasePath == "" {
		c.Site.BasePath = "/blog"
	}
	if c.Site.Language == "" {
		c.Site.Language = "zh-CN"
	}
	if c.Server.Listen == "" {
		c.Server.Listen = "0.0.0.0:8090"
	}
	if c.Content.Dir == "" {
		c.Content.Dir = "content"
	}
	if c.Data.Dir == "" {
		c.Data.Dir = "data"
	}
	if c.Giscus.Mapping == "" {
		c.Giscus.Mapping = "pathname"
	}
	if c.Giscus.Lang == "" {
		c.Giscus.Lang = "zh-CN"
	}
	c.Site.BasePath = NormalizeBasePath(c.Site.BasePath)
	c.Site.BaseURL = strings.TrimRight(c.Site.BaseURL, "/")
}

func (c *Config) applyEnv() {
	if v := os.Getenv("BBLOG_LISTEN"); v != "" {
		c.Server.Listen = v
	}
	if v := os.Getenv("BBLOG_CONTENT_DIR"); v != "" {
		c.Content.Dir = v
	}
	if v := os.Getenv("BBLOG_DATA_DIR"); v != "" {
		c.Data.Dir = v
	}
}

// NormalizeBasePath 保证返回值形如 "/blog"；空值表示站点挂在根路径。
func NormalizeBasePath(p string) string {
	p = strings.TrimSpace(p)
	p = strings.Trim(p, "/")
	if p == "" {
		return ""
	}
	return "/" + p
}

func (c *Config) ContentDir() string { return c.resolve(c.Content.Dir) }

func (c *Config) DataDir() string { return c.resolve(c.Data.Dir) }

func (c *Config) resolve(p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Clean(filepath.Join(c.ConfigDir, p))
}

// URL 拼接站点基础路径，入参需以 "/" 开头。
func (c *Config) URL(p string) string {
	if p == "" {
		p = "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return c.Site.BasePath + p
}

// AbsURL 返回含域名与基础路径的绝对地址，用于 RSS 与 og:url。
func (c *Config) AbsURL(p string) string {
	return c.Site.BaseURL + c.URL(p)
}

// PublishToken 从环境变量读取发布令牌，未设置时为空。
func PublishToken() string { return os.Getenv("PUBLISH_TOKEN") }
