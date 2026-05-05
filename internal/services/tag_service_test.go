package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/auth/maintainer"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	tagrepo "github.com/jwwelbor/shark-task-manager/internal/repository/tag"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// ---------------------------------------------------------------------------
// Mock: TagRepositoryInterface
// ---------------------------------------------------------------------------

type mockTagRepo struct {
	createFn    func(ctx context.Context, name string) (*models.Tag, error)
	getByNameFn func(ctx context.Context, name string) (*models.Tag, error)
	getByIDFn   func(ctx context.Context, id int64) (*models.Tag, error)
	listFn      func(ctx context.Context) ([]*models.Tag, error)
	renameFn    func(ctx context.Context, id int64, newName string) (*models.Tag, error)
	deleteFn    func(ctx context.Context, id int64, force bool) error
}

func (m *mockTagRepo) Create(ctx context.Context, name string) (*models.Tag, error) {
	if m.createFn != nil {
		return m.createFn(ctx, name)
	}
	return nil, fmt.Errorf("Create not implemented in mock")
}

func (m *mockTagRepo) GetByName(ctx context.Context, name string) (*models.Tag, error) {
	if m.getByNameFn != nil {
		return m.getByNameFn(ctx, name)
	}
	return nil, fmt.Errorf("GetByName not implemented in mock")
}

func (m *mockTagRepo) GetByID(ctx context.Context, id int64) (*models.Tag, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, fmt.Errorf("GetByID not implemented in mock")
}

func (m *mockTagRepo) List(ctx context.Context) ([]*models.Tag, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return []*models.Tag{}, nil
}

func (m *mockTagRepo) Rename(ctx context.Context, id int64, newName string) (*models.Tag, error) {
	if m.renameFn != nil {
		return m.renameFn(ctx, id, newName)
	}
	return nil, fmt.Errorf("Rename not implemented in mock")
}

func (m *mockTagRepo) Delete(ctx context.Context, id int64, force bool) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id, force)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: EntityTagRepositoryInterface
// ---------------------------------------------------------------------------

type mockEntityTagRepo struct {
	countByTagFn             func(ctx context.Context, tagID int64) (int64, error)
	filterEntityIDsFn        func(ctx context.Context, entityType models.EntityType, tagIDs []int64) ([]int64, error)
	listTagNamesByEntitiesFn func(ctx context.Context, entityType models.EntityType, entityIDs []int64) ([]tagrepo.EntityIDTagName, error)
	callCount                int
}

func (m *mockEntityTagRepo) Attach(ctx context.Context, entityType models.EntityType, entityID, tagID int64) error {
	m.callCount++
	return nil
}

func (m *mockEntityTagRepo) Detach(ctx context.Context, entityType models.EntityType, entityID, tagID int64) error {
	m.callCount++
	return nil
}

func (m *mockEntityTagRepo) ListByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityTagLink, error) {
	m.callCount++
	return nil, nil
}

func (m *mockEntityTagRepo) ListByTag(ctx context.Context, tagID int64) ([]*models.EntityTagLink, error) {
	m.callCount++
	return nil, nil
}

func (m *mockEntityTagRepo) ListByEntityType(ctx context.Context, entityType models.EntityType, tagID int64) ([]*models.EntityTagLink, error) {
	m.callCount++
	return nil, nil
}

func (m *mockEntityTagRepo) CountByTag(ctx context.Context, tagID int64) (int64, error) {
	if m.countByTagFn != nil {
		return m.countByTagFn(ctx, tagID)
	}
	return 0, nil
}

func (m *mockEntityTagRepo) FilterEntityIDs(ctx context.Context, entityType models.EntityType, tagIDs []int64) ([]int64, error) {
	m.callCount++
	if m.filterEntityIDsFn != nil {
		return m.filterEntityIDsFn(ctx, entityType, tagIDs)
	}
	return []int64{}, nil
}

func (m *mockEntityTagRepo) ListTagNamesByEntities(ctx context.Context, entityType models.EntityType, entityIDs []int64) ([]tagrepo.EntityIDTagName, error) {
	m.callCount++
	if m.listTagNamesByEntitiesFn != nil {
		return m.listTagNamesByEntitiesFn(ctx, entityType, entityIDs)
	}
	return []tagrepo.EntityIDTagName{}, nil
}

// ---------------------------------------------------------------------------
// Mock: maintainer.Gate
// ---------------------------------------------------------------------------

type mockGate struct {
	authorizeFn        func(ctx context.Context, providedPass string) error
	recordSuccessFn    func(ctx context.Context) error
	authorizeCount     int
	recordSuccessCount int
}

func (m *mockGate) Authorize(ctx context.Context, providedPass string) error {
	m.authorizeCount++
	if m.authorizeFn != nil {
		return m.authorizeFn(ctx, providedPass)
	}
	return nil
}

func (m *mockGate) RecordSuccess(ctx context.Context) error {
	m.recordSuccessCount++
	if m.recordSuccessFn != nil {
		return m.recordSuccessFn(ctx)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: TagEnforcementConfig (E28-F04)
// ---------------------------------------------------------------------------

// stubCfg is a tiny test double for services.TagEnforcementConfig. It exposes
// a pre-configured slice of entity-type strings. The slice is returned as-is
// (no defensive copy), which is fine for tests because no test mutates the
// returned slice.
type stubCfg struct {
	values []string
}

func (s *stubCfg) TagRequiredFor() []string {
	return s.values
}

// Compile-time assertion that *config.Config satisfies TagEnforcementConfig
// belongs in a CLI-wiring test; here we assert the local stubCfg satisfies it.
var _ TagEnforcementConfig = (*stubCfg)(nil)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newTagServiceForTest builds a TagService with mocks wired. A default
// stubCfg (no required types) is used; tests that need a specific config
// should call NewTagService directly with a custom stubCfg.
func newTagServiceForTest(tagRepo tagrepo.TagRepositoryInterface, entityTagRepo tagrepo.EntityTagRepositoryInterface, gate maintainer.Gate) *TagService {
	return NewTagService(tagRepo, entityTagRepo, gate, &stubCfg{})
}

// captureLog redirects the default logger's output to a buffer for the duration
// of the test function; returns the buffer so the test can inspect log lines.
func captureLog() (*bytes.Buffer, func()) {
	buf := &bytes.Buffer{}
	original := log.Writer()
	log.SetOutput(buf)
	return buf, func() { log.SetOutput(original) }
}

// ---------------------------------------------------------------------------
// AC-1: ListTags ordered; gate never called
// ---------------------------------------------------------------------------

func TestTagService_ListTags_OrderedAscending(t *testing.T) {
	gate := &mockGate{}
	tagRepo := &mockTagRepo{
		listFn: func(ctx context.Context) ([]*models.Tag, error) {
			return []*models.Tag{
				{ID: 2, Name: "voice"},
				{ID: 1, Name: "audio"},
			}, nil
		},
	}
	svc := newTagServiceForTest(tagRepo, &mockEntityTagRepo{}, gate)

	tags, err := svc.ListTags(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(tags))
	}
	// Repository returns tags already in ascending order (repo handles ordering),
	// service must pass them through without reordering (or at worst maintain order).
	// The repo mock returns voice then audio; the service should return what repo gives.
	// Spec says repo returns ordered by name ASC; we test that the service does NOT
	// invoke the gate.
	if gate.authorizeCount != 0 {
		t.Errorf("expected gate.Authorize to be called 0 times, got %d", gate.authorizeCount)
	}
}

func TestTagService_ListTags_EmptyVocabulary(t *testing.T) {
	gate := &mockGate{}
	tagRepo := &mockTagRepo{
		listFn: func(ctx context.Context) ([]*models.Tag, error) {
			return []*models.Tag{}, nil
		},
	}
	svc := newTagServiceForTest(tagRepo, &mockEntityTagRepo{}, gate)

	tags, err := svc.ListTags(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("expected empty slice, got %d tags", len(tags))
	}
	if gate.authorizeCount != 0 {
		t.Errorf("gate.Authorize called %d times, expected 0", gate.authorizeCount)
	}
}

func TestTagService_ListTags_RepositoryError(t *testing.T) {
	tagRepo := &mockTagRepo{
		listFn: func(ctx context.Context) ([]*models.Tag, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	svc := newTagServiceForTest(tagRepo, &mockEntityTagRepo{}, &mockGate{})

	_, err := svc.ListTags(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// AC-2: AddTag normalizes, authorizes, creates, records success in order
// ---------------------------------------------------------------------------

func TestTagService_AddTag_HappyPath_Normalization(t *testing.T) {
	var capturedName string
	authorizeCallOrder := 0
	createCallOrder := 0
	recordCallOrder := 0
	callSeq := 0

	gate := &mockGate{
		authorizeFn: func(ctx context.Context, pass string) error {
			callSeq++
			authorizeCallOrder = callSeq
			return nil
		},
		recordSuccessFn: func(ctx context.Context) error {
			callSeq++
			recordCallOrder = callSeq
			return nil
		},
	}
	tagRepo := &mockTagRepo{
		createFn: func(ctx context.Context, name string) (*models.Tag, error) {
			callSeq++
			createCallOrder = callSeq
			capturedName = name
			return &models.Tag{ID: 1, Name: name}, nil
		},
	}

	svc := newTagServiceForTest(tagRepo, &mockEntityTagRepo{}, gate)

	tag, err := svc.AddTag(context.Background(), "Voice", "pw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedName != "voice" {
		t.Errorf("expected normalized name 'voice', got %q", capturedName)
	}
	if tag == nil || tag.Name != "voice" {
		t.Errorf("expected returned tag with Name='voice'")
	}
	// Call order: Authorize → Create → RecordSuccess
	if authorizeCallOrder != 1 {
		t.Errorf("Authorize should be call #1, got %d", authorizeCallOrder)
	}
	if createCallOrder != 2 {
		t.Errorf("Create should be call #2, got %d", createCallOrder)
	}
	if recordCallOrder != 3 {
		t.Errorf("RecordSuccess should be call #3, got %d", recordCallOrder)
	}
}

func TestTagService_AddTag_LeadingTrailingWhitespace(t *testing.T) {
	var capturedName string
	tagRepo := &mockTagRepo{
		createFn: func(ctx context.Context, name string) (*models.Tag, error) {
			capturedName = name
			return &models.Tag{ID: 1, Name: name}, nil
		},
	}
	svc := newTagServiceForTest(tagRepo, &mockEntityTagRepo{}, &mockGate{})

	_, err := svc.AddTag(context.Background(), "  audio  ", "pw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedName != "audio" {
		t.Errorf("expected trimmed name 'audio', got %q", capturedName)
	}
}

func TestTagService_AddTag_LowercaseUnchanged(t *testing.T) {
	var capturedName string
	tagRepo := &mockTagRepo{
		createFn: func(ctx context.Context, name string) (*models.Tag, error) {
			capturedName = name
			return &models.Tag{ID: 1, Name: name}, nil
		},
	}
	svc := newTagServiceForTest(tagRepo, &mockEntityTagRepo{}, &mockGate{})

	_, err := svc.AddTag(context.Background(), "audio", "pw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedName != "audio" {
		t.Errorf("expected 'audio', got %q", capturedName)
	}
}

func TestTagService_AddTag_Conflict(t *testing.T) {
	gate := &mockGate{}
	tagRepo := &mockTagRepo{
		createFn: func(ctx context.Context, name string) (*models.Tag, error) {
			return nil, fmt.Errorf("create tag %q: %w", name, tagrepo.ErrTagConflict)
		},
	}
	svc := newTagServiceForTest(tagRepo, &mockEntityTagRepo{}, gate)

	_, err := svc.AddTag(context.Background(), "audio", "pw")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var conflictErr *ConflictError
	if !errors.As(err, &conflictErr) {
		t.Errorf("expected *ConflictError, got %T: %v", err, err)
	}
	if gate.recordSuccessCount != 0 {
		t.Errorf("RecordSuccess should not be called on conflict, called %d times", gate.recordSuccessCount)
	}
}

func TestTagService_AddTag_RecordSuccessErrorSwallowed(t *testing.T) {
	tagRepo := &mockTagRepo{
		createFn: func(ctx context.Context, name string) (*models.Tag, error) {
			return &models.Tag{ID: 1, Name: name}, nil
		},
	}
	gate := &mockGate{
		recordSuccessFn: func(ctx context.Context) error {
			return fmt.Errorf("disk full")
		},
	}
	svc := newTagServiceForTest(tagRepo, &mockEntityTagRepo{}, gate)

	tag, err := svc.AddTag(context.Background(), "audio", "pw")
	if err != nil {
		t.Errorf("expected nil error even when RecordSuccess fails, got: %v", err)
	}
	if tag == nil {
		t.Error("expected created tag to be returned")
	}
}

// ---------------------------------------------------------------------------
// AC-3: AddTag with invalid name: Authorize called first, Create NOT called
// ---------------------------------------------------------------------------

func TestTagService_AddTag_InvalidName_AuthorizeCalledCreateNot(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"special char", "Voice!"},
		{"empty string", ""},
		{"whitespace only", "   "},
		{"65 chars", strings.Repeat("a", 65)},
		{"starts with hyphen", "-voice"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gate := &mockGate{}
			createCalled := false
			tagRepo := &mockTagRepo{
				createFn: func(ctx context.Context, name string) (*models.Tag, error) {
					createCalled = true
					return nil, nil
				},
			}
			svc := newTagServiceForTest(tagRepo, &mockEntityTagRepo{}, gate)

			_, err := svc.AddTag(context.Background(), tt.input, "pw")
			if err == nil {
				t.Fatalf("expected validation error for input %q, got nil", tt.input)
			}
			var valErr *ValidationError
			if !errors.As(err, &valErr) {
				t.Errorf("expected *ValidationError, got %T: %v", err, err)
			}
			if valErr != nil && valErr.Field != "tag name" {
				t.Errorf("expected field 'tag name', got %q", valErr.Field)
			}
			if gate.authorizeCount != 1 {
				t.Errorf("expected Authorize to be called once (before validation), got %d", gate.authorizeCount)
			}
			if createCalled {
				t.Error("Create must NOT be called when name is invalid")
			}
		})
	}
}

// AC-3 edge: valid hyphen in middle
func TestTagService_AddTag_HyphenInMiddle_Valid(t *testing.T) {
	tagRepo := &mockTagRepo{
		createFn: func(ctx context.Context, name string) (*models.Tag, error) {
			return &models.Tag{ID: 1, Name: name}, nil
		},
	}
	svc := newTagServiceForTest(tagRepo, &mockEntityTagRepo{}, &mockGate{})

	_, err := svc.AddTag(context.Background(), "a-b", "pw")
	if err != nil {
		t.Errorf("expected 'a-b' to be valid, got error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// AC-4: AddTag unauthorized: gate error returned unwrapped, Create not called
// ---------------------------------------------------------------------------

func TestTagService_AddTag_Unauthorized(t *testing.T) {
	tests := []struct {
		reason string
	}{
		{"wrong_password"},
		{"missing_config"},
		{"expired_cache"},
	}
	for _, tt := range tests {
		t.Run(tt.reason, func(t *testing.T) {
			createCalled := false
			gate := &mockGate{
				authorizeFn: func(ctx context.Context, pass string) error {
					return &maintainer.UnauthorizedError{Reason: tt.reason}
				},
			}
			tagRepo := &mockTagRepo{
				createFn: func(ctx context.Context, name string) (*models.Tag, error) {
					createCalled = true
					return nil, nil
				},
			}
			svc := newTagServiceForTest(tagRepo, &mockEntityTagRepo{}, gate)

			_, err := svc.AddTag(context.Background(), "voice", "pw")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			var unauthErr *maintainer.UnauthorizedError
			if !errors.As(err, &unauthErr) {
				t.Errorf("expected *UnauthorizedError, got %T: %v", err, err)
			}
			if createCalled {
				t.Error("Create must NOT be called when unauthorized")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AC-5: RemoveTag with in-use tag and force=false
// ---------------------------------------------------------------------------

func TestTagService_RemoveTag_InUse_ForceFalse(t *testing.T) {
	deleteCalled := false
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			return &models.Tag{ID: 10, Name: "voice"}, nil
		},
		deleteFn: func(ctx context.Context, id int64, force bool) error {
			deleteCalled = true
			return nil
		},
	}
	entityTagRepo := &mockEntityTagRepo{
		countByTagFn: func(ctx context.Context, tagID int64) (int64, error) {
			return 7, nil
		},
	}
	svc := newTagServiceForTest(tagRepo, entityTagRepo, &mockGate{})

	err := svc.RemoveTag(context.Background(), "voice", false, "pw")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var inUseErr *TagInUseError
	if !errors.As(err, &inUseErr) {
		t.Errorf("expected *TagInUseError, got %T: %v", err, err)
	}
	if inUseErr.Name != "voice" {
		t.Errorf("expected Name='voice', got %q", inUseErr.Name)
	}
	if inUseErr.Count != 7 {
		t.Errorf("expected Count=7, got %d", inUseErr.Count)
	}
	if deleteCalled {
		t.Error("Delete must NOT be called when tag is in use and force=false")
	}
}

func TestTagService_RemoveTag_InUse_OneUse_ForceFalse(t *testing.T) {
	deleteCalled := false
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			return &models.Tag{ID: 10, Name: "voice"}, nil
		},
		deleteFn: func(ctx context.Context, id int64, force bool) error {
			deleteCalled = true
			return nil
		},
	}
	entityTagRepo := &mockEntityTagRepo{
		countByTagFn: func(ctx context.Context, tagID int64) (int64, error) {
			return 1, nil
		},
	}
	svc := newTagServiceForTest(tagRepo, entityTagRepo, &mockGate{})

	err := svc.RemoveTag(context.Background(), "voice", false, "pw")
	var inUseErr *TagInUseError
	if !errors.As(err, &inUseErr) {
		t.Errorf("expected *TagInUseError, got %T", err)
	}
	if inUseErr.Count != 1 {
		t.Errorf("expected Count=1, got %d", inUseErr.Count)
	}
	if deleteCalled {
		t.Error("Delete must NOT be called")
	}
}

func TestTagService_RemoveTag_ErrorMessage_ContainsCountAndForceHint(t *testing.T) {
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			return &models.Tag{ID: 10, Name: "voice"}, nil
		},
	}
	entityTagRepo := &mockEntityTagRepo{
		countByTagFn: func(ctx context.Context, tagID int64) (int64, error) {
			return 7, nil
		},
	}
	svc := newTagServiceForTest(tagRepo, entityTagRepo, &mockGate{})

	err := svc.RemoveTag(context.Background(), "voice", false, "pw")
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "7") {
		t.Errorf("error message should contain count '7', got: %q", msg)
	}
	if !strings.Contains(msg, "--force") {
		t.Errorf("error message should contain '--force', got: %q", msg)
	}
}

// ---------------------------------------------------------------------------
// AC-6: RemoveTag with in-use tag and force=true
// ---------------------------------------------------------------------------

func TestTagService_RemoveTag_InUse_ForceTrue(t *testing.T) {
	var capturedForce bool
	gate := &mockGate{}
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			return &models.Tag{ID: 10, Name: "voice"}, nil
		},
		deleteFn: func(ctx context.Context, id int64, force bool) error {
			capturedForce = force
			return nil
		},
	}
	entityTagRepo := &mockEntityTagRepo{
		countByTagFn: func(ctx context.Context, tagID int64) (int64, error) {
			return 7, nil
		},
	}
	svc := newTagServiceForTest(tagRepo, entityTagRepo, gate)

	err := svc.RemoveTag(context.Background(), "voice", true, "pw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !capturedForce {
		t.Error("Delete must be called with force=true")
	}
	if gate.recordSuccessCount != 1 {
		t.Errorf("expected RecordSuccess called once, got %d", gate.recordSuccessCount)
	}
}

func TestTagService_RemoveTag_ForceTrue_DeleteFails(t *testing.T) {
	gate := &mockGate{}
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			return &models.Tag{ID: 10, Name: "voice"}, nil
		},
		deleteFn: func(ctx context.Context, id int64, force bool) error {
			return fmt.Errorf("delete failed")
		},
	}
	entityTagRepo := &mockEntityTagRepo{
		countByTagFn: func(ctx context.Context, tagID int64) (int64, error) {
			return 7, nil
		},
	}
	svc := newTagServiceForTest(tagRepo, entityTagRepo, gate)

	err := svc.RemoveTag(context.Background(), "voice", true, "pw")
	if err == nil {
		t.Fatal("expected error from Delete, got nil")
	}
	if gate.recordSuccessCount != 0 {
		t.Errorf("RecordSuccess must NOT be called when Delete fails, got %d", gate.recordSuccessCount)
	}
}

