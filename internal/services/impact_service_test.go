package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// readImpactServiceSource returns the raw source of impact_service.go,
// located relative to this test file so the guard test does not depend on
// the working directory the test binary is invoked from.
func readImpactServiceSource(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed; cannot locate impact_service.go")
	}
	path := filepath.Join(filepath.Dir(thisFile), "impact_service.go")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	return string(data)
}

// countingImpactEntityRepo is a minimal EntityRepository that counts calls
// to GetByKey so tests can assert entity lookup never ran when validation
// short-circuits (test-plan.md TC-010..TC-013).
type countingImpactEntityRepo struct {
	getByKeyCalls int
	entity        models.Entity
	err           error
}

func (r *countingImpactEntityRepo) GetByKey(_ context.Context, _ string) (models.Entity, error) {
	r.getByKeyCalls++
	if r.err != nil {
		return nil, r.err
	}
	return r.entity, nil
}
func (r *countingImpactEntityRepo) GetByID(_ context.Context, _ int64) (models.Entity, error) {
	return nil, nil
}
func (r *countingImpactEntityRepo) UpdateStatus(_ context.Context, _ int64, _ string) error {
	return nil
}
func (r *countingImpactEntityRepo) UpdateStatusIfCurrent(_ context.Context, _ int64, _ string, _ string) (bool, error) {
	return true, nil
}
func (r *countingImpactEntityRepo) Update(_ context.Context, _ models.Entity) error { return nil }
func (r *countingImpactEntityRepo) GetContextData(_ context.Context, _ int64) (*string, error) {
	return nil, nil
}
func (r *countingImpactEntityRepo) UpdateContextData(_ context.Context, _ int64, _ *string) error {
	return nil
}

// countingImpactNoteRepo is a minimal NoteEntityNoteRepository that counts
// calls to Create so tests can assert no note is ever persisted when
// validation short-circuits (test-plan.md TC-010..TC-013).
type countingImpactNoteRepo struct {
	createCalls int
	lastNote    *models.EntityNote
	err         error
}

func (r *countingImpactNoteRepo) Create(_ context.Context, note *models.EntityNote) error {
	r.createCalls++
	if r.err != nil {
		return r.err
	}
	note.ID = 1
	r.lastNote = note
	return nil
}
func (r *countingImpactNoteRepo) GetByEntity(_ context.Context, _ models.EntityType, _ int64) ([]*models.EntityNote, error) {
	return nil, nil
}
func (r *countingImpactNoteRepo) GetByEntityAndType(_ context.Context, _ models.EntityType, _ int64, _ []string) ([]*models.EntityNote, error) {
	return nil, nil
}
func (r *countingImpactNoteRepo) Search(_ context.Context, _ string, _ []string, _ *models.EntityType, _ string, _ string) ([]*models.EntityNote, error) {
	return nil, nil
}
func (r *countingImpactNoteRepo) SearchWithTimePeriod(_ context.Context, _ string, _ []string, _ string, _ string, _ string, _ string) ([]*models.EntityNote, error) {
	return nil, nil
}

// newImpactTestHarness wires a real NoteService (not a mock of NoteService
// itself) over a mocked entity-type repository and a mocked
// NoteEntityNoteRepository, per test-plan.md's "Mock-seam correction": a
// test that mocks ImpactNoteRecorder directly cannot observe whether
// validation short-circuited before entity lookup, so TC-010..TC-013 must
// drive the real NoteService and assert on the underlying repository call
// counts instead.
func newImpactTestHarness(t *testing.T, entityRepo *countingImpactEntityRepo, noteRepo *countingImpactNoteRepo) *ImpactService {
	t.Helper()
	registry := NewEntityRegistry()
	registry.Register(models.EntityTypeFeature, entityRepo)

	noteSvc, err := NewNoteService(noteRepo, registry)
	if err != nil {
		t.Fatalf("NewNoteService() unexpected error: %v", err)
	}

	impactSvc, err := NewImpactService(noteSvc)
	if err != nil {
		t.Fatalf("NewImpactService() unexpected error: %v", err)
	}
	return impactSvc
}

