package validation

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

// StatusValidator provides config-driven status validation
type StatusValidator struct {
	workflow *config.WorkflowConfig
}

// NewStatusValidator creates a new validator with the given workflow config.
// If workflow is nil, it uses the default workflow.
func NewStatusValidator(workflow *config.WorkflowConfig) *StatusValidator {
	if workflow == nil {
		workflow = config.DefaultWorkflow()
	}
	return &StatusValidator{
		workflow: workflow,
	}
}

// ValidateStatus checks if a status is defined in the workflow
func (v *StatusValidator) ValidateStatus(status string) error {
	if v.workflow == nil || v.workflow.StatusFlow == nil {
		return fmt.Errorf("workflow config not initialized")
	}

	// Check if status exists in the workflow
	_, exists := v.workflow.StatusFlow[status]
	if !exists {
		return fmt.Errorf("invalid task status: must be one of [%s]: got %q",
			v.getAllStatusesString(), status)
	}

	return nil
}

// ValidateTransition checks if a transition from one status to another is allowed
func (v *StatusValidator) ValidateTransition(fromStatus, toStatus string) error {
	if v.workflow == nil || v.workflow.StatusFlow == nil {
		return fmt.Errorf("workflow config not initialized")
	}

	// First validate both statuses exist
	if err := v.ValidateStatus(fromStatus); err != nil {
		return fmt.Errorf("invalid from status: %w", err)
	}
	if err := v.ValidateStatus(toStatus); err != nil {
		return fmt.Errorf("invalid to status: %w", err)
	}

	// Check if transition is allowed
	allowedTransitions := v.workflow.StatusFlow[fromStatus]
	for _, allowed := range allowedTransitions {
		if allowed == toStatus {
			return nil // Transition is valid
		}
	}

	return fmt.Errorf("invalid transition from %q to %q: allowed transitions from %q are [%s]",
		fromStatus, toStatus, fromStatus, strings.Join(allowedTransitions, ", "))
}

// CanTransition checks if a transition is allowed (returns bool instead of error)
func (v *StatusValidator) CanTransition(fromStatus, toStatus string) bool {
	return v.ValidateTransition(fromStatus, toStatus) == nil
}

// IsValidStatus checks if a status is defined (returns bool instead of error)
func (v *StatusValidator) IsValidStatus(status string) bool {
	return v.ValidateStatus(status) == nil
}

// GetAllStatuses returns all defined statuses in the workflow
func (v *StatusValidator) GetAllStatuses() []string {
	if v.workflow == nil || v.workflow.StatusFlow == nil {
		return []string{}
	}

	statuses := make([]string, 0, len(v.workflow.StatusFlow))
	for status := range v.workflow.StatusFlow {
		statuses = append(statuses, status)
	}
	sort.Strings(statuses)
	return statuses
}

// getAllStatusesString returns all statuses as a comma-separated string
func (v *StatusValidator) getAllStatusesString() string {
	return strings.Join(v.GetAllStatuses(), ", ")
}

// GetStartStatuses returns the statuses that tasks can be created with
func (v *StatusValidator) GetStartStatuses() []string {
	if v.workflow == nil || v.workflow.SpecialStatuses == nil {
		return []string{}
	}

	startStatuses := v.workflow.SpecialStatuses[config.StartStatusKey]
	return append([]string{}, startStatuses...) // Return copy
}

// GetCompleteStatuses returns the terminal statuses (completed/cancelled)
func (v *StatusValidator) GetCompleteStatuses() []string {
	if v.workflow == nil || v.workflow.SpecialStatuses == nil {
		return []string{}
	}

	completeStatuses := v.workflow.SpecialStatuses[config.CompleteStatusKey]
	return append([]string{}, completeStatuses...) // Return copy
}

// IsStartStatus checks if a status is a valid starting status
func (v *StatusValidator) IsStartStatus(status string) bool {
	startStatuses := v.GetStartStatuses()
	for _, s := range startStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// IsCompleteStatus checks if a status is a terminal/complete status
func (v *StatusValidator) IsCompleteStatus(status string) bool {
	completeStatuses := v.GetCompleteStatuses()
	for _, s := range completeStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// GetAllowedTransitions returns the list of statuses that can be transitioned to from the given status
func (v *StatusValidator) GetAllowedTransitions(fromStatus string) []string {
	if v.workflow == nil || v.workflow.StatusFlow == nil {
		return []string{}
	}

	transitions := v.workflow.StatusFlow[fromStatus]
	return append([]string{}, transitions...) // Return copy
}

// GetWorkflow returns the workflow config used by this validator
func (v *StatusValidator) GetWorkflow() *config.WorkflowConfig {
	return v.workflow
}
