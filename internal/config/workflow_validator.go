package config

import (
	"fmt"
	"strings"
)

// ValidationError represents a workflow configuration validation error
type ValidationError struct {
	Message string
	Fix     string // Suggested fix for the error
}

func (e *ValidationError) Error() string {
	if e.Fix != "" {
		return fmt.Sprintf("%s. Fix: %s", e.Message, e.Fix)
	}
	return e.Message
}

// ValidateWorkflow validates workflow configuration for correctness
//
// Validation rules (REQ-F-002):
// 1. Required special statuses (_start_, _complete_) must be defined
// 2. All status references in transitions must be defined
// 3. All statuses must be reachable from _start_ statuses
// 4. All statuses must have a path to _complete_ statuses
// 5. No circular references with no terminal path
//
// Returns:
// - nil if workflow is valid
// - ValidationError with actionable message and fix suggestion if invalid
func ValidateWorkflow(workflow *WorkflowConfig) error {
	if workflow == nil {
		return &ValidationError{
			Message: "workflow config is nil",
			Fix:     "provide a valid workflow configuration",
		}
	}

	// Rule 1: Check for required special statuses
	if err := validateSpecialStatuses(workflow); err != nil {
		return err
	}

	// Rule 2: Check all status references are defined
	if err := validateStatusReferences(workflow); err != nil {
		return err
	}

	// Rule 3: Check all statuses are reachable from _start_
	if err := validateReachability(workflow); err != nil {
		return err
	}

	// Rule 4: Check all statuses have path to _complete_
	if err := validateTerminalPaths(workflow); err != nil {
		return err
	}

	return nil
}

// validateSpecialStatuses checks that _start_ and _complete_ are defined
func validateSpecialStatuses(workflow *WorkflowConfig) error {
	startStatuses, hasStart := workflow.SpecialStatuses[StartStatusKey]
	if !hasStart || len(startStatuses) == 0 {
		return &ValidationError{
			Message: fmt.Sprintf("missing required special status '%s'", StartStatusKey),
			Fix:     fmt.Sprintf("add 'special_statuses.%s' array with at least one initial status (e.g., ['todo'])", StartStatusKey),
		}
	}

	completeStatuses, hasComplete := workflow.SpecialStatuses[CompleteStatusKey]
	if !hasComplete || len(completeStatuses) == 0 {
		return &ValidationError{
			Message: fmt.Sprintf("missing required special status '%s'", CompleteStatusKey),
			Fix:     fmt.Sprintf("add 'special_statuses.%s' array with at least one terminal status (e.g., ['completed'])", CompleteStatusKey),
		}
	}

	// Verify start statuses exist in status flow
	for _, status := range startStatuses {
		if _, exists := workflow.StatusFlow[status]; !exists {
			return &ValidationError{
				Message: fmt.Sprintf("start status '%s' is not defined in status_flow", status),
				Fix:     fmt.Sprintf("add '%s' to status_flow map or remove from %s array", status, StartStatusKey),
			}
		}
	}

	// Verify complete statuses exist in status flow
	for _, status := range completeStatuses {
		if _, exists := workflow.StatusFlow[status]; !exists {
			return &ValidationError{
				Message: fmt.Sprintf("complete status '%s' is not defined in status_flow", status),
				Fix:     fmt.Sprintf("add '%s' to status_flow map or remove from %s array", status, CompleteStatusKey),
			}
		}
	}

	return nil
}

// validateStatusReferences checks that all status references in transitions are defined
func validateStatusReferences(workflow *WorkflowConfig) error {
	// Collect all defined statuses
	definedStatuses := make(map[string]bool)
	for status := range workflow.StatusFlow {
		definedStatuses[status] = true
	}

	// Check all transitions reference defined statuses
	var undefinedStatuses []string
	for fromStatus, transitions := range workflow.StatusFlow {
		for _, toStatus := range transitions {
			if !definedStatuses[toStatus] {
				undefinedStatuses = append(undefinedStatuses, fmt.Sprintf("%s → %s", fromStatus, toStatus))
			}
		}
	}

	if len(undefinedStatuses) > 0 {
		return &ValidationError{
			Message: fmt.Sprintf("undefined status references in transitions: %s", strings.Join(undefinedStatuses, ", ")),
			Fix:     "add missing statuses to status_flow map or remove invalid transition references",
		}
	}

	return nil
}

