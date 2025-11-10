package tasks

type TaskSchedulerInterface interface {
	Start()
	Stop()
	EnqueueTask(task TaskInterface) error
}
