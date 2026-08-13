package services

import (
	"context"
	"database/sql"
	"errors"
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
	// UpdateNoResequence updates a task without cascading execution_order
	// changes to siblings. Used to preserve intentional duplicate-order
	// groups (parallel work). Wired from `--parallel` on `shark task update`.
	UpdateNoResequence(ctx context.Context, task *models.Task) error
	Delete(ctx context.Context, id int64) error

	// Query operations
	List(ctx context.Context) ([]*models.Task, error)
	ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error)
	ListByFeatureKey(ctx context.Context, featureKey string) ([]*models.Task, error)
	ListByEpic(ctx context.Context, epicKey string) ([]*models.Task, error)

	// Dependency operations
	GetTaskDependencies(ctx context.Context, taskKey string) ([]*models.Task, error)
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
	UpdateStatusIfCurrent(ctx context.Context, taskID int64, expectedStatus models.TaskStatus, newStatus models.TaskStatus) (bool, error)

	// StatusUpdateRawWithTx performs the same operation as StatusUpdateRaw but within a
	// caller-provided transaction. Services use this to own the transaction boundary when
	// multiple repository operations must be atomic (Standard 8: services own transactions).
	// Returns the list of auto-unblocked task keys.
	StatusUpdateRawWithTx(ctx context.Context, tx *sql.Tx, params models.StatusUpdateParams) ([]string, error)

	// BeginTx starts a new database transaction for use by the service layer.
	// Services own transaction boundaries per Standard 8; repositories participate
	// in service-owned transactions by accepting *sql.Tx parameters.
	BeginTx(ctx context.Context) (*sql.Tx, error)

	// Search operations
	FindByFileChanged(ctx context.Context, filePath string) ([]*models.Task, error)

	// Key prefix search - returns all tasks whose key starts with the given prefix.
	// Used for key generation to avoid UNIQUE constraint collisions.
	ListByKeyPrefix(ctx context.Context, prefix string) ([]*models.Task, error)

	// Display data - single-query aggregation via task_display_data view
	GetTaskDisplayDataRaw(ctx context.Context, taskID int64) (*repository.TaskDisplayDataRaw, error)

	// GetRejectionCounts returns rejection counts and last rejection timestamps
	// for a batch of tasks. Counts come from entity_notes rows with
	// note_type='rejection' (written when a backward/forced status transition
	// stores a rejection reason). Used by GetTask/ListTasks to populate the
	// derived RejectionCount/LastRejectionAt fields on models.Task.
	GetRejectionCounts(ctx context.Context, taskIDs []int64) (map[int64]int, map[int64]*time.Time, error)
}

// taskDependencyClearer is an optional repository capability used only for an
// explicit ClearDependsOn request. Keeping it narrow avoids changing ordinary
// update semantics for relationship-backed dependencies.
type taskDependencyClearer interface {
	UpdateClearingDependencies(ctx context.Context, task *models.Task, skipResequence bool) error
}

