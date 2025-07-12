package tasks

import (
	"github.com/lysyi3m/rss-comb/app/config"
)

// TaskSchedulerInterface defines the interface for task scheduling operations.
// Used by the main application to manage background task processing.
// This interface provides task queue management, worker pool control, configuration
// management, and monitoring capabilities.
// Example usage:
//   scheduler := NewTaskScheduler(processor, feedRepo, configs, interval, workerCount)
//   scheduler.Start()
//   defer scheduler.Stop()
//   scheduler.EnqueueTask(NewProcessFeedTask(feedID, feedConfig, processor))
type TaskSchedulerInterface interface {
	Start()
	Stop()
	EnqueueTask(task Task) error
	GetStats() TaskStats
	Health() map[string]interface{}
	OnConfigUpdate(filePath string, config *config.FeedConfig, isDelete bool) error
}