package tasks

import (
	"github.com/lysyi3m/rss-comb/app/feed_config"
)

// ProcessorInterface defines the interface for feed processing operations.
// Defined here as the tasks package is the primary consumer of this interface.
// Used by task scheduler and individual tasks to process feeds and manage filtering.
// This interface provides the core feed processing functionality including fetching,
// parsing, filtering, and storing feed items. Configuration is injected per operation
// for clean dependency management and improved testability.
type ProcessorInterface interface {
	ProcessFeed(feedID string, feedConfig *feed_config.FeedConfig) error
	IsFeedEnabled(feedConfig *feed_config.FeedConfig) bool
	ReapplyFilters(feedID string, feedConfig *feed_config.FeedConfig) (int, int, error)
}

// TaskSchedulerInterface defines the interface for task scheduling operations.
// Used by the main application to manage background task processing.
// This interface provides task queue management, worker pool control, configuration
// management, and monitoring capabilities.
// Example usage:
//   scheduler := NewTaskScheduler(configCache, feedRepo, processor, contentExtractor)
//   scheduler.Start()
//   defer scheduler.Stop()
//   scheduler.EnqueueTask(NewProcessFeedTask(feedID, feedConfig, processor))
type TaskSchedulerInterface interface {
	Start()
	Stop()
	EnqueueTask(task Task) error
}
