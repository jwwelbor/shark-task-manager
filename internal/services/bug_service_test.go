package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// mockBugRepo implements BugRepository for testing.
type mockBugRepo struct {
	createFn          func(ctx context.Context, bug *models.Bug) error
	getByKeyFn        func(ctx context.Context, key string) (*models.Bug, error)
	getByIDFn         func(ctx context.Context, id int64) (*models.Bug, error)
	updateFn          func(ctx context.Context, bug *models.Bug) error
	deleteFn          func(ctx context.Context, id int64) error
	updateStatusFn    func(ctx context.Context, id int64, status models.BugStatus) error
	getNextKeyFn      func(ctx context.Context) (string, error)
	listFn            func(ctx context.Context, filters *repository.BugListFilters) ([]*models.Bug, error)
	countByStatusFn   func(ctx context.Context) (map[string]int, error)
	countBySeverityFn func(ctx context.Context) (map[string]int, error)
}

func (m *mockBugRepo) Create(ctx context.Context, bug *models.Bug) error {
	if m.createFn != nil {
		return m.createFn(ctx, bug)
	}
	return nil
}

func (m *mockBugRepo) GetByKey(ctx context.Context, key string) (*models.Bug, error) {
	if m.getByKeyFn != nil {
		return m.getByKeyFn(ctx, key)
	}
	return nil, fmt.Errorf("not found: %s", key)
}

func (m *mockBugRepo) GetByID(ctx context.Context, id int64) (*models.Bug, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, fmt.Errorf("not found: %d", id)
}

func (m *mockBugRepo) Update(ctx context.Context, bug *models.Bug) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, bug)
	}
	return nil
}

func (m *mockBugRepo) Delete(ctx context.Context, id int64) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockBugRepo) UpdateStatus(ctx context.Context, id int64, status models.BugStatus) error {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, id, status)
	}
	return nil
}

func (m *mockBugRepo) GetNextKey(ctx context.Context) (string, error) {
	if m.getNextKeyFn != nil {
		return m.getNextKeyFn(ctx)
	}
	return "B001", nil
}

func (m *mockBugRepo) List(ctx context.Context, filters *repository.BugListFilters) ([]*models.Bug, error) {
	if m.listFn != nil {
		return m.listFn(ctx, filters)
	}
	return []*models.Bug{}, nil
}

func (m *mockBugRepo) CountByStatus(ctx context.Context) (map[string]int, error) {
	if m.countByStatusFn != nil {
		return m.countByStatusFn(ctx)
	}
	return map[string]int{}, nil
}

func (m *mockBugRepo) CountBySeverity(ctx context.Context) (map[string]int, error) {
	if m.countBySeverityFn != nil {
		return m.countBySeverityFn(ctx)
	}
	return map[string]int{}, nil
}

// bugLinkEpicRepo is a stub epic repo for link validation in bug tests.
type bugLinkEpicRepo struct {
	getByKeyFn func(ctx context.Context, key string) (*models.Epic, error)
}

func (m *bugLinkEpicRepo) GetByKey(ctx context.Context, key string) (*models.Epic, error) {
	if m.getByKeyFn != nil {
		return m.getByKeyFn(ctx, key)
	}
	return nil, fmt.Errorf("not found")
}

// Stub out remaining EpicRepository interface methods.
func (m *bugLinkEpicRepo) GetByID(context.Context, int64) (*models.Epic, error) { return nil, nil }
func (m *bugLinkEpicRepo) Create(context.Context, *models.Epic) error           { return nil }
func (m *bugLinkEpicRepo) Update(context.Context, *models.Epic) error           { return nil }
func (m *bugLinkEpicRepo) Delete(context.Context, int64) error                  { return nil }
func (m *bugLinkEpicRepo) List(context.Context, *models.EpicStatus) ([]*models.Epic, error) {
	return nil, nil
}
func (m *bugLinkEpicRepo) GetByFilePath(context.Context, string) (*models.Epic, error) {
	return nil, nil
}
func (m *bugLinkEpicRepo) UpdateFilePath(context.Context, string, *string) error { return nil }
func (m *bugLinkEpicRepo) UpdateKey(context.Context, string, string) error       { return nil }
func (m *bugLinkEpicRepo) GetFeatureProgressDataByEpic(context.Context, int64) ([]repository.FeatureProgressData, error) {
	return nil, nil
}
func (m *bugLinkEpicRepo) GetFeatureStatusBreakdown(context.Context, int64) (map[models.FeatureStatus]int, error) {
	return nil, nil
}
func (m *bugLinkEpicRepo) GetFeatureStatusBreakdownByKey(context.Context, string) (map[models.FeatureStatus]int, error) {
	return nil, nil
}
func (m *bugLinkEpicRepo) UpdateStatus(context.Context, int64, models.EpicStatus) error {
	return nil
}
func (m *bugLinkEpicRepo) GetFeatureStatusRollup(context.Context, int64) (map[string]int, error) {
	return nil, nil
}
func (m *bugLinkEpicRepo) GetTaskStatusRollup(context.Context, int64) (map[string]int, error) {
	return nil, nil
}
func (m *bugLinkEpicRepo) CascadeStatusToFeaturesAndTasks(context.Context, int64, models.FeatureStatus, models.TaskStatus) error {
	return nil
}
func (m *bugLinkEpicRepo) GetEpicDisplayDataRaw(context.Context, int64) (*repository.EpicDisplayDataRaw, error) {
	return nil, nil
}

// bugLinkFeatureRepo is a stub feature repo for link validation in bug tests.
type bugLinkFeatureRepo struct {
	getByKeyFn func(ctx context.Context, key string) (*models.Feature, error)
}

func (m *bugLinkFeatureRepo) GetByKey(ctx context.Context, key string) (*models.Feature, error) {
	if m.getByKeyFn != nil {
		return m.getByKeyFn(ctx, key)
	}
	return nil, fmt.Errorf("not found")
}

// Stub out remaining FeatureRepository interface methods.
func (m *bugLinkFeatureRepo) GetByID(context.Context, int64) (*models.Feature, error) {
	return nil, nil
}
func (m *bugLinkFeatureRepo) Create(context.Context, *models.Feature) error   { return nil }
func (m *bugLinkFeatureRepo) Update(context.Context, *models.Feature) error   { return nil }
func (m *bugLinkFeatureRepo) Delete(context.Context, int64) error             { return nil }
func (m *bugLinkFeatureRepo) List(context.Context) ([]*models.Feature, error) { return nil, nil }
func (m *bugLinkFeatureRepo) ListByEpic(context.Context, int64) ([]*models.Feature, error) {
	return nil, nil
}
func (m *bugLinkFeatureRepo) ListByEpicAndStatus(context.Context, int64, models.FeatureStatus) ([]*models.Feature, error) {
	return nil, nil
}
func (m *bugLinkFeatureRepo) GetByFilePath(context.Context, string) (*models.Feature, error) {
	return nil, nil
}
func (m *bugLinkFeatureRepo) UpdateFilePath(context.Context, string, *string) error { return nil }
func (m *bugLinkFeatureRepo) UpdateKey(context.Context, string, string) error       { return nil }
func (m *bugLinkFeatureRepo) GetTaskStatusBreakdown(context.Context, int64) (map[models.TaskStatus]int, error) {
	return nil, nil
}
func (m *bugLinkFeatureRepo) GetTaskCount(context.Context, int64) (int, error)     { return 0, nil }
func (m *bugLinkFeatureRepo) SetStatusOverride(context.Context, int64, bool) error { return nil }
func (m *bugLinkFeatureRepo) UpdateStatus(context.Context, int64, models.FeatureStatus) error {
	return nil
}
func (m *bugLinkFeatureRepo) UpdateStatusIfNotOverridden(context.Context, int64, models.FeatureStatus) (bool, error) {
	return false, nil
}
func (m *bugLinkFeatureRepo) CascadeStatusToTasks(context.Context, int64, models.TaskStatus) error {
	return nil
}
func (m *bugLinkFeatureRepo) GetFeatureDisplayDataRaw(context.Context, int64) (*repository.FeatureDisplayDataRaw, error) {
	return nil, nil
}

