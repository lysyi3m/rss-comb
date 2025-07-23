package tasks

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/feed"
)

type RefilterFeedTask struct {
	Task
	FeedConfig *feed.Config
	filterer   *feed.Filterer
	feedRepo   database.FeedRepository
	itemRepo   database.ItemRepository
}

func NewRefilterFeedTask(feedName string, feedConfig *feed.Config, filterer *feed.Filterer, feedRepo database.FeedRepository, itemRepo database.ItemRepository) *RefilterFeedTask {
	return &RefilterFeedTask{
		Task:       NewTask(TaskTypeRefilterFeed, feedName),
		FeedConfig: feedConfig,
		filterer:   filterer,
		feedRepo:   feedRepo,
		itemRepo:   itemRepo,
	}
}

func (t *RefilterFeedTask) Execute(ctx context.Context) error {

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}


	items, err := t.itemRepo.GetAllItems(t.FeedName)
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

	filteredItems := t.filterer.Run(feedItems, t.FeedConfig)

	updatedCount := 0
	errorCount := 0

	for i, filteredItem := range filteredItems {
		originalItem := items[i]

		if originalItem.IsFiltered != filteredItem.IsFiltered || originalItem.FilterReason != filteredItem.FilterReason {
			err := t.itemRepo.UpdateItemFilterStatus(originalItem.ID, filteredItem.IsFiltered, filteredItem.FilterReason)
			if err != nil {
				slog.Error("Failed to update item filter status", "item_id", originalItem.ID, "error", err)
				errorCount++
			} else {
				updatedCount++
			}
		}
	}

  slog.Info("Task completed",
    "type", "RefilterFeed",
    "feed", t.FeedName,
    "duration", t.GetDuration(),
    "success", updatedCount,
    "errors", errorCount)

	return nil
}
