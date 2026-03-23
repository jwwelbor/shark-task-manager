package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/taskcreation"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// WorkSessionStats contains aggregated statistics for a task's work sessions.
type WorkSessionStats struct {
	TotalSessions   int
	TotalDuration   time.Duration
	AverageDuration time.Duration
	MedianDuration  time.Duration
	ActiveSession   bool
}

// TaskWorkSessions contains work sessions and statistics for a task.
type TaskWorkSessions struct {
	TaskKey   string
	TaskTitle string
	Sessions  []*models.WorkSession
	Stats     *WorkSessionStats
}

// WorkSessionRepository defines the interface for work session data access.
// This interface is satisfied by *repository.WorkSessionRepository (via workSessionAdapter).
type WorkSessionRepository interface {
	// GetByTaskID retrieves all work sessions for a task
	GetByTaskID(ctx context.Context, taskID int64) ([]*models.WorkSession, error)
	// GetSessionStatsByTaskID retrieves aggregated statistics for a task's sessions
	GetSessionStatsByTaskID(ctx context.Context, taskID int64) (*WorkSessionStats, error)
	// GetActiveSessionByTaskID retrieves the currently active (not yet ended) work session for a task, if any
	GetActiveSessionByTaskID(ctx context.Context, taskID int64) (*models.WorkSession, error)
	// GetSessionAnalyticsByFeature retrieves session analytics aggregated for all tasks in a feature
	GetSessionAnalyticsByFeature(ctx context.Context, featureID int64, agentType *string) (*SessionAnalytics, error)
	// GetSessionAnalyticsByEpic retrieves session analytics aggregated for all tasks in an epic
	GetSessionAnalyticsByEpic(ctx context.Context, epicID int64, agentType *string) (*SessionAnalytics, error)
}

// TaskRepository defines the repository interface needed by TaskService.
// This interface is satisfied by *repository.TaskRepository.
type TaskRepository interface {
	// CRUD operations
	Create(ctx context.Context, task *models.Task) error
	GetByKey(ctx context.Context, key string) (*models.Task, error)
	GetByID(ctx context.Context, id int64) (*models.Task, error)
	Update(ctx context.Context, task *models.Task) error
	Delete(ctx context.Context, id int64) error

	// Query operations
	List(ctx context.Context) ([]*models.Task, error)
	ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error)
	ListByFeatureKey(ctx context.Context, featureKey string) ([]*models.Task, error)
	ListByEpic(ctx context.Context, epicKey string) ([]*models.Task, error)

	// Dependency operations
	GetTaskDependents(ctx context.Context, taskKey string) ([]*models.Task, error)

	// Status operations (service layer will wrap these with business logic)
	UpdateStatus(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error
	UpdateStatusForced(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string, rejectionReason *string, documentPath *string, force bool) error
	// UpdateStatusForcedWithUnblock atomically updates status and auto-unblocks dependent tasks.
	// Returns the list of task keys that were automatically unblocked.
	UpdateStatusForcedWithUnblock(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string, rejectionReason *string, documentPath *string, force bool) ([]string, error)

	// StatusUpdateRaw performs an atomic status update without any business-logic validation.
	// It updates the task status, creates a history record, optionally creates a rejection note,
	// and auto-unblocks dependents on terminal statuses -- all in a single transaction.
	// All validation (transition validity, backward checks, reason requirements) must be
	// performed by the service layer BEFORE calling this method.
	// Returns the list of auto-unblocked task keys.
	StatusUpdateRaw(ctx context.Context, params models.StatusUpdateParams) ([]string, error)

	// Search operations
	FindByFileChanged(ctx context.Context, filePath string) ([]*models.Task, error)

	// Key prefix search - returns all tasks whose key starts with the given prefix.
	// Used for key generation to avoid UNIQUE constraint collisions.
	ListByKeyPrefix(ctx context.Context, prefix string) ([]*models.Task, error)

	// Display data - single-query aggregation via task_display_data view
	GetTaskDisplayDataRaw(ctx context.Context, taskID int64) (*repository.TaskDisplayDataRaw, error)
}

// TaskHistoryRepository defines the repository interface for task history access.
// This interface is satisfied by *repository.TaskHistoryRepository.
type TaskHistoryRepository interface {
	// GetHistoryByTaskKey retrieves all history records for a task by its key
	GetHistoryByTaskKey(ctx context.Context, taskKey string) ([]*models.TaskHistory, error)
	// ListWithFilters retrieves history records with optional filters
	ListWithFilters(ctx context.Context, filters HistoryFilters) ([]*models.TaskHistory, error)
}

// HistoryFilters defines filters for querying task history.
// This mirrors repository.HistoryFilters to avoid direct repository dependency in services.
type HistoryFilters struct {
	Agent      *string    // Filter by agent ID
	Since      *time.Time // Filter by timestamp (>= since)
	EpicKey    *string    // Filter by epic key
	FeatureKey *string    // Filter by feature key
	OldStatus  *string    // Filter by old status
	NewStatus  *string    // Filter by new status
	Limit      int        // Maximum number of records to return (default 50)
	Offset     int        // Number of records to skip for pagination
}

// TaskService provides business logic for task operations.
// It orchestrates task lifecycle, status transitions, dependency validation,
// and coordinates with workflow and taskcreation services.
// AnalyticsEpicRepository is the minimal epic repo interface needed for analytics scope resolution.
type AnalyticsEpicRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Epic, error)
}

// AnalyticsFeatureRepository is the minimal feature repo interface needed for analytics scope resolution.
type AnalyticsFeatureRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Feature, error)
}

