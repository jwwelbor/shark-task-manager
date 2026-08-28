// Package contracts exercises the public seams shared by E39 consumers.
package contracts

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/api"
	viewerapi "github.com/jwwelbor/shark-task-manager/internal/api/viewer"
	"github.com/jwwelbor/shark-task-manager/internal/auth/maintainer"
	commands "github.com/jwwelbor/shark-task-manager/internal/cli/commands"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/keys"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	claimrepo "github.com/jwwelbor/shark-task-manager/internal/repository/claim"
	tagrepo "github.com/jwwelbor/shark-task-manager/internal/repository/tag"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	viewerserver "github.com/jwwelbor/shark-task-manager/internal/viewer/server"
)

// TC-001 proves the complete, narrow I-01 v1 seam using the production
// service, typed repository, generic registry adapter, and SQLite. It contains
// only identity, type, persisted base record, and typed ContextData operations.
func TestTC001_I01QuestionEntityAndPlatformRegistration(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := db.InitDB(filepath.Join(t.TempDir(), "questions.db"))
	if err != nil {
		t.Fatalf("InitDB() error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	questionRepo := repository.NewQuestionRepository(repository.NewDB(sqlDB))
	questionSvc, err := services.NewQuestionService(questionRepo)
	if err != nil {
		t.Fatalf("NewQuestionService() error = %v", err)
	}
	created, err := questionSvc.CreateQuestion(ctx, services.CreateQuestionInput{
		Title: "Release decision needed", Summary: "Choose the gate", Requester: "release-manager", Blocking: true,
	})
	if err != nil {
		t.Fatalf("CreateQuestion() error = %v", err)
	}
	if created.Key != "Q001" {
		t.Fatalf("canonical key = %q, want Q001", created.Key)
	}

	parsed := keys.NewKeyService().Parse("Q001")
	if parsed.EntityType != keys.EntityTypeQuestion || parsed.Normalized != "Q001" {
		t.Fatalf("Parse(Q001) = %#v, want canonical Question key", parsed)
	}

	registry := services.NewEntityRegistry()
	registry.Register(models.EntityTypeQuestion, services.NewQuestionRepositoryAdapter(questionRepo))
	adapter, err := registry.GetRepository(models.EntityTypeQuestion)
	if err != nil {
		t.Fatalf("GetRepository(question) error = %v", err)
	}
	persisted, err := adapter.GetByKey(ctx, "Q001")
	if err != nil {
		t.Fatalf("adapter.GetByKey(Q001) error = %v", err)
	}
	if persisted.GetEntityType() != models.EntityTypeQuestion || persisted.GetKey() != "Q001" || persisted.GetTitle() != "Release decision needed" || persisted.GetStatus() != "draft" {
		t.Fatalf("persisted I-01 base record = %#v", persisted)
	}

	contextSvc, err := services.NewContextService(registry)
	if err != nil {
		t.Fatalf("NewContextService() error = %v", err)
	}
	if err := contextSvc.SetContextField(ctx, models.EntityTypeQuestion, "Q001", "current_step", "awaiting decision"); err != nil {
		t.Fatalf("SetContextField() error = %v", err)
	}
	contextData, err := contextSvc.GetContext(ctx, models.EntityTypeQuestion, "Q001")
	if err != nil || contextData == nil || contextData.Progress == nil || contextData.Progress.CurrentStep == nil || *contextData.Progress.CurrentStep != "awaiting decision" {
		t.Fatalf("GetContext() = %#v, %v, want typed ContextData", contextData, err)
	}
	if err := contextSvc.ClearContext(ctx, models.EntityTypeQuestion, "Q001"); err != nil {
		t.Fatalf("ClearContext() error = %v", err)
	}
	if cleared, err := contextSvc.GetContext(ctx, models.EntityTypeQuestion, "Q001"); err != nil || cleared != nil {
		t.Fatalf("GetContext() after ClearContext = %#v, %v, want nil", cleared, err)
	}

	for _, field := range []string{"question_state", "metadata"} {
		if err := contextSvc.SetContextField(ctx, models.EntityTypeQuestion, "Q001", field, "forbidden"); err == nil {
			t.Errorf("SetContextField(%q) error = nil, want rejection", field)
		}
	}
	if unchanged, err := contextSvc.GetContext(ctx, models.EntityTypeQuestion, "Q001"); err != nil || unchanged != nil {
		t.Fatalf("invalid context write mutated Question: %#v, %v", unchanged, err)
	}
}

// TC-007 executes the complete finite REQ-F-003 association and lease matrix
// against a real Question row and SQLite.  Every rejected partition is paired
// with a storage count/snapshot so a future generic-surface registration cannot
// appear to work while partially mutating durable state.
// tc007Fixture is the shared SQLite-backed fixture for TC-007: one real
// Question row pair (q1, q2) plus the production services each subtest
// exercises. Subtests compute their own "before" snapshots at the point
// they run, so they don't depend on execution order among themselves.
type tc007Fixture struct {
	ctx          context.Context
	dbPath       string
	sqlDB        *sql.DB
	repoDB       *repository.DB
	questionRepo *repository.QuestionRepository
	historyRepo  *repository.EntityHistoryRepository
	registry     *services.EntityRegistry
	adapter      services.EntityRepository
	contextSvc   *services.ContextService
	q1, q2       *models.Question
}

func setupTC007Fixture(t *testing.T) *tc007Fixture {
	t.Helper()
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "question-surfaces.db")
	sqlDB, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB() error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	repoDB := repository.NewDB(sqlDB)
	questionRepo := repository.NewQuestionRepository(repoDB)
	historyRepo := repository.NewEntityHistoryRepository(repoDB)
	questionSvc, err := services.NewQuestionService(questionRepo)
	if err != nil {
		t.Fatalf("NewQuestionService() error = %v", err)
	}
	questionSvc.SetHistoryRepo(historyRepo)
	q1, err := questionSvc.CreateQuestion(ctx, services.CreateQuestionInput{Title: "Surface contract", Summary: "Verify generic services", Requester: "contract"})
	if err != nil {
		t.Fatalf("CreateQuestion(Q001) error = %v", err)
	}
	q2, err := questionSvc.CreateQuestion(ctx, services.CreateQuestionInput{Title: "Relationship target", Summary: "Existing endpoint", Requester: "contract"})
	if err != nil {
		t.Fatalf("CreateQuestion(Q002) error = %v", err)
	}

	registry := services.NewEntityRegistry()
	adapter := services.NewQuestionRepositoryAdapter(questionRepo)
	registry.Register(models.EntityTypeQuestion, adapter)
	if _, err := registry.GetRepository(models.EntityTypeQuestion); err != nil {
		t.Fatalf("registry.GetRepository(question) error = %v", err)
	}
	contextSvc, err := services.NewContextService(registry)
	if err != nil {
		t.Fatalf("NewContextService() error = %v", err)
	}
	return &tc007Fixture{
		ctx: ctx, dbPath: dbPath, sqlDB: sqlDB, repoDB: repoDB,
		questionRepo: questionRepo, historyRepo: historyRepo,
		registry: registry, adapter: adapter, contextSvc: contextSvc,
		q1: q1, q2: q2,
	}
}

// TC-007 executes the complete finite REQ-F-003 association and lease matrix
// against a real Question row and SQLite. Every rejected partition is paired
// with a storage count/snapshot so a future generic-surface registration cannot
// appear to work while partially mutating durable state. Each concern is its
// own subtest sharing the fixture above so a failure isolates to one surface.
func TestTC007_QuestionGenericAssociationsAndLeaseIsolation(t *testing.T) {
	f := setupTC007Fixture(t)

	t.Run("registry_lookup_of_unknown_type_is_read_only", func(t *testing.T) { tc007RegistryLookup(t, f) })
	t.Run("context_lifecycle_rejects_reserved_fields", func(t *testing.T) { tc007ContextLifecycle(t, f) })
	t.Run("notes_reject_empty_content_without_partial_write", func(t *testing.T) { tc007Notes(t, f) })
	t.Run("history_is_read_only_and_unknown_key_leaves_no_row", func(t *testing.T) { tc007History(t, f) })
	t.Run("documents_reject_missing_question_without_partial_link", func(t *testing.T) { tc007Documents(t, f) })
	t.Run("tags_reject_unregistered_names_without_partial_attach", func(t *testing.T) { tc007Tags(t, f) })
	t.Run("relationships_reject_missing_endpoints_before_insert", func(t *testing.T) { tc007Relationships(t, f) })
	t.Run("claim_lease_lifecycle_never_touches_question_state", func(t *testing.T) { tc007ClaimLeaseLifecycle(t, f) })
	t.Run("claim_expiry_reclaims_without_touching_question_state", func(t *testing.T) { tc007ClaimExpiry(t, f) })
}

func tc007RegistryLookup(t *testing.T, f *tc007Fixture) {
	questionsBefore := countTC007(t, f.sqlDB, "questions")
	if _, err := f.registry.GetRepository(models.EntityType("unknown")); err == nil {
		t.Fatal("registry.GetRepository(unknown) error = nil")
	}
	if got := countTC007(t, f.sqlDB, "questions"); got != questionsBefore {
		t.Fatalf("unknown registry lookup changed Question rows: got %d, want %d", got, questionsBefore)
	}
}

// tc007ContextLifecycle proves both the successful set/get/clear round trip
// and the rejected future fields leave the Question's durable context
// exactly where the caller expects, using only the existing typed lifecycle.
func tc007ContextLifecycle(t *testing.T, f *tc007Fixture) {
	ctx, contextSvc := f.ctx, f.contextSvc
	if err := contextSvc.SetContextField(ctx, models.EntityTypeQuestion, f.q1.Key, "current_step", "surface lifecycle"); err != nil {
		t.Fatalf("SetContextField(question) error = %v", err)
	}
	contextBeforeInvalid := questionStateTC007(t, f.sqlDB, f.q1.Key)
	if got, err := contextSvc.GetContext(ctx, models.EntityTypeQuestion, f.q1.Key); err != nil || got == nil || got.Progress == nil || got.Progress.CurrentStep == nil || *got.Progress.CurrentStep != "surface lifecycle" {
		t.Fatalf("GetContext(question) = %#v, %v, want typed current_step", got, err)
	}
	for _, field := range []string{"question_state", "metadata"} {
		if err := contextSvc.SetContextField(ctx, models.EntityTypeQuestion, f.q1.Key, field, "forbidden"); err == nil {
			t.Errorf("SetContextField(%q) error = nil, want rejection", field)
		}
		if got := questionStateTC007(t, f.sqlDB, f.q1.Key); got != contextBeforeInvalid {
			t.Fatalf("invalid context field %q mutated Question state: got %q, want %q", field, got, contextBeforeInvalid)
		}
	}
	if err := contextSvc.ClearContext(ctx, models.EntityTypeQuestion, f.q1.Key); err != nil {
		t.Fatalf("ClearContext(question) error = %v", err)
	}
	if got, err := contextSvc.GetContext(ctx, models.EntityTypeQuestion, f.q1.Key); err != nil || got != nil {
		t.Fatalf("GetContext(question) after ClearContext = %#v, %v, want nil", got, err)
	}
}

// tc007Notes proves a valid typed note persists exactly one association and
// empty content is rejected by the real note repository without another row.
func tc007Notes(t *testing.T, f *tc007Fixture) {
	ctx := f.ctx
	noteSvc, err := services.NewNoteService(repository.NewEntityNoteRepository(f.repoDB), f.registry)
	if err != nil {
		t.Fatalf("NewNoteService() error = %v", err)
	}
	if _, err := noteSvc.AddNote(ctx, models.EntityTypeQuestion, f.q1.Key, "question", "durable note", "contract"); err != nil {
		t.Fatalf("AddNote(question) error = %v", err)
	}
	if notes, err := noteSvc.ListNotes(ctx, models.EntityTypeQuestion, f.q1.Key, nil); err != nil || len(notes) != 1 {
		t.Fatalf("ListNotes(question) = %#v, %v, want one note", notes, err)
	}
	notesBefore := countTC007Where(t, f.sqlDB, "entity_notes", "entity_type = 'question' AND entity_id = ?", f.q1.ID)
	if _, err := noteSvc.AddNote(ctx, models.EntityTypeQuestion, f.q1.Key, "question", " ", "contract"); err == nil {
		t.Fatal("AddNote(empty content) error = nil")
	}
	if got := countTC007Where(t, f.sqlDB, "entity_notes", "entity_type = 'question' AND entity_id = ?", f.q1.ID); got != notesBefore {
		t.Fatalf("invalid note changed rows: got %d, want %d", got, notesBefore)
	}
}

// tc007History proves history is read-only: creation yields an auditable row
// and an unknown key cannot create one while resolving the generic adapter.
func tc007History(t *testing.T, f *tc007Fixture) {
	ctx := f.ctx
	historySvc := services.NewEntityHistoryService(f.historyRepo, f.registry)
	if history, err := historySvc.GetHistory(ctx, models.EntityTypeQuestion, f.q1.Key); err != nil || len(history) == 0 {
		t.Fatalf("GetHistory(question) = %#v, %v, want creation row", history, err)
	}
	historyBefore := countTC007Where(t, f.sqlDB, "entity_history", "entity_type = 'question' AND entity_id = ?", f.q1.ID)
	if _, err := historySvc.GetHistory(ctx, models.EntityTypeQuestion, "Q404"); err == nil {
		t.Fatal("GetHistory(Q404) error = nil")
	}
	if got := countTC007Where(t, f.sqlDB, "entity_history", "entity_type = 'question' AND entity_id = ?", f.q1.ID); got != historyBefore {
		t.Fatalf("unknown history lookup changed rows: got %d, want %d", got, historyBefore)
	}
}

// tc007Documents proves the production entity lookup supplies Question's
// type/ID to the generic link service, and a missing Question must not
// create either link.
func tc007Documents(t *testing.T, f *tc007Fixture) {
	ctx := f.ctx
	docSvc := services.NewEntityDocumentService(repository.NewDocumentRepository(f.repoDB), repository.NewEntityDocumentRepository(f.repoDB), services.EntityLookupFnFromRepo(f.adapter), ".")
	if _, err := docSvc.LinkDocumentByKey(ctx, f.q1.Key, "Question spec", "docs/question-spec.md"); err != nil {
		t.Fatalf("LinkDocumentByKey(question) error = %v", err)
	}
	if docs, err := docSvc.ListDocumentsByKey(ctx, f.q1.Key); err != nil || len(docs) != 1 {
		t.Fatalf("ListDocumentsByKey(question) = %#v, %v, want one document", docs, err)
	}
	docLinksBefore := countTC007Where(t, f.sqlDB, "entity_documents", "entity_type = 'question' AND entity_id = ?", f.q1.ID)
	if _, err := docSvc.LinkDocumentByKey(ctx, "Q404", "missing Question", "docs/missing.md"); err == nil {
		t.Fatal("LinkDocumentByKey(Q404) error = nil")
	}
	if got := countTC007Where(t, f.sqlDB, "entity_documents", "entity_type = 'question' AND entity_id = ?", f.q1.ID); got != docLinksBefore {
		t.Fatalf("missing Question document link changed rows: got %d, want %d", got, docLinksBefore)
	}
	if err := docSvc.UnlinkDocumentByKey(ctx, f.q1.Key, "Question spec"); err != nil {
		t.Fatalf("UnlinkDocumentByKey(question) error = %v", err)
	}
	if docs, err := docSvc.ListDocumentsByKey(ctx, f.q1.Key); err != nil || len(docs) != 0 {
		t.Fatalf("ListDocumentsByKey(question) after unlink = %#v, %v, want no links", docs, err)
	}
	if got := countTC007Where(t, f.sqlDB, "entity_documents", "entity_type = 'question' AND entity_id = ?", f.q1.ID); got != 0 {
		t.Fatalf("UnlinkDocumentByKey(question) left %d durable links, want 0", got)
	}
	if err := docSvc.UnlinkDocumentByKey(ctx, "Q404", "Question spec"); err == nil {
		t.Fatal("UnlinkDocumentByKey(Q404) error = nil")
	}
	if got := countTC007Where(t, f.sqlDB, "entity_documents", "entity_type = 'question' AND entity_id = ?", f.q1.ID); got != 0 {
		t.Fatalf("missing Question document unlink changed rows: got %d, want 0", got)
	}
}

// tc007Tags exercises the production tag vocabulary and polymorphic
// association store.
func tc007Tags(t *testing.T, f *tc007Fixture) {
	ctx := f.ctx
	tags := tagrepo.NewTagRepository(f.repoDB)
	if _, err := tags.Create(ctx, "platform"); err != nil {
		t.Fatalf("create tag vocabulary: %v", err)
	}
	tagSvc := services.NewTagService(tagrepo.NewTagRepository(f.repoDB), tagrepo.NewEntityTagRepository(f.repoDB), maintainer.NewFileGate(t.TempDir(), nil, time.Minute), services.EmptyTagEnforcementConfig{})
	if err := tagSvc.AttachMany(ctx, models.EntityTypeQuestion, f.q1.ID, []string{"platform"}); err != nil {
		t.Fatalf("AttachMany(question) error = %v", err)
	}
	if names, err := tagSvc.ListTagsForEntity(ctx, models.EntityTypeQuestion, f.q1.ID); err != nil || len(names) != 1 || names[0] != "platform" {
		t.Fatalf("ListTagsForEntity(question) = %#v, %v", names, err)
	}
	tagsBefore := countTC007Where(t, f.sqlDB, "entity_tags", "entity_type = 'question' AND entity_id = ?", f.q1.ID)
	if err := tagSvc.AttachMany(ctx, models.EntityTypeQuestion, f.q1.ID, []string{"unregistered"}); err == nil {
		t.Fatal("AttachMany(unregistered) error = nil")
	}
	if got := countTC007Where(t, f.sqlDB, "entity_tags", "entity_type = 'question' AND entity_id = ?", f.q1.ID); got != tagsBefore {
		t.Fatalf("unregistered tag changed rows: got %d, want %d", got, tagsBefore)
	}
}

// tc007Relationships proves the generic relationship command resolves both
// endpoint keys before it reaches typed storage (a missing Question is
// rejected with no association), and that the repository's own structural
// invalid-endpoint partition is rejected before INSERT.
func tc007Relationships(t *testing.T, f *tc007Fixture) {
	ctx := f.ctx
	relsBeforeMissing := countTC007Where(t, f.sqlDB, "entity_relationships", "from_entity_type = 'question' AND from_entity_id = ?", f.q1.ID)
	_ = runSharkTC011Failure(t, f.dbPath, "link", f.q1.Key, "Q404", "--type=linked_to")
	if got := countTC007Where(t, f.sqlDB, "entity_relationships", "from_entity_type = 'question' AND from_entity_id = ?", f.q1.ID); got != relsBeforeMissing {
		t.Fatalf("missing relationship endpoint changed rows: got %d, want %d", got, relsBeforeMissing)
	}

	relRepo := repository.NewEntityRelationshipRepository(f.repoDB)
	if err := relRepo.Create(ctx, &models.EntityRelationship{FromEntityType: models.EntityTypeQuestion, FromEntityID: f.q1.ID, ToEntityType: models.EntityTypeQuestion, ToEntityID: f.q2.ID, RelationshipType: models.EntityRelLinkedTo}); err != nil {
		t.Fatalf("Create(question relationship) error = %v", err)
	}
	if rels, err := relRepo.GetOutgoing(ctx, models.EntityTypeQuestion, f.q1.ID, []models.EntityRelationshipType{models.EntityRelLinkedTo}); err != nil || len(rels) != 1 || rels[0].ToEntityID != f.q2.ID {
		t.Fatalf("GetOutgoing(question relationship) = %#v, %v, want Q002 linked_to", rels, err)
	}
	relsBefore := countTC007Where(t, f.sqlDB, "entity_relationships", "from_entity_type = 'question' AND from_entity_id = ?", f.q1.ID)
	if err := relRepo.Create(ctx, &models.EntityRelationship{FromEntityType: models.EntityTypeQuestion, FromEntityID: f.q1.ID, ToEntityType: models.EntityTypeQuestion, ToEntityID: 0, RelationshipType: models.EntityRelLinkedTo}); err == nil {
		t.Fatal("Create(relationship missing endpoint) error = nil")
	}
	if got := countTC007Where(t, f.sqlDB, "entity_relationships", "from_entity_type = 'question' AND from_entity_id = ?", f.q1.ID); got != relsBefore {
		t.Fatalf("invalid relationship changed rows: got %d, want %d", got, relsBefore)
	}
}

// tc007ClaimLeaseLifecycle proves claims and their heartbeat/release
// lifecycle are intentionally separate from Question state: non-empty
// typed context is set as a snapshot sentinel first, so this proves lease
// operations neither transition status nor erase/replace context data.
func tc007ClaimLeaseLifecycle(t *testing.T, f *tc007Fixture) {
	ctx := f.ctx
	if err := f.contextSvc.SetContextField(ctx, models.EntityTypeQuestion, f.q1.Key, "current_step", "lease isolation sentinel"); err != nil {
		t.Fatalf("SetContextField(lease sentinel) error = %v", err)
	}
	beforeLease := questionStateTC007(t, f.sqlDB, f.q1.Key)
	claimSvc := services.NewClaimService(claimrepo.NewRepository(f.repoDB), ptrTC007(time.Hour))
	claimed, err := claimSvc.Claim(ctx, services.ClaimInput{EntityType: "question", EntityKey: f.q1.Key, ClaimedBy: "contract", SessionID: "session-1"})
	if err != nil {
		t.Fatalf("Claim(question) error = %v", err)
	}
	if err := claimSvc.Heartbeat(ctx, "question", f.q1.Key, "wrong-session", nil, ""); err == nil {
		t.Fatal("Heartbeat(wrong session) error = nil")
	}
	if released, err := claimSvc.Release(ctx, "question", f.q1.Key, "wrong-session", "", false); err != nil || released {
		t.Fatalf("Release(wrong session) = %v, %v, want false, nil", released, err)
	}
	if err := claimSvc.Heartbeat(ctx, "question", f.q1.Key, claimed.SessionID, nil, ""); err != nil {
		t.Fatalf("Heartbeat(question) error = %v", err)
	}
	if released, err := claimSvc.Release(ctx, "question", f.q1.Key, claimed.SessionID, "released", false); err != nil || !released {
		t.Fatalf("Release(question) = %v, %v, want true, nil", released, err)
	}
	if afterLease := questionStateTC007(t, f.sqlDB, f.q1.Key); afterLease != beforeLease {
		t.Fatalf("claim lifecycle changed Question status/context: before=%q after=%q", beforeLease, afterLease)
	}
}

// tc007ClaimExpiry proves expiry is a real delete from the generic lease
// store, not a read-only IsExpired calculation: it makes the persisted
// heartbeat stale, reclaims it, and proves the Question base state is still
// byte-identical to the snapshot taken before the claim.
func tc007ClaimExpiry(t *testing.T, f *tc007Fixture) {
	ctx := f.ctx
	beforeLease := questionStateTC007(t, f.sqlDB, f.q1.Key)
	claimSvc := services.NewClaimService(claimrepo.NewRepository(f.repoDB), ptrTC007(time.Hour))
	expired, err := claimSvc.Claim(ctx, services.ClaimInput{EntityType: "question", EntityKey: f.q1.Key, ClaimedBy: "contract", SessionID: "session-expired"})
	if err != nil {
		t.Fatalf("Claim(question for expiry) error = %v", err)
	}
	if _, err := f.sqlDB.ExecContext(ctx, "UPDATE entity_claims SET last_heartbeat = ? WHERE id = ?", time.Now().UTC().Add(-2*time.Hour), expired.ID); err != nil {
		t.Fatalf("make Question claim expired: %v", err)
	}
	if reclaimed, err := claimSvc.ReclaimExpired(ctx); err != nil || reclaimed != 1 {
		t.Fatalf("ReclaimExpired(question) = %d, %v, want 1, nil", reclaimed, err)
	}
	if claim, err := claimSvc.Get(ctx, "question", f.q1.Key); err != nil || claim != nil {
		t.Fatalf("Get(question) after expiry = %#v, %v, want nil", claim, err)
	}
	if afterExpiry := questionStateTC007(t, f.sqlDB, f.q1.Key); afterExpiry != beforeLease {
		t.Fatalf("claim expiry changed Question status/context: before=%q after=%q", beforeLease, afterExpiry)
	}
}

// TC-008 exercises the actual Cobra registrations and HTTP route patterns for
// Question. Unit tests intentionally mock the service seam; this contract does
// not. It would fail if a Question-specific command worked while a generic
// dispatcher failed to classify Q001, or if a handler method existed but was
// omitted from the registered mux.
func TestTC008_QuestionRegistrationTransportMatrix(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "question-transport.db")
	shark := buildSharkTC008(t)
	run := func(args ...string) string {
		t.Helper()
		command := exec.Command(shark, append([]string{"--db", dbPath, "--json"}, args...)...)
		command.Dir = projectRootTC011(t)
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("shark %s failed: %v\n%s", strings.Join(args, " "), err, output)
		}
		return string(output)
	}

	// Direct Question commands and their generic counterparts intentionally
	// alternate. Each row proves registration plus the real caller path.
	if output := run("question", "create", "Specific transport", "--summary", "specific summary", "--requester", "specific-owner", "--blocking"); !strings.Contains(output, `"key": "Q001"`) {
		t.Fatalf("specific create output = %s", output)
	}
	if output := run("create", "question", "Generic transport", "--summary", "generic summary", "--requester", "generic-owner"); !strings.Contains(output, `"key": "Q002"`) {
		t.Fatalf("generic create output = %s", output)
	}
	for _, invocation := range [][]string{
		{"question", "get", "Q001"}, {"get", "Q001"}, {"question", "status", "Q001"},
		{"question", "list", "--status", "draft", "--requester", "specific-owner", "--blocking=true", "--limit=1", "--offset=0"},
		{"list", "question", "--status", "draft", "--requester", "specific-owner", "--blocking=true", "--limit=1", "--offset=0"},
	} {
		if output := run(invocation...); !strings.Contains(output, `"key": "Q001"`) {
			t.Fatalf("registered transport %q output = %s", invocation, output)
		}
	}
	if output := run("question", "update", "Q001", "--title", "Specific transport updated"); !strings.Contains(output, "Specific transport updated") {
		t.Fatalf("specific update output = %s", output)
	}
	if output := run("update", "Q002", "--summary", "Generic transport updated", "--blocking"); !strings.Contains(output, "Generic transport updated") {
		t.Fatalf("generic update output = %s", output)
	}
	if output := run("link", "Q001", "Q002", "--type=linked_to"); !strings.Contains(output, `"from_entity_type": "question"`) {
		t.Fatalf("generic link output = %s", output)
	}
	if output := run("create", "note", "Q001", "Question transport note", "--type=comment"); !strings.Contains(output, `"entity_type": "question"`) {
		t.Fatalf("generic note output = %s", output)
	}
	if output := run("related-docs", "add", "Question transport document", "docs/plan/E39-question-and-decision-workflow-management/E39-F01-question-entity-and-platform-registration/spec.md", "--question", "Q001"); !strings.Contains(output, `"linked_to": "question"`) {
		t.Fatalf("generic related-docs add output = %s", output)
	}
	if output := run("related-docs", "list", "--question", "Q001"); !strings.Contains(output, "Question transport document") {
		t.Fatalf("generic related-docs list output = %s", output)
	}
	if output := run("related-docs", "delete", "Question transport document", "--question", "Q001"); !strings.Contains(output, `"parent": "question"`) {
		t.Fatalf("generic related-docs delete output = %s", output)
	}
	if output := run("context", "set", "Q001", "--field", "current_step", "--value", "transport-check"); !strings.Contains(output, `"entity_type": "question"`) {
		t.Fatalf("generic context set output = %s", output)
	}
	if output := run("context", "get", "Q001"); !strings.Contains(output, "transport-check") {
		t.Fatalf("generic context get output = %s", output)
	}
	if output := run("history", "Q001"); !strings.Contains(output, "draft") {
		t.Fatalf("generic history output = %s", output)
	}

	// The HTTP half uses real service/repository dependencies and every public
	// method-qualified route. This catches an omitted route registration rather
	// than merely testing handler methods directly.
	sqlDB, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB() error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	questionSvc, err := services.NewQuestionService(repository.NewQuestionRepository(repository.NewDB(sqlDB)))
	if err != nil {
		t.Fatalf("NewQuestionService() error = %v", err)
	}
	mux := http.NewServeMux()
	api.NewQuestionHandler(questionSvc).RegisterRoutes(mux)
	for _, tc := range []struct {
		method, path, body string
		wantCode           int
	}{
		{http.MethodPost, "/api/v1/questions", `{"title":"HTTP transport","summary":"http summary","requester":"http-owner"}`, http.StatusCreated},
		{http.MethodGet, "/api/v1/questions/Q003", "", http.StatusOK},
		{http.MethodGet, "/api/v1/questions?requester=http-owner&limit=1", "", http.StatusOK},
		{http.MethodPatch, "/api/v1/questions/Q003", `{"summary":"http updated"}`, http.StatusOK},
		{http.MethodDelete, "/api/v1/questions/Q003", "", http.StatusNoContent},
	} {
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body)))
		if recorder.Code != tc.wantCode {
			t.Fatalf("%s %s status=%d body=%s, want %d", tc.method, tc.path, recorder.Code, recorder.Body.String(), tc.wantCode)
		}
	}

	if output := run("delete", "Q002"); !strings.Contains(output, `"deleted": "Q002"`) {
		t.Fatalf("generic delete output = %s", output)
	}
	if output := run("question", "delete", "Q001"); !strings.Contains(output, `"deleted": "Q001"`) {
		t.Fatalf("specific delete output = %s", output)
	}
}

