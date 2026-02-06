package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/dependency"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// ValidateTaskDependencies validates that adding a task with given dependencies
// would not create circular dependencies. This should be called before creating
// or updating a task with dependencies.
func (r *TaskRepository) ValidateTaskDependencies(ctx context.Context, task *models.Task) error {
	if task.DependsOn == nil || *task.DependsOn == "" || *task.DependsOn == "[]" {
		return nil
	}

	// Parse dependencies JSON
	var dependencies []string
	if err := json.Unmarshal([]byte(*task.DependsOn), &dependencies); err != nil {
		return fmt.Errorf("invalid dependencies JSON: %w", err)
	}

	// Build dependency graph from all tasks
	detector := dependency.NewDetector()

	// Get all tasks in the feature
	allTasks, err := r.ListByFeature(ctx, task.FeatureID)
	if err != nil {
		return fmt.Errorf("failed to list tasks for validation: %w", err)
	}

	// Build graph from existing tasks
	for _, t := range allTasks {
		// Skip the task we're validating if it already exists
		if t.Key == task.Key {
			continue
		}

		if t.DependsOn != nil && *t.DependsOn != "" && *t.DependsOn != "[]" {
			var deps []string
			if err := json.Unmarshal([]byte(*t.DependsOn), &deps); err != nil {
				continue
			}
			for _, dep := range deps {
				detector.AddDependency(t.Key, dep)
			}
		}
	}

	// Validate that all dependencies exist
	existingKeys := make(map[string]bool)
	for _, t := range allTasks {
		existingKeys[t.Key] = true
	}

	for _, dep := range dependencies {
		if dep == task.Key {
			return fmt.Errorf("task cannot depend on itself: %s", task.Key)
		}
		if !existingKeys[dep] {
			return fmt.Errorf("dependency does not exist: %s", dep)
		}
	}

	// Validate each new dependency for circular references
	for _, dep := range dependencies {
		if err := detector.ValidateDependency(ctx, task.Key, dep); err != nil {
			return err
		}
	}

	return nil
}

