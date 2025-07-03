package cache

import (
	"testing"
)

func TestGenerateFeedKey(t *testing.T) {
	cache := &Cache{}

	// Test consistent key generation
	url1 := "https://example.com/feed.xml"
	url2 := "https://different.com/feed.xml"

	key1a := cache.GenerateFeedKey(url1)
	key1b := cache.GenerateFeedKey(url1)
	key2 := cache.GenerateFeedKey(url2)

	// Same URL should generate same key
	if key1a != key1b {
		t.Errorf("Expected same key for same URL, got %s != %s", key1a, key1b)
	}

	// Different URLs should generate different keys
	if key1a == key2 {
		t.Errorf("Expected different keys for different URLs, but got same: %s", key1a)
	}

	// Keys should have correct prefix
	expectedPrefix := "feed:"
	if !containsPrefix(key1a, expectedPrefix) {
		t.Errorf("Expected key to start with %s, got %s", expectedPrefix, key1a)
	}
}

func TestGenerateMetricsKey(t *testing.T) {
	cache := &Cache{}

	feedID := "test-feed-id"
	key := cache.GenerateMetricsKey(feedID)

	expectedKey := "metrics:test-feed-id"
	if key != expectedKey {
		t.Errorf("Expected key %s, got %s", expectedKey, key)
	}
}

func TestCacheOperationsWithoutRedis(t *testing.T) {
	// These tests don't require a Redis connection
	// They test the cache logic without actual Redis operations

	t.Run("GenerateFeedKey consistency", func(t *testing.T) {
		cache := &Cache{}
		url := "https://example.com/feed.xml"
		
		key1 := cache.GenerateFeedKey(url)
		key2 := cache.GenerateFeedKey(url)
		
		if key1 != key2 {
			t.Errorf("Expected consistent key generation, got %s != %s", key1, key2)
		}
		
		if len(key1) == 0 {
			t.Error("Expected non-empty key")
		}
	})

	t.Run("GenerateMetricsKey format", func(t *testing.T) {
		cache := &Cache{}
		feedID := "12345"
		
		key := cache.GenerateMetricsKey(feedID)
		expected := "metrics:12345"
		
		if key != expected {
			t.Errorf("Expected key %s, got %s", expected, key)
		}
	})
}

// Helper function to check if string starts with prefix
func containsPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// Mock tests for cache operations (would require Redis in integration tests)
func TestCacheInterface(t *testing.T) {
	// Test that our Cache struct would implement expected interface behavior
	// This is more of a compile-time check
	
	t.Run("Cache struct has required methods", func(t *testing.T) {
		// This test ensures our Cache struct has the required methods
		// In a real scenario, you'd define an interface and test against it
		
		cache := &Cache{}
		
		// Test method signatures exist (compile-time check)
		_ = cache.GenerateFeedKey("test")
		_ = cache.GenerateMetricsKey("test")
		
		// These would fail without Redis, but we're just testing method existence
		// _, _ = cache.Get("test")
		// _ = cache.Set("test", "value", time.Minute)
		// _ = cache.Delete("test")
		// _, _ = cache.Exists("test")
		// _, _ = cache.GetTTL("test")
		// _ = cache.SetFeedData("url", "content", time.Minute)
		// _, _, _ = cache.GetFeedData("url")
		// _ = cache.FlushAll()
		// _, _ = cache.GetStats()
		// _ = cache.Health()
		// _ = cache.Close()
	})
}

// Integration tests would go here if Redis is available
// For example:
/*
func TestCacheWithRedis(t *testing.T) {
	// Skip if Redis not available
	cache, err := NewCache("localhost:6379")
	if err != nil {
		t.Skip("Redis not available for integration tests")
	}
	defer cache.Close()

	t.Run("Basic operations", func(t *testing.T) {
		key := "test:key"
		value := "test value"
		
		// Test Set
		err := cache.Set(key, value, time.Minute)
		if err != nil {
			t.Fatalf("Failed to set value: %v", err)
		}
		
		// Test Get
		retrieved, err := cache.Get(key)
		if err != nil {
			t.Fatalf("Failed to get value: %v", err)
		}
		
		if retrieved != value {
			t.Errorf("Expected %s, got %s", value, retrieved)
		}
		
		// Test Exists
		exists, err := cache.Exists(key)
		if err != nil {
			t.Fatalf("Failed to check existence: %v", err)
		}
		
		if !exists {
			t.Error("Expected key to exist")
		}
		
		// Test Delete
		err = cache.Delete(key)
		if err != nil {
			t.Fatalf("Failed to delete key: %v", err)
		}
		
		// Verify deletion
		exists, err = cache.Exists(key)
		if err != nil {
			t.Fatalf("Failed to check existence after deletion: %v", err)
		}
		
		if exists {
			t.Error("Expected key to not exist after deletion")
		}
	})
}
*/