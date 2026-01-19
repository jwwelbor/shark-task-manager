package repository

import (
	"context"
	"database/sql"
	"encoding/json"
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

// SearchWithTimePeriod searches for notes with optional time period filtering
// since: filter notes created after this timestamp (YYYY-MM-DD format, optional)
// until: filter notes created before this timestamp (YYYY-MM-DD format, optional)
func (r *TaskNoteRepository) SearchWithTimePeriod(ctx context.Context, query string, noteTypes []string, epicKey string, featureKey string, since string, until string) ([]*models.TaskNote, error) {
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

	// Add time period filters
	if since != "" {
		sqlQuery += " AND tn.created_at >= ?"
		args = append(args, since+" 00:00:00")
	}

	if until != "" {
		sqlQuery += " AND tn.created_at <= ?"
		args = append(args, until+" 23:59:59")
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

// RejectionNoteMetadata represents the metadata structure for rejection notes
type RejectionNoteMetadata struct {
	HistoryID    int64  `json:"history_id"`
	FromStatus   string `json:"from_status"`
	ToStatus     string `json:"to_status"`
	DocumentPath string `json:"document_path,omitempty"`
}

// CreateRejectionNote creates a task note with note_type=rejection and metadata linking to history
func (r *TaskNoteRepository) CreateRejectionNote(
	ctx context.Context,
	taskID int64,
	historyID int64,
	fromStatus string,
	toStatus string,
	reason string,
	rejectedBy string,
	documentPath *string,
) (*models.TaskNote, error) {
	// Validate inputs
	if taskID == 0 {
		return nil, fmt.Errorf("failed to create rejection note: task_id must be greater than 0")
	}
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("failed to create rejection note: reason cannot be empty or whitespace-only")
	}

	// Build metadata structure
	metadata := RejectionNoteMetadata{
		HistoryID:  historyID,
		FromStatus: fromStatus,
		ToStatus:   toStatus,
	}

	// Add document_path if provided (only include if non-nil)
	if documentPath != nil && *documentPath != "" {
		metadata.DocumentPath = *documentPath
	}

	// Marshal metadata to JSON
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create rejection note: failed to marshal metadata: %w", err)
	}

	metadataStr := string(metadataJSON)

	// Create the note
	note := &models.TaskNote{
		TaskID:    taskID,
		NoteType:  models.NoteTypeRejection,
		Content:   reason,
		CreatedBy: &rejectedBy,
		Metadata:  &metadataStr,
	}

	if err := note.Validate(); err != nil {
		return nil, fmt.Errorf("failed to create rejection note: validation failed: %w", err)
	}

	// Insert into database
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
		return nil, fmt.Errorf("failed to create rejection note: failed to insert into database: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to create rejection note: failed to get last insert id: %w", err)
	}

	note.ID = id
	return note, nil
}

// CreateRejectionNoteWithTx creates a task note with note_type=rejection within a transaction
func (r *TaskNoteRepository) CreateRejectionNoteWithTx(
	ctx context.Context,
	tx *sql.Tx,
	taskID int64,
	historyID int64,
	fromStatus string,
	toStatus string,
	reason string,
	rejectedBy string,
	documentPath *string,
) (*models.TaskNote, error) {
	// Validate inputs
	if taskID == 0 {
		return nil, fmt.Errorf("failed to create rejection note: task_id must be greater than 0")
	}
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("failed to create rejection note: reason cannot be empty or whitespace-only")
	}

	// Build metadata structure
	metadata := RejectionNoteMetadata{
		HistoryID:  historyID,
		FromStatus: fromStatus,
		ToStatus:   toStatus,
	}

	// Add document_path if provided (only include if non-nil)
	if documentPath != nil && *documentPath != "" {
		metadata.DocumentPath = *documentPath
	}

	// Marshal metadata to JSON
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create rejection note: failed to marshal metadata: %w", err)
	}

	metadataStr := string(metadataJSON)

	// Create the note
	note := &models.TaskNote{
		TaskID:    taskID,
		NoteType:  models.NoteTypeRejection,
		Content:   reason,
		CreatedBy: &rejectedBy,
		Metadata:  &metadataStr,
	}

	if err := note.Validate(); err != nil {
		return nil, fmt.Errorf("failed to create rejection note: validation failed: %w", err)
	}

	// Insert into database within transaction
	query := `
		INSERT INTO task_notes (
			task_id, note_type, content, created_by, metadata
		)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := tx.ExecContext(ctx, query,
		note.TaskID,
		note.NoteType,
		note.Content,
		note.CreatedBy,
		note.Metadata,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create rejection note: failed to insert into database: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to create rejection note: failed to get last insert id: %w", err)
	}

	note.ID = id
	return note, nil
}

// RejectionHistoryEntry represents a single rejection in task rejection history
type RejectionHistoryEntry struct {
	ID             int64   `json:"id"`
	Timestamp      string  `json:"timestamp"`
	FromStatus     string  `json:"from_status"`
	ToStatus       string  `json:"to_status"`
	RejectedBy     string  `json:"rejected_by"`
	Reason         string  `json:"reason"`
	ReasonDocument *string `json:"reason_document"`
	HistoryID      int64   `json:"history_id"`
}

// GetRejectionHistory retrieves rejection history for a task, ordered by most recent first
func (r *TaskNoteRepository) GetRejectionHistory(ctx context.Context, taskID int64) ([]*RejectionHistoryEntry, error) {
	if taskID == 0 {
		return nil, fmt.Errorf("failed to get rejection history: task_id must be greater than 0")
	}

	query := `
		SELECT id, created_at, content, created_by, metadata
		FROM task_notes
		WHERE task_id = ? AND note_type = ?
		ORDER BY id DESC
	`

	rows, err := r.db.QueryContext(ctx, query, taskID, models.NoteTypeRejection)
	if err != nil {
		return nil, fmt.Errorf("failed to query rejection history: %w", err)
	}
	defer rows.Close()

	var entries []*RejectionHistoryEntry
	for rows.Next() {
		var id int64
		var createdAt string
		var content string
		var createdBy *string
		var metadataStr *string

		err := rows.Scan(&id, &createdAt, &content, &createdBy, &metadataStr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rejection history: %w", err)
		}

		// Parse metadata JSON to extract status transition and document path
		var metadata RejectionNoteMetadata
		if metadataStr != nil && *metadataStr != "" {
			// Ignore error - continue with empty metadata if JSON is invalid
			_ = json.Unmarshal([]byte(*metadataStr), &metadata)
		}

		// Build rejection history entry
		var rejectedBy string
		if createdBy != nil {
			rejectedBy = *createdBy
		}

		entry := &RejectionHistoryEntry{
			ID:         id,
			Timestamp:  createdAt,
			FromStatus: metadata.FromStatus,
			ToStatus:   metadata.ToStatus,
			RejectedBy: rejectedBy,
			Reason:     content,
			HistoryID:  metadata.HistoryID,
		}

		// Include document path if present
		if metadata.DocumentPath != "" {
			entry.ReasonDocument = &metadata.DocumentPath
		}

		entries = append(entries, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rejection history: %w", err)
	}

	// Return empty slice if no rejections (not nil, not error)
	if entries == nil {
		entries = make([]*RejectionHistoryEntry, 0)
	}

	return entries, nil
}
