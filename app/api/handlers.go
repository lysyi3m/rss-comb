package api

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lysyi3m/rss-comb/app/cfg"
	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/feed"
	"github.com/lysyi3m/rss-comb/app/services"
)

type Handler struct {
	cfg      *cfg.Cfg
	feedRepo *database.FeedRepository
	itemRepo *database.ItemRepository
}

func NewHandler(
	cfg *cfg.Cfg,
	feedRepo *database.FeedRepository,
	itemRepo *database.ItemRepository,
) *Handler {
	return &Handler{
		cfg:      cfg,
		feedRepo: feedRepo,
		itemRepo: itemRepo,
	}
}

func (h *Handler) GetFeed(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	dbFeed, err := h.feedRepo.GetFeed(name)
	if err != nil {
		slog.Error("Database error", "operation", "get_feed", "feed", name, "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	if dbFeed == nil {
		slog.Error("Feed not found in database", "feed", name)
		c.Status(http.StatusNotFound)
		return
	}

	settings, err := dbFeed.GetSettings()
	if err != nil {
		slog.Error("Failed to get feed settings", "feed", name, "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	items, err := h.itemRepo.GetVisibleItems(name, settings.MaxItems)
	if err != nil {
		slog.Error("Database error", "operation", "get_items", "feed", name, "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	rss, err := feed.GenerateRSS(*dbFeed, items, h.cfg)
	if err != nil {
		slog.Error("RSS generation error", "feed", name, "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Header("X-Feed-Items", strconv.FormatInt(int64(len(items)), 10))
	c.Header("X-Feed-Name", name)
	c.Header("X-Last-Updated", dbFeed.UpdatedAt.In(h.cfg.Location).Format(time.RFC3339))

	c.String(http.StatusOK, rss)
}

func (h *Handler) GetHealth(c *gin.Context) {
	health := map[string]interface{}{
		"timestamp": time.Now().In(h.cfg.Location).Format(time.RFC3339),
	}

	if feedCount, err := h.feedRepo.GetFeedCount(); err == nil {
		health["feeds"] = feedCount
	}

	c.JSON(http.StatusOK, health)
}

func (h *Handler) APIReloadFeed(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing feed name parameter"})
		return
	}

	config, err := services.SyncFeedConfig(c.Request.Context(), h.cfg.FeedsDir, name, h.feedRepo)
	if err != nil {
		slog.Error("Failed to sync feed config", "feed", name, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to reload configuration",
			"details": err.Error(),
		})
		return
	}

	err = services.RefilterFeed(c.Request.Context(), name, h.feedRepo, h.itemRepo)
	if err != nil {
		slog.Error("Error refiltering feed", "feed", name, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to refilter feed items",
			"details": err.Error(),
		})
		return
	}

	response := gin.H{
		"success": true,
		"message": "Configuration reloaded and feed items refiltered successfully",
		"feed": gin.H{
			"name": name,
			"url":  config.URL,
		},
	}

	c.JSON(http.StatusOK, response)
}
