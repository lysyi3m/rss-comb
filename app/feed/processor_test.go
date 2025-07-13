package feed

import (
	"fmt"
	"testing"
	"time"

	"github.com/lysyi3m/rss-comb/app/config"
	"github.com/lysyi3m/rss-comb/app/version"
)

func TestApplyFilters(t *testing.T) {
	processor := &Processor{}

	// Test item
	item := Item{
		Title:       "Technology News: Latest Updates",
		Description: "This is a technology article about programming",
		Content:     "Full content about programming and technology",
		AuthorName:  "John Doe",
		Link:        "https://example.com/tech-news",
		Categories:  []string{"Technology", "Programming", "c++", "1c"},
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
		{
			name: "Category exclude with special characters (c++)",
			filters: []config.Filter{
				{
					Field:    "categories",
					Excludes: []string{"c++"},
				},
			},
			expected: true, // Should be filtered (matches c++ in categories)
		},
		{
			name: "Category exclude with numbers (1c)",
			filters: []config.Filter{
				{
					Field:    "categories",
					Excludes: []string{"1c"},
				},
			},
			expected: true, // Should be filtered (matches 1c in categories)
		},
		{
			name: "Category include with special characters (c++)",
			filters: []config.Filter{
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
	processor := NewProcessor(nil, nil, fmt.Sprintf("RSS Comb/%s", version.GetVersion()), "8080")

	if processor == nil {
		t.Fatal("Expected processor to be created")
	}

	if processor.parser == nil {
		t.Error("Expected parser to be initialized")
	}

	if processor.client == nil {
		t.Error("Expected HTTP client to be initialized")
	}

	// Check default timeout
	if processor.client.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", processor.client.Timeout)
	}
}


// TestGetStats - removed since GetStats() method was identified as dead code
// The method only returned HTTP client timeout which is not meaningful runtime statistics