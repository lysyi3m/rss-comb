package api

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lysyi3m/rss-comb/internal/cache"
	"github.com/lysyi3m/rss-comb/internal/config"
	"github.com/lysyi3m/rss-comb/internal/database"
)

// Handler handles HTTP requests for the RSS API
type Handler struct {
	feedRepo  database.FeedRepositoryInterface
	itemRepo  *database.ItemRepository
	cache     cache.CacheInterface
	generator *RSSGenerator
	configs   map[string]*config.FeedConfig
}

// NewHandler creates a new API handler
func NewHandler(fr database.FeedRepositoryInterface, ir *database.ItemRepository,
	c cache.CacheInterface, configs map[string]*config.FeedConfig) *Handler {
	return &Handler{
		feedRepo:  fr,
		itemRepo:  ir,
		cache:     c,
		generator: NewRSSGenerator(),
		configs:   configs,
	}
}

// GetFeed handles the main feed endpoint
func (h *Handler) GetFeed(c *gin.Context) {
	feedURL := c.Query("url")
	if feedURL == "" {
		c.Header("Content-Type", "application/rss+xml; charset=utf-8")
		c.String(http.StatusBadRequest, h.generator.GenerateError("", "", "Missing 'url' parameter"))
		return
	}

	// Find matching configuration
	var feedConfig *config.FeedConfig

	for _, cfg := range h.configs {
		if cfg.Feed.URL == feedURL {
			feedConfig = cfg
			break
		}
	}

	// If not registered, redirect to original feed
	if feedConfig == nil {
		log.Printf("Feed not registered: %s, redirecting to original", feedURL)
		c.Redirect(http.StatusFound, feedURL)
		return
	}

	// Check cache first
	if feedConfig.Settings.GetCacheDuration() > 0 {
		cached, hit, err := h.cache.GetFeedData(feedURL)
		if err != nil {
			log.Printf("Cache error for %s: %v", feedURL, err)
		} else if hit {
			log.Printf("Cache hit for feed: %s", feedURL)
			c.Header("Content-Type", "application/rss+xml; charset=utf-8")
			c.Header("X-Cache", "HIT")
			c.Header("Cache-Control", fmt.Sprintf("public, max-age=%d", int(feedConfig.Settings.GetCacheDuration().Seconds())))
			c.String(http.StatusOK, cached)
			return
		}
	}

	// Get feed from database
	feed, err := h.feedRepo.GetFeedByURL(feedURL)
	if err != nil {
		log.Printf("Database error getting feed %s: %v", feedURL, err)
		c.Header("Content-Type", "application/rss+xml; charset=utf-8")
		c.String(http.StatusInternalServerError, h.generator.GenerateError(feedConfig.Feed.Name, feedURL, "Database error"))
		return
	}

	// If feed not found in database, return empty feed
	if feed == nil {
		log.Printf("Feed not yet processed: %s", feedURL)
		c.Header("Content-Type", "application/rss+xml; charset=utf-8")
		c.Header("Cache-Control", "no-cache, max-age=60") // Short cache for empty feeds
		c.String(http.StatusOK, h.generator.GenerateEmpty(feedConfig.Feed.Name, feedURL))
		return
	}

	// Get feed items
	items, err := h.itemRepo.GetVisibleItems(feed.ID, feedConfig.Settings.MaxItems)
	if err != nil {
		log.Printf("Database error getting items for feed %s: %v", feedURL, err)
		c.Header("Content-Type", "application/rss+xml; charset=utf-8")
		c.String(http.StatusInternalServerError, h.generator.GenerateError(feed.Name, feedURL, "Failed to retrieve items"))
		return
	}

	// Generate RSS
	rss, err := h.generator.Generate(*feed, items)
	if err != nil {
		log.Printf("RSS generation error for feed %s: %v", feedURL, err)
		c.Header("Content-Type", "application/rss+xml; charset=utf-8")
		c.String(http.StatusInternalServerError, h.generator.GenerateError(feed.Name, feedURL, "RSS generation failed"))
		return
	}

	// Cache the result
	if feedConfig.Settings.GetCacheDuration() > 0 {
		if err := h.cache.SetFeedData(feedURL, rss, feedConfig.Settings.GetCacheDuration()); err != nil {
			log.Printf("Failed to cache feed %s: %v", feedURL, err)
		}
	}

	// Set response headers
	c.Header("Content-Type", "application/rss+xml; charset=utf-8")
	c.Header("X-Cache", "MISS")
	c.Header("Cache-Control", fmt.Sprintf("public, max-age=%d", int(feedConfig.Settings.GetCacheDuration().Seconds())))
	c.Header("X-Feed-Items", strconv.Itoa(len(items)))
	
	if feed.LastSuccess != nil {
		c.Header("X-Last-Updated", feed.LastSuccess.Format(time.RFC3339))
	}

	log.Printf("Served feed %s with %d items", feedURL, len(items))
	c.String(http.StatusOK, rss)
}

