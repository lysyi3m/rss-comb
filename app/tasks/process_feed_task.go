package tasks

import (
	"context"
	"fmt"
	"log"

	"github.com/lysyi3m/rss-comb/app/config"
	"github.com/lysyi3m/rss-comb/app/feed"
)

// ProcessFeedTask represents a task to process a feed
type ProcessFeedTask struct {
	BaseTask
	FeedID     string
	FeedConfig *config.FeedConfig
	processor  feed.ProcessorInterface
}

// NewProcessFeedTask creates a new process feed task
func NewProcessFeedTask(feedID string, feedConfig *config.FeedConfig, processor feed.ProcessorInterface) *ProcessFeedTask {
	description := fmt.Sprintf("Process feed %s (%s)", feedID, feedConfig.Feed.Title)
	
	return &ProcessFeedTask{
		BaseTask:   NewBaseTask(feedID, TaskTypeProcessFeed, PriorityNormal, description),
		FeedID:     feedID,
		FeedConfig: feedConfig,
		processor:  processor,
	}
}

// Execute processes the feed
func (t *ProcessFeedTask) Execute(ctx context.Context) error {
	log.Printf("Executing ProcessFeedTask for feed %s", t.FeedID)
	
	// Check if context is cancelled before starting
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	// Process the feed
	err := t.processor.ProcessFeed(t.FeedID, t.FeedConfig)
	if err != nil {
		log.Printf("ProcessFeedTask failed for feed %s: %v", t.FeedID, err)
		return fmt.Errorf("failed to process feed %s: %w", t.FeedID, err)
	}
	
	log.Printf("ProcessFeedTask completed successfully for feed %s", t.FeedID)
	return nil
}

// GetFeedID returns the feed ID for this task
func (t *ProcessFeedTask) GetFeedID() string {
	return t.FeedID
}

// GetFeedConfig returns the feed config for this task
func (t *ProcessFeedTask) GetFeedConfig() *config.FeedConfig {
	return t.FeedConfig
}