package feed

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lysyi3m/rss-comb/app/cfg"
	"github.com/lysyi3m/rss-comb/app/database"
)

func setupTestConfig() {
	// Clear os.Args to prevent config parsing from failing
	oldArgs := os.Args
	os.Args = []string{"test"}
	defer func() { os.Args = oldArgs }()

	// Set default environment variables if not set
	if os.Getenv("PORT") == "" {
		os.Setenv("PORT", "8080")
	}

	cfg.Load()
}

func TestGenerateRSS(t *testing.T) {
	setupTestConfig()
	generator := NewGenerator()

	// Create sample feed with published date
	feedPublishedTime := time.Date(2023, 7, 1, 12, 0, 0, 0, time.UTC)
	feed := database.Feed{
		ID:              "test-feed-uuid",
		Name:            "test-feed",
		Title:           "Test Feed",
		Link:            "https://example.com",
		FeedURL:         "https://example.com/feed.xml",
		FeedPublishedAt: &feedPublishedTime,
	}

	// Create sample items
	publishedTime := time.Date(2023, 7, 3, 10, 0, 0, 0, time.UTC)
	updatedTime := time.Date(2023, 7, 3, 11, 0, 0, 0, time.UTC)

	items := []database.Item{
		{
			ID:          "item-1-uuid",
			FeedID:      "test-feed-uuid",
			GUID:        "item-1",
			Title:       "Test Item 1",
			Link:        "https://example.com/item1",
			Description: "Test Item 1 Description",
			Content:     "Test Item 1 Content",
			PublishedAt: publishedTime,
			UpdatedAt:   &updatedTime,
			Authors:     []string{"test@example.com (Test Author)"},
			Categories:  []string{"Technology", "Programming"},
		},
		{
			ID:          "item-2-uuid",
			FeedID:      "test-feed-uuid",
			GUID:        "item-2",
			Title:       "Test Item 2",
			Link:        "https://example.com/item2",
			Description: "Test Item 2 Description",
			PublishedAt: publishedTime,
		},
	}

	rss, err := generator.Run(feed, items)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify RSS structure
	if !strings.Contains(rss, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("RSS should contain XML declaration")
	}

	if !strings.Contains(rss, `<rss version="2.0"`) {
		t.Error("RSS should contain RSS 2.0 declaration")
	}

	if !strings.Contains(rss, `xmlns:content="http://purl.org/rss/1.0/modules/content/"`) {
		t.Error("RSS should contain content namespace")
	}

	if !strings.Contains(rss, `xmlns:atom="http://www.w3.org/2005/Atom"`) {
		t.Error("RSS should contain atom namespace")
	}

	// Verify channel metadata
	if !strings.Contains(rss, "<title>Test Feed</title>") {
		t.Error("RSS should contain feed title")
	}

	if !strings.Contains(rss, "<link>https://example.com</link>") {
		t.Error("RSS should contain feed link")
	}

	if !strings.Contains(rss, "Processed feed from https://example.com/feed.xml") {
		t.Error("RSS should contain processed feed description")
	}

	if !strings.Contains(rss, `<atom:link href="http://localhost:8080/feeds/test-feed" rel="self" type="application/rss+xml" />`) {
		t.Error("RSS should contain atom:link self reference")
	}

	// Verify feed pubDate is included
	if !strings.Contains(rss, "<pubDate>Sat, 01 Jul 2023 12:00:00 +0000</pubDate>") {
		t.Error("RSS should contain feed pubDate when FeedPublishedAt is set")
	}

	// Verify items
	if !strings.Contains(rss, "<title>Test Item 1</title>") {
		t.Error("RSS should contain first item title")
	}

	if !strings.Contains(rss, "<link>https://example.com/item1</link>") {
		t.Error("RSS should contain first item link")
	}

	if !strings.Contains(rss, `<guid isPermaLink="false">item-1</guid>`) {
		t.Error("RSS should contain first item GUID")
	}

	if !strings.Contains(rss, "<description>Test Item 1 Description</description>") {
		t.Error("RSS should contain first item description")
	}

	if !strings.Contains(rss, "<content:encoded><![CDATA[Test Item 1 Content]]></content:encoded>") {
		t.Error("RSS should contain first item content")
	}

	if !strings.Contains(rss, "<author>test@example.com (Test Author)</author>") {
		t.Error("RSS should contain first item author")
	}

	if !strings.Contains(rss, "<category>Technology</category>") {
		t.Error("RSS should contain first item category")
	}

	if !strings.Contains(rss, "<category>Programming</category>") {
		t.Error("RSS should contain first item second category")
	}

	if !strings.Contains(rss, "<pubDate>Mon, 03 Jul 2023 10:00:00 +0000</pubDate>") {
		t.Error("RSS should contain first item published date")
	}

	// Verify second item
	if !strings.Contains(rss, "<title>Test Item 2</title>") {
		t.Error("RSS should contain second item title")
	}

	if !strings.Contains(rss, `<guid isPermaLink="false">item-2</guid>`) {
		t.Error("RSS should contain second item GUID")
	}

	// Verify proper XML structure
	if !strings.Contains(rss, "</channel>") {
		t.Error("RSS should contain closing channel tag")
	}

	if !strings.Contains(rss, "</rss>") {
		t.Error("RSS should contain closing rss tag")
	}
}

