package tasks

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/feed"
)

type ExtractContentTask struct {
	Task
	FeedConfig       *feed.Config
	httpClient       *http.Client
	contentExtractor *feed.ContentExtractor
	feedRepo         database.FeedRepository
	itemRepo         database.ItemRepository
	userAgent        string
}

func NewExtractContentTask(feedName string, feedConfig *feed.Config, httpClient *http.Client, contentExtractor *feed.ContentExtractor, feedRepo database.FeedRepository, itemRepo database.ItemRepository, userAgent string) *ExtractContentTask {
	return &ExtractContentTask{
		Task:             NewTask(TaskTypeExtractContent, feedName),
		FeedConfig:       feedConfig,
		httpClient:       httpClient,
		contentExtractor: contentExtractor,
		feedRepo:         feedRepo,
		itemRepo:         itemRepo,
		userAgent:        userAgent,
	}
}

func (t *ExtractContentTask) Execute(ctx context.Context) error {

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if !t.FeedConfig.Settings.ExtractContent {
		slog.Debug("Content extraction disabled for feed", "feed", t.FeedName)
		return nil
	}

	items, err := t.itemRepo.GetItemsForExtraction(t.FeedName, t.FeedConfig.Settings.MaxItems)
	if err != nil {
		return fmt.Errorf("failed to get items for content extraction: %w", err)
	}

	if len(items) == 0 {
		slog.Debug("No items need content extraction", "feed", t.FeedName)
		return nil
	}

	successCount := 0
	errorCount := 0

	for _, item := range items {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		extractCtx, cancel := context.WithTimeout(ctx, time.Duration(t.FeedConfig.Settings.Timeout)*time.Second)

		err := t.extractContentForItem(extractCtx, item)
		cancel()

		if err != nil {
			slog.Error("Failed to extract content for item", "item_id", item.ID, "url", item.Link, "error", err)
			errorCount++

			now := time.Now().UTC()
			err = t.itemRepo.UpdateExtractionStatus(item.ID, "failed", &now, err.Error())
			if err != nil {
				slog.Error("Failed to update content extraction status", "item_id", item.ID, "error", err)
			}
		} else {
			successCount++
		}
	}

	slog.Info("Task completed",
    "type", t.GetType(),
    "feed", t.FeedName,
    "duration", t.GetDuration(),
    "success", successCount,
    "errors", errorCount)

	return nil
}

func (t *ExtractContentTask) extractContentForItem(ctx context.Context, item database.ItemForExtraction) error {
	if item.Link == "" {
		return fmt.Errorf("item has no link")
	}

	data, err := t.fetchArticleContent(ctx, item.Link)
	if err != nil {
		return fmt.Errorf("failed to fetch article content: %w", err)
	}

	extractedContent, err := t.contentExtractor.Run(data)
	if err != nil {
		return fmt.Errorf("failed to extract content: %w", err)
	}

	now := time.Now().UTC()
	err = t.itemRepo.UpdateExtractedContentAndStatus(item.ID, extractedContent, "success", &now, "")
	if err != nil {
		return fmt.Errorf("failed to update extracted content and status: %w", err)
	}

	slog.Debug("Content extracted successfully", "item_id", item.ID, "url", item.Link, "content_length", len(extractedContent))
	return nil
}

func (t *ExtractContentTask) fetchArticleContent(ctx context.Context, url string) ([]byte, error) {
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
