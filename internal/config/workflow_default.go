package config

// DefaultWorkflow returns the backward-compatible default workflow
// that matches Shark's current hardcoded status progression.
//
// This workflow is used when:
// - .sharkconfig.json is missing
// - .sharkconfig.json lacks a status_flow section
// - status_flow section is invalid (fallback for safety)
//
// Default workflow transitions:
//
//	todo → in_progress → ready_for_review → completed
//	         ↕                  ↕
//	      blocked ←─────────────┘
//
// This ensures existing Shark projects continue working unchanged
// when upgrading to the configurable workflow system.
func DefaultWorkflow() *WorkflowConfig {
	return &WorkflowConfig{
		Version: DefaultWorkflowVersion,

		// Status transitions matching current hardcoded behavior
		StatusFlow: map[string][]string{
			"todo": {"in_progress", "blocked"},
			"in_progress": {"ready_for_review", "blocked"},
			"ready_for_review": {"completed", "in_progress"}, // Can return to in_progress
			"completed": {},                                   // Terminal status
			"blocked": {"todo", "in_progress"},                // Can unblock to todo or in_progress
		},

		// Metadata for each status (UI display and agent targeting)
		StatusMetadata: map[string]StatusMetadata{
			"todo": {
				Color:       "gray",
				Description: "Task is ready to be started",
				Phase:       "planning",
				AgentTypes:  []string{"business-analyst", "project-manager", "developer"},
			},
			"in_progress": {
				Color:       "blue",
				Description: "Task is actively being worked on",
				Phase:       "development",
				AgentTypes:  []string{"developer", "backend", "frontend", "api-developer"},
			},
			"ready_for_review": {
				Color:       "yellow",
				Description: "Implementation complete, awaiting code review",
				Phase:       "review",
				AgentTypes:  []string{"tech-lead", "senior-developer"},
			},
			"completed": {
				Color:       "green",
				Description: "Task reviewed, approved, and merged",
				Phase:       "done",
				AgentTypes:  []string{}, // No agents target completed tasks
			},
			"blocked": {
				Color:       "red",
				Description: "Task blocked by external dependency or issue",
				Phase:       "blocked",
				AgentTypes:  []string{"project-manager", "tech-lead"},
			},
		},

		// Special statuses define workflow entry and exit points
		SpecialStatuses: map[string][]string{
			StartStatusKey:    {"todo"},      // New tasks start in "todo"
			CompleteStatusKey: {"completed"}, // Tasks complete in "completed"
		},
	}
}

// IsDefaultStatus checks if a status exists in the default workflow
// Useful for validation and migration scenarios
func IsDefaultStatus(status string) bool {
	defaultStatuses := []string{"todo", "in_progress", "ready_for_review", "completed", "blocked"}
	for _, s := range defaultStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// DefaultStatuses returns all statuses in the default workflow
func DefaultStatuses() []string {
	return []string{"todo", "in_progress", "ready_for_review", "completed", "blocked"}
}
