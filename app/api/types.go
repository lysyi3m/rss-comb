package api

import (
	"github.com/lysyi3m/rss-comb/app/cfg"
	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/feed"
)

type Handler struct {
	cfg      *cfg.Cfg
	feedRepo *database.FeedRepository
	itemRepo *database.ItemRepository
	filterer *feed.Filterer
}
