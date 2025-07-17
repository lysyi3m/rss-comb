package tasks

import (
	"context"
	"time"
)

// TaskType represents the type of task
type TaskType string

const (
	TaskTypeSyncFeedConfig TaskType = "sync_feed_config"
	TaskTypeProcessFeed    TaskType = "process_feed"
	TaskTypeRefilterFeed   TaskType = "refilter_feed"
	TaskTypeExtractContent TaskType = "extract_content"
)

// Priority represents task priority level
type Priority int

const (
	PriorityTop    Priority = 0 // Config sync tasks
	PriorityHigh   Priority = 1 // Feed processing, refilter tasks
	PriorityNormal Priority = 2 // Content extraction tasks
)

// Task represents a unit of work that can be executed by the scheduler
type Task interface {
	// Execute runs the task with the given context
	Execute(ctx context.Context) error
	
	// GetID returns the unique identifier for this task
	GetID() string
	
	// GetType returns the type of task
	GetType() TaskType
	
	// GetPriority returns the priority level of this task
	GetPriority() Priority
	
	// GetCreatedAt returns when the task was created
	GetCreatedAt() time.Time
	
	// GetDescription returns a human-readable description of the task
	GetDescription() string
	
	// GetFeedID returns the feed ID this task operates on
	GetFeedID() string
}

// BaseTask provides common functionality for all tasks
type BaseTask struct {
	ID          string
	Type        TaskType
	Priority    Priority
	CreatedAt   time.Time
	Description string
	FeedID      string
}

// GetID returns the task ID
func (t *BaseTask) GetID() string {
	return t.ID
}

// GetType returns the task type
func (t *BaseTask) GetType() TaskType {
	return t.Type
}

// GetPriority returns the task priority
func (t *BaseTask) GetPriority() Priority {
	return t.Priority
}

// GetCreatedAt returns when the task was created
func (t *BaseTask) GetCreatedAt() time.Time {
	return t.CreatedAt
}

// GetDescription returns the task description
func (t *BaseTask) GetDescription() string {
	return t.Description
}

// GetFeedID returns the feed ID this task operates on
func (t *BaseTask) GetFeedID() string {
	return t.FeedID
}

// NewBaseTask creates a new base task with common fields
func NewBaseTask(id string, taskType TaskType, priority Priority, description string, feedID string) BaseTask {
	return BaseTask{
		ID:          id,
		Type:        taskType,
		Priority:    priority,
		CreatedAt:   time.Now(),
		Description: description,
		FeedID:      feedID,
	}
}
