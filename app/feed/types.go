package feed

import (
	"net/http"

	"github.com/lysyi3m/rss-comb/app/config_sync"
	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/parser"
)

// Processor handles feed processing including fetching, parsing, filtering, and storage
type Processor struct {
	parser      *parser.Parser
	feedRepo    *database.FeedRepository
	itemRepo    *database.ItemRepository
	configCache *config_sync.ConfigCacheHandler
	client      *http.Client
	userAgent   string
}