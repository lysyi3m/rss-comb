package tasks

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/lysyi3m/rss-comb/app/config_sync"
	"github.com/lysyi3m/rss-comb/app/database"
)

// Compile-time interface compliance checks
var _ TaskSchedulerInterface = (*TaskScheduler)(nil)

// TaskScheduler manages the execution of tasks using a generic task queue
type TaskScheduler struct {
	processor    ProcessorInterface
	feedRepo     database.FeedScheduler
	configCache  *config_sync.ConfigCacheHandler
	interval     time.Duration
	workerCount  int
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	taskQueue    chan Task
}


// NewTaskScheduler creates a new generic task scheduler
func NewTaskScheduler(processor ProcessorInterface, feedRepo database.FeedScheduler,
	configCache *config_sync.ConfigCacheHandler, interval time.Duration, workerCount int) TaskSchedulerInterface {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &TaskScheduler{
		processor:   processor,
		feedRepo:    feedRepo,
		configCache: configCache,
		interval:    interval,
		workerCount: workerCount,
		ctx:         ctx,
		cancel:      cancel,
		taskQueue:   make(chan Task, 100), // Buffer of 100 tasks
	}
}

// Start begins the scheduler operation
func (s *TaskScheduler) Start() {
	slog.Debug("Task scheduler starting", "workers", s.workerCount, "interval", s.interval)

	// Start worker pool
	for i := 0; i < s.workerCount; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}

	// Start scheduler loop
	s.wg.Add(1)
	go s.schedulerLoop()

	slog.Debug("Task scheduler started")
}

// Stop gracefully stops the scheduler
func (s *TaskScheduler) Stop() {
	slog.Debug("Task scheduler stopping")
	s.cancel()
	
	// Wait for all goroutines to finish first
	s.wg.Wait()
	
	// Close the task queue after all goroutines have stopped
	close(s.taskQueue)
	
	slog.Debug("Task scheduler stopped")
}

// EnqueueTask adds a task to the queue
func (s *TaskScheduler) EnqueueTask(task Task) error {
	select {
	case s.taskQueue <- task:
		slog.Debug("Task enqueued", "description", task.GetDescription(), "type", task.GetType())
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	default:
		return fmt.Errorf("task queue is full")
	}
}

// schedulerLoop is the main scheduling loop for automatic feed processing
func (s *TaskScheduler) schedulerLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Process feeds immediately on start
	s.enqueueDueFeeds()

	for {
		select {
		case <-s.ctx.Done():
			slog.Debug("Scheduler loop stopping")
			return
		case <-ticker.C:
			s.enqueueDueFeeds()
		}
	}
}

// enqueueDueFeeds finds feeds that need processing and creates ProcessFeedTasks
func (s *TaskScheduler) enqueueDueFeeds() {
	feeds, err := s.feedRepo.GetFeedsDueForRefresh()
	if err != nil {
		slog.Error("Failed to get feeds for refresh", "error", err)
		return
	}

	if len(feeds) == 0 {
		slog.Debug("No feeds due for processing")
		return
	}

	// Filter out disabled feeds before logging and processing
	enabledFeeds := make([]database.Feed, 0, len(feeds))
	for _, feed := range feeds {
		feedConfig, ok := s.configCache.GetConfig(feed.ConfigFile)
		if ok && s.processor.IsFeedEnabled(feedConfig) {
			enabledFeeds = append(enabledFeeds, feed)
		}
	}

	if len(enabledFeeds) == 0 {
		if len(feeds) > 0 {
			slog.Debug("All feeds disabled", "total", len(feeds))
		}
		return
	}

	slog.Debug("Feeds due for processing", "count", len(enabledFeeds))

	// Create and enqueue ProcessFeedTasks for enabled feeds
	for _, feed := range enabledFeeds {
		feedConfig, ok := s.configCache.GetConfig(feed.ConfigFile)
		if !ok {
			slog.Warn("Feed configuration not found, skipping", "feed_id", feed.ID)
			continue
		}
		
		task := NewProcessFeedTask(feed.ID, feedConfig, s.processor)
		
		select {
		case s.taskQueue <- task:
			slog.Debug("ProcessFeedTask enqueued", "title", feed.Title)
		case <-s.ctx.Done():
			return
		default:
			slog.Warn("Task queue full, skipping feed", "title", feed.Title)
		}
	}

}

// worker processes tasks from the task queue
func (s *TaskScheduler) worker(id int) {
	defer s.wg.Done()
	slog.Debug("Worker started", "worker_id", id)

	for {
		select {
		case task, ok := <-s.taskQueue:
			if !ok {
				slog.Debug("Worker stopping - queue closed", "worker_id", id)
				return
			}

			s.executeTask(id, task)

		case <-s.ctx.Done():
			slog.Debug("Worker stopping - context cancelled", "worker_id", id)
			return
		}
	}
}

// executeTask executes a single task
func (s *TaskScheduler) executeTask(workerID int, task Task) {
	slog.Debug("Worker executing task", "worker_id", workerID, "task", task.GetDescription())
	start := time.Now()

	// Execute the task with a timeout context
	taskCtx, cancel := context.WithTimeout(s.ctx, 5*time.Minute) // 5 minute timeout per task
	defer cancel()

	err := task.Execute(taskCtx)
	duration := time.Since(start)

	if err != nil {
		slog.Error("Worker task execution failed", "worker_id", workerID, "task", task.GetDescription(), "error", err)
	} else {
		slog.Debug("Worker task completed", "worker_id", workerID, "task", task.GetDescription(), "duration", duration.String())
	}
}





