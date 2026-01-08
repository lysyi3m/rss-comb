package feed

import "github.com/lysyi3m/rss-comb/app/types"

// Type alias for feed metadata (actual definition in app/types/feed.go)
type Metadata = types.Metadata

// Configuration types

type Config struct {
	Name     string         // Derived from filename (without .yml extension)
	URL      string         `yaml:"url"`
	Enabled  bool           `yaml:"enabled"`
	Settings types.Settings `yaml:"settings"`
	Filters  []types.Filter `yaml:"filters"`
}
