package sprint

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
)

// SprintRepository handles CRUD operations for sprints.
type SprintRepository struct {
	db *dbconn.DB
}

// NewSprintRepository creates a new SprintRepository.
func NewSprintRepository(db *dbconn.DB) *SprintRepository {
	return &SprintRepository{db: db}
}

// sprintSelectColumns is the ordered list of columns for scanning a Sprint row.
const sprintSelectColumns = `id, key, name, goal, start_date, end_date, status, slug, file_path, created_at, updated_at`

// scanSprint scans a single Sprint row from the given scanner.
func scanSprint(scanner interface {
	Scan(dest ...interface{}) error
}) (*models.Sprint, error) {
	sprint := &models.Sprint{}
	err := scanner.Scan(
		&sprint.ID,
		&sprint.Key,
		&sprint.Name,
		&sprint.Goal,
		&sprint.StartDate,
		&sprint.EndDate,
		&sprint.Status,
		&sprint.Slug,
		&sprint.FilePath,
		&sprint.CreatedAt,
		&sprint.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return sprint, nil
}

// Create creates a new sprint record.
func (r *SprintRepository) Create(ctx context.Context, sprint *models.Sprint) error {
	if err := sprint.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		INSERT INTO sprints (
			key, name, goal, start_date, end_date, status, slug, file_path
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		sprint.Key,
		sprint.Name,
		sprint.Goal,
		sprint.StartDate,
		sprint.EndDate,
		sprint.Status,
		sprint.Slug,
		sprint.FilePath,
	)
	if err != nil {
		return fmt.Errorf("failed to create sprint: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	sprint.ID = id
	return nil
}

// GetByKey retrieves a sprint by its key (case-insensitive).
func (r *SprintRepository) GetByKey(ctx context.Context, key string) (*models.Sprint, error) {
	query := fmt.Sprintf(`SELECT %s FROM sprints WHERE UPPER(key) = UPPER(?)`, sprintSelectColumns)

	sprint, err := scanSprint(r.db.QueryRowContext(ctx, query, key))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("sprint not found with key %q", key)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get sprint: %w", err)
	}

	return sprint, nil
}

// GetByID retrieves a sprint by its database ID.
func (r *SprintRepository) GetByID(ctx context.Context, id int64) (*models.Sprint, error) {
	query := fmt.Sprintf(`SELECT %s FROM sprints WHERE id = ?`, sprintSelectColumns)

	sprint, err := scanSprint(r.db.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("sprint not found with id %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get sprint: %w", err)
	}

	return sprint, nil
}

// Update updates an existing sprint record.
func (r *SprintRepository) Update(ctx context.Context, sprint *models.Sprint) error {
	if err := sprint.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		UPDATE sprints
		SET name = ?, goal = ?, start_date = ?, end_date = ?,
			status = ?, slug = ?, file_path = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		sprint.Name,
		sprint.Goal,
		sprint.StartDate,
		sprint.EndDate,
		sprint.Status,
		sprint.Slug,
		sprint.FilePath,
		sprint.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update sprint: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("sprint not found with id %d", sprint.ID)
	}

	return nil
}

// Delete deletes a sprint by its database ID.
func (r *SprintRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM sprints WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete sprint: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("sprint not found with id %d", id)
	}

	return nil
}

