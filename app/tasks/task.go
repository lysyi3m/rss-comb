package tasks

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

type TaskType string

const (
	TaskTypeProcessFeed  TaskType = "process_feed"
	TaskTypeRefilterFeed TaskType = "refilter_feed"
)

type TaskInterface interface {
	Execute(ctx context.Context) error
	GetID() string
	GetType() TaskType
	GetFeedName() string
	Start()
	GetDuration() time.Duration
}

type Task struct {
	ID        string
	Type      TaskType
	FeedName  string
	StartedAt *time.Time
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
		ID:       uniqueID,
		Type:     taskType,
		FeedName: feedName,
	}
}
