package api

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// NewServer creates a new HTTP server with all routes configured
func NewServer(handler *Handler) *gin.Engine {
	// Set Gin mode (can be controlled via GIN_MODE environment variable)
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	// Middleware
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: func(param gin.LogFormatterParams) string {
			return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
				param.ClientIP,
				param.TimeStamp.Format(time.RFC3339),
				param.Method,
				param.Path,
				param.Request.Proto,
				param.StatusCode,
				param.Latency,
				param.Request.UserAgent(),
				param.ErrorMessage,
			)
		},
	}))
	
	r.Use(gin.Recovery())

	// CORS middleware for API endpoints
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})

	// Routes
	setupRoutes(r, handler)

	return r
}

// setupRoutes configures all the application routes
func setupRoutes(r *gin.Engine, handler *Handler) {
	// Main feed endpoint
	r.GET("/feed", handler.GetFeed)

	// Health and status endpoints
	r.GET("/health", handler.HealthCheck)
	r.GET("/stats", handler.GetStats)

	// API endpoints
	api := r.Group("/api/v1")
	{
		api.GET("/feeds", handler.ListFeeds)
		api.GET("/feeds/details", handler.GetFeedDetails)
		api.POST("/feeds/reapply-filters", handler.ReapplyFilters)
	}

	// Root endpoint with basic information
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"service":     "RSS Comb",
			"version":     "1.0.0",
			"description": "RSS/Atom feed proxy with normalization, deduplication, and filtering",
			"endpoints": map[string]string{
				"feed":           "/feed?url=<feed-url>",
				"health":         "/health",
				"stats":          "/stats",
				"feeds":          "/api/v1/feeds",
				"details":        "/api/v1/feeds/details?url=<feed-url>",
				"reapply-filters": "/api/v1/feeds/reapply-filters?url=<feed-url> (POST)",
			},
			"documentation": "https://github.com/lysyi3m/rss-comb",
		})
	})

	// Favicon handler (return 204 to avoid 404s)
	r.GET("/favicon.ico", func(c *gin.Context) {
		c.Status(204)
	})
}

// ServerConfig holds server configuration options
type ServerConfig struct {
	Host         string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DefaultServerConfig returns default server configuration
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Host:         "",
		Port:         "8080",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}