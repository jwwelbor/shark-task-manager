package models

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
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
	BaseEntity                   // 9 shared fields + 10 accessor methods
	Status      ChangeCardStatus `json:"status" db:"status"`
	Priority    int              `json:"priority" db:"priority"`
	RequestedBy *string          `json:"requested_by,omitempty" db:"requested_by"`
	AssignedTo  *string          `json:"assigned_to,omitempty" db:"assigned_to"`
	// LEGACY: EpicID is a legacy field for direct entity linking.
	// Migrate to entity_relationships table via EntityRelationshipService.
	// This field will be removed once all callers are migrated.
	EpicID *int64 `json:"epic_id,omitempty" db:"epic_id"`
	// LEGACY: FeatureID is a legacy field for direct entity linking.
	// Migrate to entity_relationships table via EntityRelationshipService.
	// This field will be removed once all callers are migrated.
	FeatureID *int64 `json:"feature_id,omitempty" db:"feature_id"`
	// LEGACY: RelatedTaskID is a legacy field for direct entity linking.
	// Migrate to entity_relationships table via EntityRelationshipService.
	// This field will be removed once all callers are migrated.
	RelatedTaskID  *int64  `json:"related_task_id,omitempty" db:"related_task_id"`
	Justification  *string `json:"justification,omitempty" db:"justification"`
	ImpactAnalysis *string `json:"impact_analysis,omitempty" db:"impact_analysis"`
	RollbackPlan   *string `json:"rollback_plan,omitempty" db:"rollback_plan"`
}

// Entity interface implementation for ChangeCard.

func (c *ChangeCard) GetEntityType() EntityType { return EntityTypeChange }
func (c *ChangeCard) GetStatus() string         { return string(c.Status) }
func (c *ChangeCard) SetStatus(status string)   { c.Status = ChangeCardStatus(status) }

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