func TestGenerateWithMinimalData(t *testing.T) {
	setupTestConfig()
	generator := NewGenerator()

	// Create minimal feed data
	feed := database.Feed{
		ID:      "minimal-feed-uuid",
		Name:    "minimal-feed",
		Title:   "Minimal Feed",
		Link:    "",
		FeedURL: "https://example.com/minimal.xml",
	}

	// Create minimal item
	items := []database.Item{
		{
			ID:          "minimal-item-uuid",
			FeedID:      "minimal-feed-uuid",
			GUID:        "minimal-item",
			Title:       "Minimal Item",
			Link:        "",
			Description: "",
			Content:     "",
			PublishedAt: time.Time{}, // Use zero time instead of nil
			UpdatedAt:   nil,
			Authors:     []string{},
			Categories:  []string{},
		},
	}

	rss, err := generator.Run(feed, items)
	if err != nil {
		t.Fatalf("Expected no error with minimal data, got: %v", err)
	}

	// Verify it still generates valid RSS
	if !strings.Contains(rss, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("Minimal RSS should contain XML declaration")
	}

	if !strings.Contains(rss, "<title>Minimal Feed</title>") {
		t.Error("Minimal RSS should contain feed title")
	}

	if !strings.Contains(rss, "<title>Minimal Item</title>") {
		t.Error("Minimal RSS should contain item title")
	}

	if !strings.Contains(rss, `<guid isPermaLink="false">minimal-item</guid>`) {
		t.Error("Minimal RSS should contain item GUID")
	}

	// Verify proper atom:link with default port
	if !strings.Contains(rss, `<atom:link href="http://localhost:8080/feeds/minimal-feed" rel="self" type="application/rss+xml" />`) {
		t.Error("Minimal RSS should contain atom:link with default port")
	}

	// Verify no feed-level pubDate element when FeedPublishedAt is nil  
	// Items will still have pubDate (using zero time), but the channel should not
	if strings.Contains(rss, "<channel>") {
		// Check that there's no pubDate immediately under channel
		channelStart := strings.Index(rss, "<channel>")
		firstItemStart := strings.Index(rss, "<item>")
		if firstItemStart == -1 {
			firstItemStart = len(rss)
		}
		channelSection := rss[channelStart:firstItemStart]
		if strings.Contains(channelSection, "<pubDate>") {
			t.Error("Minimal RSS should not contain feed-level pubDate when FeedPublishedAt is nil")
		}
	}
}

