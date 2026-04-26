package services

import (
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// CreateEpicInput contains the parameters for creating a new epic.
type CreateEpicInput struct {
	// Required fields
	Title string `json:"title"`

	// Optional fields
	Description   *string `json:"description,omitempty"`
	Status        string  `json:"status,omitempty"`         // Defaults to "draft" if empty
	Priority      string  `json:"priority,omitempty"`       // Defaults to "medium" if empty
	BusinessValue *string `json:"business_value,omitempty"` // Optional priority value
	FilePath      *string `json:"file_path,omitempty"`      // Custom file path (relative)
	CustomKey     string  `json:"custom_key,omitempty"`     // Override auto-generated key
	Force         bool    `json:"force,omitempty"`          // Force file reassignment

	// E28-F04 REQ-F-011: Tags attached to the epic on creation. Each name
	// must be registered in the vocabulary (see TagService.AttachMany) and
	// is validated+normalized by TagService.ValidateName. Empty/nil means
	// no tags. Required when Config.TagRequiredFor contains "epic"
	// (REQ-F-008 / AC-16).
	Tags []string `json:"tags,omitempty"`
	// Size is an optional canonical Fibonacci size value {1,2,3,5,8,13}.
	// Nil means "no size set" (stores NULL). Use models.ParseSize to convert
	// t-shirt labels (XS/S/M/L/XL/XXL) to numeric form before setting.
	// E07-F42 REQ-F-004.
	Size *int `json:"size,omitempty"`
}

// EpicUpdates contains fields that can be updated on an existing epic.
// Only non-nil pointer fields will be updated.
type EpicUpdates struct {
	Title         *string            `json:"title,omitempty"`
	Description   *string            `json:"description,omitempty"`
	Status        *models.EpicStatus `json:"status,omitempty"`
	Priority      *models.Priority   `json:"priority,omitempty"`
	BusinessValue *models.Priority   `json:"business_value,omitempty"`
	FilePath      *string            `json:"file_path,omitempty"`

	// E28-F04 REQ-F-010: Tags to attach additively on update. Empty/nil
	// means no tag change (see AC-18b). Removal on update is explicitly
	// NOT supported — use `shark epic tag rm` (REQ-F-014).
	Tags []string `json:"tags,omitempty"`
	// Size updates the size when non-nil. Use models.ParseSize to convert
	// t-shirt labels before setting. E07-F42 REQ-F-005.
	Size *int `json:"size,omitempty"`
	// ClearSize when true sets the epic's size to NULL regardless of the
	// Size field value. ClearSize takes precedence over Size.
	// Corresponds to `--size clear` on the CLI. E07-F42 REQ-F-005.
	ClearSize bool `json:"clear_size,omitempty"`
}

// CreateFeatureInput contains the parameters for creating a new feature.
type CreateFeatureInput struct {
	// Required fields
	EpicKey string `json:"epic_key"`
	Title   string `json:"title"`

	// Optional fields
	Description    *string `json:"description,omitempty"`
	Status         string  `json:"status,omitempty"`          // Defaults to "draft" if empty
	ExecutionOrder *int    `json:"execution_order,omitempty"` // Position in feature execution sequence
	FilePath       *string `json:"file_path,omitempty"`       // Custom file path (relative)
	Force          bool    `json:"force,omitempty"`           // Force file reassignment

	// E28-F04 REQ-F-011: Tags attached to the feature on creation. Each
	// name must be registered in the vocabulary (see TagService.AttachMany)
	// and is validated+normalized by TagService.ValidateName. Empty/nil
	// means no tags. Required when Config.TagRequiredFor contains
	// "feature" (REQ-F-008 / AC-16).
	Tags []string `json:"tags,omitempty"`
	// Size is an optional canonical Fibonacci size value {1,2,3,5,8,13}.
	// Nil means "no size set" (stores NULL). Use models.ParseSize to convert
	// t-shirt labels (XS/S/M/L/XL/XXL) to numeric form before setting.
	// E07-F42 REQ-F-004.
	Size *int `json:"size,omitempty"`
}

// FeatureUpdates contains fields that can be updated on an existing feature.
// Only non-nil pointer fields will be updated.
type FeatureUpdates struct {
	Title          *string               `json:"title,omitempty"`
	Description    *string               `json:"description,omitempty"`
	Status         *models.FeatureStatus `json:"status,omitempty"`
	ExecutionOrder *int                  `json:"execution_order,omitempty"`
	FilePath       *string               `json:"file_path,omitempty"`

	// E28-F04 REQ-F-010: Tags to attach additively on update. Empty/nil
	// means no tag change (see AC-18b). Removal on update is explicitly
	// NOT supported — use `shark feature tag rm` (REQ-F-014).
	Tags []string `json:"tags,omitempty"`
	// Size updates the size when non-nil. Use models.ParseSize to convert
	// t-shirt labels before setting. E07-F42 REQ-F-005.
	Size *int `json:"size,omitempty"`
	// ClearSize when true sets the feature's size to NULL regardless of the
	// Size field value. ClearSize takes precedence over Size.
	// Corresponds to `--size clear` on the CLI. E07-F42 REQ-F-005.
	ClearSize bool `json:"clear_size,omitempty"`
}

// EpicFilters contains criteria for filtering epic lists.
type EpicFilters struct {
	Status string `json:"status,omitempty"` // Filter by status
	// Tags filters epics to those tagged with ALL of the supplied names
	// (AND semantics, E28-F05 REQ-F-005). Empty/nil means no tag filter.
	Tags []string `json:"tags,omitempty"`
}

// EpicHealthInfo contains health analysis for an epic.
type EpicHealthInfo struct {
	EpicKey string   `json:"epic_key"`
	Status  string   `json:"status"` // healthy, warning, critical
	Reasons []string `json:"reasons,omitempty"`
}

// Impediment represents a blocked task that impedes progress.
type Impediment struct {
	TaskKey  string `json:"task_key"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Priority int    `json:"priority"`
	AgeDays  int    `json:"age_days"` // Days since task was blocked/last updated
}

// FeatureRollup aggregates feature statuses for an epic.
type FeatureRollup struct {
	EpicKey       string         `json:"epic_key"`
	TotalFeatures int            `json:"total_features"`
	StatusCounts  map[string]int `json:"status_counts"`
}

// EpicProgressInfo contains progress metrics for an epic.
type EpicProgressInfo struct {
	EpicKey       string         `json:"epic_key"`
	ProgressPct   float64        `json:"progress_pct"` // Overall completion percentage
	TotalFeatures int            `json:"total_features"`
	TaskRollup    map[string]int `json:"task_rollup,omitempty"` // Task status counts across all features
}

// FeatureFilters contains criteria for filtering feature lists.
type FeatureFilters struct {
	EpicKey string `json:"epic_key,omitempty"` // Filter by epic
	Status  string `json:"status,omitempty"`   // Filter by status
	// Tags filters features to those tagged with ALL of the supplied names
	// (AND semantics, E28-F05 REQ-F-005). Empty/nil means no tag filter.
	Tags []string `json:"tags,omitempty"`
}

// FeatureProgressInfo contains progress metrics for a feature.
type FeatureProgressInfo struct {
	FeatureKey         string  `json:"feature_key"`
	WeightedProgress   float64 `json:"weighted_progress"`   // Weighted progress percentage
	CompletionProgress float64 `json:"completion_progress"` // Raw completion percentage
	TotalTasks         int     `json:"total_tasks"`
	CompletedTasks     int     `json:"completed_tasks"`
	WeightedRatio      string  `json:"weighted_ratio"`   // "3.4/5"
	CompletionRatio    string  `json:"completion_ratio"` // "2/5"
}

// FeatureCompleteResult contains the result of a CompleteFeature operation.
type FeatureCompleteResult struct {
	FeatureKey      string         `json:"feature_key"`
	TotalCount      int            `json:"total_count"`
	CompletedCount  int            `json:"completed_count"`
	AffectedTasks   []string       `json:"affected_tasks"`
	StatusBreakdown map[string]int `json:"status_breakdown"`
	// RequiresForce is true when there are incomplete tasks and force was not set.
	// The caller should display a warning and ask the user to retry with --force.
	RequiresForce bool `json:"requires_force"`
}

// FeatureHealthInfo contains health analysis for a feature.
type FeatureHealthInfo struct {
	FeatureKey string   `json:"feature_key"`
	Status     string   `json:"status"` // healthy, warning, critical
	Reasons    []string `json:"reasons,omitempty"`
}

// WorkBreakdown categorizes remaining work by responsibility.
type WorkBreakdown struct {
	FeatureKey     string `json:"feature_key"`
	TotalTasks     int    `json:"total_tasks"`
	CompletedTasks int    `json:"completed_tasks"`
	AgentWork      int    `json:"agent_work"`
	HumanWork      int    `json:"human_work"`
	BlockedWork    int    `json:"blocked_work"`
	NotStarted     int    `json:"not_started"`
}

// EpicCompleteResult contains the result of a CompleteEpic operation.
type EpicCompleteResult struct {
	EpicKey      string `json:"epic_key"`
	FeatureCount int    `json:"feature_count"`
	TotalCount   int    `json:"total_count"`
	// CompletedCount is the total number of tasks now completed (pre-existing + newly completed).
	CompletedCount  int            `json:"completed_count"`
	AffectedTasks   []string       `json:"affected_tasks"`
	StatusBreakdown map[string]int `json:"status_breakdown"`
	// RequiresForce is true when there are incomplete tasks and force was not set.
	// The caller should display a warning and ask the user to retry with --force.
	RequiresForce bool `json:"requires_force"`
	// IncompleteDetails holds a summary of incomplete task counts per feature (for the warning message).
	IncompleteDetails map[string]FeatureIncompleteDetails `json:"incomplete_details,omitempty"`
	// ForceCompleted is true when the operation completed tasks that were not yet done.
	ForceCompleted bool `json:"force_completed"`
}

// FeatureIncompleteDetails holds incomplete task counts for a feature during epic complete.
type FeatureIncompleteDetails struct {
	TotalTasks      int            `json:"total_tasks"`
	CompletedTasks  int            `json:"completed_tasks"`
	IncompleteCount int            `json:"incomplete_count"`
	StatusBreakdown map[string]int `json:"status_breakdown"`
}

// ProblematicTask holds a summary of a problematic task for display.
type ProblematicTask struct {
	Key           string  `json:"key"`
	Status        string  `json:"status"`
	BlockedReason *string `json:"blocked_reason,omitempty"`
}

// FeatureActionItems contains tasks requiring immediate attention for a feature.
type FeatureActionItems struct {
	FeatureKey       string            `json:"feature_key"`
	AwaitingApproval []*ActionTaskItem `json:"awaiting_approval"`
	Blocked          []*ActionTaskItem `json:"blocked"`
	InProgress       []*ActionTaskItem `json:"in_progress"`
}

// ActionTaskItem represents a single actionable task.
type ActionTaskItem struct {
	TaskKey       string    `json:"task_key"`
	Title         string    `json:"title"`
	Status        string    `json:"status"`
	AgeDays       *int      `json:"age_days,omitempty"`
	BlockedReason *string   `json:"blocked_reason,omitempty"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// EpicRecalcResult contains the result of a RecalculateStatus operation on an epic.
type EpicRecalcResult struct {
	EpicKey        string `json:"epic_key"`
	PreviousStatus string `json:"previous_status"`
	NewStatus      string `json:"new_status"`
	WasChanged     bool   `json:"was_changed"`
}
