package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// DocumentRepository manages document data access
type DocumentRepository struct {
	db *DB
}

// NewDocumentRepository creates a new document repository
func NewDocumentRepository(db *DB) *DocumentRepository {
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

// LinkToEpic links a document to an epic
func (r *DocumentRepository) LinkToEpic(ctx context.Context, epicID, documentID int64) error {
	query := `
		INSERT OR IGNORE INTO epic_documents (epic_id, document_id)
		VALUES (?, ?)
	`

	_, err := r.db.ExecContext(ctx, query, epicID, documentID)
	if err != nil {
		return fmt.Errorf("failed to link document to epic: %w", err)
	}

	return nil
}

// LinkToFeature links a document to a feature
func (r *DocumentRepository) LinkToFeature(ctx context.Context, featureID, documentID int64) error {
	query := `
		INSERT OR IGNORE INTO feature_documents (feature_id, document_id)
		VALUES (?, ?)
	`

	_, err := r.db.ExecContext(ctx, query, featureID, documentID)
	if err != nil {
		return fmt.Errorf("failed to link document to feature: %w", err)
	}

	return nil
}

// LinkToTask links a document to a task
func (r *DocumentRepository) LinkToTask(ctx context.Context, taskID, documentID int64) error {
	query := `
		INSERT OR IGNORE INTO task_documents (task_id, document_id)
		VALUES (?, ?)
	`

	_, err := r.db.ExecContext(ctx, query, taskID, documentID)
	if err != nil {
		return fmt.Errorf("failed to link document to task: %w", err)
	}

	return nil
}

// UnlinkFromEpic removes a document link from an epic
func (r *DocumentRepository) UnlinkFromEpic(ctx context.Context, epicID, documentID int64) error {
	query := `DELETE FROM epic_documents WHERE epic_id = ? AND document_id = ?`

	_, err := r.db.ExecContext(ctx, query, epicID, documentID)
	if err != nil {
		return fmt.Errorf("failed to unlink document from epic: %w", err)
	}

	return nil
}

// UnlinkFromFeature removes a document link from a feature
func (r *DocumentRepository) UnlinkFromFeature(ctx context.Context, featureID, documentID int64) error {
	query := `DELETE FROM feature_documents WHERE feature_id = ? AND document_id = ?`

	_, err := r.db.ExecContext(ctx, query, featureID, documentID)
	if err != nil {
		return fmt.Errorf("failed to unlink document from feature: %w", err)
	}

	return nil
}

// UnlinkFromTask removes a document link from a task
func (r *DocumentRepository) UnlinkFromTask(ctx context.Context, taskID, documentID int64) error {
	query := `DELETE FROM task_documents WHERE task_id = ? AND document_id = ?`

	_, err := r.db.ExecContext(ctx, query, taskID, documentID)
	if err != nil {
		return fmt.Errorf("failed to unlink document from task: %w", err)
	}

	return nil
}

// ListForEpic returns all documents linked to an epic
func (r *DocumentRepository) ListForEpic(ctx context.Context, epicID int64) ([]*models.Document, error) {
	query := `
		SELECT d.id, d.title, d.file_path, d.created_at
		FROM documents d
		INNER JOIN epic_documents ed ON d.id = ed.document_id
		WHERE ed.epic_id = ?
	`

	rows, err := r.db.QueryContext(ctx, query, epicID)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents for epic: %w", err)
	}
	defer rows.Close()

	var docs []*models.Document
	for rows.Next() {
		doc := &models.Document{}
		err := rows.Scan(&doc.ID, &doc.Title, &doc.FilePath, &doc.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}
		docs = append(docs, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating documents: %w", err)
	}

	return docs, nil
}

// ListForFeature returns all documents linked to a feature
func (r *DocumentRepository) ListForFeature(ctx context.Context, featureID int64) ([]*models.Document, error) {
	query := `
		SELECT d.id, d.title, d.file_path, d.created_at
		FROM documents d
		INNER JOIN feature_documents fd ON d.id = fd.document_id
		WHERE fd.feature_id = ?
	`

	rows, err := r.db.QueryContext(ctx, query, featureID)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents for feature: %w", err)
	}
	defer rows.Close()

	var docs []*models.Document
	for rows.Next() {
		doc := &models.Document{}
		err := rows.Scan(&doc.ID, &doc.Title, &doc.FilePath, &doc.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}
		docs = append(docs, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating documents: %w", err)
	}

	return docs, nil
}

// ListForTask returns all documents linked to a task
func (r *DocumentRepository) ListForTask(ctx context.Context, taskID int64) ([]*models.Document, error) {
	query := `
		SELECT d.id, d.title, d.file_path, d.created_at
		FROM documents d
		INNER JOIN task_documents td ON d.id = td.document_id
		WHERE td.task_id = ?
	`

	rows, err := r.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents for task: %w", err)
	}
	defer rows.Close()

	var docs []*models.Document
	for rows.Next() {
		doc := &models.Document{}
		err := rows.Scan(&doc.ID, &doc.Title, &doc.FilePath, &doc.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}
		docs = append(docs, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating documents: %w", err)
	}

	return docs, nil
}
