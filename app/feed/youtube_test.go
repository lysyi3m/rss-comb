package feed

import "testing"

func TestYouTubeParse_AtomFeed(t *testing.T) {
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

	yt := youtubeType{}
	metadata, items, err := yt.Parse([]byte(youtubeAtom))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

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

	if items[1].Description != "Another description here." {
		t.Errorf("Expected second item description, got %q", items[1].Description)
	}
}

func TestYouTubeParse_StandardDescriptionTakesPrecedence(t *testing.T) {
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

	yt := youtubeType{}
	_, items, err := yt.Parse([]byte(atomWithSummary))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if items[0].Description != "Standard Atom summary" {
		t.Errorf("Standard description should take precedence, got %q", items[0].Description)
	}
}

func TestYouTubeParse_FeedImageNotOverriddenWhenPresent(t *testing.T) {
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

	yt := youtubeType{}
	metadata, _, err := yt.Parse([]byte(atomWithImage))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if metadata.ImageURL != "https://example.com/feed-logo.png" {
		t.Errorf("Feed image should not be overridden when present, got %q", metadata.ImageURL)
	}
}
