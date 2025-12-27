package status

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// StatusService provides dashboard and reporting functionality
type StatusService struct {
	db              *repository.DB
	epicRepo        *repository.EpicRepository
	featureRepo     *repository.FeatureRepository
	taskRepo        *repository.TaskRepository
	taskHistoryRepo *repository.TaskHistoryRepository
}

// NewStatusService creates a new StatusService instance
func NewStatusService(database *repository.DB) *StatusService {
	return &StatusService{
		db:              database,
		epicRepo:        repository.NewEpicRepository(database),
		featureRepo:     repository.NewFeatureRepository(database),
		taskRepo:        repository.NewTaskRepository(database),
		taskHistoryRepo: repository.NewTaskHistoryRepository(database),
	}
}

// GetDashboard generates a complete status dashboard based on the request
func (s *StatusService) GetDashboard(ctx context.Context, req *StatusRequest) (*StatusDashboard, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Check context
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Get project summary
	summary, err := s.getProjectSummary(ctx, req.EpicKey)
	if err != nil {
		return nil, err
	}

	// Get epic breakdown
	epics, err := s.getEpics(ctx, req.EpicKey)
	if err != nil {
		return nil, err
	}

	// Get active tasks
	activeTasks, err := s.getActiveTasks(ctx, req.EpicKey)
	if err != nil {
		return nil, err
	}

	// Get blocked tasks
	blockedTasks, err := s.getBlockedTasks(ctx, req.EpicKey)
	if err != nil {
		return nil, err
	}

	// Get recent completions
	var recentCompletions []*CompletionInfo
	if req.RecentWindow != "" {
		recentCompletions, err = s.getRecentCompletions(ctx, req.EpicKey, req.RecentWindow)
		if err != nil {
			return nil, err
		}
	}

	dashboard := &StatusDashboard{
		Summary:           summary,
		Epics:             epics,
		ActiveTasks:       activeTasks,
		BlockedTasks:      blockedTasks,
		RecentCompletions: recentCompletions,
	}

	// Add filter info if applicable
	if req.EpicKey != "" || req.RecentWindow != "" || req.IncludeArchived {
		dashboard.Filter = &DashboardFilter{
			IncludeArchived: req.IncludeArchived,
		}
		if req.EpicKey != "" {
			dashboard.Filter.EpicKey = &req.EpicKey
		}
		if req.RecentWindow != "" {
			dashboard.Filter.RecentWindow = &req.RecentWindow
		}
	}

	return dashboard, nil
}

// getProjectSummary retrieves overall project statistics
func (s *StatusService) getProjectSummary(ctx context.Context, epicKey string) (*ProjectSummary, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var epicFilter string
	var args []interface{}
	if epicKey != "" {
		epicFilter = "WHERE e.key = ?"
		args = append(args, epicKey)
	}

	query := `
		SELECT
			COUNT(DISTINCT e.id) as total_epics,
			COUNT(DISTINCT CASE WHEN e.status = 'active' THEN e.id END) as active_epics,
			COUNT(DISTINCT f.id) as total_features,
			COUNT(DISTINCT CASE WHEN f.status = 'active' THEN f.id END) as active_features,
			COUNT(DISTINCT t.id) as total_tasks,
			COUNT(DISTINCT CASE WHEN t.status = 'todo' THEN t.id END) as todo_tasks,
			COUNT(DISTINCT CASE WHEN t.status = 'in_progress' THEN t.id END) as in_progress_tasks,
			COUNT(DISTINCT CASE WHEN t.status = 'ready_for_review' THEN t.id END) as ready_for_review_tasks,
			COUNT(DISTINCT CASE WHEN t.status = 'completed' THEN t.id END) as completed_tasks,
			COUNT(DISTINCT CASE WHEN t.status = 'blocked' THEN t.id END) as blocked_tasks
		FROM epics e
		LEFT JOIN features f ON e.id = f.epic_id
		LEFT JOIN tasks t ON f.id = t.feature_id
		` + epicFilter

	var totalEpics, activeEpics, totalFeatures, activeFeatures int
	var totalTasks, todoTasks, inProgressTasks, readyForReviewTasks, completedTasks, blockedTasks int

	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&totalEpics, &activeEpics, &totalFeatures, &activeFeatures,
		&totalTasks, &todoTasks, &inProgressTasks, &readyForReviewTasks, &completedTasks, &blockedTasks,
	)
	if err != nil {
		return nil, fmt.Errorf("query project summary: %w", err)
	}

	// Calculate overall progress
	var overallProgress float64
	if totalTasks > 0 {
		overallProgress = (float64(completedTasks) / float64(totalTasks)) * 100.0
	}

	return &ProjectSummary{
		Epics:    &CountBreakdown{Total: totalEpics, Active: activeEpics},
		Features: &CountBreakdown{Total: totalFeatures, Active: activeFeatures},
		Tasks: &StatusBreakdown{
			Total:          totalTasks,
			Todo:           todoTasks,
			InProgress:     inProgressTasks,
			ReadyForReview: readyForReviewTasks,
			Completed:      completedTasks,
			Blocked:        blockedTasks,
		},
		OverallProgress: overallProgress,
		BlockedCount:    blockedTasks,
	}, nil
}

