package sprint

import (
	"context"
	"fmt"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
)

// SprintAnalyticsRepository provides read-only aggregate queries over
// sprint_assignments joined to entity tables (tasks, bugs, change_cards,
// tech_debts) and task_history. It satisfies the SprintAnalyticsRepository
// interface defined in internal/services/sprint_analytics_dto.go.
//
// All queries use parameterized ? placeholders — no string interpolation.
// Errors are wrapped with fmt.Errorf("failed to ...: %w", err).
type SprintAnalyticsRepository struct {
	db *dbconn.DB
}

// NewSprintAnalyticsRepository creates a SprintAnalyticsRepository backed by db.
func NewSprintAnalyticsRepository(db *dbconn.DB) *SprintAnalyticsRepository {
	return &SprintAnalyticsRepository{db: db}
}

// GetVelocityData returns the last `limit` completed sprints ordered oldest-first
// (ascending end_date). For each sprint it aggregates:
//   - CompletedSize: Σ COALESCE(entity.size, 0) across all assigned entities.
//   - UnsizedCompleted: count of assigned entities where size IS NULL.
//
// Only sprints with status = 'completed' are included.
//
// When limit <= 0 the function returns an empty slice without hitting the database.
//
// Query design (TC-NF-02): the UNION ALL CTE approach gives SQLite one
// sub-query per entity type, each joined to sprint_assignments on
// (sprint_id, entity_type, entity_id). The outer aggregate groups by sprint.
// The idx_sprint_assignments_sprint index on sprint_assignments(sprint_id) is
// used in the join; EXPLAIN QUERY PLAN shows SEARCH TABLE sprint_assignments
// USING INDEX idx_sprint_assignments_sprint.
func (r *SprintAnalyticsRepository) GetVelocityData(ctx context.Context, limit int) ([]VelocityRow, error) {
	if limit <= 0 {
		return []VelocityRow{}, nil
	}

	// CTE: union the four entity types so each assignment row carries the
	// entity's size (or NULL when the entity has no size set).
	//
	// We use UNION ALL rather than a single LEFT JOIN with CASE because
	// SQLite's query planner can use the entity-type index on each sub-query
	// independently, while a single query with CASE would require a full scan
	// of sprint_assignments to evaluate the CASE expression.
	query := `
		WITH entity_sizes AS (
			SELECT sa.sprint_id,
			       COALESCE(t.size, 0)  AS sz,
			       CASE WHEN t.size IS NULL THEN 1 ELSE 0 END AS is_unsized
			FROM sprint_assignments sa
			JOIN tasks t ON t.id = sa.entity_id AND sa.entity_type = 'task'

			UNION ALL

			SELECT sa.sprint_id,
			       COALESCE(b.size, 0)  AS sz,
			       CASE WHEN b.size IS NULL THEN 1 ELSE 0 END AS is_unsized
			FROM sprint_assignments sa
			JOIN bugs b ON b.id = sa.entity_id AND sa.entity_type = 'bug'

			UNION ALL

			SELECT sa.sprint_id,
			       COALESCE(cc.size, 0) AS sz,
			       CASE WHEN cc.size IS NULL THEN 1 ELSE 0 END AS is_unsized
			FROM sprint_assignments sa
			JOIN change_cards cc ON cc.id = sa.entity_id AND sa.entity_type = 'change_card'

			UNION ALL

			SELECT sa.sprint_id,
			       COALESCE(td.size, 0) AS sz,
			       CASE WHEN td.size IS NULL THEN 1 ELSE 0 END AS is_unsized
			FROM sprint_assignments sa
			JOIN tech_debts td ON td.id = sa.entity_id AND sa.entity_type = 'tech_debt'
		),
		-- Identify the last N completed sprints (newest first), then sort oldest-first.
		recent_completed AS (
			SELECT id, key, name
			FROM sprints
			WHERE status = 'completed'
			ORDER BY end_date DESC
			LIMIT ?
		)
		SELECT rc.key, rc.name,
		       COALESCE(SUM(es.sz),          0) AS completed_size,
		       COALESCE(SUM(es.is_unsized),   0) AS unsized_completed
		FROM recent_completed rc
		LEFT JOIN entity_sizes es ON es.sprint_id = rc.id
		GROUP BY rc.id, rc.key, rc.name
		ORDER BY (
			SELECT end_date FROM sprints WHERE id = rc.id
		) ASC
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get velocity data: %w", err)
	}
	defer rows.Close()

	var result []VelocityRow
	for rows.Next() {
		var row VelocityRow
		if err := rows.Scan(&row.SprintKey, &row.SprintName, &row.CompletedSize, &row.UnsizedCompleted); err != nil {
			return nil, fmt.Errorf("failed to scan velocity row: %w", err)
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate velocity rows: %w", err)
	}

	if result == nil {
		return []VelocityRow{}, nil
	}
	return result, nil
}

// GetSprintAssignedEntities returns ALL assignment rows for the given sprint,
// including soft-deleted rows (removed_at IS NOT NULL). The Size field is
// populated from the corresponding entity table and is nil when the entity has
// no size set (size IS NULL in the entity table).
//
// Used by the service layer for burndown reconstruction and sprint summary
// calculations. The caller is responsible for filtering active vs. removed
// assignments using the RemovedAt field.
//
// Query design (TC-NF-02): filters on sprint_assignments.sprint_id which is
// covered by idx_sprint_assignments_sprint.
func (r *SprintAnalyticsRepository) GetSprintAssignedEntities(ctx context.Context, sprintID int64) ([]AssignedEntity, error) {
	// We use a UNION ALL to resolve the polymorphic entity_type → size join.
	// Each branch handles one entity type and LEFT JOINs to the entity table.
	// The outer UNION collects all entity types for the sprint.
	//
	// Using CASE WHEN ... END in a single query would also work, but UNION ALL
	// per entity type enables the query planner to use the (entity_type, entity_id)
	// composite index on each branch independently.
	query := `
		SELECT sa.entity_type, sa.entity_id, sa.assigned_at, sa.removed_at, t.size
		FROM sprint_assignments sa
		JOIN tasks t ON t.id = sa.entity_id
		WHERE sa.sprint_id = ? AND sa.entity_type = 'task'

		UNION ALL

		SELECT sa.entity_type, sa.entity_id, sa.assigned_at, sa.removed_at, b.size
		FROM sprint_assignments sa
		JOIN bugs b ON b.id = sa.entity_id
		WHERE sa.sprint_id = ? AND sa.entity_type = 'bug'

		UNION ALL

		SELECT sa.entity_type, sa.entity_id, sa.assigned_at, sa.removed_at, cc.size
		FROM sprint_assignments sa
		JOIN change_cards cc ON cc.id = sa.entity_id
		WHERE sa.sprint_id = ? AND sa.entity_type = 'change_card'

		UNION ALL

		SELECT sa.entity_type, sa.entity_id, sa.assigned_at, sa.removed_at, td.size
		FROM sprint_assignments sa
		JOIN tech_debts td ON td.id = sa.entity_id
		WHERE sa.sprint_id = ? AND sa.entity_type = 'tech_debt'
	`

	rows, err := r.db.QueryContext(ctx, query, sprintID, sprintID, sprintID, sprintID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sprint assigned entities: %w", err)
	}
	defer rows.Close()

	var result []AssignedEntity
	for rows.Next() {
		var e AssignedEntity
		if err := rows.Scan(
			&e.EntityType,
			&e.EntityID,
			flexTime{&e.AssignedAt},
			flexNullTime{&e.RemovedAt},
			&e.Size,
		); err != nil {
			return nil, fmt.Errorf("failed to scan assigned entity: %w", err)
		}
		result = append(result, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate assigned entity rows: %w", err)
	}

	if result == nil {
		return []AssignedEntity{}, nil
	}
	return result, nil
}

// GetCompletionEvents returns task_history rows that represent "entity completed"
// status transitions for all tasks assigned to the sprint, filtered to events
// whose timestamp falls within [startDate, endDate] (inclusive).
//
// Non-task entities (bugs, change_cards, tech_debts) do not have a task_history
// equivalent. Their completion is handled by the service layer via point-in-time
// current status checks (see spec §4.3 Decision 2). This function returns only
// task events.
//
// The new_status field is returned so the service layer can apply its own
// definition of "terminal" statuses (workflow-config-driven, not hardcoded here).
//
// Query design (TC-NF-02): filters on sprint_assignments.sprint_id which is
// covered by idx_sprint_assignments_sprint. The task_history join is covered by
// idx_task_history_task_id.
func (r *SprintAnalyticsRepository) GetCompletionEvents(
	ctx context.Context,
	sprintID int64,
	startDate, endDate time.Time,
) ([]TaskCompletionEvent, error) {
	// Join sprint_assignments (filtered to task type) with task_history on
	// task_id. Filter history events to within the sprint window.
	//
	// idx_sprint_assignments_sprint covers the sprint_id filter.
	// idx_task_history_task_id covers the task_history join.
	query := `
		SELECT th.task_id, 'task' AS entity_type, th.new_status, th.timestamp
		FROM sprint_assignments sa
		JOIN task_history th ON th.task_id = sa.entity_id
		WHERE sa.sprint_id    = ?
		  AND sa.entity_type  = 'task'
		  AND th.timestamp   >= ?
		  AND th.timestamp   <= ?
		ORDER BY th.timestamp ASC
	`

	rows, err := r.db.QueryContext(ctx, query, sprintID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get completion events: %w", err)
	}
	defer rows.Close()

	var result []TaskCompletionEvent
	for rows.Next() {
		var ev TaskCompletionEvent
		if err := rows.Scan(
			&ev.EntityID,
			&ev.EntityType,
			&ev.NewStatus,
			&ev.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("failed to scan completion event: %w", err)
		}
		result = append(result, ev)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate completion event rows: %w", err)
	}

	if result == nil {
		return []TaskCompletionEvent{}, nil
	}
	return result, nil
}

// GetCycleTimeByPhase returns the average time (in days) spent in each workflow
// phase for tasks assigned to the sprint, computed from consecutive task_history
// transitions. The Phase field is the old_status of each transition (i.e., the
// status the task was leaving).
//
// When the sprint has no task_history rows (e.g., work_sessions unavailable or
// tasks were never transitioned), the function returns an empty slice — not an
// error. The service layer interprets an empty slice as "no cycle-time data
// available" and sets CycleTimeByPhase = nil in the summary result (TC-S-06).
//
// Average is computed as: AVG(julianday(next.timestamp) - julianday(prev.timestamp))
// using SQLite's julianday() to get fractional days.
func (r *SprintAnalyticsRepository) GetCycleTimeByPhase(ctx context.Context, sprintID int64) ([]PhaseTimeRow, error) {
	// For each task assigned to the sprint, pair consecutive task_history rows
	// to get the time spent in each phase (old_status → new_status transition).
	// We use a self-join on task_history (current row and the next row for the
	// same task ordered by timestamp) to compute elapsed time per phase.
	//
	// The window function LAG/LEAD approach would be cleaner but SQLite's
	// window function support requires v3.25+. The self-join approach works on
	// all SQLite versions supported by this project.
	query := `
		WITH sprint_tasks AS (
			SELECT sa.entity_id AS task_id
			FROM sprint_assignments sa
			WHERE sa.sprint_id   = ?
			  AND sa.entity_type = 'task'
		),
		history_pairs AS (
			SELECT th1.task_id,
			       th1.old_status                                              AS phase,
			       julianday(th2.timestamp) - julianday(th1.timestamp)        AS days_elapsed
			FROM task_history th1
			JOIN task_history th2
			  ON th2.task_id   = th1.task_id
			 AND th2.timestamp = (
			       SELECT MIN(th3.timestamp)
			       FROM task_history th3
			       WHERE th3.task_id   = th1.task_id
			         AND th3.timestamp > th1.timestamp
			     )
			WHERE th1.task_id IN (SELECT task_id FROM sprint_tasks)
			  AND th1.old_status IS NOT NULL
		)
		SELECT phase,
		       AVG(days_elapsed) AS average_days
		FROM history_pairs
		GROUP BY phase
		ORDER BY phase
	`

	rows, err := r.db.QueryContext(ctx, query, sprintID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cycle time by phase: %w", err)
	}
	defer rows.Close()

	var result []PhaseTimeRow
	for rows.Next() {
		var row PhaseTimeRow
		// average_days may be NULL when AVG produces no non-NULL rows (e.g.,
		// when history events exist but the self-join produces no pairs). Use
		// a *float64 to handle the NULL case gracefully and skip rows with
		// no meaningful average.
		var avgDays *float64
		if err := rows.Scan(&row.Phase, &avgDays); err != nil {
			return nil, fmt.Errorf("failed to scan cycle time row: %w", err)
		}
		if avgDays != nil {
			row.AverageDays = *avgDays
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate cycle time rows: %w", err)
	}

	// Return empty slice (not nil) when no history exists — the service layer
	// distinguishes empty from nil to set CycleTimeByPhase correctly (TC-S-06).
	if result == nil {
		return []PhaseTimeRow{}, nil
	}
	return result, nil
}
