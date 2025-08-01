package feed

import (
	"testing"
	"time"
)

func TestFilterer_ApplyFilters_NoFilters(t *testing.T) {
	filterer := NewFilterer()

	items := []Item{
		{Title: "Test Item 1", Description: "Test description"},
		{Title: "Test Item 2", Description: "Another description"},
	}

	feedConfig := &Config{
		Filters: []ConfigFilter{}, // No filters
	}

	result := filterer.Run(items, feedConfig)

	if len(result) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result))
	}

	// When no filters are applied, all items should be unfiltered
	for i, item := range result {
		if item.IsFiltered {
			t.Errorf("Item %d should not be filtered when no filters are configured", i)
		}
	}
}

func TestFilterer_ApplyFilters_TitleIncludeFilter(t *testing.T) {
	filterer := NewFilterer()

	items := []Item{
		{Title: "Breaking News: Important Update", Description: "News description"},
		{Title: "Sports Update", Description: "Sports description"},
		{Title: "Weather Report", Description: "Weather description"},
	}

	feedConfig := &Config{
		Filters: []ConfigFilter{
			{
				Field:    "title",
				Includes: []string{"news", "update"},
			},
		},
	}

	result := filterer.Run(items, feedConfig)

	if len(result) != 3 {
		t.Errorf("Expected 3 items, got %d", len(result))
	}

	// First item should pass (contains "news" and "update")
	if result[0].IsFiltered {
		t.Errorf("First item should not be filtered, contains included terms")
	}

	// Second item should pass (contains "update")
	if result[1].IsFiltered {
		t.Errorf("Second item should not be filtered, contains 'update'")
	}

	// Third item should be filtered (doesn't contain "news" or "update")
	if !result[2].IsFiltered {
		t.Errorf("Third item should be filtered, doesn't contain included terms")
	}
}

func TestFilterer_ApplyFilters_TitleExcludeFilter(t *testing.T) {
	filterer := NewFilterer()

	items := []Item{
		{Title: "Breaking News", Description: "News description"},
		{Title: "Sports Update", Description: "Sports description"},
		{Title: "Advertisement: Buy Now!", Description: "Ad description"},
	}

	feedConfig := &Config{
		Filters: []ConfigFilter{
			{
				Field:    "title",
				Excludes: []string{"advertisement", "ad"},
			},
		},
	}

	result := filterer.Run(items, feedConfig)

	if len(result) != 3 {
		t.Errorf("Expected 3 items, got %d", len(result))
	}

	// First two items should pass
	if result[0].IsFiltered {
		t.Errorf("First item should not be filtered")
	}
	if result[1].IsFiltered {
		t.Errorf("Second item should not be filtered")
	}

	// Third item should be filtered (contains "advertisement")
	if !result[2].IsFiltered {
		t.Errorf("Third item should be filtered, contains excluded term")
	}
}

func TestFilterer_ApplyFilters_CombinedIncludeExclude(t *testing.T) {
	filterer := NewFilterer()

	items := []Item{
		{Title: "Tech News Update", Description: "Technology news"},
		{Title: "Tech Advertisement", Description: "Technology ad"},
		{Title: "Sports News", Description: "Sports update"},
		{Title: "Weather Report", Description: "Weather info"},
	}

	feedConfig := &Config{
		Filters: []ConfigFilter{
			{
				Field:    "title",
				Includes: []string{"tech", "news"},
				Excludes: []string{"advertisement", "ad"},
			},
		},
	}

	result := filterer.Run(items, feedConfig)

	// First item: contains "tech" and "news" (included) and doesn't contain excludes -> pass
	if result[0].IsFiltered {
		t.Errorf("First item should not be filtered")
	}

	// Second item: contains "tech" (included) but also contains "advertisement" (excluded) -> filtered
	if !result[1].IsFiltered {
		t.Errorf("Second item should be filtered due to excluded term")
	}

	// Third item: contains "news" (included) and doesn't contain excludes -> pass
	if result[2].IsFiltered {
		t.Errorf("Third item should not be filtered")
	}

	// Fourth item: doesn't contain any includes -> filtered
	if !result[3].IsFiltered {
		t.Errorf("Fourth item should be filtered, no included terms")
	}
}

