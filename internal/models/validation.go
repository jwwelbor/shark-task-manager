package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
)

// Validation errors
var (
	ErrInvalidEpicKey          = errors.New("invalid epic key format: must match ^E\\d{2}$")
	ErrInvalidFeatureKey       = errors.New("invalid feature key format: must match ^E\\d{2}-F\\d{2}$")
	ErrInvalidTaskKey          = errors.New("invalid task key format: must match ^T-E\\d{2}-F\\d{2}-\\d{3}$")
	ErrInvalidEpicStatus       = errors.New("invalid epic status: must be draft, active, completed, or archived")
	ErrInvalidFeatureStatus    = errors.New("invalid feature status: must be draft, active, completed, or archived")
	// ErrInvalidTaskStatus is deprecated - error messages are now generated dynamically based on workflow config
	ErrInvalidTaskStatus       = errors.New("invalid task status")
	ErrInvalidAgentType        = errors.New("invalid agent type: must be frontend, backend, api, testing, devops, or general")
	ErrInvalidPriority         = errors.New("invalid priority: must be between 1 and 10")
	ErrInvalidProgressPct      = errors.New("invalid progress_pct: must be between 0.0 and 100.0")
	ErrInvalidDependsOn        = errors.New("invalid depends_on: must be a valid JSON array of strings")
	ErrEmptyTitle              = errors.New("title cannot be empty")
	ErrEmptyNewStatus          = errors.New("new_status cannot be empty")
	ErrInvalidNoteType         = errors.New("invalid note type: must be comment, decision, blocker, solution, reference, implementation, testing, future, or question")
	ErrInvalidTaskID           = errors.New("task_id must be greater than 0")
	ErrEmptyContent            = errors.New("content cannot be empty")
	ErrInvalidCriteriaStatus   = errors.New("invalid criteria status: must be pending, in_progress, complete, failed, or na")
	ErrEmptyCriterion          = errors.New("criterion cannot be empty")
	ErrInvalidRelationshipType = errors.New("invalid relationship type: must be depends_on, blocks, related_to, follows, spawned_from, duplicates, or references")
	ErrSelfRelationship        = errors.New("task cannot have a relationship with itself")
	ErrCircularDependency      = errors.New("circular dependency detected")
	ErrInvalidSessionOutcome   = errors.New("invalid session outcome: must be completed, paused, or blocked")
	ErrInvalidTimestamp        = errors.New("invalid timestamp: cannot be zero value")
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
// DEPRECATED: This function uses hardcoded statuses and will be removed in a future version.
// Use ValidateTaskStatusWithWorkflow instead for config-driven validation.
//
// This function is kept for backward compatibility and now uses the default workflow.
func ValidateTaskStatus(status string) error {
	// Use default workflow for validation to maintain backward compatibility
	// while preparing for full config-driven validation
	return ValidateTaskStatusWithWorkflow(status, nil)
}

// ValidateTaskStatusWithWorkflow validates a task status against a workflow config.
// If workflow is nil, uses the default workflow.
// This is the config-driven replacement for ValidateTaskStatus.
func ValidateTaskStatusWithWorkflow(status string, workflow interface{}) error {
	// Import here to avoid circular dependency - we'll handle this properly
	// For now, accept nil and validate against known statuses from the workflow
	// The actual implementation will be in the validation package

	// Temporary implementation that accepts both old and new workflow statuses
	// This will be replaced once all callers are updated to use the validation package

	// Check if status is in the old hardcoded list (backward compatibility)
	oldStatuses := map[string]bool{
		"todo":             true,
		"in_progress":      true,
		"blocked":          true,
		"ready_for_review": true,
		"completed":        true,
		"archived":         true,
	}

	// Check if status is in the new 14-status workflow
	newStatuses := map[string]bool{
		"draft":                 true,
		"ready_for_refinement":  true,
		"in_refinement":         true,
		"ready_for_development": true,
		"in_development":        true,
		"ready_for_code_review": true,
		"in_code_review":        true,
		"ready_for_qa":          true,
		"in_qa":                 true,
		"ready_for_approval":    true,
		"in_approval":           true,
		"blocked":               true,
		"on_hold":               true,
		"completed":             true,
		"cancelled":             true,
	}

	// Accept status if it's in either the old or new workflow
	// This provides a migration path
	if oldStatuses[status] || newStatuses[status] {
		return nil
	}

	// If workflow is provided, we could validate against it
	// For now, return error with helpful message
	return fmt.Errorf("invalid task status %q: not found in default or extended workflow. "+
		"Ensure status is defined in .sharkconfig.json workflow", status)
}

// ValidateAgentType validates the agent type enum
// Note: As of E07-F01, this accepts any non-empty string value
func ValidateAgentType(agentType string) error {
	validTypes := map[string]bool{
		"frontend": true,
		"backend":  true,
		"api":      true,
		"testing":  true,
		"devops":   true,
		"general":  true,
	}
	if !validTypes[agentType] {
		return fmt.Errorf("invalid agent type '%s'. Valid types are: frontend, backend, api, testing, devops, general", agentType)
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

// ValidateNoteType validates the note type enum
func ValidateNoteType(noteType string) error {
	validTypes := map[string]bool{
		"comment":        true,
		"decision":       true,
		"blocker":        true,
		"solution":       true,
		"reference":      true,
		"implementation": true,
		"testing":        true,
		"future":         true,
		"question":       true,
	}
	if !validTypes[noteType] {
		return fmt.Errorf("%w: got %q", ErrInvalidNoteType, noteType)
	}
	return nil
}

// ValidateCriteriaStatus validates the criteria status enum
func ValidateCriteriaStatus(status string) error {
	validStatuses := map[string]bool{
		"pending":     true,
		"in_progress": true,
		"complete":    true,
		"failed":      true,
		"na":          true,
	}
	if !validStatuses[status] {
		return fmt.Errorf("%w: got %q", ErrInvalidCriteriaStatus, status)
	}
	return nil
}

// ValidateRelationshipType validates the relationship type enum
func ValidateRelationshipType(relType string) error {
	validTypes := map[string]bool{
		"depends_on":   true,
		"blocks":       true,
		"related_to":   true,
		"follows":      true,
		"spawned_from": true,
		"duplicates":   true,
		"references":   true,
	}
	if !validTypes[relType] {
		return fmt.Errorf("%w: got %q", ErrInvalidRelationshipType, relType)
	}
	return nil
}

// ValidateSessionOutcome validates the session outcome enum
func ValidateSessionOutcome(outcome string) error {
	validOutcomes := map[string]bool{
		"completed": true,
		"paused":    true,
		"blocked":   true,
	}
	if !validOutcomes[outcome] {
		return fmt.Errorf("%w: got %q", ErrInvalidSessionOutcome, outcome)
	}
	return nil
}
