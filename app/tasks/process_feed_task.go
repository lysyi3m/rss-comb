package tasks

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lysyi3m/rss-comb/app/config"
)

type ProcessFeedTask struct {
	BaseTask
	FeedID     string
	FeedConfig *config.FeedConfig
	processor  ProcessorInterface
}

func NewProcessFeedTask(feedID string, feedConfig *config.FeedConfig, processor ProcessorInterface) *ProcessFeedTask {
	description := fmt.Sprintf("Process feed %s (%s)", feedID, feedConfig.Feed.Title)
	
	return &ProcessFeedTask{
		BaseTask:   NewBaseTask(feedID, TaskTypeProcessFeed, description),
		FeedID:     feedID,
		FeedConfig: feedConfig,
		processor:  processor,
	}
}

func (t *ProcessFeedTask) Execute(ctx context.Context) error {
	slog.Debug("Task started", "type", "ProcessFeed", "feed_id", t.FeedID)
	
	// Fast-fail on cancellation to avoid unnecessary work
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	err := t.processor.ProcessFeed(t.FeedID, t.FeedConfig)
	if err != nil {
		slog.Error("Task failed", "type", "ProcessFeed", "feed_id", t.FeedID, "error", err)
		return fmt.Errorf("failed to process feed %s: %w", t.FeedID, err)
	}
	
	slog.Debug("Task completed", "type", "ProcessFeed", "feed_id", t.FeedID)
	return nil
}

func (t *ProcessFeedTask) GetFeedID() string {
	return t.FeedID
}

func (t *ProcessFeedTask) GetFeedConfig() *config.FeedConfig {
	return t.FeedConfig
}