func buildSharkTC008(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "shark")
	command := exec.Command("go", "build", "-buildvcs=false", "-o", binary, "./cmd/shark")
	command.Dir = projectRootTC011(t)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build shark test binary: %v\n%s", err, output)
	}
	return binary
}

func countTC007(t *testing.T, sqlDB *sql.DB, table string) int {
	t.Helper()
	return countTC007Where(t, sqlDB, table, "1 = 1")
}

func countTC007Where(t *testing.T, sqlDB *sql.DB, table, predicate string, args ...any) int {
	t.Helper()
	var count int
	if err := sqlDB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", table, predicate), args...).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return count
}

func questionStateTC007(t *testing.T, sqlDB *sql.DB, key string) string {
	t.Helper()
	var status, contextData string
	if err := sqlDB.QueryRow("SELECT status, COALESCE(context_data, '') FROM questions WHERE key = ?", key).Scan(&status, &contextData); err != nil {
		t.Fatalf("snapshot Question %s: %v", key, err)
	}
	return status + "\x00" + contextData
}

func ptrTC007(value time.Duration) *time.Duration { return &value }

// TC-013 exercises the F01 runtime-evidence procedure through the production
// CLI, HTTP handler, service, registry, adapter, and SQLite database. It
// deliberately captures only metadata-bearing output: the sentinel stored in
// ContextData must never appear in any returned transport payload.
func TestTC013_QuestionRuntimeEvidenceAndI01Closure(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "question-runtime.db")

	createdOutput := runSharkTC013(t, dbPath, "--json", "question", "create", "Runtime question", "--summary", "Runtime summary", "--requester", "runtime-owner", "--blocking")
	if !strings.Contains(createdOutput, `"key": "Q001"`) {
		t.Fatalf("question create output = %s, want Q001", createdOutput)
	}
	runSharkTC013(t, dbPath, "--json", "question", "create", "Linked runtime question", "--summary", "Linked summary", "--requester", "runtime-owner")

	sqlDB, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB() error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	questionRepo := repository.NewQuestionRepository(repository.NewDB(sqlDB))
	questionSvc, err := services.NewQuestionService(questionRepo)
	if err != nil {
		t.Fatalf("NewQuestionService() error = %v", err)
	}
	registry := services.NewEntityRegistry()
	registry.Register(models.EntityTypeQuestion, services.NewQuestionRepositoryAdapter(questionRepo))
	contextSvc, err := services.NewContextService(registry)
	if err != nil {
		t.Fatalf("NewContextService() error = %v", err)
	}
	const sentinel = "TC013-context-sentinel-must-not-project"
	if err := contextSvc.SetContextField(ctx, models.EntityTypeQuestion, "Q001", "current_step", sentinel); err != nil {
		t.Fatalf("SetContextField() error = %v", err)
	}

	// Search must find the persisted Question through the real CLI/index path,
	// but ContextData is never indexed or returned. This also proves the F01
	// Question service wired its incremental indexer rather than relying on a
	// test-only rebuild.
	searchOutput := runSharkTC013(t, dbPath, "--json", "search", "Runtime", "--type=question")
	if !strings.Contains(searchOutput, `"entity_type": "question"`) || !strings.Contains(searchOutput, `"key": "Q001"`) || strings.Contains(searchOutput, sentinel) {
		t.Fatalf("Question search output did not preserve metadata-only discovery: %s", searchOutput)
	}

	// Exercise the production viewer wiring and its existing generic history
	// endpoint. Q001 must resolve as a Question, and the response must not
	// project the ContextData sentinel.
	viewerServices := viewerserver.WireServices(repository.NewDB(sqlDB), projectRootTC011(t))
	viewerMux := http.NewServeMux()
	viewerapi.NewViewerHandler(viewerServices.ViewerService).RegisterRoutes(viewerMux, "/api/v1/viewer")
	viewerReq := httptest.NewRequest(http.MethodGet, "/api/v1/viewer/history/Q001", nil)
	viewerResponse := httptest.NewRecorder()
	viewerMux.ServeHTTP(viewerResponse, viewerReq)
	if viewerResponse.Code != http.StatusOK || !strings.Contains(viewerResponse.Body.String(), `"entity_type":"question"`) || strings.Contains(viewerResponse.Body.String(), sentinel) {
		t.Fatalf("Question viewer output did not preserve metadata-only projection: status=%d body=%s", viewerResponse.Code, viewerResponse.Body.String())
	}

	mux := http.NewServeMux()
	api.NewQuestionHandler(questionSvc).RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/questions/Q001", nil)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/questions/Q001 status = %d, body = %s", response.Code, response.Body.String())
	}
	var apiQuestion models.Question
	if err := json.Unmarshal(response.Body.Bytes(), &apiQuestion); err != nil {
		t.Fatalf("decode Question API response: %v", err)
	}
	if apiQuestion.Key != "Q001" || strings.Contains(response.Body.String(), sentinel) {
		t.Fatalf("Question API response did not preserve metadata-only projection: %s", response.Body.String())
	}

	// This is intentionally a real generic CLI call. A direct relationship
	// repository call would bypass the closed type mapper that F01 must register.
	runSharkTC013(t, dbPath, "--json", "link", "Q001", "Q002", "--type=linked_to")

	nextOutput := runSharkTC013(t, dbPath, "next", "Q001")
	if !strings.Contains(nextOutput, `"action": "pause"`) || strings.Contains(nextOutput, sentinel) {
		t.Fatalf("shark next Q001 output = %s, want redacted pause envelope", nextOutput)
	}

	var questions, relationships, history int
	if err := sqlDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM questions WHERE key = 'Q001'").Scan(&questions); err != nil {
		t.Fatalf("count Question before delete: %v", err)
	}
	if err := sqlDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM entity_relationships WHERE from_entity_type = 'question' AND from_entity_id = 1").Scan(&relationships); err != nil {
		t.Fatalf("count Question relationships before delete: %v", err)
	}
	if err := sqlDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM entity_history WHERE entity_type = 'question' AND entity_id = 1").Scan(&history); err != nil {
		t.Fatalf("count Question history before delete: %v", err)
	}
	if questions != 1 || relationships != 1 || history == 0 {
		t.Fatalf("pre-delete evidence counts = questions:%d relationships:%d history:%d, want 1/1/non-zero", questions, relationships, history)
	}

	deletedOutput := runSharkTC013(t, dbPath, "--json", "question", "delete", "Q001")
	if !strings.Contains(deletedOutput, `"deleted": "Q001"`) {
		t.Fatalf("question delete output = %s", deletedOutput)
	}
	for table, predicate := range map[string]string{
		"questions":            "key = 'Q001'",
		"entity_relationships": "from_entity_type = 'question' AND from_entity_id = 1",
		"entity_history":       "entity_type = 'question' AND entity_id = 1",
	} {
		var count int
		if err := sqlDB.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", table, predicate)).Scan(&count); err != nil {
			t.Fatalf("count %s after delete: %v", table, err)
		}
		if count != 0 {
			t.Errorf("%s rows after Question delete = %d, want 0", table, count)
		}
	}
}

