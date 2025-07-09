package config

import (
	"time"
)

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

