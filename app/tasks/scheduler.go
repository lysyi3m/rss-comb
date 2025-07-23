package tasks

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/lysyi3m/rss-comb/app/cfg"
	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/feed"
)

var _ TaskSchedulerInterface = (*Scheduler)(nil)

type Scheduler struct {
	feedRepo         database.FeedRepository
	itemRepo         database.ItemRepository
	configCache      *feed.ConfigCache
	httpClient       *http.Client
	parser           *feed.Parser
	filterer         *feed.Filterer
	contentExtractor *feed.ContentExtractor
	userAgent        string
	interval         time.Duration
	workerCount      int
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	taskQueue        chan TaskInterface
}

func NewScheduler(configCache *feed.ConfigCache, feedRepo database.FeedRepository,
	itemRepo database.ItemRepository, httpClient *http.Client, parser *feed.Parser, filterer *feed.Filterer,
	contentExtractor *feed.ContentExtractor) TaskSchedulerInterface {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := cfg.Get()

	return &Scheduler{
		feedRepo:         feedRepo,
		itemRepo:         itemRepo,
		configCache:      configCache,
		httpClient:       httpClient,
		parser:           parser,
		filterer:         filterer,
		contentExtractor: contentExtractor,
		userAgent:        cfg.UserAgent,
		interval:         time.Duration(cfg.SchedulerInterval) * time.Second,
		workerCount:      cfg.WorkerCount,
		ctx:              ctx,
		cancel:           cancel,
		taskQueue:        make(chan TaskInterface, 300),
	}
}

func (s *Scheduler) Start() {
	for i := 0; i < s.workerCount; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		s.enqueueStartupTasks()

		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				s.enqueueTasks()
			}
		}
	}()

}

func (s *Scheduler) Stop() {
	s.cancel()
	s.wg.Wait()
	close(s.taskQueue)
}

func (s *Scheduler) EnqueueTask(task TaskInterface) error {
	select {
	case s.taskQueue <- task:
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	default:
		return fmt.Errorf("task queue is full")
	}
}

func (s *Scheduler) enqueueStartupTasks() {
	feedConfigs := s.configCache.GetConfigs()
	if len(feedConfigs) == 0 {
		slog.Debug("No feed configurations found")
		return
	}

	slog.Debug("Processing feed configurations", "count", len(feedConfigs))

	for _, feedConfig := range feedConfigs {
		syncTask := NewSyncFeedConfigTask(feedConfig.Name, feedConfig, s.feedRepo)
		if err := s.EnqueueTask(syncTask); err != nil {
			slog.Warn("Failed to enqueue SyncFeedConfigTask", "feed", feedConfig.Name, "error", err)
			continue
		}

    if !feedConfig.Settings.Enabled {
      slog.Debug("Feed disabled, skipping ProcessFeedTask", "feed", feedConfig.Name)
      continue
    }

    processTask := NewProcessFeedTask(feedConfig.Name, feedConfig, s.httpClient, s.parser, s.filterer, s.feedRepo, s.itemRepo, s.userAgent)
    if err := s.EnqueueTask(processTask); err != nil {
      slog.Warn("Failed to enqueue ProcessFeedTask", "feed", feedConfig.Name, "error", err)
    }
	}
}

func (s *Scheduler) enqueueTasks() {
	feedConfigs := s.configCache.GetEnabledConfigs()
	if len(feedConfigs) == 0 {
		slog.Debug("No enabled feed configurations found")
		return
	}

	slog.Debug("Processing enabled feed configurations for task scheduling", "count", len(feedConfigs))

	for _, feedConfig := range feedConfigs {
		feed, err := s.feedRepo.GetFeed(feedConfig.Name)
		if err != nil {
			slog.Warn("Failed to get feed from database, skipping", "feed", feedConfig.Name, "error", err)
			continue
		}
		if feed == nil {
			slog.Warn("Feed not found in database, skipping", "feed", feedConfig.Name)
			continue
		}

		now := time.Now().UTC()
		if feed.NextFetchAt != nil && feed.NextFetchAt.After(now) {
			slog.Debug("Feed not due for refresh yet", "feed", feedConfig.Name, "next_fetch_at", feed.NextFetchAt)
		} else {
			processTask := NewProcessFeedTask(feedConfig.Name, feedConfig, s.httpClient, s.parser, s.filterer, s.feedRepo, s.itemRepo, s.userAgent)
			if err := s.EnqueueTask(processTask); err != nil {
				slog.Warn("Failed to enqueue ProcessFeedTask", "feed", feedConfig.Name, "error", err)
			}
		}

		if feedConfig.Settings.ExtractContent {
			extractTask := NewExtractContentTask(feedConfig.Name, feedConfig, s.httpClient, s.contentExtractor, s.feedRepo, s.itemRepo, s.userAgent)
			if err := s.EnqueueTask(extractTask); err != nil {
				slog.Warn("Failed to enqueue ExtractContentTask", "feed", feedConfig.Name, "error", err)
			}
		}
	}
}

func (s *Scheduler) worker(id int) {
	defer s.wg.Done()

	for {
		select {
		case task, ok := <-s.taskQueue:
			if !ok {
				return
			}
			s.executeTask(id, task)

		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Scheduler) executeTask(workerID int, task TaskInterface) {
	task.Start()

	taskCtx, cancel := context.WithTimeout(s.ctx, 5*time.Minute)
	defer cancel()

	err := task.Execute(taskCtx)

	if err != nil {
		slog.Error("Worker task execution failed", "worker_id", workerID, "type", string(task.GetType()), "id", task.GetID(), "retry_count", task.GetRetryCount(), "error", err)

		if task.CanRetry() {
			task.IncrementRetryCount()
			retryDelay := time.Duration(1<<uint(task.GetRetryCount()-1)) * time.Second
			if retryDelay > 30*time.Second {
				retryDelay = 30 * time.Second
			}

			slog.Warn("Task retry scheduled", "type", string(task.GetType()), "feed", task.GetFeedName(), "retry_count", task.GetRetryCount(), "max_retries", task.GetMaxRetries(), "delay", retryDelay.String())

			go func() {
				time.Sleep(retryDelay)
				select {
				case <-s.ctx.Done():
					slog.Debug("Scheduler stopped, skipping task retry", "type", string(task.GetType()), "id", task.GetID())
					return
				default:
					if retryErr := s.EnqueueTask(task); retryErr != nil {
						slog.Error("Failed to re-enqueue task for retry", "type", string(task.GetType()), "id", task.GetID(), "retry_count", task.GetRetryCount(), "error", retryErr)
					}
				}
			}()
		} else {
			slog.Error("Task failed after maximum retries", "type", string(task.GetType()), "id", task.GetID(), "retry_count", task.GetRetryCount(), "max_retries", task.GetMaxRetries(), "last_error", err)
		}
	}
}
