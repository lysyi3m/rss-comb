package api

import (
	"github.com/lysyi3m/rss-comb/app/config_sync"
	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/feed"
	"github.com/lysyi3m/rss-comb/app/tasks"
)

// Handler handles HTTP requests for the RSS API
type Handler struct {
	feedRepo     database.FeedReader
	itemRepo     database.ItemReader
	generator    feed.GeneratorInterface
	configCache  *config_sync.ConfigCacheHandler
	processor    feed.ProcessorInterface
	scheduler    tasks.TaskSchedulerInterface
	userAgent    string
}