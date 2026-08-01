// Package question persists the bounded Question base record.
package question

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	repoerr "github.com/jwwelbor/shark-task-manager/internal/repository/repoerr"
)

const questionSelectColumns = `id, key, title, slug, description, status, summary, blocking, requester,
	context_data, file_path, size, created_at, updated_at`

// QuestionListFilter is the finite, index-backed Question list filter.
// A zero Limit uses the public default of 50; callers may request 1 through 100.
type QuestionListFilter struct {
	Status    *models.QuestionStatus
	Requester *string
	Blocking  *bool
	Limit     int
	Offset    int
}

// QuestionRepository handles persistence for Question base records.
type QuestionRepository struct {
	db *dbconn.DB
}

// NewQuestionRepository creates a Question repository backed by db.
func NewQuestionRepository(db *dbconn.DB) *QuestionRepository {
	return &QuestionRepository{db: db}
}

func scanQuestion(scanner interface{ Scan(...any) error }) (*models.Question, error) {
	question := &models.Question{}
	if err := scanner.Scan(
		&question.ID,
		&question.Key,
		&question.Title,
		&question.Slug,
		&question.Description,
		&question.Status,
		&question.Summary,
		&question.Blocking,
		&question.Requester,
		&question.ContextData,
		&question.FilePath,
		&question.Size,
		&question.CreatedAt,
		&question.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return question, nil
}

// Create persists question. An empty key is allocated from the closed Q001-Q999
// range. The caller receives an ID/key only after the INSERT succeeds.
func (r *QuestionRepository) Create(ctx context.Context, question *models.Question) error {
	if question == nil {
		return errors.New("create question: question is required")
	}

	candidate := *question
	allocated := candidate.Key == ""
	for attempt := 0; attempt < 8; attempt++ {
		if allocated {
			key, err := r.GenerateNextKey(ctx)
			if err != nil {
				return fmt.Errorf("allocate question key: %w", err)
			}
			candidate.Key = key
		}
		if err := candidate.Validate(); err != nil {
			return fmt.Errorf("validate question: %w", err)
		}
		result, err := r.db.ExecContext(ctx, `
		INSERT INTO questions (
			key, title, slug, description, status, summary, blocking, requester,
			context_data, file_path, size
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, candidate.Key, candidate.Title, candidate.Slug, candidate.Description,
			candidate.Status, candidate.Summary, candidate.Blocking, candidate.Requester,
			candidate.ContextData, candidate.FilePath, candidate.Size)
		if err != nil {
			if allocated && strings.Contains(strings.ToLower(err.Error()), "unique") {
				continue
			}
			return fmt.Errorf("create question: %w", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("read created question ID: %w", err)
		}
		question.ID = id
		question.Key = candidate.Key
		return nil
	}
	return errors.New("allocate question key: concurrent allocation retry limit exceeded")
}

// GenerateNextKey returns the next never-reused Question key or an allocation
// error when Q999 is already persisted.
func (r *QuestionRepository) GenerateNextKey(ctx context.Context) (string, error) {
	var maxKey int
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(CAST(SUBSTR(key, 2) AS INTEGER)), 0)
		FROM questions
		WHERE key GLOB 'Q[0-9][0-9][0-9]'
	`).Scan(&maxKey)
	if err != nil {
		return "", fmt.Errorf("find highest persisted question key: %w", err)
	}
	if maxKey >= 999 {
		return "", errors.New("Question key allocation exhausted at Q999")
	}
	return fmt.Sprintf("Q%03d", maxKey+1), nil
}

// GetByKey returns the canonical Question key. Parsing/normalization belongs to
// keys.Service; this persistence seam accepts only the stored canonical key.
func (r *QuestionRepository) GetByKey(ctx context.Context, key string) (*models.Question, error) {
	if err := models.ValidateQuestionKey(key); err != nil {
		return nil, fmt.Errorf("get question by key: %w", err)
	}
	question, err := scanQuestion(r.db.QueryRowContext(ctx,
		fmt.Sprintf("SELECT %s FROM questions WHERE key = ?", questionSelectColumns), key))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("question not found with key %q: %w", key, repoerr.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get question by key: %w", err)
	}
	return question, nil
}

// GetByID returns a Question by database ID.
func (r *QuestionRepository) GetByID(ctx context.Context, id int64) (*models.Question, error) {
	question, err := scanQuestion(r.db.QueryRowContext(ctx,
		fmt.Sprintf("SELECT %s FROM questions WHERE id = ?", questionSelectColumns), id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("question not found with ID %d: %w", id, repoerr.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get question by ID: %w", err)
	}
	return question, nil
}

// Update persists Question base fields. Workflow status changes use UpdateStatus.
func (r *QuestionRepository) Update(ctx context.Context, question *models.Question) error {
	if question == nil {
		return errors.New("update question: question is required")
	}
	if err := question.Validate(); err != nil {
		return fmt.Errorf("validate question: %w", err)
	}
	result, err := r.db.ExecContext(ctx, `
		UPDATE questions
		SET title = ?, slug = ?, description = ?, summary = ?, blocking = ?, requester = ?,
			context_data = ?, file_path = ?, size = ?
		WHERE id = ?
	`, question.Title, question.Slug, question.Description, question.Summary, question.Blocking,
		question.Requester, question.ContextData, question.FilePath, question.Size, question.ID)
	if err != nil {
		return fmt.Errorf("update question: %w", err)
	}
	if err := requireRowsAffected(result, "update", question.ID); err != nil {
		return err
	}
	return nil
}

// Delete removes a Question. The question migration owns dependent generic-row
// cleanup triggers because those polymorphic tables cannot express foreign keys.
func (r *QuestionRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM questions WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete question: %w", err)
	}
	return requireRowsAffected(result, "delete", id)
}

// UpdateStatus updates only the stored base status.
func (r *QuestionRepository) UpdateStatus(ctx context.Context, id int64, status models.QuestionStatus) error {
	result, err := r.db.ExecContext(ctx, "UPDATE questions SET status = ? WHERE id = ?", status, id)
	if err != nil {
		return fmt.Errorf("update question status: %w", err)
	}
	return requireRowsAffected(result, "update status", id)
}

// UpdateStatusIfCurrent atomically updates status if it still equals expected.
func (r *QuestionRepository) UpdateStatusIfCurrent(ctx context.Context, id int64, expected, next models.QuestionStatus) (bool, error) {
	updated, err := dbconn.ConditionalStatusUpdate(ctx, r.db, "questions", id, string(expected), string(next), false)
	if err != nil {
		return false, fmt.Errorf("conditionally update question status: %w", err)
	}
	return updated, nil
}

// ConfigureWorkflow atomically persists an already-validated initial
// QuestionState and its concise audit evidence. The service owns validation;
// this repository method owns the parameterized state/note/history transaction.
func (r *QuestionRepository) ConfigureWorkflow(ctx context.Context, id int64, expectedStatus models.QuestionStatus, expectedContextData, contextData *string, resolutionOwner string) error {
	if contextData == nil {
		return errors.New("configure Question workflow: context data is required")
	}
	tx, err := r.db.BeginTxContext(ctx)
	if err != nil {
		return fmt.Errorf("begin configure Question workflow transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	result, err := tx.ExecContext(ctx, `
		UPDATE questions SET context_data = ?, status = 'open'
		WHERE id = ? AND status = ? AND context_data IS ?
	`, contextData, id, expectedStatus, expectedContextData)
	if err != nil {
		return fmt.Errorf("persist configured Question state: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read configured Question state rows: %w", err)
	}
	if rows != 1 {
		return errors.New("configure Question workflow: Question changed or is already configured")
	}
	note := "Question workflow configured; responder identities are retained in Question-owned state."
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO entity_notes (entity_type, entity_id, note_type, content, created_by)
		VALUES ('question', ?, 'implementation', ?, ?)
	`, id, note, resolutionOwner); err != nil {
		return fmt.Errorf("record configured Question note: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO entity_history (entity_type, entity_id, to_status, changed_by, notes, forced, changed_at)
		VALUES ('question', ?, 'open', ?, ?, 0, CURRENT_TIMESTAMP)
	`, id, resolutionOwner, note); err != nil {
		return fmt.Errorf("record configured Question history: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit configured Question workflow: %w", err)
	}
	return nil
}

// RecordResponse commits the serial state, bounded audit note, and history
// together. The service has already validated the caller's active lease and
// response data; this method owns the durable all-or-nothing boundary.
func (r *QuestionRepository) RecordResponse(ctx context.Context, id int64, expectedStatus, status models.QuestionStatus, expectedContextData, contextData *string, responder string) error {
	if contextData == nil {
		return errors.New("record Question response: context data is required")
	}
	tx, err := r.db.BeginTxContext(ctx)
	if err != nil {
		return fmt.Errorf("begin record Question response transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, `UPDATE questions SET context_data = ?, status = ? WHERE id = ? AND status = ? AND context_data IS ?`, contextData, status, id, expectedStatus, expectedContextData)
	if err != nil {
		return fmt.Errorf("persist Question response state: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read Question response rows: %w", err)
	}
	if rows != 1 {
		return errors.New("record Question response: Question changed or is not answerable")
	}
	note := "Question response recorded for configured responder."
	if _, err := tx.ExecContext(ctx, `INSERT INTO entity_notes (entity_type, entity_id, note_type, content, created_by) VALUES ('question', ?, 'implementation', ?, ?)`, id, note, responder); err != nil {
		return fmt.Errorf("record Question response note: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO entity_history (entity_type, entity_id, to_status, changed_by, notes, forced, changed_at) VALUES ('question', ?, ?, ?, ?, 0, CURRENT_TIMESTAMP)`, id, status, responder, note); err != nil {
		return fmt.Errorf("record Question response history: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit Question response: %w", err)
	}
	return nil
}

// FollowUpWorkExists reports whether key identifies a durable Shark work item.
func (r *QuestionRepository) FollowUpWorkExists(ctx context.Context, key string) (bool, error) {
	var found bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(
		SELECT 1 FROM epics WHERE key = ? UNION ALL SELECT 1 FROM features WHERE key = ?
		UNION ALL SELECT 1 FROM tasks WHERE key = ? UNION ALL SELECT 1 FROM bugs WHERE key = ?
		UNION ALL SELECT 1 FROM change_cards WHERE key = ? UNION ALL SELECT 1 FROM tech_debts WHERE key = ?
	)`, key, key, key, key, key, key).Scan(&found)
	if err != nil {
		return false, fmt.Errorf("check follow-up work destination: %w", err)
	}
	return found, nil
}

// NoteExists reports whether noteID identifies a durable entity note.
func (r *QuestionRepository) NoteExists(ctx context.Context, noteID string) (bool, error) {
	var found bool
	if err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM entity_notes WHERE id = ?)", noteID).Scan(&found); err != nil {
		return false, fmt.Errorf("check local clarification note: %w", err)
	}
	return found, nil
}

// Resolve atomically persists classified resolution state, concise audit
// evidence, history, and the conditional terminal status.
func (r *QuestionRepository) Resolve(ctx context.Context, id int64, expectedStatus, status models.QuestionStatus, expectedContextData, contextData *string, owner, kind string) error {
	return r.close(ctx, id, expectedStatus, status, expectedContextData, contextData, owner, "Question resolved with classified "+kind+" provenance.")
}

// Withdraw atomically persists withdrawal or supersession provenance, concise
// audit evidence, history, and a non-terminal conditional status update.
func (r *QuestionRepository) Withdraw(ctx context.Context, id int64, expectedStatus, status models.QuestionStatus, expectedContextData, contextData *string, owner, reason string) error {
	return r.close(ctx, id, expectedStatus, status, expectedContextData, contextData, owner, "Question terminal provenance recorded.")
}

func (r *QuestionRepository) close(ctx context.Context, id int64, expected, status models.QuestionStatus, expectedContextData, contextData *string, owner, note string) error {
	if contextData == nil {
		return errors.New("close Question: context data is required")
	}
	tx, err := r.db.BeginTxContext(ctx)
	if err != nil {
		return fmt.Errorf("begin close Question transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, `UPDATE questions SET context_data = ?, status = ? WHERE id = ? AND status = ? AND context_data IS ?`, contextData, status, id, expected, expectedContextData)
	if err != nil {
		return fmt.Errorf("persist terminal Question state: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read terminal Question rows: %w", err)
	}
	if rows != 1 {
		return errors.New("close Question: Question is not eligible for terminal operation")
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO entity_notes (entity_type, entity_id, note_type, content, created_by) VALUES ('question', ?, 'implementation', ?, ?)`, id, note, owner); err != nil {
		return fmt.Errorf("record terminal Question note: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO entity_history (entity_type, entity_id, to_status, changed_by, notes, forced, changed_at) VALUES ('question', ?, ?, ?, ?, 0, CURRENT_TIMESTAMP)`, id, status, owner, note); err != nil {
		return fmt.Errorf("record terminal Question history: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit terminal Question: %w", err)
	}
	return nil
}

// List returns the finite Question page in canonical key order.
func (r *QuestionRepository) List(ctx context.Context, filter QuestionListFilter) ([]*models.Question, error) {
	limit := filter.Limit
	if limit == 0 {
		limit = 50
	}
	if limit < 1 || limit > 100 {
		return nil, fmt.Errorf("list questions: limit must be between 1 and 100, got %d", limit)
	}
	if filter.Offset < 0 {
		return nil, fmt.Errorf("list questions: offset must be zero or greater, got %d", filter.Offset)
	}

	query := fmt.Sprintf("SELECT %s FROM questions", questionSelectColumns)
	conditions := make([]string, 0, 3)
	args := make([]any, 0, 5)
	if filter.Status != nil {
		conditions = append(conditions, "status = ?")
		args = append(args, *filter.Status)
	}
	if filter.Requester != nil {
		conditions = append(conditions, "requester = ?")
		args = append(args, *filter.Requester)
	}
	if filter.Blocking != nil {
		conditions = append(conditions, "blocking = ?")
		args = append(args, *filter.Blocking)
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY key ASC LIMIT ? OFFSET ?"
	args = append(args, limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list questions: %w", err)
	}
	defer rows.Close()

	questions := make([]*models.Question, 0)
	for rows.Next() {
		question, scanErr := scanQuestion(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan question list row: %w", scanErr)
		}
		questions = append(questions, question)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate question list: %w", err)
	}
	return questions, nil
}

// ListOpenCandidates returns the bounded persisted candidates for the focused
// responder read. It deliberately does not inspect Question-owned state: the
// service validates that state and derives the current responder.
func (r *QuestionRepository) ListOpenCandidates(ctx context.Context, limit, offset int) ([]*models.Question, error) {
	if limit < 1 || limit > 100 {
		return nil, fmt.Errorf("list open Question candidates: limit must be between 1 and 100, got %d", limit)
	}
	if offset < 0 {
		return nil, fmt.Errorf("list open Question candidates: offset must be zero or greater, got %d", offset)
	}

	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT %s FROM questions
		WHERE status IN ('open', 'answering')
		ORDER BY key ASC
		LIMIT ? OFFSET ?`, questionSelectColumns), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list open Question candidates: %w", err)
	}
	defer rows.Close()

	questions := make([]*models.Question, 0)
	for rows.Next() {
		question, scanErr := scanQuestion(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan open Question candidate: %w", scanErr)
		}
		questions = append(questions, question)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate open Question candidates: %w", err)
	}
	return questions, nil
}

func requireRowsAffected(result sql.Result, operation string, id int64) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s question: read rows affected: %w", operation, err)
	}
	if rows == 0 {
		return fmt.Errorf("question not found with ID %d: %w", id, repoerr.ErrNotFound)
	}
	return nil
}