// getEpics retrieves epic breakdown with progress
func (s *StatusService) getEpics(ctx context.Context, epicKey string) ([]*EpicSummary, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var epicFilter string
	var args []interface{}
	if epicKey != "" {
		epicFilter = "WHERE e.key = ?"
		args = append(args, epicKey)
	}

	query := `
		SELECT
			e.id, e.key, e.title,
			COUNT(DISTINCT f.id) as total_features,
			SUM(CASE WHEN f.status = 'active' THEN 1 ELSE 0 END) as active_features,
			COUNT(DISTINCT t.id) as total_tasks,
			SUM(CASE WHEN t.status = 'completed' THEN 1 ELSE 0 END) as completed_tasks,
			SUM(CASE WHEN t.status = 'blocked' THEN 1 ELSE 0 END) as blocked_tasks
		FROM epics e
		LEFT JOIN features f ON e.id = f.epic_id
		LEFT JOIN tasks t ON f.id = t.feature_id
		` + epicFilter + `
		GROUP BY e.id, e.key, e.title
		ORDER BY e.key ASC
	`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query epics: %w", err)
	}
	defer rows.Close()

	var epics []*EpicSummary
	for rows.Next() {
		var id int64
		var key, title string
		var totalFeatures, activeFeatures, totalTasks, completedTasks, blockedTasks int

		if err := rows.Scan(&id, &key, &title, &totalFeatures, &activeFeatures, &totalTasks, &completedTasks, &blockedTasks); err != nil {
			return nil, fmt.Errorf("scan epic row: %w", err)
		}

		// Calculate progress
		var progress float64
		if totalTasks > 0 {
			progress = (float64(completedTasks) / float64(totalTasks)) * 100.0
		}

		// Determine health
		health := s.determineEpicHealth(progress, blockedTasks)

		epics = append(epics, &EpicSummary{
			Key:             key,
			Title:           title,
			ProgressPercent: progress,
			Health:          health,
			TasksTotal:      totalTasks,
			TasksCompleted:  completedTasks,
			TasksBlocked:    blockedTasks,
			FeaturesTotal:   totalFeatures,
			FeaturesActive:  activeFeatures,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate epic rows: %w", err)
	}

	return epics, nil
}

// getActiveTasks retrieves in-progress tasks grouped by agent type
func (s *StatusService) getActiveTasks(ctx context.Context, epicKey string) (map[string][]*TaskInfo, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var args []interface{}

	query := `
		SELECT
			t.key, t.title, t.agent_type, t.priority, t.started_at,
			f.key as feature_key, e.key as epic_key
		FROM tasks t
		JOIN features f ON t.feature_id = f.id
		JOIN epics e ON f.epic_id = e.id
		WHERE t.status = 'in_progress'
	`

	if epicKey != "" {
		query += " AND e.key = ?"
		args = append(args, epicKey)
	}

	query += " ORDER BY t.agent_type ASC, t.key ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query active tasks: %w", err)
	}
	defer rows.Close()

	groups := make(map[string][]*TaskInfo)
	for rows.Next() {
		var task TaskInfo
		var agentType sql.NullString
		var startedAt sql.NullTime
		var featureKey, epicKeyStr string

		if err := rows.Scan(&task.Key, &task.Title, &agentType, &task.Priority, &startedAt, &featureKey, &epicKeyStr); err != nil {
			return nil, fmt.Errorf("scan active task row: %w", err)
		}

		task.Feature = featureKey
		task.Epic = epicKeyStr

		agent := "unassigned"
		if agentType.Valid && agentType.String != "" {
			agent = agentType.String
			task.AgentType = &agentType.String
		}

		if startedAt.Valid {
			startedStr := startedAt.Time.Format(time.RFC3339)
			task.StartedAt = &startedStr
		}

		groups[agent] = append(groups[agent], &task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active task rows: %w", err)
	}

	return groups, nil
}

