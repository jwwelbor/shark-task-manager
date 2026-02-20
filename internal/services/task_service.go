package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/taskcreation"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// TaskRelationshipRepository defines the interface for accessing task relationships.
type TaskRelationshipRepository = config.TaskRelationshipRepository

// TaskRelationshipQueryRepository defines the extended interface for querying task relationships
// by direction and type. This interface is satisfied by *repository.TaskRelationshipRepository.
// It is used by the deps/blocked-by/blocks commands which need richer relationship queries
// than the basic ListRelatedTaskKeys method provided by config.TaskRelationshipRepository.
type TaskRelationshipQueryRepository interface {
	// GetByTaskID retrieves all relationships for a task (both incoming and outgoing)
	GetByTaskID(ctx context.Context, taskID int64) ([]*models.TaskRelationship, error)
	// GetOutgoing retrieves all outgoing relationships for a task filtered by type
	GetOutgoing(ctx context.Context, taskID int64, relTypes []string) ([]*models.TaskRelationship, error)
	// GetIncoming retrieves all incoming relationships for a task filtered by type
	GetIncoming(ctx context.Context, taskID int64, relTypes []string) ([]*models.TaskRelationship, error)
}

// TaskWritableDocumentRepository defines the interface for document write operations on tasks.
// This interface is satisfied by *repository.DocumentRepository.
// The existing config.DocumentRepository only exposes read-only List methods; this interface
// adds the write operations needed by LinkDocument and UnlinkDocument.
type TaskWritableDocumentRepository interface {
	CreateOrGet(ctx context.Context, title, filePath string) (*models.Document, error)
	GetByTitle(ctx context.Context, title string) (*models.Document, error)
	LinkToTask(ctx context.Context, taskID, documentID int64) error
	UnlinkFromTask(ctx context.Context, taskID, documentID int64) error
}

// TaskDependencyRepository defines the interface for managing task dependency relationships.
// This interface is satisfied by *repository.TaskRelationshipRepository.
type TaskDependencyRepository interface {
	// Create creates a new task relationship (used for adding dependencies)
	Create(ctx context.Context, rel *models.TaskRelationship) error
	// GetOutgoing retrieves all outgoing relationships for a task filtered by type
	GetOutgoing(ctx context.Context, taskID int64, relTypes []string) ([]*models.TaskRelationship, error)
	// GetIncoming retrieves all incoming relationships for a task filtered by type
	GetIncoming(ctx context.Context, taskID int64, relTypes []string) ([]*models.TaskRelationship, error)
	// DeleteByTasksAndType removes a specific relationship between two tasks by type
	DeleteByTasksAndType(ctx context.Context, fromTaskID, toTaskID int64, relType string) error
	// Delete removes a relationship by ID
	Delete(ctx context.Context, id int64) error
	// DetectCycle checks if creating a relationship would create a circular dependency
	DetectCycle(ctx context.Context, fromTaskID, toTaskID int64, relType string) error
}

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
	ListByEpic(ctx context.Context, epicKey string) ([]*models.Task, error)

	// Dependency operations
	GetTaskDependents(ctx context.Context, taskKey string) ([]*models.Task, error)

	// Status operations (service layer will wrap these with business logic)
	UpdateStatus(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error
	UpdateStatusForced(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string, rejectionReason *string, documentPath *string, force bool) error
	// UpdateStatusForcedWithUnblock atomically updates status and auto-unblocks dependent tasks.
	// Returns the list of task keys that were automatically unblocked.
	UpdateStatusForcedWithUnblock(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string, rejectionReason *string, documentPath *string, force bool) ([]string, error)

	// Search operations
	FindByFileChanged(ctx context.Context, filePath string) ([]*models.Task, error)

	// Key prefix search - returns all tasks whose key starts with the given prefix.
	// Used for key generation to avoid UNIQUE constraint collisions.
	ListByKeyPrefix(ctx context.Context, prefix string) ([]*models.Task, error)
}

// TaskNoteRepository defines the note repository interface for rejection notes.
type TaskNoteRepository interface {
	CreateRejectionNote(ctx context.Context, entityType models.EntityType, entityID int64,
		historyID int64, fromStatus, toStatus, reason, rejectedBy string, documentPath *string) (*models.EntityNote, error)
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
	repo            TaskRepository
	workflowSvc     *workflow.Service
	creatorSvc      *taskcreation.Creator
	noteRepo        TaskNoteRepository
	historyRepo     TaskHistoryRepository
	docRepo         config.DocumentRepository
	writableDocRepo TaskWritableDocumentRepository
	relRepo         config.TaskRelationshipRepository
	relQueryRepo    TaskRelationshipQueryRepository
	depRepo         TaskDependencyRepository
	sessionRepo     WorkSessionRepository
	epicRepo        AnalyticsEpicRepository
	featureRepo     AnalyticsFeatureRepository
	featureService  *FeatureService // optional: triggers progress recalc on status change
}

// NewTaskService creates a new TaskService with the required dependencies.
// The workflow service is automatically scoped to the task level.
// creatorSvc, noteRepo, docRepo, and relRepo can be nil for graceful degradation.
//
// Parameters:
//   - repo: task repository for data access (required, panics if nil)
//   - workflowSvc: workflow service for status validation and transitions (required, panics if nil)
//   - creatorSvc: task creation service for key generation and file creation (optional)
//   - noteRepo: note repository for rejection tracking (optional)
//
// Returns:
//   - *TaskService: configured task service instance
//
// Panics:
//   - If repo is nil (required dependency)
//   - If workflowSvc is nil (required dependency)
func NewTaskService(repo TaskRepository, workflowSvc *workflow.Service, creatorSvc *taskcreation.Creator, noteRepo TaskNoteRepository) *TaskService {
	if repo == nil {
		panic("TaskService requires a non-nil TaskRepository")
	}
	if workflowSvc == nil {
		panic("TaskService requires a non-nil workflow.Service")
	}
	return &TaskService{
		repo:        repo,
		workflowSvc: workflowSvc.ForLevel(workflow.LevelTask),
		creatorSvc:  creatorSvc,
		noteRepo:    noteRepo,
		docRepo:     nil,
		relRepo:     nil,
	}
}

