// Package workflow provides a centralized service for accessing workflow configuration
// across all CLI commands. It wraps the config package's WorkflowConfig with
// additional convenience methods and automatic project root detection.
package workflow

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/entitytype"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// Service provides centralized access to workflow configuration.
// It loads and caches the workflow config from .sharkconfig.json,
// providing a single source of truth for status ordering, metadata, and transitions.
//
// A Service wraps a single WorkflowConfig for one entity level (epic, feature, or task).
// Use ForLevel() to obtain a Service for a different level.
type Service struct {
	workflow    *config.WorkflowConfig
	projectRoot string
	level       string                     // "epic", "feature", or "task"
	multiLevel  *config.MultiLevelWorkflow // holds all three level configs
}

// NewService creates a new WorkflowService that loads configuration from the project root.
// If the config file is missing or invalid, it falls back to the default workflow.
// The returned service is configured for the task level (backward compatible).
//
// Parameters:
//   - projectRoot: path to project root directory (where .sharkconfig.json lives)
//
// Returns:
//   - *Service: initialized service with loaded or default workflow config (task level)
func NewService(projectRoot string) *Service {
	configPath := filepath.Join(projectRoot, ".sharkconfig.json")
	multi := config.LoadMultiLevelWorkflowOrDefault(configPath)

	return &Service{
		workflow:    multi.GetWorkflowForLevel(LevelTask),
		projectRoot: projectRoot,
		level:       LevelTask,
		multiLevel:  multi,
	}
}

// GetWorkflow returns the underlying workflow configuration.
// Never returns nil - falls back to default workflow if not configured.
func (s *Service) GetWorkflow() *config.WorkflowConfig {
	return s.workflow
}

// ForLevel returns a Service instance configured for the specified entity level.
// The returned service shares the same parsed config but operates on the
// level-specific workflow. All existing methods (IsValidTransition, GetValidTransitions,
// GetStatusMetadata, etc.) work on the level-specific workflow automatically.
//
// Parameters:
//   - level: LevelEpic, LevelFeature, or LevelTask
//
// Returns:
//   - *Service: configured for the specified level
func (s *Service) ForLevel(level string) *Service {
	level = entitytype.WorkflowLevelOrSelf(level)

	multi := s.multiLevel
	if multi == nil {
		multi = &config.MultiLevelWorkflow{}
	}
	return &Service{
		workflow:    multi.GetWorkflowForLevel(level),
		projectRoot: s.projectRoot,
		level:       level,
		multiLevel:  multi,
	}
}

// GetLevel returns the entity level this service is configured for.
func (s *Service) GetLevel() string {
	return s.level
}

// GetInitialStatusString returns the first entry status as a plain string.
// Level-agnostic: works for epic, feature, and task levels.
// Unlike GetInitialStatus() which returns models.TaskStatus, this method
// returns a plain string suitable for any entity type.
func (s *Service) GetInitialStatusString() string {
	startStatuses, exists := s.workflow.SpecialStatuses[config.StartStatusKey]
	if !exists || len(startStatuses) == 0 {
		switch s.level {
		case LevelEpic, LevelFeature:
			return "draft"
		case LevelSprint:
			return "planning"
		default:
			return "todo"
		}
	}
	return startStatuses[0]
}

// ValidateTransition checks if a transition is valid and returns a descriptive error if not.
// Delegates to config.ValidateTransition for the actual check.
func (s *Service) ValidateTransition(fromStatus, toStatus string) error {
	return config.ValidateTransition(s.workflow, fromStatus, toStatus)
}

// GetInitialStatus returns the first entry status for new tasks.
// Reads from special_statuses._start_[0] in workflow config.
// Falls back to "todo" if not configured.
func (s *Service) GetInitialStatus() models.TaskStatus {
	startStatuses, exists := s.workflow.SpecialStatuses[config.StartStatusKey]
	if !exists || len(startStatuses) == 0 {
		return models.TaskStatus("todo")
	}
	return models.TaskStatus(startStatuses[0])
}

// GetEntryStatuses returns all valid entry statuses for new tasks.
// Reads from special_statuses._start_ in workflow config.
func (s *Service) GetEntryStatuses() []string {
	startStatuses, exists := s.workflow.SpecialStatuses[config.StartStatusKey]
	if !exists {
		return []string{"todo"}
	}
	return startStatuses
}

