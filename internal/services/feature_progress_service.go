package services

import (
	"context"
	"fmt"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// FeatureProgressRepo defines the minimal feature repository interface needed by FeatureProgressService.
// This is a subset of FeatureRepository focused on progress/breakdown operations.
type FeatureProgressRepo interface {
	GetByKey(ctx context.Context, key string) (*models.Feature, error)
	GetByID(ctx context.Context, id int64) (*models.Feature, error)
	Update(ctx context.Context, feature *models.Feature) error
	GetTaskStatusBreakdown(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error)
	GetTaskCount(ctx context.Context, featureID int64) (int, error)
}

// FeatureProgressTaskRepo defines the minimal task repository interface needed by FeatureProgressService.
// This is a subset of FeatureTaskCounter focused on listing and batch queries.
type FeatureProgressTaskRepo interface {
	ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error)
	GetStatusBreakdownMapBatch(ctx context.Context, featureIDs []int64) (map[int64]map[models.TaskStatus]int, error)
	GetTaskCountsForFeatures(ctx context.Context, featureIDs []int64) (map[int64]int, error)
}

// FeatureProgressService handles progress calculation, health analysis, and status breakdown
// for features. It is a focused sub-service extracted from FeatureService to implement SRP.
type FeatureProgressService struct {
	repo        FeatureProgressRepo
	taskRepo    FeatureProgressTaskRepo
	workflowSvc *workflow.Service
}

// NewFeatureProgressService creates a new FeatureProgressService.
// taskRepo may be nil for graceful degradation on task-dependent methods.
func NewFeatureProgressService(repo FeatureProgressRepo, taskRepo FeatureProgressTaskRepo, workflowSvc *workflow.Service) *FeatureProgressService {
	requireNonNil(repo, "FeatureProgressService requires a non-nil FeatureProgressRepo")
	requireNonNil(workflowSvc, "FeatureProgressService requires a non-nil workflow.Service")
	return &FeatureProgressService{
		repo:        repo,
		taskRepo:    taskRepo,
		workflowSvc: workflowSvc,
	}
}

// GetProgress retrieves progress metrics for a feature.
// Computes both weighted progress (using workflow config progress weights)
// and completion progress (raw task completion percentage).
func (s *FeatureProgressService) GetProgress(ctx context.Context, key string) (*FeatureProgressInfo, error) {
	feature, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature %s: %w", key, err)
	}
	if feature == nil {
		return nil, fmt.Errorf("feature not found: %s", key)
	}

	return s.calculateProgressForFeature(ctx, key, feature.ID)
}

// calculateProgressForFeature computes progress metrics for a feature by its ID.
// This is the single source of truth for feature progress calculation.
func (s *FeatureProgressService) calculateProgressForFeature(ctx context.Context, key string, featureID int64) (*FeatureProgressInfo, error) {
	statusBreakdown, err := s.repo.GetTaskStatusBreakdown(ctx, featureID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task status breakdown for feature %s: %w", key, err)
	}

	return calculateFeatureProgressInfo(key, statusBreakdown, s.workflowSvc), nil
}

// RecalculateAndSetProgress recalculates the cached progress_pct for a feature
// and persists it. When weighted progress reaches 100%, cascade statuses may
// advance one workflow-configured step; when progress drops below 100%,
// completed features reopen back to the aggregation status.
func (s *FeatureProgressService) RecalculateAndSetProgress(ctx context.Context, featureID int64) error {
	feature, err := s.repo.GetByID(ctx, featureID)
	if err != nil {
		return fmt.Errorf("failed to get feature by ID %d: %w", featureID, err)
	}
	if feature == nil {
		return fmt.Errorf("feature not found with id %d", featureID)
	}

	progressInfo, err := s.calculateProgressForFeature(ctx, feature.Key, featureID)
	if err != nil {
		return fmt.Errorf("failed to calculate progress for feature %d: %w", featureID, err)
	}

	feature.ProgressPct = progressInfo.WeightedProgress
	derivedStatus, err := s.deriveFeatureProgressStatus(feature, progressInfo)
	if err != nil {
		return fmt.Errorf("failed to derive status for feature %d: %w", featureID, err)
	}
	feature.Status = derivedStatus

	if err := s.repo.Update(ctx, feature); err != nil {
		return fmt.Errorf("failed to update feature progress: %w", err)
	}

	return nil
}

// RecalculateAndSetProgressByKey recalculates progress for a feature identified by key.
func (s *FeatureProgressService) RecalculateAndSetProgressByKey(ctx context.Context, key string) error {
	feature, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get feature %s: %w", key, err)
	}
	if feature == nil {
		return fmt.Errorf("feature not found: %s", key)
	}

	return s.RecalculateAndSetProgress(ctx, feature.ID)
}

