package models

import (
	"time"
)

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
	ID            int64                  `json:"id" db:"id"`
	Key           string                 `json:"key" db:"key"`
	Title         string                 `json:"title" db:"title"`
	Description   *string                `json:"description,omitempty" db:"description"`
	Status        EpicStatus             `json:"status" db:"status"`
	Priority      Priority               `json:"priority" db:"priority"`
	BusinessValue *Priority              `json:"business_value,omitempty" db:"business_value"`
	Slug          *string                `json:"slug,omitempty" db:"slug"`
	FilePath      *string                `json:"file_path,omitempty" db:"file_path"`
	ContextData   *string                `json:"context_data,omitempty" db:"context_data"`
	Metadata      map[string]interface{} `json:"metadata,omitempty" db:"-"` // Not persisted to DB, derived from related data
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at" db:"updated_at"`
}

// Entity interface implementation for Epic.

func (e *Epic) GetID() int64              { return e.ID }
func (e *Epic) GetKey() string            { return e.Key }
func (e *Epic) GetTitle() string          { return e.Title }
func (e *Epic) GetEntityType() EntityType { return EntityTypeEpic }
func (e *Epic) GetStatus() string         { return string(e.Status) }
func (e *Epic) SetStatus(status string)   { e.Status = EpicStatus(status) }
func (e *Epic) GetCreatedAt() time.Time   { return e.CreatedAt }
func (e *Epic) GetUpdatedAt() time.Time   { return e.UpdatedAt }

func (e *Epic) GetSlug() string {
	if e.Slug != nil {
		return *e.Slug
	}
	return ""
}

func (e *Epic) GetDescription() string {
	if e.Description != nil {
		return *e.Description
	}
	return ""
}

func (e *Epic) GetFilePath() string {
	if e.FilePath != nil {
		return *e.FilePath
	}
	return ""
}

func (e *Epic) GetContextData() *string     { return e.ContextData }
func (e *Epic) SetContextData(data *string) { e.ContextData = data }

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
