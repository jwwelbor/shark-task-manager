package commands

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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
//
// REWORK (UAT round 1, HIGH-1): the production contract is now the
// architecture.md-declared flag form —
//
//	shark impact record <entity-key> --source-kind=<k> --source-key=<k> \
//	  --source-pointer=<p> --impact-file=<path>
//
// — not the old two-positional-args `<content-or-@file>` form. Every test
// below drives `runImpactRecord` through a freshly-built cobra.Command with
// the same flags the production `impactRecordCmd` registers, via
// `cmd.SetArgs(...)` + `cmd.Execute()`, mirroring change_test.go's
// buildChangeCreateCmdForTagTest pattern — this exercises real cobra flag
// parsing (including MarkFlagRequired enforcement), not just a direct Go
// function call with pre-populated package vars.
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

// buildImpactRecordCmdForTest returns a fresh `shark impact record` command
// with the same flags (bound to the same package-level vars) as the
// production impactRecordCmd, per this repo's isolated-cobra-command test
// convention (change_test.go's buildChangeCreateCmdForTagTest). Building a
// FRESH command per test — rather than reusing the shared, root-wired
// impactRecordCmd — avoids cross-test flag-state leakage while still
// exercising the identical production argument shape.
func buildImpactRecordCmdForTest() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "record <entity-key>",
		Args: cobra.ExactArgs(1),
		RunE: runImpactRecord,
	}
	cmd.Flags().StringVar(&impactSourceKind, "source-kind", "", "source kind")
	cmd.Flags().StringVar(&impactSourceKey, "source-key", "", "source key")
	cmd.Flags().StringVar(&impactSourcePointer, "source-pointer", "", "source pointer")
	cmd.Flags().StringVar(&impactFile, "impact-file", "", "impact file")
	_ = cmd.MarkFlagRequired("source-kind")
	_ = cmd.MarkFlagRequired("source-key")
	_ = cmd.MarkFlagRequired("source-pointer")
	_ = cmd.MarkFlagRequired("impact-file")
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	return cmd
}

// writeImpactFile writes content to a new temp file and returns its path.
func writeImpactFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "impact.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write temp impact file: %v", err)
	}
	return path
}

// captureStdout is defined once package-wide in sharkdata_cmd_test.go; reused
// here to capture the human-readable / --json output of runImpactRecord.

// boundedImpactJSON is the impact-file body: the "bounded" I-04 content the
// caller is responsible for (affected_artifacts etc). It deliberately omits
// source_kind/source_key/source_pointer — those come from the CLI flags and
// must be merged in by the command, proving the flags are the authority, not
// mere unused bookkeeping.
const boundedImpactJSON = `{"affected_artifacts":["spec.md"]}`

// TestImpactCmd_RegisteredOnRoot verifies `shark impact record` is reachable
// as a subcommand of the registered `impact` command group, per
// architecture.md's `shark impact record <entity-key> --source-kind=...
// --source-key=... --source-pointer=... --impact-file=...` shape.
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

// TestImpactRecordCmd_FlagsRegistered verifies the production impactRecordCmd
// registers exactly the four architecture.md-declared flags and no
// leftover positional-content argument slot (Args is ExactArgs(1), the
// entity key only).
func TestImpactRecordCmd_FlagsRegistered(t *testing.T) {
	for _, name := range []string{"source-kind", "source-key", "source-pointer", "impact-file"} {
		if impactRecordCmd.Flags().Lookup(name) == nil {
			t.Errorf("--%s flag not registered on impactRecordCmd", name)
		}
	}
}

