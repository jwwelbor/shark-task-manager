package services

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/taskcreation"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// TaskRelationshipRepository defines the interface for accessing task relationships.
type TaskRelationshipRepository = config.TaskRelationshipRepository

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
}

// TaskNoteRepository defines the note repository interface for rejection notes.
type TaskNoteRepository interface {
	CreateRejectionNote(ctx context.Context, entityType models.EntityType, entityID int64,
		historyID int64, fromStatus, toStatus, reason, rejectedBy string, documentPath *string) (*models.EntityNote, error)
}

// TaskService provides business logic for task operations.
// It orchestrates task lifecycle, status transitions, dependency validation,
// and coordinates with workflow and taskcreation services.
type TaskService struct {
	repo        TaskRepository
	workflowSvc *workflow.Service
	creatorSvc  *taskcreation.Creator
	noteRepo    TaskNoteRepository
	docRepo     config.DocumentRepository
	relRepo     config.TaskRelationshipRepository
}

// NewTaskService creates a new TaskService with the required dependencies.
// The workflow service is automatically scoped to the task level.
// creatorSvc, noteRepo, docRepo, and relRepo can be nil for graceful degradation.
//
// Parameters:
//   - repo: task repository for data access (required)
//   - workflowSvc: workflow service for status validation and transitions (required)
//   - creatorSvc: task creation service for key generation and file creation (optional)
//   - noteRepo: note repository for rejection tracking (optional)
//
// Returns:
//   - *TaskService: configured task service instance
func NewTaskService(repo TaskRepository, workflowSvc *workflow.Service, creatorSvc *taskcreation.Creator, noteRepo TaskNoteRepository) *TaskService {
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
//   - repo: task repository for data access (required)
//   - workflowSvc: workflow service for status validation and transitions (required)
//   - creatorSvc: task creation service for key generation and file creation (optional)
//   - noteRepo: note repository for rejection tracking (optional)
//   - docRepo: document repository for related documents (optional)
//   - relRepo: relationship repository for related tasks (optional)
//
// Returns:
//   - *TaskService: configured task service instance with relationship support
func NewTaskServiceWithRelationships(repo TaskRepository, workflowSvc *workflow.Service, creatorSvc *taskcreation.Creator, noteRepo TaskNoteRepository, docRepo config.DocumentRepository, relRepo config.TaskRelationshipRepository) *TaskService {
	return &TaskService{
		repo:        repo,
		workflowSvc: workflowSvc.ForLevel(workflow.LevelTask),
		creatorSvc:  creatorSvc,
		noteRepo:    noteRepo,
		docRepo:     docRepo,
		relRepo:     relRepo,
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
	// TODO: Implementation in F02-F07
	return nil, fmt.Errorf("not implemented")
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
	// TODO: Implementation in F02-F07
	return nil, fmt.Errorf("not implemented")
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
	// TODO: Implementation in F02-F07
	return nil, fmt.Errorf("not implemented")
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
	// TODO: Implementation in F02-F07
	return fmt.Errorf("not implemented")
}

// ListTasks retrieves tasks matching the given filters.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - filters: filter criteria (epic, feature, status, agent, show_all, etc.)
//
// Returns:
//   - []*models.Task: list of matching tasks (empty if none found)
//   - error: repository errors
//
// Errors:
//   - RepositoryError: database query failed
func (s *TaskService) ListTasks(ctx context.Context, filters TaskFilters) ([]*models.Task, error) {
	// TODO: Implementation in F02-F07
	return nil, fmt.Errorf("not implemented")
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
	// TODO: Implementation in F02-F07
	return nil, fmt.Errorf("not implemented")
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
	// TODO: Implementation in F02-F07
	return nil, fmt.Errorf("not implemented")
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
	// TODO: Implementation in F02-F07
	return nil, fmt.Errorf("not implemented")
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
	// TODO: Implementation in F02-F07
	return nil, fmt.Errorf("not implemented")
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
	// TODO: Implementation in F02-F07
	return nil, fmt.Errorf("not implemented")
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
	// TODO: Implementation in F02-F07
	return nil, nil, fmt.Errorf("not implemented")
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
	// TODO: Implementation in F02-F07
	return nil, fmt.Errorf("not implemented")
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
	// TODO: Implementation in F02-F07
	return nil, fmt.Errorf("not implemented")
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
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: task key to validate
//   - targetStatus: status the task wants to transition to
//
// Returns:
//   - error: DependencyError if dependencies not met, or repository errors
//
// Errors:
//   - DependencyError: one or more dependencies are not satisfied
//   - NotFoundError: task not found
//   - RepositoryError: database query failed
func (s *TaskService) ValidateDependencies(ctx context.Context, key string, targetStatus string) error {
	// TODO: Implementation in F02-F07
	return fmt.Errorf("not implemented")
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
	// TODO: Implementation in F02-F07
	return nil, fmt.Errorf("not implemented")
}

// resolveAction looks up the orchestrator action for a given status.
// Returns nil if no action is defined or if document/relationship repositories are not available.
//
//nolint:unused // Will be used when TransitionStatus and GetNextStatus are implemented
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
