package feed

import "testing"

func TestPodcastParse_ITunesMetadata(t *testing.T) {
	podcastRSS := `<?xml version="1.0"?>
<rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd">
  <channel>
    <title>My Podcast</title>
    <link>https://example.com</link>
    <description>A great podcast</description>
    <itunes:author>John Doe</itunes:author>
    <itunes:image href="https://example.com/cover.jpg"/>
    <itunes:explicit>false</itunes:explicit>
    <itunes:owner>
      <itunes:name>John Doe</itunes:name>
      <itunes:email>john@example.com</itunes:email>
    </itunes:owner>
    <item>
      <title>Episode 1</title>
      <link>https://example.com/ep1</link>
      <description>First episode</description>
      <pubDate>Mon, 03 Jul 2023 10:00:00 GMT</pubDate>
      <enclosure url="https://example.com/ep1.mp3" length="12345678" type="audio/mpeg"/>
      <itunes:duration>3600</itunes:duration>
      <itunes:episode>1</itunes:episode>
      <itunes:season>2</itunes:season>
      <itunes:episodeType>full</itunes:episodeType>
    </item>
  </channel>
</rss>`

	pt := podcastType{}
	metadata, items, err := pt.Parse([]byte(podcastRSS))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if metadata.ITunesAuthor != "John Doe" {
		t.Errorf("Expected iTunes author 'John Doe', got %q", metadata.ITunesAuthor)
	}
	if metadata.ITunesImage != "https://example.com/cover.jpg" {
		t.Errorf("Expected iTunes image, got %q", metadata.ITunesImage)
	}
	if metadata.ITunesExplicit != "false" {
		t.Errorf("Expected iTunes explicit 'false', got %q", metadata.ITunesExplicit)
	}
	if metadata.ITunesOwnerName != "John Doe" {
		t.Errorf("Expected iTunes owner name 'John Doe', got %q", metadata.ITunesOwnerName)
	}
	if metadata.ITunesOwnerEmail != "john@example.com" {
		t.Errorf("Expected iTunes owner email, got %q", metadata.ITunesOwnerEmail)
	}

	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(items))
	}

	item := items[0]
	if item.ITunesDuration != 3600 {
		t.Errorf("Expected duration 3600, got %d", item.ITunesDuration)
	}
	if item.ITunesEpisode != 1 {
		t.Errorf("Expected episode 1, got %d", item.ITunesEpisode)
	}
	if item.ITunesSeason != 2 {
		t.Errorf("Expected season 2, got %d", item.ITunesSeason)
	}
	if item.ITunesEpisodeType != "full" {
		t.Errorf("Expected episode type 'full', got %q", item.ITunesEpisodeType)
	}
	if item.EnclosureURL != "https://example.com/ep1.mp3" {
		t.Errorf("Expected enclosure URL, got %q", item.EnclosureURL)
	}
	if item.EnclosureType != "audio/mpeg" {
		t.Errorf("Expected enclosure type 'audio/mpeg', got %q", item.EnclosureType)
	}
}

func TestPodcastParse_InvalidFeed(t *testing.T) {
	pt := podcastType{}
	_, _, err := pt.Parse([]byte("invalid xml"))

	if err == nil {
		t.Error("Expected error for invalid XML")
	}
}
