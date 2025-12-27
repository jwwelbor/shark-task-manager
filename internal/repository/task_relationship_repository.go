package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// TaskRelationshipRepository handles CRUD operations for task relationships
type TaskRelationshipRepository struct {
	db *DB
}

// NewTaskRelationshipRepository creates a new TaskRelationshipRepository
func NewTaskRelationshipRepository(db *DB) *TaskRelationshipRepository {
	return &TaskRelationshipRepository{db: db}
}

// Create creates a new task relationship
func (r *TaskRelationshipRepository) Create(ctx context.Context, rel *models.TaskRelationship) error {
	if err := rel.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		INSERT INTO task_relationships (
			from_task_id, to_task_id, relationship_type
		)
		VALUES (?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		rel.FromTaskID,
		rel.ToTaskID,
		rel.RelationshipType,
	)
	if err != nil {
		// Check for UNIQUE constraint violation
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("relationship already exists: from_task_id=%d to_task_id=%d type=%s",
				rel.FromTaskID, rel.ToTaskID, rel.RelationshipType)
		}
		return fmt.Errorf("failed to create task relationship: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	rel.ID = id
	return nil
}

// GetByID retrieves a task relationship by its ID
func (r *TaskRelationshipRepository) GetByID(ctx context.Context, id int64) (*models.TaskRelationship, error) {
	query := `
		SELECT id, from_task_id, to_task_id, relationship_type, created_at
		FROM task_relationships
		WHERE id = ?
	`

	rel := &models.TaskRelationship{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&rel.ID,
		&rel.FromTaskID,
		&rel.ToTaskID,
		&rel.RelationshipType,
		&rel.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task relationship not found with id %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task relationship: %w", err)
	}

	return rel, nil
}

// GetByTaskID retrieves all relationships for a task (both incoming and outgoing)
func (r *TaskRelationshipRepository) GetByTaskID(ctx context.Context, taskID int64) ([]*models.TaskRelationship, error) {
	query := `
		SELECT id, from_task_id, to_task_id, relationship_type, created_at
		FROM task_relationships
		WHERE from_task_id = ? OR to_task_id = ?
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, taskID, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to query task relationships: %w", err)
	}
	defer rows.Close()

	var relationships []*models.TaskRelationship
	for rows.Next() {
		rel := &models.TaskRelationship{}
		err := rows.Scan(
			&rel.ID,
			&rel.FromTaskID,
			&rel.ToTaskID,
			&rel.RelationshipType,
			&rel.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task relationship: %w", err)
		}
		relationships = append(relationships, rel)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating task relationships: %w", err)
	}

	return relationships, nil
}

// GetOutgoing retrieves all outgoing relationships for a task (from_task_id = taskID)
func (r *TaskRelationshipRepository) GetOutgoing(ctx context.Context, taskID int64, relTypes []string) ([]*models.TaskRelationship, error) {
	if len(relTypes) == 0 {
		// Get all outgoing relationships
		query := `
			SELECT id, from_task_id, to_task_id, relationship_type, created_at
			FROM task_relationships
			WHERE from_task_id = ?
			ORDER BY created_at ASC
		`

		rows, err := r.db.QueryContext(ctx, query, taskID)
		if err != nil {
			return nil, fmt.Errorf("failed to query outgoing relationships: %w", err)
		}
		defer rows.Close()

		return r.scanRelationships(rows)
	}

	// Build query with IN clause for specific types
	placeholders := make([]string, len(relTypes))
	args := make([]interface{}, len(relTypes)+1)
	args[0] = taskID

	for i, relType := range relTypes {
		placeholders[i] = "?"
		args[i+1] = relType
	}

	query := fmt.Sprintf(`
		SELECT id, from_task_id, to_task_id, relationship_type, created_at
		FROM task_relationships
		WHERE from_task_id = ? AND relationship_type IN (%s)
		ORDER BY created_at ASC
	`, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query outgoing relationships by type: %w", err)
	}
	defer rows.Close()

	return r.scanRelationships(rows)
}

// GetIncoming retrieves all incoming relationships for a task (to_task_id = taskID)
func (r *TaskRelationshipRepository) GetIncoming(ctx context.Context, taskID int64, relTypes []string) ([]*models.TaskRelationship, error) {
	if len(relTypes) == 0 {
		// Get all incoming relationships
		query := `
			SELECT id, from_task_id, to_task_id, relationship_type, created_at
			FROM task_relationships
			WHERE to_task_id = ?
			ORDER BY created_at ASC
		`

		rows, err := r.db.QueryContext(ctx, query, taskID)
		if err != nil {
			return nil, fmt.Errorf("failed to query incoming relationships: %w", err)
		}
		defer rows.Close()

		return r.scanRelationships(rows)
	}

	// Build query with IN clause for specific types
	placeholders := make([]string, len(relTypes))
	args := make([]interface{}, len(relTypes)+1)
	args[0] = taskID

	for i, relType := range relTypes {
		placeholders[i] = "?"
		args[i+1] = relType
	}

	query := fmt.Sprintf(`
		SELECT id, from_task_id, to_task_id, relationship_type, created_at
		FROM task_relationships
		WHERE to_task_id = ? AND relationship_type IN (%s)
		ORDER BY created_at ASC
	`, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query incoming relationships by type: %w", err)
	}
	defer rows.Close()

	return r.scanRelationships(rows)
}

// Delete deletes a task relationship by ID
func (r *TaskRelationshipRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM task_relationships WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete task relationship: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task relationship not found with id %d", id)
	}

	return nil
}

// DeleteByTasksAndType deletes a specific relationship between two tasks
func (r *TaskRelationshipRepository) DeleteByTasksAndType(ctx context.Context, fromTaskID, toTaskID int64, relType string) error {
	query := `
		DELETE FROM task_relationships
		WHERE from_task_id = ? AND to_task_id = ? AND relationship_type = ?
	`

	result, err := r.db.ExecContext(ctx, query, fromTaskID, toTaskID, relType)
	if err != nil {
		return fmt.Errorf("failed to delete task relationship: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("relationship not found: from_task_id=%d to_task_id=%d type=%s",
			fromTaskID, toTaskID, relType)
	}

	return nil
}

// scanRelationships is a helper function to scan multiple relationships from query results
func (r *TaskRelationshipRepository) scanRelationships(rows *sql.Rows) ([]*models.TaskRelationship, error) {
	var relationships []*models.TaskRelationship
	for rows.Next() {
		rel := &models.TaskRelationship{}
		err := rows.Scan(
			&rel.ID,
			&rel.FromTaskID,
			&rel.ToTaskID,
			&rel.RelationshipType,
			&rel.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task relationship: %w", err)
		}
		relationships = append(relationships, rel)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating task relationships: %w", err)
	}

	return relationships, nil
}

// DetectCycle checks if creating a relationship would create a circular dependency
// Returns error if cycle is detected, nil otherwise
// Only checks for depends_on and blocks relationship types (hard dependencies)
func (r *TaskRelationshipRepository) DetectCycle(ctx context.Context, fromTaskID, toTaskID int64, relType string) error {
	// Only check cycles for dependency types that create blocking relationships
	if relType != "depends_on" && relType != "blocks" {
		return nil // Other relationship types don't create cycles
	}

	// Use depth-first search to detect cycles
	visited := make(map[int64]bool)
	recStack := make(map[int64]bool)

	// Build adjacency list for the dependency graph
	// For depends_on: A depends_on B means B -> A in the graph (B must come first)
	// For blocks: A blocks B means A -> B in the graph (A blocks B)

	hasCycle := r.detectCycleDFS(ctx, fromTaskID, toTaskID, relType, visited, recStack)
	if hasCycle {
		// Build cycle path for error message
		path := r.buildCyclePath(ctx, fromTaskID, toTaskID, relType)
		return fmt.Errorf("%w: %s", models.ErrCircularDependency, path)
	}

	return nil
}

// detectCycleDFS performs depth-first search to detect cycles
func (r *TaskRelationshipRepository) detectCycleDFS(ctx context.Context, currentTask, targetTask int64, relType string, visited, recStack map[int64]bool) bool {
	// If adding the new relationship, simulate it
	if currentTask == targetTask {
		return true // Would create direct cycle
	}

	visited[currentTask] = true
	recStack[currentTask] = true

	// Get all outgoing relationships of dependency types from current task
	rels, err := r.GetOutgoing(ctx, currentTask, []string{"depends_on", "blocks"})
	if err != nil {
		return false // Ignore query errors in cycle detection
	}

	for _, rel := range rels {
		// Determine next task based on relationship semantics
		var nextTask int64
		if rel.RelationshipType == "depends_on" {
			// A depends_on B: B comes before A, so traverse from A to B
			nextTask = rel.ToTaskID
		} else if rel.RelationshipType == "blocks" {
			// A blocks B: A must complete before B, so traverse from A to B
			nextTask = rel.ToTaskID
		}

		if nextTask == targetTask {
			// Found path back to target task - would create cycle
			return true
		}

		if !visited[nextTask] {
			if r.detectCycleDFS(ctx, nextTask, targetTask, relType, visited, recStack) {
				return true
			}
		} else if recStack[nextTask] {
			// Found a node in recursion stack - cycle exists
			return true
		}
	}

	recStack[currentTask] = false
	return false
}

// buildCyclePath builds a human-readable cycle path for error messages
func (r *TaskRelationshipRepository) buildCyclePath(ctx context.Context, fromTaskID, toTaskID int64, relType string) string {
	// For now, return a simple message
	// In the future, could traverse the graph to build the actual path
	return fmt.Sprintf("task %d -> task %d (via %s)", fromTaskID, toTaskID, relType)
}
