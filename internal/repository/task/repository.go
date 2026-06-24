// Package repository provides data access layer with context support.
//
// All repository methods accept context.Context as the first parameter to support:
// - Request cancellation
// - Timeout management
// - Distributed tracing
// - Request-scoped values
//
// Callers should create contexts appropriately:
// - HTTP handlers: Use r.Context() from http.Request
// - CLI commands: Use context.WithTimeout(context.Background(), timeout)
// - Tests: Use context.Background() or context.WithTimeout()
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	task, err := repo.GetByID(ctx, taskID)
//	if err != nil {
//	    return err
//	}
package task

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	repoerr "github.com/jwwelbor/shark-task-manager/internal/repository/repoerr"
	"github.com/jwwelbor/shark-task-manager/internal/repository/repoutil"
	"github.com/jwwelbor/shark-task-manager/internal/slug"
)

var tracer = repoutil.NewTracer("internal/repository/task")

// NoteCreator is a minimal interface for creating rejection notes within a transaction.
// It is defined here (in the task package scope) to avoid an import cycle between
// the task repository and the note repository sub-packages.
//
// EntityNoteRepository satisfies this interface; callers can inject it via
// NewTaskRepositoryWithNoteCreator or via the root-package NewTaskRepository wrapper
// in aliases.go which wires note.EntityNoteRepository as the NoteCreator.
type NoteCreator interface {
	CreateRejectionNoteWithTx(ctx context.Context, tx *sql.Tx,
		entityType models.EntityType, entityID int64, historyID int64,
		fromStatus, toStatus, reason, rejectedBy string, documentPath *string,
	) (*models.EntityNote, error)
}

// TaskRepository handles CRUD operations for tasks
type TaskRepository struct {
	db          *dbconn.DB
	noteCreator NoteCreator // optional; nil means rejection notes are silently skipped
}

