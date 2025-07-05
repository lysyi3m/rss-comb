package feed

// FeedProcessor defines the interface for feed processing
type FeedProcessor interface {
	ProcessFeed(feedID, configFile string) error
	IsFeedEnabled(configFile string) bool
	ReapplyFilters(feedID, configFile string) (int, int, error)
}