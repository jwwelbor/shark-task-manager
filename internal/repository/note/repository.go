// Package note provides the EntityNoteRepository for storing entity notes
// (comments, decisions, rejections, etc.) for epics, features, and tasks.
package note

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	"github.com/jwwelbor/shark-task-manager/internal/repository/repoutil"
)

// noteTracer is the package-level OpenTelemetry tracer for note repository operations.
// Per-package tracers improve observability by attributing spans to the correct sub-package.
var noteTracer = repoutil.NewTracer("internal/repository/note")

// EntityNoteRepository handles CRUD operations for entity notes (epics, features, tasks)
type EntityNoteRepository struct {
	db *dbconn.DB
}

// NewEntityNoteRepository creates a new EntityNoteRepository
func NewEntityNoteRepository(db *dbconn.DB) *EntityNoteRepository {
	return &EntityNoteRepository{db: db}
}

// Create creates a new entity note
func (r *EntityNoteRepository) Create(ctx context.Context, note *models.EntityNote) (retErr error) {
	ctx, span := noteTracer.Start(ctx, "EntityNoteRepository.Create",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "INSERT"),
			attribute.String("db.table", "entity_notes"),
			attribute.String("db.entity_type", string(note.EntityType)),
			attribute.Int64("db.entity_id", note.EntityID),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	if err := note.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	query := `
		INSERT INTO entity_notes (
			entity_type, entity_id, note_type, content, created_by, metadata
		)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		note.EntityType,
		note.EntityID,
		note.NoteType,
		note.Content,
		note.CreatedBy,
		note.Metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to create entity note: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	note.ID = id
	return nil
}

// GetByID retrieves an entity note by its ID
func (r *EntityNoteRepository) GetByID(ctx context.Context, id int64) (_ *models.EntityNote, retErr error) {
	ctx, span := noteTracer.Start(ctx, "EntityNoteRepository.GetByID",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "SELECT"),
			attribute.String("db.table", "entity_notes"),
			attribute.Int64("db.id", id),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	query := `
		SELECT id, entity_type, entity_id, note_type, content, created_by, metadata, created_at
		FROM entity_notes
		WHERE id = ?
	`

	note := &models.EntityNote{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&note.ID,
		&note.EntityType,
		&note.EntityID,
		&note.NoteType,
		&note.Content,
		&note.CreatedBy,
		&note.Metadata,
		&note.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("entity note not found with id %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get entity note: %w", err)
	}

	return note, nil
}

// GetByEntity retrieves all notes for a specific entity, ordered by created_at ASC
func (r *EntityNoteRepository) GetByEntity(ctx context.Context, entityType models.EntityType, entityID int64) (_ []*models.EntityNote, retErr error) {
	ctx, span := noteTracer.Start(ctx, "EntityNoteRepository.GetByEntity",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "SELECT"),
			attribute.String("db.table", "entity_notes"),
			attribute.String("db.entity_type", string(entityType)),
			attribute.Int64("db.entity_id", entityID),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	query := `
		SELECT id, entity_type, entity_id, note_type, content, created_by, metadata, created_at
		FROM entity_notes
		WHERE entity_type = ? AND entity_id = ?
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to query entity notes: %w", err)
	}
	defer rows.Close()

	var notes []*models.EntityNote
	for rows.Next() {
		note := &models.EntityNote{}
		err := rows.Scan(
			&note.ID,
			&note.EntityType,
			&note.EntityID,
			&note.NoteType,
			&note.Content,
			&note.CreatedBy,
			&note.Metadata,
			&note.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entity note: %w", err)
		}
		notes = append(notes, note)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating entity notes: %w", err)
	}

	return notes, nil
}

// GetByEntityAndType retrieves notes for an entity filtered by note type(s)
func (r *EntityNoteRepository) GetByEntityAndType(ctx context.Context, entityType models.EntityType, entityID int64, noteTypes []string) ([]*models.EntityNote, error) {
	if len(noteTypes) == 0 {
		return r.GetByEntity(ctx, entityType, entityID)
	}

	// Build query with IN clause for multiple types
	placeholders := make([]string, len(noteTypes))
	args := make([]interface{}, len(noteTypes)+2)
	args[0] = entityType
	args[1] = entityID

	for i, noteType := range noteTypes {
		placeholders[i] = "?"
		args[i+2] = noteType
	}

	query := fmt.Sprintf(`
		SELECT id, entity_type, entity_id, note_type, content, created_by, metadata, created_at
		FROM entity_notes
		WHERE entity_type = ? AND entity_id = ? AND note_type IN (%s)
		ORDER BY created_at ASC
	`, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query entity notes by type: %w", err)
	}
	defer rows.Close()

	var notes []*models.EntityNote
	for rows.Next() {
		note := &models.EntityNote{}
		err := rows.Scan(
			&note.ID,
			&note.EntityType,
			&note.EntityID,
			&note.NoteType,
			&note.Content,
			&note.CreatedBy,
			&note.Metadata,
			&note.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entity note: %w", err)
		}
		notes = append(notes, note)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating entity notes: %w", err)
	}

	return notes, nil
}

