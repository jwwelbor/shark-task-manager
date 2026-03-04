package repository

import (
	"context"
	"fmt"
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

// bugTerminalPlaceholders returns the SQL placeholder string for the terminal statuses.
const bugTerminalStatusPlaceholders = "?,?,?"

// GetStatusSummary returns aggregate status and severity counts for all bugs.
func (r *BugRepository) GetStatusSummary(ctx context.Context) (*BugStatusSummary, error) {
	summary := &BugStatusSummary{
		ByStatus:       make(map[string]int),
		BySeverity:     make(map[string]int),
		OpenBySeverity: make(map[string]int),
	}

	// Total count
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM bugs`).Scan(&summary.Total); err != nil {
		return nil, fmt.Errorf("failed to count bugs: %w", err)
	}

	// Per-status counts
	rowsStatus, err := r.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM bugs GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("failed to count bugs by status: %w", err)
	}
	defer rowsStatus.Close()
	for rowsStatus.Next() {
		var status string
		var count int
		if err := rowsStatus.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan status count: %w", err)
		}
		summary.ByStatus[status] = count
	}
	if err = rowsStatus.Err(); err != nil {
		return nil, fmt.Errorf("error iterating bug status counts: %w", err)
	}

	// Per-severity counts (all bugs)
	rowsSev, err := r.db.QueryContext(ctx, `SELECT severity, COUNT(*) FROM bugs GROUP BY severity`)
	if err != nil {
		return nil, fmt.Errorf("failed to count bugs by severity: %w", err)
	}
	defer rowsSev.Close()
	for rowsSev.Next() {
		var severity string
		var count int
		if err := rowsSev.Scan(&severity, &count); err != nil {
			return nil, fmt.Errorf("failed to scan severity count: %w", err)
		}
		summary.BySeverity[severity] = count
	}
	if err = rowsSev.Err(); err != nil {
		return nil, fmt.Errorf("error iterating bug severity counts: %w", err)
	}

	// Per-severity counts for open bugs (excluding terminal statuses)
	openSevQuery := `SELECT severity, COUNT(*) FROM bugs WHERE status NOT IN (` + bugTerminalStatusPlaceholders + `) GROUP BY severity`
	rowsOpenSev, err := r.db.QueryContext(ctx, openSevQuery, bugTerminalStatuses...)
	if err != nil {
		return nil, fmt.Errorf("failed to count open bugs by severity: %w", err)
	}
	defer rowsOpenSev.Close()
	for rowsOpenSev.Next() {
		var severity string
		var count int
		if err := rowsOpenSev.Scan(&severity, &count); err != nil {
			return nil, fmt.Errorf("failed to scan open severity count: %w", err)
		}
		summary.OpenBySeverity[severity] = count
	}
	if err = rowsOpenSev.Err(); err != nil {
		return nil, fmt.Errorf("error iterating open bug severity counts: %w", err)
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
func (r *BugRepository) GetFeatureBugSummary(ctx context.Context, featureKey string) (*BugFeatureSummary, error) {
	summary := &BugFeatureSummary{
		OpenBySeverity: make(map[string]int),
	}

	// Total linked bugs
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM bugs WHERE linked_entity_key = ?`, featureKey,
	).Scan(&summary.TotalLinked); err != nil {
		return nil, fmt.Errorf("failed to count linked bugs: %w", err)
	}

	// Open linked bugs
	openCountQuery := `SELECT COUNT(*) FROM bugs WHERE linked_entity_key = ? AND status NOT IN (` + bugTerminalStatusPlaceholders + `)`
	openArgs := append([]interface{}{featureKey}, bugTerminalStatuses...)
	if err := r.db.QueryRowContext(ctx, openCountQuery, openArgs...).Scan(&summary.OpenCount); err != nil {
		return nil, fmt.Errorf("failed to count open linked bugs: %w", err)
	}

	// Open bugs by severity
	openSevQuery := `
		SELECT severity, COUNT(*)
		FROM bugs
		WHERE linked_entity_key = ?
		  AND status NOT IN (` + bugTerminalStatusPlaceholders + `)
		GROUP BY severity`
	rows, err := r.db.QueryContext(ctx, openSevQuery, openArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to count open linked bugs by severity: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var severity string
		var count int
		if err := rows.Scan(&severity, &count); err != nil {
			return nil, fmt.Errorf("failed to scan severity count: %w", err)
		}
		summary.OpenBySeverity[severity] = count
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating open linked bug severity counts: %w", err)
	}

	return summary, nil
}
