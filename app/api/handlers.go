package api

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lysyi3m/rss-comb/app/config"
	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/feed"
)

// Handler handles HTTP requests for the RSS API
type Handler struct {
	feedRepo  database.FeedRepositoryInterface
	itemRepo  *database.ItemRepository
	generator *RSSGenerator
	configs   map[string]*config.FeedConfig
	processor feed.FeedProcessor
	userAgent string
}

// NewHandler creates a new API handler
func NewHandler(fr database.FeedRepositoryInterface, ir *database.ItemRepository,
	configs map[string]*config.FeedConfig, processor feed.FeedProcessor, port string, userAgent string) *Handler {
	return &Handler{
		feedRepo:  fr,
		itemRepo:  ir,
		generator: NewRSSGenerator(port),
		configs:   configs,
		processor: processor,
		userAgent: userAgent,
	}
}

// GetFeedByID handles the new ID-based feed endpoint
func (h *Handler) GetFeedByID(c *gin.Context) {
	feedID := c.Param("id")
	if feedID == "" {
		c.Header("Content-Type", "application/xml; charset=utf-8")
		c.String(http.StatusBadRequest, h.generator.GenerateError("", "", "Missing feed ID"))
		return
	}

	// Find matching configuration by feed ID
	var feedConfig *config.FeedConfig
	for _, cfg := range h.configs {
		if cfg.Feed.ID == feedID {
			feedConfig = cfg
			break
		}
	}

	// If not registered, return 404
	if feedConfig == nil {
		log.Printf("Feed ID not found: %s", feedID)
		c.Status(http.StatusNotFound)
		return
	}

	// Get feed from database
	feed, err := h.feedRepo.GetFeedByID(feedID)
	if err != nil {
		log.Printf("Database error getting feed %s: %v", feedID, err)
		c.Header("Content-Type", "application/xml; charset=utf-8")
		c.String(http.StatusInternalServerError, h.generator.GenerateError(feedConfig.Feed.Title, feedConfig.Feed.URL, "Database error"))
		return
	}

	// If feed not found in database, return empty feed
	if feed == nil {
		log.Printf("Feed not yet processed: %s", feedID)
		c.Header("Content-Type", "application/xml; charset=utf-8")
		c.String(http.StatusOK, h.generator.GenerateEmpty(feedConfig.Feed.Title, feedConfig.Feed.URL))
		return
	}

	// Get feed items
	items, err := h.itemRepo.GetVisibleItems(feed.ID, feedConfig.Settings.MaxItems)
	if err != nil {
		log.Printf("Database error getting items for feed %s: %v", feedID, err)
		c.Header("Content-Type", "application/xml; charset=utf-8")
		c.String(http.StatusInternalServerError, h.generator.GenerateError(feed.Title, feed.FeedURL, "Failed to retrieve items"))
		return
	}

	// Generate RSS
	rss, err := h.generator.Generate(*feed, items)
	if err != nil {
		log.Printf("RSS generation error for feed %s: %v", feedID, err)
		c.Header("Content-Type", "application/xml; charset=utf-8")
		c.String(http.StatusInternalServerError, h.generator.GenerateError(feed.Title, feed.FeedURL, "RSS generation failed"))
		return
	}

	// Set response headers
	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Header("X-Feed-Items", strconv.Itoa(len(items)))
	c.Header("X-Feed-ID", feedID)
	
	if feed.LastSuccess != nil {
		c.Header("X-Last-Updated", feed.LastSuccess.Format(time.RFC3339))
	}

	log.Printf("Served feed %s with %d items", feedID, len(items))
	c.String(http.StatusOK, rss)
}


// GetStats handles the statistics endpoint
func (h *Handler) GetStats(c *gin.Context) {
	stats := map[string]interface{}{
		"timestamp": time.Now().In(time.Local).Format(time.RFC3339),
	}

	// Get enabled feed count
	if enabledFeedCount, err := h.feedRepo.GetEnabledFeedCount(); err == nil {
		stats["enabled_feeds"] = enabledFeedCount
	}

	stats["loaded_configurations"] = len(h.configs)

	c.JSON(http.StatusOK, stats)
}

