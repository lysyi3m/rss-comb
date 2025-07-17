package feed

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"html"
	"time"

	"github.com/lysyi3m/rss-comb/app/config"
	"github.com/lysyi3m/rss-comb/app/database"
)

// NewGenerator creates a new RSS generator
func NewGenerator(port string) *Generator {
	if port == "" {
		port = "8080"
	}
	return &Generator{Port: port}
}

// Generate creates RSS 2.0 XML from feed and items data
func (g *Generator) Generate(feed database.Feed, items []database.Item) (string, error) {
	var buf bytes.Buffer

	// XML declaration
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	buf.WriteString("\n")
	
	// RSS root element with namespaces
	buf.WriteString(`<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/" xmlns:atom="http://www.w3.org/2005/Atom">`)
	buf.WriteString("\n  <channel>\n")

	// Channel metadata
	g.writeElement(&buf, "title", feed.Title, 4)
	g.writeElement(&buf, "link", feed.Link, 4)
	g.writeElement(&buf, "description", fmt.Sprintf("Processed feed from %s", feed.FeedURL), 4)
	
	// Self-referencing link (Atom namespace)
	selfLink := fmt.Sprintf("http://localhost:%s/feeds/%s", g.Port, feed.FeedID)
	buf.WriteString(fmt.Sprintf("    <atom:link href=\"%s\" rel=\"self\" type=\"application/xml\" />\n", 
		html.EscapeString(selfLink)))
	
	g.writeElement(&buf, "lastBuildDate", time.Now().In(time.Local).Format(time.RFC1123Z), 4)
	g.writeElement(&buf, "generator", fmt.Sprintf("RSS-Comb/%s", config.GetVersion()), 4)
	// Language (only include if available)
	if feed.Language != "" {
		g.writeElement(&buf, "language", feed.Language, 4)
	}

	// Feed image if available
	if feed.ImageURL != "" {
		buf.WriteString("    <image>\n")
		g.writeElement(&buf, "url", feed.ImageURL, 6)
		g.writeElement(&buf, "title", feed.Title, 6)
		g.writeElement(&buf, "link", feed.Link, 6)
		buf.WriteString("    </image>\n")
	}

	// Items
	for _, item := range items {
		g.writeItem(&buf, item)
	}

	buf.WriteString("  </channel>\n</rss>")

	return buf.String(), nil
}

// writeItem writes a single RSS item
func (g *Generator) writeItem(buf *bytes.Buffer, item database.Item) {
	buf.WriteString("    <item>\n")

	// GUID (required)
	if item.GUID != "" {
		buf.WriteString(fmt.Sprintf("      <guid isPermaLink=\"%t\">", g.isURL(item.GUID)))
		xml.EscapeText(buf, []byte(item.GUID))
		buf.WriteString("</guid>\n")
	}

	// Title
	if item.Title != "" {
		g.writeElement(buf, "title", item.Title, 6)
	}

	// Link
	if item.Link != "" {
		g.writeElement(buf, "link", item.Link, 6)
	}

	// Description (required)
	description := item.Description
	if description == "" {
		description = "No description available"
	}
	g.writeElement(buf, "description", description, 6)

	// Content (if different from description)
	if item.Content != "" && item.Content != item.Description {
		buf.WriteString("      <content:encoded><![CDATA[")
		buf.WriteString(item.Content)
		buf.WriteString("]]></content:encoded>\n")
	}

	// Published date
	if item.PublishedAt != nil {
		g.writeElement(buf, "pubDate", item.PublishedAt.Format(time.RFC1123Z), 6)
	}

	// Authors (use first author for RSS 2.0 compatibility)
	if len(item.Authors) > 0 && item.Authors[0] != "" {
		g.writeElement(buf, "author", item.Authors[0], 6)
	}

	// Categories
	for _, category := range item.Categories {
		if category != "" {
			g.writeElement(buf, "category", category, 6)
		}
	}

	buf.WriteString("    </item>\n")
}

// writeElement writes an XML element with proper escaping
func (g *Generator) writeElement(buf *bytes.Buffer, tag, content string, indent int) {
	if content == "" {
		return
	}

	// Add indentation
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

// isURL checks if a string looks like a URL (for GUID isPermaLink attribute)
func (g *Generator) isURL(s string) bool {
	return (len(s) > 7 && s[:7] == "http://") || (len(s) > 8 && s[:8] == "https://")
}
