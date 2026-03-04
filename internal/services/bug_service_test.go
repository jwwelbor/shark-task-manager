package services

import (
	"context"
	"fmt"
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

func newBugService(repo *mockBugRepo, epicRepo *bugLinkEpicRepo, featureRepo *bugLinkFeatureRepo, taskRepo *bugLinkTaskRepo) *BugService {
	wfSvc := newBugWorkflowSvc()
	if epicRepo == nil {
		epicRepo = &bugLinkEpicRepo{}
	}
	if featureRepo == nil {
		featureRepo = &bugLinkFeatureRepo{}
	}
	if taskRepo == nil {
		taskRepo = &bugLinkTaskRepo{}
	}
	return NewBugService(repo, wfSvc, epicRepo, featureRepo, taskRepo)
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

	bug, err := svc.CreateBug(ctx, CreateBugInput{
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

func TestBugService_CreateBug_EmptyTitle(t *testing.T) {
	ctx := context.Background()
	svc := newBugService(&mockBugRepo{}, nil, nil, nil)

	_, err := svc.CreateBug(ctx, CreateBugInput{
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

	_, err := svc.CreateBug(ctx, CreateBugInput{
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

	_, err := svc.CreateBug(ctx, CreateBugInput{
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
				return &models.Epic{ID: 7, Key: "E07", Title: "Test Epic"}, nil
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

	_, err := svc.CreateBug(ctx, CreateBugInput{
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

	_, err := svc.CreateBug(ctx, CreateBugInput{
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
	_, err := svc.CreateBug(ctx, CreateBugInput{
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

	_, err := svc.CreateBug(ctx, CreateBugInput{
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

	_, err := svc.CreateBug(ctx, CreateBugInput{
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
				return &models.Bug{
					ID:       1,
					Key:      "B001",
					Title:    "Test Bug",
					Status:   "reported",
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
			return &models.Bug{
				ID:       1,
				Key:      "B001",
				Title:    "Old Title",
				Status:   "reported",
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
			return &models.Bug{
				ID: 1, Key: "B001", Title: "Old Title",
				Status: "reported", Severity: models.BugSeverityHigh,
			}, nil
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
			return &models.Bug{
				ID: 1, Key: "B001", Title: "Bug",
				Status: "reported", Severity: models.BugSeverityHigh,
			}, nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	badSeverity := models.BugSeverity("extreme")
	_, err := svc.UpdateBug(ctx, "B001", BugUpdates{Severity: &badSeverity})
	if err == nil {
		t.Fatal("expected error for invalid severity update")
	}
}

func TestBugService_UpdateBug_ClearLinkedEntity(t *testing.T) {
	ctx := context.Background()

	entityType := "epic"
	entityKey := "E07"
	var capturedBug *models.Bug

	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return &models.Bug{
				ID: 1, Key: "B001", Title: "Linked Bug",
				Status: "reported", Severity: models.BugSeverityHigh,
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
			return &models.Bug{ID: 1, Key: "B001", Title: "To Delete",
				Status: "reported", Severity: models.BugSeverityLow}, nil
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
				{ID: 1, Key: "B001", Title: "Bug 1", Status: "reported", Severity: models.BugSeverityHigh},
				{ID: 2, Key: "B002", Title: "Bug 2", Status: "triaged", Severity: models.BugSeverityLow},
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

// --- AdvanceBugStatus tests ---

func TestBugService_AdvanceBugStatus(t *testing.T) {
	ctx := context.Background()

	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return &models.Bug{
				ID: 1, Key: "B001", Title: "Test Bug",
				Status: "reported", Severity: models.BugSeverityHigh,
			}, nil
		},
		updateStatusFn: func(ctx context.Context, id int64, status models.BugStatus) error {
			if id != 1 {
				t.Errorf("expected id=1, got %d", id)
			}
			return nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	bug, err := svc.AdvanceBugStatus(ctx, "B001")
	if err != nil {
		t.Fatalf("AdvanceBugStatus() error = %v", err)
	}
	// Status should have changed from "reported"
	if bug.Status == "reported" {
		t.Error("expected status to change from 'reported'")
	}
}

func TestBugService_AdvanceBugStatus_NotFound(t *testing.T) {
	ctx := context.Background()

	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return nil, fmt.Errorf("bug not found with key %q", key)
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	_, err := svc.AdvanceBugStatus(ctx, "B999")
	if err == nil {
		t.Fatal("expected error for non-existent bug")
	}
}

// --- SetBugStatus tests ---

func TestBugService_SetBugStatus_Force(t *testing.T) {
	ctx := context.Background()

	var capturedStatus models.BugStatus
	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return &models.Bug{
				ID: 1, Key: "B001", Title: "Test",
				Status: "reported", Severity: models.BugSeverityHigh,
			}, nil
		},
		updateStatusFn: func(ctx context.Context, id int64, status models.BugStatus) error {
			capturedStatus = status
			return nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	bug, err := svc.SetBugStatus(ctx, "B001", "triaged", true)
	if err != nil {
		t.Fatalf("SetBugStatus() error = %v", err)
	}
	if bug.Status != "triaged" {
		t.Errorf("expected status 'triaged', got %s", bug.Status)
	}
	if capturedStatus != "triaged" {
		t.Errorf("expected repository to receive status 'triaged', got %s", capturedStatus)
	}
}

func TestBugService_SetBugStatus_InvalidStatus(t *testing.T) {
	ctx := context.Background()

	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return &models.Bug{
				ID: 1, Key: "B001", Title: "Test",
				Status: "reported", Severity: models.BugSeverityHigh,
			}, nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	// "completely_made_up" should not be a valid workflow status
	_, err := svc.SetBugStatus(ctx, "B001", "completely_made_up", false)
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

// --- TriageBug tests ---

func TestBugService_TriageBug_UpdatesSeverity(t *testing.T) {
	ctx := context.Background()

	var capturedBug *models.Bug
	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return &models.Bug{
				ID: 1, Key: "B001", Title: "Triageable Bug",
				Status: "reported", Severity: models.BugSeverityLow,
			}, nil
		},
		updateFn: func(ctx context.Context, bug *models.Bug) error {
			capturedBug = bug
			return nil
		},
		updateStatusFn: func(ctx context.Context, id int64, status models.BugStatus) error {
			return nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	newSeverity := models.BugSeverityCritical
	bug, err := svc.TriageBug(ctx, "B001", TriageBugInput{Severity: &newSeverity})
	if err != nil {
		t.Fatalf("TriageBug() error = %v", err)
	}
	if bug == nil {
		t.Fatal("expected non-nil bug")
	}
	if capturedBug.Severity != models.BugSeverityCritical {
		t.Errorf("expected severity 'critical' after triage, got %s", capturedBug.Severity)
	}
}

func TestBugService_TriageBug_SetsAssignedTo(t *testing.T) {
	ctx := context.Background()

	var capturedBug *models.Bug
	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return &models.Bug{
				ID: 1, Key: "B001", Title: "Assignable Bug",
				Status: "reported", Severity: models.BugSeverityHigh,
			}, nil
		},
		updateFn: func(ctx context.Context, bug *models.Bug) error {
			capturedBug = bug
			return nil
		},
		updateStatusFn: func(ctx context.Context, id int64, status models.BugStatus) error {
			return nil
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	assignee := "dev-agent"
	bug, err := svc.TriageBug(ctx, "B001", TriageBugInput{AssignedTo: &assignee})
	if err != nil {
		t.Fatalf("TriageBug() error = %v", err)
	}
	if bug == nil {
		t.Fatal("expected non-nil bug")
	}
	if capturedBug.ContextData == nil || !strings.Contains(*capturedBug.ContextData, "dev-agent") {
		t.Errorf("expected context_data to contain assignee 'dev-agent', got %v", capturedBug.ContextData)
	}
}

func TestBugService_TriageBug_NotFound(t *testing.T) {
	ctx := context.Background()

	repo := &mockBugRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Bug, error) {
			return nil, fmt.Errorf("bug not found with key %q", key)
		},
	}

	svc := newBugService(repo, nil, nil, nil)

	_, err := svc.TriageBug(ctx, "B999", TriageBugInput{})
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
				return &models.Feature{ID: 1, Key: "E07-F01", Title: "Test Feature"}, nil
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

	_, err := svc.CreateBug(ctx, CreateBugInput{
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
				return &models.Task{ID: 1, Key: "E07-F01-001", Title: "Test Task"}, nil
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

	_, err := svc.CreateBug(ctx, CreateBugInput{
		Title:            "Task-linked bug",
		Severity:         models.BugSeverityLow,
		LinkedEntityType: "task",
		LinkedEntityKey:  "E07-F01-001",
	})
	if err != nil {
		t.Fatalf("CreateBug() with task link error = %v", err)
	}
}
