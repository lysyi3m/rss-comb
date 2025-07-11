package config_watcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lysyi3m/rss-comb/app/config_loader"
)

func TestConfigWatcher(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "config-watcher-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create initial config file
	configFile := filepath.Join(tempDir, "test.yml")
	initialConfig := `feed:
  id: "test-feed"
  url: "https://example.com/feed.xml"
  title: "Test Feed"

settings:
  enabled: true
  refresh_interval: 3600
  max_items: 50

filters:
  - field: "title"
    includes: ["test"]`

	err = os.WriteFile(configFile, []byte(initialConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Create config watcher
	loader := config_loader.NewLoader(tempDir)
	watcher, err := NewConfigWatcher(loader, tempDir)
	if err != nil {
		t.Fatalf("Failed to create config watcher: %v", err)
	}
	defer watcher.Stop()

	// Verify initial config loaded
	configs := watcher.GetConfigs()
	if len(configs) != 1 {
		t.Fatalf("Expected 1 config, got %d", len(configs))
	}

	config, exists := configs[configFile]
	if !exists {
		t.Fatalf("Config file not found in loaded configs")
	}

	if config.Feed.ID != "test-feed" {
		t.Errorf("Expected feed ID 'test-feed', got '%s'", config.Feed.ID)
	}

	// Set up update handler to track changes
	updateReceived := make(chan bool, 1)
	watcher.AddUpdateHandler(&TestUpdateHandler{
		updateChan: updateReceived,
	})

	// Start watcher in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		watcher.Start(ctx)
	}()

	// Give watcher time to start
	time.Sleep(100 * time.Millisecond)

	// Modify config file
	updatedConfig := `feed:
  id: "test-feed"
  url: "https://example.com/updated-feed.xml"
  title: "Updated Test Feed"

settings:
  enabled: false
  refresh_interval: 1800
  max_items: 100

filters:
  - field: "title"
    includes: ["updated"]`

	err = os.WriteFile(configFile, []byte(updatedConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to update config file: %v", err)
	}

	// Wait for update notification
	select {
	case <-updateReceived:
		// Update received, check new config
		updatedConfigs := watcher.GetConfigs()
		if len(updatedConfigs) != 1 {
			t.Fatalf("Expected 1 config after update, got %d", len(updatedConfigs))
		}

		updatedConfigObj, exists := updatedConfigs[configFile]
		if !exists {
			t.Fatalf("Updated config file not found")
		}

		if updatedConfigObj.Feed.URL != "https://example.com/updated-feed.xml" {
			t.Errorf("Expected updated URL, got '%s'", updatedConfigObj.Feed.URL)
		}

		if updatedConfigObj.Settings.Enabled != false {
			t.Errorf("Expected enabled=false, got %v", updatedConfigObj.Settings.Enabled)
		}

	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for config update")
	}
}

// TestUpdateHandler implements ConfigUpdateHandler for testing
type TestUpdateHandler struct {
	updateChan chan bool
}

func (h *TestUpdateHandler) OnConfigUpdate(filePath string, config *FeedConfig, isDelete bool) error {
	select {
	case h.updateChan <- true:
	default:
		// Channel full, ignore
	}
	return nil
}

func TestConfigFileDeletion(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "config-deletion-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create config file
	configFile := filepath.Join(tempDir, "test.yml")
	configContent := `feed:
  id: "test-feed"
  url: "https://example.com/feed.xml"
  title: "Test Feed"
settings:
  enabled: true`

	err = os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Create config watcher
	loader := config_loader.NewLoader(tempDir)
	watcher, err := NewConfigWatcher(loader, tempDir)
	if err != nil {
		t.Fatalf("Failed to create config watcher: %v", err)
	}
	defer watcher.Stop()

	// Verify initial config loaded
	configs := watcher.GetConfigs()
	if len(configs) != 1 {
		t.Fatalf("Expected 1 config, got %d", len(configs))
	}

	// Set up update handler to track deletions
	deletionReceived := make(chan bool, 1)
	handler := &DeletionTestHandler{
		deletionChan: deletionReceived,
	}
	watcher.AddUpdateHandler(handler)

	// Start watcher in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		watcher.Start(ctx)
	}()

	// Give watcher time to start
	time.Sleep(100 * time.Millisecond)

	// Delete the config file
	err = os.Remove(configFile)
	if err != nil {
		t.Fatalf("Failed to delete config file: %v", err)
	}

	// Wait for deletion notification
	select {
	case <-deletionReceived:
		// Deletion received, check configs
		updatedConfigs := watcher.GetConfigs()
		if len(updatedConfigs) != 0 {
			t.Fatalf("Expected 0 configs after deletion, got %d", len(updatedConfigs))
		}

	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for config deletion")
	}
}

// DeletionTestHandler implements ConfigUpdateHandler for testing deletions
type DeletionTestHandler struct {
	deletionChan chan bool
}

func (h *DeletionTestHandler) OnConfigUpdate(filePath string, config *FeedConfig, isDelete bool) error {
	if isDelete {
		select {
		case h.deletionChan <- true:
		default:
			// Channel full, ignore
		}
	}
	return nil
}