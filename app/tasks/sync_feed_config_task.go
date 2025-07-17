package tasks

import (
	"context"
	"fmt"
	"log/slog"

	"time"

	"github.com/lysyi3m/rss-comb/app/feed_config"
	"github.com/lysyi3m/rss-comb/app/database"
)

// SyncFeedConfigTask represents a task to sync feed configuration to database
type SyncFeedConfigTask struct {
	BaseTask
	ConfigFile   string
	FeedConfig   *feed_config.FeedConfig
	feedRepo     database.FeedRepository
}

// NewSyncFeedConfigTask creates a new sync feed config task
func NewSyncFeedConfigTask(configFile string, feedConfig *feed_config.FeedConfig, feedRepo database.FeedRepository) *SyncFeedConfigTask {
	description := fmt.Sprintf("Sync config for feed %s (%s)", feedConfig.Feed.ID, feedConfig.Feed.Title)
	
	return &SyncFeedConfigTask{
		BaseTask:   NewBaseTask(feedConfig.Feed.ID+"-sync", TaskTypeSyncFeedConfig, PriorityTop, description, feedConfig.Feed.ID),
		ConfigFile: configFile,
		FeedConfig: feedConfig,
		feedRepo:   feedRepo,
	}
}

// Execute syncs the feed configuration to the database
func (t *SyncFeedConfigTask) Execute(ctx context.Context) error {
	slog.Debug("Task started", "type", "SyncFeedConfig", "feed_id", t.GetFeedID())
	
	// Fast-fail on cancellation to avoid unnecessary work
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	// Sync feed metadata to database (URL, title, enabled status)
	dbID, urlChanged, err := t.feedRepo.UpsertFeedWithChangeDetection(
		t.ConfigFile, t.FeedConfig.Feed.ID, t.FeedConfig.Feed.URL, t.FeedConfig.Feed.Title)
	if err != nil {
		slog.Error("Task failed", "type", "SyncFeedConfig", "feed_id", t.GetFeedID(), "error", err)
		return fmt.Errorf("failed to sync feed config to database: %w", err)
	}
	
	// Update enabled status
	if err := t.feedRepo.SetFeedEnabled(dbID, t.FeedConfig.Settings.Enabled); err != nil {
		slog.Warn("Failed to update feed enabled status", "feed_id", t.GetFeedID(), "enabled", t.FeedConfig.Settings.Enabled, "error", err)
	}
	
	// If URL changed, reset next_fetch_at to trigger immediate processing
	if urlChanged {
		if err := t.feedRepo.UpdateNextFetch(dbID, time.Time{}); err != nil {
			slog.Warn("Failed to reset next fetch time after URL change", "feed_id", t.GetFeedID(), "error", err)
		} else {
			slog.Info("Reset next fetch time due to URL change", "feed_id", t.GetFeedID())
		}
	}
	
	slog.Debug("Task completed", "type", "SyncFeedConfig", "feed_id", t.GetFeedID(), "url_changed", urlChanged)
	return nil
}