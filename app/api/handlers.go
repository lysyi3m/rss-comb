package api

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/feed"
	"github.com/lysyi3m/rss-comb/app/tasks"
)

func NewHandler(configCache *feed.ConfigCache, feedRepo database.FeedRepository,
	itemRepo database.ItemRepository, filterer *feed.Filterer,
	scheduler tasks.TaskSchedulerInterface) *Handler {
	return &Handler{
		feedRepo:    feedRepo,
		itemRepo:    itemRepo,
		generator:   feed.NewGenerator(),
		configCache: configCache,
		filterer:    filterer,
		scheduler:   scheduler,
	}
}

func (h *Handler) GetFeed(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	feedConfig, err := h.configCache.GetConfig(name)
	if err != nil {
		slog.Error("Feed configuration not found", "feed", name, "error", err)
		c.Status(http.StatusNotFound)
		return
	}

	feed, err := h.feedRepo.GetFeed(name)
	if err != nil {
		slog.Error("Database error", "operation", "get_feed", "feed", name, "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	if feed == nil {
		slog.Error("Feed not found in database", "feed", name)
		c.Status(http.StatusNotFound)
		return
	}

	items, err := h.itemRepo.GetVisibleItems(name, feedConfig.Settings.MaxItems)
	if err != nil {
		slog.Error("Database error", "operation", "get_items", "feed", name, "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	rss, err := h.generator.Run(*feed, items)
	if err != nil {
		slog.Error("RSS generation error", "feed", name, "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Header("X-Feed-Items", strconv.Itoa(len(items)))
	c.Header("X-Feed-Name", name)
	c.Header("X-Last-Updated", feed.UpdatedAt.Format(time.RFC3339))

	c.String(http.StatusOK, rss)
}

func (h *Handler) GetHealth(c *gin.Context) {
	health := map[string]interface{}{
		"timestamp": time.Now().In(time.Local).Format(time.RFC3339),
	}

	if feedCount, err := h.feedRepo.GetFeedCount(); err == nil {
		health["feeds"] = feedCount
	}

	health["loaded_configurations"] = h.configCache.GetConfigCount()

	c.JSON(http.StatusOK, health)
}

func (h *Handler) APIListFeeds(c *gin.Context) {
	configs := h.configCache.GetConfigs()

	feeds := make([]map[string]interface{}, 0, len(configs))

	for _, feedConfig := range configs {
		feedInfo := map[string]interface{}{
			"name":             feedConfig.Name,
			"url":              feedConfig.URL,
			"title":            "",
			"enabled":          feedConfig.Settings.Enabled,
			"max_items":        feedConfig.Settings.MaxItems,
			"refresh_interval": (time.Duration(feedConfig.Settings.RefreshInterval) * time.Second).String(),
			"filters":          len(feedConfig.Filters),
		}

		if feed, err := h.feedRepo.GetFeed(feedConfig.Name); err == nil && feed != nil {
			feedInfo["title"] = feed.Title
			feedInfo["last_fetched_at"] = feed.LastFetchedAt
			feedInfo["next_fetch_at"] = feed.NextFetchAt
			feedInfo["updated_at"] = feed.UpdatedAt
		}

		if itemCount, err := h.itemRepo.GetItemCount(feedConfig.Name); err == nil {
			feedInfo["item_count"] = itemCount
		}

		feeds = append(feeds, feedInfo)
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"feeds": feeds,
		"total": len(feeds),
	})
}

func (h *Handler) APIGetFeedDetails(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing feed name parameter"})
		return
	}

	feedConfig, err := h.configCache.GetConfig(name)
	if err != nil {
		slog.Error("Feed configuration not found", "feed", name, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Feed configuration not found"})
		return
	}

	feed, err := h.feedRepo.GetFeed(name)
	if err != nil {
		slog.Error("Database error", "operation", "get_feed", "feed", name, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if feed == nil {
		slog.Error("Feed not found in database", "feed", name)
		c.JSON(http.StatusNotFound, gin.H{"error": "Feed not found in database"})
		return
	}

	details := map[string]interface{}{
		"name":             name,
		"url":              feedConfig.URL,
		"title":            feed.Title,
		"enabled":          feedConfig.Settings.Enabled,
		"max_items":        feedConfig.Settings.MaxItems,
		"refresh_interval": (time.Duration(feedConfig.Settings.RefreshInterval) * time.Second).String(),
		"timeout":          (time.Duration(feedConfig.Settings.Timeout) * time.Second).String(),
		"filters":          feedConfig.Filters,
	}

	details["database"] = map[string]interface{}{
		"id":              feed.ID,
		"name":            feed.Name,
		"last_fetched_at": feed.LastFetchedAt,
		"next_fetch_at":   feed.NextFetchAt,
		"created_at":      feed.CreatedAt,
		"updated_at":      feed.UpdatedAt,
	}

	if total, visible, filtered, err := h.itemRepo.GetItemStats(name); err == nil {
		details["items"] = map[string]interface{}{
			"total":    total,
			"visible":  visible,
			"filtered": filtered,
		}
	}

	c.JSON(http.StatusOK, details)
}

func (h *Handler) APIReloadFeed(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing feed name parameter"})
		return
	}

	_, err := h.configCache.GetConfig(name)
	if err != nil {
		slog.Error("Feed configuration not found", "feed", name, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Feed configuration not found"})
		return
	}

	feed, err := h.feedRepo.GetFeed(name)
	if err != nil {
		slog.Error("Database error", "operation", "get_feed", "feed", name, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if feed == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Feed not found in database"})
		return
	}

	feedConfig, err := h.configCache.LoadConfig(name)
	if err != nil {
		slog.Error("Error reloading configuration", "feed", name, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to reload configuration",
			"details": err.Error(),
		})
		return
	}

	syncFeedTask := tasks.NewSyncFeedConfigTask(name, feedConfig, h.feedRepo)
	err = h.scheduler.EnqueueTask(syncFeedTask)
	if err != nil {
		slog.Error("Error enqueueing sync task", "feed", name, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to enqueue sync task",
			"details": err.Error(),
		})
		return
	}

	refilterFeedTask := tasks.NewRefilterFeedTask(name, feedConfig, h.filterer, h.feedRepo, h.itemRepo)
	err = h.scheduler.EnqueueTask(refilterFeedTask)
	if err != nil {
		slog.Error("Error enqueueing refilter task", "feed", name, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to enqueue refilter task",
			"details": err.Error(),
		})
		return
	}

	response := gin.H{
		"success": true,
		"message": "Configuration reloaded and tasks enqueued successfully",
		"feed": gin.H{
			"name":  name,
			"title": feed.Title,
			"url":   feedConfig.URL,
		},
		"tasks": []gin.H{
			{
				"id":   syncFeedTask.ID,
				"type": syncFeedTask.Type,
			},
			{
				"id":   refilterFeedTask.ID,
				"type": refilterFeedTask.Type,
			},
		},
	}

	c.JSON(http.StatusOK, response)
}
