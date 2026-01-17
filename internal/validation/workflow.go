package validation

import (
	"errors"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

// ErrReasonRequired is returned when a backward status transition is attempted without a reason
var ErrReasonRequired = errors.New("reason is required for backward status transitions")

// PhaseOrder defines the hierarchy of workflow phases
// Lower order = earlier in workflow, higher order = later in workflow
var PhaseOrder = map[string]int{
	"planning":     1,
	"development":  2,
	"review":       3,
	"qa":           4,
	"approval":     5,
	"done":         6,
	"any":          0,
	"cancelled":    0,
	"on_hold":      0,
	"blocked":      0,
	"paused":       0,
}

// IsBackwardTransition detects when a status transition moves backward in the workflow.
//
// A backward transition occurs when:
// - The new status has a lower phase order than the current status
// - Both statuses have defined phases (not "any" or special statuses)
// - Neither phase is "any" (0) or empty
//
// Special statuses (blocked, on_hold, cancelled, etc.) with phase="any" are never
// considered backward transitions, even if moving from a later phase. This allows
// these statuses to be used without triggering rejection reason requirements.
//
// Parameters:
//   - currentStatus: The task's current status
//   - newStatus: The status being transitioned to
//   - workflow: The workflow configuration containing phase metadata
//
// Returns:
//   - true if the transition is backward (e.g., review → development)
//   - false if the transition is forward, same-phase, or involves special statuses
func IsBackwardTransition(currentStatus, newStatus string, workflow *config.WorkflowConfig) bool {
	if workflow == nil {
		return false
	}

	// Get metadata for both statuses
	currentMeta, currentExists := workflow.GetStatusMetadata(currentStatus)
	newMeta, newExists := workflow.GetStatusMetadata(newStatus)

	// If either status doesn't exist in metadata, it's not a backward transition
	if !currentExists || !newExists {
		return false
	}

	// Get phase order for both statuses
	currentOrder, currentHasPhase := PhaseOrder[currentMeta.Phase]
	newOrder, newHasPhase := PhaseOrder[newMeta.Phase]

	// If either phase is not defined, assume it's "any" (0) which is special
	if !currentHasPhase {
		currentOrder = 0
	}
	if !newHasPhase {
		newOrder = 0
	}

	// Handle empty phase string as "any" (0)
	if currentMeta.Phase == "" {
		currentOrder = 0
	}
	if newMeta.Phase == "" {
		newOrder = 0
	}

	// Backward transition is when:
	// - new phase order < current phase order AND
	// - new phase order > 0 (not special/any) AND
	// - current phase order > 0 (not special/any)
	//
	// If either phase is 0 (special/any), it's not a backward rejection transition
	return newOrder < currentOrder && newOrder > 0 && currentOrder > 0
}

// ValidateReasonForStatusTransition validates that a reason is provided for backward status transitions.
//
// Rules:
// - Backward transitions (e.g., review → development) require a non-empty reason UNLESS:
//   - workflow.RequireRejectionReason is false (config option disabled), OR
//   - --force is true
// - Forward transitions (e.g., development → review) don't require a reason
// - Same-phase transitions (e.g., development → blocked) don't require a reason
// - If force is true, reason is not required
// - If status is empty, no validation is performed (no status change)
// - Default behavior: Require rejection reason (backward compatibility)
//   - If workflow is nil, defaults to requiring reason (true)
//   - Only disabled when explicitly set to false in config
//
// Parameters:
//   - newStatus: The new status being transitioned to (empty string means no change)
//   - currentStatus: The task's current status
//   - reason: The reason provided for the transition
//   - force: Whether to bypass validation
//   - workflow: The workflow configuration for phase metadata and RequireRejectionReason setting
//
// Returns:
// - nil if validation passes
// - ErrReasonRequired if a backward transition is attempted without a reason (and force is false and config requires it)
func ValidateReasonForStatusTransition(newStatus, currentStatus, reason string, force bool, workflow *config.WorkflowConfig) error {
	// If no status change is requested, no validation needed
	if newStatus == "" {
		return nil
	}

	// If force is true, bypass validation
	if force {
		return nil
	}

	// Check if this is a backward transition
	if IsBackwardTransition(currentStatus, newStatus, workflow) {
		// Check if rejection reasons are required by config
		// Default to true if workflow is nil (backward compatibility)
		requireReason := true
		if workflow != nil {
			requireReason = workflow.RequireRejectionReason
		}

		// Only require reason if config enables it
		if requireReason && reason == "" {
			return ErrReasonRequired
		}
	}

	return nil
}