// NewTaskServiceWithRelationships creates a TaskService with document and relationship repositories.
// Use this constructor when orchestrator actions need to populate related documents and tasks.
//
// Parameters:
//   - repo: task repository for data access (required, panics if nil)
//   - workflowSvc: workflow service for status validation and transitions (required, panics if nil)
//   - creatorSvc: task creation service for key generation and file creation (optional)
//   - noteRepo: note repository for rejection tracking (optional)
//   - docRepo: document repository for related documents (optional)
//   - relRepo: relationship repository for related tasks (optional)
//
// Returns:
//   - *TaskService: configured task service instance with relationship support
//
// Panics:
//   - If repo is nil (required dependency)
//   - If workflowSvc is nil (required dependency)
func NewTaskServiceWithRelationships(repo TaskRepository, workflowSvc *workflow.Service, creatorSvc *taskcreation.Creator, noteRepo TaskNoteRepository, docRepo config.DocumentRepository, relRepo config.TaskRelationshipRepository, sessionRepo WorkSessionRepository) *TaskService {
	if repo == nil {
		panic("TaskService requires a non-nil TaskRepository")
	}
	if workflowSvc == nil {
		panic("TaskService requires a non-nil workflow.Service")
	}
	return &TaskService{
		repo:        repo,
		workflowSvc: workflowSvc.ForLevel(workflow.LevelTask),
		creatorSvc:  creatorSvc,
		noteRepo:    noteRepo,
		docRepo:     docRepo,
		relRepo:     relRepo,
		sessionRepo: sessionRepo,
	}
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
	// Validate required fields
	if input.EpicKey == "" {
		return nil, fmt.Errorf("failed to create task: epic key is required")
	}
	if input.FeatureKey == "" {
		return nil, fmt.Errorf("failed to create task: feature key is required")
	}
	if input.Title == "" {
		return nil, fmt.Errorf("failed to create task: title is required")
	}

	// Validate optional fields
	if input.Priority > 10 {
		return nil, fmt.Errorf("failed to create task: priority must be between 1 and 10")
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
			return nil, fmt.Errorf("failed to create task: %w", err)
		}
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
		return nil, fmt.Errorf("failed to create task: could not query existing keys: %w", err)
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
	task := &models.Task{
		Key:            taskKey,
		Title:          input.Title,
		Status:         models.TaskStatus(s.workflowSvc.GetDefaultStatus()),
		Priority:       priority,
		AgentType:      &agentType,
		ExecutionOrder: nil,
	}

	// Validate model
	if err := task.Validate(); err != nil {
		return nil, fmt.Errorf("failed to create task: validation error: %w", err)
	}

	// Save to repository
	if err := s.repo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

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
	task, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get task %s: %w", key, err)
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
	// Get existing task
	task, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to update task %s: %w", key, err)
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
		return nil, fmt.Errorf("failed to update task %s: validation error: %w", key, err)
	}

	// Save updated task
	if err := s.repo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to update task %s: %w", key, err)
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
	// Get task
	task, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete task %s: %w", key, err)
	}

	// Check for dependent tasks
	dependents, err := s.repo.GetTaskDependents(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete task %s: error checking dependents: %w", key, err)
	}
	if len(dependents) > 0 {
		return fmt.Errorf("failed to delete task %s: task has dependent tasks that must be deleted first", key)
	}

	// Delete task
	if err := s.repo.Delete(ctx, task.ID); err != nil {
		return fmt.Errorf("failed to delete task %s: %w", key, err)
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
	// Scope the DB query to the requested epic/feature for efficiency.
	var tasks []*models.Task
	var err error

	switch {
	case filters.FeatureKey != "" && s.featureRepo != nil:
		// Look up the feature to get its DB ID, then query tasks for that feature.
		feature, ferr := s.featureRepo.GetByKey(ctx, filters.FeatureKey)
		if ferr != nil {
			return nil, fmt.Errorf("feature not found: %s: %w", filters.FeatureKey, ferr)
		}
		tasks, err = s.repo.ListByFeature(ctx, feature.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to list tasks for feature %s: %w", filters.FeatureKey, err)
		}
	case filters.EpicKey != "":
		tasks, err = s.repo.ListByEpic(ctx, filters.EpicKey)
		if err != nil {
			return nil, fmt.Errorf("failed to list tasks for epic %s: %w", filters.EpicKey, err)
		}
	default:
		tasks, err = s.repo.List(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list tasks: %w", err)
		}
	}

	// Apply filters
	var filtered []*models.Task
	for _, task := range tasks {
		// Filter by status
		if filters.Status != "" && string(task.Status) != filters.Status {
			continue
		}

		// Filter by agent type
		if filters.AgentType != "" {
			if task.AgentType == nil || *task.AgentType != filters.AgentType {
				continue
			}
		}

		// Filter by blocked
		if filters.Blocked && string(task.Status) != "blocked" {
			continue
		}

		// Exclude completed unless show_all
		if !filters.ShowAll && string(task.Status) == "completed" {
			continue
		}

		// Filter by title search (case-insensitive substring)
		if filters.TitleSearch != "" {
			if !strings.Contains(strings.ToLower(task.Title), strings.ToLower(filters.TitleSearch)) {
				continue
			}
		}

		// Filter by priority range
		if filters.MinPriority > 0 && task.Priority < filters.MinPriority {
			continue
		}
		if filters.MaxPriority > 0 && task.Priority > filters.MaxPriority {
			continue
		}

		filtered = append(filtered, task)
	}

	// Sort by execution order (ascending), then priority (descending)
	sortTasks(filtered)

	return filtered, nil
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

// GetNextTask retrieves the next available task for the given agent/filters.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - filters: criteria for task selection (agent type, epic, priority, etc.)
//
// Returns:
//   - *models.Task: the next task to work on, or nil if no tasks available
//   - error: repository errors
//
// Errors:
//   - RepositoryError: database query failed
func (s *TaskService) GetNextTask(ctx context.Context, filters NextTaskFilters) (*models.Task, error) {
	// TODO: Implementation in F02-F07
	return nil, fmt.Errorf("not implemented")
}

// StartTask transitions a task to in_progress (or appropriate start status).
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: task key to start
//   - agentID: optional agent identifier for tracking who started the task
//
// Returns:
//   - *models.Task: the updated task
//   - error: validation errors, workflow errors, or repository errors
//
// Errors:
//   - NotFoundError: task not found
//   - WorkflowError: invalid transition (task not in a startable status)
//   - DependencyError: task dependencies not met
//   - RepositoryError: database update failed
func (s *TaskService) StartTask(ctx context.Context, key string, agentID string) (*models.Task, error) {
	// Step 1: Get task
	task, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to start task %s: %w", key, err)
	}

	// Step 2: Validate workflow transition
	if err := s.workflowSvc.ValidateTransition(string(task.Status), "in_progress"); err != nil {
		return nil, fmt.Errorf("cannot start task %s in status %s: %w", key, task.Status, err)
	}

	// Step 3: Update status to in_progress
	if err := s.repo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), &agentID, nil); err != nil {
		return nil, fmt.Errorf("failed to update task %s status: %w", key, err)
	}

	// Step 4: Return updated task
	task.Status = models.TaskStatus("in_progress")
	s.recalculateFeatureProgress(ctx, task.FeatureID)
	return task, nil
}

