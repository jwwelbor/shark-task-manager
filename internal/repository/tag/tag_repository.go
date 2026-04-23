package tag

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	"github.com/jwwelbor/shark-task-manager/internal/repository/repoutil"
)

// tagTracer is the package-level OpenTelemetry tracer for tag repository operations.
var tagTracer = repoutil.NewTracer("internal/repository/tag")

// Sentinel errors for vocabulary operations (AC-T3, AC-T4).

// ErrTagInUse is returned by Delete(force=false) when entity_tags rows exist for
// the tag being deleted (ADR-9).
var ErrTagInUse = errors.New("tag is in use by one or more entities")

// ErrTagConflict is returned by Rename when the target name already exists in
// the tags table (ADR-8).
var ErrTagConflict = errors.New("tag name already exists")

// ErrTagNotFound is returned by GetByName, GetByID, and Rename when the tag row
// cannot be found.
var ErrTagNotFound = errors.New("tag not found")

// TagRepository handles CRUD operations for the tags vocabulary table.
// It owns SQL only — no business logic, config reads, or password checks live here.
type TagRepository struct {
	db *dbconn.DB
}

// NewTagRepository creates a new TagRepository backed by the given DB connection.
func NewTagRepository(db *dbconn.DB) *TagRepository {
	return &TagRepository{db: db}
}

// Create inserts a new tag row with the given name.
// Returns the fully-populated Tag on success, or an error if the name is already
// taken (UNIQUE constraint on tags.name COLLATE NOCASE).
func (r *TagRepository) Create(ctx context.Context, name string) (_ *models.Tag, retErr error) {
	ctx, span := tagTracer.Start(ctx, "TagRepository.Create",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "INSERT"),
			attribute.String("db.table", "tags"),
			attribute.String("tag.name", name),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	query := `INSERT INTO tags (name) VALUES (?) RETURNING id, name, created_at, updated_at`

	t := &models.Tag{}
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&t.ID, &t.Name, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, fmt.Errorf("create tag %q: %w", name, ErrTagConflict)
		}
		return nil, fmt.Errorf("create tag %q: %w", name, err)
	}
	return t, nil
}

// GetByName looks up a tag by name using NOCASE collation (case-insensitive).
// Returns ErrTagNotFound (wrapped) if the name does not exist.
func (r *TagRepository) GetByName(ctx context.Context, name string) (_ *models.Tag, retErr error) {
	ctx, span := tagTracer.Start(ctx, "TagRepository.GetByName",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "SELECT"),
			attribute.String("db.table", "tags"),
			attribute.String("tag.name", name),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	query := `SELECT id, name, created_at, updated_at FROM tags WHERE name = ? COLLATE NOCASE`

	t := &models.Tag{}
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&t.ID, &t.Name, &t.CreatedAt, &t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("get tag by name %q: %w", name, ErrTagNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get tag by name %q: %w", name, err)
	}
	return t, nil
}

