package api

import (
	"cmp"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lysyi3m/rss-comb/app/cfg"
)

func NewServer(handler *Handler) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

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

	cfg := cfg.Get()
	setupRoutes(r, handler, cfg)

	return r
}

func setupRoutes(r *gin.Engine, handler *Handler, cfg *cfg.Cfg) {
	r.GET("/feeds/:name", handler.GetFeed)

	r.GET("/health", handler.GetHealth)

	if cfg.APIAccessKey != "" {
		api := r.Group("/api")
		api.Use(authMiddleware(cfg.APIAccessKey))
		{
			api.GET("/feeds", handler.APIListFeeds)
			api.GET("/feeds/:name/details", handler.APIGetFeedDetails)
			api.POST("/feeds/:name/reload", handler.APIReloadFeed)
		}
	} else {
		slog.Info("API endpoints disabled", "reason", "API_ACCESS_KEY not set")
	}

	// Root endpoint with basic information
	r.GET("/", func(c *gin.Context) {
		endpoints := map[string]string{
			"feed":   "/feeds/<name>",
			"health": "/health",
		}

		if cfg.APIAccessKey != "" {
			endpoints["feeds"] = "/api/feeds (requires X-API-Key header)"
			endpoints["details"] = "/api/feeds/<name>/details (requires X-API-Key header)"
			endpoints["reload"] = "/api/feeds/<name>/reload (POST, requires X-API-Key header)"
		}

		c.JSON(200, gin.H{
			"service":     "RSS Comb",
			"version":     cfg.Version,
			"description": "RSS/Atom feed proxy with normalization, automatic deduplication, and filtering",
			"endpoints":   endpoints,
			"api_status": map[string]interface{}{
				"enabled": cfg.APIAccessKey != "",
				"header":  "X-API-Key",
			},
			"documentation": "https://github.com/lysyi3m/rss-comb",
		})
	})

}

func authMiddleware(apiAccessKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Use cmp.Or to coalesce API key from X-API-Key header or Authorization Bearer token
		authHeader := c.GetHeader("Authorization")
		var bearerToken string
		if strings.HasPrefix(authHeader, "Bearer ") {
			bearerToken = strings.TrimPrefix(authHeader, "Bearer ")
		}
		providedKey := cmp.Or(c.GetHeader("X-API-Key"), bearerToken)

		if providedKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "API key required",
				"message": "Provide API key in X-API-Key header or Authorization: Bearer <key>",
			})
			c.Abort()
			return
		}

		if providedKey != apiAccessKey {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid API key",
				"message": "The provided API key is not valid",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
