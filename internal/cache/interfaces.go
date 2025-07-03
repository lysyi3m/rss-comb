package cache

import "time"

// CacheInterface defines the interface for cache operations
type CacheInterface interface {
	Get(key string) (string, error)
	Set(key string, value interface{}, ttl time.Duration) error
	Delete(key string) error
	Exists(key string) (bool, error)
	GetTTL(key string) (time.Duration, error)
	GenerateFeedKey(feedURL string) string
	GenerateMetricsKey(feedID string) string
	SetFeedData(feedURL, rssContent string, ttl time.Duration) error
	GetFeedData(feedURL string) (string, bool, error)
	Close() error
	FlushAll() error
	GetStats() (map[string]interface{}, error)
	Health() map[string]interface{}
}