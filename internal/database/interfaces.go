package database

import "time"

// FeedRepositoryInterface defines the interface for feed repository operations
type FeedRepositoryInterface interface {
	GetFeedsDueForRefresh() ([]Feed, error)
	UpsertFeed(configFile, feedURL, feedName string) (string, error)
	UpdateFeedMetadata(feedID string, iconURL string) error
	UpdateNextFetch(feedID string, nextFetch time.Time) error
	GetFeedByConfigFile(configFile string) (*Feed, error)
	GetFeedByURL(feedURL string) (*Feed, error)
	SetFeedActive(feedID string, active bool) error
	GetFeedCount() (int, error)
}