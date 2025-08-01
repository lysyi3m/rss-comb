package tasks

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/feed"
)

type ProcessFeedTask struct {
	Task
	FeedConfig *feed.Config
	httpClient *http.Client
	parser     *feed.Parser
	filterer   *feed.Filterer
	feedRepo   database.FeedRepository
	itemRepo   database.ItemRepository
	userAgent  string
}

func NewProcessFeedTask(feedName string, feedConfig *feed.Config, httpClient *http.Client, parser *feed.Parser, filterer *feed.Filterer, feedRepo database.FeedRepository, itemRepo database.ItemRepository, userAgent string) *ProcessFeedTask {
	return &ProcessFeedTask{
		Task:       NewTask(TaskTypeProcessFeed, feedName),
		FeedConfig: feedConfig,
		httpClient: httpClient,
		parser:     parser,
		filterer:   filterer,
		feedRepo:   feedRepo,
		itemRepo:   itemRepo,
		userAgent:  userAgent,
	}
}

func (t *ProcessFeedTask) Execute(ctx context.Context) error {

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if !t.FeedConfig.Settings.Enabled {
		slog.Debug("Feed disabled, skipping", "feed", t.FeedName)
		return nil
	}

	data, err := t.fetchFeed(ctx, t.FeedConfig.URL)
	if err != nil {
		return fmt.Errorf("failed to fetch feed: %w", err)
	}

	metadata, items, err := t.parser.Run(data)
	if err != nil {
		return fmt.Errorf("failed to parse feed: %w", err)
	}

	err = t.storeFeedMetadata(ctx, metadata)
	if err != nil {
		return fmt.Errorf("failed to store feed metadata: %w", err)
	}

	duplicateCount := 0
	filteredCount := 0
	newCount := 0

	if len(items) > 0 {
		var nonDuplicateItems []feed.Item
		for _, item := range items {
			isDuplicate, _, err := t.itemRepo.CheckDuplicate(t.FeedName, item.ContentHash)
			if err != nil {
				return fmt.Errorf("failed to check for duplicates: %w", err)
			}

			if isDuplicate {
				duplicateCount++
			} else {
				nonDuplicateItems = append(nonDuplicateItems, item)
			}
		}

		if len(nonDuplicateItems) > 0 {
			filteredItems := t.filterer.Run(nonDuplicateItems, t.FeedConfig)

			for _, item := range filteredItems {
				if item.IsFiltered {
					filteredCount++
				} else {
          newCount++
        }
			}

			err = t.storeFilteredItems(ctx, filteredItems)
			if err != nil {
				return fmt.Errorf("failed to store items: %w", err)
			}
		}
	}

  slog.Info("Task completed",
    "type", "ProcessedFeed",
    "feed", t.FeedName,
    "duration", t.GetDuration(),
    "total", len(items),
		"duplicates", duplicateCount,
		"filtered", filteredCount,
    "new", newCount)

	return nil
}

func (t *ProcessFeedTask) fetchFeed(ctx context.Context, url string) ([]byte, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(t.FeedConfig.Settings.Timeout)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(timeoutCtx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", t.userAgent)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}

func (t *ProcessFeedTask) storeFeedMetadata(ctx context.Context, metadata *feed.Metadata) error {
	now := time.Now().UTC()
	nextFetch := now.Add(time.Duration(t.FeedConfig.Settings.RefreshInterval) * time.Second)

	err := t.feedRepo.UpdateFeedMetadata(t.FeedName, metadata.Title, metadata.Link, metadata.Description, metadata.ImageURL, metadata.Language, metadata.FeedPublishedAt, nextFetch)
	if err != nil {
		return fmt.Errorf("failed to update feed metadata and next fetch time: %w", err)
	}

	return nil
}

func (t *ProcessFeedTask) storeFilteredItems(ctx context.Context, items []feed.Item) error {
	for _, item := range items {
		dbItem := database.FeedItem{
			GUID:            item.GUID,
			Link:            item.Link,
			Title:           item.Title,
			Description:     item.Description,
			Content:         item.Content,
			PublishedAt:     item.PublishedAt,
			UpdatedAt:       item.UpdatedAt,
			Authors:         item.Authors,
			Categories:      item.Categories,
			IsFiltered:      item.IsFiltered,
			ContentHash:     item.ContentHash,
			EnclosureURL:    item.EnclosureURL,
			EnclosureLength: item.EnclosureLength,
			EnclosureType:   item.EnclosureType,
		}

		err := t.itemRepo.UpsertItem(t.FeedName, dbItem)
		if err != nil {
			return fmt.Errorf("failed to upsert item: %w", err)
		}
	}

	return nil
}
