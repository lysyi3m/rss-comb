package feed

import (
	"testing"

	"github.com/lysyi3m/rss-comb/app/types"
)

func TestParse_InvalidFeed(t *testing.T) {
	_, _, err := Parse([]byte("invalid xml"))

	if err == nil {
		t.Error("Expected error for invalid XML")
	}
}

func TestContentHashGeneration(t *testing.T) {
	item1 := types.Item{
		Title: "Test Title",
		Link:  "https://example.com/item1",
	}

	item2 := types.Item{
		Title: "Test Title",
		Link:  "https://example.com/item1",
	}

	item3 := types.Item{
		Title: "Different Title",
		Link:  "https://example.com/item1",
	}

	hash1 := generateContentHash(item1)
	hash2 := generateContentHash(item2)
	hash3 := generateContentHash(item3)

	if hash1 != hash2 {
		t.Error("Expected same hash for identical items")
	}

	if hash1 == hash3 {
		t.Error("Expected different hash for different items")
	}
}

func TestNormalizeURL(t *testing.T) {
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
			result := normalizeURL(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeURL(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParse_YouTubeAtomFeed(t *testing.T) {
	youtubeAtom := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns:yt="http://www.youtube.com/xml/schemas/2015"
      xmlns:media="http://search.yahoo.com/mrss/"
      xmlns="http://www.w3.org/2005/Atom">
  <title>Test Playlist</title>
  <author><name>Test Channel</name></author>
  <entry>
    <id>yt:video:dQw4w9WgXcQ</id>
    <yt:videoId>dQw4w9WgXcQ</yt:videoId>
    <title>Test Video Title</title>
    <link rel="alternate" href="https://www.youtube.com/watch?v=dQw4w9WgXcQ"/>
    <author><name>Test Channel</name></author>
    <published>2025-01-15T10:00:00+00:00</published>
    <media:group>
      <media:title>Test Video Title</media:title>
      <media:description>This is the video description with details.</media:description>
      <media:thumbnail url="https://i4.ytimg.com/vi/dQw4w9WgXcQ/hqdefault.jpg" width="480" height="360"/>
    </media:group>
  </entry>
  <entry>
    <id>yt:video:abc123XYZ_-</id>
    <yt:videoId>abc123XYZ_-</yt:videoId>
    <title>Second Video</title>
    <link rel="alternate" href="https://www.youtube.com/watch?v=abc123XYZ_-"/>
    <author><name>Test Channel</name></author>
    <published>2025-01-14T10:00:00+00:00</published>
    <media:group>
      <media:title>Second Video</media:title>
      <media:description>Another description here.</media:description>
      <media:thumbnail url="https://i4.ytimg.com/vi/abc123XYZ_-/hqdefault.jpg" width="480" height="360"/>
    </media:group>
  </entry>
</feed>`

	metadata, items, err := Parse([]byte(youtubeAtom))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Feed-level metadata inferred from items
	if metadata.Title != "Test Playlist" {
		t.Errorf("Expected feed title 'Test Playlist', got %q", metadata.Title)
	}
	if metadata.ImageURL != "https://i4.ytimg.com/vi/dQw4w9WgXcQ/hqdefault.jpg" {
		t.Errorf("Expected feed image from first item thumbnail, got %q", metadata.ImageURL)
	}
	if metadata.ITunesAuthor != "Test Channel" {
		t.Errorf("Expected iTunes author 'Test Channel', got %q", metadata.ITunesAuthor)
	}

	if len(items) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(items))
	}

	// First item
	item := items[0]
	if item.GUID != "yt:video:dQw4w9WgXcQ" {
		t.Errorf("Expected GUID 'yt:video:dQw4w9WgXcQ', got %q", item.GUID)
	}
	if item.Description != "This is the video description with details." {
		t.Errorf("Expected description from media:description, got %q", item.Description)
	}
	if item.ITunesImage != "https://i4.ytimg.com/vi/dQw4w9WgXcQ/hqdefault.jpg" {
		t.Errorf("Expected iTunes image from media:thumbnail, got %q", item.ITunesImage)
	}

	// Second item
	if items[1].Description != "Another description here." {
		t.Errorf("Expected second item description, got %q", items[1].Description)
	}
}

func TestParse_YouTubeAtomFeed_StandardDescriptionTakesPrecedence(t *testing.T) {
	atomWithSummary := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns:media="http://search.yahoo.com/mrss/"
      xmlns="http://www.w3.org/2005/Atom">
  <title>Test Feed</title>
  <entry>
    <id>test:1</id>
    <title>Test</title>
    <link rel="alternate" href="https://example.com/1"/>
    <summary>Standard Atom summary</summary>
    <published>2025-01-15T10:00:00+00:00</published>
    <media:group>
      <media:description>Media description (should be ignored)</media:description>
    </media:group>
  </entry>
</feed>`

	_, items, err := Parse([]byte(atomWithSummary))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if items[0].Description != "Standard Atom summary" {
		t.Errorf("Standard description should take precedence, got %q", items[0].Description)
	}
}

func TestParse_FeedImageNotOverriddenWhenPresent(t *testing.T) {
	atomWithImage := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns:media="http://search.yahoo.com/mrss/"
      xmlns="http://www.w3.org/2005/Atom">
  <title>Test Feed</title>
  <logo>https://example.com/feed-logo.png</logo>
  <entry>
    <id>test:1</id>
    <title>Test</title>
    <link rel="alternate" href="https://example.com/1"/>
    <published>2025-01-15T10:00:00+00:00</published>
    <media:group>
      <media:thumbnail url="https://example.com/thumb.jpg" width="480" height="360"/>
    </media:group>
  </entry>
</feed>`

	metadata, _, err := Parse([]byte(atomWithImage))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if metadata.ImageURL != "https://example.com/feed-logo.png" {
		t.Errorf("Feed image should not be overridden when present, got %q", metadata.ImageURL)
	}
}

func TestNormalizeItem_WithTrackingParams(t *testing.T) {
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

	_, items, err := Parse([]byte(rssData))
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
