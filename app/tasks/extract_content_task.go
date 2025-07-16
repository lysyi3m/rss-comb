package tasks

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lysyi3m/rss-comb/app/config"
)

// ContentExtractionInterface defines the interface for content extraction operations
type ContentExtractionInterface interface {
	ExtractContentForFeed(ctx context.Context, feedID string, feedConfig *config.FeedConfig) error
}

// ExtractContentTask represents a task for extracting content from feed items
type ExtractContentTask struct {
	BaseTask
	FeedID        string
	FeedConfig    *config.FeedConfig
	extractor     ContentExtractionInterface
}

// NewExtractContentTask creates a new content extraction task
func NewExtractContentTask(feedID string, feedConfig *config.FeedConfig, extractor ContentExtractionInterface) *ExtractContentTask {
	description := fmt.Sprintf("Extract content for feed %s (%s)", feedID, feedConfig.Feed.Title)
	
	return &ExtractContentTask{
		BaseTask:   NewBaseTask(feedID+"-extract", TaskTypeExtractContent, description),
		FeedID:     feedID,
		FeedConfig: feedConfig,
		extractor:  extractor,
	}
}

// Execute runs the content extraction task
func (t *ExtractContentTask) Execute(ctx context.Context) error {
	slog.Debug("Task started", "type", "ExtractContent", "feed_id", t.FeedID)
	
	// Fast-fail on cancellation to avoid unnecessary work
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	// Check if content extraction is enabled for this feed
	if !t.FeedConfig.Settings.ExtractContent {
		slog.Debug("Content extraction disabled for feed", "feed_id", t.FeedID)
		return nil
	}
	
	// Extract content with timeout
	extractCtx, cancel := context.WithTimeout(ctx, t.FeedConfig.Settings.GetExtractionTimeout())
	defer cancel()
	
	err := t.extractor.ExtractContentForFeed(extractCtx, t.FeedID, t.FeedConfig)
	if err != nil {
		slog.Error("Task failed", "type", "ExtractContent", "feed_id", t.FeedID, "error", err)
		return fmt.Errorf("failed to extract content for feed %s: %w", t.FeedID, err)
	}
	
	slog.Debug("Task completed", "type", "ExtractContent", "feed_id", t.FeedID)
	return nil
}

// GetFeedID returns the feed ID for this task
func (t *ExtractContentTask) GetFeedID() string {
	return t.FeedID
}

// GetFeedConfig returns the feed configuration for this task
func (t *ExtractContentTask) GetFeedConfig() *config.FeedConfig {
	return t.FeedConfig
}