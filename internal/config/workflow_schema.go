package config

import "fmt"

// WorkflowConfig defines the structure for configurable status workflows in .sharkconfig.json
//
// Example JSON configuration:
//
//	{
//	  "status_flow_version": "1.0",
//	  "status_flow": {
//	    "todo": ["in_progress", "blocked"],
//	    "in_progress": ["ready_for_review", "blocked"],
//	    "ready_for_review": ["completed", "in_progress", "blocked"],
//	    "completed": [],
//	    "blocked": ["todo", "in_progress"]
//	  },
//	  "status_metadata": {
//	    "todo": {
//	      "color": "gray",
//	      "description": "Task is ready to be started",
//	      "phase": "planning",
//	      "agent_types": ["business-analyst", "project-manager"]
//	    },
//	    "in_progress": {
//	      "color": "blue",
//	      "description": "Task is actively being worked on",
//	      "phase": "development",
//	      "agent_types": ["developer", "backend", "frontend"]
//	    },
//	    "ready_for_review": {
//	      "color": "yellow",
//	      "description": "Task implementation complete, awaiting review",
//	      "phase": "review",
//	      "agent_types": ["tech-lead", "qa"]
//	    },
//	    "completed": {
//	      "color": "green",
//	      "description": "Task approved and merged",
//	      "phase": "done",
//	      "agent_types": []
//	    },
//	    "blocked": {
//	      "color": "red",
//	      "description": "Task blocked by external dependency",
//	      "phase": "blocked",
//	      "agent_types": ["project-manager"]
//	    }
//	  },
//	  "special_statuses": {
//	    "_start_": ["todo"],
//	    "_complete_": ["completed"]
//	  }
//	}
type WorkflowConfig struct {
	// Version of the workflow config schema (default: "1.0")
	// Used for future schema evolution and migration
	Version string `json:"status_flow_version"`

	// StatusFlow defines valid status transitions
	// Key: current status, Value: array of valid next statuses
	// Empty array means terminal status (no transitions out)
	StatusFlow map[string][]string `json:"status_flow"`

	// StatusMetadata provides additional metadata for each status
	// Optional: missing metadata fields default gracefully
	StatusMetadata map[string]StatusMetadata `json:"status_metadata"`

	// SpecialStatuses defines workflow entry and exit points
	// _start_: array of initial statuses (e.g., ["todo", "backlog"])
	// _complete_: array of terminal statuses (e.g., ["completed", "archived"])
	SpecialStatuses map[string][]string `json:"special_statuses"`
}

// StatusMetadata provides UI and agent-targeting metadata for a status
// All fields are optional
type StatusMetadata struct {
	// Color for display in CLI/UI (e.g., "red", "green", "blue", "#FF5733")
	// Used for colored terminal output (unless --no-color)
	Color string `json:"color,omitempty"`

	// Human-readable description of what this status means
	Description string `json:"description,omitempty"`

	// Workflow phase grouping (e.g., "planning", "development", "review", "qa", "done")
	// Used for task filtering: `shark task list --phase=development`
	Phase string `json:"phase,omitempty"`

	// Agent types that should see tasks in this status
	// Used for agent-targeted queries: `shark task list --agent=qa`
	// Examples: ["developer", "backend", "frontend", "qa", "business-analyst", "tech-lead"]
	AgentTypes []string `json:"agent_types,omitempty"`

	// OrchestratorAction specifies the action for orchestrators when task enters this status
	// Optional field for workflow-driven agent spawning (Phase 1 feature)
	OrchestratorAction *OrchestratorAction `json:"orchestrator_action,omitempty" yaml:"orchestrator_action,omitempty"`
}

// Special status keys used in SpecialStatuses map
const (
	// StartStatusKey defines initial statuses where new tasks begin
	StartStatusKey = "_start_"

	// CompleteStatusKey defines terminal statuses where tasks end
	CompleteStatusKey = "_complete_"
)

// Default version for workflow configs
const DefaultWorkflowVersion = "1.0"

// GetStatusMetadata returns metadata for a given status
// Returns empty metadata if status not found
func (w *WorkflowConfig) GetStatusMetadata(status string) (StatusMetadata, bool) {
	if w.StatusMetadata == nil {
		return StatusMetadata{}, false
	}
	meta, found := w.StatusMetadata[status]
	return meta, found
}

// GetStatusesByAgentType returns all statuses that include the given agent type
// Returns empty slice if no statuses match
func (w *WorkflowConfig) GetStatusesByAgentType(agentType string) []string {
	if w.StatusMetadata == nil {
		return []string{}
	}

	var statuses []string
	for status, meta := range w.StatusMetadata {
		for _, at := range meta.AgentTypes {
			if at == agentType {
				statuses = append(statuses, status)
				break
			}
		}
	}
	return statuses
}

// GetStatusesByPhase returns all statuses in the given phase
// Returns empty slice if no statuses match
func (w *WorkflowConfig) GetStatusesByPhase(phase string) []string {
	if w.StatusMetadata == nil {
		return []string{}
	}

	var statuses []string
	for status, meta := range w.StatusMetadata {
		if meta.Phase == phase {
			statuses = append(statuses, status)
		}
	}
	return statuses
}

// getPhaseOrder returns the ordering of phases for backward transition detection
// Lower numbers represent earlier phases in the workflow
// Returns -1 for unknown/any phases (which don't participate in backward detection)
func getPhaseOrder(phase string) int {
	phaseOrder := map[string]int{
		"planning":    0,
		"development": 1,
		"review":      2,
		"qa":          3,
		"approval":    4,
		"done":        5,
		"any":         -1, // Special phase that doesn't participate in order
		"blocked":     -1, // Special phase that doesn't participate in order
	}

	if order, found := phaseOrder[phase]; found {
		return order
	}

	// Unknown phases are treated as non-participating (-1)
	return -1
}

// IsBackwardTransition determines if a transition from one status to another is backward
// based on phase ordering. A backward transition is one where the new phase is ordered
// before (lower order number) the current phase.
//
// Returns:
//   - (false, nil) for forward transitions or same phase
//   - (true, nil) for backward transitions
//   - (false, error) if either status is not found in metadata
//
// Special cases:
//   - Transitions to/from "any" phase are not considered backward
//   - Transitions to/from "blocked" phase are not considered backward
//   - If either status lacks phase metadata, returns (false, nil) - not backward
func (w *WorkflowConfig) IsBackwardTransition(fromStatus, toStatus string) (bool, error) {
	// Get metadata for both statuses
	fromMeta, fromFound := w.GetStatusMetadata(fromStatus)
	toMeta, toFound := w.GetStatusMetadata(toStatus)

	// If either status is not found in metadata, return error
	if !fromFound || !toFound {
		return false, fmt.Errorf("status not found in metadata: from=%s (found=%v), to=%s (found=%v)",
			fromStatus, fromFound, toStatus, toFound)
	}

	// If either status lacks phase information, treat as not backward
	if fromMeta.Phase == "" || toMeta.Phase == "" {
		return false, nil
	}

	// Get phase orders
	fromOrder := getPhaseOrder(fromMeta.Phase)
	toOrder := getPhaseOrder(toMeta.Phase)

	// If either phase is "any" or "blocked" (order = -1), not backward
	if fromOrder == -1 || toOrder == -1 {
		return false, nil
	}

	// Backward if new phase order is less than current phase order
	return toOrder < fromOrder, nil
}
