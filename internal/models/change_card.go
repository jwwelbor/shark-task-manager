package models

import (
	"fmt"
	"strings"
	"time"
)

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
	Slug           string           `json:"slug,omitempty" db:"slug"`
	FilePath       string           `json:"file_path,omitempty" db:"file_path"`
	ContextData    *string          `json:"context_data,omitempty" db:"context_data"` // JSON
	CreatedAt      time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at" db:"updated_at"`
}

// Validate performs structural validation only.
// Does NOT validate status against workflow -- that is the service layer's responsibility.
func (c *ChangeCard) Validate() error {
	if strings.TrimSpace(c.Title) == "" {
		return fmt.Errorf("change-card title cannot be empty")
	}
	if strings.TrimSpace(string(c.Status)) == "" {
		return fmt.Errorf("change-card status cannot be empty")
	}
	return nil
}
