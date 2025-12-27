package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// WorkSessionRepository handles CRUD operations for work sessions
type WorkSessionRepository struct {
	db *DB
}

// NewWorkSessionRepository creates a new WorkSessionRepository
func NewWorkSessionRepository(db *DB) *WorkSessionRepository {
	return &WorkSessionRepository{db: db}
}

// Create creates a new work session
func (r *WorkSessionRepository) Create(ctx context.Context, session *models.WorkSession) error {
	if err := session.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Check if there's already an active session for this task
	activeSession, err := r.GetActiveSessionByTaskID(ctx, session.TaskID)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check for active session: %w", err)
	}
	if activeSession != nil {
		return fmt.Errorf("task already has an active work session started at %s", activeSession.StartedAt.Format(time.RFC3339))
	}

	query := `
		INSERT INTO work_sessions (
			task_id, agent_id, started_at, ended_at, outcome, session_notes, context_snapshot
		)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		session.TaskID,
		session.AgentID,
		session.StartedAt,
		session.EndedAt,
		session.Outcome,
		session.SessionNotes,
		session.ContextSnapshot,
	)
	if err != nil {
		return fmt.Errorf("failed to create work session: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	session.ID = id
	return nil
}

// GetByID retrieves a work session by its ID
func (r *WorkSessionRepository) GetByID(ctx context.Context, id int64) (*models.WorkSession, error) {
	query := `
		SELECT id, task_id, agent_id, started_at, ended_at, outcome, session_notes, context_snapshot, created_at
		FROM work_sessions
		WHERE id = ?
	`

	session := &models.WorkSession{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&session.ID,
		&session.TaskID,
		&session.AgentID,
		&session.StartedAt,
		&session.EndedAt,
		&session.Outcome,
		&session.SessionNotes,
		&session.ContextSnapshot,
		&session.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("work session not found with id %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get work session: %w", err)
	}

	return session, nil
}

// GetByTaskID retrieves all work sessions for a task
func (r *WorkSessionRepository) GetByTaskID(ctx context.Context, taskID int64) ([]*models.WorkSession, error) {
	query := `
		SELECT id, task_id, agent_id, started_at, ended_at, outcome, session_notes, context_snapshot, created_at
		FROM work_sessions
		WHERE task_id = ?
		ORDER BY started_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to query work sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*models.WorkSession
	for rows.Next() {
		session := &models.WorkSession{}
		err := rows.Scan(
			&session.ID,
			&session.TaskID,
			&session.AgentID,
			&session.StartedAt,
			&session.EndedAt,
			&session.Outcome,
			&session.SessionNotes,
			&session.ContextSnapshot,
			&session.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan work session: %w", err)
		}
		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating work sessions: %w", err)
	}

	return sessions, nil
}

// GetActiveSessionByTaskID retrieves the active session for a task (ended_at IS NULL)
func (r *WorkSessionRepository) GetActiveSessionByTaskID(ctx context.Context, taskID int64) (*models.WorkSession, error) {
	query := `
		SELECT id, task_id, agent_id, started_at, ended_at, outcome, session_notes, context_snapshot, created_at
		FROM work_sessions
		WHERE task_id = ? AND ended_at IS NULL
		LIMIT 1
	`

	session := &models.WorkSession{}
	err := r.db.QueryRowContext(ctx, query, taskID).Scan(
		&session.ID,
		&session.TaskID,
		&session.AgentID,
		&session.StartedAt,
		&session.EndedAt,
		&session.Outcome,
		&session.SessionNotes,
		&session.ContextSnapshot,
		&session.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active work session: %w", err)
	}

	return session, nil
}

