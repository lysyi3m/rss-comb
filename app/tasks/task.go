package tasks

import (
	"context"
	"time"
)

// TaskType represents the type of task
type TaskType string

const (
	TaskTypeProcessFeed TaskType = "process_feed"
	TaskTypeRefilterFeed TaskType = "refilter_feed"
)

// TaskPriority represents the priority of a task
type TaskPriority int

const (
	PriorityLow    TaskPriority = 1
	PriorityNormal TaskPriority = 5
	PriorityHigh   TaskPriority = 10
)

// Task represents a unit of work that can be executed by the scheduler
type Task interface {
	// Execute runs the task with the given context
	Execute(ctx context.Context) error
	
	// GetID returns the unique identifier for this task
	GetID() string
	
	// GetType returns the type of task
	GetType() TaskType
	
	// GetPriority returns the priority of this task
	GetPriority() TaskPriority
	
	// GetCreatedAt returns when the task was created
	GetCreatedAt() time.Time
	
	// GetDescription returns a human-readable description of the task
	GetDescription() string
}

// BaseTask provides common functionality for all tasks
type BaseTask struct {
	ID          string
	Type        TaskType
	Priority    TaskPriority
	CreatedAt   time.Time
	Description string
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
func (t *BaseTask) GetPriority() TaskPriority {
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

// NewBaseTask creates a new base task with common fields
func NewBaseTask(id string, taskType TaskType, priority TaskPriority, description string) BaseTask {
	return BaseTask{
		ID:          id,
		Type:        taskType,
		Priority:    priority,
		CreatedAt:   time.Now(),
		Description: description,
	}
}