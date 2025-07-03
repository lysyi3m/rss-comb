package api

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"html"
	"time"

	"github.com/lysyi3m/rss-comb/internal/database"
)

// RSSGenerator handles generating RSS 2.0 XML from feed data
type RSSGenerator struct{}

// NewRSSGenerator creates a new RSS generator
func NewRSSGenerator() *RSSGenerator {
	return &RSSGenerator{}
}

// Generate creates RSS 2.0 XML from feed and items data
func (g *RSSGenerator) Generate(feed database.Feed, items []database.Item) (string, error) {
	var buf bytes.Buffer

	// XML declaration
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	buf.WriteString("\n")
	
	// RSS root element with namespaces
	buf.WriteString(`<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/" xmlns:atom="http://www.w3.org/2005/Atom">`)
	buf.WriteString("\n  <channel>\n")

	// Channel metadata
	g.writeElement(&buf, "title", feed.Name, 4)
	g.writeElement(&buf, "link", feed.URL, 4)
	g.writeElement(&buf, "description", fmt.Sprintf("Processed feed from %s", feed.URL), 4)
	
	// Self-referencing link (Atom namespace)
	selfLink := fmt.Sprintf("http://localhost:8080/feed?url=%s", feed.URL)
	buf.WriteString(fmt.Sprintf("    <atom:link href=\"%s\" rel=\"self\" type=\"application/rss+xml\" />\n", 
		html.EscapeString(selfLink)))
	
	g.writeElement(&buf, "lastBuildDate", time.Now().Format(time.RFC1123Z), 4)
	g.writeElement(&buf, "generator", "RSS-Comb/1.0", 4)
	// Language (only include if available)
	if feed.Language != "" {
		g.writeElement(&buf, "language", feed.Language, 4)
	}

	// Feed icon if available
	if feed.IconURL != "" {
		buf.WriteString("    <image>\n")
		g.writeElement(&buf, "url", feed.IconURL, 6)
		g.writeElement(&buf, "title", feed.Name, 6)
		g.writeElement(&buf, "link", feed.URL, 6)
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
func (g *RSSGenerator) writeItem(buf *bytes.Buffer, item database.Item) {
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
	if item.PublishedDate != nil {
		g.writeElement(buf, "pubDate", item.PublishedDate.Format(time.RFC1123Z), 6)
	}

	// Author
	if item.AuthorName != "" {
		author := item.AuthorName
		if item.AuthorEmail != "" {
			author = fmt.Sprintf("%s (%s)", item.AuthorEmail, item.AuthorName)
		}
		g.writeElement(buf, "author", author, 6)
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
func (g *RSSGenerator) writeElement(buf *bytes.Buffer, tag, content string, indent int) {
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
func (g *RSSGenerator) isURL(s string) bool {
	return len(s) > 7 && (s[:7] == "http://" || s[:8] == "https://")
}

// GenerateEmpty creates an empty RSS feed template
func (g *RSSGenerator) GenerateEmpty(feedName, feedURL string) string {
	if feedName == "" {
		feedName = "Empty Feed"
	}
	if feedURL == "" {
		feedURL = "http://localhost:8080/feed"
	}

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
  <channel>
    <title>%s</title>
    <link>%s</link>
    <description>Feed is being processed. Please check back later.</description>
    <lastBuildDate>%s</lastBuildDate>
    <generator>RSS-Comb/1.0</generator>
  </channel>
</rss>`, 
		html.EscapeString(feedName), 
		html.EscapeString(feedURL), 
		time.Now().Format(time.RFC1123Z))
}

// GenerateError creates an RSS feed with error information
func (g *RSSGenerator) GenerateError(feedName, feedURL, errorMsg string) string {
	if feedName == "" {
		feedName = "Error Feed"
	}
	if feedURL == "" {
		feedURL = "http://localhost:8080/feed"
	}

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>%s - Error</title>
    <link>%s</link>
    <description>Error processing feed: %s</description>
    <lastBuildDate>%s</lastBuildDate>
    <generator>RSS-Comb/1.0</generator>
    <item>
      <title>Feed Processing Error</title>
      <description>%s</description>
      <pubDate>%s</pubDate>
      <guid isPermaLink="false">error-%d</guid>
    </item>
  </channel>
</rss>`, 
		html.EscapeString(feedName), 
		html.EscapeString(feedURL), 
		html.EscapeString(errorMsg),
		time.Now().Format(time.RFC1123Z),
		html.EscapeString(errorMsg),
		time.Now().Format(time.RFC1123Z),
		time.Now().Unix())
}