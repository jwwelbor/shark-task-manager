package services

import "time"

// EpicFilters contains criteria for filtering epic lists.
type EpicFilters struct {
	Status string `json:"status,omitempty"` // Filter by status
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