// TC-002 drives the I-02 producer through the real keyed-next CLI boundary.
// It proves that the workflow bundle, Question service, existing lease, and
// complete NextResponse envelope select exactly the first pending responder.
func TestTC002_I02SerialQuestionKeyedDispatch(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "i02-keyed-dispatch.db")
	runSharkTC013(t, dbPath, "--json", "question", "create", "I-02 dispatch question", "--summary", "Question summary", "--requester", "release-owner")
	runSharkTC013(t, dbPath, "--json", "question", "configure-workflow", "Q001", "--resolution-owner", "release-owner", "--responder", "alice", "--responder", "bob", "--responder", "carol")

	assertFirstPending := func(wantResponder, wantStatus string) {
		t.Helper()
		output := runSharkTC013(t, dbPath, "--json", "next", "Q001")
		var response commands.NextResponse
		if err := json.Unmarshal([]byte(output), &response); err != nil {
			t.Fatalf("TC-002 decode keyed next = %v\n%s", err, output)
		}
		if response.EntityKey != "Q001" || response.EntityType != "question" || response.Status != wantStatus || response.Action != "spawn_agent" {
			t.Fatalf("TC-002 keyed next envelope = %#v, want Q001/question/%s/spawn_agent", response, wantStatus)
		}
		if !strings.Contains(response.Prompt, "currently routed responder: "+wantResponder) {
			t.Fatalf("TC-002 prompt = %q, want first pending responder %q", response.Prompt, wantResponder)
		}
		for _, other := range []string{"alice", "bob", "carol"} {
			if other != wantResponder && strings.Contains(response.Prompt, other) {
				t.Fatalf("TC-002 prompt exposes non-current responder %q: %q", other, response.Prompt)
			}
		}
	}

	assertFirstPending("alice", "open")

	// A lease makes the keyed path pause rather than dispatch a competing
	// responder. Recording the bounded response alone still cannot expose bob;
	// only the parent-loop release makes the next serial dispatch available.
	runSharkTC013(t, dbPath, "--json", "claim", "Q001", "--by", "alice", "--session", "alice-session")
	claimedOutput := runSharkTC013(t, dbPath, "--json", "next", "Q001")
	var claimed commands.NextResponse
	if err := json.Unmarshal([]byte(claimedOutput), &claimed); err != nil || claimed.Action != "pause" {
		t.Fatalf("TC-002 keyed next while claimed = %s, decode error = %v, want pause", claimedOutput, err)
	}
	runSharkTC013(t, dbPath, "--json", "question", "respond", "Q001", "--session", "alice-session", "--responder", "alice", "--summary", "approved", "--evidence-pointer", "docs/plan/E39-question-and-decision-workflow-management/E39-F02-serial-question-workflow-and-resolution-provenance/spec.md")
	stillClaimedOutput := runSharkTC013(t, dbPath, "--json", "next", "Q001")
	var stillClaimed commands.NextResponse
	if err := json.Unmarshal([]byte(stillClaimedOutput), &stillClaimed); err != nil || stillClaimed.Action != "pause" || strings.Contains(stillClaimedOutput, "approved") {
		t.Fatalf("TC-002 keyed next after response before release = %s, decode error = %v, want redacted pause", stillClaimedOutput, err)
	}
	runSharkTC013(t, dbPath, "--json", "release", "Q001", "--session", "alice-session", "--outcome", "pass")
	assertFirstPending("bob", "answering")
}

