package models

import (
	"database/sql"
	"time"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusTodo           TaskStatus = "todo"
	TaskStatusInProgress     TaskStatus = "in_progress"
	TaskStatusBlocked        TaskStatus = "blocked"
	TaskStatusReadyForReview TaskStatus = "ready_for_review"
	TaskStatusCompleted      TaskStatus = "completed"
	TaskStatusArchived       TaskStatus = "archived"
)

// AgentType represents the type of agent assigned to a task
type AgentType string

const (
	AgentTypeFrontend AgentType = "frontend"
	AgentTypeBackend  AgentType = "backend"
	AgentTypeAPI      AgentType = "api"
	AgentTypeTesting  AgentType = "testing"
	AgentTypeDevOps   AgentType = "devops"
	AgentTypeGeneral  AgentType = "general"
)

// Task represents an atomic work unit within a feature
type Task struct {
	ID             int64        `json:"id" db:"id"`
	FeatureID      int64        `json:"feature_id" db:"feature_id"`
	Key            string       `json:"key" db:"key"`
	Title          string       `json:"title" db:"title"`
	Description    *string      `json:"description,omitempty" db:"description"`
	Status         TaskStatus   `json:"status" db:"status"`
	AgentType      *AgentType   `json:"agent_type,omitempty" db:"agent_type"`
	Priority       int          `json:"priority" db:"priority"`
	DependsOn      *string      `json:"depends_on,omitempty" db:"depends_on"` // JSON array
	AssignedAgent  *string      `json:"assigned_agent,omitempty" db:"assigned_agent"`
	FilePath       *string      `json:"file_path,omitempty" db:"file_path"`
	BlockedReason  *string      `json:"blocked_reason,omitempty" db:"blocked_reason"`
	ExecutionOrder *int         `json:"execution_order,omitempty" db:"execution_order"`
	CreatedAt      time.Time    `json:"created_at" db:"created_at"`
	StartedAt      sql.NullTime `json:"started_at,omitempty" db:"started_at"`
	CompletedAt    sql.NullTime `json:"completed_at,omitempty" db:"completed_at"`
	BlockedAt      sql.NullTime `json:"blocked_at,omitempty" db:"blocked_at"`
	UpdatedAt      time.Time    `json:"updated_at" db:"updated_at"`
}

// Validate validates the Task fields
func (t *Task) Validate() error {
	if err := ValidateTaskKey(t.Key); err != nil {
		return err
	}
	if t.Title == "" {
		return ErrEmptyTitle
	}
	if err := ValidateTaskStatus(string(t.Status)); err != nil {
		return err
	}
	if t.AgentType != nil {
		if err := ValidateAgentType(string(*t.AgentType)); err != nil {
			return err
		}
	}
	if t.Priority < 1 || t.Priority > 10 {
		return ErrInvalidPriority
	}
	if t.DependsOn != nil {
		if err := ValidateDependsOn(*t.DependsOn); err != nil {
			return err
		}
	}
	return nil
}
