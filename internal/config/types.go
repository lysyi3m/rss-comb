package config

import "time"

// FeedConfig represents a complete feed configuration
type FeedConfig struct {
	Feed     FeedInfo     `yaml:"feed"`
	Settings FeedSettings `yaml:"settings"`
	Filters  []Filter     `yaml:"filters"`
}

// FeedInfo contains basic feed information
type FeedInfo struct {
	URL  string `yaml:"url"`
	Name string `yaml:"name"`
}

// FeedSettings contains feed processing settings
type FeedSettings struct {
	Enabled         bool          `yaml:"enabled"`
	Deduplication   bool          `yaml:"deduplication"`
	RefreshInterval int           `yaml:"refresh_interval"` // seconds
	CacheDuration   int           `yaml:"cache_duration"`   // seconds
	MaxItems        int           `yaml:"max_items"`
	Timeout         int           `yaml:"timeout"`          // seconds
	UserAgent       string        `yaml:"user_agent"`
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

// GetCacheDuration returns the cache duration as time.Duration
func (s *FeedSettings) GetCacheDuration() time.Duration {
	if s.CacheDuration <= 0 {
		return 300 * time.Second // default 5 minutes
	}
	return time.Duration(s.CacheDuration) * time.Second
}

// GetTimeout returns the timeout as time.Duration
func (s *FeedSettings) GetTimeout() time.Duration {
	if s.Timeout <= 0 {
		return 30 * time.Second // default 30 seconds
	}
	return time.Duration(s.Timeout) * time.Second
}