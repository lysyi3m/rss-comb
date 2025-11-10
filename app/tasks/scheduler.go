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
	feedRepo         *database.FeedRepository
	itemRepo         *database.ItemRepository
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

func NewScheduler(feedRepo *database.FeedRepository,
	itemRepo *database.ItemRepository, httpClient *http.Client, parser *feed.Parser, filterer *feed.Filterer,
	contentExtractor *feed.ContentExtractor) TaskSchedulerInterface {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := cfg.Get()

	return &Scheduler{
		feedRepo:         feedRepo,
		itemRepo:         itemRepo,
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

		s.enqueueTasks()

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

func (s *Scheduler) enqueueTasks() {
	feeds, err := s.feedRepo.GetEnabledFeedsForScheduling()
	if err != nil {
		slog.Error("Failed to get enabled feeds from database", "error", err)
		return
	}

	if len(feeds) == 0 {
		return
	}

	now := time.Now().UTC()
	for _, feed := range feeds {
		if feed.NextFetchAt != nil && feed.NextFetchAt.After(now) {
			continue
		}

		processTask := NewProcessFeedTask(feed.Name, s.httpClient, s.parser, s.filterer, s.contentExtractor, s.feedRepo, s.itemRepo, s.userAgent)
		if err := s.EnqueueTask(processTask); err != nil {
			slog.Warn("Failed to enqueue ProcessFeedTask", "feed", feed.Name, "error", err)
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
		slog.Error("Task execution failed",
			"worker_id", workerID,
			"type", string(task.GetType()),
			"feed", task.GetFeedName(),
			"error", err)
	}
}
