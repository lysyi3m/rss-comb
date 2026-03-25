package jobs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/feed"
	"github.com/lysyi3m/rss-comb/app/types"
)

func processFeed(
	ctx context.Context,
	feedName string,
	feedRepo *database.FeedRepository,
	itemRepo *database.ItemRepository,
	jobRepo *database.JobRepository,
	httpClient *http.Client,
	userAgent string,
) error {
	start := time.Now()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	dbFeed, err := feedRepo.GetFeed(feedName)
	if err != nil {
		return fmt.Errorf("failed to get feed from database: %w", err)
	}
	if dbFeed == nil {
		return fmt.Errorf("feed not found in database")
	}

	if !dbFeed.IsEnabled {
		return nil
	}

	settings, err := dbFeed.GetSettings()
	if err != nil {
		return fmt.Errorf("failed to get feed settings: %w", err)
	}

	filters, err := dbFeed.GetFilters()
	if err != nil {
		return fmt.Errorf("failed to get feed filters: %w", err)
	}

	metadata, items, contentHash, newContentHash, err := fetchAndParseFeed(ctx, feedName, dbFeed.FeedURL, dbFeed.FeedType, settings, feedRepo, httpClient, userAgent)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	nextFetch := now.Add(time.Duration(settings.RefreshInterval) * time.Second)
	err = feedRepo.UpdateFeedMetadata(feedName, metadata, newContentHash, nextFetch)
	if err != nil {
		return fmt.Errorf("failed to update feed metadata: %w", err)
	}

	if contentHash != nil && *contentHash == newContentHash {
		slog.Info("Feed unchanged, skipping item processing",
			"feed", feedName,
			"duration", time.Since(start))
		return nil
	}

	duplicateCount := 0
	filteredCount := 0
	newCount := 0
	extractionJobCount := 0
	mediaJobCount := 0

	if len(items) == 0 {
		slog.Info("No parsed items found, skipping item processing",
			"feed", feedName,
			"duration", time.Since(start))
		return nil
	}

	for _, item := range items {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		isDuplicate, _, err := itemRepo.CheckDuplicate(feedName, item.ContentHash)
		if err != nil {
			return fmt.Errorf("failed to check for duplicates: %w", err)
		}

		if isDuplicate {
			duplicateCount++
			continue
		}

		filteredItems := feed.Filter([]types.Item{item}, filters)
		processedItem := filteredItems[0]

		if processedItem.IsFiltered {
			filteredCount++
		} else {
			newCount++
		}

		if !processedItem.IsFiltered && settings.ExtractContent {
			processedItem.ContentExtractionStatus = stringPtr("pending")
		}

		if !processedItem.IsFiltered && dbFeed.FeedType == "youtube" {
			processedItem.MediaStatus = stringPtr("pending")
		}

		itemID, err := itemRepo.UpsertItem(feedName, processedItem)
		if err != nil {
			return fmt.Errorf("failed to upsert item: %w", err)
		}

		if processedItem.ContentExtractionStatus != nil && *processedItem.ContentExtractionStatus == "pending" {
			if _, err := jobRepo.CreateJob("extract_content", dbFeed.ID, &itemID, 3); err != nil {
				slog.Error("Failed to create extract_content job", "feed", feedName, "item_id", itemID, "error", err)
			} else {
				extractionJobCount++
			}
		}

		if processedItem.MediaStatus != nil && *processedItem.MediaStatus == "pending" {
			if _, err := jobRepo.CreateJob("download_media", dbFeed.ID, &itemID, 3); err != nil {
				slog.Error("Failed to create download_media job", "feed", feedName, "item_id", itemID, "error", err)
			} else {
				mediaJobCount++
			}
		}
	}

	logData := []interface{}{
		"feed", feedName,
		"duration", time.Since(start),
		"total", len(items),
		"duplicates", duplicateCount,
		"filtered", filteredCount,
		"new", newCount,
	}

	if settings.ExtractContent {
		logData = append(logData, "extraction_jobs", extractionJobCount)
	}

	if dbFeed.FeedType == "youtube" {
		logData = append(logData, "media_jobs", mediaJobCount)
	}

	slog.Info("Feed processed", logData...)

	return nil
}

func stringPtr(s string) *string {
	return &s
}

func fetchAndParseFeed(
	ctx context.Context,
	feedName string,
	feedURL string,
	feedType string,
	settings *types.Settings,
	feedRepo *database.FeedRepository,
	httpClient *http.Client,
	userAgent string,
) (*feed.Metadata, []types.Item, *string, string, error) {
	data, err := fetchURL(ctx, feedURL, settings.Timeout, httpClient, userAgent, false)
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to fetch feed: %w", err)
	}

	ft := feed.ForType(feedType)
	metadata, items, err := ft.Parse(data)
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to parse feed: %w", err)
	}

	contentHash, err := feedRepo.GetFeedContentHash(feedName)
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to get stored content hash: %w", err)
	}

	hash := sha256.Sum256(data)
	newContentHash := hex.EncodeToString(hash[:8])

	return metadata, items, contentHash, newContentHash, nil
}