// EndSession ends an active work session with outcome and notes
func (r *WorkSessionRepository) EndSession(ctx context.Context, sessionID int64, outcome models.SessionOutcome, notes *string) error {
	if err := models.ValidateSessionOutcome(string(outcome)); err != nil {
		return err
	}

	query := `
		UPDATE work_sessions
		SET ended_at = ?, outcome = ?, session_notes = ?
		WHERE id = ? AND ended_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, time.Now(), outcome, notes, sessionID)
	if err != nil {
		return fmt.Errorf("failed to end work session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("work session %d not found or already ended", sessionID)
	}

	return nil
}

// Update updates a work session
func (r *WorkSessionRepository) Update(ctx context.Context, session *models.WorkSession) error {
	if err := session.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		UPDATE work_sessions
		SET task_id = ?, agent_id = ?, started_at = ?, ended_at = ?, outcome = ?, session_notes = ?, context_snapshot = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		session.TaskID,
		session.AgentID,
		session.StartedAt,
		session.EndedAt,
		session.Outcome,
		session.SessionNotes,
		session.ContextSnapshot,
		session.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update work session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("work session not found with id %d", session.ID)
	}

	return nil
}

// Delete deletes a work session
func (r *WorkSessionRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM work_sessions WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete work session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("work session not found with id %d", id)
	}

	return nil
}

// GetSessionStats returns session statistics for a task
type SessionStats struct {
	TotalSessions   int
	TotalDuration   time.Duration
	AverageDuration time.Duration
	MedianDuration  time.Duration
	ActiveSession   bool
}

// GetSessionStatsByTaskID returns session statistics for a task
func (r *WorkSessionRepository) GetSessionStatsByTaskID(ctx context.Context, taskID int64) (*SessionStats, error) {
	sessions, err := r.GetByTaskID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if len(sessions) == 0 {
		return &SessionStats{}, nil
	}

	stats := &SessionStats{
		TotalSessions: len(sessions),
	}

	var durations []time.Duration
	var totalDuration time.Duration

	for _, session := range sessions {
		if session.IsActive() {
			stats.ActiveSession = true
		} else if session.EndedAt.Valid {
			duration := session.Duration()
			durations = append(durations, duration)
			totalDuration += duration
		}
	}

	stats.TotalDuration = totalDuration

	if len(durations) > 0 {
		stats.AverageDuration = totalDuration / time.Duration(len(durations))

		// Calculate median (sort durations and take middle value)
		// For simplicity, using average for now
		// TODO: Implement proper median calculation if needed
		stats.MedianDuration = stats.AverageDuration
	}

	return stats, nil
}

// SessionAnalytics represents analytics for sessions across multiple tasks
type SessionAnalytics struct {
	TotalSessions          int
	TotalDuration          time.Duration
	AverageDuration        time.Duration
	MedianDuration         time.Duration
	TasksWithSessions      int
	TasksWithPauses        int
	AverageSessionsPerTask float64
	PauseRate              float64 // Percentage of sessions that were paused
}

// GetSessionAnalyticsByEpic returns session analytics for all tasks in an epic
func (r *WorkSessionRepository) GetSessionAnalyticsByEpic(ctx context.Context, epicID int64, agentType *string) (*SessionAnalytics, error) {
	// Build query to get all sessions for tasks in the epic
	query := `
		SELECT ws.id, ws.task_id, ws.agent_id, ws.started_at, ws.ended_at, ws.outcome, ws.session_notes, ws.context_snapshot, ws.created_at
		FROM work_sessions ws
		JOIN tasks t ON ws.task_id = t.id
		JOIN features f ON t.feature_id = f.id
		WHERE f.epic_id = ?
	`

	args := []interface{}{epicID}

	if agentType != nil {
		query += ` AND t.agent_type = ?`
		args = append(args, *agentType)
	}

	query += ` ORDER BY ws.started_at DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query work sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*models.WorkSession
	taskIDs := make(map[int64]bool)
	pausedTaskIDs := make(map[int64]bool)

	for rows.Next() {
		session := &models.WorkSession{}
		err := rows.Scan(
			&session.ID,
			&session.TaskID,
			&session.AgentID,
			&session.StartedAt,
			&session.EndedAt,
			&session.Outcome,
			&session.SessionNotes,
			&session.ContextSnapshot,
			&session.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan work session: %w", err)
		}
		sessions = append(sessions, session)
		taskIDs[session.TaskID] = true

		if session.Outcome != nil && *session.Outcome == models.SessionOutcomePaused {
			pausedTaskIDs[session.TaskID] = true
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating work sessions: %w", err)
	}

	analytics := &SessionAnalytics{
		TotalSessions:     len(sessions),
		TasksWithSessions: len(taskIDs),
		TasksWithPauses:   len(pausedTaskIDs),
	}

	if len(sessions) == 0 {
		return analytics, nil
	}

	var durations []time.Duration
	var totalDuration time.Duration
	pausedCount := 0

	for _, session := range sessions {
		if !session.IsActive() && session.EndedAt.Valid {
			duration := session.Duration()
			durations = append(durations, duration)
			totalDuration += duration
		}

		if session.Outcome != nil && *session.Outcome == models.SessionOutcomePaused {
			pausedCount++
		}
	}

	analytics.TotalDuration = totalDuration

	if len(durations) > 0 {
		analytics.AverageDuration = totalDuration / time.Duration(len(durations))
		analytics.MedianDuration = analytics.AverageDuration // Simplified
	}

	if analytics.TasksWithSessions > 0 {
		analytics.AverageSessionsPerTask = float64(len(sessions)) / float64(analytics.TasksWithSessions)
	}

	if len(sessions) > 0 {
		analytics.PauseRate = float64(pausedCount) / float64(len(sessions)) * 100.0
	}

	return analytics, nil
}

// GetSessionAnalyticsByFeature returns session analytics for all tasks in a feature
func (r *WorkSessionRepository) GetSessionAnalyticsByFeature(ctx context.Context, featureID int64, agentType *string) (*SessionAnalytics, error) {
	// Build query to get all sessions for tasks in the feature
	query := `
		SELECT ws.id, ws.task_id, ws.agent_id, ws.started_at, ws.ended_at, ws.outcome, ws.session_notes, ws.context_snapshot, ws.created_at
		FROM work_sessions ws
		JOIN tasks t ON ws.task_id = t.id
		WHERE t.feature_id = ?
	`

	args := []interface{}{featureID}

	if agentType != nil {
		query += ` AND t.agent_type = ?`
		args = append(args, *agentType)
	}

	query += ` ORDER BY ws.started_at DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query work sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*models.WorkSession
	taskIDs := make(map[int64]bool)
	pausedTaskIDs := make(map[int64]bool)

	for rows.Next() {
		session := &models.WorkSession{}
		err := rows.Scan(
			&session.ID,
			&session.TaskID,
			&session.AgentID,
			&session.StartedAt,
			&session.EndedAt,
			&session.Outcome,
			&session.SessionNotes,
			&session.ContextSnapshot,
			&session.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan work session: %w", err)
		}
		sessions = append(sessions, session)
		taskIDs[session.TaskID] = true

		if session.Outcome != nil && *session.Outcome == models.SessionOutcomePaused {
			pausedTaskIDs[session.TaskID] = true
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating work sessions: %w", err)
	}

	analytics := &SessionAnalytics{
		TotalSessions:     len(sessions),
		TasksWithSessions: len(taskIDs),
		TasksWithPauses:   len(pausedTaskIDs),
	}

	if len(sessions) == 0 {
		return analytics, nil
	}

	var durations []time.Duration
	var totalDuration time.Duration
	pausedCount := 0

	for _, session := range sessions {
		if !session.IsActive() && session.EndedAt.Valid {
			duration := session.Duration()
			durations = append(durations, duration)
			totalDuration += duration
		}

		if session.Outcome != nil && *session.Outcome == models.SessionOutcomePaused {
			pausedCount++
		}
	}

	analytics.TotalDuration = totalDuration

	if len(durations) > 0 {
		analytics.AverageDuration = totalDuration / time.Duration(len(durations))
		analytics.MedianDuration = analytics.AverageDuration // Simplified
	}

	if analytics.TasksWithSessions > 0 {
		analytics.AverageSessionsPerTask = float64(len(sessions)) / float64(analytics.TasksWithSessions)
	}

	if len(sessions) > 0 {
		analytics.PauseRate = float64(pausedCount) / float64(len(sessions)) * 100.0
	}

	return analytics, nil
}
