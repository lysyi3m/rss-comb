package config

// FeedConfig represents a complete feed configuration
type FeedConfig struct {
	Feed     FeedInfo     `yaml:"feed"`
	Settings FeedSettings `yaml:"settings"`
	Filters  []Filter     `yaml:"filters"`
}

// FeedInfo contains basic feed information
type FeedInfo struct {
	ID   string `yaml:"id"`
	URL  string `yaml:"url"`
	Name string `yaml:"name"`
}

// FeedSettings contains feed processing settings
type FeedSettings struct {
	Enabled         bool `yaml:"enabled"`
	Deduplication   bool `yaml:"deduplication"`
	RefreshInterval int  `yaml:"refresh_interval"` // seconds
	MaxItems        int  `yaml:"max_items"`
	Timeout         int  `yaml:"timeout"` // seconds
}

// Filter represents a content filter rule
type Filter struct {
	Field    string   `yaml:"field"`
	Includes []string `yaml:"includes"`
	Excludes []string `yaml:"excludes"`
}