// TestRunImpactRecord_TC007_ValidImpactFile is test-plan.md TC-007 under the
// architecture.md flag contract: a valid impact-file plus all required
// flags on an existing key writes exactly one reference note, with
// source_kind/source_key/source_pointer merged in from the flags.
func TestRunImpactRecord_TC007_ValidImpactFile(t *testing.T) {
	origJSON := cli.GlobalConfig.JSON
	defer func() { cli.GlobalConfig.JSON = origJSON }()
	cli.GlobalConfig.JSON = true

	entityRepo := &mockImpactEntityRepo{
		entity: &models.Feature{BaseEntity: models.BaseEntity{ID: 9, Key: "E34-F07", Title: "Test Feature"}},
	}
	noteRepo := &mockImpactNoteRepo{}
	wireImpactSvcOverride(t, entityRepo, noteRepo)

	filePath := writeImpactFile(t, boundedImpactJSON)
	cmd := buildImpactRecordCmdForTest()
	cmd.SetArgs([]string{
		"E34-F07",
		"--source-kind=tech_debt",
		"--source-key=TD-042",
		"--source-pointer=docs/td/TD-042.md",
		"--impact-file=" + filePath,
	})

	var out string
	var execErr error
	out = captureStdout(t, func() {
		execErr = cmd.Execute()
	})
	if execErr != nil {
		t.Fatalf("Execute() unexpected error: %v", execErr)
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
	if noteRepo.lastNote.EntityID != 9 {
		t.Errorf("expected note written against resolved entity ID 9, got %d", noteRepo.lastNote.EntityID)
	}

	var persisted map[string]interface{}
	if err := json.Unmarshal([]byte(noteRepo.lastNote.Content), &persisted); err != nil {
		t.Fatalf("expected note content to be valid JSON, got %q: %v", noteRepo.lastNote.Content, err)
	}
	if persisted["source_kind"] != "tech_debt" {
		t.Errorf("expected --source-kind to be merged into persisted content, got %v", persisted["source_kind"])
	}
	if persisted["source_key"] != "TD-042" {
		t.Errorf("expected --source-key to be merged into persisted content, got %v", persisted["source_key"])
	}
	if persisted["source_pointer"] != "docs/td/TD-042.md" {
		t.Errorf("expected --source-pointer to be merged into persisted content, got %v", persisted["source_pointer"])
	}
	artifacts, ok := persisted["affected_artifacts"].([]interface{})
	if !ok || len(artifacts) != 1 || artifacts[0] != "spec.md" {
		t.Errorf("expected affected_artifacts from the impact-file to survive the merge, got %v", persisted["affected_artifacts"])
	}

	// --json output echoes the created note (AC-4 Test Matrix, TC-007 row).
	var echoed models.EntityNote
	if err := json.Unmarshal([]byte(out), &echoed); err != nil {
		t.Fatalf("expected --json output to be the created note, got unparseable output %q: %v", out, err)
	}
	if echoed.Content != noteRepo.lastNote.Content {
		t.Errorf("expected --json output to echo the persisted note content, got %q", echoed.Content)
	}
}

// TestRunImpactRecord_FlagOverridesFileSourceFields proves the CLI flags are
// the authority: an impact-file that already carries different
// source_kind/source_key/source_pointer values is overridden by the flags,
// per architecture.md's parent-owned ADR-adoption boundary.
func TestRunImpactRecord_FlagOverridesFileSourceFields(t *testing.T) {
	entityRepo := &mockImpactEntityRepo{
		entity: &models.Feature{BaseEntity: models.BaseEntity{ID: 9, Key: "E34-F07", Title: "Test Feature"}},
	}
	noteRepo := &mockImpactNoteRepo{}
	wireImpactSvcOverride(t, entityRepo, noteRepo)

	fileContent := `{"source_kind":"question","source_key":"Q-1","source_pointer":"stale.md","affected_artifacts":["spec.md"]}`
	filePath := writeImpactFile(t, fileContent)
	cmd := buildImpactRecordCmdForTest()
	cmd.SetArgs([]string{
		"E34-F07",
		"--source-kind=adr",
		"--source-key=ADR-014",
		"--source-pointer=docs/adr/ADR-014.md",
		"--impact-file=" + filePath,
	})

	_ = captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute() unexpected error: %v", err)
		}
	})

	var persisted map[string]interface{}
	if err := json.Unmarshal([]byte(noteRepo.lastNote.Content), &persisted); err != nil {
		t.Fatalf("expected note content to be valid JSON: %v", err)
	}
	if persisted["source_kind"] != "adr" {
		t.Errorf("expected the --source-kind flag to override the file's value, got %v", persisted["source_kind"])
	}
	if persisted["source_key"] != "ADR-014" {
		t.Errorf("expected the --source-key flag to override the file's value, got %v", persisted["source_key"])
	}
	if persisted["source_pointer"] != "docs/adr/ADR-014.md" {
		t.Errorf("expected the --source-pointer flag to override the file's value, got %v", persisted["source_pointer"])
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

	filePath := writeImpactFile(t, boundedImpactJSON)
	cmd := buildImpactRecordCmdForTest()
	cmd.SetArgs([]string{
		"E34-F07",
		"--source-kind=tech_debt",
		"--source-key=TD-042",
		"--source-pointer=docs/td/TD-042.md",
		"--impact-file=" + filePath,
	})

	_ = captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute() unexpected error: %v", err)
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
	if notes[0].NoteType != models.NoteTypeReference {
		t.Errorf("expected persisted note_type=reference, got %q", notes[0].NoteType)
	}
	var persisted map[string]interface{}
	if err := json.Unmarshal([]byte(notes[0].Content), &persisted); err != nil {
		t.Fatalf("expected persisted content to be valid JSON: %v", err)
	}
	if persisted["source_key"] != "TD-042" {
		t.Errorf("expected persisted content to carry the merged source_key, got %v", persisted["source_key"])
	}
}