// TestImpactService_RecordImpact_ValidContent_DelegatesToNoteCreation proves
// the note-delegation half of this task: a minimally-shaped I-04 payload is
// written through the existing note-creation path as a single `reference`
// note, not a new persistence mechanism.
func TestImpactService_RecordImpact_ValidContent_DelegatesToNoteCreation(t *testing.T) {
	entityRepo := &countingImpactEntityRepo{
		entity: &models.Feature{BaseEntity: models.BaseEntity{ID: 9, Key: "E34-F07-001", Title: "Test Feature"}},
	}
	noteRepo := &countingImpactNoteRepo{}
	svc := newImpactTestHarness(t, entityRepo, noteRepo)

	content := `{"source_kind":"tech_debt","source_key":"TD-042","affected_artifacts":["spec.md"]}`
	note, err := svc.RecordImpact(context.Background(), models.EntityTypeFeature, "E34-F07-001", content, "")
	if err != nil {
		t.Fatalf("RecordImpact() unexpected error: %v", err)
	}
	if note == nil {
		t.Fatal("RecordImpact() returned nil note on success")
	}
	if entityRepo.getByKeyCalls != 1 {
		t.Errorf("expected exactly 1 GetByKey call, got %d", entityRepo.getByKeyCalls)
	}
	if noteRepo.createCalls != 1 {
		t.Errorf("expected exactly 1 Create call, got %d", noteRepo.createCalls)
	}
	if noteRepo.lastNote == nil {
		t.Fatal("expected a note to be captured by the mocked repository")
	}
	if noteRepo.lastNote.NoteType != models.NoteTypeReference {
		t.Errorf("expected note_type=reference, got %q", noteRepo.lastNote.NoteType)
	}
	if noteRepo.lastNote.Content != content {
		t.Errorf("expected note content to equal input content exactly, got %q", noteRepo.lastNote.Content)
	}
	if noteRepo.lastNote.EntityID != 9 {
		t.Errorf("expected note to be written against resolved entity ID 9, got %d", noteRepo.lastNote.EntityID)
	}
}

// TestImpactService_RecordImpact_MissingSourceKind is test-plan.md TC-010:
// a buggy implementation that validates only source_key/affected_artifacts
// (skipping source_kind) would pass a test that only checks the overall
// error return, so this asserts the message specifically names
// source_kind and that neither mocked repository was ever invoked.
func TestImpactService_RecordImpact_MissingSourceKind(t *testing.T) {
	entityRepo := &countingImpactEntityRepo{
		entity: &models.Feature{BaseEntity: models.BaseEntity{ID: 9, Key: "E34-F07-001", Title: "Test Feature"}},
	}
	noteRepo := &countingImpactNoteRepo{}
	svc := newImpactTestHarness(t, entityRepo, noteRepo)

	content := `{"source_key":"Q-1","affected_artifacts":["x"]}`
	note, err := svc.RecordImpact(context.Background(), models.EntityTypeFeature, "E34-F07-001", content, "")

	if err == nil {
		t.Fatal("expected an error for content missing source_kind, got nil")
	}
	if note != nil {
		t.Errorf("expected nil note on validation failure, got %+v", note)
	}
	if !errors.Is(err, ErrImpactInvalidShape) {
		t.Errorf("expected error to wrap ErrImpactInvalidShape, got %v", err)
	}
	if !strings.Contains(err.Error(), "source_kind") {
		t.Errorf("expected error message to name source_kind specifically, got %q", err.Error())
	}
	if entityRepo.getByKeyCalls != 0 {
		t.Errorf("expected zero entity lookups before validation failure, got %d", entityRepo.getByKeyCalls)
	}
	if noteRepo.createCalls != 0 {
		t.Errorf("expected zero note-repository calls before validation failure, got %d", noteRepo.createCalls)
	}
}

