package tasks

// TaskSchedulerInterface defines the interface for task scheduling operations
type TaskSchedulerInterface interface {
	Start()
	Stop()
	EnqueueTask(task Task) error
	GetStats() TaskStats
	Health() map[string]interface{}
}