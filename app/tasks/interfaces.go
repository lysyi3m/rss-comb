package tasks

// TaskSchedulerInterface defines the interface for task scheduling operations.
// Used by the main application to manage background task processing.
// This interface provides task queue management, worker pool control, configuration
// management, and monitoring capabilities.
// Example usage:
//
//	scheduler := NewScheduler(configCache, feedRepo, httpClient, parser, filterer, contentExtractor, userAgent)
//	scheduler.Start()
//	defer scheduler.Stop()
//	scheduler.EnqueueTask(NewProcessFeedTask(...))
type TaskSchedulerInterface interface {
	Start()
	Stop()
	EnqueueTask(task TaskInterface) error
}