func TestGenerateWithSpecialCharacters(t *testing.T) {
	setupTestConfig()
	generator := NewGenerator()

	// Create feed with special characters
	feed := database.Feed{
		ID:      "special-feed-uuid",
		Name:    "special-feed",
		Title:   "Feed with <special> & \"characters\"",
		Link:    "https://example.com",
		FeedURL: "https://example.com/feed.xml",
	}

	// Create item with special characters
	items := []database.Item{
		{
			ID:          "special-item-uuid",
			FeedID:      "special-feed-uuid",
			GUID:        "special-item",
			Title:       "Item with <tags> & \"quotes\"",
			Link:        "https://example.com/item",
			Description: "Description with <em>emphasis</em> & \"quotes\"",
			Content:     "Content with <strong>bold</strong> & special chars: <>&\"'",
			Authors:     []string{"test@example.com (Author with <brackets>)"},
			Categories:  []string{"Category with <brackets>", "Category & Ampersand"},
		},
	}

	rss, err := generator.Run(feed, items)
	if err != nil {
		t.Fatalf("Expected no error with special characters, got: %v", err)
	}

	// Verify special characters are properly escaped in XML
	if !strings.Contains(rss, "Feed with &lt;special&gt; &amp; &#34;characters&#34;") {
		t.Error("Feed title should have escaped special characters")
	}

	if !strings.Contains(rss, "Item with &lt;tags&gt; &amp; &#34;quotes&#34;") {
		t.Error("Item title should have escaped special characters")
	}

	if !strings.Contains(rss, "Description with &lt;em&gt;emphasis&lt;/em&gt; &amp; &#34;quotes&#34;") {
		t.Error("Item description should have escaped special characters")
	}

	// Content should be in CDATA, so it shouldn't be escaped
	if !strings.Contains(rss, "<content:encoded><![CDATA[Content with <strong>bold</strong> & special chars: <>&\"']]></content:encoded>") {
		t.Error("Item content should be in CDATA without escaping")
	}

	if !strings.Contains(rss, "Author with &lt;brackets&gt;") {
		t.Error("Author name should have escaped special characters")
	}

	if !strings.Contains(rss, "Category with &lt;brackets&gt;") {
		t.Error("Category should have escaped special characters")
	}

	if !strings.Contains(rss, "Category &amp; Ampersand") {
		t.Error("Category with ampersand should be escaped")
	}
}

func TestGenerateWithEmptyItems(t *testing.T) {
	setupTestConfig()
	generator := NewGenerator()

	// Create feed with no items
	feed := database.Feed{
		ID:      "empty-feed-uuid",
		Name:    "empty-feed",
		Title:   "Empty Feed",
		Link:    "https://example.com",
		FeedURL: "https://example.com/feed.xml",
	}

	items := []database.Item{}

	rss, err := generator.Run(feed, items)
	if err != nil {
		t.Fatalf("Expected no error with empty items, got: %v", err)
	}

	// Verify it still generates valid RSS
	if !strings.Contains(rss, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("Empty items RSS should contain XML declaration")
	}

	if !strings.Contains(rss, "<title>Empty Feed</title>") {
		t.Error("Empty items RSS should contain feed title")
	}

	// Verify it doesn't contain any items
	if strings.Contains(rss, "<item>") {
		t.Error("Empty items RSS should not contain any items")
	}

	if !strings.Contains(rss, "</channel>") {
		t.Error("Empty items RSS should contain closing channel tag")
	}

	if !strings.Contains(rss, "</rss>") {
		t.Error("Empty items RSS should contain closing rss tag")
	}
}

func TestIsURLMethod(t *testing.T) {
	setupTestConfig()
	generator := NewGenerator()

	tests := []struct {
		input    string
		expected bool
	}{
		{"", false},
		{"http://example.com", true},
		{"https://example.com", true},
		{"ftp://example.com", false},
		{"not-a-url", false},
		{"http://", false},
		{"https://", false},
		{"mailto:test@example.com", false},
	}

	for _, test := range tests {
		result := generator.isURL(test.input)
		if result != test.expected {
			t.Errorf("For input '%s', expected %v, got %v", test.input, test.expected, result)
		}
	}
}

func TestGenerateWithDefaultConfig(t *testing.T) {
	setupTestConfig()
	generator := NewGenerator()

	// Create feed to test default behavior (no BaseUrl set)
	feed := database.Feed{
		ID:      "default-config-test-uuid",
		Name:    "default-config-test",
		Title:   "Default Config Test Feed",
		Link:    "https://example.com",
		FeedURL: "https://example.com/feed.xml",
	}

	items := []database.Item{}

	rss, err := generator.Run(feed, items)
	if err != nil {
		t.Fatalf("Expected no error with default config, got: %v", err)
	}

	// With no BaseUrl set, should use localhost:8080 format
	if !strings.Contains(rss, `<atom:link href="http://localhost:8080/feeds/default-config-test" rel="self" type="application/rss+xml" />`) {
		t.Error("RSS should contain localhost atom:link when BaseUrl is not set in config")
	}
}

