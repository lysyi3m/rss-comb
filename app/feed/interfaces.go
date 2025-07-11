package feed

import (
	"net/http"
	
	"github.com/lysyi3m/rss-comb/app/database"
)

// ProcessorInterface defines the interface for feed processing operations
type ProcessorInterface interface {
	ProcessFeed(feedID, configFile string) error
	IsFeedEnabled(configFile string) bool
	ReapplyFilters(feedID, configFile string) (int, int, error)
}

// ParserInterface defines the interface for feed parsing operations
type ParserInterface interface {
	Parse(data []byte) (*Metadata, []Item, error)
}

// GeneratorInterface defines the interface for RSS generation operations
type GeneratorInterface interface {
	Generate(feed database.Feed, items []database.Item) (string, error)
	GenerateEmpty(title, feedURL string) string
}

// HTTPClient defines the interface for HTTP operations
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}