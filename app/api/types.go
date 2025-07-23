package api

import (
	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/feed"
	"github.com/lysyi3m/rss-comb/app/tasks"
)

type GeneratorInterface interface {
	Run(feed database.Feed, items []database.Item) (string, error)
}

var _ GeneratorInterface = (*feed.Generator)(nil)

type Handler struct {
	feedRepo    database.FeedRepository
	itemRepo    database.ItemRepository
	generator   GeneratorInterface
	configCache *feed.ConfigCache
	filterer    *feed.Filterer
	scheduler   tasks.TaskSchedulerInterface
}
