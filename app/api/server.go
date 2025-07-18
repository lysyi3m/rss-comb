package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lysyi3m/rss-comb/app/config"
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
	cfg := config.Get()
	setupRoutes(r, handler, cfg.GetAPIAccessKey())

	return r
}

// setupRoutes configures all the application routes
func setupRoutes(r *gin.Engine, handler *Handler, apiAccessKey string) {
	// Main feed endpoint
	r.GET("/feeds/:id", handler.GetFeedByID)

	// Health endpoint
	r.GET("/health", handler.GetHealth)

	// API endpoints (conditionally enabled with authentication)
	if apiAccessKey != "" {
		api := r.Group("/api")
		api.Use(authMiddleware(apiAccessKey))
		{
			api.GET("/feeds", handler.APIListFeeds)
			api.GET("/feeds/:id/details", handler.APIGetFeedDetailsByID)
			api.POST("/feeds/:id/reload", handler.APIReloadFeedByID)
		}
		slog.Info("API endpoints enabled with authentication")
	} else {
		slog.Info("API endpoints disabled", "reason", "API_ACCESS_KEY not set")
	}

	// Root endpoint with basic information
	r.GET("/", func(c *gin.Context) {
		endpoints := map[string]string{
			"feed":   "/feeds/<id>",
			"health": "/health",
		}

		// Add API endpoints if authentication is enabled
		if apiAccessKey != "" {
			endpoints["feeds"] = "/api/feeds (requires X-API-Key header)"
			endpoints["details"] = "/api/feeds/<id>/details (requires X-API-Key header)"
			endpoints["reload"] = "/api/feeds/<id>/reload (POST, requires X-API-Key header)"
		}

		c.JSON(200, gin.H{
			"service":     "RSS Comb",
			"version":     config.GetVersion(),
			"description": "RSS/Atom feed proxy with normalization, automatic deduplication, and filtering",
			"endpoints":   endpoints,
			"api_status":  map[string]interface{}{
				"enabled": apiAccessKey != "",
				"header": "X-API-Key",
			},
			"documentation": "https://github.com/lysyi3m/rss-comb",
		})
	})

	// Favicon handler (return 204 to avoid 404s)
	r.GET("/favicon.ico", func(c *gin.Context) {
		c.Status(204)
	})
}

// authMiddleware creates authentication middleware for API endpoints
func authMiddleware(apiAccessKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get API key from X-API-Key header
		providedKey := c.GetHeader("X-API-Key")

		// Also check Authorization header with Bearer prefix
		if providedKey == "" {
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				providedKey = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		// Check if API key is provided and matches
		if providedKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "API key required",
				"message": "Provide API key in X-API-Key header or Authorization: Bearer <key>",
			})
			c.Abort()
			return
		}

		if providedKey != apiAccessKey {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid API key",
				"message": "The provided API key is not valid",
			})
			c.Abort()
			return
		}

		// Continue to next middleware/handler
		c.Next()
	}
}

