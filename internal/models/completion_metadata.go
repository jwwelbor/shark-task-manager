package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// VerificationStatus represents the verification state of a task
type VerificationStatus string

const (
	VerificationStatusPending     VerificationStatus = "pending"
	VerificationStatusVerified    VerificationStatus = "verified"
	VerificationStatusNeedsRework VerificationStatus = "needs_rework"
)

// CompletionMetadata contains rich metadata about task completion
type CompletionMetadata struct {
	CompletedBy        *string            `json:"completed_by,omitempty"`
	CompletionNotes    *string            `json:"completion_notes,omitempty"`
	FilesChanged       []string           `json:"files_changed,omitempty"`
	TestsPassed        bool               `json:"tests_passed"`
	VerificationStatus VerificationStatus `json:"verification_status"`
	TimeSpentMinutes   *int               `json:"time_spent_minutes,omitempty"`
	CompletedAt        *time.Time         `json:"completed_at,omitempty"`
}

// Validate validates the CompletionMetadata fields
func (cm *CompletionMetadata) Validate() error {
	// Validate verification status
	switch cm.VerificationStatus {
	case VerificationStatusPending, VerificationStatusVerified, VerificationStatusNeedsRework:
		// Valid status
	case "":
		// Default to pending
		cm.VerificationStatus = VerificationStatusPending
	default:
		return fmt.Errorf("invalid verification status: %s (must be pending, verified, or needs_rework)", cm.VerificationStatus)
	}

	// Validate time spent is non-negative
	if cm.TimeSpentMinutes != nil && *cm.TimeSpentMinutes < 0 {
		return fmt.Errorf("time_spent_minutes cannot be negative")
	}

	// Validate file paths are non-empty strings
	for i, file := range cm.FilesChanged {
		if file == "" {
			return fmt.Errorf("files_changed[%d] cannot be empty string", i)
		}
	}

	return nil
}

// ToJSON converts FilesChanged array to JSON string for storage
func (cm *CompletionMetadata) ToJSON() (string, error) {
	if len(cm.FilesChanged) == 0 {
		return "[]", nil
	}

	data, err := json.Marshal(cm.FilesChanged)
	if err != nil {
		return "", fmt.Errorf("failed to marshal files_changed: %w", err)
	}

	return string(data), nil
}

// FromJSON parses JSON string into FilesChanged array
func (cm *CompletionMetadata) FromJSON(jsonStr string) error {
	if jsonStr == "" || jsonStr == "null" {
		cm.FilesChanged = []string{}
		return nil
	}

	if err := json.Unmarshal([]byte(jsonStr), &cm.FilesChanged); err != nil {
		return fmt.Errorf("failed to unmarshal files_changed: %w", err)
	}

	return nil
}

// NewCompletionMetadata creates a new CompletionMetadata with default values
func NewCompletionMetadata() *CompletionMetadata {
	return &CompletionMetadata{
		FilesChanged:       []string{},
		TestsPassed:        false,
		VerificationStatus: VerificationStatusPending,
	}
}
