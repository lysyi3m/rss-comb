package tasks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/feed"
)

type ProcessFeedTask struct {
	Task
	FeedConfig       *feed.Config
	httpClient       *http.Client
	parser           *feed.Parser
	filterer         *feed.Filterer
	contentExtractor *feed.ContentExtractor
	feedRepo         *database.FeedRepository
	itemRepo         *database.ItemRepository
	userAgent        string
}

func NewProcessFeedTask(feedName string, feedConfig *feed.Config, httpClient *http.Client, parser *feed.Parser, filterer *feed.Filterer, contentExtractor *feed.ContentExtractor, feedRepo *database.FeedRepository, itemRepo *database.ItemRepository, userAgent string) *ProcessFeedTask {
	return &ProcessFeedTask{
		Task:             NewTask(TaskTypeProcessFeed, feedName),
		FeedConfig:       feedConfig,
		httpClient:       httpClient,
		parser:           parser,
		filterer:         filterer,
		contentExtractor: contentExtractor,
		feedRepo:         feedRepo,
		itemRepo:         itemRepo,
		userAgent:        userAgent,
	}
}

func (t *ProcessFeedTask) Execute(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if !t.FeedConfig.Settings.Enabled {
		return nil
	}

	metadata, items, contentHash, newContentHash, err := t.fetchAndParseFeed(ctx)
	if err != nil {
		return err
	}

	err = t.storeFeedMetadataWithHash(ctx, metadata, newContentHash)
	if err != nil {
		return fmt.Errorf("failed to store feed metadata with hash: %w", err)
	}

	if contentHash != nil && *contentHash == newContentHash {
		slog.Info("Feed unchanged, skipping item processing",
			"feed", t.FeedName,
			"duration", t.GetDuration())
		return nil
	}

	duplicateCount := 0
	filteredCount := 0
	newCount := 0
	extractionSuccessCount := 0
	extractionFailureCount := 0

	if len(items) == 0 {
		slog.Info("No parsed items found, skipping item processing",
			"feed", t.FeedName,
			"duration", t.GetDuration())
		return nil
	}

	for _, item := range items {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		isDuplicate, _, err := t.itemRepo.CheckDuplicate(t.FeedName, item.ContentHash)
		if err != nil {
			return fmt.Errorf("failed to check for duplicates: %w", err)
		}

		if isDuplicate {
			duplicateCount++
			continue
		}

		filteredItems := t.filterer.Run([]feed.Item{item}, t.FeedConfig)
		processedItem := filteredItems[0] // filterer always returns same number of items

		if processedItem.IsFiltered {
			filteredCount++
		} else {
			newCount++
		}

		if !processedItem.IsFiltered && t.FeedConfig.Settings.ExtractContent {
			extractedContent, extractionErr := t.fetchAndExtractContent(ctx, processedItem)
			if extractionErr != nil {
				slog.Warn("Failed to extract content for item",
					"feed", t.FeedName,
					"item_link", processedItem.Link,
					"error", extractionErr)
				extractionFailureCount++
			} else if extractedContent != "" {
				processedItem.Content = extractedContent
				extractionSuccessCount++
			}
		}

		err = t.storeItem(ctx, processedItem)
		if err != nil {
			return fmt.Errorf("failed to store item: %w", err)
		}
	}

	logData := []interface{}{
		"type", t.GetType(),
		"feed", t.FeedName,
		"duration", t.GetDuration(),
		"total", len(items),
		"duplicates", duplicateCount,
		"filtered", filteredCount,
		"new", newCount,
	}

	if t.FeedConfig.Settings.ExtractContent {
		logData = append(logData, "extraction_success", extractionSuccessCount, "extraction_failure", extractionFailureCount)
	}

	slog.Info("Task completed", logData...)

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

func (t *ProcessFeedTask) fetchAndParseFeed(ctx context.Context) (*feed.Metadata, []feed.Item, *string, string, error) {
	data, err := t.fetchFeed(ctx, t.FeedConfig.URL)
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to fetch feed: %w", err)
	}

  metadata, items, err := t.parser.Run(data)
  if err != nil {
    return nil, nil, nil, "", fmt.Errorf("failed to parse feed: %w", err)
  }

  contentHash, err := t.feedRepo.GetFeedContentHash(t.FeedName)
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to get stored content hash: %w", err)
	}

	hash := sha256.Sum256(data)
	newContentHash := hex.EncodeToString(hash[:8])

	return metadata, items, contentHash, newContentHash, nil
}

func (t *ProcessFeedTask) storeFeedMetadataWithHash(ctx context.Context, metadata *feed.Metadata, contentHash string) error {
	now := time.Now().UTC()
	nextFetch := now.Add(time.Duration(t.FeedConfig.Settings.RefreshInterval) * time.Second)

	err := t.feedRepo.UpdateFeedMetadataWithHash(t.FeedName, metadata.Title, metadata.Link, metadata.Description, metadata.ImageURL, metadata.Language, metadata.FeedPublishedAt, metadata.FeedUpdatedAt, contentHash, nextFetch)
	if err != nil {
		return fmt.Errorf("failed to update feed metadata with hash and next fetch time: %w", err)
	}

	return nil
}

func (t *ProcessFeedTask) fetchContent(ctx context.Context, url string) ([]byte, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(t.FeedConfig.Settings.Timeout)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(timeoutCtx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", t.userAgent)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(strings.ToLower(contentType), "text/html") {
		return nil, fmt.Errorf("content type is not HTML: %s", contentType)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}

func (t *ProcessFeedTask) fetchAndExtractContent(ctx context.Context, item feed.Item) (string, error) {
	if item.Link == "" {
		return "", fmt.Errorf("item has no link")
	}

	data, err := t.fetchContent(ctx, item.Link)
	if err != nil {
		return "", fmt.Errorf("failed to fetch article content: %w", err)
	}

	extractedContent, err := t.contentExtractor.Run(data)
	if err != nil {
		return "", fmt.Errorf("failed to extract content: %w", err)
	}

	return extractedContent, nil
}

func (t *ProcessFeedTask) storeItem(ctx context.Context, item feed.Item) error {
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

	return nil
}