// CompleteTask marks a task as complete (transitions to ready_for_review or similar).
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: task key to complete
//   - notes: optional completion notes
//
// Returns:
//   - *models.Task: the updated task
//   - error: validation errors, workflow errors, or repository errors
//
// Errors:
//   - NotFoundError: task not found
//   - WorkflowError: invalid transition (task not in a completable status)
//   - RepositoryError: database update failed
func (s *TaskService) CompleteTask(ctx context.Context, key string, notes string) (*models.Task, error) {
	// Step 1: Get task
	task, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to complete task %s: %w", key, err)
	}

	// Step 2: Validate workflow transition (typically in_progress -> ready_for_review)
	if err := s.workflowSvc.ValidateTransition(string(task.Status), "ready_for_review"); err != nil {
		return nil, fmt.Errorf("cannot complete task %s in status %s: %w", key, task.Status, err)
	}

	// Step 3: Update status to ready_for_review with notes
	var notesPtr *string
	if notes != "" {
		notesPtr = &notes
	}
	if err := s.repo.UpdateStatus(ctx, task.ID, models.TaskStatus("ready_for_review"), nil, notesPtr); err != nil {
		return nil, fmt.Errorf("failed to update task %s status: %w", key, err)
	}

	// Step 4: Return updated task
	task.Status = models.TaskStatus("ready_for_review")
	s.recalculateFeatureProgress(ctx, task.FeatureID)
	return task, nil
}

// ApproveTask approves a completed task (final completion).
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: task key to approve
//   - notes: optional approval notes
//
// Returns:
//   - *models.Task: the updated task
//   - error: validation errors, workflow errors, or repository errors
//
// Errors:
//   - NotFoundError: task not found
//   - WorkflowError: invalid transition (task not in an approvable status)
//   - RepositoryError: database update failed
func (s *TaskService) ApproveTask(ctx context.Context, key string, notes string) (*models.Task, error) {
	// Step 1: Get task
	task, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to approve task %s: %w", key, err)
	}

	// Step 2: Validate workflow transition (typically ready_for_review -> completed)
	if err := s.workflowSvc.ValidateTransition(string(task.Status), "completed"); err != nil {
		return nil, fmt.Errorf("cannot approve task %s in status %s: %w", key, task.Status, err)
	}

	// Step 3: Update status to completed with optional notes
	var notesPtr *string
	if notes != "" {
		notesPtr = &notes
	}
	if err := s.repo.UpdateStatus(ctx, task.ID, models.TaskStatus("completed"), nil, notesPtr); err != nil {
		return nil, fmt.Errorf("failed to update task %s status: %w", key, err)
	}

	// Step 4: Return updated task
	task.Status = models.TaskStatus("completed")
	s.recalculateFeatureProgress(ctx, task.FeatureID)
	return task, nil
}

// ReopenTask moves a task back to in_progress.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: task key to reopen
//   - notes: optional notes explaining why task was reopened
//
// Returns:
//   - *models.Task: the updated task
//   - error: validation errors, workflow errors, or repository errors
//
// Errors:
//   - NotFoundError: task not found
//   - WorkflowError: invalid transition
//   - RepositoryError: database update failed
func (s *TaskService) ReopenTask(ctx context.Context, key string, notes string) (*models.Task, error) {
	// Step 1: Get task
	task, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to reopen task %s: %w", key, err)
	}

	// Step 2: Validate workflow transition (typically ready_for_review -> in_progress)
	if err := s.workflowSvc.ValidateTransition(string(task.Status), "in_progress"); err != nil {
		return nil, fmt.Errorf("cannot reopen task %s in status %s: %w", key, task.Status, err)
	}

	// Step 3: Update status to in_progress with notes explaining why reopened
	var notesPtr *string
	if notes != "" {
		notesPtr = &notes
	}
	if err := s.repo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), nil, notesPtr); err != nil {
		return nil, fmt.Errorf("failed to update task %s status: %w", key, err)
	}

	// Step 4: Return updated task
	task.Status = models.TaskStatus("in_progress")
	s.recalculateFeatureProgress(ctx, task.FeatureID)
	return task, nil
}

// BlockTask marks a task as blocked with a reason.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: task key to block
//   - reason: explanation of what is blocking the task (required)
//
// Returns:
//   - *models.Task: the updated task
//   - error: validation errors, workflow errors, or repository errors
//
// Errors:
//   - NotFoundError: task not found
//   - ValidationError: missing or empty reason
//   - RepositoryError: database update failed
func (s *TaskService) BlockTask(ctx context.Context, key string, reason string) (*models.Task, error) {
	// Step 1: Validate reason is provided
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("block reason cannot be empty")
	}

	// Step 2: Get task
	task, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to block task %s: %w", key, err)
	}

	// Step 3: Validate workflow transition to blocked
	if err := s.workflowSvc.ValidateTransition(string(task.Status), "blocked"); err != nil {
		return nil, fmt.Errorf("cannot block task %s in status %s: %w", key, task.Status, err)
	}

	// Step 4: Update status to blocked with reason as notes
	if err := s.repo.UpdateStatus(ctx, task.ID, models.TaskStatus("blocked"), nil, &reason); err != nil {
		return nil, fmt.Errorf("failed to update task %s status: %w", key, err)
	}

	// Step 5: Return updated task
	task.Status = models.TaskStatus("blocked")
	s.recalculateFeatureProgress(ctx, task.FeatureID)
	return task, nil
}