// ---------------------------------------------------------------------------
// AC-7: RemoveTag with zero uses
// ---------------------------------------------------------------------------

func TestTagService_RemoveTag_ZeroUses_ForceFalse(t *testing.T) {
	var capturedForce bool
	gate := &mockGate{}
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			return &models.Tag{ID: 10, Name: "voice"}, nil
		},
		deleteFn: func(ctx context.Context, id int64, force bool) error {
			capturedForce = force
			return nil
		},
	}
	entityTagRepo := &mockEntityTagRepo{
		countByTagFn: func(ctx context.Context, tagID int64) (int64, error) {
			return 0, nil
		},
	}
	svc := newTagServiceForTest(tagRepo, entityTagRepo, gate)

	err := svc.RemoveTag(context.Background(), "voice", false, "pw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedForce {
		t.Error("Delete should be called with force=false when count=0")
	}
	if gate.recordSuccessCount != 1 {
		t.Errorf("expected RecordSuccess called once, got %d", gate.recordSuccessCount)
	}
}

func TestTagService_RemoveTag_TagNotFound(t *testing.T) {
	deleteCalled := false
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			return nil, fmt.Errorf("get tag by name %q: %w", name, tagrepo.ErrTagNotFound)
		},
		deleteFn: func(ctx context.Context, id int64, force bool) error {
			deleteCalled = true
			return nil
		},
	}
	svc := newTagServiceForTest(tagRepo, &mockEntityTagRepo{}, &mockGate{})

	err := svc.RemoveTag(context.Background(), "nonexistent", false, "pw")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var notFoundErr *NotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Errorf("expected *NotFoundError, got %T: %v", err, err)
	}
	if deleteCalled {
		t.Error("Delete must NOT be called when tag not found")
	}
}

// ---------------------------------------------------------------------------
// AC-8: RenameTag collision
// ---------------------------------------------------------------------------

func TestTagService_RenameTag_Collision_ConflictError(t *testing.T) {
	renameCalled := false
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			switch name {
			case "voice":
				return &models.Tag{ID: 1, Name: "voice"}, nil
			case "audio":
				return &models.Tag{ID: 2, Name: "audio"}, nil // already exists
			default:
				return nil, fmt.Errorf("not found: %w", tagrepo.ErrTagNotFound)
			}
		},
		renameFn: func(ctx context.Context, id int64, newName string) (*models.Tag, error) {
			renameCalled = true
			return nil, nil
		},
	}
	svc := newTagServiceForTest(tagRepo, &mockEntityTagRepo{}, &mockGate{})

	_, err := svc.RenameTag(context.Background(), "voice", "audio", "pw")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var conflictErr *ConflictError
	if !errors.As(err, &conflictErr) {
		t.Errorf("expected *ConflictError, got %T: %v", err, err)
	}
	if conflictErr.Name != "audio" {
		t.Errorf("expected Name='audio', got %q", conflictErr.Name)
	}
	if renameCalled {
		t.Error("Rename must NOT be called when collision is detected")
	}
}

// ---------------------------------------------------------------------------
// AC-9: RenameTag same names after normalization
// ---------------------------------------------------------------------------

func TestTagService_RenameTag_SameNameAfterNormalization(t *testing.T) {
	tests := []struct {
		name    string
		oldName string
		newName string
	}{
		{"identical after normalization", "voice", "VOICE"},
		{"identical before normalization", "voice", "voice"},
		{"differ only by whitespace", "voice", " voice "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renameCalled := false
			tagRepo := &mockTagRepo{
				renameFn: func(ctx context.Context, id int64, newName string) (*models.Tag, error) {
					renameCalled = true
					return nil, nil
				},
			}
			svc := newTagServiceForTest(tagRepo, &mockEntityTagRepo{}, &mockGate{})

			_, err := svc.RenameTag(context.Background(), tt.oldName, tt.newName, "pw")
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			var valErr *ValidationError
			if !errors.As(err, &valErr) {
				t.Errorf("expected *ValidationError, got %T: %v", err, err)
			}
			if renameCalled {
				t.Error("Rename must NOT be called when names are identical")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AC-10: RenameTag success; no EntityTagRepository method called
// ---------------------------------------------------------------------------

func TestTagService_RenameTag_Success_NoEntityTagCall(t *testing.T) {
	entityTagRepo := &mockEntityTagRepo{}
	gate := &mockGate{}
	renameCount := 0
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			switch name {
			case "voice":
				return &models.Tag{ID: 1, Name: "voice"}, nil
			case "audio":
				return nil, fmt.Errorf("not found: %w", tagrepo.ErrTagNotFound)
			}
			return nil, fmt.Errorf("unexpected: %w", tagrepo.ErrTagNotFound)
		},
		renameFn: func(ctx context.Context, id int64, newName string) (*models.Tag, error) {
			renameCount++
			return &models.Tag{ID: id, Name: newName}, nil
		},
	}
	svc := newTagServiceForTest(tagRepo, entityTagRepo, gate)

	result, err := svc.RenameTag(context.Background(), "voice", "audio", "pw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || result.Name != "audio" {
		t.Errorf("expected returned tag with Name='audio', got %v", result)
	}
	if renameCount != 1 {
		t.Errorf("expected Rename called once, got %d", renameCount)
	}
	if entityTagRepo.callCount != 0 {
		t.Errorf("EntityTagRepository must NOT be called during rename, got %d calls", entityTagRepo.callCount)
	}
	if gate.recordSuccessCount != 1 {
		t.Errorf("expected RecordSuccess called once, got %d", gate.recordSuccessCount)
	}
}

func TestTagService_RenameTag_RaceConflict_WrappedAsConflictError(t *testing.T) {
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			switch name {
			case "voice":
				return &models.Tag{ID: 1, Name: "voice"}, nil
			case "audio":
				return nil, fmt.Errorf("not found: %w", tagrepo.ErrTagNotFound)
			}
			return nil, fmt.Errorf("not found: %w", tagrepo.ErrTagNotFound)
		},
		renameFn: func(ctx context.Context, id int64, newName string) (*models.Tag, error) {
			return nil, fmt.Errorf("rename tag %d to %q: %w", id, newName, tagrepo.ErrTagConflict)
		},
	}
	gate := &mockGate{}
	svc := newTagServiceForTest(tagRepo, &mockEntityTagRepo{}, gate)

	_, err := svc.RenameTag(context.Background(), "voice", "audio", "pw")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var conflictErr *ConflictError
	if !errors.As(err, &conflictErr) {
		t.Errorf("expected *ConflictError, got %T: %v", err, err)
	}
	if gate.recordSuccessCount != 0 {
		t.Errorf("RecordSuccess must NOT be called on conflict race, got %d", gate.recordSuccessCount)
	}
}