// deriveFeatureProgressStatus returns the status that should be persisted after
// a progress recalculation.
//
// Rules:
//   - status_override=true: never touch the status
//   - no tasks: leave the status unchanged
//   - weighted progress >= 100%: advance one workflow step only from cascade statuses
//   - weighted progress < 100%: reopen completed features to the aggregation status
//   - other terminal statuses (e.g. archived) are preserved
func (s *FeatureProgressService) deriveFeatureProgressStatus(feature *models.Feature, progressInfo *FeatureProgressInfo) (models.FeatureStatus, error) {
	return deriveFeatureProgressStatus(feature, progressInfo, s.workflowSvc)
}

// GetTaskCounts returns the total task count for each of the given feature IDs in a
// single batch query. Degrades gracefully if taskRepo is nil (returns empty map).
func (s *FeatureProgressService) GetTaskCounts(ctx context.Context, featureIDs []int64) (map[int64]int, error) {
	if s.taskRepo == nil || len(featureIDs) == 0 {
		return map[int64]int{}, nil
	}
	return s.taskRepo.GetTaskCountsForFeatures(ctx, featureIDs)
}

// GetTaskStatusBreakdown returns the count of tasks per status for a feature.
func (s *FeatureProgressService) GetTaskStatusBreakdown(ctx context.Context, key string) (map[string]int, error) {
	feature, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature %s: %w", key, err)
	}
	if feature == nil {
		return nil, fmt.Errorf("feature not found: %s", key)
	}

	breakdown, err := s.repo.GetTaskStatusBreakdown(ctx, feature.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task status breakdown for feature %s: %w", key, err)
	}

	// Convert map[models.TaskStatus]int to map[string]int
	result := make(map[string]int, len(breakdown))
	for k, v := range breakdown {
		result[string(k)] = v
	}
	return result, nil
}

// GetTaskStatusBreakdownByFeatureID returns the enriched task status breakdown for a feature
// using its database ID directly, avoiding a redundant key-based lookup.
func (s *FeatureProgressService) GetTaskStatusBreakdownByFeatureID(ctx context.Context, featureID int64) ([]workflow.StatusCount, error) {
	rawBreakdown, err := s.repo.GetTaskStatusBreakdown(ctx, featureID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task status breakdown for feature ID %d: %w", featureID, err)
	}

	// Convert to map[string]int for NewStatusBreakdown
	counts := make(map[string]int, len(rawBreakdown))
	for k, v := range rawBreakdown {
		counts[string(k)] = v
	}

	// Use task-level workflow service to enrich with phase/color/order metadata
	taskWorkflowSvc := s.workflowSvc.ForLevel(workflow.LevelTask)
	breakdown := workflow.NewStatusBreakdown(counts, taskWorkflowSvc)
	return breakdown.Counts, nil
}

// GetEnrichedTaskStatusBreakdown returns task status counts for a feature,
// enriched with workflow metadata (phase, color, order) from the task-level workflow.
func (s *FeatureProgressService) GetEnrichedTaskStatusBreakdown(ctx context.Context, key string) ([]workflow.StatusCount, error) {
	feature, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature %s: %w", key, err)
	}
	if feature == nil {
		return nil, fmt.Errorf("feature not found: %s", key)
	}

	rawBreakdown, err := s.repo.GetTaskStatusBreakdown(ctx, feature.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task status breakdown for feature %s: %w", key, err)
	}

	// Convert to map[string]int for NewStatusBreakdown
	counts := make(map[string]int, len(rawBreakdown))
	for k, v := range rawBreakdown {
		counts[string(k)] = v
	}

	// Use task-level workflow service to enrich with phase/color/order metadata
	taskWorkflowSvc := s.workflowSvc.ForLevel(workflow.LevelTask)
	breakdown := workflow.NewStatusBreakdown(counts, taskWorkflowSvc)
	return breakdown.Counts, nil
}

// GetStatusBreakdownBatch fetches task status breakdowns for multiple features in one query.
// Returns a map of featureID -> (taskStatus -> count).
// Returns nil, nil if taskRepo is not available (graceful degradation).
func (s *FeatureProgressService) GetStatusBreakdownBatch(ctx context.Context, featureIDs []int64) (map[int64]map[models.TaskStatus]int, error) {
	if s.taskRepo == nil {
		return nil, nil
	}
	return s.taskRepo.GetStatusBreakdownMapBatch(ctx, featureIDs)
}

