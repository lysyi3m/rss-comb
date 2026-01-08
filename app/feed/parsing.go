package feed

import (
	"bytes"
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"net/url"
	"strconv"
	"strings"

	"github.com/lysyi3m/rss-comb/app/types"
	"github.com/mmcdole/gofeed"
)

func Parse(data []byte) (*Metadata, []types.Item, error) {
	gofeedParser := gofeed.NewParser()
	feed, err := gofeedParser.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	metadata := &Metadata{
		Title:       decodeHTMLEntities(feed.Title),
		Link:        feed.Link,
		Description: decodeHTMLEntities(feed.Description),
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

	// Extract iTunes podcast metadata
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
		normalized := normalizeItem(item)
		normalized.ContentHash = generateContentHash(normalized)
		items = append(items, normalized)
	}

	return metadata, items, nil
}

func normalizeURL(rawURL string) string {
	if rawURL == "" {
		return rawURL
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	// List of common tracking parameters to remove
	trackingParams := []string{
		// UTM parameters (Google Analytics)
		"utm_source", "utm_medium", "utm_campaign", "utm_term", "utm_content",
		// Facebook parameters
		"fbclid", "fb_action_ids", "fb_action_types", "fb_ref", "fb_source",
		// Google parameters
		"gclid", "gclsrc", "dclid",
		// Twitter parameters
		"twclid",
		// Microsoft parameters
		"msclkid",
		// Generic tracking parameters
		"ref", "referrer", "source", "campaign", "medium",
		// Email marketing parameters
		"mc_cid", "mc_eid",
		// Other common tracking parameters
		"_ga", "_gl", "igshid", "hsCtaTracking", "hsa_acc", "hsa_ad", "hsa_cam", "hsa_grp", "hsa_kw", "hsa_mt", "hsa_net", "hsa_src", "hsa_tgt", "hsa_ver",
	}

	query := parsedURL.Query()

	for _, param := range trackingParams {
		query.Del(param)
	}

	parsedURL.RawQuery = query.Encode()

	return parsedURL.String()
}

func normalizeItem(item *gofeed.Item) types.Item {
	normalizedLink := normalizeURL(item.Link)

	normalized := types.Item{
		GUID:        cmp.Or(item.GUID, normalizedLink),
		Title:       decodeHTMLEntities(item.Title),
		Link:        normalizedLink,
		Description: decodeHTMLEntities(item.Description),
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

	// Extract iTunes podcast episode metadata
	if item.ITunesExt != nil {
		// Duration: gofeed normalizes to seconds string, but handle different formats
		if item.ITunesExt.Duration != "" {
			if duration, err := strconv.Atoi(item.ITunesExt.Duration); err == nil {
				normalized.ITunesDuration = duration
			}
		}

		// Episode and season numbers
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

		// Episode type and image
		normalized.ITunesEpisodeType = item.ITunesExt.EpisodeType
		normalized.ITunesImage = item.ITunesExt.Image
	}

	return normalized
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

func decodeHTMLEntities(s string) string {
	if s == "" {
		return s
	}
	return html.UnescapeString(s)
}