// TC-003 proves that the three public I-03 producer boundaries expose the
// same compact handoff over the real SQLite-backed I-01/I-02 composition. It
// deliberately executes the Cobra binary instead of substituting the blocker
// or command dispatch seams: drift at one boundary is the contract risk.
func TestTC003_I03QuestionBlockingGatePublicContract(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "i03-public-contract.db")
	sqlDB, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("TC-003 InitDB() error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	repoDB := repository.NewDB(sqlDB)
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E01", Title: "I-03 contract epic"}, Status: models.EpicStatusActive, Priority: models.PriorityMedium}
	if err := repository.NewEpicRepository(repoDB).Create(ctx, epic); err != nil {
		t.Fatalf("TC-003 seed epic: %v", err)
	}
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E01-F01", Title: "I-03 blocked feature"}, EpicID: epic.ID, Status: models.FeatureStatusDraft}
	if err := repository.NewFeatureRepository(repoDB).Create(ctx, feature); err != nil {
		t.Fatalf("TC-003 seed feature: %v", err)
	}
	// The peer stays live so TC-003 can prove that keyed-next cascade drops a
	// blocked child and continues to an unrelated eligible sibling.
	unlinkedFeature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E01-F02", Title: "I-03 unlinked feature"}, EpicID: epic.ID, Status: models.FeatureStatus("research")}
	if err := repository.NewFeatureRepository(repoDB).Create(ctx, unlinkedFeature); err != nil {
		t.Fatalf("TC-003 seed unlinked feature: %v", err)
	}

	// These two public calls create the I-01 record and its I-02 workflow
	// state. The link is then driven through the normal generic transport.
	runSharkTC013(t, dbPath, "--json", "question", "create", "I-03 gate", "--summary", "Choose release", "--requester", "release-owner", "--blocking")
	runSharkTC013(t, dbPath, "--json", "question", "configure-workflow", "Q001", "--resolution-owner", "release-owner", "--responder", "alice")
	runSharkTC013(t, dbPath, "--json", "link", "Q001", "E01-F01", "--type", "question_blocks")

	want := services.QuestionBlock{QuestionKey: "Q001", Summary: "Choose release", ResolutionOwner: "release-owner", CurrentResponder: "alice"}
	assertHandoff := func(label, output string) {
		t.Helper()
		if index := strings.LastIndex(output, "\nexit status "); index >= 0 {
			output = output[:index]
		}
		output = strings.TrimSpace(output)
		var envelope struct {
			Action        string                  `json:"action"`
			QuestionBlock *services.QuestionBlock `json:"question_block"`
		}
		if err := json.Unmarshal([]byte(output), &envelope); err != nil {
			t.Fatalf("TC-003 decode %s: %v\n%s", label, err, output)
		}
		if envelope.QuestionBlock == nil || *envelope.QuestionBlock != want {
			t.Fatalf("TC-003 %s handoff = %#v, want %#v", label, envelope.QuestionBlock, want)
		}
		encoded, _ := json.Marshal(envelope.QuestionBlock)
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(encoded, &fields); err != nil || len(fields) != 4 {
			t.Fatalf("TC-003 %s compact fields = %v, decode error = %v", label, fields, err)
		}
	}

	nextOutput := runSharkTC013(t, dbPath, "--json", "next", "E01-F01")
	assertHandoff("next", nextOutput)
	var next commands.NextResponse
	if err := json.Unmarshal([]byte(nextOutput), &next); err != nil || next.Action != "pause" || next.Prompt != "" {
		t.Fatalf("TC-003 next = %#v, err=%v; want pause without prompt", next, err)
	}

	// TC-003: execute the keyed public cascade over the real SQLite hierarchy.
	// The direct child remains paused with its compact handoff, but it is not a
	// live cascade candidate; the sibling must be selected without inheriting
	// the blocked child's handoff or a Question-owned dispatch prompt.
	cascadeNextOutput := runSharkTC013(t, dbPath, "--json", "next", "E01")
	var cascadeNext commands.NextResponse
	if err := json.Unmarshal([]byte(cascadeNextOutput), &cascadeNext); err != nil {
		t.Fatalf("TC-003 decode keyed cascade next: %v\n%s", err, cascadeNextOutput)
	}
	if cascadeNext.EntityKey != "E01-F02" || cascadeNext.Action != "spawn_agent" || cascadeNext.QuestionBlock != nil {
		t.Fatalf("TC-003 keyed cascade next = %#v, want unblocked sibling E01-F02 dispatch", cascadeNext)
	}
	if got := cascadeNext.ResolvedVia; len(got) != 1 || got[0] != "E01" {
		t.Fatalf("TC-003 keyed cascade resolved_via = %v, want [E01]", got)
	}

	// TC-003: when every selected cascade child is parked by the same direct
	// Question gate, the public keyed-next result remains a compact pause. The
	// parent must not erase the only actionable I-03 handoff while collapsing
	// the all-parked child set into its normal pause result.
	runSharkTC013(t, dbPath, "--json", "link", "Q001", "E01-F02", "--type", "question_blocks")
	allParkedBefore := databaseSnapshotTC011(t, sqlDB)
	allParkedOutput := runSharkTC013(t, dbPath, "--json", "next", "E01")
	assertHandoff("next cascade all parked", allParkedOutput)
	var allParked commands.NextResponse
	if err := json.Unmarshal([]byte(allParkedOutput), &allParked); err != nil {
		t.Fatalf("TC-003 decode all-parked keyed cascade next: %v\n%s", err, allParkedOutput)
	}
	if allParked.EntityKey != "E01" || allParked.Action != "pause" || allParked.Prompt != "" {
		t.Fatalf("TC-003 all-parked keyed cascade next = %#v, want compact parent pause", allParked)
	}
	if after := databaseSnapshotTC011(t, sqlDB); after != allParkedBefore {
		t.Fatal("TC-003 all-parked keyed cascade next mutated durable state")
	}
	runSharkTC013(t, dbPath, "--json", "unlink", "Q001", "E01-F02", "--type", "question_blocks")
	runOutput := runSharkTC013(t, dbPath, "--json", "run", "E01-F01", "--dry-run")
	assertHandoff("run dry-run", runOutput)
	var runResult struct {
		Outcome string `json:"outcome"`
	}
	if err := json.Unmarshal([]byte(runOutput), &runResult); err != nil || runResult.Outcome != "paused" {
		t.Fatalf("TC-003 run = %s, err=%v; want paused", runOutput, err)
	}

	// TC-003: the normal public runner must return the identical compact
	// handoff without starting a worker when the directly requested candidate
	// is blocked.
	normalRunOutput := runSharkTC013(t, dbPath, "--json", "run", "E01-F01")
	assertHandoff("run normal", normalRunOutput)
	if err := json.Unmarshal([]byte(normalRunOutput), &runResult); err != nil || runResult.Outcome != "paused" {
		t.Fatalf("TC-003 normal run = %s, err=%v; want paused", normalRunOutput, err)
	}

	// TC-003: restore the second direct gate edge so every selected child is
	// parked. The sibling-fall-through matrix is TC-308 below; this assertion
	// specifically proves that an all-parked public cascade retains the compact
	// handoff from its first blocked child.
	runSharkTC013(t, dbPath, "--json", "link", "Q001", "E01-F02", "--type", "question_blocks")
	// This exercises the public in-process child preflight rather than a
	// controller-only seam.
	cascadeRunOutput := runSharkTC013(t, dbPath, "--json", "run", "E01", "--dry-run")
	assertHandoff("run cascade dry-run", cascadeRunOutput)
	if err := json.Unmarshal([]byte(cascadeRunOutput), &runResult); err != nil || runResult.Outcome != "paused" {
		t.Fatalf("TC-003 cascade run = %s, err=%v; want paused", cascadeRunOutput, err)
	}
	normalCascadeOutput := runSharkTC013(t, dbPath, "--json", "run", "E01")
	assertHandoff("run cascade normal", normalCascadeOutput)
	if err := json.Unmarshal([]byte(normalCascadeOutput), &runResult); err != nil || runResult.Outcome != "paused" {
		t.Fatalf("TC-003 normal cascade run = %s, err=%v; want paused", normalCascadeOutput, err)
	}
	runSharkTC013(t, dbPath, "--json", "unlink", "Q001", "E01-F02", "--type", "question_blocks")
	before := databaseSnapshotTC011(t, sqlDB)

	advanceOutput := runSharkTC011Failure(t, dbPath, "--json", "status", "advance", "E01-F01")
	assertHandoff("status advance", advanceOutput)
	if !strings.Contains(advanceOutput, `"code": "QUESTION_BLOCKED"`) {
		t.Fatalf("TC-003 blocked advance output = %s, want QUESTION_BLOCKED", advanceOutput)
	}
	if after := databaseSnapshotTC011(t, sqlDB); after != before {
		t.Fatal("TC-003 paused/rejected public boundaries mutated durable state")
	}

	// TC-311: a gate edge changes neither the Question's own lifecycle caller
	// nor legacy generic blocks on a peer candidate.
	questionNext := runSharkTC013(t, dbPath, "--json", "next", "Q001")
	var questionResponse commands.NextResponse
	if err := json.Unmarshal([]byte(questionNext), &questionResponse); err != nil || questionResponse.Action != "spawn_agent" || questionResponse.QuestionBlock != nil {
		t.Fatalf("TC-311 Question lifecycle next = %#v, err=%v; want ordinary dispatch", questionResponse, err)
	}
	runSharkTC013(t, dbPath, "link", "Q001", "E01-F02", "--type", "blocks")
	peerNext := runSharkTC013(t, dbPath, "--json", "next", "E01-F02")
	var peerResponse commands.NextResponse
	if err := json.Unmarshal([]byte(peerNext), &peerResponse); err != nil || peerResponse.Action == "pause" || peerResponse.QuestionBlock != nil {
		t.Fatalf("TC-311 generic blocks peer next = %#v, err=%v; want ordinary non-gate dispatch", peerResponse, err)
	}

	// TC-310: only the Question lifecycle closure clears the direct predicate;
	// the linked feature remains untouched until the caller retries its normal
	// supported transition.
	runSharkTC013(t, dbPath, "--json", "question", "withdraw", "Q001", "--owner", "release-owner", "--reason", "decision withdrawn")
	resumed := runSharkTC013(t, dbPath, "--json", "status", "advance", "E01-F01")
	if strings.Contains(resumed, "question_block") || strings.Contains(resumed, "QUESTION_BLOCKED") {
		t.Fatalf("TC-310 closed Question still blocks feature advance: %s", resumed)
	}
}

