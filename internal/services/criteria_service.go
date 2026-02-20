package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/taskfile"
)

// CriteriaRepository defines the repository interface for task acceptance criteria.
type CriteriaRepository interface {
	Create(ctx context.Context, criteria *models.TaskCriteria) error
	GetByID(ctx context.Context, id int64) (*models.TaskCriteria, error)
	GetByTaskID(ctx context.Context, taskID int64) ([]*models.TaskCriteria, error)
	Update(ctx context.Context, criteria *models.TaskCriteria) error
	UpdateStatus(ctx context.Context, id int64, status models.CriteriaStatus, notes *string) error
	Delete(ctx context.Context, id int64) error
	DeleteByTaskID(ctx context.Context, taskID int64) error
	GetSummaryByTaskID(ctx context.Context, taskID int64) (*repository.CriteriaSummary, error)
}

// CriteriaTaskRepository defines the task repository interface needed by CriteriaService.
type CriteriaTaskRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Task, error)
	ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error)
}

// CriteriaFeatureRepository defines the feature repository interface needed by CriteriaService.
type CriteriaFeatureRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Feature, error)
}

// ImportCriteriaInput contains the parameters for importing criteria to a task.
type ImportCriteriaInput struct {
	TaskKey  string
	Criteria []string // Criterion text strings to import
}

// FeatureCriteriaResult aggregates criteria for all tasks in a feature.
type FeatureCriteriaResult struct {
	FeatureKey string                  `json:"feature_key"`
	Tasks      []*TaskCriteriaResult   `json:"tasks"`
	Summary    *FeatureCriteriaSummary `json:"summary"`
}

// TaskCriteriaResult aggregates criteria for a single task.
type TaskCriteriaResult struct {
	TaskKey  string                      `json:"task_key"`
	TaskID   int64                       `json:"task_id"`
	Criteria []*models.TaskCriteria      `json:"criteria"`
	Summary  *repository.CriteriaSummary `json:"summary"`
}

// FeatureCriteriaSummary provides aggregate criteria counts across a feature.
type FeatureCriteriaSummary struct {
	TotalTasks      int     `json:"total_tasks"`
	TotalCriteria   int     `json:"total_criteria"`
	PendingCount    int     `json:"pending_count"`
	InProgressCount int     `json:"in_progress_count"`
	CompleteCount   int     `json:"complete_count"`
	FailedCount     int     `json:"failed_count"`
	NACount         int     `json:"na_count"`
	CompletionPct   float64 `json:"completion_pct"`
}

// CriteriaService provides business logic for task acceptance criteria operations.
type CriteriaService struct {
	criteriaRepo CriteriaRepository
	taskRepo     CriteriaTaskRepository
	featureRepo  CriteriaFeatureRepository
}

// NewCriteriaService creates a new CriteriaService with injected dependencies.
//
// Parameters:
//   - criteriaRepo: criteria repository for data access (required, panics if nil)
//   - taskRepo: task repository for task lookups (required, panics if nil)
//   - featureRepo: feature repository for feature lookups (optional, needed for GetFeatureCriteria)
func NewCriteriaService(
	criteriaRepo CriteriaRepository,
	taskRepo CriteriaTaskRepository,
	featureRepo CriteriaFeatureRepository,
) *CriteriaService {
	if criteriaRepo == nil {
		panic("CriteriaService requires a non-nil CriteriaRepository")
	}
	if taskRepo == nil {
		panic("CriteriaService requires a non-nil CriteriaTaskRepository")
	}
	return &CriteriaService{
		criteriaRepo: criteriaRepo,
		taskRepo:     taskRepo,
		featureRepo:  featureRepo,
	}
}