type TaskService struct {
	repo              TaskRepository
	entitySvc         *EntityService
	creatorSvc        *taskcreation.Creator
	historyRepo       TaskHistoryRepository
	entityHistoryRepo EntityHistoryRecorder // optional: records to entity_history table
	docRepo           config.DocumentRepository
	relRepo           config.TaskRelationshipRepository // for template placeholder population (ListRelatedTaskKeys)
	sessionRepo       WorkSessionRepository
	epicRepo          AnalyticsEpicRepository
	featureRepo       AnalyticsFeatureRepository
	featureService    *FeatureService // optional: triggers progress recalc on status change
	enrichRepo        config.TemplateEnrichmentRepository
	docSvc            *EntityDocumentService // shared document operations; built by SetWritableDocRepo

	// Sub-services for delegating extracted functionality.
	// When non-nil, method calls are delegated to the sub-service instead of
	// using the inline implementations in this file.
	queryService   *TaskQueryService
	historyService *TaskHistoryService

	tracer trace.Tracer // optional; defaults to otel.Tracer("shark/services/task") if nil
}

// NewTaskService creates a new TaskService with the required dependencies.
// The entitySvc provides shared transition logic; it is automatically scoped to task level.
// creatorSvc, docRepo, and relRepo can be nil for graceful degradation.
// Rejection note creation is handled by EntityService (via SetNoteRepo).
//
// Parameters:
//   - repo: task repository for data access (required, panics if nil)
//   - entitySvc: entity service for shared validation and transition helpers (required, panics if nil)
//   - creatorSvc: task creation service for key generation and file creation (optional)
//
// Returns:
//   - *TaskService: configured task service instance
//
// Panics:
//   - If repo is nil (required dependency)
//   - If entitySvc is nil (required dependency)
func NewTaskService(repo TaskRepository, entitySvc *EntityService, creatorSvc *taskcreation.Creator) *TaskService {
	requireNonNil(repo, "TaskService requires a non-nil TaskRepository")
	requireNonNil(entitySvc, "TaskService requires a non-nil EntityService")
	return &TaskService{
		repo:       repo,
		entitySvc:  entitySvc.ForLevel(workflow.LevelTask),
		creatorSvc: creatorSvc,
		docRepo:    nil,
	}
}

// SetDocRepo sets the read-only document repository on the service.
// This enables GetRelatedDocuments and document listing operations.
func (s *TaskService) SetDocRepo(docRepo config.DocumentRepository) {
	s.docRepo = docRepo
}

// SetRelRepo sets the task relationship repository for template placeholder population.
func (s *TaskService) SetRelRepo(relRepo config.TaskRelationshipRepository) {
	s.relRepo = relRepo
}

// SetEnrichRepo sets the template enrichment repository on the service.
// This enables enrichment data population for template rendering.
func (s *TaskService) SetEnrichRepo(enrichRepo config.TemplateEnrichmentRepository) {
	s.enrichRepo = enrichRepo
}

// SetQueryService sets the query sub-service for delegating list/query/display operations.
// When set, calls to ListTasks, GetTasksByStatus, GetTasksByAgent, GetBlockedTasks,
// SearchByFile, and GetTaskDisplayData are delegated to the sub-service.
func (s *TaskService) SetQueryService(svc *TaskQueryService) {
	s.queryService = svc
}

// SetHistoryService sets the history sub-service for delegating history/analytics operations.
// When set, calls to GetTaskHistory, ListHistory, GetWorkSessions, and GetSessionAnalytics
// are delegated to the sub-service.
func (s *TaskService) SetHistoryService(svc *TaskHistoryService) {
	s.historyService = svc
}

// SetTracer sets the OpenTelemetry tracer for the service.
// When nil, getTracer falls back to the OTel global tracer (noop until provider is wired).
func (s *TaskService) SetTracer(t trace.Tracer) {
	s.tracer = t
}

// getTracer returns the configured tracer or falls back to the OTel global tracer.
func (s *TaskService) getTracer() trace.Tracer {
	if s.tracer != nil {
		return s.tracer
	}
	return otel.Tracer("shark/services/task")
}

// getOrInitQueryService returns the query sub-service, lazily initializing it from
// the task repo if it has not been set via SetQueryService.
func (s *TaskService) getOrInitQueryService() *TaskQueryService {
	if s.queryService == nil {
		s.queryService = NewTaskQueryService(s.repo)
	}
	return s.queryService
}

