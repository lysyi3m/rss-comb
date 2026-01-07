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
