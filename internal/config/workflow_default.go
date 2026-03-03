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
			"todo":             {"in_progress", "blocked"},
			"in_progress":      {"ready_for_review", "blocked"},
			"ready_for_review": {"completed", "in_progress"}, // Can return to in_progress
			"completed":        {},                           // Terminal status
			"blocked":          {"todo", "in_progress"},      // Can unblock to todo or in_progress
		},

		// Metadata for each status (UI display, agent targeting, and progress tracking)
		StatusMetadata: map[string]StatusMetadata{
			"todo": {
				Color:          "gray",
				Description:    "Task is ready to be started",
				Phase:          "planning",
				AgentTypes:     []string{"business-analyst", "project-manager", "developer"},
				ProgressWeight: 0.0,
			},
			"in_progress": {
				Color:          "blue",
				Description:    "Task is actively being worked on",
				Phase:          "development",
				AgentTypes:     []string{"developer", "backend", "frontend", "api-developer"},
				ProgressWeight: 0.5,
			},
			"ready_for_review": {
				Color:          "yellow",
				Description:    "Implementation complete, awaiting code review",
				Phase:          "review",
				AgentTypes:     []string{"tech-lead", "senior-developer"},
				ProgressWeight: 0.9,
			},
			"completed": {
				Color:          "green",
				Description:    "Task reviewed, approved, and merged",
				Phase:          "done",
				AgentTypes:     []string{}, // No agents target completed tasks
				ProgressWeight: 1.0,
			},
			"blocked": {
				Color:          "red",
				Description:    "Task blocked by external dependency or issue",
				Phase:          "blocked",
				AgentTypes:     []string{"project-manager", "tech-lead"},
				ProgressWeight: 0.0,
			},
		},

		// Special statuses define workflow entry and exit points
		SpecialStatuses: map[string][]string{
			StartStatusKey:    {"todo"},      // New tasks start in "todo"
			CompleteStatusKey: {"completed"}, // Tasks complete in "completed"
		},

		// Require rejection reasons for backward transitions
		RequireRejectionReason: true,
	}
}

// DefaultEpicWorkflow returns the backward-compatible default epic workflow.
// Matches the current hardcoded epic status set: draft, active, completed, archived.
func DefaultEpicWorkflow() *WorkflowConfig {
	return &WorkflowConfig{
		Version: DefaultWorkflowVersion,
		StatusFlow: map[string][]string{
			"draft":     {"active", "archived"},
			"active":    {"completed", "archived"},
			"completed": {"archived"},
			"archived":  {},
		},
		StatusMetadata: map[string]StatusMetadata{
			"draft":     {Color: "gray", Description: "Epic created, not yet started", Phase: "planning", IsPlanning: true},
			"active":    {Color: "blue", Description: "Epic in progress, aggregating features", Phase: "execution", AggregatesFrom: "features"},
			"completed": {Color: "green", Description: "All features complete", Phase: "done"},
			"archived":  {Color: "gray", Description: "Epic archived", Phase: "done"},
		},
		SpecialStatuses: map[string][]string{
			StartStatusKey:       {"draft"},
			CompleteStatusKey:    {"completed", "archived"},
			AggregationStatusKey: {"active"},
		},
	}
}

