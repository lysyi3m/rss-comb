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
	CreatedAt       time.Time
	UpdatedAt       time.Time // Tracks last successful processing (replaces last_success)

	// Configuration fields
	FeedType   string          // Feed type: "", "podcast", "youtube"
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

// jsonStringSlice implements sql.Scanner for reading JSON arrays from TEXT columns.
type jsonStringSlice struct {
	dest *[]string
}

func JSONStringSlice(dest *[]string) *jsonStringSlice {
	return &jsonStringSlice{dest: dest}
}

func (j *jsonStringSlice) Scan(src interface{}) error {
	if src == nil {
		*j.dest = []string{}
		return nil
	}

	var data []byte
	switch v := src.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return fmt.Errorf("jsonStringSlice: unsupported type %T", src)
	}

	if len(data) == 0 {
		*j.dest = []string{}
		return nil
	}

	return json.Unmarshal(data, j.dest)
}

func encodeStringSlice(s []string) string {
	if s == nil {
		return "[]"
	}
	b, _ := json.Marshal(s)
	return string(b)
}

// sqliteTimeFmt is the canonical timestamp format matching SQLite's datetime() output.
const sqliteTimeFmt = "2006-01-02 15:04:05"

// sqliteTime formats a time.Time for SQLite TEXT storage, matching datetime('now') format.
func sqliteTime(t time.Time) string {
	return t.UTC().Format(sqliteTimeFmt)
}

// sqliteTimePtr formats a *time.Time, returning nil for nil input.
func sqliteTimePtr(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return sqliteTime(*t)
}
