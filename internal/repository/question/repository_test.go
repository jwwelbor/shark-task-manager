package question

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	repoerr "github.com/jwwelbor/shark-task-manager/internal/repository/repoerr"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

func questionTestRepository(t *testing.T) *QuestionRepository {
	t.Helper()
	database := test.NewIsolatedTestDB(t)
	return NewQuestionRepository(dbconn.NewDB(database))
}

func testQuestion(key, requester string, blocking bool) *models.Question {
	return &models.Question{
		BaseEntity: models.BaseEntity{
			Key:         key,
			Title:       "Question " + key,
			Description: test.StringPtr("optional description"),
		},
		Status:    models.QuestionStatusDraft,
		Summary:   "A bounded question summary",
		Requester: requester,
		Blocking:  blocking,
	}
}

// questionAssociationCounts records every polymorphic association that the
// Question delete triggers own. Keeping this complete makes the success and
// rollback cases protect the same schema contract.
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

func seedQuestionDependentAssociations(t *testing.T, ctx context.Context, database interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, question, relatedQuestion *models.Question) {
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

func countQuestionDependentAssociations(t *testing.T, ctx context.Context, database interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, question *models.Question) questionAssociationCounts {
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

func TestQuestionRepositoryCreateAllocatesAndPersistsBaseRecord(t *testing.T) {
	ctx := context.Background()
	repo := questionTestRepository(t)
	question := testQuestion("", "alice", true)

	if err := repo.Create(ctx, question); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if question.ID == 0 {
		t.Fatal("Create() did not assign an ID")
	}
	if question.Key != "Q001" {
		t.Fatalf("Create() key = %q, want Q001", question.Key)
	}

	got, err := repo.GetByKey(ctx, "Q001")
	if err != nil {
		t.Fatalf("GetByKey() error = %v", err)
	}
	if got.Title != question.Title || got.Summary != question.Summary || got.Requester != "alice" || !got.Blocking {
		t.Errorf("GetByKey() = %#v, want persisted Question fields", got)
	}
	if got.Description == nil || *got.Description != "optional description" {
		t.Errorf("GetByKey() description = %v, want optional description", got.Description)
	}
}

func TestQuestionRepositoryCreateRejectsDuplicateAndExhaustedKeyspace(t *testing.T) {
	ctx := context.Background()
	repo := questionTestRepository(t)

	if err := repo.Create(ctx, testQuestion("Q001", "alice", false)); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}
	duplicate := testQuestion("Q001", "bob", false)
	if err := repo.Create(ctx, duplicate); err == nil || !strings.Contains(strings.ToLower(err.Error()), "unique") {
		t.Fatalf("Create(duplicate) error = %v, want unique constraint error", err)
	}
	if duplicate.ID != 0 {
		t.Errorf("Create(duplicate) assigned unpersisted ID %d", duplicate.ID)
	}

	if err := repo.Create(ctx, testQuestion("Q999", "alice", false)); err != nil {
		t.Fatalf("Create(Q999) error = %v", err)
	}
	exhausted := testQuestion("", "alice", false)
	if err := repo.Create(ctx, exhausted); err == nil || !strings.Contains(strings.ToLower(err.Error()), "allocate") {
		t.Fatalf("Create(exhausted) error = %v, want allocation error", err)
	}
	if exhausted.ID != 0 || exhausted.Key != "" {
		t.Errorf("Create(exhausted) mutated unpersisted identity: ID=%d Key=%q", exhausted.ID, exhausted.Key)
	}
}

// TestQuestionRepositoryCreateConcurrentAllocation_TC001 executes the real
// repository allocation path from two goroutines against one initialized
// SQLite database. The unique key constraint is the concurrency authority:
// both calls must persist distinct keys, or a losing call must surface the
// constraint as an allocation failure without receiving an identity.
func TestQuestionRepositoryCreateConcurrentAllocation_TC001(t *testing.T) {
	ctx := context.Background()
	database := test.NewIsolatedTestDB(t)
	repos := [2]*QuestionRepository{
		NewQuestionRepository(dbconn.NewDB(database)),
		NewQuestionRepository(dbconn.NewDB(database)),
	}

	type result struct {
		question *models.Question
		err      error
	}
	start := make(chan struct{})
	results := make(chan result, len(repos))
	var ready sync.WaitGroup
	ready.Add(len(repos))
	for i, repo := range repos {
		go func(i int, repo *QuestionRepository) {
			question := testQuestion("", "concurrent-requester", false)
			question.Title = "Concurrent Question " + string(rune('A'+i))
			ready.Done()
			<-start
			results <- result{question: question, err: repo.Create(ctx, question)}
		}(i, repo)
	}
	ready.Wait()
	close(start)

	persisted := make(map[string]int64, len(repos))
	for range repos {
		outcome := <-results
		if outcome.err != nil {
			t.Fatalf("concurrent Create() error = %v", outcome.err)
		}
		if outcome.question.ID == 0 {
			t.Fatal("successful concurrent Create() did not assign an ID")
		}
		if err := models.ValidateQuestionKey(outcome.question.Key); err != nil {
			t.Fatalf("successful concurrent Create() key = %q: %v", outcome.question.Key, err)
		}
		if priorID, duplicate := persisted[outcome.question.Key]; duplicate {
			t.Fatalf("concurrent Create() reused key %q for IDs %d and %d", outcome.question.Key, priorID, outcome.question.ID)
		}
		persisted[outcome.question.Key] = outcome.question.ID
		stored, err := repos[0].GetByKey(ctx, outcome.question.Key)
		if err != nil {
			t.Fatalf("GetByKey(%q) after concurrent Create(): %v", outcome.question.Key, err)
		}
		if stored.ID != outcome.question.ID {
			t.Fatalf("GetByKey(%q).ID = %d, want %d", outcome.question.Key, stored.ID, outcome.question.ID)
		}
	}
	if len(persisted) != len(repos) {
		t.Fatalf("concurrent Create() persisted %d records, want %d", len(persisted), len(repos))
	}
}

func TestQuestionRepositoryGetByKeyIsStrictAndNotFoundIsTyped(t *testing.T) {
	ctx := context.Background()
	repo := questionTestRepository(t)
	if err := repo.Create(ctx, testQuestion("Q001", "alice", false)); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if _, err := repo.GetByKey(ctx, "q001"); err == nil {
		t.Fatal("GetByKey(q001) error = nil, want strict canonical key rejection")
	}
	_, err := repo.GetByKey(ctx, "Q002")
	if !errors.Is(err, repoerr.ErrNotFound) {
		t.Fatalf("GetByKey(Q002) error = %v, want repository not found", err)
	}
}

// TestQuestionRepositoryFollowUpWorkExists locks in both sides of the
// follow_up_work resolution-destination check QuestionService.
// validateResolutionDestination depends on: a nonexistent key must report
// false (not error), and a real Shark work item must report true.
func TestQuestionRepositoryFollowUpWorkExists(t *testing.T) {
	ctx := context.Background()
	database := test.NewIsolatedTestDB(t)
	repo := NewQuestionRepository(dbconn.NewDB(database))

	found, err := repo.FollowUpWorkExists(ctx, "E99")
	if err != nil {
		t.Fatalf("FollowUpWorkExists(nonexistent) error = %v", err)
	}
	if found {
		t.Error("FollowUpWorkExists(nonexistent) = true, want false")
	}

	if _, err := database.Exec(`INSERT INTO epics (key, title, status, priority) VALUES ('E99', 'Test Epic', 'active', 'medium')`); err != nil {
		t.Fatalf("seed epic: %v", err)
	}
	found, err = repo.FollowUpWorkExists(ctx, "E99")
	if err != nil {
		t.Fatalf("FollowUpWorkExists(epic) error = %v", err)
	}
	if !found {
		t.Error("FollowUpWorkExists(epic) = false, want true")
	}
}

// TestQuestionRepositoryNoteExists locks in both sides of the
// local_clarification resolution-destination check QuestionService.
// validateResolutionDestination depends on: a nonexistent note ID must
// report false (not error), and a real note must report true.
func TestQuestionRepositoryNoteExists(t *testing.T) {
	ctx := context.Background()
	database := test.NewIsolatedTestDB(t)
	repo := NewQuestionRepository(dbconn.NewDB(database))

	found, err := repo.NoteExists(ctx, "999999")
	if err != nil {
		t.Fatalf("NoteExists(nonexistent) error = %v", err)
	}
	if found {
		t.Error("NoteExists(nonexistent) = true, want false")
	}

	res, err := database.Exec(`INSERT INTO entity_notes (entity_type, entity_id, note_type, content, created_by) VALUES ('question', 1, 'comment', 'clarified inline', 'tester')`)
	if err != nil {
		t.Fatalf("seed note: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("read seeded note id: %v", err)
	}
	found, err = repo.NoteExists(ctx, fmt.Sprintf("%d", id))
	if err != nil {
		t.Fatalf("NoteExists(note) error = %v", err)
	}
	if !found {
		t.Error("NoteExists(note) = false, want true")
	}
}

func TestQuestionRepositoryListUsesBoundedExactFiltersAndKeyOrder(t *testing.T) {
	ctx := context.Background()
	repo := questionTestRepository(t)
	for _, question := range []*models.Question{
		testQuestion("Q001", "alice", true),
		testQuestion("Q002", "alice", false),
		testQuestion("Q003", "bob", true),
	} {
		if err := repo.Create(ctx, question); err != nil {
			t.Fatalf("Create(%s) error = %v", question.Key, err)
		}
	}

	status := models.QuestionStatusDraft
	requester := "alice"
	blocking := true
	items, err := repo.List(ctx, QuestionListFilter{
		Status:    &status,
		Requester: &requester,
		Blocking:  &blocking,
		Limit:     1,
		Offset:    0,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 || items[0].Key != "Q001" {
		t.Fatalf("List() = %#v, want Q001 only", items)
	}

	if _, err := repo.List(ctx, QuestionListFilter{Limit: 101}); err == nil {
		t.Fatal("List(limit=101) error = nil, want bounded limit error")
	}
	if _, err := repo.List(ctx, QuestionListFilter{Limit: 1, Offset: -1}); err == nil {
		t.Fatal("List(offset=-1) error = nil, want offset error")
	}
}

// TC-402: focused responder reads must receive a finite, canonical-key ordered
// candidate set. State validation and responder selection remain service work.
func TestQuestionRepositoryListOpenCandidatesTC402UsesFiniteOpenPage(t *testing.T) {
	ctx := context.Background()
	repo := questionTestRepository(t)
	for _, question := range []*models.Question{
		testQuestion("Q003", "owner", true),
		testQuestion("Q001", "owner", true),
		testQuestion("Q002", "owner", true),
		testQuestion("Q004", "owner", true),
	} {
		if err := repo.Create(ctx, question); err != nil {
			t.Fatalf("Create(%s) error = %v", question.Key, err)
		}
	}
	for _, update := range []struct {
		key    string
		status models.QuestionStatus
	}{
		{"Q001", models.QuestionStatusOpen},
		{"Q002", models.QuestionStatus("answering")},
		{"Q003", models.QuestionStatus("ready")},
		{"Q004", models.QuestionStatus("resolved")},
	} {
		question, err := repo.GetByKey(ctx, update.key)
		if err != nil {
			t.Fatalf("GetByKey(%s) error = %v", update.key, err)
		}
		if err := repo.UpdateStatus(ctx, question.ID, update.status); err != nil {
			t.Fatalf("UpdateStatus(%s) error = %v", update.key, err)
		}
	}

	items, err := repo.ListOpenCandidates(ctx, 1, 1)
	if err != nil {
		t.Fatalf("ListOpenCandidates() error = %v", err)
	}
	if len(items) != 1 || items[0].Key != "Q002" {
		t.Fatalf("ListOpenCandidates(limit=1, offset=1) = %#v, want Q002 only", items)
	}

	for _, filter := range []struct{ limit, offset int }{{0, 0}, {101, 0}, {1, -1}} {
		if _, err := repo.ListOpenCandidates(ctx, filter.limit, filter.offset); err == nil {
			t.Errorf("ListOpenCandidates(%d, %d) error = nil, want finite page validation", filter.limit, filter.offset)
		}
	}
}

func TestQuestionRepositoryUpdateTouchesTimestampAndDeleteCleansAssociations(t *testing.T) {
	ctx := context.Background()
	database := test.NewIsolatedTestDB(t)
	repo := NewQuestionRepository(dbconn.NewDB(database))
	question := testQuestion("Q001", "alice", false)
	if err := repo.Create(ctx, question); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	question.Title = "Updated Question"
	question.Blocking = true
	if err := repo.Update(ctx, question); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	updated, err := repo.GetByKey(ctx, "Q001")
	if err != nil {
		t.Fatalf("GetByKey(after update) error = %v", err)
	}
	if updated.Title != "Updated Question" || !updated.Blocking {
		t.Errorf("Update() did not persist fields: %#v", updated)
	}
	relatedQuestion := testQuestion("Q002", "bob", false)
	if err := repo.Create(ctx, relatedQuestion); err != nil {
		t.Fatalf("Create(related question): %v", err)
	}
	seedQuestionDependentAssociations(t, ctx, database, question, relatedQuestion)
	if err := repo.Delete(ctx, question.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if counts := countQuestionDependentAssociations(t, ctx, database, question); counts != (questionAssociationCounts{}) {
		t.Errorf("Delete() left Question associations: %#v", counts)
	}
}

// TC-102: configuration is one transaction: a second attempt does not replace
// state or append a typed note/history row.
func TestQuestionRepositoryConfigureWorkflowIsAtomicAndOneTime_TC102(t *testing.T) {
	ctx := context.Background()
	database := test.NewIsolatedTestDB(t)
	repo := NewQuestionRepository(dbconn.NewDB(database))
	question := testQuestion("Q001", "owner", false)
	question.Status = models.QuestionStatusOpen
	if err := repo.Create(ctx, question); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	state := models.QuestionState{ResolutionOwner: "release-owner", Responders: []models.QuestionResponder{{Identity: "alice", Status: models.QuestionResponderPending}}}
	encoded, err := models.EncodeQuestionState(question.ContextData, state)
	if err != nil {
		t.Fatalf("EncodeQuestionState() error = %v", err)
	}
	if err := repo.ConfigureWorkflow(ctx, question.ID, models.QuestionStatusOpen, nil, encoded, "release-owner"); err != nil {
		t.Fatalf("TC-102 ConfigureWorkflow() error = %v", err)
	}
	stored, err := repo.GetByKey(ctx, "Q001")
	if err != nil {
		t.Fatalf("GetByKey() error = %v", err)
	}
	if stored.ContextData == nil || *stored.ContextData != *encoded {
		t.Fatalf("configured context = %v, want %q", stored.ContextData, *encoded)
	}
	var notes, history int
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM entity_notes WHERE entity_type = 'question' AND entity_id = ?`, question.ID).Scan(&notes); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM entity_history WHERE entity_type = 'question' AND entity_id = ?`, question.ID).Scan(&history); err != nil {
		t.Fatal(err)
	}
	if notes != 1 || history != 1 {
		t.Fatalf("TC-102 audit rows = notes:%d history:%d, want 1 each", notes, history)
	}
	if err := repo.ConfigureWorkflow(ctx, question.ID, models.QuestionStatusOpen, nil, encoded, "release-owner"); err == nil {
		t.Fatal("TC-102 second ConfigureWorkflow() error = nil")
	}
	var afterNotes, afterHistory int
	_ = database.QueryRowContext(ctx, `SELECT COUNT(*) FROM entity_notes WHERE entity_type = 'question' AND entity_id = ?`, question.ID).Scan(&afterNotes)
	_ = database.QueryRowContext(ctx, `SELECT COUNT(*) FROM entity_history WHERE entity_type = 'question' AND entity_id = ?`, question.ID).Scan(&afterHistory)
	if afterNotes != notes || afterHistory != history {
		t.Fatalf("second configuration changed audit rows: notes %d->%d history %d->%d", notes, afterNotes, history, afterHistory)
	}
}

// TC-102: a failure after the state and typed-note writes must roll the whole
// configuration transaction back. This uses the production repository against
// SQLite rather than a mock so the transaction boundary is the one callers use.
func TestQuestionRepositoryConfigureWorkflowRollsBackStateAndAuditOnHistoryFailure_TC102(t *testing.T) {
	ctx := context.Background()
	database := test.NewIsolatedTestDB(t)
	repo := NewQuestionRepository(dbconn.NewDB(database))
	question := testQuestion("Q001", "owner", false)
	question.Status = models.QuestionStatusOpen
	if err := repo.Create(ctx, question); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	state := models.QuestionState{
		ResolutionOwner: "release-owner",
		Responders:      []models.QuestionResponder{{Identity: "alice", Status: models.QuestionResponderPending}},
	}
	encoded, err := models.EncodeQuestionState(question.ContextData, state)
	if err != nil {
		t.Fatalf("EncodeQuestionState() error = %v", err)
	}
	_, err = database.ExecContext(ctx, `
		CREATE TRIGGER reject_question_configuration_history
		BEFORE INSERT ON entity_history
		WHEN NEW.entity_type = 'question' AND NEW.entity_id = `+fmt.Sprint(question.ID)+`
		BEGIN
			SELECT RAISE(ABORT, 'reject Question configuration history');
		END`)
	if err != nil {
		t.Fatalf("create history rejection trigger: %v", err)
	}

	err = repo.ConfigureWorkflow(ctx, question.ID, models.QuestionStatusOpen, nil, encoded, "release-owner")
	if err == nil {
		t.Fatal("TC-102 ConfigureWorkflow() error = nil, want history write rejection")
	}
	stored, err := repo.GetByKey(ctx, question.Key)
	if err != nil {
		t.Fatalf("GetByKey() error = %v", err)
	}
	if stored.ContextData != nil {
		t.Fatalf("TC-102 rejected configuration persisted context data = %q", *stored.ContextData)
	}
	var notes, history int
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM entity_notes WHERE entity_type = 'question' AND entity_id = ?`, question.ID).Scan(&notes); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM entity_history WHERE entity_type = 'question' AND entity_id = ?`, question.ID).Scan(&history); err != nil {
		t.Fatal(err)
	}
	if notes != 0 || history != 0 {
		t.Fatalf("TC-102 rejected configuration audit rows = notes:%d history:%d, want 0 each", notes, history)
	}
}

// TC-105: response state and both audit rows share one repository transaction.
func TestQuestionRepositoryRecordResponseRollsBackOnHistoryFailure_TC105(t *testing.T) {
	ctx := context.Background()
	database := test.NewIsolatedTestDB(t)
	repo := NewQuestionRepository(dbconn.NewDB(database))
	question := testQuestion("Q001", "owner", false)
	question.Status = models.QuestionStatus("answering")
	if err := repo.Create(ctx, question); err != nil {
		t.Fatal(err)
	}
	state := models.QuestionState{ResolutionOwner: "release-owner", Responders: []models.QuestionResponder{{Identity: "alice", Status: models.QuestionResponderCompleted}}, Responses: []models.QuestionResponse{{SessionID: "session-a", Responder: "alice", Summary: "approved", EvidencePointer: "docs/spec.md", RecordedAt: time.Now().UTC()}}}
	encoded, err := models.EncodeQuestionState(nil, state)
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.ExecContext(ctx, `CREATE TRIGGER reject_question_response_history BEFORE INSERT ON entity_history WHEN NEW.entity_type = 'question' AND NEW.entity_id = `+fmt.Sprint(question.ID)+` BEGIN SELECT RAISE(ABORT, 'reject Question response history'); END`)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.RecordResponse(ctx, question.ID, "open", "ready_for_resolution", nil, encoded, "alice"); err == nil {
		t.Fatal("TC-105 RecordResponse() error = nil")
	}
	stored, err := repo.GetByKey(ctx, question.Key)
	if err != nil {
		t.Fatal(err)
	}
	if stored.ContextData != nil || stored.Status != "answering" {
		t.Fatalf("TC-105 rollback stored = %#v", stored)
	}
	var notes, history int
	_ = database.QueryRowContext(ctx, `SELECT COUNT(*) FROM entity_notes WHERE entity_type = 'question' AND entity_id = ?`, question.ID).Scan(&notes)
	_ = database.QueryRowContext(ctx, `SELECT COUNT(*) FROM entity_history WHERE entity_type = 'question' AND entity_id = ?`, question.ID).Scan(&history)
	if notes != 0 || history != 0 {
		t.Fatalf("TC-105 rollback audit = %d notes, %d history", notes, history)
	}
}

// TC-105: two responders can observe the same answering snapshot, but only
// one response transaction may consume it. The status remains answering after
// each non-final response, so this test specifically proves the context
// snapshot predicate rather than relying on a status transition.
func TestQuestionRepositoryRecordResponseRejectsStaleAnsweringSnapshot_TC105(t *testing.T) {
	ctx := context.Background()
	database := test.NewIsolatedTestDB(t)
	repo := NewQuestionRepository(dbconn.NewDB(database))
	question := testQuestion("Q001", "owner", false)
	question.Status = "answering"
	if err := repo.Create(ctx, question); err != nil {
		t.Fatal(err)
	}
	beforeState := models.QuestionState{ResolutionOwner: "release-owner", Responders: []models.QuestionResponder{
		{Identity: "alice", Status: models.QuestionResponderPending},
		{Identity: "bob", Status: models.QuestionResponderPending},
	}}
	before, err := models.EncodeQuestionState(nil, beforeState)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `UPDATE questions SET context_data = ? WHERE id = ?`, before, question.ID); err != nil {
		t.Fatal(err)
	}
	afterState := beforeState
	afterState.Responders[0].Status = models.QuestionResponderCompleted
	afterState.Responses = []models.QuestionResponse{{SessionID: "session-a", Responder: "alice", Summary: "approved", EvidencePointer: "docs/spec.md", RecordedAt: time.Now().UTC()}}
	after, err := models.EncodeQuestionState(before, afterState)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.RecordResponse(ctx, question.ID, "answering", "answering", before, after, "alice"); err != nil {
		t.Fatalf("TC-105 first snapshot writer error = %v", err)
	}
	if err := repo.RecordResponse(ctx, question.ID, "answering", "answering", before, after, "alice"); err == nil {
		t.Fatal("TC-105 stale snapshot writer error = nil")
	}
	stored, err := repo.GetByKey(ctx, question.Key)
	if err != nil {
		t.Fatal(err)
	}
	if stored.ContextData == nil || *stored.ContextData != *after || stored.Status != "answering" {
		t.Fatalf("TC-105 stale conflict stored = %#v", stored)
	}
	var notes, history int
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM entity_notes WHERE entity_type = 'question' AND entity_id = ?`, question.ID).Scan(&notes); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM entity_history WHERE entity_type = 'question' AND entity_id = ?`, question.ID).Scan(&history); err != nil {
		t.Fatal(err)
	}
	if notes != 1 || history != 1 {
		t.Fatalf("TC-105 stale conflict audit = %d notes, %d history; want 1 each", notes, history)
	}
}

// TC-106: classified resolution has one transaction for state, typed note,
// history, and resolved status. A rejected history insert rolls all four back.
func TestQuestionRepositoryResolveRollsBackTerminalProvenanceOnHistoryFailure_TC106(t *testing.T) {
	ctx := context.Background()
	database := test.NewIsolatedTestDB(t)
	repo := NewQuestionRepository(dbconn.NewDB(database))
	question := testQuestion("Q001", "owner", false)
	question.Status = "ready_for_resolution"
	if err := repo.Create(ctx, question); err != nil {
		t.Fatal(err)
	}
	state := models.QuestionState{ResolutionOwner: "release-owner", Responders: []models.QuestionResponder{{Identity: "alice", Status: models.QuestionResponderCompleted}}, Responses: []models.QuestionResponse{{SessionID: "session-a", Responder: "alice", Summary: "approved", EvidencePointer: "docs/spec.md", RecordedAt: time.Now().UTC()}}}
	base, err := models.EncodeQuestionState(nil, state)
	if err != nil {
		t.Fatal(err)
	}
	terminal := strings.Replace(*base, `}}`, `,"resolution_kind":"feature_change","resolution_pointer":"docs/spec.md"}}`, 1)
	_, err = database.ExecContext(ctx, `CREATE TRIGGER reject_question_resolution_history BEFORE INSERT ON entity_history WHEN NEW.entity_type = 'question' AND NEW.entity_id = `+fmt.Sprint(question.ID)+` BEGIN SELECT RAISE(ABORT, 'reject Question resolution history'); END`)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Resolve(ctx, question.ID, "ready_for_resolution", "resolved", nil, &terminal, "release-owner", "feature_change"); err == nil {
		t.Fatal("TC-106 Resolve() error = nil")
	}
	stored, err := repo.GetByKey(ctx, question.Key)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != "ready_for_resolution" || stored.ContextData != nil {
		t.Fatalf("TC-106 rollback stored = %#v", stored)
	}
	var notes, history int
	_ = database.QueryRowContext(ctx, `SELECT COUNT(*) FROM entity_notes WHERE entity_type = 'question' AND entity_id = ?`, question.ID).Scan(&notes)
	_ = database.QueryRowContext(ctx, `SELECT COUNT(*) FROM entity_history WHERE entity_type = 'question' AND entity_id = ?`, question.ID).Scan(&history)
	if notes != 0 || history != 0 {
		t.Fatalf("TC-106 rollback audit = %d notes, %d history", notes, history)
	}
}

// TC-107: terminal withdrawal writes status, bounded provenance, note, and
// history together through the production Question repository.
func TestQuestionRepositoryWithdrawPersistsTerminalProvenance_TC107(t *testing.T) {
	ctx := context.Background()
	database := test.NewIsolatedTestDB(t)
	repo := NewQuestionRepository(dbconn.NewDB(database))
	question := testQuestion("Q001", "owner", false)
	question.Status = models.QuestionStatusOpen
	if err := repo.Create(ctx, question); err != nil {
		t.Fatal(err)
	}
	state := models.QuestionState{ResolutionOwner: "release-owner", Responders: []models.QuestionResponder{{Identity: "alice", Status: models.QuestionResponderPending}}}
	encoded, err := models.EncodeQuestionState(nil, state)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `UPDATE questions SET context_data = ? WHERE id = ?`, encoded, question.ID); err != nil {
		t.Fatal(err)
	}
	terminal := `{"question_state":` + *encoded + `,"question_terminal_provenance":{"status":"withdrawn","reason":"no longer needed"}}`
	if err := repo.Withdraw(ctx, question.ID, "open", "withdrawn", encoded, &terminal, "release-owner", "no longer needed"); err != nil {
		t.Fatalf("TC-107 Withdraw() error = %v", err)
	}
	stored, err := repo.GetByKey(ctx, question.Key)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != "withdrawn" || stored.ContextData == nil || !strings.Contains(*stored.ContextData, "question_terminal_provenance") {
		t.Fatalf("TC-107 stored = %#v", stored)
	}
	var notes, history int
	_ = database.QueryRowContext(ctx, `SELECT COUNT(*) FROM entity_notes WHERE entity_type = 'question' AND entity_id = ?`, question.ID).Scan(&notes)
	_ = database.QueryRowContext(ctx, `SELECT COUNT(*) FROM entity_history WHERE entity_type = 'question' AND entity_id = ?`, question.ID).Scan(&history)
	if notes != 1 || history != 1 {
		t.Fatalf("TC-107 audit = %d notes, %d history", notes, history)
	}
}