// taskDeleteWithTx is the optional transaction-aware delete seam used when
// aggregate maintenance is wired. Keeping it separate preserves lightweight
// service mocks that do not need database transaction behavior.
type taskDeleteWithTx interface {
	DeleteWithTx(ctx context.Context, tx *sql.Tx, id int64) error
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
	repo                 TaskRepository
	entitySvc            *EntityService
	creatorSvc           *taskcreation.Creator
	historyRepo          TaskHistoryRepository
	entityHistoryRepo    EntityHistoryRecorder // optional: records to entity_history table
	docRepo              config.DocumentRepository
	relRepo              config.TaskRelationshipRepository // for template placeholder population (ListRelatedTaskKeys)
	sessionRepo          WorkSessionRepository
	epicRepo             AnalyticsEpicRepository
	featureRepo          AnalyticsFeatureRepository
	featureService       *FeatureService               // optional: triggers progress recalc on status change
	aggregateCoordinator *AggregateMutationCoordinator // optional: owns transactional parent aggregate maintenance
	enrichRepo           config.TemplateEnrichmentRepository
	docSvc               *EntityDocumentService // shared document operations; built by SetWritableDocRepo
	searchIndexer        SearchIndexer
	// tagSvc is optional — nil disables tag integration.
	// TagQuerier extends TagAttacher with EntityIDsByTags for list filtering (F05).
	tagSvc TagQuerier

	// sizeCfg is optional — nil disables size enforcement on create.
	// When set, CreateTask requires input.Size when EntityTypeTask is listed
	// in cfg.SizeRequiredFor(). Mirrors tag enforcement.
	sizeCfg SizeEnforcementConfig

	// Cascade reopen dependencies (all optional; cascade fires only when all are non-nil).
	cascadeDB          txBeginner
	cascadeFeatureRepo CascadeFeatureRepo
	cascadeEpicRepo    CascadeEpicRepo
	cascadeHistQuerier ParentReopenHistoryQuerier
	cascadeHistTx      EntityHistoryTxRecorder

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
//   - bool: true when an existing file was linked (vs. a new placeholder created)
//   - error: validation errors, repository errors, or file creation errors
//
// Errors:
//   - ValidationError: if input validation fails (missing epic/feature, invalid priority)
//   - ConflictError: if task key already exists or file path is claimed
//   - RepositoryError: if database operation fails
func (s *TaskService) CreateTask(ctx context.Context, input CreateTaskInput) (*models.Task, bool, error) {
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
		return nil, false, recordSpanError(span, fmt.Errorf("failed to create task: epic key is required"))
	}
	if input.FeatureKey == "" {
		return nil, false, recordSpanError(span, fmt.Errorf("failed to create task: feature key is required"))
	}
	if input.Title == "" {
		return nil, false, recordSpanError(span, fmt.Errorf("failed to create task: title is required"))
	}

	// Validate optional fields
	if input.Priority > 10 {
		return nil, false, recordSpanError(span, fmt.Errorf("failed to create task: priority must be between 1 and 10"))
	}

	if err := enforceTagsRequired(ctx, s.tagSvc, models.EntityTypeTask, input.Tags); err != nil {
		return nil, false, recordSpanError(span, err)
	}
	if err := enforceSizeRequired(s.sizeCfg, models.EntityTypeTask, input.Size); err != nil {
		return nil, false, recordSpanError(span, err)
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
			Size:           input.Size,
			Body:           input.Body,
		}

		result, err := s.creatorSvc.CreateTask(ctx, creatorInput)
		if err != nil {
			return nil, false, recordSpanError(span, fmt.Errorf("failed to create task: %w", err))
		}

		if err := attachTagsIfAny(ctx, s.tagSvc, models.EntityTypeTask, result.Task.ID, input.Tags); err != nil {
			return nil, false, recordSpanError(span, err)
		}
		if err := indexEntityIfConfigured(ctx, s.searchIndexer, models.EntityTypeTask, result.Task.ID); err != nil {
			return nil, false, recordSpanError(span, err)
		}

		if s.aggregateCoordinator == nil {
			s.maybeReopenParentFeature(ctx, input.FeatureKey, result.Task.Key)
		}
		return result.Task, result.FileWasLinked, nil
	}
	if s.aggregateCoordinator != nil {
		return nil, false, recordSpanError(span, fmt.Errorf("failed to create task: transactional creator is required when aggregate maintenance is enabled"))
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
		return nil, false, recordSpanError(span, fmt.Errorf("failed to create task: could not query existing keys: %w", err))
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
		Title: input.Title,
		Size:  input.Size}, Status: models.TaskStatus(s.entitySvc.GetWorkflowService().GetDefaultStatus()),
		Priority:       priority,
		AgentType:      &agentType,
		ExecutionOrder: nil,
	}

	// Validate model
	if err := task.Validate(); err != nil {
		return nil, false, recordSpanError(span, fmt.Errorf("failed to create task: validation error: %w", err))
	}

	// Save to repository
	if err := s.repo.Create(ctx, task); err != nil {
		return nil, false, recordSpanError(span, fmt.Errorf("failed to create task: %w", err))
	}

	if err := attachTagsIfAny(ctx, s.tagSvc, models.EntityTypeTask, task.ID, input.Tags); err != nil {
		return nil, false, recordSpanError(span, err)
	}
	if err := indexEntityIfConfigured(ctx, s.searchIndexer, models.EntityTypeTask, task.ID); err != nil {
		return nil, false, recordSpanError(span, err)
	}

	if s.aggregateCoordinator == nil {
		s.maybeReopenParentFeature(ctx, input.FeatureKey, task.Key)
	}
	return task, false, nil
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
	if err := s.enrichRejectionCounts(ctx, []*models.Task{task}); err != nil {
		return nil, recordSpanError(span, err)
	}
	return task, nil
}