// TestRunImpactRecord_TC008_ImpactFileReadsRealBytes is test-plan.md TC-008
// under the flag contract: --impact-file reads real file bytes (not the
// literal path string), and the file's own affected_artifacts content
// survives the source-field merge.
func TestRunImpactRecord_TC008_ImpactFileReadsRealBytes(t *testing.T) {
	entityRepo := &mockImpactEntityRepo{
		entity: &models.Feature{BaseEntity: models.BaseEntity{ID: 9, Key: "E34-F07", Title: "Test Feature"}},
	}
	noteRepo := &mockImpactNoteRepo{}
	wireImpactSvcOverride(t, entityRepo, noteRepo)

	filePath := writeImpactFile(t, boundedImpactJSON)
	cmd := buildImpactRecordCmdForTest()
	cmd.SetArgs([]string{
		"E34-F07",
		"--source-kind=tech_debt",
		"--source-key=TD-042",
		"--source-pointer=docs/td/TD-042.md",
		"--impact-file=" + filePath,
	})

	_ = captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute() unexpected error: %v", err)
		}
	})

	if noteRepo.createCalls != 1 {
		t.Fatalf("expected exactly 1 Create call, got %d", noteRepo.createCalls)
	}
	if noteRepo.lastNote.Content == filePath {
		t.Errorf("note content must not be the literal --impact-file path, got %q", noteRepo.lastNote.Content)
	}
	var persisted map[string]interface{}
	if err := json.Unmarshal([]byte(noteRepo.lastNote.Content), &persisted); err != nil {
		t.Fatalf("expected note content to be the file's (merged) JSON bytes, got %q: %v", noteRepo.lastNote.Content, err)
	}
}

