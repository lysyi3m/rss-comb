package tasks

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lysyi3m/rss-comb/app/feed_config"
)

type ProcessFeedTask struct {
	BaseTask
	FeedConfig *feed_config.FeedConfig
	processor  ProcessorInterface
}

func NewProcessFeedTask(feedID string, feedConfig *feed_config.FeedConfig, processor ProcessorInterface) *ProcessFeedTask {
	description := fmt.Sprintf("Process feed %s (%s)", feedID, feedConfig.Feed.Title)
	
	return &ProcessFeedTask{
		BaseTask:   NewBaseTask(feedID, TaskTypeProcessFeed, PriorityHigh, description, feedID),
		FeedConfig: feedConfig,
		processor:  processor,
	}
}

func (t *ProcessFeedTask) Execute(ctx context.Context) error {
	slog.Debug("Task started", "type", "ProcessFeed", "feed_id", t.GetFeedID())
	
	// Fast-fail on cancellation to avoid unnecessary work
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	err := t.processor.ProcessFeed(t.GetFeedID(), t.FeedConfig)
	if err != nil {
		slog.Error("Task failed", "type", "ProcessFeed", "feed_id", t.GetFeedID(), "error", err)
		return fmt.Errorf("failed to process feed %s: %w", t.GetFeedID(), err)
	}
	
	slog.Debug("Task completed", "type", "ProcessFeed", "feed_id", t.GetFeedID())
	return nil
}

func (t *ProcessFeedTask) GetFeedConfig() *feed_config.FeedConfig {
	return t.FeedConfig
}