// UpdateStatus updates only the status field of a sprint (atomic operation).
func (r *SprintRepository) UpdateStatus(ctx context.Context, id int64, status models.SprintStatus) error {
	query := `UPDATE sprints SET status = ? WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update sprint status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("sprint not found with id %d", id)
	}

	return nil
}

// GetNextKey returns the next available sprint key (e.g., S001, S002, ...).
func (r *SprintRepository) GetNextKey(ctx context.Context) (string, error) {
	query := `SELECT COALESCE(MAX(CAST(SUBSTR(key, 2) AS INTEGER)), 0) FROM sprints`

	var maxNum int
	err := r.db.QueryRowContext(ctx, query).Scan(&maxNum)
	if err != nil {
		return "", fmt.Errorf("failed to get next sprint key: %w", err)
	}

	nextKey := fmt.Sprintf("S%03d", maxNum+1)
	return nextKey, nil
}

// SprintListFilters defines filter options for listing sprints.
type SprintListFilters struct {
	Status *models.SprintStatus
}

// List retrieves all sprints, optionally filtered.
func (r *SprintRepository) List(ctx context.Context, filters *SprintListFilters) ([]*models.Sprint, error) {
	query := fmt.Sprintf(`SELECT %s FROM sprints`, sprintSelectColumns)

	var conditions []string
	var args []interface{}

	if filters != nil {
		if filters.Status != nil {
			conditions = append(conditions, "status = ?")
			args = append(args, *filters.Status)
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list sprints: %w", err)
	}
	defer rows.Close()

	var sprints []*models.Sprint
	for rows.Next() {
		sprint, err := scanSprint(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sprint: %w", err)
		}
		sprints = append(sprints, sprint)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sprints: %w", err)
	}

	return sprints, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Assignment CRUD methods (T-E19-F03-002)
//
// All methods follow the parameterized-query pattern established above.
// Entity-type validation is performed at the app layer via
// models.ValidateSprintAssignmentEntityType before any DB write — this
// matches the post-B018 convention (no CHECK constraint on entity_type).
//
// When adding a new assignable entity type in the future, update only
// models.ValidateSprintAssignmentEntityType — no DB migration is needed.
// ─────────────────────────────────────────────────────────────────────────────

// scanAssignment scans a single sprint_assignments row from the given scanner.
func scanAssignment(scanner interface {
	Scan(dest ...interface{}) error
}) (*models.SprintAssignment, error) {
	a := &models.SprintAssignment{}
	return a, scanner.Scan(
		&a.ID,
		&a.SprintID,
		&a.EntityType,
		&a.EntityID,
		&a.AssignedAt,
		&a.RemovedAt,
	)
}

// AddAssignment creates a sprint_assignments row for the given entity.
//
// Validates entity_type against the allowlist before inserting.
// Returns an error if a duplicate active assignment exists for the same entity
// (detected by the partial unique index idx_sprint_assignments_active_one).
// The caller should inspect the error message to identify the conflicting sprint
// when implementing user-facing messages.
func (r *SprintRepository) AddAssignment(ctx context.Context, assignment *models.SprintAssignment) error {
	if err := assignment.Validate(); err != nil {
		return fmt.Errorf("invalid assignment: %w", err)
	}

	query := `
		INSERT INTO sprint_assignments (sprint_id, entity_type, entity_id, assigned_at)
		VALUES (?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		assignment.SprintID,
		assignment.EntityType,
		assignment.EntityID,
		assignment.AssignedAt,
	)
	if err != nil {
		// Surface the partial unique index violation as a recognisable conflict.
		// SQLite returns "UNIQUE constraint failed" in the error message when the
		// partial index idx_sprint_assignments_active_one fires.
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "unique") {
			return fmt.Errorf("entity %s/%d is already actively assigned to a sprint (conflict): %w",
				assignment.EntityType, assignment.EntityID, err)
		}
		return fmt.Errorf("failed to add sprint assignment: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id for sprint assignment: %w", err)
	}

	assignment.ID = id
	return nil
}

