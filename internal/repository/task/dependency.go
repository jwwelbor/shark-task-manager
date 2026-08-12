package task

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/dependency"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// Constants for dependency block reason prefixes.
// These are used to identify tasks blocked due to dependency relationships
// (as opposed to manual blocks) for auto-unblock eligibility.
const (
	DependencyReopenedBlockReasonPrefix = "Prerequisite task "
	AutoBlockedReasonPrefix             = "Auto-blocked:"
)

// ValidateTaskDependencies validates that adding a task with given dependencies
// would not create circular dependencies. This should be called before creating
// or updating a task with dependencies.
//
// Scope contract: dependency existence and cycle detection are both scoped to
// task.FeatureID — the depends_on JSON field only supports same-feature
// dependencies. This is intentional: cross-feature dependencies are supported
// via the entity_relationships table instead (see EntityRelationshipService,
// `shark task link --depends-on`), which is unaffected by this method.
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
			return r.dependencyNotFoundError(ctx, dep)
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

// dependencyNotFoundError builds the error returned when a dependency key is
// absent from the task's own feature. A key that exists in another feature
// gets a distinct, accurate message instead of the misleading "does not
// exist" — the depends_on JSON field simply doesn't support cross-feature
// dependencies (see ValidateTaskDependencies doc comment), so the message
// points the caller at the supported alternative.
func (r *TaskRepository) dependencyNotFoundError(ctx context.Context, dep string) error {
	_, err := r.GetByKey(ctx, dep)
	if err == nil {
		return fmt.Errorf("dependency %s exists in a different feature: depends_on only supports same-feature dependencies; use 'shark task link --depends-on' to create a cross-feature dependency", dep)
	}
	if strings.Contains(err.Error(), "not found") {
		return fmt.Errorf("dependency does not exist: %s", dep)
	}
	// GetByKey failed for a reason other than not-found (e.g. a transient DB
	// error) — propagate it instead of claiming the dependency doesn't exist.
	return fmt.Errorf("dependency does not exist: %s (existence check failed: %w)", dep, err)
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

// GetTaskDependencies returns all tasks that the given task depends on.
// This is the prerequisite-facing companion to GetTaskDependents and is used
// for dispatch validation before work starts.
func (r *TaskRepository) GetTaskDependencies(ctx context.Context, taskKey string) ([]*models.Task, error) {
	task, err := r.GetByKey(ctx, taskKey)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	dependencies := []*models.Task{}

	if task.DependsOn != nil && *task.DependsOn != "" && *task.DependsOn != "[]" {
		var deps []string
		if err := json.Unmarshal([]byte(*task.DependsOn), &deps); err != nil {
			return nil, fmt.Errorf("invalid depends_on JSON for %s: %w", task.Key, err)
		}

		for _, depKey := range deps {
			if seen[depKey] {
				continue
			}
			dep, err := r.GetByKey(ctx, depKey)
			if err != nil {
				return nil, fmt.Errorf("failed to get dependency %s for %s: %w", depKey, task.Key, err)
			}
			dependencies = append(dependencies, dep)
			seen[dep.Key] = true
		}
	}

	relQuery := `
		SELECT t.id, t.feature_id, t.key, t.title, t.description, t.status
		FROM entity_relationships er
		JOIN tasks t ON t.id = er.to_entity_id
		WHERE er.from_entity_type = 'task' AND er.from_entity_id = ?
		  AND er.to_entity_type = 'task'
		  AND er.relationship_type = 'depends_on'
	`
	rows, err := r.db.QueryContext(ctx, relQuery, task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to query entity_relationships for dependencies: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		dep := &models.Task{}
		if err := rows.Scan(&dep.ID, &dep.FeatureID, &dep.Key, &dep.Title, &dep.Description, &dep.Status); err != nil {
			return nil, fmt.Errorf("failed to scan relationship dependency: %w", err)
		}
		if seen[dep.Key] {
			continue
		}
		dependencies = append(dependencies, dep)
		seen[dep.Key] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating relationship dependencies for %s: %w", task.Key, err)
	}

	return dependencies, nil
}

// ReopenTaskWithAutoBlock reopens a task and automatically blocks all dependent tasks.
// This is the recommended method to use when reopening tasks with dependents.
func (r *TaskRepository) ReopenTaskWithAutoBlock(ctx context.Context, taskID int64, agent *string, notes *string) error {
	return r.ReopenTaskWithAutoBlockWithPolicy(ctx, taskID, agent, notes, models.TaskDependencyStatusPolicy{})
}

// ReopenTaskWithAutoBlockWithPolicy applies caller-resolved workflow status
// semantics while retaining the legacy convenience API above.
func (r *TaskRepository) ReopenTaskWithAutoBlockWithPolicy(ctx context.Context, taskID int64, agent *string, notes *string, policy models.TaskDependencyStatusPolicy) error {
	// Start transaction for atomic operations
	tx, err := r.db.BeginTxContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := r.ReopenTaskWithAutoBlockWithPolicyWithTx(ctx, tx, taskID, agent, notes, policy); err != nil {
		return err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ReopenTaskWithAutoBlockWithTx reopens a task and blocks all dependents within a
// caller-provided transaction. This allows the service layer to own the transaction
// boundary when this operation must be composed with other atomic operations.
func (r *TaskRepository) ReopenTaskWithAutoBlockWithTx(ctx context.Context, tx *sql.Tx, taskID int64, agent *string, notes *string) error {
	return r.ReopenTaskWithAutoBlockWithPolicyWithTx(ctx, tx, taskID, agent, notes, models.TaskDependencyStatusPolicy{})
}

// ReopenTaskWithAutoBlockWithPolicyWithTx is the transaction-aware form of
// ReopenTaskWithAutoBlockWithPolicy.
func (r *TaskRepository) ReopenTaskWithAutoBlockWithPolicyWithTx(ctx context.Context, tx *sql.Tx, taskID int64, agent *string, notes *string, policy models.TaskDependencyStatusPolicy) error {
	// Get the task being reopened
	var taskKey string
	err := tx.QueryRowContext(ctx, "SELECT key FROM tasks WHERE id = ?", taskID).Scan(&taskKey)
	if err != nil {
		return fmt.Errorf("failed to get task key: %w", err)
	}

	// Reopen the task
	if err := r.reopenTaskInTx(ctx, tx, taskID, agent, notes, policy.ReopenStatus); err != nil {
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
		if err := r.blockTaskAndDependentsInTx(ctx, tx, dependent, taskKey, blockedTasks, policy); err != nil {
			return fmt.Errorf("failed to block dependent %s: %w", dependent.Key, err)
		}
	}

	return nil
}

// reopenTaskInTx reopens a task within a transaction.
// NOTE: Workflow validation is handled by the service layer. This method performs
// the raw status update without checking the current status.
func (r *TaskRepository) reopenTaskInTx(ctx context.Context, tx *sql.Tx, taskID int64, agent *string, notes *string, reopenStatus models.TaskStatus) error {
	if reopenStatus == "" {
		reopenStatus = defaultTaskReopenStatus
	}
	// Get current task state for history record
	var currentStatus string
	err := tx.QueryRowContext(ctx, "SELECT status FROM tasks WHERE id = ?", taskID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		return fmt.Errorf("task not found with id %d", taskID)
	}
	if err != nil {
		return fmt.Errorf("failed to get current task status: %w", err)
	}

	// Update status and clear completed_at
	query := `UPDATE tasks SET status = ?, completed_at = NULL WHERE id = ?`
	_, err = tx.ExecContext(ctx, query, reopenStatus, taskID)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Create history record
	historyQuery := `INSERT INTO task_history (task_id, old_status, new_status, agent, notes, forced) VALUES (?, ?, ?, ?, ?, ?)`
	_, err = tx.ExecContext(ctx, historyQuery, taskID, currentStatus, reopenStatus, agent, notes, false)
	if err != nil {
		return fmt.Errorf("failed to create history record: %w", err)
	}

	return nil
}

// getTaskDependentsInTx returns all tasks that depend on the given task within a transaction.
// It checks both the legacy depends_on JSON field and the entity_relationships table.
func (r *TaskRepository) getTaskDependentsInTx(ctx context.Context, tx *sql.Tx, taskKey string) ([]*models.Task, error) {
	// Get the task's id and feature_id
	var taskID, featureID int64
	err := tx.QueryRowContext(ctx, "SELECT id, feature_id FROM tasks WHERE key = ?", taskKey).Scan(&taskID, &featureID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task info: %w", err)
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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tasks for dependents of %s: %w", taskKey, err)
	}

	// Build a map of task ID -> task for quick lookup
	taskByID := make(map[int64]*models.Task)
	for _, t := range allTasks {
		taskByID[t.ID] = t
	}

	// Track dependent keys to avoid duplicates
	seen := make(map[string]bool)
	dependents := []*models.Task{}

	// 1. Check legacy depends_on JSON field
	for _, t := range allTasks {
		if t.DependsOn == nil || *t.DependsOn == "" || *t.DependsOn == "[]" {
			continue
		}

		var deps []string
		if err := json.Unmarshal([]byte(*t.DependsOn), &deps); err != nil {
			continue
		}

		for _, dep := range deps {
			if dep == taskKey && !seen[t.Key] {
				dependents = append(dependents, t)
				seen[t.Key] = true
				break
			}
		}
	}

	// 2. Check entity_relationships table for incoming depends_on relationships
	// (tasks where from_task depends_on to_task=completedTask)
	relQuery := `
		SELECT from_entity_id FROM entity_relationships
		WHERE to_entity_type = 'task' AND to_entity_id = ?
		  AND from_entity_type = 'task'
		  AND relationship_type = 'depends_on'
	`
	relRows, err := tx.QueryContext(ctx, relQuery, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to query entity_relationships for dependents: %w", err)
	}
	defer relRows.Close()

	for relRows.Next() {
		var fromTaskID int64
		if err := relRows.Scan(&fromTaskID); err != nil {
			return nil, fmt.Errorf("failed to scan relationship: %w", err)
		}

		if t, ok := taskByID[fromTaskID]; ok && !seen[t.Key] {
			dependents = append(dependents, t)
			seen[t.Key] = true
		}
	}
	if err := relRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating relationship dependents for %s: %w", taskKey, err)
	}

	return dependents, nil
}

// blockTaskAndDependentsInTx recursively blocks a task and all its dependents within a transaction
func (r *TaskRepository) blockTaskAndDependentsInTx(ctx context.Context, tx *sql.Tx, task *models.Task, reopenedTaskKey string, blockedTasks map[string]bool, policy models.TaskDependencyStatusPolicy) error {
	// Skip if already processed
	if blockedTasks[task.Key] {
		return nil
	}

	// Skip tasks already in a terminal status.
	//
	// FOLLOW-UP: this reopen path has no channel for a service-resolved terminal
	// list — supplying one means threading it through the exported
	// ReopenTaskWithAutoBlock / ReopenTaskWithAutoBlockWithTx methods and their
	// service callers. Until then it routes through isTerminalTaskStatus with
	// the documented default, so the literal set lives in exactly one place in
	// this file rather than being restated here.
	if isTerminalTaskStatus(policy.TerminalStatuses, task.Status) {
		return nil
	}

	// Block this task
	reason := fmt.Sprintf("%s%s was reopened", DependencyReopenedBlockReasonPrefix, reopenedTaskKey)
	query := `UPDATE tasks SET status = ?, blocked_at = CURRENT_TIMESTAMP, blocked_reason = ? WHERE id = ?`
	blockedStatus := policy.BlockedStatuses
	if len(blockedStatus) == 0 {
		blockedStatus = defaultTaskBlockedStatuses
	}
	_, err := tx.ExecContext(ctx, query, blockedStatus[0], reason, task.ID)
	if err != nil {
		return fmt.Errorf("failed to block task: %w", err)
	}

	// Create history record
	historyQuery := `INSERT INTO task_history (task_id, old_status, new_status, agent, notes, forced) VALUES (?, ?, ?, ?, ?, ?)`
	_, err = tx.ExecContext(ctx, historyQuery, task.ID, task.Status, blockedStatus[0], nil, reason, false)
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
		if err := r.blockTaskAndDependentsInTx(ctx, tx, dependent, reopenedTaskKey, blockedTasks, policy); err != nil {
			return err
		}
	}

	return nil
}

// defaultTaskTerminalStatuses is the fallback terminal-status list used when a
// caller does not supply terminalStatuses. It preserves the historical
// hardcoded pair so existing callers keep their behavior. New callers should
// pass workflow.Service.ForLevel(workflow.LevelTask).GetTerminalStatuses()
// instead — the repository layer must not own this business rule, it only
// applies the list the service hands it. Same contract as
// defaultBugTerminalStatuses / defaultChangeCardTerminalStatuses.
var defaultTaskTerminalStatuses = []string{"completed", "archived"}

var defaultTaskExecutionStatuses = []string{"in_progress"}

var defaultTaskBlockedStatuses = []string{"blocked"}

const defaultTaskUnblockedStatus models.TaskStatus = "todo"

const defaultTaskReopenStatus models.TaskStatus = "in_progress"

// isTerminalTaskStatus reports whether status appears in terminalStatuses
// (case-insensitively), falling back to defaultTaskTerminalStatuses when the
// caller supplied none.
func isTerminalTaskStatus(terminalStatuses []string, status models.TaskStatus) bool {
	return taskStatusInList(terminalStatuses, defaultTaskTerminalStatuses, status)
}

func isExecutionTaskStatus(executionStatuses []string, status models.TaskStatus) bool {
	return taskStatusInList(executionStatuses, defaultTaskExecutionStatuses, status)
}

func isBlockedTaskStatus(blockedStatuses []string, status models.TaskStatus) bool {
	return taskStatusInList(blockedStatuses, defaultTaskBlockedStatuses, status)
}

func taskStatusInList(statuses, fallback []string, status models.TaskStatus) bool {
	list := statuses
	if len(list) == 0 {
		list = fallback
	}
	for _, t := range list {
		if strings.EqualFold(t, string(status)) {
			return true
		}
	}
	return false
}

// AutoUnblockDependents checks all tasks that depend on the completed task and
// unblocks those whose dependencies are all satisfied. Only tasks blocked with
// a dependency-pattern reason (set by ReopenTaskWithAutoBlock) are eligible.
// Returns the keys of tasks that were auto-unblocked.
//
// terminalStatuses is the caller-supplied (service-resolved) terminal-status
// list used to decide dependency satisfaction; pass nil to accept the
// documented default.
func (r *TaskRepository) AutoUnblockDependents(ctx context.Context, tx *sql.Tx, completedTaskKey string, terminalStatuses []string) ([]string, error) {
	return r.AutoUnblockDependentsWithPolicy(ctx, tx, completedTaskKey, terminalStatuses, nil, "")
}

// AutoUnblockDependentsWithPolicy applies a service-resolved dependency status
// policy while retaining the historical AutoUnblockDependents API for direct
// repository callers.
func (r *TaskRepository) AutoUnblockDependentsWithPolicy(ctx context.Context, tx *sql.Tx, completedTaskKey string, terminalStatuses, blockedStatuses []string, unblockedStatus models.TaskStatus) ([]string, error) {
	// Get dependents of the completed task
	dependents, err := r.getTaskDependentsInTx(ctx, tx, completedTaskKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependents: %w", err)
	}

	var unblocked []string
	for _, dependent := range dependents {
		// Only consider blocked tasks
		if !isBlockedTaskStatus(blockedStatuses, dependent.Status) {
			continue
		}

		// Only auto-unblock tasks blocked due to dependencies (not manual blocks)
		if !isDependencyBlocked(dependent) {
			continue
		}

		// Check if ALL dependencies of this task are now terminal
		allSatisfied, err := r.allDependenciesSatisfiedInTx(ctx, tx, dependent, terminalStatuses)
		if err != nil {
			return nil, fmt.Errorf("failed to check dependencies for %s: %w", dependent.Key, err)
		}

		if !allSatisfied {
			continue
		}

		// Unblock the task: set status to todo, clear blocked_at and blocked_reason
		if err := r.unblockTaskInTx(ctx, tx, dependent, unblockedStatus); err != nil {
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
	return strings.HasPrefix(reason, DependencyReopenedBlockReasonPrefix) ||
		strings.HasPrefix(reason, AutoBlockedReasonPrefix)
}

// allDependenciesSatisfiedInTx checks whether every dependency of the task is
// in a terminal status. It checks both the legacy depends_on JSON field and the
// entity_relationships table.
//
// Terminality is decided by the caller-supplied terminalStatuses list (resolved
// from the task workflow by the service layer) rather than a hardcoded
// completed/archived pair, so custom workflows that rename terminal statuses
// keep unblocking correctly. Empty list => documented default.
func (r *TaskRepository) allDependenciesSatisfiedInTx(ctx context.Context, tx *sql.Tx, task *models.Task, terminalStatuses []string) (bool, error) {
	// 1. Check legacy depends_on JSON field
	if task.DependsOn != nil && *task.DependsOn != "" && *task.DependsOn != "[]" {
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

			if !isTerminalTaskStatus(terminalStatuses, models.TaskStatus(status)) {
				return false, nil
			}
		}
	}

	// 2. Check entity_relationships table for outgoing depends_on relationships
	relQuery := `
		SELECT t.status FROM entity_relationships er
		JOIN tasks t ON t.id = er.to_entity_id
		WHERE er.from_entity_type = 'task' AND er.from_entity_id = ?
		  AND er.to_entity_type = 'task'
		  AND er.relationship_type = 'depends_on'
	`
	rows, err := tx.QueryContext(ctx, relQuery, task.ID)
	if err != nil {
		return false, fmt.Errorf("failed to query entity_relationships for %s: %w", task.Key, err)
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		if err := rows.Scan(&status); err != nil {
			return false, fmt.Errorf("failed to scan relationship dependency status: %w", err)
		}

		if !isTerminalTaskStatus(terminalStatuses, models.TaskStatus(status)) {
			return false, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("error iterating relationship dependencies for %s: %w", task.Key, err)
	}

	return true, nil
}

// unblockTaskInTx transitions a task from blocked to todo within a transaction,
// clearing blocked_at and blocked_reason, and recording history.
func (r *TaskRepository) unblockTaskInTx(ctx context.Context, tx *sql.Tx, task *models.Task, unblockedStatus models.TaskStatus) error {
	if unblockedStatus == "" {
		unblockedStatus = defaultTaskUnblockedStatus
	}
	query := `UPDATE tasks SET status = ?, blocked_at = NULL, blocked_reason = NULL WHERE id = ?`
	_, err := tx.ExecContext(ctx, query, unblockedStatus, task.ID)
	if err != nil {
		return fmt.Errorf("failed to unblock task: %w", err)
	}

	notes := "Auto-unblocked: all dependencies satisfied"
	historyQuery := `INSERT INTO task_history (task_id, old_status, new_status, agent, notes, forced) VALUES (?, ?, ?, ?, ?, ?)`
	_, err = tx.ExecContext(ctx, historyQuery, task.ID, task.Status, unblockedStatus, nil, notes, false)
	if err != nil {
		return fmt.Errorf("failed to create history record: %w", err)
	}

	return nil
}
