package database

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/lysyi3m/rss-comb/app/types"
)

type Feed struct {
	ID              string // Database UUID
	Name            string // Configuration feed identifier derived from filename
	FeedURL         string // RSS/Atom feed URL from configuration
	Link            string // Homepage URL from feed's <link> element (RSS 2.0 spec)
	Title           string // Custom title from config (optional override)
	SourceTitle     string // Title from source feed
	Description     string // Feed's original description from RSS/Atom
	ImageURL        string
	Language        string
	LastFetchedAt   *time.Time
	NextFetchAt     *time.Time
	FeedPublishedAt *time.Time // Feed's own pubDate/published from RSS/Atom
	FeedUpdatedAt   *time.Time // Feed's own updated/lastBuildDate from RSS/Atom
	ContentHash     *string    // SHA-256 hash of raw feed content for change detection
	CreatedAt       time.Time
	UpdatedAt       time.Time // Tracks last successful processing (replaces last_success)

	// Configuration fields
	IsEnabled  bool            // Whether the feed is enabled
	Settings   json.RawMessage // JSONB feed settings
	Filters    json.RawMessage // JSONB feed filters
	ConfigHash *string         // SHA-256 hash of config file for change detection

	// iTunes podcast extension fields
	ITunesAuthor     string
	ITunesImage      string
	ITunesExplicit   string
	ITunesOwnerName  string
	ITunesOwnerEmail string
}

func (f *Feed) DisplayTitle() string {
	if f.Title != "" {
		return f.Title
	}
	return f.SourceTitle
}

func (f *Feed) GetSettings() (*types.Settings, error) {
	if f.Settings == nil {
		return &types.Settings{
			RefreshInterval: 1800,
			MaxItems:        50,
			Timeout:         30,
		}, nil
	}

	var settings types.Settings
	if err := json.Unmarshal(f.Settings, &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
	}
	return &settings, nil
}

func (f *Feed) GetFilters() ([]types.Filter, error) {
	if f.Filters == nil {
		return []types.Filter{}, nil
	}

	var filters []types.Filter
	if err := json.Unmarshal(f.Filters, &filters); err != nil {
		return nil, fmt.Errorf("failed to unmarshal filters: %w", err)
	}
	return filters, nil
}

type Item struct {
	ID        string
	FeedID    string
	CreatedAt time.Time
	types.Item
}
