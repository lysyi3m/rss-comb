package feed

import (
	"bytes"
	"fmt"
	"html"
	"strconv"

	"github.com/lysyi3m/rss-comb/app/cfg"
	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/types"
	"github.com/mmcdole/gofeed"
)

type podcastType struct{}

func (podcastType) Parse(data []byte) (*Metadata, []types.Item, error) {
	feed, err := parseWithGofeed(data)
	if err != nil {
		return nil, nil, err
	}

	metadata := extractBaseMetadata(feed)

	if feed.ITunesExt != nil {
		metadata.ITunesAuthor = feed.ITunesExt.Author
		metadata.ITunesImage = feed.ITunesExt.Image
		metadata.ITunesExplicit = feed.ITunesExt.Explicit
		if feed.ITunesExt.Owner != nil {
			metadata.ITunesOwnerName = feed.ITunesExt.Owner.Name
			metadata.ITunesOwnerEmail = feed.ITunesExt.Owner.Email
		}
	}

	items := make([]types.Item, 0, len(feed.Items))
	for _, item := range feed.Items {
		normalized := normalizeBaseItem(item)
		normalizePodcastItem(&normalized, item)
		normalized.ContentHash = generateContentHash(normalized)
		items = append(items, normalized)
	}

	return metadata, items, nil
}

func normalizePodcastItem(normalized *types.Item, item *gofeed.Item) {
	if item.ITunesExt != nil {
		if item.ITunesExt.Duration != "" {
			if duration, err := strconv.Atoi(item.ITunesExt.Duration); err == nil {
				normalized.ITunesDuration = duration
			}
		}

		if item.ITunesExt.Episode != "" {
			if episode, err := strconv.Atoi(item.ITunesExt.Episode); err == nil {
				normalized.ITunesEpisode = episode
			}
		}

		if item.ITunesExt.Season != "" {
			if season, err := strconv.Atoi(item.ITunesExt.Season); err == nil {
				normalized.ITunesSeason = season
			}
		}

		normalized.ITunesEpisodeType = item.ITunesExt.EpisodeType
		normalized.ITunesImage = item.ITunesExt.Image
	}
}

func (podcastType) Build(feed database.Feed, items []database.Item, cfg *cfg.Cfg) (string, error) {
	var buf bytes.Buffer

	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	buf.WriteString("\n")
	buf.WriteString(`<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/" xmlns:atom="http://www.w3.org/2005/Atom" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd">`)
	buf.WriteString("\n  <channel>\n")

	writeChannelHeader(&buf, feed, items, cfg)
	writeITunesFeedElements(&buf, feed)

	for _, item := range items {
		writeBaseItem(&buf, item, cfg)

		if item.EnclosureURL != "" && item.EnclosureType != "" {
			buf.WriteString(fmt.Sprintf("      <enclosure url=\"%s\" length=\"%d\" type=\"%s\" />\n",
				html.EscapeString(item.EnclosureURL),
				item.EnclosureLength,
				html.EscapeString(item.EnclosureType)))
		}

		writeITunesItemElements(&buf, item)
		buf.WriteString("    </item>\n")
	}

	buf.WriteString("  </channel>\n</rss>")

	return buf.String(), nil
}
