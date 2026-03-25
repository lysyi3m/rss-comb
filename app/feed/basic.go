package feed

import (
	"bytes"

	"github.com/lysyi3m/rss-comb/app/cfg"
	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/types"
)

type basicType struct{}

func (basicType) Parse(data []byte) (*Metadata, []types.Item, error) {
	feed, err := parseWithGofeed(data)
	if err != nil {
		return nil, nil, err
	}

	metadata := extractBaseMetadata(feed)

	items := make([]types.Item, 0, len(feed.Items))
	for _, item := range feed.Items {
		normalized := normalizeBaseItem(item)
		normalized.ContentHash = generateContentHash(normalized)
		items = append(items, normalized)
	}

	return metadata, items, nil
}

func (basicType) Build(feed database.Feed, items []database.Item, cfg *cfg.Cfg) (string, error) {
	var buf bytes.Buffer

	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	buf.WriteString("\n")
	buf.WriteString(`<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/" xmlns:atom="http://www.w3.org/2005/Atom">`)
	buf.WriteString("\n  <channel>\n")

	writeChannelHeader(&buf, feed, items, cfg)

	for _, item := range items {
		writeBaseItem(&buf, item, cfg)
		buf.WriteString("    </item>\n")
	}

	buf.WriteString("  </channel>\n</rss>")

	return buf.String(), nil
}