// GetByID looks up a tag by its primary key.
// Returns ErrTagNotFound (wrapped) if the id does not exist.
func (r *TagRepository) GetByID(ctx context.Context, id int64) (_ *models.Tag, retErr error) {
	ctx, span := tagTracer.Start(ctx, "TagRepository.GetByID",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "SELECT"),
			attribute.String("db.table", "tags"),
			attribute.Int64("tag.id", id),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	query := `SELECT id, name, created_at, updated_at FROM tags WHERE id = ?`

	t := &models.Tag{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&t.ID, &t.Name, &t.CreatedAt, &t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("get tag by id %d: %w", id, ErrTagNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get tag by id %d: %w", id, err)
	}
	return t, nil
}

// List returns all tags in the vocabulary, ordered by name ascending.
// Returns an empty (non-nil) slice when no tags exist.
func (r *TagRepository) List(ctx context.Context) (_ []*models.Tag, retErr error) {
	ctx, span := tagTracer.Start(ctx, "TagRepository.List",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "SELECT"),
			attribute.String("db.table", "tags"),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	query := `SELECT id, name, created_at, updated_at FROM tags ORDER BY name ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()

	tags := make([]*models.Tag, 0)
	for rows.Next() {
		t := &models.Tag{}
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("list tags: scan: %w", err)
		}
		tags = append(tags, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list tags: iterate: %w", err)
	}
	return tags, nil
}

// Rename updates the tag name in place via a single UPDATE statement (ADR-8).
// entity_tags rows are NOT touched — the tag ID is immutable.
// Returns ErrTagConflict (wrapped) if newName already exists.
// Returns ErrTagNotFound (wrapped) if id does not exist.
func (r *TagRepository) Rename(ctx context.Context, id int64, newName string) (_ *models.Tag, retErr error) {
	ctx, span := tagTracer.Start(ctx, "TagRepository.Rename",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "UPDATE"),
			attribute.String("db.table", "tags"),
			attribute.Int64("tag.id", id),
			attribute.String("tag.new_name", newName),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	query := `
		UPDATE tags
		SET name = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
		RETURNING id, name, created_at, updated_at
	`

	t := &models.Tag{}
	err := r.db.QueryRowContext(ctx, query, newName, id).Scan(
		&t.ID, &t.Name, &t.CreatedAt, &t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("rename tag %d: %w", id, ErrTagNotFound)
	}
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, fmt.Errorf("rename tag %d to %q: %w", id, newName, ErrTagConflict)
		}
		return nil, fmt.Errorf("rename tag %d to %q: %w", id, newName, err)
	}
	return t, nil
}

// Delete removes the tag identified by id (ADR-9).
//
// When force is false and entity_tags rows exist for this tag, returns ErrTagInUse.
// When force is true, deletes all entity_tags rows for this tag in the same
// transaction before removing the tags row.
//
// Returns an error if id is not found.
func (r *TagRepository) Delete(ctx context.Context, id int64, force bool) (retErr error) {
	ctx, span := tagTracer.Start(ctx, "TagRepository.Delete",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "DELETE"),
			attribute.String("db.table", "tags"),
			attribute.Int64("tag.id", id),
			attribute.Bool("tag.force", force),
		))
	defer func() { repoutil.RecordSpanError(span, retErr); span.End() }()

	tx, err := r.db.BeginTxContext(ctx)
	if err != nil {
		return fmt.Errorf("delete tag %d: begin transaction: %w", id, err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Verify the tag exists.
	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM tags WHERE id = ?`, id).Scan(&exists); err != nil {
		return fmt.Errorf("delete tag %d: existence check: %w", id, err)
	}
	if exists == 0 {
		return fmt.Errorf("delete tag %d: %w", id, ErrTagNotFound)
	}

	// Count usages in entity_tags.
	var count int64
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM entity_tags WHERE tag_id = ?`, id).Scan(&count); err != nil {
		return fmt.Errorf("delete tag %d: usage count: %w", id, err)
	}

	if count > 0 && !force {
		return fmt.Errorf("delete tag %d: %w (used by %d entities)", id, ErrTagInUse, count)
	}

	if count > 0 && force {
		// Remove entity_tags rows first.
		if _, err := tx.ExecContext(ctx, `DELETE FROM entity_tags WHERE tag_id = ?`, id); err != nil {
			return fmt.Errorf("delete tag %d: remove entity_tags: %w", id, err)
		}
	}

	// Remove the tag itself.
	result, err := tx.ExecContext(ctx, `DELETE FROM tags WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete tag %d: %w", id, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete tag %d: rows affected: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("delete tag %d: %w", id, ErrTagNotFound)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("delete tag %d: commit: %w", id, err)
	}
	return nil
}

// isUniqueConstraintError returns true when err is a SQLite UNIQUE constraint violation.
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint failed") ||
		strings.Contains(msg, "unique constraint")
}