// NewTaskRepository creates a TaskRepository without rejection note support.
// This is the simple constructor for use within the task package and tests.
// For full rejection note support, use NewTaskRepositoryWithNoteCreator or
// the root package NewTaskRepository (defined in aliases.go).
func NewTaskRepository(db *dbconn.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

// NewTaskRepositoryWithNoteCreator creates a TaskRepository with explicit rejection note support.
// When noteCreator is non-nil, rejection notes are created on forced status updates.
// When noteCreator is nil, rejection note creation is silently skipped (graceful degradation).
// Most callers should use the root package NewTaskRepository (defined in aliases.go) which
// automatically wires note.EntityNoteRepository as the NoteCreator.
func NewTaskRepositoryWithNoteCreator(db *dbconn.DB, noteCreator NoteCreator) *TaskRepository {
	return &TaskRepository{db: db, noteCreator: noteCreator}
}

// NewTaskRepositoryWithWorkflow creates a new TaskRepository.
// Deprecated: The workflow parameter is ignored. Use NewTaskRepository instead.
// This constructor is kept temporarily for backward compatibility with callers
// that pass a workflow config. It will be removed in a future cleanup.
func NewTaskRepositoryWithWorkflow(db *dbconn.DB, _ *config.WorkflowConfig) *TaskRepository {
	return &TaskRepository{db: db}
}

// Create creates a new task
func (r *TaskRepository) Create(ctx context.Context, task *models.Task) (retErr error) {
	ctx, span := tracer.Start(ctx, "TaskRepository.Create",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "INSERT"),
			attribute.String("db.table", "tasks"),
			attribute.String("db.key", task.Key),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	if err := task.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Validate dependencies before creating
	if err := r.ValidateTaskDependencies(ctx, task); err != nil {
		return fmt.Errorf("dependency validation failed: %w", err)
	}

	// Generate slug from title if not already set
	if task.Slug == nil {
		generatedSlug := slug.Generate(task.Title)
		task.Slug = &generatedSlug
	}

	query := `
		INSERT INTO tasks (
			feature_id, key, title, slug, description, status, agent_type, priority,
			depends_on, assigned_agent, file_path, blocked_reason, execution_order, size
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		task.FeatureID,
		task.Key,
		task.Title,
		task.Slug,
		task.Description,
		task.Status,
		task.AgentType,
		task.Priority,
		task.DependsOn,
		task.AssignedAgent,
		task.FilePath,
		task.BlockedReason,
		task.ExecutionOrder,
		task.Size,
	)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	task.ID = id
	return nil
}

// GetByID retrieves a task by its ID
func (r *TaskRepository) GetByID(ctx context.Context, id int64) (_ *models.Task, retErr error) {
	ctx, span := tracer.Start(ctx, "TaskRepository.GetByID",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "SELECT"),
			attribute.String("db.table", "tasks"),
			attribute.Int64("db.id", id),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	query := `
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data, size
		FROM tasks
		WHERE id = ?
	`

	task := &models.Task{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&task.ID,
		&task.FeatureID,
		&task.Key,
		&task.Title,
		&task.Slug,
		&task.Description,
		&task.Status,
		&task.AgentType,
		&task.Priority,
		&task.DependsOn,
		&task.AssignedAgent,
		&task.FilePath,
		&task.BlockedReason,
		&task.ExecutionOrder,
		&task.CreatedAt,
		&task.StartedAt,
		&task.CompletedAt,
		&task.BlockedAt,
		&task.UpdatedAt,
		&task.CompletedBy,
		&task.CompletionNotes,
		&task.FilesChanged,
		&task.TestsPassed,
		&task.VerificationStatus,
		&task.TimeSpentMinutes,
		&task.ContextData,
		&task.Size,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found with id %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	return task, nil
}

// GetByIDs retrieves multiple tasks by their IDs in a single query.
// Returns only found tasks; missing IDs are silently skipped.
// Returns empty slice (not nil) for empty or nil input.
func (r *TaskRepository) GetByIDs(ctx context.Context, ids []int64) ([]*models.Task, error) {
	if len(ids) == 0 {
		return []*models.Task{}, nil
	}

	// Build parameterized IN clause
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := `
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data, size
		FROM tasks
		WHERE id IN (` + strings.Join(placeholders, ", ") + `)
	`

	tasks, err := r.queryTasks(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	if tasks == nil {
		return []*models.Task{}, nil
	}
	return tasks, nil
}

// GetByKey retrieves a task by its key, supporting both numeric and slugged formats.
// Supports two key formats:
// 1. Numeric: T-E04-F01-001
// 2. Slugged: T-E04-F01-001-task-name
//
// Lookup strategy:
// 1. Try exact match on the key column (handles legacy numeric keys)
// 2. If not found and key contains a slug suffix, parse and match numeric key + slug
func (r *TaskRepository) GetByKey(ctx context.Context, key string) (_ *models.Task, retErr error) {
	ctx, span := tracer.Start(ctx, "TaskRepository.GetByKey",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "SELECT"),
			attribute.String("db.table", "tasks"),
			attribute.String("db.key", key),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	if key == "" {
		return nil, fmt.Errorf("task key cannot be empty")
	}

	query := `
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data, size
		FROM tasks
		WHERE key = ?
	`

	task := &models.Task{}
	err := r.db.QueryRowContext(ctx, query, key).Scan(
		&task.ID,
		&task.FeatureID,
		&task.Key,
		&task.Title,
		&task.Slug,
		&task.Description,
		&task.Status,
		&task.AgentType,
		&task.Priority,
		&task.DependsOn,
		&task.AssignedAgent,
		&task.FilePath,
		&task.BlockedReason,
		&task.ExecutionOrder,
		&task.CreatedAt,
		&task.StartedAt,
		&task.CompletedAt,
		&task.BlockedAt,
		&task.UpdatedAt,
		&task.CompletedBy,
		&task.CompletionNotes,
		&task.FilesChanged,
		&task.TestsPassed,
		&task.VerificationStatus,
		&task.TimeSpentMinutes,
		&task.ContextData,
		&task.Size,
	)

	if err == nil {
		// Found by exact match on key column
		return task, nil
	}

	if err != sql.ErrNoRows {
		// Unexpected error
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// Not found by exact match - try parsing as slugged key
	// Expected format: T-E##-F##-###-slug-text
	// Parse to extract numeric key and slug
	numericKey, slug, ok := parseSluggedKey(key)
	if !ok {
		// Cannot parse as slugged key, return not found
		return nil, fmt.Errorf("task not found with key %s", key)
	}

	// Try lookup by numeric key + slug match
	queryWithSlug := `
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data, size
		FROM tasks
		WHERE key = ? AND slug = ?
	`

	err = r.db.QueryRowContext(ctx, queryWithSlug, numericKey, slug).Scan(
		&task.ID,
		&task.FeatureID,
		&task.Key,
		&task.Title,
		&task.Slug,
		&task.Description,
		&task.Status,
		&task.AgentType,
		&task.Priority,
		&task.DependsOn,
		&task.AssignedAgent,
		&task.FilePath,
		&task.BlockedReason,
		&task.ExecutionOrder,
		&task.CreatedAt,
		&task.StartedAt,
		&task.CompletedAt,
		&task.BlockedAt,
		&task.UpdatedAt,
		&task.CompletedBy,
		&task.CompletionNotes,
		&task.FilesChanged,
		&task.TestsPassed,
		&task.VerificationStatus,
		&task.TimeSpentMinutes,
		&task.ContextData,
		&task.Size,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found with key %s", key)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	return task, nil
}

// parseSluggedKey parses a slugged task key into numeric key and slug components.
// Input format: T-E##-F##-###-slug-text
// Returns: numericKey (T-E##-F##-###), slug (slug-text), ok (true if valid format)
//
// Uses the shared SplitAtNthHyphen helper to split at the 4th hyphen, which
// separates the numeric task key from the slug suffix.
func parseSluggedKey(key string) (numericKey string, slugVal string, ok bool) {
	// Task key format: T-E##-F##-###
	// Minimum length with slug: T-E1-F1-1-slug = at least 14 characters
	if len(key) < 14 {
		return "", "", false
	}

	// Check prefix
	if !strings.HasPrefix(key, "T-") {
		return "", "", false
	}

	// Split at the 4th hyphen: T-E##-F##-###-slug-text
	//                          ^  ^   ^   ^
	//                          1  2   3   4
	numericKey, slugVal, ok = repoutil.SplitAtNthHyphen(key, 4)
	if !ok || slugVal == "" {
		return "", "", false
	}

	// Validate numeric key format matches T-E##-F##-###
	// At minimum: T-E1-F1-1
	if len(numericKey) < 9 {
		return "", "", false
	}

	return numericKey, slugVal, true
}

// GetByFilePath retrieves a task by its file path
// Returns sql.ErrNoRows if no task found with that file path
func (r *TaskRepository) GetByFilePath(ctx context.Context, filePath string) (*models.Task, error) {
	query := `
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data, size
		FROM tasks
		WHERE file_path = ?
	`

	task := &models.Task{}
	err := r.db.QueryRowContext(ctx, query, filePath).Scan(
		&task.ID,
		&task.FeatureID,
		&task.Key,
		&task.Title,
		&task.Slug,
		&task.Description,
		&task.Status,
		&task.AgentType,
		&task.Priority,
		&task.DependsOn,
		&task.AssignedAgent,
		&task.FilePath,
		&task.BlockedReason,
		&task.ExecutionOrder,
		&task.CreatedAt,
		&task.StartedAt,
		&task.CompletedAt,
		&task.BlockedAt,
		&task.UpdatedAt,
		&task.CompletedBy,
		&task.CompletionNotes,
		&task.FilesChanged,
		&task.TestsPassed,
		&task.VerificationStatus,
		&task.TimeSpentMinutes,
		&task.ContextData,
		&task.Size,
	)

	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task by file path: %w", err)
	}

	return task, nil
}

// UpdateFilePath updates the file_path for a task
// Pass nil to clear the file path
func (r *TaskRepository) UpdateFilePath(ctx context.Context, taskKey string, newFilePath *string) error {
	query := `
		UPDATE tasks
		SET file_path = ?, updated_at = CURRENT_TIMESTAMP
		WHERE key = ?
	`

	result, err := r.db.ExecContext(ctx, query, newFilePath, taskKey)
	if err != nil {
		return fmt.Errorf("failed to update file path: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task not found: %s", taskKey)
	}

	return nil
}

// ListByFeature retrieves all tasks for a feature
func (r *TaskRepository) ListByFeature(ctx context.Context, featureID int64) (_ []*models.Task, retErr error) {
	ctx, span := tracer.Start(ctx, "TaskRepository.ListByFeature",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "SELECT"),
			attribute.String("db.table", "tasks"),
			attribute.Int64("db.feature_id", featureID),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	query := `
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data, size
		FROM tasks
		WHERE feature_id = ?
		ORDER BY execution_order NULLS LAST, priority ASC, created_at ASC, key ASC
	`

	return r.queryTasks(ctx, query, featureID)
}

// ListByFeatureKey retrieves all tasks for a feature using the feature key directly.
// This avoids the two-step lookup (feature key → feature ID → tasks) and is more
// efficient for remote databases where each round-trip has network latency.
func (r *TaskRepository) ListByFeatureKey(ctx context.Context, featureKey string) ([]*models.Task, error) {
	query := `
		SELECT t.id, t.feature_id, t.key, t.title, t.slug, t.description, t.status, t.agent_type, t.priority,
		       t.depends_on, t.assigned_agent, t.file_path, t.blocked_reason, t.execution_order,
		       t.created_at, t.started_at, t.completed_at, t.blocked_at, t.updated_at,
		       t.completed_by, t.completion_notes, t.files_changed, t.tests_passed,
		       t.verification_status, t.time_spent_minutes, t.context_data, t.size
		FROM tasks t
		INNER JOIN features f ON t.feature_id = f.id
		WHERE f.key = ?
		ORDER BY t.execution_order NULLS LAST, t.priority ASC, t.created_at ASC, t.key ASC
	`

	return r.queryTasks(ctx, query, featureKey)
}

// ListByEpic retrieves all tasks for an epic (via features)
func (r *TaskRepository) ListByEpic(ctx context.Context, epicKey string) (_ []*models.Task, retErr error) {
	ctx, span := tracer.Start(ctx, "TaskRepository.ListByEpic",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "SELECT"),
			attribute.String("db.table", "tasks"),
			attribute.String("db.key", epicKey),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	query := `
		SELECT t.id, t.feature_id, t.key, t.title, t.slug, t.description, t.status, t.agent_type, t.priority,
		       t.depends_on, t.assigned_agent, t.file_path, t.blocked_reason, t.execution_order,
		       t.created_at, t.started_at, t.completed_at, t.blocked_at, t.updated_at,
		       t.completed_by, t.completion_notes, t.files_changed, t.tests_passed,
		       t.verification_status, t.time_spent_minutes, t.context_data, t.size
		FROM tasks t
		INNER JOIN features f ON t.feature_id = f.id
		INNER JOIN epics e ON f.epic_id = e.id
		WHERE e.key = ?
		ORDER BY t.execution_order NULLS LAST, t.priority ASC, t.created_at ASC, t.key ASC
	`

	return r.queryTasks(ctx, query, epicKey)
}

// ListBlockedTasksByEpic retrieves all blocked tasks for an epic.
// This method is more efficient than loading all tasks and filtering client-side.
func (r *TaskRepository) ListBlockedTasksByEpic(ctx context.Context, epicKey string) ([]*models.Task, error) {
	query := `
		SELECT t.id, t.feature_id, t.key, t.title, t.slug, t.description, t.status, t.agent_type, t.priority,
		       t.depends_on, t.assigned_agent, t.file_path, t.blocked_reason, t.execution_order,
		       t.created_at, t.started_at, t.completed_at, t.blocked_at, t.updated_at,
		       t.completed_by, t.completion_notes, t.files_changed, t.tests_passed,
		       t.verification_status, t.time_spent_minutes, t.context_data, t.size
		FROM tasks t
		INNER JOIN features f ON t.feature_id = f.id
		INNER JOIN epics e ON f.epic_id = e.id
		WHERE e.key = ? AND t.status = ?
		ORDER BY t.blocked_at DESC NULLS LAST, t.priority ASC, t.created_at ASC, t.key ASC
	`

	return r.queryTasks(ctx, query, epicKey, models.TaskStatus("blocked"))
}

// FilterByStatus retrieves tasks filtered by status
func (r *TaskRepository) FilterByStatus(ctx context.Context, status models.TaskStatus) ([]*models.Task, error) {
	query := `
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data, size
		FROM tasks
		WHERE status = ?
		ORDER BY execution_order NULLS LAST, priority ASC, created_at ASC, key ASC
	`

	return r.queryTasks(ctx, query, status)
}

// FilterByAgentType retrieves tasks filtered by agent type
func (r *TaskRepository) FilterByAgentType(ctx context.Context, agentType string) ([]*models.Task, error) {
	query := `
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data, size
		FROM tasks
		WHERE agent_type = ?
		ORDER BY execution_order NULLS LAST, priority ASC, created_at ASC, key ASC
	`

	return r.queryTasks(ctx, query, agentType)
}

// FilterCombined retrieves tasks with multiple filter criteria
func (r *TaskRepository) FilterCombined(ctx context.Context, status *models.TaskStatus, epicKey *string, agentType *string, maxPriority *int) (_ []*models.Task, retErr error) {
	ctx, span := tracer.Start(ctx, "TaskRepository.FilterCombined",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "SELECT"),
			attribute.String("db.table", "tasks"),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	query := `
		SELECT t.id, t.feature_id, t.key, t.title, t.slug, t.description, t.status, t.agent_type, t.priority,
		       t.depends_on, t.assigned_agent, t.file_path, t.blocked_reason, t.execution_order,
		       t.created_at, t.started_at, t.completed_at, t.blocked_at, t.updated_at,
		       t.completed_by, t.completion_notes, t.files_changed, t.tests_passed,
		       t.verification_status, t.time_spent_minutes, t.context_data, t.size
		FROM tasks t
	`

	args := []interface{}{}
	conditions := []string{}

	if epicKey != nil {
		query += `
		INNER JOIN features f ON t.feature_id = f.id
		INNER JOIN epics e ON f.epic_id = e.id
		`
		conditions = append(conditions, "e.key = ?")
		args = append(args, *epicKey)
	}

	if status != nil {
		conditions = append(conditions, "t.status = ?")
		args = append(args, *status)
	}

	if agentType != nil {
		conditions = append(conditions, "t.agent_type = ?")
		args = append(args, *agentType)
	}

	if maxPriority != nil {
		conditions = append(conditions, "t.priority <= ?")
		args = append(args, *maxPriority)
	}

	if len(conditions) > 0 {
		query += " WHERE "
		for i, cond := range conditions {
			if i > 0 {
				query += " AND "
			}
			query += cond
		}
	}

	query += " ORDER BY t.execution_order NULLS LAST, t.priority ASC, t.created_at ASC, t.key ASC"

	tasks, err := r.queryTasks(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	return tasks, nil
}

// ListByKeyPrefix retrieves all tasks whose key starts with the given prefix.
// Used by key generation to find globally unique keys regardless of feature_id.
func (r *TaskRepository) ListByKeyPrefix(ctx context.Context, prefix string) ([]*models.Task, error) {
	query := `
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data, size
		FROM tasks
		WHERE key LIKE ?
		ORDER BY key ASC
	`

	return r.queryTasks(ctx, query, prefix+"%")
}

// List retrieves all tasks
func (r *TaskRepository) List(ctx context.Context) (_ []*models.Task, retErr error) {
	ctx, span := tracer.Start(ctx, "TaskRepository.List",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "SELECT"),
			attribute.String("db.table", "tasks"),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	query := `
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data, size
		FROM tasks
		ORDER BY execution_order NULLS LAST, priority ASC, created_at ASC, key ASC
	`

	tasks, err := r.queryTasks(ctx, query)
	if err != nil {
		return nil, err
	}

	return tasks, nil
}

// ListWithViewerRelationships returns all tasks with their relationship data
// pre-resolved via the viewer_task_relationships view. One DB round-trip replaces
// the N+1 per-task calls that previously caused Hierarchy and FeatureTasks to hang.
// The RelationshipsJSON field contains a JSON array from the view.
func (r *TaskRepository) ListWithViewerRelationships(ctx context.Context) ([]*models.ViewerTaskWithRelationships, error) {
	query := `
		SELECT t.id, t.feature_id, t.key, t.title, t.slug, t.description, t.status,
		       t.agent_type, t.priority, t.depends_on, t.assigned_agent, t.file_path,
		       t.blocked_reason, t.execution_order, t.created_at, t.started_at,
		       t.completed_at, t.blocked_at, t.updated_at, t.completed_by,
		       t.completion_notes, t.files_changed, t.tests_passed,
		       t.verification_status, t.time_spent_minutes, t.context_data, t.size,
		       COALESCE(vtr.relationships_json, '[]') AS relationships_json
		FROM tasks t
		LEFT JOIN viewer_task_relationships vtr ON vtr.task_id = t.id
		ORDER BY t.execution_order NULLS LAST, t.priority ASC, t.created_at ASC, t.key ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks with viewer relationships: %w", err)
	}
	defer rows.Close()

	var result []*models.ViewerTaskWithRelationships
	for rows.Next() {
		row := &models.ViewerTaskWithRelationships{Task: &models.Task{}}
		var relsJSON string
		err := rows.Scan(
			&row.Task.ID, &row.Task.FeatureID, &row.Task.Key, &row.Task.Title,
			&row.Task.Slug, &row.Task.Description, &row.Task.Status,
			&row.Task.AgentType, &row.Task.Priority, &row.Task.DependsOn,
			&row.Task.AssignedAgent, &row.Task.FilePath,
			&row.Task.BlockedReason, &row.Task.ExecutionOrder,
			&row.Task.CreatedAt, &row.Task.StartedAt,
			&row.Task.CompletedAt, &row.Task.BlockedAt, &row.Task.UpdatedAt,
			&row.Task.CompletedBy, &row.Task.CompletionNotes,
			&row.Task.FilesChanged, &row.Task.TestsPassed,
			&row.Task.VerificationStatus, &row.Task.TimeSpentMinutes,
			&row.Task.ContextData, &row.Task.Size,
			&relsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan viewer task row: %w", err)
		}
		row.RelationshipsJSON = relsJSON
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating viewer task rows: %w", err)
	}
	return result, nil
}

