package services

import (
	"context"
	"fmt"
	"strings"

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
func NewTaskServiceWithRelationships(repo TaskRepository, workflowSvc *workflow.Service, creatorSvc *taskcreation.Creator, noteRepo TaskNoteRepository, docRepo config.DocumentRepository, relRepo config.TaskRelationshipRepository) *TaskService {
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

	// Generate task key - use simple format that will be validated by the model
	// Repository can override if needed for actual database sequence
	taskKey := fmt.Sprintf("T-%s-%s-001", input.EpicKey, input.FeatureKey)

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
//   - filters: filter criteria (epic, feature, status, agent, show_all, etc.)
//
// Returns:
//   - []*models.Task: list of matching tasks (empty if none found)
//   - error: repository errors
//
// Errors:
//   - RepositoryError: database query failed
func (s *TaskService) ListTasks(ctx context.Context, filters TaskFilters) ([]*models.Task, error) {
	// Get all tasks
	tasks, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
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
