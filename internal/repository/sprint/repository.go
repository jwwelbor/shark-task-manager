package sprint

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	repoerr "github.com/jwwelbor/shark-task-manager/internal/repository/repoerr"
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
		flexTime{&sprint.StartDate},
		flexTime{&sprint.EndDate},
		&sprint.Status,
		&sprint.Slug,
		&sprint.FilePath,
		flexTime{&sprint.CreatedAt},
		flexTime{&sprint.UpdatedAt},
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
		return nil, fmt.Errorf("sprint not found with key %q: %w", key, repoerr.ErrNotFound)
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
		return nil, fmt.Errorf("sprint not found with id %d: %w", id, repoerr.ErrNotFound)
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
		return fmt.Errorf("sprint not found with id %d: %w", sprint.ID, repoerr.ErrNotFound)
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
		return fmt.Errorf("sprint not found with id %d: %w", id, repoerr.ErrNotFound)
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
		return fmt.Errorf("sprint not found with id %d: %w", id, repoerr.ErrNotFound)
	}

	return nil
}

// UpdateStatusTx updates the sprint status within a caller-supplied transaction.
// Used by CloseSprintWithCarryover to atomically advance sprint status.
func (r *SprintRepository) UpdateStatusTx(ctx context.Context, tx *sql.Tx, id int64, status models.SprintStatus) error {
	query := `UPDATE sprints SET status = ? WHERE id = ?`

	result, err := tx.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update sprint status in transaction: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("sprint not found with id %d: %w", id, repoerr.ErrNotFound)
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

// ---------------------------------------------------------------------------
// BacklogItem projection type
// ---------------------------------------------------------------------------

// BacklogItem is a read-only projection returned by ListBacklog. It is NOT a
// stored model — it is assembled at query time from sprint_assignments joined
// to the four entity tables (tasks, bugs, change_cards, tech_debts) via a
// UNION ALL query.
//
// AgentType and Priority are present only for entities that carry those
// columns (currently only tasks have AgentType; bugs and tech_debts have
// neither). For entities without these columns the UNION query returns NULL,
// which maps to the zero value of the respective Go type.
// BacklogItem is a read-only projection returned by ListBacklog (assigned items)
// and ListUnassignedBacklog (unassigned items).
//
// Key and EntityKey carry the same value — both field names are supported for
// backward compatibility (EntityKey was introduced in T-E19-F03; Key was added
// in T-E19-F05 to match the service-layer interface spec).
type BacklogItem struct {
	AssignmentID   int64     `json:"assignment_id,omitempty"`
	SprintID       int64     `json:"sprint_id,omitempty"`
	EntityType     string    `json:"entity_type"`
	EntityID       int64     `json:"entity_id"`
	EntityKey      string    `json:"entity_key"`
	Key            string    `json:"key"` // alias for EntityKey; populated from the same DB column
	Title          string    `json:"title"`
	Status         string    `json:"status,omitempty"`
	AgentType      *string   `json:"agent_type,omitempty"`
	Priority       int       `json:"priority,omitempty"`
	ExecutionOrder *int      `json:"execution_order,omitempty"`
	Size           *int      `json:"size,omitempty"`
	AssignedAt     time.Time `json:"assigned_at,omitempty"`
}

// ---------------------------------------------------------------------------
// Assignment CRUD methods (T-E19-F03-002 scope — implemented here to support
// the UNION query tests and Tx method tests in this package).
// ---------------------------------------------------------------------------

// AddAssignment inserts a sprint_assignments row for the given entity.
// Returns an error if the entity already has an active assignment (partial
// unique index idx_sprint_assignments_active_one fires at DB level).
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
		time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("failed to add assignment: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	assignment.ID = id
	return nil
}

// RemoveAssignment soft-deletes an active assignment by setting removed_at to now.
// Returns an error if no active assignment exists.
func (r *SprintRepository) RemoveAssignment(ctx context.Context, sprintID int64, entityType string, entityID int64) error {
	query := `
		UPDATE sprint_assignments
		SET removed_at = ?
		WHERE sprint_id = ? AND entity_type = ? AND entity_id = ? AND removed_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query, time.Now().UTC(), sprintID, entityType, entityID)
	if err != nil {
		return fmt.Errorf("failed to remove assignment: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("no active assignment found for entity_type=%q entity_id=%d in sprint %d",
			entityType, entityID, sprintID)
	}
	return nil
}

// GetActiveAssignment returns the active (non-removed) assignment for an entity,
// or nil if no active assignment exists.
func (r *SprintRepository) GetActiveAssignment(ctx context.Context, entityType string, entityID int64) (*models.SprintAssignment, error) {
	query := `
		SELECT id, sprint_id, entity_type, entity_id, assigned_at, removed_at
		FROM sprint_assignments
		WHERE entity_type = ? AND entity_id = ? AND removed_at IS NULL
		LIMIT 1
	`
	row := r.db.QueryRowContext(ctx, query, entityType, entityID)
	a := &models.SprintAssignment{}
	err := row.Scan(&a.ID, &a.SprintID, &a.EntityType, &a.EntityID, flexTime{&a.AssignedAt}, flexNullTime{&a.RemovedAt})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active assignment: %w", err)
	}
	return a, nil
}

// ListAssignments returns all active (non-removed) assignments for a sprint.
// If entityType is non-nil, results are limited to that entity type.
func (r *SprintRepository) ListAssignments(ctx context.Context, sprintID int64, entityType *string) ([]*models.SprintAssignment, error) {
	query := `
		SELECT id, sprint_id, entity_type, entity_id, assigned_at, removed_at
		FROM sprint_assignments
		WHERE sprint_id = ? AND removed_at IS NULL
	`
	args := []interface{}{sprintID}
	if entityType != nil {
		query += " AND entity_type = ?"
		args = append(args, *entityType)
	}
	query += " ORDER BY assigned_at"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list assignments: %w", err)
	}
	defer rows.Close()

	var assignments []*models.SprintAssignment
	for rows.Next() {
		a := &models.SprintAssignment{}
		if err := rows.Scan(&a.ID, &a.SprintID, &a.EntityType, &a.EntityID, flexTime{&a.AssignedAt}, flexNullTime{&a.RemovedAt}); err != nil {
			return nil, fmt.Errorf("failed to scan assignment: %w", err)
		}
		assignments = append(assignments, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating assignments: %w", err)
	}
	return assignments, nil
}

// ---------------------------------------------------------------------------
// Backlog query (T-E19-F03-003 scope)
// ---------------------------------------------------------------------------

// ListBacklog returns BacklogItem rows for all active assignments in a sprint.
//
// The query is a UNION ALL of four static sub-selects — one per entity type
// (bug, change_card, task, tech_debt, listed in lexicographic order). Static
// SQL with parameterized sprint_id bindings is used throughout; dynamic table
// names via string interpolation are intentionally avoided to prevent SQL
// injection (see spec §4.1.3 rationale and .claude/rules/go/input-sanitization.md).
//
// Extension point: to add a fifth entity type, add a new sub-select block
// below following the same pattern (sa.id, sa.sprint_id, '<type>' AS entity_type,
// sa.entity_id, <table>.key, <table>.title, <table>.status, <agent_type_or_null>,
// <priority_or_null>, <size_or_null>, sa.assigned_at).
//
// If entityType is non-nil, only the matching sub-select is executed (the
// other three are replaced by SELECT with no matching rows). When blockedOnly
// is true, the caller is expected to have set relevant status filters — the
// repository receives a pre-computed list of blocked status values passed via
// the blockedStatuses variadic argument.
func (r *SprintRepository) ListBacklog(ctx context.Context, sprintID int64, entityType *string, blockedOnly bool, blockedStatuses ...string) ([]*BacklogItem, error) {
	// Build the UNION ALL of four sub-selects. Each sub-select contributes the
	// entity_type literal as a constant column so callers never need to infer
	// the type from the key format.
	//
	// Columns returned per row:
	//   sa.id, sa.sprint_id, entity_type, sa.entity_id,
	//   <table>.key, <table>.title, <table>.status,
	//   <agent_type or NULL>, <priority or NULL>, <size or NULL>,
	//   sa.assigned_at

	// Helper to build a single sub-select. tableName, typeLabel, agentTypeExpr,
	// priorityExpr are all static strings from this function — no user input.
	type subSelect struct {
		typeLabel     string
		tableName     string
		agentTypeExpr string // SQL expression for agent_type column
		priorityExpr  string // SQL expression for priority column
		execOrderExpr string // SQL expression for execution_order column
	}

	// Defined in lexicographic order per spec recommendation.
	subSelects := []subSelect{
		{"bug", "bugs", "NULL", "NULL", "NULL"},
		{"change_card", "change_cards", "NULL", "cc.priority", "NULL"},
		{"task", "tasks", "t.agent_type", "t.priority", "t.execution_order"},
		{"tech_debt", "tech_debts", "NULL", "NULL", "NULL"},
	}

	// Table aliases used in the sub-selects above
	tableAliases := map[string]string{
		"bugs":         "b",
		"change_cards": "cc",
		"tasks":        "t",
		"tech_debts":   "td",
	}

	// Build blocked-status IN clause if needed
	var blockedClause string
	if blockedOnly && len(blockedStatuses) > 0 {
		placeholders := make([]string, len(blockedStatuses))
		for i := range blockedStatuses {
			placeholders[i] = "?"
		}
		// blockedClause is constructed from a static template; the values are
		// passed as bound parameters in the args slice.
		blockedClause = fmt.Sprintf("AND %%s.status IN (%s)", strings.Join(placeholders, ","))
	}

	var parts []string
	var args []interface{}

	for _, ss := range subSelects {
		// Skip sub-selects for non-matching entity types when a filter is set
		if entityType != nil && *entityType != ss.typeLabel {
			continue
		}

		alias := tableAliases[ss.tableName]
		var statusFilter string
		if blockedOnly && len(blockedStatuses) > 0 {
			statusFilter = fmt.Sprintf(blockedClause, alias)
		}

		//nolint:gosec // tableName and alias come from the hardcoded subSelects slice; no user input
		part := fmt.Sprintf(`
			SELECT sa.id, sa.sprint_id, '%s' AS entity_type, sa.entity_id,
			       %s.key, %s.title, %s.status,
			       %s AS agent_type, %s AS priority, %s AS execution_order, %s.size, sa.assigned_at
			FROM sprint_assignments sa
			JOIN %s %s ON %s.id = sa.entity_id
			WHERE sa.sprint_id = ? AND sa.removed_at IS NULL AND sa.entity_type = '%s'%s`,
			ss.typeLabel,
			alias, alias, alias,
			ss.agentTypeExpr, ss.priorityExpr, ss.execOrderExpr, alias,
			ss.tableName, alias, alias,
			ss.typeLabel,
			statusFilter,
		)
		parts = append(parts, part)
		// sprintID must come before blockedStatuses to match the WHERE clause
		// placeholder order: WHERE sa.sprint_id = ? ... AND alias.status IN (?, ...)
		args = append(args, sprintID)
		if blockedOnly && len(blockedStatuses) > 0 {
			for _, s := range blockedStatuses {
				args = append(args, s)
			}
		}
	}

	if len(parts) == 0 {
		// No matching sub-selects (e.g., unsupported entityType filter)
		return []*BacklogItem{}, nil
	}

	query := strings.Join(parts, "\n\nUNION ALL\n\n")
	query += "\n\nORDER BY entity_type, priority DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query backlog: %w", err)
	}
	defer rows.Close()

	var items []*BacklogItem
	for rows.Next() {
		item := &BacklogItem{}
		var agentType sql.NullString
		var priority sql.NullInt64
		var execOrder sql.NullInt64
		var size sql.NullInt64
		if err := rows.Scan(
			&item.AssignmentID,
			&item.SprintID,
			&item.EntityType,
			&item.EntityID,
			&item.EntityKey,
			&item.Title,
			&item.Status,
			&agentType,
			&priority,
			&execOrder,
			&size,
			&item.AssignedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan backlog item: %w", err)
		}
		item.Key = item.EntityKey // keep both fields in sync
		if agentType.Valid {
			s := agentType.String
			item.AgentType = &s
		}
		if priority.Valid {
			item.Priority = int(priority.Int64)
		}
		if execOrder.Valid {
			val := int(execOrder.Int64)
			item.ExecutionOrder = &val
		}
		if size.Valid {
			s := int(size.Int64)
			item.Size = &s
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating backlog items: %w", err)
	}
	return items, nil
}

// ListAssignmentsForCarryover returns all active assignments for a sprint where
// the entity is NOT in a "completed" status. This is used exclusively by
// CloseSprintWithCarryover in the service layer to identify work that has not
// finished and must be moved or dropped.
//
// The implementation uses a UNION ALL across all four entity tables to resolve
// entity status at query time, following the same static-SQL pattern as
// ListBacklog (no dynamic table names, all parameterized).
//
// Entities in any status NOT in the completedStatuses set are returned.
// If completedStatuses is empty, ALL active assignments are returned (the
// caller is responsible for passing a meaningful set).
func (r *SprintRepository) ListAssignmentsForCarryover(ctx context.Context, sprintID int64, completedStatuses ...string) ([]*models.SprintAssignment, error) {
	// Default completed statuses when none are provided. This ensures that
	// entities marked "completed" are always excluded from the carryover
	// list even when the caller does not explicitly specify which statuses
	// represent terminal/completed work. The service layer may pass a
	// workflow-derived list for non-default workflows.
	if len(completedStatuses) == 0 {
		completedStatuses = []string{"completed"}
	}

	// Build NOT IN clause for completed statuses
	var completedFilter string
	var filterArgs []interface{}
	if len(completedStatuses) > 0 {
		placeholders := make([]string, len(completedStatuses))
		for i, s := range completedStatuses {
			placeholders[i] = "?"
			filterArgs = append(filterArgs, s)
		}
		completedFilter = fmt.Sprintf(" AND %%s.status NOT IN (%s)", strings.Join(placeholders, ","))
	}

	// Build UNION ALL across four entity tables.
	// Each sub-select returns (sa.id, sa.sprint_id, entity_type, sa.entity_id)
	// filtered to exclude completed statuses.
	type tableEntry struct {
		typeLabel string
		tableName string
		alias     string
	}
	tables := []tableEntry{
		{"bug", "bugs", "b"},
		{"change_card", "change_cards", "cc"},
		{"task", "tasks", "t"},
		{"tech_debt", "tech_debts", "td"},
	}

	var parts []string
	var args []interface{}

	for _, tbl := range tables {
		var statusExclude string
		if completedFilter != "" {
			statusExclude = fmt.Sprintf(completedFilter, tbl.alias)
		}
		//nolint:gosec // tbl fields come from the hardcoded tables slice; no user input
		part := fmt.Sprintf(`
			SELECT sa.id, sa.sprint_id, '%s' AS entity_type, sa.entity_id, sa.assigned_at
			FROM sprint_assignments sa
			JOIN %s %s ON %s.id = sa.entity_id
			WHERE sa.sprint_id = ? AND sa.removed_at IS NULL AND sa.entity_type = '%s'%s`,
			tbl.typeLabel,
			tbl.tableName, tbl.alias, tbl.alias,
			tbl.typeLabel,
			statusExclude,
		)
		parts = append(parts, part)
		// sprintID must come before filterArgs to match the ? placeholder order:
		// WHERE sa.sprint_id = ? ... AND alias.status NOT IN (?, ...)
		args = append(args, sprintID)
		args = append(args, filterArgs...)
	}

	query := strings.Join(parts, "\n\nUNION ALL\n\n")

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query assignments for carryover: %w", err)
	}
	defer rows.Close()

	var assignments []*models.SprintAssignment
	for rows.Next() {
		a := &models.SprintAssignment{}
		if err := rows.Scan(&a.ID, &a.SprintID, &a.EntityType, &a.EntityID, flexTime{&a.AssignedAt}); err != nil {
			return nil, fmt.Errorf("failed to scan carryover assignment: %w", err)
		}
		assignments = append(assignments, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating carryover assignments: %w", err)
	}
	return assignments, nil
}

// ---------------------------------------------------------------------------
// Entity ID resolution helpers (T-E19-F03-003 scope)
//
// Each method queries a single, statically-named table using UPPER(key) for
// case-insensitive lookup. Separate typed methods are used (instead of a
// single method with a switch on entity type) to avoid dynamic table names,
// which would require string interpolation and introduce SQL injection risk
// (see spec §4.1.4 rationale).
// ---------------------------------------------------------------------------

// GetTaskIDByKey resolves a task key to its database ID.
// Returns an error if the key does not exist.
func (r *SprintRepository) GetTaskIDByKey(ctx context.Context, key string) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		`SELECT id FROM tasks WHERE UPPER(key) = UPPER(?)`, key,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("task not found with key %q: %w", key, repoerr.ErrNotFound)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get task id by key: %w", err)
	}
	return id, nil
}

// GetBugIDByKey resolves a bug key to its database ID.
// Returns an error if the key does not exist.
func (r *SprintRepository) GetBugIDByKey(ctx context.Context, key string) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		`SELECT id FROM bugs WHERE UPPER(key) = UPPER(?)`, key,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("bug not found with key %q: %w", key, repoerr.ErrNotFound)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get bug id by key: %w", err)
	}
	return id, nil
}

// GetChangeCardIDByKey resolves a change-card key to its database ID.
// Returns an error if the key does not exist.
func (r *SprintRepository) GetChangeCardIDByKey(ctx context.Context, key string) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		`SELECT id FROM change_cards WHERE UPPER(key) = UPPER(?)`, key,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("change_card not found with key %q: %w", key, repoerr.ErrNotFound)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get change_card id by key: %w", err)
	}
	return id, nil
}

// GetTechDebtIDByKey resolves a tech-debt key to its database ID.
// Returns an error if the key does not exist.
func (r *SprintRepository) GetTechDebtIDByKey(ctx context.Context, key string) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		`SELECT id FROM tech_debts WHERE UPPER(key) = UPPER(?)`, key,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("tech_debt not found with key %q: %w", key, repoerr.ErrNotFound)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get tech_debt id by key: %w", err)
	}
	return id, nil
}

// ReassignToSprintTx updates the sprint_id on a set of active sprint_assignments
// rows to newSprintID. This is used by the carryover path in
// SprintService.CloseSprintWithCarryover to move incomplete work to the next
// sprint atomically.
//
// The method accepts *sql.Tx so that the service layer owns the transaction
// boundary (follows .claude/rules/go/patterns.md "Transaction ownership in
// service"). If assignmentIDs is empty, the call is a no-op and returns nil.
//
// All updates execute in a single IN (…) clause for efficiency; with SQLite WAL
// mode this is well within the <2s target for 200 assignments (REQ-NF-001).
func (r *SprintRepository) ReassignToSprintTx(ctx context.Context, tx *sql.Tx, assignmentIDs []int64, newSprintID int64) error {
	if len(assignmentIDs) == 0 {
		return nil
	}

	// Build parameterized IN clause — static placeholders, no string interpolation.
	placeholders := strings.Repeat("?,", len(assignmentIDs))
	placeholders = placeholders[:len(placeholders)-1] // trim trailing comma

	query := fmt.Sprintf(
		`UPDATE sprint_assignments SET sprint_id = ? WHERE id IN (%s)`,
		placeholders,
	)

	args := make([]interface{}, 0, 1+len(assignmentIDs))
	args = append(args, newSprintID)
	for _, id := range assignmentIDs {
		args = append(args, id)
	}

	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to reassign assignments to sprint %d: %w", newSprintID, err)
	}

	return nil
}

// DropAssignmentsTx soft-deletes a set of sprint_assignments by setting
// removed_at = NOW(). Used by the carryover-to-backlog path in
// SprintService.CloseSprintWithCarryover.
//
// Like ReassignToSprintTx, this method operates within the caller-supplied
// *sql.Tx so the service owns rollback/commit. An empty assignmentIDs slice
// is a no-op.
func (r *SprintRepository) DropAssignmentsTx(ctx context.Context, tx *sql.Tx, assignmentIDs []int64) error {
	if len(assignmentIDs) == 0 {
		return nil
	}

	placeholders := strings.Repeat("?,", len(assignmentIDs))
	placeholders = placeholders[:len(placeholders)-1]

	query := fmt.Sprintf(
		`UPDATE sprint_assignments SET removed_at = ? WHERE id IN (%s)`,
		placeholders,
	)

	args := make([]interface{}, 0, 1+len(assignmentIDs))
	args = append(args, time.Now().UTC())
	for _, id := range assignmentIDs {
		args = append(args, id)
	}

	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to drop assignments: %w", err)
	}

	return nil
}

// ─── E19-F05-002: Capacity CRUD and unassigned-backlog query methods ─────────

// AssignmentWithSize joins sprint_assignments with entity size data.
// Used by the service layer to compute capacity allocation and readiness scores.
// Title is included so the service can populate UnsizedEntities/OversizedEntities
// lists without an extra query. DependsOn is included for task dependency checking
// (Factor 2 of the readiness score). Non-task entities always have empty DependsOn.
type AssignmentWithSize struct {
	EntityType string
	EntityID   int64
	Key        string
	Title      string
	AgentType  *string
	Size       *int
	DependsOn  string // JSON array string from tasks.depends_on; empty for non-task entities
}

// GetCapacity returns all sprint_capacity rows for a sprint, ordered by
// agent_type. Returns an empty (non-nil) slice when no rows exist.
func (r *SprintRepository) GetCapacity(ctx context.Context, sprintID int64) ([]*models.SprintCapacity, error) {
	query := `
		SELECT id, sprint_id, agent_type, capacity_points, created_at, updated_at
		FROM sprint_capacity
		WHERE sprint_id = ?
		ORDER BY agent_type
	`
	rows, err := r.db.QueryContext(ctx, query, sprintID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sprint capacity: %w", err)
	}
	defer rows.Close()

	result := make([]*models.SprintCapacity, 0)
	for rows.Next() {
		c := &models.SprintCapacity{}
		if err := rows.Scan(&c.ID, &c.SprintID, &c.AgentType, &c.CapacityPoints, flexTime{&c.CreatedAt}, flexTime{&c.UpdatedAt}); err != nil {
			return nil, fmt.Errorf("failed to scan sprint capacity: %w", err)
		}
		result = append(result, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sprint capacity: %w", err)
	}
	return result, nil
}

// SetCapacity upserts a capacity row for (sprint_id, agent_type).
// Uses INSERT OR REPLACE semantics via ON CONFLICT DO UPDATE to ensure
// at most one row per (sprint_id, agent_type) pair.
func (r *SprintRepository) SetCapacity(ctx context.Context, c *models.SprintCapacity) error {
	query := `
		INSERT INTO sprint_capacity (sprint_id, agent_type, capacity_points)
		VALUES (?, ?, ?)
		ON CONFLICT(sprint_id, agent_type)
		DO UPDATE SET capacity_points = excluded.capacity_points,
		              updated_at = CURRENT_TIMESTAMP
	`
	result, err := r.db.ExecContext(ctx, query, c.SprintID, c.AgentType, c.CapacityPoints)
	if err != nil {
		return fmt.Errorf("failed to set sprint capacity: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id for sprint capacity: %w", err)
	}
	if id > 0 {
		c.ID = id
	}
	return nil
}

// BulkAssign inserts multiple sprint_assignments in a single transaction.
// Entities already actively assigned (removed_at IS NULL) are silently skipped.
// Returns the count of rows actually inserted.
func (r *SprintRepository) BulkAssign(ctx context.Context, sprintID int64, assignments []models.SprintAssignment) (int, error) {
	if len(assignments) == 0 {
		return 0, nil
	}

	tx, err := r.db.BeginTxContext(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction for bulk assign: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	inserted := 0
	for _, a := range assignments {
		// INSERT OR IGNORE skips rows that violate the partial unique index
		// idx_sprint_assignments_active_one (entity_type, entity_id) WHERE removed_at IS NULL.
		result, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO sprint_assignments (sprint_id, entity_type, entity_id, assigned_at)
			 VALUES (?, ?, ?, CURRENT_TIMESTAMP)`,
			sprintID, a.EntityType, a.EntityID,
		)
		if err != nil {
			return 0, fmt.Errorf("failed to insert sprint assignment for entity_type=%q entity_id=%d: %w",
				a.EntityType, a.EntityID, err)
		}
		n, err := result.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("failed to get rows affected for assignment: %w", err)
		}
		inserted += int(n)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit bulk assign transaction: %w", err)
	}

	return inserted, nil
}