func TestFilterer_ApplyFilters_MultipleFields(t *testing.T) {
	filterer := NewFilterer()

	items := []Item{
		{Title: "News Update", Description: "Technology article", Authors: []string{"tech@example.com (Tech Writer)"}},
		{Title: "Random Article", Description: "Random content", Authors: []string{"spam@example.com (Spammer)"}},
		{Title: "Sports News", Description: "Sports update", Authors: []string{"sports@example.com (Sports Writer)"}},
	}

	feedConfig := &Config{
		Filters: []ConfigFilter{
			{
				Field:    "title",
				Includes: []string{"news"},
			},
			{
				Field:    "authors",
				Excludes: []string{"spam"},
			},
		},
	}

	result := filterer.Run(items, feedConfig)

	// First item: title contains "news" and author doesn't contain "spam" -> pass
	if result[0].IsFiltered {
		t.Errorf("First item should not be filtered")
	}

	// Second item: title doesn't contain "news" -> filtered
	if !result[1].IsFiltered {
		t.Errorf("Second item should be filtered, title doesn't contain 'news'")
	}

	// Third item: title contains "news" and author doesn't contain "spam" -> pass
	if result[2].IsFiltered {
		t.Errorf("Third item should not be filtered")
	}
}

func TestFilterer_ApplyFilters_AuthorsField(t *testing.T) {
	filterer := NewFilterer()

	items := []Item{
		{Title: "Article 1", Authors: []string{"john@example.com (John Doe)", "jane@example.com (Jane Smith)"}},
		{Title: "Article 2", Authors: []string{"spammer@example.com (Spammer)"}},
	}

	feedConfig := &Config{
		Filters: []ConfigFilter{
			{
				Field:    "authors",
				Includes: []string{"john", "jane"},
			},
		},
	}

	result := filterer.Run(items, feedConfig)

	// First item: authors contain "john" and "jane" -> pass
	if result[0].IsFiltered {
		t.Errorf("First item should not be filtered")
	}

	// Second item: authors don't contain "john" or "jane" -> filtered
	if !result[1].IsFiltered {
		t.Errorf("Second item should be filtered")
	}
}

func TestFilterer_ApplyFilters_CategoriesField(t *testing.T) {
	filterer := NewFilterer()

	items := []Item{
		{Title: "Article 1", Categories: []string{"Technology", "News"}},
		{Title: "Article 2", Categories: []string{"Sports", "Entertainment"}},
	}

	feedConfig := &Config{
		Filters: []ConfigFilter{
			{
				Field:    "categories",
				Includes: []string{"technology", "news"},
			},
		},
	}

	result := filterer.Run(items, feedConfig)

	// First item: categories contain "technology" and "news" -> pass
	if result[0].IsFiltered {
		t.Errorf("First item should not be filtered")
	}

	// Second item: categories don't contain "technology" or "news" -> filtered
	if !result[1].IsFiltered {
		t.Errorf("Second item should be filtered")
	}
}

func TestFilterer_ApplyFilters_CaseInsensitive(t *testing.T) {
	filterer := NewFilterer()

	items := []Item{
		{Title: "BREAKING NEWS UPDATE"},
		{Title: "tech announcement"},
		{Title: "Sports Report"},
	}

	feedConfig := &Config{
		Filters: []ConfigFilter{
			{
				Field:    "title",
				Includes: []string{"News", "TECH"},
			},
		},
	}

	result := filterer.Run(items, feedConfig)

	// First item: title contains "NEWS" (case insensitive match with "News") -> pass
	if result[0].IsFiltered {
		t.Errorf("First item should not be filtered (case insensitive)")
	}

	// Second item: title contains "tech" (case insensitive match with "TECH") -> pass
	if result[1].IsFiltered {
		t.Errorf("Second item should not be filtered (case insensitive)")
	}

	// Third item: doesn't contain "news" or "tech" -> filtered
	if !result[2].IsFiltered {
		t.Errorf("Third item should be filtered")
	}
}

func TestFilterer_ApplyFilters_UnknownField(t *testing.T) {
	filterer := NewFilterer()

	items := []Item{
		{Title: "Test Article", Description: "Test description"},
	}

	feedConfig := &Config{
		Filters: []ConfigFilter{
			{
				Field:    "unknown_field",
				Includes: []string{"test"},
			},
		},
	}

	result := filterer.Run(items, feedConfig)

	// Item should be filtered because unknown field returns empty string
	if !result[0].IsFiltered {
		t.Errorf("Item should be filtered when using unknown field")
	}
}