// validateReachability checks that all statuses are reachable from _start_ statuses
// Uses breadth-first search (BFS) to traverse the workflow graph
func validateReachability(workflow *WorkflowConfig) error {
	startStatuses := workflow.SpecialStatuses[StartStatusKey]

	// BFS to find all reachable statuses
	reachable := make(map[string]bool)
	queue := make([]string, len(startStatuses))
	copy(queue, startStatuses)

	for _, status := range startStatuses {
		reachable[status] = true
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Add all transitions from current status
		for _, next := range workflow.StatusFlow[current] {
			if !reachable[next] {
				reachable[next] = true
				queue = append(queue, next)
			}
		}
	}

	// Check if any statuses are unreachable
	var unreachable []string
	for status := range workflow.StatusFlow {
		if !reachable[status] {
			unreachable = append(unreachable, status)
		}
	}

	if len(unreachable) > 0 {
		return &ValidationError{
			Message: fmt.Sprintf("unreachable statuses (no path from %s): %s", StartStatusKey, strings.Join(unreachable, ", ")),
			Fix:     fmt.Sprintf("add transitions to make these statuses reachable from %s, or remove them", strings.Join(startStatuses, ", ")),
		}
	}

	return nil
}

// validateTerminalPaths checks that all statuses have a path to _complete_ statuses
// Uses reverse BFS from complete statuses to check backward reachability
func validateTerminalPaths(workflow *WorkflowConfig) error {
	completeStatuses := workflow.SpecialStatuses[CompleteStatusKey]

	// Build reverse graph (status → statuses that can reach it)
	reverseGraph := make(map[string][]string)
	for fromStatus, transitions := range workflow.StatusFlow {
		for _, toStatus := range transitions {
			reverseGraph[toStatus] = append(reverseGraph[toStatus], fromStatus)
		}
	}

	// BFS backward from complete statuses
	canReachComplete := make(map[string]bool)
	queue := make([]string, len(completeStatuses))
	copy(queue, completeStatuses)

	for _, status := range completeStatuses {
		canReachComplete[status] = true
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Add all statuses that can reach current
		for _, prev := range reverseGraph[current] {
			if !canReachComplete[prev] {
				canReachComplete[prev] = true
				queue = append(queue, prev)
			}
		}
	}

	// Check if any statuses are dead-ends (can't reach complete)
	var deadEnds []string
	for status := range workflow.StatusFlow {
		if !canReachComplete[status] {
			deadEnds = append(deadEnds, status)
		}
	}

	if len(deadEnds) > 0 {
		return &ValidationError{
			Message: fmt.Sprintf("dead-end statuses (no path to %s): %s", CompleteStatusKey, strings.Join(deadEnds, ", ")),
			Fix:     fmt.Sprintf("add transitions from these statuses to reach %s, or remove them", strings.Join(completeStatuses, ", ")),
		}
	}

	return nil
}

// ValidateTransition checks if a status transition is valid according to workflow
// This is used at runtime when updating task status
//
// Returns:
// - nil if transition is valid
// - ValidationError with current status, attempted status, and valid next statuses
func ValidateTransition(workflow *WorkflowConfig, fromStatus, toStatus string) error {
	// Check if fromStatus exists in workflow
	validNext, exists := workflow.StatusFlow[fromStatus]
	if !exists {
		return &ValidationError{
			Message: fmt.Sprintf("status '%s' is not defined in workflow", fromStatus),
			Fix:     "add this status to workflow config or use --force to override",
		}
	}

	// Check if toStatus is in valid next statuses
	for _, valid := range validNext {
		if valid == toStatus {
			return nil // Valid transition
		}
	}

	// Invalid transition
	if len(validNext) == 0 {
		return &ValidationError{
			Message: fmt.Sprintf("cannot transition from '%s' (terminal status)", fromStatus),
			Fix:     "use --force to override workflow validation",
		}
	}

	return &ValidationError{
		Message: fmt.Sprintf("invalid transition from '%s' to '%s'", fromStatus, toStatus),
		Fix:     fmt.Sprintf("valid transitions from '%s': %s. Use --force to override", fromStatus, strings.Join(validNext, ", ")),
	}
}