// ListByFeatureWithViewerRelationships returns tasks for a specific feature with
// relationship data pre-resolved via the viewer_task_relationships view.
// One DB round-trip replaces per-task relationship calls in FeatureTasks.
func (r *TaskRepository) ListByFeatureWithViewerRelationships(ctx context.Context, featureID int64) ([]*models.ViewerTaskWithRelationships, error) {
	query := `
		SELECT t.id, t.feature_id, t.key, t.title, t.slug, t.description, t.status,
		       t.agent_type, t.priority, t.depends_on, t.assigned_agent, t.file_path,
		       t.blocked_reason, t.execution_order, t.created_at, t.started_at,
		       t.completed_at, t.blocked_at, t.updated_at, t.completed_by,
		       t.completion_notes, t.files_changed, t.tests_passed,
		       t.verification_status, t.time_spent_minutes, t.context_data, t.size,
		       COALESCE(vtr.relationships_json, '[]') AS relationships_json
		FROM tasks t
		LEFT JOIN viewer_task_relationships vtr ON vtr.task_id = t.id
		WHERE t.feature_id = ?
		ORDER BY t.execution_order NULLS LAST, t.priority ASC, t.created_at ASC, t.key ASC
	`
	rows, err := r.db.QueryContext(ctx, query, featureID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks by feature with viewer relationships: %w", err)
	}
	defer rows.Close()

	var result []*models.ViewerTaskWithRelationships
	for rows.Next() {
		row := &models.ViewerTaskWithRelationships{Task: &models.Task{}}
		var relsJSON string
		err := rows.Scan(
			&row.Task.ID, &row.Task.FeatureID, &row.Task.Key, &row.Task.Title,
			&row.Task.Slug, &row.Task.Description, &row.Task.Status,
			&row.Task.AgentType, &row.Task.Priority, &row.Task.DependsOn,
			&row.Task.AssignedAgent, &row.Task.FilePath,
			&row.Task.BlockedReason, &row.Task.ExecutionOrder,
			&row.Task.CreatedAt, &row.Task.StartedAt,
			&row.Task.CompletedAt, &row.Task.BlockedAt, &row.Task.UpdatedAt,
			&row.Task.CompletedBy, &row.Task.CompletionNotes,
			&row.Task.FilesChanged, &row.Task.TestsPassed,
			&row.Task.VerificationStatus, &row.Task.TimeSpentMinutes,
			&row.Task.ContextData, &row.Task.Size,
			&relsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan viewer task row by feature: %w", err)
		}
		row.RelationshipsJSON = relsJSON
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating viewer task rows by feature: %w", err)
	}
	return result, nil
}

// BeginTx starts a new database transaction for use by the service layer.
// Services own transaction boundaries per Standard 8; repositories participate
// in service-owned transactions by accepting *sql.Tx parameters.
func (r *TaskRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTxContext(ctx)
}

// Update updates an existing task.
// It starts an internal transaction to handle execution_order cascade atomically.
// For service-owned transactions, use updateWithTx directly after calling BeginTx.
func (r *TaskRepository) Update(ctx context.Context, task *models.Task) (retErr error) {
	ctx, span := tracer.Start(ctx, "TaskRepository.Update",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "UPDATE"),
			attribute.String("db.table", "tasks"),
			attribute.String("db.key", task.Key),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	return r.updateInternal(ctx, task, false)
}

