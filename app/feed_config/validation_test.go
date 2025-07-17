package feed_config

import (
	"testing"
)

func TestValidateConfig(t *testing.T) {
	// Test with nil feedConfig
	err := ValidateConfig(nil)
	if err == nil {
		t.Error("Expected error for nil feedConfig, got none")
	}

	// Test with empty feed ID
	feedConfig := &FeedConfig{
		Feed: FeedInfo{
			ID:    "",
			URL:   "https://example.com/feed.xml",
			Title: "Test Feed",
		},
	}
	err = ValidateConfig(feedConfig)
	if err == nil {
		t.Error("Expected error for empty feed ID, got none")
	}

	// Test with empty URL
	feedConfig.Feed.ID = "test-feed"
	feedConfig.Feed.URL = ""
	err = ValidateConfig(feedConfig)
	if err == nil {
		t.Error("Expected error for empty URL, got none")
	}

	// Test with empty title
	feedConfig.Feed.URL = "https://example.com/feed.xml"
	feedConfig.Feed.Title = ""
	err = ValidateConfig(feedConfig)
	if err == nil {
		t.Error("Expected error for empty title, got none")
	}

	// Test with negative refresh interval
	feedConfig.Feed.Title = "Test Feed"
	feedConfig.Settings.RefreshInterval = -1
	err = ValidateConfig(feedConfig)
	if err == nil {
		t.Error("Expected error for negative refresh interval, got none")
	}

	// Test with negative max items
	feedConfig.Settings.RefreshInterval = 3600
	feedConfig.Settings.MaxItems = -1
	err = ValidateConfig(feedConfig)
	if err == nil {
		t.Error("Expected error for negative max items, got none")
	}

	// Test with negative timeout
	feedConfig.Settings.MaxItems = 100
	feedConfig.Settings.Timeout = -1
	err = ValidateConfig(feedConfig)
	if err == nil {
		t.Error("Expected error for negative timeout, got none")
	}

	// Test with invalid filter field
	feedConfig.Settings.Timeout = 30
	feedConfig.Filters = []Filter{
		{
			Field:    "invalid_field",
			Includes: []string{"test"},
		},
	}
	err = ValidateConfig(feedConfig)
	if err == nil {
		t.Error("Expected error for invalid filter field, got none")
	}

	// Test with filter having no includes or excludes
	feedConfig.Filters = []Filter{
		{
			Field:    "title",
			Includes: []string{},
			Excludes: []string{},
		},
	}
	err = ValidateConfig(feedConfig)
	if err == nil {
		t.Error("Expected error for filter with no includes or excludes, got none")
	}

	// Test with valid feedConfig
	feedConfig.Filters = []Filter{
		{
			Field:    "title",
			Includes: []string{"test"},
		},
	}
	err = ValidateConfig(feedConfig)
	if err != nil {
		t.Errorf("Expected no error for valid feedConfig, got: %v", err)
	}
}

func TestValidateConfigFilterFields(t *testing.T) {
	feedConfig := &FeedConfig{
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
	validFields := []string{"title", "description", "content", "authors", "link", "categories"}
	for _, field := range validFields {
		feedConfig.Filters = []Filter{
			{
				Field:    field,
				Includes: []string{"test"},
			},
		}
		err := ValidateConfig(feedConfig)
		if err != nil {
			t.Errorf("Expected no error for valid filter field '%s', got: %v", field, err)
		}
	}

	// Test invalid filter field
	feedConfig.Filters = []Filter{
		{
			Field:    "invalid_field",
			Includes: []string{"test"},
		},
	}
	err := ValidateConfig(feedConfig)
	if err == nil {
		t.Error("Expected error for invalid filter field, got none")
	}
}
