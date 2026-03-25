package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/lysyi3m/rss-comb/app/database"
)

type Scheduler struct {
	interval time.Duration
	feedRepo *database.FeedRepository
	jobRepo  *database.JobRepository
}

func NewScheduler(interval time.Duration, feedRepo *database.FeedRepository, jobRepo *database.JobRepository) *Scheduler {
	return &Scheduler{
		interval: interval,
		feedRepo: feedRepo,
		jobRepo:  jobRepo,
	}
}

// Run starts the scheduler loop. It creates fetch_feed jobs for due feeds
// and resets stale jobs on each tick. Blocks until ctx is cancelled.
func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	slog.Info("Scheduler started", "interval", s.interval)

	// Immediate tick on startup — don't wait for the first interval
	s.tick()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Scheduler stopped")
			return
		case <-ticker.C:
			s.tick()
		}
	}
}

func (s *Scheduler) tick() {
	feeds, err := s.feedRepo.GetDueFeeds()
	if err != nil {
		slog.Error("Scheduler failed to get due feeds", "error", err)
		return
	}

	for _, f := range feeds {
		if _, err := s.jobRepo.CreateJob("fetch_feed", f.ID, nil, 0); err != nil {
			slog.Error("Scheduler failed to create fetch_feed job", "feed", f.Name, "error", err)
		}
	}

	resetCount, err := s.jobRepo.ResetStaleJobs(10 * time.Minute)
	if err != nil {
		slog.Error("Scheduler failed to reset stale jobs", "error", err)
		return
	}
	if resetCount > 0 {
		slog.Warn("Reset stale jobs", "count", resetCount)
	}
}
