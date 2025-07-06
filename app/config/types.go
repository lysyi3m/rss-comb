package config

import (
	"os"
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
	ID   string `yaml:"id"`
	URL  string `yaml:"url"`
	Name string `yaml:"name"`
}

// FeedSettings contains feed processing settings
type FeedSettings struct {
	Enabled         bool          `yaml:"enabled"`
	Deduplication   bool          `yaml:"deduplication"`
	RefreshInterval int           `yaml:"refresh_interval"` // seconds
	MaxItems        int           `yaml:"max_items"`
	Timeout         int           `yaml:"timeout"`          // seconds
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

// GetUserAgent returns the global user agent from environment or default
func GetUserAgent() string {
	if userAgent := os.Getenv("USER_AGENT"); userAgent != "" {
		return userAgent
	}
	return "RSS Comb/1.0"
}