// GetTerminalStatuses returns all terminal statuses (no transitions out).
// Reads from special_statuses._complete_ in workflow config.
func (s *Service) GetTerminalStatuses() []string {
	completeStatuses, exists := s.workflow.SpecialStatuses[config.CompleteStatusKey]
	if !exists {
		return []string{"completed"}
	}
	return completeStatuses
}

// GetAggregationStatuses returns the aggregation statuses for this workflow level.
// Reads from special_statuses._aggregation_ in workflow config.
// Falls back to ["active"] if not configured or empty.
func (s *Service) GetAggregationStatuses() []string {
	aggStatuses, exists := s.workflow.SpecialStatuses[config.AggregationStatusKey]
	if !exists || len(aggStatuses) == 0 {
		return []string{"active"}
	}
	return aggStatuses
}

// IsTerminalStatus returns true if the given status is a terminal status.
func (s *Service) IsTerminalStatus(status string) bool {
	for _, terminal := range s.GetTerminalStatuses() {
		if strings.EqualFold(terminal, status) {
			return true
		}
	}
	return false
}

// IsParkingStatus reports whether the given status is a parking step (e.g.
// blocked, on_hold) whose resume target is computed from history rather than a
// static outcome (route-based schema: the step's parking flag). Old status
// aliases are resolved first. Returns false for legacy workflows that do not
// define steps, or for unknown statuses (graceful nil-workflow degradation,
// mirroring IsTerminalStatus).
func (s *Service) IsParkingStatus(status string) bool {
	if s.workflow == nil {
		return false
	}
	status = s.aliasResolve(status)
	if st, ok := s.workflow.GetStep(status); ok && st != nil {
		return st.Parking
	}
	return false
}

// IsBlockedStatus reports whether the given status sits in the "blocked" phase
// (the cross-vocabulary signal for an entity halted by an external blocker).
// It reads the phase from status metadata, which is populated for both
// route-based (derived) and legacy workflows. Returns false for a nil workflow
// or unknown status.
func (s *Service) IsBlockedStatus(status string) bool {
	if s.workflow == nil {
		return false
	}
	return strings.EqualFold(s.getStatusPhase(s.aliasResolve(status)), "blocked")
}

// GetAllStatuses returns all defined statuses ordered by workflow phase.
// Phase order: planning -> development -> review -> qa -> approval -> done -> any
func (s *Service) GetAllStatuses() []string {
	return s.GetAllStatusesOrdered()
}

// GetAllStatusesOrdered returns all statuses from the workflow config,
// ordered by phase hierarchy: planning -> development -> review -> qa -> approval -> done -> any
func (s *Service) GetAllStatusesOrdered() []string {
	// Get all statuses from status_flow keys
	statusSet := make(map[string]bool)
	for status := range s.workflow.StatusFlow {
		statusSet[status] = true
	}
	// Also include statuses that only appear as transition targets
	for _, targets := range s.workflow.StatusFlow {
		for _, target := range targets {
			statusSet[target] = true
		}
	}

	// Convert to slice
	statuses := make([]string, 0, len(statusSet))
	for status := range statusSet {
		statuses = append(statuses, status)
	}

	// Sort by phase order
	phaseOrder := map[string]int{
		"planning":    0,
		"development": 1,
		"review":      2,
		"qa":          3,
		"approval":    4,
		"done":        5,
		"any":         6,
		"":            7, // Unknown phase at end
	}

	sort.Slice(statuses, func(i, j int) bool {
		phaseI := s.getStatusPhase(statuses[i])
		phaseJ := s.getStatusPhase(statuses[j])

		orderI, okI := phaseOrder[phaseI]
		orderJ, okJ := phaseOrder[phaseJ]

		// Unknown phases go to end
		if !okI {
			orderI = 99
		}
		if !okJ {
			orderJ = 99
		}

		if orderI != orderJ {
			return orderI < orderJ
		}

		// Same phase - sort alphabetically
		return statuses[i] < statuses[j]
	})

	return statuses
}

// getStatusPhase returns the phase for a given status from metadata
func (s *Service) getStatusPhase(status string) string {
	if meta, found := s.workflow.GetStatusMetadata(status); found {
		return meta.Phase
	}
	return ""
}

