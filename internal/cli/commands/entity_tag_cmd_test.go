package commands

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// E28-F04 T-005 — entity_tag_cmd factory tests.
//
// AC coverage (per test-plan.md §1.3):
//   - AC-23: `shark bug tag add B001 voice` attach + idempotency.
//   - AC-24: `shark bug tag rm B001 voice` detach + idempotency.
//   - AC-25: `shark bug tag rm B001 does-not-exist` error path.
//   - AC-26: `shark idea tag add <id> voice` — factory is entity-agnostic.
//   - AC-26b: full entity-type × (add/rm) × (happy, unregistered) table.
// ---------------------------------------------------------------------------

// mockEntityTagService implements entityTagServiceIface with function
// fields. A thin call log records every method invocation in order so
// tests can assert AttachMany was called with the expected args and was
// NOT called after an error.
type mockEntityTagService struct {
	attachManyFn      func(ctx context.Context, entityType models.EntityType, entityID int64, names []string) error
	detachOneFn       func(ctx context.Context, entityType models.EntityType, entityID int64, name string) error
	listTagsFn        func(ctx context.Context) ([]*models.Tag, error)
	attachManyCalls   int
	detachOneCalls    int
	lastAttachNames   []string
	lastAttachEntType models.EntityType
	lastAttachEntID   int64
	lastDetachName    string
}

func (m *mockEntityTagService) AttachMany(ctx context.Context, entityType models.EntityType, entityID int64, names []string) error {
	m.attachManyCalls++
	m.lastAttachEntType = entityType
	m.lastAttachEntID = entityID
	m.lastAttachNames = append([]string(nil), names...)
	if m.attachManyFn != nil {
		return m.attachManyFn(ctx, entityType, entityID, names)
	}
	return nil
}

func (m *mockEntityTagService) DetachOne(ctx context.Context, entityType models.EntityType, entityID int64, name string) error {
	m.detachOneCalls++
	m.lastDetachName = name
	if m.detachOneFn != nil {
		return m.detachOneFn(ctx, entityType, entityID, name)
	}
	return nil
}

func (m *mockEntityTagService) ListTags(ctx context.Context) ([]*models.Tag, error) {
	if m.listTagsFn != nil {
		return m.listTagsFn(ctx)
	}
	return []*models.Tag{}, nil
}

// Satisfy the embedded tagServiceIface surface — these methods are not
// invoked by the entity_tag_cmd factory but are required at compile time.
func (m *mockEntityTagService) AddTag(ctx context.Context, name, pass string) (*models.Tag, error) {
	return nil, fmt.Errorf("AddTag not implemented in mockEntityTagService")
}
func (m *mockEntityTagService) RemoveTag(ctx context.Context, name string, force bool, pass string) error {
	return fmt.Errorf("RemoveTag not implemented in mockEntityTagService")
}
func (m *mockEntityTagService) RenameTag(ctx context.Context, oldName, newName, pass string) (*models.Tag, error) {
	return nil, fmt.Errorf("RenameTag not implemented in mockEntityTagService")
}

// Compile-time assertion: mockEntityTagService must satisfy entityTagServiceIface.
var _ entityTagServiceIface = (*mockEntityTagService)(nil)

// buildEntityTagFixture returns a fresh cobra root wiring the tag
// subcommand for the given entity type, plus the mock service and
// stderr buffer. The returned resolveKey always succeeds with id=42 so
// unit tests focus on the tag-service interactions, not on entity
// lookup plumbing.
func buildEntityTagFixture(entityType models.EntityType) (*cobra.Command, *mockEntityTagService, *bytes.Buffer, *bytes.Buffer) {
	svc := &mockEntityTagService{}
	root := &cobra.Command{Use: "shark"}
	root.PersistentFlags().Bool("json", false, "JSON output")

	parent := &cobra.Command{Use: string(entityType)}
	root.AddCommand(parent)

	resolveKey := func(ctx context.Context, key string) (int64, error) {
		return 42, nil
	}
	parent.AddCommand(makeEntityTagCmd(entityType, resolveKey, svc))

	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	root.SilenceErrors = true
	root.SilenceUsage = true
	return root, svc, &outBuf, &errBuf
}

// withPlainTextJSONGlobal resets cli.GlobalConfig.JSON to false for the
// duration of the test, restoring the previous value on cleanup. The
// entity_tag commands branch on this global inside the snippet helper.
func withPlainTextJSONGlobal(t *testing.T) {
	t.Helper()
	orig := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = false
	t.Cleanup(func() { cli.GlobalConfig.JSON = orig })
}

// ---------------------------------------------------------------------------
// AC-23: bug tag add — attach + idempotent rerun
// ---------------------------------------------------------------------------

