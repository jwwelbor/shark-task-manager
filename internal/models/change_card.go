package models

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// changeCardKeyPattern matches CC-### format (CC- followed by exactly 3 digits).
var changeCardKeyPattern = regexp.MustCompile(`^CC-\d{3}$`)

// ErrInvalidChangeCardKey is returned when a change-card key does not match the expected format.
var ErrInvalidChangeCardKey = errors.New("invalid change-card key format: must match CC-### (e.g., CC-001, CC-042)")

// ValidateChangeCardKey validates the change-card key format (CC-### where ### is 3 digits).
func ValidateChangeCardKey(key string) error {
	if key == "" {
		return ErrEmptyKey
	}
	if !changeCardKeyPattern.MatchString(key) {
		return fmt.Errorf("%w: got %q", ErrInvalidChangeCardKey, key)
	}
	return nil
}

// ChangeCardStatus represents the workflow status of a change-card.
type ChangeCardStatus string

// ChangeCard represents a lightweight enhancement proposal or change request.
type ChangeCard struct {
	ID             int64            `json:"id" db:"id"`
	Key            string           `json:"key" db:"key"`
	Title          string           `json:"title" db:"title"`
	Description    *string          `json:"description,omitempty" db:"description"`
	Status         ChangeCardStatus `json:"status" db:"status"`
	Priority       int              `json:"priority" db:"priority"`
	RequestedBy    *string          `json:"requested_by,omitempty" db:"requested_by"`
	AssignedTo     *string          `json:"assigned_to,omitempty" db:"assigned_to"`
	EpicID         *int64           `json:"epic_id,omitempty" db:"epic_id"`
	FeatureID      *int64           `json:"feature_id,omitempty" db:"feature_id"`
	RelatedTaskID  *int64           `json:"related_task_id,omitempty" db:"related_task_id"`
	Justification  *string          `json:"justification,omitempty" db:"justification"`
	ImpactAnalysis *string          `json:"impact_analysis,omitempty" db:"impact_analysis"`
	RollbackPlan   *string          `json:"rollback_plan,omitempty" db:"rollback_plan"`
	Slug           *string          `json:"slug,omitempty" db:"slug"`
	FilePath       *string          `json:"file_path,omitempty" db:"file_path"`
	ContextData    *string          `json:"context_data,omitempty" db:"context_data"` // JSON
	CreatedAt      time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at" db:"updated_at"`
}

// Entity interface implementation for ChangeCard.

func (c *ChangeCard) GetID() int64              { return c.ID }
func (c *ChangeCard) GetKey() string            { return c.Key }
func (c *ChangeCard) GetTitle() string          { return c.Title }
func (c *ChangeCard) GetEntityType() EntityType { return EntityTypeChange }
func (c *ChangeCard) GetStatus() string         { return string(c.Status) }
func (c *ChangeCard) SetStatus(status string)   { c.Status = ChangeCardStatus(status) }
func (c *ChangeCard) GetCreatedAt() time.Time   { return c.CreatedAt }
func (c *ChangeCard) GetUpdatedAt() time.Time   { return c.UpdatedAt }

func (c *ChangeCard) GetSlug() string {
	if c.Slug != nil {
		return *c.Slug
	}
	return ""
}

func (c *ChangeCard) GetDescription() string {
	if c.Description != nil {
		return *c.Description
	}
	return ""
}

func (c *ChangeCard) GetFilePath() string {
	if c.FilePath != nil {
		return *c.FilePath
	}
	return ""
}

func (c *ChangeCard) GetContextData() *string     { return c.ContextData }
func (c *ChangeCard) SetContextData(data *string) { c.ContextData = data }

// Validate performs structural validation only.
// Does NOT validate status against workflow -- that is the service layer's responsibility.
func (c *ChangeCard) Validate() error {
	if strings.TrimSpace(c.Title) == "" {
		return fmt.Errorf("change-card title cannot be empty")
	}
	if strings.TrimSpace(string(c.Status)) == "" {
		return fmt.Errorf("change-card status cannot be empty")
	}
	// Validate key format if key is set (key may be empty before assignment)
	if c.Key != "" {
		if err := ValidateChangeCardKey(c.Key); err != nil {
			return err
		}
	}
	return nil
}
