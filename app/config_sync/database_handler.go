package config_sync

import (
	"fmt"
	"log"
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

// OnConfigUpdate implements the ConfigUpdateHandler interface with comprehensive handling
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
			log.Printf("Database sync warning: Config file no longer exists: %s", relPath)
			return nil // Don't treat as an error, file might have been moved/renamed
		}
		return fmt.Errorf("failed to stat config file %s: %w", relPath, err)
	}

	// Validate configuration before processing
	if err := config.ValidateConfig(cfg); err != nil {
		log.Printf("Database sync error: Invalid configuration in %s: %v", relPath, err)
		return fmt.Errorf("invalid configuration in %s: %w", relPath, err)
	}

	// Register or update the feed in the database
	dbID, urlChanged, err := h.feedRepo.UpsertFeedWithChangeDetection(
		filePath, cfg.Feed.ID, cfg.Feed.URL, cfg.Feed.Title)
	if err != nil {
		log.Printf("Database sync error: Failed to register feed %s: %v", relPath, err)
		return fmt.Errorf("failed to register feed %s: %w", relPath, err)
	}

	// Log the operation
	if urlChanged {
		log.Printf("Database sync: Feed updated - %s (ID: %s, DB ID: %s, New URL: %s)",
			cfg.Feed.Title, cfg.Feed.ID, dbID, cfg.Feed.URL)
	} else {
		log.Printf("Database sync: Feed registered - %s (ID: %s, DB ID: %s)",
			cfg.Feed.Title, cfg.Feed.ID, dbID)
	}

	// For immediate processing, we'll reset the next_fetch time to NULL
	// which will cause the scheduler to pick it up in the next cycle
	if cfg.Settings.Enabled {
		// Reset next_fetch to NULL to trigger immediate processing
		if err := h.feedRepo.UpdateNextFetch(dbID, time.Time{}); err != nil {
			log.Printf("Database sync warning: Failed to reset next_fetch for immediate processing: %v", err)
		} else {
			log.Printf("Database sync: Feed scheduled for immediate processing: %s", cfg.Feed.Title)
		}
	} else {
		log.Printf("Database sync: Skipping immediate processing for disabled feed: %s", cfg.Feed.Title)
	}

	return nil
}

// handleConfigDeletion handles deletion of configuration files
func (h *DatabaseSyncHandler) handleConfigDeletion(filePath, relPath string, cfg *config.FeedConfig) error {
	log.Printf("Database sync: Processing deletion of config file: %s (ID: %s)", relPath, cfg.Feed.ID)

	// Find the feed in the database by feed ID
	dbFeed, err := h.feedRepo.GetFeedByID(cfg.Feed.ID)
	if err != nil {
		log.Printf("Database sync error: Failed to find feed %s in database: %v", cfg.Feed.ID, err)
		return fmt.Errorf("failed to find feed %s in database: %w", cfg.Feed.ID, err)
	}

	if dbFeed == nil {
		log.Printf("Database sync warning: Feed %s not found in database (already deleted?)", cfg.Feed.ID)
		return nil // Feed doesn't exist in database, nothing to do
	}

	// Disable the feed in the database (preserving data)
	if err := h.feedRepo.SetFeedEnabled(dbFeed.ID, false); err != nil {
		log.Printf("Database sync error: Failed to disable feed %s: %v", cfg.Feed.ID, err)
		return fmt.Errorf("failed to disable feed %s: %w", cfg.Feed.ID, err)
	}

	log.Printf("Database sync: Feed disabled in database - %s (ID: %s, DB ID: %s)",
		cfg.Feed.Title, cfg.Feed.ID, dbFeed.ID)
	log.Printf("Database sync: Feed data preserved in database for potential restoration")

	return nil
}