// bugLinkTaskRepo is a stub task repo for link validation in bug tests.
type bugLinkTaskRepo struct {
	getByKeyFn func(ctx context.Context, key string) (*models.Task, error)
}

func (m *bugLinkTaskRepo) GetByKey(ctx context.Context, key string) (*models.Task, error) {
	if m.getByKeyFn != nil {
		return m.getByKeyFn(ctx, key)
	}
	return nil, fmt.Errorf("not found")
}

func newBugWorkflowSvc() *workflow.Service {
	return workflow.NewService("")
}

// mockBugEntityRepo is a minimal EntityRepository adapter for bug tests.
type mockBugEntityRepo struct {
	bugRepo *mockBugRepo
}

func (m *mockBugEntityRepo) GetByKey(ctx context.Context, key string) (models.Entity, error) {
	return m.bugRepo.GetByKey(ctx, key)
}

func (m *mockBugEntityRepo) GetByID(ctx context.Context, id int64) (models.Entity, error) {
	return m.bugRepo.GetByID(ctx, id)
}

func (m *mockBugEntityRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	return m.bugRepo.UpdateStatus(ctx, id, models.BugStatus(status))
}

func (m *mockBugEntityRepo) Update(ctx context.Context, entity models.Entity) error {
	bug, ok := entity.(*models.Bug)
	if !ok {
		return fmt.Errorf("expected *models.Bug, got %T", entity)
	}
	return m.bugRepo.Update(ctx, bug)
}

func (m *mockBugEntityRepo) GetContextData(ctx context.Context, id int64) (*string, error) {
	return nil, nil
}

func (m *mockBugEntityRepo) UpdateContextData(ctx context.Context, id int64, data *string) error {
	return nil
}

func newBugService(repo *mockBugRepo, epicRepo *bugLinkEpicRepo, featureRepo *bugLinkFeatureRepo, taskRepo *bugLinkTaskRepo) *BugService {
	return newBugServiceWithTagSvc(repo, epicRepo, featureRepo, taskRepo, nil)
}

// newBugServiceWithTagSvc is the E28-F04 variant that also wires a
// TagQuerier (typically *MockTagService). Callers that don't exercise
// tag paths should use newBugService, which passes nil.
func newBugServiceWithTagSvc(
	repo *mockBugRepo,
	epicRepo *bugLinkEpicRepo,
	featureRepo *bugLinkFeatureRepo,
	taskRepo *bugLinkTaskRepo,
	tagSvc TagQuerier,
) *BugService {
	wfSvc := newBugWorkflowSvc()
	entitySvc := NewEntityService(wfSvc)
	entityRepo := &mockBugEntityRepo{bugRepo: repo}
	if epicRepo == nil {
		epicRepo = &bugLinkEpicRepo{}
	}
	if featureRepo == nil {
		featureRepo = &bugLinkFeatureRepo{}
	}
	if taskRepo == nil {
		taskRepo = &bugLinkTaskRepo{}
	}
	return NewBugService(repo, entitySvc, entityRepo, epicRepo, featureRepo, taskRepo, "", tagSvc)
}

// --- CreateBug tests ---

