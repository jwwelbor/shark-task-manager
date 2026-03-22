package models

import (
	"errors"
	"fmt"
	"time"
)

// EntityHistory represents a status change audit trail entry for any entity type.
// It generalizes TaskHistory to support Epic, Feature, Task, Bug, and ChangeCard
// entities using polymorphic EntityType + EntityID fields.
//
// Unlike TaskHistory, EntityHistory does NOT validate status values in the model layer.
// Status value validation belongs in the service layer via workflow.Service.
type EntityHistory struct {
	ID              int64      `json:"id" db:"id"`
	EntityType      EntityType `json:"entity_type" db:"entity_type"`
	EntityID        int64      `json:"entity_id" db:"entity_id"`
	FromStatus      *string    `json:"from_status,omitempty" db:"from_status"`
	ToStatus        string     `json:"to_status" db:"to_status"`
	ChangedBy       *string    `json:"changed_by,omitempty" db:"changed_by"`
	Notes           *string    `json:"notes,omitempty" db:"notes"`
	Forced          bool       `json:"forced" db:"forced"`
	RejectionReason *string    `json:"rejection_reason,omitempty" db:"rejection_reason"`
	ChangedAt       time.Time  `json:"changed_at" db:"changed_at"`
}

// Validate validates the EntityHistory fields (structural validation only).
// It checks that required fields are present and valid but does NOT validate
// that status values are valid workflow statuses. Status validation is the
// service layer's responsibility via workflow.Service.
func (h *EntityHistory) Validate() error {
	if h.EntityType == "" {
		return errors.New("entity_type cannot be empty")
	}
	if !ValidEntityTypes[h.EntityType] {
		return fmt.Errorf("invalid entity_type: %s", h.EntityType)
	}
	if h.EntityID <= 0 {
		return errors.New("entity_id must be positive")
	}
	if h.ToStatus == "" {
		return errors.New("to_status cannot be empty")
	}
	return nil
}
