package feed

import (
	"bytes"
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/mmcdole/gofeed"
)

type Parser struct {
	gofeedParser *gofeed.Parser
}

func NewParser() *Parser {
	return &Parser{
		gofeedParser: gofeed.NewParser(),
	}
}

func (p *Parser) Run(data []byte) (*Metadata, []Item, error) {
	feed, err := p.gofeedParser.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	metadata := &Metadata{
		Title:       feed.Title,
		Link:        feed.Link,
		Description: feed.Description,
		Language:    feed.Language,
	}

	if feed.Image != nil {
		metadata.ImageURL = feed.Image.URL
	}

	if feed.PublishedParsed != nil {
		metadata.FeedPublishedAt = feed.PublishedParsed
	}
	items := make([]Item, 0, len(feed.Items))
	for _, item := range feed.Items {
		normalized := p.normalizeItem(item)
		normalized.ContentHash = p.generateContentHash(normalized)
		items = append(items, normalized)
	}

	return metadata, items, nil
}

func (p *Parser) normalizeItem(item *gofeed.Item) Item {
	normalized := Item{
		GUID:        cmp.Or(item.GUID, item.Link),
		Title:       item.Title,
		Link:        item.Link,
		Description: item.Description,
		Content:     item.Content,
	}

	if item.PublishedParsed != nil {
		normalized.PublishedAt = *item.PublishedParsed
	}

	if item.UpdatedParsed != nil {
		normalized.UpdatedAt = item.UpdatedParsed
	}

	normalized.Authors = p.extractAuthors(item)

	if item.Categories != nil {
		normalized.Categories = item.Categories
	}

	// Extract first enclosure if available (RSS 2.0 spec allows only one per item)
	if len(item.Enclosures) > 0 && item.Enclosures[0] != nil {
		enclosure := item.Enclosures[0]
		normalized.EnclosureURL = enclosure.URL
		normalized.EnclosureType = enclosure.Type
		
		// Parse length as int64, handle potential parsing errors
		if enclosure.Length != "" {
			if length, err := strconv.ParseInt(enclosure.Length, 10, 64); err == nil {
				normalized.EnclosureLength = length
			}
		}
	}

	return normalized
}

func (p *Parser) generateContentHash(item Item) string {
	content := fmt.Sprintf("%s|%s",
		item.Title,
		item.Link)

	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

func (p *Parser) extractAuthors(item *gofeed.Item) []string {
	var authors []string

	if len(item.Authors) > 0 {
		for _, author := range item.Authors {
			if author != nil {
				authorStr := p.formatAuthor(author.Name, author.Email)
				if authorStr != "" {
					authors = append(authors, authorStr)
				}
			}
		}
	} else if item.Author != nil {
		authorStr := p.formatAuthor(item.Author.Name, item.Author.Email)
		if authorStr != "" {
			authors = append(authors, authorStr)
		}
	}

	return authors
}

func (p *Parser) formatAuthor(name, email string) string {
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

