package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/lysyi3m/rss-comb/internal/database"
	"github.com/lysyi3m/rss-comb/internal/feed"
)

// Scheduler manages the background processing of feeds
type Scheduler struct {
	processor    feed.FeedProcessor
	feedRepo     database.FeedRepositoryInterface
	interval     time.Duration
	workerCount  int
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	jobQueue     chan database.Feed
	stats        *Stats
	mu           sync.RWMutex
}

// Stats holds scheduler statistics
type Stats struct {
	TotalProcessed   int64
	TotalErrors      int64
	CurrentWorkers   int
	QueueSize        int
	LastProcessedAt  *time.Time
	AverageProcessTime time.Duration
	processTimes     []time.Duration
}

// NewScheduler creates a new feed scheduler
func NewScheduler(processor feed.FeedProcessor, feedRepo database.FeedRepositoryInterface,
	interval time.Duration, workerCount int) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Scheduler{
		processor:   processor,
		feedRepo:    feedRepo,
		interval:    interval,
		workerCount: workerCount,
		ctx:         ctx,
		cancel:      cancel,
		jobQueue:    make(chan database.Feed, 100), // Buffer of 100 jobs
		stats: &Stats{
			CurrentWorkers: workerCount,
			processTimes:   make([]time.Duration, 0, 100), // Keep last 100 processing times
		},
	}
}

// Start begins the scheduler operation
func (s *Scheduler) Start() {
	log.Printf("Starting scheduler with %d workers, interval: %v", s.workerCount, s.interval)

	// Start worker pool
	for i := 0; i < s.workerCount; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}

	// Start scheduler loop
	s.wg.Add(1)
	go s.schedulerLoop()

	log.Println("Scheduler started successfully")
}

// Stop gracefully stops the scheduler
func (s *Scheduler) Stop() {
	log.Println("Stopping scheduler...")
	s.cancel()
	
	// Close the job queue to signal workers to stop
	close(s.jobQueue)
	
	// Wait for all workers to finish
	s.wg.Wait()
	
	log.Println("Scheduler stopped")
}

// schedulerLoop is the main scheduling loop
func (s *Scheduler) schedulerLoop() {
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

// enqueueDueFeeds finds feeds that need processing and adds them to the queue
func (s *Scheduler) enqueueDueFeeds() {
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
			log.Printf("No enabled feeds to process (all %d feeds are disabled)", len(feeds))
		}
		return
	}

	log.Printf("Found %d enabled feeds due for processing", len(enabledFeeds))

	// Process only enabled feeds
	for _, feed := range enabledFeeds {
		select {
		case s.jobQueue <- feed:
			log.Printf("Enqueued feed for processing: %s", feed.Name)
		case <-s.ctx.Done():
			return
		default:
			log.Printf("Warning: job queue full, skipping feed: %s", feed.Name)
		}
	}

	// Update queue size stat
	s.mu.Lock()
	s.stats.QueueSize = len(s.jobQueue)
	s.mu.Unlock()
}

// worker processes feeds from the job queue
func (s *Scheduler) worker(id int) {
	defer s.wg.Done()
	log.Printf("Worker %d started", id)

	for {
		select {
		case feed, ok := <-s.jobQueue:
			if !ok {
				log.Printf("Worker %d: job queue closed, stopping", id)
				return
			}

			s.processFeed(id, feed)

		case <-s.ctx.Done():
			log.Printf("Worker %d: context cancelled, stopping", id)
			return
		}
	}
}

// processFeed processes a single feed
func (s *Scheduler) processFeed(workerID int, feed database.Feed) {
	log.Printf("Worker %d processing feed: %s (%s)", workerID, feed.Name, feed.URL)
	start := time.Now()

	err := s.processor.ProcessFeed(feed.ID, feed.ConfigFile)
	duration := time.Since(start)

	s.mu.Lock()
	defer s.mu.Unlock()

	// Update statistics
	s.stats.TotalProcessed++
	now := time.Now()
	s.stats.LastProcessedAt = &now
	s.stats.QueueSize = len(s.jobQueue)

	// Update average processing time
	s.stats.processTimes = append(s.stats.processTimes, duration)
	if len(s.stats.processTimes) > 100 {
		s.stats.processTimes = s.stats.processTimes[1:] // Keep only last 100
	}
	s.updateAverageProcessTime()

	if err != nil {
		s.stats.TotalErrors++
		log.Printf("Worker %d error processing feed %s: %v", workerID, feed.Name, err)
	} else {
		log.Printf("Worker %d successfully processed feed %s in %v", workerID, feed.Name, duration)
	}
}

// updateAverageProcessTime calculates the average processing time
func (s *Scheduler) updateAverageProcessTime() {
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
func (s *Scheduler) GetStats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	statsCopy := *s.stats
	statsCopy.QueueSize = len(s.jobQueue)
	return statsCopy
}

// TriggerImmediate triggers immediate processing of all due feeds
func (s *Scheduler) TriggerImmediate() {
	log.Println("Triggering immediate feed processing...")
	go s.enqueueDueFeeds()
}

// SetWorkerCount dynamically adjusts the number of workers (not implemented for simplicity)
// In a production environment, you might want to implement dynamic scaling
func (s *Scheduler) SetWorkerCount(count int) {
	log.Printf("Dynamic worker scaling not implemented, current workers: %d", s.workerCount)
}

// Health returns the health status of the scheduler
func (s *Scheduler) Health() map[string]interface{} {
	stats := s.GetStats()
	
	health := map[string]interface{}{
		"status":              "healthy",
		"workers":             stats.CurrentWorkers,
		"queue_size":          stats.QueueSize,
		"total_processed":     stats.TotalProcessed,
		"total_errors":        stats.TotalErrors,
		"average_process_time": stats.AverageProcessTime.String(),
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