// TestImpactService_RecordImpact_MissingSourceKey is test-plan.md TC-011:
// same shape as TC-010, catching a partial-validation bug that checks
// source_kind/affected_artifacts but not source_key.
func TestImpactService_RecordImpact_MissingSourceKey(t *testing.T) {
	entityRepo := &countingImpactEntityRepo{
		entity: &models.Feature{BaseEntity: models.BaseEntity{ID: 9, Key: "E34-F07-001", Title: "Test Feature"}},
	}
	noteRepo := &countingImpactNoteRepo{}
	svc := newImpactTestHarness(t, entityRepo, noteRepo)

	content := `{"source_kind":"question","affected_artifacts":["x"]}`
	note, err := svc.RecordImpact(context.Background(), models.EntityTypeFeature, "E34-F07-001", content, "")

	if err == nil {
		t.Fatal("expected an error for content missing source_key, got nil")
	}
	if note != nil {
		t.Errorf("expected nil note on validation failure, got %+v", note)
	}
	if !errors.Is(err, ErrImpactInvalidShape) {
		t.Errorf("expected error to wrap ErrImpactInvalidShape, got %v", err)
	}
	if !strings.Contains(err.Error(), "source_key") {
		t.Errorf("expected error message to name source_key specifically, got %q", err.Error())
	}
	if entityRepo.getByKeyCalls != 0 {
		t.Errorf("expected zero entity lookups before validation failure, got %d", entityRepo.getByKeyCalls)
	}
	if noteRepo.createCalls != 0 {
		t.Errorf("expected zero note-repository calls before validation failure, got %d", noteRepo.createCalls)
	}
}

// TestImpactService_RecordImpact_AffectedArtifactsEmpty is test-plan.md
// TC-012 (boundary case a): affected_artifacts present but an empty array
// must fail distinctly from the "field absent" cases above — a buggy impl
// that only checks presence, not non-emptiness, would pass TC-010/TC-011
// but fail this one.
func TestImpactService_RecordImpact_AffectedArtifactsEmpty(t *testing.T) {
	entityRepo := &countingImpactEntityRepo{
		entity: &models.Feature{BaseEntity: models.BaseEntity{ID: 9, Key: "E34-F07-001", Title: "Test Feature"}},
	}
	noteRepo := &countingImpactNoteRepo{}
	svc := newImpactTestHarness(t, entityRepo, noteRepo)

	content := `{"source_kind":"question","source_key":"Q-1","affected_artifacts":[]}`
	note, err := svc.RecordImpact(context.Background(), models.EntityTypeFeature, "E34-F07-001", content, "")

	if err == nil {
		t.Fatal("expected an error for empty affected_artifacts, got nil")
	}
	if note != nil {
		t.Errorf("expected nil note on validation failure, got %+v", note)
	}
	if !errors.Is(err, ErrImpactInvalidShape) {
		t.Errorf("expected error to wrap ErrImpactInvalidShape, got %v", err)
	}
	if !strings.Contains(err.Error(), "affected_artifacts") {
		t.Errorf("expected error message to name affected_artifacts specifically, got %q", err.Error())
	}
	if entityRepo.getByKeyCalls != 0 {
		t.Errorf("expected zero entity lookups before validation failure, got %d", entityRepo.getByKeyCalls)
	}
	if noteRepo.createCalls != 0 {
		t.Errorf("expected zero note-repository calls before validation failure, got %d", noteRepo.createCalls)
	}
}

// TestImpactService_RecordImpact_AffectedArtifactsWrongType is test-plan.md
// TC-012 (boundary case b): affected_artifacts present but with the wrong
// JSON type (a string instead of an array) must also fail — closing the
// gap a presence-and-emptiness-only check would leave open.
func TestImpactService_RecordImpact_AffectedArtifactsWrongType(t *testing.T) {
	entityRepo := &countingImpactEntityRepo{
		entity: &models.Feature{BaseEntity: models.BaseEntity{ID: 9, Key: "E34-F07-001", Title: "Test Feature"}},
	}
	noteRepo := &countingImpactNoteRepo{}
	svc := newImpactTestHarness(t, entityRepo, noteRepo)

	content := `{"source_kind":"question","source_key":"Q-1","affected_artifacts":"x"}`
	note, err := svc.RecordImpact(context.Background(), models.EntityTypeFeature, "E34-F07-001", content, "")

	if err == nil {
		t.Fatal("expected an error for wrong-typed affected_artifacts, got nil")
	}
	if note != nil {
		t.Errorf("expected nil note on validation failure, got %+v", note)
	}
	if !errors.Is(err, ErrImpactInvalidShape) {
		t.Errorf("expected error to wrap ErrImpactInvalidShape, got %v", err)
	}
	if !strings.Contains(err.Error(), "affected_artifacts") {
		t.Errorf("expected error message to name affected_artifacts specifically, got %q", err.Error())
	}
	if entityRepo.getByKeyCalls != 0 {
		t.Errorf("expected zero entity lookups before validation failure, got %d", entityRepo.getByKeyCalls)
	}
	if noteRepo.createCalls != 0 {
		t.Errorf("expected zero note-repository calls before validation failure, got %d", noteRepo.createCalls)
	}
}

