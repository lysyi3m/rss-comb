package jobs

import (
	"context"
	"fmt"
	"net/http"

	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/services"
)

// FetchFeedHandler returns a HandlerFunc that processes a feed by resolving
// the feed name from the job's FeedID and calling services.ProcessFeed.
func FetchFeedHandler(
	feedRepo *database.FeedRepository,
	itemRepo *database.ItemRepository,
	httpClient *http.Client,
	userAgent string,
) HandlerFunc {
	return func(ctx context.Context, job *database.Job) error {
		feed, err := feedRepo.GetFeedByID(job.FeedID)
		if err != nil {
			return fmt.Errorf("failed to get feed by ID: %w", err)
		}
		if feed == nil {
			return fmt.Errorf("feed not found for ID: %s", job.FeedID)
		}

		return services.ProcessFeed(ctx, feed.Name, feedRepo, itemRepo, httpClient, userAgent)
	}
}