// CreateTask creates a new task with automatic key generation and file creation.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - input: task creation parameters including epic, feature, title, agent, priority, etc.
//
// Returns:
//   - *models.Task: the created task with generated key and ID
//   - error: validation errors, repository errors, or file creation errors
//
// Errors:
//   - ValidationError: if input validation fails (missing epic/feature, invalid priority)
//   - ConflictError: if task key already exists or file path is claimed
//   - RepositoryError: if database operation fails
func (s *TaskService) CreateTask(ctx context.Context, input CreateTaskInput) (*models.Task, error) {
	ctx, span := s.getTracer().Start(ctx, "TaskService.CreateTask",
		trace.WithAttributes(
			attribute.String("task.epic_key", input.EpicKey),
			attribute.String("task.feature_key", input.FeatureKey),
			attribute.String("task.title", input.Title),
		),
	)
	defer span.End()

	// Validate required fields
	if input.EpicKey == "" {
		return nil, recordSpanError(span, fmt.Errorf("failed to create task: epic key is required"))
	}
	if input.FeatureKey == "" {
		return nil, recordSpanError(span, fmt.Errorf("failed to create task: feature key is required"))
	}
	if input.Title == "" {
		return nil, recordSpanError(span, fmt.Errorf("failed to create task: title is required"))
	}

	// Validate optional fields
	if input.Priority > 10 {
		return nil, recordSpanError(span, fmt.Errorf("failed to create task: priority must be between 1 and 10"))
	}

	// Set default priority if not provided
	priority := input.Priority
	if priority == 0 {
		priority = 5 // Default priority
	}

	// Delegate to creatorSvc when available - it handles proper key generation,
	// file creation, and all task creation orchestration.
	if s.creatorSvc != nil {
		// Map services.CreateTaskInput to taskcreation.CreateTaskInput
		// DependsOn []string -> string (join with comma for creator)
		dependsOnStr := strings.Join(input.DependsOn, ",")

		creatorInput := taskcreation.CreateTaskInput{
			EpicKey:        input.EpicKey,
			FeatureKey:     input.FeatureKey,
			Title:          input.Title,
			Description:    input.Description,
			AgentType:      input.AgentType,
			Priority:       priority,
			DependsOn:      dependsOnStr,
			ExecutionOrder: input.ExecutionOrder,
			Filename:       input.FilePath,
			Force:          input.Force,
			Create:         input.CreateFile,
		}

		result, err := s.creatorSvc.CreateTask(ctx, creatorInput)
		if err != nil {
			return nil, recordSpanError(span, fmt.Errorf("failed to create task: %w", err))
		}
		s.maybeReopenParentFeature(ctx, input.FeatureKey, result.Task.Key)
		return result.Task, nil
	}

	// Fallback path (no creatorSvc): generate key via repository prefix search.
	// This path should rarely be hit in production since GetTaskService() always
	// wires a creatorSvc, but it keeps the service functional when creatorSvc is nil
	// (e.g., in unit tests that don't need file creation).

	// Build key prefix and find next available sequence number
	epicKey := strings.ToUpper(input.EpicKey)
	featureKey := strings.ToUpper(input.FeatureKey)
	// Normalise feature key: if it contains the epic prefix already, strip it
	if strings.Contains(featureKey, "-") {
		parts := strings.SplitN(featureKey, "-", 2)
		featureKey = parts[len(parts)-1]
	}
	keyPrefix := fmt.Sprintf("T-%s-%s-", epicKey, featureKey)

	// Find existing tasks with this prefix to determine next sequence number
	existing, err := s.repo.ListByKeyPrefix(ctx, keyPrefix)
	if err != nil {
		return nil, recordSpanError(span, fmt.Errorf("failed to create task: could not query existing keys: %w", err))
	}
	nextSeq := len(existing) + 1
	taskKey := fmt.Sprintf("%s%03d", keyPrefix, nextSeq)

	// Ensure generated key is not already taken (handles gaps from deletions)
	for {
		taken := false
		for _, t := range existing {
			if strings.EqualFold(t.Key, taskKey) {
				taken = true
				break
			}
		}
		if !taken {
			break
		}
		nextSeq++
		taskKey = fmt.Sprintf("%s%03d", keyPrefix, nextSeq)
	}

	// Create task model
	agentType := input.AgentType
	task := &models.Task{BaseEntity: models.BaseEntity{Key: taskKey,
		Title: input.Title}, Status: models.TaskStatus(s.entitySvc.GetWorkflowService().GetDefaultStatus()),
		Priority:       priority,
		AgentType:      &agentType,
		ExecutionOrder: nil,
	}

	// Validate model
	if err := task.Validate(); err != nil {
		return nil, recordSpanError(span, fmt.Errorf("failed to create task: validation error: %w", err))
	}

	// Save to repository
	if err := s.repo.Create(ctx, task); err != nil {
		return nil, recordSpanError(span, fmt.Errorf("failed to create task: %w", err))
	}

	s.maybeReopenParentFeature(ctx, input.FeatureKey, task.Key)
	return task, nil
}

// GetTask retrieves a task by key.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: task key in any format (T-E07-F01-001, E07-F01-001, slugged variants)
//
// Returns:
//   - *models.Task: the task if found
//   - error: NotFoundError if task doesn't exist, or repository errors
//
// Errors:
//   - NotFoundError: task with given key not found
//   - RepositoryError: database query failed
func (s *TaskService) GetTask(ctx context.Context, key string) (*models.Task, error) {
	ctx, span := s.getTracer().Start(ctx, "TaskService.GetTask",
		trace.WithAttributes(attribute.String("task.key", key)),
	)
	defer span.End()

	task, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, recordSpanError(span, fmt.Errorf("failed to get task %s: %w", key, err))
	}
	return task, nil
}

// UpdateTask updates task fields (title, description, priority, etc).
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: task key to update
//   - updates: fields to update (only non-nil fields are updated)
//
// Returns:
//   - *models.Task: the updated task
//   - error: validation errors, not found errors, or repository errors
//
// Errors:
//   - NotFoundError: task not found
//   - ValidationError: invalid field values
//   - RepositoryError: database update failed
func (s *TaskService) UpdateTask(ctx context.Context, key string, updates TaskUpdates) (*models.Task, error) {
	ctx, span := s.getTracer().Start(ctx, "TaskService.UpdateTask",
		trace.WithAttributes(attribute.String("task.key", key)),
	)
	defer span.End()

	// Get existing task
	task, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, recordSpanError(span, fmt.Errorf("failed to update task %s: %w", key, err))
	}

	// Apply non-nil updates
	if updates.Title != nil {
		task.Title = *updates.Title
	}
	if updates.Description != nil {
		task.Description = updates.Description
	}
	if updates.Priority != nil {
		task.Priority = *updates.Priority
	}
	if updates.AgentType != nil {
		task.AgentType = updates.AgentType
	}
	if updates.ExecutionOrder != nil {
		task.ExecutionOrder = updates.ExecutionOrder
	}
	if updates.FilePath != nil {
		task.FilePath = updates.FilePath
	}

	// Validate updated task
	if err := task.Validate(); err != nil {
		return nil, recordSpanError(span, fmt.Errorf("failed to update task %s: validation error: %w", key, err))
	}

	// Save updated task
	if err := s.repo.Update(ctx, task); err != nil {
		return nil, recordSpanError(span, fmt.Errorf("failed to update task %s: %w", key, err))
	}

	return task, nil
}