// UnblockTask removes the blocked status from a task.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: task key to unblock
//
// Returns:
//   - *models.Task: the updated task
//   - []string: list of dependent task keys that were also auto-unblocked
//   - error: validation errors, workflow errors, or repository errors
//
// Errors:
//   - NotFoundError: task not found
//   - WorkflowError: task is not currently blocked
//   - RepositoryError: database update failed
func (s *TaskService) UnblockTask(ctx context.Context, key string) (*models.Task, []string, error) {
	// Step 1: Get task
	task, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unblock task %s: %w", key, err)
	}

	// Step 2: Validate task is currently blocked
	if task.Status != models.TaskStatus("blocked") {
		return nil, nil, fmt.Errorf("cannot unblock task %s: task is not blocked (current status: %s)", key, task.Status)
	}

	// Step 3: Validate workflow transition from blocked to todo
	if err := s.workflowSvc.ValidateTransition(string(task.Status), "todo"); err != nil {
		return nil, nil, fmt.Errorf("cannot unblock task %s: %w", key, err)
	}

	// Step 4: Update status to todo
	if err := s.repo.UpdateStatus(ctx, task.ID, models.TaskStatus("todo"), nil, nil); err != nil {
		return nil, nil, fmt.Errorf("failed to update task %s status: %w", key, err)
	}

	// Step 5: Return updated task with empty suggestions (can be enhanced later with dependency analysis)
	task.Status = models.TaskStatus("todo")
	s.recalculateFeatureProgress(ctx, task.FeatureID)
	return task, []string{}, nil
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
	// Step 1: Get task
	task, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get task %s: %w", key, err)
	}

	fromStatus := string(task.Status)

	// Step 2: Validate transition (unless forced)
	if !opts.Force {
		if err := s.workflowSvc.ValidateTransition(fromStatus, targetStatus); err != nil {
			return nil, fmt.Errorf("invalid transition for task %s: %w", key, err)
		}
	}

	// Step 3: Prepare options
	var agentPtr *string
	if opts.Agent != "" {
		agentPtr = &opts.Agent
	}
	var reasonPtr *string
	if opts.Reason != "" {
		reasonPtr = &opts.Reason
	}
	var docPathPtr *string
	if opts.DocumentPath != "" {
		docPathPtr = &opts.DocumentPath
	}

	// Step 4: Perform transition with auto-unblock
	unblockedKeys, err := s.repo.UpdateStatusForcedWithUnblock(ctx, task.ID, models.TaskStatus(targetStatus), agentPtr, nil, reasonPtr, docPathPtr, opts.Force)
	if err != nil {
		return nil, fmt.Errorf("failed to transition task %s to %s: %w", key, targetStatus, err)
	}
	s.recalculateFeatureProgress(ctx, task.FeatureID)

	// Step 5: Build result with orchestrator action for new status
	result := &TransitionResult{
		EntityType:         "task",
		EntityKey:          task.Key,
		FromStatus:         fromStatus,
		ToStatus:           targetStatus,
		Transitioned:       true,
		Message:            fmt.Sprintf("Transitioned: %s -> %s", fromStatus, targetStatus),
		IsForced:           opts.Force,
		Reason:             opts.Reason,
		OrchestratorAction: s.resolveAction(ctx, task, targetStatus),
	}

	// Step 6: Store auto-unblocked keys in result message if any
	if len(unblockedKeys) > 0 {
		result.Message = fmt.Sprintf("Transitioned: %s -> %s (auto-unblocked: %s)",
			fromStatus, targetStatus, strings.Join(unblockedKeys, ", "))
	}

	return result, nil
}

// GetNextStatus returns available status transitions for a task.
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
	task, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get task %s: %w", key, err)
	}

	currentStatus := string(task.Status)
	transitions := s.workflowSvc.GetTransitionInfo(currentStatus)
	currentMeta := s.workflowSvc.GetStatusMetadata(currentStatus)

	// Wrap transitions with orchestrator action support
	wrapped := make([]TransitionInfoWithAction, 0, len(transitions))
	for _, t := range transitions {
		wrapped = append(wrapped, TransitionInfoWithAction{
			TransitionInfo:     t,
			OrchestratorAction: s.resolveAction(ctx, task, t.TargetStatus),
		})
	}

	return &NextStatusInfo{
		EntityType:           "task",
		EntityKey:            key,
		CurrentStatus:        currentStatus,
		CurrentPhase:         currentMeta.Phase,
		AvailableTransitions: wrapped,
		IsTerminal:           s.workflowSvc.IsTerminalStatus(currentStatus),
	}, nil
}

// ValidateStatus checks if a status is valid in the task workflow.
//
// Parameters:
//   - status: status string to validate
//
// Returns:
//   - error: WorkflowError if status is invalid
func (s *TaskService) ValidateStatus(status string) error {
	return s.workflowSvc.ValidateStatus(status)
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
	// Get task dependencies
	dependents, err := s.repo.GetTaskDependents(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get dependencies for task %s: %w", key, err)
	}

	if len(dependents) == 0 {
		return nil // No dependencies to validate
	}

	// Check each dependency is completed
	for _, dep := range dependents {
		if dep.Status != models.TaskStatus("completed") {
			return fmt.Errorf("dependency not met: task %s depends on %s which is in status %s (must be completed)", key, dep.Key, dep.Status)
		}
	}

	// Detect circular dependencies using DFS
	if err := s.detectCircularDependency(ctx, key, make(map[string]bool), make(map[string]bool)); err != nil {
		return fmt.Errorf("circular dependency detected: %w", err)
	}

	return nil
}

// detectCircularDependency uses depth-first search to detect cycles in the dependency graph.
func (s *TaskService) detectCircularDependency(ctx context.Context, taskKey string, visited map[string]bool, recStack map[string]bool) error {
	visited[taskKey] = true
	recStack[taskKey] = true

	dependents, err := s.repo.GetTaskDependents(ctx, taskKey)
	if err != nil {
		return err
	}

	for _, dep := range dependents {
		if !visited[dep.Key] {
			if err := s.detectCircularDependency(ctx, dep.Key, visited, recStack); err != nil {
				return err
			}
		} else if recStack[dep.Key] {
			return fmt.Errorf("cycle detected: %s → %s", taskKey, dep.Key)
		}
	}

	recStack[taskKey] = false
	return nil
}