// UpdateNoResequence updates a task without cascading execution_order changes
// to siblings. Used to preserve intentional duplicate-order groups (parallel
// work) when callers re-assign a task's order via `shark task update --parallel`.
// Equivalent to Update in every other respect.
func (r *TaskRepository) UpdateNoResequence(ctx context.Context, task *models.Task) (retErr error) {
	ctx, span := tracer.Start(ctx, "TaskRepository.UpdateNoResequence",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "UPDATE"),
			attribute.String("db.table", "tasks"),
			attribute.String("db.key", task.Key),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	return r.updateInternal(ctx, task, true)
}

// updateInternal performs the task update. When forceSkipCascade is true the
// execution_order resequence is suppressed regardless of whether the order has
// actually changed.
//
// In the skip-cascade path (TD-008), two optimizations apply:
//  1. Dependency validation is bypassed — the --parallel update path renumbers
//     an existing task and never changes DependsOn, so re-validating the
//     existing graph wastes a SELECT round-trip.
//  2. No transaction is opened — the operation is a single non-cascading row
//     update, so BEGIN/COMMIT add latency (meaningful on Turso, negligible on
//     local SQLite) without any atomicity benefit.
func (r *TaskRepository) updateInternal(ctx context.Context, task *models.Task, forceSkipCascade bool) error {
	if err := task.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Skip-cascade fast path: single-row UPDATE, no transaction, no dep validation.
	// Used by UpdateNoResequence (--parallel renumber); DependsOn is unchanged in
	// this path so circular-dependency re-validation is unnecessary.
	if forceSkipCascade {
		return r.updateRowDirect(ctx, task)
	}

	// Validate dependencies before updating
	if err := r.ValidateTaskDependencies(ctx, task); err != nil {
		return fmt.Errorf("dependency validation failed: %w", err)
	}

	// Check if execution_order is being changed - if so, cascade to other tasks.
	var needsCascade bool
	if task.ExecutionOrder != nil {
		oldTask, err := r.GetByID(ctx, task.ID)
		if err != nil {
			return fmt.Errorf("failed to get old task: %w", err)
		}

		// Check if order actually changed
		needsCascade = (oldTask.ExecutionOrder == nil) ||
			(oldTask.ExecutionOrder != nil && *oldTask.ExecutionOrder != *task.ExecutionOrder)
	}

	// Start transaction for cascade updates
	tx, err := r.db.BeginTxContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := r.updateWithTx(ctx, tx, task, needsCascade); err != nil {
		return err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// updateRowDirect performs a single-row task UPDATE outside any transaction.
// Used by the --parallel update path (forceSkipCascade=true) where no sibling
// cascade is needed and atomicity across multiple rows is irrelevant. See
// TD-008 for the rationale.
func (r *TaskRepository) updateRowDirect(ctx context.Context, task *models.Task) error {
	query := `
		UPDATE tasks
		SET title = ?, description = ?, status = ?, agent_type = ?, priority = ?,
		    depends_on = ?, assigned_agent = ?, file_path = ?, blocked_reason = ?, execution_order = ?, context_data = ?, size = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		task.Title,
		task.Description,
		task.Status,
		task.AgentType,
		task.Priority,
		task.DependsOn,
		task.AssignedAgent,
		task.FilePath,
		task.BlockedReason,
		task.ExecutionOrder,
		task.ContextData,
		task.Size,
		task.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("task not found with id %d: %w", task.ID, repoerr.ErrNotFound)
	}

	return nil
}

// updateWithTx performs the task update within a caller-provided transaction.
// The needsCascade flag indicates whether execution_order cascade is required.
// Validation must be performed by the caller before invoking this method.
func (r *TaskRepository) updateWithTx(ctx context.Context, tx *sql.Tx, task *models.Task, needsCascade bool) error {
	// If cascade is needed, get all tasks BEFORE updating, then resequence ALL tasks
	if needsCascade {
		// Get all tasks in the same feature (before any updates)
		allTasks, err := r.listByFeatureInTx(ctx, tx, task.FeatureID)
		if err != nil {
			return fmt.Errorf("failed to list tasks for cascade: %w", err)
		}

		// Convert to repoutil.OrderedItem format
		var items []repoutil.OrderedItem
		for _, t := range allTasks {
			items = append(items, repoutil.OrderedItem{
				ID:             t.ID,
				ExecutionOrder: t.ExecutionOrder,
			})
		}

		// Resequence
		resequenced := repoutil.ResequenceOrders(items, task.ID, task.ExecutionOrder)

		// Update ALL tasks with new orders
		updateQuery := "UPDATE tasks SET execution_order = ? WHERE id = ?"
		for _, item := range resequenced {
			_, err := tx.ExecContext(ctx, updateQuery, item.ExecutionOrder, item.ID)
			if err != nil {
				return fmt.Errorf("failed to cascade update order for task %d: %w", item.ID, err)
			}
		}

		// Now update the main task's other fields (execution_order already updated above)
		query := `
			UPDATE tasks
			SET title = ?, description = ?, status = ?, agent_type = ?, priority = ?,
			    depends_on = ?, assigned_agent = ?, file_path = ?, blocked_reason = ?, context_data = ?, size = ?
			WHERE id = ?
		`

		result, err := tx.ExecContext(ctx, query,
			task.Title,
			task.Description,
			task.Status,
			task.AgentType,
			task.Priority,
			task.DependsOn,
			task.AssignedAgent,
			task.FilePath,
			task.BlockedReason,
			task.ContextData,
			task.Size,
			task.ID,
		)
		if err != nil {
			return fmt.Errorf("failed to update task: %w", err)
		}

		rows, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}
		if rows == 0 {
			return fmt.Errorf("task not found with id %d", task.ID)
		}
	} else {
		// No cascade needed, just update the task normally
		query := `
			UPDATE tasks
			SET title = ?, description = ?, status = ?, agent_type = ?, priority = ?,
			    depends_on = ?, assigned_agent = ?, file_path = ?, blocked_reason = ?, execution_order = ?, context_data = ?, size = ?
			WHERE id = ?
		`

		result, err := tx.ExecContext(ctx, query,
			task.Title,
			task.Description,
			task.Status,
			task.AgentType,
			task.Priority,
			task.DependsOn,
			task.AssignedAgent,
			task.FilePath,
			task.BlockedReason,
			task.ExecutionOrder,
			task.ContextData,
			task.Size,
			task.ID,
		)
		if err != nil {
			return fmt.Errorf("failed to update task: %w", err)
		}

		rows, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}
		if rows == 0 {
			return fmt.Errorf("task not found with id %d", task.ID)
		}
	}

	return nil
}

// listByFeatureInTx lists tasks by feature within a transaction
func (r *TaskRepository) listByFeatureInTx(ctx context.Context, tx *sql.Tx, featureID int64) ([]*models.Task, error) {
	query := `
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, updated_at, context_data, size
		FROM tasks
		WHERE feature_id = ?
		ORDER BY execution_order ASC
	`

	rows, err := tx.QueryContext(ctx, query, featureID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		task := &models.Task{}
		err := rows.Scan(
			&task.ID,
			&task.FeatureID,
			&task.Key,
			&task.Title,
			&task.Slug,
			&task.Description,
			&task.Status,
			&task.AgentType,
			&task.Priority,
			&task.DependsOn,
			&task.AssignedAgent,
			&task.FilePath,
			&task.BlockedReason,
			&task.ExecutionOrder,
			&task.CreatedAt,
			&task.UpdatedAt,
			&task.ContextData,
			&task.Size,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tasks: %w", err)
	}

	return tasks, nil
}

// UpdateStatus atomically updates task status, timestamps, and creates history record
func (r *TaskRepository) UpdateStatus(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) (retErr error) {
	ctx, span := tracer.Start(ctx, "TaskRepository.UpdateStatus",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "UPDATE"),
			attribute.String("db.table", "tasks"),
			attribute.Int64("db.id", taskID),
			attribute.String("db.new_status", string(newStatus)),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	// For backward transitions, use notes as rejection reason if provided
	// This maintains API compatibility while supporting the rejection reason requirement
	return r.UpdateStatusForced(ctx, taskID, newStatus, agent, notes, notes, nil, false)
}

// UpdateStatusForced atomically updates task status with optional validation bypass
func (r *TaskRepository) UpdateStatusForced(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string, rejectionReason *string, documentPath *string, force bool) (retErr error) {
	ctx, span := tracer.Start(ctx, "TaskRepository.UpdateStatusForced",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "UPDATE"),
			attribute.String("db.table", "tasks"),
			attribute.Int64("db.id", taskID),
			attribute.String("db.new_status", string(newStatus)),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	_, err := r.updateStatusForcedInternal(ctx, taskID, newStatus, agent, notes, rejectionReason, documentPath, force)
	return err
}

// UpdateStatusForcedWithUnblock atomically updates task status and, when the new
// status is completed or archived, auto-unblocks dependent tasks whose dependencies
// are all satisfied. Returns the keys of auto-unblocked tasks.
func (r *TaskRepository) UpdateStatusForcedWithUnblock(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string, rejectionReason *string, documentPath *string, force bool) ([]string, error) {
	return r.updateStatusForcedInternal(ctx, taskID, newStatus, agent, notes, rejectionReason, documentPath, force)
}

// updateStatusForcedInternal is the shared implementation for UpdateStatusForced
// and UpdateStatusForcedWithUnblock. It performs the status update and auto-unblock
// in a single transaction, returning any auto-unblocked task keys.
func (r *TaskRepository) updateStatusForcedInternal(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string, rejectionReason *string, documentPath *string, force bool) ([]string, error) {
	// NOTE: All workflow validation (status enum, transition validity, backward transition checks)
	// is now handled by the service layer via executeStatusTransition() -> StatusUpdateRaw().
	// This method performs the raw database update without business-logic validation.
	// The force parameter is kept for backward compatibility but no longer affects behavior
	// since validation has been removed from this layer.

	// Start transaction
	tx, err := r.db.BeginTxContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	unblockedKeys, err := r.updateStatusForcedInternalWithTx(ctx, tx, taskID, newStatus, agent, notes, rejectionReason, documentPath, force)
	if err != nil {
		return nil, err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return unblockedKeys, nil
}

// updateStatusForcedInternalWithTx performs the forced status update within a caller-provided
// transaction. All validation must be performed by the caller before invoking this method.
func (r *TaskRepository) updateStatusForcedInternalWithTx(ctx context.Context, tx *sql.Tx, taskID int64, newStatus models.TaskStatus, agent *string, notes *string, rejectionReason *string, documentPath *string, force bool) ([]string, error) {
	// Get current task state
	var currentStatus string
	var taskKey string
	var startedAt, completedAt, blockedAt sql.NullTime
	err := tx.QueryRowContext(ctx, "SELECT key, status, started_at, completed_at, blocked_at FROM tasks WHERE id = ?", taskID).
		Scan(&taskKey, &currentStatus, &startedAt, &completedAt, &blockedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found with id %d", taskID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get current task status: %w", err)
	}

	if force {
		// Log warning when force is used
		fmt.Printf("WARNING: Forced status update from %s to %s (taskID=%d)\n", currentStatus, newStatus, taskID)
	}

	// Update status and timestamps
	now := time.Now()
	query := "UPDATE tasks SET status = ?"
	args := []interface{}{newStatus}

	// Set appropriate timestamp based on new status
	if newStatus == models.TaskStatus("in_progress") && !startedAt.Valid {
		query += ", started_at = ?"
		args = append(args, now)
	} else if newStatus == models.TaskStatus("completed") && !completedAt.Valid {
		query += ", completed_at = ?"
		args = append(args, now)
	} else if newStatus == models.TaskStatus("blocked") && !blockedAt.Valid {
		query += ", blocked_at = ?"
		args = append(args, now)
	}

	query += " WHERE id = ?"
	args = append(args, taskID)

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update task status: %w", err)
	}

	// Create history record with rejection reason support
	historyQuery := `
		INSERT INTO task_history (task_id, old_status, new_status, agent, notes, rejection_reason, forced)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	result, err := tx.ExecContext(ctx, historyQuery, taskID, currentStatus, newStatus, agent, notes, rejectionReason, force)
	if err != nil {
		return nil, fmt.Errorf("failed to create history record: %w", err)
	}

	historyID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get history record id: %w", err)
	}

	// Create rejection note if rejection reason is provided and NoteCreator is available
	if rejectionReason != nil && strings.TrimSpace(*rejectionReason) != "" && r.noteCreator != nil {
		rejectedBy := "system"
		if agent != nil && *agent != "" {
			rejectedBy = *agent
		}

		_, err := r.noteCreator.CreateRejectionNoteWithTx(
			ctx, tx, models.EntityTypeTask, taskID, historyID,
			currentStatus, string(newStatus),
			*rejectionReason, rejectedBy, documentPath,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create rejection note: %w", err)
		}
	}

	// Auto-unblock dependents when transitioning to completed or archived
	var unblockedKeys []string
	if newStatus == models.TaskStatus("completed") || newStatus == models.TaskStatus("archived") {
		unblockedKeys, err = r.AutoUnblockDependents(ctx, tx, taskKey)
		if err != nil {
			return nil, fmt.Errorf("failed to auto-unblock dependents: %w", err)
		}
	}

	return unblockedKeys, nil
}

// StatusUpdateRaw performs an atomic status update without any business-logic validation.
// All validation (transition validity, backward transition checks, reason requirements)
// must be performed by the calling service layer BEFORE invoking this method.
//
// This method performs the following in a single transaction:
//   - Updates the task status and relevant timestamps (started_at, completed_at, blocked_at)
//   - Creates a task_history record
//   - Optionally creates a rejection note (when RejectionReason is non-empty)
//   - Auto-unblocks dependent tasks when transitioning to completed/archived
//
// Returns the list of auto-unblocked task keys.
func (r *TaskRepository) StatusUpdateRaw(ctx context.Context, params models.StatusUpdateParams) ([]string, error) {
	// Start transaction
	tx, err := r.db.BeginTxContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	unblockedKeys, err := r.StatusUpdateRawWithTx(ctx, tx, params)
	if err != nil {
		return nil, err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return unblockedKeys, nil
}

// StatusUpdateRawWithTx performs the raw status update within a caller-provided transaction.
// This allows the service layer to own the transaction boundary when multiple repository
// operations must be atomic. All validation must be performed by the caller before invoking
// this method.
//
// Returns the list of auto-unblocked task keys.
func (r *TaskRepository) StatusUpdateRawWithTx(ctx context.Context, tx *sql.Tx, params models.StatusUpdateParams) ([]string, error) {
	// Update status and timestamps
	now := time.Now()
	query := "UPDATE tasks SET status = ?"
	args := []interface{}{params.NewStatus}

	// Set appropriate timestamp based on new status
	if params.NewStatus == models.TaskStatus("in_progress") && !params.StartedAt.Valid {
		query += ", started_at = ?"
		args = append(args, now)
	} else if params.NewStatus == models.TaskStatus("completed") && !params.CompletedAt.Valid {
		query += ", completed_at = ?"
		args = append(args, now)
	} else if params.NewStatus == models.TaskStatus("blocked") && !params.BlockedAt.Valid {
		query += ", blocked_at = ?"
		args = append(args, now)
	}

	query += " WHERE id = ?"
	args = append(args, params.TaskID)

	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update task status: %w", err)
	}

	// Create history record
	historyQuery := `
		INSERT INTO task_history (task_id, old_status, new_status, agent, notes, rejection_reason, forced)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	result, err := tx.ExecContext(ctx, historyQuery, params.TaskID, params.OldStatus, params.NewStatus, params.Agent, params.Notes, params.RejectionReason, params.Force)
	if err != nil {
		return nil, fmt.Errorf("failed to create history record: %w", err)
	}

	historyID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get history record id: %w", err)
	}

	// Create rejection note if rejection reason is provided and NoteCreator is available
	if params.RejectionReason != nil && strings.TrimSpace(*params.RejectionReason) != "" && r.noteCreator != nil {
		rejectedBy := "system"
		if params.Agent != nil && *params.Agent != "" {
			rejectedBy = *params.Agent
		}

		_, err := r.noteCreator.CreateRejectionNoteWithTx(
			ctx, tx, models.EntityTypeTask, params.TaskID, historyID,
			params.OldStatus, string(params.NewStatus),
			*params.RejectionReason, rejectedBy, params.DocumentPath,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create rejection note: %w", err)
		}
	}

	// Auto-unblock dependents when transitioning to completed or archived
	var unblockedKeys []string
	if params.NewStatus == models.TaskStatus("completed") || params.NewStatus == models.TaskStatus("archived") {
		unblockedKeys, err = r.AutoUnblockDependents(ctx, tx, params.TaskKey)
		if err != nil {
			return nil, fmt.Errorf("failed to auto-unblock dependents: %w", err)
		}
	}

	return unblockedKeys, nil
}

// UpdateStatusWithAction updates a task's status and returns the updated task with orchestrator action
// This method combines status update with retrieval of orchestrator action from workflow config
// Returns:
// - *models.Task: The updated task
// - *config.PopulatedAction: The orchestrator action for the new status (nil if not defined)
// - error: Any error that occurred during the update
func (r *TaskRepository) UpdateStatusWithAction(ctx context.Context, taskKey string, newStatus string, workflow *config.WorkflowConfig) (*models.Task, *config.PopulatedAction, error) {
	// Get task by key
	task, err := r.GetByKey(ctx, taskKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get task: %w", err)
	}

	// Update task status using existing method
	taskStatus := models.TaskStatus(newStatus)
	if err := r.UpdateStatus(ctx, task.ID, taskStatus, nil, nil); err != nil {
		return nil, nil, fmt.Errorf("failed to update task status: %w", err)
	}

	// Fetch updated task
	updatedTask, err := r.GetByID(ctx, task.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get updated task: %w", err)
	}

	// Get orchestrator action for new status from workflow config
	action, err := r.getOrchestratorAction(ctx, updatedTask, newStatus, workflow)
	if err != nil {
		// Log warning but don't fail - action is optional
		fmt.Printf("WARNING: Failed to get orchestrator action for status %s: %v\n", newStatus, err)
		action = nil
	}

	return updatedTask, action, nil
}

// getOrchestratorAction retrieves and populates orchestrator action for a status
// Returns nil if no action is defined for the status (not an error)
func (r *TaskRepository) getOrchestratorAction(ctx context.Context, task *models.Task, status string, workflow *config.WorkflowConfig) (*config.PopulatedAction, error) {
	// Check if workflow is configured
	if workflow == nil || workflow.StatusMetadata == nil {
		return nil, nil // No workflow - no actions
	}

	// Get status metadata
	metadata, exists := workflow.StatusMetadata[status]
	if !exists {
		return nil, nil // Status not in config - no actions
	}

	// Check if action is defined
	if metadata.OrchestratorAction == nil {
		return nil, nil // No action for this status - OK
	}

	// Return populated action
	populatedAction := metadata.OrchestratorAction.ToPopulatedAction(config.TaskPlaceholders(task))

	return populatedAction, nil
}

// Delete deletes a task (and its history via CASCADE)
func (r *TaskRepository) Delete(ctx context.Context, id int64) (retErr error) {
	ctx, span := tracer.Start(ctx, "TaskRepository.Delete",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "DELETE"),
			attribute.String("db.table", "tasks"),
			attribute.Int64("db.id", id),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	query := "DELETE FROM tasks WHERE id = ?"

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("task not found with id %d", id)
	}

	return nil
}

// GetStatusBreakdownMapBatch returns status counts as maps for multiple features in a single query.
// This method avoids N+1 query problems when fetching breakdowns for many features.
func (r *TaskRepository) GetStatusBreakdownMapBatch(ctx context.Context, featureIDs []int64) (map[int64]map[models.TaskStatus]int, error) {
	if len(featureIDs) == 0 {
		return make(map[int64]map[models.TaskStatus]int), nil
	}

	// Build the query with IN clause
	placeholders := make([]string, len(featureIDs))
	args := make([]interface{}, len(featureIDs))
	for i, id := range featureIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT feature_id, status, COUNT(*) as count
		FROM tasks
		WHERE feature_id IN (%s)
		GROUP BY feature_id, status
	`, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get batch status breakdown: %w", err)
	}
	defer rows.Close()

	// Initialize result map with empty maps for each feature
	result := make(map[int64]map[models.TaskStatus]int)
	for _, featureID := range featureIDs {
		result[featureID] = make(map[models.TaskStatus]int)
	}

	// Fill in actual counts from query
	for rows.Next() {
		var featureID int64
		var status models.TaskStatus
		var count int

		if err := rows.Scan(&featureID, &status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan batch status breakdown: %w", err)
		}

		result[featureID][status] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating batch status breakdown: %w", err)
	}

	return result, nil
}

