package database

import (
	"time"
)

type FeedItem struct {
	GUID        string
	Title       string
	Link        string
	Description string
	Content     string
	PublishedAt time.Time // Changed from *time.Time to time.Time (NOT NULL)
	UpdatedAt   *time.Time
	Authors     []string // Multiple authors in format "email (name)" or "name"
	Categories  []string

	ContentHash     string
	IsFiltered      bool
	FilterReason    string
	EnclosureURL    string // RSS enclosure URL
	EnclosureLength int64  // RSS enclosure length in bytes
	EnclosureType   string // RSS enclosure MIME type
}

type FeedRepository interface {
	GetFeed(feedName string) (*Feed, error)
	GetFeedCount() (int, error)

	UpsertFeed(feedName, feedURL string) error
	UpdateFeedMetadata(feedName string, title string, link string, description string, imageURL string, language string, feedPublishedAt *time.Time, nextFetch time.Time) error
}

type ItemForExtraction struct {
	ID   string
	Link string
}

type ItemRepository interface {
	GetVisibleItems(feedName string, limit int) ([]Item, error)
	GetAllItems(feedName string) ([]Item, error)
	GetItemCount(feedName string) (int, error)
	GetItemStats(feedName string) (int, int, int, error)

	UpsertItem(feedName string, item FeedItem) error
	UpdateItemFilterStatus(itemID string, isFiltered bool, reason string) error

	CheckDuplicate(contentHash, feedName string) (bool, *string, error)

	GetItemsForExtraction(feedName string, limit int) ([]ItemForExtraction, error)
	UpdateExtractionStatus(itemID string, status string, extractedAt *time.Time, error string) error
	UpdateExtractedContentAndStatus(itemID string, content string, status string, extractedAt *time.Time, errorMsg string) error
}