// getBlockedTasks retrieves blocked tasks
func (s *StatusService) getBlockedTasks(ctx context.Context, epicKey string) ([]*BlockedTaskInfo, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var args []interface{}

	query := `
		SELECT
			t.key, t.title, t.agent_type, t.blocked_reason, t.blocked_at,
			f.key as feature_key, e.key as epic_key
		FROM tasks t
		JOIN features f ON t.feature_id = f.id
		JOIN epics e ON f.epic_id = e.id
		WHERE t.status = 'blocked'
	`

	if epicKey != "" {
		query += " AND e.key = ?"
		args = append(args, epicKey)
	}

	query += " ORDER BY t.priority DESC, t.blocked_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query blocked tasks: %w", err)
	}
	defer rows.Close()

	var blockedTasks []*BlockedTaskInfo
	for rows.Next() {
		var task BlockedTaskInfo
		var agentType, blockedReason sql.NullString
		var blockedAt sql.NullTime
		var featureKey, epicKeyStr string

		if err := rows.Scan(&task.Key, &task.Title, &agentType, &blockedReason, &blockedAt, &featureKey, &epicKeyStr); err != nil {
			return nil, fmt.Errorf("scan blocked task row: %w", err)
		}

		task.Feature = featureKey
		task.Epic = epicKeyStr

		if agentType.Valid && agentType.String != "" {
			task.AgentType = &agentType.String
		}
		if blockedReason.Valid && blockedReason.String != "" {
			task.BlockedReason = &blockedReason.String
		}
		if blockedAt.Valid {
			blockedAtStr := blockedAt.Time.Format(time.RFC3339)
			task.BlockedAt = &blockedAtStr
		}

		blockedTasks = append(blockedTasks, &task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate blocked task rows: %w", err)
	}

	return blockedTasks, nil
}

// getRecentCompletions retrieves recently completed tasks
func (s *StatusService) getRecentCompletions(ctx context.Context, epicKey string, window string) ([]*CompletionInfo, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// For now, return empty list - will implement timeframe filtering in future iteration
	return []*CompletionInfo{}, nil
}

// determineEpicHealth calculates health status based on progress and blocked count
func (s *StatusService) determineEpicHealth(progress float64, blockedCount int) string {
	// Critical: <25% progress OR >3 blocked tasks
	if progress < 25.0 || blockedCount > 3 {
		return "critical"
	}

	// Warning: 25-74% progress OR 1-3 blocked tasks
	if progress < 75.0 || blockedCount > 0 {
		return "warning"
	}

	// Healthy: ≥75% progress AND no blocked tasks
	return "healthy"
}
