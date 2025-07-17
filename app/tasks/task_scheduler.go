package tasks

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/lysyi3m/rss-comb/app/config"
	"github.com/lysyi3m/rss-comb/app/feed_config"
	"github.com/lysyi3m/rss-comb/app/database"
)

// Compile-time interface compliance checks
var _ TaskSchedulerInterface = (*TaskScheduler)(nil)


// TaskScheduler manages the execution of tasks using priority-based task queues
type TaskScheduler struct {
	processor        ProcessorInterface
	feedRepo         database.FeedRepository
	configCache      *feed_config.ConfigCacheHandler
	contentExtractor ContentExtractionInterface
	interval         time.Duration
	workerCount      int
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	// Priority-based task queues
	topPriorityQueue    chan Task // Priority 0: Config sync tasks
	highPriorityQueue   chan Task // Priority 1: Feed processing, refilter tasks
	normalPriorityQueue chan Task // Priority 2: Content extraction tasks
}

// NewTaskScheduler creates a new priority-based task scheduler
func NewTaskScheduler(configCache *feed_config.ConfigCacheHandler, feedRepo database.FeedRepository,
	processor ProcessorInterface, contentExtractor ContentExtractionInterface) TaskSchedulerInterface {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := config.Get()
	
	return &TaskScheduler{
		processor:           processor,
		feedRepo:            feedRepo,
		configCache:         configCache,
		contentExtractor:    contentExtractor,
		interval:            time.Duration(cfg.GetSchedulerInterval()) * time.Second,
		workerCount:         cfg.GetWorkerCount(),
		ctx:                 ctx,
		cancel:              cancel,
		topPriorityQueue:    make(chan Task, 50),  // Priority 0: Config sync
		highPriorityQueue:   make(chan Task, 100), // Priority 1: Feed processing
		normalPriorityQueue: make(chan Task, 100), // Priority 2: Content extraction
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
	
	// Close all task queues after all goroutines have stopped
	close(s.topPriorityQueue)
	close(s.highPriorityQueue)
	close(s.normalPriorityQueue)
	
	slog.Debug("Task scheduler stopped")
}

// EnqueueTask adds a task to the appropriate priority queue
func (s *TaskScheduler) EnqueueTask(task Task) error {
	var targetQueue chan Task
	
	switch task.GetPriority() {
	case PriorityTop:
		targetQueue = s.topPriorityQueue
	case PriorityHigh:
		targetQueue = s.highPriorityQueue
	case PriorityNormal:
		targetQueue = s.normalPriorityQueue
	default:
		return fmt.Errorf("unknown task priority: %d", task.GetPriority())
	}
	
	select {
	case targetQueue <- task:
		slog.Debug("Task enqueued", "description", task.GetDescription(), "type", task.GetType(), "priority", task.GetPriority())
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	default:
		return fmt.Errorf("task queue is full for priority %d", task.GetPriority())
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
	// First, ensure all configured feeds are synced to database
	allConfigs := s.configCache.GetAllConfigs()
	for configFile, feedConfig := range allConfigs {
		// Check if feed exists in database
		existingFeed, err := s.feedRepo.GetFeedByID(feedConfig.Feed.ID)
		if err != nil {
			slog.Error("Failed to check feed existence", "feed_id", feedConfig.Feed.ID, "error", err)
			continue
		}
		
		// If feed doesn't exist in database, create SyncFeedConfigTask
		if existingFeed == nil {
			syncTask := NewSyncFeedConfigTask(configFile, feedConfig, s.feedRepo)
			if err := s.EnqueueTask(syncTask); err != nil {
				slog.Warn("Failed to enqueue SyncFeedConfigTask", "feed_id", feedConfig.Feed.ID, "error", err)
			} else {
				slog.Debug("SyncFeedConfigTask enqueued for new feed", "feed_id", feedConfig.Feed.ID)
			}
		}
	}
	
	// Then, get feeds that are due for refresh
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
		
		// Enqueue ProcessFeedTask (Priority High)
		processTask := NewProcessFeedTask(feed.ID, feedConfig, s.processor)
		if err := s.EnqueueTask(processTask); err != nil {
			slog.Warn("Failed to enqueue ProcessFeedTask", "title", feed.Title, "error", err)
			continue
		}
		
		// If content extraction is enabled, enqueue ExtractContentTask (Priority Normal)
		if feedConfig.Settings.ExtractContent {
			extractTask := NewExtractContentTask(feed.ID, feedConfig, s.contentExtractor)
			if err := s.EnqueueTask(extractTask); err != nil {
				slog.Warn("Failed to enqueue ExtractContentTask", "title", feed.Title, "error", err)
			}
		}
	}
}

// worker processes tasks from priority queues (highest priority first)
func (s *TaskScheduler) worker(id int) {
	defer s.wg.Done()
	slog.Debug("Worker started", "worker_id", id)

	for {
		select {
		// Priority 0: Top priority (Config sync)
		case task, ok := <-s.topPriorityQueue:
			if !ok {
				slog.Debug("Worker stopping - top priority queue closed", "worker_id", id)
				return
			}
			s.executeTask(id, task)

		// Priority 1: High priority (Feed processing, refilter)
		case task, ok := <-s.highPriorityQueue:
			if !ok {
				slog.Debug("Worker stopping - high priority queue closed", "worker_id", id)
				return
			}
			s.executeTask(id, task)

		// Priority 2: Normal priority (Content extraction)
		case task, ok := <-s.normalPriorityQueue:
			if !ok {
				slog.Debug("Worker stopping - normal priority queue closed", "worker_id", id)
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
