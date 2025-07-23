package tasks

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/feed"
)

type SyncFeedConfigTask struct {
	Task
	FeedName   string
	FeedConfig *feed.Config
	feedRepo   database.FeedRepository
}

func NewSyncFeedConfigTask(feedName string, feedConfig *feed.Config, feedRepo database.FeedRepository) *SyncFeedConfigTask {
	return &SyncFeedConfigTask{
		Task:       NewTask(TaskTypeSyncFeedConfig, feedName),
		FeedName:   feedName,
		FeedConfig: feedConfig,
		feedRepo:   feedRepo,
	}
}

func (t *SyncFeedConfigTask) Execute(ctx context.Context) error {

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	err := t.feedRepo.UpsertFeed(
		t.FeedConfig.Name,
		t.FeedConfig.URL)
	if err != nil {
		slog.Error("Task failed", "type", "SyncFeedConfig", "feed", t.FeedName, "error", err)
		return fmt.Errorf("failed to sync feed config to database: %w", err)
	}

  slog.Info("Task completed",
    "type", "SyncFeedConfig",
    "feed", t.FeedName,
    "duration", t.GetDuration())

	return nil
}
