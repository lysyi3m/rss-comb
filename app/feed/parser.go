package feed

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mmcdole/gofeed"
)

// NewParser creates a new feed parser
func NewParser() *Parser {
	return &Parser{
		gofeedParser: gofeed.NewParser(),
	}
}

// Parse parses feed data and returns metadata and normalized items
func (p *Parser) Parse(data []byte) (*Metadata, []Item, error) {
	feed, err := p.gofeedParser.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	// Extract feed metadata
	metadata := &Metadata{
		Title:       feed.Title,
		Link:        feed.Link,
		Description: feed.Description,
		Language:    feed.Language,
	}

	// Set feed image URL if available
	if feed.Image != nil {
		metadata.ImageURL = feed.Image.URL
	}

	// Set published timestamp (when feed was last published/updated)
	if feed.PublishedParsed != nil {
		metadata.FeedPublishedAt = feed.PublishedParsed
	}

	// Process feed items
	items := make([]Item, 0, len(feed.Items))
	for _, item := range feed.Items {
		normalized := p.normalizeItem(item)
		normalized.ContentHash = p.generateContentHash(normalized)
		items = append(items, normalized)
	}

	slog.Debug("Feed parsed", "title", metadata.Title, "items_count", len(items))
	return metadata, items, nil
}

// normalizeItem converts a gofeed.Item to our Item format
func (p *Parser) normalizeItem(item *gofeed.Item) Item {
	normalized := Item{
		GUID:        p.coalesce(item.GUID, item.Link),
		Title:       item.Title,
		Link:        item.Link,
		Description: item.Description,
		Content:     item.Content,
	}

	// Set published date
	if item.PublishedParsed != nil {
		normalized.PublishedAt = item.PublishedParsed
	}

	// Set updated date
	if item.UpdatedParsed != nil {
		normalized.UpdatedAt = item.UpdatedParsed
	}

	// Set author information using modern Authors field with fallback to deprecated Author
	normalized.Authors = p.extractAuthors(item)

	// Set categories
	if item.Categories != nil {
		normalized.Categories = item.Categories
	}

	return normalized
}

// generateContentHash generates a hash for content deduplication
func (p *Parser) generateContentHash(item Item) string {
	// Use only title and link for hash generation
	// This prevents duplicate detection when only description changes (e.g., article updates)
	content := fmt.Sprintf("%s|%s",
		item.Title,
		item.Link)

	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// extractAuthors extracts authors from gofeed item using modern Authors field with fallback
func (p *Parser) extractAuthors(item *gofeed.Item) []string {
	var authors []string
	
	// Prefer the modern Authors field
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
		// Fallback to deprecated Author field if Authors is empty
		authorStr := p.formatAuthor(item.Author.Name, item.Author.Email)
		if authorStr != "" {
			authors = append(authors, authorStr)
		}
	}
	
	return authors
}

// formatAuthor formats author name and email into a single string
// Returns "email (name)" if both are present, otherwise just name or email
func (p *Parser) formatAuthor(name, email string) string {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	
	// Only apply email (name) format if both are non-empty after trimming
	if name != "" && email != "" {
		return fmt.Sprintf("%s (%s)", email, name)
	} else if name != "" {
		return name
	} else if email != "" {
		return email
	}
	
	return ""
}

// coalesce returns the first non-empty string from the provided values
func (p *Parser) coalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
