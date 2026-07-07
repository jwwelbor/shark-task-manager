package bug

import (
	"context"
	"fmt"
	"strings"
)

// BugStatusSummary holds aggregate counts across all bug statuses and severities.
type BugStatusSummary struct {
	// Total is the count of all bugs.
	Total int `json:"total"`
	// ByStatus holds per-status bug counts.
	ByStatus map[string]int `json:"by_status"`
	// BySeverity holds per-severity bug counts across all bugs.
	BySeverity map[string]int `json:"by_severity"`
	// OpenBySeverity holds per-severity counts for open (non-terminal) bugs only.
	// Terminal statuses excluded: resolved, wont_fix, duplicate.
	OpenBySeverity map[string]int `json:"open_by_severity"`
}

// BugResolutionStats holds aggregate resolution-time metrics for bugs.
type BugResolutionStats struct {
	// ResolvedCount is the number of bugs in terminal statuses (resolved, wont_fix, duplicate).
	ResolvedCount int `json:"resolved_count"`
	// AvgResolutionSecs is the mean seconds from created_at to updated_at for resolved bugs.
	// Nil when ResolvedCount is 0.
	AvgResolutionSecs *float64 `json:"avg_resolution_time_seconds"`
}

// BugFeatureSummary holds aggregate bug counts for bugs linked to a specific feature.
type BugFeatureSummary struct {
	// TotalLinked is the count of all bugs linked to the feature.
	TotalLinked int `json:"total_linked"`
	// OpenCount is the count of non-terminal bugs linked to the feature.
	OpenCount int `json:"open_count"`
	// OpenBySeverity holds per-severity counts of open bugs linked to the feature.
	OpenBySeverity map[string]int `json:"open_by_severity"`
}

// bugTerminalStatuses are statuses that mark a bug as resolved / closed.
// These are excluded from "open" counts and used for resolution-time calculations.
var bugTerminalStatuses = []interface{}{"resolved", "wont_fix", "duplicate"}

// bugTerminalStatusPlaceholders is the SQL placeholder string for bugTerminalStatuses.
// Derived from the slice length so the two stay in sync automatically.
var bugTerminalStatusPlaceholders = buildPlaceholders(len(bugTerminalStatuses))

// buildPlaceholders returns a comma-separated "?,?,?" string for n parameters.
func buildPlaceholders(n int) string {
	p := make([]string, n)
	for i := range p {
		p[i] = "?"
	}
	return strings.Join(p, ",")
}

// GetStatusSummary returns aggregate status and severity counts for all bugs.
// Uses a single GROUP BY status, severity query and post-processes in Go to
// derive total, per-status, per-severity, and open-by-severity counts.
func (r *BugRepository) GetStatusSummary(ctx context.Context) (*BugStatusSummary, error) {
	summary := &BugStatusSummary{
		ByStatus:       make(map[string]int),
		BySeverity:     make(map[string]int),
		OpenBySeverity: make(map[string]int),
	}

	// Build a set of terminal statuses for O(1) lookup during post-processing.
	terminalSet := make(map[string]bool, len(bugTerminalStatuses))
	for _, s := range bugTerminalStatuses {
		terminalSet[s.(string)] = true
	}

	// Single query: group by both status and severity to derive all four counts
	// in one round-trip. The result is post-processed in Go.
	rows, err := r.db.QueryContext(ctx, `SELECT status, severity, COUNT(*) FROM bugs GROUP BY status, severity`)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate bug counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status, severity string
		var count int
		if err := rows.Scan(&status, &severity, &count); err != nil {
			return nil, fmt.Errorf("failed to scan bug aggregate row: %w", err)
		}
		summary.Total += count
		summary.ByStatus[status] += count
		summary.BySeverity[severity] += count
		if !terminalSet[status] {
			summary.OpenBySeverity[severity] += count
		}
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating bug aggregate rows: %w", err)
	}

	return summary, nil
}

// GetResolutionStats returns resolution count and average resolution time for bugs.
// Resolution time is approximated as the difference between updated_at and created_at
// for bugs in terminal statuses (resolved, wont_fix, duplicate).
// AvgResolutionSecs is nil when no bugs have been resolved.
func (r *BugRepository) GetResolutionStats(ctx context.Context) (*BugResolutionStats, error) {
	stats := &BugResolutionStats{}

	query := `
		SELECT
			COUNT(*),
			AVG((JULIANDAY(updated_at) - JULIANDAY(created_at)) * 86400.0)
		FROM bugs
		WHERE status IN (` + bugTerminalStatusPlaceholders + `)`

	var avgSecs *float64
	if err := r.db.QueryRowContext(ctx, query, bugTerminalStatuses...).Scan(&stats.ResolvedCount, &avgSecs); err != nil {
		return nil, fmt.Errorf("failed to get bug resolution stats: %w", err)
	}

	// Only set AvgResolutionSecs when there are resolved bugs
	if stats.ResolvedCount > 0 {
		stats.AvgResolutionSecs = avgSecs
	}

	return stats, nil
}

// GetFeatureBugSummary returns aggregate bug counts for bugs linked to a specific feature.
// featureKey must match the linked_entity_key field of bugs (case-sensitive).
// OpenCount and OpenBySeverity exclude terminal statuses (resolved, wont_fix, duplicate).
// Uses a single conditional-aggregation query to derive all three counts in one round-trip.
func (r *BugRepository) GetFeatureBugSummary(ctx context.Context, featureKey string) (*BugFeatureSummary, error) {
	summary := &BugFeatureSummary{
		OpenBySeverity: make(map[string]int),
	}

	// Single query: group by severity and use conditional aggregation to compute
	// total linked count and open count simultaneously, eliminating two extra round-trips.
	//
	// Each row returns:
	//   severity         – severity bucket
	//   total_in_bucket  – all bugs (open + terminal) for this feature+severity
	//   open_in_bucket   – non-terminal bugs for this feature+severity
	query := `
		SELECT
			severity,
			COUNT(*) AS total_in_bucket,
			COUNT(CASE WHEN status NOT IN (` + bugTerminalStatusPlaceholders + `) THEN 1 END) AS open_in_bucket
		FROM bugs
		WHERE UPPER(linked_entity_key) = UPPER(?)
		GROUP BY severity`

	// Args: terminal statuses first (for the IN clause), then featureKey.
	args := append(bugTerminalStatuses, featureKey)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate linked bug counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var severity string
		var totalInBucket, openInBucket int
		if err := rows.Scan(&severity, &totalInBucket, &openInBucket); err != nil {
			return nil, fmt.Errorf("failed to scan linked bug aggregate row: %w", err)
		}
		summary.TotalLinked += totalInBucket
		summary.OpenCount += openInBucket
		if openInBucket > 0 {
			summary.OpenBySeverity[severity] = openInBucket
		}
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating linked bug aggregate rows: %w", err)
	}

	return summary, nil
}
