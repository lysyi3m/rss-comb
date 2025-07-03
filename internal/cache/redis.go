package cache

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache wraps Redis client for feed caching operations
type Cache struct {
	client *redis.Client
	ctx    context.Context
}

// NewCache creates a new Redis cache client
func NewCache(addr string) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     "", // no password
		DB:           0,  // default DB
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	ctx := context.Background()
	
	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Printf("Connected to Redis at %s", addr)
	
	return &Cache{
		client: client,
		ctx:    ctx,
	}, nil
}

// Get retrieves a value from cache
func (c *Cache) Get(key string) (string, error) {
	val, err := c.client.Get(c.ctx, key).Result()
	if err == redis.Nil {
		return "", nil // Key doesn't exist
	}
	if err != nil {
		return "", fmt.Errorf("failed to get key %s: %w", key, err)
	}
	return val, nil
}

// Set stores a value in cache with TTL
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) error {
	var data []byte
	var err error

	// Handle different value types
	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		data, err = json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value for key %s: %w", key, err)
		}
	}

	err = c.client.Set(c.ctx, key, data, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set key %s: %w", key, err)
	}

	return nil
}

// Delete removes a key from cache
func (c *Cache) Delete(key string) error {
	err := c.client.Del(c.ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}
	return nil
}

// Exists checks if a key exists in cache
func (c *Cache) Exists(key string) (bool, error) {
	count, err := c.client.Exists(c.ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence of key %s: %w", key, err)
	}
	return count > 0, nil
}

// GetTTL returns the remaining TTL for a key
func (c *Cache) GetTTL(key string) (time.Duration, error) {
	ttl, err := c.client.TTL(c.ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get TTL for key %s: %w", key, err)
	}
	return ttl, nil
}

// GenerateFeedKey generates a consistent cache key for a feed URL
func (c *Cache) GenerateFeedKey(feedURL string) string {
	hash := sha256.Sum256([]byte(feedURL))
	return fmt.Sprintf("feed:%x", hash[:8]) // Use first 8 bytes for shorter keys
}

// GenerateMetricsKey generates a cache key for metrics data
func (c *Cache) GenerateMetricsKey(feedID string) string {
	return fmt.Sprintf("metrics:%s", feedID)
}

// SetFeedData stores processed feed data in cache
func (c *Cache) SetFeedData(feedURL, rssContent string, ttl time.Duration) error {
	key := c.GenerateFeedKey(feedURL)
	
	// Store both the RSS content and metadata
	feedData := map[string]interface{}{
		"content":    rssContent,
		"cached_at":  time.Now().Unix(),
		"expires_at": time.Now().Add(ttl).Unix(),
	}
	
	return c.Set(key, feedData, ttl)
}

// GetFeedData retrieves processed feed data from cache
func (c *Cache) GetFeedData(feedURL string) (string, bool, error) {
	key := c.GenerateFeedKey(feedURL)
	
	data, err := c.Get(key)
	if err != nil {
		return "", false, err
	}
	if data == "" {
		return "", false, nil // Cache miss
	}
	
	var feedData map[string]interface{}
	if err := json.Unmarshal([]byte(data), &feedData); err != nil {
		// If unmarshal fails, treat as cache miss and delete invalid data
		c.Delete(key)
		return "", false, nil
	}
	
	content, ok := feedData["content"].(string)
	if !ok {
		// Invalid data format, delete and return miss
		c.Delete(key)
		return "", false, nil
	}
	
	return content, true, nil
}

// Close closes the Redis connection
func (c *Cache) Close() error {
	return c.client.Close()
}

// FlushAll clears all cached data (use with caution)
func (c *Cache) FlushAll() error {
	err := c.client.FlushAll(c.ctx).Err()
	if err != nil {
		return fmt.Errorf("failed to flush cache: %w", err)
	}
	return nil
}

// GetStats returns cache statistics
func (c *Cache) GetStats() (map[string]interface{}, error) {
	info, err := c.client.Info(c.ctx, "memory", "stats").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache stats: %w", err)
	}

	// Parse basic stats
	stats := map[string]interface{}{
		"connected": true,
		"info":      info,
	}

	// Get database size
	dbSize, err := c.client.DBSize(c.ctx).Result()
	if err == nil {
		stats["key_count"] = dbSize
	}

	return stats, nil
}

// Health returns cache health information
func (c *Cache) Health() map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"type":   "redis",
	}

	// Test connection
	if err := c.client.Ping(c.ctx).Err(); err != nil {
		health["status"] = "unhealthy"
		health["error"] = err.Error()
		return health
	}

	// Get basic info
	if stats, err := c.GetStats(); err == nil {
		health["key_count"] = stats["key_count"]
	}

	return health
}