package feed

import (
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
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Test metadata
	if metadata.Title != "Test Feed" {
		t.Errorf("Expected title 'Test Feed', got: %s", metadata.Title)
	}
	if metadata.Link != "https://example.com" {
		t.Errorf("Expected link 'https://example.com', got: %s", metadata.Link)
	}
	if metadata.Description != "Test Description" {
		t.Errorf("Expected description 'Test Description', got: %s", metadata.Description)
	}
	if metadata.Language != "en-us" {
		t.Errorf("Expected language 'en-us', got: %s", metadata.Language)
	}
	if metadata.ImageURL != "https://example.com/icon.png" {
		t.Errorf("Expected image URL 'https://example.com/icon.png', got: %s", metadata.ImageURL)
	}

	// Test items
	if len(items) != 2 {
		t.Fatalf("Expected 2 items, got: %d", len(items))
	}

	item1 := items[0]
	if item1.Title != "Test Item 1" {
		t.Errorf("Expected title 'Test Item 1', got: %s", item1.Title)
	}
	if item1.Link != "https://example.com/item1" {
		t.Errorf("Expected link 'https://example.com/item1', got: %s", item1.Link)
	}
	if item1.GUID != "item-1" {
		t.Errorf("Expected GUID 'item-1', got: %s", item1.GUID)
	}
	if len(item1.Categories) != 2 {
		t.Errorf("Expected 2 categories, got: %d", len(item1.Categories))
	}
	if item1.ContentHash == "" {
		t.Error("Expected content hash to be generated")
	}
}

func TestParseAtom(t *testing.T) {
	atomData := `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Test Atom Feed</title>
  <link href="https://example.com"/>
  <updated>2023-07-03T12:00:00Z</updated>
  <author>
    <name>Test Author</name>
  </author>
  <id>urn:uuid:1234567890</id>
  <entry>
    <title>Test Entry</title>
    <link href="https://example.com/entry1"/>
    <id>urn:uuid:entry-1</id>
    <updated>2023-07-03T10:00:00Z</updated>
    <content type="html">Test content</content>
  </entry>
</feed>`

	parser := NewParser()
	metadata, items, err := parser.Parse([]byte(atomData))

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if metadata.Title != "Test Atom Feed" {
		t.Errorf("Expected title 'Test Atom Feed', got: %s", metadata.Title)
	}

	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got: %d", len(items))
	}

	item := items[0]
	if item.Title != "Test Entry" {
		t.Errorf("Expected title 'Test Entry', got: %s", item.Title)
	}
	if item.Link != "https://example.com/entry1" {
		t.Errorf("Expected link 'https://example.com/entry1', got: %s", item.Link)
	}
	if item.ContentHash == "" {
		t.Error("Expected content hash to be generated")
	}
}

func TestParseInvalidFeed(t *testing.T) {
	parser := NewParser()
	_, _, err := parser.Parse([]byte("invalid xml"))

	if err == nil {
		t.Error("Expected error for invalid XML")
	}
}

func TestContentHashGeneration(t *testing.T) {
	parser := NewParser()
	
	item1 := Item{
		Title: "Test Title",
		Link:  "https://example.com/item1",
	}
	
	item2 := Item{
		Title: "Test Title",
		Link:  "https://example.com/item1",
	}
	
	item3 := Item{
		Title: "Different Title",
		Link:  "https://example.com/item1",
	}
	
	hash1 := parser.generateContentHash(item1)
	hash2 := parser.generateContentHash(item2)
	hash3 := parser.generateContentHash(item3)
	
	if hash1 != hash2 {
		t.Error("Expected same hash for identical items")
	}
	
	if hash1 == hash3 {
		t.Error("Expected different hash for different items")
	}
}

func TestCoalesceFunction(t *testing.T) {
	parser := NewParser()
	
	result := parser.coalesce("", "second", "third")
	if result != "second" {
		t.Errorf("Expected 'second', got: %s", result)
	}
	
	result = parser.coalesce("first", "second", "third")
	if result != "first" {
		t.Errorf("Expected 'first', got: %s", result)
	}
	
	result = parser.coalesce("", "", "")
	if result != "" {
		t.Errorf("Expected empty string, got: %s", result)
	}
}