// BuildDependencyGraphForFeature builds a dependency graph for all tasks in a feature.
// This can be used to analyze dependencies, detect cycles, or find dependency chains.
func (r *TaskRepository) BuildDependencyGraphForFeature(ctx context.Context, featureID int64) (*dependency.Detector, error) {
	detector := dependency.NewDetector()

	tasks, err := r.ListByFeature(ctx, featureID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	for _, task := range tasks {
		if task.DependsOn != nil && *task.DependsOn != "" && *task.DependsOn != "[]" {
			var deps []string
			if err := json.Unmarshal([]byte(*task.DependsOn), &deps); err != nil {
				continue
			}
			for _, dep := range deps {
				detector.AddDependency(task.Key, dep)
			}
		}
	}

	return detector, nil
}

// GetTaskDependents returns all tasks that depend on the given task.
// This is useful for cascading operations like auto-blocking.
func (r *TaskRepository) GetTaskDependents(ctx context.Context, taskKey string) ([]*models.Task, error) {
	// First get the task to find its feature
	task, err := r.GetByKey(ctx, taskKey)
	if err != nil {
		return nil, err
	}

	// Get all tasks in the feature
	allTasks, err := r.ListByFeature(ctx, task.FeatureID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	// Find tasks that depend on this task
	dependents := []*models.Task{}
	for _, t := range allTasks {
		if t.DependsOn == nil || *t.DependsOn == "" || *t.DependsOn == "[]" {
			continue
		}

		var deps []string
		if err := json.Unmarshal([]byte(*t.DependsOn), &deps); err != nil {
			continue
		}

		for _, dep := range deps {
			if dep == taskKey {
				dependents = append(dependents, t)
				break
			}
		}
	}

	return dependents, nil
}

// ReopenTaskWithAutoBlock reopens a task and automatically blocks all dependent tasks.
// This is the recommended method to use when reopening tasks with dependents.
func (r *TaskRepository) ReopenTaskWithAutoBlock(ctx context.Context, taskID int64, agent *string, notes *string) error {
	// Start transaction for atomic operations
	tx, err := r.db.BeginTxContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Get the task being reopened
	var taskKey string
	err = tx.QueryRowContext(ctx, "SELECT key FROM tasks WHERE id = ?", taskID).Scan(&taskKey)
	if err != nil {
		return fmt.Errorf("failed to get task key: %w", err)
	}

	// Reopen the task (using existing ReopenTaskForced since we're in a transaction)
	err = r.reopenTaskInTx(ctx, tx, taskID, agent, notes, false)
	if err != nil {
		return fmt.Errorf("failed to reopen task: %w", err)
	}

	// Get all dependents (need to query before transaction commits)
	dependents, err := r.getTaskDependentsInTx(ctx, tx, taskKey)
	if err != nil {
		return fmt.Errorf("failed to get dependents: %w", err)
	}

	// Block all non-completed dependents and their transitive dependents
	blockedTasks := make(map[string]bool)
	for _, dependent := range dependents {
		if err := r.blockTaskAndDependentsInTx(ctx, tx, dependent, taskKey, blockedTasks); err != nil {
			return fmt.Errorf("failed to block dependent %s: %w", dependent.Key, err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// reopenTaskInTx reopens a task within a transaction
func (r *TaskRepository) reopenTaskInTx(ctx context.Context, tx *sql.Tx, taskID int64, agent *string, notes *string, force bool) error {
	// Get current task state
	var currentStatus string
	err := tx.QueryRowContext(ctx, "SELECT status FROM tasks WHERE id = ?", taskID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		return fmt.Errorf("task not found with id %d", taskID)
	}
	if err != nil {
		return fmt.Errorf("failed to get current task status: %w", err)
	}

	// Validate transition if not forcing
	currentTaskStatus := models.TaskStatus(currentStatus)
	if !force {
		// Only allow reopening from ready_for_review
		if currentTaskStatus != models.TaskStatusReadyForReview {
			return fmt.Errorf("invalid status transition from %s to in_progress", currentStatus)
		}
	}

	// Update status and clear completed_at
	query := `UPDATE tasks SET status = ?, completed_at = NULL WHERE id = ?`
	_, err = tx.ExecContext(ctx, query, models.TaskStatusInProgress, taskID)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Create history record
	historyQuery := `INSERT INTO task_history (task_id, old_status, new_status, agent, notes, forced) VALUES (?, ?, ?, ?, ?, ?)`
	_, err = tx.ExecContext(ctx, historyQuery, taskID, currentStatus, models.TaskStatusInProgress, agent, notes, force)
	if err != nil {
		return fmt.Errorf("failed to create history record: %w", err)
	}

	return nil
}

// getTaskDependentsInTx returns all tasks that depend on the given task within a transaction
func (r *TaskRepository) getTaskDependentsInTx(ctx context.Context, tx *sql.Tx, taskKey string) ([]*models.Task, error) {
	// Get the task's feature_id
	var featureID int64
	err := tx.QueryRowContext(ctx, "SELECT feature_id FROM tasks WHERE key = ?", taskKey).Scan(&featureID)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature_id: %w", err)
	}

	// Get all tasks in the feature
	query := `
		SELECT id, feature_id, key, title, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data
		FROM tasks
		WHERE feature_id = ?
	`

	rows, err := tx.QueryContext(ctx, query, featureID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer rows.Close()

	var allTasks []*models.Task
	for rows.Next() {
		task := &models.Task{}
		err := rows.Scan(
			&task.ID, &task.FeatureID, &task.Key, &task.Title, &task.Description,
			&task.Status, &task.AgentType, &task.Priority, &task.DependsOn,
			&task.AssignedAgent, &task.FilePath, &task.BlockedReason, &task.ExecutionOrder,
			&task.CreatedAt, &task.StartedAt, &task.CompletedAt, &task.BlockedAt, &task.UpdatedAt,
			&task.CompletedBy, &task.CompletionNotes, &task.FilesChanged, &task.TestsPassed,
			&task.VerificationStatus, &task.TimeSpentMinutes, &task.ContextData,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		allTasks = append(allTasks, task)
	}

	// Find tasks that depend on taskKey
	dependents := []*models.Task{}
	for _, t := range allTasks {
		if t.DependsOn == nil || *t.DependsOn == "" || *t.DependsOn == "[]" {
			continue
		}

		var deps []string
		if err := json.Unmarshal([]byte(*t.DependsOn), &deps); err != nil {
			continue
		}

		for _, dep := range deps {
			if dep == taskKey {
				dependents = append(dependents, t)
				break
			}
		}
	}

	return dependents, nil
}

// blockTaskAndDependentsInTx recursively blocks a task and all its dependents within a transaction
func (r *TaskRepository) blockTaskAndDependentsInTx(ctx context.Context, tx *sql.Tx, task *models.Task, reopenedTaskKey string, blockedTasks map[string]bool) error {
	// Skip if already processed
	if blockedTasks[task.Key] {
		return nil
	}

	// Skip completed and archived tasks
	if task.Status == models.TaskStatusCompleted || task.Status == models.TaskStatusArchived {
		return nil
	}

	// Block this task
	reason := fmt.Sprintf("Prerequisite task %s was reopened", reopenedTaskKey)
	query := `UPDATE tasks SET status = ?, blocked_at = CURRENT_TIMESTAMP, blocked_reason = ? WHERE id = ?`
	_, err := tx.ExecContext(ctx, query, models.TaskStatusBlocked, reason, task.ID)
	if err != nil {
		return fmt.Errorf("failed to block task: %w", err)
	}

	// Create history record
	historyQuery := `INSERT INTO task_history (task_id, old_status, new_status, agent, notes, forced) VALUES (?, ?, ?, ?, ?, ?)`
	_, err = tx.ExecContext(ctx, historyQuery, task.ID, task.Status, models.TaskStatusBlocked, nil, reason, false)
	if err != nil {
		return fmt.Errorf("failed to create history record: %w", err)
	}

	// Mark as blocked
	blockedTasks[task.Key] = true

	// Find and block all dependents of this task
	dependents, err := r.getTaskDependentsInTx(ctx, tx, task.Key)
	if err != nil {
		return fmt.Errorf("failed to get dependents of %s: %w", task.Key, err)
	}

	for _, dependent := range dependents {
		if err := r.blockTaskAndDependentsInTx(ctx, tx, dependent, reopenedTaskKey, blockedTasks); err != nil {
			return err
		}
	}

	return nil
}

// AutoUnblockDependents checks all tasks that depend on the completed task and
// unblocks those whose dependencies are all satisfied. Only tasks blocked with
// a dependency-pattern reason (set by ReopenTaskWithAutoBlock) are eligible.
// Returns the keys of tasks that were auto-unblocked.
func (r *TaskRepository) AutoUnblockDependents(ctx context.Context, tx *sql.Tx, completedTaskKey string) ([]string, error) {
	// Get dependents of the completed task
	dependents, err := r.getTaskDependentsInTx(ctx, tx, completedTaskKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependents: %w", err)
	}

	var unblocked []string
	for _, dependent := range dependents {
		// Only consider blocked tasks
		if dependent.Status != models.TaskStatusBlocked {
			continue
		}

		// Only auto-unblock tasks blocked due to dependencies (not manual blocks)
		if !isDependencyBlocked(dependent) {
			continue
		}

		// Check if ALL dependencies of this task are now completed/archived
		allSatisfied, err := r.allDependenciesSatisfiedInTx(ctx, tx, dependent)
		if err != nil {
			return nil, fmt.Errorf("failed to check dependencies for %s: %w", dependent.Key, err)
		}

		if !allSatisfied {
			continue
		}

		// Unblock the task: set status to todo, clear blocked_at and blocked_reason
		if err := r.unblockTaskInTx(ctx, tx, dependent); err != nil {
			return nil, fmt.Errorf("failed to unblock %s: %w", dependent.Key, err)
		}

		unblocked = append(unblocked, dependent.Key)
	}

	return unblocked, nil
}

// isDependencyBlocked returns true if the task was blocked due to a dependency
// (by ReopenTaskWithAutoBlock), not a manual block.
func isDependencyBlocked(task *models.Task) bool {
	if task.BlockedReason == nil || *task.BlockedReason == "" {
		return false
	}
	reason := *task.BlockedReason
	return strings.HasPrefix(reason, "Prerequisite task ") ||
		strings.HasPrefix(reason, "Auto-blocked:")
}

// allDependenciesSatisfiedInTx checks whether every task in the dependent's
// depends_on list is completed or archived.
func (r *TaskRepository) allDependenciesSatisfiedInTx(ctx context.Context, tx *sql.Tx, task *models.Task) (bool, error) {
	if task.DependsOn == nil || *task.DependsOn == "" || *task.DependsOn == "[]" {
		return true, nil
	}

	var deps []string
	if err := json.Unmarshal([]byte(*task.DependsOn), &deps); err != nil {
		return false, fmt.Errorf("invalid depends_on JSON for %s: %w", task.Key, err)
	}

	for _, depKey := range deps {
		var status string
		err := tx.QueryRowContext(ctx, "SELECT status FROM tasks WHERE key = ?", depKey).Scan(&status)
		if err == sql.ErrNoRows {
			// Dependency task doesn't exist - treat as unsatisfied
			return false, nil
		}
		if err != nil {
			return false, fmt.Errorf("failed to check status of dependency %s: %w", depKey, err)
		}

		depStatus := models.TaskStatus(status)
		if depStatus != models.TaskStatusCompleted && depStatus != models.TaskStatusArchived {
			return false, nil
		}
	}

	return true, nil
}

// unblockTaskInTx transitions a task from blocked to todo within a transaction,
// clearing blocked_at and blocked_reason, and recording history.
func (r *TaskRepository) unblockTaskInTx(ctx context.Context, tx *sql.Tx, task *models.Task) error {
	query := `UPDATE tasks SET status = ?, blocked_at = NULL, blocked_reason = NULL WHERE id = ?`
	_, err := tx.ExecContext(ctx, query, models.TaskStatusTodo, task.ID)
	if err != nil {
		return fmt.Errorf("failed to unblock task: %w", err)
	}

	notes := "Auto-unblocked: all dependencies satisfied"
	historyQuery := `INSERT INTO task_history (task_id, old_status, new_status, agent, notes, forced) VALUES (?, ?, ?, ?, ?, ?)`
	_, err = tx.ExecContext(ctx, historyQuery, task.ID, task.Status, models.TaskStatusTodo, nil, notes, false)
	if err != nil {
		return fmt.Errorf("failed to create history record: %w", err)
	}

	return nil
}
