package feed

import (
	"bytes"
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"html"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/lysyi3m/rss-comb/app/cfg"
	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/types"
	"github.com/mmcdole/gofeed"
)

func parseWithGofeed(data []byte) (*gofeed.Feed, error) {
	parser := gofeed.NewParser()
	feed, err := parser.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed: %w", err)
	}
	return feed, nil
}

func extractBaseMetadata(feed *gofeed.Feed) *Metadata {
	metadata := &Metadata{
		Title:       html.UnescapeString(feed.Title),
		Link:        feed.Link,
		Description: html.UnescapeString(feed.Description),
		Language:    feed.Language,
	}

	if feed.Image != nil {
		metadata.ImageURL = feed.Image.URL
	}

	if feed.PublishedParsed != nil {
		metadata.FeedPublishedAt = feed.PublishedParsed
	}

	if feed.UpdatedParsed != nil {
		metadata.FeedUpdatedAt = feed.UpdatedParsed
	}

	return metadata
}

func normalizeBaseItem(item *gofeed.Item) types.Item {
	normalizedLink := normalizeURL(item.Link)

	normalized := types.Item{
		GUID:        cmp.Or(item.GUID, normalizedLink),
		Title:       html.UnescapeString(item.Title),
		Link:        normalizedLink,
		Description: html.UnescapeString(item.Description),
		Content:     item.Content,
	}

	if item.PublishedParsed != nil {
		normalized.PublishedAt = *item.PublishedParsed
	}

	if item.UpdatedParsed != nil {
		normalized.UpdatedAt = item.UpdatedParsed
	}

	normalized.Authors = extractAuthors(item)

	if item.Categories != nil {
		normalized.Categories = item.Categories
	}

	if len(item.Enclosures) > 0 && item.Enclosures[0] != nil {
		enclosure := item.Enclosures[0]
		normalized.EnclosureURL = enclosure.URL
		normalized.EnclosureType = enclosure.Type

		if enclosure.Length != "" {
			if length, err := strconv.ParseInt(enclosure.Length, 10, 64); err == nil {
				normalized.EnclosureLength = length
			}
		}
	}

	return normalized
}

func normalizeURL(rawURL string) string {
	if rawURL == "" {
		return rawURL
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	trackingParams := []string{
		"utm_source", "utm_medium", "utm_campaign", "utm_term", "utm_content",
		"fbclid", "fb_action_ids", "fb_action_types", "fb_ref", "fb_source",
		"gclid", "gclsrc", "dclid",
		"twclid",
		"msclkid",
		"ref", "referrer", "source", "campaign", "medium",
		"mc_cid", "mc_eid",
		"_ga", "_gl", "igshid", "hsCtaTracking", "hsa_acc", "hsa_ad", "hsa_cam", "hsa_grp", "hsa_kw", "hsa_mt", "hsa_net", "hsa_src", "hsa_tgt", "hsa_ver",
	}

	query := parsedURL.Query()

	for _, param := range trackingParams {
		query.Del(param)
	}

	parsedURL.RawQuery = query.Encode()

	return parsedURL.String()
}

func generateContentHash(item types.Item) string {
	content := fmt.Sprintf("%s|%s",
		item.Title,
		item.Link)

	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

func extractAuthors(item *gofeed.Item) []string {
	var authors []string

	if len(item.Authors) > 0 {
		for _, author := range item.Authors {
			if author != nil {
				authorStr := formatAuthor(author.Name, author.Email)
				if authorStr != "" {
					authors = append(authors, authorStr)
				}
			}
		}
	} else if item.Author != nil {
		authorStr := formatAuthor(item.Author.Name, item.Author.Email)
		if authorStr != "" {
			authors = append(authors, authorStr)
		}
	}

	return authors
}

func formatAuthor(name, email string) string {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)

	if name != "" && email != "" {
		return fmt.Sprintf("%s (%s)", email, name)
	} else if name != "" {
		return name
	} else if email != "" {
		return email
	}

	return ""
}


func writeChannelHeader(buf *bytes.Buffer, feed database.Feed, items []database.Item, cfg *cfg.Cfg) {
	writeElement(buf, "title", feed.DisplayTitle(), 4)
	writeElement(buf, "link", feed.Link, 4)
	description := feed.Description
	if description == "" {
		description = fmt.Sprintf("Processed feed from %s", feed.FeedURL)
	}
	writeElement(buf, "description", description, 4)

	var selfLink string
	if cfg.BaseUrl != "" {
		selfLink = fmt.Sprintf("%s/feeds/%s", cfg.BaseUrl, feed.Name)
	} else {
		selfLink = fmt.Sprintf("http://localhost:%s/feeds/%s", cfg.Port, feed.Name)
	}
	buf.WriteString(fmt.Sprintf("    <atom:link href=\"%s\" rel=\"self\" type=\"application/rss+xml\" />\n",
		html.EscapeString(selfLink)))

	if feed.FeedPublishedAt != nil {
		writeElement(buf, "pubDate", feed.FeedPublishedAt.In(cfg.Location).Format(time.RFC1123Z), 4)
	}

	lastBuildDate := time.Now().In(cfg.Location)
	if len(items) > 0 {
		lastBuildDate = cmp.Or(items[0].PublishedAt, items[0].CreatedAt, lastBuildDate).In(cfg.Location)
	}

	writeElement(buf, "lastBuildDate", lastBuildDate.Format(time.RFC1123Z), 4)
	writeElement(buf, "generator", fmt.Sprintf("RSS-Comb/%s", cfg.Version), 4)
	if feed.Language != "" {
		writeElement(buf, "language", feed.Language, 4)
	}

	if feed.ImageURL != "" {
		buf.WriteString("    <image>\n")
		writeElement(buf, "url", feed.ImageURL, 6)
		writeElement(buf, "title", feed.DisplayTitle(), 6)
		writeElement(buf, "link", feed.Link, 6)
		buf.WriteString("    </image>\n")
	}
}

func writeBaseItem(buf *bytes.Buffer, item database.Item, cfg *cfg.Cfg) {
	buf.WriteString("    <item>\n")

	if item.GUID != "" {
		buf.WriteString(fmt.Sprintf("      <guid isPermaLink=\"%t\">", strings.HasPrefix(item.GUID, "http://") || strings.HasPrefix(item.GUID, "https://")))
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
}

func writeITunesFeedElements(buf *bytes.Buffer, feed database.Feed) {
	if feed.ITunesAuthor != "" {
		writeElement(buf, "itunes:author", feed.ITunesAuthor, 4)
	}

	if feed.ITunesImage != "" {
		buf.WriteString(fmt.Sprintf("    <itunes:image href=\"%s\" />\n",
			html.EscapeString(feed.ITunesImage)))
	}

	if feed.ITunesExplicit != "" {
		writeElement(buf, "itunes:explicit", feed.ITunesExplicit, 4)
	}

	if feed.ITunesOwnerName != "" || feed.ITunesOwnerEmail != "" {
		buf.WriteString("    <itunes:owner>\n")
		if feed.ITunesOwnerName != "" {
			writeElement(buf, "itunes:name", feed.ITunesOwnerName, 6)
		}
		if feed.ITunesOwnerEmail != "" {
			writeElement(buf, "itunes:email", feed.ITunesOwnerEmail, 6)
		}
		buf.WriteString("    </itunes:owner>\n")
	}
}

func writeITunesItemElements(buf *bytes.Buffer, item database.Item) {
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
