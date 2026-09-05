package commands

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// Mock seams (test-plan.md Caller-Path Contract for TC-007..TC-009): mock
// only the entity-type repository and NoteEntityNoteRepository, then wire a
// REAL services.NoteService and REAL services.ImpactService over them via
// impactSvcOverride. Never mock ImpactService/NoteService directly — mirrors
// impact_service_test.go's newImpactTestHarness / note_service_test.go's
// mockNoteEntityRepo+mockNoteEntityNoteRepo two-seam pattern.
// ---------------------------------------------------------------------------

// mockImpactEntityRepo is a minimal services.EntityRepository that counts
// GetByKey calls so tests can assert the command genuinely drove entity-key
// resolution rather than bypassing it.
type mockImpactEntityRepo struct {
	getByKeyCalls int
	lastKey       string
	entity        models.Entity
	err           error
}

func (r *mockImpactEntityRepo) GetByKey(_ context.Context, key string) (models.Entity, error) {
	r.getByKeyCalls++
	r.lastKey = key
	if r.err != nil {
		return nil, r.err
	}
	return r.entity, nil
}
func (r *mockImpactEntityRepo) GetByID(_ context.Context, _ int64) (models.Entity, error) {
	return nil, nil
}
func (r *mockImpactEntityRepo) UpdateStatus(_ context.Context, _ int64, _ string) error { return nil }
func (r *mockImpactEntityRepo) UpdateStatusIfCurrent(_ context.Context, _ int64, _ string, _ string) (bool, error) {
	return true, nil
}
func (r *mockImpactEntityRepo) Update(_ context.Context, _ models.Entity) error { return nil }
func (r *mockImpactEntityRepo) GetContextData(_ context.Context, _ int64) (*string, error) {
	return nil, nil
}
func (r *mockImpactEntityRepo) UpdateContextData(_ context.Context, _ int64, _ *string) error {
	return nil
}

// mockImpactNoteRepo is a minimal services.NoteEntityNoteRepository that
// counts Create calls and captures the last note written.
type mockImpactNoteRepo struct {
	createCalls int
	lastNote    *models.EntityNote
	err         error
}

func (r *mockImpactNoteRepo) Create(_ context.Context, note *models.EntityNote) error {
	r.createCalls++
	if r.err != nil {
		return r.err
	}
	note.ID = 1
	r.lastNote = note
	return nil
}
func (r *mockImpactNoteRepo) GetByEntity(_ context.Context, _ models.EntityType, _ int64) ([]*models.EntityNote, error) {
	return nil, nil
}
func (r *mockImpactNoteRepo) GetByEntityAndType(_ context.Context, _ models.EntityType, _ int64, _ []string) ([]*models.EntityNote, error) {
	return nil, nil
}
func (r *mockImpactNoteRepo) Search(_ context.Context, _ string, _ []string, _ *models.EntityType, _ string, _ string) ([]*models.EntityNote, error) {
	return nil, nil
}
func (r *mockImpactNoteRepo) SearchWithTimePeriod(_ context.Context, _ string, _ []string, _ string, _ string, _ string, _ string) ([]*models.EntityNote, error) {
	return nil, nil
}

// wireImpactSvcOverride builds a REAL NoteService + REAL ImpactService over
// the given mocked repositories and installs it as impactSvcOverride for the
// duration of the test.
func wireImpactSvcOverride(t *testing.T, entityRepo *mockImpactEntityRepo, noteRepo *mockImpactNoteRepo) {
	t.Helper()
	registry := services.NewEntityRegistry()
	registry.Register(models.EntityTypeFeature, entityRepo)

	noteSvc, err := services.NewNoteService(noteRepo, registry)
	if err != nil {
		t.Fatalf("NewNoteService() unexpected error: %v", err)
	}
	impactSvc, err := services.NewImpactService(noteSvc)
	if err != nil {
		t.Fatalf("NewImpactService() unexpected error: %v", err)
	}

	origOverride := impactSvcOverride
	impactSvcOverride = impactSvc
	t.Cleanup(func() { impactSvcOverride = origOverride })
}

// testImpactCmd returns a minimal cobra.Command with a non-nil context,
// mirroring newChangeCmdWithCtx() in change_test.go.
func testImpactCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(context.Background())
	return cmd
}

// captureStdout is defined once package-wide in sharkdata_cmd_test.go; reused
// here to capture the human-readable / --json output of runImpactRecord.

const validImpactJSON = `{"source_kind":"tech_debt","source_key":"TD-042","affected_artifacts":["spec.md"]}`

