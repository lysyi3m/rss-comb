package database

import "time"

// FeedRepositoryInterface defines the interface for feed repository operations
type FeedRepositoryInterface interface {
	GetFeedsDueForRefresh() ([]Feed, error)
	UpsertFeed(configFile, feedID, feedURL, feedTitle string) (string, error)
	UpsertFeedWithChangeDetection(configFile, feedID, feedURL, feedTitle string) (string, bool, error)
	UpdateFeedMetadata(feedID string, iconURL string, language string) error
	UpdateNextFetch(feedID string, nextFetch time.Time) error
	GetFeedByConfigFile(configFile string) (*Feed, error)
	GetFeedByURL(feedURL string) (*Feed, error)
	GetFeedByID(feedID string) (*Feed, error)
	SetFeedEnabled(feedID string, enabled bool) error
	GetFeedCount() (int, error)
	GetEnabledFeedCount() (int, error)
}