// ListUnassignedBacklog returns entities eligible for sprint assignment that are
// not already in any active or planning sprint. Uses a NOT EXISTS correlated
// subquery (not N+1) to satisfy the 500ms performance budget at 500 entities.
//
// entityTypes filters to specific entity types (e.g., ["task", "bug"]).
// Supported values: "task", "bug", "change_card", "tech_debt".
//
// For tasks: excludes statuses "completed", "archived", "cancelled".
// For other entity types: no status filter (all non-archived items returned).
//
// Results are ordered by priority DESC, execution_order ASC NULLS LAST.
func (r *SprintRepository) ListUnassignedBacklog(ctx context.Context, entityTypes []string) ([]BacklogItem, error) {
	// Build a set of requested entity types for quick lookup.
	wantType := make(map[string]bool, len(entityTypes))
	for _, et := range entityTypes {
		wantType[et] = true
	}
	// Default: include all entity types when the slice is empty.
	includeAll := len(entityTypes) == 0

	var parts []string
	var args []interface{}

	// Tasks sub-select: excludes terminal statuses and entities in active/planning sprints.
	if includeAll || wantType["task"] {
		parts = append(parts, `
			SELECT 'task' AS entity_type,
			       t.id AS entity_id,
			       t.key,
			       t.title,
			       COALESCE(t.priority, 5) AS priority,
			       t.size,
			       t.agent_type,
			       t.execution_order
			FROM tasks t
			WHERE t.status NOT IN ('completed', 'archived', 'cancelled')
			  AND NOT EXISTS (
			      SELECT 1
			      FROM sprint_assignments sa
			      JOIN sprints s ON s.id = sa.sprint_id
			      WHERE sa.entity_type = 'task'
			        AND sa.entity_id = t.id
			        AND sa.removed_at IS NULL
			        AND s.status IN ('planning', 'active')
			  )`)
	}

	// Bugs sub-select.
	if includeAll || wantType["bug"] {
		parts = append(parts, `
			SELECT 'bug' AS entity_type,
			       b.id AS entity_id,
			       b.key,
			       b.title,
			       5 AS priority,
			       b.size,
			       NULL AS agent_type,
			       NULL AS execution_order
			FROM bugs b
			WHERE NOT EXISTS (
			      SELECT 1
			      FROM sprint_assignments sa
			      JOIN sprints s ON s.id = sa.sprint_id
			      WHERE sa.entity_type = 'bug'
			        AND sa.entity_id = b.id
			        AND sa.removed_at IS NULL
			        AND s.status IN ('planning', 'active')
			  )`)
	}

	// Change-cards sub-select.
	if includeAll || wantType["change_card"] {
		parts = append(parts, `
			SELECT 'change_card' AS entity_type,
			       cc.id AS entity_id,
			       cc.key,
			       cc.title,
			       COALESCE(cc.priority, 5) AS priority,
			       cc.size,
			       NULL AS agent_type,
			       NULL AS execution_order
			FROM change_cards cc
			WHERE NOT EXISTS (
			      SELECT 1
			      FROM sprint_assignments sa
			      JOIN sprints s ON s.id = sa.sprint_id
			      WHERE sa.entity_type = 'change_card'
			        AND sa.entity_id = cc.id
			        AND sa.removed_at IS NULL
			        AND s.status IN ('planning', 'active')
			  )`)
	}

	// Tech-debt sub-select.
	if includeAll || wantType["tech_debt"] {
		parts = append(parts, `
			SELECT 'tech_debt' AS entity_type,
			       td.id AS entity_id,
			       td.key,
			       td.title,
			       5 AS priority,
			       td.size,
			       NULL AS agent_type,
			       NULL AS execution_order
			FROM tech_debts td
			WHERE NOT EXISTS (
			      SELECT 1
			      FROM sprint_assignments sa
			      JOIN sprints s ON s.id = sa.sprint_id
			      WHERE sa.entity_type = 'tech_debt'
			        AND sa.entity_id = td.id
			        AND sa.removed_at IS NULL
			        AND s.status IN ('planning', 'active')
			  )`)
	}

	if len(parts) == 0 {
		return []BacklogItem{}, nil
	}

	query := strings.Join(parts, "\nUNION ALL\n")
	query += "\nORDER BY priority DESC, execution_order ASC NULLS LAST"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query unassigned backlog: %w", err)
	}
	defer rows.Close()

	items := make([]BacklogItem, 0)
	for rows.Next() {
		var item BacklogItem
		var agentType sql.NullString
		var size sql.NullInt64
		var execOrder sql.NullInt64
		if err := rows.Scan(
			&item.EntityType,
			&item.EntityID,
			&item.EntityKey,
			&item.Title,
			&item.Priority,
			&size,
			&agentType,
			&execOrder,
		); err != nil {
			return nil, fmt.Errorf("failed to scan unassigned backlog item: %w", err)
		}
		item.Key = item.EntityKey
		if agentType.Valid {
			s := agentType.String
			item.AgentType = &s
		}
		if size.Valid {
			s := int(size.Int64)
			item.Size = &s
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating unassigned backlog: %w", err)
	}
	return items, nil
}