// DeleteTask deletes a task by key.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: task key to delete
//
// Returns:
//   - error: NotFoundError if task doesn't exist, or repository errors
//
// Errors:
//   - NotFoundError: task not found
//   - ConstraintError: task has dependent tasks that must be deleted first
//   - RepositoryError: database delete failed
func (s *TaskService) DeleteTask(ctx context.Context, key string) error {
	ctx, span := s.getTracer().Start(ctx, "TaskService.DeleteTask",
		trace.WithAttributes(attribute.String("task.key", key)),
	)
	defer span.End()

	// Get task
	task, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return recordSpanError(span, fmt.Errorf("failed to delete task %s: %w", key, err))
	}

	// Check for dependent tasks
	dependents, err := s.repo.GetTaskDependents(ctx, key)
	if err != nil {
		return recordSpanError(span, fmt.Errorf("failed to delete task %s: error checking dependents: %w", key, err))
	}
	if len(dependents) > 0 {
		return recordSpanError(span, fmt.Errorf("failed to delete task %s: task has dependent tasks that must be deleted first", key))
	}

	// Delete task
	if err := s.repo.Delete(ctx, task.ID); err != nil {
		return recordSpanError(span, fmt.Errorf("failed to delete task %s: %w", key, err))
	}

	return nil
}

// ListTasks retrieves tasks matching the given filters.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - filters: filter criteria (epic, feature, status, agent, show_all, title_search, priority, etc.)
//
// Returns:
//   - []*models.Task: list of matching tasks (empty if none found)
//   - error: repository errors
//
// Errors:
//   - RepositoryError: database query failed
func (s *TaskService) ListTasks(ctx context.Context, filters TaskFilters) ([]*models.Task, error) {
	ctx, span := s.getTracer().Start(ctx, "TaskService.ListTasks",
		trace.WithAttributes(attribute.String("task.filter.epic", filters.EpicKey)),
	)
	defer span.End()

	tasks, err := s.getOrInitQueryService().ListTasks(ctx, filters)
	if err != nil {
		return nil, recordSpanError(span, err)
	}
	span.SetAttributes(attribute.Int("task.result_count", len(tasks)))
	return tasks, nil
}

// sortTasks sorts tasks by execution order ascending, then priority descending
func sortTasks(tasks []*models.Task) {
	for i := 0; i < len(tasks); i++ {
		for j := i + 1; j < len(tasks); j++ {
			// Get execution orders
			order1 := 0
			if tasks[i].ExecutionOrder != nil {
				order1 = *tasks[i].ExecutionOrder
			}
			order2 := 0
			if tasks[j].ExecutionOrder != nil {
				order2 = *tasks[j].ExecutionOrder
			}

			// Compare execution order first
			if order1 > order2 {
				tasks[i], tasks[j] = tasks[j], tasks[i]
			} else if order1 == order2 {
				// If order is same, compare priority (descending)
				if tasks[i].Priority < tasks[j].Priority {
					tasks[i], tasks[j] = tasks[j], tasks[i]
				}
			}
		}
	}
}

// TransitionStatus validates and performs a status transition on a task.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: task key to transition
//   - targetStatus: desired new status
//   - opts: transition options (force, reason, agent, etc.)
//
// Returns:
//   - *TransitionResult: details of the transition including orchestrator actions
//   - error: validation errors, workflow errors, or repository errors
//
// Errors:
//   - NotFoundError: task not found
//   - WorkflowError: invalid transition (unless forced)
//   - ValidationError: missing required reason for backward transitions
//   - RepositoryError: database update failed
func (s *TaskService) TransitionStatus(ctx context.Context, key string, targetStatus string, opts TransitionOptions) (*TransitionResult, error) {
	ctx, span := s.getTracer().Start(ctx, "TaskService.TransitionStatus",
		trace.WithAttributes(
			attribute.String("task.key", key),
			attribute.String("task.target_status", targetStatus),
		),
	)
	defer span.End()

	// Create task-specific adapter that routes UpdateStatus through StatusUpdateRaw
	adapter := s.makeTaskEntityAdapter(opts)

	// Delegate full transition flow to EntityService:
	// validation, normalization, force-reason check, backward detection,
	// status update (via adapter), history recording, rejection notes, action resolution
	result, err := s.entitySvc.TransitionStatus(
		ctx, adapter, models.EntityTypeTask, key, targetStatus, opts,
		DefaultTransitionFeatures(),
		s.makeResolveActionFn(ctx),
	)
	if err != nil {
		return nil, recordSpanError(span, err)
	}

	// Post-hook: auto-unblock dependents (task-specific behavior)
	// StatusUpdateRaw returns unblocked keys; retrieve them from the adapter
	if len(adapter.unblockedKeys) > 0 {
		result.Message = fmt.Sprintf("Transitioned: %s -> %s (auto-unblocked: %s)",
			result.FromStatus, result.ToStatus, strings.Join(adapter.unblockedKeys, ", "))
	}

	// Post-hook: recalculate feature progress
	task, taskErr := s.repo.GetByKey(ctx, key)
	if taskErr == nil {
		s.recalculateFeatureProgress(ctx, task.FeatureID)
	}

	return result, nil
}

