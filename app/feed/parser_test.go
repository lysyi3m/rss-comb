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
	metadata, items, err := parser.Run([]byte(rssData))

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
	metadata, items, err := parser.Run([]byte(atomData))

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
	_, _, err := parser.Run([]byte("invalid xml"))

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

func TestParseRSSWithEnclosure(t *testing.T) {
	parser := NewParser()
	
	// Sample RSS 2.0 feed with enclosure
	rssData := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
<channel>
	<title>Test Podcast</title>
	<link>https://example.com</link>
	<description>A test podcast feed</description>
	<item>
		<title>Episode 1</title>
		<link>https://example.com/episode1</link>
		<description>First episode</description>
		<guid>episode1</guid>
		<pubDate>Wed, 01 Feb 2023 10:00:00 +0000</pubDate>
		<enclosure url="https://example.com/audio/episode1.mp3" length="24576000" type="audio/mpeg" />
	</item>
</channel>
</rss>`

	metadata, items, err := parser.Run([]byte(rssData))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if metadata.Title != "Test Podcast" {
		t.Errorf("Expected title 'Test Podcast', got: %s", metadata.Title)
	}

	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got: %d", len(items))
	}

	item := items[0]
	
	// Verify basic item fields
	if item.Title != "Episode 1" {
		t.Errorf("Expected title 'Episode 1', got: %s", item.Title)
	}
	
	if item.GUID != "episode1" {
		t.Errorf("Expected GUID 'episode1', got: %s", item.GUID)
	}

	// Verify enclosure fields
	if item.EnclosureURL != "https://example.com/audio/episode1.mp3" {
		t.Errorf("Expected enclosure URL 'https://example.com/audio/episode1.mp3', got: %s", item.EnclosureURL)
	}
	
	if item.EnclosureLength != 24576000 {
		t.Errorf("Expected enclosure length 24576000, got: %d", item.EnclosureLength)
	}
	
	if item.EnclosureType != "audio/mpeg" {
		t.Errorf("Expected enclosure type 'audio/mpeg', got: %s", item.EnclosureType)
	}
}

func TestParseRSSWithoutEnclosure(t *testing.T) {
	parser := NewParser()
	
	// Sample RSS 2.0 feed without enclosure
	rssData := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
<channel>
	<title>Test Blog</title>
	<link>https://example.com</link>
	<description>A test blog feed</description>
	<item>
		<title>Blog Post 1</title>
		<link>https://example.com/post1</link>
		<description>First blog post</description>
		<guid>post1</guid>
		<pubDate>Wed, 01 Feb 2023 10:00:00 +0000</pubDate>
	</item>
</channel>
</rss>`

	_, items, err := parser.Run([]byte(rssData))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got: %d", len(items))
	}

	item := items[0]
	
	// Verify enclosure fields are empty/zero
	if item.EnclosureURL != "" {
		t.Errorf("Expected empty enclosure URL, got: %s", item.EnclosureURL)
	}
	
	if item.EnclosureLength != 0 {
		t.Errorf("Expected enclosure length 0, got: %d", item.EnclosureLength)
	}
	
	if item.EnclosureType != "" {
		t.Errorf("Expected empty enclosure type, got: %s", item.EnclosureType)
	}
}