// GetTaskCountForFeature returns the total number of tasks for a given feature
func (r *TaskRepository) GetTaskCountForFeature(ctx context.Context, featureID int64) (int, error) {
	query := `SELECT COUNT(*) FROM tasks WHERE feature_id = ?`

	var count int
	err := r.db.QueryRowContext(ctx, query, featureID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get task count for feature: %w", err)
	}

	return count, nil
}

// BulkCreate creates multiple tasks in a single transaction.
// Returns number of tasks created and error.
func (r *TaskRepository) BulkCreate(ctx context.Context, tasks []*models.Task) (int, error) {
	if len(tasks) == 0 {
		return 0, nil
	}

	// Validate all tasks before inserting
	for i, task := range tasks {
		if err := task.Validate(); err != nil {
			return 0, fmt.Errorf("validation failed for task %d: %w", i, err)
		}
	}

	// Start transaction
	tx, err := r.db.BeginTxContext(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	count, err := r.bulkCreateWithTx(ctx, tx, tasks)
	if err != nil {
		return count, err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return count, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return count, nil
}

// bulkCreateWithTx inserts multiple tasks within a caller-provided transaction.
// Validation must be performed by the caller before invoking this method.
// Returns the number of tasks successfully inserted.
func (r *TaskRepository) bulkCreateWithTx(ctx context.Context, tx *sql.Tx, tasks []*models.Task) (int, error) {
	// Prepare statement for efficiency
	query := `
		INSERT INTO tasks (
			feature_id, key, title, description, status, agent_type, priority,
			depends_on, assigned_agent, file_path, blocked_reason
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Insert all tasks
	count := 0
	for _, task := range tasks {
		result, err := stmt.ExecContext(ctx,
			task.FeatureID,
			task.Key,
			task.Title,
			task.Description,
			task.Status,
			task.AgentType,
			task.Priority,
			task.DependsOn,
			task.AssignedAgent,
			task.FilePath,
			task.BlockedReason,
		)
		if err != nil {
			return count, fmt.Errorf("failed to insert task %s: %w", task.Key, err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return count, fmt.Errorf("failed to get last insert id for task %s: %w", task.Key, err)
		}

		task.ID = id
		count++
	}

	return count, nil
}

// GetByKeys retrieves multiple tasks by their keys
// Returns map of key -> task, missing keys are omitted
func (r *TaskRepository) GetByKeys(ctx context.Context, keys []string) (map[string]*models.Task, error) {
	if len(keys) == 0 {
		return make(map[string]*models.Task), nil
	}

	// Build dynamic IN clause
	query := `
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data, size
		FROM tasks
		WHERE key IN (?` + strings.Repeat(", ?", len(keys)-1) + `)`

	// Convert keys to []interface{} for query
	args := make([]interface{}, len(keys))
	for i, key := range keys {
		args[i] = key
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks by keys: %w", err)
	}
	defer rows.Close()

	// Build result map
	result := make(map[string]*models.Task)
	for rows.Next() {
		task := &models.Task{}
		err := rows.Scan(
			&task.ID,
			&task.FeatureID,
			&task.Key,
			&task.Title,
			&task.Slug,
			&task.Description,
			&task.Status,
			&task.AgentType,
			&task.Priority,
			&task.DependsOn,
			&task.AssignedAgent,
			&task.FilePath,
			&task.BlockedReason,
			&task.ExecutionOrder,
			&task.CreatedAt,
			&task.StartedAt,
			&task.CompletedAt,
			&task.BlockedAt,
			&task.UpdatedAt,
			&task.CompletedBy,
			&task.CompletionNotes,
			&task.FilesChanged,
			&task.TestsPassed,
			&task.VerificationStatus,
			&task.TimeSpentMinutes,
			&task.ContextData,
			&task.Size,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		result[task.Key] = task
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tasks: %w", err)
	}

	return result, nil
}

// UpdateMetadata updates only metadata fields (title, description, file_path)
// Does NOT update status, priority, agent_type (database-only fields)
func (r *TaskRepository) UpdateMetadata(ctx context.Context, task *models.Task) error {
	if err := task.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		UPDATE tasks
		SET title = ?, description = ?, file_path = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		task.Title,
		task.Description,
		task.FilePath,
		task.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update task metadata: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("task not found with id %d", task.ID)
	}

	return nil
}

// GetMaxSequenceForFeature gets the maximum task sequence number for a feature
// Returns 0 if no tasks exist for the feature
func (r *TaskRepository) GetMaxSequenceForFeature(ctx context.Context, featureKey string) (int, error) {
	// Task keys are in format: T-E##-F##-###
	// We need to extract the sequence number (###) from the key
	// Use SQL to parse the key and find the maximum sequence
	query := `
		SELECT COALESCE(MAX(CAST(SUBSTR(t.key, -3) AS INTEGER)), 0) as max_sequence
		FROM tasks t
		INNER JOIN features f ON t.feature_id = f.id
		WHERE f.key = ? AND t.key LIKE 'T-' || ? || '-%'
	`

	var maxSequence int
	err := r.db.QueryRowContext(ctx, query, featureKey, featureKey).Scan(&maxSequence)
	if err != nil {
		return 0, fmt.Errorf("failed to get max sequence for feature %s: %w", featureKey, err)
	}

	return maxSequence, nil
}

// UpdateKey updates the key of a task
func (r *TaskRepository) UpdateKey(ctx context.Context, oldKey string, newKey string) error {
	// Validate new key doesn't already exist
	existing, err := r.GetByKey(ctx, newKey)
	if err == nil && existing != nil {
		return fmt.Errorf("task with key %s already exists", newKey)
	}

	query := `
		UPDATE tasks
		SET key = ?, updated_at = CURRENT_TIMESTAMP
		WHERE key = ?
	`

	result, err := r.db.ExecContext(ctx, query, newKey, oldKey)
	if err != nil {
		return fmt.Errorf("update task key: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("task not found: %s", oldKey)
	}

	return nil
}

// queryTasks is a helper function to execute task queries
func (r *TaskRepository) queryTasks(ctx context.Context, query string, args ...interface{}) ([]*models.Task, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		task := &models.Task{}
		err := rows.Scan(
			&task.ID,
			&task.FeatureID,
			&task.Key,
			&task.Title,
			&task.Slug,
			&task.Description,
			&task.Status,
			&task.AgentType,
			&task.Priority,
			&task.DependsOn,
			&task.AssignedAgent,
			&task.FilePath,
			&task.BlockedReason,
			&task.ExecutionOrder,
			&task.CreatedAt,
			&task.StartedAt,
			&task.CompletedAt,
			&task.BlockedAt,
			&task.UpdatedAt,
			&task.CompletedBy,
			&task.CompletionNotes,
			&task.FilesChanged,
			&task.TestsPassed,
			&task.VerificationStatus,
			&task.TimeSpentMinutes,
			&task.ContextData,
			&task.Size,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tasks: %w", err)
	}

	return tasks, nil
}

// UpdateCompletionMetadata updates completion metadata for a task
func (r *TaskRepository) UpdateCompletionMetadata(ctx context.Context, taskKey string, metadata *models.CompletionMetadata) error {
	if err := metadata.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Convert files_changed array to JSON
	filesJSON, err := metadata.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to convert files_changed to JSON: %w", err)
	}

	query := `
		UPDATE tasks
		SET completed_by = ?,
		    completion_notes = ?,
		    files_changed = ?,
		    tests_passed = ?,
		    verification_status = ?,
		    time_spent_minutes = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE key = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		metadata.CompletedBy,
		metadata.CompletionNotes,
		filesJSON,
		metadata.TestsPassed,
		metadata.VerificationStatus,
		metadata.TimeSpentMinutes,
		taskKey,
	)
	if err != nil {
		return fmt.Errorf("failed to update completion metadata: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task not found: %s", taskKey)
	}

	return nil
}

// GetCompletionMetadata retrieves completion metadata for a task
func (r *TaskRepository) GetCompletionMetadata(ctx context.Context, taskKey string) (*models.CompletionMetadata, error) {
	query := `
		SELECT completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, completed_at
		FROM tasks
		WHERE key = ?
	`

	metadata := models.NewCompletionMetadata()
	var filesJSON *string
	var completedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, taskKey).Scan(
		&metadata.CompletedBy,
		&metadata.CompletionNotes,
		&filesJSON,
		&metadata.TestsPassed,
		&metadata.VerificationStatus,
		&metadata.TimeSpentMinutes,
		&completedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found with key %s", taskKey)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get completion metadata: %w", err)
	}

	// Parse files_changed JSON
	if filesJSON != nil && *filesJSON != "" {
		if err := metadata.FromJSON(*filesJSON); err != nil {
			return nil, fmt.Errorf("failed to parse files_changed JSON: %w", err)
		}
	}

	// Set completed_at if valid
	if completedAt.Valid {
		metadata.CompletedAt = &completedAt.Time
	}

	return metadata, nil
}