// HealthCheck handles the health check endpoint
func (h *Handler) HealthCheck(c *gin.Context) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "1.0.0",
	}

	// Get feed count
	if feedCount, err := h.feedRepo.GetFeedCount(); err == nil {
		health["feeds"] = feedCount
	}

	// Get cache health
	if h.cache != nil {
		cacheHealth := h.cache.Health()
		health["cache"] = cacheHealth
	}

	// Check configuration count
	health["configurations"] = len(h.configs)

	c.JSON(http.StatusOK, health)
}

// GetStats handles the statistics endpoint
func (h *Handler) GetStats(c *gin.Context) {
	stats := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"feeds":     map[string]interface{}{},
	}

	// Get overall feed count
	if feedCount, err := h.feedRepo.GetFeedCount(); err == nil {
		stats["total_feeds"] = feedCount
	}

	stats["loaded_configurations"] = len(h.configs)

	// Get cache stats
	if h.cache != nil {
		if cacheStats, err := h.cache.GetStats(); err == nil {
			stats["cache"] = cacheStats
		}
	}

	c.JSON(http.StatusOK, stats)
}

// ListFeeds handles listing all configured feeds
func (h *Handler) ListFeeds(c *gin.Context) {
	feeds := make([]map[string]interface{}, 0, len(h.configs))

	for configFile, config := range h.configs {
		feedInfo := map[string]interface{}{
			"name":            config.Feed.Name,
			"url":             config.Feed.URL,
			"config_file":     configFile,
			"enabled":         config.Settings.Enabled,
			"max_items":       config.Settings.MaxItems,
			"refresh_interval": config.Settings.GetRefreshInterval().String(),
			"cache_duration":  config.Settings.GetCacheDuration().String(),
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

// ReloadConfigs reloads the feed configurations
func (h *Handler) ReloadConfigs(configs map[string]*config.FeedConfig) {
	h.configs = configs
	log.Printf("Reloaded %d feed configurations in API handler", len(configs))
}

// GetFeedDetails handles detailed information about a specific feed
func (h *Handler) GetFeedDetails(c *gin.Context) {
	feedURL := c.Query("url")
	if feedURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'url' parameter"})
		return
	}

	// Find configuration
	var configFile string
	var feedConfig *config.FeedConfig
	for file, cfg := range h.configs {
		if cfg.Feed.URL == feedURL {
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
		"name":            feedConfig.Feed.Name,
		"url":             feedConfig.Feed.URL,
		"config_file":     configFile,
		"enabled":         feedConfig.Settings.Enabled,
		"max_items":       feedConfig.Settings.MaxItems,
		"refresh_interval": feedConfig.Settings.GetRefreshInterval().String(),
		"cache_duration":  feedConfig.Settings.GetCacheDuration().String(),
		"timeout":         feedConfig.Settings.GetTimeout().String(),
		"user_agent":      feedConfig.Settings.UserAgent,
		"deduplication":   feedConfig.Settings.Deduplication,
		"filters":         feedConfig.Filters,
	}

	// Get feed from database
	if feed, err := h.feedRepo.GetFeedByURL(feedURL); err == nil && feed != nil {
		details["database"] = map[string]interface{}{
			"id":           feed.ID,
			"last_fetched": feed.LastFetched,
			"last_success": feed.LastSuccess,
			"next_fetch":   feed.NextFetch,
			"is_active":    feed.IsActive,
			"created_at":   feed.CreatedAt,
			"updated_at":   feed.UpdatedAt,
		}

		// Get item statistics
		if total, visible, duplicates, filtered, err := h.itemRepo.GetItemStats(feed.ID); err == nil {
			details["items"] = map[string]interface{}{
				"total":      total,
				"visible":    visible,
				"duplicates": duplicates,
				"filtered":   filtered,
			}
		}
	}

	c.JSON(http.StatusOK, details)
}