func TestLastBuildDateWithItems(t *testing.T) {
	setupTestConfig()
	generator := NewGenerator()

	// Create feed
	feed := database.Feed{
		ID:      "lastbuild-test-uuid",
		Name:    "lastbuild-test",
		Title:   "Last Build Date Test Feed",
		Link:    "https://example.com",
		FeedURL: "https://example.com/feed.xml",
	}

	// Create items with different published dates
	olderTime := time.Date(2023, 7, 1, 10, 0, 0, 0, time.UTC)
	newerTime := time.Date(2023, 7, 5, 15, 30, 0, 0, time.UTC)
	
	// Items should be sorted by published_at DESC (as they come from database)
	items := []database.Item{
		{
			ID:          "item-2-uuid", 
			FeedID:      "lastbuild-test-uuid",
			GUID:        "item-2",
			Title:       "Newer Item",
			PublishedAt: newerTime,
		},
		{
			ID:          "item-1-uuid",
			FeedID:      "lastbuild-test-uuid",
			GUID:        "item-1",
			Title:       "Older Item",
			PublishedAt: olderTime,
		},
	}

	rss, err := generator.Run(feed, items)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// lastBuildDate should use the most recent item's timestamp (newerTime)
	if !strings.Contains(rss, "<lastBuildDate>Wed, 05 Jul 2023 15:30:00 +0000</lastBuildDate>") {
		t.Error("RSS should use most recent item's PublishedAt for lastBuildDate")
	}
}

func TestLastBuildDateWithoutItems(t *testing.T) {
	setupTestConfig()
	generator := NewGenerator()

	// Create feed with no items
	feed := database.Feed{
		ID:      "empty-lastbuild-test-uuid",
		Name:    "empty-lastbuild-test", 
		Title:   "Empty Last Build Date Test Feed",
		Link:    "https://example.com",
		FeedURL: "https://example.com/feed.xml",
	}

	items := []database.Item{}

	rss, err := generator.Run(feed, items)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// lastBuildDate should exist (fallback to current time) but we can't test exact timestamp
	if !strings.Contains(rss, "<lastBuildDate>") {
		t.Error("RSS should contain lastBuildDate even when no items exist")
	}
}

func TestLastBuildDateWithCreatedAtFallback(t *testing.T) {
	setupTestConfig()
	generator := NewGenerator()

	// Create feed
	feed := database.Feed{
		ID:      "createdat-test-uuid",
		Name:    "createdat-test",
		Title:   "Created At Fallback Test Feed",
		Link:    "https://example.com",
		FeedURL: "https://example.com/feed.xml",
	}

	// Create item with nil published_at but valid created_at
	createdTime := time.Date(2023, 7, 10, 12, 0, 0, 0, time.UTC)
	
	items := []database.Item{
		{
			ID:          "item-createdat-uuid",
			FeedID:      "createdat-test-uuid",
			GUID:        "item-createdat",
			Title:       "Item with CreatedAt only",
			PublishedAt: time.Time{}, // Use zero time instead of nil // No published date
			CreatedAt:   createdTime,
		},
	}

	rss, err := generator.Run(feed, items)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// lastBuildDate should use created_at as fallback when published_at is nil
	if !strings.Contains(rss, "<lastBuildDate>Mon, 10 Jul 2023 12:00:00 +0000</lastBuildDate>") {
		t.Error("RSS should use item's CreatedAt for lastBuildDate when PublishedAt is nil")
	}
}

