package config

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
  deduplication: true
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

	// Load configuration
	loader := NewLoader(tempDir)
	configs, err := loader.LoadAll()
	if err != nil {
		t.Fatal(err)
	}

	if len(configs) != 1 {
		t.Errorf("Expected 1 config, got %d", len(configs))
	}

	// Get the config
	var config *FeedConfig
	for _, cfg := range configs {
		config = cfg
		break
	}

	// Validate loaded values
	if config.Feed.ID != "test" {
		t.Errorf("Expected ID 'test', got '%s'", config.Feed.ID)
	}
	if config.Feed.URL != "https://example.com/feed.xml" {
		t.Errorf("Expected URL 'https://example.com/feed.xml', got '%s'", config.Feed.URL)
	}
	if config.Feed.Title != "Test Feed" {
		t.Errorf("Expected title 'Test Feed', got '%s'", config.Feed.Title)
	}
	if config.Settings.GetRefreshInterval() != 1800*time.Second {
		t.Errorf("Expected refresh interval 1800s, got %v", config.Settings.GetRefreshInterval())
	}
	if config.Settings.MaxItems != 25 {
		t.Errorf("Expected max items 25, got %d", config.Settings.MaxItems)
	}
	if len(config.Filters) != 1 {
		t.Errorf("Expected 1 filter, got %d", len(config.Filters))
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

	// Load configuration
	loader := NewLoader(tempDir)
	configs, err := loader.LoadAll()
	if err != nil {
		t.Fatal(err)
	}

	// Get the config
	var config *FeedConfig
	for _, cfg := range configs {
		config = cfg
		break
	}

	// Validate default values
	if config.Settings.GetRefreshInterval() != 3600*time.Second {
		t.Errorf("Expected default refresh interval 3600s, got %v", config.Settings.GetRefreshInterval())
	}
	if config.Settings.MaxItems != 100 {
		t.Errorf("Expected default max items 100, got %d", config.Settings.MaxItems)
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

	// Load configuration
	loader := NewLoader(tempDir)
	_, err = loader.LoadAll()
	if err == nil {
		t.Error("Expected error for invalid configuration")
	}
}

func TestEmptyDirectory(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Load from empty directory
	loader := NewLoader(tempDir)
	configs, err := loader.LoadAll()
	if err != nil {
		t.Fatal(err)
	}

	if len(configs) != 0 {
		t.Errorf("Expected 0 configs from empty directory, got %d", len(configs))
	}
}

func TestGetUserAgent(t *testing.T) {
	// Test default user agent
	defaultUserAgent := GetUserAgent()
	if defaultUserAgent != "RSS Comb/1.0" {
		t.Errorf("Expected default user agent 'RSS Comb/1.0', got '%s'", defaultUserAgent)
	}

	// Test custom user agent from environment
	os.Setenv("USER_AGENT", "Custom User Agent/2.0")
	defer os.Unsetenv("USER_AGENT")
	
	customUserAgent := GetUserAgent()
	if customUserAgent != "Custom User Agent/2.0" {
		t.Errorf("Expected custom user agent 'Custom User Agent/2.0', got '%s'", customUserAgent)
	}
}