// GetNextStatus returns available status transitions for a task.
// Delegates to EntityService.GetNextStatus for shared logic.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: task key to check
//
// Returns:
//   - *NextStatusInfo: current status and available transitions with orchestrator actions
//   - error: not found errors or repository errors
//
// Errors:
//   - NotFoundError: task not found
//   - RepositoryError: database query failed
func (s *TaskService) GetNextStatus(ctx context.Context, key string) (*NextStatusInfo, error) {
	ctx, span := s.getTracer().Start(ctx, "TaskService.GetNextStatus",
		trace.WithAttributes(attribute.String("task.key", key)),
	)
	defer span.End()

	// Use a simple read-only adapter (no StatusUpdateRaw needed for GetNextStatus)
	adapter := &taskEntityRepoAdapter{repo: s.repo}
	info, err := s.entitySvc.GetNextStatus(ctx, adapter, models.EntityTypeTask, key, s.makeResolveActionFn(ctx))
	if err != nil {
		return nil, recordSpanError(span, err)
	}
	return info, nil
}

// ValidateStatus checks if a status is valid in the task workflow.
//
// Parameters:
//   - status: status string to validate
//
// Returns:
//   - error: WorkflowError if status is invalid
func (s *TaskService) ValidateStatus(status string) error {
	return s.entitySvc.GetWorkflowService().ValidateStatus(status)
}

// ValidateDependencies checks if a task's dependencies are met for the given transition.
// Includes circular dependency detection using depth-first search.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: task key to validate
//   - targetStatus: status the task wants to transition to
//
// Returns:
//   - error: DependencyError if dependencies not met, circular dependency detected, or repository errors
//
// Errors:
//   - DependencyError: one or more dependencies are not satisfied or circular dependency found
//   - NotFoundError: task not found
//   - RepositoryError: database query failed
func (s *TaskService) ValidateDependencies(ctx context.Context, key string, targetStatus string) error {
	dependents, err := s.repo.GetTaskDependents(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get dependencies for task %s: %w", key, err)
	}

	if len(dependents) == 0 {
		return nil
	}

	for _, dep := range dependents {
		if dep.Status != models.TaskStatus("completed") {
			return fmt.Errorf("dependency not met: task %s depends on %s which is in status %s (must be completed)", key, dep.Key, dep.Status)
		}
	}

	return nil
}

// makeResolveActionFn returns a ResolveActionFn callback that generates
// task-specific placeholders for orchestrator action resolution.
// Closes over ctx for enrichment data fetching.
func (s *TaskService) makeResolveActionFn(ctx context.Context) ResolveActionFn {
	return func(entity models.Entity, status string) *config.PopulatedAction {
		task, ok := entity.(*models.Task)
		if !ok {
			return nil
		}

		// Fetch enrichment data (optional, graceful degradation)
		var enrichment *config.TemplateEnrichmentData
		if s.enrichRepo != nil {
			data, err := s.enrichRepo.GetTaskEnrichment(ctx, task.ID)
			if err != nil {
				slog.Warn("Failed to fetch enrichment data for task", "task", task.Key, "error", err)
			} else {
				enrichment = data
			}
		}

		// Determine which placeholder function to use based on available repositories
		var placeholders map[string]string
		if s.docRepo != nil && s.relRepo != nil {
			placeholders = config.TaskPlaceholdersWithRelated(ctx, task, s.docRepo, s.relRepo, enrichment)
		} else {
			placeholders = config.TaskPlaceholders(task)
			config.ApplyEnrichmentData(enrichment, placeholders)
		}

		// Delegate workflow lookup + PopulatedAction construction to shared helper
		return s.entitySvc.ResolveActionForStatus(status, placeholders)
	}
}

// ListTasksWithPagination retrieves a paginated list of tasks matching the given filters.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - filters: criteria for task filtering (epic, feature, status, agent, limit, offset)
//
// Returns:
//   - []*models.Task: filtered and paginated tasks
//   - int: total count of tasks (before pagination) for UI pagination controls
//   - error: repository errors
//
// Pagination behavior:
//   - Limit=0: returns all filtered tasks
//   - Offset >= total: returns empty slice
//   - Applies pagination after filtering and sorting
//
// Errors:
//   - RepositoryError: database query failed
func (s *TaskService) ListTasksWithPagination(ctx context.Context, filters TaskFilters) ([]*models.Task, int, error) {
	return s.getOrInitQueryService().ListTasksWithPagination(ctx, filters)
}

// GetTasksByStatus groups tasks by status and returns count per status.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - filters: additional filters to apply (epic, feature, agent)
//
// Returns:
//   - map[string]int: status -> count mapping
//   - error: repository errors
//
// Errors:
//   - RepositoryError: database query failed
func (s *TaskService) GetTasksByStatus(ctx context.Context, filters TaskFilters) (map[string]int, error) {
	return s.getOrInitQueryService().GetTasksByStatus(ctx, filters)
}

