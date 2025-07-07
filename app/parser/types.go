package parser

import "time"

// FeedMetadata contains metadata about the parsed feed
type FeedMetadata struct {
	Title       string
	Link        string
	Description string
	IconURL     string
	Language    string
	Updated     *time.Time
}

// NormalizedItem represents a normalized feed item
type NormalizedItem struct {
	GUID          string
	Title         string
	Link          string
	Description   string
	Content       string
	PublishedDate *time.Time
	UpdatedDate   *time.Time
	AuthorName    string
	AuthorEmail   string
	Categories    []string

	ContentHash   string
	IsFiltered    bool
	FilterReason  string
}