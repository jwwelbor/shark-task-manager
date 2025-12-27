package models

import (
	"database/sql"
	"time"
)

// SessionOutcome represents the outcome of a work session
type SessionOutcome string

const (
	SessionOutcomeCompleted SessionOutcome = "completed" // Task completed in this session
	SessionOutcomePaused    SessionOutcome = "paused"    // Work paused, will resume later
	SessionOutcomeBlocked   SessionOutcome = "blocked"   // Work blocked by external dependency
)

// WorkSession represents a single work session on a task
type WorkSession struct {
	ID              int64           `json:"id" db:"id"`
	TaskID          int64           `json:"task_id" db:"task_id"`
	AgentID         *string         `json:"agent_id,omitempty" db:"agent_id"`
	StartedAt       time.Time       `json:"started_at" db:"started_at"`
	EndedAt         sql.NullTime    `json:"ended_at,omitempty" db:"ended_at"`
	Outcome         *SessionOutcome `json:"outcome,omitempty" db:"outcome"`
	SessionNotes    *string         `json:"session_notes,omitempty" db:"session_notes"`
	ContextSnapshot *string         `json:"context_snapshot,omitempty" db:"context_snapshot"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
}

// Validate validates the WorkSession fields
func (ws *WorkSession) Validate() error {
	if ws.TaskID == 0 {
		return ErrInvalidTaskID
	}
	if ws.StartedAt.IsZero() {
		return ErrInvalidTimestamp
	}
	if ws.Outcome != nil {
		if err := ValidateSessionOutcome(string(*ws.Outcome)); err != nil {
			return err
		}
	}
	return nil
}

// IsActive returns true if the session is still active (not ended)
func (ws *WorkSession) IsActive() bool {
	return !ws.EndedAt.Valid
}

// Duration returns the duration of the session
// For active sessions (EndedAt is NULL), returns duration from StartedAt to now
func (ws *WorkSession) Duration() time.Duration {
	if ws.EndedAt.Valid {
		return ws.EndedAt.Time.Sub(ws.StartedAt)
	}
	return time.Since(ws.StartedAt)
}
