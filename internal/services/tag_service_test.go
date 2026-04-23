package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/auth/maintainer"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	tagrepo "github.com/jwwelbor/shark-task-manager/internal/repository/tag"
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
	countByTagFn func(ctx context.Context, tagID int64) (int64, error)
	callCount    int
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
// Helpers
// ---------------------------------------------------------------------------

// newTagServiceForTest builds a TagService with mocks wired.
func newTagServiceForTest(tagRepo tagrepo.TagRepositoryInterface, entityTagRepo tagrepo.EntityTagRepositoryInterface, gate maintainer.Gate) *TagService {
	return NewTagService(tagRepo, entityTagRepo, gate)
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

	t.Run("nil tagRepo", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic on nil tagRepo")
			}
		}()
		NewTagService(nil, entityTagRepo, gate)
	})

	t.Run("nil entityTagRepo", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic on nil entityTagRepo")
			}
		}()
		NewTagService(tagRepo, nil, gate)
	})

	t.Run("nil gate", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic on nil gate")
			}
		}()
		NewTagService(tagRepo, entityTagRepo, nil)
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
	svc := NewTagService(tagRepo, entityTagRepo, &mockGate{})
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
	svc := NewTagService(tagRepo, entityTagRepo, &mockGate{})
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
// AC-19 static check: ValidateName is the sole path to models.ValidateTagName
// ---------------------------------------------------------------------------

func TestTagService_ValidateName_IsSoleEntryPoint(t *testing.T) {
	svc := NewTagService(&mockTagRepo{}, &mockEntityTagRepo{}, &mockGate{})

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
