package tasks

import (
	"context"
	"testing"
	"time"

	"github.com/lysyi3m/rss-comb/app/config"
	"github.com/lysyi3m/rss-comb/app/config_sync"
	"github.com/lysyi3m/rss-comb/app/database"
)

// MockFeedRepository implements a simple mock for testing
type MockFeedRepository struct {
	feeds []database.Feed
	err   error
}

func (m *MockFeedRepository) GetFeedsDueForRefresh() ([]database.Feed, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.feeds, nil
}

func (m *MockFeedRepository) UpsertFeed(configFile, feedID, feedURL, feedName string) (string, error) {
	return "test-id", nil
}

func (m *MockFeedRepository) UpsertFeedWithChangeDetection(configFile, feedID, feedURL, feedName string) (string, bool, error) {
	return "test-id", false, nil
}

func (m *MockFeedRepository) UpdateFeedMetadata(feedID string, link string, imageURL string, language string) error {
	return nil
}

func (m *MockFeedRepository) UpdateNextFetch(feedID string, nextFetch time.Time) error {
	return nil
}

func (m *MockFeedRepository) GetFeedByConfigFile(configFile string) (*database.Feed, error) {
	return nil, nil
}

func (m *MockFeedRepository) GetFeedByURL(feedURL string) (*database.Feed, error) {
	return nil, nil
}

func (m *MockFeedRepository) GetFeedByID(feedID string) (*database.Feed, error) {
	return nil, nil
}

func (m *MockFeedRepository) SetFeedEnabled(feedID string, enabled bool) error {
	return nil
}

func (m *MockFeedRepository) GetFeedCount() (int, error) {
	return len(m.feeds), nil
}

func (m *MockFeedRepository) GetEnabledFeedCount() (int, error) {
	count := 0
	for _, feed := range m.feeds {
		if feed.IsEnabled {
			count++
		}
	}
	return count, nil
}

// MockProcessor implements a simple mock for testing
type MockProcessor struct {
	processedFeeds []string
	shouldError    bool
}

// Ensure MockProcessor implements ProcessorInterface interface
var _ ProcessorInterface = (*MockProcessor)(nil)

func (m *MockProcessor) ProcessFeed(feedID string, feedConfig *config.FeedConfig) error {
	m.processedFeeds = append(m.processedFeeds, feedID)
	if m.shouldError {
		return &testError{"mock error"}
	}
	return nil
}

func (m *MockProcessor) IsFeedEnabled(feedConfig *config.FeedConfig) bool {
	// Mock implementation - return true for all feeds except those with "disabled" in the name
	if feedConfig == nil {
		return false
	}
	return true
}

func (m *MockProcessor) ReapplyFilters(feedID string, feedConfig *config.FeedConfig) (int, int, error) {
	// Mock implementation - return 0 updated items, 0 errors
	if m.shouldError {
		return 0, 1, &testError{"mock reapply error"}
	}
	return 0, 0, nil
}

// MockContentExtractionInterface implements a simple mock for testing
type MockContentExtractionInterface struct {
	shouldError bool
}

func (m *MockContentExtractionInterface) ExtractContentForFeed(ctx context.Context, feedID string, feedConfig *config.FeedConfig) error {
	if m.shouldError {
		return &testError{"mock extraction error"}
	}
	return nil
}

