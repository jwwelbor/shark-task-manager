package question_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	questionrepo "github.com/jwwelbor/shark-task-manager/internal/repository/question"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestQuestionServiceDeleteQuestionRollsBackAllAssociationsWhenCleanupTriggerFails
// verifies the production QuestionService.DeleteQuestion caller path. A real
// SQLite cleanup-target trigger aborts the parent delete, so the test protects
// both the service lookup/delete orchestration and atomic restoration of every
// Question-owned polymorphic association.
func TestQuestionServiceDeleteQuestionRollsBackAllAssociationsWhenCleanupTriggerFails(t *testing.T) {
	ctx := context.Background()
	database := test.NewIsolatedTestDB(t)
	repo := questionrepo.NewQuestionRepository(dbconn.NewDB(database))
	svc, err := services.NewQuestionService(repo)
	if err != nil {
		t.Fatalf("NewQuestionService() error = %v", err)
	}

	question := createQuestion(t, ctx, svc, "Question Q001", "alice")
	if question.Key != "Q001" {
		t.Fatalf("CreateQuestion() key = %q, want Q001", question.Key)
	}
	relatedQuestion := createQuestion(t, ctx, svc, "Question Q002", "bob")
	seedQuestionDependentAssociations(t, ctx, database, question, relatedQuestion)

	beforeQuestion := snapshotQuestion(t, ctx, database, question.Key)
	beforeAssociations := countQuestionDependentAssociations(t, ctx, database, question)
	wantAssociations := questionAssociationCounts{
		notes: 1, history: 1, documents: 1, relationships: 2,
		tags: 1, claims: 1, workSessions: 1, advanceGuards: 1,
	}
	if beforeAssociations != wantAssociations {
		t.Fatalf("seeded Question associations = %#v, want %#v", beforeAssociations, wantAssociations)
	}

	// This deliberately injects a real SQLite failure beneath the Question
	// cleanup trigger. RAISE(ABORT) must undo the parent DELETE and cleanup
	// writes regardless of the order SQLite executes sibling cleanup triggers.
	if _, err := database.ExecContext(ctx, `
		CREATE TRIGGER question_test_note_cleanup_failure
		BEFORE DELETE ON entity_notes
		WHEN OLD.entity_type = 'question'
		BEGIN
			SELECT RAISE(ABORT, 'injected Question cleanup trigger failure');
		END;`); err != nil {
		t.Fatalf("create failing cleanup trigger: %v", err)
	}

	err = svc.DeleteQuestion(ctx, "Q001")
	if err == nil || !strings.Contains(err.Error(), "injected Question cleanup trigger failure") {
		t.Fatalf("DeleteQuestion(Q001) error = %v, want injected cleanup-trigger failure", err)
	}

	if afterQuestion := snapshotQuestion(t, ctx, database, "Q001"); afterQuestion != beforeQuestion {
		t.Errorf("failed DeleteQuestion(Q001) changed Question: before=%#v after=%#v", beforeQuestion, afterQuestion)
	}
	if afterAssociations := countQuestionDependentAssociations(t, ctx, database, question); afterAssociations != beforeAssociations {
		t.Errorf("failed DeleteQuestion(Q001) changed associations: before=%#v after=%#v", beforeAssociations, afterAssociations)
	}
}

func createQuestion(t *testing.T, ctx context.Context, svc *services.QuestionService, title, requester string) *models.Question {
	t.Helper()
	question, err := svc.CreateQuestion(ctx, services.CreateQuestionInput{
		Title: title, Summary: "A bounded question summary", Requester: requester,
	})
	if err != nil {
		t.Fatalf("CreateQuestion(%q) error = %v", title, err)
	}
	return question
}

type questionSnapshot struct {
	id        int64
	key       string
	title     string
	status    string
	summary   string
	blocking  bool
	requester string
}

func snapshotQuestion(t *testing.T, ctx context.Context, database *sql.DB, key string) questionSnapshot {
	t.Helper()
	var snapshot questionSnapshot
	if err := database.QueryRowContext(ctx, `
		SELECT id, key, title, status, summary, blocking, requester
		FROM questions WHERE key = ?`, key).Scan(
		&snapshot.id, &snapshot.key, &snapshot.title, &snapshot.status,
		&snapshot.summary, &snapshot.blocking, &snapshot.requester,
	); err != nil {
		t.Fatalf("snapshot Question %s: %v", key, err)
	}
	return snapshot
}

type questionAssociationCounts struct {
	notes         int
	history       int
	documents     int
	relationships int
	tags          int
	claims        int
	workSessions  int
	advanceGuards int
}

