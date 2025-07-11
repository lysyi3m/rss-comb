package database

import (
	"time"
)

// FeedItem represents a normalized feed item for database operations
type FeedItem struct {
	GUID          string
	Title         string
	Link          string
	Description   string
	Content       string
	PublishedDate *time.Time
	UpdatedDate   *time.Time
	AuthorName    string
	AuthorEmail   string
	Categories    []string
	
	ContentHash   string
	IsFiltered    bool
	FilterReason  string
}

// FeedReader defines read operations for feeds
type FeedReader interface {
	GetFeedByID(feedID string) (*Feed, error)
	GetFeedByConfigFile(configFile string) (*Feed, error)
	GetFeedByURL(feedURL string) (*Feed, error)
	GetFeedCount() (int, error)
	GetEnabledFeedCount() (int, error)
}

// FeedWriter defines write operations for feeds
type FeedWriter interface {
	UpsertFeed(configFile, feedID, feedURL, feedTitle string) (string, error)
	UpsertFeedWithChangeDetection(configFile, feedID, feedURL, feedTitle string) (string, bool, error)
	UpdateFeedMetadata(feedID string, link string, iconURL string, language string) error
	SetFeedEnabled(feedID string, enabled bool) error
}

// FeedScheduler defines scheduling operations for feeds
type FeedScheduler interface {
	GetFeedsDueForRefresh() ([]Feed, error)
	UpdateNextFetch(feedID string, nextFetch time.Time) error
}

// FeedRepositoryInterface combines all feed repository operations
// This interface is kept for backward compatibility and components that need all operations
type FeedRepositoryInterface interface {
	FeedReader
	FeedWriter
	FeedScheduler
}

// FeedManager combines read, write, and scheduling operations (commonly used together)
type FeedManager interface {
	FeedReader
	FeedWriter
	FeedScheduler
}

// ItemReader defines read operations for feed items
type ItemReader interface {
	GetVisibleItems(feedID string, limit int) ([]Item, error)
	GetAllItems(feedID string) ([]Item, error)
	GetItemCount(feedID string) (int, error)
	GetItemStats(feedID string) (int, int, int, error)
}

// ItemWriter defines write operations for feed items
type ItemWriter interface {
	StoreItem(feedID string, item FeedItem) error
	UpdateItemFilterStatus(itemID string, isFiltered bool, reason string) error
}

// ItemDuplicateChecker defines duplicate checking operations for feed items
type ItemDuplicateChecker interface {
	CheckDuplicate(contentHash, feedID string, excludeFiltered bool) (bool, *string, error)
}

// ItemRepositoryInterface combines all item repository operations
type ItemRepositoryInterface interface {
	ItemReader
	ItemWriter
	ItemDuplicateChecker
}