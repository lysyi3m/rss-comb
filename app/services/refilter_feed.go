package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/feed"
)

func RefilterFeed(
	ctx context.Context,
	feedName string,
	feedRepo *database.FeedRepository,
	itemRepo *database.ItemRepository,
	filterer *feed.Filterer,
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

	filters, err := dbFeed.GetFilters()
	if err != nil {
		return fmt.Errorf("failed to get feed filters: %w", err)
	}

	items, err := itemRepo.GetAllItems(feedName)
	if err != nil {
		return fmt.Errorf("failed to get feed items: %w", err)
	}

	feedItems := make([]feed.Item, len(items))
	for i, item := range items {
		feedItems[i] = feed.Item{
			GUID:        item.GUID,
			Title:       item.Title,
			Link:        item.Link,
			Description: item.Description,
			Content:     item.Content,
			PublishedAt: item.PublishedAt,
			UpdatedAt:   item.UpdatedAt,
			Authors:     item.Authors,
			Categories:  item.Categories,
			ContentHash: item.ContentHash,
		}
	}

	filteredItems := filterer.Run(feedItems, filters)

	updatedCount := 0
	errorCount := 0

	for i, filteredItem := range filteredItems {
		originalItem := items[i]

		if originalItem.IsFiltered != filteredItem.IsFiltered {
			err := itemRepo.UpdateItemFilterStatus(originalItem.ID, filteredItem.IsFiltered)
			if err != nil {
				slog.Error("Failed to update item filter status", "item_id", originalItem.ID, "error", err)
				errorCount++
			} else {
				updatedCount++
			}
		}
	}

	slog.Info("Feed refiltered",
		"feed", feedName,
		"duration", time.Since(start),
		"success", updatedCount,
		"errors", errorCount)

	return nil
}
