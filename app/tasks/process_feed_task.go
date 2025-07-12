package tasks

import (
	"context"
	"fmt"
	"log"

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
		BaseTask:   NewBaseTask(feedID, TaskTypeProcessFeed, PriorityNormal, description),
		FeedID:     feedID,
		FeedConfig: feedConfig,
		processor:  processor,
	}
}

func (t *ProcessFeedTask) Execute(ctx context.Context) error {
	log.Printf("Executing ProcessFeedTask for feed %s", t.FeedID)
	
	// Fast-fail on cancellation to avoid unnecessary work
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	err := t.processor.ProcessFeed(t.FeedID, t.FeedConfig)
	if err != nil {
		log.Printf("ProcessFeedTask failed for feed %s: %v", t.FeedID, err)
		return fmt.Errorf("failed to process feed %s: %w", t.FeedID, err)
	}
	
	log.Printf("ProcessFeedTask completed successfully for feed %s", t.FeedID)
	return nil
}

func (t *ProcessFeedTask) GetFeedID() string {
	return t.FeedID
}

func (t *ProcessFeedTask) GetFeedConfig() *config.FeedConfig {
	return t.FeedConfig
}