package feed

import (
	"time"

	"github.com/lysyi3m/rss-comb/app/types"
)

// Feed processing types

type Metadata struct {
	Title           string
	Link            string
	Description     string
	ImageURL        string
	Language        string
	FeedPublishedAt *time.Time
	FeedUpdatedAt   *time.Time
}

type Item struct {
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

// Configuration types

type Config struct {
	Name     string         // Derived from filename (without .yml extension)
	URL      string         `yaml:"url"`
	Enabled  bool           `yaml:"enabled"`
	Settings types.Settings `yaml:"settings"`
	Filters  []types.Filter `yaml:"filters"`
}