// ImportCriteria creates acceptance criteria for a task from a list of criterion strings.
// Skips empty strings. Returns the created criteria.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - input: task key and list of criterion strings
//
// Returns:
//   - []*models.TaskCriteria: the created criteria
//   - error: task not found, validation errors, or repository errors
func (s *CriteriaService) ImportCriteria(ctx context.Context, input ImportCriteriaInput) ([]*models.TaskCriteria, error) {
	if input.TaskKey == "" {
		return nil, fmt.Errorf("task key is required")
	}

	task, err := s.taskRepo.GetByKey(ctx, input.TaskKey)
	if err != nil {
		return nil, fmt.Errorf("failed to find task %s: %w", input.TaskKey, err)
	}

	var created []*models.TaskCriteria
	for _, text := range input.Criteria {
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}

		criterion := &models.TaskCriteria{
			TaskID:    task.ID,
			Criterion: text,
			Status:    models.CriteriaStatusPending,
		}

		if err := s.criteriaRepo.Create(ctx, criterion); err != nil {
			return nil, fmt.Errorf("failed to create criterion for task %s: %w", input.TaskKey, err)
		}

		created = append(created, criterion)
	}

	return created, nil
}

// ListCriteria returns all acceptance criteria for a task identified by key.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - taskKey: the task key (e.g., "E07-F01-001")
//
// Returns:
//   - []*models.TaskCriteria: the task's acceptance criteria (may be empty)
//   - error: task not found, or repository errors
func (s *CriteriaService) ListCriteria(ctx context.Context, taskKey string) ([]*models.TaskCriteria, error) {
	task, err := s.taskRepo.GetByKey(ctx, taskKey)
	if err != nil {
		return nil, fmt.Errorf("failed to find task %s: %w", taskKey, err)
	}

	criteria, err := s.criteriaRepo.GetByTaskID(ctx, task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list criteria for task %s: %w", taskKey, err)
	}

	return criteria, nil
}

// CheckCriterion marks an acceptance criterion as complete.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - criterionID: database ID of the criterion to mark complete
//   - notes: optional verification notes
//
// Returns:
//   - error: criterion not found, or repository errors
func (s *CriteriaService) CheckCriterion(ctx context.Context, criterionID int64, notes *string) error {
	if criterionID <= 0 {
		return fmt.Errorf("criterion ID must be positive")
	}

	if err := s.criteriaRepo.UpdateStatus(ctx, criterionID, models.CriteriaStatusComplete, notes); err != nil {
		return fmt.Errorf("failed to check criterion %d: %w", criterionID, err)
	}

	return nil
}

// FailCriterion marks an acceptance criterion as failed.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - criterionID: database ID of the criterion to mark failed
//   - notes: optional failure notes
//
// Returns:
//   - error: criterion not found, or repository errors
func (s *CriteriaService) FailCriterion(ctx context.Context, criterionID int64, notes *string) error {
	if criterionID <= 0 {
		return fmt.Errorf("criterion ID must be positive")
	}

	if err := s.criteriaRepo.UpdateStatus(ctx, criterionID, models.CriteriaStatusFailed, notes); err != nil {
		return fmt.Errorf("failed to fail criterion %d: %w", criterionID, err)
	}

	return nil
}

// TaskCriteriaWithSummary contains a task's criteria and their aggregate summary.
type TaskCriteriaWithSummary struct {
	TaskKey  string                      `json:"task_key"`
	Title    string                      `json:"title"`
	Criteria []*models.TaskCriteria      `json:"criteria"`
	Summary  *repository.CriteriaSummary `json:"summary"`
}

// CriterionUpdateResult contains the result after updating a criterion status.
type CriterionUpdateResult struct {
	TaskKey     string                      `json:"task_key"`
	CriterionID int64                       `json:"criterion_id"`
	Status      models.CriteriaStatus       `json:"status"`
	Summary     *repository.CriteriaSummary `json:"summary"`
}