// DefaultBugWorkflow returns the default bug workflow for bug tracking entities.
// Bug workflow statuses: reported, triaged, in_fix, in_verification, resolved, wont_fix, duplicate.
func DefaultBugWorkflow() *WorkflowConfig {
	return &WorkflowConfig{
		Version: DefaultWorkflowVersion,
		StatusFlow: map[string][]string{
			"reported":        {"triaged", "duplicate", "wont_fix"},
			"triaged":         {"in_fix", "wont_fix", "duplicate"},
			"in_fix":          {"in_verification", "triaged"},
			"in_verification": {"resolved", "in_fix"},
			"resolved":        {},
			"wont_fix":        {"triaged"},
			"duplicate":       {},
		},
		StatusMetadata: map[string]StatusMetadata{
			"reported":        {Color: "red", Description: "Bug reported, awaiting triage", Phase: "planning", AgentTypes: []string{"qa", "developer"}, ProgressWeight: 0.0},
			"triaged":         {Color: "yellow", Description: "Bug triaged, ready for fix", Phase: "planning", AgentTypes: []string{"tech-lead", "developer"}, ProgressWeight: 0.2},
			"in_fix":          {Color: "blue", Description: "Fix in progress", Phase: "development", AgentTypes: []string{"developer", "backend", "frontend"}, ProgressWeight: 0.5},
			"in_verification": {Color: "cyan", Description: "Fix applied, awaiting verification", Phase: "review", AgentTypes: []string{"qa"}, ProgressWeight: 0.8},
			"resolved":        {Color: "green", Description: "Bug verified as fixed", Phase: "done", AgentTypes: []string{}, ProgressWeight: 1.0},
			"wont_fix":        {Color: "gray", Description: "Bug will not be fixed", Phase: "done", AgentTypes: []string{}, ProgressWeight: 1.0},
			"duplicate":       {Color: "gray", Description: "Bug is a duplicate of another", Phase: "done", AgentTypes: []string{}, ProgressWeight: 1.0},
		},
		SpecialStatuses: map[string][]string{
			StartStatusKey:    {"reported"},
			CompleteStatusKey: {"resolved", "wont_fix", "duplicate"},
		},
		RequireRejectionReason: true,
	}
}

// DefaultChangeCardWorkflow returns the default change-card workflow.
// Change-card statuses: proposed, approved, in_progress, completed, declined.
func DefaultChangeCardWorkflow() *WorkflowConfig {
	return &WorkflowConfig{
		Version: DefaultWorkflowVersion,
		StatusFlow: map[string][]string{
			"proposed":    {"approved", "declined"},
			"approved":    {"in_progress", "declined"},
			"in_progress": {"completed", "approved"},
			"completed":   {},
			"declined":    {"proposed"},
		},
		StatusMetadata: map[string]StatusMetadata{
			"proposed":    {Color: "yellow", Description: "Change proposed, awaiting approval", Phase: "planning", AgentTypes: []string{"project-manager", "tech-lead"}, ProgressWeight: 0.0},
			"approved":    {Color: "cyan", Description: "Change approved, ready to implement", Phase: "planning", AgentTypes: []string{"developer", "tech-lead"}, ProgressWeight: 0.2},
			"in_progress": {Color: "blue", Description: "Change implementation in progress", Phase: "development", AgentTypes: []string{"developer", "backend", "frontend"}, ProgressWeight: 0.5},
			"completed":   {Color: "green", Description: "Change completed and verified", Phase: "done", AgentTypes: []string{}, ProgressWeight: 1.0},
			"declined":    {Color: "red", Description: "Change declined", Phase: "done", AgentTypes: []string{}, ProgressWeight: 1.0},
		},
		SpecialStatuses: map[string][]string{
			StartStatusKey:    {"proposed"},
			CompleteStatusKey: {"completed", "declined"},
		},
		RequireRejectionReason: true,
	}
}

// DefaultFeatureWorkflow returns the backward-compatible default feature workflow.
// Matches the current hardcoded feature status set: draft, active, completed, archived.
func DefaultFeatureWorkflow() *WorkflowConfig {
	return &WorkflowConfig{
		Version: DefaultWorkflowVersion,
		StatusFlow: map[string][]string{
			"draft":     {"active", "archived"},
			"active":    {"completed", "archived"},
			"completed": {"archived"},
			"archived":  {},
		},
		StatusMetadata: map[string]StatusMetadata{
			"draft":     {Color: "gray", Description: "Feature created, not yet started", Phase: "planning", IsPlanning: true},
			"active":    {Color: "blue", Description: "Feature in progress, aggregating tasks", Phase: "execution", AggregatesFrom: "tasks"},
			"completed": {Color: "green", Description: "All tasks complete", Phase: "done"},
			"archived":  {Color: "gray", Description: "Feature archived", Phase: "done"},
		},
		SpecialStatuses: map[string][]string{
			StartStatusKey:       {"draft"},
			CompleteStatusKey:    {"completed", "archived"},
			AggregationStatusKey: {"active"},
		},
	}
}
