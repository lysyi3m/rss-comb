package feed

import (
	"bytes"
	"cmp"
	"encoding/xml"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/lysyi3m/rss-comb/app/cfg"
	"github.com/lysyi3m/rss-comb/app/database"
)

func GenerateRSS(feed database.Feed, items []database.Item, cfg *cfg.Cfg) (string, error) {
	var buf bytes.Buffer

	hasPodcastData := hasITunesData(feed, items)

	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	buf.WriteString("\n")
	if hasPodcastData {
		buf.WriteString(`<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/" xmlns:atom="http://www.w3.org/2005/Atom" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd">`)
	} else {
		buf.WriteString(`<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/" xmlns:atom="http://www.w3.org/2005/Atom">`)
	}
	buf.WriteString("\n  <channel>\n")

	writeElement(&buf, "title", feed.Title, 4)
	writeElement(&buf, "link", feed.Link, 4)
	description := feed.Description
	if description == "" {
		description = fmt.Sprintf("Processed feed from %s", feed.FeedURL)
	}
	writeElement(&buf, "description", description, 4)

	var selfLink string
	if cfg.BaseUrl != "" {
		selfLink = fmt.Sprintf("%s/feeds/%s", cfg.BaseUrl, feed.Name)
	} else {
		selfLink = fmt.Sprintf("http://localhost:%s/feeds/%s", cfg.Port, feed.Name)
	}
	buf.WriteString(fmt.Sprintf("    <atom:link href=\"%s\" rel=\"self\" type=\"application/rss+xml\" />\n",
		html.EscapeString(selfLink)))

	if feed.FeedPublishedAt != nil {
		writeElement(&buf, "pubDate", feed.FeedPublishedAt.In(cfg.Location).Format(time.RFC1123Z), 4)
	}

	lastBuildDate := time.Now().In(cfg.Location)
	if len(items) > 0 {
		lastBuildDate = cmp.Or(items[0].PublishedAt, items[0].CreatedAt, lastBuildDate).In(cfg.Location)
	}

	writeElement(&buf, "lastBuildDate", lastBuildDate.Format(time.RFC1123Z), 4)
	writeElement(&buf, "generator", fmt.Sprintf("RSS-Comb/%s", cfg.Version), 4)
	if feed.Language != "" {
		writeElement(&buf, "language", feed.Language, 4)
	}

	if feed.ImageURL != "" {
		buf.WriteString("    <image>\n")
		writeElement(&buf, "url", feed.ImageURL, 6)
		writeElement(&buf, "title", feed.Title, 6)
		writeElement(&buf, "link", feed.Link, 6)
		buf.WriteString("    </image>\n")
	}

	if feed.ITunesAuthor != "" {
		writeElement(&buf, "itunes:author", feed.ITunesAuthor, 4)
	}

	if feed.ITunesImage != "" {
		buf.WriteString(fmt.Sprintf("    <itunes:image href=\"%s\" />\n",
			html.EscapeString(feed.ITunesImage)))
	}

	if feed.ITunesExplicit != "" {
		writeElement(&buf, "itunes:explicit", feed.ITunesExplicit, 4)
	}

	if feed.ITunesOwnerName != "" || feed.ITunesOwnerEmail != "" {
		buf.WriteString("    <itunes:owner>\n")
		if feed.ITunesOwnerName != "" {
			writeElement(&buf, "itunes:name", feed.ITunesOwnerName, 6)
		}
		if feed.ITunesOwnerEmail != "" {
			writeElement(&buf, "itunes:email", feed.ITunesOwnerEmail, 6)
		}
		buf.WriteString("    </itunes:owner>\n")
	}

	for _, item := range items {
		writeItem(&buf, item, cfg)
	}

	buf.WriteString("  </channel>\n</rss>")

	return buf.String(), nil
}

func writeItem(buf *bytes.Buffer, item database.Item, cfg *cfg.Cfg) {
	buf.WriteString("    <item>\n")

	if item.GUID != "" {
		buf.WriteString(fmt.Sprintf("      <guid isPermaLink=\"%t\">", isURL(item.GUID)))
		xml.EscapeText(buf, []byte(item.GUID))
		buf.WriteString("</guid>\n")
	}

	if item.Title != "" {
		writeElement(buf, "title", item.Title, 6)
	}

	if item.Link != "" {
		writeElement(buf, "link", item.Link, 6)
	}

	writeElement(buf, "description", cmp.Or(item.Description, "No description available"), 6)

	if item.Content != "" && item.Content != item.Description {
		buf.WriteString("      <content:encoded><![CDATA[")
		buf.WriteString(item.Content)
		buf.WriteString("]]></content:encoded>\n")
	}

	writeElement(buf, "pubDate", item.PublishedAt.In(cfg.Location).Format(time.RFC1123Z), 6)

	if len(item.Authors) > 0 && item.Authors[0] != "" {
		writeElement(buf, "author", item.Authors[0], 6)
	}

	for _, category := range item.Categories {
		if category != "" {
			writeElement(buf, "category", category, 6)
		}
	}

	if item.EnclosureURL != "" && item.EnclosureType != "" {
		buf.WriteString(fmt.Sprintf("      <enclosure url=\"%s\" length=\"%d\" type=\"%s\" />\n",
			html.EscapeString(item.EnclosureURL),
			item.EnclosureLength,
			html.EscapeString(item.EnclosureType)))
	}

	if item.ITunesDuration > 0 {
		writeElement(buf, "itunes:duration", formatDuration(item.ITunesDuration), 6)
	}

	if item.ITunesEpisode > 0 {
		writeElement(buf, "itunes:episode", fmt.Sprintf("%d", item.ITunesEpisode), 6)
	}

	if item.ITunesSeason > 0 {
		writeElement(buf, "itunes:season", fmt.Sprintf("%d", item.ITunesSeason), 6)
	}

	if item.ITunesEpisodeType != "" {
		writeElement(buf, "itunes:episodeType", item.ITunesEpisodeType, 6)
	}

	if item.ITunesImage != "" {
		buf.WriteString(fmt.Sprintf("      <itunes:image href=\"%s\" />\n",
			html.EscapeString(item.ITunesImage)))
	}

	buf.WriteString("    </item>\n")
}

func writeElement(buf *bytes.Buffer, tag, content string, indent int) {
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

func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func formatDuration(seconds int) string {
	if seconds <= 0 {
		return ""
	}

	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60

	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
	}
	return fmt.Sprintf("%02d:%02d", minutes, secs)
}

func hasITunesData(feed database.Feed, items []database.Item) bool {
	if feed.ITunesAuthor != "" || feed.ITunesImage != "" || feed.ITunesExplicit != "" ||
		feed.ITunesOwnerName != "" || feed.ITunesOwnerEmail != "" {
		return true
	}

	for _, item := range items {
		if item.ITunesDuration > 0 || item.ITunesEpisode > 0 || item.ITunesSeason > 0 ||
			item.ITunesEpisodeType != "" || item.ITunesImage != "" {
			return true
		}
	}

	return false
}
