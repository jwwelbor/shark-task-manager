package config

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

	// ProgressWeight indicates how much this status contributes to overall progress (0.0-1.0)
	// Used by CalculateProgress() to recognize partial completion:
	// - 0.0: not started (todo, draft, backlog)
	// - 0.5: in progress (in_development, in_progress, in_review)
	// - 0.9: nearly complete (ready_for_approval)
	// - 1.0: complete (completed, archived)
	// Default: 0.0 if not specified
	ProgressWeight float64 `json:"progress_weight"`

	// Responsibility defines who is responsible for work in this status
	// Values: "agent", "human", "qa_team", "none"
	// Used for work breakdown calculations (E07-F23)
	Responsibility string `json:"responsibility,omitempty"`

	// BlocksFeature indicates if tasks in this status block the feature progress
	// Used to identify blocked work in work breakdown calculations
	BlocksFeature bool `json:"blocks_feature,omitempty"`

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