// GetStatusMetadata returns metadata for a given status.
// Returns empty metadata if status not found in config.
func (s *Service) GetStatusMetadata(status string) StatusInfo {
	meta, found := s.workflow.GetStatusMetadata(status)
	if !found {
		return StatusInfo{
			Name: status,
		}
	}

	return StatusInfo{
		Name:                status,
		Color:               meta.Color,
		DisplayToken:        meta.DisplayToken,
		Description:         meta.Description,
		Phase:               meta.Phase,
		AgentTypes:          meta.AgentTypes,
		ExcludeFromProgress: meta.ExcludeFromProgress,
		SprintBucket:        meta.SprintBucket,
	}
}

// HasOrchestratorAction returns true when the status metadata declares the
// requested orchestrator action for this workflow level.
func (s *Service) HasOrchestratorAction(status, action string) bool {
	meta, found := s.workflow.GetStatusMetadata(s.NormalizeStatus(status))
	if !found || meta.OrchestratorAction == nil {
		return false
	}
	return strings.EqualFold(meta.OrchestratorAction.Action, action)
}

// GetSingleNextStatus returns the only valid transition from the given status.
// It returns ok=false when the status is terminal, unknown, or branches to
// multiple next statuses.
func (s *Service) GetSingleNextStatus(currentStatus string) (next string, ok bool) {
	transitions := s.GetValidTransitions(currentStatus)
	if len(transitions) != 1 {
		return "", false
	}
	return transitions[0], true
}

// GetStatusesByPhase returns all statuses in the given phase.
// Phase examples: "planning", "development", "review", "qa", "done"
func (s *Service) GetStatusesByPhase(phase string) []string {
	return s.workflow.GetStatusesByPhase(phase)
}

// GetStatusesByAgentType returns all statuses that include the given agent type.
func (s *Service) GetStatusesByAgentType(agentType string) []string {
	return s.workflow.GetStatusesByAgentType(agentType)
}

// GetValidTransitions returns the valid next statuses for a given current status.
// Returns empty slice if status is terminal or not found.
func (s *Service) GetValidTransitions(currentStatus string) []string {
	// Normalize to lowercase for case-insensitive lookup
	for status, transitions := range s.workflow.StatusFlow {
		if strings.EqualFold(status, currentStatus) {
			return transitions
		}
	}
	return []string{}
}

// GetTransitionInfo returns detailed information about valid transitions from a status.
func (s *Service) GetTransitionInfo(currentStatus string) []TransitionInfo {
	transitions := s.GetValidTransitions(currentStatus)
	result := make([]TransitionInfo, 0, len(transitions))

	for _, target := range transitions {
		info := TransitionInfo{
			TargetStatus: target,
		}

		// Add metadata if available
		if meta, found := s.workflow.GetStatusMetadata(target); found {
			info.Description = meta.Description
			info.Phase = meta.Phase
			info.AgentTypes = meta.AgentTypes
			info.Color = meta.Color
		}

		result = append(result, info)
	}

	return result
}

// IsValidTransition checks if transitioning from current to target status is valid.
// For route-based workflows, old status aliases are resolved first (E35-F05).
func (s *Service) IsValidTransition(currentStatus, targetStatus string) bool {
	currentStatus = s.aliasResolve(currentStatus)
	targetStatus = s.aliasResolve(targetStatus)
	transitions := s.GetValidTransitions(currentStatus)
	for _, valid := range transitions {
		if strings.EqualFold(valid, targetStatus) {
			return true
		}
	}
	return false
}

// IsValidStatus checks if a status is defined in the workflow.
// For route-based workflows, an old status alias counts as valid (input shim).
func (s *Service) IsValidStatus(status string) bool {
	status = s.aliasResolve(status)
	// Check if status is in status_flow keys
	for key := range s.workflow.StatusFlow {
		if strings.EqualFold(key, status) {
			return true
		}
	}
	// Also check if it appears as a transition target
	for _, targets := range s.workflow.StatusFlow {
		for _, target := range targets {
			if strings.EqualFold(target, status) {
				return true
			}
		}
	}
	return false
}

// aliasResolve resolves an old status alias to its new step for route-based
// workflows; for legacy workflows it returns the input unchanged.
func (s *Service) aliasResolve(status string) string {
	if s.workflow != nil && s.workflow.HasSteps() {
		return s.workflow.ResolveAlias(status)
	}
	return status
}