// TC-004 proves X-06 producer v1 through the public Question boundary. It
// composes the I-01 record, I-02 serial lifecycle, I-03 direct blocker, the
// F04 focused reads, and the explicit full read without creating an E38 queue,
// copied mutable state, or authority over the linked Feature.
func TestTC004_X06ProducerPublicQuestionHandoffIsReadOnly(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "x06-producer.db")
	sqlDB, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("TC-004 InitDB() error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	repoDB := repository.NewDB(sqlDB)
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E01", Title: "X-06 producer epic"}, Status: models.EpicStatusDraft, Priority: models.PriorityMedium}
	if err := repository.NewEpicRepository(repoDB).Create(ctx, epic); err != nil {
		t.Fatalf("TC-004 seed epic: %v", err)
	}
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E01-F01", Title: "X-06 linked work"}, EpicID: epic.ID, Status: models.FeatureStatusDraft}
	if err := repository.NewFeatureRepository(repoDB).Create(ctx, feature); err != nil {
		t.Fatalf("TC-004 seed feature: %v", err)
	}

	// Use the public lifecycle commands to make the durable I-01/I-02 record;
	// source relationship creation deliberately also uses the public generic
	// command so this remains a provider-neutral handoff test.
	runSharkTC013(t, dbPath, "--json", "question", "create", "X-06 decision", "--summary", "Choose the release", "--requester", "release-owner", "--blocking")
	runSharkTC013(t, dbPath, "--json", "question", "configure-workflow", "Q001", "--resolution-owner", "release-owner", "--responder", "alice")
	runSharkTC013(t, dbPath, "--json", "link", "Q001", "E01-F01", "--type", "question_blocks")

	before := databaseSnapshotTC011(t, sqlDB)
	assertCompact := func(label, payload string) {
		t.Helper()
		for _, forbidden := range []string{"context_data", "responses", "evidence_pointer", "resolution_pointer", "resolution_kind", "relationship_id", "prompt", "credential"} {
			if strings.Contains(payload, forbidden) {
				t.Fatalf("TC-004 %s leaks forbidden compact field %q: %s", label, forbidden, payload)
			}
		}
	}

	openOutput := runSharkTC013(t, dbPath, "--json", "question", "open-by-responder", "alice", "--limit", "50", "--offset", "0")
	if !strings.Contains(openOutput, `"key": "Q001"`) {
		t.Fatalf("TC-004 open-by-responder = %s, want Q001", openOutput)
	}
	assertCompact("CLI open-by-responder", openOutput)

	blockingOutput := runSharkTC013(t, dbPath, "--json", "question", "blocking-for", "E01-F01", "--limit", "50", "--offset", "0")
	if !strings.Contains(blockingOutput, `"question_key": "Q001"`) || !strings.Contains(blockingOutput, `"current_responder": "alice"`) {
		t.Fatalf("TC-004 blocking-for = %s, want compact I-03 handoff", blockingOutput)
	}
	assertCompact("CLI blocking-for", blockingOutput)

	fullOutput := runSharkTC013(t, dbPath, "--json", "question", "full", "Q001", "--actor", "alice")
	if !strings.Contains(fullOutput, `"key": "Q001"`) || !strings.Contains(fullOutput, `"responders"`) || strings.Contains(fullOutput, "context_data") {
		t.Fatalf("TC-004 full read = %s, want authorized bounded full projection", fullOutput)
	}
	// I-02's public resume seam remains a Question key and current responder;
	// it does not receive a linked-work action or a host-owned queue payload.
	nextOutput := runSharkTC013(t, dbPath, "--json", "next", "Q001")
	var next commands.NextResponse
	if err := json.Unmarshal([]byte(nextOutput), &next); err != nil {
		t.Fatalf("TC-004 decode keyed next: %v\n%s", err, nextOutput)
	}
	if next.EntityKey != "Q001" || next.Action != "spawn_agent" || strings.Contains(nextOutput, "E01-F01") {
		t.Fatalf("TC-004 keyed next = %#v, want Question-only serial handoff", next)
	}

	// The HTTP path is wired through the production viewer composition root,
	// not a test-created service with optional read dependencies omitted.
	svcs := viewerserver.WireServices(repository.NewDB(sqlDB), projectRootTC011(t))
	mux := http.NewServeMux()
	api.NewQuestionHandler(svcs.QuestionService).RegisterRoutes(mux)
	for _, tc := range []struct {
		path string
		want string
	}{
		{"/api/v1/questions/open-by-responder?responder=alice", `"key":"Q001"`},
		{"/api/v1/questions/blocking-for?entity_key=E01-F01", `"question_key":"Q001"`},
		{"/api/v1/questions/Q001/full?actor=release-owner", `"resolution_owner":"release-owner"`},
	} {
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), tc.want) {
			t.Fatalf("TC-004 GET %s = %d %s, want %q", tc.path, recorder.Code, recorder.Body.String(), tc.want)
		}
		if !strings.Contains(tc.path, "/full?") {
			assertCompact("HTTP "+tc.path, recorder.Body.String())
		}
	}

	// TC-407's forbidden manifest remains black-box observable: no base full
	// switch, generic responder filter, queue route, or E38 activation route.
	for _, args := range [][]string{
		{"question", "get", "Q001", "--full"},
		{"question", "list", "--responder", "alice"},
		{"question", "queue", "Q001"},
		{"e38", "resume", "Q001"},
	} {
		output := runSharkTC011Failure(t, dbPath, args...)
		if strings.TrimSpace(output) == "" {
			t.Fatalf("TC-407 %s returned an empty rejection", strings.Join(args, " "))
		}
	}
	for _, path := range []string{
		"/api/v1/questions?responder=alice",
		"/api/v1/questions/Q001/queue",
		"/api/v1/e38/questions/Q001",
	} {
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code < http.StatusBadRequest {
			t.Fatalf("TC-407 forbidden GET %s status=%d body=%s", path, recorder.Code, recorder.Body.String())
		}
	}
	if after := databaseSnapshotTC011(t, sqlDB); after != before {
		t.Fatal("TC-004/TC-408 public producer reads or rejected probes mutated durable state")
	}
}

// TC-406/TC-407/TC-408 drive the public CLI and the complete production HTTP
// server over one temporary SQLite database.  The focused-route unit tests use
// mocked services and TC-004 mounts only the Question handler; this contract
// protects the wiring and baseline compact-read boundaries that are otherwise
// easy to bypass while adding a new explicit full-read route.
func TestTC406TC407TC408_PublicQuestionReadsStayCompactAndReadOnly(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "f04-public-read-only.db")

	// Seed through the public lifecycle so the CLI, generic search, focused
	// reads, and the production HTTP server observe the same persisted state.
	runSharkTC013(t, dbPath, "--json", "question", "create", "F04 boundary", "--summary", "Safe summary", "--requester", "owner", "--blocking")
	runSharkTC013(t, dbPath, "--json", "question", "configure-workflow", "Q001", "--resolution-owner", "owner", "--responder", "alice")

	sqlDB, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("TC-408 InitDB() error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	questionRepo := repository.NewQuestionRepository(repository.NewDB(sqlDB))
	registry := services.NewEntityRegistry()
	registry.Register(models.EntityTypeQuestion, services.NewQuestionRepositoryAdapter(questionRepo))
	contextSvc, err := services.NewContextService(registry)
	if err != nil {
		t.Fatalf("TC-406 construct Question context service: %v", err)
	}
	if err := contextSvc.SetContextField(ctx, models.EntityTypeQuestion, "Q001", "current_step", "TC-406-private-context"); err != nil {
		t.Fatalf("TC-406 seed context sentinel: %v", err)
	}

	before := databaseSnapshotTC011(t, sqlDB)
	assertCompact := func(label, payload string) {
		t.Helper()
		for _, forbidden := range []string{"context_data", "responses", "evidence_pointer", "resolution_pointer", "resolution_kind", "relationship_id", "prompt", "credential"} {
			if strings.Contains(payload, forbidden) {
				t.Fatalf("%s leaked compact-only field %q: %s", label, forbidden, payload)
			}
		}
		if strings.Contains(payload, "TC-406-private-context") {
			t.Fatalf("%s leaked persisted context sentinel: %s", label, payload)
		}
	}

	// TC-406: existing CLI reads retain their compact projections and the
	// generic discovery surface does not grow focused filters or a full switch.
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"question get", []string{"--json", "question", "get", "Q001"}, `"key": "Q001"`},
		{"question list", []string{"--json", "question", "list"}, `"key": "Q001"`},
		{"search", []string{"--json", "search", "F04 boundary", "--type=question"}, `"key": "Q001"`},
	} {
		output := runSharkTC013(t, dbPath, tc.args...)
		if !strings.Contains(output, tc.want) {
			t.Fatalf("TC-406 CLI %s = %s, want %s", tc.name, output, tc.want)
		}
		assertCompact("TC-406 CLI "+tc.name, output)
	}
	for _, args := range [][]string{
		{"question", "get", "Q001", "--full"},
		{"question", "list", "--responder", "alice"},
		{"question", "open-by-responder", " alice "},
	} {
		if output := runSharkTC011Failure(t, dbPath, args...); strings.TrimSpace(output) == "" {
			t.Fatalf("TC-407 CLI %s returned an empty rejection", strings.Join(args, " "))
		}
	}
	// Defect-class sweep: both public identity-bearing CLI reads must reject
	// malformed identity forms at their transport boundary, as their HTTP
	// counterparts do below. This prevents one public surface from treating an
	// invalid policy identity as a distinct, service-visible principal.
	for _, identity := range []string{string([]byte{0xff}), strings.Repeat("a", 257), "bearer token"} {
		for _, args := range [][]string{
			{"question", "open-by-responder", identity},
			{"question", "full", "Q001", "--actor", identity},
		} {
			if output := runSharkTC011Failure(t, dbPath, args...); strings.TrimSpace(output) == "" {
				t.Fatalf("TC-401/TC-405 malformed CLI identity %q returned an empty rejection", identity)
			}
		}
	}

	// StartServer is the production HTTP composition root: it supplies the
	// real service graph, route registration, viewer route, and middleware.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("TC-406 listen: %v", err)
	}
	serverCtx, cancel := context.WithCancel(context.Background())
	ready := make(chan struct{})
	done := make(chan error, 1)
	projectRoot := projectRootTC011(t)
	go func() {
		done <- viewerserver.StartServer(serverCtx, viewerserver.Options{
			Listener: listener, DB: repository.NewDB(sqlDB), ProjectRoot: projectRoot, Ready: ready,
		})
	}()
	select {
	case <-ready:
	case err := <-done:
		t.Fatalf("TC-406 production server stopped before ready: %v", err)
	}
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("TC-406 production server shutdown: %v", err)
		}
	})

	baseURL := "http://" + listener.Addr().String()
	get := func(path string, wantStatus int, want string) string {
		t.Helper()
		response, err := http.Get(baseURL + path) // #nosec G107 -- loopback test server
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatalf("read GET %s: %v", path, err)
		}
		payload := string(body)
		if (wantStatus == 0 && response.StatusCode < http.StatusBadRequest) || (wantStatus != 0 && response.StatusCode != wantStatus) || (want != "" && !strings.Contains(payload, want)) {
			t.Fatalf("GET %s = %d %s, want %d containing %q", path, response.StatusCode, payload, wantStatus, want)
		}
		return payload
	}

	// TC-406: static full route wins, generic get/list remain compact, and the
	// existing viewer surface stays a safe generic projection.
	for _, tc := range []struct{ path, want string }{
		{"/api/v1/questions/Q001", `"key":"Q001"`},
		{"/api/v1/questions", `"key":"Q001"`},
		{"/api/v1/viewer/history/Q001", `"entity_type":"question"`},
		{"/api/v1/questions/open-by-responder?responder=alice", `"key":"Q001"`},
	} {
		assertCompact("TC-406 production HTTP "+tc.path, get(tc.path, http.StatusOK, tc.want))
	}
	get("/api/v1/questions/open-by-responder?responder=%20alice%20", 0, "")
	for _, path := range []string{
		"/api/v1/questions/open-by-responder?responder=%FF",
		"/api/v1/questions/open-by-responder?responder=" + strings.Repeat("a", 257),
		"/api/v1/questions/open-by-responder?responder=bearer%20token",
		"/api/v1/questions/Q001/full?actor=%FF",
		"/api/v1/questions/Q001/full?actor=" + strings.Repeat("a", 257),
		"/api/v1/questions/Q001/full?actor=bearer%20token",
	} {
		get(path, 0, "")
	}
	full := get("/api/v1/questions/Q001/full?actor=alice", http.StatusOK, `"responders"`)
	if strings.Contains(full, "context_data") || strings.Contains(full, "TC-406-private-context") {
		t.Fatalf("TC-406 explicit full projection leaked raw context: %s", full)
	}

	// TC-407: forbidden base filters/switches and queue/activation routes stay
	// rejected by the actual public server, not merely absent from a local mux.
	for _, path := range []string{
		"/api/v1/questions?responder=alice",
		"/api/v1/questions/Q001/full?actor=alice&unexpected=value",
		"/api/v1/questions/Q001/queue",
		"/api/v1/e38/questions/Q001",
	} {
		get(path, 0, "")
	}
	if after := databaseSnapshotTC011(t, sqlDB); after != before {
		t.Fatal("TC-408 CLI/production HTTP reads or rejected probes mutated durable state")
	}
}

// TC-308 drives the public Cobra runner over a real hierarchy instead of
// substituting the runner's child callback.  It pins the dispatch outcome
// matrix that the helper-level tests cannot observe: a blocked first child is
// parked while its independent sibling runs, in both normal and dry-run mode.
// The temporary feature workflow makes the sibling an agent-free
// advance-to-terminal action, so the normal command remains hermetic while
// still traversing the actual cascade and lease boundaries.
func TestTC308_PublicRunCascadeFallsThroughBlockedChildInNormalAndDryRun(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		t.Run(map[bool]string{false: "normal", true: "dry_run"}[dryRun], func(t *testing.T) {
			dbPath, sqlDB, projectRoot := publicRunCascadeFixtureTC308(t)
			defer sqlDB.Close()
			actionOutput := runSharkTC308(t, projectRoot, dbPath, "--json", "admin", "workflow", "show-action", "E02-F02", "research")
			if !strings.Contains(actionOutput, `"action": "advance_status"`) {
				t.Fatalf("TC-308 custom sibling action = %s, want advance_status", actionOutput)
			}

			args := []string{"--json", "run", "E02"}
			if dryRun {
				args = append(args, "--dry-run")
			}
			output := runSharkTC308(t, projectRoot, dbPath, args...)
			var result struct {
				EntityKey     string                  `json:"entity_key"`
				Outcome       string                  `json:"outcome"`
				QuestionBlock *services.QuestionBlock `json:"question_block"`
				Stages        []struct {
					Status string `json:"status"`
					Action string `json:"action"`
				} `json:"stages"`
			}
			if err := json.Unmarshal([]byte(output), &result); err != nil {
				t.Fatalf("TC-308 decode public cascade run: %v\n%s", err, output)
			}
			if result.EntityKey != "E02" || result.Outcome != "paused" || result.QuestionBlock != nil {
				t.Fatalf("TC-308 public cascade run = %#v question_block=%+v, want sibling-driven parent pause without the parked-child handoff\n%s", result, result.QuestionBlock, output)
			}
			if len(result.Stages) == 0 || result.Stages[0].Status != "research" || result.Stages[0].Action != "advance_status" {
				t.Fatalf("TC-308 cascade stages = %#v, want the unblocked research sibling to run after the blocked child", result.Stages)
			}

			var blockedStatus, siblingStatus string
			if err := sqlDB.QueryRow("SELECT status FROM features WHERE key = 'E02-F01'").Scan(&blockedStatus); err != nil {
				t.Fatalf("TC-308 read blocked child status: %v", err)
			}
			if err := sqlDB.QueryRow("SELECT status FROM features WHERE key = 'E02-F02'").Scan(&siblingStatus); err != nil {
				t.Fatalf("TC-308 read sibling status: %v", err)
			}
			if blockedStatus != "draft" {
				t.Fatalf("TC-308 blocked child status = %q, want draft", blockedStatus)
			}
			if dryRun {
				if siblingStatus != "research" {
					t.Fatalf("TC-308 dry-run sibling status = %q, want research", siblingStatus)
				}
			} else if siblingStatus != "completed" {
				t.Fatalf("TC-308 normal sibling status = %q, want completed", siblingStatus)
			}
		})
	}
}

