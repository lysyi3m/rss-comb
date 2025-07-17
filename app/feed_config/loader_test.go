package feed_config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadValidConfig(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Create test YAML file
	content := `
feed:
  id: "test"
  url: "https://example.com/feed.xml"
  title: "Test Feed"

settings:
  enabled: true
  refresh_interval: 1800
  max_items: 25
  timeout: 15

filters:
  - field: "title"
    includes:
      - "technology"
    excludes:
      - "spam"
`

	err := os.WriteFile(filepath.Join(tempDir, "test.yml"), []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Load feedConfig
	loader := NewLoader(tempDir)
	feedConfigs, err := loader.LoadAll()
	if err != nil {
		t.Fatal(err)
	}

	if len(feedConfigs) != 1 {
		t.Errorf("Expected 1 feedConfig, got %d", len(feedConfigs))
	}

	// Get the feedConfig
	var feedConfig *FeedConfig
	for _, tempConfig := range feedConfigs {
		feedConfig = tempConfig
		break
	}

	// Validate loaded values
	if feedConfig.Feed.ID != "test" {
		t.Errorf("Expected ID 'test', got '%s'", feedConfig.Feed.ID)
	}
	if feedConfig.Feed.URL != "https://example.com/feed.xml" {
		t.Errorf("Expected URL 'https://example.com/feed.xml', got '%s'", feedConfig.Feed.URL)
	}
	if feedConfig.Feed.Title != "Test Feed" {
		t.Errorf("Expected title 'Test Feed', got '%s'", feedConfig.Feed.Title)
	}
	if feedConfig.Settings.GetRefreshInterval() != 1800*time.Second {
		t.Errorf("Expected refresh interval 1800s, got %v", feedConfig.Settings.GetRefreshInterval())
	}
	if feedConfig.Settings.MaxItems != 25 {
		t.Errorf("Expected max items 25, got %d", feedConfig.Settings.MaxItems)
	}
	if len(feedConfig.Filters) != 1 {
		t.Errorf("Expected 1 filter, got %d", len(feedConfig.Filters))
	}
}

func TestLoadConfigWithDefaults(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Create minimal test YAML file
	content := `
feed:
  id: "test-defaults"
  url: "https://example.com/feed.xml"
  title: "Test Feed"

settings:
  enabled: true
`

	err := os.WriteFile(filepath.Join(tempDir, "test.yml"), []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Load feedConfig
	loader := NewLoader(tempDir)
	feedConfigs, err := loader.LoadAll()
	if err != nil {
		t.Fatal(err)
	}

	// Get the feedConfig
	var feedConfig *FeedConfig
	for _, tempConfig := range feedConfigs {
		feedConfig = tempConfig
		break
	}

	// Validate default values
	if feedConfig.Settings.GetRefreshInterval() != 3600*time.Second {
		t.Errorf("Expected default refresh interval 3600s, got %v", feedConfig.Settings.GetRefreshInterval())
	}
	if feedConfig.Settings.MaxItems != 100 {
		t.Errorf("Expected default max items 100, got %d", feedConfig.Settings.MaxItems)
	}
}

func TestInvalidConfig(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Create invalid YAML file (missing feed ID and URL)
	content := `
feed:
  title: "Test Feed"

settings:
  enabled: true
`

	err := os.WriteFile(filepath.Join(tempDir, "invalid.yml"), []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Load feedConfig
	loader := NewLoader(tempDir)
	_, err = loader.LoadAll()
	if err == nil {
		t.Error("Expected error for invalid feedConfig")
	}
}

func TestEmptyDirectory(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Load from empty directory
	loader := NewLoader(tempDir)
	feedConfigs, err := loader.LoadAll()
	if err != nil {
		t.Fatal(err)
	}

	if len(feedConfigs) != 0 {
		t.Errorf("Expected 0 feedConfigs from empty directory, got %d", len(feedConfigs))
	}
}