func TestFilterer_ApplyFilters_EmptyValues(t *testing.T) {
	filterer := NewFilterer()

	items := []Item{
		{Title: "", Description: "", Content: ""},
		{Title: "Test", Description: "Test", Content: "Test"},
	}

	feedConfig := &Config{
		Filters: []ConfigFilter{
			{
				Field:    "title",
				Includes: []string{"test"},
			},
		},
	}

	result := filterer.Run(items, feedConfig)

	// First item: empty title doesn't contain "test" -> filtered
	if !result[0].IsFiltered {
		t.Errorf("First item should be filtered (empty title)")
	}

	// Second item: title contains "test" -> pass
	if result[1].IsFiltered {
		t.Errorf("Second item should not be filtered")
	}
}

func TestFilterer_ApplyFilters_PreservesOriginalData(t *testing.T) {
	filterer := NewFilterer()

	originalTime := time.Now()
	items := []Item{
		{
			GUID:        "test-guid-1",
			Title:       "Test Article",
			Link:        "https://example.com/1",
			Description: "Test description",
			Content:     "Test content",
			PublishedAt: originalTime,
			UpdatedAt:   &originalTime,
			Authors:     []string{"author@example.com"},
			Categories:  []string{"test"},
			ContentHash: "hash123",
		},
	}

	feedConfig := &Config{
		Filters: []ConfigFilter{
			{
				Field:    "title",
				Includes: []string{"test"},
			},
		},
	}

	result := filterer.Run(items, feedConfig)

	if len(result) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(result))
	}

	item := result[0]

	// Check that all original data is preserved
	if item.GUID != "test-guid-1" {
		t.Errorf("GUID not preserved: expected 'test-guid-1', got '%s'", item.GUID)
	}
	if item.Title != "Test Article" {
		t.Errorf("Title not preserved: expected 'Test Article', got '%s'", item.Title)
	}
	if item.Link != "https://example.com/1" {
		t.Errorf("Link not preserved: expected 'https://example.com/1', got '%s'", item.Link)
	}
	if item.Description != "Test description" {
		t.Errorf("Description not preserved: expected 'Test description', got '%s'", item.Description)
	}
	if item.Content != "Test content" {
		t.Errorf("Content not preserved: expected 'Test content', got '%s'", item.Content)
	}
	if item.PublishedAt != originalTime {
		t.Errorf("PublishedAt not preserved")
	}
	if item.UpdatedAt != &originalTime {
		t.Errorf("UpdatedAt not preserved")
	}
	if len(item.Authors) != 1 || item.Authors[0] != "author@example.com" {
		t.Errorf("Authors not preserved: expected ['author@example.com'], got %v", item.Authors)
	}
	if len(item.Categories) != 1 || item.Categories[0] != "test" {
		t.Errorf("Categories not preserved: expected ['test'], got %v", item.Categories)
	}
	if item.ContentHash != "hash123" {
		t.Errorf("ContentHash not preserved: expected 'hash123', got '%s'", item.ContentHash)
	}

	// Check that filter status is set correctly
	if item.IsFiltered {
		t.Errorf("Item should not be filtered")
	}
}

func TestFilterer_MatchesFieldFilter(t *testing.T) {
	filterer := NewFilterer()

	item := Item{
		Title:       "Test Title",
		Description: "Test Description",
		Content:     "Test Content",
		Authors:     []string{"author1@example.com", "author2@example.com"},
		Link:        "https://example.com",
		Categories:  []string{"cat1", "cat2"},
	}

	// Test string fields
	stringTests := []struct {
		field    string
		pattern  string
		expected bool
	}{
		{"title", "test", true},
		{"title", "xyz", false},
		{"description", "description", true},
		{"content", "content", true},
		{"link", "example.com", true},
		{"unknown", "test", false},
	}

	for _, test := range stringTests {
		result := filterer.matchesFieldFilter(item, test.field, test.pattern)
		if result != test.expected {
			t.Errorf("matchesFieldFilter(%s, %s): expected %v, got %v", test.field, test.pattern, test.expected, result)
		}
	}

	// Test array fields
	if !filterer.matchesFieldFilter(item, "authors", "author1") {
		t.Errorf("Should match first author")
	}
	if !filterer.matchesFieldFilter(item, "authors", "author2") {
		t.Errorf("Should match second author")
	}
	if filterer.matchesFieldFilter(item, "authors", "nonexistent") {
		t.Errorf("Should not match nonexistent author")
	}

	if !filterer.matchesFieldFilter(item, "categories", "cat1") {
		t.Errorf("Should match first category")
	}
	if !filterer.matchesFieldFilter(item, "categories", "cat2") {
		t.Errorf("Should match second category")
	}
	if filterer.matchesFieldFilter(item, "categories", "nonexistent") {
		t.Errorf("Should not match nonexistent category")
	}
}