// TestRunImpactRecord_TC008_MissingImpactFile is TC-008's edge case: a
// missing or unreadable --impact-file path must fail clearly, before any
// repository call — not a silent empty-content write.
func TestRunImpactRecord_TC008_MissingImpactFile(t *testing.T) {
	entityRepo := &mockImpactEntityRepo{
		entity: &models.Feature{BaseEntity: models.BaseEntity{ID: 9, Key: "E34-F07", Title: "Test Feature"}},
	}
	noteRepo := &mockImpactNoteRepo{}
	wireImpactSvcOverride(t, entityRepo, noteRepo)

	missingPath := filepath.Join(t.TempDir(), "does-not-exist.json")
	cmd := buildImpactRecordCmdForTest()
	cmd.SetArgs([]string{
		"E34-F07",
		"--source-kind=tech_debt",
		"--source-key=TD-042",
		"--source-pointer=docs/td/TD-042.md",
		"--impact-file=" + missingPath,
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected an error for a missing --impact-file path, got nil")
	}

	if entityRepo.getByKeyCalls != 0 {
		t.Errorf("expected zero entity lookups when the impact-file read fails, got %d", entityRepo.getByKeyCalls)
	}
	if noteRepo.createCalls != 0 {
		t.Errorf("expected zero note-repository calls when the impact-file read fails, got %d", noteRepo.createCalls)
	}
}

// TestRunImpactRecord_MissingRequiredFlag proves each of the four
// architecture.md-declared flags is enforced as required by cobra itself
// (MarkFlagRequired), not merely documented — omitting any one must fail
// Execute() before RunE ever runs, with zero repository calls.
func TestRunImpactRecord_MissingRequiredFlag(t *testing.T) {
	filePath := writeImpactFile(t, boundedImpactJSON)

	cases := []struct {
		name string
		args []string
	}{
		{
			name: "missing --source-kind",
			args: []string{"E34-F07", "--source-key=TD-042", "--source-pointer=docs/td/TD-042.md", "--impact-file=" + filePath},
		},
		{
			name: "missing --source-key",
			args: []string{"E34-F07", "--source-kind=tech_debt", "--source-pointer=docs/td/TD-042.md", "--impact-file=" + filePath},
		},
		{
			name: "missing --source-pointer",
			args: []string{"E34-F07", "--source-kind=tech_debt", "--source-key=TD-042", "--impact-file=" + filePath},
		},
		{
			name: "missing --impact-file",
			args: []string{"E34-F07", "--source-kind=tech_debt", "--source-key=TD-042", "--source-pointer=docs/td/TD-042.md"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entityRepo := &mockImpactEntityRepo{
				entity: &models.Feature{BaseEntity: models.BaseEntity{ID: 9, Key: "E34-F07", Title: "Test Feature"}},
			}
			noteRepo := &mockImpactNoteRepo{}
			wireImpactSvcOverride(t, entityRepo, noteRepo)

			cmd := buildImpactRecordCmdForTest()
			cmd.SetArgs(tc.args)

			err := cmd.Execute()
			if err == nil {
				t.Fatalf("expected an error for %s, got nil", tc.name)
			}
			if entityRepo.getByKeyCalls != 0 || noteRepo.createCalls != 0 {
				t.Errorf("expected zero repository calls for %s, got GetByKey=%d Create=%d", tc.name, entityRepo.getByKeyCalls, noteRepo.createCalls)
			}
		})
	}
}

// TestRunImpactRecord_EmptyFlagValue covers test-plan.md TC-010/TC-011's
// CLI-level partition: cobra's MarkFlagRequired only rejects an *omitted*
// flag (Changed == false); `--source-kind=` (an explicitly empty value)
// passes cobra's required-flag check and reaches runImpactRecord's own
// strings.TrimSpace + empty-string guard (impact_cmd.go:105-116). That
// guard fires before os.ReadFile, before the source-field merge, and before
// ImpactService is ever consulted — so this is a CLI-layer rejection, never
// the merged-content shape-validation path service-level TC-010/TC-011
// (impact_service_test.go) cover. Without this test, all four `if x == ""`
// branches were live but unexercised code.
func TestRunImpactRecord_EmptyFlagValue(t *testing.T) {
	filePath := writeImpactFile(t, boundedImpactJSON)

	cases := []struct {
		name string
		args []string
	}{
		{
			name: "empty --source-kind",
			args: []string{"E34-F07", "--source-kind=", "--source-key=TD-042", "--source-pointer=docs/td/TD-042.md", "--impact-file=" + filePath},
		},
		{
			name: "whitespace-only --source-key",
			args: []string{"E34-F07", "--source-kind=tech_debt", "--source-key=   ", "--source-pointer=docs/td/TD-042.md", "--impact-file=" + filePath},
		},
		{
			name: "empty --source-pointer",
			args: []string{"E34-F07", "--source-kind=tech_debt", "--source-key=TD-042", "--source-pointer=", "--impact-file=" + filePath},
		},
		{
			name: "empty --impact-file",
			args: []string{"E34-F07", "--source-kind=tech_debt", "--source-key=TD-042", "--source-pointer=docs/td/TD-042.md", "--impact-file="},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entityRepo := &mockImpactEntityRepo{
				entity: &models.Feature{BaseEntity: models.BaseEntity{ID: 9, Key: "E34-F07", Title: "Test Feature"}},
			}
			noteRepo := &mockImpactNoteRepo{}
			wireImpactSvcOverride(t, entityRepo, noteRepo)

			cmd := buildImpactRecordCmdForTest()
			cmd.SetArgs(tc.args)

			err := cmd.Execute()
			if err == nil {
				t.Fatalf("expected an error for %s, got nil", tc.name)
			}
			if entityRepo.getByKeyCalls != 0 || noteRepo.createCalls != 0 {
				t.Errorf("expected zero repository calls for %s (CLI-layer rejection must precede any repo call), got GetByKey=%d Create=%d", tc.name, entityRepo.getByKeyCalls, noteRepo.createCalls)
			}
		})
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

	filePath := writeImpactFile(t, `{"source_kind":"question","source_key":"Q-1","affected_artifacts":["x"]}`)
	cmd := buildImpactRecordCmdForTest()
	cmd.SetArgs([]string{
		"E34-F99",
		"--source-kind=question",
		"--source-key=Q-1",
		"--source-pointer=docs/questions/Q-1.md",
		"--impact-file=" + filePath,
	})

	err := cmd.Execute()
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
