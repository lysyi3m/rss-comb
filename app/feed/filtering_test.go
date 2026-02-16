package feed

import (
	"testing"
	"time"

	"github.com/lysyi3m/rss-comb/app/types"
)

func TestFilterer_ApplyFilters_NoFilters(t *testing.T) {
	items := []types.Item{
		{Title: "Test Item 1", Description: "Test description"},
		{Title: "Test Item 2", Description: "Another description"},
	}

	feedConfig := &Config{
		Filters: []types.Filter{}, // No filters
	}

	result := Filter(items, feedConfig.Filters)

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
	items := []types.Item{
		{Title: "Breaking News: Important Update", Description: "News description"},
		{Title: "Sports Update", Description: "Sports description"},
		{Title: "Weather Report", Description: "Weather description"},
	}

	feedConfig := &Config{
		Filters: []types.Filter{
			{
				Field:    "title",
				Includes: []string{"news", "update"},
			},
		},
	}

	result := Filter(items, feedConfig.Filters)

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
	items := []types.Item{
		{Title: "Breaking News", Description: "News description"},
		{Title: "Sports Update", Description: "Sports description"},
		{Title: "Advertisement: Buy Now!", Description: "Ad description"},
	}

	feedConfig := &Config{
		Filters: []types.Filter{
			{
				Field:    "title",
				Excludes: []string{"advertisement", "ad"},
			},
		},
	}

	result := Filter(items, feedConfig.Filters)

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
	items := []types.Item{
		{Title: "Tech News Update", Description: "Technology news"},
		{Title: "Tech Advertisement", Description: "Technology ad"},
		{Title: "Sports News", Description: "Sports update"},
		{Title: "Weather Report", Description: "Weather info"},
	}

	feedConfig := &Config{
		Filters: []types.Filter{
			{
				Field:    "title",
				Includes: []string{"tech", "news"},
				Excludes: []string{"advertisement", "ad"},
			},
		},
	}

	result := Filter(items, feedConfig.Filters)

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
	items := []types.Item{
		{Title: "News Update", Description: "Technology article", Authors: []string{"tech@example.com (Tech Writer)"}},
		{Title: "Random Article", Description: "Random content", Authors: []string{"spam@example.com (Spammer)"}},
		{Title: "Sports News", Description: "Sports update", Authors: []string{"sports@example.com (Sports Writer)"}},
	}

	feedConfig := &Config{
		Filters: []types.Filter{
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

	result := Filter(items, feedConfig.Filters)

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
	items := []types.Item{
		{Title: "Article 1", Authors: []string{"john@example.com (John Doe)", "jane@example.com (Jane Smith)"}},
		{Title: "Article 2", Authors: []string{"spammer@example.com (Spammer)"}},
	}

	feedConfig := &Config{
		Filters: []types.Filter{
			{
				Field:    "authors",
				Includes: []string{"john", "jane"},
			},
		},
	}

	result := Filter(items, feedConfig.Filters)

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
	items := []types.Item{
		{Title: "Article 1", Categories: []string{"Technology", "News"}},
		{Title: "Article 2", Categories: []string{"Sports", "Entertainment"}},
	}

	feedConfig := &Config{
		Filters: []types.Filter{
			{
				Field:    "categories",
				Includes: []string{"technology", "news"},
			},
		},
	}

	result := Filter(items, feedConfig.Filters)

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
	items := []types.Item{
		{Title: "BREAKING NEWS UPDATE"},
		{Title: "tech announcement"},
		{Title: "Sports Report"},
	}

	feedConfig := &Config{
		Filters: []types.Filter{
			{
				Field:    "title",
				Includes: []string{"News", "TECH"},
			},
		},
	}

	result := Filter(items, feedConfig.Filters)

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
	items := []types.Item{
		{Title: "Test Article", Description: "Test description"},
	}

	feedConfig := &Config{
		Filters: []types.Filter{
			{
				Field:    "unknown_field",
				Includes: []string{"test"},
			},
		},
	}

	result := Filter(items, feedConfig.Filters)

	// Item should be filtered because unknown field returns empty string
	if !result[0].IsFiltered {
		t.Errorf("Item should be filtered when using unknown field")
	}
}

func TestFilterer_ApplyFilters_EmptyValues(t *testing.T) {
	items := []types.Item{
		{Title: "", Description: "", Content: ""},
		{Title: "Test", Description: "Test", Content: "Test"},
	}

	feedConfig := &Config{
		Filters: []types.Filter{
			{
				Field:    "title",
				Includes: []string{"test"},
			},
		},
	}

	result := Filter(items, feedConfig.Filters)

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
	originalTime := time.Now()
	items := []types.Item{
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
		Filters: []types.Filter{
			{
				Field:    "title",
				Includes: []string{"test"},
			},
		},
	}

	result := Filter(items, feedConfig.Filters)

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

func TestMatchesFieldFilter(t *testing.T) {
	item := types.Item{
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
		result := matchesFieldFilter(item, test.field, test.pattern)
		if result != test.expected {
			t.Errorf("matchesFieldFilter(%s, %s): expected %v, got %v", test.field, test.pattern, test.expected, result)
		}
	}

	// Test array fields
	if !matchesFieldFilter(item, "authors", "author1") {
		t.Errorf("Should match first author")
	}
	if !matchesFieldFilter(item, "authors", "author2") {
		t.Errorf("Should match second author")
	}
	if matchesFieldFilter(item, "authors", "nonexistent") {
		t.Errorf("Should not match nonexistent author")
	}

	if !matchesFieldFilter(item, "categories", "cat1") {
		t.Errorf("Should match first category")
	}
	if !matchesFieldFilter(item, "categories", "cat2") {
		t.Errorf("Should match second category")
	}
	if matchesFieldFilter(item, "categories", "nonexistent") {
		t.Errorf("Should not match nonexistent category")
	}
}

func TestFilterer_ArrayFilterBugFix(t *testing.T) {
	// Test the specific bug case you mentioned
	items := []types.Item{
		{
			Title:      "Test Article",
			Categories: []string{"Category ABC", "Category XYZ", "C Category"},
		},
	}

	// This should match only the exact "C Category" element, not as substring of joined string
	feedConfig := &Config{
		Filters: []types.Filter{
			{
				Field:    "categories",
				Includes: []string{"C Category"},
			},
		},
	}

	result := Filter(items, feedConfig.Filters)

	// Item should NOT be filtered because "C Category" exists as exact match
	if result[0].IsFiltered {
		t.Errorf("Item should not be filtered - 'C Category' exists as exact element")
	}

	// Test case that should be filtered
	items2 := []types.Item{
		{
			Title:      "Test Article 2",
			Categories: []string{"Category ABC", "Category XYZ"}, // No "C Category"
		},
	}

	result2 := Filter(items2, feedConfig.Filters)

	// This item should be filtered because "C Category" doesn't exist as exact match
	if !result2[0].IsFiltered {
		t.Errorf("Item should be filtered - 'C Category' does not exist as exact element")
	}

	// Test authors field with similar issue
	items3 := []types.Item{
		{
			Title:   "Test Article 3",
			Authors: []string{"john@example.com (John Doe)", "jane@example.com (Jane Smith)", "jo@example.com (Jo)"},
		},
	}

	feedConfig3 := &Config{
		Filters: []types.Filter{
			{
				Field:    "authors",
				Includes: []string{"jo@example.com"}, // Should match exactly, not as substring
			},
		},
	}

	result3 := Filter(items3, feedConfig3.Filters)

	// Should NOT be filtered because "jo@example.com" exists as substring in the third author
	if result3[0].IsFiltered {
		t.Errorf("Item should not be filtered - 'jo@example.com' exists in author element")
	}
}

func TestFilterer_ArrayFilterExactMatch(t *testing.T) {
	// Test that we match individual elements, not joined strings
	items := []types.Item{
		{
			Title:      "Test Article",
			Categories: []string{"Tech News", "Breaking"},
		},
	}

	// This should NOT match because "Tech" and "News" are in same element "Tech News"
	// but "News Breaking" doesn't exist as single element
	feedConfig := &Config{
		Filters: []types.Filter{
			{
				Field:    "categories",
				Includes: []string{"News Breaking"}, // This should not match
			},
		},
	}

	result := Filter(items, feedConfig.Filters)

	// Should be filtered because "News Breaking" doesn't exist as exact element
	if !result[0].IsFiltered {
		t.Errorf("Item should be filtered - 'News Breaking' does not exist as exact element")
	}

	// Test positive case - should match "Tech News" exactly
	feedConfig2 := &Config{
		Filters: []types.Filter{
			{
				Field:    "categories",
				Includes: []string{"Tech News"},
			},
		},
	}

	result2 := Filter(items, feedConfig2.Filters)

	// Should NOT be filtered because "Tech News" exists as exact element
	if result2[0].IsFiltered {
		t.Errorf("Item should not be filtered - 'Tech News' exists as exact element")
	}
}

func TestFilterer_MatchesPattern(t *testing.T) {
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
		result := matchesPattern(test.value, test.pattern)
		if result != test.expected {
			t.Errorf("matchesPattern('%s', '%s'): expected %v, got %v", test.value, test.pattern, test.expected, result)
		}
	}
}

func TestFilterer_WhitespaceNormalization(t *testing.T) {
	// Test various whitespace scenarios
	items := []types.Item{
		{
			Title: "Test\u00a0with\u00a0NBSP", // non-breaking spaces
		},
		{
			Title: "Test  with  double  spaces",
		},
		{
			Title: "Test\twith\ttabs",
		},
		{
			Title: "Test\nwith\nnewlines",
		},
		{
			Title: "Test\u2009with\u2009thin\u2009spaces", // thin spaces
		},
	}

	filters := []types.Filter{
		{
			Field:    "title",
			Includes: []string{"test with"},
		},
	}

	result := Filter(items, filters)

	// All items should pass because after normalization they all contain "test with"
	for i, item := range result {
		if item.IsFiltered {
			t.Errorf("Item %d should not be filtered after whitespace normalization. Title: %q", i, item.Title)
		}
	}
}

func TestFilterer_CyrillicWithNBSP(t *testing.T) {
	// Test Cyrillic text with NBSP instead of regular space
	items := []types.Item{
		{
			Title: "Тестовый\u00a0заголовок статьи", // Test title with NBSP
		},
	}

	filters := []types.Filter{
		{
			Field:    "title",
			Excludes: []string{"Тестовый заголовок"}, // Filter with regular space
		},
	}

	result := Filter(items, filters)

	// Item should be filtered even though the title has NBSP instead of regular space
	if !result[0].IsFiltered {
		t.Errorf("Item should be filtered despite NBSP in title")
	}
}

func TestFilterer_MultipleConsecutiveSpaces(t *testing.T) {
	items := []types.Item{
		{
			Title: "Breaking    News    Update", // multiple spaces
		},
	}

	filters := []types.Filter{
		{
			Field:    "title",
			Includes: []string{"breaking news update"},
		},
	}

	result := Filter(items, filters)

	// Should match after collapsing multiple spaces
	if result[0].IsFiltered {
		t.Errorf("Item should not be filtered after collapsing multiple spaces")
	}
}

func TestFilterer_LeadingTrailingWhitespace(t *testing.T) {
	items := []types.Item{
		{
			Title: "  Breaking News  ",
		},
	}

	filters := []types.Filter{
		{
			Field:    "title",
			Includes: []string{"breaking news"},
		},
	}

	result := Filter(items, filters)

	// Should match after trimming whitespace
	if result[0].IsFiltered {
		t.Errorf("Item should not be filtered after trimming whitespace")
	}
}

func TestFilterer_UnicodeNormalization(t *testing.T) {
	// Test Cyrillic 'й' which can be represented two ways:
	// 1. As single character 'й' (U+0439) = composed form
	// 2. As 'и' (U+0438) + combining breve (U+0306) = decomposed form

	// Create test title with decomposed 'й' (и + combining breve)
	// This is how it appears in some RSS feeds
	titleDecomposedBytes := []byte{
		// Новый (New)
		208, 157, // Н
		208, 190, // о
		208, 178, // в
		209, 139, // ы
		208, 184, 204, 134, // й (decomposed: и + combining breve)
		32, // space
		// тест (test)
		209, 130, // т
		208, 181, // е
		209, 129, // с
		209, 130, // т
	}

	items := []types.Item{
		{
			// Title with composed 'й' (normal form)
			Title: "Новый тест",
		},
		{
			// Title with decomposed 'й' (и + combining breve)
			Title: string(titleDecomposedBytes),
		},
	}

	filters := []types.Filter{
		{
			Field: "title",
			// Filter pattern with decomposed form
			Excludes: []string{string(titleDecomposedBytes)},
		},
	}

	result := Filter(items, filters)

	// Both items should be filtered despite different Unicode representations
	for i, item := range result {
		if !item.IsFiltered {
			t.Errorf("Item %d should be filtered after Unicode normalization. Title: %q (bytes: %v)", i, item.Title, []byte(item.Title))
		}
	}
}

func TestFilterer_UnicodeNormalizationReverse(t *testing.T) {
	// Test the reverse case: composed filter pattern, decomposed title

	// Create title with decomposed 'й'
	titleDecomposedBytes := []byte{
		// Новый with decomposed й
		208, 157, 208, 190, 208, 178, 209, 139,
		208, 184, 204, 134, // й (decomposed)
		32,                         // space
		208, 177, 208, 187, 208, 190, 208, 179, // блог (blog)
	}

	items := []types.Item{
		{
			// Title with decomposed 'й'
			Title: string(titleDecomposedBytes),
		},
	}

	filters := []types.Filter{
		{
			Field: "title",
			// Filter pattern with composed form (normal)
			Excludes: []string{"Новый блог"},
		},
	}

	result := Filter(items, filters)

	// Should be filtered after normalization
	if !result[0].IsFiltered {
		t.Errorf("Item should be filtered after Unicode normalization. Title bytes: %v", []byte(result[0].Title))
	}
}

func TestFilterer_RegexPattern_Basic(t *testing.T) {
	items := []types.Item{
		{Title: "Tech News Update"},
		{Title: "Technology Article"},
		{Title: "Sports News"},
		{Title: "Random Post"},
	}

	filters := []types.Filter{
		{
			Field:    "title",
			Includes: []string{"/^tech/"}, // Regex: starts with "tech" (case insensitive)
		},
	}

	result := Filter(items, filters)

	// First two items should pass (start with "tech")
	if result[0].IsFiltered {
		t.Errorf("First item should not be filtered (matches /^tech/)")
	}
	if result[1].IsFiltered {
		t.Errorf("Second item should not be filtered (matches /^tech/)")
	}

	// Last two items should be filtered (don't start with "tech")
	if !result[2].IsFiltered {
		t.Errorf("Third item should be filtered (doesn't match /^tech/)")
	}
	if !result[3].IsFiltered {
		t.Errorf("Fourth item should be filtered (doesn't match /^tech/)")
	}
}

func TestFilterer_RegexPattern_Exclude(t *testing.T) {
	items := []types.Item{
		{Title: "Мобильная разработка за неделю"},
		{Title: "Новости кибербезопасности за неделю"},
		{Title: "ТОП-5 ИБ-событий недели"},
		{Title: "Регулярная статья о программировании"},
	}

	filters := []types.Filter{
		{
			Field:    "title",
			Excludes: []string{"/за неделю|недели/"}, // Regex: matches weekly digests
		},
	}

	result := Filter(items, filters)

	// First three items should be filtered (match regex)
	if !result[0].IsFiltered {
		t.Errorf("First item should be filtered (matches weekly pattern)")
	}
	if !result[1].IsFiltered {
		t.Errorf("Second item should be filtered (matches weekly pattern)")
	}
	if !result[2].IsFiltered {
		t.Errorf("Third item should be filtered (matches weekly pattern)")
	}

	// Fourth item should pass (doesn't match)
	if result[3].IsFiltered {
		t.Errorf("Fourth item should not be filtered (doesn't match weekly pattern)")
	}
}

func TestFilterer_RegexPattern_CaseInsensitive(t *testing.T) {
	items := []types.Item{
		{Title: "BREAKING NEWS"},
		{Title: "breaking news"},
		{Title: "Breaking News"},
		{Title: "other content"},
	}

	filters := []types.Filter{
		{
			Field:    "title",
			Includes: []string{"/^breaking/"}, // Should match all cases
		},
	}

	result := Filter(items, filters)

	// First three should pass (case insensitive)
	for i := 0; i < 3; i++ {
		if result[i].IsFiltered {
			t.Errorf("Item %d should not be filtered (case insensitive regex)", i)
		}
	}

	// Fourth should be filtered
	if !result[3].IsFiltered {
		t.Errorf("Fourth item should be filtered")
	}
}

func TestFilterer_RegexPattern_Categories(t *testing.T) {
	items := []types.Item{
		{Title: "Article 1", Categories: []string{"Angular", "React", "Vue"}},
		{Title: "Article 2", Categories: []string{"Python", "Go", "Rust"}},
		{Title: "Article 3", Categories: []string{"JavaScript", "TypeScript"}},
	}

	filters := []types.Filter{
		{
			Field:    "categories",
			Excludes: []string{"/^(angular|vue|react)$/"}, // Exclude frontend frameworks
		},
	}

	result := Filter(items, filters)

	// First item should be filtered (has Angular, Vue, React)
	if !result[0].IsFiltered {
		t.Errorf("First item should be filtered (matches frontend frameworks)")
	}

	// Second item should pass (no frontend frameworks)
	if result[1].IsFiltered {
		t.Errorf("Second item should not be filtered (no frontend frameworks)")
	}

	// Third item should pass (JavaScript/TypeScript don't match exact pattern)
	if result[2].IsFiltered {
		t.Errorf("Third item should not be filtered (JS/TS don't match exact pattern)")
	}
}

func TestFilterer_RegexPattern_MixedWithSubstring(t *testing.T) {
	// Test that regex and substring patterns work together
	items := []types.Item{
		{Title: "Tech Weekly Digest"},
		{Title: "Tech Article"},
		{Title: "Weekly Sports Update"},
		{Title: "Random Article"},
	}

	filters := []types.Filter{
		{
			Field:    "title",
			Includes: []string{"tech"},           // Substring
			Excludes: []string{"/weekly|digest/"}, // Regex
		},
	}

	result := Filter(items, filters)

	// First item: has "tech" but also has "weekly" -> filtered
	if !result[0].IsFiltered {
		t.Errorf("First item should be filtered (excluded by regex)")
	}

	// Second item: has "tech" and no excludes -> pass
	if result[1].IsFiltered {
		t.Errorf("Second item should not be filtered")
	}

	// Third item: no "tech" -> filtered
	if !result[2].IsFiltered {
		t.Errorf("Third item should be filtered (no 'tech')")
	}

	// Fourth item: no "tech" -> filtered
	if !result[3].IsFiltered {
		t.Errorf("Fourth item should be filtered (no 'tech')")
	}
}

func TestFilterer_RegexPattern_Invalid(t *testing.T) {
	// Test that invalid regex falls back to literal matching
	items := []types.Item{
		{Title: "Test /[invalid/ pattern"},
		{Title: "Another article"},
	}

	filters := []types.Filter{
		{
			Field:    "title",
			Includes: []string{"/[invalid/"}, // Invalid regex (unclosed bracket)
		},
	}

	result := Filter(items, filters)

	// First item should pass (contains literal "/[invalid/" substring)
	// Invalid regex falls back to substring match of "/[invalid/"
	if result[0].IsFiltered {
		t.Errorf("First item should not be filtered (contains literal '/[invalid/')")
	}

	// Second item should be filtered (doesn't contain literal "/[invalid/")
	if !result[1].IsFiltered {
		t.Errorf("Second item should be filtered (no literal match)")
	}
}

func TestFilterer_RegexCache(t *testing.T) {
	// Test that cache works correctly
	items := []types.Item{
		{Title: "Tech Article 1"},
		{Title: "Tech Article 2"},
	}

	filters := []types.Filter{
		{
			Field:    "title",
			Includes: []string{"/^tech/"}, // Same pattern used multiple times
		},
	}

	// Process items multiple times
	for i := 0; i < 3; i++ {
		result := Filter(items, filters)
		for j, item := range result {
			if item.IsFiltered {
				t.Errorf("Iteration %d, Item %d should not be filtered", i, j)
			}
		}
	}
}

func TestFilterer_ClearRegexCache(t *testing.T) {
	items := []types.Item{
		{Title: "Tech Article"},
	}

	filters := []types.Filter{
		{
			Field:    "title",
			Includes: []string{"/^tech/"},
		},
	}

	// First processing - populates cache
	result1 := Filter(items, filters)
	if result1[0].IsFiltered {
		t.Errorf("Item should not be filtered before cache clear")
	}

	// Clear cache
	ClearRegexCache()

	// Second processing - should work the same after cache clear
	result2 := Filter(items, filters)
	if result2[0].IsFiltered {
		t.Errorf("Item should not be filtered after cache clear")
	}
}

func TestIsRegexPattern(t *testing.T) {
	tests := []struct {
		pattern  string
		expected bool
	}{
		{"/^test$/", true},
		{"/pattern/", true},
		{"/.*/", true},
		{"pattern", false},
		{"/only-start", false},
		{"only-end/", false},
		{"/", false},
		{"//", true}, // Edge case: empty regex
		{"", false},
	}

	for _, test := range tests {
		result := isRegexPattern(test.pattern)
		if result != test.expected {
			t.Errorf("isRegexPattern(%q): expected %v, got %v", test.pattern, test.expected, result)
		}
	}
}

func TestExtractRegexPattern(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/^test$/", "(?i)^test$"},
		{"/pattern/", "(?i)pattern"},
		{"/(?i)already-case-insensitive/", "(?i)already-case-insensitive"},
		{"/(?m)multiline/", "(?i)(?m)multiline"}, // Adds (?i) even with other flags
		{"//", "(?i)"},
	}

	for _, test := range tests {
		result := extractRegexPattern(test.input)
		if result != test.expected {
			t.Errorf("extractRegexPattern(%q): expected %q, got %q", test.input, test.expected, result)
		}
	}
}