// GetTasksByAgent groups tasks by agent type and returns count per agent.
// Excludes completed tasks by default unless ShowAll filter is set.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - filters: filters to apply (ShowAll to include completed tasks)
//
// Returns:
//   - map[string]int: agent_type -> count mapping
//   - error: repository errors
//
// Errors:
//   - RepositoryError: database query failed
func (s *TaskService) GetTasksByAgent(ctx context.Context, filters TaskFilters) (map[string]int, error) {
	return s.getOrInitQueryService().GetTasksByAgent(ctx, filters)
}

// GetBlockedTasks returns all blocked tasks matching the given filters.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - filters: additional filters to apply (epic, feature, agent)
//
// Returns:
//   - []*models.Task: blocked tasks
//   - error: repository errors
//
// Errors:
//   - RepositoryError: database query failed
func (s *TaskService) GetBlockedTasks(ctx context.Context, filters TaskFilters) ([]*models.Task, error) {
	return s.getOrInitQueryService().GetBlockedTasks(ctx, filters)
}

// SearchByFile finds tasks that reference or are related to the given file path,
// optionally filtered by epic key, feature key, or status.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - filePath: file path to search for in task associations
//   - filters: optional filters (EpicKey, FeatureKey, Status)
//
// Returns:
//   - []*models.Task: tasks that reference the given file matching the filters
//   - error: repository errors
func (s *TaskService) SearchByFile(ctx context.Context, filePath string, filters TaskFilters) ([]*models.Task, error) {
	return s.getOrInitQueryService().SearchByFile(ctx, filePath, filters)
}

// SetWritableDocRepo sets the writable document repository for link/unlink operations.
// This is optional; commands needing document write operations must call this before use.
func (s *TaskService) SetWritableDocRepo(writableRepo EntityDocumentRepository, linkRepo EntityDocumentLinkRepository) {
	s.docSvc = NewEntityDocumentService(
		writableRepo,
		linkRepo,
		EntityLookupFnFromRepo(&taskSvcKeyLookup{repo: s.repo}),
	)
}

// taskSvcKeyLookup adapts TaskRepository to EntityKeyLookup.
type taskSvcKeyLookup struct {
	repo TaskRepository
}

func (a *taskSvcKeyLookup) GetByKey(ctx context.Context, key string) (models.Entity, error) {
	return a.repo.GetByKey(ctx, key)
}

// GetTaskRepository returns the task repository for use by tree traversal functions.
// This is a low-level accessor needed by the dependency tree visualization in CLI commands.
func (s *TaskService) GetTaskRepository() TaskRepository {
	return s.repo
}

// GetTaskByID retrieves a task by its database ID.
// This is used by history display to resolve task keys from history records,
// which store task IDs rather than keys.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - id: database ID of the task
//
// Returns:
//   - *models.Task: the task if found
//   - error: NotFoundError if task doesn't exist, or repository errors
func (s *TaskService) GetTaskByID(ctx context.Context, id int64) (*models.Task, error) {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get task by ID %d: %w", id, err)
	}
	return task, nil
}

// LinkDocument links a document to a task, creating the document record if it doesn't exist.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - taskKey: the task to link the document to
//   - title: document title
//   - path: document file path
//
// Returns:
//   - *models.Document: the created or retrieved document
//   - error: if task not found, writableDocRepo not configured, or database operation fails
func (s *TaskService) LinkDocument(ctx context.Context, taskKey, title, path string) (*models.Document, error) {
	if s.docSvc == nil {
		return nil, fmt.Errorf("writable document repository not configured")
	}
	return s.docSvc.LinkDocumentByKey(ctx, taskKey, title, path)
}

// UnlinkDocument removes the link between a document and a task.
// This operation is idempotent: it succeeds even if the document is not linked.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - taskKey: the task to unlink the document from
//   - title: document title to look up and unlink
//
// Returns:
//   - error: if task not found, writableDocRepo not configured, or database operation fails
func (s *TaskService) UnlinkDocument(ctx context.Context, taskKey, title string) error {
	if s.docSvc == nil {
		return fmt.Errorf("writable document repository not configured")
	}
	return s.docSvc.UnlinkDocumentByKey(ctx, taskKey, title)
}

// ListRelatedDocuments retrieves all documents linked to a task.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - taskKey: the task key to list documents for
//
// Returns:
//   - []*models.Document: list of linked documents
//   - error: if task not found, docRepo not configured, or database operation fails
func (s *TaskService) ListRelatedDocuments(ctx context.Context, taskKey string) ([]*models.Document, error) {
	if s.docRepo == nil {
		return nil, fmt.Errorf("document repository not configured")
	}

	task, err := s.repo.GetByKey(ctx, taskKey)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	docs, err := s.docRepo.ListForTask(ctx, task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents for task %s: %w", taskKey, err)
	}

	return docs, nil
}

// RelationshipWithTask combines relationship and task info for output.
// This type is returned by GetTaskRelationships, GetTaskBlockedBy, GetTaskBlocks.
type RelationshipWithTask struct {
	RelationshipType string `json:"relationship_type"`
	Direction        string `json:"direction"` // "outgoing" or "incoming"
	TaskKey          string `json:"task_key"`
	TaskTitle        string `json:"task_title"`
	TaskStatus       string `json:"task_status"`
}

// TaskDisplayData holds all supplementary data needed to render a task detail view.
// This is returned by GetTaskDisplayData which fetches everything in a single SQL query.
type TaskDisplayData struct {
	BlockedBy    []RelationshipWithTask
	Blocks       []RelationshipWithTask
	Dependencies []*models.Task
	RelatedDocs  []*models.Document
	Notes        []*models.EntityNote
}

