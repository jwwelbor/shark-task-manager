package models

import (
	"time"
)

// CriteriaStatus represents the status of a task criterion
type CriteriaStatus string

const (
	CriteriaStatusPending    CriteriaStatus = "pending"     // Not yet verified
	CriteriaStatusInProgress CriteriaStatus = "in_progress" // Currently being worked on
	CriteriaStatusComplete   CriteriaStatus = "complete"    // Successfully verified
	CriteriaStatusFailed     CriteriaStatus = "failed"      // Verification failed
	CriteriaStatusNA         CriteriaStatus = "na"          // Not applicable
)

// TaskCriteria represents an acceptance criterion for a task
type TaskCriteria struct {
	ID                int64          `json:"id" db:"id"`
	TaskID            int64          `json:"task_id" db:"task_id"`
	Criterion         string         `json:"criterion" db:"criterion"`
	Status            CriteriaStatus `json:"status" db:"status"`
	VerifiedAt        *time.Time     `json:"verified_at,omitempty" db:"verified_at"`
	VerificationNotes *string        `json:"verification_notes,omitempty" db:"verification_notes"`
	CreatedAt         time.Time      `json:"created_at" db:"created_at"`
}

// Validate validates the TaskCriteria fields
func (tc *TaskCriteria) Validate() error {
	if tc.TaskID == 0 {
		return ErrInvalidTaskID
	}
	if tc.Criterion == "" {
		return ErrEmptyCriterion
	}
	if err := ValidateCriteriaStatus(string(tc.Status)); err != nil {
		return err
	}
	return nil
}
