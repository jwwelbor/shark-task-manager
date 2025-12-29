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
