package parser

import (
	"strings"
	"testing"
)

func TestParseRSS2(t *testing.T) {
	rssData := `<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <link>https://example.com</link>
    <description>Test Description</description>
    <language>en-us</language>
    <lastBuildDate>Mon, 03 Jul 2023 12:00:00 GMT</lastBuildDate>
    <image>
      <url>https://example.com/icon.png</url>
      <title>Test Feed</title>
      <link>https://example.com</link>
    </image>
    <item>
      <title>Test Item 1</title>
      <link>https://example.com/item1</link>
      <description>Test Item 1 Description</description>
      <guid>item-1</guid>
      <pubDate>Mon, 03 Jul 2023 10:00:00 GMT</pubDate>
      <author>test@example.com (Test Author)</author>
      <category>Technology</category>
      <category>Programming</category>
    </item>
    <item>
      <title>Test Item 2</title>
      <link>https://example.com/item2</link>
      <description>Test Item 2 Description</description>
      <guid>item-2</guid>
      <pubDate>Mon, 03 Jul 2023 11:00:00 GMT</pubDate>
    </item>
  </channel>
</rss>`

	parser := NewParser()
	metadata, items, err := parser.Parse([]byte(rssData))
	if err != nil {
		t.Fatal(err)
	}

	// Test metadata
	if metadata.Title != "Test Feed" {
		t.Errorf("Expected title 'Test Feed', got '%s'", metadata.Title)
	}
	if metadata.Link != "https://example.com" {
		t.Errorf("Expected link 'https://example.com', got '%s'", metadata.Link)
	}
	if metadata.Description != "Test Description" {
		t.Errorf("Expected description 'Test Description', got '%s'", metadata.Description)
	}
	if metadata.Language != "en-us" {
		t.Errorf("Expected language 'en-us', got '%s'", metadata.Language)
	}
	if metadata.IconURL != "https://example.com/icon.png" {
		t.Errorf("Expected icon URL 'https://example.com/icon.png', got '%s'", metadata.IconURL)
	}

	// Test items
	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	}

	// Test first item
	item1 := items[0]
	if item1.Title != "Test Item 1" {
		t.Errorf("Expected first item title 'Test Item 1', got '%s'", item1.Title)
	}
	if item1.Link != "https://example.com/item1" {
		t.Errorf("Expected first item link 'https://example.com/item1', got '%s'", item1.Link)
	}
	if item1.GUID != "item-1" {
		t.Errorf("Expected first item GUID 'item-1', got '%s'", item1.GUID)
	}
	if len(item1.Categories) != 2 {
		t.Errorf("Expected 2 categories for first item, got %d", len(item1.Categories))
	}
	if item1.ContentHash == "" {
		t.Error("Expected content hash to be generated")
	}

	// Test second item
	item2 := items[1]
	if item2.Title != "Test Item 2" {
		t.Errorf("Expected second item title 'Test Item 2', got '%s'", item2.Title)
	}
	if item2.GUID != "item-2" {
		t.Errorf("Expected second item GUID 'item-2', got '%s'", item2.GUID)
	}

	// Test that content hashes are different
	if item1.ContentHash == item2.ContentHash {
		t.Error("Expected different content hashes for different items")
	}
}

func TestParseAtom(t *testing.T) {
	atomData := `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Test Atom Feed</title>
  <link href="https://example.com"/>
  <id>https://example.com/feed</id>
  <updated>2023-07-03T12:00:00Z</updated>
  <subtitle>Test Atom Description</subtitle>
  
  <entry>
    <title>Atom Entry 1</title>
    <link href="https://example.com/atom1"/>
    <id>atom-1</id>
    <updated>2023-07-03T10:00:00Z</updated>
    <published>2023-07-03T10:00:00Z</published>
    <summary>Atom Entry 1 Summary</summary>
    <content type="html">Atom Entry 1 Content</content>
    <author>
      <name>Atom Author</name>
      <email>atom@example.com</email>
    </author>
    <category term="atom"/>
  </entry>
</feed>`

	parser := NewParser()
	metadata, items, err := parser.Parse([]byte(atomData))
	if err != nil {
		t.Fatal(err)
	}

	// Test metadata
	if metadata.Title != "Test Atom Feed" {
		t.Errorf("Expected title 'Test Atom Feed', got '%s'", metadata.Title)
	}
	if metadata.Link != "https://example.com" {
		t.Errorf("Expected link 'https://example.com', got '%s'", metadata.Link)
	}

	// Test items
	if len(items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(items))
	}

	item := items[0]
	if item.Title != "Atom Entry 1" {
		t.Errorf("Expected item title 'Atom Entry 1', got '%s'", item.Title)
	}
	if item.Content != "Atom Entry 1 Content" {
		t.Errorf("Expected item content 'Atom Entry 1 Content', got '%s'", item.Content)
	}
	if item.AuthorName != "Atom Author" {
		t.Errorf("Expected author name 'Atom Author', got '%s'", item.AuthorName)
	}
	if item.AuthorEmail != "atom@example.com" {
		t.Errorf("Expected author email 'atom@example.com', got '%s'", item.AuthorEmail)
	}
}

