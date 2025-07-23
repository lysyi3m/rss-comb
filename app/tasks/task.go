package tasks

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

type TaskType string

const (
	TaskTypeExtractContent TaskType = "extract_content"
	TaskTypeProcessFeed    TaskType = "process_feed"
	TaskTypeRefilterFeed   TaskType = "refilter_feed"
	TaskTypeSyncFeedConfig TaskType = "sync_feed_config"
)

const (
	DefaultMaxRetries = 3
)

type TaskInterface interface {
	Execute(ctx context.Context) error
	GetID() string
	GetType() TaskType
	GetFeedName() string
	GetRetryCount() int
	GetMaxRetries() int
	IncrementRetryCount()
	CanRetry() bool
	Start()
	GetDuration() time.Duration
}

type Task struct {
	ID         string
	Type       TaskType
	FeedName   string
	RetryCount int
	MaxRetries int
	StartedAt  *time.Time
}

func (t *Task) GetID() string {
	return t.ID
}

func (t *Task) GetType() TaskType {
	return t.Type
}

func (t *Task) GetFeedName() string {
	return t.FeedName
}

func (t *Task) GetRetryCount() int {
	return t.RetryCount
}

func (t *Task) GetMaxRetries() int {
	return t.MaxRetries
}

func (t *Task) IncrementRetryCount() {
	t.RetryCount++
}

func (t *Task) CanRetry() bool {
	return t.RetryCount < t.MaxRetries
}

func (t *Task) Start() {
	now := time.Now()
	t.StartedAt = &now
}

func (t *Task) GetDuration() time.Duration {
	if t.StartedAt == nil {
		return 0
	}
	return time.Since(*t.StartedAt)
}

func NewTask(taskType TaskType, feedName string) Task {
	uniqueID := fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Intn(10000))

	return Task{
		ID:         uniqueID,
		Type:       taskType,
		FeedName:   feedName,
		RetryCount: 0,
		MaxRetries: DefaultMaxRetries,
	}
}
