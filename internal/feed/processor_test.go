package feed

import (
	"testing"
	"time"

	"github.com/lysyi3m/rss-comb/internal/config"
	"github.com/lysyi3m/rss-comb/internal/parser"
)

func TestApplyFilters(t *testing.T) {
	processor := &Processor{}

	// Test item
	item := parser.NormalizedItem{
		Title:       "Technology News: Latest Updates",
		Description: "This is a technology article about programming",
		Content:     "Full content about programming and technology",
		AuthorName:  "John Doe",
		Link:        "https://example.com/tech-news",
		Categories:  []string{"Technology", "Programming"},
	}

	tests := []struct {
		name     string
		filters  []config.Filter
		expected bool
		reason   string
	}{
		{
			name: "Include filter matches",
			filters: []config.Filter{
				{
					Field:    "title",
					Includes: []string{"technology"},
				},
			},
			expected: false, // Should not be filtered (matches include)
		},
		{
			name: "Include filter doesn't match",
			filters: []config.Filter{
				{
					Field:    "title",
					Includes: []string{"sports"},
				},
			},
			expected: true, // Should be filtered (doesn't match include)
		},
		{
			name: "Exclude filter matches",
			filters: []config.Filter{
				{
					Field:    "title",
					Excludes: []string{"news"},
				},
			},
			expected: true, // Should be filtered (matches exclude)
		},
		{
			name: "Exclude filter doesn't match",
			filters: []config.Filter{
				{
					Field:    "title",
					Excludes: []string{"sports"},
				},
			},
			expected: false, // Should not be filtered (doesn't match exclude)
		},
		{
			name: "Include and exclude - include matches, exclude doesn't",
			filters: []config.Filter{
				{
					Field:    "title",
					Includes: []string{"technology"},
					Excludes: []string{"sports"},
				},
			},
			expected: false, // Should not be filtered
		},
		{
			name: "Include and exclude - both match (exclude wins)",
			filters: []config.Filter{
				{
					Field:    "title",
					Includes: []string{"technology"},
					Excludes: []string{"news"},
				},
			},
			expected: true, // Should be filtered (exclude takes precedence)
		},
		{
			name: "Multiple filters - all pass",
			filters: []config.Filter{
				{
					Field:    "title",
					Includes: []string{"technology"},
				},
				{
					Field:    "description",
					Includes: []string{"programming"},
				},
			},
			expected: false, // Should not be filtered
		},
		{
			name: "Multiple filters - one fails",
			filters: []config.Filter{
				{
					Field:    "title",
					Includes: []string{"technology"},
				},
				{
					Field:    "description",
					Includes: []string{"sports"},
				},
			},
			expected: true, // Should be filtered (second filter fails)
		},
		{
			name: "Categories filter",
			filters: []config.Filter{
				{
					Field:    "categories",
					Includes: []string{"programming"},
				},
			},
			expected: false, // Should not be filtered (categories contains programming)
		},
		{
			name: "Author filter",
			filters: []config.Filter{
				{
					Field:    "author",
					Includes: []string{"john"},
				},
			},
			expected: false, // Should not be filtered (author contains john)
		},
		{
			name: "Case insensitive matching",
			filters: []config.Filter{
				{
					Field:    "title",
					Includes: []string{"TECHNOLOGY"},
				},
			},
			expected: false, // Should not be filtered (case insensitive)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered, reason := processor.applyFilters(item, tt.filters)
			if filtered != tt.expected {
				t.Errorf("Expected filtered=%v, got %v. Reason: %s", tt.expected, filtered, reason)
			}
			if filtered && reason == "" {
				t.Error("Expected reason to be provided when item is filtered")
			}
		})
	}
}

func TestGetFieldValue(t *testing.T) {
	processor := &Processor{}

	item := parser.NormalizedItem{
		Title:       "Test Title",
		Description: "Test Description",
		Content:     "Test Content",
		AuthorName:  "Test Author",
		Link:        "https://example.com",
		Categories:  []string{"cat1", "cat2"},
	}

	tests := []struct {
		field    string
		expected string
	}{
		{"title", "Test Title"},
		{"description", "Test Description"},
		{"content", "Test Content"},
		{"author", "Test Author"},
		{"link", "https://example.com"},
		{"categories", "cat1 cat2"},
		{"invalid", ""},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			result := processor.getFieldValue(item, tt.field)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestNewProcessor(t *testing.T) {
	// This is a simple test to ensure NewProcessor doesn't panic
	// In a real scenario, you'd pass actual instances
	configs := make(map[string]*config.FeedConfig)
	processor := NewProcessor(nil, nil, nil, configs)

	if processor == nil {
		t.Error("Expected processor to be created")
	}

	if processor.configs == nil {
		t.Error("Expected configs to be initialized")
	}

	if processor.client == nil {
		t.Error("Expected HTTP client to be initialized")
	}

	// Check default timeout
	if processor.client.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", processor.client.Timeout)
	}
}

func TestReloadConfigs(t *testing.T) {
	processor := &Processor{configs: make(map[string]*config.FeedConfig)}

	newConfigs := map[string]*config.FeedConfig{
		"test.yaml": {
			Feed: config.FeedInfo{
				URL:  "https://example.com/feed.xml",
				Name: "Test Feed",
			},
		},
	}

	processor.ReloadConfigs(newConfigs)

	if len(processor.configs) != 1 {
		t.Errorf("Expected 1 config after reload, got %d", len(processor.configs))
	}

	if processor.configs["test.yaml"] == nil {
		t.Error("Expected test.yaml config to be loaded")
	}
}

func TestGetStats(t *testing.T) {
	configs := map[string]*config.FeedConfig{
		"feed1.yaml": {},
		"feed2.yaml": {},
	}
	processor := NewProcessor(nil, nil, nil, configs)

	stats := processor.GetStats()

	if stats["loaded_configs"] != 2 {
		t.Errorf("Expected loaded_configs=2, got %v", stats["loaded_configs"])
	}

	if stats["client_timeout"] == "" {
		t.Error("Expected client_timeout to be set")
	}
}