// NormalizeStatus returns the canonical case for a status name.
// Returns the input unchanged if status is not found.
//
// For route-based workflows it also applies the alias compat shim (E35-F05):
// an old status name (e.g. "ready_for_qa") is resolved to its new step (e.g.
// "qa") so hooks, scripts, and muscle memory keep working during the
// deprecation window.
func (s *Service) NormalizeStatus(status string) string {
	if s.workflow != nil && s.workflow.HasSteps() {
		resolved := s.workflow.ResolveAlias(status)
		if resolved != status {
			return resolved
		}
	}
	for key := range s.workflow.StatusFlow {
		if strings.EqualFold(key, status) {
			return key
		}
	}
	return status
}

// ResolveAlias maps an old status name to its new step via the route-based
// alias map, or returns the input unchanged. Exposed for history-read
// resolution (E35-F05, §7) where an entity parked under an old status name is
// read after migration.
func (s *Service) ResolveAlias(status string) string {
	if s.workflow == nil {
		return status
	}
	return s.workflow.ResolveAlias(status)
}

// StatusAliasMap returns the old-status -> new-step map for the active workflow
// (empty for legacy workflows). Used by the one-shot status migration.
func (s *Service) StatusAliasMap() map[string]string {
	if s.workflow == nil {
		return map[string]string{}
	}
	m, _ := s.workflow.AliasMap()
	return m
}

// GetPhases returns all unique phases from status metadata, in workflow order.
func (s *Service) GetPhases() []string {
	phaseSet := make(map[string]bool)
	for _, meta := range s.workflow.StatusMetadata {
		if meta.Phase != "" {
			phaseSet[meta.Phase] = true
		}
	}

	// Convert to ordered slice
	phaseOrder := []string{"planning", "development", "review", "qa", "approval", "done", "any"}
	result := make([]string, 0, len(phaseSet))

	for _, phase := range phaseOrder {
		if phaseSet[phase] {
			result = append(result, phase)
			delete(phaseSet, phase)
		}
	}

	// Add any remaining phases not in standard order
	for phase := range phaseSet {
		result = append(result, phase)
	}

	return result
}

// FormattedStatus represents a status formatted for display with color and metadata.
type FormattedStatus struct {
	Status      string // Raw status (e.g., "in_progress")
	Colored     string // With ANSI codes (e.g., "\033[33min_progress\033[0m")
	Description string // Human-readable (e.g., "Code implementation in progress")
	Phase       string // Workflow phase (e.g., "development")
	ColorName   string // Color name (e.g., "yellow")
}

// FormatStatusForDisplay returns a formatted status with color and metadata.
// If colorEnabled is true, the Colored field will contain ANSI color codes.
func (s *Service) FormatStatusForDisplay(status string, colorEnabled bool) FormattedStatus {
	meta := s.GetStatusMetadata(status)

	formatted := FormattedStatus{
		Status:      status,
		Description: meta.Description,
		Phase:       meta.Phase,
		ColorName:   meta.Color,
	}

	if colorEnabled && meta.Color != "" {
		formatted.Colored = colorizeStatus(status, meta.Color)
	} else {
		formatted.Colored = status
	}

	return formatted
}

// FormatStatusCount formats a StatusCount for display with color.
func (s *Service) FormatStatusCount(sc StatusCount, colorEnabled bool) string {
	if colorEnabled && sc.Color != "" {
		return colorizeStatus(sc.Status, sc.Color)
	}
	return sc.Status
}

// colorizeStatus applies ANSI color codes to a status string.
func colorizeStatus(status, colorName string) string {
	colorCodes := map[string]string{
		"red":     "\033[31m",
		"green":   "\033[32m",
		"yellow":  "\033[33m",
		"blue":    "\033[34m",
		"magenta": "\033[35m",
		"cyan":    "\033[36m",
		"white":   "\033[37m",
		"gray":    "\033[90m",
		"orange":  "\033[38;5;208m",
		"purple":  "\033[38;5;141m",
	}

	reset := "\033[0m"
	colorCode, found := colorCodes[colorName]
	if !found {
		return status
	}

	return colorCode + status + reset
}

// GetColorForStatus returns the color name for a status, or empty string if not configured.
func (s *Service) GetColorForStatus(status string) string {
	meta := s.GetStatusMetadata(status)
	return meta.Color
}

// ValidateStatus returns nil if the given status is valid in the current workflow,
// or a descriptive error listing valid statuses otherwise.
func (s *Service) ValidateStatus(status string) error {
	if s.IsValidStatus(status) {
		return nil
	}

	allStatuses := s.GetAllStatusesOrdered()
	return fmt.Errorf("invalid status %q; valid statuses: %s", status, strings.Join(allStatuses, ", "))
}

