package config

import (
	"testing"
)

func TestValidateConfig(t *testing.T) {
	// Test with nil config
	err := ValidateConfig(nil)
	if err == nil {
		t.Error("Expected error for nil config, got none")
	}

	// Test with empty feed ID
	config := &FeedConfig{
		Feed: FeedInfo{
			ID:    "",
			URL:   "https://example.com/feed.xml",
			Title: "Test Feed",
		},
	}
	err = ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for empty feed ID, got none")
	}

	// Test with empty URL
	config.Feed.ID = "test-feed"
	config.Feed.URL = ""
	err = ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for empty URL, got none")
	}

	// Test with empty title
	config.Feed.URL = "https://example.com/feed.xml"
	config.Feed.Title = ""
	err = ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for empty title, got none")
	}

	// Test with negative refresh interval
	config.Feed.Title = "Test Feed"
	config.Settings.RefreshInterval = -1
	err = ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for negative refresh interval, got none")
	}

	// Test with negative max items
	config.Settings.RefreshInterval = 3600
	config.Settings.MaxItems = -1
	err = ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for negative max items, got none")
	}

	// Test with negative timeout
	config.Settings.MaxItems = 100
	config.Settings.Timeout = -1
	err = ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for negative timeout, got none")
	}

	// Test with invalid filter field
	config.Settings.Timeout = 30
	config.Filters = []Filter{
		{
			Field:    "invalid_field",
			Includes: []string{"test"},
		},
	}
	err = ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for invalid filter field, got none")
	}

	// Test with filter having no includes or excludes
	config.Filters = []Filter{
		{
			Field:    "title",
			Includes: []string{},
			Excludes: []string{},
		},
	}
	err = ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for filter with no includes or excludes, got none")
	}

	// Test with valid config
	config.Filters = []Filter{
		{
			Field:    "title",
			Includes: []string{"test"},
		},
	}
	err = ValidateConfig(config)
	if err != nil {
		t.Errorf("Expected no error for valid config, got: %v", err)
	}
}

func TestValidateConfigFilterFields(t *testing.T) {
	config := &FeedConfig{
		Feed: FeedInfo{
			ID:    "test-feed",
			URL:   "https://example.com/feed.xml",
			Title: "Test Feed",
		},
		Settings: FeedSettings{
			RefreshInterval: 3600,
			MaxItems:        100,
			Timeout:         30,
		},
	}

	// Test all valid filter fields
	validFields := []string{"title", "description", "content", "author", "link", "categories"}
	for _, field := range validFields {
		config.Filters = []Filter{
			{
				Field:    field,
				Includes: []string{"test"},
			},
		}
		err := ValidateConfig(config)
		if err != nil {
			t.Errorf("Expected no error for valid filter field '%s', got: %v", field, err)
		}
	}

	// Test invalid filter field
	config.Filters = []Filter{
		{
			Field:    "invalid_field",
			Includes: []string{"test"},
		},
	}
	err := ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for invalid filter field, got none")
	}
}