func TestBugTagAdd_AttachesOnce(t *testing.T) {
	withPlainTextJSONGlobal(t)
	root, svc, _, _ := buildEntityTagFixture(models.EntityTypeBug)

	root.SetArgs([]string{"bug", "tag", "add", "B001", "voice"})
	if err := root.Execute(); err != nil {
		t.Fatalf("first Execute() error = %v", err)
	}
	if svc.attachManyCalls != 1 {
		t.Errorf("attachManyCalls after first run = %d, want 1", svc.attachManyCalls)
	}
	if svc.lastAttachEntType != models.EntityTypeBug {
		t.Errorf("attach entity type = %q, want %q", svc.lastAttachEntType, models.EntityTypeBug)
	}
	if svc.lastAttachEntID != 42 {
		t.Errorf("attach entity id = %d, want 42", svc.lastAttachEntID)
	}
	if len(svc.lastAttachNames) != 1 || svc.lastAttachNames[0] != "voice" {
		t.Errorf("attach names = %v, want [voice]", svc.lastAttachNames)
	}
}

func TestBugTagAdd_IdempotentRerun(t *testing.T) {
	// AC-23: re-running `tag add` with the same args is a no-op at the
	// repo layer (INSERT OR IGNORE). The CLI surface sees two attach
	// calls, both returning nil — exit 0 on both runs.
	withPlainTextJSONGlobal(t)
	root, svc, _, _ := buildEntityTagFixture(models.EntityTypeBug)

	for i := 0; i < 2; i++ {
		root.SetArgs([]string{"bug", "tag", "add", "B001", "voice"})
		if err := root.Execute(); err != nil {
			t.Fatalf("Execute() iteration %d error = %v", i, err)
		}
	}
	if svc.attachManyCalls != 2 {
		t.Errorf("attachManyCalls = %d, want 2 (one per run)", svc.attachManyCalls)
	}
}

// ---------------------------------------------------------------------------
// AC-24: bug tag rm — detach + idempotent rerun
// ---------------------------------------------------------------------------

