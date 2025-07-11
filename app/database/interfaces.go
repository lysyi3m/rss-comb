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

// FeedReader defines read operations for feeds.
// Used by components that only need to read feed information, such as API handlers.
// This interface provides access to feed metadata and counts without write permissions.
type FeedReader interface {
	GetFeedByID(feedID string) (*Feed, error)
	GetFeedByConfigFile(configFile string) (*Feed, error)
	GetFeedByURL(feedURL string) (*Feed, error)
	GetFeedCount() (int, error)
	GetEnabledFeedCount() (int, error)
}

// FeedWriter defines write operations for feeds.
// Used by components that need to create, update, or modify feed records.
// This interface provides full write access to feed metadata and configuration.
type FeedWriter interface {
	UpsertFeed(configFile, feedID, feedURL, feedTitle string) (string, error)
	UpsertFeedWithChangeDetection(configFile, feedID, feedURL, feedTitle string) (string, bool, error)
	UpdateFeedMetadata(feedID string, link string, imageURL string, language string) error
	SetFeedEnabled(feedID string, enabled bool) error
}

// FeedScheduler defines scheduling operations for feeds.
// Used by the task scheduler to manage feed processing timing.
// This interface provides access to scheduling-related operations.
type FeedScheduler interface {
	GetFeedsDueForRefresh() ([]Feed, error)
	UpdateNextFetch(feedID string, nextFetch time.Time) error
}

// FeedRepositoryInterface combines all feed repository operations.
// This interface is kept for backward compatibility and components that need all operations.
// Use specific interfaces (FeedReader, FeedWriter, FeedScheduler) when possible for better separation of concerns.
type FeedRepositoryInterface interface {
	FeedReader
	FeedWriter
	FeedScheduler
}


// ItemReader defines read operations for feed items.
// Used by components that need to display or serve feed items, such as API handlers.
// This interface provides read-only access to feed item data.
type ItemReader interface {
	GetVisibleItems(feedID string, limit int) ([]Item, error)
	GetAllItems(feedID string) ([]Item, error)
	GetItemCount(feedID string) (int, error)
	GetItemStats(feedID string) (int, int, int, error)
}

// ItemWriter defines write operations for feed items.
// Used by components that need to store or modify feed items.
// This interface provides write access to feed item data.
type ItemWriter interface {
	StoreItem(feedID string, item FeedItem) error
	UpdateItemFilterStatus(itemID string, isFiltered bool, reason string) error
}

// ItemDuplicateChecker defines duplicate checking operations for feed items.
// Used by components that need to check for duplicate content before storage.
// This interface provides specialized deduplication functionality within a feed.
type ItemDuplicateChecker interface {
	CheckDuplicate(contentHash, feedID string) (bool, *string, error)
}

// ItemRepositoryInterface combines all item repository operations.
// This interface is kept for backward compatibility and components that need all operations.
// Use specific interfaces (ItemReader, ItemWriter, ItemDuplicateChecker) when possible for better separation of concerns.
type ItemRepositoryInterface interface {
	ItemReader
	ItemWriter
	ItemDuplicateChecker
}