// FindByFileChanged searches for tasks that created or modified a specific file
func (r *TaskRepository) FindByFileChanged(ctx context.Context, filePath string) ([]*models.Task, error) {
	query := `
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data, size
		FROM tasks
		WHERE files_changed IS NOT NULL
		  AND files_changed LIKE ?
		ORDER BY completed_at DESC NULLS LAST
	`

	// Use SQL LIKE pattern for partial matching
	pattern := "%" + filePath + "%"

	return r.queryTasks(ctx, query, pattern)
}

// GetUnverifiedTasks retrieves all tasks with verification_status != 'verified'
func (r *TaskRepository) GetUnverifiedTasks(ctx context.Context) ([]*models.Task, error) {
	query := `
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data, size
		FROM tasks
		WHERE verification_status != 'verified'
		  AND status IN ('ready_for_review', 'completed')
		ORDER BY completed_at DESC NULLS LAST
	`

	return r.queryTasks(ctx, query)
}

// FilterByMetadataAgentType retrieves tasks filtered by agent type from workflow metadata
// Uses status metadata to find statuses that include the specified agent type,
// then returns all tasks in those statuses
func (r *TaskRepository) FilterByMetadataAgentType(ctx context.Context, agentType string, workflow *config.WorkflowConfig) ([]*models.Task, error) {
	if workflow == nil {
		return []*models.Task{}, nil
	}

	// Get statuses that include this agent type
	statuses := workflow.GetStatusesByAgentType(agentType)
	if len(statuses) == 0 {
		// No statuses match this agent type - return empty list
		return []*models.Task{}, nil
	}

	// Build SQL query with IN clause for multiple statuses
	placeholders := make([]string, len(statuses))
	args := make([]interface{}, len(statuses))
	for i, status := range statuses {
		placeholders[i] = "?"
		args[i] = status
	}

	query := fmt.Sprintf(`
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data, size
		FROM tasks
		WHERE status IN (%s)
		ORDER BY execution_order NULLS LAST, priority ASC, created_at ASC, key ASC
	`, strings.Join(placeholders, ", "))

	return r.queryTasks(ctx, query, args...)
}

