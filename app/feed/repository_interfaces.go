package feed

import (
	"github.com/lysyi3m/rss-comb/app/database"
)

// FeedRepositoryInterface is an alias for the unified feed repository interface
type FeedRepositoryInterface = database.FeedRepository

// ItemRepositoryInterface is an alias for the unified item repository interface
type ItemRepositoryInterface = database.ItemRepository