func TestBugTagRm_Detaches(t *testing.T) {
	withPlainTextJSONGlobal(t)
	root, svc, _, _ := buildEntityTagFixture(models.EntityTypeBug)

	root.SetArgs([]string{"bug", "tag", "rm", "B001", "voice"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if svc.detachOneCalls != 1 {
		t.Errorf("detachOneCalls = %d, want 1", svc.detachOneCalls)
	}
	if svc.lastDetachName != "voice" {
		t.Errorf("detach name = %q, want %q", svc.lastDetachName, "voice")
	}
}

func TestBugTagRm_IdempotentRerun(t *testing.T) {
	// AC-24: repository-level detach is a no-op if the attachment is
	// absent, so the service returns nil and the CLI exits 0 on repeat.
	withPlainTextJSONGlobal(t)
	root, svc, _, _ := buildEntityTagFixture(models.EntityTypeBug)

	for i := 0; i < 2; i++ {
		root.SetArgs([]string{"bug", "tag", "rm", "B001", "voice"})
		if err := root.Execute(); err != nil {
			t.Fatalf("Execute() iteration %d error = %v", i, err)
		}
	}
	if svc.detachOneCalls != 2 {
		t.Errorf("detachOneCalls = %d, want 2", svc.detachOneCalls)
	}
}

// ---------------------------------------------------------------------------
// AC-25: bug tag rm with an unregistered name — exit 1 + vocab snippet
// ---------------------------------------------------------------------------

func TestBugTagRm_UnregisteredNameErrors(t *testing.T) {
	withPlainTextJSONGlobal(t)
	root, svc, _, errBuf := buildEntityTagFixture(models.EntityTypeBug)

	// Vocabulary list used by the shared snippet helper.
	svc.listTagsFn = func(ctx context.Context) ([]*models.Tag, error) {
		return []*models.Tag{
			{ID: 1, Name: "voice"},
			{ID: 2, Name: "auth"},
		}, nil
	}
	svc.detachOneFn = func(ctx context.Context, entityType models.EntityType, entityID int64, name string) error {
		return &services.NotFoundError{Name: name}
	}

	root.SetArgs([]string{"bug", "tag", "rm", "B001", "does-not-exist"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for unregistered tag, got nil")
	}

	// The shared helper wraps the error with "exit code N:".
	if !strings.Contains(err.Error(), "exit code 1") {
		t.Errorf("error = %v, want wrapper 'exit code 1'", err)
	}

	stderr := errBuf.String()
	if !strings.Contains(stderr, "tag not found: does-not-exist") {
		t.Errorf("stderr missing error body, got %q", stderr)
	}
	// Vocabulary snippet
	if !strings.Contains(stderr, "voice") || !strings.Contains(stderr, "auth") {
		t.Errorf("stderr missing vocabulary snippet, got %q", stderr)
	}
	// Exact remediation line per AC-25.
	want := "To add it: shark tags add does-not-exist"
	if !strings.Contains(stderr, want) {
		t.Errorf("stderr missing %q, got %q", want, stderr)
	}
}

// ---------------------------------------------------------------------------
// AC-26: idea tag add — factory works with EntityTypeIdea identically
// ---------------------------------------------------------------------------

func TestIdeaTagAdd_BehavesLikeOtherEntities(t *testing.T) {
	withPlainTextJSONGlobal(t)
	root, svc, _, _ := buildEntityTagFixture(models.EntityTypeIdea)

	root.SetArgs([]string{"idea", "tag", "add", "7", "voice"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() idea tag add error = %v", err)
	}
	if svc.attachManyCalls != 1 {
		t.Errorf("attachManyCalls = %d, want 1", svc.attachManyCalls)
	}
	if svc.lastAttachEntType != models.EntityTypeIdea {
		t.Errorf("attach entity type = %q, want %q", svc.lastAttachEntType, models.EntityTypeIdea)
	}
}

// ---------------------------------------------------------------------------
// AC-26b: full table — every entity type × (add/rm) × (happy, unregistered)
// ---------------------------------------------------------------------------

func TestEntityTagCmd_AllEntityTypes_Table(t *testing.T) {
	entityTypes := []models.EntityType{
		models.EntityTypeTask,
		models.EntityTypeFeature,
		models.EntityTypeEpic,
		models.EntityTypeBug,
		models.EntityTypeChange,
		models.EntityTypeIdea,
	}

	for _, et := range entityTypes {
		for _, verb := range []string{"add", "rm"} {
			name := fmt.Sprintf("%s/%s/happy", et, verb)
			t.Run(name, func(t *testing.T) {
				withPlainTextJSONGlobal(t)
				root, svc, _, _ := buildEntityTagFixture(et)
				root.SetArgs([]string{string(et), "tag", verb, "key-1", "voice"})
				if err := root.Execute(); err != nil {
					t.Fatalf("happy path %s error = %v", verb, err)
				}
				if verb == "add" && svc.attachManyCalls != 1 {
					t.Errorf("attachManyCalls = %d, want 1", svc.attachManyCalls)
				}
				if verb == "rm" && svc.detachOneCalls != 1 {
					t.Errorf("detachOneCalls = %d, want 1", svc.detachOneCalls)
				}
			})

			unregisteredName := fmt.Sprintf("%s/%s/unregistered", et, verb)
			t.Run(unregisteredName, func(t *testing.T) {
				withPlainTextJSONGlobal(t)
				root, svc, _, _ := buildEntityTagFixture(et)

				// Populate the vocabulary for the snippet helper.
				svc.listTagsFn = func(ctx context.Context) ([]*models.Tag, error) {
					return []*models.Tag{{ID: 1, Name: "voice"}}, nil
				}
				// Make the service path fail with the typed error that
				// matches each verb (attach → UnregisteredTagError,
				// rm → NotFoundError).
				if verb == "add" {
					svc.attachManyFn = func(ctx context.Context, entityType models.EntityType, entityID int64, names []string) error {
						return &services.UnregisteredTagError{Name: names[0]}
					}
				} else {
					svc.detachOneFn = func(ctx context.Context, entityType models.EntityType, entityID int64, n string) error {
						return &services.NotFoundError{Name: n}
					}
				}

				root.SetArgs([]string{string(et), "tag", verb, "key-1", "ghost"})
				err := root.Execute()
				if err == nil {
					t.Fatal("expected error for unregistered tag, got nil")
				}
				// Spot check: the wrapped error carries the typed inner
				// error so exit-code mapping works.
				if verb == "add" {
					var inner *services.UnregisteredTagError
					if !errors.As(err, &inner) {
						t.Errorf("expected *UnregisteredTagError, got %T: %v", err, err)
					}
				} else {
					var inner *services.NotFoundError
					if !errors.As(err, &inner) {
						t.Errorf("expected *NotFoundError, got %T: %v", err, err)
					}
				}
			})
		}
	}
}

// ---------------------------------------------------------------------------
// Resolver error path: entity key not found
// ---------------------------------------------------------------------------

func TestEntityTagCmd_ResolveKeyFailureDoesNotTouchTagService(t *testing.T) {
	withPlainTextJSONGlobal(t)
	svc := &mockEntityTagService{}
	root := &cobra.Command{Use: "shark"}
	parent := &cobra.Command{Use: "bug"}
	root.AddCommand(parent)

	resolveKey := func(ctx context.Context, key string) (int64, error) {
		return 0, fmt.Errorf("nothing here")
	}
	parent.AddCommand(makeEntityTagCmd(models.EntityTypeBug, resolveKey, svc))

	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	root.SilenceErrors = true
	root.SilenceUsage = true

	root.SetArgs([]string{"bug", "tag", "add", "B999", "voice"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when resolve fails")
	}
	if svc.attachManyCalls != 0 {
		t.Errorf("attachManyCalls = %d, want 0 when resolve fails", svc.attachManyCalls)
	}
	if !strings.Contains(err.Error(), "B999") {
		t.Errorf("error should mention key, got: %v", err)
	}
}
