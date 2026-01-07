package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lysyi3m/rss-comb/app/cfg"
	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/feed"
	"github.com/lysyi3m/rss-comb/app/services"
)

func NewHandler(
	cfg *cfg.Cfg,
	feedRepo *database.FeedRepository,
	itemRepo *database.ItemRepository,
	filterer *feed.Filterer,
) *Handler {
	return &Handler{
		cfg:      cfg,
		feedRepo: feedRepo,
		itemRepo: itemRepo,
		filterer: filterer,
	}
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
		feedConfig.Enabled,
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

	err = services.RefilterFeed(c.Request.Context(), name, h.feedRepo, h.itemRepo, h.filterer)
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
			"url":  feedConfig.URL,
		},
	}

	c.JSON(http.StatusOK, response)
}
