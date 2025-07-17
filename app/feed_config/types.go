package feed_config

import (
	"time"
)

// FeedConfig represents a complete feed configuration
type FeedConfig struct {
	Feed     FeedInfo     `yaml:"feed"`
	Settings FeedSettings `yaml:"settings"`
	Filters  []Filter     `yaml:"filters"`
}

// FeedInfo contains basic feed information
type FeedInfo struct {
	ID    string `yaml:"id"`
	URL   string `yaml:"url"`
	Title string `yaml:"title"`
}

// FeedSettings contains feed processing settings
type FeedSettings struct {
	Enabled            bool `yaml:"enabled"`
	RefreshInterval    int  `yaml:"refresh_interval"`    // seconds
	MaxItems           int  `yaml:"max_items"`
	Timeout            int  `yaml:"timeout"`             // seconds
	ExtractContent     bool `yaml:"extract_content"`     // enable content extraction
	ExtractionTimeout  int  `yaml:"extraction_timeout"`  // seconds
}

// Filter represents a content filter rule
type Filter struct {
	Field    string   `yaml:"field"`
	Includes []string `yaml:"includes"`
	Excludes []string `yaml:"excludes"`
}

// GetRefreshInterval returns the refresh interval as time.Duration
func (s *FeedSettings) GetRefreshInterval() time.Duration {
	if s.RefreshInterval <= 0 {
		return 3600 * time.Second // default 1 hour
	}
	return time.Duration(s.RefreshInterval) * time.Second
}

// GetTimeout returns the timeout as time.Duration
func (s *FeedSettings) GetTimeout() time.Duration {
	if s.Timeout <= 0 {
		return 30 * time.Second // default 30 seconds
	}
	return time.Duration(s.Timeout) * time.Second
}

// GetExtractionTimeout returns the extraction timeout as time.Duration
func (s *FeedSettings) GetExtractionTimeout() time.Duration {
	if s.ExtractionTimeout <= 0 {
		return 10 * time.Second // default 10 seconds
	}
	return time.Duration(s.ExtractionTimeout) * time.Second
}
