package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
)

// Validation errors
var (
	ErrInvalidEpicKey      = errors.New("invalid epic key format: must match ^E\\d{2}$")
	ErrInvalidFeatureKey   = errors.New("invalid feature key format: must match ^E\\d{2}-F\\d{2}$")
	ErrInvalidTaskKey      = errors.New("invalid task key format: must match ^T-E\\d{2}-F\\d{2}-\\d{3}$")
	ErrInvalidEpicStatus   = errors.New("invalid epic status: must be draft, active, completed, or archived")
	ErrInvalidFeatureStatus = errors.New("invalid feature status: must be draft, active, completed, or archived")
	ErrInvalidTaskStatus   = errors.New("invalid task status: must be todo, in_progress, blocked, ready_for_review, completed, or archived")
	ErrInvalidAgentType    = errors.New("invalid agent type: must be frontend, backend, api, testing, devops, or general")
	ErrInvalidPriority     = errors.New("invalid priority: must be between 1 and 10")
	ErrInvalidProgressPct  = errors.New("invalid progress_pct: must be between 0.0 and 100.0")
	ErrInvalidDependsOn    = errors.New("invalid depends_on: must be a valid JSON array of strings")
	ErrEmptyTitle          = errors.New("title cannot be empty")
	ErrEmptyNewStatus      = errors.New("new_status cannot be empty")
)

// Key format regex patterns
var (
	epicKeyPattern    = regexp.MustCompile(`^E\d{2}$`)
	featureKeyPattern = regexp.MustCompile(`^E\d{2}-F\d{2}$`)
	taskKeyPattern    = regexp.MustCompile(`^T-E\d{2}-F\d{2}-\d{3}$`)
)

// ValidateEpicKey validates the epic key format
func ValidateEpicKey(key string) error {
	if !epicKeyPattern.MatchString(key) {
		return fmt.Errorf("%w: got %q", ErrInvalidEpicKey, key)
	}
	return nil
}

// ValidateFeatureKey validates the feature key format
func ValidateFeatureKey(key string) error {
	if !featureKeyPattern.MatchString(key) {
		return fmt.Errorf("%w: got %q", ErrInvalidFeatureKey, key)
	}
	return nil
}

// ValidateTaskKey validates the task key format
func ValidateTaskKey(key string) error {
	if !taskKeyPattern.MatchString(key) {
		return fmt.Errorf("%w: got %q", ErrInvalidTaskKey, key)
	}
	return nil
}

// ValidateEpicStatus validates the epic status enum
func ValidateEpicStatus(status string) error {
	validStatuses := map[string]bool{
		"draft":     true,
		"active":    true,
		"completed": true,
		"archived":  true,
	}
	if !validStatuses[status] {
		return fmt.Errorf("%w: got %q", ErrInvalidEpicStatus, status)
	}
	return nil
}

// ValidateFeatureStatus validates the feature status enum
func ValidateFeatureStatus(status string) error {
	validStatuses := map[string]bool{
		"draft":     true,
		"active":    true,
		"completed": true,
		"archived":  true,
	}
	if !validStatuses[status] {
		return fmt.Errorf("%w: got %q", ErrInvalidFeatureStatus, status)
	}
	return nil
}

// ValidateTaskStatus validates the task status enum
func ValidateTaskStatus(status string) error {
	validStatuses := map[string]bool{
		"todo":             true,
		"in_progress":      true,
		"blocked":          true,
		"ready_for_review": true,
		"completed":        true,
		"archived":         true,
	}
	if !validStatuses[status] {
		return fmt.Errorf("%w: got %q", ErrInvalidTaskStatus, status)
	}
	return nil
}

// ValidateAgentType validates the agent type enum
// Note: As of E07-F01, this accepts any non-empty string value
func ValidateAgentType(agentType string) error {
	// Accept any non-empty string value for agent type
	// This allows for custom agent types beyond the predefined set
	if agentType == "" {
		return fmt.Errorf("agent type cannot be empty")
	}
	return nil
}

// ValidatePriority validates the priority level (for Epic and other entities)
func ValidatePriority(priority string) error {
	validPriorities := map[string]bool{
		"high":   true,
		"medium": true,
		"low":    true,
	}
	if !validPriorities[priority] {
		return fmt.Errorf("invalid priority: must be high, medium, or low, got %q", priority)
	}
	return nil
}

// ValidateDependsOn validates the JSON format of the depends_on field
func ValidateDependsOn(dependsOn string) error {
	if dependsOn == "" || dependsOn == "null" {
		return nil // Empty or null is valid
	}

	var deps []string
	if err := json.Unmarshal([]byte(dependsOn), &deps); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidDependsOn, err)
	}

	// Optionally validate each dependency is a valid task key
	for _, dep := range deps {
		if err := ValidateTaskKey(dep); err != nil {
			return fmt.Errorf("invalid task key in depends_on: %w", err)
		}
	}

	return nil
}
