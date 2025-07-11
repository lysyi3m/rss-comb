package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/lysyi3m/rss-comb/app/database"
)

// DatabaseSyncHandler handles synchronizing configuration changes with the database
type DatabaseSyncHandler struct {
	feedRepo database.FeedRepositoryInterface
	feedsDir string
}

// NewDatabaseSyncHandler creates a new database sync handler
func NewDatabaseSyncHandler(feedRepo database.FeedRepositoryInterface, feedsDir string) *DatabaseSyncHandler {
	return &DatabaseSyncHandler{
		feedRepo: feedRepo,
		feedsDir: feedsDir,
	}
}

// OnConfigUpdate implements the ConfigUpdateHandler interface with comprehensive handling
func (h *DatabaseSyncHandler) OnConfigUpdate(filePath string, config *FeedConfig, isDelete bool) error {
	relPath, _ := filepath.Rel(h.feedsDir, filePath)
	
	if isDelete {
		return h.handleConfigDeletion(filePath, relPath, config)
	}
	
	return h.handleConfigUpsert(filePath, relPath, config)
}

// handleConfigUpsert handles creation and modification of configuration files
func (h *DatabaseSyncHandler) handleConfigUpsert(filePath, relPath string, config *FeedConfig) error {
	// Validate that the file still exists (handles edge cases)
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			log.Printf("Database sync warning: Config file no longer exists: %s", relPath)
			return nil // Don't treat as an error, file might have been moved/renamed
		}
		return fmt.Errorf("failed to stat config file %s: %w", relPath, err)
	}

	// Validate configuration before processing
	if err := ValidateConfig(config); err != nil {
		log.Printf("Database sync error: Invalid configuration in %s: %v", relPath, err)
		return fmt.Errorf("invalid configuration in %s: %w", relPath, err)
	}

	// Register or update the feed in the database
	dbID, urlChanged, err := h.feedRepo.UpsertFeedWithChangeDetection(
		filePath, config.Feed.ID, config.Feed.URL, config.Feed.Title)
	if err != nil {
		log.Printf("Database sync error: Failed to register feed %s: %v", relPath, err)
		return fmt.Errorf("failed to register feed %s: %w", relPath, err)
	}

	// Log the operation
	if urlChanged {
		log.Printf("Database sync: Feed updated - %s (ID: %s, DB ID: %s, New URL: %s)", 
			config.Feed.Title, config.Feed.ID, dbID, config.Feed.URL)
	} else {
		log.Printf("Database sync: Feed registered - %s (ID: %s, DB ID: %s)", 
			config.Feed.Title, config.Feed.ID, dbID)
	}

	// For immediate processing, we'll reset the next_fetch time to NULL
	// which will cause the scheduler to pick it up in the next cycle
	if config.Settings.Enabled {
		// Reset next_fetch to NULL to trigger immediate processing
		if err := h.feedRepo.UpdateNextFetch(dbID, time.Time{}); err != nil {
			log.Printf("Database sync warning: Failed to reset next_fetch for immediate processing: %v", err)
		} else {
			log.Printf("Database sync: Feed scheduled for immediate processing: %s", config.Feed.Title)
		}
	} else {
		log.Printf("Database sync: Skipping immediate processing for disabled feed: %s", config.Feed.Title)
	}

	return nil
}

// handleConfigDeletion handles deletion of configuration files
func (h *DatabaseSyncHandler) handleConfigDeletion(filePath, relPath string, config *FeedConfig) error {
	log.Printf("Database sync: Processing deletion of config file: %s (ID: %s)", relPath, config.Feed.ID)

	// Find the feed in the database by feed ID
	dbFeed, err := h.feedRepo.GetFeedByID(config.Feed.ID)
	if err != nil {
		log.Printf("Database sync error: Failed to find feed %s in database: %v", config.Feed.ID, err)
		return fmt.Errorf("failed to find feed %s in database: %w", config.Feed.ID, err)
	}

	if dbFeed == nil {
		log.Printf("Database sync warning: Feed %s not found in database (already deleted?)", config.Feed.ID)
		return nil // Feed doesn't exist in database, nothing to do
	}

	// Disable the feed in the database (preserving data)
	if err := h.feedRepo.SetFeedEnabled(dbFeed.ID, false); err != nil {
		log.Printf("Database sync error: Failed to disable feed %s: %v", config.Feed.ID, err)
		return fmt.Errorf("failed to disable feed %s: %w", config.Feed.ID, err)
	}

	log.Printf("Database sync: Feed disabled in database - %s (ID: %s, DB ID: %s)", 
		config.Feed.Title, config.Feed.ID, dbFeed.ID)
	log.Printf("Database sync: Feed data preserved in database for potential restoration")

	return nil
}