// GetAssignmentsWithSize returns all active assignments for a sprint with size,
// agent_type, title, and depends_on data joined from the appropriate entity table.
// Uses a UNION ALL to join each entity type's table.
// Title and DependsOn are included so the service can build readiness score lists
// (UnsizedEntities, OversizedEntities) and evaluate Factor 2 (dependency satisfaction)
// without additional queries — all computation is then purely in-memory.
func (r *SprintRepository) GetAssignmentsWithSize(ctx context.Context, sprintID int64) ([]AssignmentWithSize, error) {
	// Columns: entity_type, entity_id, key, title, size, agent_type, depends_on
	query := `
		SELECT sa.entity_type, sa.entity_id, t.key, COALESCE(t.title,''), t.size, t.agent_type,
		       COALESCE(t.depends_on, '[]')
		FROM sprint_assignments sa
		JOIN tasks t ON sa.entity_id = t.id
		WHERE sa.sprint_id = ? AND sa.removed_at IS NULL AND sa.entity_type = 'task'

		UNION ALL

		SELECT sa.entity_type, sa.entity_id, b.key, COALESCE(b.title,''), b.size, NULL AS agent_type,
		       '[]' AS depends_on
		FROM sprint_assignments sa
		JOIN bugs b ON sa.entity_id = b.id
		WHERE sa.sprint_id = ? AND sa.removed_at IS NULL AND sa.entity_type = 'bug'

		UNION ALL

		SELECT sa.entity_type, sa.entity_id, cc.key, COALESCE(cc.title,''), cc.size, NULL AS agent_type,
		       '[]' AS depends_on
		FROM sprint_assignments sa
		JOIN change_cards cc ON sa.entity_id = cc.id
		WHERE sa.sprint_id = ? AND sa.removed_at IS NULL AND sa.entity_type = 'change_card'

		UNION ALL

		SELECT sa.entity_type, sa.entity_id, td.key, COALESCE(td.title,''), td.size, NULL AS agent_type,
		       '[]' AS depends_on
		FROM sprint_assignments sa
		JOIN tech_debts td ON sa.entity_id = td.id
		WHERE sa.sprint_id = ? AND sa.removed_at IS NULL AND sa.entity_type = 'tech_debt'
	`
	rows, err := r.db.QueryContext(ctx, query, sprintID, sprintID, sprintID, sprintID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignments with size: %w", err)
	}
	defer rows.Close()

	result := make([]AssignmentWithSize, 0)
	for rows.Next() {
		var a AssignmentWithSize
		var agentType sql.NullString
		var size sql.NullInt64
		if err := rows.Scan(&a.EntityType, &a.EntityID, &a.Key, &a.Title, &size, &agentType, &a.DependsOn); err != nil {
			return nil, fmt.Errorf("failed to scan assignment with size: %w", err)
		}
		if agentType.Valid {
			s := agentType.String
			a.AgentType = &s
		}
		if size.Valid {
			s := int(size.Int64)
			a.Size = &s
		}
		result = append(result, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating assignments with size: %w", err)
	}
	return result, nil
}

// ─── End E19-F05-002 ──────────────────────────────────────────────────────────

// CreateCompletionTx inserts a sprint_completions row within the provided
// transaction. On success, completion.ID is set to the newly inserted row's
// primary key.
//
// sprint_completions has a UNIQUE constraint on sprint_id (one record per
// sprint), so calling this twice for the same sprint will return an error.
// PlannedSizeSum, CompletedSizeSum, and NextSprintID are nullable fields — a
// nil pointer maps to SQL NULL.
func (r *SprintRepository) CreateCompletionTx(ctx context.Context, tx *sql.Tx, completion *models.SprintCompletion) error {
	query := `
		INSERT INTO sprint_completions (
			sprint_id,
			completed_at,
			planned_entity_count,
			completed_entity_count,
			carried_over_count,
			dropped_count,
			planned_size_sum,
			completed_size_sum,
			carryover_mode,
			next_sprint_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := tx.ExecContext(ctx, query,
		completion.SprintID,
		completion.CompletedAt,
		completion.PlannedEntityCount,
		completion.CompletedEntityCount,
		completion.CarriedOverCount,
		completion.DroppedCount,
		completion.PlannedSizeSum,   // nil → SQL NULL
		completion.CompletedSizeSum, // nil → SQL NULL
		completion.CarryoverMode,
		completion.NextSprintID, // nil → SQL NULL
	)
	if err != nil {
		return fmt.Errorf("failed to create sprint_completions record for sprint %d: %w", completion.SprintID, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get sprint_completions insert id: %w", err)
	}

	completion.ID = id
	return nil
}
