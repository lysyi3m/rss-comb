package feed

import (
	"strings"
	"testing"
	"time"

	"github.com/lysyi3m/rss-comb/app/database"
)

func TestGenerateRSS(t *testing.T) {
	generator := NewGenerator("8080")
	
	// Create sample feed
	feed := database.Feed{
		ID:      "test-feed-uuid",
		FeedID:  "test-feed",
		Title:   "Test Feed",
		Link:    "https://example.com",
		FeedURL: "https://example.com/feed.xml",
	}
	
	// Create sample items
	publishedTime := time.Date(2023, 7, 3, 10, 0, 0, 0, time.UTC)
	updatedTime := time.Date(2023, 7, 3, 11, 0, 0, 0, time.UTC)
	
	items := []database.Item{
		{
			ID:            "item-1-uuid",
			FeedID:        "test-feed-uuid",
			GUID:          "item-1",
			Title:         "Test Item 1",
			Link:          "https://example.com/item1",
			Description:   "Test Item 1 Description",
			Content:       "Test Item 1 Content",
			PublishedDate: &publishedTime,
			UpdatedDate:   &updatedTime,
			AuthorName:    "Test Author",
			AuthorEmail:   "test@example.com",
			Categories:    []string{"Technology", "Programming"},
		},
		{
			ID:            "item-2-uuid",
			FeedID:        "test-feed-uuid",
			GUID:          "item-2",
			Title:         "Test Item 2",
			Link:          "https://example.com/item2",
			Description:   "Test Item 2 Description",
			PublishedDate: &publishedTime,
		},
	}
	
	rss, err := generator.Generate(feed, items)
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
	
	if !strings.Contains(rss, `<atom:link href="http://localhost:8080/feeds/test-feed" rel="self" type="application/xml" />`) {
		t.Error("RSS should contain atom:link self reference")
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

func TestGenerateEmpty(t *testing.T) {
	generator := NewGenerator("8080")
	
	rss := generator.GenerateEmpty("Empty Feed", "https://example.com/feed.xml")
	
	// Verify basic structure
	if !strings.Contains(rss, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("Empty RSS should contain XML declaration")
	}
	
	if !strings.Contains(rss, `<rss version="2.0"`) {
		t.Error("Empty RSS should contain RSS 2.0 declaration")
	}
	
	if !strings.Contains(rss, "<title>Empty Feed</title>") {
		t.Error("Empty RSS should contain provided title")
	}
	
	if !strings.Contains(rss, "<link>https://example.com/feed.xml</link>") {
		t.Error("Empty RSS should contain provided URL as link")
	}
	
	if !strings.Contains(rss, "<description>Feed is being processed. Please check back later.</description>") {
		t.Error("Empty RSS should contain processing message")
	}
	
	// Verify it doesn't contain any items
	if strings.Contains(rss, "<item>") {
		t.Error("Empty RSS should not contain any items")
	}
	
	if !strings.Contains(rss, "</channel>") {
		t.Error("Empty RSS should contain closing channel tag")
	}
	
	if !strings.Contains(rss, "</rss>") {
		t.Error("Empty RSS should contain closing rss tag")
	}
}

func TestGenerateWithMinimalData(t *testing.T) {
	generator := NewGenerator("9000")
	
	// Create minimal feed data
	feed := database.Feed{
		ID:      "minimal-feed-uuid",
		FeedID:  "minimal-feed",
		Title:   "Minimal Feed",
		Link:    "",
		FeedURL: "https://example.com/minimal.xml",
	}
	
	// Create minimal item
	items := []database.Item{
		{
			ID:            "minimal-item-uuid",
			FeedID:        "minimal-feed-uuid",
			GUID:          "minimal-item",
			Title:         "Minimal Item",
			Link:          "",
			Description:   "",
			Content:       "",
			PublishedDate: nil,
			UpdatedDate:   nil,
			AuthorName:    "",
			AuthorEmail:   "",
			Categories:    []string{},
		},
	}
	
	rss, err := generator.Generate(feed, items)
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
	
	// Verify proper atom:link with custom port
	if !strings.Contains(rss, `<atom:link href="http://localhost:9000/feeds/minimal-feed" rel="self" type="application/xml" />`) {
		t.Error("Minimal RSS should contain atom:link with custom port")
	}
}

func TestGenerateWithSpecialCharacters(t *testing.T) {
	generator := NewGenerator("8080")
	
	// Create feed with special characters
	feed := database.Feed{
		ID:      "special-feed-uuid",
		FeedID:  "special-feed",
		Title:   "Feed with <special> & \"characters\"",
		Link:    "https://example.com",
		FeedURL: "https://example.com/feed.xml",
	}
	
	// Create item with special characters
	items := []database.Item{
		{
			ID:            "special-item-uuid",
			FeedID:        "special-feed-uuid",
			GUID:          "special-item",
			Title:         "Item with <tags> & \"quotes\"",
			Link:          "https://example.com/item",
			Description:   "Description with <em>emphasis</em> & \"quotes\"",
			Content:       "Content with <strong>bold</strong> & special chars: <>&\"'",
			AuthorName:    "Author with <brackets>",
			AuthorEmail:   "test@example.com",
			Categories:    []string{"Category with <brackets>", "Category & Ampersand"},
		},
	}
	
	rss, err := generator.Generate(feed, items)
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
	generator := NewGenerator("8080")
	
	// Create feed with no items
	feed := database.Feed{
		ID:      "empty-feed-uuid",
		FeedID:  "empty-feed",
		Title:   "Empty Feed",
		Link:    "https://example.com",
		FeedURL: "https://example.com/feed.xml",
	}
	
	items := []database.Item{}
	
	rss, err := generator.Generate(feed, items)
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
	generator := NewGenerator("8080")
	
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