// taskDependencyJSON is the JSON helper for dependency rows from the task_display_data view.
type taskDependencyJSON struct {
	Key    string `json:"key"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

// GetTaskDisplayData fetches all data needed to display a task in a single SQL query
// via the task_display_data view. This reduces round-trips from ~5 to 1, critical for
// Turso cloud databases where each round-trip costs ~150-200ms.
func (s *TaskService) GetTaskDisplayData(ctx context.Context, task *models.Task) (*TaskDisplayData, error) {
	return s.getOrInitQueryService().GetTaskDisplayData(ctx, task)
}

// GetWorkSessions retrieves work sessions and statistics for a task.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - taskKey: the task key to get sessions for
//
// Returns:
//   - *TaskWorkSessions: work sessions and aggregated statistics
//   - error: if task not found, session repo not configured, or database operation fails
func (s *TaskService) GetWorkSessions(ctx context.Context, taskKey string) (*TaskWorkSessions, error) {
	if s.historyService == nil {
		return nil, fmt.Errorf("work session repository not configured")
	}
	// historyService.GetWorkSessions requires taskID and taskTitle; look up task first.
	task, err := s.repo.GetByKey(ctx, taskKey)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}
	return s.historyService.GetWorkSessions(ctx, taskKey, task.ID, task.Title)
}

// SetHistoryRepo sets the task history repository for this service.
// This is used by CLI global accessors to wire optional history repository
// after initial construction via NewTaskService.
func (s *TaskService) SetHistoryRepo(repo TaskHistoryRepository) {
	s.historyRepo = repo
	if s.historyService == nil {
		s.historyService = &TaskHistoryService{historyRepo: repo}
	} else {
		s.historyService.historyRepo = repo
	}
}

// SetEntityHistoryRepo sets the entity history recorder for recording status transitions
// to the polymorphic entity_history table. Optional — degrades gracefully when nil.
func (s *TaskService) SetEntityHistoryRepo(repo EntityHistoryRecorder) {
	s.entityHistoryRepo = repo
}

// SetSessionRepo sets the work session repository for analytics.
// This is used by CLI global accessors to wire optional session repository
// after initial construction via NewTaskService.
func (s *TaskService) SetSessionRepo(repo WorkSessionRepository) {
	s.sessionRepo = repo
	if s.historyService == nil {
		s.historyService = &TaskHistoryService{sessionRepo: repo}
	} else {
		s.historyService.sessionRepo = repo
	}
}

// SetEpicRepo sets the analytics epic repository for scope resolution.
// This is used by CLI global accessors to wire optional epic repository
// after initial construction via NewTaskService.
func (s *TaskService) SetEpicRepo(repo AnalyticsEpicRepository) {
	s.epicRepo = repo
}

// SetFeatureRepo sets the analytics feature repository for scope resolution.
// This is used by CLI global accessors to wire optional feature repository
// after initial construction via NewTaskService.
func (s *TaskService) SetFeatureRepo(repo AnalyticsFeatureRepository) {
	s.featureRepo = repo
}

// SetFeatureService sets the feature service for write-through progress recalculation.
// When set, every status-mutating operation on a task triggers a progress recalculation
// on the parent feature. This is non-fatal: errors are silently ignored.
// This is used by CLI global accessors to wire the optional feature service
// after initial construction via NewTaskService.
func (s *TaskService) SetFeatureService(featureService *FeatureService) {
	s.featureService = featureService
}

// recalculateFeatureProgress triggers a feature progress recalculation after a task
// status change. Non-fatal: logs a warning on error and ignores nil featureService.
// Only recalculates when featureID is non-zero (i.e., task is associated with a feature).
func (s *TaskService) recalculateFeatureProgress(ctx context.Context, featureID int64) {
	if s.featureService == nil || featureID == 0 {
		return
	}
	if err := s.featureService.RecalculateAndSetProgress(ctx, featureID); err != nil {
		slog.Warn("feature progress recalculation failed after task status change",
			"feature_id", featureID, "error", err)
	}
}

// maybeReopenParentFeature checks if the parent feature is in a terminal status
// and reopens it to the first aggregation status. Best-effort: logs a warning
// on failure, never fails the caller.
//
// Parameters:
//   - ctx: context for cancellation
//   - featureKey: key of the parent feature (e.g., "E01-F01")
//   - taskKey: key of the newly created task (for audit logging)
func (s *TaskService) maybeReopenParentFeature(ctx context.Context, featureKey string, taskKey string) {
	if s.featureService == nil {
		return
	}

	feature, err := s.featureService.GetFeature(ctx, featureKey)
	if err != nil {
		slog.Warn("auto-reopen check for feature failed", "feature", featureKey, "error", err)
		return
	}
	if feature == nil {
		return
	}

	featureWf := s.entitySvc.GetWorkflowService().ForLevel(workflow.LevelFeature)
	if !featureWf.IsTerminalStatus(string(feature.Status)) {
		return
	}

	aggStatuses := featureWf.GetAggregationStatuses()
	targetStatus := models.FeatureStatus(aggStatuses[0])

	oldStatus := string(feature.Status)
	_, err = s.featureService.UpdateFeature(ctx, feature.Key, FeatureUpdates{
		Status: &targetStatus,
	})
	if err != nil {
		slog.Warn("auto-reopen of feature failed", "feature", featureKey, "error", err)
		return
	}

	// Record history for the auto-reopen
	notes := fmt.Sprintf("auto-reopened: new task %s created under terminal feature", taskKey)
	recordEntityHistory(ctx, s.entityHistoryRepo, models.EntityTypeFeature, feature.ID,
		oldStatus, string(targetStatus), false, EntityHistoryOpts{
			Agent:  "system",
			Reason: notes,
		})
}

// statusTransitionOpts bundles metadata for routing through StatusUpdateRaw
// via the taskEntityRepoAdapter. Used by makeTaskEntityAdapter to pass
// agent/reason/documentPath/force to the adapter.
type statusTransitionOpts struct {
	// agent is the optional agent performing the transition.
	agent *string
	// notes is optional notes for the transition.
	notes *string
	// reason is optional rejection/block reason.
	reason *string
	// documentPath is optional path to a related document.
	documentPath *string
	// force is passed through to StatusUpdateRaw for repository-level handling.
	force bool
}

// taskEntityRepoAdapter adapts the typed TaskRepository to the EntityRepository
// interface, routing UpdateStatus through StatusUpdateRaw to preserve
// agent/reason/documentPath handling and capture auto-unblocked keys.
type taskEntityRepoAdapter struct {
	repo TaskRepository
	// opts is set before each TransitionStatus call to pass transition-specific
	// metadata (agent, reason, documentPath, force) to StatusUpdateRaw.
	opts *statusTransitionOpts
	// lastTask caches the task fetched by GetByKey so UpdateStatus can access
	// timestamp fields without an extra DB round-trip.
	lastTask *models.Task
	// unblockedKeys captures the auto-unblocked task keys from StatusUpdateRaw,
	// retrieved by TaskService after EntityService.TransitionStatus returns.
	unblockedKeys []string
}

func (a *taskEntityRepoAdapter) GetByKey(ctx context.Context, key string) (models.Entity, error) {
	task, err := a.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, nil
	}
	a.lastTask = task
	return task, nil
}

func (a *taskEntityRepoAdapter) GetByID(ctx context.Context, id int64) (models.Entity, error) {
	task, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, nil
	}
	return task, nil
}

func (a *taskEntityRepoAdapter) UpdateStatus(ctx context.Context, id int64, status string) error {
	if a.opts == nil {
		// Fallback: simple status update without metadata
		return a.repo.UpdateStatus(ctx, id, models.TaskStatus(status), nil, nil)
	}

	// Use cached task from GetByKey (EntityService calls GetByKey before UpdateStatus)
	task := a.lastTask
	if task == nil || task.ID != id {
		// Fallback: fetch task if cache miss
		var err error
		task, err = a.repo.GetByID(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to get task for status update: %w", err)
		}
	}

	params := models.StatusUpdateParams{
		TaskID:          id,
		NewStatus:       models.TaskStatus(status),
		Agent:           a.opts.agent,
		Notes:           a.opts.notes,
		RejectionReason: a.opts.reason,
		DocumentPath:    a.opts.documentPath,
		Force:           a.opts.force,
		OldStatus:       string(task.Status),
		TaskKey:         task.Key,
		StartedAt:       task.StartedAt,
		CompletedAt:     task.CompletedAt,
		BlockedAt:       task.BlockedAt,
	}

	unblockedKeys, err := a.repo.StatusUpdateRaw(ctx, params)
	if err != nil {
		return err
	}
	a.unblockedKeys = unblockedKeys
	return nil
}

func (a *taskEntityRepoAdapter) Update(ctx context.Context, entity models.Entity) error {
	task, ok := entity.(*models.Task)
	if !ok {
		return fmt.Errorf("taskEntityRepoAdapter: expected *models.Task, got %T", entity)
	}
	return a.repo.Update(ctx, task)
}

func (a *taskEntityRepoAdapter) GetContextData(ctx context.Context, id int64) (*string, error) {
	task, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return task.ContextData, nil
}

func (a *taskEntityRepoAdapter) UpdateContextData(ctx context.Context, id int64, data *string) error {
	task, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	task.ContextData = data
	return a.repo.Update(ctx, task)
}

// makeTaskEntityAdapter creates a taskEntityRepoAdapter configured with the
// given TransitionOptions for routing through StatusUpdateRaw.
func (s *TaskService) makeTaskEntityAdapter(opts TransitionOptions) *taskEntityRepoAdapter {
	var agentPtr, reasonPtr, docPathPtr *string
	if opts.Agent != "" {
		agentPtr = &opts.Agent
	}
	if opts.Reason != "" {
		reasonPtr = &opts.Reason
	}
	if opts.DocumentPath != "" {
		docPathPtr = &opts.DocumentPath
	}

	return &taskEntityRepoAdapter{
		repo: s.repo,
		opts: &statusTransitionOpts{
			agent:        agentPtr,
			reason:       reasonPtr,
			documentPath: docPathPtr,
			force:        opts.Force,
		},
	}
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
func (s *TaskService) GetTaskHistory(ctx context.Context, taskKey string) ([]*models.TaskHistory, error) {
	if s.historyService == nil {
		return nil, fmt.Errorf("history repository not configured")
	}
	return s.historyService.GetTaskHistory(ctx, taskKey)
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
func (s *TaskService) ListHistory(ctx context.Context, filters HistoryFilters) ([]*models.TaskHistory, error) {
	if s.historyService == nil {
		return nil, fmt.Errorf("history repository not configured")
	}
	return s.historyService.ListHistory(ctx, filters)
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
func (s *TaskService) GetSessionAnalytics(ctx context.Context, input SessionAnalyticsInput) (*SessionAnalytics, error) {
	if s.historyService == nil {
		return nil, fmt.Errorf("work session repository not configured")
	}
	return s.historyService.GetSessionAnalytics(ctx, input)
}
