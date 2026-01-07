package services

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
	"github.com/lysyi3m/rss-comb/app/types"
)

func ProcessFeed(
	ctx context.Context,
	feedName string,
	feedRepo *database.FeedRepository,
	itemRepo *database.ItemRepository,
	httpClient *http.Client,
	parser *feed.Parser,
	filterer *feed.Filterer,
	contentExtractor *feed.ContentExtractor,
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

	metadata, items, contentHash, newContentHash, err := fetchAndParseFeed(ctx, feedName, dbFeed.FeedURL, settings, feedRepo, httpClient, parser, userAgent)
	if err != nil {
		return err
	}

	err = storeFeedMetadataWithHash(ctx, feedName, metadata, newContentHash, settings, feedRepo)
	if err != nil {
		return fmt.Errorf("failed to store feed metadata with hash: %w", err)
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
	extractionSuccessCount := 0
	extractionFailureCount := 0

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

		filteredItems := filterer.Run([]types.Item{item}, filters)
		processedItem := filteredItems[0]

		if processedItem.IsFiltered {
			filteredCount++
		} else {
			newCount++
		}

		if !processedItem.IsFiltered && settings.ExtractContent {
			extractedContent, extractionErr := fetchAndExtractContent(ctx, processedItem, settings, httpClient, contentExtractor, userAgent)
			if extractionErr != nil {
				slog.Warn("Failed to extract content for item",
					"feed", feedName,
					"item_link", processedItem.Link,
					"error", extractionErr)
				extractionFailureCount++
			} else if extractedContent != "" {
				processedItem.Content = extractedContent
				extractionSuccessCount++
			}
		}

		err = storeItem(ctx, feedName, processedItem, itemRepo)
		if err != nil {
			return fmt.Errorf("failed to store item: %w", err)
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
		logData = append(logData, "extraction_success", extractionSuccessCount, "extraction_failure", extractionFailureCount)
	}

	slog.Info("Feed processed", logData...)

	return nil
}

func fetchFeed(ctx context.Context, url string, timeout int, httpClient *http.Client, userAgent string) ([]byte, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(timeoutCtx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
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

func fetchAndParseFeed(
	ctx context.Context,
	feedName string,
	feedURL string,
	settings *types.Settings,
	feedRepo *database.FeedRepository,
	httpClient *http.Client,
	parser *feed.Parser,
	userAgent string,
) (*feed.Metadata, []types.Item, *string, string, error) {
	data, err := fetchFeed(ctx, feedURL, settings.Timeout, httpClient, userAgent)
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to fetch feed: %w", err)
	}

	metadata, items, err := parser.Run(data)
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

func storeFeedMetadataWithHash(
	ctx context.Context,
	feedName string,
	metadata *feed.Metadata,
	contentHash string,
	settings *types.Settings,
	feedRepo *database.FeedRepository,
) error {
	now := time.Now().UTC()
	nextFetch := now.Add(time.Duration(settings.RefreshInterval) * time.Second)

	err := feedRepo.UpdateFeedMetadataWithHash(feedName, metadata.Title, metadata.Link, metadata.Description, metadata.ImageURL, metadata.Language, metadata.FeedPublishedAt, metadata.FeedUpdatedAt, contentHash, nextFetch)
	if err != nil {
		return fmt.Errorf("failed to update feed metadata with hash and next fetch time: %w", err)
	}

	return nil
}

func fetchContent(ctx context.Context, url string, timeout int, httpClient *http.Client, userAgent string) ([]byte, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(timeoutCtx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
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

func fetchAndExtractContent(
	ctx context.Context,
	item types.Item,
	settings *types.Settings,
	httpClient *http.Client,
	contentExtractor *feed.ContentExtractor,
	userAgent string,
) (string, error) {
	if item.Link == "" {
		return "", fmt.Errorf("item has no link")
	}

	data, err := fetchContent(ctx, item.Link, settings.Timeout, httpClient, userAgent)
	if err != nil {
		return "", fmt.Errorf("failed to fetch article content: %w", err)
	}

	extractedContent, err := contentExtractor.Run(data)
	if err != nil {
		return "", fmt.Errorf("failed to extract content: %w", err)
	}

	return extractedContent, nil
}

func storeItem(ctx context.Context, feedName string, item types.Item, itemRepo *database.ItemRepository) error {
	err := itemRepo.UpsertItem(feedName, item)
	if err != nil {
		return fmt.Errorf("failed to upsert item: %w", err)
	}

	return nil
}
