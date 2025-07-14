package feed

import (
	"net/http"
	"time"

	"github.com/mmcdole/gofeed"
)

// Processor handles feed processing including fetching, parsing, filtering, and storage
type Processor struct {
	parser      *Parser
	generator   *Generator
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
	Title       string
	Link        string
	Description string
	ImageURL    string
	Language    string
	Published   *time.Time
}

// Item represents a normalized feed item
type Item struct {
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