// GetHealth analyzes the health of a feature based on blocked tasks and approval age.
// Degrades gracefully if taskRepo is nil (returns healthy).
func (s *FeatureProgressService) GetHealth(ctx context.Context, key string) (*FeatureHealthInfo, error) {
	feature, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature %s: %w", key, err)
	}
	if feature == nil {
		return nil, fmt.Errorf("feature not found: %s", key)
	}

	health := &FeatureHealthInfo{
		FeatureKey: key,
		Status:     "healthy",
	}

	if s.taskRepo == nil {
		return health, nil
	}

	tasks, err := s.taskRepo.ListByFeature(ctx, feature.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks for feature %s: %w", key, err)
	}

	// Count blocked tasks
	var blockedTasks []*models.Task
	for _, t := range tasks {
		if string(t.Status) == "blocked" {
			blockedTasks = append(blockedTasks, t)
		}
	}

	if len(blockedTasks) >= 2 {
		health.Status = "critical"
		health.Reasons = append(health.Reasons, fmt.Sprintf("%d blocked tasks", len(blockedTasks)))
	} else if len(blockedTasks) == 1 {
		health.Status = "warning"
		health.Reasons = append(health.Reasons, "1 blocked task")
	}

	// Check for high-priority blocked tasks (priority 1-3 is high)
	for _, t := range blockedTasks {
		if t.Priority <= 3 && health.Status != "critical" {
			health.Status = "critical"
			health.Reasons = append(health.Reasons, fmt.Sprintf("high-priority task %s is blocked", t.Key))
		}
	}

	// Check for old approval tasks
	now := time.Now()
	for _, t := range tasks {
		meta := s.workflowSvc.GetStatusMetadata(string(t.Status))
		if meta.Phase == "approval" || meta.Phase == "review" {
			ageDays := int(now.Sub(t.UpdatedAt).Hours() / 24)
			if ageDays > 3 && health.Status == "healthy" {
				health.Status = "warning"
				health.Reasons = append(health.Reasons, fmt.Sprintf("task %s awaiting approval for %d days", t.Key, ageDays))
			}
		}
	}

	return health, nil
}

// GetWorkBreakdown categorizes remaining work by responsibility using workflow config.
func (s *FeatureProgressService) GetWorkBreakdown(ctx context.Context, key string) (*WorkBreakdown, error) {
	feature, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature %s: %w", key, err)
	}
	if feature == nil {
		return nil, fmt.Errorf("feature not found: %s", key)
	}

	statusBreakdown, err := s.repo.GetTaskStatusBreakdown(ctx, feature.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task status breakdown for feature %s: %w", key, err)
	}

	wf := s.workflowSvc.GetWorkflow()

	wb := &WorkBreakdown{FeatureKey: key}

	for status, count := range statusBreakdown {
		statusStr := string(status)

		// Determine responsibility from workflow config
		responsibility := "none"
		if wf != nil && wf.StatusMetadata != nil {
			if meta, found := wf.StatusMetadata[statusStr]; found {
				// Skip statuses excluded from progress (e.g., cancelled)
				if meta.ExcludeFromProgress {
					continue
				}
				if meta.Responsibility != "" {
					responsibility = meta.Responsibility
				}
			}
		}

		wb.TotalTasks += count

		// Check if terminal (completed)
		if s.workflowSvc.IsTerminalStatus(statusStr) {
			wb.CompletedTasks += count
			continue
		}

		// Check if blocked
		if statusStr == "blocked" {
			wb.BlockedWork += count
			continue
		}

		// Categorize by responsibility
		switch responsibility {
		case "agent":
			wb.AgentWork += count
		case "human", "qa_team":
			wb.HumanWork += count
		default:
			wb.NotStarted += count
		}
	}

	return wb, nil
}

// GetActionItems returns tasks requiring immediate attention for a feature.
// Groups tasks into awaiting_approval, blocked, and in_progress categories.
// Degrades gracefully if taskRepo is nil (returns empty result).
func (s *FeatureProgressService) GetActionItems(ctx context.Context, key string) (*FeatureActionItems, error) {
	feature, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature %s: %w", key, err)
	}
	if feature == nil {
		return nil, fmt.Errorf("feature not found: %s", key)
	}

	result := &FeatureActionItems{FeatureKey: key}

	if s.taskRepo == nil {
		return result, nil
	}

	tasks, err := s.taskRepo.ListByFeature(ctx, feature.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks for feature %s: %w", key, err)
	}

	now := time.Now()
	for _, t := range tasks {
		statusStr := string(t.Status)
		meta := s.workflowSvc.GetStatusMetadata(statusStr)

		item := &ActionTaskItem{
			TaskKey:   t.Key,
			Title:     t.Title,
			Status:    statusStr,
			UpdatedAt: t.UpdatedAt,
		}

		// Categorize by status phase and characteristics
		if statusStr == "blocked" {
			ageDays := int(now.Sub(t.UpdatedAt).Hours() / 24)
			item.AgeDays = &ageDays
			result.Blocked = append(result.Blocked, item)
		} else if meta.Phase == "approval" || meta.Phase == "review" {
			ageDays := int(now.Sub(t.UpdatedAt).Hours() / 24)
			item.AgeDays = &ageDays
			result.AwaitingApproval = append(result.AwaitingApproval, item)
		} else if meta.Phase == "development" || meta.Phase == "execution" {
			result.InProgress = append(result.InProgress, item)
		}
	}

	return result, nil
}
