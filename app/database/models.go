package database

import (
	"time"
)

// Feed represents a feed record in the database
type Feed struct {
	ID          string     // Database UUID
	FeedID      string     // Configuration feed ID
	ConfigFile  string
	FeedURL     string     // RSS/Atom feed URL from configuration
	Link        string     // Homepage URL from feed's <link> element (RSS 2.0 spec)
	Title       string
	ImageURL    string
	Language    string
	LastFetched *time.Time
	LastSuccess *time.Time
	NextFetch   *time.Time
	Enabled     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Item represents a feed item record in the database
type Item struct {
	ID            string
	FeedID        string
	GUID          string
	Link          string
	Title         string
	Description   string
	Content       string
	PublishedDate *time.Time
	UpdatedDate   *time.Time
	AuthorName    string
	AuthorEmail   string
	Categories    []string
	IsFiltered    bool
	FilterReason  string
	ContentHash   string
	CreatedAt     time.Time
}