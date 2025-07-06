package database

import "time"

// FeedRepositoryInterface defines the interface for feed repository operations
type FeedRepositoryInterface interface {
	GetFeedsDueForRefresh() ([]Feed, error)
	UpsertFeed(configFile, feedID, feedURL, feedName string) (string, error)
	UpdateFeedMetadata(feedID string, iconURL string, language string) error
	UpdateNextFetch(feedID string, nextFetch time.Time) error
	GetFeedByConfigFile(configFile string) (*Feed, error)
	GetFeedByURL(feedURL string) (*Feed, error)
	GetFeedByID(feedID string) (*Feed, error)
	SetFeedActive(feedID string, active bool) error
	GetFeedCount() (int, error)
}