// GetDependencyTree retrieves the full dependency tree for a task.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: task key to analyze
//
// Returns:
//   - *DependencyTree: hierarchical structure of dependencies and dependents
//   - error: not found errors or repository errors
//
// Errors:
//   - NotFoundError: task not found
//   - RepositoryError: database query failed
func (s *TaskService) GetDependencyTree(ctx context.Context, key string) (*DependencyTree, error) {
	// Get root task
	task, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependency tree for task %s: %w", key, err)
	}

	// Build dependency tree
	tree := &DependencyTree{
		Task: &TaskNode{
			Key:         task.Key,
			Title:       task.Title,
			Status:      string(task.Status),
			Priority:    task.Priority,
			IsCompleted: task.Status == models.TaskStatus("completed"),
			IsBlocked:   task.Status == models.TaskStatus("blocked"),
			UpdatedAt:   task.UpdatedAt,
		},
		Dependencies: []*TaskNode{},
		Dependents:   []*TaskNode{},
		Blocked:      false,
		BlockedBy:    []string{},
		CanStart:     true,
		Depth:        0,
	}

	// Get dependencies (tasks this task depends on)
	dependents, err := s.repo.GetTaskDependents(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependencies: %w", err)
	}

	for _, dep := range dependents {
		depNode := &TaskNode{
			Key:         dep.Key,
			Title:       dep.Title,
			Status:      string(dep.Status),
			Priority:    dep.Priority,
			IsCompleted: dep.Status == models.TaskStatus("completed"),
			IsBlocked:   dep.Status == models.TaskStatus("blocked"),
			UpdatedAt:   dep.UpdatedAt,
		}
		tree.Dependencies = append(tree.Dependencies, depNode)

		// Check if dependency blocks starting
		if dep.Status != models.TaskStatus("completed") {
			tree.Blocked = true
			tree.BlockedBy = append(tree.BlockedBy, dep.Key)
			tree.CanStart = false
		}
	}

	return tree, nil
}