func TestBugService_CreateBug(t *testing.T) {
	ctx := context.Background()

	repo := &mockBugRepo{
		getNextKeyFn: func(ctx context.Context) (string, error) {
			return "B001", nil
		},
		createFn: func(ctx context.Context, bug *models.Bug) error {
			if bug.Key != "B001" {
				t.Errorf("expected key B001, got %s", bug.Key)
			}
			if bug.Title != "Login page crashes on submit" {
				t.Errorf("expected title 'Login page crashes on submit', got %s", bug.Title)
			}
			if bug.Severity != models.BugSeverityHigh {
				t.Errorf("expected severity high, got %s", bug.Severity)
			}
			bug.ID = 1
			return nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	bug, _, err := svc.CreateBug(ctx, CreateBugInput{
		Title:    "Login page crashes on submit",
		Severity: models.BugSeverityHigh,
	})
	if err != nil {
		t.Fatalf("CreateBug() error = %v", err)
	}
	if bug == nil {
		t.Fatal("expected bug, got nil")
	}
	if bug.Key != "B001" {
		t.Errorf("expected key B001, got %s", bug.Key)
	}
	if bug.Slug == nil || *bug.Slug == "" {
		t.Error("expected non-empty slug")
	}
}

// TestBugService_CreateBug_BodyHonored verifies that when CreateBugInput.Body
// is supplied, the rendered markdown file's body region contains the supplied
// content (frontmatter remains intact).
func TestBugService_CreateBug_BodyHonored(t *testing.T) {
	ctx := context.Background()

	root := t.TempDir()
	repo := &mockBugRepo{
		getNextKeyFn: func(ctx context.Context) (string, error) {
			return "B042", nil
		},
		createFn: func(ctx context.Context, bug *models.Bug) error {
			bug.ID = 42
			return nil
		},
	}
	wfSvc := newBugWorkflowSvc()
	entitySvc := NewEntityService(wfSvc)
	entityRepo := &mockBugEntityRepo{bugRepo: repo}
	svc := NewBugService(repo, entitySvc, entityRepo, &bugLinkEpicRepo{}, &bugLinkFeatureRepo{}, &bugLinkTaskRepo{}, root, nil)

	customBody := "## Custom Description\n\nThis is what the user piped in."
	bug, _, err := svc.CreateBug(ctx, CreateBugInput{
		Title:    "Body override test",
		Severity: models.BugSeverityMedium,
		Body:     customBody,
	})
	if err != nil {
		t.Fatalf("CreateBug() error = %v", err)
	}
	if bug == nil {
		t.Fatal("expected bug, got nil")
	}

	// Read the file written by the service and assert the body is present.
	raw, err := os.ReadFile(filepath.Join(root, "docs/plan/bugs/B042.md"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	contents := string(raw)
	if !strings.Contains(contents, "## Custom Description") {
		t.Errorf("expected custom body in file, got:\n%s", contents)
	}
	// Frontmatter (bug_key) must still be present.
	if !strings.Contains(contents, "bug_key: B042") {
		t.Errorf("expected frontmatter to be preserved; got:\n%s", contents)
	}
	// Default placeholder text must NOT appear (body fully replaced).
	if strings.Contains(contents, "[Describe the bug and how to reproduce it]") {
		t.Errorf("default placeholder should be replaced; got:\n%s", contents)
	}
}

func TestBugService_CreateBug_EmptyTitle(t *testing.T) {
	ctx := context.Background()
	svc := newBugService(&mockBugRepo{}, nil, nil, nil)

	_, _, err := svc.CreateBug(ctx, CreateBugInput{
		Title:    "",
		Severity: models.BugSeverityHigh,
	})
	if err == nil {
		t.Fatal("expected error for empty title")
	}
	if !strings.Contains(err.Error(), "title") {
		t.Errorf("expected error about title, got: %v", err)
	}
}

func TestBugService_CreateBug_InvalidSeverity(t *testing.T) {
	ctx := context.Background()
	svc := newBugService(&mockBugRepo{}, nil, nil, nil)

	_, _, err := svc.CreateBug(ctx, CreateBugInput{
		Title:    "Some bug",
		Severity: models.BugSeverity("unknown"),
	})
	if err == nil {
		t.Fatal("expected error for invalid severity")
	}
	if !strings.Contains(err.Error(), "severity") {
		t.Errorf("expected error about severity, got: %v", err)
	}
}

// TestBugService_CreateBug_DefaultSeverity_WhenOmitted is the B008 regression
// test: when the caller omits Severity, CreateBug must default it to "medium"
// and succeed (rather than returning `invalid severity ""`). This matches the
// bug-report-then-triage workflow — `bug triage --severity=<S>` exists to set
// severity later once the bug has been investigated.
func TestBugService_CreateBug_DefaultSeverity_WhenOmitted(t *testing.T) {
	ctx := context.Background()

	var capturedSeverity models.BugSeverity
	repo := &mockBugRepo{
		getNextKeyFn: func(ctx context.Context) (string, error) {
			return "B001", nil
		},
		createFn: func(ctx context.Context, bug *models.Bug) error {
			capturedSeverity = bug.Severity
			bug.ID = 1
			return nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	bug, _, err := svc.CreateBug(ctx, CreateBugInput{
		Title: "Unspecified-severity bug",
		// Severity intentionally omitted.
	})
	if err != nil {
		t.Fatalf("CreateBug() with omitted severity error = %v; want nil (B008 regression)", err)
	}
	if bug == nil {
		t.Fatal("expected bug, got nil")
	}
	if bug.Severity != models.BugSeverityMedium {
		t.Errorf("expected default severity %q, got %q", models.BugSeverityMedium, bug.Severity)
	}
	if capturedSeverity != models.BugSeverityMedium {
		t.Errorf("expected repo to receive severity %q, got %q", models.BugSeverityMedium, capturedSeverity)
	}
}

func TestBugService_CreateBug_WithDescription(t *testing.T) {
	ctx := context.Background()

	var capturedBug *models.Bug
	repo := &mockBugRepo{
		getNextKeyFn: func(ctx context.Context) (string, error) { return "B002", nil },
		createFn: func(ctx context.Context, bug *models.Bug) error {
			capturedBug = bug
			bug.ID = 2
			return nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	_, _, err := svc.CreateBug(ctx, CreateBugInput{
		Title:       "Bug with description",
		Severity:    models.BugSeverityMedium,
		Description: "Detailed description here",
	})
	if err != nil {
		t.Fatalf("CreateBug() error = %v", err)
	}
	if capturedBug.Description == nil || *capturedBug.Description != "Detailed description here" {
		t.Errorf("expected description to be set, got %v", capturedBug.Description)
	}
}

func TestBugService_CreateBug_WithLinkedEntity(t *testing.T) {
	ctx := context.Background()

	epicRepo := &bugLinkEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			if key == "E07" {
				return &models.Epic{BaseEntity: models.BaseEntity{ID: 7, Key: "E07", Title: "Test Epic"}}, nil
			}
			return nil, fmt.Errorf("not found")
		},
	}

	var capturedBug *models.Bug
	repo := &mockBugRepo{
		getNextKeyFn: func(ctx context.Context) (string, error) { return "B003", nil },
		createFn: func(ctx context.Context, bug *models.Bug) error {
			capturedBug = bug
			bug.ID = 3
			return nil
		},
	}

	svc := newBugService(repo, epicRepo, nil, nil)

	_, _, err := svc.CreateBug(ctx, CreateBugInput{
		Title:            "Bug linked to epic",
		Severity:         models.BugSeverityCritical,
		LinkedEntityType: "epic",
		LinkedEntityKey:  "E07",
	})
	if err != nil {
		t.Fatalf("CreateBug() with linked entity error = %v", err)
	}
	if capturedBug.LinkedEntityType == nil || *capturedBug.LinkedEntityType != "epic" {
		t.Errorf("expected linked entity type 'epic', got %v", capturedBug.LinkedEntityType)
	}
	if capturedBug.LinkedEntityKey == nil || *capturedBug.LinkedEntityKey != "E07" {
		t.Errorf("expected linked entity key 'E07', got %v", capturedBug.LinkedEntityKey)
	}
}

func TestBugService_CreateBug_LinkedEntityNotFound(t *testing.T) {
	ctx := context.Background()

	epicRepo := &bugLinkEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, fmt.Errorf("epic not found")
		},
	}

	svc := newBugService(&mockBugRepo{}, epicRepo, nil, nil)

	_, _, err := svc.CreateBug(ctx, CreateBugInput{
		Title:            "Bug with bad link",
		Severity:         models.BugSeverityHigh,
		LinkedEntityType: "epic",
		LinkedEntityKey:  "E99",
	})
	if err == nil {
		t.Fatal("expected error for non-existent linked entity")
	}
	if !strings.Contains(err.Error(), "linked entity") {
		t.Errorf("expected error about linked entity, got: %v", err)
	}
}

func TestBugService_CreateBug_LinkedEntityTypeMissingKey(t *testing.T) {
	ctx := context.Background()
	svc := newBugService(&mockBugRepo{}, nil, nil, nil)

	// Type provided but no key
	_, _, err := svc.CreateBug(ctx, CreateBugInput{
		Title:            "Bug missing key",
		Severity:         models.BugSeverityHigh,
		LinkedEntityType: "epic",
		LinkedEntityKey:  "",
	})
	if err == nil {
		t.Fatal("expected error when entity type provided but key is missing")
	}
}

func TestBugService_CreateBug_InvalidLinkedEntityType(t *testing.T) {
	ctx := context.Background()
	svc := newBugService(&mockBugRepo{}, nil, nil, nil)

	_, _, err := svc.CreateBug(ctx, CreateBugInput{
		Title:            "Bug with invalid entity type",
		Severity:         models.BugSeverityHigh,
		LinkedEntityType: "sprint", // invalid type
		LinkedEntityKey:  "S01",
	})
	if err == nil {
		t.Fatal("expected error for invalid linked entity type")
	}
}

func TestBugService_CreateBug_GetNextKeyError(t *testing.T) {
	ctx := context.Background()

	repo := &mockBugRepo{
		getNextKeyFn: func(ctx context.Context) (string, error) {
			return "", fmt.Errorf("database error")
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	_, _, err := svc.CreateBug(ctx, CreateBugInput{
		Title:    "Some bug",
		Severity: models.BugSeverityHigh,
	})
	if err == nil {
		t.Fatal("expected error when GetNextKey fails")
	}
	if !strings.Contains(err.Error(), "key") {
		t.Errorf("expected error about key generation, got: %v", err)
	}
}

// --- GetBug tests ---

func TestBugService_GetBug(t *testing.T) {
	ctx := context.Background()

	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			if key == "B001" {
				return &models.Bug{BaseEntity: models.BaseEntity{ID: 1,
					Key:   "B001",
					Title: "Test Bug"}, Status: "reported",
					Severity: models.BugSeverityHigh,
				}, nil
			}
			return nil, fmt.Errorf("not found: %s", key)
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	bug, err := svc.GetBug(ctx, "B001")
	if err != nil {
		t.Fatalf("GetBug() error = %v", err)
	}
	if bug.Key != "B001" {
		t.Errorf("expected key B001, got %s", bug.Key)
	}
}

func TestBugService_GetBug_NotFound(t *testing.T) {
	ctx := context.Background()

	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return nil, fmt.Errorf("bug not found with key %q", key)
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	_, err := svc.GetBug(ctx, "B999")
	if err == nil {
		t.Fatal("expected error for non-existent bug")
	}
}

// --- UpdateBug tests ---

func TestBugService_UpdateBug_Title(t *testing.T) {
	ctx := context.Background()

	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return &models.Bug{BaseEntity: models.BaseEntity{ID: 1,
				Key:   "B001",
				Title: "Old Title"}, Status: "reported",
				Severity: models.BugSeverityLow,
			}, nil
		},
		updateFn: func(ctx context.Context, bug *models.Bug) error {
			if bug.Title != "New Title" {
				t.Errorf("expected title 'New Title', got %s", bug.Title)
			}
			return nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	newTitle := "New Title"
	bug, err := svc.UpdateBug(ctx, "B001", BugUpdates{Title: &newTitle})
	if err != nil {
		t.Fatalf("UpdateBug() error = %v", err)
	}
	if bug.Title != "New Title" {
		t.Errorf("expected title 'New Title', got %s", bug.Title)
	}
}

func TestBugService_UpdateBug_EmptyTitle(t *testing.T) {
	ctx := context.Background()

	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return &models.Bug{BaseEntity: models.BaseEntity{ID: 1, Key: "B001", Title: "Old Title"}, Status: "reported", Severity: models.BugSeverityHigh}, nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	empty := ""
	_, err := svc.UpdateBug(ctx, "B001", BugUpdates{Title: &empty})
	if err == nil {
		t.Fatal("expected error for empty title update")
	}
}

func TestBugService_UpdateBug_InvalidSeverity(t *testing.T) {
	ctx := context.Background()

	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return &models.Bug{BaseEntity: models.BaseEntity{ID: 1, Key: "B001", Title: "Bug"}, Status: "reported", Severity: models.BugSeverityHigh}, nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	badSeverity := models.BugSeverity("extreme")
	_, err := svc.UpdateBug(ctx, "B001", BugUpdates{Severity: &badSeverity})
	if err == nil {
		t.Fatal("expected error for invalid severity update")
	}
}

func TestBugService_UpdateBug_FilePath(t *testing.T) {
	ctx := context.Background()

	var capturedBug *models.Bug
	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return &models.Bug{BaseEntity: models.BaseEntity{ID: 1,
				Key:   "B001",
				Title: "Bug With File"}, Status: "reported",
				Severity: models.BugSeverityMedium,
			}, nil
		},
		updateFn: func(ctx context.Context, bug *models.Bug) error {
			capturedBug = bug
			return nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	newPath := "docs/plan/bugs/B001-custom.md"
	bug, err := svc.UpdateBug(ctx, "B001", BugUpdates{FilePath: &newPath})
	if err != nil {
		t.Fatalf("UpdateBug() error = %v", err)
	}
	if bug.FilePath == nil || *bug.FilePath != newPath {
		t.Errorf("expected file_path %q, got %v", newPath, bug.FilePath)
	}
	if capturedBug == nil {
		t.Fatal("expected Update to be called")
	}
	if capturedBug.FilePath == nil || *capturedBug.FilePath != newPath {
		t.Errorf("expected captured file_path %q, got %v", newPath, capturedBug.FilePath)
	}
}

func TestBugService_UpdateBug_ClearLinkedEntity(t *testing.T) {
	ctx := context.Background()

	entityType := "epic"
	entityKey := "E07"
	var capturedBug *models.Bug

	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return &models.Bug{BaseEntity: models.BaseEntity{ID: 1, Key: "B001", Title: "Linked Bug"}, Status: "reported", Severity: models.BugSeverityHigh,
				LinkedEntityType: &entityType,
				LinkedEntityKey:  &entityKey,
			}, nil
		},
		updateFn: func(ctx context.Context, bug *models.Bug) error {
			capturedBug = bug
			return nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	// Provide empty strings to clear the link
	emptyType := ""
	emptyKey := ""
	_, err := svc.UpdateBug(ctx, "B001", BugUpdates{
		LinkedEntityType: &emptyType,
		LinkedEntityKey:  &emptyKey,
	})
	if err != nil {
		t.Fatalf("UpdateBug() clear link error = %v", err)
	}
	if capturedBug.LinkedEntityType != nil || capturedBug.LinkedEntityKey != nil {
		t.Errorf("expected linked entity to be cleared, got type=%v key=%v",
			capturedBug.LinkedEntityType, capturedBug.LinkedEntityKey)
	}
}

// --- DeleteBug tests ---

func TestBugService_DeleteBug(t *testing.T) {
	ctx := context.Background()

	deleteCalled := false
	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return &models.Bug{BaseEntity: models.BaseEntity{ID: 1, Key: "B001", Title: "To Delete"}, Status: "reported", Severity: models.BugSeverityLow}, nil
		},
		deleteFn: func(ctx context.Context, id int64) error {
			deleteCalled = true
			if id != 1 {
				t.Errorf("expected delete id=1, got %d", id)
			}
			return nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	err := svc.DeleteBug(ctx, "B001")
	if err != nil {
		t.Fatalf("DeleteBug() error = %v", err)
	}
	if !deleteCalled {
		t.Error("expected Delete() to be called on repository")
	}
}

func TestBugService_DeleteBug_NotFound(t *testing.T) {
	ctx := context.Background()

	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return nil, fmt.Errorf("bug not found with key %q", key)
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	err := svc.DeleteBug(ctx, "B999")
	if err == nil {
		t.Fatal("expected error for non-existent bug")
	}
}

// --- ListBugs tests ---

func TestBugService_ListBugs(t *testing.T) {
	ctx := context.Background()

	repo := &mockBugRepo{
		listFn: func(ctx context.Context, filters *repository.BugListFilters) ([]*models.Bug, error) {
			return []*models.Bug{
				{BaseEntity: models.BaseEntity{ID: 1, Key: "B001", Title: "Bug 1"}, Status: "reported", Severity: models.BugSeverityHigh},
				{BaseEntity: models.BaseEntity{ID: 2, Key: "B002", Title: "Bug 2"}, Status: "triaged", Severity: models.BugSeverityLow},
			}, nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	bugs, err := svc.ListBugs(ctx, BugFilters{})
	if err != nil {
		t.Fatalf("ListBugs() error = %v", err)
	}
	if len(bugs) != 2 {
		t.Errorf("expected 2 bugs, got %d", len(bugs))
	}
}

func TestBugService_ListBugs_WithStatusFilter(t *testing.T) {
	ctx := context.Background()

	var capturedFilters *repository.BugListFilters
	repo := &mockBugRepo{
		listFn: func(ctx context.Context, filters *repository.BugListFilters) ([]*models.Bug, error) {
			capturedFilters = filters
			return []*models.Bug{}, nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	status := models.BugStatus("reported")
	_, err := svc.ListBugs(ctx, BugFilters{Status: &status})
	if err != nil {
		t.Fatalf("ListBugs() error = %v", err)
	}

	if capturedFilters == nil || capturedFilters.Status == nil || *capturedFilters.Status != "reported" {
		t.Errorf("expected status filter 'reported' to be passed to repository, got %v", capturedFilters)
	}
}

func TestBugService_ListBugs_WithSeverityFilter(t *testing.T) {
	ctx := context.Background()

	var capturedFilters *repository.BugListFilters
	repo := &mockBugRepo{
		listFn: func(ctx context.Context, filters *repository.BugListFilters) ([]*models.Bug, error) {
			capturedFilters = filters
			return []*models.Bug{}, nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	severity := models.BugSeverityCritical
	_, err := svc.ListBugs(ctx, BugFilters{Severity: &severity})
	if err != nil {
		t.Fatalf("ListBugs() error = %v", err)
	}

	if capturedFilters == nil || capturedFilters.Severity == nil || *capturedFilters.Severity != models.BugSeverityCritical {
		t.Errorf("expected severity filter 'critical' to be passed to repository, got %v", capturedFilters)
	}
}

func TestBugService_ListBugs_ShowAllFalse_ExcludesTerminal(t *testing.T) {
	ctx := context.Background()

	var capturedFilters *repository.BugListFilters
	repo := &mockBugRepo{
		listFn: func(ctx context.Context, filters *repository.BugListFilters) ([]*models.Bug, error) {
			capturedFilters = filters
			return []*models.Bug{}, nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	_, err := svc.ListBugs(ctx, BugFilters{ShowAll: false})
	if err != nil {
		t.Fatalf("ListBugs() error = %v", err)
	}

	if capturedFilters == nil {
		t.Fatal("expected filters to be passed to repository")
	}
	if capturedFilters.IncludeTerminal {
		t.Error("expected IncludeTerminal to be false when ShowAll is false")
	}
}

func TestBugService_ListBugs_ShowAllTrue_IncludesTerminal(t *testing.T) {
	ctx := context.Background()

	var capturedFilters *repository.BugListFilters
	repo := &mockBugRepo{
		listFn: func(ctx context.Context, filters *repository.BugListFilters) ([]*models.Bug, error) {
			capturedFilters = filters
			return []*models.Bug{}, nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	_, err := svc.ListBugs(ctx, BugFilters{ShowAll: true})
	if err != nil {
		t.Fatalf("ListBugs() error = %v", err)
	}

	if capturedFilters == nil {
		t.Fatal("expected filters to be passed to repository")
	}
	if !capturedFilters.IncludeTerminal {
		t.Error("expected IncludeTerminal to be true when ShowAll is true")
	}
}

// --- TransitionStatus tests ---

func TestBugService_TransitionStatus_ValidTransition(t *testing.T) {
	ctx := context.Background()

	currentStatus := models.BugStatus("draft")

	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return &models.Bug{BaseEntity: models.BaseEntity{ID: 1, Key: "B001", Title: "Test"}, Status: currentStatus, Severity: models.BugSeverityHigh}, nil
		},
		updateStatusFn: func(ctx context.Context, id int64, status models.BugStatus) error {
			currentStatus = status
			return nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	result, err := svc.TransitionStatus(ctx, "B001", "development", TransitionOptions{})
	if err != nil {
		t.Fatalf("TransitionStatus() error = %v", err)
	}
	if result.ToStatus != "development" {
		t.Errorf("expected status 'development', got %s", result.ToStatus)
	}
}

// TestBugService_TransitionStatus_ResolvesLegacyAliasCurrentStatus guards the
// route-based-workflow.md §5 "input compat shim" guarantee: an entity still
// parked under a pre-migration status name (bug.yaml's "draft" step has
// `aliases: [reported]`) must still be able to transition, not fail with
// "status 'reported' is not defined in workflow". FromStatus and the recorded
// history must keep the raw stored value ("reported"), not the resolved one —
// audit trails record what actually happened; only workflow lookups resolve.
func TestBugService_TransitionStatus_ResolvesLegacyAliasCurrentStatus(t *testing.T) {
	ctx := context.Background()

	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return &models.Bug{BaseEntity: models.BaseEntity{ID: 1, Key: "B001", Title: "Test"}, Status: "reported", Severity: models.BugSeverityHigh}, nil
		},
		updateStatusFn: func(ctx context.Context, id int64, status models.BugStatus) error {
			return nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)
	historyRecorder := &mockEntityHistoryRecorder{}
	svc.entitySvc.SetHistoryRepo(historyRecorder)

	result, err := svc.TransitionStatus(ctx, "B001", "development", TransitionOptions{})
	if err != nil {
		t.Fatalf("TransitionStatus() error = %v", err)
	}
	if result.FromStatus != "reported" {
		t.Errorf("expected FromStatus to preserve the raw stored value 'reported', got %q", result.FromStatus)
	}
	if result.ToStatus != "development" {
		t.Errorf("expected status 'development', got %s", result.ToStatus)
	}
	if len(historyRecorder.created) != 1 {
		t.Fatalf("expected exactly one history record, got %d", len(historyRecorder.created))
	}
	from := historyRecorder.created[0].FromStatus
	if from == nil || *from != "reported" {
		t.Errorf("expected recorded history from_status to preserve raw value 'reported', got %v", from)
	}
}

func TestBugService_TransitionStatus_InvalidStatus(t *testing.T) {
	ctx := context.Background()

	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return &models.Bug{BaseEntity: models.BaseEntity{ID: 1, Key: "B001", Title: "Test"}, Status: "reported", Severity: models.BugSeverityHigh}, nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	_, err := svc.TransitionStatus(ctx, "B001", "completely_made_up", TransitionOptions{})
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestBugService_TransitionStatus_NotFound(t *testing.T) {
	ctx := context.Background()

	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return nil, fmt.Errorf("bug not found with key %q", key)
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	_, err := svc.TransitionStatus(ctx, "B999", "triaged", TransitionOptions{})
	if err == nil {
		t.Fatal("expected error for non-existent bug")
	}
}

// --- GetNextStatus tests ---

func TestBugService_GetNextStatus(t *testing.T) {
	ctx := context.Background()

	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return &models.Bug{BaseEntity: models.BaseEntity{ID: 1, Key: "B001", Title: "Test Bug"}, Status: "draft", Severity: models.BugSeverityHigh}, nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	info, err := svc.GetNextStatus(ctx, "B001")
	if err != nil {
		t.Fatalf("GetNextStatus() error = %v", err)
	}
	if info.CurrentStatus != "draft" {
		t.Errorf("expected current status 'draft', got %s", info.CurrentStatus)
	}
	if len(info.AvailableTransitions) == 0 {
		t.Error("expected at least one available transition from 'draft'")
	}
}

func TestBugService_GetNextStatus_NotFound(t *testing.T) {
	ctx := context.Background()

	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return nil, fmt.Errorf("bug not found with key %q", key)
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	_, err := svc.GetNextStatus(ctx, "B999")
	if err == nil {
		t.Fatal("expected error for non-existent bug")
	}
}

// --- validateLinkedEntity tests ---

func TestBugService_CreateBug_LinkedToFeature(t *testing.T) {
	ctx := context.Background()

	featureRepo := &bugLinkFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			if key == "E07-F01" {
				return &models.Feature{BaseEntity: models.BaseEntity{ID: 1, Key: "E07-F01", Title: "Test Feature"}}, nil
			}
			return nil, fmt.Errorf("not found")
		},
	}

	var capturedBug *models.Bug
	repo := &mockBugRepo{
		getNextKeyFn: func(ctx context.Context) (string, error) { return "B004", nil },
		createFn: func(ctx context.Context, bug *models.Bug) error {
			capturedBug = bug
			bug.ID = 4
			return nil
		},
	}

	svc := newBugService(repo, nil, featureRepo, nil)

	_, _, err := svc.CreateBug(ctx, CreateBugInput{
		Title:            "Feature-linked bug",
		Severity:         models.BugSeverityMedium,
		LinkedEntityType: "feature",
		LinkedEntityKey:  "E07-F01",
	})
	if err != nil {
		t.Fatalf("CreateBug() with feature link error = %v", err)
	}
	if capturedBug.LinkedEntityType == nil || *capturedBug.LinkedEntityType != "feature" {
		t.Errorf("expected linked entity type 'feature', got %v", capturedBug.LinkedEntityType)
	}
}

func TestBugService_CreateBug_LinkedToTask(t *testing.T) {
	ctx := context.Background()

	taskRepo := &bugLinkTaskRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Task, error) {
			if key == "E07-F01-001" {
				return &models.Task{BaseEntity: models.BaseEntity{ID: 1, Key: "E07-F01-001", Title: "Test Task"}}, nil
			}
			return nil, fmt.Errorf("not found")
		},
	}

	repo := &mockBugRepo{
		getNextKeyFn: func(ctx context.Context) (string, error) { return "B005", nil },
		createFn: func(ctx context.Context, bug *models.Bug) error {
			bug.ID = 5
			return nil
		},
	}

	svc := newBugService(repo, nil, nil, taskRepo)

	_, _, err := svc.CreateBug(ctx, CreateBugInput{
		Title:            "Task-linked bug",
		Severity:         models.BugSeverityLow,
		LinkedEntityType: "task",
		LinkedEntityKey:  "E07-F01-001",
	})
	if err != nil {
		t.Fatalf("CreateBug() with task link error = %v", err)
	}
}

// --- Delegation-specific tests (TC-F09-003 through TC-F09-017) ---

// TC-F09-005: TransitionStatus returns result with correct fields
func TestBugService_TransitionStatus_ReturnsResult(t *testing.T) {
	ctx := context.Background()

	currentStatus := models.BugStatus("draft")

	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return &models.Bug{
				BaseEntity: models.BaseEntity{ID: 1, Key: "B001", Title: "Test"},
				Status:     currentStatus,
				Severity:   models.BugSeverityHigh,
			}, nil
		},
		updateStatusFn: func(ctx context.Context, id int64, status models.BugStatus) error {
			currentStatus = status
			return nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	result, err := svc.TransitionStatus(ctx, "B001", "development", TransitionOptions{})
	if err != nil {
		t.Fatalf("TransitionStatus() error = %v", err)
	}
	if result.ToStatus != "development" {
		t.Errorf("expected result ToStatus 'development', got %s", result.ToStatus)
	}
	if !result.Transitioned {
		t.Error("expected Transitioned to be true")
	}
}

// TC-F09-006: TransitionStatus propagates EntityService errors
func TestBugService_TransitionStatus_PropagatesEntityServiceErrors(t *testing.T) {
	ctx := context.Background()

	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return &models.Bug{
				BaseEntity: models.BaseEntity{ID: 1, Key: "B001", Title: "Test"},
				Status:     "reported",
				Severity:   models.BugSeverityHigh,
			}, nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	// "completely_invalid" is not a valid workflow status
	_, err := svc.TransitionStatus(ctx, "B001", "completely_invalid", TransitionOptions{})
	if err == nil {
		t.Fatal("expected error for invalid target status")
	}
}

// TC-F09-007: TransitionStatus handles entity not found
func TestBugService_TransitionStatus_HandlesNotFound(t *testing.T) {
	ctx := context.Background()

	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return nil, fmt.Errorf("bug not found: %s", key)
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	_, err := svc.TransitionStatus(ctx, "B999", "triaged", TransitionOptions{})
	if err == nil {
		t.Fatal("expected error for non-existent bug")
	}
}

// TC-F09-010: GetNextStatus with terminal status returns no transitions
func TestBugService_GetNextStatus_NoTransitions(t *testing.T) {
	ctx := context.Background()

	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			// "resolved" has no valid transitions in the default bug workflow
			return &models.Bug{
				BaseEntity: models.BaseEntity{ID: 1, Key: "B001", Title: "Test"},
				Status:     "resolved",
				Severity:   models.BugSeverityHigh,
			}, nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	info, err := svc.GetNextStatus(ctx, "B001")
	if err != nil {
		t.Fatalf("GetNextStatus() error = %v", err)
	}
	if len(info.AvailableTransitions) != 0 {
		t.Errorf("expected no available transitions for terminal status, got %d", len(info.AvailableTransitions))
	}
}

// TC-F09-013: makeResolveActionFn callback handles non-Bug entity gracefully
func TestBugService_makeResolveActionFn_NonBugEntity(t *testing.T) {
	repo := &mockBugRepo{}
	svc := newBugService(repo, nil, nil, nil)

	resolveActionFn := svc.makeResolveActionFn()

	// Pass a non-Bug entity (Epic)
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E01", Title: "Test Epic"}}
	action := resolveActionFn(epic, "reported")

	if action != nil {
		t.Error("expected nil action for non-Bug entity")
	}
}

// ---------------------------------------------------------------------------
// E28-F04 T-005 — Tag integration tests (AC-15, AC-15b, AC-16, AC-17,
// AC-17b, AC-18, AC-18b for the bug row).
// ---------------------------------------------------------------------------

// TestBugService_CreateBug_NoTagsAndNoRequirement covers AC-15 (bug row).
// When no tags are supplied and no enforcement is configured, the service
// MUST still invoke EnforceRequired exactly once (fast-path returning nil)
// and MUST NOT invoke AttachMany. The bug is persisted.
func TestBugService_CreateBug_NoTagsAndNoRequirement(t *testing.T) {
	ctx := context.Background()

	repo := &mockBugRepo{
		getNextKeyFn: func(ctx context.Context) (string, error) { return "B001", nil },
		createFn: func(ctx context.Context, bug *models.Bug) error {
			bug.ID = 1
			return nil
		},
	}

	tagSvc := NewMockTagService() // no enforcement; no tags
	svc := newBugServiceWithTagSvc(repo, nil, nil, nil, tagSvc)

	bug, _, err := svc.CreateBug(ctx, CreateBugInput{
		Title:    "No tags here",
		Severity: models.BugSeverityLow,
		Tags:     nil,
	})
	if err != nil {
		t.Fatalf("CreateBug() error = %v", err)
	}
	if bug == nil {
		t.Fatal("expected bug, got nil")
	}
	if tagSvc.EnforceRequiredCalls != 1 {
		t.Errorf("EnforceRequiredCalls = %d, want 1", tagSvc.EnforceRequiredCalls)
	}
	if tagSvc.AttachManyCalls != 0 {
		t.Errorf("AttachManyCalls = %d, want 0 (no tags supplied)", tagSvc.AttachManyCalls)
	}
	if tagSvc.LastEnforceEntityType != models.EntityTypeBug {
		t.Errorf("EnforceRequired entityType = %q, want %q",
			tagSvc.LastEnforceEntityType, models.EntityTypeBug)
	}
}

// TestBugService_CreateBug_NilTagSvcIsSkippedCleanly covers AC-15b.
// Confirms the graceful-degradation property of REQ-F-018: a nil tagSvc
// must not panic or produce errors; tag hooks simply do not run.
func TestBugService_CreateBug_NilTagSvcIsSkippedCleanly(t *testing.T) {
	ctx := context.Background()

	repo := &mockBugRepo{
		getNextKeyFn: func(ctx context.Context) (string, error) { return "B001", nil },
		createFn: func(ctx context.Context, bug *models.Bug) error {
			bug.ID = 1
			return nil
		},
	}
	// Explicit nil tagSvc — production code paths that predate F04 wiring.
	svc := newBugServiceWithTagSvc(repo, nil, nil, nil, nil)

	bug, _, err := svc.CreateBug(ctx, CreateBugInput{
		Title:    "Nil tagSvc bug",
		Severity: models.BugSeverityMedium,
		Tags:     []string{"voice"}, // even with tags, nil svc is OK
	})
	if err != nil {
		t.Fatalf("CreateBug() with nil tagSvc error = %v", err)
	}
	if bug == nil {
		t.Fatal("expected bug, got nil")
	}
}

// TestBugService_CreateBug_RequiredTypeMissingTagsAborts covers AC-16.
// When EnforceRequired returns *TagRequiredError, the service MUST return
// that error unchanged AND MUST NOT invoke repo.Create. This proves the
// pre-persistence ordering of the enforcement check (REQ-F-008).
func TestBugService_CreateBug_RequiredTypeMissingTagsAborts(t *testing.T) {
	ctx := context.Background()

	createCalled := false
	repo := &mockBugRepo{
		getNextKeyFn: func(ctx context.Context) (string, error) { return "B001", nil },
		createFn: func(ctx context.Context, bug *models.Bug) error {
			createCalled = true
			bug.ID = 1
			return nil
		},
	}

	tagSvc := NewMockTagService().WithEnforceRequiredFn(
		func(ctx context.Context, entityType models.EntityType, names []string) error {
			return &TagRequiredError{EntityType: string(entityType)}
		},
	)
	svc := newBugServiceWithTagSvc(repo, nil, nil, nil, tagSvc)

	_, _, err := svc.CreateBug(ctx, CreateBugInput{
		Title:    "Should fail enforcement",
		Severity: models.BugSeverityLow,
		Tags:     nil,
	})
	if err == nil {
		t.Fatal("expected TagRequiredError, got nil")
	}
	var required *TagRequiredError
	if !errors.As(err, &required) {
		t.Fatalf("expected *TagRequiredError, got %T: %v", err, err)
	}
	if required.EntityType != "bug" {
		t.Errorf("TagRequiredError.EntityType = %q, want %q", required.EntityType, "bug")
	}
	if createCalled {
		t.Error("repo.Create was invoked after enforcement failure (REQ-F-008 violation)")
	}
	if tagSvc.AttachManyCalls != 0 {
		t.Errorf("AttachManyCalls = %d, want 0 after enforcement failure", tagSvc.AttachManyCalls)
	}
}

// TestBugService_CreateBug_TagsProvidedAttachAfterPersist covers AC-17.
// When tags are supplied, the service MUST:
//  1. Invoke EnforceRequired first (returns nil because tags present).
//  2. Persist the entity (repo.Create).
//  3. Invoke AttachMany AFTER the entity has an ID.
//
// The event log proves the exact ordering; AttachMany receives the post-
// insert ID.
func TestBugService_CreateBug_TagsProvidedAttachAfterPersist(t *testing.T) {
	ctx := context.Background()

	tagSvc := NewMockTagService()

	repo := &mockBugRepo{
		getNextKeyFn: func(ctx context.Context) (string, error) { return "B001", nil },
		createFn: func(ctx context.Context, bug *models.Bug) error {
			bug.ID = 42
			tagSvc.RecordEvent("Create")
			return nil
		},
	}
	svc := newBugServiceWithTagSvc(repo, nil, nil, nil, tagSvc)

	_, _, err := svc.CreateBug(ctx, CreateBugInput{
		Title:    "Bug with tags",
		Severity: models.BugSeverityHigh,
		Tags:     []string{"voice", "auth"},
	})
	if err != nil {
		t.Fatalf("CreateBug() error = %v", err)
	}
	if tagSvc.EnforceRequiredCalls != 1 {
		t.Errorf("EnforceRequiredCalls = %d, want 1", tagSvc.EnforceRequiredCalls)
	}
	if tagSvc.AttachManyCalls != 1 {
		t.Errorf("AttachManyCalls = %d, want 1", tagSvc.AttachManyCalls)
	}
	if tagSvc.LastAttachEntityID != 42 {
		t.Errorf("AttachMany entityID = %d, want 42 (post-insert id)", tagSvc.LastAttachEntityID)
	}
	if tagSvc.LastAttachEntityType != models.EntityTypeBug {
		t.Errorf("AttachMany entityType = %q, want %q",
			tagSvc.LastAttachEntityType, models.EntityTypeBug)
	}
	// AC-17 ordering assertion: EnforceRequired → Create → AttachMany.
	gotEvents := tagSvc.EventsCopy()
	wantEvents := []string{"EnforceRequired", "Create", "AttachMany"}
	if !sliceEq(gotEvents, wantEvents) {
		t.Errorf("event order = %v, want %v", gotEvents, wantEvents)
	}
}

// TestBugService_CreateBug_AttachFailurePropagates covers AC-17b.
// When AttachMany fails (e.g., an unregistered tag), the error surfaces
// to the caller UNCHANGED and the entity REMAINS PERSISTED (matches ADR-
// F04-2: no transactions in F04; partial-write semantics accepted).
func TestBugService_CreateBug_AttachFailurePropagates(t *testing.T) {
	ctx := context.Background()

	createCalled := false
	repo := &mockBugRepo{
		getNextKeyFn: func(ctx context.Context) (string, error) { return "B001", nil },
		createFn: func(ctx context.Context, bug *models.Bug) error {
			createCalled = true
			bug.ID = 5
			return nil
		},
	}
	tagSvc := NewMockTagService().WithAttachManyFn(
		func(ctx context.Context, entityType models.EntityType, entityID int64, names []string) error {
			return &UnregisteredTagError{Name: "ghost"}
		},
	)
	svc := newBugServiceWithTagSvc(repo, nil, nil, nil, tagSvc)

	_, _, err := svc.CreateBug(ctx, CreateBugInput{
		Title:    "Attach will fail",
		Severity: models.BugSeverityLow,
		Tags:     []string{"ghost"},
	})
	if err == nil {
		t.Fatal("expected UnregisteredTagError, got nil")
	}
	var unregistered *UnregisteredTagError
	if !errors.As(err, &unregistered) {
		t.Fatalf("expected *UnregisteredTagError unchanged, got %T: %v", err, err)
	}
	if !createCalled {
		t.Error("entity was not persisted before AttachMany failure (expected persisted per ADR-F04-2)")
	}
}

// TestBugService_UpdateBug_TagsAdditive covers AC-18.
// A non-empty updates.Tags triggers exactly one AttachMany call; DetachOne
// is NEVER invoked on update (removal goes through `shark bug tag rm`).
func TestBugService_UpdateBug_TagsAdditive(t *testing.T) {
	ctx := context.Background()

	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return &models.Bug{
				BaseEntity: models.BaseEntity{ID: 1, Key: "B001", Title: "Existing"},
				Status:     "reported",
				Severity:   models.BugSeverityHigh,
			}, nil
		},
		updateFn: func(ctx context.Context, bug *models.Bug) error { return nil },
	}

	tagSvc := NewMockTagService()
	svc := newBugServiceWithTagSvc(repo, nil, nil, nil, tagSvc)

	_, err := svc.UpdateBug(ctx, "B001", BugUpdates{Tags: []string{"voice"}})
	if err != nil {
		t.Fatalf("UpdateBug() with tags error = %v", err)
	}
	if tagSvc.AttachManyCalls != 1 {
		t.Errorf("AttachManyCalls = %d, want 1", tagSvc.AttachManyCalls)
	}
	if tagSvc.DetachOneCalls != 0 {
		t.Errorf("DetachOneCalls = %d, want 0 (update is additive only)", tagSvc.DetachOneCalls)
	}
	if !sliceEq(tagSvc.LastAttachNames, []string{"voice"}) {
		t.Errorf("AttachMany names = %v, want [voice]", tagSvc.LastAttachNames)
	}
	if tagSvc.LastAttachEntityID != 1 {
		t.Errorf("AttachMany entityID = %d, want 1", tagSvc.LastAttachEntityID)
	}
}

// TestBugService_UpdateBug_EmptyTagsIsNoOp covers AC-18b.
// Both nil and explicit empty-slice update.Tags must result in zero tag
// service calls. The update itself still proceeds (title/severity/etc.).
func TestBugService_UpdateBug_EmptyTagsIsNoOp(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name string
		tags []string
	}{
		{"nil tags", nil},
		{"empty slice tags", []string{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockBugRepo{
				getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
					return &models.Bug{
						BaseEntity: models.BaseEntity{ID: 1, Key: "B001", Title: "Existing"},
						Status:     "reported",
						Severity:   models.BugSeverityHigh,
					}, nil
				},
				updateFn: func(ctx context.Context, bug *models.Bug) error { return nil },
			}
			tagSvc := NewMockTagService()
			svc := newBugServiceWithTagSvc(repo, nil, nil, nil, tagSvc)

			// Also change title to make the update meaningful.
			newTitle := "Updated"
			_, err := svc.UpdateBug(ctx, "B001", BugUpdates{
				Title: &newTitle,
				Tags:  tc.tags,
			})
			if err != nil {
				t.Fatalf("UpdateBug() error = %v", err)
			}
			if tagSvc.AttachManyCalls != 0 {
				t.Errorf("AttachManyCalls = %d, want 0 for %s", tagSvc.AttachManyCalls, tc.name)
			}
			if tagSvc.DetachOneCalls != 0 {
				t.Errorf("DetachOneCalls = %d, want 0 for %s", tagSvc.DetachOneCalls, tc.name)
			}
			if tagSvc.EnforceRequiredCalls != 0 {
				t.Errorf("EnforceRequiredCalls = %d, want 0 on update for %s",
					tagSvc.EnforceRequiredCalls, tc.name)
			}
		})
	}
}

// sliceEq is a small helper used by E28-F04 tag-integration tests above.
// Two slices are equal when they share length and all element indices
// match. nil and empty are treated as equal.
func sliceEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TC-F09-014: makeResolveActionFn callback returns nil for unconfigured status
func TestBugService_makeResolveActionFn_UnconfiguredStatus(t *testing.T) {
	repo := &mockBugRepo{}
	svc := newBugService(repo, nil, nil, nil)

	resolveActionFn := svc.makeResolveActionFn()

	bug := &models.Bug{
		BaseEntity: models.BaseEntity{Key: "B001", Title: "Test Bug"},
		Status:     "reported",
		Severity:   models.BugSeverityHigh,
	}
	// "reported" status has no orchestrator action in default workflow
	action := resolveActionFn(bug, "reported")

	if action != nil {
		t.Error("expected nil action for status without configured orchestrator action")
	}
}

// ============================================================================
// TC-SVC-A through TC-SVC-E: Size field propagation (E07-F42-005)
// ============================================================================

func TestBugService_CreateBug_PropagatesSize(t *testing.T) {
	// TC-SVC-A: CreateBug propagates Size to repository.
	ctx := context.Background()
	var capturedBug *models.Bug

	repo := &mockBugRepo{
		getNextKeyFn: func(ctx context.Context) (string, error) { return "B001", nil },
		createFn: func(ctx context.Context, bug *models.Bug) error {
			capturedBug = bug
			bug.ID = 1
			return nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	size := 5
	bug, _, err := svc.CreateBug(ctx, CreateBugInput{
		Title:    "Bug with size",
		Severity: models.BugSeverityHigh,
		Size:     &size,
	})

	if err != nil {
		t.Fatalf("CreateBug() error = %v", err)
	}
	if bug == nil {
		t.Fatal("expected bug, got nil")
	}
	if capturedBug == nil {
		t.Fatal("repo Create was not called")
	}
	if capturedBug.Size == nil {
		t.Fatal("expected capturedBug.Size to be non-nil")
	}
	if *capturedBug.Size != 5 {
		t.Errorf("expected capturedBug.Size=5, got %d", *capturedBug.Size)
	}
}

func TestBugService_CreateBug_NilSizePropagated(t *testing.T) {
	// TC-SVC-E: CreateBug passes Size=nil when not provided.
	ctx := context.Background()
	var capturedBug *models.Bug

	repo := &mockBugRepo{
		getNextKeyFn: func(ctx context.Context) (string, error) { return "B001", nil },
		createFn: func(ctx context.Context, bug *models.Bug) error {
			capturedBug = bug
			bug.ID = 1
			return nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	bug, _, err := svc.CreateBug(ctx, CreateBugInput{
		Title:    "Bug without size",
		Severity: models.BugSeverityLow,
	})

	if err != nil {
		t.Fatalf("CreateBug() error = %v", err)
	}
	if bug == nil {
		t.Fatal("expected bug, got nil")
	}
	if capturedBug == nil {
		t.Fatal("repo Create was not called")
	}
	if capturedBug.Size != nil {
		t.Errorf("expected capturedBug.Size to be nil, got %d", *capturedBug.Size)
	}
}

func TestBugService_UpdateBug_SetsSize(t *testing.T) {
	// TC-SVC-C: UpdateBug with Size=ptr(8) updates the field.
	ctx := context.Background()
	var capturedBug *models.Bug

	size8 := 8
	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return &models.Bug{BaseEntity: models.BaseEntity{ID: 1, Key: key}}, nil
		},
		updateFn: func(ctx context.Context, bug *models.Bug) error {
			capturedBug = bug
			return nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	bug, err := svc.UpdateBug(ctx, "B001", BugUpdates{Size: &size8})
	if err != nil {
		t.Fatalf("UpdateBug() error = %v", err)
	}
	if bug == nil {
		t.Fatal("expected bug, got nil")
	}
	if capturedBug == nil {
		t.Fatal("repo Update was not called")
	}
	if capturedBug.Size == nil {
		t.Fatal("expected capturedBug.Size to be non-nil")
	}
	if *capturedBug.Size != 8 {
		t.Errorf("expected capturedBug.Size=8, got %d", *capturedBug.Size)
	}
}

func TestBugService_UpdateBug_ClearSize(t *testing.T) {
	// TC-SVC-B: UpdateBug with ClearSize=true sets model.Size = nil.
	ctx := context.Background()
	var capturedBug *models.Bug

	existingSize := 5
	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return &models.Bug{BaseEntity: models.BaseEntity{ID: 1, Key: key, Size: &existingSize}}, nil
		},
		updateFn: func(ctx context.Context, bug *models.Bug) error {
			capturedBug = bug
			return nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	bug, err := svc.UpdateBug(ctx, "B001", BugUpdates{ClearSize: true})
	if err != nil {
		t.Fatalf("UpdateBug() error = %v", err)
	}
	if bug == nil {
		t.Fatal("expected bug, got nil")
	}
	if capturedBug == nil {
		t.Fatal("repo Update was not called")
	}
	if capturedBug.Size != nil {
		t.Errorf("expected capturedBug.Size=nil (ClearSize=true), got %d", *capturedBug.Size)
	}
}

func TestBugService_UpdateBug_NoSizeChange(t *testing.T) {
	// TC-SVC-D: UpdateBug with neither Size nor ClearSize leaves size unchanged.
	ctx := context.Background()
	var capturedBug *models.Bug

	existingSize := 3
	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return &models.Bug{BaseEntity: models.BaseEntity{ID: 1, Key: key, Size: &existingSize}}, nil
		},
		updateFn: func(ctx context.Context, bug *models.Bug) error {
			capturedBug = bug
			return nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	bug, err := svc.UpdateBug(ctx, "B001", BugUpdates{})
	if err != nil {
		t.Fatalf("UpdateBug() error = %v", err)
	}
	if bug == nil {
		t.Fatal("expected bug, got nil")
	}
	if capturedBug == nil {
		t.Fatal("repo Update was not called")
	}
	if capturedBug.Size == nil {
		t.Fatal("expected capturedBug.Size to remain non-nil")
	}
	if *capturedBug.Size != 3 {
		t.Errorf("expected capturedBug.Size=3 (unchanged), got %d", *capturedBug.Size)
	}
}
