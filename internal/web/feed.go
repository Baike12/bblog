package web

import (
	"encoding/xml"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type rssFeed struct {
	XMLName   xml.Name   `xml:"rss"`
	Version   string     `xml:"version,attr"`
	AtomNS    string     `xml:"xmlns:atom,attr"`
	Channel   rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	Description   string    `xml:"description"`
	Language      string    `xml:"language"`
	LastBuildDate string    `xml:"lastBuildDate"`
	AtomLink      atomLink  `xml:"atom:link"`
	Items         []rssItem `xml:"item"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

type rssItem struct {
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	GUID        guid     `xml:"guid"`
	PubDate     string   `xml:"pubDate"`
	Description string   `xml:"description"`
	Categories  []string `xml:"category,omitempty"`
}

type guid struct {
	IsPermaLink bool   `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

func (s *Server) handleRSS(c *gin.Context) {
	posts := s.Index().PublishedPosts()
	feedURL := s.cfg.AbsURL("/rss.xml")
	items := make([]rssItem, 0, len(posts))
	for _, e := range posts {
		link := s.absolute(e.URL)
		items = append(items, rssItem{
			Title:       e.Title,
			Link:        link,
			GUID:        guid{IsPermaLink: true, Value: link},
			PubDate:     e.Date.Format(time.RFC1123Z),
			Description: e.Summary,
			Categories:  e.Tags,
		})
	}
	lastBuild := time.Now()
	if len(posts) > 0 {
		lastBuild = posts[0].Date
	}
	feed := rssFeed{
		Version: "2.0",
		AtomNS:  "http://www.w3.org/2005/Atom",
		Channel: rssChannel{
			Title:         s.cfg.Site.Title,
			Link:          s.cfg.AbsURL("/"),
			Description:   s.cfg.Site.Description,
			Language:      s.cfg.Site.Language,
			LastBuildDate: lastBuild.Format(time.RFC1123Z),
			AtomLink:      atomLink{Href: feedURL, Rel: "self", Type: "application/rss+xml"},
			Items:         items,
		},
	}
	writeXML(c, feed)
}

type urlset struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

func (s *Server) handleSitemap(c *gin.Context) {
	urls := []sitemapURL{
		{Loc: s.cfg.AbsURL("/")},
		{Loc: s.cfg.AbsURL("/archive/")},
		{Loc: s.cfg.AbsURL("/tags/")},
	}
	for _, e := range s.Index().PublishedPosts() {
		lastmod := e.Date
		if !e.Updated.IsZero() {
			lastmod = e.Updated
		}
		urls = append(urls, sitemapURL{Loc: s.absolute(e.URL), LastMod: lastmod.Format("2006-01-02")})
	}
	for _, e := range s.Index().Pages() {
		urls = append(urls, sitemapURL{Loc: s.absolute(e.URL)})
	}
	writeXML(c, urlset{Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9", URLs: urls})
}

func writeXML(c *gin.Context, v any) {
	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Status(http.StatusOK)
	if _, err := c.Writer.Write([]byte(xml.Header)); err != nil {
		return
	}
	enc := xml.NewEncoder(c.Writer)
	enc.Indent("", "  ")
	if err := enc.Encode(v); err != nil {
		c.Error(err)
	}
}