func seedQuestionDependentAssociations(t *testing.T, ctx context.Context, database *sql.DB, question, relatedQuestion *models.Question) {
	t.Helper()

	result, err := database.ExecContext(ctx, "INSERT INTO documents (title, file_path) VALUES ('question-test-document', '/tmp/question-test-document')")
	if err != nil {
		t.Fatalf("seed document: %v", err)
	}
	documentID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("seed document ID: %v", err)
	}
	result, err = database.ExecContext(ctx, "INSERT INTO tags (name) VALUES ('question-test-tag')")
	if err != nil {
		t.Fatalf("seed tag: %v", err)
	}
	tagID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("seed tag ID: %v", err)
	}

	for _, insert := range []struct {
		query string
		args  []any
	}{
		{"INSERT INTO entity_notes (entity_type, entity_id, note_type, content) VALUES ('question', ?, 'comment', 'note')", []any{question.ID}},
		{"INSERT INTO entity_history (entity_type, entity_id, to_status) VALUES ('question', ?, 'draft')", []any{question.ID}},
		{"INSERT INTO entity_documents (entity_type, entity_id, document_id) VALUES ('question', ?, ?)", []any{question.ID, documentID}},
		{"INSERT INTO entity_relationships (from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type) VALUES ('question', ?, 'question', ?, 'linked_to')", []any{question.ID, relatedQuestion.ID}},
		{"INSERT INTO entity_relationships (from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type) VALUES ('question', ?, 'question', ?, 'linked_to')", []any{relatedQuestion.ID, question.ID}},
		{"INSERT INTO entity_tags (entity_type, entity_id, tag_id) VALUES ('question', ?, ?)", []any{question.ID, tagID}},
		{"INSERT INTO entity_claims (entity_type, entity_key, claimed_by, session_id) VALUES ('question', ?, 'test', 'question-test-claim')", []any{question.Key}},
		{"INSERT INTO work_sessions (entity_type, entity_key, agent_id, session_id, started_at) VALUES ('question', ?, 'test', 'question-test-session', CURRENT_TIMESTAMP)", []any{question.Key}},
		{"INSERT INTO advance_guard_consumptions (entity_type, entity_id, session_id, from_status, outcome) VALUES ('question', ?, 'question-test-guard', 'draft', 'accepted')", []any{question.ID}},
	} {
		if _, err := database.ExecContext(ctx, insert.query, insert.args...); err != nil {
			t.Fatalf("seed association %q: %v", insert.query, err)
		}
	}
}

func countQuestionDependentAssociations(t *testing.T, ctx context.Context, database *sql.DB, question *models.Question) questionAssociationCounts {
	t.Helper()
	counts := questionAssociationCounts{}
	for _, query := range []struct {
		name string
		stmt string
		args []any
		to   *int
	}{
		{"notes", "SELECT COUNT(*) FROM entity_notes WHERE entity_type = 'question' AND entity_id = ?", []any{question.ID}, &counts.notes},
		{"history", "SELECT COUNT(*) FROM entity_history WHERE entity_type = 'question' AND entity_id = ?", []any{question.ID}, &counts.history},
		{"documents", "SELECT COUNT(*) FROM entity_documents WHERE entity_type = 'question' AND entity_id = ?", []any{question.ID}, &counts.documents},
		{"relationships", "SELECT COUNT(*) FROM entity_relationships WHERE (from_entity_type = 'question' AND from_entity_id = ?) OR (to_entity_type = 'question' AND to_entity_id = ?)", []any{question.ID, question.ID}, &counts.relationships},
		{"tags", "SELECT COUNT(*) FROM entity_tags WHERE entity_type = 'question' AND entity_id = ?", []any{question.ID}, &counts.tags},
		{"claims", "SELECT COUNT(*) FROM entity_claims WHERE entity_type = 'question' AND entity_key = ?", []any{question.Key}, &counts.claims},
		{"work sessions", "SELECT COUNT(*) FROM work_sessions WHERE entity_type = 'question' AND entity_key = ?", []any{question.Key}, &counts.workSessions},
		{"advance guards", "SELECT COUNT(*) FROM advance_guard_consumptions WHERE entity_type = 'question' AND entity_id = ?", []any{question.ID}, &counts.advanceGuards},
	} {
		if err := database.QueryRowContext(ctx, query.stmt, query.args...).Scan(query.to); err != nil {
			t.Fatalf("count %s: %v", query.name, err)
		}
	}
	return counts
}
