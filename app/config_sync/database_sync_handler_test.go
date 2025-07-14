package config_sync

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lysyi3m/rss-comb/app/config"
	"github.com/lysyi3m/rss-comb/app/database"
)

// MockFeedRepository implements FeedRepositoryInterface for testing
type MockFeedRepository struct {
	feeds       map[string]*database.Feed
	nextFetches map[string]time.Time
	enabledMap  map[string]bool
}

func NewMockFeedRepository() *MockFeedRepository {
	return &MockFeedRepository{
		feeds:       make(map[string]*database.Feed),
		nextFetches: make(map[string]time.Time),
		enabledMap:  make(map[string]bool),
	}
}

func (m *MockFeedRepository) UpsertFeedWithChangeDetection(configFile, feedID, feedURL, feedTitle string) (string, bool, error) {
	dbID := feedID + "-db"
	urlChanged := false
	
	if existingFeed, exists := m.feeds[feedID]; exists {
		if existingFeed.FeedURL != feedURL {
			urlChanged = true
		}
		existingFeed.FeedURL = feedURL
		existingFeed.Title = feedTitle
		existingFeed.ConfigFile = configFile
	} else {
		m.feeds[feedID] = &database.Feed{
			ID:         dbID,
			FeedID:     feedID,
			ConfigFile: configFile,
			FeedURL:    feedURL,
			Title:      feedTitle,
			Enabled:    true,
		}
	}
	
	return dbID, urlChanged, nil
}

func (m *MockFeedRepository) GetFeedByID(feedID string) (*database.Feed, error) {
	if feed, exists := m.feeds[feedID]; exists {
		return feed, nil
	}
	return nil, nil
}

func (m *MockFeedRepository) SetFeedEnabled(feedID string, enabled bool) error {
	m.enabledMap[feedID] = enabled
	return nil
}

func (m *MockFeedRepository) UpdateNextFetch(feedID string, nextFetch time.Time) error {
	m.nextFetches[feedID] = nextFetch
	return nil
}

// Implement other required methods with minimal implementations
func (m *MockFeedRepository) GetFeedsDueForRefresh() ([]database.Feed, error) { return nil, nil }
func (m *MockFeedRepository) UpsertFeed(configFile, feedID, feedURL, feedTitle string) (string, error) { return "", nil }
func (m *MockFeedRepository) UpdateFeedMetadata(feedID string, link string, imageURL string, language string) error { return nil }
func (m *MockFeedRepository) UpdateFeedTimestamp(feedID string, feedPublishedAt *time.Time) error { return nil }
func (m *MockFeedRepository) GetFeedByConfigFile(configFile string) (*database.Feed, error) { return nil, nil }
func (m *MockFeedRepository) GetFeedByURL(feedURL string) (*database.Feed, error) { return nil, nil }
func (m *MockFeedRepository) GetFeedCount() (int, error) { return 0, nil }
func (m *MockFeedRepository) GetEnabledFeedCount() (int, error) { return 0, nil }

func TestDatabaseSyncHandlerConfigUpsert(t *testing.T) {
	// Create a temporary directory for test
	tempDir := t.TempDir()
	
	// Create test config file
	configFile := filepath.Join(tempDir, "test.yml")
	if err := os.WriteFile(configFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	
	// Create mock repository
	mockRepo := NewMockFeedRepository()
	
	// Create handler
	handler := NewDatabaseSyncHandler(mockRepo, tempDir)
	
	// Create test config
	cfg := &config.FeedConfig{
		Feed: config.FeedInfo{
			ID:    "test-feed",
			URL:   "https://example.com/feed.xml",
			Title: "Test Feed",
		},
		Settings: config.FeedSettings{
			Enabled: true,
		},
	}
	
	// Test config upsert (creation)
	err := handler.OnConfigUpdate(configFile, cfg, false)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	
	// Verify feed was registered
	if _, exists := mockRepo.feeds["test-feed"]; !exists {
		t.Error("Feed was not registered in mock repository")
	}
	
	// Verify next_fetch was reset for immediate processing
	if nextFetch, exists := mockRepo.nextFetches["test-feed-db"]; !exists || !nextFetch.IsZero() {
		t.Error("Expected next_fetch to be reset to zero time for immediate processing")
	}
}

func TestDatabaseSyncHandlerConfigDeletion(t *testing.T) {
	// Create mock repository with existing feed
	mockRepo := NewMockFeedRepository()
	mockRepo.feeds["test-feed"] = &database.Feed{
		ID:     "test-feed-db",
		FeedID: "test-feed",
		Title:  "Test Feed",
	}
	
	// Create handler
	handler := NewDatabaseSyncHandler(mockRepo, "/tmp")
	
	// Create test config
	cfg := &config.FeedConfig{
		Feed: config.FeedInfo{
			ID:    "test-feed",
			URL:   "https://example.com/feed.xml",
			Title: "Test Feed",
		},
	}
	
	// Test config deletion
	err := handler.OnConfigUpdate("/tmp/test.yml", cfg, true)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	
	// Verify feed was disabled
	if enabled, exists := mockRepo.enabledMap["test-feed-db"]; !exists || enabled {
		t.Error("Expected feed to be disabled in mock repository")
	}
}

func TestDatabaseSyncHandlerValidation(t *testing.T) {
	// Test with nil config
	err := config.ValidateConfig(nil)
	if err == nil {
		t.Error("Expected error for nil config, got none")
	}
	
	// Test with empty feed ID
	cfg := &config.FeedConfig{
		Feed: config.FeedInfo{
			ID:    "",
			URL:   "https://example.com/feed.xml",
			Title: "Test Feed",
		},
	}
	err = config.ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for empty feed ID, got none")
	}
	
	// Test with empty URL
	cfg.Feed.ID = "test-feed"
	cfg.Feed.URL = ""
	err = config.ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for empty URL, got none")
	}
	
	// Test with empty title
	cfg.Feed.URL = "https://example.com/feed.xml"
	cfg.Feed.Title = ""
	err = config.ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for empty title, got none")
	}
	
	// Test with valid config
	cfg.Feed.Title = "Test Feed"
	cfg.Filters = []config.Filter{
		{
			Field:    "title",
			Includes: []string{"test"},
		},
	}
	err = config.ValidateConfig(cfg)
	if err != nil {
		t.Errorf("Expected no error for valid config, got: %v", err)
	}
}

func TestDatabaseSyncHandlerNonExistentFile(t *testing.T) {
	mockRepo := NewMockFeedRepository()
	handler := NewDatabaseSyncHandler(mockRepo, "/tmp")
	
	// Create test config
	cfg := &config.FeedConfig{
		Feed: config.FeedInfo{
			ID:    "test-feed",
			URL:   "https://example.com/feed.xml",
			Title: "Test Feed",
		},
		Settings: config.FeedSettings{
			Enabled: true,
		},
	}
	
	// Test with non-existent file (should not return error, just log warning)
	err := handler.OnConfigUpdate("/tmp/nonexistent.yml", cfg, false)
	if err != nil {
		t.Errorf("Expected no error for non-existent file, got: %v", err)
	}
}