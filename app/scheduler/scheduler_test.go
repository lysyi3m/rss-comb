package scheduler

import (
	"testing"
	"time"

	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/feed"
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

func (m *MockFeedRepository) UpdateFeedMetadata(feedID string, iconURL string, language string) error {
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
		if feed.Enabled {
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

// Ensure MockProcessor implements FeedProcessor interface
var _ feed.FeedProcessor = (*MockProcessor)(nil)

func (m *MockProcessor) ProcessFeed(feedID, configFile string) error {
	m.processedFeeds = append(m.processedFeeds, feedID)
	if m.shouldError {
		return &testError{"mock error"}
	}
	return nil
}

func (m *MockProcessor) IsFeedEnabled(configFile string) bool {
	// Mock implementation - return true for all feeds except those with "disabled" in the name
	return true
}

func (m *MockProcessor) ReapplyFilters(feedID, configFile string) (int, int, error) {
	// Mock implementation - return 0 updated items, 0 errors
	if m.shouldError {
		return 0, 1, &testError{"mock reapply error"}
	}
	return 0, 0, nil
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestNewScheduler(t *testing.T) {
	mockRepo := &MockFeedRepository{}
	mockProcessor := &MockProcessor{}

	scheduler := NewScheduler(mockProcessor, mockRepo, time.Second, 2)

	if scheduler == nil {
		t.Fatal("Expected scheduler to be created")
	}

	if scheduler.workerCount != 2 {
		t.Errorf("Expected worker count 2, got %d", scheduler.workerCount)
	}

	if scheduler.interval != time.Second {
		t.Errorf("Expected interval 1s, got %v", scheduler.interval)
	}

	if scheduler.stats == nil {
		t.Error("Expected stats to be initialized")
	}

	if scheduler.stats.CurrentWorkers != 2 {
		t.Errorf("Expected current workers 2, got %d", scheduler.stats.CurrentWorkers)
	}
}

func TestGetStats(t *testing.T) {
	mockRepo := &MockFeedRepository{}
	mockProcessor := &MockProcessor{}

	scheduler := NewScheduler(mockProcessor, mockRepo, time.Second, 3)

	stats := scheduler.GetStats()

	if stats.CurrentWorkers != 3 {
		t.Errorf("Expected current workers 3, got %d", stats.CurrentWorkers)
	}

	if stats.TotalProcessed != 0 {
		t.Errorf("Expected total processed 0, got %d", stats.TotalProcessed)
	}

	if stats.TotalErrors != 0 {
		t.Errorf("Expected total errors 0, got %d", stats.TotalErrors)
	}

	if stats.QueueSize != 0 {
		t.Errorf("Expected queue size 0, got %d", stats.QueueSize)
	}
}

func TestHealth(t *testing.T) {
	mockRepo := &MockFeedRepository{}
	mockProcessor := &MockProcessor{}

	scheduler := NewScheduler(mockProcessor, mockRepo, time.Second, 2)

	health := scheduler.Health()

	if health["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", health["status"])
	}

	if health["workers"] != 2 {
		t.Errorf("Expected workers 2, got %v", health["workers"])
	}

	if health["queue_size"] != 0 {
		t.Errorf("Expected queue size 0, got %v", health["queue_size"])
	}

	if health["total_processed"] != int64(0) {
		t.Errorf("Expected total processed 0, got %v", health["total_processed"])
	}

	if health["total_errors"] != int64(0) {
		t.Errorf("Expected total errors 0, got %v", health["total_errors"])
	}

	// Test with some processing statistics (error rate > 10% for degraded)
	scheduler.mu.Lock()
	scheduler.stats.TotalProcessed = 10
	scheduler.stats.TotalErrors = 2 // 20% error rate
	scheduler.mu.Unlock()

	health = scheduler.Health()
	expectedErrorRate := 0.2
	if health["error_rate"] != expectedErrorRate {
		t.Errorf("Expected error rate %f, got %v", expectedErrorRate, health["error_rate"])
	}

	if health["status"] != "degraded" {
		t.Errorf("Expected status 'degraded' with 20%% error rate, got %v", health["status"])
	}
}

func TestHealthWithHighErrorRate(t *testing.T) {
	mockRepo := &MockFeedRepository{}
	mockProcessor := &MockProcessor{}

	scheduler := NewScheduler(mockProcessor, mockRepo, time.Second, 2)

	// Simulate high error rate
	scheduler.mu.Lock()
	scheduler.stats.TotalProcessed = 10
	scheduler.stats.TotalErrors = 6
	scheduler.mu.Unlock()

	health := scheduler.Health()

	if health["status"] != "unhealthy" {
		t.Errorf("Expected status 'unhealthy' with 60%% error rate, got %v", health["status"])
	}

	expectedErrorRate := 0.6
	if health["error_rate"] != expectedErrorRate {
		t.Errorf("Expected error rate %f, got %v", expectedErrorRate, health["error_rate"])
	}
}

func TestUpdateAverageProcessTime(t *testing.T) {
	mockRepo := &MockFeedRepository{}
	mockProcessor := &MockProcessor{}

	scheduler := NewScheduler(mockProcessor, mockRepo, time.Second, 1)

	// Test with no process times
	scheduler.updateAverageProcessTime()
	if scheduler.stats.AverageProcessTime != 0 {
		t.Errorf("Expected average process time 0 with no data, got %v", scheduler.stats.AverageProcessTime)
	}

	// Add some process times
	scheduler.stats.processTimes = []time.Duration{
		time.Second,
		2 * time.Second,
		3 * time.Second,
	}

	scheduler.updateAverageProcessTime()
	expected := 2 * time.Second
	if scheduler.stats.AverageProcessTime != expected {
		t.Errorf("Expected average process time %v, got %v", expected, scheduler.stats.AverageProcessTime)
	}
}

func TestProcessFeedStatistics(t *testing.T) {
	mockRepo := &MockFeedRepository{}
	mockProcessor := &MockProcessor{}

	scheduler := NewScheduler(mockProcessor, mockRepo, time.Second, 1)

	feed := database.Feed{
		ID:         "test-id",
		ConfigFile: "test.yml",
		Title:      "Test Feed",
		URL:        "https://example.com/feed.xml",
	}

	// Test successful processing
	scheduler.processFeed(0, feed)

	stats := scheduler.GetStats()
	if stats.TotalProcessed != 1 {
		t.Errorf("Expected total processed 1, got %d", stats.TotalProcessed)
	}

	if stats.TotalErrors != 0 {
		t.Errorf("Expected total errors 0, got %d", stats.TotalErrors)
	}

	if stats.LastProcessedAt == nil {
		t.Error("Expected last processed at to be set")
	}

	if len(mockProcessor.processedFeeds) != 1 {
		t.Errorf("Expected 1 processed feed, got %d", len(mockProcessor.processedFeeds))
	}

	if mockProcessor.processedFeeds[0] != "test-id" {
		t.Errorf("Expected processed feed ID 'test-id', got '%s'", mockProcessor.processedFeeds[0])
	}

	// Test error processing
	mockProcessor.shouldError = true
	scheduler.processFeed(0, feed)

	stats = scheduler.GetStats()
	if stats.TotalProcessed != 2 {
		t.Errorf("Expected total processed 2, got %d", stats.TotalProcessed)
	}

	if stats.TotalErrors != 1 {
		t.Errorf("Expected total errors 1, got %d", stats.TotalErrors)
	}
}

func TestSchedulerLifecycle(t *testing.T) {
	mockRepo := &MockFeedRepository{
		feeds: []database.Feed{
			{
				ID:         "test-id",
				ConfigFile: "test.yml",
				Title:      "Test Feed",
				URL:        "https://example.com/feed.xml",
			},
		},
	}
	mockProcessor := &MockProcessor{}

	scheduler := NewScheduler(mockProcessor, mockRepo, 100*time.Millisecond, 1)

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