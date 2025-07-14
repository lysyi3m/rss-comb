package tasks

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lysyi3m/rss-comb/app/config"
)

// RefilterFeedTask represents a task to reapply filters to a feed
type RefilterFeedTask struct {
	BaseTask
	FeedID     string
	FeedConfig *config.FeedConfig
	processor  ProcessorInterface
}

// NewRefilterFeedTask creates a new refilter feed task
func NewRefilterFeedTask(feedID string, feedConfig *config.FeedConfig, processor ProcessorInterface) *RefilterFeedTask {
	description := fmt.Sprintf("Refilter feed %s (%s)", feedID, feedConfig.Feed.Title)
	
	return &RefilterFeedTask{
		BaseTask:   NewBaseTask(feedID, TaskTypeRefilterFeed, PriorityHigh, description),
		FeedID:     feedID,
		FeedConfig: feedConfig,
		processor:  processor,
	}
}

// Execute reapplies filters to the feed items
func (t *RefilterFeedTask) Execute(ctx context.Context) error {
	slog.Debug("Task started", "type", "RefilterFeed", "feed_id", t.FeedID)
	
	// Check if context is cancelled before starting
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	// Reapply filters
	updatedCount, errorCount, err := t.processor.ReapplyFilters(t.FeedID, t.FeedConfig)
	if err != nil {
		slog.Error("Task failed", "type", "RefilterFeed", "feed_id", t.FeedID, "error", err)
		return fmt.Errorf("failed to refilter feed %s: %w", t.FeedID, err)
	}
	
	slog.Info("Task completed", "type", "RefilterFeed", "feed_id", t.FeedID, "updated", updatedCount, "errors", errorCount)
	
	return nil
}

// GetFeedID returns the feed ID for this task
func (t *RefilterFeedTask) GetFeedID() string {
	return t.FeedID
}

// GetFeedConfig returns the feed config for this task
func (t *RefilterFeedTask) GetFeedConfig() *config.FeedConfig {
	return t.FeedConfig
}
