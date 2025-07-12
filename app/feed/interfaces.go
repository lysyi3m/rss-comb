package feed

import (
	"github.com/lysyi3m/rss-comb/app/config"
	"github.com/lysyi3m/rss-comb/app/database"
)

// ProcessorInterface defines the interface for feed processing operations.
// Used by task scheduler and API handlers to process feeds and manage filtering.
// This interface provides the core feed processing functionality including fetching,
// parsing, filtering, and storing feed items. Configuration is injected per operation
// for clean dependency management and improved testability.
type ProcessorInterface interface {
	ProcessFeed(feedID string, feedConfig *config.FeedConfig) error
	IsFeedEnabled(feedConfig *config.FeedConfig) bool
	ReapplyFilters(feedID string, feedConfig *config.FeedConfig) (int, int, error)
}

// ParserInterface defines the interface for feed parsing operations.
// Used by feed processor to parse RSS/Atom feed data into normalized structures.
// This interface provides standardized parsing functionality for various feed formats.
type ParserInterface interface {
	Parse(data []byte) (*Metadata, []Item, error)
}

// GeneratorInterface defines the interface for RSS generation operations.
// Used by API handlers to generate RSS 2.0 XML output from database feed data.
// This interface provides RSS generation and formatting functionality.
type GeneratorInterface interface {
	Generate(feed database.Feed, items []database.Item) (string, error)
	GenerateEmpty(title, feedURL string) string
}