// ListFeeds handles listing all configured feeds
func (h *Handler) ListFeeds(c *gin.Context) {
	feeds := make([]map[string]interface{}, 0, len(h.configs))

	for configFile, config := range h.configs {
		feedInfo := map[string]interface{}{
			"name":            config.Feed.Title,
			"url":             config.Feed.URL,
			"config_file":     configFile,
			"enabled":         config.Settings.Enabled,
			"max_items":       config.Settings.MaxItems,
			"refresh_interval": config.Settings.GetRefreshInterval().String(),
			"filters":         len(config.Filters),
		}

		// Get feed from database if available
		if feed, err := h.feedRepo.GetFeedByURL(config.Feed.URL); err == nil && feed != nil {
			feedInfo["last_fetched"] = feed.LastFetched
			feedInfo["last_success"] = feed.LastSuccess
			feedInfo["next_fetch"] = feed.NextFetch
			
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



// GetFeedDetailsByID handles detailed information about a specific feed by ID
func (h *Handler) GetFeedDetailsByID(c *gin.Context) {
	feedID := c.Param("id")
	if feedID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing feed ID parameter"})
		return
	}

	// Find configuration by feed ID
	var configFile string
	var feedConfig *config.FeedConfig
	for file, cfg := range h.configs {
		if cfg.Feed.ID == feedID {
			configFile = file
			feedConfig = cfg
			break
		}
	}

	if feedConfig == nil {
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
		"deduplication":    feedConfig.Settings.Deduplication,
		"filters":          feedConfig.Filters,
	}

	// Get feed from database
	if feed, err := h.feedRepo.GetFeedByID(feedID); err == nil && feed != nil {
		details["database"] = map[string]interface{}{
			"id":           feed.ID,
			"feed_id":      feed.FeedID,
			"last_fetched": feed.LastFetched,
			"last_success": feed.LastSuccess,
			"next_fetch":   feed.NextFetch,
			"enabled":      feed.Enabled,
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

// ReapplyFiltersByID handles the filter re-application endpoint by feed ID
func (h *Handler) ReapplyFiltersByID(c *gin.Context) {
	feedID := c.Param("id")
	if feedID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing feed ID parameter"})
		return
	}

	// Find configuration by feed ID
	var configFile string
	var feedConfig *config.FeedConfig
	for file, cfg := range h.configs {
		if cfg.Feed.ID == feedID {
			configFile = file
			feedConfig = cfg
			break
		}
	}

	if feedConfig == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Feed not configured"})
		return
	}

	// Get feed from database
	feed, err := h.feedRepo.GetFeedByID(feedID)
	if err != nil {
		log.Printf("Database error getting feed %s: %v", feedID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if feed == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Feed not found in database"})
		return
	}

	// Re-apply filters
	updatedCount, errorCount, err := h.processor.ReapplyFilters(feed.ID, configFile)
	if err != nil {
		log.Printf("Error re-applying filters for feed %s: %v", feedID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to re-apply filters",
			"details": err.Error(),
		})
		return
	}

	// Get updated statistics
	total, visible, filtered, err := h.itemRepo.GetItemStats(feed.ID)
	if err != nil {
		log.Printf("Warning: failed to get updated stats for feed %s: %v", feedID, err)
	}

	response := gin.H{
		"success": true,
		"message": "Filters re-applied successfully",
		"feed": gin.H{
			"id":   feedConfig.Feed.ID,
			"name": feedConfig.Feed.Title,
			"url":  feedConfig.Feed.URL,
		},
		"results": gin.H{
			"updated_items": updatedCount,
			"errors":        errorCount,
		},
	}

	// Add updated stats if available
	if err == nil {
		response["updated_stats"] = gin.H{
			"total_items":    total,
			"visible_items":  visible,
			"filtered_items": filtered,
		}
	}

	log.Printf("Successfully re-applied filters for feed %s: %d items updated, %d errors", 
		feedID, updatedCount, errorCount)

	c.JSON(http.StatusOK, response)
}