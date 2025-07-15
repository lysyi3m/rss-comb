package config_sync

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/lysyi3m/rss-comb/app/config"
)

// DatabaseSyncHandler handles synchronizing configuration changes with the database
type DatabaseSyncHandler struct {
	feedRepo FeedSyncRepository
	feedsDir string
}

// NewDatabaseSyncHandler creates a new database sync handler
func NewDatabaseSyncHandler(feedRepo FeedSyncRepository, feedsDir string) *DatabaseSyncHandler {
	return &DatabaseSyncHandler{
		feedRepo: feedRepo,
		feedsDir: feedsDir,
	}
}

// OnConfigUpdate synchronizes configuration changes with the database
func (h *DatabaseSyncHandler) OnConfigUpdate(filePath string, cfg *config.FeedConfig, isDelete bool) error {
	relPath, _ := filepath.Rel(h.feedsDir, filePath)

	if isDelete {
		return h.handleConfigDeletion(filePath, relPath, cfg)
	}

	return h.handleConfigUpsert(filePath, relPath, cfg)
}

// handleConfigUpsert handles creation and modification of configuration files
func (h *DatabaseSyncHandler) handleConfigUpsert(filePath, relPath string, cfg *config.FeedConfig) error {
	// Validate that the file still exists (handles edge cases)
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			slog.Warn("Config file no longer exists", "file", relPath)
			return nil // Don't treat as an error, file might have been moved/renamed
		}
		return fmt.Errorf("failed to stat config file %s: %w", relPath, err)
	}

	// Validate configuration before processing
	if err := config.ValidateConfig(cfg); err != nil {
		slog.Error("Invalid configuration", "file", relPath, "error", err)
		return fmt.Errorf("invalid configuration in %s: %w", relPath, err)
	}

	// Register or update the feed in the database
	dbID, urlChanged, err := h.feedRepo.UpsertFeedWithChangeDetection(
		filePath, cfg.Feed.ID, cfg.Feed.URL, cfg.Feed.Title)
	if err != nil {
		slog.Error("Failed to register feed", "file", relPath, "error", err)
		return fmt.Errorf("failed to register feed %s: %w", relPath, err)
	}

	// Log the operation
	if urlChanged {
		slog.Info("Feed updated", "title", cfg.Feed.Title, "feed_id", cfg.Feed.ID, "db_id", dbID, "new_url", cfg.Feed.URL)
	}

	// Schedule for processing by resetting next_fetch time to NULL
	// which will cause the scheduler to pick it up in the next cycle
	if cfg.Settings.Enabled {
		// Reset next_fetch to NULL to schedule for processing
		if err := h.feedRepo.UpdateNextFetch(dbID, time.Time{}); err != nil {
			slog.Warn("Failed to schedule feed for processing", "error", err)
		}
	} else {
		slog.Info("Feed disabled, skipping processing schedule", "title", cfg.Feed.Title)
	}

	return nil
}

// handleConfigDeletion handles deletion of configuration files
func (h *DatabaseSyncHandler) handleConfigDeletion(filePath, relPath string, cfg *config.FeedConfig) error {
	slog.Info("Processing deletion of config file", "file", relPath, "feed_id", cfg.Feed.ID)

	// Find the feed in the database by feed ID
	dbFeed, err := h.feedRepo.GetFeedByID(cfg.Feed.ID)
	if err != nil {
		slog.Error("Failed to find feed in database", "feed_id", cfg.Feed.ID, "error", err)
		return fmt.Errorf("failed to find feed %s in database: %w", cfg.Feed.ID, err)
	}

	if dbFeed == nil {
		slog.Warn("Feed not found in database (already deleted?)", "feed_id", cfg.Feed.ID)
		return nil // Feed doesn't exist in database, nothing to do
	}

	// Disable the feed in the database (preserving data)
	if err := h.feedRepo.SetFeedEnabled(dbFeed.ID, false); err != nil {
		slog.Error("Failed to disable feed", "feed_id", cfg.Feed.ID, "error", err)
		return fmt.Errorf("failed to disable feed %s: %w", cfg.Feed.ID, err)
	}

	slog.Info("Feed disabled in database", "title", cfg.Feed.Title, "feed_id", cfg.Feed.ID, "db_id", dbFeed.ID)
	slog.Info("Feed data preserved in database for potential restoration")

	return nil
}
