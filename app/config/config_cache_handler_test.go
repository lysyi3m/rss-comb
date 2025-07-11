package config

import (
	"testing"
)

func TestConfigCacheHandler(t *testing.T) {
	// Create initial configs
	initialConfigs := map[string]*FeedConfig{
		"feed1.yml": {
			Feed: FeedInfo{
				ID:    "feed1",
				URL:   "https://example.com/feed1.xml",
				Title: "Feed 1",
			},
		},
		"feed2.yml": {
			Feed: FeedInfo{
				ID:    "feed2",
				URL:   "https://example.com/feed2.xml",
				Title: "Feed 2",
			},
		},
	}

	// Create handler
	handler := NewConfigCacheHandler("Test component", initialConfigs)

	// Test initial state
	if handler.GetConfigCount() != 2 {
		t.Errorf("Expected 2 configs, got %d", handler.GetConfigCount())
	}

	// Test GetConfig
	config, exists := handler.GetConfig("feed1.yml")
	if !exists {
		t.Error("Expected feed1.yml to exist")
	}
	if config.Feed.ID != "feed1" {
		t.Errorf("Expected feed ID 'feed1', got '%s'", config.Feed.ID)
	}

	// Test GetAllConfigs
	allConfigs := handler.GetAllConfigs()
	if len(allConfigs) != 2 {
		t.Errorf("Expected 2 configs in GetAllConfigs, got %d", len(allConfigs))
	}

	// Test config update
	newConfig := &FeedConfig{
		Feed: FeedInfo{
			ID:    "feed3",
			URL:   "https://example.com/feed3.xml",
			Title: "Feed 3",
		},
	}
	
	err := handler.OnConfigUpdate("feed3.yml", newConfig, false)
	if err != nil {
		t.Errorf("Expected no error updating config, got: %v", err)
	}

	if handler.GetConfigCount() != 3 {
		t.Errorf("Expected 3 configs after update, got %d", handler.GetConfigCount())
	}

	// Test config deletion
	err = handler.OnConfigUpdate("feed1.yml", initialConfigs["feed1.yml"], true)
	if err != nil {
		t.Errorf("Expected no error deleting config, got: %v", err)
	}

	if handler.GetConfigCount() != 2 {
		t.Errorf("Expected 2 configs after deletion, got %d", handler.GetConfigCount())
	}

	// Test that deleted config is gone
	_, exists = handler.GetConfig("feed1.yml")
	if exists {
		t.Error("Expected feed1.yml to be deleted")
	}

	// Test that other configs still exist
	_, exists = handler.GetConfig("feed2.yml")
	if !exists {
		t.Error("Expected feed2.yml to still exist")
	}

	_, exists = handler.GetConfig("feed3.yml")
	if !exists {
		t.Error("Expected feed3.yml to still exist")
	}
}

func TestConfigCacheHandlerIsolation(t *testing.T) {
	// Create initial configs
	initialConfigs := map[string]*FeedConfig{
		"feed1.yml": {
			Feed: FeedInfo{
				ID:    "feed1",
				URL:   "https://example.com/feed1.xml",
				Title: "Feed 1",
			},
		},
	}

	// Create handler
	handler := NewConfigCacheHandler("Test component", initialConfigs)

	// Get all configs
	allConfigs := handler.GetAllConfigs()

	// Modify the returned map (should not affect internal state)
	allConfigs["feed2.yml"] = &FeedConfig{
		Feed: FeedInfo{
			ID:    "feed2",
			URL:   "https://example.com/feed2.xml",
			Title: "Feed 2",
		},
	}

	// Verify internal state is unchanged
	if handler.GetConfigCount() != 1 {
		t.Errorf("Expected 1 config (internal state should be unchanged), got %d", handler.GetConfigCount())
	}

	// Verify that a fresh call returns the correct state
	freshConfigs := handler.GetAllConfigs()
	if len(freshConfigs) != 1 {
		t.Errorf("Expected 1 config in fresh call, got %d", len(freshConfigs))
	}
}