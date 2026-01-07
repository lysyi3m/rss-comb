package api

import (
	"github.com/lysyi3m/rss-comb/app/cfg"
	"github.com/lysyi3m/rss-comb/app/database"
)

type Handler struct {
	cfg      *cfg.Cfg
	feedRepo *database.FeedRepository
	itemRepo *database.ItemRepository
}