func TestFilterer_ArrayFilterBugFix(t *testing.T) {
	filterer := NewFilterer()

	// Test the specific bug case you mentioned
	items := []Item{
		{
			Title:      "Test Article",
			Categories: []string{"Category ABC", "Category XYZ", "C Category"},
		},
	}

	// This should match only the exact "C Category" element, not as substring of joined string
	feedConfig := &Config{
		Filters: []ConfigFilter{
			{
				Field:    "categories",
				Includes: []string{"C Category"},
			},
		},
	}

	result := filterer.Run(items, feedConfig)

	// Item should NOT be filtered because "C Category" exists as exact match
	if result[0].IsFiltered {
		t.Errorf("Item should not be filtered - 'C Category' exists as exact element")
	}

	// Test case that should be filtered
	items2 := []Item{
		{
			Title:      "Test Article 2",
			Categories: []string{"Category ABC", "Category XYZ"}, // No "C Category"
		},
	}

	result2 := filterer.Run(items2, feedConfig)

	// This item should be filtered because "C Category" doesn't exist as exact match
	if !result2[0].IsFiltered {
		t.Errorf("Item should be filtered - 'C Category' does not exist as exact element")
	}

	// Test authors field with similar issue
	items3 := []Item{
		{
			Title:   "Test Article 3",
			Authors: []string{"john@example.com (John Doe)", "jane@example.com (Jane Smith)", "jo@example.com (Jo)"},
		},
	}

	feedConfig3 := &Config{
		Filters: []ConfigFilter{
			{
				Field:    "authors",
				Includes: []string{"jo@example.com"}, // Should match exactly, not as substring
			},
		},
	}

	result3 := filterer.Run(items3, feedConfig3)

	// Should NOT be filtered because "jo@example.com" exists as substring in the third author
	if result3[0].IsFiltered {
		t.Errorf("Item should not be filtered - 'jo@example.com' exists in author element")
	}
}

func TestFilterer_ArrayFilterExactMatch(t *testing.T) {
	filterer := NewFilterer()

	// Test that we match individual elements, not joined strings
	items := []Item{
		{
			Title:      "Test Article",
			Categories: []string{"Tech News", "Breaking"},
		},
	}

	// This should NOT match because "Tech" and "News" are in same element "Tech News"
	// but "News Breaking" doesn't exist as single element
	feedConfig := &Config{
		Filters: []ConfigFilter{
			{
				Field:    "categories",
				Includes: []string{"News Breaking"}, // This should not match
			},
		},
	}

	result := filterer.Run(items, feedConfig)

	// Should be filtered because "News Breaking" doesn't exist as exact element
	if !result[0].IsFiltered {
		t.Errorf("Item should be filtered - 'News Breaking' does not exist as exact element")
	}

	// Test positive case - should match "Tech News" exactly
	feedConfig2 := &Config{
		Filters: []ConfigFilter{
			{
				Field:    "categories",
				Includes: []string{"Tech News"},
			},
		},
	}

	result2 := filterer.Run(items, feedConfig2)

	// Should NOT be filtered because "Tech News" exists as exact element
	if result2[0].IsFiltered {
		t.Errorf("Item should not be filtered - 'Tech News' exists as exact element")
	}
}

func TestFilterer_MatchesPattern(t *testing.T) {
	filterer := NewFilterer()

	tests := []struct {
		value    string
		pattern  string
		expected bool
	}{
		{"Hello World", "hello", true},
		{"Hello World", "WORLD", true},
		{"Hello World", "xyz", false},
		{"", "test", false},
		{"test", "", true}, // Empty pattern matches everything
		{"UPPERCASE", "upper", true},
		{"lowercase", "LOWER", true},
	}

	for _, test := range tests {
		result := filterer.matchesPattern(test.value, test.pattern)
		if result != test.expected {
			t.Errorf("matchesPattern('%s', '%s'): expected %v, got %v", test.value, test.pattern, test.expected, result)
		}
	}
}