// StatusHelpText returns formatted help text listing all statuses grouped by phase.
// Each phase is shown as a header followed by its statuses with description and color info.
func (s *Service) StatusHelpText() string {
	var b strings.Builder

	phases := s.GetPhases()
	for i, phase := range phases {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("[%s]\n", phase))

		statuses := s.GetStatusesByPhase(phase)
		for _, status := range statuses {
			meta := s.GetStatusMetadata(status)
			line := fmt.Sprintf("  %-30s", status)
			if meta.Description != "" {
				line += " - " + meta.Description
			}
			if meta.Color != "" {
				line += fmt.Sprintf(" (%s)", meta.Color)
			}
			b.WriteString(line + "\n")
		}
	}

	return b.String()
}

// StatusFlagDescription returns a short one-line description suitable for Cobra flag help text.
// Format: "Status filter (todo|in_progress|completed|blocked)"
func (s *Service) StatusFlagDescription() string {
	allStatuses := s.GetAllStatusesOrdered()
	return "Status filter (" + strings.Join(allStatuses, "|") + ")"
}

// IsCompletedStatus returns true if the given status represents a completed/terminal state.
// This is a semantic alias for IsTerminalStatus, intended for use in commands that check
// whether a task is "done" rather than whether it has no outgoing transitions.
func (s *Service) IsCompletedStatus(status string) bool {
	return s.IsTerminalStatus(status)
}

// IsBackwardTransition checks if transitioning from one status to another
// moves backward in the workflow phase ordering.
// Delegates to the underlying WorkflowConfig.IsBackwardTransition().
// Returns (false, nil) when workflow config is nil (graceful degradation).
func (s *Service) IsBackwardTransition(fromStatus, toStatus string) (bool, error) {
	if s.workflow == nil {
		return false, nil
	}
	return s.workflow.IsBackwardTransition(fromStatus, toStatus)
}

// GetDefaultStatus returns the initial status for new tasks as a string.
// This is a convenience wrapper around GetInitialStatus that returns a string
// instead of models.TaskStatus.
func (s *Service) GetDefaultStatus() string {
	return string(s.GetInitialStatus())
}

// --- Route-based outcome routing (E35-F02, decisions D2/D4) ---

// IsRouteBased reports whether the active workflow uses the consolidated
// per-step (steps:) schema with outcome routing.
func (s *Service) IsRouteBased() bool {
	return s.workflow != nil && s.workflow.HasSteps()
}

// GetOutcomes returns the outcome→target map for the given step/status.
// Returns nil when the workflow is not route-based or the step has no outcomes
// (terminal/parking steps).
func (s *Service) GetOutcomes(status string) map[string]string {
	if s.workflow == nil {
		return nil
	}
	status = s.aliasResolve(status)
	st, ok := s.workflow.GetStep(status)
	if !ok || st == nil {
		return nil
	}
	return st.Outcomes
}

// GetValidOutcomes returns the sorted outcome names defined for a step/status.
func (s *Service) GetValidOutcomes(status string) []string {
	outcomes := s.GetOutcomes(status)
	if len(outcomes) == 0 {
		return []string{}
	}
	keys := make([]string, 0, len(outcomes))
	for k := range outcomes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Release resolves a semantic outcome to its target status using the
// route-based outcomes map (decision D4: `advance` becomes `release(outcome)`).
// The caller emits an outcome (pass/fail/blocked/…); the engine resolves
// step.outcomes[outcome] and returns the target status. The caller then
// performs the transition.
//
// Returns an error when the workflow is not route-based, the step is unknown,
// or the outcome is not defined for the step.
func (s *Service) Release(fromStatus, outcome string) (string, error) {
	if !s.IsRouteBased() {
		return "", fmt.Errorf("outcome routing requires a route-based (steps:) workflow; use --status to set a target directly")
	}
	fromStatus = s.aliasResolve(fromStatus)
	target, ok := s.workflow.ResolveOutcome(fromStatus, outcome)
	if !ok {
		valid := s.GetValidOutcomes(fromStatus)
		if len(valid) == 0 {
			return "", fmt.Errorf("step %q defines no outcomes (terminal or parking step)", fromStatus)
		}
		return "", fmt.Errorf("no outcome %q defined for step %q (valid outcomes: %s)", outcome, fromStatus, strings.Join(valid, ", "))
	}
	return target, nil
}
