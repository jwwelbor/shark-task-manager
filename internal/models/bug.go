package models

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// BugStatus represents the workflow status of a bug.
// Valid values are determined by the workflow configuration, not hardcoded.
type BugStatus string

// BugSeverity represents the severity level of a bug.
type BugSeverity string

const (
	BugSeverityCritical BugSeverity = "critical"
	BugSeverityHigh     BugSeverity = "high"
	BugSeverityMedium   BugSeverity = "medium"
	BugSeverityLow      BugSeverity = "low"
)

// ValidBugSeverities is the set of valid severity values.
var ValidBugSeverities = map[BugSeverity]bool{
	BugSeverityCritical: true,
	BugSeverityHigh:     true,
	BugSeverityMedium:   true,
	BugSeverityLow:      true,
}

// Bug represents a bug report entity.
type Bug struct {
	BaseEntity             // 9 shared fields + 10 accessor methods
	Status     BugStatus   `json:"status" db:"status"`
	Severity   BugSeverity `json:"severity" db:"severity"`
	// LEGACY: LinkedEntityType is a legacy field for direct entity linking.
	// Migrate to entity_relationships table via EntityRelationshipService.
	// This field will be removed once all callers are migrated.
	LinkedEntityType *string `json:"linked_entity_type,omitempty" db:"linked_entity_type"` // "epic", "feature", "task"
	// LEGACY: LinkedEntityKey is a legacy field for direct entity linking.
	// Migrate to entity_relationships table via EntityRelationshipService.
	// This field will be removed once all callers are migrated.
	LinkedEntityKey *string `json:"linked_entity_key,omitempty" db:"linked_entity_key"`
}

// bugKeyPattern matches B followed by exactly 3 digits.
var bugKeyPattern = regexp.MustCompile(`^B\d{3}$`)

// ErrInvalidBugKey is returned when a bug key does not match the expected format.
var ErrInvalidBugKey = errors.New("invalid bug key format: must match B### (e.g., B001, B042)")

// Entity interface implementation for Bug.

func (b *Bug) GetEntityType() EntityType { return EntityTypeBug }
func (b *Bug) GetStatus() string         { return string(b.Status) }
func (b *Bug) SetStatus(status string)   { b.Status = BugStatus(status) }

// Validate performs structural validation on the Bug model.
// It does NOT check workflow status validity (that is the service layer's job).
func (b *Bug) Validate() error {
	if err := ValidateBugKey(b.Key); err != nil {
		return err
	}
	if strings.TrimSpace(b.Title) == "" {
		return ErrEmptyTitle
	}
	if strings.TrimSpace(string(b.Status)) == "" {
		return errors.New("bug status cannot be empty")
	}
	if !ValidBugSeverities[b.Severity] {
		return fmt.Errorf("invalid severity %q: must be one of critical, high, medium, low", b.Severity)
	}
	return nil
}

// ValidateBugKey validates the bug key format (B### where ### is 3 digits).
func ValidateBugKey(key string) error {
	if key == "" {
		return ErrEmptyKey
	}
	if !bugKeyPattern.MatchString(key) {
		return fmt.Errorf("%w: got %q", ErrInvalidBugKey, key)
	}
	return nil
}
