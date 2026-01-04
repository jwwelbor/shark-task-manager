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

// ============================================================================
// Cascading Status Calculation Types (E07-F14)
// ============================================================================

// StatusChangeResult represents the outcome of a status calculation
type StatusChangeResult struct {
	EntityType     string    `json:"entity_type"`     // "feature" or "epic"
	EntityKey      string    `json:"entity_key"`      // e.g., "E07-F14"
	EntityID       int64     `json:"entity_id"`       // Database ID
	PreviousStatus string    `json:"previous_status"` // Status before change
	NewStatus      string    `json:"new_status"`      // Status after change
	WasChanged     bool      `json:"was_changed"`     // true if status actually changed
	WasSkipped     bool      `json:"was_skipped"`     // true if override prevented update
	SkipReason     string    `json:"skip_reason,omitempty"`
	CalculatedAt   time.Time `json:"calculated_at"`
}

// TaskStatusCounts provides task distribution for feature status calculation
type TaskStatusCounts struct {
	Todo           int `json:"todo"`
	InProgress     int `json:"in_progress"`
	ReadyForReview int `json:"ready_for_review"`
	Blocked        int `json:"blocked"`
	Completed      int `json:"completed"`
	Archived       int `json:"archived"`
	Total          int `json:"total"`
}

// FeatureStatusCounts provides feature distribution for epic status calculation
type FeatureStatusCounts struct {
	Draft     int `json:"draft"`
	Active    int `json:"active"`
	Completed int `json:"completed"`
	Archived  int `json:"archived"`
	Total     int `json:"total"`
}

// RecalculationSummary summarizes a batch recalculation
type RecalculationSummary struct {
	EpicsUpdated    int                  `json:"epics_updated"`
	FeaturesUpdated int                  `json:"features_updated"`
	FeaturesSkipped int                  `json:"features_skipped"`
	Changes         []StatusChangeResult `json:"changes"`
	StartedAt       time.Time            `json:"started_at"`
	CompletedAt     time.Time            `json:"completed_at"`
	DurationMs      int64                `json:"duration_ms"`
}

// StatusSource indicates where a status value comes from
type StatusSource string

const (
	StatusSourceCalculated StatusSource = "calculated"
	StatusSourceManual     StatusSource = "manual"
)
