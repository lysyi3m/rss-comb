package feed

import (
	"bytes"
	"fmt"
	"html"

	"github.com/lysyi3m/rss-comb/app/cfg"
	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/types"
	"github.com/mmcdole/gofeed"
)

type youtubeType struct{}

func (youtubeType) Parse(data []byte) (*Metadata, []types.Item, error) {
	feed, err := parseWithGofeed(data)
	if err != nil {
		return nil, nil, err
	}

	metadata := extractBaseMetadata(feed)

	items := make([]types.Item, 0, len(feed.Items))
	for _, item := range feed.Items {
		normalized := normalizeBaseItem(item)
		normalizeYouTubeItem(&normalized, item)
		normalized.ContentHash = generateContentHash(normalized)
		items = append(items, normalized)
	}

	// Infer feed image from first item thumbnail (YouTube feeds lack feed-level images)
	if metadata.ImageURL == "" && len(items) > 0 && items[0].ITunesImage != "" {
		metadata.ImageURL = items[0].ITunesImage
	}

	// Infer iTunes author from first item author (YouTube feeds have per-entry authors)
	if metadata.ITunesAuthor == "" && len(items) > 0 && len(items[0].Authors) > 0 {
		metadata.ITunesAuthor = items[0].Authors[0]
	}

	return metadata, items, nil
}

func normalizeYouTubeItem(normalized *types.Item, item *gofeed.Item) {
	if normalized.Description == "" {
		normalized.Description = extractMediaDescription(item)
	}

	if normalized.ITunesImage == "" {
		normalized.ITunesImage = extractMediaThumbnail(item)
	}
}

// YouTube Atom feeds store descriptions in media:group, not standard fields.
func extractMediaDescription(item *gofeed.Item) string {
	if mediaGroup, ok := item.Extensions["media"]["group"]; ok && len(mediaGroup) > 0 {
		if descs, ok := mediaGroup[0].Children["description"]; ok && len(descs) > 0 {
			return html.UnescapeString(descs[0].Value)
		}
	}
	return ""
}

func extractMediaThumbnail(item *gofeed.Item) string {
	if mediaGroup, ok := item.Extensions["media"]["group"]; ok && len(mediaGroup) > 0 {
		if thumbs, ok := mediaGroup[0].Children["thumbnail"]; ok && len(thumbs) > 0 {
			if url, ok := thumbs[0].Attrs["url"]; ok {
				return url
			}
		}
	}
	return ""
}

func (youtubeType) Build(feed database.Feed, items []database.Item, cfg *cfg.Cfg) (string, error) {
	var buf bytes.Buffer

	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	buf.WriteString("\n")
	buf.WriteString(`<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/" xmlns:atom="http://www.w3.org/2005/Atom" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd">`)
	buf.WriteString("\n  <channel>\n")

	writeChannelHeader(&buf, feed, items, cfg)
	writeITunesFeedElements(&buf, feed)

	for _, item := range items {
		writeBaseItem(&buf, item, cfg)

		if item.MediaPath != "" && item.MediaSize > 0 {
			mediaURL := fmt.Sprintf("%s/media/%s", cfg.BaseUrl, item.MediaPath)
			buf.WriteString(fmt.Sprintf("      <enclosure url=\"%s\" length=\"%d\" type=\"%s\" />\n",
				html.EscapeString(mediaURL),
				item.MediaSize,
				"audio/mpeg"))
		}

		writeITunesItemElements(&buf, item)
		buf.WriteString("    </item>\n")
	}

	buf.WriteString("  </channel>\n</rss>")

	return buf.String(), nil
}