// TestImpactCmd_RegisteredOnRoot verifies `shark impact record` is reachable
// as a subcommand of the registered `impact` command group, per spec.md's
// `shark impact record <entity-key> <content-or-@file> [--json]` shape.
func TestImpactCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range impactCmd.Commands() {
		if sub.Name() == "record" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("`record` is not a subcommand of impactCmd — `shark impact record` is not wired")
	}

	rootHasImpact := false
	for _, sub := range cli.RootCmd.Commands() {
		if sub.Name() == "impact" {
			rootHasImpact = true
			break
		}
	}
	if !rootHasImpact {
		t.Fatal("`impact` is not registered on the root command")
	}
}

// TestRunImpactRecord_TC007_ValidInlineContent is test-plan.md TC-007: a
// valid inline-JSON content on an existing key writes exactly one reference
// note. Per the Caller-Path Contract, this asserts GetByKey was called with
// the exact input key and Create was called exactly once with
// note_type=reference and content equal to the input bytes — a buggy impl
// that swallows the key argument and always writes to a hardcoded key would
// still pass a test that only checks the overall error return.
func TestRunImpactRecord_TC007_ValidInlineContent(t *testing.T) {
	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = true

	entityRepo := &mockImpactEntityRepo{
		entity: &models.Feature{BaseEntity: models.BaseEntity{ID: 9, Key: "E34-F07", Title: "Test Feature"}},
	}
	noteRepo := &mockImpactNoteRepo{}
	wireImpactSvcOverride(t, entityRepo, noteRepo)

	cmd := testImpactCmd()
	var runErr error
	out := captureStdout(t, func() {
		runErr = runImpactRecord(cmd, []string{"E34-F07", validImpactJSON})
	})
	if runErr != nil {
		t.Fatalf("runImpactRecord() unexpected error: %v", runErr)
	}

	if entityRepo.getByKeyCalls != 1 {
		t.Errorf("expected exactly 1 GetByKey call, got %d", entityRepo.getByKeyCalls)
	}
	if entityRepo.lastKey != "E34-F07" {
		t.Errorf("expected GetByKey called with the exact input key, got %q", entityRepo.lastKey)
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
	if noteRepo.lastNote.Content != validImpactJSON {
		t.Errorf("expected note content to equal input content exactly, got %q", noteRepo.lastNote.Content)
	}
	if noteRepo.lastNote.EntityID != 9 {
		t.Errorf("expected note written against resolved entity ID 9, got %d", noteRepo.lastNote.EntityID)
	}

	// --json output echoes the created note (AC-4 Test Matrix, TC-007 row).
	var echoed models.EntityNote
	if err := json.Unmarshal([]byte(out), &echoed); err != nil {
		t.Fatalf("expected --json output to be the created note, got unparseable output %q: %v", out, err)
	}
	if echoed.Content != validImpactJSON {
		t.Errorf("expected --json output to echo the note content, got %q", echoed.Content)
	}
}

// TestRunImpactRecord_TC007_RealRepositoryPair additionally runs against a
// real (in-memory sqlite) repository pair rather than mocks, per
// test-plan.md's Observability Design section: a pure-mock assertion is not
// runtime evidence that exactly one note is genuinely persisted and
// queryable afterward.
func TestRunImpactRecord_TC007_RealRepositoryPair(t *testing.T) {
	sqlDB, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatalf("db.InitDB(:memory:) failed: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	repoDB := repository.NewDB(sqlDB)

	ctx := context.Background()

	epicRepo := repository.NewEpicRepository(repoDB)
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E34", Title: "Test Epic"}, Status: "active", Priority: "high"}
	if err := epicRepo.Create(ctx, epic); err != nil {
		t.Fatalf("failed to create epic fixture: %v", err)
	}

	featureRepo := repository.NewFeatureRepository(repoDB)
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E34-F07", Title: "Test Feature"}, EpicID: epic.ID, Status: "active"}
	if err := featureRepo.Create(ctx, feature); err != nil {
		t.Fatalf("failed to create feature fixture: %v", err)
	}

	registry := services.NewEntityRegistry()
	registry.Register(models.EntityTypeFeature, services.NewFeatureRepositoryAdapter(featureRepo))

	noteRepo := repository.NewEntityNoteRepository(repoDB)
	noteSvc, err := services.NewNoteService(noteRepo, registry)
	if err != nil {
		t.Fatalf("NewNoteService() unexpected error: %v", err)
	}
	impactSvc, err := services.NewImpactService(noteSvc)
	if err != nil {
		t.Fatalf("NewImpactService() unexpected error: %v", err)
	}

	origOverride := impactSvcOverride
	impactSvcOverride = impactSvc
	t.Cleanup(func() { impactSvcOverride = origOverride })

	cmd := testImpactCmd()
	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = false
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	_ = captureStdout(t, func() {
		if err := runImpactRecord(cmd, []string{"E34-F07", validImpactJSON}); err != nil {
			t.Fatalf("runImpactRecord() unexpected error: %v", err)
		}
	})

	// Confirm the note is genuinely queryable afterward, not merely
	// mock-recorded: query the real repository pair directly.
	notes, err := noteSvc.ListNotes(ctx, models.EntityTypeFeature, "E34-F07", nil)
	if err != nil {
		t.Fatalf("ListNotes() unexpected error: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected exactly 1 persisted note, got %d", len(notes))
	}
	if notes[0].Content != validImpactJSON {
		t.Errorf("expected persisted note content to equal input content, got %q", notes[0].Content)
	}
	if notes[0].NoteType != models.NoteTypeReference {
		t.Errorf("expected persisted note_type=reference, got %q", notes[0].NoteType)
	}
}

// TestRunImpactRecord_TC008_AtFilePrefix is test-plan.md TC-008: the @file
// content form reads real file bytes. Per the Notes for Agent guidance, this
// uses a real temp file (never a mocked os.ReadFile) so the @-prefix parsing
// is genuinely exercised — a buggy impl that forgets to strip the @ prefix
// would pass the literal string "@path/to/file.json" as note content instead
// of the file's bytes, which only a real-file test catches.
func TestRunImpactRecord_TC008_AtFilePrefix(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "impact.json")
	if err := os.WriteFile(filePath, []byte(validImpactJSON), 0o600); err != nil {
		t.Fatalf("failed to write temp impact file: %v", err)
	}

	entityRepo := &mockImpactEntityRepo{
		entity: &models.Feature{BaseEntity: models.BaseEntity{ID: 9, Key: "E34-F07", Title: "Test Feature"}},
	}
	noteRepo := &mockImpactNoteRepo{}
	wireImpactSvcOverride(t, entityRepo, noteRepo)

	cmd := testImpactCmd()
	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = false
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	_ = captureStdout(t, func() {
		if err := runImpactRecord(cmd, []string{"E34-F07", "@" + filePath}); err != nil {
			t.Fatalf("runImpactRecord() unexpected error: %v", err)
		}
	})

	if noteRepo.createCalls != 1 {
		t.Fatalf("expected exactly 1 Create call, got %d", noteRepo.createCalls)
	}
	if noteRepo.lastNote.Content != validImpactJSON {
		t.Errorf("expected note content to equal the file's bytes, got %q", noteRepo.lastNote.Content)
	}
	if strings.HasPrefix(noteRepo.lastNote.Content, "@") {
		t.Errorf("note content must not be the literal @path string, got %q", noteRepo.lastNote.Content)
	}
}

// TestRunImpactRecord_TC008_MissingFile is TC-008's edge case: a missing or
// unreadable @file path must fail clearly, before any repository call — not
// a silent empty-content write.
func TestRunImpactRecord_TC008_MissingFile(t *testing.T) {
	entityRepo := &mockImpactEntityRepo{
		entity: &models.Feature{BaseEntity: models.BaseEntity{ID: 9, Key: "E34-F07", Title: "Test Feature"}},
	}
	noteRepo := &mockImpactNoteRepo{}
	wireImpactSvcOverride(t, entityRepo, noteRepo)

	cmd := testImpactCmd()
	missingPath := filepath.Join(t.TempDir(), "does-not-exist.json")

	err := runImpactRecord(cmd, []string{"E34-F07", "@" + missingPath})
	if err == nil {
		t.Fatal("expected an error for a missing @file path, got nil")
	}

	if entityRepo.getByKeyCalls != 0 {
		t.Errorf("expected zero entity lookups when the @file read fails, got %d", entityRepo.getByKeyCalls)
	}
	if noteRepo.createCalls != 0 {
		t.Errorf("expected zero note-repository calls when the @file read fails, got %d", noteRepo.createCalls)
	}
}

// TestRunImpactRecord_TC009_TargetKeyNotFound is test-plan.md TC-009: a
// target key that does not exist must exit non-zero, with the not-found
// signal originating from the mocked entity-type repository's GetByKey
// (never a canned not-found short-circuit), and the note repository must
// never be called.
func TestRunImpactRecord_TC009_TargetKeyNotFound(t *testing.T) {
	entityRepo := &mockImpactEntityRepo{
		err: errors.New("feature not found: E34-F99"),
	}
	noteRepo := &mockImpactNoteRepo{}
	wireImpactSvcOverride(t, entityRepo, noteRepo)

	cmd := testImpactCmd()
	content := `{"source_kind":"question","source_key":"Q-1","affected_artifacts":["x"]}`

	err := runImpactRecord(cmd, []string{"E34-F99", content})
	if err == nil {
		t.Fatal("expected an error for a nonexistent target key, got nil")
	}

	if entityRepo.getByKeyCalls != 1 {
		t.Errorf("expected the not-found signal to come from exactly 1 GetByKey call, got %d", entityRepo.getByKeyCalls)
	}
	if noteRepo.createCalls != 0 {
		t.Errorf("expected zero note-repository calls for a nonexistent target key, got %d", noteRepo.createCalls)
	}
}