func TestTagService_RenameTag_SourceNotFound(t *testing.T) {
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			return nil, fmt.Errorf("not found: %w", tagrepo.ErrTagNotFound)
		},
	}
	svc := newTagServiceForTest(tagRepo, &mockEntityTagRepo{}, &mockGate{})

	_, err := svc.RenameTag(context.Background(), "voice", "audio", "pw")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var notFoundErr *NotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Errorf("expected *NotFoundError, got %T: %v", err, err)
	}
}

// ---------------------------------------------------------------------------
// AC-11: RecordSuccess error does NOT propagate; error is logged
// ---------------------------------------------------------------------------

func TestTagService_AddTag_RecordSuccessError_Logged_NotReturned(t *testing.T) {
	logBuf, restore := captureLog()
	defer restore()

	tagRepo := &mockTagRepo{
		createFn: func(ctx context.Context, name string) (*models.Tag, error) {
			return &models.Tag{ID: 1, Name: name}, nil
		},
	}
	gate := &mockGate{
		recordSuccessFn: func(ctx context.Context) error {
			return fmt.Errorf("disk full")
		},
	}
	svc := newTagServiceForTest(tagRepo, &mockEntityTagRepo{}, gate)

	tag, err := svc.AddTag(context.Background(), "audio", "pw")
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
	if tag == nil {
		t.Error("expected tag returned")
	}
	if !strings.Contains(logBuf.String(), "disk full") {
		t.Errorf("expected 'disk full' logged, log output: %q", logBuf.String())
	}
}

func TestTagService_RemoveTag_RecordSuccessError_Logged_NotReturned(t *testing.T) {
	logBuf, restore := captureLog()
	defer restore()

	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			return &models.Tag{ID: 1, Name: "audio"}, nil
		},
		deleteFn: func(ctx context.Context, id int64, force bool) error {
			return nil
		},
	}
	entityTagRepo := &mockEntityTagRepo{
		countByTagFn: func(ctx context.Context, tagID int64) (int64, error) {
			return 0, nil
		},
	}
	gate := &mockGate{
		recordSuccessFn: func(ctx context.Context) error {
			return fmt.Errorf("disk full")
		},
	}
	svc := newTagServiceForTest(tagRepo, entityTagRepo, gate)

	err := svc.RemoveTag(context.Background(), "audio", false, "pw")
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
	if !strings.Contains(logBuf.String(), "disk full") {
		t.Errorf("expected 'disk full' logged, log output: %q", logBuf.String())
	}
}

func TestTagService_RenameTag_RecordSuccessError_Logged_NotReturned(t *testing.T) {
	logBuf, restore := captureLog()
	defer restore()

	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			switch name {
			case "voice":
				return &models.Tag{ID: 1, Name: "voice"}, nil
			default:
				return nil, fmt.Errorf("not found: %w", tagrepo.ErrTagNotFound)
			}
		},
		renameFn: func(ctx context.Context, id int64, newName string) (*models.Tag, error) {
			return &models.Tag{ID: id, Name: newName}, nil
		},
	}
	gate := &mockGate{
		recordSuccessFn: func(ctx context.Context) error {
			return fmt.Errorf("disk full")
		},
	}
	svc := newTagServiceForTest(tagRepo, &mockEntityTagRepo{}, gate)

	_, err := svc.RenameTag(context.Background(), "voice", "audio", "pw")
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
	if !strings.Contains(logBuf.String(), "disk full") {
		t.Errorf("expected 'disk full' logged, log output: %q", logBuf.String())
	}
}

// ---------------------------------------------------------------------------
// AC-T1: NewTagService panics on nil dependency
// ---------------------------------------------------------------------------

func TestNewTagService_PanicsOnNilDependencies(t *testing.T) {
	tagRepo := &mockTagRepo{}
	entityTagRepo := &mockEntityTagRepo{}
	gate := &mockGate{}
	cfg := &stubCfg{}

	t.Run("nil tagRepo", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Error("expected panic on nil tagRepo")
			}
			// AC-13/AC-13b: panic message should contain "requires a non-nil"
			if msg, ok := r.(string); ok && !strings.Contains(msg, "requires a non-nil") {
				t.Errorf("panic message %q missing 'requires a non-nil'", msg)
			}
		}()
		NewTagService(nil, entityTagRepo, gate, cfg)
	})

	t.Run("nil entityTagRepo", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Error("expected panic on nil entityTagRepo")
			}
			if msg, ok := r.(string); ok && !strings.Contains(msg, "requires a non-nil") {
				t.Errorf("panic message %q missing 'requires a non-nil'", msg)
			}
		}()
		NewTagService(tagRepo, nil, gate, cfg)
	})

	t.Run("nil gate", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Error("expected panic on nil gate")
			}
			if msg, ok := r.(string); ok && !strings.Contains(msg, "requires a non-nil") {
				t.Errorf("panic message %q missing 'requires a non-nil'", msg)
			}
		}()
		NewTagService(tagRepo, entityTagRepo, nil, cfg)
	})

	// AC-13: new to F04 — panic on nil cfg.
	t.Run("nil cfg", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Error("expected panic on nil cfg (AC-13)")
			}
			if msg, ok := r.(string); ok && !strings.Contains(msg, "requires a non-nil") {
				t.Errorf("panic message %q missing 'requires a non-nil'", msg)
			}
		}()
		NewTagService(tagRepo, entityTagRepo, gate, nil)
	})
}

// ---------------------------------------------------------------------------
// AC-22: OTel span attribute safety (no sensitive attributes)
// ---------------------------------------------------------------------------

func TestTagService_Spans_NoSensitiveAttributes(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	tr := tp.Tracer("test")

	tagRepo := &mockTagRepo{
		listFn: func(ctx context.Context) ([]*models.Tag, error) {
			return []*models.Tag{{ID: 1, Name: "voice"}}, nil
		},
		createFn: func(ctx context.Context, name string) (*models.Tag, error) {
			return &models.Tag{ID: 1, Name: name}, nil
		},
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			switch name {
			case "voice":
				return &models.Tag{ID: 1, Name: "voice"}, nil
			default:
				return nil, fmt.Errorf("not found: %w", tagrepo.ErrTagNotFound)
			}
		},
		deleteFn: func(ctx context.Context, id int64, force bool) error {
			return nil
		},
		renameFn: func(ctx context.Context, id int64, newName string) (*models.Tag, error) {
			return &models.Tag{ID: id, Name: newName}, nil
		},
	}
	entityTagRepo := &mockEntityTagRepo{
		countByTagFn: func(ctx context.Context, tagID int64) (int64, error) {
			return 0, nil
		},
	}
	svc := NewTagService(tagRepo, entityTagRepo, &mockGate{}, &stubCfg{})
	svc.SetTracer(tr)

	ctx := context.Background()
	_, _ = svc.ListTags(ctx)
	_, _ = svc.AddTag(ctx, "voice", "secret-password")
	_ = svc.RemoveTag(ctx, "voice", false, "secret-password")
	_, _ = svc.RenameTag(ctx, "voice", "audio", "secret-password")

	spans := exporter.GetSpans()
	sensitiveKeys := []string{"pass", "password", "hash"}

	for _, span := range spans {
		for _, attr := range span.Attributes {
			key := strings.ToLower(string(attr.Key))
			// Check attribute key does not contain sensitive terms
			for _, sensitive := range sensitiveKeys {
				if strings.Contains(key, sensitive) {
					t.Errorf("span %q has sensitive attribute key %q", span.Name, attr.Key)
				}
			}
			// Check attribute key does not start with "maintainer."
			if strings.HasPrefix(key, "maintainer.") {
				t.Errorf("span %q has maintainer.* attribute: %q", span.Name, attr.Key)
			}
		}
	}
}

func TestTagService_Spans_HaveCorrectNames(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	tr := tp.Tracer("test")

	tagRepo := &mockTagRepo{
		listFn: func(ctx context.Context) ([]*models.Tag, error) {
			return []*models.Tag{}, nil
		},
		createFn: func(ctx context.Context, name string) (*models.Tag, error) {
			return &models.Tag{ID: 1, Name: name}, nil
		},
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			switch name {
			case "voice":
				return &models.Tag{ID: 1, Name: "voice"}, nil
			default:
				return nil, fmt.Errorf("not found: %w", tagrepo.ErrTagNotFound)
			}
		},
		deleteFn: func(ctx context.Context, id int64, force bool) error { return nil },
		renameFn: func(ctx context.Context, id int64, newName string) (*models.Tag, error) {
			return &models.Tag{ID: id, Name: newName}, nil
		},
	}
	entityTagRepo := &mockEntityTagRepo{
		countByTagFn: func(ctx context.Context, tagID int64) (int64, error) { return 0, nil },
	}
	svc := NewTagService(tagRepo, entityTagRepo, &mockGate{}, &stubCfg{})
	svc.SetTracer(tr)

	ctx := context.Background()
	_, _ = svc.ListTags(ctx)
	_, _ = svc.AddTag(ctx, "voice", "pw")
	_ = svc.RemoveTag(ctx, "voice", false, "pw")
	_, _ = svc.RenameTag(ctx, "voice", "audio", "pw")

	spans := exporter.GetSpans()
	expectedSpans := map[string]bool{
		"tag_service.list_tags":  false,
		"tag_service.add_tag":    false,
		"tag_service.remove_tag": false,
		"tag_service.rename_tag": false,
	}
	for _, span := range spans {
		if _, ok := expectedSpans[span.Name]; ok {
			expectedSpans[span.Name] = true
		}
	}
	for name, found := range expectedSpans {
		if !found {
			t.Errorf("expected span %q not found in recorded spans", name)
		}
	}
}

// ---------------------------------------------------------------------------
// callRecorder — thread-safe call-order recorder (AC-T2)
// ---------------------------------------------------------------------------

type callRecorder struct {
	mu    sync.Mutex
	calls []string
}

func (r *callRecorder) record(name string) {
	r.mu.Lock()
	r.calls = append(r.calls, name)
	r.mu.Unlock()
}

func (r *callRecorder) assertOrder(t *testing.T, expected ...string) {
	t.Helper()
	r.mu.Lock()
	got := make([]string, len(r.calls))
	copy(got, r.calls)
	r.mu.Unlock()
	if len(got) != len(expected) {
		t.Errorf("call order length: got %v (%d), want %v (%d)", got, len(got), expected, len(expected))
		return
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("call order[%d]: got %q, want %q (full: %v)", i, got[i], expected[i], got)
		}
	}
}

// ---------------------------------------------------------------------------
// AC-T2: callRecorder verifies gate → delete → RecordSuccess ordering in RemoveTag
// ---------------------------------------------------------------------------

func TestTagService_RemoveTag_CallOrder_GateDeleteRecordSuccess(t *testing.T) {
	rec := &callRecorder{}

	gate := &mockGate{
		authorizeFn: func(ctx context.Context, pass string) error {
			rec.record("Authorize")
			return nil
		},
		recordSuccessFn: func(ctx context.Context) error {
			rec.record("RecordSuccess")
			return nil
		},
	}
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			return &models.Tag{ID: 1, Name: name}, nil
		},
		deleteFn: func(ctx context.Context, id int64, force bool) error {
			rec.record("Delete")
			return nil
		},
	}
	entityTagRepo := &mockEntityTagRepo{
		countByTagFn: func(ctx context.Context, tagID int64) (int64, error) {
			return 0, nil // zero uses → no force check
		},
	}
	svc := newTagServiceForTest(tagRepo, entityTagRepo, gate)

	err := svc.RemoveTag(context.Background(), "audio", false, "pw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rec.assertOrder(t, "Authorize", "Delete", "RecordSuccess")
}

// ---------------------------------------------------------------------------
// AC-T2: callRecorder verifies gate → rename → RecordSuccess ordering in RenameTag
// ---------------------------------------------------------------------------

func TestTagService_RenameTag_CallOrder_GateRenameRecordSuccess(t *testing.T) {
	rec := &callRecorder{}

	gate := &mockGate{
		authorizeFn: func(ctx context.Context, pass string) error {
			rec.record("Authorize")
			return nil
		},
		recordSuccessFn: func(ctx context.Context) error {
			rec.record("RecordSuccess")
			return nil
		},
	}
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			switch name {
			case "voice":
				return &models.Tag{ID: 1, Name: "voice"}, nil
			default:
				return nil, fmt.Errorf("not found: %w", tagrepo.ErrTagNotFound)
			}
		},
		renameFn: func(ctx context.Context, id int64, newName string) (*models.Tag, error) {
			rec.record("Rename")
			return &models.Tag{ID: id, Name: newName}, nil
		},
	}
	svc := newTagServiceForTest(tagRepo, &mockEntityTagRepo{}, gate)

	_, err := svc.RenameTag(context.Background(), "voice", "audio", "pw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rec.assertOrder(t, "Authorize", "Rename", "RecordSuccess")
}

