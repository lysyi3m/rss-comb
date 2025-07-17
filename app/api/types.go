package api

import (
	"github.com/lysyi3m/rss-comb/app/feed_config"
	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/tasks"
)

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
