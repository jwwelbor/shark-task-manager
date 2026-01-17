package models

import "time"

// TaskHistory represents an audit trail entry for task status changes
type TaskHistory struct {
	ID                int64     `json:"id" db:"id"`
	TaskID            int64     `json:"task_id" db:"task_id"`
	OldStatus         *string   `json:"old_status,omitempty" db:"old_status"`
	NewStatus         string    `json:"new_status" db:"new_status"`
	Agent             *string   `json:"agent,omitempty" db:"agent"`
	Notes             *string   `json:"notes,omitempty" db:"notes"`
	RejectionReason   *string   `json:"rejection_reason,omitempty" db:"rejection_reason"`
	Timestamp         time.Time `json:"timestamp" db:"timestamp"`
}

// Validate validates the TaskHistory fields
func (th *TaskHistory) Validate() error {
	if th.NewStatus == "" {
		return ErrEmptyNewStatus
	}
	// Validate that new_status is a valid TaskStatus
	if err := ValidateTaskStatus(th.NewStatus); err != nil {
		return err
	}
	// Validate old_status if provided
	if th.OldStatus != nil && *th.OldStatus != "" {
		if err := ValidateTaskStatus(*th.OldStatus); err != nil {
			return err
		}
	}
	return nil
}
