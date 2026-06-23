package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// WorkflowValidationError represents a workflow configuration validation error
type WorkflowValidationError struct {
	Message string
	Fix     string // Suggested fix for the error
}

func (e *WorkflowValidationError) Error() string {
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
// - WorkflowValidationError with actionable message and fix suggestion if invalid
func ValidateWorkflow(workflow *WorkflowConfig) error {
	if workflow == nil {
		return &WorkflowValidationError{
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

	// Rule 5: Route-based (steps:) specific checks (E35-F06). The legacy-map
	// rules above already cover reachability/terminal-paths via the derived
	// maps; these add the checks only the route-based schema can express.
	if err := validateRouteBased(workflow); err != nil {
		return err
	}

	return nil
}

// validateRouteBased runs the checks specific to the consolidated steps: schema
// (E35-F06): the start step exists, every workable step defines the core
// outcome vocabulary with targets that resolve to real steps (D7), and no old
// status alias is claimed by two steps. It is a no-op for legacy workflows.
func validateRouteBased(workflow *WorkflowConfig) error {
	if !workflow.HasSteps() {
		return nil
	}

	// Start step must be defined.
	if workflow.Start != "" {
		if _, ok := workflow.GetStep(workflow.Start); !ok {
			return &WorkflowValidationError{
				Message: fmt.Sprintf("start step %q is not defined in steps", workflow.Start),
				Fix:     fmt.Sprintf("add a %q step or point start: at an existing step", workflow.Start),
			}
		}
	}

	// Core outcomes present + outcome targets resolve to real steps (D7).
	if errs := workflow.ValidateCoreOutcomes(); len(errs) > 0 {
		return &WorkflowValidationError{
			Message: errs[0].Error(),
			Fix:     "every non-terminal, non-parking step must define pass/fail/blocked outcomes, each targeting a defined step",
		}
	}

	// No old-status alias may be claimed by two steps.
	if _, aliasErrs := workflow.AliasMap(); len(aliasErrs) > 0 {
		return &WorkflowValidationError{
			Message: aliasErrs[0].Error(),
			Fix:     "give each old status name a single alias home (one step's aliases: list)",
		}
	}

	return nil
}

// validateSpecialStatuses checks that _start_ and _complete_ are defined
func validateSpecialStatuses(workflow *WorkflowConfig) error {
	startStatuses, hasStart := workflow.SpecialStatuses[StartStatusKey]
	if !hasStart || len(startStatuses) == 0 {
		return &WorkflowValidationError{
			Message: fmt.Sprintf("missing required special status '%s'", StartStatusKey),
			Fix:     fmt.Sprintf("add 'special_statuses.%s' array with at least one initial status (e.g., ['todo'])", StartStatusKey),
		}
	}

	completeStatuses, hasComplete := workflow.SpecialStatuses[CompleteStatusKey]
	if !hasComplete || len(completeStatuses) == 0 {
		return &WorkflowValidationError{
			Message: fmt.Sprintf("missing required special status '%s'", CompleteStatusKey),
			Fix:     fmt.Sprintf("add 'special_statuses.%s' array with at least one terminal status (e.g., ['completed'])", CompleteStatusKey),
		}
	}

	// Verify start statuses exist in status flow
	for _, status := range startStatuses {
		if _, exists := workflow.StatusFlow[status]; !exists {
			return &WorkflowValidationError{
				Message: fmt.Sprintf("start status '%s' is not defined in status_flow", status),
				Fix:     fmt.Sprintf("add '%s' to status_flow map or remove from %s array", status, StartStatusKey),
			}
		}
	}

	// Verify complete statuses exist in status flow
	for _, status := range completeStatuses {
		if _, exists := workflow.StatusFlow[status]; !exists {
			return &WorkflowValidationError{
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
		return &WorkflowValidationError{
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
		return &WorkflowValidationError{
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
		return &WorkflowValidationError{
			Message: fmt.Sprintf("dead-end statuses (no path to %s): %s", CompleteStatusKey, strings.Join(deadEnds, ", ")),
			Fix:     fmt.Sprintf("add transitions from these statuses to reach %s, or remove them", strings.Join(completeStatuses, ", ")),
		}
	}

	return nil
}

// WorkflowValidationFinding represents a single validation finding.
type WorkflowValidationFinding struct {
	Level   string `json:"level"`   // "error", "warning", "info"
	Message string `json:"message"` // Human-readable message
	Entity  string `json:"entity"`  // Entity type affected (e.g., "task", "epic")
	File    string `json:"file"`    // Source file path
}

// ValidateWorkflowFiles validates both .sharkconfig.json and .sharkworkflow.json,
// checking for JSON structure, duplicate definitions, and missing required sub-keys.
// Returns a list of findings (errors, warnings, info).
func ValidateWorkflowFiles(configPath string) []WorkflowValidationFinding {
	var results []WorkflowValidationFinding

	// Load the multi-level workflow to get source tracking
	multi, err := LoadMultiLevelWorkflow(configPath)
	if err != nil {
		results = append(results, WorkflowValidationFinding{
			Level:   "error",
			Message: fmt.Sprintf("Failed to load workflow config: %v", err),
			File:    configPath,
		})
		return results
	}

	// Report sources for each entity
	entityLevels := []string{"epic", "feature", "task", "bug", "change", "tech_debt"}
	for _, level := range entityLevels {
		source := multi.Sources[level]
		results = append(results, WorkflowValidationFinding{
			Level:   "info",
			Message: fmt.Sprintf("%s_workflow source: %s", level, source),
			Entity:  level,
			File:    source,
		})
	}

	// Check for duplicate definitions
	configData, configErr := readRawConfigKeys(configPath)
	var workflowFilePath string
	if configErr == nil {
		workflowFilePath, _ = resolveWorkflowFilePath(configPath, configData)
	}
	if workflowFilePath != "" {
		workflowData, workflowErr := readRawConfigKeys(workflowFilePath)

		if workflowErr == nil && workflowData != nil {
			entityWorkflowKeys := []string{"epic_workflow", "feature_workflow", "task_workflow", "bug_workflow", "change_workflow", "tech_debt_workflow"}
			for _, key := range entityWorkflowKeys {
				_, inConfig := configData[key]
				_, inWorkflow := workflowData[key]
				if inConfig && inWorkflow {
					level := strings.TrimSuffix(key, "_workflow")
					results = append(results, WorkflowValidationFinding{
						Level:   "warning",
						Message: fmt.Sprintf("Duplicate definition: %s is defined in both %s and %s. Workflow file takes precedence.", key, configPath, workflowFilePath),
						Entity:  level,
					})
				}
			}
		}
	}

	// Validate individual workflow configs
	workflowMap := map[string]*WorkflowConfig{
		"epic":      multi.Epic,
		"feature":   multi.Feature,
		"task":      multi.Task,
		"bug":       multi.Bug,
		"change":    multi.Change,
		"tech_debt": multi.TechDebt,
	}

	for level, wf := range workflowMap {
		if wf == nil {
			continue
		}
		// Check required sub-keys
		if len(wf.StatusFlow) == 0 {
			results = append(results, WorkflowValidationFinding{
				Level:   "warning",
				Message: fmt.Sprintf("%s_workflow: missing or empty status_flow", level),
				Entity:  level,
				File:    multi.Sources[level],
			})
		}
		if len(wf.StatusMetadata) == 0 {
			results = append(results, WorkflowValidationFinding{
				Level:   "warning",
				Message: fmt.Sprintf("%s_workflow: missing or empty status_metadata", level),
				Entity:  level,
				File:    multi.Sources[level],
			})
		}
		// Run full validation
		if err := ValidateWorkflow(wf); err != nil {
			results = append(results, WorkflowValidationFinding{
				Level:   "error",
				Message: fmt.Sprintf("%s_workflow: %v", level, err),
				Entity:  level,
				File:    multi.Sources[level],
			})
		}
	}

	// Check for legacy key deprecation
	if multi.HasLegacyTaskKeys {
		results = append(results, WorkflowValidationFinding{
			Level:   "warning",
			Message: "Legacy top-level status_flow keys coexist with task_workflow block. Migrate to task_workflow block. Run `shark init update` to auto-migrate.",
			Entity:  "task",
			File:    configPath,
		})
	}

	return results
}

// readRawConfigKeys reads a JSON file and returns the top-level keys as a map.
func readRawConfigKeys(path string) (map[string]json.RawMessage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// ValidateTransition checks if a status transition is valid according to workflow
// This is used at runtime when updating task status
//
// Returns:
// - nil if transition is valid
// - ValidationError with current status, attempted status, and valid next statuses
func ValidateTransition(workflow *WorkflowConfig, fromStatus, toStatus string) error {
	// Check if fromStatus exists in workflow (case-insensitive)
	var validNext []string
	for key, targets := range workflow.StatusFlow {
		if strings.EqualFold(key, fromStatus) {
			validNext = targets
			break
		}
	}
	if validNext == nil {
		return &WorkflowValidationError{
			Message: fmt.Sprintf("status '%s' is not defined in workflow", fromStatus),
			Fix:     "add this status to workflow config or use --force to override",
		}
	}

	// Check if toStatus is in valid next statuses (case-insensitive)
	for _, valid := range validNext {
		if strings.EqualFold(valid, toStatus) {
			return nil // Valid transition
		}
	}

	// Invalid transition
	if len(validNext) == 0 {
		return &WorkflowValidationError{
			Message: fmt.Sprintf("cannot transition from '%s' (terminal status)", fromStatus),
			Fix:     "use --force to override workflow validation",
		}
	}

	return &WorkflowValidationError{
		Message: fmt.Sprintf("invalid transition from '%s' to '%s'", fromStatus, toStatus),
		Fix:     fmt.Sprintf("valid transitions from '%s': %s. Use --force to override", fromStatus, strings.Join(validNext, ", ")),
	}
}
