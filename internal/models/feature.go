package models

import (
	"time"
)

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
	ID             int64                  `json:"id" db:"id"`
	EpicID         int64                  `json:"epic_id" db:"epic_id"`
	Key            string                 `json:"key" db:"key"`
	Title          string                 `json:"title" db:"title"`
	Slug           *string                `json:"slug,omitempty" db:"slug"`
	Description    *string                `json:"description,omitempty" db:"description"`
	Status         FeatureStatus          `json:"status" db:"status"`
	StatusOverride bool                   `json:"status_override" db:"status_override"` // E07-F14: Manual override flag
	ProgressPct    float64                `json:"progress_pct" db:"progress_pct"`
	ExecutionOrder *int                   `json:"execution_order,omitempty" db:"execution_order"`
	FilePath       *string                `json:"file_path,omitempty" db:"file_path"`
	ContextData    *string                `json:"context_data,omitempty" db:"context_data"`
	Metadata       map[string]interface{} `json:"metadata,omitempty" db:"-"` // Not persisted to DB, derived from related data
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" db:"updated_at"`
}

// Entity interface implementation for Feature.

func (f *Feature) GetID() int64              { return f.ID }
func (f *Feature) GetKey() string            { return f.Key }
func (f *Feature) GetTitle() string          { return f.Title }
func (f *Feature) GetEntityType() EntityType { return EntityTypeFeature }
func (f *Feature) GetStatus() string         { return string(f.Status) }
func (f *Feature) SetStatus(status string)   { f.Status = FeatureStatus(status) }
func (f *Feature) GetCreatedAt() time.Time   { return f.CreatedAt }
func (f *Feature) GetUpdatedAt() time.Time   { return f.UpdatedAt }

func (f *Feature) GetSlug() string {
	if f.Slug != nil {
		return *f.Slug
	}
	return ""
}

func (f *Feature) GetDescription() string {
	if f.Description != nil {
		return *f.Description
	}
	return ""
}

func (f *Feature) GetFilePath() string {
	if f.FilePath != nil {
		return *f.FilePath
	}
	return ""
}

func (f *Feature) GetContextData() *string     { return f.ContextData }
func (f *Feature) SetContextData(data *string) { f.ContextData = data }

// IsAutoStatus returns true if status is automatically derived from tasks
func (f *Feature) IsAutoStatus() bool {
	return !f.StatusOverride
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
