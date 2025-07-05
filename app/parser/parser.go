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

	// Convert to raw data map for storage
	normalized.RawData = p.itemToMap(item)

	return normalized
}

// generateContentHash generates a hash for content deduplication
func (p *Parser) generateContentHash(item NormalizedItem) string {
	// Use title, link, and description for hash generation
	// This provides good deduplication while being resilient to minor changes
	content := fmt.Sprintf("%s|%s|%s",
		item.Title,
		item.Link,
		item.Description)

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

// itemToMap converts a gofeed.Item to a map for JSON storage
func (p *Parser) itemToMap(item *gofeed.Item) map[string]interface{} {
	itemMap := make(map[string]interface{})
	
	// Basic fields
	itemMap["title"] = item.Title
	itemMap["description"] = item.Description
	itemMap["content"] = item.Content
	itemMap["link"] = item.Link
	itemMap["guid"] = item.GUID
	
	// Dates
	if item.Published != "" {
		itemMap["published"] = item.Published
	}
	if item.Updated != "" {
		itemMap["updated"] = item.Updated
	}
	
	// Author
	if item.Author != nil {
		authorMap := make(map[string]interface{})
		authorMap["name"] = item.Author.Name
		authorMap["email"] = item.Author.Email
		itemMap["author"] = authorMap
	}
	
	// Categories
	if len(item.Categories) > 0 {
		itemMap["categories"] = item.Categories
	}
	
	// Enclosures (for podcasts, etc.)
	if len(item.Enclosures) > 0 {
		enclosures := make([]map[string]interface{}, len(item.Enclosures))
		for i, enc := range item.Enclosures {
			enclosures[i] = map[string]interface{}{
				"url":    enc.URL,
				"type":   enc.Type,
				"length": enc.Length,
			}
		}
		itemMap["enclosures"] = enclosures
	}
	
	// Custom fields (extensions)
	if item.Extensions != nil {
		itemMap["extensions"] = item.Extensions
	}
	
	return itemMap
}