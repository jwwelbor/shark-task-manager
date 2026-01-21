// Package status provides status calculation and analysis for features and epics.
//
// This package uses config-driven calculations to:
// - Calculate weighted progress (recognizes partial work like ready_for_approval = 90%)
// - Break down work by responsibility (agent, human, blocked)
// - Identify action items (what needs attention)
// - Provide status context (e.g., "active (waiting)" vs "active (development)")
//
// All calculations are config-driven using progress_weight, responsibility, and
// blocks_feature metadata from .sharkconfig.json (configured via E07-F14).
package status

import "github.com/jwwelbor/shark-task-manager/internal/progress"

// ProgressInfo is re-exported from progress package for backward compatibility
type ProgressInfo = progress.ProgressInfo

// WorkSummary breaks down work by responsibility
// Used to show who's responsible for remaining work
type WorkSummary struct {
	TotalTasks     int // Total tasks
	CompletedTasks int // Fully completed
	AgentWork      int // Tasks with responsibility="agent"
	HumanWork      int // Tasks with responsibility="human" or "qa_team"
	BlockedWork    int // Tasks with blocks_feature=true
	NotStarted     int // Tasks with progress_weight=0.0 (excluding completed)
}

// ActionItems contains tasks requiring immediate attention
// Used to surface what needs PM/developer attention
type ActionItems struct {
	AwaitingApproval []*TaskActionItem // ready_for_approval status
	Blocked          []*TaskActionItem // blocked status
	InProgress       []*TaskActionItem // in_progress/in_development status
}

// TaskActionItem represents a single actionable task
type TaskActionItem struct {
	TaskKey       string  // E07-F23-003
	Title         string  // Task title
	Status        string  // Current status
	AgeDays       *int    // Days in current status (for waiting tasks)
	BlockedReason *string // Reason for blocking (if blocked)
}

// FeatureStatusInfo contains comprehensive status information for a feature
// Returned by repository layer, used by CLI for display
type FeatureStatusInfo struct {
	Feature         interface{}    // *models.Feature (avoid import cycle)
	StatusBreakdown map[string]int // Status -> count
	Tasks           []interface{}  // []*models.Task (avoid import cycle)
	Progress        *ProgressInfo  // Calculated progress metrics
	WorkSummary     *WorkSummary   // Work breakdown
	StatusContext   string         // "active (waiting)", "active (blocked)"
	ActionItems     *ActionItems   // Tasks needing attention
}
