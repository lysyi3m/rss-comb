package config_sync

import (
	"github.com/lysyi3m/rss-comb/app/database"
)

// FeedSyncRepository combines feed repository operations needed by the config sync package.
// Defined here as the config_sync package is the consumer of this interface.
// Used by database sync handler for feed configuration synchronization.
type FeedSyncRepository interface {
	database.FeedReader
	database.FeedWriter
	database.FeedScheduler
}

// Compile-time interface compliance check
// Ensures that the database implementation satisfies the interface defined here
var _ FeedSyncRepository = (*database.FeedRepository)(nil)