func TestParseRSSWithMultipleEnclosures(t *testing.T) {
	parser := NewParser()
	
	// Sample RSS 2.0 feed with multiple enclosures (should only pick first one)
	rssData := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
<channel>
	<title>Test Feed</title>
	<link>https://example.com</link>
	<description>A test feed</description>
	<item>
		<title>Multi-enclosure Item</title>
		<link>https://example.com/item1</link>
		<description>Item with multiple enclosures</description>
		<guid>item1</guid>
		<pubDate>Wed, 01 Feb 2023 10:00:00 +0000</pubDate>
		<enclosure url="https://example.com/file1.mp3" length="1000000" type="audio/mpeg" />
		<enclosure url="https://example.com/file2.pdf" length="2000000" type="application/pdf" />
	</item>
</channel>
</rss>`

	_, items, err := parser.Run([]byte(rssData))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got: %d", len(items))
	}

	item := items[0]
	
	// Verify only first enclosure is used (as per RSS 2.0 spec and our design)
	if item.EnclosureURL != "https://example.com/file1.mp3" {
		t.Errorf("Expected first enclosure URL 'https://example.com/file1.mp3', got: %s", item.EnclosureURL)
	}
	
	if item.EnclosureLength != 1000000 {
		t.Errorf("Expected first enclosure length 1000000, got: %d", item.EnclosureLength)
	}
	
	if item.EnclosureType != "audio/mpeg" {
		t.Errorf("Expected first enclosure type 'audio/mpeg', got: %s", item.EnclosureType)
	}
}

func TestParser_normalizeURL(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "URL with UTM parameters",
			input:    "https://example.com/article?utm_source=twitter&utm_medium=social&utm_campaign=test",
			expected: "https://example.com/article",
		},
		{
			name:     "URL with Facebook tracking",
			input:    "https://example.com/page?fbclid=IwAR123456789&other=keep",
			expected: "https://example.com/page?other=keep",
		},
		{
			name:     "URL with Google click ID",
			input:    "https://example.com/landing?gclid=abc123&page=home",
			expected: "https://example.com/landing?page=home",
		},
		{
			name:     "URL with multiple tracking parameters",
			input:    "https://example.com/content?utm_source=email&utm_medium=newsletter&fbclid=xyz789&ref=homepage&title=article",
			expected: "https://example.com/content?title=article",
		},
		{
			name:     "URL without tracking parameters",
			input:    "https://example.com/clean?page=1&sort=date",
			expected: "https://example.com/clean?page=1&sort=date",
		},
		{
			name:     "URL without query parameters",
			input:    "https://example.com/simple",
			expected: "https://example.com/simple",
		},
		{
			name:     "Empty URL",
			input:    "",
			expected: "",
		},
		{
			name:     "Invalid URL",
			input:    "not-a-valid-url",
			expected: "not-a-valid-url",
		},
		{
			name:     "URL with only tracking parameters",
			input:    "https://example.com/page?utm_source=test&fbclid=123",
			expected: "https://example.com/page",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.normalizeURL(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeURL(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParser_normalizeItem_WithTrackingParams(t *testing.T) {
	parser := NewParser()
	
	// Test RSS data with tracking parameters
	rssData := `<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <link>https://example.com</link>
    <description>Test Description</description>
    <item>
      <title>Test Item</title>
      <link>https://example.com/article?utm_source=twitter&utm_medium=social&fbclid=IwAR123456789</link>
      <description>Test Description</description>
      <pubDate>Mon, 03 Jul 2023 10:00:00 GMT</pubDate>
    </item>
  </channel>
</rss>`

	_, items, err := parser.Run([]byte(rssData))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got: %d", len(items))
	}

	item := items[0]
	expectedLink := "https://example.com/article"
	
	if item.Link != expectedLink {
		t.Errorf("Expected normalized link %q, got %q", expectedLink, item.Link)
	}
	
	if item.GUID != expectedLink {
		t.Errorf("Expected GUID to be normalized link %q, got %q", expectedLink, item.GUID)
	}
}

func TestParseRSSWithHTMLEntities(t *testing.T) {
	rssData := `<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <title>Test Feed &amp; Special Characters</title>
    <link>https://example.com</link>
    <description>A test feed with &quot;HTML entities&quot; &amp; special chars</description>
    <item>
      <title>Company didn&#8217;t fix users&#8217; security issues</title>
      <link>https://example.com/item1</link>
      <description>This article discusses &lt;privacy&gt; issues with &quot;smart&quot; devices &amp; IoT security.</description>
      <content:encoded><![CDATA[Full article content with &amp; entities and &#8220;curly quotes&#8221;]]></content:encoded>
      <guid>item-1</guid>
      <pubDate>Mon, 03 Jul 2023 10:00:00 GMT</pubDate>
    </item>
  </channel>
</rss>`

	parser := NewParser()
	metadata, items, err := parser.Run([]byte(rssData))

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Test metadata HTML entity decoding
	expectedTitle := "Test Feed & Special Characters"
	if metadata.Title != expectedTitle {
		t.Errorf("Expected metadata title %q, got %q", expectedTitle, metadata.Title)
	}

	expectedDesc := `A test feed with "HTML entities" & special chars`
	if metadata.Description != expectedDesc {
		t.Errorf("Expected metadata description %q, got %q", expectedDesc, metadata.Description)
	}

	// Test item HTML entity decoding
	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got: %d", len(items))
	}

	item := items[0]
	expectedItemTitle := "Company didn\u2019t fix users\u2019 security issues"
	if item.Title != expectedItemTitle {
		t.Errorf("Expected item title %q, got %q", expectedItemTitle, item.Title)
	}

	expectedItemDesc := `This article discusses <privacy> issues with "smart" devices & IoT security.`
	if item.Description != expectedItemDesc {
		t.Errorf("Expected item description %q, got %q", expectedItemDesc, item.Description)
	}

	expectedContent := "Full article content with &amp; entities and &#8220;curly quotes&#8221;"
	if item.Content != expectedContent {
		t.Errorf("Expected item content %q, got %q", expectedContent, item.Content)
	}
}

