package feed

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestConfigCacheLoadValidConfig(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Create test YAML file
	content := `
url: "https://example.com/feed.xml"

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
	configCache := NewConfigCache(tempDir)
	err = configCache.Run()
	if err != nil {
		t.Fatal(err)
	}

	if configCache.GetConfigCount() != 1 {
		t.Errorf("Expected 1 feedConfig, got %d", configCache.GetConfigCount())
	}

	// Get the feedConfig by name
	feedConfig, err := configCache.GetConfig("test")
	if err != nil {
		t.Fatal(err)
	}

	// Validate loaded values
	if feedConfig.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", feedConfig.Name)
	}
	if feedConfig.URL != "https://example.com/feed.xml" {
		t.Errorf("Expected URL 'https://example.com/feed.xml', got '%s'", feedConfig.URL)
	}
	if time.Duration(feedConfig.Settings.RefreshInterval)*time.Second != 1800*time.Second {
		t.Errorf("Expected refresh interval 1800s, got %v", time.Duration(feedConfig.Settings.RefreshInterval)*time.Second)
	}
	if feedConfig.Settings.MaxItems != 25 {
		t.Errorf("Expected max items 25, got %d", feedConfig.Settings.MaxItems)
	}
	if len(feedConfig.Filters) != 1 {
		t.Errorf("Expected 1 filter, got %d", len(feedConfig.Filters))
	}
}

func TestConfigCacheLoadConfigWithDefaults(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Create minimal test YAML file
	content := `
url: "https://example.com/feed.xml"

settings:
  enabled: true
`

	err := os.WriteFile(filepath.Join(tempDir, "test.yml"), []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Load feedConfig
	configCache := NewConfigCache(tempDir)
	err = configCache.Run()
	if err != nil {
		t.Fatal(err)
	}

	// Get the feedConfig by name
	feedConfig, err := configCache.GetConfig("test")
	if err != nil {
		t.Fatal(err)
	}

	// Validate default values
	if time.Duration(feedConfig.Settings.RefreshInterval)*time.Second != 3600*time.Second {
		t.Errorf("Expected default refresh interval 3600s, got %v", time.Duration(feedConfig.Settings.RefreshInterval)*time.Second)
	}
	if feedConfig.Settings.MaxItems != 100 {
		t.Errorf("Expected default max items 100, got %d", feedConfig.Settings.MaxItems)
	}
}

func TestConfigCacheInvalidConfig(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Create invalid YAML file (missing feed URL)
	content := `
settings:
  enabled: true
`

	err := os.WriteFile(filepath.Join(tempDir, "invalid.yml"), []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Load feedConfig
	configCache := NewConfigCache(tempDir)
	err = configCache.Run()
	if err == nil {
		t.Error("Expected error for invalid feedConfig")
	}
}

func TestConfigCacheEmptyDirectory(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Load from empty directory
	configCache := NewConfigCache(tempDir)
	err := configCache.Run()
	if err != nil {
		t.Fatal(err)
	}

	if configCache.GetConfigCount() != 0 {
		t.Errorf("Expected 0 feedConfigs from empty directory, got %d", configCache.GetConfigCount())
	}
}

func TestConfigCacheReloadConfig(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Create initial test YAML file
	initialContent := `
url: "https://example.com/feed.xml"

settings:
  enabled: true
`

	configFile := filepath.Join(tempDir, "test.yml")
	err := os.WriteFile(configFile, []byte(initialContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Load initial config
	configCache := NewConfigCache(tempDir)
	err = configCache.Run()
	if err != nil {
		t.Fatal(err)
	}

	// Verify initial config can be loaded
	_, err = configCache.GetConfig("test")
	if err != nil {
		t.Fatal(err)
	}
	// Title no longer part of config - comes from feed source

	// Update the file on disk with new content
	updatedContent := `
url: "https://example.com/new-feed.xml"

settings:
  enabled: true
  max_items: 50
`

	err = os.WriteFile(configFile, []byte(updatedContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Reload config from disk
	reloadedConfig, err := configCache.LoadConfig("test")
	if err != nil {
		t.Fatal(err)
	}

	if reloadedConfig.URL != "https://example.com/new-feed.xml" {
		t.Errorf("Expected updated URL 'https://example.com/new-feed.xml', got '%s'", reloadedConfig.URL)
	}
	// Title no longer part of config - comes from feed source
	if reloadedConfig.Settings.MaxItems != 50 {
		t.Errorf("Expected updated max_items 50, got %d", reloadedConfig.Settings.MaxItems)
	}

	// Test loading non-existent config
	_, err = configCache.LoadConfig("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent config")
	}

	// Test loading invalid config
	invalidContent := `invalid yaml content`
	err = os.WriteFile(configFile, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = configCache.LoadConfig("test")
	if err == nil {
		t.Error("Expected error for invalid config file")
	}
}

func TestConfigCacheGetConfigs(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Create multiple test YAML files
	configs := []struct {
		filename string
		content  string
	}{
		{
			"feed1.yml",
			`
url: "https://example.com/feed1.xml"
settings:
  enabled: true
`,
		},
		{
			"feed2.yml",
			`
url: "https://example.com/feed2.xml"
settings:
  enabled: true
`,
		},
	}

	for _, config := range configs {
		err := os.WriteFile(filepath.Join(tempDir, config.filename), []byte(config.content), 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Load feedConfigs
	configCache := NewConfigCache(tempDir)
	err := configCache.Run()
	if err != nil {
		t.Fatal(err)
	}

	// Get all configs
	allConfigs := configCache.GetConfigs()
	if len(allConfigs) != 2 {
		t.Errorf("Expected 2 configs, got %d", len(allConfigs))
	}

	// Verify it's a copy (modifying returned map shouldn't affect cache)
	delete(allConfigs, "feed1")
	if configCache.GetConfigCount() != 2 {
		t.Error("Modifying returned configs map affected the cache")
	}
}

// Validation tests

func TestConfigCacheValidateConfigNil(t *testing.T) {
	configCache := NewConfigCache("")
	err := configCache.validateConfig(nil)
	if err == nil {
		t.Error("Expected error for nil feedConfig, got none")
	}
}

func TestConfigCacheValidateConfigRequiredFields(t *testing.T) {
	configCache := NewConfigCache("")

	// Test with empty feed name
	feedConfig := &Config{
		Name: "",
		URL:  "https://example.com/feed.xml",
	}
	err := configCache.validateConfig(feedConfig)
	if err == nil {
		t.Error("Expected error for empty feed name, got none")
	}

	// Test with empty URL
	feedConfig.Name = "test-feed"
	feedConfig.URL = ""
	err = configCache.validateConfig(feedConfig)
	if err == nil {
		t.Error("Expected error for empty URL, got none")
	}
}

func TestConfigCacheValidateConfigNegativeValues(t *testing.T) {
	configCache := NewConfigCache("")

	feedConfig := &Config{
		Name: "test-feed",
		URL:  "https://example.com/feed.xml",
	}

	// Test with negative refresh interval
	feedConfig.Settings.RefreshInterval = -1
	err := configCache.validateConfig(feedConfig)
	if err == nil {
		t.Error("Expected error for negative refresh interval, got none")
	}

	// Test with negative max items
	feedConfig.Settings.RefreshInterval = 3600
	feedConfig.Settings.MaxItems = -1
	err = configCache.validateConfig(feedConfig)
	if err == nil {
		t.Error("Expected error for negative max items, got none")
	}

	// Test with negative timeout
	feedConfig.Settings.MaxItems = 100
	feedConfig.Settings.Timeout = -1
	err = configCache.validateConfig(feedConfig)
	if err == nil {
		t.Error("Expected error for negative timeout, got none")
	}
}

func TestConfigCacheValidateConfigFilters(t *testing.T) {
	configCache := NewConfigCache("")

	feedConfig := &Config{
		Name: "test-feed",
		URL:  "https://example.com/feed.xml",
		Settings: ConfigSettings{
			RefreshInterval: 3600,
			MaxItems:        100,
			Timeout:         30,
		},
	}

	// Test with invalid filter field
	feedConfig.Filters = []ConfigFilter{
		{
			Field:    "invalid_field",
			Includes: []string{"test"},
		},
	}
	err := configCache.validateConfig(feedConfig)
	if err == nil {
		t.Error("Expected error for invalid filter field, got none")
	}

	// Test with filter having no includes or excludes
	feedConfig.Filters = []ConfigFilter{
		{
			Field:    "title",
			Includes: []string{},
			Excludes: []string{},
		},
	}
	err = configCache.validateConfig(feedConfig)
	if err == nil {
		t.Error("Expected error for filter with no includes or excludes, got none")
	}

	// Test with valid feedConfig
	feedConfig.Filters = []ConfigFilter{
		{
			Field:    "title",
			Includes: []string{"test"},
		},
	}
	err = configCache.validateConfig(feedConfig)
	if err != nil {
		t.Errorf("Expected no error for valid feedConfig, got: %v", err)
	}
}

func TestConfigCacheValidateConfigValidFilterFields(t *testing.T) {
	configCache := NewConfigCache("")

	feedConfig := &Config{
		Name: "test-feed",
		URL:  "https://example.com/feed.xml",
		Settings: ConfigSettings{
			RefreshInterval: 3600,
			MaxItems:        100,
			Timeout:         30,
		},
	}

	// Test all valid filter fields
	validFields := []string{"title", "description", "content", "authors", "link", "categories"}
	for _, field := range validFields {
		feedConfig.Filters = []ConfigFilter{
			{
				Field:    field,
				Includes: []string{"test"},
			},
		}
		err := configCache.validateConfig(feedConfig)
		if err != nil {
			t.Errorf("Expected no error for valid filter field '%s', got: %v", field, err)
		}
	}

	// Test invalid filter field
	feedConfig.Filters = []ConfigFilter{
		{
			Field:    "invalid_field",
			Includes: []string{"test"},
		},
	}
	err := configCache.validateConfig(feedConfig)
	if err == nil {
		t.Error("Expected error for invalid filter field, got none")
	}
}

func TestConfigCacheGetConfig(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Create multiple test YAML files with different feed names
	configs := []struct {
		filename string
		content  string
	}{
		{
			"feed1.yml",
			`
url: "https://example.com/feed1.xml"
settings:
  enabled: true
`,
		},
		{
			"feed2.yml",
			`
url: "https://example.com/feed2.xml"
settings:
  enabled: true
`,
		},
		{
			"special-chars-feed.yml",
			`
url: "https://example.com/special.xml"
settings:
  enabled: false
`,
		},
	}

	for _, config := range configs {
		err := os.WriteFile(filepath.Join(tempDir, config.filename), []byte(config.content), 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Load configs
	configCache := NewConfigCache(tempDir)
	err := configCache.Run()
	if err != nil {
		t.Fatal(err)
	}

	// Test getting existing feed by name
	feedConfig, err := configCache.GetConfig("feed1")
	if err != nil {
		t.Fatalf("Expected no error for existing feed name, got: %v", err)
	}
	if feedConfig == nil {
		t.Fatal("Expected config to be returned, got nil")
	}
	if feedConfig.Name != "feed1" {
		t.Errorf("Expected feed name 'feed1', got '%s'", feedConfig.Name)
	}
	if feedConfig.URL != "https://example.com/feed1.xml" {
		t.Errorf("Expected feed URL 'https://example.com/feed1.xml', got '%s'", feedConfig.URL)
	}
	if !feedConfig.Settings.Enabled {
		t.Error("Expected feed to be enabled")
	}

	// Test getting another existing feed by name
	feedConfig2, err := configCache.GetConfig("feed2")
	if err != nil {
		t.Fatalf("Expected no error for existing feed name, got: %v", err)
	}
	if feedConfig2 == nil {
		t.Fatal("Expected config to be returned, got nil")
	}
	if feedConfig2.Name != "feed2" {
		t.Errorf("Expected feed name 'feed2', got '%s'", feedConfig2.Name)
	}

	// Test getting feed with special characters in name
	feedConfig3, err := configCache.GetConfig("special-chars-feed")
	if err != nil {
		t.Fatalf("Expected no error for existing feed name with special chars, got: %v", err)
	}
	if feedConfig3 == nil {
		t.Fatal("Expected config to be returned, got nil")
	}
	if feedConfig3.Name != "special-chars-feed" {
		t.Errorf("Expected feed name 'special-chars-feed', got '%s'", feedConfig3.Name)
	}
	if feedConfig3.Settings.Enabled {
		t.Error("Expected feed to be disabled")
	}

	// Test getting non-existent feed by name
	_, err = configCache.GetConfig("non-existent-feed")
	if err == nil {
		t.Error("Expected error for non-existent feed name, got none")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected error message to contain 'not found', got: %v", err)
	}

	// Test with empty feed name
	_, err = configCache.GetConfig("")
	if err == nil {
		t.Error("Expected error for empty feed name, got none")
	}

	// Test case sensitivity - feed names should be case sensitive
	_, err = configCache.GetConfig("FEED1")
	if err == nil {
		t.Error("Expected error for case-mismatched feed name, got none")
	}
}

func TestConfigCacheGetConfigEmptyCache(t *testing.T) {
	// Create temp directory with no files
	tempDir := t.TempDir()

	// Load empty cache
	configCache := NewConfigCache(tempDir)
	err := configCache.Run()
	if err != nil {
		t.Fatal(err)
	}

	// Test getting feed by name from empty cache
	_, err = configCache.GetConfig("any-feed")
	if err == nil {
		t.Error("Expected error for feed name in empty cache, got none")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected error message to contain 'not found', got: %v", err)
	}
}