// ImportCriteriaFromFile reads criteria from a task's markdown file and imports them.
// Returns the number of imported criteria.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - taskKey: the task key to import criteria for
//
// Returns:
//   - int: number of criteria imported
//   - error: task not found, no file path, file parse errors, or repository errors
func (s *CriteriaService) ImportCriteriaFromFile(ctx context.Context, taskKey string) (int, error) {
	task, err := s.taskRepo.GetByKey(ctx, taskKey)
	if err != nil {
		return 0, fmt.Errorf("failed to find task %s: %w", taskKey, err)
	}

	if task.FilePath == nil || *task.FilePath == "" {
		return 0, fmt.Errorf("task %s has no file path", taskKey)
	}

	items, err := taskfile.ParseCriteriaFromFile(*task.FilePath)
	if err != nil {
		return 0, fmt.Errorf("failed to parse criteria from file: %w", err)
	}

	importCount := 0
	for _, item := range items {
		criterion := &models.TaskCriteria{
			TaskID:    task.ID,
			Criterion: item.Criterion,
			Status:    item.Status,
		}
		if err := s.criteriaRepo.Create(ctx, criterion); err != nil {
			return importCount, fmt.Errorf("failed to import criterion for task %s: %w", taskKey, err)
		}
		importCount++
	}

	return importCount, nil
}

// GetTaskCriteriaWithSummary returns all acceptance criteria and a summary for a task.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - taskKey: the task key (e.g., "E07-F01-001")
//
// Returns:
//   - *TaskCriteriaWithSummary: the task's criteria and summary
//   - error: task not found, or repository errors
func (s *CriteriaService) GetTaskCriteriaWithSummary(ctx context.Context, taskKey string) (*TaskCriteriaWithSummary, error) {
	task, err := s.taskRepo.GetByKey(ctx, taskKey)
	if err != nil {
		return nil, fmt.Errorf("failed to find task %s: %w", taskKey, err)
	}

	criteria, err := s.criteriaRepo.GetByTaskID(ctx, task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list criteria for task %s: %w", taskKey, err)
	}

	summary, err := s.criteriaRepo.GetSummaryByTaskID(ctx, task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get criteria summary for task %s: %w", taskKey, err)
	}

	return &TaskCriteriaWithSummary{
		TaskKey:  taskKey,
		Title:    task.Title,
		Criteria: criteria,
		Summary:  summary,
	}, nil
}

// CheckCriterionForTask marks a criterion as complete, verifying it belongs to the given task.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - taskKey: the task key the criterion must belong to
//   - criterionID: database ID of the criterion to mark complete
//   - notes: optional verification notes
//
// Returns:
//   - *CriterionUpdateResult: the result including the updated summary
//   - error: task not found, criterion not found, ownership mismatch, or repository errors
func (s *CriteriaService) CheckCriterionForTask(ctx context.Context, taskKey string, criterionID int64, notes *string) (*CriterionUpdateResult, error) {
	task, err := s.taskRepo.GetByKey(ctx, taskKey)
	if err != nil {
		return nil, fmt.Errorf("failed to find task %s: %w", taskKey, err)
	}

	criterion, err := s.criteriaRepo.GetByID(ctx, criterionID)
	if err != nil {
		return nil, fmt.Errorf("criterion %d not found: %w", criterionID, err)
	}
	if criterion.TaskID != task.ID {
		return nil, fmt.Errorf("criterion %d does not belong to task %s", criterionID, taskKey)
	}

	if err := s.criteriaRepo.UpdateStatus(ctx, criterionID, models.CriteriaStatusComplete, notes); err != nil {
		return nil, fmt.Errorf("failed to mark criterion %d as complete: %w", criterionID, err)
	}

	summary, err := s.criteriaRepo.GetSummaryByTaskID(ctx, task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated summary for task %s: %w", taskKey, err)
	}

	return &CriterionUpdateResult{
		TaskKey:     taskKey,
		CriterionID: criterionID,
		Status:      models.CriteriaStatusComplete,
		Summary:     summary,
	}, nil
}

