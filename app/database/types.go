package database

import (
	"time"
)

type Feed struct {
	ID              string // Database UUID
	Name            string // Configuration feed identifier derived from filename
	FeedURL         string // RSS/Atom feed URL from configuration
	Link            string // Homepage URL from feed's <link> element (RSS 2.0 spec)
	Title           string
	Description     string // Feed's original description from RSS/Atom
	ImageURL        string
	Language        string
	LastFetchedAt   *time.Time
	NextFetchAt     *time.Time
	FeedPublishedAt *time.Time // Feed's own pubDate/published from RSS/Atom
	CreatedAt       time.Time
	UpdatedAt       time.Time // Tracks last successful processing (replaces last_success)
}

type Item struct {
	ID                      string
	FeedID                  string
	GUID                    string
	Link                    string
	Title                   string
	Description             string
	Content                 string
	PublishedAt             time.Time // Changed from *time.Time to time.Time (NOT NULL)
	UpdatedAt               *time.Time
	Authors                 []string // Multiple authors in format "email (name)" or "name"
	Categories              []string
	IsFiltered              bool
	FilterReason            string
	ContentHash             string
	CreatedAt               time.Time
	ContentExtractedAt      *time.Time
	ContentExtractionStatus string // pending, success, failed, skipped
	ContentExtractionError  string
	ExtractionAttempts      int
	EnclosureURL            string // RSS enclosure URL
	EnclosureLength         int64  // RSS enclosure length in bytes
	EnclosureType           string // RSS enclosure MIME type
}