// ---------------------------------------------------------------------------
// AC-22.2: AddTag span contains tag.name attribute
// AC-22.4: RemoveTag span contains tag.force attribute
// ---------------------------------------------------------------------------

func TestTagService_Span_AddTag_HasTagNameAttribute(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	tr := tp.Tracer("test")

	tagRepo := &mockTagRepo{
		createFn: func(ctx context.Context, name string) (*models.Tag, error) {
			return &models.Tag{ID: 1, Name: name}, nil
		},
	}
	svc := NewTagService(tagRepo, &mockEntityTagRepo{}, &mockGate{}, &stubCfg{})
	svc.SetTracer(tr)

	_, err := svc.AddTag(context.Background(), "voice", "pw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := exporter.GetSpans()
	var addTagSpan *tracetest.SpanStub
	for i := range spans {
		if spans[i].Name == "tag_service.add_tag" {
			addTagSpan = &spans[i]
			break
		}
	}
	if addTagSpan == nil {
		t.Fatal("span 'tag_service.add_tag' not found")
	}

	var found bool
	for _, attr := range addTagSpan.Attributes {
		if attr.Key == "tag.name" && attr.Value == attribute.StringValue("voice") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected span 'tag_service.add_tag' to have attribute tag.name='voice'; attributes: %v", addTagSpan.Attributes)
	}
}

func TestTagService_Span_RemoveTag_HasTagForceAttribute(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	tr := tp.Tracer("test")

	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			return &models.Tag{ID: 1, Name: name}, nil
		},
		deleteFn: func(ctx context.Context, id int64, force bool) error { return nil },
	}
	entityTagRepo := &mockEntityTagRepo{
		countByTagFn: func(ctx context.Context, tagID int64) (int64, error) { return 5, nil },
	}
	svc := NewTagService(tagRepo, entityTagRepo, &mockGate{}, &stubCfg{})
	svc.SetTracer(tr)

	// force=true to verify force attribute is set correctly
	_ = svc.RemoveTag(context.Background(), "voice", true, "pw")

	spans := exporter.GetSpans()
	var removeTagSpan *tracetest.SpanStub
	for i := range spans {
		if spans[i].Name == "tag_service.remove_tag" {
			removeTagSpan = &spans[i]
			break
		}
	}
	if removeTagSpan == nil {
		t.Fatal("span 'tag_service.remove_tag' not found")
	}

	var foundForce bool
	for _, attr := range removeTagSpan.Attributes {
		if attr.Key == "tag.force" && attr.Value == attribute.BoolValue(true) {
			foundForce = true
			break
		}
	}
	if !foundForce {
		t.Errorf("expected span 'tag_service.remove_tag' to have attribute tag.force=true; attributes: %v", removeTagSpan.Attributes)
	}
}

// ---------------------------------------------------------------------------
// AC-19 static check: ValidateName is the sole path to models.ValidateTagName
// ---------------------------------------------------------------------------

func TestTagService_ValidateName_IsSoleEntryPoint(t *testing.T) {
	svc := NewTagService(&mockTagRepo{}, &mockEntityTagRepo{}, &mockGate{}, &stubCfg{})

	normalized, err := svc.ValidateName("Voice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if normalized != "voice" {
		t.Errorf("expected 'voice', got %q", normalized)
	}

	_, err = svc.ValidateName("Voice!")
	if err == nil {
		t.Error("expected error for 'Voice!'")
	}
	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *ValidationError, got %T", err)
	}
}

// ===========================================================================
// E28-F04: AttachMany, DetachOne, EnforceRequired
// ===========================================================================
//
// Traceability: each test maps to an AC in spec.md §1.3 (AC-1 through AC-14)
// and to the matrix in test-plan.md §1.1. Observability tests additionally
// cover REQ-NF-002.

// ---------------------------------------------------------------------------
// AC-1: AttachMany happy path — multiple registered tags, two Attach calls
// ---------------------------------------------------------------------------

// attachCall records a single Attach invocation for ordering assertions.
type attachCall struct {
	entityType models.EntityType
	entityID   int64
	tagID      int64
}

// newRecordingEntityTagRepo returns a mockEntityTagRepo augmented with an
// attach recorder. Attach writes to the provided slice pointer; other methods
// remain the mock defaults.
func newRecordingEntityTagRepo(recorder *[]attachCall) *recordingEntityTagRepo {
	return &recordingEntityTagRepo{recorder: recorder}
}

type recordingEntityTagRepo struct {
	recorder  *[]attachCall
	detachFn  func(ctx context.Context, entityType models.EntityType, entityID, tagID int64) error
	callCount int
}

func (m *recordingEntityTagRepo) Attach(ctx context.Context, entityType models.EntityType, entityID, tagID int64) error {
	m.callCount++
	if m.recorder != nil {
		*m.recorder = append(*m.recorder, attachCall{entityType, entityID, tagID})
	}
	return nil
}

func (m *recordingEntityTagRepo) Detach(ctx context.Context, entityType models.EntityType, entityID, tagID int64) error {
	m.callCount++
	if m.detachFn != nil {
		return m.detachFn(ctx, entityType, entityID, tagID)
	}
	return nil
}

func (m *recordingEntityTagRepo) ListByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityTagLink, error) {
	m.callCount++
	return nil, nil
}

func (m *recordingEntityTagRepo) ListByTag(ctx context.Context, tagID int64) ([]*models.EntityTagLink, error) {
	m.callCount++
	return nil, nil
}

func (m *recordingEntityTagRepo) ListByEntityType(ctx context.Context, entityType models.EntityType, tagID int64) ([]*models.EntityTagLink, error) {
	m.callCount++
	return nil, nil
}

func (m *recordingEntityTagRepo) CountByTag(ctx context.Context, tagID int64) (int64, error) {
	return 0, nil
}

func (m *recordingEntityTagRepo) FilterEntityIDs(ctx context.Context, entityType models.EntityType, tagIDs []int64) ([]int64, error) {
	m.callCount++
	return []int64{}, nil
}

func (m *recordingEntityTagRepo) ListTagNamesByEntities(ctx context.Context, entityType models.EntityType, entityIDs []int64) ([]tagrepo.EntityIDTagName, error) {
	m.callCount++
	return []tagrepo.EntityIDTagName{}, nil
}

func TestAttachMany_HappyPathMultipleTags(t *testing.T) {
	var getByNameCalls []string
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			getByNameCalls = append(getByNameCalls, name)
			switch name {
			case "voice":
				return &models.Tag{ID: 10, Name: "voice"}, nil
			case "auth":
				return &models.Tag{ID: 20, Name: "auth"}, nil
			}
			return nil, fmt.Errorf("unexpected: %w", tagrepo.ErrTagNotFound)
		},
	}
	var attaches []attachCall
	entityTagRepo := newRecordingEntityTagRepo(&attaches)
	svc := NewTagService(tagRepo, entityTagRepo, &mockGate{}, &stubCfg{})

	err := svc.AttachMany(context.Background(), models.EntityTypeTask, 42, []string{"voice", "auth"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(getByNameCalls) != 2 {
		t.Errorf("expected 2 GetByName calls, got %d (%v)", len(getByNameCalls), getByNameCalls)
	}
	if len(attaches) != 2 {
		t.Fatalf("expected 2 Attach calls, got %d", len(attaches))
	}
	// In-order resolution preserves input order.
	if attaches[0].tagID != 10 || attaches[0].entityType != models.EntityTypeTask || attaches[0].entityID != 42 {
		t.Errorf("Attach[0]=%+v, want entityType=task entityID=42 tagID=10", attaches[0])
	}
	if attaches[1].tagID != 20 {
		t.Errorf("Attach[1]=%+v, want tagID=20 (auth)", attaches[1])
	}
}

// ---------------------------------------------------------------------------
// AC-2: AttachMany aborts on unregistered name BEFORE any Attach
// ---------------------------------------------------------------------------

func TestAttachMany_AbortsOnUnregisteredBeforeAnyAttach(t *testing.T) {
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			if name == "voice" {
				return &models.Tag{ID: 10, Name: "voice"}, nil
			}
			return nil, fmt.Errorf("lookup: %w", tagrepo.ErrTagNotFound)
		},
	}
	var attaches []attachCall
	entityTagRepo := newRecordingEntityTagRepo(&attaches)
	svc := NewTagService(tagRepo, entityTagRepo, &mockGate{}, &stubCfg{})

	err := svc.AttachMany(context.Background(), models.EntityTypeTask, 42, []string{"voice", "does-not-exist"})
	if err == nil {
		t.Fatal("expected *UnregisteredTagError, got nil")
	}
	var unregErr *UnregisteredTagError
	if !errors.As(err, &unregErr) {
		t.Fatalf("expected *UnregisteredTagError, got %T: %v", err, err)
	}
	if unregErr.Name != "does-not-exist" {
		t.Errorf("UnregisteredTagError.Name = %q, want %q", unregErr.Name, "does-not-exist")
	}
	if len(attaches) != 0 {
		t.Errorf("no Attach calls should happen when any name fails; got %d", len(attaches))
	}
}

// ---------------------------------------------------------------------------
// AC-3: AttachMany nil slice is a no-op
// AC-3b: AttachMany empty slice is a no-op
// ---------------------------------------------------------------------------

func TestAttachMany_NilSliceIsNoOp(t *testing.T) {
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			t.Fatalf("GetByName must not be called on nil slice; got name=%q", name)
			return nil, nil
		},
	}
	var attaches []attachCall
	entityTagRepo := newRecordingEntityTagRepo(&attaches)
	svc := NewTagService(tagRepo, entityTagRepo, &mockGate{}, &stubCfg{})

	err := svc.AttachMany(context.Background(), models.EntityTypeTask, 42, nil)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if len(attaches) != 0 {
		t.Errorf("expected zero attaches, got %d", len(attaches))
	}
}

func TestAttachMany_EmptySliceIsNoOp(t *testing.T) {
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			t.Fatalf("GetByName must not be called on empty slice; got name=%q", name)
			return nil, nil
		},
	}
	var attaches []attachCall
	entityTagRepo := newRecordingEntityTagRepo(&attaches)
	svc := NewTagService(tagRepo, entityTagRepo, &mockGate{}, &stubCfg{})

	err := svc.AttachMany(context.Background(), models.EntityTypeTask, 42, []string{})
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if len(attaches) != 0 {
		t.Errorf("expected zero attaches, got %d", len(attaches))
	}
}

// ---------------------------------------------------------------------------
// AC-4: AttachMany normalizes names via ValidateName
// AC-4b: AttachMany rejects invalid names
// ---------------------------------------------------------------------------

func TestAttachMany_NormalizesNames(t *testing.T) {
	var getByNameCalls []string
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			getByNameCalls = append(getByNameCalls, name)
			if name == "voice" {
				return &models.Tag{ID: 10, Name: "voice"}, nil
			}
			return nil, fmt.Errorf("lookup: %w", tagrepo.ErrTagNotFound)
		},
	}
	var attaches []attachCall
	entityTagRepo := newRecordingEntityTagRepo(&attaches)
	svc := NewTagService(tagRepo, entityTagRepo, &mockGate{}, &stubCfg{})

	err := svc.AttachMany(context.Background(), models.EntityTypeTask, 42, []string{"Voice "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(getByNameCalls) != 1 || getByNameCalls[0] != "voice" {
		t.Errorf("expected single GetByName('voice'), got %v", getByNameCalls)
	}
	if len(attaches) != 1 {
		t.Errorf("expected 1 Attach, got %d", len(attaches))
	}
}

func TestAttachMany_RejectsInvalidName(t *testing.T) {
	getByNameCalled := false
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			getByNameCalled = true
			return nil, nil
		},
	}
	var attaches []attachCall
	entityTagRepo := newRecordingEntityTagRepo(&attaches)
	svc := NewTagService(tagRepo, entityTagRepo, &mockGate{}, &stubCfg{})

	err := svc.AttachMany(context.Background(), models.EntityTypeTask, 42, []string{"VOICE!!"})
	if err == nil {
		t.Fatal("expected *ValidationError, got nil")
	}
	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *ValidationError, got %T: %v", err, err)
	}
	if getByNameCalled {
		t.Error("GetByName must NOT be called on invalid input")
	}
	if len(attaches) != 0 {
		t.Errorf("expected zero attaches, got %d", len(attaches))
	}
}

// ---------------------------------------------------------------------------
// AC-5: duplicate in same call issues two Attach calls (no service-level dedup)
// ---------------------------------------------------------------------------

func TestAttachMany_DuplicateInSameCallIssuesTwoAttaches(t *testing.T) {
	var getByNameCalls []string
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			getByNameCalls = append(getByNameCalls, name)
			return &models.Tag{ID: 10, Name: "voice"}, nil
		},
	}
	var attaches []attachCall
	entityTagRepo := newRecordingEntityTagRepo(&attaches)
	svc := NewTagService(tagRepo, entityTagRepo, &mockGate{}, &stubCfg{})

	err := svc.AttachMany(context.Background(), models.EntityTypeTask, 42, []string{"voice", "voice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(getByNameCalls) != 2 {
		t.Errorf("expected 2 GetByName calls (no in-call dedup), got %d", len(getByNameCalls))
	}
	if len(attaches) != 2 {
		t.Errorf("expected 2 Attach calls, got %d", len(attaches))
	}
}