// enrichRejectionCounts populates the derived RejectionCount and LastRejectionAt
// fields on each task by issuing a single batched query against entity_notes.
// No-op for an empty slice. Mirrors the pattern used by GetTaskWithTags for tag
// enrichment.
func (s *TaskService) enrichRejectionCounts(ctx context.Context, tasks []*models.Task) error {
	if len(tasks) == 0 {
		return nil
	}
	ids := make([]int64, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ID
	}
	counts, lastTimes, err := s.repo.GetRejectionCounts(ctx, ids)
	if err != nil {
		return fmt.Errorf("failed to load rejection counts: %w", err)
	}
	for _, t := range tasks {
		t.RejectionCount = counts[t.ID]
		if last, ok := lastTimes[t.ID]; ok {
			t.LastRejectionAt = last
		}
	}
	return nil
}

// GetTaskWithTags returns the task and the sorted list of tag names attached to
// it. When tagSvc is nil the tags slice is nil (graceful degradation —
// consistent with F04 REQ-F-018). When ListTagsForEntity fails the method
// returns (nil, nil, wrappedErr) per AC-T3.
func (s *TaskService) GetTaskWithTags(ctx context.Context, key string) (*models.Task, []string, error) {
	ctx, span := s.getTracer().Start(ctx, "TaskService.GetTaskWithTags",
		trace.WithAttributes(attribute.String("task.key", key)),
	)
	defer span.End()

	task, err := s.GetTask(ctx, key)
	if err != nil {
		return nil, nil, err
	}
	if s.tagSvc == nil {
		return task, nil, nil
	}
	names, err := s.tagSvc.ListTagsForEntity(ctx, models.EntityTypeTask, task.ID)
	if err != nil {
		return nil, nil, recordSpanError(span, fmt.Errorf("load tags for task %s: %w", key, err))
	}
	return task, names, nil
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

	// Three-branch Size update logic (E07-F42 AC-T1).
	if updates.ClearSize {
		task.Size = nil
	} else if updates.Size != nil {
		task.Size = updates.Size
	}
	// else: leave task.Size unchanged (no-op)

	// Three-branch DependsOn update logic (B048): ClearDependsOn takes
	// precedence and nulls the JSON-encoded dependency column; DependsOn
	// sets it to the new JSON-encoded list; otherwise leave unchanged.
	if updates.ClearDependsOn {
		task.DependsOn = nil
	} else if updates.DependsOn != nil {
		task.DependsOn = updates.DependsOn
	}
	// else: leave task.DependsOn unchanged (no-op)

	// Validate updated task
	if err := task.Validate(); err != nil {
		return nil, recordSpanError(span, fmt.Errorf("failed to update task %s: validation error: %w", key, err))
	}

	// Save updated task. When --parallel was passed (SkipResequence=true), use
	// the no-resequence path so siblings keep their existing execution_order
	// values instead of being renumbered.
	var saveErr error
	if updates.ClearDependsOn {
		clearer, ok := s.repo.(taskDependencyClearer)
		if !ok {
			return nil, recordSpanError(span, fmt.Errorf("task repository does not support clearing dependency relationships"))
		}
		saveErr = clearer.UpdateClearingDependencies(ctx, task, updates.SkipResequence)
	} else if updates.SkipResequence {
		saveErr = s.repo.UpdateNoResequence(ctx, task)
	} else {
		saveErr = s.repo.Update(ctx, task)
	}
	if saveErr != nil {
		return nil, recordSpanError(span, fmt.Errorf("failed to update task %s: %w", key, saveErr))
	}

	// `--tag` on update is additive only; detach goes through `shark task tag rm`.
	if err := attachTagsIfAny(ctx, s.tagSvc, models.EntityTypeTask, task.ID, updates.Tags); err != nil {
		return nil, recordSpanError(span, err)
	}
	if err := indexEntityIfConfigured(ctx, s.searchIndexer, models.EntityTypeTask, task.ID); err != nil {
		return nil, recordSpanError(span, err)
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

	// Delete task and refresh the parent cache before committing when aggregate
	// coordination is available.
	if s.aggregateCoordinator != nil {
		txRepo, ok := s.repo.(taskDeleteWithTx)
		if !ok {
			return recordSpanError(span, fmt.Errorf("task repository does not support transactional delete"))
		}
		tx, err := s.repo.BeginTx(ctx)
		if err != nil {
			return recordSpanError(span, fmt.Errorf("failed to begin task aggregate transaction: %w", err))
		}
		defer rollbackAfterAggregateMutation(tx)
		if err := txRepo.DeleteWithTx(ctx, tx, task.ID); err != nil {
			return recordSpanError(span, fmt.Errorf("failed to delete task %s: %w", key, err))
		}
		if err := s.aggregateCoordinator.RefreshFeatureAndEpic(ctx, tx, task.FeatureID, cascadeTrigger{
			triggerKey: task.Key, triggerKind: "deletion", triggerType: models.EntityTypeTask, startLeg: cascadeLegFeature, featureID: task.FeatureID,
		}); err != nil {
			return recordSpanError(span, fmt.Errorf("failed to maintain task aggregates: %w", err))
		}
		if err := tx.Commit(); err != nil {
			return recordSpanError(span, fmt.Errorf("failed to commit task aggregate transaction: %w", err))
		}
	} else if err := s.repo.Delete(ctx, task.ID); err != nil {
		return recordSpanError(span, fmt.Errorf("failed to delete task %s: %w", key, err))
	}
	if err := removeEntityFromIndexIfConfigured(ctx, s.searchIndexer, models.EntityTypeTask, task.ID); err != nil {
		return recordSpanError(span, err)
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

	// Block 1: pre-filter by tag IDs (E28-F05 §2.5.2).
	var taggedIDSet map[int64]struct{}
	if len(filters.Tags) > 0 {
		if s.tagSvc == nil {
			return nil, recordSpanError(span, &TagFilterUnavailableError{})
		}
		ids, err := s.tagSvc.EntityIDsByTags(ctx, models.EntityTypeTask, filters.Tags, TagQueryOpAnd)
		if err != nil {
			return nil, recordSpanError(span, err)
		}
		if len(ids) == 0 {
			return []*models.Task{}, nil // REQ-F-017 short-circuit
		}
		taggedIDSet = make(map[int64]struct{}, len(ids))
		for _, id := range ids {
			taggedIDSet[id] = struct{}{}
		}
	}

	tasks, err := s.getOrInitQueryService().ListTasks(ctx, filters)
	if err != nil {
		return nil, recordSpanError(span, err)
	}

	// Block 2: post-filter in-memory (E28-F05 §2.5.2).
	tasks = filterByTagIDs(tasks, taggedIDSet, func(t *models.Task) int64 { return t.ID })

	// Block 3: enrich tasks with rejection counts before any rejection-aware
	// filtering or downstream display (warning indicator in task_table, JSON
	// rejection_count field, --has-rejections filter).
	if err := s.enrichRejectionCounts(ctx, tasks); err != nil {
		return nil, recordSpanError(span, err)
	}

	// Block 4: apply HasRejections filter after enrichment populates counts.
	if filters.HasRejections {
		filtered := tasks[:0]
		for _, t := range tasks {
			if t.RejectionCount > 0 {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
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

	// Create task-specific adapter that routes UpdateStatus through StatusUpdateRaw.
	// Aggregate coordination keeps that status write and the parent refresh in one
	// caller-owned transaction.
	adapter := s.makeTaskEntityAdapter(opts)
	var aggregateTx *sql.Tx
	if s.aggregateCoordinator != nil {
		var txErr error
		aggregateTx, txErr = s.repo.BeginTx(ctx)
		if txErr != nil {
			return nil, recordSpanError(span, fmt.Errorf("failed to begin task aggregate transaction: %w", txErr))
		}
		defer rollbackAfterAggregateMutation(aggregateTx)
		adapter.tx = aggregateTx
	}

	// Delegate full transition flow to EntityService:
	// validation, normalization, force-reason check, backward detection,
	// status update (via adapter), history recording, rejection notes, action resolution
	transitionFeatures := DefaultTransitionFeatures()
	var result *TransitionResult
	var err error
	if aggregateTx != nil {
		// TaskRepository.StatusUpdateRawWithTx creates the rejection note in the
		// same transaction, so EntityService must not create a duplicate.
		transitionFeatures.CreateRejectionNotes = false
		result, err = s.entitySvc.TransitionStatusWithTx(
			ctx, aggregateTx, adapter, models.EntityTypeTask, key, targetStatus, opts,
			transitionFeatures, s.makeResolveActionFn(ctx),
		)
	} else {
		result, err = s.entitySvc.TransitionStatus(
			ctx, adapter, models.EntityTypeTask, key, targetStatus, opts,
			transitionFeatures, s.makeResolveActionFn(ctx),
		)
	}
	if err != nil {
		return nil, recordSpanError(span, err)
	}

	// Post-hook: auto-unblock dependents (task-specific behavior)
	// StatusUpdateRaw returns unblocked keys; retrieve them from the adapter
	if len(adapter.unblockedKeys) > 0 {
		result.Message = fmt.Sprintf("Transitioned: %s -> %s (auto-unblocked: %s)",
			result.FromStatus, result.ToStatus, strings.Join(adapter.unblockedKeys, ", "))
	}

	// Post-hook: recalculate feature progress. Aggregate coordination is the
	// authoritative path; the legacy fallback is retained for partially wired
	// test and embedding environments.
	task := adapter.lastTask
	var taskErr error
	if s.aggregateCoordinator != nil {
		if task == nil {
			return nil, recordSpanError(span, fmt.Errorf("task %s was not loaded for aggregate maintenance", key))
		}
		triggerKind := "transition"
		taskWf := s.entitySvc.GetWorkflowService().ForLevel(workflow.LevelTask)
		if result.Transitioned && taskWf.IsTerminalStatus(result.FromStatus) && !taskWf.IsTerminalStatus(result.ToStatus) {
			triggerKind = "regression"
		}
		if err := s.aggregateCoordinator.RefreshFeatureAndEpic(ctx, aggregateTx, task.FeatureID, cascadeTrigger{
			triggerKey: task.Key, triggerKind: triggerKind, triggerType: models.EntityTypeTask, startLeg: cascadeLegFeature, featureID: task.FeatureID,
		}); err != nil {
			return nil, recordSpanError(span, fmt.Errorf("failed to maintain task aggregates: %w", err))
		}
		if err := aggregateTx.Commit(); err != nil {
			return nil, recordSpanError(span, fmt.Errorf("failed to commit task aggregate transaction: %w", err))
		}
	} else {
		if task == nil {
			task, taskErr = s.repo.GetByKey(ctx, key)
		}
		if taskErr == nil && task != nil {
			s.recalculateFeatureProgress(ctx, task.FeatureID)
		}
	}

	// Cascade post-hook: reopen terminal parents when a task regresses from
	// a terminal status to a non-terminal status (AC-01 / REQ-F-001).
	if s.aggregateCoordinator == nil && s.cascadeEnabled() && taskErr == nil && task != nil {
		taskWf := s.entitySvc.GetWorkflowService().ForLevel(workflow.LevelTask)
		if taskWf.IsTerminalStatus(result.FromStatus) && !taskWf.IsTerminalStatus(result.ToStatus) {
			cascadeParentReopens(ctx, s.cascadeDepsBundle(), cascadeTrigger{
				triggerKey:  key,
				triggerKind: "regression",
				triggerType: models.EntityTypeTask,
				startLeg:    cascadeLegFeature,
				featureID:   task.FeatureID,
			})
		}
	}
	if err := indexEntityIfConfigured(ctx, s.searchIndexer, models.EntityTypeTask, result.EntityID); err != nil {
		return nil, recordSpanError(span, err)
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
	if info != nil && !info.IsTerminal {
		if err := s.ValidateDependencies(ctx, key, info.CurrentStatus); err != nil {
			return nil, recordSpanError(span, err)
		}
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
	dependencies, err := s.repo.GetTaskDependencies(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get dependencies for task %s: %w", key, err)
	}

	if len(dependencies) == 0 {
		return nil
	}

	// Terminal classification is config-driven: a dependency is satisfied when
	// its status is terminal in the *task* workflow. Delegating to
	// workflow.Service.IsTerminalStatus (rather than a hardcoded
	// completed/archived pair) keeps custom workflows that rename or add
	// terminal statuses working, and matches CascadeService.dependenciesSatisfied
	// so "is this dependency satisfied?" has one answer across the codebase.
	// A nil workflow yields "not terminal", so a dependency blocks rather than
	// silently passing (mirrors plan_hierarchy_service.go).
	taskWf := s.entitySvc.GetWorkflowService().ForLevel(workflow.LevelTask)
	var terminal []string
	if taskWf != nil {
		terminal = taskWf.GetTerminalStatuses()
	}
	for _, dep := range dependencies {
		if taskWf == nil || !taskWf.IsTerminalStatus(string(dep.Status)) {
			return fmt.Errorf("dependency not met: task %s depends on %s which is in status %s (must be one of: %s)",
				key, dep.Key, dep.Status, strings.Join(terminal, ", "))
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

		// Fresh transition context: suppress RESUME CONTEXT preamble in templates.
		// is_resume="true" is reserved for shark get (display_service.go).
		placeholders["is_resume"] = "false"

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
func (s *TaskService) SetWritableDocRepo(writableRepo EntityDocumentRepository, linkRepo EntityDocumentLinkRepository, projectRoot string) {
	s.docSvc = NewEntityDocumentService(
		writableRepo,
		linkRepo,
		EntityLookupFnFromRepo(&taskSvcKeyLookup{repo: s.repo}),
		projectRoot,
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

// RelationshipWithTask combines relationship and related-entity info for
// output. This type is returned by GetTaskRelationships, GetTaskBlockedBy,
// GetTaskBlocks, and (via TaskDisplayData) GetTaskDisplayData.
//
// B049: the related entity is not always a task -- task_display_data's
// blocked_by_json/blocks_json/relationships_json now cover cross-entity
// relationships (e.g. a task blocking a feature). The TaskKey/TaskTitle/
// TaskStatus field names are kept as-is for backward compatibility with
// existing JSON consumers; they hold the *related entity's* key/title/status
// regardless of its type. EntityType disambiguates what that entity actually
// is ("task", "feature", "epic", "bug", "change", "tech_debt", "question").
// Producers that only ever resolve tasks (e.g. resolveTaskRelationships) set
// EntityType to "task".
type RelationshipWithTask struct {
	RelationshipType string `json:"relationship_type"`
	Direction        string `json:"direction"` // "outgoing" or "incoming"
	TaskKey          string `json:"task_key"`
	TaskTitle        string `json:"task_title"`
	TaskStatus       string `json:"task_status"`
	EntityType       string `json:"entity_type,omitempty"`
}

// TaskDisplayData holds all supplementary data needed to render a task detail view.
// This is returned by GetTaskDisplayData which fetches everything in a single SQL query.
type TaskDisplayData struct {
	BlockedBy     []RelationshipWithTask
	Blocks        []RelationshipWithTask
	Relationships []RelationshipWithTask
	Dependencies  []*models.Task
	RelatedDocs   []*models.Document
	Notes         []*models.EntityNote
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

// SetTagService wires the optional TagQuerier dependency. When nil, tag
// hooks in CreateTask, UpdateTask, and ListTasks are skipped silently.
// TagQuerier extends TagAttacher with EntityIDsByTags for list filtering (F05).
func (s *TaskService) SetTagService(tagSvc TagQuerier) {
	s.tagSvc = tagSvc
}

// SetSearchIndexer wires the optional search indexer used after task writes.
func (s *TaskService) SetSearchIndexer(indexer SearchIndexer) {
	s.searchIndexer = indexer
}

// SetSizeEnforcement wires the optional SizeEnforcementConfig. When nil or
// when the config does not list "task" in SizeRequiredFor, CreateTask accepts
// nil Size silently. Mirrors SetTagService.
func (s *TaskService) SetSizeEnforcement(cfg SizeEnforcementConfig) {
	s.sizeCfg = cfg
}

// SetFeatureService sets the feature service for write-through progress recalculation.
// When set, every status-mutating operation on a task triggers a progress recalculation
// on the parent feature. This is non-fatal: logs a warning on error.
// This is used by CLI global accessors to wire the optional feature service
// after initial construction via NewTaskService.
func (s *TaskService) SetFeatureService(featureService *FeatureService) {
	s.featureService = featureService
}

// SetAggregateMutationCoordinator wires the transactional aggregate-maintenance
// seam for progress-affecting task writes. The creator hook runs inside the
// task creation transaction, so parent cache failures roll back the new task.
func (s *TaskService) SetAggregateMutationCoordinator(coordinator *AggregateMutationCoordinator) {
	s.aggregateCoordinator = coordinator
	if s.creatorSvc != nil {
		s.creatorSvc.SetAfterCreateHook(func(ctx context.Context, tx *sql.Tx, task *models.Task) error {
			if coordinator == nil {
				return nil
			}
			return coordinator.RefreshFeatureAndEpic(ctx, tx, task.FeatureID, cascadeTrigger{
				triggerKey: task.Key, triggerKind: "creation", triggerType: models.EntityTypeTask, startLeg: cascadeLegFeature, featureID: task.FeatureID,
			})
		})
	}
}

// SetCascadeDeps wires the optional cascade reopen dependencies.
// All five parameters must be non-nil for the cascade to fire; any nil value
// disables the cascade silently (graceful degradation per AC-T5).
func (s *TaskService) SetCascadeDeps(db txBeginner, fr CascadeFeatureRepo, er CascadeEpicRepo, hq ParentReopenHistoryQuerier, ht EntityHistoryTxRecorder) {
	s.cascadeDB = db
	s.cascadeFeatureRepo = fr
	s.cascadeEpicRepo = er
	s.cascadeHistQuerier = hq
	s.cascadeHistTx = ht
}

// cascadeEnabled returns true iff all five cascade dependencies are non-nil.
func (s *TaskService) cascadeEnabled() bool {
	return s.cascadeDB != nil && s.cascadeFeatureRepo != nil && s.cascadeEpicRepo != nil && s.cascadeHistQuerier != nil && s.cascadeHistTx != nil
}

// cascadeDepsBundle packages the cascade dependencies into the cascadeDeps struct.
func (s *TaskService) cascadeDepsBundle() cascadeDeps {
	return cascadeDeps{
		db:             s.cascadeDB,
		featureRepo:    s.cascadeFeatureRepo,
		epicRepo:       s.cascadeEpicRepo,
		historyQuerier: s.cascadeHistQuerier,
		historyTx:      s.cascadeHistTx,
		workflowSvc:    &workflowProviderAdapter{svc: s.entitySvc.GetWorkflowService()},
	}
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
// and reopens it. When cascade deps are wired, it uses cascadeParentReopens for
// the full two-level walk (feature + epic) with history lookup. When cascade deps
// are not wired, it falls back to the legacy single-level aggregation-status jump.
//
// Best-effort: logs a warning on failure, never fails the caller.
//
// Parameters:
//   - ctx: context for cancellation
//   - featureKey: key of the parent feature (e.g., "E01-F01")
//   - taskKey: key of the newly created task (for audit logging)
func (s *TaskService) maybeReopenParentFeature(ctx context.Context, featureKey string, taskKey string) {
	if !s.cascadeEnabled() {
		// Preserve legacy behavior when cascade dependencies are not wired (e.g., in tests).
		s.legacyMaybeReopenParentFeature(ctx, featureKey, taskKey)
		return
	}

	// Look up the feature by key to obtain its ID for the cascade trigger.
	if s.featureService == nil {
		return
	}
	feature, err := s.featureService.GetFeature(ctx, featureKey)
	if err != nil || feature == nil {
		return
	}

	cascadeParentReopens(ctx, s.cascadeDepsBundle(), cascadeTrigger{
		triggerKey:  taskKey,
		triggerKind: "creation",
		triggerType: models.EntityTypeTask,
		startLeg:    cascadeLegFeature,
		featureID:   feature.ID,
	})
}

// legacyMaybeReopenParentFeature is the original inline reopen logic preserved
// verbatim as a fallback for when cascade dependencies are not wired (AC-T3).
func (s *TaskService) legacyMaybeReopenParentFeature(ctx context.Context, featureKey string, taskKey string) {
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

	// Best-effort side effect: on no candidate OR an ambiguous config we skip
	// the reopen and warn rather than guessing a target, EXCEPT that a workflow
	// with no aggregation step still falls back to its initial status, matching
	// the other reopen paths.
	target, err := featureWf.PrimaryAggregationStatus()
	if err != nil {
		var noCandidate *config.NoCandidateError
		if errors.As(err, &noCandidate) {
			target = featureWf.GetInitialStatusString()
		} else {
			slog.Warn("auto-reopen of feature skipped", "feature", featureKey, "error", err)
			return
		}
	}
	targetStatus := models.FeatureStatus(target)

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
	// tx, when set, makes the status write participate in the caller-owned
	// aggregate transaction.
	tx *sql.Tx
	// opts is set before each TransitionStatus call to pass transition-specific
	// metadata (agent, reason, documentPath, force) to StatusUpdateRaw.
	opts *statusTransitionOpts
	// lastTask caches the task fetched by GetByKey so UpdateStatus can access
	// timestamp fields without an extra DB round-trip.
	lastTask *models.Task
	// unblockedKeys captures the auto-unblocked task keys from StatusUpdateRaw,
	// retrieved by TaskService after EntityService.TransitionStatus returns.
	unblockedKeys []string
	// terminalStatuses is the task-level terminal-status list resolved from the
	// workflow service. Passed down to StatusUpdateRaw so the repository's
	// auto-unblock gate and dependency-satisfaction check use configured
	// terminality instead of a hardcoded completed/archived pair.
	terminalStatuses []string
	// executionStatuses and blockedStatuses classify phase transitions for the
	// repository's historical timestamp columns; unblockedStatus is the task
	// workflow's entry target for dependency recovery.
	executionStatuses []string
	blockedStatuses   []string
	unblockedStatus   models.TaskStatus
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

		TerminalStatuses:  a.terminalStatuses,
		ExecutionStatuses: a.executionStatuses,
		BlockedStatuses:   a.blockedStatuses,
		UnblockedStatus:   a.unblockedStatus,
	}

	var unblockedKeys []string
	var err error
	if a.tx != nil {
		unblockedKeys, err = a.repo.StatusUpdateRawWithTx(ctx, a.tx, params)
	} else {
		unblockedKeys, err = a.repo.StatusUpdateRaw(ctx, params)
	}
	if err != nil {
		return err
	}
	a.unblockedKeys = unblockedKeys
	return nil
}

func (a *taskEntityRepoAdapter) UpdateStatusIfCurrent(ctx context.Context, id int64, expectedCurrentStatus, newStatus string) (bool, error) {
	task := a.lastTask
	if task == nil || task.ID != id {
		var err error
		task, err = a.repo.GetByID(ctx, id)
		if err != nil {
			return false, fmt.Errorf("failed to get task for guarded status update: %w", err)
		}
	}
	if task == nil {
		return false, nil
	}
	// The status comparison above (and a.lastTask generally) is a cheap
	// fast-path only — it can be stale under concurrent writers. Correctness
	// comes from Guarded:true below, which makes the WHERE clause of the
	// UPDATE itself the compare-and-swap, evaluated atomically by SQLite as
	// part of one statement. Do not rely on this pre-check to reject a stale
	// caller; it exists only to skip building params when obviously stale.
	if !strings.EqualFold(string(task.Status), expectedCurrentStatus) {
		return false, nil
	}

	if a.opts == nil {
		return a.repo.UpdateStatusIfCurrent(ctx, id, models.TaskStatus(expectedCurrentStatus), models.TaskStatus(newStatus))
	}

	params := models.StatusUpdateParams{
		TaskID:          id,
		NewStatus:       models.TaskStatus(newStatus),
		Agent:           a.opts.agent,
		Notes:           a.opts.notes,
		RejectionReason: a.opts.reason,
		DocumentPath:    a.opts.documentPath,
		Force:           a.opts.force,
		OldStatus:       expectedCurrentStatus,
		TaskKey:         task.Key,
		StartedAt:       task.StartedAt,
		CompletedAt:     task.CompletedAt,
		BlockedAt:       task.BlockedAt,
		Guarded:         true,

		TerminalStatuses:  a.terminalStatuses,
		ExecutionStatuses: a.executionStatuses,
		BlockedStatuses:   a.blockedStatuses,
		UnblockedStatus:   a.unblockedStatus,
	}

	var unblockedKeys []string
	var err error
	if a.tx != nil {
		unblockedKeys, err = a.repo.StatusUpdateRawWithTx(ctx, a.tx, params)
	} else {
		unblockedKeys, err = a.repo.StatusUpdateRaw(ctx, params)
	}
	if errors.Is(err, models.ErrGuardedUpdateStale) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	a.unblockedKeys = unblockedKeys
	return true, nil
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
		terminalStatuses:  s.taskTerminalStatuses(),
		executionStatuses: s.taskExecutionStatuses(),
		blockedStatuses:   s.taskStatusesByPhase("blocked"),
		unblockedStatus:   models.TaskStatus(s.entitySvc.GetWorkflowService().GetDefaultStatus()),
	}
}

// taskTerminalStatuses returns the configured terminal statuses for the task
// workflow. Used to hand the repository layer a config-driven terminal set
// instead of letting it apply its hardcoded fallback.
func (s *TaskService) taskTerminalStatuses() []string {
	wf := s.entitySvc.GetWorkflowService().ForLevel(workflow.LevelTask)
	if wf == nil {
		return nil
	}
	return wf.GetTerminalStatuses()
}

func (s *TaskService) taskStatusesByPhase(phase string) []string {
	wf := s.entitySvc.GetWorkflowService().ForLevel(workflow.LevelTask)
	if wf == nil {
		return nil
	}
	return wf.GetStatusesByPhase(phase)
}

// taskExecutionStatuses accepts the current workflow vocabulary (execution)
// and the established task-workflow vocabulary (development). The latter is
// the default shipped task workflow's active work phase.
func (s *TaskService) taskExecutionStatuses() []string {
	statuses := s.taskStatusesByPhase("execution")
	if len(statuses) != 0 {
		return statuses
	}
	return s.taskStatusesByPhase("development")
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
