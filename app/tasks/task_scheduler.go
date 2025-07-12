package tasks

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/lysyi3m/rss-comb/app/config"
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
	stats        *TaskStats
	mu           sync.RWMutex
}

// TaskStats holds scheduler statistics
type TaskStats struct {
	TotalProcessed     int64
	TotalErrors        int64
	CurrentWorkers     int
	QueueSize          int
	LastProcessedAt    *time.Time
	TaskCounts         map[TaskType]int64
}

// NewTaskScheduler creates a new generic task scheduler
func NewTaskScheduler(processor ProcessorInterface, feedRepo database.FeedScheduler,
	configs map[string]*config.FeedConfig, interval time.Duration, workerCount int) TaskSchedulerInterface {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &TaskScheduler{
		processor:   processor,
		feedRepo:    feedRepo,
		configCache: config_sync.NewConfigCacheHandler("Task scheduler", configs),
		interval:    interval,
		workerCount: workerCount,
		ctx:         ctx,
		cancel:      cancel,
		taskQueue:   make(chan Task, 100), // Buffer of 100 tasks
		stats: &TaskStats{
			CurrentWorkers: workerCount,
			TaskCounts:     make(map[TaskType]int64),
		},
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
		s.mu.Lock()
		s.stats.QueueSize = len(s.taskQueue)
		s.mu.Unlock()
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
		s.mu.Lock()
		s.stats.TotalErrors++
		s.mu.Unlock()
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

	// Update queue size stat
	s.mu.Lock()
	s.stats.QueueSize = len(s.taskQueue)
	s.mu.Unlock()
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

	s.mu.Lock()
	defer s.mu.Unlock()

	// Update statistics
	s.stats.TotalProcessed++
	s.stats.TaskCounts[task.GetType()]++
	now := time.Now()
	s.stats.LastProcessedAt = &now
	s.stats.QueueSize = len(s.taskQueue)

	if err != nil {
		s.stats.TotalErrors++
		slog.Error("Worker task execution failed", "worker_id", workerID, "task", task.GetDescription(), "error", err)
	} else {
		slog.Debug("Worker task completed", "worker_id", workerID, "task", task.GetDescription(), "duration", duration.String())
	}
}


// GetStats returns current scheduler statistics
func (s *TaskScheduler) GetStats() TaskStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	statsCopy := *s.stats
	statsCopy.QueueSize = len(s.taskQueue)
	
	// Copy the task counts map
	statsCopy.TaskCounts = make(map[TaskType]int64)
	for k, v := range s.stats.TaskCounts {
		statsCopy.TaskCounts[k] = v
	}
	
	return statsCopy
}

// Health returns the health status of the scheduler
func (s *TaskScheduler) Health() map[string]interface{} {
	stats := s.GetStats()
	
	health := map[string]interface{}{
		"status":              "healthy",
		"workers":             stats.CurrentWorkers,
		"queue_size":          stats.QueueSize,
		"total_processed":     stats.TotalProcessed,
		"total_errors":        stats.TotalErrors,
		"task_counts":         stats.TaskCounts,
	}

	if stats.LastProcessedAt != nil {
		health["last_processed_at"] = stats.LastProcessedAt.Format(time.RFC3339)
		health["last_processed_ago"] = time.Since(*stats.LastProcessedAt).String()
	}

	// Determine health status based on error rate
	if stats.TotalProcessed > 0 {
		errorRate := float64(stats.TotalErrors) / float64(stats.TotalProcessed)
		if errorRate > 0.5 {
			health["status"] = "unhealthy"
		} else if errorRate > 0.1 {
			health["status"] = "degraded"
		}
		health["error_rate"] = errorRate
	}

	return health
}

// GetConfigHandler returns the config cache handler for direct registration with config watcher
func (s *TaskScheduler) GetConfigHandler() *config_sync.ConfigCacheHandler {
	return s.configCache
}

