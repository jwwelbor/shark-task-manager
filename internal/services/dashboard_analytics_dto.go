package services

// BugAnalyticsResult contains bug analytics data for CLI/JSON output.
// Serialized key names match the JSON contract defined in the architecture document (section 8.2).
type BugAnalyticsResult struct {
	// TotalBugs is the count of all bugs regardless of status.
	TotalBugs int `json:"total_bugs"`
	// BugsByStatus contains a per-status count of all bugs.
	BugsByStatus map[string]int `json:"bugs_by_status"`
	// BugsBySeverity contains a per-severity count of all bugs.
	BugsBySeverity map[string]int `json:"bugs_by_severity"`
	// ResolvedCount is the number of bugs in terminal statuses (resolved, wont_fix, duplicate).
	ResolvedCount int `json:"resolved_count"`
	// AvgResolutionTimeSecs is the mean seconds from creation to terminal status.
	// Nil when ResolvedCount is 0 (no resolved bugs to average over).
	AvgResolutionTimeSecs *float64 `json:"avg_resolution_time_seconds"`
}

// ChangeCardAnalyticsResult contains change-card analytics data for CLI/JSON output.
// Serialized key names match the JSON contract defined in the architecture document (section 8.3).
type ChangeCardAnalyticsResult struct {
	// TotalChangeCards is the count of all change-cards regardless of status.
	TotalChangeCards int `json:"total_change_cards"`
	// ChangeCardsByStatus contains a per-status count of all change-cards.
	ChangeCardsByStatus map[string]int `json:"change_cards_by_status"`
	// ApprovalRate is the fraction of decided cards that were approved (ApprovedCount / DecidedCount).
	// Nil when DecidedCount is 0 (avoid division by zero).
	ApprovalRate *float64 `json:"approval_rate"`
	// DecidedCount is the number of change-cards that received a decision
	// (approved + in_progress + completed + declined).
	DecidedCount int `json:"decided_count"`
	// CompletedCount is the number of change-cards that reached the completed status.
	CompletedCount int `json:"completed_count"`
	// AvgCompletionTimeSecs is the mean seconds from creation to completed status.
	// Nil when CompletedCount is 0.
	AvgCompletionTimeSecs *float64 `json:"avg_completion_time_seconds"`
}

// DashboardAnalyticsResult is the combined analytics output used when no entity type
// filter is applied (i.e., shark analytics without --type flag).
// Fields use omitempty so that sections absent from the output are cleanly excluded from JSON.
type DashboardAnalyticsResult struct {
	// Bugs contains bug analytics, or nil if no bugs exist / bug repo is not configured.
	Bugs *BugAnalyticsResult `json:"bugs,omitempty"`
	// ChangeCards contains change-card analytics, or nil if no change-cards exist.
	ChangeCards *ChangeCardAnalyticsResult `json:"change_cards,omitempty"`
}