// resolveAction looks up the orchestrator action for a given status.
// Returns nil if no action is defined or if document/relationship repositories are not available.
func (s *TaskService) resolveAction(ctx context.Context, task *models.Task, status string) *config.PopulatedAction {
	wf := s.workflowSvc.GetWorkflow()
	if wf == nil || wf.StatusMetadata == nil {
		return nil
	}
	meta, exists := wf.StatusMetadata[status]
	if !exists || meta.OrchestratorAction == nil {
		return nil
	}

	// Determine which placeholder function to use based on available repositories
	var placeholders map[string]string
	if s.docRepo != nil && s.relRepo != nil {
		// Use function that includes related documents and tasks
		placeholders = config.TaskPlaceholdersWithRelated(ctx, task, s.docRepo, s.relRepo)
	} else {
		// Fall back to basic placeholders
		placeholders = config.TaskPlaceholders(task)
	}

	return &config.PopulatedAction{
		Action:      meta.OrchestratorAction.Action,
		AgentType:   meta.OrchestratorAction.AgentType,
		Skills:      meta.OrchestratorAction.Skills,
		Instruction: meta.OrchestratorAction.PopulateTemplate(placeholders),
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
	// Get filtered tasks using existing ListTasks
	allTasks, err := s.ListTasks(ctx, filters)
	if err != nil {
		return nil, 0, err
	}

	total := len(allTasks)

	// Apply pagination
	start := filters.Offset
	if start > total {
		return []*models.Task{}, total, nil
	}

	// If limit is 0, return all tasks after offset
	if filters.Limit == 0 {
		return allTasks[start:], total, nil
	}

	end := start + filters.Limit
	if end > total {
		end = total
	}

	return allTasks[start:end], total, nil
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
	// Always show all statuses for aggregation
	filters.ShowAll = true

	// Get filtered tasks
	tasks, err := s.ListTasks(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks by status: %w", err)
	}

	// Group by status
	statusMap := make(map[string]int)
	for _, task := range tasks {
		statusMap[string(task.Status)]++
	}

	return statusMap, nil
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
	// Get filtered tasks
	tasks, err := s.ListTasks(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks by agent: %w", err)
	}

	// Group by agent type
	agentMap := make(map[string]int)
	for _, task := range tasks {
		if task.AgentType != nil {
			agentMap[*task.AgentType]++
		}
	}

	return agentMap, nil
}

// SetDepRepo sets the dependency repository on the service.
// This is used when the service is created via NewTaskService (which does not accept depRepo)
// but the caller needs dependency management functionality (e.g., AddDependency, RemoveDependency).
func (s *TaskService) SetDepRepo(depRepo TaskDependencyRepository) {
	s.depRepo = depRepo
}

// AddDependency creates a dependency relationship between two tasks.
// The task identified by taskKey will depend on the task identified by depKey,
// meaning depKey must be completed before taskKey can proceed.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - taskKey: the key of the dependent task (e.g., "E07-F01-002")
//   - depKey: the key of the dependency task (e.g., "E07-F01-001")
//
// Returns:
//   - error: validation errors (task not found, self-dependency) or repository errors
//
// Errors:
//   - NotFoundError: task or dependency task not found
//   - ValidationError: task cannot depend on itself
//   - RepositoryError: database operation failed
func (s *TaskService) AddDependency(ctx context.Context, taskKey, depKey string) error {
	if s.depRepo == nil {
		return fmt.Errorf("dependency repository not configured")
	}

	task, err := s.repo.GetByKey(ctx, taskKey)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	dep, err := s.repo.GetByKey(ctx, depKey)
	if err != nil {
		return fmt.Errorf("dependency task not found: %w", err)
	}

	if task.ID == dep.ID {
		return fmt.Errorf("task cannot depend on itself")
	}

	rel := &models.TaskRelationship{
		FromTaskID:       task.ID,
		ToTaskID:         dep.ID,
		RelationshipType: "depends_on",
	}

	if err := s.depRepo.Create(ctx, rel); err != nil {
		return fmt.Errorf("failed to add dependency from %s to %s: %w", taskKey, depKey, err)
	}

	return nil
}

// RemoveDependency removes a dependency relationship between two tasks.
// The dependency from taskKey to depKey will be removed.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - taskKey: the key of the dependent task (e.g., "E07-F01-002")
//   - depKey: the key of the dependency task to remove (e.g., "E07-F01-001")
//
// Returns:
//   - error: validation errors (task not found) or repository errors
//
// Errors:
//   - NotFoundError: task or dependency task not found
//   - RepositoryError: database operation failed or dependency does not exist
func (s *TaskService) RemoveDependency(ctx context.Context, taskKey, depKey string) error {
	if s.depRepo == nil {
		return fmt.Errorf("dependency repository not configured")
	}

	task, err := s.repo.GetByKey(ctx, taskKey)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	dep, err := s.repo.GetByKey(ctx, depKey)
	if err != nil {
		return fmt.Errorf("dependency task not found: %w", err)
	}

	if err := s.depRepo.DeleteByTasksAndType(ctx, task.ID, dep.ID, "depends_on"); err != nil {
		return fmt.Errorf("failed to remove dependency from %s to %s: %w", taskKey, depKey, err)
	}

	return nil
}

// ListDependencies returns all tasks that the given task depends on.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - taskKey: the key of the task to list dependencies for
//
// Returns:
//   - []*models.Task: the tasks that taskKey depends on
//   - error: validation errors (task not found) or repository errors
//
// Errors:
//   - NotFoundError: task not found
//   - RepositoryError: database operation failed
func (s *TaskService) ListDependencies(ctx context.Context, taskKey string) ([]*models.Task, error) {
	if s.depRepo == nil {
		return nil, fmt.Errorf("dependency repository not configured")
	}

	task, err := s.repo.GetByKey(ctx, taskKey)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	// Get all outgoing "depends_on" relationships (tasks that this task depends on)
	rels, err := s.depRepo.GetOutgoing(ctx, task.ID, []string{"depends_on"})
	if err != nil {
		return nil, fmt.Errorf("failed to list dependencies for %s: %w", taskKey, err)
	}

	// Resolve each relationship to its target task
	tasks := make([]*models.Task, 0, len(rels))
	for _, rel := range rels {
		depTask, err := s.repo.GetByID(ctx, rel.ToTaskID)
		if err != nil {
			// Log warning but continue — dependency may have been deleted
			continue
		}
		tasks = append(tasks, depTask)
	}

	return tasks, nil
}

// UnlinkFile removes all outgoing typed relationships from the given task.
// This is used to unlink a task from relationships it depends on or blocks.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - taskKey: the key of the task to unlink relationships from
//   - relType: the relationship type to remove (e.g., "depends_on", "blocks")
//   - targetKey: the key of the target task to unlink from
//
// Returns:
//   - error: validation errors (task not found) or repository errors
//
// Errors:
//   - NotFoundError: task or target task not found
//   - RepositoryError: database operation failed
func (s *TaskService) UnlinkFile(ctx context.Context, taskKey, relType, targetKey string) error {
	if s.depRepo == nil {
		return fmt.Errorf("dependency repository not configured")
	}

	task, err := s.repo.GetByKey(ctx, taskKey)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	target, err := s.repo.GetByKey(ctx, targetKey)
	if err != nil {
		return fmt.Errorf("target task not found: %w", err)
	}

	if err := s.depRepo.DeleteByTasksAndType(ctx, task.ID, target.ID, relType); err != nil {
		return fmt.Errorf("failed to unlink %s from %s (%s): %w", taskKey, targetKey, relType, err)
	}

	return nil
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
	// Set blocked filter
	filters.Blocked = true
	filters.ShowAll = true // Include all blocked tasks regardless of completion

	// Get filtered tasks
	tasks, err := s.ListTasks(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get blocked tasks: %w", err)
	}

	return tasks, nil
}

// UnlinkRelationships removes relationship links between tasks.
// If targetKeys is empty, all relationships of the given type are removed.
// If targetKeys has entries, only those specific relationships are removed.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - taskKey: the source task key
//   - relType: the relationship type to unlink
//   - targetKeys: specific target task keys (empty = remove all of relType)
//
// Returns:
//   - int: number of relationships removed
//   - error: if task not found, dependency repo not configured, or database operation fails
func (s *TaskService) UnlinkRelationships(ctx context.Context, taskKey, relType string, targetKeys []string) (int, error) {
	if s.depRepo == nil {
		return 0, fmt.Errorf("dependency repository not configured")
	}

	task, err := s.repo.GetByKey(ctx, taskKey)
	if err != nil {
		return 0, fmt.Errorf("task not found: %w", err)
	}

	if len(targetKeys) == 0 {
		// Remove all relationships of this type
		rels, err := s.depRepo.GetOutgoing(ctx, task.ID, []string{relType})
		if err != nil {
			return 0, fmt.Errorf("failed to get relationships for %s: %w", taskKey, err)
		}

		count := 0
		for _, rel := range rels {
			if err := s.depRepo.Delete(ctx, rel.ID); err != nil {
				return count, fmt.Errorf("failed to delete relationship %d: %w", rel.ID, err)
			}
			count++
		}
		return count, nil
	}

	// Remove specific target relationships
	count := 0
	for _, targetKey := range targetKeys {
		target, err := s.repo.GetByKey(ctx, targetKey)
		if err != nil {
			return count, fmt.Errorf("target task not found: %w", err)
		}

		if err := s.depRepo.DeleteByTasksAndType(ctx, task.ID, target.ID, relType); err != nil {
			return count, fmt.Errorf("failed to unlink %s from %s (%s): %w", taskKey, targetKey, relType, err)
		}
		count++
	}

	return count, nil
}

// CreateTypedRelationship creates a typed relationship between two tasks.
// For depends_on and blocks relationships, circular dependency detection is performed.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - taskKey: the source task key (e.g., "E07-F01-002")
//   - targetKey: the target task key (e.g., "E07-F01-001")
//   - relType: the relationship type (depends_on, blocks, related_to, follows, spawned_from, duplicates, references)
//
// Returns:
//   - *models.Task: the resolved target task (for display purposes)
//   - error: task not found, circular dependency, or repository errors
func (s *TaskService) CreateTypedRelationship(ctx context.Context, taskKey, targetKey, relType string) (*models.Task, error) {
	if s.depRepo == nil {
		return nil, fmt.Errorf("relationship repository not configured")
	}

	task, err := s.repo.GetByKey(ctx, taskKey)
	if err != nil {
		return nil, fmt.Errorf("task %s not found: %w", taskKey, err)
	}

	targetTask, err := s.repo.GetByKey(ctx, targetKey)
	if err != nil {
		return nil, fmt.Errorf("target task %s not found: %w", targetKey, err)
	}

	// Check for cycles on directed relationship types
	if relType == "depends_on" || relType == "blocks" {
		if err := s.depRepo.DetectCycle(ctx, task.ID, targetTask.ID, relType); err != nil {
			return nil, fmt.Errorf("circular dependency detected: %w", err)
		}
	}

	rel := &models.TaskRelationship{
		FromTaskID:       task.ID,
		ToTaskID:         targetTask.ID,
		RelationshipType: models.RelationshipType(relType),
	}

	if err := s.depRepo.Create(ctx, rel); err != nil {
		return nil, fmt.Errorf("failed to create %s relationship from %s to %s: %w", relType, taskKey, targetKey, err)
	}

	return targetTask, nil
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
	tasks, err := s.repo.FindByFileChanged(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to search tasks by file %s: %w", filePath, err)
	}

	// Apply filters
	if filters.Status != "" || filters.EpicKey != "" || filters.FeatureKey != "" {
		filtered := make([]*models.Task, 0, len(tasks))
		epicKeyUpper := strings.ToUpper(filters.EpicKey)
		featureKeyUpper := strings.ToUpper(filters.FeatureKey)
		for _, t := range tasks {
			if filters.Status != "" && string(t.Status) != filters.Status {
				continue
			}
			// Filter by epic: task key contains the epic key (e.g., T-E10-F03-001 contains "E10")
			if epicKeyUpper != "" && !strings.Contains(strings.ToUpper(t.Key), epicKeyUpper) {
				continue
			}
			// Filter by feature: task key contains the feature key portion (e.g., "F03" or "E10-F03")
			if featureKeyUpper != "" && !strings.Contains(strings.ToUpper(t.Key), featureKeyUpper) {
				continue
			}
			filtered = append(filtered, t)
		}
		return filtered, nil
	}

	return tasks, nil
}

// SetRelQueryRepo sets the relationship query repository for dep/blocked-by/blocks commands.
// This is optional; commands needing relationship queries must call this before use.
func (s *TaskService) SetRelQueryRepo(relQueryRepo TaskRelationshipQueryRepository) {
	s.relQueryRepo = relQueryRepo
}

// SetWritableDocRepo sets the writable document repository for link/unlink operations.
// This is optional; commands needing document write operations must call this before use.
func (s *TaskService) SetWritableDocRepo(writableDocRepo TaskWritableDocumentRepository) {
	s.writableDocRepo = writableDocRepo
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

// GetRelQueryRepo returns the relationship query repository for use by tree traversal functions.
// Returns nil if relationship queries are not configured.
func (s *TaskService) GetRelQueryRepo() TaskRelationshipQueryRepository {
	return s.relQueryRepo
}

// GetTaskRelationships retrieves all relationships for a task, optionally filtered by type.
// Returns relationship records enriched with the related task's key, title, and status.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - taskKey: the task key to query relationships for
//   - typeFilter: optional list of relationship types to include (empty = all types)
//
// Returns:
//   - []RelationshipWithTask: relationship records with related task info
//   - error: if task not found, relQueryRepo not configured, or database operation fails
func (s *TaskService) GetTaskRelationships(ctx context.Context, taskKey string, typeFilter []string) ([]RelationshipWithTask, error) {
	if s.relQueryRepo == nil {
		return nil, fmt.Errorf("relationship query repository not configured")
	}

	task, err := s.repo.GetByKey(ctx, taskKey)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	allRels, err := s.relQueryRepo.GetByTaskID(ctx, task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get relationships for %s: %w", taskKey, err)
	}

	var result []RelationshipWithTask
	for _, rel := range allRels {
		// Filter by type if specified
		if len(typeFilter) > 0 {
			found := false
			for _, ft := range typeFilter {
				if string(rel.RelationshipType) == ft {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		direction := "outgoing"
		relatedTaskID := rel.ToTaskID
		if rel.FromTaskID != task.ID {
			direction = "incoming"
			relatedTaskID = rel.FromTaskID
		}

		relatedTask, err := s.repo.GetByID(ctx, relatedTaskID)
		if err != nil {
			continue // Skip if related task not found
		}

		result = append(result, RelationshipWithTask{
			RelationshipType: string(rel.RelationshipType),
			Direction:        direction,
			TaskKey:          relatedTask.Key,
			TaskTitle:        relatedTask.Title,
			TaskStatus:       string(relatedTask.Status),
		})
	}

	return result, nil
}

// GetTaskBlockedBy retrieves tasks that this task depends on (incoming dependencies).
// These are the tasks that must be completed before this task can proceed.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - taskKey: the task key to query dependencies for
//
// Returns:
//   - []RelationshipWithTask: tasks this task depends on
//   - error: if task not found, relQueryRepo not configured, or database operation fails
func (s *TaskService) GetTaskBlockedBy(ctx context.Context, taskKey string) ([]RelationshipWithTask, error) {
	if s.relQueryRepo == nil {
		return nil, fmt.Errorf("relationship query repository not configured")
	}

	task, err := s.repo.GetByKey(ctx, taskKey)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	deps, err := s.relQueryRepo.GetOutgoing(ctx, task.ID, []string{"depends_on"})
	if err != nil {
		return nil, fmt.Errorf("failed to get dependencies for %s: %w", taskKey, err)
	}

	var result []RelationshipWithTask
	for _, rel := range deps {
		depTask, err := s.repo.GetByID(ctx, rel.ToTaskID)
		if err != nil {
			continue
		}
		result = append(result, RelationshipWithTask{
			RelationshipType: "depends_on",
			Direction:        "outgoing",
			TaskKey:          depTask.Key,
			TaskTitle:        depTask.Title,
			TaskStatus:       string(depTask.Status),
		})
	}

	return result, nil
}

// GetTaskBlocks retrieves tasks that depend on this task completing.
// Includes both implicit (depends_on this task) and explicit (blocks relationship) dependents.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - taskKey: the task key to query
//
// Returns:
//   - []RelationshipWithTask: tasks blocked by this task
//   - error: if task not found, relQueryRepo not configured, or database operation fails
func (s *TaskService) GetTaskBlocks(ctx context.Context, taskKey string) ([]RelationshipWithTask, error) {
	if s.relQueryRepo == nil {
		return nil, fmt.Errorf("relationship query repository not configured")
	}

	task, err := s.repo.GetByKey(ctx, taskKey)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	// Get incoming depends_on (others depend on this task)
	incoming, err := s.relQueryRepo.GetIncoming(ctx, task.ID, []string{"depends_on"})
	if err != nil {
		return nil, fmt.Errorf("failed to get incoming dependencies for %s: %w", taskKey, err)
	}

	// Get outgoing blocks relationships (this task explicitly blocks others)
	outgoing, err := s.relQueryRepo.GetOutgoing(ctx, task.ID, []string{"blocks"})
	if err != nil {
		return nil, fmt.Errorf("failed to get explicit blocks for %s: %w", taskKey, err)
	}

	allBlocked := append(incoming, outgoing...)

	var result []RelationshipWithTask
	for _, rel := range allBlocked {
		var blockedTaskID int64
		var direction string
		if rel.FromTaskID != task.ID {
			// incoming depends_on: the blocking task is rel.FromTaskID
			blockedTaskID = rel.FromTaskID
			direction = "incoming"
		} else {
			// outgoing blocks: the blocked task is rel.ToTaskID
			blockedTaskID = rel.ToTaskID
			direction = "outgoing"
		}

		blockedTask, err := s.repo.GetByID(ctx, blockedTaskID)
		if err != nil {
			continue
		}
		result = append(result, RelationshipWithTask{
			RelationshipType: string(rel.RelationshipType),
			Direction:        direction,
			TaskKey:          blockedTask.Key,
			TaskTitle:        blockedTask.Title,
			TaskStatus:       string(blockedTask.Status),
		})
	}

	return result, nil
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
	if s.writableDocRepo == nil {
		return nil, fmt.Errorf("writable document repository not configured")
	}

	task, err := s.repo.GetByKey(ctx, taskKey)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	doc, err := s.writableDocRepo.CreateOrGet(ctx, title, path)
	if err != nil {
		return nil, fmt.Errorf("failed to create or get document: %w", err)
	}

	if err := s.writableDocRepo.LinkToTask(ctx, task.ID, doc.ID); err != nil {
		return nil, fmt.Errorf("failed to link document to task %s: %w", taskKey, err)
	}

	return doc, nil
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
	if s.writableDocRepo == nil {
		return fmt.Errorf("writable document repository not configured")
	}

	task, err := s.repo.GetByKey(ctx, taskKey)
	if err != nil {
		// Task doesn't exist - idempotent, treat as success
		return nil
	}

	doc, err := s.writableDocRepo.GetByTitle(ctx, title)
	if err != nil {
		// Document doesn't exist - idempotent, treat as success
		return nil
	}

	if err := s.writableDocRepo.UnlinkFromTask(ctx, task.ID, doc.ID); err != nil {
		return fmt.Errorf("failed to unlink document from task %s: %w", taskKey, err)
	}

	return nil
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
	if s.sessionRepo == nil {
		return nil, fmt.Errorf("work session repository not configured")
	}

	task, err := s.repo.GetByKey(ctx, taskKey)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	sessions, err := s.sessionRepo.GetByTaskID(ctx, task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get work sessions for %s: %w", taskKey, err)
	}

	stats, err := s.sessionRepo.GetSessionStatsByTaskID(ctx, task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session stats for %s: %w", taskKey, err)
	}

	return &TaskWorkSessions{
		TaskKey:   task.Key,
		TaskTitle: task.Title,
		Sessions:  sessions,
		Stats:     stats,
	}, nil
}

// SetHistoryRepo sets the task history repository for this service.
// This is used by CLI global accessors to wire optional history repository
// after initial construction via NewTaskService.
func (s *TaskService) SetHistoryRepo(repo TaskHistoryRepository) {
	s.historyRepo = repo
}

// SetSessionRepo sets the work session repository for analytics.
// This is used by CLI global accessors to wire optional session repository
// after initial construction via NewTaskService.
func (s *TaskService) SetSessionRepo(repo WorkSessionRepository) {
	s.sessionRepo = repo
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
// status change. Non-fatal: silently ignores errors and nil featureService.
// Only recalculates when featureID is non-zero (i.e., task is associated with a feature).
func (s *TaskService) recalculateFeatureProgress(ctx context.Context, featureID int64) {
	if s.featureService == nil || featureID == 0 {
		return
	}
	_ = s.featureService.RecalculateAndSetProgress(ctx, featureID)
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
	if s.historyRepo == nil {
		return nil, fmt.Errorf("history repository not configured")
	}

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
func (s *TaskService) ListHistory(ctx context.Context, filters HistoryFilters) ([]*models.TaskHistory, error) {
	if s.historyRepo == nil {
		return nil, fmt.Errorf("history repository not configured")
	}

	histories, err := s.historyRepo.ListWithFilters(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list task history: %w", err)
	}

	return histories, nil
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
	if s.sessionRepo == nil {
		return nil, fmt.Errorf("work session repository not configured")
	}

	// Convert AgentType string to *string for repo methods (nil means no filter)
	var agentTypePtr *string
	if input.AgentType != "" {
		agentTypePtr = &input.AgentType
	}

	if input.FeatureKey != "" {
		if s.featureRepo == nil {
			return nil, fmt.Errorf("feature repository not configured")
		}
		// Get feature to resolve ID
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
		// Get epic to resolve ID
		epic, err := s.epicRepo.GetByKey(ctx, input.EpicKey)
		if err != nil {
			return nil, fmt.Errorf("epic not found: %s: %w", input.EpicKey, err)
		}
		return s.sessionRepo.GetSessionAnalyticsByEpic(ctx, epic.ID, agentTypePtr)
	}

	return nil, fmt.Errorf("either EpicKey or FeatureKey must be specified")
}