// FilterByMetadataPhase retrieves tasks filtered by workflow phase from metadata
// Uses status metadata to find statuses in the specified phase,
// then returns all tasks in those statuses
func (r *TaskRepository) FilterByMetadataPhase(ctx context.Context, phase string, workflow *config.WorkflowConfig) ([]*models.Task, error) {
	if workflow == nil {
		return []*models.Task{}, nil
	}

	// Get statuses in this phase
	statuses := workflow.GetStatusesByPhase(phase)
	if len(statuses) == 0 {
		// No statuses in this phase - return empty list
		return []*models.Task{}, nil
	}

	// Build SQL query with IN clause for multiple statuses
	placeholders := make([]string, len(statuses))
	args := make([]interface{}, len(statuses))
	for i, status := range statuses {
		placeholders[i] = "?"
		args[i] = status
	}

	query := fmt.Sprintf(`
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data, size
		FROM tasks
		WHERE status IN (%s)
		ORDER BY execution_order NULLS LAST, priority ASC, created_at ASC, key ASC
	`, strings.Join(placeholders, ", "))

	return r.queryTasks(ctx, query, args...)
}

// GetRejectionCounts returns rejection counts and last rejection timestamps for given tasks
// Uses efficient LEFT JOIN with COUNT aggregation to avoid N+1 queries
// Returns two maps: taskID -> rejection count and taskID -> last rejection time
func (r *TaskRepository) GetRejectionCounts(ctx context.Context, taskIDs []int64) (map[int64]int, map[int64]*time.Time, error) {
	if len(taskIDs) == 0 {
		return make(map[int64]int), make(map[int64]*time.Time), nil
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(taskIDs))
	args := make([]interface{}, len(taskIDs))
	for i, id := range taskIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := `
		SELECT
			t.id,
			COALESCE(COUNT(tn.id), 0) as rejection_count,
			datetime(MAX(tn.created_at)) as last_rejection_at
		FROM tasks t
		LEFT JOIN entity_notes tn ON t.id = tn.entity_id AND tn.entity_type = 'task' AND tn.note_type = 'rejection'
		WHERE t.id IN (` + strings.Join(placeholders, ",") + `)
		GROUP BY t.id
	`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query rejection counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[int64]int)
	lastTimes := make(map[int64]*time.Time)

	for rows.Next() {
		var taskID int64
		var rejectionCount int
		var lastRejectionAtStr sql.NullString

		if err := rows.Scan(&taskID, &rejectionCount, &lastRejectionAtStr); err != nil {
			return nil, nil, fmt.Errorf("failed to scan rejection counts: %w", err)
		}

		counts[taskID] = rejectionCount
		if lastRejectionAtStr.Valid {
			// Parse the timestamp string
			parsedTime, err := time.Parse("2006-01-02 15:04:05", lastRejectionAtStr.String)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to parse last_rejection_at: %w", err)
			}
			lastTimes[taskID] = &parsedTime
		}
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating rejection counts: %w", err)
	}

	// Ensure all requested task IDs are in the result maps (even if 0 rejections)
	for _, taskID := range taskIDs {
		if _, ok := counts[taskID]; !ok {
			counts[taskID] = 0
		}
	}

	return counts, lastTimes, nil
}

