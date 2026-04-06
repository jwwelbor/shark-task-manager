package models

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// TechDebtStatus represents the workflow status of a tech-debt item.
// Valid values are determined by the workflow configuration, not hardcoded.
type TechDebtStatus string

// TechDebtCategory represents the category of technical debt.
type TechDebtCategory string

const (
	TechDebtCategoryCodeQuality   TechDebtCategory = "code-quality"
	TechDebtCategoryArchitecture  TechDebtCategory = "architecture"
	TechDebtCategoryDependency    TechDebtCategory = "dependency"
	TechDebtCategoryTesting       TechDebtCategory = "testing"
	TechDebtCategoryPerformance   TechDebtCategory = "performance"
	TechDebtCategoryDocumentation TechDebtCategory = "documentation"
)

// ValidTechDebtCategories is the set of valid category values.
var ValidTechDebtCategories = map[TechDebtCategory]bool{
	TechDebtCategoryCodeQuality:   true,
	TechDebtCategoryArchitecture:  true,
	TechDebtCategoryDependency:    true,
	TechDebtCategoryTesting:       true,
	TechDebtCategoryPerformance:   true,
	TechDebtCategoryDocumentation: true,
}

// TechDebtSeverity represents the severity level of a tech-debt item.
type TechDebtSeverity string

const (
	TechDebtSeverityCritical TechDebtSeverity = "critical"
	TechDebtSeverityHigh     TechDebtSeverity = "high"
	TechDebtSeverityMedium   TechDebtSeverity = "medium"
	TechDebtSeverityLow      TechDebtSeverity = "low"
)

// ValidTechDebtSeverities is the set of valid severity values.
var ValidTechDebtSeverities = map[TechDebtSeverity]bool{
	TechDebtSeverityCritical: true,
	TechDebtSeverityHigh:     true,
	TechDebtSeverityMedium:   true,
	TechDebtSeverityLow:      true,
}

// TechDebt represents a technical debt item entity.
type TechDebt struct {
	BaseEntity                      // 9 shared fields + 10 accessor methods
	Status         TechDebtStatus   `json:"status" db:"status"`
	Category       TechDebtCategory `json:"category" db:"category"`
	Severity       TechDebtSeverity `json:"severity" db:"severity"`
	EffortEstimate *string          `json:"effort_estimate,omitempty" db:"effort_estimate"`
}

// techDebtKeyPattern matches TD- followed by exactly 3 digits.
var techDebtKeyPattern = regexp.MustCompile(`^TD-\d{3}$`)

// ErrInvalidTechDebtKey is returned when a tech-debt key does not match the expected format.
var ErrInvalidTechDebtKey = errors.New("invalid tech-debt key format: must match TD-### (e.g., TD-001, TD-042)")

// Entity interface implementation for TechDebt.

func (td *TechDebt) GetEntityType() EntityType { return EntityTypeTechDebt }
func (td *TechDebt) GetStatus() string         { return string(td.Status) }
func (td *TechDebt) SetStatus(status string)   { td.Status = TechDebtStatus(status) }

// Validate performs structural validation on the TechDebt model.
// It does NOT check workflow status validity (that is the service layer's job).
func (td *TechDebt) Validate() error {
	if err := ValidateTechDebtKey(td.Key); err != nil {
		return err
	}
	if strings.TrimSpace(td.Title) == "" {
		return ErrEmptyTitle
	}
	if strings.TrimSpace(string(td.Status)) == "" {
		return errors.New("tech-debt status cannot be empty")
	}
	if !ValidTechDebtCategories[td.Category] {
		return fmt.Errorf("invalid category %q: must be one of code-quality, architecture, dependency, testing, performance, documentation", td.Category)
	}
	if !ValidTechDebtSeverities[td.Severity] {
		return fmt.Errorf("invalid severity %q: must be one of critical, high, medium, low", td.Severity)
	}
	return nil
}

// ValidateTechDebtKey validates the tech-debt key format (TD-### where ### is 3 digits).
func ValidateTechDebtKey(key string) error {
	if key == "" {
		return ErrEmptyKey
	}
	if !techDebtKeyPattern.MatchString(key) {
		return fmt.Errorf("%w: got %q", ErrInvalidTechDebtKey, key)
	}
	return nil
}
