package tasks

import (
	"context"
	"testing"
	"time"

	"github.com/lysyi3m/rss-comb/app/feed_config"
	"github.com/lysyi3m/rss-comb/app/database"
)

// MockAppConfig for testing
type MockAppConfig struct {
	WorkerCount       int
	SchedulerInterval int
	Port              string
	UserAgent         string
	APIAccessKey      string
}

func (c *MockAppConfig) GetWorkerCount() int { return c.WorkerCount }
func (c *MockAppConfig) GetSchedulerInterval() int { return c.SchedulerInterval }
func (c *MockAppConfig) GetPort() string { return c.Port }
func (c *MockAppConfig) GetUserAgent() string { return c.UserAgent }
func (c *MockAppConfig) GetAPIAccessKey() string { return c.APIAccessKey }

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

func (m *MockFeedRepository) UpdateFeedMetadata(feedID string, link string, imageURL string, language string, feedPublishedAt *time.Time) error {
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

func (m *MockProcessor) ProcessFeed(feedID string, feedConfig *feed_config.FeedConfig) error {
	m.processedFeeds = append(m.processedFeeds, feedID)
	if m.shouldError {
		return &testError{"mock error"}
	}
	return nil
}

func (m *MockProcessor) ReapplyFilters(feedID string, feedConfig *feed_config.FeedConfig) (int, int, error) {
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

func (m *MockContentExtractionInterface) ExtractContentForFeed(ctx context.Context, feedID string, feedConfig *feed_config.FeedConfig) error {
	if m.shouldError {
		return &testError{"mock extraction error"}
	}
	return nil
}

// Helper function to create test configs
func createTestConfigs() map[string]*feed_config.FeedConfig {
	return map[string]*feed_config.FeedConfig{
		"test.yml": {
			Feed: feed_config.FeedInfo{
				ID:    "test-feed",
				Title: "Test Feed",
				URL:   "https://example.com/feed.xml",
			},
			Settings: feed_config.FeedSettings{
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
	t.Skip("Skipping test that requires global config initialization")
}

func TestTaskSchedulerExecuteTask(t *testing.T) {
	t.Skip("Skipping test that requires global config initialization")
}

func TestTaskSchedulerLifecycle(t *testing.T) {
	t.Skip("Skipping test that requires global config initialization")
}

func TestEnqueueTask(t *testing.T) {
	t.Skip("Skipping test that requires global config initialization")
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