package tasks

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/feed"
)

// TaskScheduler manages the execution of tasks using a generic task queue
type TaskScheduler struct {
	processor    feed.FeedProcessor
	feedRepo     database.FeedRepositoryInterface
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
	AverageProcessTime time.Duration
	processTimes       []time.Duration
	TaskCounts         map[TaskType]int64
}

// NewTaskScheduler creates a new generic task scheduler
func NewTaskScheduler(processor feed.FeedProcessor, feedRepo database.FeedRepositoryInterface,
	interval time.Duration, workerCount int) *TaskScheduler {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &TaskScheduler{
		processor:   processor,
		feedRepo:    feedRepo,
		interval:    interval,
		workerCount: workerCount,
		ctx:         ctx,
		cancel:      cancel,
		taskQueue:   make(chan Task, 100), // Buffer of 100 tasks
		stats: &TaskStats{
			CurrentWorkers: workerCount,
			processTimes:   make([]time.Duration, 0, 100),
			TaskCounts:     make(map[TaskType]int64),
		},
	}
}

// Start begins the scheduler operation
func (s *TaskScheduler) Start() {
	log.Printf("Starting task scheduler with %d workers, interval: %v", s.workerCount, s.interval)

	// Start worker pool
	for i := 0; i < s.workerCount; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}

	// Start scheduler loop
	s.wg.Add(1)
	go s.schedulerLoop()

	log.Println("Task scheduler started successfully")
}

// Stop gracefully stops the scheduler
func (s *TaskScheduler) Stop() {
	log.Println("Stopping task scheduler...")
	s.cancel()
	
	// Wait for all goroutines to finish first
	s.wg.Wait()
	
	// Close the task queue after all goroutines have stopped
	close(s.taskQueue)
	
	log.Println("Task scheduler stopped")
}

// EnqueueTask adds a task to the queue
func (s *TaskScheduler) EnqueueTask(task Task) error {
	select {
	case s.taskQueue <- task:
		log.Printf("Enqueued task: %s (%s)", task.GetDescription(), task.GetType())
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
			log.Println("Scheduler loop stopping...")
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
		log.Printf("Error getting feeds due for refresh: %v", err)
		s.mu.Lock()
		s.stats.TotalErrors++
		s.mu.Unlock()
		return
	}

	if len(feeds) == 0 {
		log.Printf("No feeds due for processing")
		return
	}

	// Filter out disabled feeds before logging and processing
	enabledFeeds := make([]database.Feed, 0, len(feeds))
	for _, feed := range feeds {
		if s.processor.IsFeedEnabled(feed.ConfigFile) {
			enabledFeeds = append(enabledFeeds, feed)
		}
	}

	if len(enabledFeeds) == 0 {
		if len(feeds) > 0 {
			log.Printf("No enabled feeds to process (%d feeds are disabled in config)", len(feeds))
		}
		return
	}

	log.Printf("Found %d enabled feeds due for processing", len(enabledFeeds))

	// Create and enqueue ProcessFeedTasks for enabled feeds
	for _, feed := range enabledFeeds {
		task := NewProcessFeedTask(feed.ID, feed.ConfigFile, s.processor)
		
		select {
		case s.taskQueue <- task:
			log.Printf("Enqueued ProcessFeedTask: %s", feed.Title)
		case <-s.ctx.Done():
			return
		default:
			log.Printf("Warning: task queue full, skipping feed: %s", feed.Title)
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
	log.Printf("Worker %d started", id)

	for {
		select {
		case task, ok := <-s.taskQueue:
			if !ok {
				log.Printf("Worker %d: task queue closed, stopping", id)
				return
			}

			s.executeTask(id, task)

		case <-s.ctx.Done():
			log.Printf("Worker %d: context cancelled, stopping", id)
			return
		}
	}
}

// executeTask executes a single task
func (s *TaskScheduler) executeTask(workerID int, task Task) {
	log.Printf("Worker %d executing task: %s", workerID, task.GetDescription())
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

	// Update average processing time
	s.stats.processTimes = append(s.stats.processTimes, duration)
	if len(s.stats.processTimes) > 100 {
		s.stats.processTimes = s.stats.processTimes[1:] // Keep only last 100
	}
	s.updateAverageProcessTime()

	if err != nil {
		s.stats.TotalErrors++
		log.Printf("Worker %d error executing task %s: %v", workerID, task.GetDescription(), err)
	} else {
		log.Printf("Worker %d successfully executed task %s in %v", workerID, task.GetDescription(), duration)
	}
}

// updateAverageProcessTime calculates the average processing time
func (s *TaskScheduler) updateAverageProcessTime() {
	if len(s.stats.processTimes) == 0 {
		s.stats.AverageProcessTime = 0
		return
	}

	var total time.Duration
	for _, t := range s.stats.processTimes {
		total += t
	}
	s.stats.AverageProcessTime = total / time.Duration(len(s.stats.processTimes))
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

// TriggerImmediate triggers immediate processing of all due feeds
func (s *TaskScheduler) TriggerImmediate() {
	log.Println("Triggering immediate feed processing...")
	go s.enqueueDueFeeds()
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
		"average_process_time": stats.AverageProcessTime.String(),
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

// GetQueuedTasks returns information about tasks currently in the queue
func (s *TaskScheduler) GetQueuedTasks() []map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Create a snapshot of current tasks in queue
	tasks := make([]Task, 0, len(s.taskQueue))
	queueCopy := make(chan Task, len(s.taskQueue))
	
	// Drain the queue and copy to both tasks slice and new queue
	for len(s.taskQueue) > 0 {
		select {
		case task := <-s.taskQueue:
			tasks = append(tasks, task)
			queueCopy <- task
		default:
			break
		}
	}
	
	// Restore the queue
	s.taskQueue = queueCopy
	
	// Sort tasks by priority (highest first) and then by creation time
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].GetPriority() != tasks[j].GetPriority() {
			return tasks[i].GetPriority() > tasks[j].GetPriority()
		}
		return tasks[i].GetCreatedAt().Before(tasks[j].GetCreatedAt())
	})
	
	// Convert to response format
	result := make([]map[string]interface{}, len(tasks))
	for i, task := range tasks {
		result[i] = map[string]interface{}{
			"id":          task.GetID(),
			"type":        task.GetType(),
			"priority":    task.GetPriority(),
			"created_at":  task.GetCreatedAt().Format(time.RFC3339),
			"description": task.GetDescription(),
		}
	}
	
	return result
}