package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/feed"
	"github.com/lysyi3m/rss-comb/app/services"
)

// FetchFeedHandler returns a HandlerFunc that processes a feed by resolving
// the feed name from the job's FeedID and calling services.ProcessFeed.
func FetchFeedHandler(
	feedRepo *database.FeedRepository,
	itemRepo *database.ItemRepository,
	jobRepo *database.JobRepository,
	httpClient *http.Client,
	userAgent string,
) HandlerFunc {
	return func(ctx context.Context, job *database.Job) error {
		dbFeed, err := feedRepo.GetFeedByID(job.FeedID)
		if err != nil {
			return fmt.Errorf("failed to get feed by ID: %w", err)
		}
		if dbFeed == nil {
			return fmt.Errorf("feed not found for ID: %s", job.FeedID)
		}

		return services.ProcessFeed(ctx, dbFeed.Name, feedRepo, itemRepo, jobRepo, httpClient, userAgent)
	}
}

// ExtractContentHandler returns a HandlerFunc that fetches HTML content
// from an item's link and extracts clean text using go-readability.
func ExtractContentHandler(
	feedRepo *database.FeedRepository,
	itemRepo *database.ItemRepository,
	httpClient *http.Client,
	userAgent string,
) HandlerFunc {
	return func(ctx context.Context, job *database.Job) error {
		if job.ItemID == nil {
			return fmt.Errorf("extract_content job has no item_id")
		}

		item, err := itemRepo.GetItemByID(*job.ItemID)
		if err != nil {
			return fmt.Errorf("failed to get item: %w", err)
		}
		if item == nil {
			return fmt.Errorf("item not found for ID: %s", *job.ItemID)
		}

		dbFeed, err := feedRepo.GetFeedByID(job.FeedID)
		if err != nil {
			return fmt.Errorf("failed to get feed: %w", err)
		}
		if dbFeed == nil {
			return fmt.Errorf("feed not found for ID: %s", job.FeedID)
		}

		settings, err := dbFeed.GetSettings()
		if err != nil {
			return fmt.Errorf("failed to get feed settings: %w", err)
		}

		if item.Link == "" {
			return handleExtractionFailure(itemRepo, *job.ItemID, job, fmt.Errorf("item has no link"))
		}

		data, err := services.Fetch(ctx, item.Link, settings.Timeout, httpClient, userAgent, true)
		if err != nil {
			return handleExtractionFailure(itemRepo, *job.ItemID, job, err)
		}

		extractedContent, err := feed.Extract(data)
		if err != nil {
			return handleExtractionFailure(itemRepo, *job.ItemID, job, err)
		}

		if err := itemRepo.UpdateContentExtractionStatus(*job.ItemID, "ready", extractedContent); err != nil {
			return fmt.Errorf("failed to update extraction status: %w", err)
		}

		slog.Info("Content extracted successfully", "item_id", *job.ItemID, "feed_id", job.FeedID)
		return nil
	}
}

// handleExtractionFailure checks if this is the last retry attempt.
// On final failure, marks the item as 'failed' and returns nil (job completes).
// Otherwise returns the error so the job will be retried.
func handleExtractionFailure(itemRepo *database.ItemRepository, itemID string, job *database.Job, extractionErr error) error {
	if job.Retries >= job.MaxRetries-1 {
		slog.Warn("Content extraction permanently failed, item will use original content",
			"item_id", itemID, "error", extractionErr, "retries", job.Retries+1)
		if err := itemRepo.UpdateContentExtractionStatus(itemID, "failed", ""); err != nil {
			slog.Error("Failed to mark item extraction as failed", "item_id", itemID, "error", err)
		}
		return nil
	}
	return fmt.Errorf("content extraction failed: %w", extractionErr)
}
