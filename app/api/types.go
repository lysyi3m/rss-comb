package api

import (
	"github.com/lysyi3m/rss-comb/app/feed_config"
	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/feed"
	"github.com/lysyi3m/rss-comb/app/tasks"
)

// GeneratorInterface defines the interface for RSS generation operations.
// Defined here as the api package is the consumer of this interface.
// Used by API handlers to generate RSS 2.0 XML output from database feed data.
// This interface provides RSS generation and formatting functionality.
type GeneratorInterface interface {
	Generate(feed database.Feed, items []database.Item) (string, error)
}

// Compile-time interface compliance check
// Ensures that the feed.Generator implementation satisfies the interface defined here
var _ GeneratorInterface = (*feed.Generator)(nil)

// Handler handles HTTP requests for the RSS API
type Handler struct {
	feedRepo    database.FeedRepository
	itemRepo    database.ItemRepository
	generator   GeneratorInterface
	configCache *feed_config.ConfigCacheHandler
	processor   tasks.ProcessorInterface
	scheduler   tasks.TaskSchedulerInterface
	userAgent   string
}
