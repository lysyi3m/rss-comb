package feed

import (
	"github.com/lysyi3m/rss-comb/app/cfg"
	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/types"
)

type FeedType interface {
	Parse(data []byte) (*Metadata, []types.Item, error)
	Build(feed database.Feed, items []database.Item, cfg *cfg.Cfg) (string, error)
}

func ForType(typ string) FeedType {
	switch typ {
	case "youtube":
		return youtubeType{}
	case "podcast":
		return podcastType{}
	default:
		return basicType{}
	}
}