// Search searches for notes across entities containing the query string.
// When entityType is nil, searches across ALL entity types.
// For task entity type, supports filtering by epicKey and featureKey via joins.
func (r *EntityNoteRepository) Search(ctx context.Context, query string, noteTypes []string, entityType *models.EntityType, epicKey string, featureKey string) ([]*models.EntityNote, error) {
	var sqlQuery string
	var args []interface{}

	// When filtering by epic/feature keys, we need to join through tasks->features->epics
	// This only applies to task-type notes for backward compatibility
	if epicKey != "" || featureKey != "" {
		sqlQuery = `
			SELECT en.id, en.entity_type, en.entity_id, en.note_type, en.content, en.created_by, en.metadata, en.created_at
			FROM entity_notes AS en
			INNER JOIN tasks AS t ON en.entity_id = t.id AND en.entity_type = 'task'
			INNER JOIN features AS f ON t.feature_id = f.id
			INNER JOIN epics AS e ON f.epic_id = e.id
			WHERE en.content LIKE ?
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
			SELECT id, entity_type, entity_id, note_type, content, created_by, metadata, created_at
			FROM entity_notes AS en
			WHERE en.content LIKE ?
		`
		args = append(args, "%"+query+"%")

		if entityType != nil {
			sqlQuery += " AND en.entity_type = ?"
			args = append(args, *entityType)
		}
	}

	// Add note type filter if provided
	if len(noteTypes) > 0 {
		placeholders := make([]string, len(noteTypes))
		for i, noteType := range noteTypes {
			placeholders[i] = "?"
			args = append(args, noteType)
		}
		sqlQuery += fmt.Sprintf(" AND en.note_type IN (%s)", strings.Join(placeholders, ","))
	}

	// Order by created_at descending (most recent first)
	sqlQuery += " ORDER BY en.created_at DESC"

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search entity notes: %w", err)
	}
	defer rows.Close()

	var notes []*models.EntityNote
	for rows.Next() {
		note := &models.EntityNote{}
		err := rows.Scan(
			&note.ID,
			&note.EntityType,
			&note.EntityID,
			&note.NoteType,
			&note.Content,
			&note.CreatedBy,
			&note.Metadata,
			&note.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entity note: %w", err)
		}
		notes = append(notes, note)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating search results: %w", err)
	}

	return notes, nil
}

// SearchWithTimePeriod searches for notes with optional time period filtering.
// since: filter notes created after this timestamp (YYYY-MM-DD format, optional)
// until: filter notes created before this timestamp (YYYY-MM-DD format, optional)
func (r *EntityNoteRepository) SearchWithTimePeriod(ctx context.Context, query string, noteTypes []string, epicKey string, featureKey string, since string, until string) ([]*models.EntityNote, error) {
	var sqlQuery string
	var args []interface{}

	if epicKey != "" || featureKey != "" {
		// Join with tasks, features, epics if filtering by epic/feature
		sqlQuery = `
			SELECT en.id, en.entity_type, en.entity_id, en.note_type, en.content, en.created_by, en.metadata, en.created_at
			FROM entity_notes AS en
			INNER JOIN tasks AS t ON en.entity_id = t.id AND en.entity_type = 'task'
			INNER JOIN features AS f ON t.feature_id = f.id
			INNER JOIN epics AS e ON f.epic_id = e.id
			WHERE en.content LIKE ?
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
			SELECT id, entity_type, entity_id, note_type, content, created_by, metadata, created_at
			FROM entity_notes AS en
			WHERE en.content LIKE ?
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
		sqlQuery += fmt.Sprintf(" AND en.note_type IN (%s)", strings.Join(placeholders, ","))
	}

	// Add time period filters
	if since != "" {
		sqlQuery += " AND en.created_at >= ?"
		args = append(args, since+" 00:00:00")
	}

	if until != "" {
		sqlQuery += " AND en.created_at <= ?"
		args = append(args, until+" 23:59:59")
	}

	// Order by created_at descending (most recent first)
	sqlQuery += " ORDER BY en.created_at DESC"

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search entity notes: %w", err)
	}
	defer rows.Close()

	var notes []*models.EntityNote
	for rows.Next() {
		note := &models.EntityNote{}
		err := rows.Scan(
			&note.ID,
			&note.EntityType,
			&note.EntityID,
			&note.NoteType,
			&note.Content,
			&note.CreatedBy,
			&note.Metadata,
			&note.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entity note: %w", err)
		}
		notes = append(notes, note)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating search results: %w", err)
	}

	return notes, nil
}

// Delete deletes an entity note by ID
func (r *EntityNoteRepository) Delete(ctx context.Context, id int64) (retErr error) {
	ctx, span := noteTracer.Start(ctx, "EntityNoteRepository.Delete",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "DELETE"),
			attribute.String("db.table", "entity_notes"),
			attribute.Int64("db.id", id),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	query := `DELETE FROM entity_notes WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete entity note: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("entity note not found with id %d", id)
	}

	return nil
}

