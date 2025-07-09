package tasks

import (
	"context"
	"fmt"
	"log"

	"github.com/lysyi3m/rss-comb/app/feed"
)

// RefilterFeedTask represents a task to reapply filters to a feed
type RefilterFeedTask struct {
	BaseTask
	FeedID     string
	ConfigFile string
	processor  feed.FeedProcessor
}

// NewRefilterFeedTask creates a new refilter feed task
func NewRefilterFeedTask(feedID, configFile string, processor feed.FeedProcessor) *RefilterFeedTask {
	description := fmt.Sprintf("Refilter feed %s from config %s", feedID, configFile)
	
	return &RefilterFeedTask{
		BaseTask:   NewBaseTask(feedID, TaskTypeRefilterFeed, PriorityHigh, description),
		FeedID:     feedID,
		ConfigFile: configFile,
		processor:  processor,
	}
}

// Execute reapplies filters to the feed items
func (t *RefilterFeedTask) Execute(ctx context.Context) error {
	log.Printf("Executing RefilterFeedTask for feed %s", t.FeedID)
	
	// Check if context is cancelled before starting
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	// Reapply filters
	updatedCount, errorCount, err := t.processor.ReapplyFilters(t.FeedID, t.ConfigFile)
	if err != nil {
		log.Printf("RefilterFeedTask failed for feed %s: %v", t.FeedID, err)
		return fmt.Errorf("failed to refilter feed %s: %w", t.FeedID, err)
	}
	
	log.Printf("RefilterFeedTask completed successfully for feed %s: %d items updated, %d errors", 
		t.FeedID, updatedCount, errorCount)
	
	return nil
}

// GetFeedID returns the feed ID for this task
func (t *RefilterFeedTask) GetFeedID() string {
	return t.FeedID
}

// GetConfigFile returns the config file for this task
func (t *RefilterFeedTask) GetConfigFile() string {
	return t.ConfigFile
}