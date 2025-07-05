package api

import (
	"strings"
	"testing"
	"time"

	"github.com/lysyi3m/rss-comb/internal/database"
)

func TestRSSGeneratorGenerate(t *testing.T) {
	generator := NewRSSGenerator("8080")

	// Create test feed
	feed := database.Feed{
		ID:      "test-feed-id",
		Name:    "Test Feed",
		URL:     "https://example.com/feed.xml",
		IconURL: "https://example.com/icon.png",
	}

	// Create test items
	publishedDate := time.Date(2023, 7, 3, 12, 0, 0, 0, time.UTC)
	items := []database.Item{
		{
			ID:            "item-1",
			GUID:          "https://example.com/item1",
			Title:         "Test Item 1",
			Link:          "https://example.com/item1",
			Description:   "Test description 1",
			Content:       "Test content 1",
			PublishedDate: &publishedDate,
			AuthorName:    "Test Author",
			AuthorEmail:   "test@example.com",
			Categories:    []string{"Technology", "Programming"},
		},
		{
			ID:          "item-2",
			GUID:        "item-2",
			Title:       "Test Item 2",
			Link:        "https://example.com/item2",
			Description: "Test description 2",
		},
	}

	rss, err := generator.Generate(feed, items)
	if err != nil {
		t.Fatalf("Failed to generate RSS: %v", err)
	}

	// Test basic structure
	if !strings.Contains(rss, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("RSS should contain XML declaration")
	}

	if !strings.Contains(rss, `<rss version="2.0"`) {
		t.Error("RSS should contain RSS 2.0 declaration")
	}

	if !strings.Contains(rss, "<channel>") {
		t.Error("RSS should contain channel element")
	}

	// Test feed metadata
	if !strings.Contains(rss, "<title>Test Feed</title>") {
		t.Error("RSS should contain feed title")
	}

	if !strings.Contains(rss, "<link>https://example.com/feed.xml</link>") {
		t.Error("RSS should contain feed link")
	}

	if !strings.Contains(rss, "<generator>RSS-Comb/1.0</generator>") {
		t.Error("RSS should contain generator")
	}

	// Test image element
	if !strings.Contains(rss, "<image>") {
		t.Error("RSS should contain image element when icon URL is provided")
	}

	// Test items
	if !strings.Contains(rss, "<item>") {
		t.Error("RSS should contain item elements")
	}

	if !strings.Contains(rss, "<title>Test Item 1</title>") {
		t.Error("RSS should contain item title")
	}

	if !strings.Contains(rss, `<guid isPermaLink="true">https://example.com/item1</guid>`) {
		t.Error("RSS should contain GUID with correct isPermaLink for URL")
	}

	if !strings.Contains(rss, `<guid isPermaLink="false">item-2</guid>`) {
		t.Error("RSS should contain GUID with correct isPermaLink for non-URL")
	}

	// Test author format
	if !strings.Contains(rss, "<author>test@example.com (Test Author)</author>") {
		t.Error("RSS should contain properly formatted author")
	}

	// Test categories
	if !strings.Contains(rss, "<category>Technology</category>") {
		t.Error("RSS should contain categories")
	}

	// Test content encoding
	if !strings.Contains(rss, "<content:encoded><![CDATA[Test content 1]]></content:encoded>") {
		t.Error("RSS should contain content:encoded for content different from description")
	}

	// Count items
	itemCount := strings.Count(rss, "<item>")
	if itemCount != 2 {
		t.Errorf("Expected 2 items, found %d", itemCount)
	}
}

func TestRSSGeneratorGenerateEmpty(t *testing.T) {
	generator := NewRSSGenerator("8080")

	rss := generator.GenerateEmpty("Test Feed", "https://example.com/feed.xml")

	// Test basic structure
	if !strings.Contains(rss, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("Empty RSS should contain XML declaration")
	}

	if !strings.Contains(rss, "<title>Test Feed</title>") {
		t.Error("Empty RSS should contain feed title")
	}

	if !strings.Contains(rss, "Feed is being processed") {
		t.Error("Empty RSS should contain processing message")
	}

	// Should not contain any items
	if strings.Contains(rss, "<item>") {
		t.Error("Empty RSS should not contain any items")
	}
}

func TestRSSGeneratorGenerateError(t *testing.T) {
	generator := NewRSSGenerator("8080")

	rss := generator.GenerateError("Test Feed", "https://example.com/feed.xml", "Connection timeout")

	// Test basic structure
	if !strings.Contains(rss, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("Error RSS should contain XML declaration")
	}

	if !strings.Contains(rss, "<title>Test Feed - Error</title>") {
		t.Error("Error RSS should contain error title")
	}

	if !strings.Contains(rss, "Connection timeout") {
		t.Error("Error RSS should contain error message")
	}

	// Should contain one error item
	itemCount := strings.Count(rss, "<item>")
	if itemCount != 1 {
		t.Errorf("Error RSS should contain exactly 1 error item, found %d", itemCount)
	}

	if !strings.Contains(rss, "<title>Feed Processing Error</title>") {
		t.Error("Error RSS should contain error item title")
	}
}

func TestIsURL(t *testing.T) {
	generator := NewRSSGenerator("8080")

	tests := []struct {
		input    string
		expected bool
	}{
		{"https://example.com", true},
		{"http://example.com", true},
		{"https://example.com/path", true},
		{"http://example.com/path", true},
		{"item-123", false},
		{"guid-456", false},
		{"", false},
		{"ftp://example.com", false},
		{"httpx://example.com", false},
	}

	for _, test := range tests {
		result := generator.isURL(test.input)
		if result != test.expected {
			t.Errorf("isURL(%q) = %v, expected %v", test.input, result, test.expected)
		}
	}
}

func TestGenerateWithEscaping(t *testing.T) {
	generator := NewRSSGenerator("8080")

	// Test with content that needs escaping
	feed := database.Feed{
		ID:   "test-feed-id",
		Name: "Feed with <tags> & \"quotes\"",
		URL:  "https://example.com/feed.xml",
	}

	items := []database.Item{
		{
			ID:          "item-1",
			GUID:        "item-1",
			Title:       "Title with <tags> & \"quotes\"",
			Description: "Description with <html> & special chars",
		},
	}

	rss, err := generator.Generate(feed, items)
	if err != nil {
		t.Fatalf("Failed to generate RSS: %v", err)
	}

	// Test that special characters are properly escaped
	if strings.Contains(rss, "<tags>") {
		t.Error("RSS should not contain unescaped HTML tags")
	}

	if !strings.Contains(rss, "&lt;") || !strings.Contains(rss, "&gt;") {
		t.Error("RSS should contain escaped HTML entities")
	}

	// Should still be valid XML structure
	if !strings.Contains(rss, "<title>") && !strings.Contains(rss, "</title>") {
		t.Error("RSS should still contain proper XML structure")
	}
}