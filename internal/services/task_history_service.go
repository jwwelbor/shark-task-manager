package services

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// TaskHistoryTaskRepository defines the task repository methods needed by TaskHistoryService.
type TaskHistoryTaskRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Task, error)
}

// TaskHistoryFeatureRepository defines the feature repository methods needed by TaskHistoryService.
type TaskHistoryFeatureRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Feature, error)
}

// TaskHistoryEpicRepository defines the epic repository methods needed by TaskHistoryService.
type TaskHistoryEpicRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Epic, error)
}

// TaskHistoryService handles history retrieval and work session analytics for tasks.
// It is a focused sub-service extracted from TaskService to reduce its size.
type TaskHistoryService struct {
	historyRepo TaskHistoryRepository
	sessionRepo WorkSessionRepository
	featureRepo TaskHistoryFeatureRepository
	epicRepo    TaskHistoryEpicRepository
}

// NewTaskHistoryService creates a new TaskHistoryService.
//
// Parameters:
//   - historyRepo: task history repository (required, panics if nil)
func NewTaskHistoryService(historyRepo TaskHistoryRepository) *TaskHistoryService {
	requireNonNil(historyRepo, "TaskHistoryService requires a non-nil TaskHistoryRepository")
	return &TaskHistoryService{historyRepo: historyRepo}
}

// SetSessionRepo sets the work session repository for analytics.
func (s *TaskHistoryService) SetSessionRepo(sessionRepo WorkSessionRepository) {
	s.sessionRepo = sessionRepo
}

// SetFeatureRepo sets the feature repository for scope resolution in analytics.
func (s *TaskHistoryService) SetFeatureRepo(featureRepo TaskHistoryFeatureRepository) {
	s.featureRepo = featureRepo
}

// SetEpicRepo sets the epic repository for scope resolution in analytics.
func (s *TaskHistoryService) SetEpicRepo(epicRepo TaskHistoryEpicRepository) {
	s.epicRepo = epicRepo
}

// GetTaskHistory retrieves the complete status change history for a task.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - taskKey: the task key to get history for
//
// Returns:
//   - []*models.TaskHistory: all history records in chronological order
//   - error: if history repo not configured, or database operation fails
func (s *TaskHistoryService) GetTaskHistory(ctx context.Context, taskKey string) ([]*models.TaskHistory, error) {
	histories, err := s.historyRepo.GetHistoryByTaskKey(ctx, taskKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get task history for %s: %w", taskKey, err)
	}

	return histories, nil
}

// ListHistory retrieves task history records with optional filters.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - filters: optional filters for limiting results
//
// Returns:
//   - []*models.TaskHistory: matching history records
//   - error: if history repo not configured, or database operation fails
func (s *TaskHistoryService) ListHistory(ctx context.Context, filters HistoryFilters) ([]*models.TaskHistory, error) {
	histories, err := s.historyRepo.ListWithFilters(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list task history: %w", err)
	}

	return histories, nil
}

// GetWorkSessions retrieves work sessions and statistics for a task.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - taskKey: the task key to get sessions for
//   - taskID: the database ID of the task (avoids an extra lookup)
//   - taskTitle: the title of the task (avoids an extra lookup)
//
// Returns:
//   - *TaskWorkSessions: work sessions and aggregated statistics
//   - error: if session repo not configured, or database operation fails
func (s *TaskHistoryService) GetWorkSessions(ctx context.Context, taskKey string, taskID int64, taskTitle string) (*TaskWorkSessions, error) {
	if s.sessionRepo == nil {
		return nil, fmt.Errorf("work session repository not configured")
	}

	sessions, err := s.sessionRepo.GetByTaskID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get work sessions for %s: %w", taskKey, err)
	}

	stats, err := s.sessionRepo.GetSessionStatsByTaskID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session stats for %s: %w", taskKey, err)
	}

	return &TaskWorkSessions{
		TaskKey:   taskKey,
		TaskTitle: taskTitle,
		Sessions:  sessions,
		Stats:     stats,
	}, nil
}

// GetSessionAnalytics retrieves aggregated work session analytics for a feature or epic.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - input: specifies scope (epic or feature) and optional agent filter
//
// Returns:
//   - *SessionAnalytics: aggregated analytics data
//   - error: if session repo not configured, entity not found, or database operation fails
func (s *TaskHistoryService) GetSessionAnalytics(ctx context.Context, input SessionAnalyticsInput) (*SessionAnalytics, error) {
	if s.sessionRepo == nil {
		return nil, fmt.Errorf("work session repository not configured")
	}

	var agentTypePtr *string
	if input.AgentType != "" {
		agentTypePtr = &input.AgentType
	}

	if input.FeatureKey != "" {
		if s.featureRepo == nil {
			return nil, fmt.Errorf("feature repository not configured")
		}
		feature, err := s.featureRepo.GetByKey(ctx, input.FeatureKey)
		if err != nil {
			return nil, fmt.Errorf("feature not found: %s: %w", input.FeatureKey, err)
		}
		return s.sessionRepo.GetSessionAnalyticsByFeature(ctx, feature.ID, agentTypePtr)
	}

	if input.EpicKey != "" {
		if s.epicRepo == nil {
			return nil, fmt.Errorf("epic repository not configured")
		}
		epic, err := s.epicRepo.GetByKey(ctx, input.EpicKey)
		if err != nil {
			return nil, fmt.Errorf("epic not found: %s: %w", input.EpicKey, err)
		}
		return s.sessionRepo.GetSessionAnalyticsByEpic(ctx, epic.ID, agentTypePtr)
	}

	return nil, fmt.Errorf("either EpicKey or FeatureKey must be specified")
}
