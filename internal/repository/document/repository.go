package document

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
)

// DocumentRepository manages document data access
type DocumentRepository struct {
	db *dbconn.DB
}

// NewDocumentRepository creates a new document repository
func NewDocumentRepository(db *dbconn.DB) *DocumentRepository {
	return &DocumentRepository{db: db}
}

// CreateOrGet creates a new document or returns existing one with same title and path
func (r *DocumentRepository) CreateOrGet(ctx context.Context, title, filePath string) (*models.Document, error) {
	// Try to get existing document first
	doc, err := r.getByTitleAndPath(ctx, title, filePath)
	if err == nil {
		return doc, nil
	}
	if !errors.Is(err, sql.ErrNoRows) && err.Error() != "document not found" {
		return nil, err
	}

	// Create new document
	query := `
		INSERT INTO documents (title, file_path)
		VALUES (?, ?)
	`

	result, err := r.db.ExecContext(ctx, query, title, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create document: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return &models.Document{
		ID:       id,
		Title:    title,
		FilePath: filePath,
	}, nil
}

// GetByID retrieves a document by ID
func (r *DocumentRepository) GetByID(ctx context.Context, id int64) (*models.Document, error) {
	query := `
		SELECT id, title, file_path, created_at
		FROM documents
		WHERE id = ?
	`

	doc := &models.Document{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&doc.ID,
		&doc.Title,
		&doc.FilePath,
		&doc.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("document not found with id %d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	return doc, nil
}

// getByTitleAndPath retrieves a document by title and file path
func (r *DocumentRepository) getByTitleAndPath(ctx context.Context, title, filePath string) (*models.Document, error) {
	query := `
		SELECT id, title, file_path, created_at
		FROM documents
		WHERE title = ? AND file_path = ?
	`

	doc := &models.Document{}
	err := r.db.QueryRowContext(ctx, query, title, filePath).Scan(
		&doc.ID,
		&doc.Title,
		&doc.FilePath,
		&doc.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("document not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	return doc, nil
}

// GetByTitle retrieves a document by title only
func (r *DocumentRepository) GetByTitle(ctx context.Context, title string) (*models.Document, error) {
	query := `
		SELECT id, title, file_path, created_at
		FROM documents
		WHERE title = ?
	`

	doc := &models.Document{}
	err := r.db.QueryRowContext(ctx, query, title).Scan(
		&doc.ID,
		&doc.Title,
		&doc.FilePath,
		&doc.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("document not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	return doc, nil
}

// Delete removes a document
func (r *DocumentRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM documents WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	return nil
}
