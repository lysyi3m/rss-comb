package database

import (
	"time"
)

// FeedItem represents a normalized feed item for database operations
type FeedItem struct {
	GUID        string
	Title       string
	Link        string
	Description string
	Content     string
	PublishedAt *time.Time
	UpdatedAt   *time.Time
	Authors     []string // Multiple authors in format "email (name)" or "name"
	Categories  []string
	
	ContentHash  string
	IsFiltered   bool
	FilterReason string
}

// FeedRepository defines all feed-related database operations.
// Used by components that need to interact with feed data.
type FeedRepository interface {
	// Read operations
	GetFeedByID(feedID string) (*Feed, error)
	GetFeedCount() (int, error)
	GetEnabledFeedCount() (int, error)
	
	// Write operations
	UpsertFeedWithChangeDetection(configFile, feedID, feedURL, feedTitle string) (string, bool, error)
	UpdateFeedMetadata(feedID string, link string, imageURL string, language string, feedPublishedAt *time.Time) error
	SetFeedEnabled(feedID string, enabled bool) error
	
	// Scheduling operations
	GetFeedsDueForRefresh() ([]Feed, error)
	UpdateNextFetch(feedID string, nextFetch time.Time) error
}

// ItemForExtraction represents minimal data needed for content extraction
type ItemForExtraction struct {
	ID   string
	Link string
}

// ItemRepository defines all item-related database operations.
// Used by components that need to interact with feed item data.
type ItemRepository interface {
	// Read operations
	GetVisibleItems(feedID string, limit int) ([]Item, error)
	GetAllItems(feedID string) ([]Item, error)
	GetItemCount(feedID string) (int, error)
	GetItemStats(feedID string) (int, int, int, error)
	
	// Write operations
	StoreItem(feedID string, item FeedItem) error
	UpdateItemFilterStatus(itemID string, isFiltered bool, reason string) error
	
	// Duplicate checking operations
	CheckDuplicate(contentHash, feedID string) (bool, *string, error)
	
	// Content extraction operations
	GetItemsForExtraction(feedID string, limit int) ([]ItemForExtraction, error)
	UpdateExtractionStatus(itemID string, status string, extractedAt *time.Time, error string) error
	IncrementExtractionAttempts(itemID string) error
	UpdateExtractedContent(itemID string, content string) error
}

