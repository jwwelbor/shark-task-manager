package changecard

import (
	"context"
	"fmt"
)

// ChangeCardStatusSummary holds aggregate counts across all change-card statuses.
type ChangeCardStatusSummary struct {
	// Total is the count of all change-cards.
	Total int `json:"total"`
	// ByStatus holds per-status change-card counts.
	ByStatus map[string]int `json:"by_status"`
}

// ChangeCardThroughputStats holds aggregate throughput metrics for change-cards.
type ChangeCardThroughputStats struct {
	// DecidedCount is the number of change-cards that have been decided
	// (approved + in_progress + completed + declined).
	DecidedCount int `json:"decided_count"`
	// ApprovedCount is the number of change-cards with a positive outcome
	// (approved + in_progress + completed).
	ApprovedCount int `json:"approved_count"`
	// DeclinedCount is the number of declined change-cards.
	DeclinedCount int `json:"declined_count"`
	// ApprovalRate is the ratio of ApprovedCount to DecidedCount.
	// Nil when DecidedCount is 0 (avoid division by zero).
	ApprovalRate *float64 `json:"approval_rate"`
	// CompletedCount is the number of completed change-cards.
	CompletedCount int `json:"completed_count"`
	// AvgCompletionSecs is the mean seconds from created_at to updated_at for completed change-cards.
	// Nil when CompletedCount is 0.
	AvgCompletionSecs *float64 `json:"avg_completion_time_seconds"`
}

// GetStatusSummary returns aggregate status counts for all change-cards.
func (r *ChangeCardRepository) GetStatusSummary(ctx context.Context) (*ChangeCardStatusSummary, error) {
	summary := &ChangeCardStatusSummary{
		ByStatus: make(map[string]int),
	}

	// Total count
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM change_cards`).Scan(&summary.Total); err != nil {
		return nil, fmt.Errorf("failed to count change-cards: %w", err)
	}

	// Per-status counts
	rows, err := r.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM change_cards GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("failed to count change-cards by status: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan status count: %w", err)
		}
		summary.ByStatus[status] = count
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating change-card status counts: %w", err)
	}

	return summary, nil
}

// GetThroughputStats returns throughput metrics for change-cards.
//
// Decided statuses: approved, in_progress, completed, declined.
// Approved (positive outcome) statuses: approved, in_progress, completed.
// ApprovalRate is nil when DecidedCount == 0.
// AvgCompletionSecs is approximated using updated_at - created_at for completed cards.
// AvgCompletionSecs is nil when CompletedCount == 0.
func (r *ChangeCardRepository) GetThroughputStats(ctx context.Context) (*ChangeCardThroughputStats, error) {
	stats := &ChangeCardThroughputStats{}

	// Count decided, approved, declined, completed in one pass.
	// COALESCE handles the case where the table is empty (SUM returns NULL).
	countQuery := `
		SELECT
			COALESCE(SUM(CASE WHEN status IN ('approved','in_progress','completed','declined') THEN 1 ELSE 0 END), 0) AS decided_count,
			COALESCE(SUM(CASE WHEN status IN ('approved','in_progress','completed') THEN 1 ELSE 0 END), 0) AS approved_count,
			COALESCE(SUM(CASE WHEN status = 'declined' THEN 1 ELSE 0 END), 0) AS declined_count,
			COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0) AS completed_count
		FROM change_cards`

	if err := r.db.QueryRowContext(ctx, countQuery).Scan(
		&stats.DecidedCount,
		&stats.ApprovedCount,
		&stats.DeclinedCount,
		&stats.CompletedCount,
	); err != nil {
		return nil, fmt.Errorf("failed to get change-card throughput counts: %w", err)
	}

	// Compute approval rate only when there are decided cards
	if stats.DecidedCount > 0 {
		rate := float64(stats.ApprovedCount) / float64(stats.DecidedCount)
		stats.ApprovalRate = &rate
	}

	// Compute average completion time for completed cards
	if stats.CompletedCount > 0 {
		avgQuery := `
			SELECT AVG((JULIANDAY(updated_at) - JULIANDAY(created_at)) * 86400.0)
			FROM change_cards
			WHERE status = 'completed'`

		var avgSecs *float64
		if err := r.db.QueryRowContext(ctx, avgQuery).Scan(&avgSecs); err != nil {
			return nil, fmt.Errorf("failed to get average completion time: %w", err)
		}
		stats.AvgCompletionSecs = avgSecs
	}

	return stats, nil
}
