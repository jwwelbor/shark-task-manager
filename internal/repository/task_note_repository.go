package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// TaskNoteRepository handles CRUD operations for task notes
type TaskNoteRepository struct {
	db *DB
}

// NewTaskNoteRepository creates a new TaskNoteRepository
func NewTaskNoteRepository(db *DB) *TaskNoteRepository {
	return &TaskNoteRepository{db: db}
}

// Create creates a new task note
func (r *TaskNoteRepository) Create(ctx context.Context, note *models.TaskNote) error {
	if err := note.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		INSERT INTO task_notes (
			task_id, note_type, content, created_by, metadata
		)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		note.TaskID,
		note.NoteType,
		note.Content,
		note.CreatedBy,
		note.Metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to create task note: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	note.ID = id
	return nil
}

// GetByID retrieves a task note by its ID
func (r *TaskNoteRepository) GetByID(ctx context.Context, id int64) (*models.TaskNote, error) {
	query := `
		SELECT id, task_id, note_type, content, created_by, metadata, created_at
		FROM task_notes
		WHERE id = ?
	`

	note := &models.TaskNote{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&note.ID,
		&note.TaskID,
		&note.NoteType,
		&note.Content,
		&note.CreatedBy,
		&note.Metadata,
		&note.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task note not found with id %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task note: %w", err)
	}

	return note, nil
}

// GetByTaskID retrieves all notes for a task
func (r *TaskNoteRepository) GetByTaskID(ctx context.Context, taskID int64) ([]*models.TaskNote, error) {
	query := `
		SELECT id, task_id, note_type, content, created_by, metadata, created_at
		FROM task_notes
		WHERE task_id = ?
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to query task notes: %w", err)
	}
	defer rows.Close()

	var notes []*models.TaskNote
	for rows.Next() {
		note := &models.TaskNote{}
		err := rows.Scan(
			&note.ID,
			&note.TaskID,
			&note.NoteType,
			&note.Content,
			&note.CreatedBy,
			&note.Metadata,
			&note.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task note: %w", err)
		}
		notes = append(notes, note)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating task notes: %w", err)
	}

	return notes, nil
}

// GetByTaskIDAndType retrieves notes for a task filtered by type(s)
func (r *TaskNoteRepository) GetByTaskIDAndType(ctx context.Context, taskID int64, noteTypes []string) ([]*models.TaskNote, error) {
	if len(noteTypes) == 0 {
		return r.GetByTaskID(ctx, taskID)
	}

	// Build query with IN clause for multiple types
	placeholders := make([]string, len(noteTypes))
	args := make([]interface{}, len(noteTypes)+1)
	args[0] = taskID

	for i, noteType := range noteTypes {
		placeholders[i] = "?"
		args[i+1] = noteType
	}

	query := fmt.Sprintf(`
		SELECT id, task_id, note_type, content, created_by, metadata, created_at
		FROM task_notes
		WHERE task_id = ? AND note_type IN (%s)
		ORDER BY created_at ASC
	`, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query task notes by type: %w", err)
	}
	defer rows.Close()

	var notes []*models.TaskNote
	for rows.Next() {
		note := &models.TaskNote{}
		err := rows.Scan(
			&note.ID,
			&note.TaskID,
			&note.NoteType,
			&note.Content,
			&note.CreatedBy,
			&note.Metadata,
			&note.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task note: %w", err)
		}
		notes = append(notes, note)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating task notes: %w", err)
	}

	return notes, nil
}

// Search searches for notes across all tasks containing the query string
func (r *TaskNoteRepository) Search(ctx context.Context, query string, noteTypes []string, epicKey string, featureKey string) ([]*models.TaskNote, error) {
	var sqlQuery string
	var args []interface{}

	if epicKey != "" || featureKey != "" {
		// Join with tasks, features, epics if filtering by epic/feature
		sqlQuery = `
			SELECT tn.id, tn.task_id, tn.note_type, tn.content, tn.created_by, tn.metadata, tn.created_at
			FROM task_notes AS tn
			INNER JOIN tasks AS t ON tn.task_id = t.id
			INNER JOIN features AS f ON t.feature_id = f.id
			INNER JOIN epics AS e ON f.epic_id = e.id
			WHERE tn.content LIKE ?
		`
		args = append(args, "%"+query+"%")

		if epicKey != "" {
			sqlQuery += " AND e.key = ?"
			args = append(args, epicKey)
		}
		if featureKey != "" {
			sqlQuery += " AND f.key = ?"
			args = append(args, featureKey)
		}
	} else {
		sqlQuery = `
			SELECT id, task_id, note_type, content, created_by, metadata, created_at
			FROM task_notes AS tn
			WHERE tn.content LIKE ?
		`
		args = append(args, "%"+query+"%")
	}

	// Add note type filter if provided
	if len(noteTypes) > 0 {
		placeholders := make([]string, len(noteTypes))
		for i, noteType := range noteTypes {
			placeholders[i] = "?"
			args = append(args, noteType)
		}
		sqlQuery += fmt.Sprintf(" AND tn.note_type IN (%s)", strings.Join(placeholders, ","))
	}

	// Order by created_at descending (most recent first)
	sqlQuery += " ORDER BY tn.created_at DESC"

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search task notes: %w", err)
	}
	defer rows.Close()

	var notes []*models.TaskNote
	for rows.Next() {
		note := &models.TaskNote{}
		err := rows.Scan(
			&note.ID,
			&note.TaskID,
			&note.NoteType,
			&note.Content,
			&note.CreatedBy,
			&note.Metadata,
			&note.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task note: %w", err)
		}
		notes = append(notes, note)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating search results: %w", err)
	}

	return notes, nil
}

// Delete deletes a task note by ID
func (r *TaskNoteRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM task_notes WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete task note: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task note not found with id %d", id)
	}

	return nil
}
