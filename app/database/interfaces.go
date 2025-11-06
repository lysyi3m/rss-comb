package database

import (
	"time"
)

type FeedItem struct {
	GUID        string
	Title       string
	Link        string
	Description string
	Content     string
	PublishedAt time.Time // Changed from *time.Time to time.Time (NOT NULL)
	UpdatedAt   *time.Time
	Authors     []string // Multiple authors in format "email (name)" or "name"
	Categories  []string

	ContentHash     string
	IsFiltered      bool
	EnclosureURL    string // RSS enclosure URL
	EnclosureLength int64  // RSS enclosure length in bytes
	EnclosureType   string // RSS enclosure MIME type
}
