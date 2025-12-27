package status

import (
	"fmt"
	"regexp"
	"time"
)

// ValidTimeframes defines the allowed values for recent completion windows
var ValidTimeframes = map[string]bool{
	"24h": true,
	"1d":  true,
	"48h": true,
	"7d":  true,
	"30d": true,
	"90d": true,
}

// AgentTypesOrder defines canonical ordering for agent types in output
var AgentTypesOrder = []string{
	"frontend",
	"backend",
	"api",
	"testing",
	"devops",
	"general",
	"unassigned",
}

// StatusDashboard is the complete dashboard output structure
type StatusDashboard struct {
	Summary           *ProjectSummary        `json:"summary"`
	Epics             []*EpicSummary         `json:"epics"`
	ActiveTasks       map[string][]*TaskInfo `json:"active_tasks"`
	BlockedTasks      []*BlockedTaskInfo     `json:"blocked_tasks"`
	RecentCompletions []*CompletionInfo      `json:"recent_completions,omitempty"`
	Filter            *DashboardFilter       `json:"filter,omitempty"`
}

// ProjectSummary contains high-level statistics about the entire project
type ProjectSummary struct {
	Epics           *CountBreakdown  `json:"epics"`
	Features        *CountBreakdown  `json:"features"`
	Tasks           *StatusBreakdown `json:"tasks"`
	OverallProgress float64          `json:"overall_progress"`
	BlockedCount    int              `json:"blocked_count"`
}

// CountBreakdown provides total and active counts for a resource
type CountBreakdown struct {
	Total  int `json:"total"`
	Active int `json:"active"`
}

// StatusBreakdown provides counts grouped by status
type StatusBreakdown struct {
	Total          int `json:"total"`
	Todo           int `json:"todo"`
	InProgress     int `json:"in_progress"`
	ReadyForReview int `json:"ready_for_review"`
	Completed      int `json:"completed"`
	Blocked        int `json:"blocked"`
}

// EpicSummary contains aggregated information about a single epic
type EpicSummary struct {
	Key             string  `json:"key"`
	Title           string  `json:"title"`
	ProgressPercent float64 `json:"progress_percent"`
	Health          string  `json:"health"` // "healthy", "warning", "critical"
	TasksTotal      int     `json:"tasks_total"`
	TasksCompleted  int     `json:"tasks_completed"`
	TasksBlocked    int     `json:"tasks_blocked"`
	FeaturesTotal   int     `json:"features_total"`
	FeaturesActive  int     `json:"features_active"`
}

// TaskInfo represents an active task in the dashboard
type TaskInfo struct {
	Key       string  `json:"key"`
	Title     string  `json:"title"`
	Feature   string  `json:"feature"`
	Epic      string  `json:"epic"`
	AgentType *string `json:"agent_type,omitempty"`
	Priority  int     `json:"priority"`
	StartedAt *string `json:"started_at,omitempty"`
}

// BlockedTaskInfo represents a blocked task with additional context
type BlockedTaskInfo struct {
	Key           string  `json:"key"`
	Title         string  `json:"title"`
	Feature       string  `json:"feature"`
	Epic          string  `json:"epic"`
	BlockedReason *string `json:"blocked_reason,omitempty"`
	BlockedAt     *string `json:"blocked_at,omitempty"`
	AgentType     *string `json:"agent_type,omitempty"`
}

// CompletionInfo represents a recently completed task
type CompletionInfo struct {
	Key          string    `json:"key"`
	Title        string    `json:"title"`
	Feature      string    `json:"feature"`
	Epic         string    `json:"epic"`
	CompletedAt  time.Time `json:"completed_at"`
	CompletedAgo *string   `json:"completed_ago,omitempty"`
	AgentType    *string   `json:"agent_type,omitempty"`
}

// DashboardFilter contains the filter criteria applied to the dashboard
type DashboardFilter struct {
	EpicKey         *string `json:"epic_key,omitempty"`
	RecentWindow    *string `json:"recent_window,omitempty"`
	IncludeArchived bool    `json:"include_archived"`
}

// StatusRequest represents the request parameters for generating a dashboard
type StatusRequest struct {
	EpicKey         string
	RecentWindow    string
	IncludeArchived bool
}

// Validate checks if the request parameters are valid
func (r *StatusRequest) Validate() error {
	// Validate epic key format if provided
	if r.EpicKey != "" && !isValidEpicKey(r.EpicKey) {
		return fmt.Errorf("invalid epic key format: %s (expected format: E[0-9]+)", r.EpicKey)
	}

	// Validate timeframe if provided
	if r.RecentWindow != "" && !ValidTimeframes[r.RecentWindow] {
		return fmt.Errorf("invalid timeframe: %s (valid: 24h, 1d, 48h, 7d, 30d, 90d)", r.RecentWindow)
	}

	return nil
}

// isValidEpicKey checks if the epic key matches the expected pattern
func isValidEpicKey(key string) bool {
	// Epic key pattern: E followed by digits (E01, E04, E123, etc.)
	epicPattern := regexp.MustCompile(`^E\d+$`)
	return epicPattern.MatchString(key)
}
