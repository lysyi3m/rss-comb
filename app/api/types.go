package api

import (
	"github.com/lysyi3m/rss-comb/app/config_sync"
	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/feed"
	"github.com/lysyi3m/rss-comb/app/generator"
	"github.com/lysyi3m/rss-comb/app/tasks"
)

// Handler handles HTTP requests for the RSS API
type Handler struct {
	feedRepo     database.FeedRepositoryInterface
	itemRepo     *database.ItemRepository
	generator    *generator.RSSGenerator
	configCache  *config_sync.ConfigCacheHandler
	processor    feed.FeedProcessor
	scheduler    *tasks.TaskScheduler
	userAgent    string
}