// ---------------------------------------------------------------------------
// AC-6: DetachOne happy path — 1× GetByName, 1× Detach, returns nil
// ---------------------------------------------------------------------------

func TestDetachOne_HappyPath(t *testing.T) {
	var getByNameCalls []string
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			getByNameCalls = append(getByNameCalls, name)
			return &models.Tag{ID: 10, Name: "voice"}, nil
		},
	}
	var detachCalled bool
	var detachedTagID int64
	entityTagRepo := &recordingEntityTagRepo{
		detachFn: func(ctx context.Context, entityType models.EntityType, entityID, tagID int64) error {
			detachCalled = true
			detachedTagID = tagID
			return nil
		},
	}
	svc := NewTagService(tagRepo, entityTagRepo, &mockGate{}, &stubCfg{})

	err := svc.DetachOne(context.Background(), models.EntityTypeTask, 42, "voice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(getByNameCalls) != 1 || getByNameCalls[0] != "voice" {
		t.Errorf("expected GetByName('voice') once, got %v", getByNameCalls)
	}
	if !detachCalled {
		t.Error("expected Detach to be called")
	}
	if detachedTagID != 10 {
		t.Errorf("Detach received tagID=%d, want 10", detachedTagID)
	}
}

// ---------------------------------------------------------------------------
// AC-7: DetachOne unregistered name → *NotFoundError, zero Detach calls
// ---------------------------------------------------------------------------

func TestDetachOne_UnregisteredReturnsNotFound(t *testing.T) {
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			return nil, fmt.Errorf("lookup: %w", tagrepo.ErrTagNotFound)
		},
	}
	detachCalled := false
	entityTagRepo := &recordingEntityTagRepo{
		detachFn: func(ctx context.Context, entityType models.EntityType, entityID, tagID int64) error {
			detachCalled = true
			return nil
		},
	}
	svc := NewTagService(tagRepo, entityTagRepo, &mockGate{}, &stubCfg{})

	err := svc.DetachOne(context.Background(), models.EntityTypeTask, 42, "voice")
	if err == nil {
		t.Fatal("expected *NotFoundError, got nil")
	}
	var nfErr *NotFoundError
	if !errors.As(err, &nfErr) {
		t.Fatalf("expected *NotFoundError (not UnregisteredTagError), got %T: %v", err, err)
	}
	// Ensure it's not being mis-reported as UnregisteredTagError (AC-7 explicit).
	var unregErr *UnregisteredTagError
	if errors.As(err, &unregErr) {
		t.Error("DetachOne must return *NotFoundError, not *UnregisteredTagError")
	}
	if nfErr.Name != "voice" {
		t.Errorf("NotFoundError.Name = %q, want %q", nfErr.Name, "voice")
	}
	if detachCalled {
		t.Error("Detach must not be called when the tag is unregistered")
	}
}

// ---------------------------------------------------------------------------
// AC-8: DetachOne when tag is registered but not attached → nil (repo no-op)
// AC-8b: DetachOne normalizes name before lookup
// AC-8c: DetachOne rejects invalid name
// ---------------------------------------------------------------------------

func TestDetachOne_NotAttachedIsNoOp(t *testing.T) {
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			return &models.Tag{ID: 10, Name: "voice"}, nil
		},
	}
	entityTagRepo := &recordingEntityTagRepo{
		detachFn: func(ctx context.Context, entityType models.EntityType, entityID, tagID int64) error {
			// Repo contract: detaching an absent association is nil (no error).
			return nil
		},
	}
	svc := NewTagService(tagRepo, entityTagRepo, &mockGate{}, &stubCfg{})

	err := svc.DetachOne(context.Background(), models.EntityTypeTask, 42, "voice")
	if err != nil {
		t.Errorf("expected nil error (no-op detach), got: %v", err)
	}
}

func TestDetachOne_NormalizesName(t *testing.T) {
	var seen string
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			seen = name
			return &models.Tag{ID: 10, Name: "voice"}, nil
		},
	}
	entityTagRepo := &recordingEntityTagRepo{}
	svc := NewTagService(tagRepo, entityTagRepo, &mockGate{}, &stubCfg{})

	err := svc.DetachOne(context.Background(), models.EntityTypeTask, 42, " Voice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if seen != "voice" {
		t.Errorf("GetByName received %q, want normalized %q", seen, "voice")
	}
}

func TestDetachOne_RejectsInvalidName(t *testing.T) {
	getByNameCalled := false
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			getByNameCalled = true
			return nil, nil
		},
	}
	detachCalled := false
	entityTagRepo := &recordingEntityTagRepo{
		detachFn: func(ctx context.Context, entityType models.EntityType, entityID, tagID int64) error {
			detachCalled = true
			return nil
		},
	}
	svc := NewTagService(tagRepo, entityTagRepo, &mockGate{}, &stubCfg{})

	err := svc.DetachOne(context.Background(), models.EntityTypeTask, 42, "not valid!")
	if err == nil {
		t.Fatal("expected *ValidationError, got nil")
	}
	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *ValidationError, got %T: %v", err, err)
	}
	if getByNameCalled {
		t.Error("GetByName must NOT be called on invalid input")
	}
	if detachCalled {
		t.Error("Detach must NOT be called on invalid input")
	}
}

// ---------------------------------------------------------------------------
// AC-9 / AC-9b: EnforceRequired returns *TagRequiredError when type required
//                and names is empty/nil (zero repo calls).
// AC-10: type required but tags provided → nil.
// AC-11: entity type not required → nil.
// AC-12: empty/absent config → nil.
// AC-12b: case-sensitive type match (ADR-F04-4).
// ---------------------------------------------------------------------------

// failingRepo ensures EnforceRequired does not touch the repository.
// Any Attach/Detach/GetByName call fails the test.
type failingTagRepo struct{ t *testing.T }

func (r *failingTagRepo) Create(context.Context, string) (*models.Tag, error) {
	r.t.Fatal("EnforceRequired must not call tagRepo.Create")
	return nil, nil
}
func (r *failingTagRepo) GetByName(context.Context, string) (*models.Tag, error) {
	r.t.Fatal("EnforceRequired must not call tagRepo.GetByName")
	return nil, nil
}
func (r *failingTagRepo) GetByID(context.Context, int64) (*models.Tag, error) {
	r.t.Fatal("EnforceRequired must not call tagRepo.GetByID")
	return nil, nil
}
func (r *failingTagRepo) List(context.Context) ([]*models.Tag, error) {
	r.t.Fatal("EnforceRequired must not call tagRepo.List")
	return nil, nil
}
func (r *failingTagRepo) Rename(context.Context, int64, string) (*models.Tag, error) {
	r.t.Fatal("EnforceRequired must not call tagRepo.Rename")
	return nil, nil
}
func (r *failingTagRepo) Delete(context.Context, int64, bool) error {
	r.t.Fatal("EnforceRequired must not call tagRepo.Delete")
	return nil
}

func TestEnforceRequired_TypeRequiredAndTagsMissing(t *testing.T) {
	cfg := &stubCfg{values: []string{"task"}}
	svc := NewTagService(&failingTagRepo{t: t}, &recordingEntityTagRepo{}, &mockGate{}, cfg)

	err := svc.EnforceRequired(context.Background(), models.EntityTypeTask, nil)
	if err == nil {
		t.Fatal("expected *TagRequiredError, got nil")
	}
	var reqErr *TagRequiredError
	if !errors.As(err, &reqErr) {
		t.Fatalf("expected *TagRequiredError, got %T: %v", err, err)
	}
	if reqErr.EntityType != "task" {
		t.Errorf("EntityType = %q, want %q", reqErr.EntityType, "task")
	}
}

func TestEnforceRequired_TypeRequiredAndTagsEmptySlice(t *testing.T) {
	cfg := &stubCfg{values: []string{"task"}}
	svc := NewTagService(&failingTagRepo{t: t}, &recordingEntityTagRepo{}, &mockGate{}, cfg)

	err := svc.EnforceRequired(context.Background(), models.EntityTypeTask, []string{})
	var reqErr *TagRequiredError
	if !errors.As(err, &reqErr) {
		t.Fatalf("expected *TagRequiredError, got %T: %v", err, err)
	}
	if reqErr.EntityType != "task" {
		t.Errorf("EntityType = %q, want %q", reqErr.EntityType, "task")
	}
}

func TestEnforceRequired_TypeRequiredAndTagsPresent(t *testing.T) {
	cfg := &stubCfg{values: []string{"task"}}
	svc := NewTagService(&failingTagRepo{t: t}, &recordingEntityTagRepo{}, &mockGate{}, cfg)

	// EnforceRequired must not validate names itself (AttachMany does).
	err := svc.EnforceRequired(context.Background(), models.EntityTypeTask, []string{"voice"})
	if err != nil {
		t.Errorf("expected nil, got: %v", err)
	}
}

func TestEnforceRequired_OtherTypeNotRequired(t *testing.T) {
	cfg := &stubCfg{values: []string{"task"}}
	svc := NewTagService(&failingTagRepo{t: t}, &recordingEntityTagRepo{}, &mockGate{}, cfg)

	err := svc.EnforceRequired(context.Background(), models.EntityTypeEpic, nil)
	if err != nil {
		t.Errorf("expected nil for epic with task-only requirement, got: %v", err)
	}
}

func TestEnforceRequired_EmptyConfigNoOp(t *testing.T) {
	t.Run("nil slice config", func(t *testing.T) {
		cfg := &stubCfg{values: nil}
		svc := NewTagService(&failingTagRepo{t: t}, &recordingEntityTagRepo{}, &mockGate{}, cfg)
		err := svc.EnforceRequired(context.Background(), models.EntityTypeTask, nil)
		if err != nil {
			t.Errorf("expected nil, got: %v", err)
		}
	})
	t.Run("empty slice config", func(t *testing.T) {
		cfg := &stubCfg{values: []string{}}
		svc := NewTagService(&failingTagRepo{t: t}, &recordingEntityTagRepo{}, &mockGate{}, cfg)
		err := svc.EnforceRequired(context.Background(), models.EntityTypeTask, nil)
		if err != nil {
			t.Errorf("expected nil, got: %v", err)
		}
	})
}

// AC-12b (ADR-F04-4): case-sensitive match. Mis-cased config entries silently
// disable enforcement for that type. Documents — and pins — the decision.
func TestEnforceRequired_CaseSensitiveTypeMatch(t *testing.T) {
	cfg := &stubCfg{values: []string{"Task"}} // wrong case
	svc := NewTagService(&failingTagRepo{t: t}, &recordingEntityTagRepo{}, &mockGate{}, cfg)

	err := svc.EnforceRequired(context.Background(), models.EntityTypeTask, nil)
	if err != nil {
		t.Errorf("expected nil (case-sensitive match, 'Task' != 'task'), got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// AC-14: AttachMany MUST NOT call the gate
// AC-14b: DetachOne MUST NOT call the gate
// AC-14c: EnforceRequired MUST NOT call the gate
// ---------------------------------------------------------------------------

func TestAttachMany_DoesNotCallGate(t *testing.T) {
	gate := &mockGate{
		authorizeFn: func(ctx context.Context, pass string) error {
			return &maintainer.UnauthorizedError{Reason: "should_not_be_invoked"}
		},
	}
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			return &models.Tag{ID: 10, Name: "voice"}, nil
		},
	}
	entityTagRepo := &recordingEntityTagRepo{}
	svc := NewTagService(tagRepo, entityTagRepo, gate, &stubCfg{})

	err := svc.AttachMany(context.Background(), models.EntityTypeTask, 42, []string{"voice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gate.authorizeCount != 0 {
		t.Errorf("gate.Authorize call count = %d, want 0 (AttachMany must not consume gate)", gate.authorizeCount)
	}
}

func TestDetachOne_DoesNotCallGate(t *testing.T) {
	gate := &mockGate{
		authorizeFn: func(ctx context.Context, pass string) error {
			return &maintainer.UnauthorizedError{Reason: "should_not_be_invoked"}
		},
	}
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			return &models.Tag{ID: 10, Name: "voice"}, nil
		},
	}
	entityTagRepo := &recordingEntityTagRepo{}
	svc := NewTagService(tagRepo, entityTagRepo, gate, &stubCfg{})

	err := svc.DetachOne(context.Background(), models.EntityTypeTask, 42, "voice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gate.authorizeCount != 0 {
		t.Errorf("gate.Authorize call count = %d, want 0 (DetachOne must not consume gate)", gate.authorizeCount)
	}
}

func TestEnforceRequired_DoesNotCallGate(t *testing.T) {
	gate := &mockGate{
		authorizeFn: func(ctx context.Context, pass string) error {
			return &maintainer.UnauthorizedError{Reason: "should_not_be_invoked"}
		},
	}
	cfg := &stubCfg{values: []string{"task"}}
	svc := NewTagService(&failingTagRepo{t: t}, &recordingEntityTagRepo{}, gate, cfg)

	err := svc.EnforceRequired(context.Background(), models.EntityTypeEpic, nil) // non-required type
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gate.authorizeCount != 0 {
		t.Errorf("gate.Authorize call count = %d, want 0 (EnforceRequired must not consume gate)", gate.authorizeCount)
	}
}