// TestImpactService_RecordImpact_MalformedJSON is test-plan.md TC-013:
// malformed (non-JSON) content must fail with a JSON-parse-specific error
// distinct from a missing-field validation message, so a caller can tell
// "malformed JSON" apart from "well-formed but incomplete" input.
func TestImpactService_RecordImpact_MalformedJSON(t *testing.T) {
	entityRepo := &countingImpactEntityRepo{
		entity: &models.Feature{BaseEntity: models.BaseEntity{ID: 9, Key: "E34-F07-001", Title: "Test Feature"}},
	}
	noteRepo := &countingImpactNoteRepo{}
	svc := newImpactTestHarness(t, entityRepo, noteRepo)

	note, err := svc.RecordImpact(context.Background(), models.EntityTypeFeature, "E34-F07-001", "not-json-at-all", "")

	if err == nil {
		t.Fatal("expected an error for malformed JSON content, got nil")
	}
	if note != nil {
		t.Errorf("expected nil note on validation failure, got %+v", note)
	}
	if !errors.Is(err, ErrImpactMalformedContent) {
		t.Errorf("expected error to wrap ErrImpactMalformedContent, got %v", err)
	}
	if errors.Is(err, ErrImpactInvalidShape) {
		t.Errorf("malformed-JSON error must not also be classified as a missing-field validation error: %v", err)
	}
	for _, field := range []string{"source_kind", "source_key", "affected_artifacts"} {
		if strings.Contains(err.Error(), field) {
			t.Errorf("malformed-JSON error message must not name field %q — that would conflate it with the missing-field messages (TC-010/TC-011/TC-012): %q", field, err.Error())
		}
	}
	if entityRepo.getByKeyCalls != 0 {
		t.Errorf("expected zero entity lookups before validation failure, got %d", entityRepo.getByKeyCalls)
	}
	if noteRepo.createCalls != 0 {
		t.Errorf("expected zero note-repository calls before validation failure, got %d", noteRepo.createCalls)
	}
}

// TestImpactServiceNoNewPersistenceIntroduced is test-plan.md TC-014: a
// structural guard proving REQ-NF-001's "zero new Shark database columns,
// tables, or relationship types" claim for this task's file, mirroring
// E34-F06's TestDefectClassSweepNoGoPersistenceIntroduced pattern but
// scoped to impact_service.go (the only file this task adds) rather than a
// whole-tree scan, so it can't be tripped by an unrelated persistence
// change elsewhere in the repo.
func TestImpactServiceNoNewPersistenceIntroduced(t *testing.T) {
	src := readImpactServiceSource(t)

	forbidden := []string{
		"database/sql",
		"sql.Tx",
		"sql.DB",
		"repository.",
		"CREATE TABLE",
		"ALTER TABLE",
		"INSERT INTO",
	}
	for _, token := range forbidden {
		if strings.Contains(src, token) {
			t.Errorf("internal/services/impact_service.go must not contain %q — RecordImpact must delegate to the existing note-creation path only, introducing no new persistence surface (REQ-NF-001); found it in the file", token)
		}
	}

	if !strings.Contains(src, "AddNote") {
		t.Error("internal/services/impact_service.go must delegate to NoteService.AddNote (or AddNoteWithMetadata) — no delegation call found")
	}
}