// FailCriterionForTask marks a criterion as failed, verifying it belongs to the given task.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - taskKey: the task key the criterion must belong to
//   - criterionID: database ID of the criterion to mark failed
//   - notes: optional failure notes
//
// Returns:
//   - *CriterionUpdateResult: the result including the updated summary
//   - error: task not found, criterion not found, ownership mismatch, or repository errors
func (s *CriteriaService) FailCriterionForTask(ctx context.Context, taskKey string, criterionID int64, notes *string) (*CriterionUpdateResult, error) {
	task, err := s.taskRepo.GetByKey(ctx, taskKey)
	if err != nil {
		return nil, fmt.Errorf("failed to find task %s: %w", taskKey, err)
	}

	criterion, err := s.criteriaRepo.GetByID(ctx, criterionID)
	if err != nil {
		return nil, fmt.Errorf("criterion %d not found: %w", criterionID, err)
	}
	if criterion.TaskID != task.ID {
		return nil, fmt.Errorf("criterion %d does not belong to task %s", criterionID, taskKey)
	}

	if err := s.criteriaRepo.UpdateStatus(ctx, criterionID, models.CriteriaStatusFailed, notes); err != nil {
		return nil, fmt.Errorf("failed to mark criterion %d as failed: %w", criterionID, err)
	}

	summary, err := s.criteriaRepo.GetSummaryByTaskID(ctx, task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated summary for task %s: %w", taskKey, err)
	}

	return &CriterionUpdateResult{
		TaskKey:     taskKey,
		CriterionID: criterionID,
		Status:      models.CriteriaStatusFailed,
		Summary:     summary,
	}, nil
}

// GetFeatureCriteria aggregates all acceptance criteria for all tasks in a feature.
// Requires featureRepo to be set (non-nil).
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - featureKey: the feature key (e.g., "E07-F01")
//
// Returns:
//   - *FeatureCriteriaResult: aggregated criteria and summary across all tasks in the feature
//   - error: feature not found, featureRepo nil, or repository errors
func (s *CriteriaService) GetFeatureCriteria(ctx context.Context, featureKey string) (*FeatureCriteriaResult, error) {
	if s.featureRepo == nil {
		return nil, fmt.Errorf("feature repository is required for GetFeatureCriteria")
	}

	feature, err := s.featureRepo.GetByKey(ctx, featureKey)
	if err != nil {
		return nil, fmt.Errorf("failed to find feature %s: %w", featureKey, err)
	}

	tasks, err := s.taskRepo.ListByFeature(ctx, feature.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks for feature %s: %w", featureKey, err)
	}

	result := &FeatureCriteriaResult{
		FeatureKey: featureKey,
		Tasks:      make([]*TaskCriteriaResult, 0, len(tasks)),
		Summary: &FeatureCriteriaSummary{
			TotalTasks: len(tasks),
		},
	}

	for _, task := range tasks {
		criteria, err := s.criteriaRepo.GetByTaskID(ctx, task.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get criteria for task %s: %w", task.Key, err)
		}

		summary, err := s.criteriaRepo.GetSummaryByTaskID(ctx, task.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get criteria summary for task %s: %w", task.Key, err)
		}

		taskResult := &TaskCriteriaResult{
			TaskKey:  task.Key,
			TaskID:   task.ID,
			Criteria: criteria,
			Summary:  summary,
		}
		result.Tasks = append(result.Tasks, taskResult)

		// Aggregate into feature-level summary
		result.Summary.TotalCriteria += summary.TotalCount
		result.Summary.PendingCount += summary.PendingCount
		result.Summary.InProgressCount += summary.InProgressCount
		result.Summary.CompleteCount += summary.CompleteCount
		result.Summary.FailedCount += summary.FailedCount
		result.Summary.NACount += summary.NACount
	}

	// Calculate feature-level completion percentage
	if result.Summary.TotalCriteria > 0 {
		result.Summary.CompletionPct = float64(result.Summary.CompleteCount+result.Summary.NACount) /
			float64(result.Summary.TotalCriteria) * 100.0
	}

	return result, nil
}