// RejectionNoteMetadata represents the metadata structure for rejection notes
type RejectionNoteMetadata struct {
	HistoryID    int64  `json:"history_id"`
	FromStatus   string `json:"from_status"`
	ToStatus     string `json:"to_status"`
	DocumentPath string `json:"document_path,omitempty"`
}

// CreateRejectionNote creates an entity note with note_type=rejection and metadata linking to history
func (r *EntityNoteRepository) CreateRejectionNote(
	ctx context.Context,
	entityType models.EntityType,
	entityID int64,
	historyID int64,
	fromStatus string,
	toStatus string,
	reason string,
	rejectedBy string,
	documentPath *string,
) (_ *models.EntityNote, retErr error) {
	ctx, span := noteTracer.Start(ctx, "EntityNoteRepository.CreateRejectionNote",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "INSERT"),
			attribute.String("db.table", "entity_notes"),
			attribute.String("db.entity_type", string(entityType)),
			attribute.Int64("db.entity_id", entityID),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()
	// Validate inputs
	if entityID == 0 {
		return nil, fmt.Errorf("failed to create rejection note: entity_id must be greater than 0")
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
	note := &models.EntityNote{
		EntityType: entityType,
		EntityID:   entityID,
		NoteType:   models.NoteTypeRejection,
		Content:    reason,
		CreatedBy:  &rejectedBy,
		Metadata:   &metadataStr,
	}

	if err := note.Validate(); err != nil {
		return nil, fmt.Errorf("failed to create rejection note: validation failed: %w", err)
	}

	// Insert into database
	query := `
		INSERT INTO entity_notes (
			entity_type, entity_id, note_type, content, created_by, metadata
		)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		note.EntityType,
		note.EntityID,
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

// CreateRejectionNoteWithTx creates an entity note with note_type=rejection within a transaction.
// This method satisfies the repository.NoteCreator interface used by TaskRepository.
func (r *EntityNoteRepository) CreateRejectionNoteWithTx(
	ctx context.Context,
	tx *sql.Tx,
	entityType models.EntityType,
	entityID int64,
	historyID int64,
	fromStatus string,
	toStatus string,
	reason string,
	rejectedBy string,
	documentPath *string,
) (*models.EntityNote, error) {
	// Validate inputs
	if entityID == 0 {
		return nil, fmt.Errorf("failed to create rejection note: entity_id must be greater than 0")
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
	note := &models.EntityNote{
		EntityType: entityType,
		EntityID:   entityID,
		NoteType:   models.NoteTypeRejection,
		Content:    reason,
		CreatedBy:  &rejectedBy,
		Metadata:   &metadataStr,
	}

	if err := note.Validate(); err != nil {
		return nil, fmt.Errorf("failed to create rejection note: validation failed: %w", err)
	}

	// Insert into database within transaction
	query := `
		INSERT INTO entity_notes (
			entity_type, entity_id, note_type, content, created_by, metadata
		)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := tx.ExecContext(ctx, query,
		note.EntityType,
		note.EntityID,
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

// RejectionHistoryEntry represents a single rejection in entity rejection history
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

// GetRejectionHistory retrieves rejection history for an entity, ordered by most recent first
func (r *EntityNoteRepository) GetRejectionHistory(ctx context.Context, entityType models.EntityType, entityID int64) ([]*RejectionHistoryEntry, error) {
	if entityID == 0 {
		return nil, fmt.Errorf("failed to get rejection history: entity_id must be greater than 0")
	}

	query := `
		SELECT id, created_at, content, created_by, metadata
		FROM entity_notes
		WHERE entity_type = ? AND entity_id = ? AND note_type = ?
		ORDER BY id DESC
	`

	rows, err := r.db.QueryContext(ctx, query, entityType, entityID, models.NoteTypeRejection)
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
		var rejectedByVal string
		if createdBy != nil {
			rejectedByVal = *createdBy
		}

		entry := &RejectionHistoryEntry{
			ID:         id,
			Timestamp:  createdAt,
			FromStatus: metadata.FromStatus,
			ToStatus:   metadata.ToStatus,
			RejectedBy: rejectedByVal,
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