// Helper function to create test configs
func createTestConfigs() map[string]*config.FeedConfig {
	return map[string]*config.FeedConfig{
		"test.yml": {
			Feed: config.FeedInfo{
				ID:    "test-feed",
				Title: "Test Feed",
				URL:   "https://example.com/feed.xml",
			},
			Settings: config.FeedSettings{
				Enabled: true,
			},
		},
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestNewTaskScheduler(t *testing.T) {
	mockRepo := &MockFeedRepository{}
	mockProcessor := &MockProcessor{}
	configs := createTestConfigs()
	configCache := config_sync.NewConfigCacheHandler("Test", configs)

	mockContentExtractor := &MockContentExtractionInterface{}
	scheduler := NewTaskScheduler(mockProcessor, mockRepo, configCache, mockContentExtractor, time.Second, 2)

	if scheduler == nil {
		t.Fatal("Expected scheduler to be created")
	}

	// Test that the scheduler implements the interface properly
	// The scheduler should be created successfully without any errors
}

// TestTaskSchedulerGetStats - removed since GetStats() method no longer exists
// Statistics tracking was identified as dead code and removed

// TestTaskSchedulerHealth - removed since Health() method no longer exists
// Health monitoring was identified as dead code and removed

// TestTaskSchedulerHealthWithHighErrorRate - removed since Health() method no longer exists
// High error rate monitoring was identified as dead code and removed

func TestTaskSchedulerExecuteTask(t *testing.T) {
	mockRepo := &MockFeedRepository{}
	mockProcessor := &MockProcessor{}

	configs := createTestConfigs()
	configCache := config_sync.NewConfigCacheHandler("Test", configs)
	mockContentExtractor := &MockContentExtractionInterface{}
	scheduler := NewTaskScheduler(mockProcessor, mockRepo, configCache, mockContentExtractor, time.Second, 1)

	// Test successful task execution via EnqueueTask
	task := NewProcessFeedTask("test-id", createTestConfigs()["test.yml"], mockProcessor)
	err := scheduler.EnqueueTask(task)
	if err != nil {
		t.Errorf("Expected no error enqueuing task, got %v", err)
	}

	// Cannot test execution directly since executeTask is private
	// This is better encapsulation. We can only test the public interface.
	
	// Test error processing
	mockProcessor.shouldError = true
	task2 := NewProcessFeedTask("test-id-2", createTestConfigs()["test.yml"], mockProcessor)
	err = scheduler.EnqueueTask(task2)
	if err != nil {
		t.Errorf("Expected no error enqueuing task, got %v", err)
	}
}

func TestTaskSchedulerLifecycle(t *testing.T) {
	mockRepo := &MockFeedRepository{
		feeds: []database.Feed{
			{
				ID:         "test-id",
				ConfigFile: "test.yml",
				Title:      "Test Feed",
				FeedURL:    "https://example.com/feed.xml",
			},
		},
	}
	mockProcessor := &MockProcessor{}

	configs := createTestConfigs()
	configCache := config_sync.NewConfigCacheHandler("Test", configs)
	mockContentExtractor := &MockContentExtractionInterface{}
	scheduler := NewTaskScheduler(mockProcessor, mockRepo, configCache, mockContentExtractor, 100*time.Millisecond, 1)

	// Start scheduler
	scheduler.Start()

	// Wait a bit for processing
	time.Sleep(200 * time.Millisecond)

	// Stop scheduler
	scheduler.Stop()

	// Verify processing occurred
	if len(mockProcessor.processedFeeds) == 0 {
		t.Error("Expected at least one feed to be processed")
	}
}

func TestEnqueueTask(t *testing.T) {
	mockRepo := &MockFeedRepository{}
	mockProcessor := &MockProcessor{}

	configs := createTestConfigs()
	configCache := config_sync.NewConfigCacheHandler("Test", configs)
	mockContentExtractor := &MockContentExtractionInterface{}
	scheduler := NewTaskScheduler(mockProcessor, mockRepo, configCache, mockContentExtractor, time.Second, 1)

	// Test successful enqueue
	task := NewProcessFeedTask("test-id", createTestConfigs()["test.yml"], mockProcessor)
	err := scheduler.EnqueueTask(task)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Task should be enqueued successfully without errors
	// Statistics tracking was removed as dead code
}

func TestRefilterFeedTask(t *testing.T) {
	mockProcessor := &MockProcessor{}

	task := NewRefilterFeedTask("test-id", createTestConfigs()["test.yml"], mockProcessor)

	if task.GetType() != TaskTypeRefilterFeed {
		t.Errorf("Expected task type %s, got %s", TaskTypeRefilterFeed, task.GetType())
	}

	if task.GetFeedID() != "test-id" {
		t.Errorf("Expected feed ID 'test-id', got '%s'", task.GetFeedID())
	}

	if task.GetFeedConfig().Feed.ID != "test-feed" {
		t.Errorf("Expected feed config ID 'test-feed', got '%s'", task.GetFeedConfig().Feed.ID)
	}

	// Test execution
	ctx := context.Background()
	err := task.Execute(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestProcessFeedTask(t *testing.T) {
	mockProcessor := &MockProcessor{}

	task := NewProcessFeedTask("test-id", createTestConfigs()["test.yml"], mockProcessor)

	if task.GetType() != TaskTypeProcessFeed {
		t.Errorf("Expected task type %s, got %s", TaskTypeProcessFeed, task.GetType())
	}

	if task.GetFeedID() != "test-id" {
		t.Errorf("Expected feed ID 'test-id', got '%s'", task.GetFeedID())
	}

	if task.GetFeedConfig().Feed.ID != "test-feed" {
		t.Errorf("Expected feed config ID 'test-feed', got '%s'", task.GetFeedConfig().Feed.ID)
	}

	// Test execution
	ctx := context.Background()
	err := task.Execute(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify feed was processed
	if len(mockProcessor.processedFeeds) != 1 {
		t.Errorf("Expected 1 processed feed, got %d", len(mockProcessor.processedFeeds))
	}

	if mockProcessor.processedFeeds[0] != "test-id" {
		t.Errorf("Expected processed feed ID 'test-id', got '%s'", mockProcessor.processedFeeds[0])
	}
}

// TestTaskStats - removed since statistics tracking was identified as dead code
// All stats-related functionality has been eliminated from TaskScheduler