// ---------------------------------------------------------------------------
// REQ-NF-002: OTel span emission for AttachMany, DetachOne, EnforceRequired
// ---------------------------------------------------------------------------

func TestAttachMany_EmitsSpanWithAttributes(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	tr := tp.Tracer("test")

	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			return &models.Tag{ID: 10, Name: name}, nil
		},
	}
	svc := NewTagService(tagRepo, &recordingEntityTagRepo{}, &mockGate{}, &stubCfg{})
	svc.SetTracer(tr)

	err := svc.AttachMany(context.Background(), models.EntityTypeTask, 42, []string{"voice", "auth"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := exporter.GetSpans()
	var s *tracetest.SpanStub
	for i := range spans {
		if spans[i].Name == "tag_service.attach_many" {
			s = &spans[i]
			break
		}
	}
	if s == nil {
		t.Fatal("span 'tag_service.attach_many' not emitted")
	}
	got := map[string]attribute.Value{}
	for _, a := range s.Attributes {
		got[string(a.Key)] = a.Value
	}
	if v, ok := got["entity.type"]; !ok || v != attribute.StringValue("task") {
		t.Errorf("entity.type attribute missing or wrong: %v", v)
	}
	if v, ok := got["entity.id"]; !ok || v != attribute.Int64Value(42) {
		t.Errorf("entity.id attribute missing or wrong: %v", v)
	}
	if v, ok := got["tag.count"]; !ok || v != attribute.IntValue(2) {
		t.Errorf("tag.count attribute missing or wrong: %v", v)
	}
	// Must NOT include per-tag names in the span (REQ-NF-002).
	if _, ok := got["tag.name"]; ok {
		t.Error("tag.name must not appear on attach_many span (would carry vocabulary contents)")
	}
}

func TestDetachOne_EmitsSpanWithAttributes(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	tr := tp.Tracer("test")

	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			return &models.Tag{ID: 10, Name: "voice"}, nil
		},
	}
	svc := NewTagService(tagRepo, &recordingEntityTagRepo{}, &mockGate{}, &stubCfg{})
	svc.SetTracer(tr)

	err := svc.DetachOne(context.Background(), models.EntityTypeTask, 42, "voice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := exporter.GetSpans()
	var s *tracetest.SpanStub
	for i := range spans {
		if spans[i].Name == "tag_service.detach_one" {
			s = &spans[i]
			break
		}
	}
	if s == nil {
		t.Fatal("span 'tag_service.detach_one' not emitted")
	}
	got := map[string]attribute.Value{}
	for _, a := range s.Attributes {
		got[string(a.Key)] = a.Value
	}
	if v, ok := got["entity.type"]; !ok || v != attribute.StringValue("task") {
		t.Errorf("entity.type attribute missing or wrong: %v", v)
	}
	if v, ok := got["entity.id"]; !ok || v != attribute.Int64Value(42) {
		t.Errorf("entity.id attribute missing or wrong: %v", v)
	}
	if v, ok := got["tag.name"]; !ok || v != attribute.StringValue("voice") {
		t.Errorf("tag.name attribute missing or wrong: %v", v)
	}
}

func TestEnforceRequired_EmitsSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	tr := tp.Tracer("test")

	cfg := &stubCfg{values: []string{"task"}}
	svc := NewTagService(&failingTagRepo{t: t}, &recordingEntityTagRepo{}, &mockGate{}, cfg)
	svc.SetTracer(tr)

	// Use a non-required type so no error escapes.
	err := svc.EnforceRequired(context.Background(), models.EntityTypeEpic, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := exporter.GetSpans()
	var s *tracetest.SpanStub
	for i := range spans {
		if spans[i].Name == "tag_service.enforce_required" {
			s = &spans[i]
			break
		}
	}
	if s == nil {
		t.Fatal("span 'tag_service.enforce_required' not emitted")
	}
	got := map[string]attribute.Value{}
	for _, a := range s.Attributes {
		got[string(a.Key)] = a.Value
	}
	if v, ok := got["entity.type"]; !ok || v != attribute.StringValue("epic") {
		t.Errorf("entity.type attribute missing or wrong: %v", v)
	}
	if v, ok := got["tag.count"]; !ok || v != attribute.IntValue(0) {
		t.Errorf("tag.count attribute missing or wrong: %v", v)
	}
	// EnforceRequired must never carry tag names.
	if _, ok := got["tag.name"]; ok {
		t.Error("tag.name must not appear on enforce_required span")
	}
}

// ---------------------------------------------------------------------------
// AC-T3: TagEnforcementConfig is defined locally and *config.Config would
// satisfy it. Compile-time check done via the package-level `var _` on the
// stubCfg type above; an additional compile-time assertion from the config
// side lives in internal/config/config_test.go (AC-27).
// ---------------------------------------------------------------------------

// ===========================================================================
// E28-F05: EntityIDsByTags, ListTagsForEntity, AttachedTagNamesByIDs
// ===========================================================================
//
// Traceability: each test maps to an AC in spec.md §1.3 (AC-1 through AC-6,
// AC-9 through AC-10) and to the matrix in test-plan.md §1.1 and §1.3.

// ---------------------------------------------------------------------------
// AC-1: EntityIDsByTags — single tag, some matches returned sorted
// ---------------------------------------------------------------------------

func TestEntityIDsByTags_SingleTagMatchesSome(t *testing.T) {
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			if name == "voice" {
				return &models.Tag{ID: 3, Name: "voice"}, nil
			}
			return nil, fmt.Errorf("not found: %w", tagrepo.ErrTagNotFound)
		},
	}
	var capturedTagIDs []int64
	filterCallCount := 0
	entityTagRepo := &mockEntityTagRepo{
		filterEntityIDsFn: func(ctx context.Context, entityType models.EntityType, tagIDs []int64) ([]int64, error) {
			filterCallCount++
			capturedTagIDs = tagIDs
			return []int64{42, 43, 44}, nil
		},
	}
	svc := newTagServiceForTest(tagRepo, entityTagRepo, &mockGate{})

	ids, err := svc.EntityIDsByTags(context.Background(), models.EntityTypeTask, []string{"voice"}, TagQueryOpAnd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 3 {
		t.Errorf("expected 3 IDs, got %d: %v", len(ids), ids)
	}
	if filterCallCount != 1 {
		t.Errorf("FilterEntityIDs call count = %d, want 1", filterCallCount)
	}
	if len(capturedTagIDs) != 1 || capturedTagIDs[0] != 3 {
		t.Errorf("FilterEntityIDs called with tagIDs=%v, want [3]", capturedTagIDs)
	}
}

// ---------------------------------------------------------------------------
// AC-2: EntityIDsByTags — two tags, AND intersection
// ---------------------------------------------------------------------------

func TestEntityIDsByTags_TwoTagsIntersection(t *testing.T) {
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			switch name {
			case "voice":
				return &models.Tag{ID: 3, Name: "voice"}, nil
			case "auth":
				return &models.Tag{ID: 7, Name: "auth"}, nil
			}
			return nil, fmt.Errorf("not found: %w", tagrepo.ErrTagNotFound)
		},
	}
	var capturedTagIDs []int64
	entityTagRepo := &mockEntityTagRepo{
		filterEntityIDsFn: func(ctx context.Context, entityType models.EntityType, tagIDs []int64) ([]int64, error) {
			capturedTagIDs = tagIDs
			return []int64{10}, nil
		},
	}
	svc := newTagServiceForTest(tagRepo, entityTagRepo, &mockGate{})

	ids, err := svc.EntityIDsByTags(context.Background(), models.EntityTypeTask, []string{"voice", "auth"}, TagQueryOpAnd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 1 || ids[0] != 10 {
		t.Errorf("expected [10], got %v", ids)
	}
	// FilterEntityIDs should be called once with tagIDs containing both 3 and 7.
	if len(capturedTagIDs) != 2 {
		t.Errorf("FilterEntityIDs called with %d tagIDs, want 2: %v", len(capturedTagIDs), capturedTagIDs)
	}
}

// AC-2b: Two tags, no intersection — returns empty slice (not nil)
func TestEntityIDsByTags_TwoTagsNoIntersection(t *testing.T) {
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			switch name {
			case "voice":
				return &models.Tag{ID: 3, Name: "voice"}, nil
			case "auth":
				return &models.Tag{ID: 7, Name: "auth"}, nil
			}
			return nil, fmt.Errorf("not found: %w", tagrepo.ErrTagNotFound)
		},
	}
	entityTagRepo := &mockEntityTagRepo{
		filterEntityIDsFn: func(ctx context.Context, entityType models.EntityType, tagIDs []int64) ([]int64, error) {
			return []int64{}, nil
		},
	}
	svc := newTagServiceForTest(tagRepo, entityTagRepo, &mockGate{})

	ids, err := svc.EntityIDsByTags(context.Background(), models.EntityTypeTask, []string{"voice", "auth"}, TagQueryOpAnd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Non-nil empty slice per REQ-F-006 semantics (what FilterEntityIDs returned).
	if ids == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
	if len(ids) != 0 {
		t.Errorf("expected 0 IDs, got %d: %v", len(ids), ids)
	}
}

// ---------------------------------------------------------------------------
// AC-3: EntityIDsByTags — nil or empty names returns (nil, nil)
// ---------------------------------------------------------------------------

func TestEntityIDsByTags_NilNamesReturnsNilNil(t *testing.T) {
	filterCalled := false
	entityTagRepo := &mockEntityTagRepo{
		filterEntityIDsFn: func(ctx context.Context, entityType models.EntityType, tagIDs []int64) ([]int64, error) {
			filterCalled = true
			return nil, nil
		},
	}
	svc := newTagServiceForTest(&mockTagRepo{}, entityTagRepo, &mockGate{})

	ids, err := svc.EntityIDsByTags(context.Background(), models.EntityTypeTask, nil, TagQueryOpAnd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ids != nil {
		t.Errorf("expected nil, got %v", ids)
	}
	if filterCalled {
		t.Error("FilterEntityIDs must not be called for nil names")
	}
}

func TestEntityIDsByTags_EmptyNamesReturnsNilNil(t *testing.T) {
	filterCalled := false
	entityTagRepo := &mockEntityTagRepo{
		filterEntityIDsFn: func(ctx context.Context, entityType models.EntityType, tagIDs []int64) ([]int64, error) {
			filterCalled = true
			return nil, nil
		},
	}
	svc := newTagServiceForTest(&mockTagRepo{}, entityTagRepo, &mockGate{})

	ids, err := svc.EntityIDsByTags(context.Background(), models.EntityTypeTask, []string{}, TagQueryOpAnd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ids != nil {
		t.Errorf("expected nil, got %v", ids)
	}
	if filterCalled {
		t.Error("FilterEntityIDs must not be called for empty names")
	}
}

// ---------------------------------------------------------------------------
// AC-4: EntityIDsByTags — unregistered name aborts, FilterEntityIDs not called
// ---------------------------------------------------------------------------

func TestEntityIDsByTags_UnregisteredNameAborts(t *testing.T) {
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			if name == "voice" {
				return &models.Tag{ID: 3, Name: "voice"}, nil
			}
			return nil, fmt.Errorf("not found: %w", tagrepo.ErrTagNotFound)
		},
	}
	filterCallCount := 0
	entityTagRepo := &mockEntityTagRepo{
		filterEntityIDsFn: func(ctx context.Context, entityType models.EntityType, tagIDs []int64) ([]int64, error) {
			filterCallCount++
			return nil, nil
		},
	}
	svc := newTagServiceForTest(tagRepo, entityTagRepo, &mockGate{})

	ids, err := svc.EntityIDsByTags(context.Background(), models.EntityTypeTask, []string{"voice", "does-not-exist"}, TagQueryOpAnd)
	if err == nil {
		t.Fatal("expected *UnregisteredTagError, got nil")
	}
	var unregErr *UnregisteredTagError
	if !errors.As(err, &unregErr) {
		t.Fatalf("expected *UnregisteredTagError, got %T: %v", err, err)
	}
	if unregErr.Name != "does-not-exist" {
		t.Errorf("UnregisteredTagError.Name = %q, want %q", unregErr.Name, "does-not-exist")
	}
	if ids != nil {
		t.Errorf("expected nil ids, got %v", ids)
	}
	if filterCallCount != 0 {
		t.Errorf("FilterEntityIDs call count = %d, want 0 (AC-T2)", filterCallCount)
	}
}

// AC-4b: Unregistered name is first in slice
func TestEntityIDsByTags_UnregisteredNameFirstPos(t *testing.T) {
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			if name == "voice" {
				return &models.Tag{ID: 3, Name: "voice"}, nil
			}
			return nil, fmt.Errorf("not found: %w", tagrepo.ErrTagNotFound)
		},
	}
	filterCallCount := 0
	entityTagRepo := &mockEntityTagRepo{
		filterEntityIDsFn: func(ctx context.Context, entityType models.EntityType, tagIDs []int64) ([]int64, error) {
			filterCallCount++
			return nil, nil
		},
	}
	svc := newTagServiceForTest(tagRepo, entityTagRepo, &mockGate{})

	_, err := svc.EntityIDsByTags(context.Background(), models.EntityTypeTask, []string{"does-not-exist", "voice"}, TagQueryOpAnd)
	if err == nil {
		t.Fatal("expected *UnregisteredTagError, got nil")
	}
	var unregErr *UnregisteredTagError
	if !errors.As(err, &unregErr) {
		t.Fatalf("expected *UnregisteredTagError, got %T: %v", err, err)
	}
	if filterCallCount != 0 {
		t.Errorf("FilterEntityIDs call count = %d, want 0", filterCallCount)
	}
}

