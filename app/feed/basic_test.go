package feed

import (
	"testing"

	"github.com/lysyi3m/rss-comb/app/types"
)

func TestBasicParse_InvalidFeed(t *testing.T) {
	bt := basicType{}
	_, _, err := bt.Parse([]byte("invalid xml"))

	if err == nil {
		t.Error("Expected error for invalid XML")
	}
}

func TestBasicParse_RSSFeed(t *testing.T) {
	rssData := `<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <link>https://example.com</link>
    <description>Test Description</description>
    <item>
      <title>Test Item</title>
      <link>https://example.com/article</link>
      <description>Item Description</description>
      <pubDate>Mon, 03 Jul 2023 10:00:00 GMT</pubDate>
    </item>
  </channel>
</rss>`

	bt := basicType{}
	metadata, items, err := bt.Parse([]byte(rssData))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if metadata.Title != "Test Feed" {
		t.Errorf("Expected title 'Test Feed', got %q", metadata.Title)
	}

	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(items))
	}

	if items[0].Title != "Test Item" {
		t.Errorf("Expected item title 'Test Item', got %q", items[0].Title)
	}
}

func TestBasicParse_IgnoresITunesData(t *testing.T) {
	podcastRSS := `<?xml version="1.0"?>
<rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd">
  <channel>
    <title>Podcast Feed</title>
    <link>https://example.com</link>
    <description>A podcast</description>
    <itunes:author>Podcast Author</itunes:author>
    <item>
      <title>Episode 1</title>
      <link>https://example.com/ep1</link>
      <description>Episode description</description>
      <pubDate>Mon, 03 Jul 2023 10:00:00 GMT</pubDate>
      <itunes:duration>3600</itunes:duration>
      <itunes:episode>1</itunes:episode>
    </item>
  </channel>
</rss>`

	bt := basicType{}
	metadata, items, err := bt.Parse([]byte(podcastRSS))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if metadata.ITunesAuthor != "" {
		t.Errorf("Basic parser should ignore iTunes author, got %q", metadata.ITunesAuthor)
	}

	if items[0].ITunesDuration != 0 {
		t.Errorf("Basic parser should ignore iTunes duration, got %d", items[0].ITunesDuration)
	}

	if items[0].ITunesEpisode != 0 {
		t.Errorf("Basic parser should ignore iTunes episode, got %d", items[0].ITunesEpisode)
	}
}

func TestBasicParse_TrackingParamsRemoved(t *testing.T) {
	rssData := `<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <link>https://example.com</link>
    <description>Test Description</description>
    <item>
      <title>Test Item</title>
      <link>https://example.com/article?utm_source=twitter&amp;utm_medium=social&amp;fbclid=IwAR123456789</link>
      <description>Test Description</description>
      <pubDate>Mon, 03 Jul 2023 10:00:00 GMT</pubDate>
    </item>
  </channel>
</rss>`

	bt := basicType{}
	_, items, err := bt.Parse([]byte(rssData))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expectedLink := "https://example.com/article"
	if items[0].Link != expectedLink {
		t.Errorf("Expected normalized link %q, got %q", expectedLink, items[0].Link)
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