// RemoveAssignment soft-deletes an active assignment by setting removed_at = NOW().
//
// Returns an error if no active assignment exists for (sprint_id, entity_type, entity_id).
// Soft-delete preserves velocity history: completed entities remain queryable by E19-F04.
func (r *SprintRepository) RemoveAssignment(ctx context.Context, sprintID int64, entityType string, entityID int64) error {
	query := `
		UPDATE sprint_assignments
		SET removed_at = CURRENT_TIMESTAMP
		WHERE sprint_id = ?
		  AND entity_type = ?
		  AND entity_id = ?
		  AND removed_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, sprintID, entityType, entityID)
	if err != nil {
		return fmt.Errorf("failed to remove sprint assignment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected for remove assignment: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no active assignment found for sprint_id=%d entity_type=%q entity_id=%d",
			sprintID, entityType, entityID)
	}

	return nil
}

// GetActiveAssignment returns the active assignment for an entity, or nil if none.
//
// "Active" means removed_at IS NULL — the partial unique index ensures at most one
// active assignment exists per (entity_type, entity_id) pair.
func (r *SprintRepository) GetActiveAssignment(ctx context.Context, entityType string, entityID int64) (*models.SprintAssignment, error) {
	query := `
		SELECT id, sprint_id, entity_type, entity_id, assigned_at, removed_at
		FROM sprint_assignments
		WHERE entity_type = ?
		  AND entity_id = ?
		  AND removed_at IS NULL
	`

	a, err := scanAssignment(r.db.QueryRowContext(ctx, query, entityType, entityID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil // No active assignment — not an error
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active assignment for %s/%d: %w", entityType, entityID, err)
	}

	return a, nil
}

// ListAssignments returns all active assignments for a sprint, optionally filtered by entity_type.
//
// When entityType is non-nil only rows matching that type are returned.
// Rows with removed_at set (soft-deleted) are excluded.
func (r *SprintRepository) ListAssignments(ctx context.Context, sprintID int64, entityType *string) ([]*models.SprintAssignment, error) {
	query := `
		SELECT id, sprint_id, entity_type, entity_id, assigned_at, removed_at
		FROM sprint_assignments
		WHERE sprint_id = ?
		  AND removed_at IS NULL
	`

	args := []interface{}{sprintID}
	if entityType != nil {
		query += " AND entity_type = ?"
		args = append(args, *entityType)
	}

	query += " ORDER BY assigned_at ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list sprint assignments for sprint %d: %w", sprintID, err)
	}
	defer rows.Close()

	var assignments []*models.SprintAssignment
	for rows.Next() {
		a, err := scanAssignment(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sprint assignment: %w", err)
		}
		assignments = append(assignments, a)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sprint assignments: %w", err)
	}

	return assignments, nil
}

// Entity ID resolution helpers — T-E19-F03-005
//
// Each method queries its entity's own table using a parameterized key lookup.
// Using UPPER(key) = UPPER(?) for case-insensitive matching consistent with the
// rest of the codebase (see task_repository.go, bug_repository.go patterns).
// Separate methods per entity type keep all queries static (no table-name
// interpolation), satisfying the SQL injection prevention mandate.

// GetTaskIDByKey returns the internal ID of a task given its key.
// Returns a not-found error if no task with that key exists.
func (r *SprintRepository) GetTaskIDByKey(ctx context.Context, key string) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		`SELECT id FROM tasks WHERE UPPER(key) = UPPER(?)`, key,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("task %q not found", key)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to look up task %q: %w", key, err)
	}
	return id, nil
}

// GetBugIDByKey returns the internal ID of a bug given its key.
// Returns a not-found error if no bug with that key exists.
func (r *SprintRepository) GetBugIDByKey(ctx context.Context, key string) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		`SELECT id FROM bugs WHERE UPPER(key) = UPPER(?)`, key,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("bug %q not found", key)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to look up bug %q: %w", key, err)
	}
	return id, nil
}

// GetChangeCardIDByKey returns the internal ID of a change-card given its key.
// Returns a not-found error if no change-card with that key exists.
func (r *SprintRepository) GetChangeCardIDByKey(ctx context.Context, key string) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		`SELECT id FROM change_cards WHERE UPPER(key) = UPPER(?)`, key,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("change-card %q not found", key)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to look up change-card %q: %w", key, err)
	}
	return id, nil
}

// GetTechDebtIDByKey returns the internal ID of a tech-debt item given its key.
// Returns a not-found error if no tech-debt with that key exists.
func (r *SprintRepository) GetTechDebtIDByKey(ctx context.Context, key string) (int64, error) {
	// Tech-debt keys may be passed as "TD-001" or with the normalized prefix.
	// Normalize to uppercase before lookup to support case-insensitive input.
	upperKey := strings.ToUpper(strings.TrimSpace(key))
	var id int64
	err := r.db.QueryRowContext(ctx,
		`SELECT id FROM tech_debts WHERE UPPER(key) = ?`, upperKey,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("tech-debt %q not found", key)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to look up tech-debt %q: %w", key, err)
	}
	return id, nil
}
