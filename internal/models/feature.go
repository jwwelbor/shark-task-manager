package models

import "time"

// FeatureStatus represents the status of a feature
type FeatureStatus string

const (
	FeatureStatusDraft     FeatureStatus = "draft"
	FeatureStatusActive    FeatureStatus = "active"
	FeatureStatusCompleted FeatureStatus = "completed"
	FeatureStatusArchived  FeatureStatus = "archived"
)

// Feature represents a mid-level unit within an epic
type Feature struct {
	ID               int64         `json:"id" db:"id"`
	EpicID           int64         `json:"epic_id" db:"epic_id"`
	Key              string        `json:"key" db:"key"`
	Title            string        `json:"title" db:"title"`
	Description      *string       `json:"description,omitempty" db:"description"`
	Status           FeatureStatus `json:"status" db:"status"`
	ProgressPct      float64       `json:"progress_pct" db:"progress_pct"`
	ExecutionOrder   *int          `json:"execution_order,omitempty" db:"execution_order"`
	FilePath         *string       `json:"file_path,omitempty" db:"file_path"`
	CustomFolderPath *string       `json:"custom_folder_path,omitempty" db:"custom_folder_path"`
	CreatedAt        time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at" db:"updated_at"`
}

// Validate validates the Feature fields
func (f *Feature) Validate() error {
	if err := ValidateFeatureKey(f.Key); err != nil {
		return err
	}
	if f.Title == "" {
		return ErrEmptyTitle
	}
	if err := ValidateFeatureStatus(string(f.Status)); err != nil {
		return err
	}
	if f.ProgressPct < 0.0 || f.ProgressPct > 100.0 {
		return ErrInvalidProgressPct
	}
	return nil
}