// GetTaskCountsForFeatures returns the total task count for each of the given feature IDs
// in a single batch query. This replaces N individual GetTaskCount calls with one query.
//
// Returns a map from featureID to count. Feature IDs with no tasks are not included in
// the map; callers should treat a missing key as a count of zero.
func (r *TaskRepository) GetTaskCountsForFeatures(ctx context.Context, featureIDs []int64) (map[int64]int, error) {
	if len(featureIDs) == 0 {
		return map[int64]int{}, nil
	}

	// Build parameterised IN clause
	placeholders := make([]string, len(featureIDs))
	args := make([]interface{}, len(featureIDs))
	for i, id := range featureIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(
		`SELECT feature_id, COUNT(*) FROM tasks WHERE feature_id IN (%s) GROUP BY feature_id`,
		strings.Join(placeholders, ", "),
	)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get task counts for features: %w", err)
	}
	defer rows.Close()

	counts := make(map[int64]int, len(featureIDs))
	for rows.Next() {
		var featureID int64
		var count int
		if err := rows.Scan(&featureID, &count); err != nil {
			return nil, fmt.Errorf("failed to scan task count row: %w", err)
		}
		counts[featureID] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating task counts: %w", err)
	}

	return counts, nil
}

// FeatureIDsForTaskIDs returns the distinct feature IDs that contain any of the
// given task IDs. Used by the viewer hierarchy endpoint (B017) to support
// tag-based filtering: when the hierarchy no longer carries task data, we still
// need to know which features have matching tagged tasks so we can keep those
// features in the pruned tree.
//
// Returns an empty map when taskIDs is empty.
func (r *TaskRepository) FeatureIDsForTaskIDs(ctx context.Context, taskIDs []int64) (map[int64]struct{}, error) {
	if len(taskIDs) == 0 {
		return map[int64]struct{}{}, nil
	}

	placeholders := make([]string, len(taskIDs))
	args := make([]interface{}, len(taskIDs))
	for i, id := range taskIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(
		`SELECT DISTINCT feature_id FROM tasks WHERE id IN (%s)`,
		strings.Join(placeholders, ", "),
	)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query feature IDs for task IDs: %w", err)
	}
	defer rows.Close()

	result := make(map[int64]struct{}, len(taskIDs))
	for rows.Next() {
		var featureID int64
		if err := rows.Scan(&featureID); err != nil {
			return nil, fmt.Errorf("failed to scan feature ID: %w", err)
		}
		result[featureID] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating feature IDs for task IDs: %w", err)
	}

	return result, nil
}

// FeatureTaskCounts holds the aggregate counts of tasks for a single feature.
// Used by viewer hierarchy endpoint to satisfy lazy-load contract (B017,
// E27-F02 REQ-F-002) — full task rows are never loaded.
type FeatureTaskCounts struct {
	Total   int
	Blocked int
}

// CountsByFeature returns total task count and blocked task count per feature
// in a single aggregate query — no full task rows are loaded. Used by the
// viewer hierarchy endpoint (B017) to avoid embedding task data in the payload.
//
// A task is considered "blocked" when its blocked_reason column is non-NULL.
// Features with zero tasks are omitted from the map; callers should treat a
// missing key as {Total: 0, Blocked: 0}.
func (r *TaskRepository) CountsByFeature(ctx context.Context) (map[int64]FeatureTaskCounts, error) {
	query := `
		SELECT feature_id,
		       COUNT(*) AS total,
		       SUM(CASE WHEN blocked_reason IS NOT NULL THEN 1 ELSE 0 END) AS blocked
		FROM tasks
		GROUP BY feature_id
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query task counts by feature: %w", err)
	}
	defer rows.Close()

	result := make(map[int64]FeatureTaskCounts)
	for rows.Next() {
		var featureID int64
		var total int
		var blocked int
		if err := rows.Scan(&featureID, &total, &blocked); err != nil {
			return nil, fmt.Errorf("failed to scan task counts by feature row: %w", err)
		}
		result[featureID] = FeatureTaskCounts{Total: total, Blocked: blocked}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating task counts by feature: %w", err)
	}

	return result, nil
}

// TaskDisplayDataRaw holds the raw JSON strings from the task_display_data view.
// The service layer is responsible for unmarshaling these into domain types.
type TaskDisplayDataRaw struct {
	BlockedByJSON    string
	BlocksJSON       string
	DependenciesJSON string
	DocumentsJSON    string
	NotesJSON        string
}

// GetTaskDisplayDataRaw fetches all display data for a task in a single query
// using the task_display_data view. Returns raw JSON strings that the service
// layer unmarshals into domain types.
func (r *TaskRepository) GetTaskDisplayDataRaw(ctx context.Context, taskID int64) (*TaskDisplayDataRaw, error) {
	query := `SELECT blocked_by_json, blocks_json, dependencies_json, documents_json, notes_json
		FROM task_display_data WHERE id = ?`

	raw := &TaskDisplayDataRaw{}
	err := r.db.QueryRowContext(ctx, query, taskID).Scan(
		&raw.BlockedByJSON,
		&raw.BlocksJSON,
		&raw.DependenciesJSON,
		&raw.DocumentsJSON,
		&raw.NotesJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query task_display_data for task %d: %w", taskID, err)
	}

	return raw, nil
}

// GetContextData retrieves the context data JSON string for a task by its ID.
func (r *TaskRepository) GetContextData(ctx context.Context, taskID int64) (*string, error) {
	query := `SELECT context_data FROM tasks WHERE id = ?`
	var contextData *string
	err := r.db.QueryRowContext(ctx, query, taskID).Scan(&contextData)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found with id %d", taskID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task context data: %w", err)
	}
	return contextData, nil
}

// CountByStatus returns task counts grouped by status.
func (r *TaskRepository) CountByStatus(ctx context.Context) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM tasks GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("failed to count tasks by status: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan task status count: %w", err)
		}
		counts[status] = count
	}
	return counts, rows.Err()
}

// CountBlocked returns the number of tasks that have a non-null blocked_reason.
func (r *TaskRepository) CountBlocked(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks WHERE blocked_reason IS NOT NULL`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count blocked tasks: %w", err)
	}
	return count, nil
}

// GetRecent returns the most recently created tasks, ordered by created_at DESC.
// limit must be positive; the caller (service) is responsible for bounds-checking.
// Returns an empty (non-nil) slice if no rows exist.
func (r *TaskRepository) GetRecent(ctx context.Context, limit int) (_ []*models.Task, retErr error) {
	ctx, span := tracer.Start(ctx, "TaskRepository.GetRecent",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "SELECT"),
			attribute.String("db.table", "tasks"),
			attribute.Int("db.limit", limit),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	query := `
		SELECT id, feature_id, key, title, slug, description, status, agent_type, priority,
		       depends_on, assigned_agent, file_path, blocked_reason, execution_order,
		       created_at, started_at, completed_at, blocked_at, updated_at,
		       completed_by, completion_notes, files_changed, tests_passed,
		       verification_status, time_spent_minutes, context_data, size
		FROM tasks
		ORDER BY created_at DESC
		LIMIT ?
	`

	tasks, err := r.queryTasks(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent tasks: %w", err)
	}

	if tasks == nil {
		tasks = []*models.Task{}
	}

	return tasks, nil
}

// UpdateContextData updates only the context_data field of a task.
func (r *TaskRepository) UpdateContextData(ctx context.Context, taskID int64, contextData *string) error {
	query := `UPDATE tasks SET context_data = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, contextData, taskID)
	if err != nil {
		return fmt.Errorf("failed to update task context data: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("task not found with id %d", taskID)
	}
	return nil
}
