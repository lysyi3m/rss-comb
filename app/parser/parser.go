package parser

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/mmcdole/gofeed"
)

// Parser handles parsing of RSS/Atom feeds
type Parser struct {
	gofeedParser *gofeed.Parser
}

// NewParser creates a new feed parser
func NewParser() *Parser {
	return &Parser{
		gofeedParser: gofeed.NewParser(),
	}
}

// Parse parses feed data and returns metadata and normalized items
func (p *Parser) Parse(data []byte) (*FeedMetadata, []NormalizedItem, error) {
	feed, err := p.gofeedParser.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	// Extract feed metadata
	metadata := &FeedMetadata{
		Title:       feed.Title,
		Link:        feed.Link,
		Description: feed.Description,
		Language:    feed.Language,
	}

	// Set feed icon if available
	if feed.Image != nil {
		metadata.IconURL = feed.Image.URL
	}

	// Set updated timestamp
	if feed.UpdatedParsed != nil {
		metadata.Updated = feed.UpdatedParsed
	}

	// Process feed items
	items := make([]NormalizedItem, 0, len(feed.Items))
	for _, item := range feed.Items {
		normalized := p.normalizeItem(item)
		normalized.ContentHash = p.generateContentHash(normalized)
		items = append(items, normalized)
	}

	log.Printf("Parsed feed '%s' with %d items", metadata.Title, len(items))
	return metadata, items, nil
}

// normalizeItem converts a gofeed.Item to our NormalizedItem format
func (p *Parser) normalizeItem(item *gofeed.Item) NormalizedItem {
	normalized := NormalizedItem{
		GUID:        p.coalesce(item.GUID, item.Link),
		Title:       item.Title,
		Link:        item.Link,
		Description: item.Description,
		Content:     item.Content,
	}

	// Set published date
	if item.PublishedParsed != nil {
		normalized.PublishedDate = item.PublishedParsed
	}

	// Set updated date
	if item.UpdatedParsed != nil {
		normalized.UpdatedDate = item.UpdatedParsed
	}

	// Set author information
	if item.Author != nil {
		normalized.AuthorName = item.Author.Name
		normalized.AuthorEmail = item.Author.Email
	}

	// Set categories
	if item.Categories != nil {
		normalized.Categories = item.Categories
	}

	return normalized
}

// generateContentHash generates a hash for content deduplication
func (p *Parser) generateContentHash(item NormalizedItem) string {
	// Use only title and link for hash generation
	// This prevents duplicate detection when only description changes (e.g., article updates)
	content := fmt.Sprintf("%s|%s",
		item.Title,
		item.Link)

	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
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

