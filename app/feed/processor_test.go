package feed

import (
	"testing"

	"github.com/lysyi3m/rss-comb/app/feed_config"
)

// MockAppConfig for testing
type MockAppConfig struct {
	Port              string
	UserAgent         string
	APIAccessKey      string
	WorkerCount       int
	SchedulerInterval int
}

func (c *MockAppConfig) GetPort() string { return c.Port }
func (c *MockAppConfig) GetUserAgent() string { return c.UserAgent }
func (c *MockAppConfig) GetAPIAccessKey() string { return c.APIAccessKey }
func (c *MockAppConfig) GetWorkerCount() int { return c.WorkerCount }
func (c *MockAppConfig) GetSchedulerInterval() int { return c.SchedulerInterval }

func TestApplyFilters(t *testing.T) {
	processor := &Processor{}

	// Test item
	item := Item{
		Title:       "Technology News: Latest Updates",
		Description: "This is a technology article about programming",
		Content:     "Full content about programming and technology",
		Authors:     []string{"John Doe"},
		Link:        "https://example.com/tech-news",
		Categories:  []string{"Technology", "Programming", "c++", "1c"},
	}

	tests := []struct {
		name     string
		filters  []feed_config.Filter
		expected bool
		reason   string
	}{
		{
			name: "Include filter matches",
			filters: []feed_config.Filter{
				{
					Field:    "title",
					Includes: []string{"technology"},
				},
			},
			expected: false, // Should not be filtered (matches include)
		},
		{
			name: "Include filter doesn't match",
			filters: []feed_config.Filter{
				{
					Field:    "title",
					Includes: []string{"sports"},
				},
			},
			expected: true, // Should be filtered (doesn't match include)
		},
		{
			name: "Exclude filter matches",
			filters: []feed_config.Filter{
				{
					Field:    "title",
					Excludes: []string{"news"},
				},
			},
			expected: true, // Should be filtered (matches exclude)
		},
		{
			name: "Exclude filter doesn't match",
			filters: []feed_config.Filter{
				{
					Field:    "title",
					Excludes: []string{"sports"},
				},
			},
			expected: false, // Should not be filtered (doesn't match exclude)
		},
		{
			name: "Include and exclude - include matches, exclude doesn't",
			filters: []feed_config.Filter{
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
			filters: []feed_config.Filter{
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
			filters: []feed_config.Filter{
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
			filters: []feed_config.Filter{
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
			filters: []feed_config.Filter{
				{
					Field:    "categories",
					Includes: []string{"programming"},
				},
			},
			expected: false, // Should not be filtered (categories contains programming)
		},
		{
			name: "Authors filter",
			filters: []feed_config.Filter{
				{
					Field:    "authors",
					Includes: []string{"john"},
				},
			},
			expected: false, // Should not be filtered (authors contains john)
		},
		{
			name: "Case insensitive matching",
			filters: []feed_config.Filter{
				{
					Field:    "title",
					Includes: []string{"TECHNOLOGY"},
				},
			},
			expected: false, // Should not be filtered (case insensitive)
		},
		{
			name: "Category exclude with special characters (c++)",
			filters: []feed_config.Filter{
				{
					Field:    "categories",
					Excludes: []string{"c++"},
				},
			},
			expected: true, // Should be filtered (matches c++ in categories)
		},
		{
			name: "Category exclude with numbers (1c)",
			filters: []feed_config.Filter{
				{
					Field:    "categories",
					Excludes: []string{"1c"},
				},
			},
			expected: true, // Should be filtered (matches 1c in categories)
		},
		{
			name: "Category include with special characters (c++)",
			filters: []feed_config.Filter{
				{
					Field:    "categories",
					Includes: []string{"c++"},
				},
			},
			expected: false, // Should not be filtered (includes c++)
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

	item := Item{
		Title:       "Test Title",
		Description: "Test Description",
		Content:     "Test Content",
		Authors:     []string{"Test Author"},
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
		{"authors", "Test Author"},
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

// TestNewProcessor was removed because it requires config.Load() to be called first
// which is part of the application initialization, not unit test scope

// TestGetStats - removed since GetStats() method was identified as dead code
// The method only returned HTTP client timeout which is not meaningful runtime statistics