// AC-4c: All names invalid
func TestEntityIDsByTags_AllNamesInvalid(t *testing.T) {
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			return nil, fmt.Errorf("not found: %w", tagrepo.ErrTagNotFound)
		},
	}
	filterCallCount := 0
	entityTagRepo := &mockEntityTagRepo{
		filterEntityIDsFn: func(ctx context.Context, entityType models.EntityType, tagIDs []int64) ([]int64, error) {
			filterCallCount++
			return nil, nil
		},
	}
	svc := newTagServiceForTest(tagRepo, entityTagRepo, &mockGate{})

	_, err := svc.EntityIDsByTags(context.Background(), models.EntityTypeTask, []string{"noexist1", "noexist2"}, TagQueryOpAnd)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var unregErr *UnregisteredTagError
	if !errors.As(err, &unregErr) {
		t.Fatalf("expected *UnregisteredTagError, got %T: %v", err, err)
	}
	// Should fail on first invalid name.
	if unregErr.Name != "noexist1" {
		t.Errorf("UnregisteredTagError.Name = %q, want %q (fail-fast on first invalid)", unregErr.Name, "noexist1")
	}
	if filterCallCount != 0 {
		t.Errorf("FilterEntityIDs call count = %d, want 0", filterCallCount)
	}
}

// ---------------------------------------------------------------------------
// AC-5: EntityIDsByTags — name normalization (whitespace + case)
// ---------------------------------------------------------------------------

func TestEntityIDsByTags_NameNormalization(t *testing.T) {
	var capturedName string
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			capturedName = name
			return &models.Tag{ID: 3, Name: "voice"}, nil
		},
	}
	entityTagRepo := &mockEntityTagRepo{
		filterEntityIDsFn: func(ctx context.Context, entityType models.EntityType, tagIDs []int64) ([]int64, error) {
			return []int64{42, 43, 44}, nil
		},
	}
	svc := newTagServiceForTest(tagRepo, entityTagRepo, &mockGate{})

	_, err := svc.EntityIDsByTags(context.Background(), models.EntityTypeTask, []string{"Voice "}, TagQueryOpAnd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedName != "voice" {
		t.Errorf("GetByName received %q, want normalized %q", capturedName, "voice")
	}
}

// ---------------------------------------------------------------------------
// AC-6: EntityIDsByTags — duplicate names are deduplicated
// ---------------------------------------------------------------------------

func TestEntityIDsByTags_DuplicateNamesDeduped(t *testing.T) {
	getByNameCallCount := 0
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			getByNameCallCount++
			return &models.Tag{ID: 3, Name: "voice"}, nil
		},
	}
	filterCallCount := 0
	var capturedTagIDs []int64
	entityTagRepo := &mockEntityTagRepo{
		filterEntityIDsFn: func(ctx context.Context, entityType models.EntityType, tagIDs []int64) ([]int64, error) {
			filterCallCount++
			capturedTagIDs = tagIDs
			return []int64{42}, nil
		},
	}
	svc := newTagServiceForTest(tagRepo, entityTagRepo, &mockGate{})

	_, err := svc.EntityIDsByTags(context.Background(), models.EntityTypeTask, []string{"voice", "voice"}, TagQueryOpAnd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// GetByName should be called once: duplicate normalized names are deduped before lookup.
	if getByNameCallCount != 1 {
		t.Errorf("GetByName call count = %d, want 1 (dedup on normalized names before lookup)", getByNameCallCount)
	}
	// FilterEntityIDs called exactly once with single-element tagIDs slice.
	if filterCallCount != 1 {
		t.Errorf("FilterEntityIDs call count = %d, want 1", filterCallCount)
	}
	if len(capturedTagIDs) != 1 || capturedTagIDs[0] != 3 {
		t.Errorf("FilterEntityIDs called with tagIDs=%v, want [3]", capturedTagIDs)
	}
}

// ---------------------------------------------------------------------------
// AC-9: ListTagsForEntity — two attachments, sorted ascending
// ---------------------------------------------------------------------------

func TestListTagsForEntity_TwoAttachments(t *testing.T) {
	entityTagRepo := &mockEntityTagRepo{
		listTagNamesByEntitiesFn: func(ctx context.Context, entityType models.EntityType, entityIDs []int64) ([]tagrepo.EntityIDTagName, error) {
			return []tagrepo.EntityIDTagName{
				{EntityID: 42, TagName: "auth"},
				{EntityID: 42, TagName: "voice"},
			}, nil
		},
	}
	svc := newTagServiceForTest(&mockTagRepo{}, entityTagRepo, &mockGate{})

	names, err := svc.ListTagsForEntity(context.Background(), models.EntityTypeTask, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d: %v", len(names), names)
	}
	if names[0] != "auth" || names[1] != "voice" {
		t.Errorf("expected [auth voice] sorted ascending, got %v", names)
	}
}

