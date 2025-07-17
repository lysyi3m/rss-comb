package feed

import (
	"net/http"
	"time"

	"github.com/mmcdole/gofeed"
)

// Processor handles feed processing including fetching, parsing, filtering, and storage
type Processor struct {
	parser      *Parser
	feedRepo    FeedRepositoryInterface
	itemRepo    ItemRepositoryInterface
	client      *http.Client
	userAgent   string
}

// Parser handles parsing of RSS/Atom feeds
type Parser struct {
	gofeedParser *gofeed.Parser
}

// Generator handles generating RSS 2.0 XML from feed data
type Generator struct {
	Port string // Server port for self-referencing links
}

// Metadata contains metadata about the parsed feed
type Metadata struct {
	Title           string
	Link            string
	Description     string
	ImageURL        string
	Language        string
	FeedPublishedAt *time.Time
}

// Item represents a normalized feed item
type Item struct {
	GUID        string
	Title       string
	Link        string
	Description string
	Content     string
	PublishedAt *time.Time
	UpdatedAt   *time.Time
	Authors     []string // Multiple authors in format "email (name)" or "name"
	Categories  []string

	ContentHash  string
	IsFiltered   bool
	FilterReason string
}