func TestGenerateWithEnclosure(t *testing.T) {
	setupTestConfig()
	generator := NewGenerator()

	// Create feed
	feed := database.Feed{
		ID:      "enclosure-test-uuid",
		Name:    "enclosure-test",
		Title:   "Enclosure Test Feed",
		Link:    "https://example.com",
		FeedURL: "https://example.com/feed.xml",
	}

	publishedTime := time.Date(2023, 7, 1, 10, 0, 0, 0, time.UTC)

	// Create item with enclosure (typical podcast episode)
	items := []database.Item{
		{
			ID:              "enclosure-item-uuid",
			FeedID:          "enclosure-test-uuid",
			GUID:            "enclosure-item",
			Title:           "Episode 1: Introduction",
			Link:            "https://example.com/episode1",
			Description:     "First episode of our podcast",
			Content:         "Welcome to our first episode!",
			PublishedAt:     publishedTime,
			EnclosureURL:    "https://example.com/audio/episode1.mp3",
			EnclosureLength: 24576000, // 24MB in bytes
			EnclosureType:   "audio/mpeg",
		},
	}

	rss, err := generator.Run(feed, items)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify enclosure element is present with correct attributes
	expectedEnclosure := `<enclosure url="https://example.com/audio/episode1.mp3" length="24576000" type="audio/mpeg" />`
	if !strings.Contains(rss, expectedEnclosure) {
		t.Errorf("RSS should contain enclosure element: %s", expectedEnclosure)
	}

	// Verify other item elements are still present
	if !strings.Contains(rss, "<title>Episode 1: Introduction</title>") {
		t.Error("RSS should contain item title")
	}

	if !strings.Contains(rss, "<description>First episode of our podcast</description>") {
		t.Error("RSS should contain item description")
	}
}

func TestGenerateWithoutEnclosure(t *testing.T) {
	setupTestConfig()
	generator := NewGenerator()

	// Create feed
	feed := database.Feed{
		ID:      "no-enclosure-test-uuid",
		Name:    "no-enclosure-test",
		Title:   "No Enclosure Test Feed",
		Link:    "https://example.com",
		FeedURL: "https://example.com/feed.xml",
	}

	publishedTime := time.Date(2023, 7, 1, 10, 0, 0, 0, time.UTC)

	// Create item without enclosure (typical blog post)
	items := []database.Item{
		{
			ID:              "no-enclosure-item-uuid",
			FeedID:          "no-enclosure-test-uuid",
			GUID:            "no-enclosure-item",
			Title:           "Regular Blog Post",
			Link:            "https://example.com/post1",
			Description:     "A regular blog post without media",
			PublishedAt:     publishedTime,
			EnclosureURL:    "", // No enclosure
			EnclosureLength: 0,
			EnclosureType:   "",
		},
	}

	rss, err := generator.Run(feed, items)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify no enclosure element is present
	if strings.Contains(rss, "<enclosure") {
		t.Error("RSS should not contain enclosure element when no enclosure data is present")
	}

	// Verify other item elements are still present
	if !strings.Contains(rss, "<title>Regular Blog Post</title>") {
		t.Error("RSS should contain item title")
	}
}

func TestGenerateWithPartialEnclosureData(t *testing.T) {
	setupTestConfig()
	generator := NewGenerator()

	// Create feed
	feed := database.Feed{
		ID:      "partial-enclosure-test-uuid",
		Name:    "partial-enclosure-test",
		Title:   "Partial Enclosure Test Feed",
		Link:    "https://example.com",
		FeedURL: "https://example.com/feed.xml",
	}

	publishedTime := time.Date(2023, 7, 1, 10, 0, 0, 0, time.UTC)

	// Create item with incomplete enclosure data (missing type - should not generate enclosure)
	items := []database.Item{
		{
			ID:              "partial-enclosure-item-uuid",
			FeedID:          "partial-enclosure-test-uuid",
			GUID:            "partial-enclosure-item",
			Title:           "Incomplete Enclosure Item",
			Link:            "https://example.com/incomplete",
			Description:     "Item with incomplete enclosure data",
			PublishedAt:     publishedTime,
			EnclosureURL:    "https://example.com/file.mp3",
			EnclosureLength: 12345,
			EnclosureType:   "", // Missing type - should not generate enclosure
		},
	}

	rss, err := generator.Run(feed, items)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify no enclosure element is present (URL and type are required)
	if strings.Contains(rss, "<enclosure") {
		t.Error("RSS should not contain enclosure element when required attributes are missing")
	}
}
