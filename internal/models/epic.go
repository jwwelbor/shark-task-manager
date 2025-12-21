package models

import "time"

// EpicStatus represents the status of an epic
type EpicStatus string

const (
	EpicStatusDraft     EpicStatus = "draft"
	EpicStatusActive    EpicStatus = "active"
	EpicStatusCompleted EpicStatus = "completed"
	EpicStatusArchived  EpicStatus = "archived"
)

// Priority represents priority level (used by Epic and other entities)
type Priority string

const (
	PriorityHigh   Priority = "high"
	PriorityMedium Priority = "medium"
	PriorityLow    Priority = "low"
)

// Epic represents a top-level project organization unit
type Epic struct {
	ID                int64      `json:"id" db:"id"`
	Key               string     `json:"key" db:"key"`
	Title             string     `json:"title" db:"title"`
	Description       *string    `json:"description,omitempty" db:"description"`
	Status            EpicStatus `json:"status" db:"status"`
	Priority          Priority   `json:"priority" db:"priority"`
	BusinessValue     *Priority  `json:"business_value,omitempty" db:"business_value"`
	FilePath          *string    `json:"file_path,omitempty" db:"file_path"`
	CustomFolderPath  *string    `json:"custom_folder_path,omitempty" db:"custom_folder_path"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
}

// Validate validates the Epic fields
func (e *Epic) Validate() error {
	if err := ValidateEpicKey(e.Key); err != nil {
		return err
	}
	if e.Title == "" {
		return ErrEmptyTitle
	}
	if err := ValidateEpicStatus(string(e.Status)); err != nil {
		return err
	}
	if err := ValidatePriority(string(e.Priority)); err != nil {
		return err
	}
	if e.BusinessValue != nil {
		if err := ValidatePriority(string(*e.BusinessValue)); err != nil {
			return err
		}
	}
	return nil
}
