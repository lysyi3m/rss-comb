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

// Configuration types

type Config struct {
	Name     string         // Derived from filename (without .yml extension)
	URL      string         `yaml:"url"`
	Enabled  bool           `yaml:"enabled"`
	Settings types.Settings `yaml:"settings"`
	Filters  []types.Filter `yaml:"filters"`
}
