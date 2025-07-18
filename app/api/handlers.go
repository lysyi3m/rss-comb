package api

import (
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lysyi3m/rss-comb/app/config"
	"github.com/lysyi3m/rss-comb/app/feed_config"
	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/feed"
	"github.com/lysyi3m/rss-comb/app/tasks"
)

// NewHandler creates a new API handler
func NewHandler(configCache *feed_config.ConfigCacheHandler, feedRepo database.FeedRepository, 
	itemRepo database.ItemRepository, processor tasks.ProcessorInterface,
	taskScheduler tasks.TaskSchedulerInterface) *Handler {
	cfg := config.Get()
	return &Handler{
		feedRepo:    feedRepo,
		itemRepo:    itemRepo,
		generator:   feed.NewGenerator(cfg.GetPort()),
		configCache: configCache,
		processor:   processor,
		scheduler:   taskScheduler,
		userAgent:   cfg.GetUserAgent(),
	}
}

// GetFeedByID handles the new ID-based feed endpoint
func (h *Handler) GetFeedByID(c *gin.Context) {
	feedID := c.Param("id")
	if feedID == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	// Find matching configuration by feed ID
	feedConfig, found := h.configCache.GetConfigByFeedID(feedID)
	if !found {
		c.Status(http.StatusNotFound)
		return
	}

	// Get feed from database
	feed, err := h.feedRepo.GetFeedByID(feedID)
	if err != nil {
		slog.Error("Database error", "operation", "get_feed", "feed_id", feedID, "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	// If feed not found in database, return 202 Accepted (feed registered but not processed yet)
	if feed == nil {
		c.Status(http.StatusAccepted)
		return
	}

	// Get feed items
	items, err := h.itemRepo.GetVisibleItems(feed.ID, feedConfig.Settings.MaxItems)
	if err != nil {
		slog.Error("Database error", "operation", "get_items", "feed_id", feedID, "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	// Generate RSS
	rss, err := h.generator.Generate(*feed, items)
	if err != nil {
		slog.Error("RSS generation error", "feed_id", feedID, "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	// Set response headers
	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Header("X-Feed-Items", strconv.Itoa(len(items)))
	c.Header("X-Feed-ID", feedID)

	// Use UpdatedAt as it tracks last successful processing
	c.Header("X-Last-Updated", feed.UpdatedAt.Format(time.RFC3339))

	slog.Debug("Served feed", "feed_id", feedID, "item_count", len(items))
	c.String(http.StatusOK, rss)
}

// GetHealth handles the health endpoint
func (h *Handler) GetHealth(c *gin.Context) {
	health := map[string]interface{}{
		"timestamp": time.Now().In(time.Local).Format(time.RFC3339),
	}

	// Get enabled feed count
	if enabledFeedCount, err := h.feedRepo.GetEnabledFeedCount(); err == nil {
		health["enabled_feeds"] = enabledFeedCount
	}

	health["loaded_configurations"] = h.configCache.GetConfigCount()

	c.JSON(http.StatusOK, health)
}

// APIListFeeds handles listing all configured feeds
func (h *Handler) APIListFeeds(c *gin.Context) {
	configs := h.configCache.GetAllConfigs()
	feeds := make([]map[string]interface{}, 0, len(configs))

	for configFile, feedConfig := range configs {
		feedInfo := map[string]interface{}{
			"name":            feedConfig.Feed.Title,
			"url":             feedConfig.Feed.URL,
			"config_file":     configFile,
			"enabled":         feedConfig.Settings.Enabled,
			"max_items":       feedConfig.Settings.MaxItems,
			"refresh_interval": feedConfig.Settings.GetRefreshInterval().String(),
			"filters":         len(feedConfig.Filters),
		}

		// Get feed from database if available
		if feed, err := h.feedRepo.GetFeedByID(feedConfig.Feed.ID); err == nil && feed != nil {
			feedInfo["last_fetched"] = feed.LastFetchedAt
			feedInfo["next_fetch"] = feed.NextFetchAt
			feedInfo["updated_at"] = feed.UpdatedAt

			// Get item count
			if itemCount, err := h.itemRepo.GetItemCount(feed.ID); err == nil {
				feedInfo["item_count"] = itemCount
			}
		}

		feeds = append(feeds, feedInfo)
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"feeds": feeds,
		"total": len(feeds),
	})
}

// APIGetFeedDetailsByID handles detailed information about a specific feed by ID
func (h *Handler) APIGetFeedDetailsByID(c *gin.Context) {
	feedID := c.Param("id")
	if feedID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing feed ID parameter"})
		return
	}

	// Find configuration by feed ID
	feedConfig, configFile, found := h.configCache.GetConfigAndFileByFeedID(feedID)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Feed not configured"})
		return
	}

	details := map[string]interface{}{
		"id":               feedConfig.Feed.ID,
		"name":             feedConfig.Feed.Title,
		"url":              feedConfig.Feed.URL,
		"config_file":      configFile,
		"enabled":          feedConfig.Settings.Enabled,
		"max_items":        feedConfig.Settings.MaxItems,
		"refresh_interval": feedConfig.Settings.GetRefreshInterval().String(),
		"timeout":          feedConfig.Settings.GetTimeout().String(),
		"user_agent":       h.userAgent,
		"filters":          feedConfig.Filters,
	}

	// Get feed from database
	if feed, err := h.feedRepo.GetFeedByID(feedID); err == nil && feed != nil {
		details["database"] = map[string]interface{}{
			"id":           feed.ID,
			"feed_id":      feed.FeedID,
			"last_fetched": feed.LastFetchedAt,
			"next_fetch":   feed.NextFetchAt,
			"enabled":      feed.IsEnabled,
			"created_at":   feed.CreatedAt,
			"updated_at":   feed.UpdatedAt,
		}

		// Get item statistics
		if total, visible, filtered, err := h.itemRepo.GetItemStats(feed.ID); err == nil {
			details["items"] = map[string]interface{}{
				"total":    total,
				"visible":  visible,
				"filtered": filtered,
			}
		}
	}

	c.JSON(http.StatusOK, details)
}

// APIReloadFeedByID handles the feed reload endpoint by feed ID
func (h *Handler) APIReloadFeedByID(c *gin.Context) {
	feedID := c.Param("id")
	if feedID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing feed ID parameter"})
		return
	}

	// Get feed from database (includes config file path)
	feed, err := h.feedRepo.GetFeedByID(feedID)
	if err != nil {
		slog.Error("Database error", "operation", "get_feed", "feed_id", feedID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if feed == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Feed not found in database"})
		return
	}

	// Reload configuration from file using appConfig file path from database
	configLoader := feed_config.NewLoader(filepath.Dir(feed.ConfigFile))
	updatedConfig, err := configLoader.Load(feed.ConfigFile)
	if err != nil {
		slog.Error("Error reloading configuration", "config_file", feed.ConfigFile, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to reload configuration",
			"details": err.Error(),
		})
		return
	}

	// Update the config cache with the reloaded configuration
	h.configCache.OnConfigUpdate(feed.ConfigFile, updatedConfig, false)

	// Create and enqueue SyncFeedConfigTask (Priority Top) to sync configuration changes
	syncTask := tasks.NewSyncFeedConfigTask(feed.ConfigFile, updatedConfig, h.feedRepo)
	err = h.scheduler.EnqueueTask(syncTask)
	if err != nil {
		slog.Error("Error enqueueing sync task", "feed_id", feedID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to enqueue sync task",
			"details": err.Error(),
		})
		return
	}

	// Create and enqueue RefilterFeedTask (Priority High) to reapply filters
	refilterTask := tasks.NewRefilterFeedTask(feed.ID, updatedConfig, h.processor)
	err = h.scheduler.EnqueueTask(refilterTask)
	if err != nil {
		slog.Error("Error enqueueing refilter task", "feed_id", feedID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to enqueue refilter task",
			"details": err.Error(),
		})
		return
	}

	response := gin.H{
		"success": true,
		"message": "Configuration reloaded and tasks enqueued successfully",
		"feed": gin.H{
			"id":   updatedConfig.Feed.ID,
			"name": updatedConfig.Feed.Title,
			"url":  updatedConfig.Feed.URL,
		},
		"tasks": []gin.H{
			{
				"id":          syncTask.GetID(),
				"type":        syncTask.GetType(),
				"priority":    syncTask.GetPriority(),
				"description": syncTask.GetDescription(),
				"created_at":  syncTask.GetCreatedAt().Format(time.RFC3339),
			},
			{
				"id":          refilterTask.GetID(),
				"type":        refilterTask.GetType(),
				"priority":    refilterTask.GetPriority(),
				"description": refilterTask.GetDescription(),
				"created_at":  refilterTask.GetCreatedAt().Format(time.RFC3339),
			},
		},
		"config_reloaded": true,
	}

	slog.Info("Successfully reloaded configuration and enqueued sync and refilter tasks", "feed_id", feedID, "config_file", feed.ConfigFile)

	c.JSON(http.StatusOK, response)
}