func TestParseInvalidFeed(t *testing.T) {
	invalidData := `<html><body>This is not a feed</body></html>`

	parser := NewParser()
	_, _, err := parser.Parse([]byte(invalidData))
	if err == nil {
		t.Error("Expected error for invalid feed data")
	}
}

func TestContentHashGeneration(t *testing.T) {
	parser := NewParser()

	item1 := NormalizedItem{
		Title:       "Test Title",
		Link:        "https://example.com/1",
		Description: "Test Description",
	}

	item2 := NormalizedItem{
		Title:       "Test Title",
		Link:        "https://example.com/1",
		Description: "Test Description",
	}

	item3 := NormalizedItem{
		Title:       "Different Title",
		Link:        "https://example.com/1",
		Description: "Test Description",
	}

	// Test updated deduplication logic: same title+link with different description should have same hash
	item4 := NormalizedItem{
		Title:       "Test Title",
		Link:        "https://example.com/1",
		Description: "Updated Description - Article was modified",
	}

	hash1 := parser.generateContentHash(item1)
	hash2 := parser.generateContentHash(item2)
	hash3 := parser.generateContentHash(item3)
	hash4 := parser.generateContentHash(item4)

	// Same content should produce same hash
	if hash1 != hash2 {
		t.Error("Expected same hash for identical content")
	}

	// Different title should produce different hash
	if hash1 == hash3 {
		t.Error("Expected different hash for different title")
	}

	// Same title+link with different description should produce same hash (updated deduplication logic)
	if hash1 != hash4 {
		t.Error("Expected same hash for same title+link with different description")
	}

	// Hash should be non-empty
	if hash1 == "" {
		t.Error("Expected non-empty hash")
	}
}

func TestCoalesceFunction(t *testing.T) {
	parser := NewParser()

	// Test with first value non-empty
	result := parser.coalesce("first", "second", "third")
	if result != "first" {
		t.Errorf("Expected 'first', got '%s'", result)
	}

	// Test with first value empty
	result = parser.coalesce("", "second", "third")
	if result != "second" {
		t.Errorf("Expected 'second', got '%s'", result)
	}

	// Test with all values empty
	result = parser.coalesce("", "", "")
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}

	// Test with no values
	result = parser.coalesce()
	if result != "" {
		t.Errorf("Expected empty string for no values, got '%s'", result)
	}
}

func TestItemToMap(t *testing.T) {
	parser := NewParser()

	// Create a simple test feed to get a gofeed.Item
	rssData := `<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <title>Test</title>
    <item>
      <title>Test Item</title>
      <link>https://example.com/item</link>
      <description>Test Description</description>
      <guid>test-guid</guid>
      <category>test-category</category>
    </item>
  </channel>
</rss>`

	feed, err := parser.gofeedParser.Parse(strings.NewReader(rssData))
	if err != nil {
		t.Fatal(err)
	}

	if len(feed.Items) == 0 {
		t.Fatal("Expected at least one item")
	}

	itemMap := parser.itemToMap(feed.Items[0])

	// Check basic fields
	if itemMap["title"] != "Test Item" {
		t.Errorf("Expected title 'Test Item', got '%v'", itemMap["title"])
	}
	if itemMap["link"] != "https://example.com/item" {
		t.Errorf("Expected link 'https://example.com/item', got '%v'", itemMap["link"])
	}
	if itemMap["description"] != "Test Description" {
		t.Errorf("Expected description 'Test Description', got '%v'", itemMap["description"])
	}
	if itemMap["guid"] != "test-guid" {
		t.Errorf("Expected guid 'test-guid', got '%v'", itemMap["guid"])
	}

	// Check categories
	categories, ok := itemMap["categories"].([]string)
	if !ok {
		t.Error("Expected categories to be []string")
	} else if len(categories) != 1 || categories[0] != "test-category" {
		t.Errorf("Expected categories ['test-category'], got %v", categories)
	}
}