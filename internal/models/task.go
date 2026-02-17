package models

import (
	"database/sql"
	"time"
)

// TaskStatus represents the status of a task
type TaskStatus string

// Task represents an atomic work unit within a feature
type Task struct {
	ID             int64        `json:"id" db:"id"`
	FeatureID      int64        `json:"feature_id" db:"feature_id"`
	Key            string       `json:"key" db:"key"`
	Title          string       `json:"title" db:"title"`
	Slug           *string      `json:"slug,omitempty" db:"slug"`
	Description    *string      `json:"description,omitempty" db:"description"`
	Status         TaskStatus   `json:"status" db:"status"`
	AgentType      *string      `json:"agent_type,omitempty" db:"agent_type"`
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

	// Completion metadata fields
	CompletedBy        *string             `json:"completed_by,omitempty" db:"completed_by"`
	CompletionNotes    *string             `json:"completion_notes,omitempty" db:"completion_notes"`
	FilesChanged       *string             `json:"files_changed,omitempty" db:"files_changed"` // JSON array stored as string
	TestsPassed        bool                `json:"tests_passed" db:"tests_passed"`
	VerificationStatus *VerificationStatus `json:"verification_status,omitempty" db:"verification_status"`
	TimeSpentMinutes   *int                `json:"time_spent_minutes,omitempty" db:"time_spent_minutes"`

	// Context data for resume workflow
	ContextData *string `json:"context_data,omitempty" db:"context_data"` // JSON structured resume context

	// Metadata for task properties (e.g., complexity_tier for templates)
	Metadata map[string]interface{} `json:"metadata,omitempty" db:"-"` // Not persisted to DB, derived from related data

	// Rejection metadata fields
	RejectionCount  int        `json:"rejection_count" db:"-"`             // Derived from task_notes, not stored
	LastRejectionAt *time.Time `json:"last_rejection_at,omitempty" db:"-"` // Derived from task_notes, not stored
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
		if err := ValidateAgentType(*t.AgentType); err != nil {
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
