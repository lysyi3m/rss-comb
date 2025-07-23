package feed

import (
	"bytes"
	"cmp"
	"encoding/xml"
	"fmt"
	"html"
	"time"

	"github.com/lysyi3m/rss-comb/app/cfg"
	"github.com/lysyi3m/rss-comb/app/database"
)

type Generator struct{}

func NewGenerator() *Generator {
	return &Generator{}
}

func (g *Generator) Run(feed database.Feed, items []database.Item) (string, error) {
	var buf bytes.Buffer

	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	buf.WriteString("\n")
	buf.WriteString(`<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/" xmlns:atom="http://www.w3.org/2005/Atom">`)
	buf.WriteString("\n  <channel>\n")

	g.writeElement(&buf, "title", feed.Title, 4)
	g.writeElement(&buf, "link", feed.Link, 4)
	description := feed.Description
	if description == "" {
		description = fmt.Sprintf("Processed feed from %s", feed.FeedURL)
	}
	g.writeElement(&buf, "description", description, 4)

	var selfLink string
	if cfg.Get().BaseUrl != "" {
		selfLink = fmt.Sprintf("%s/feeds/%s", cfg.Get().BaseUrl, feed.Name)
	} else {
		selfLink = fmt.Sprintf("http://localhost:%s/feeds/%s", cfg.Get().Port, feed.Name)
	}
	buf.WriteString(fmt.Sprintf("    <atom:link href=\"%s\" rel=\"self\" type=\"application/rss+xml\" />\n",
		html.EscapeString(selfLink)))

	if feed.FeedPublishedAt != nil {
		g.writeElement(&buf, "pubDate", feed.FeedPublishedAt.Format(time.RFC1123Z), 4)
	}

	lastBuildDate := time.Now().In(time.Local)
	if len(items) > 0 {
		lastBuildDate = cmp.Or(items[0].PublishedAt, items[0].CreatedAt, lastBuildDate)
	}

	g.writeElement(&buf, "lastBuildDate", lastBuildDate.Format(time.RFC1123Z), 4)
	g.writeElement(&buf, "generator", fmt.Sprintf("RSS-Comb/%s", cfg.Get().Version), 4)
	if feed.Language != "" {
		g.writeElement(&buf, "language", feed.Language, 4)
	}

	if feed.ImageURL != "" {
		buf.WriteString("    <image>\n")
		g.writeElement(&buf, "url", feed.ImageURL, 6)
		g.writeElement(&buf, "title", feed.Title, 6)
		g.writeElement(&buf, "link", feed.Link, 6)
		buf.WriteString("    </image>\n")
	}

	for _, item := range items {
		g.writeItem(&buf, item)
	}

	buf.WriteString("  </channel>\n</rss>")

	return buf.String(), nil
}

func (g *Generator) writeItem(buf *bytes.Buffer, item database.Item) {
	buf.WriteString("    <item>\n")

	if item.GUID != "" {
		buf.WriteString(fmt.Sprintf("      <guid isPermaLink=\"%t\">", g.isURL(item.GUID)))
		xml.EscapeText(buf, []byte(item.GUID))
		buf.WriteString("</guid>\n")
	}

	if item.Title != "" {
		g.writeElement(buf, "title", item.Title, 6)
	}

	if item.Link != "" {
		g.writeElement(buf, "link", item.Link, 6)
	}

	g.writeElement(buf, "description", cmp.Or(item.Description, "No description available"), 6)

	if item.Content != "" && item.Content != item.Description {
		buf.WriteString("      <content:encoded><![CDATA[")
		buf.WriteString(item.Content)
		buf.WriteString("]]></content:encoded>\n")
	}

	g.writeElement(buf, "pubDate", item.PublishedAt.Format(time.RFC1123Z), 6)

	if len(item.Authors) > 0 && item.Authors[0] != "" {
		g.writeElement(buf, "author", item.Authors[0], 6)
	}

	for _, category := range item.Categories {
		if category != "" {
			g.writeElement(buf, "category", category, 6)
		}
	}

	// Add enclosure element if present (RSS 2.0 spec: url, length, type are required)
	if item.EnclosureURL != "" && item.EnclosureType != "" {
		buf.WriteString(fmt.Sprintf("      <enclosure url=\"%s\" length=\"%d\" type=\"%s\" />\n",
			html.EscapeString(item.EnclosureURL),
			item.EnclosureLength,
			html.EscapeString(item.EnclosureType)))
	}

	buf.WriteString("    </item>\n")
}

func (g *Generator) writeElement(buf *bytes.Buffer, tag, content string, indent int) {
	if content == "" {
		return
	}

	for i := 0; i < indent; i++ {
		buf.WriteByte(' ')
	}

	buf.WriteString("<")
	buf.WriteString(tag)
	buf.WriteString(">")
	xml.EscapeText(buf, []byte(content))
	buf.WriteString("</")
	buf.WriteString(tag)
	buf.WriteString(">\n")
}

func (g *Generator) isURL(s string) bool {
	return (len(s) > 7 && s[:7] == "http://") || (len(s) > 8 && s[:8] == "https://")
}
