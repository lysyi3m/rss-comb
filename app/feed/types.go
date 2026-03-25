package feed

import "github.com/lysyi3m/rss-comb/app/types"

type Metadata = types.Metadata

type Config struct {
	Name     string         // Derived from filename (without .yml extension)
	URL      string         `yaml:"url"`
	Title    string         `yaml:"title"`
	Type     string         `yaml:"type"`
	Enabled  bool           `yaml:"enabled"`
	Settings types.Settings `yaml:"settings"`
	Filters  []types.Filter `yaml:"filters"`
}