func publicRunCascadeFixtureTC308(t *testing.T) (string, *sql.DB, string) {
	t.Helper()
	ctx := context.Background()
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "tc308-cascade.db")
	sqlDB, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("TC-308 InitDB: %v", err)
	}
	repoDB := repository.NewDB(sqlDB)
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E02", Title: "TC-308 cascade root"}, Status: models.EpicStatusActive, Priority: models.PriorityMedium}
	if err := repository.NewEpicRepository(repoDB).Create(ctx, epic); err != nil {
		t.Fatalf("TC-308 seed epic: %v", err)
	}
	blocked := &models.Feature{BaseEntity: models.BaseEntity{Key: "E02-F01", Title: "TC-308 blocked child"}, EpicID: epic.ID, Status: models.FeatureStatusDraft}
	sibling := &models.Feature{BaseEntity: models.BaseEntity{Key: "E02-F02", Title: "TC-308 runnable sibling"}, EpicID: epic.ID, Status: models.FeatureStatus("research")}
	featureRepo := repository.NewFeatureRepository(repoDB)
	if err := featureRepo.Create(ctx, blocked); err != nil {
		t.Fatalf("TC-308 seed blocked child: %v", err)
	}
	if err := featureRepo.Create(ctx, sibling); err != nil {
		t.Fatalf("TC-308 seed sibling: %v", err)
	}
	questionRepo := repository.NewQuestionRepository(repoDB)
	questionSvc, err := services.NewQuestionService(questionRepo)
	if err != nil {
		t.Fatalf("TC-308 NewQuestionService: %v", err)
	}
	question, err := questionSvc.CreateQuestion(ctx, services.CreateQuestionInput{Title: "TC-308 gate", Summary: "Keep the first child parked", Requester: "release-owner", Blocking: true})
	if err != nil {
		t.Fatalf("TC-308 create Question: %v", err)
	}
	if _, err := questionSvc.ConfigureWorkflow(ctx, services.ConfigureWorkflowInput{Key: question.Key, ResolutionOwner: "release-owner", Responders: []string{"alice"}}); err != nil {
		t.Fatalf("TC-308 configure Question: %v", err)
	}
	if err := repository.NewEntityRelationshipRepository(repoDB).Create(ctx, &models.EntityRelationship{
		FromEntityType: models.EntityTypeQuestion, FromEntityID: question.ID,
		ToEntityType: models.EntityTypeFeature, ToEntityID: blocked.ID,
		RelationshipType: models.EntityRelQuestionBlocks,
	}); err != nil {
		t.Fatalf("TC-308 link Question block: %v", err)
	}

	workflowDir := filepath.Join(tmp, "workflow")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("TC-308 create workflow directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workflowDir, "feature.yaml"), []byte(`version: "1.0"
start: research
steps:
  research:
    phase: development
    action: advance_status
    outcomes:
      pass: completed
  completed:
    phase: done
    action: archive
    terminal: true
`), 0o644); err != nil {
		t.Fatalf("TC-308 write feature workflow: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workflowDir, "epic.yaml"), []byte(`version: "1.0"
start: active
steps:
  active:
    phase: execution
    action: cascade
    outcomes:
      pass: completed
  completed:
    phase: done
    action: archive
    terminal: true
`), 0o644); err != nil {
		t.Fatalf("TC-308 write epic workflow: %v", err)
	}
	configPath := filepath.Join(tmp, ".sharkconfig.json")
	if err := os.WriteFile(configPath, []byte(fmt.Sprintf(`{"workflow_config":%q}`, workflowDir)), 0o644); err != nil {
		t.Fatalf("TC-308 write config: %v", err)
	}
	return dbPath, sqlDB, tmp
}

// runSharkTC308 builds the Cobra binary once per isolated fixture, then runs
// it with the fixture as its working directory.  That matters for this public
// test: WorkflowService intentionally discovers its project root from cwd,
// so a --config flag alone would not exercise the same workflow route used by
// the runner.
func runSharkTC308(t *testing.T, projectRoot, dbPath string, args ...string) string {
	t.Helper()
	binaryPath := filepath.Join(projectRoot, "shark")
	build := exec.Command("go", "build", "-buildvcs=false", "-o", binaryPath, "./cmd/shark")
	build.Dir = projectRootTC011(t)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("TC-308 build shark: %v\n%s", err, output)
	}
	command := exec.Command(binaryPath, append([]string{"--db", dbPath}, args...)...)
	command.Dir = projectRoot
	return runSharkCaptureSeparate(t, command, "TC-308 shark", args)
}

// runSharkCaptureSeparate runs cmd with stdout/stderr captured separately
// (not combined): `shark run` now writes its liveness stream to stderr
// unconditionally in both --json and plain mode (E40-F04 D2/D6), so a
// combined stream would corrupt the pure JSON this helper's callers decode
// from stdout. Both streams are still surfaced on failure for debugging.
func runSharkCaptureSeparate(t *testing.T, cmd *exec.Cmd, failLabel string, args []string) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s %s failed: %v\nstdout:\n%s\nstderr:\n%s", failLabel, strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// TC-312 protects F03's deliberately narrow producer boundary. The source
// scan is limited to code owned by this feature so it neither precludes F04's
// future read surfaces nor turns later E38 work into an F03 failure.
func TestTC312_F03ForbiddenV1StructuralAndBlackBoxGuard(t *testing.T) {
	projectRoot := projectRootTC011(t)
	var blockerSource string
	for _, path := range []string{
		filepath.Join(projectRoot, "internal", "services", "question_blocker.go"),
		filepath.Join(projectRoot, "internal", "cli", "commands", "link.go"),
		filepath.Join(projectRoot, "internal", "cli", "commands", "next.go"),
		filepath.Join(projectRoot, "internal", "cli", "commands", "run.go"),
		filepath.Join(projectRoot, "internal", "cli", "commands", "status_group.go"),
		filepath.Join(projectRoot, "internal", "api", "viewer", "mutation_handler.go"),
		filepath.Join(projectRoot, "internal", "api", "viewer", "mutation_service.go"),
	} {
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("TC-312 read %s: %v", path, err)
		}
		if strings.HasSuffix(path, filepath.Join("services", "question_blocker.go")) {
			blockerSource = string(contents)
		}
		for _, forbidden := range []string{"/open-by-responder", "/blocking-for", "E38", "QuestionBlocker.Create", "QuestionBlocker.Update"} {
			if strings.Contains(string(contents), forbidden) {
				t.Errorf("f03-forbidden-v1: %s contains forbidden %q", path, forbidden)
			}
		}
	}
	// The gate may make exactly one direct incoming relationship lookup. It
	// must not become a graph walker or gain a write-capable relationship seam.
	if strings.Count(blockerSource, ".GetIncoming(") != 1 {
		t.Errorf("f03-forbidden-v1: QuestionBlocker GetIncoming calls = %d, want exactly one direct lookup", strings.Count(blockerSource, ".GetIncoming("))
	}
	for _, forbidden := range []string{"GetOutgoing", "Traverse", "Recursive", "Walk", "CreateRelationship", "UnlinkEntities", "TransitionStatus", "AddNote", "Claim("} {
		if strings.Contains(blockerSource, forbidden) {
			t.Errorf("f03-forbidden-v1: QuestionBlocker contains forbidden %q", forbidden)
		}
	}

	dbPath := filepath.Join(t.TempDir(), "f03-forbidden.db")
	ctx := context.Background()
	sqlDB, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("TC-312 InitDB() error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	repoDB := repository.NewDB(sqlDB)
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E01", Title: "F03 forbidden epic"}, Status: models.EpicStatusDraft, Priority: models.PriorityMedium}
	if err := repository.NewEpicRepository(repoDB).Create(ctx, epic); err != nil {
		t.Fatalf("TC-312 seed epic: %v", err)
	}
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E01-F01", Title: "F03 forbidden target"}, EpicID: epic.ID, Status: models.FeatureStatusDraft}
	if err := repository.NewFeatureRepository(repoDB).Create(ctx, feature); err != nil {
		t.Fatalf("TC-312 seed feature: %v", err)
	}
	if err := repository.NewTaskRepository(repoDB).Create(ctx, &models.Task{BaseEntity: models.BaseEntity{Key: "T-E01-F01-001", Title: "F03 indirect target"}, FeatureID: feature.ID, Status: "development", Priority: 5}); err != nil {
		t.Fatalf("TC-312 seed task: %v", err)
	}
	runSharkTC013(t, dbPath, "question", "create", "F03 source", "--summary", "No disclosure", "--requester", "owner", "--blocking")
	runSharkTC013(t, dbPath, "question", "create", "F03 Question target", "--summary", "No target", "--requester", "owner")
	runSharkTC013(t, dbPath, "sprint", "create", "F03 sprint", "--start", "2026-07-01", "--end", "2026-07-15")
	runSharkTC013(t, dbPath, "idea", "create", "F03 idea", "--size", "S")

	for _, args := range [][]string{
		{"question", "open-by-responder"},
		{"question", "queue", "Q001"},
		{"e38", "question", "Q001"},
		{"link", "E01-F01", "Q001", "--type", "question_blocks"},
		{"link", "Q001", "Q002", "--type", "question_blocks"},
		{"link", "Q001", "S001", "--type", "question_blocks"},
		{"link", "Q001", "I-2026-07-31-01", "--type", "question_blocks"},
	} {
		before := databaseSnapshotTC011(t, sqlDB)
		output := runSharkTC011Failure(t, dbPath, args...)
		if after := databaseSnapshotTC011(t, sqlDB); after != before {
			t.Errorf("TC-312 %s mutated the database", strings.Join(args, " "))
		}
		if strings.TrimSpace(output) == "" {
			t.Errorf("TC-312 %s returned an empty rejection", strings.Join(args, " "))
		}
	}

	// A direct edge must never recurse into a child. The public keyed-next
	// path is the relevant probe because it owns cascade traversal separately.
	runSharkTC013(t, dbPath, "link", "Q001", "E01-F01", "--type", "question_blocks")
	beforeIndirect := databaseSnapshotTC011(t, sqlDB)
	indirectOutput := runSharkTC013(t, dbPath, "--json", "next", "T-E01-F01-001")
	if strings.Contains(indirectOutput, "question_block") {
		t.Errorf("TC-312 indirect child was blocked by recursive traversal: %s", indirectOutput)
	}
	if afterIndirect := databaseSnapshotTC011(t, sqlDB); afterIndirect != beforeIndirect {
		t.Error("TC-312 indirect traversal probe mutated the database")
	}

	// F03 owns no queue or E38 adapter route. F04 may add its separately
	// contracted focused reads; exercise the remaining F03 boundary through the
	// public Question API registration rather than inferring absence from text.
	questionRepo := repository.NewQuestionRepository(repository.NewDB(sqlDB))
	questionSvc, err := services.NewQuestionService(questionRepo)
	if err != nil {
		t.Fatalf("TC-312 NewQuestionService() error = %v", err)
	}
	mux := http.NewServeMux()
	api.NewQuestionHandler(questionSvc).RegisterRoutes(mux)
	for _, path := range []string{
		"/api/v1/questions/Q001/queue",
		"/api/v1/e38/questions/Q001",
	} {
		before := databaseSnapshotTC011(t, sqlDB)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code < http.StatusBadRequest {
			t.Errorf("TC-312 forbidden route %s status = %d, want rejection", path, response.Code)
		}
		if after := databaseSnapshotTC011(t, sqlDB); after != before {
			t.Errorf("TC-312 forbidden route %s mutated the database", path)
		}
	}
}

// TC-309 is intentionally a black-box matrix. The command must not merely
// construct the typed error at a helper seam: every supported key route must
// stop before its distinct transition service can write durable state.
func TestTC309_F03BlockedAdvanceMatrixIsAtomic(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "f03-blocked-advance-matrix.db")
	sqlDB, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("TC-309 InitDB() error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	repoDB := repository.NewDB(sqlDB)
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E01", Title: "blocked epic"}, Status: models.EpicStatusDraft, Priority: models.PriorityMedium}
	if err := repository.NewEpicRepository(repoDB).Create(ctx, epic); err != nil {
		t.Fatalf("TC-309 seed epic: %v", err)
	}
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E01-F01", Title: "blocked feature"}, EpicID: epic.ID, Status: models.FeatureStatusDraft}
	if err := repository.NewFeatureRepository(repoDB).Create(ctx, feature); err != nil {
		t.Fatalf("TC-309 seed feature: %v", err)
	}
	if err := repository.NewTaskRepository(repoDB).Create(ctx, &models.Task{BaseEntity: models.BaseEntity{Key: "T-E01-F01-001", Title: "blocked task"}, FeatureID: feature.ID, Status: "development", Priority: 5}); err != nil {
		t.Fatalf("TC-309 seed task: %v", err)
	}
	if err := repository.NewBugRepository(repoDB).Create(ctx, &models.Bug{BaseEntity: models.BaseEntity{Key: "B001", Title: "blocked bug"}, Status: "reported", Severity: models.BugSeverityMedium}); err != nil {
		t.Fatalf("TC-309 seed bug: %v", err)
	}
	if err := repository.NewChangeCardRepository(repoDB).Create(ctx, &models.ChangeCard{BaseEntity: models.BaseEntity{Key: "CC-001", Title: "blocked change"}, Status: "proposed", Priority: 5}); err != nil {
		t.Fatalf("TC-309 seed change: %v", err)
	}
	if err := repository.NewTechDebtRepository(repoDB).Create(ctx, &models.TechDebt{BaseEntity: models.BaseEntity{Key: "TD-001", Title: "blocked debt"}, Status: "identified", Category: models.TechDebtCategoryCodeQuality, Severity: models.TechDebtSeverityMedium}); err != nil {
		t.Fatalf("TC-309 seed tech debt: %v", err)
	}
	runSharkTC013(t, dbPath, "question", "create", "Advance gate", "--summary", "Release decision", "--requester", "owner", "--blocking")
	runSharkTC013(t, dbPath, "question", "configure-workflow", "Q001", "--resolution-owner", "owner", "--responder", "alice")

	for _, key := range []string{"E01", "E01-F01", "T-E01-F01-001", "B001", "CC-001", "TD-001"} {
		t.Run(key, func(t *testing.T) {
			runSharkTC013(t, dbPath, "link", "Q001", key, "--type", "question_blocks")
			before := databaseSnapshotTC011(t, sqlDB)
			output := runSharkTC011Failure(t, dbPath, "--json", "status", "advance", key)
			if !strings.Contains(output, `"code": "QUESTION_BLOCKED"`) || !strings.Contains(output, `"question_key": "Q001"`) {
				t.Fatalf("TC-309 %s output = %s, want compact QUESTION_BLOCKED error", key, output)
			}
			if after := databaseSnapshotTC011(t, sqlDB); after != before {
				t.Fatalf("TC-309 blocked advance for %s mutated durable state", key)
			}
		})
	}
}

// TC-102 / TC-110: generic context commands remain available to Questions,
// but cannot erase the I-02 value which those commands neither own nor expose.
// This calls the production Cobra entrypoint over SQLite rather than a service
// helper so it catches a lossy generic DTO in the supported user surface.
func TestTC102_I02GenericContextWritesPreserveQuestionOwnedState(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "i02-context-preservation.db")
	runSharkTC013(t, dbPath, "question", "create", "Configured question", "--summary", "Question summary", "--requester", "release-owner")
	runSharkTC013(t, dbPath, "question", "configure-workflow", "Q001", "--resolution-owner", "release-owner", "--responder", "alice", "--responder", "bob")

	// The generic set path must retain Question-owned routing state, so the
	// production keyed-next caller still sees Alice after the update.
	runSharkTC013(t, dbPath, "context", "set", "Q001", "--field", "current_step", "--value", "generic progress")
	output := runSharkTC013(t, dbPath, "--json", "next", "Q001")
	var next commands.NextResponse
	if err := json.Unmarshal([]byte(output), &next); err != nil {
		t.Fatalf("TC-102 decode keyed next after context set: %v\n%s", err, output)
	}
	if next.Action != "spawn_agent" || !strings.Contains(next.Prompt, "currently routed responder: alice") {
		t.Fatalf("TC-102 keyed next after context set = %#v, want Alice dispatch", next)
	}

	sqlDB, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("TC-102 InitDB() error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	beforeConfiguredClear := questionStateTC007(t, sqlDB, "Q001")
	clearOutput := runSharkTC011Failure(t, dbPath, "context", "clear", "Q001")
	if !strings.Contains(clearOutput, "Question-owned state") {
		t.Fatalf("TC-102 context clear configured Question = %q, want Question-owned-state rejection", clearOutput)
	}
	if after := questionStateTC007(t, sqlDB, "Q001"); after != beforeConfiguredClear {
		t.Fatalf("TC-102 configured context clear changed Question: got %q, want %q", after, beforeConfiguredClear)
	}

	// Complete a second Question and withdraw it so the same generic set path
	// proves it keeps both bounded response evidence and terminal provenance.
	runSharkTC013(t, dbPath, "question", "create", "Terminal question", "--summary", "Terminal summary", "--requester", "release-owner")
	runSharkTC013(t, dbPath, "question", "configure-workflow", "Q002", "--resolution-owner", "release-owner", "--responder", "alice")
	runSharkTC013(t, dbPath, "claim", "Q002", "--by", "alice", "--session", "alice-session")
	runSharkTC013(t, dbPath, "question", "respond", "Q002", "--session", "alice-session", "--responder", "alice", "--summary", "approved", "--evidence-pointer", "docs/plan/E39-question-and-decision-workflow-management/E39-F02-serial-question-workflow-and-resolution-provenance/spec.md")
	runSharkTC013(t, dbPath, "release", "Q002", "--session", "alice-session", "--outcome", "pass")
	runSharkTC013(t, dbPath, "question", "withdraw", "Q002", "--owner", "release-owner", "--reason", "no longer needed")
	runSharkTC013(t, dbPath, "context", "set", "Q002", "--field", "current_step", "--value", "retained generic progress")

	var contextData string
	if err := sqlDB.QueryRowContext(ctx, "SELECT COALESCE(context_data, '') FROM questions WHERE key = 'Q002'").Scan(&contextData); err != nil {
		t.Fatalf("TC-102 load terminal Question context: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(contextData), &fields); err != nil {
		t.Fatalf("TC-102 decode terminal Question context: %v", err)
	}
	if _, ok := fields["question_state"]; !ok {
		t.Fatal("TC-102 generic context set removed question_state")
	}
	if _, ok := fields["question_terminal_provenance"]; !ok {
		t.Fatal("TC-102 generic context set removed terminal provenance")
	}
	var state models.QuestionState
	if err := json.Unmarshal(fields["question_state"], &state); err != nil || len(state.Responses) != 1 || state.Responses[0].Responder != "alice" {
		t.Fatalf("TC-102 generic context set did not preserve bounded response: state=%#v err=%v", state, err)
	}
	beforeTerminalClear := questionStateTC007(t, sqlDB, "Q002")
	clearOutput = runSharkTC011Failure(t, dbPath, "context", "clear", "Q002")
	if !strings.Contains(clearOutput, "Question-owned state") {
		t.Fatalf("TC-102 context clear terminal Question = %q, want Question-owned-state rejection", clearOutput)
	}
	if after := questionStateTC007(t, sqlDB, "Q002"); after != beforeTerminalClear {
		t.Fatalf("TC-102 terminal context clear changed Question: got %q, want %q", after, beforeTerminalClear)
	}
}

// TC-110 keeps F02's compact base commands closed. F04 owns explicitly named
// focused reads, but it must not retrofit a full switch or responder filter
// onto F02's existing get/list transports.
func TestTC110_F02ForbiddenV1StructuralAndBlackBoxGuard(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "forbidden.db")
	runSharkTC013(t, dbPath, "question", "create", "F01 boundary question", "--summary", "guard", "--requester", "contract")
	sqlDB, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB() error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	for _, args := range [][]string{{"question", "get", "Q001", "--full"}, {"question", "list", "--responder", "alice"}} {
		before := databaseSnapshotTC011(t, sqlDB)
		output := runSharkTC011Failure(t, dbPath, args...)
		if !strings.Contains(strings.ToLower(output), "unknown flag") {
			t.Errorf("f02-forbidden-v1 shark %s output = %q, want unknown-flag error", strings.Join(args, " "), output)
		}
		if after := databaseSnapshotTC011(t, sqlDB); after != before {
			t.Errorf("f02-forbidden-v1 shark %s mutated the database", strings.Join(args, " "))
		}
	}

	mux := http.NewServeMux()
	questionRepo := repository.NewQuestionRepository(repository.NewDB(sqlDB))
	questionSvc, err := services.NewQuestionService(questionRepo)
	if err != nil {
		t.Fatalf("NewQuestionService() error = %v", err)
	}
	api.NewQuestionHandler(questionSvc).RegisterRoutes(mux)
	for _, operation := range []struct{ method, path string }{{http.MethodGet, "/api/v1/questions?responder=alice"}} {
		before := databaseSnapshotTC011(t, sqlDB)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(operation.method, operation.path, nil))
		if response.Code < http.StatusBadRequest {
			t.Errorf("f02-forbidden-v1 %s %s status = %d, want current route rejection", operation.method, operation.path, response.Code)
		}
		if after := databaseSnapshotTC011(t, sqlDB); after != before {
			t.Errorf("f02-forbidden-v1 %s %s mutated the database", operation.method, operation.path)
		}
	}
}

// TC-012 records the versioned pre-F01 registration inventory. Each row uses
// the production generic callers over a SQLite-backed, fully registered
// fixture. This prevents a new type registration from silently displacing an
// established type on any polymorphic surface.
func TestTC012_RegistrationBaselineV1(t *testing.T) {
	const baselineName = "registration-baseline-v1"
	baseline := []struct {
		name             string
		key              string
		type_            models.EntityType
		parsedType       keys.EntityType
		contextSupported bool
		nextAction       string
		nextError        string
	}{
		{"epic", "E01", models.EntityTypeEpic, keys.EntityTypeEpic, true, "spawn_agent", ""},
		{"feature", "E01-F01", models.EntityTypeFeature, keys.EntityTypeFeature, true, "spawn_agent", ""},
		{"task", "T-E01-F01-001", models.EntityTypeTask, keys.EntityTypeTask, true, "spawn_agent", ""},
		{"bug", "B001", models.EntityTypeBug, keys.EntityTypeBug, true, "spawn_agent", ""},
		{"change", "CC-001", models.EntityTypeChange, keys.EntityTypeChange, true, "spawn_agent", ""},
		{"tech_debt", "TD-001", models.EntityTypeTechDebt, keys.EntityTypeUnknown, true, "spawn_agent", ""},
		// B059: sprint now resolves through the generic transitioner (buildTransitioner/
		// SprintService.GetNextStatus), so `shark next` no longer errors with
		// "unsupported entity type: sprint" — it dispatches the shipped default
		// workflow's planning step (spawn_agent) like the other route-based types.
		{"sprint", "S001", models.EntityTypeSprint, keys.EntityTypeSprint, false, "spawn_agent", ""},
		{"idea", "I-2026-01-01-01", models.EntityTypeIdea, keys.EntityTypeUnknown, false, "", `unsupported entity type: "idea"`},
	}

	ctx := context.Background()
	dbPath, sqlDB, registry := registrationBaselineFixtureTC012(t)
	noteSvc, err := services.NewNoteService(repository.NewEntityNoteRepository(repository.NewDB(sqlDB)), registry)
	if err != nil {
		t.Fatalf("%s create NoteService: %v", baselineName, err)
	}
	contextSvc, err := services.NewContextService(registry)
	if err != nil {
		t.Fatalf("%s create ContextService: %v", baselineName, err)
	}
	historySvc := services.NewEntityHistoryService(repository.NewEntityHistoryRepository(repository.NewDB(sqlDB)), registry)
	claimTTL := time.Hour
	claimSvc := services.NewClaimService(claimrepo.NewRepository(repository.NewDB(sqlDB)), &claimTTL)

	for _, row := range baseline {
		t.Run(row.name, func(t *testing.T) {
			t.Run("key_parse_normalize", func(t *testing.T) {
				parsed := keys.NewKeyService().Parse(row.key)
				if parsed.EntityType != row.parsedType {
					t.Fatalf("%s parse %q entity type = %q, want %q", baselineName, row.key, parsed.EntityType, row.parsedType)
				}
				if row.parsedType == keys.EntityTypeUnknown && parsed.Normalized != "" {
					t.Fatalf("%s parse %q normalized unknown type to %q", baselineName, row.key, parsed.Normalized)
				}
				if row.parsedType != keys.EntityTypeUnknown && parsed.Normalized != row.key {
					t.Fatalf("%s parse/normalize %q = %#v, want canonical %q", baselineName, row.key, parsed, row.key)
				}
				if normalized := keys.Normalize(strings.ToLower(row.key)); normalized != row.key {
					t.Fatalf("%s Normalize(%q) = %q, want %q", baselineName, strings.ToLower(row.key), normalized, row.key)
				}
			})

			t.Run("generic_command_detection", func(t *testing.T) {
				if got := commands.DetectEntityType(row.key); got != string(row.type_) {
					t.Fatalf("%s generic command detection %q = %q, want %q", baselineName, row.key, got, row.type_)
				}
			})

			t.Run("registry_resolution", func(t *testing.T) {
				if _, err := registry.GetRepository(row.type_); err != nil {
					t.Fatalf("%s registry resolution for %s: %v", baselineName, row.type_, err)
				}
			})

			t.Run("notes_add_list", func(t *testing.T) {
				content := fmt.Sprintf("%s %s note", baselineName, row.name)
				if _, err := noteSvc.AddNote(ctx, row.type_, row.key, "testing", content, "contract"); err != nil {
					t.Fatalf("%s note add for %s: %v", baselineName, row.type_, err)
				}
				notes, err := noteSvc.ListNotes(ctx, row.type_, row.key, []string{"testing"})
				if err != nil || len(notes) != 1 || notes[0].Content != content {
					t.Fatalf("%s note list for %s = %#v, %v", baselineName, row.type_, notes, err)
				}
			})

			t.Run("context_get_set_clear", func(t *testing.T) {
				if !row.contextSupported {
					if _, err := contextSvc.GetContext(ctx, row.type_, row.key); err == nil {
						t.Fatalf("%s context get for %s succeeded, want unsupported error", baselineName, row.type_)
					}
					if err := contextSvc.SetContextField(ctx, row.type_, row.key, "current_step", "baseline"); err == nil {
						t.Fatalf("%s context set for %s succeeded, want unsupported error", baselineName, row.type_)
					}
					if err := contextSvc.ClearContext(ctx, row.type_, row.key); err == nil {
						t.Fatalf("%s context clear for %s succeeded, want unsupported error", baselineName, row.type_)
					}
					return
				}
				if err := contextSvc.SetContextField(ctx, row.type_, row.key, "current_step", "baseline"); err != nil {
					t.Fatalf("%s context set for %s: %v", baselineName, row.type_, err)
				}
				got, err := contextSvc.GetContext(ctx, row.type_, row.key)
				if err != nil || got == nil || got.Progress == nil || got.Progress.CurrentStep == nil || *got.Progress.CurrentStep != "baseline" {
					t.Fatalf("%s context get for %s = %#v, %v", baselineName, row.type_, got, err)
				}
				if err := contextSvc.ClearContext(ctx, row.type_, row.key); err != nil {
					t.Fatalf("%s context clear for %s: %v", baselineName, row.type_, err)
				}
				if cleared, err := contextSvc.GetContext(ctx, row.type_, row.key); err != nil || cleared != nil {
					t.Fatalf("%s context after clear for %s = %#v, %v", baselineName, row.type_, cleared, err)
				}
			})

			t.Run("history_read", func(t *testing.T) {
				history, err := historySvc.GetHistory(ctx, row.type_, row.key)
				if err != nil || len(history) != 0 {
					t.Fatalf("%s history read for %s = %#v, %v; want empty baseline", baselineName, row.type_, history, err)
				}
			})

			t.Run("claim_heartbeat_release", func(t *testing.T) {
				session := "baseline-" + row.name
				claimed, err := claimSvc.Claim(ctx, services.ClaimInput{EntityType: string(row.type_), EntityKey: row.key, ClaimedBy: "contract", SessionID: session})
				if err != nil || claimed.SessionID != session {
					t.Fatalf("%s claim for %s = %#v, %v", baselineName, row.type_, claimed, err)
				}
				if err := claimSvc.Heartbeat(ctx, string(row.type_), row.key, session, nil, "baseline"); err != nil {
					t.Fatalf("%s heartbeat for %s: %v", baselineName, row.type_, err)
				}
				released, err := claimSvc.Release(ctx, string(row.type_), row.key, session, "released", false)
				if err != nil || !released {
					t.Fatalf("%s release for %s = %v, %v", baselineName, row.type_, released, err)
				}
			})

			t.Run("keyed_next", func(t *testing.T) {
				output, err := runSharkTC012(dbPath, "next", row.key)
				if row.nextError != "" {
					if err == nil || !strings.Contains(output, row.nextError) {
						t.Fatalf("%s keyed next %s = %q, %v; want %q", baselineName, row.type_, output, err, row.nextError)
					}
					return
				}
				if err != nil {
					t.Fatalf("%s keyed next %s failed: %v\n%s", baselineName, row.type_, err, output)
				}
				var response commands.NextResponse
				if err := json.Unmarshal([]byte(output), &response); err != nil {
					t.Fatalf("%s decode keyed next %s: %v\n%s", baselineName, row.type_, err, output)
				}
				if response.EntityKey != row.key || response.EntityType != string(row.type_) || response.Action != row.nextAction {
					t.Fatalf("%s keyed next %s = %#v, want key=%q type=%q action=%q", baselineName, row.type_, response, row.key, row.type_, row.nextAction)
				}
				// B059 F-1 regression: sprint's spawn_agent dispatch must have a
				// real placeholder generator wired up. A missing "sprint" case
				// in buildPlaceholderGenerator (run.go) silently degrades to an
				// empty vars map, and text/template renders missing keys as the
				// literal string "<no value>" instead of erroring — so the
				// dispatch would "succeed" while producing a prompt no agent
				// could act on (e.g. `Plan sprint <no value>: "<no value>"`).
				// Scoped to sprint only: other baseline rows can legitimately
				// render a missing optional field (e.g. file_path) as
				// "<no value>" for fixture entities with no on-disk doc, which
				// is unrelated to this bug.
				if row.name == "sprint" && response.Action == "spawn_agent" && strings.Contains(response.Prompt, "<no value>") {
					t.Fatalf("%s keyed next %s rendered prompt contains unresolved template placeholders (<no value>): %q", baselineName, row.type_, response.Prompt)
				}
				if row.name == "sprint" && response.Action == "spawn_agent" && (!strings.Contains(response.Prompt, "S001")) {
					t.Fatalf("%s keyed next %s rendered prompt does not contain the sprint key S001: %q", baselineName, row.type_, response.Prompt)
				}
				if row.name == "sprint" && response.Action == "spawn_agent" && !strings.Contains(response.Prompt, "make\nthe sprint active") {
					t.Fatalf("%s keyed next %s rendered prompt does not describe the configured planning-to-active route: %q", baselineName, row.type_, response.Prompt)
				}
			})
		})
	}
}

func projectRootTC011(t *testing.T) string {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", ".."))
}

func runSharkTC011Failure(t *testing.T, dbPath string, args ...string) string {
	t.Helper()
	command := exec.Command("go", append([]string{"run", "-buildvcs=false", "./cmd/shark", "--db", dbPath}, args...)...)
	command.Dir = projectRootTC011(t)
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("shark %s succeeded, want rejection\n%s", strings.Join(args, " "), output)
	}
	return string(output)
}

func databaseSnapshotTC011(t *testing.T, sqlDB *sql.DB) string {
	t.Helper()
	rows, err := sqlDB.Query("SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		t.Fatalf("list database tables: %v", err)
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			t.Fatalf("scan table: %v", err)
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate tables: %v", err)
	}

	var snapshot strings.Builder
	for _, table := range tables {
		dataRows, queryErr := sqlDB.Query(fmt.Sprintf("SELECT * FROM %q ORDER BY rowid", table))
		if queryErr != nil && strings.Contains(queryErr.Error(), "no such column: rowid") {
			dataRows, queryErr = sqlDB.Query(fmt.Sprintf("SELECT * FROM %q", table))
		}
		if queryErr != nil {
			t.Fatalf("snapshot %s: %v", table, queryErr)
		}
		columns, columnErr := dataRows.Columns()
		if columnErr != nil {
			t.Fatalf("columns for %s: %v", table, columnErr)
		}
		snapshot.WriteString(table + ":" + strings.Join(columns, ",") + "\n")
		for dataRows.Next() {
			values := make([]any, len(columns))
			pointers := make([]any, len(values))
			for i := range values {
				pointers[i] = &values[i]
			}
			if err := dataRows.Scan(pointers...); err != nil {
				t.Fatalf("scan %s: %v", table, err)
			}
			snapshot.WriteString(fmt.Sprintf("%#v\n", values))
		}
		if err := dataRows.Err(); err != nil {
			t.Fatalf("iterate %s: %v", table, err)
		}
		dataRows.Close()
	}
	sum := sha256.Sum256([]byte(snapshot.String()))
	return fmt.Sprintf("%x", sum)
}

func registrationBaselineFixtureTC012(t *testing.T) (string, *sql.DB, *services.EntityRegistry) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "baseline.db")
	sqlDB, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB() error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	repoDB := repository.NewDB(sqlDB)
	seedRegistrationBaselineTC012(t, repoDB)

	// These are the actual production adapters, not a copied supported-type
	// switch. TC-012 fails if any pre-F01 type stops resolving through generic
	// registry wiring while Question is added beside it.
	registry := services.NewEntityRegistry()
	registry.Register(models.EntityTypeEpic, services.NewEpicRepositoryAdapter(repository.NewEpicRepository(repoDB)))
	registry.Register(models.EntityTypeFeature, services.NewFeatureRepositoryAdapter(repository.NewFeatureRepository(repoDB)))
	registry.Register(models.EntityTypeTask, services.NewTaskRepositoryAdapter(repository.NewTaskRepository(repoDB)))
	registry.Register(models.EntityTypeBug, services.NewBugRepositoryAdapter(repository.NewBugRepository(repoDB)))
	registry.Register(models.EntityTypeChange, services.NewChangeCardRepositoryAdapter(repository.NewChangeCardRepository(repoDB)))
	registry.Register(models.EntityTypeTechDebt, services.NewTechDebtRepositoryAdapter(repository.NewTechDebtRepository(repoDB)))
	registry.Register(models.EntityTypeSprint, services.NewSprintRepositoryAdapter(repository.NewSprintRepository(repoDB)))
	registry.Register(models.EntityTypeIdea, services.NewIdeaRepositoryAdapter(repository.NewIdeaRepository(repoDB)))
	return dbPath, sqlDB, registry
}

func seedRegistrationBaselineTC012(t *testing.T, repoDB *repository.DB) {
	t.Helper()
	ctx := context.Background()
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E01", Title: "Registration baseline epic"}, Status: models.EpicStatusDraft, Priority: models.PriorityMedium}
	if err := repository.NewEpicRepository(repoDB).Create(ctx, epic); err != nil {
		t.Fatalf("seed baseline epic: %v", err)
	}
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E01-F01", Title: "Registration baseline feature"}, EpicID: epic.ID, Status: models.FeatureStatusDraft}
	if err := repository.NewFeatureRepository(repoDB).Create(ctx, feature); err != nil {
		t.Fatalf("seed baseline feature: %v", err)
	}
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E01-F01-001", Title: "Registration baseline task"}, FeatureID: feature.ID, Status: "development", Priority: 5}
	if err := repository.NewTaskRepository(repoDB).Create(ctx, task); err != nil {
		t.Fatalf("seed baseline task: %v", err)
	}
	bug := &models.Bug{BaseEntity: models.BaseEntity{Key: "B001", Title: "Registration baseline bug"}, Status: "reported", Severity: models.BugSeverityMedium}
	if err := repository.NewBugRepository(repoDB).Create(ctx, bug); err != nil {
		t.Fatalf("seed baseline bug: %v", err)
	}
	change := &models.ChangeCard{BaseEntity: models.BaseEntity{Key: "CC-001", Title: "Registration baseline change"}, Status: "proposed", Priority: 5}
	if err := repository.NewChangeCardRepository(repoDB).Create(ctx, change); err != nil {
		t.Fatalf("seed baseline change: %v", err)
	}
	techDebt := &models.TechDebt{BaseEntity: models.BaseEntity{Key: "TD-001", Title: "Registration baseline debt"}, Status: "identified", Category: models.TechDebtCategoryCodeQuality, Severity: models.TechDebtSeverityMedium}
	if err := repository.NewTechDebtRepository(repoDB).Create(ctx, techDebt); err != nil {
		t.Fatalf("seed baseline tech debt: %v", err)
	}
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	sprint := &models.Sprint{Key: "S001", Name: "Registration baseline sprint", StartDate: start, EndDate: start.AddDate(0, 0, 14), Status: "planning"}
	if err := repository.NewSprintRepository(repoDB).Create(ctx, sprint); err != nil {
		t.Fatalf("seed baseline sprint: %v", err)
	}
	idea := &models.Idea{Key: "I-2026-01-01-01", Title: "Registration baseline idea", CreatedDate: start, Status: models.IdeaStatusNew}
	if err := repository.NewIdeaRepository(repoDB).Create(ctx, idea); err != nil {
		t.Fatalf("seed baseline idea: %v", err)
	}
}

func runSharkTC012(dbPath string, args ...string) (string, error) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("runtime.Caller failed")
	}
	command := exec.Command("go", append([]string{"run", "-buildvcs=false", "./cmd/shark", "--db", dbPath}, args...)...)
	command.Dir = filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", ".."))
	output, err := command.CombinedOutput()
	return string(output), err
}

func runSharkTC013(t *testing.T, dbPath string, args ...string) string {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	projectRoot := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", ".."))
	commandArgs := append([]string{"run", "-buildvcs=false", "./cmd/shark", "--db", dbPath}, args...)
	command := exec.Command("go", commandArgs...)
	command.Dir = projectRoot
	return runSharkCaptureSeparate(t, command, "shark", args)
}
