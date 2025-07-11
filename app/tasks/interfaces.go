package tasks

// TaskSchedulerInterface defines the interface for task scheduling operations.
// Used by the main application to manage background task processing.
// This interface provides task queue management, worker pool control, and monitoring.
// Example usage:
//   scheduler := NewTaskScheduler(processor, feedRepo, interval, workerCount)
//   scheduler.Start()
//   defer scheduler.Stop()
//   scheduler.EnqueueTask(NewProcessFeedTask(feedID, configFile, processor))
type TaskSchedulerInterface interface {
	Start()
	Stop()
	EnqueueTask(task Task) error
	GetStats() TaskStats
	Health() map[string]interface{}
}