package feed

import (
	"github.com/lysyi3m/rss-comb/app/database"
)

// FeedRepositoryInterface combines feed repository operations needed by the feed package.
// Defined here as the feed package is the primary consumer of this interface.
// Used by feed processor for comprehensive feed management including reads, writes, and scheduling.
type FeedRepositoryInterface interface {
	database.FeedReader
	database.FeedWriter
	database.FeedScheduler
}

// ItemRepositoryInterface combines item repository operations needed by the feed package.
// Defined here as the feed package is the primary consumer of this interface.
// Used by feed processor for comprehensive item management including reads, writes, and automatic deduplication.
type ItemRepositoryInterface interface {
	database.ItemReader
	database.ItemWriter
	database.ItemDuplicateChecker
	database.ItemContentExtractor
}

// Compile-time interface compliance checks
// Ensures that the database implementations satisfy the interfaces defined here
var _ FeedRepositoryInterface = (*database.FeedRepository)(nil)
var _ ItemRepositoryInterface = (*database.ItemRepository)(nil)
