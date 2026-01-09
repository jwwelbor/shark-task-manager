// Package workflow provides a centralized service for accessing workflow configuration
// across all CLI commands. It wraps the config package's WorkflowConfig with
// additional convenience methods and automatic project root detection.
package workflow

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// Service provides centralized access to workflow configuration.
// It loads and caches the workflow config from .sharkconfig.json,
// providing a single source of truth for status ordering, metadata, and transitions.
type Service struct {
	workflow    *config.WorkflowConfig
	projectRoot string
}

// NewService creates a new WorkflowService that loads configuration from the project root.
// If the config file is missing or invalid, it falls back to the default workflow.
//
// Parameters:
//   - projectRoot: path to project root directory (where .sharkconfig.json lives)
//
// Returns:
//   - *Service: initialized service with loaded or default workflow config
func NewService(projectRoot string) *Service {
	configPath := filepath.Join(projectRoot, ".sharkconfig.json")
	workflow := config.GetWorkflowOrDefault(configPath)

	return &Service{
		workflow:    workflow,
		projectRoot: projectRoot,
	}
}

// GetWorkflow returns the underlying workflow configuration.
// Never returns nil - falls back to default workflow if not configured.
func (s *Service) GetWorkflow() *config.WorkflowConfig {
	return s.workflow
}

// GetInitialStatus returns the first entry status for new tasks.
// Reads from special_statuses._start_[0] in workflow config.
// Falls back to "todo" if not configured.
func (s *Service) GetInitialStatus() models.TaskStatus {
	startStatuses, exists := s.workflow.SpecialStatuses[config.StartStatusKey]
	if !exists || len(startStatuses) == 0 {
		return models.TaskStatusTodo
	}
	return models.TaskStatus(startStatuses[0])
}

// GetEntryStatuses returns all valid entry statuses for new tasks.
// Reads from special_statuses._start_ in workflow config.
func (s *Service) GetEntryStatuses() []string {
	startStatuses, exists := s.workflow.SpecialStatuses[config.StartStatusKey]
	if !exists {
		return []string{string(models.TaskStatusTodo)}
	}
	return startStatuses
}

// GetTerminalStatuses returns all terminal statuses (no transitions out).
// Reads from special_statuses._complete_ in workflow config.
func (s *Service) GetTerminalStatuses() []string {
	completeStatuses, exists := s.workflow.SpecialStatuses[config.CompleteStatusKey]
	if !exists {
		return []string{string(models.TaskStatusCompleted)}
	}
	return completeStatuses
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
		Name:        status,
		Color:       meta.Color,
		Description: meta.Description,
		Phase:       meta.Phase,
		AgentTypes:  meta.AgentTypes,
	}
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
func (s *Service) IsValidTransition(currentStatus, targetStatus string) bool {
	transitions := s.GetValidTransitions(currentStatus)
	for _, valid := range transitions {
		if strings.EqualFold(valid, targetStatus) {
			return true
		}
	}
	return false
}

// IsValidStatus checks if a status is defined in the workflow.
func (s *Service) IsValidStatus(status string) bool {
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

// NormalizeStatus returns the canonical case for a status name.
// Returns the input unchanged if status is not found.
func (s *Service) NormalizeStatus(status string) string {
	for key := range s.workflow.StatusFlow {
		if strings.EqualFold(key, status) {
			return key
		}
	}
	return status
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
