package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lysyi3m/rss-comb/app/cfg"
	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/feed"
	"github.com/lysyi3m/rss-comb/app/tasks"
)

func NewHandler(
	cfg *cfg.Cfg,
	feedRepo *database.FeedRepository,
	itemRepo *database.ItemRepository,
	filterer *feed.Filterer,
	scheduler tasks.TaskSchedulerInterface,
) *Handler {
	return &Handler{
		cfg:       cfg,
		feedRepo:  feedRepo,
		itemRepo:  itemRepo,
		filterer:  filterer,
		scheduler: scheduler,
	}
}

func (h *Handler) GetHealth(c *gin.Context) {
	health := map[string]interface{}{
		"timestamp": time.Now().In(time.Local).Format(time.RFC3339),
	}

	if feedCount, err := h.feedRepo.GetFeedCount(); err == nil {
		health["feeds"] = feedCount
	}

	c.JSON(http.StatusOK, health)
}

func (h *Handler) APIGetFeed(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing feed name parameter"})
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

	settings, err := feed.GetSettings()
	if err != nil {
		slog.Error("Failed to get feed settings", "feed", name, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get feed settings"})
		return
	}

	filters, err := feed.GetFilters()
	if err != nil {
		slog.Error("Failed to get feed filters", "feed", name, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get feed filters"})
		return
	}

	details := map[string]interface{}{
		"name":             name,
		"url":              feed.FeedURL,
		"title":            feed.Title,
		"enabled":          feed.IsEnabled,
		"max_items":        settings.MaxItems,
		"refresh_interval": (time.Duration(settings.RefreshInterval) * time.Second).String(),
		"timeout":          (time.Duration(settings.Timeout) * time.Second).String(),
		"filters":          filters,
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

	feedConfig, hash, err := feed.LoadConfig(h.cfg.FeedsDir, name)
	if err != nil {
		slog.Error("Error loading configuration", "feed", name, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to load configuration",
			"details": err.Error(),
		})
		return
	}

	err = h.feedRepo.UpsertFeedConfig(
		feedConfig.Name,
		feedConfig.URL,
		feedConfig.Settings.Enabled,
		feedConfig.Settings,
		feedConfig.Filters,
		hash,
	)
	if err != nil {
		slog.Error("Error upserting feed config to database", "feed", name, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to save configuration to database",
			"details": err.Error(),
		})
		return
	}

	refilterFeedTask := tasks.NewRefilterFeedTask(name, h.filterer, h.feedRepo, h.itemRepo)
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
		"message": "Configuration reloaded and refilter task enqueued successfully",
		"feed": gin.H{
			"name": name,
			"url":  feedConfig.URL,
		},
		"tasks": []gin.H{
			{
				"id":   refilterFeedTask.ID,
				"type": refilterFeedTask.Type,
			},
		},
	}

	c.JSON(http.StatusOK, response)
}