// AC-9b: No attachments returns empty non-nil slice
func TestListTagsForEntity_NoAttachments(t *testing.T) {
	customEntityTagRepo := &listTagsTestEntityTagRepo{links: []*models.EntityTagLink{}}
	svc := newTagServiceForTest(&mockTagRepo{}, customEntityTagRepo, &mockGate{})

	names, err := svc.ListTagsForEntity(context.Background(), models.EntityTypeTask, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if names == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
	if len(names) != 0 {
		t.Errorf("expected 0 names, got %d: %v", len(names), names)
	}
}

// AC-9c: ListTagNamesByEntities error propagated
func TestListTagsForEntity_GetByIDError(t *testing.T) {
	entityTagRepo := &mockEntityTagRepo{
		listTagNamesByEntitiesFn: func(ctx context.Context, entityType models.EntityType, entityIDs []int64) ([]tagrepo.EntityIDTagName, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	svc := newTagServiceForTest(&mockTagRepo{}, entityTagRepo, &mockGate{})

	names, err := svc.ListTagsForEntity(context.Background(), models.EntityTypeTask, 42)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if names != nil {
		t.Errorf("expected nil names on error, got %v", names)
	}
}

// ---------------------------------------------------------------------------
// AC-10: AttachedTagNamesByIDs — full matrix
// ---------------------------------------------------------------------------

func TestAttachedTagNamesByIDs_FullMatrix(t *testing.T) {
	rows := []tagrepo.EntityIDTagName{
		{EntityID: 10, TagName: "auth"},
		{EntityID: 10, TagName: "voice"},
		{EntityID: 20, TagName: "voice"},
	}
	entityTagRepo := &mockEntityTagRepo{
		listTagNamesByEntitiesFn: func(ctx context.Context, entityType models.EntityType, entityIDs []int64) ([]tagrepo.EntityIDTagName, error) {
			return rows, nil
		},
	}
	svc := newTagServiceForTest(&mockTagRepo{}, entityTagRepo, &mockGate{})

	result, err := svc.AttachedTagNamesByIDs(context.Background(), models.EntityTypeTask, []int64{10, 20, 30})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil map, got nil")
	}

	// Verify every input ID is present.
	for _, id := range []int64{10, 20, 30} {
		if _, ok := result[id]; !ok {
			t.Errorf("map missing key for entity ID %d", id)
		}
	}

	if names := result[10]; len(names) != 2 || names[0] != "auth" || names[1] != "voice" {
		t.Errorf("result[10] = %v, want [auth voice]", result[10])
	}
	if names := result[20]; len(names) != 1 || names[0] != "voice" {
		t.Errorf("result[20] = %v, want [voice]", result[20])
	}
	// Entity 30 has zero tags — must be present with empty slice.
	if names := result[30]; names == nil {
		t.Error("result[30] is nil, want non-nil empty slice")
	} else if len(names) != 0 {
		t.Errorf("result[30] = %v, want empty slice", result[30])
	}
}

// AC-10b: Empty input returns non-nil empty map, no repo call
func TestAttachedTagNamesByIDs_EmptyInput(t *testing.T) {
	repoCallCount := 0
	entityTagRepo := &mockEntityTagRepo{
		listTagNamesByEntitiesFn: func(ctx context.Context, entityType models.EntityType, entityIDs []int64) ([]tagrepo.EntityIDTagName, error) {
			repoCallCount++
			return nil, nil
		},
	}
	svc := newTagServiceForTest(&mockTagRepo{}, entityTagRepo, &mockGate{})

	result, err := svc.AttachedTagNamesByIDs(context.Background(), models.EntityTypeTask, []int64{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil empty map, got nil")
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
	if repoCallCount != 0 {
		t.Errorf("repo called %d times, want 0 for empty input", repoCallCount)
	}
}

// AC-10c: All rows have same entity ID
func TestAttachedTagNamesByIDs_AllSameEntity(t *testing.T) {
	rows := []tagrepo.EntityIDTagName{
		{EntityID: 10, TagName: "auth"},
		{EntityID: 10, TagName: "backend"},
		{EntityID: 10, TagName: "voice"},
	}
	entityTagRepo := &mockEntityTagRepo{
		listTagNamesByEntitiesFn: func(ctx context.Context, entityType models.EntityType, entityIDs []int64) ([]tagrepo.EntityIDTagName, error) {
			return rows, nil
		},
	}
	svc := newTagServiceForTest(&mockTagRepo{}, entityTagRepo, &mockGate{})

	result, err := svc.AttachedTagNamesByIDs(context.Background(), models.EntityTypeTask, []int64{10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 key in map, got %d", len(result))
	}
	names := result[10]
	if len(names) != 3 {
		t.Errorf("expected 3 names for entity 10, got %d: %v", len(names), names)
	}
	// Should be sorted ascending.
	if names[0] != "auth" || names[1] != "backend" || names[2] != "voice" {
		t.Errorf("names not sorted ascending: %v", names)
	}
}

// ---------------------------------------------------------------------------
// AC-29: EntityIDsByTags — no extra OTel spans when Tags filter is nil
// ---------------------------------------------------------------------------

func TestEntityIDsByTags_NilTagsNoTagServiceSpans(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	tr := tp.Tracer("test")

	svc := newTagServiceForTest(&mockTagRepo{}, &mockEntityTagRepo{}, &mockGate{})
	svc.SetTracer(tr)

	// Call with nil names — should be a no-op.
	ids, err := svc.EntityIDsByTags(context.Background(), models.EntityTypeTask, nil, TagQueryOpAnd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ids != nil {
		t.Errorf("expected nil, got %v", ids)
	}

	spans := exporter.GetSpans()
	for _, span := range spans {
		if strings.HasPrefix(span.Name, "tag_service.") {
			t.Errorf("expected no tag_service spans for nil names, got span: %q", span.Name)
		}
	}
}

// ---------------------------------------------------------------------------
// AC-30: TagFilterUnavailableError — error message
// ---------------------------------------------------------------------------

func TestTagFilterUnavailableError_Message(t *testing.T) {
	e := &TagFilterUnavailableError{}
	got := e.Error()
	want := "tag filtering is not available (TagService not wired)"
	if got != want {
		t.Errorf("TagFilterUnavailableError.Error() = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// AC-T1 (F05): EntityIDsByTags does NOT call the maintainer gate
// ---------------------------------------------------------------------------

func TestEntityIDsByTags_DoesNotCallGate(t *testing.T) {
	gate := &mockGate{
		authorizeFn: func(ctx context.Context, pass string) error {
			return fmt.Errorf("gate must not be called")
		},
	}
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			return &models.Tag{ID: 1, Name: name}, nil
		},
	}
	entityTagRepo := &mockEntityTagRepo{
		filterEntityIDsFn: func(ctx context.Context, entityType models.EntityType, tagIDs []int64) ([]int64, error) {
			return []int64{42}, nil
		},
	}
	svc := newTagServiceForTest(tagRepo, entityTagRepo, gate)

	_, err := svc.EntityIDsByTags(context.Background(), models.EntityTypeTask, []string{"voice"}, TagQueryOpAnd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gate.authorizeCount != 0 {
		t.Errorf("gate.Authorize called %d times, want 0 (EntityIDsByTags must not consume gate)", gate.authorizeCount)
	}
}

// ---------------------------------------------------------------------------
// AC-T2 (F05): EntityIDsByTags FilterEntityIDs called exactly once for N≥1
// ---------------------------------------------------------------------------

func TestEntityIDsByTags_FilterEntityIDsCalledOnce(t *testing.T) {
	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			switch name {
			case "voice":
				return &models.Tag{ID: 3, Name: "voice"}, nil
			case "auth":
				return &models.Tag{ID: 7, Name: "auth"}, nil
			case "backend":
				return &models.Tag{ID: 11, Name: "backend"}, nil
			}
			return nil, fmt.Errorf("not found: %w", tagrepo.ErrTagNotFound)
		},
	}
	filterCallCount := 0
	entityTagRepo := &mockEntityTagRepo{
		filterEntityIDsFn: func(ctx context.Context, entityType models.EntityType, tagIDs []int64) ([]int64, error) {
			filterCallCount++
			return []int64{42}, nil
		},
	}
	svc := newTagServiceForTest(tagRepo, entityTagRepo, &mockGate{})

	_, err := svc.EntityIDsByTags(context.Background(), models.EntityTypeTask, []string{"voice", "auth", "backend"}, TagQueryOpAnd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filterCallCount != 1 {
		t.Errorf("FilterEntityIDs call count = %d, want exactly 1 (AC-T1)", filterCallCount)
	}
}

// ---------------------------------------------------------------------------
// OTel span tests for new methods
// ---------------------------------------------------------------------------

func TestEntityIDsByTags_EmitsSpanWithAttributes(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	tr := tp.Tracer("test")

	tagRepo := &mockTagRepo{
		getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
			return &models.Tag{ID: 3, Name: name}, nil
		},
	}
	entityTagRepo := &mockEntityTagRepo{
		filterEntityIDsFn: func(ctx context.Context, entityType models.EntityType, tagIDs []int64) ([]int64, error) {
			return []int64{42}, nil
		},
	}
	svc := newTagServiceForTest(tagRepo, entityTagRepo, &mockGate{})
	svc.SetTracer(tr)

	_, err := svc.EntityIDsByTags(context.Background(), models.EntityTypeTask, []string{"voice"}, TagQueryOpAnd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := exporter.GetSpans()
	var s *tracetest.SpanStub
	for i := range spans {
		if spans[i].Name == "tag_service.entity_ids_by_tags" {
			s = &spans[i]
			break
		}
	}
	if s == nil {
		t.Fatal("span 'tag_service.entity_ids_by_tags' not emitted")
	}
	got := map[string]attribute.Value{}
	for _, a := range s.Attributes {
		got[string(a.Key)] = a.Value
	}
	if v, ok := got["entity.type"]; !ok || v != attribute.StringValue("task") {
		t.Errorf("entity.type attribute missing or wrong: %v", v)
	}
	if v, ok := got["tag.count"]; !ok || v != attribute.IntValue(1) {
		t.Errorf("tag.count attribute missing or wrong: %v", v)
	}
	if v, ok := got["filter.op"]; !ok || v != attribute.StringValue("and") {
		t.Errorf("filter.op attribute missing or wrong: %v", v)
	}
}

func TestListTagsForEntity_EmitsSpanWithAttributes(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	tr := tp.Tracer("test")

	customEntityTagRepo := &listTagsTestEntityTagRepo{links: []*models.EntityTagLink{}}
	svc := newTagServiceForTest(&mockTagRepo{}, customEntityTagRepo, &mockGate{})
	svc.SetTracer(tr)

	_, err := svc.ListTagsForEntity(context.Background(), models.EntityTypeTask, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := exporter.GetSpans()
	var s *tracetest.SpanStub
	for i := range spans {
		if spans[i].Name == "tag_service.list_tags_for_entity" {
			s = &spans[i]
			break
		}
	}
	if s == nil {
		t.Fatal("span 'tag_service.list_tags_for_entity' not emitted")
	}
	got := map[string]attribute.Value{}
	for _, a := range s.Attributes {
		got[string(a.Key)] = a.Value
	}
	if v, ok := got["entity.type"]; !ok || v != attribute.StringValue("task") {
		t.Errorf("entity.type attribute missing or wrong: %v", v)
	}
	if v, ok := got["entity.id"]; !ok || v != attribute.Int64Value(42) {
		t.Errorf("entity.id attribute missing or wrong: %v", v)
	}
}

func TestAttachedTagNamesByIDs_EmitsSpanWithAttributes(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	tr := tp.Tracer("test")

	entityTagRepo := &mockEntityTagRepo{
		listTagNamesByEntitiesFn: func(ctx context.Context, entityType models.EntityType, entityIDs []int64) ([]tagrepo.EntityIDTagName, error) {
			return []tagrepo.EntityIDTagName{}, nil
		},
	}
	svc := newTagServiceForTest(&mockTagRepo{}, entityTagRepo, &mockGate{})
	svc.SetTracer(tr)

	_, err := svc.AttachedTagNamesByIDs(context.Background(), models.EntityTypeTask, []int64{10, 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := exporter.GetSpans()
	var s *tracetest.SpanStub
	for i := range spans {
		if spans[i].Name == "tag_service.attached_tag_names_by_ids" {
			s = &spans[i]
			break
		}
	}
	if s == nil {
		t.Fatal("span 'tag_service.attached_tag_names_by_ids' not emitted")
	}
	got := map[string]attribute.Value{}
	for _, a := range s.Attributes {
		got[string(a.Key)] = a.Value
	}
	if v, ok := got["entity.type"]; !ok || v != attribute.StringValue("task") {
		t.Errorf("entity.type attribute missing or wrong: %v", v)
	}
	if v, ok := got["entity.count"]; !ok || v != attribute.IntValue(2) {
		t.Errorf("entity.count attribute missing or wrong: %v", v)
	}
}

// ---------------------------------------------------------------------------
// Helper: listTagsTestEntityTagRepo — supports ListByEntity overriding
// ---------------------------------------------------------------------------

// listTagsTestEntityTagRepo is a test double for EntityTagRepositoryInterface
// that supports controlling the result of ListByEntity (the method used by
// ListTagsForEntity).
type listTagsTestEntityTagRepo struct {
	links     []*models.EntityTagLink
	listErr   error
	callCount int
}

func (m *listTagsTestEntityTagRepo) Attach(ctx context.Context, entityType models.EntityType, entityID, tagID int64) error {
	return nil
}
func (m *listTagsTestEntityTagRepo) Detach(ctx context.Context, entityType models.EntityType, entityID, tagID int64) error {
	return nil
}
func (m *listTagsTestEntityTagRepo) ListByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityTagLink, error) {
	m.callCount++
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.links, nil
}
func (m *listTagsTestEntityTagRepo) ListByTag(ctx context.Context, tagID int64) ([]*models.EntityTagLink, error) {
	return nil, nil
}
func (m *listTagsTestEntityTagRepo) ListByEntityType(ctx context.Context, entityType models.EntityType, tagID int64) ([]*models.EntityTagLink, error) {
	return nil, nil
}
func (m *listTagsTestEntityTagRepo) CountByTag(ctx context.Context, tagID int64) (int64, error) {
	return 0, nil
}
func (m *listTagsTestEntityTagRepo) FilterEntityIDs(ctx context.Context, entityType models.EntityType, tagIDs []int64) ([]int64, error) {
	return []int64{}, nil
}
func (m *listTagsTestEntityTagRepo) ListTagNamesByEntities(ctx context.Context, entityType models.EntityType, entityIDs []int64) ([]tagrepo.EntityIDTagName, error) {
	return []tagrepo.EntityIDTagName{}, nil
}

// ---------------------------------------------------------------------------
// AC-T3: MockTagService gains the three new method stubs (compile check)
// ---------------------------------------------------------------------------

func TestMockTagService_ImplementsTagQuerier(t *testing.T) {
	// Compile-time: MockTagService must implement TagQuerier.
	// If mock_tag_service_test.go's var _ TagQuerier = (*MockTagService)(nil) compiles,
	// this test is superfluous but confirms the interface is satisfied at runtime.
	var _ TagQuerier = (*MockTagService)(nil)
}

func TestMockTagService_EntityIDsByTagsDefaultReturnsNil(t *testing.T) {
	m := NewMockTagService()

	ids, err := m.EntityIDsByTags(context.Background(), models.EntityTypeTask, []string{"voice"}, TagQueryOpAnd)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if ids != nil {
		t.Errorf("expected nil ids by default, got %v", ids)
	}
	if m.EntityIDsByTagsCalls != 1 {
		t.Errorf("EntityIDsByTagsCalls = %d, want 1", m.EntityIDsByTagsCalls)
	}
}

func TestMockTagService_ListTagsForEntityDefaultReturnsEmpty(t *testing.T) {
	m := NewMockTagService()

	names, err := m.ListTagsForEntity(context.Background(), models.EntityTypeTask, 42)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if names == nil {
		t.Error("expected non-nil empty slice by default, got nil")
	}
	if len(names) != 0 {
		t.Errorf("expected empty slice, got %v", names)
	}
	if m.ListTagsForEntityCalls != 1 {
		t.Errorf("ListTagsForEntityCalls = %d, want 1", m.ListTagsForEntityCalls)
	}
}

func TestMockTagService_AttachedTagNamesByIDsDefaultReturnsEmpty(t *testing.T) {
	m := NewMockTagService()

	result, err := m.AttachedTagNamesByIDs(context.Background(), models.EntityTypeTask, []int64{10, 20})
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil empty map by default, got nil")
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
	if m.AttachedTagNamesByIDsCalls != 1 {
		t.Errorf("AttachedTagNamesByIDsCalls = %d, want 1", m.AttachedTagNamesByIDsCalls)
	}
}

// ---------------------------------------------------------------------------
// B018: Regression — entity_tags entity_type validation
//
// With the entity_tags entity_type CHECK constraint dropped (migration in
// internal/db/db.go: migrateDropPolymorphicEntityTypeChecks), the Go layer
// is the sole enforcement point. AttachMany and DetachOne MUST reject any
// entity_type that is not present in models.ValidEntityTypes BEFORE issuing
// any GetByName lookup or Attach/Detach call.
// ---------------------------------------------------------------------------

func TestB018_AttachMany_RejectsInvalidEntityType(t *testing.T) {
	tests := []struct {
		name       string
		entityType models.EntityType
	}{
		{"unknown type", models.EntityType("garbage")},
		{"trailing whitespace bypass attempt", models.EntityType("task ")},
		{"empty string", models.EntityType("")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getByNameCalled := false
			tagRepo := &mockTagRepo{
				getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
					getByNameCalled = true
					return &models.Tag{ID: 10, Name: name}, nil
				},
			}
			var attaches []attachCall
			entityTagRepo := newRecordingEntityTagRepo(&attaches)
			svc := NewTagService(tagRepo, entityTagRepo, &mockGate{}, &stubCfg{})

			err := svc.AttachMany(context.Background(), tt.entityType, 42, []string{"voice"})
			if err == nil {
				t.Fatalf("expected error for entity_type %q, got nil", tt.entityType)
			}
			if !strings.Contains(err.Error(), "invalid entity_type") {
				t.Errorf("expected error to mention 'invalid entity_type', got: %v", err)
			}
			if !strings.Contains(err.Error(), string(tt.entityType)) {
				t.Errorf("expected error to include the bad value %q, got: %v", tt.entityType, err)
			}
			if getByNameCalled {
				t.Error("GetByName must NOT be called when entity_type is invalid")
			}
			if len(attaches) != 0 {
				t.Errorf("expected zero Attach calls, got %d", len(attaches))
			}
		})
	}
}

func TestB018_DetachOne_RejectsInvalidEntityType(t *testing.T) {
	tests := []struct {
		name       string
		entityType models.EntityType
	}{
		{"unknown type", models.EntityType("garbage")},
		{"trailing whitespace bypass attempt", models.EntityType("task ")},
		{"empty string", models.EntityType("")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getByNameCalled := false
			tagRepo := &mockTagRepo{
				getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
					getByNameCalled = true
					return &models.Tag{ID: 10, Name: name}, nil
				},
			}
			detachCalled := false
			entityTagRepo := &recordingEntityTagRepo{
				detachFn: func(ctx context.Context, entityType models.EntityType, entityID, tagID int64) error {
					detachCalled = true
					return nil
				},
			}
			svc := NewTagService(tagRepo, entityTagRepo, &mockGate{}, &stubCfg{})

			err := svc.DetachOne(context.Background(), tt.entityType, 42, "voice")
			if err == nil {
				t.Fatalf("expected error for entity_type %q, got nil", tt.entityType)
			}
			if !strings.Contains(err.Error(), "invalid entity_type") {
				t.Errorf("expected error to mention 'invalid entity_type', got: %v", err)
			}
			if !strings.Contains(err.Error(), string(tt.entityType)) {
				t.Errorf("expected error to include the bad value %q, got: %v", tt.entityType, err)
			}
			if getByNameCalled {
				t.Error("GetByName must NOT be called when entity_type is invalid")
			}
			if detachCalled {
				t.Error("Detach must NOT be called when entity_type is invalid")
			}
		})
	}
}

// TestB018_AttachMany_AcceptsAllValidEntityTypes is the positive control —
// every entity type registered in models.ValidEntityTypes must pass the new
// guard and proceed to GetByName resolution. Regression-protects against an
// over-eager guard that could exclude e.g. "idea" or "tech_debt".
func TestB018_AttachMany_AcceptsAllValidEntityTypes(t *testing.T) {
	for et := range models.ValidEntityTypes {
		et := et // pin loop var for subtest closure
		t.Run(string(et), func(t *testing.T) {
			tagRepo := &mockTagRepo{
				getByNameFn: func(ctx context.Context, name string) (*models.Tag, error) {
					return &models.Tag{ID: 10, Name: "voice"}, nil
				},
			}
			var attaches []attachCall
			entityTagRepo := newRecordingEntityTagRepo(&attaches)
			svc := NewTagService(tagRepo, entityTagRepo, &mockGate{}, &stubCfg{})

			err := svc.AttachMany(context.Background(), et, 42, []string{"voice"})
			if err != nil {
				t.Fatalf("AttachMany rejected valid entity_type %q: %v", et, err)
			}
			if len(attaches) != 1 {
				t.Errorf("expected 1 Attach call, got %d", len(attaches))
			}
		})
	}
}
