package services

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/techdebt"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// mockTechDebtRepo implements TechDebtRepository for testing.
type mockTechDebtRepo struct {
	createFn          func(ctx context.Context, td *models.TechDebt) error
	getByKeyFn        func(ctx context.Context, key string) (*models.TechDebt, error)
	getByIDFn         func(ctx context.Context, id int64) (*models.TechDebt, error)
	updateFn          func(ctx context.Context, td *models.TechDebt) error
	deleteFn          func(ctx context.Context, id int64) error
	updateStatusFn    func(ctx context.Context, id int64, status models.TechDebtStatus) error
	generateNextKeyFn func(ctx context.Context) (string, error)
	listFn            func(ctx context.Context) ([]*models.TechDebt, error)
	listWithFiltersFn func(ctx context.Context, filters techdebt.TechDebtFilters) ([]*models.TechDebt, error)
	countByStatusFn   func(ctx context.Context) (map[string]int, error)
	countByCategoryFn func(ctx context.Context) (map[string]int, error)
}

func (m *mockTechDebtRepo) Create(ctx context.Context, td *models.TechDebt) error {
	if m.createFn != nil {
		return m.createFn(ctx, td)
	}
	return nil
}

func (m *mockTechDebtRepo) GetByKey(ctx context.Context, key string) (*models.TechDebt, error) {
	if m.getByKeyFn != nil {
		return m.getByKeyFn(ctx, key)
	}
	return nil, fmt.Errorf("not found: %s", key)
}

func (m *mockTechDebtRepo) GetByID(ctx context.Context, id int64) (*models.TechDebt, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, fmt.Errorf("not found: %d", id)
}

func (m *mockTechDebtRepo) Update(ctx context.Context, td *models.TechDebt) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, td)
	}
	return nil
}

func (m *mockTechDebtRepo) Delete(ctx context.Context, id int64) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockTechDebtRepo) UpdateStatus(ctx context.Context, id int64, status models.TechDebtStatus) error {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, id, status)
	}
	return nil
}

func (m *mockTechDebtRepo) GenerateNextKey(ctx context.Context) (string, error) {
	if m.generateNextKeyFn != nil {
		return m.generateNextKeyFn(ctx)
	}
	return "TD-001", nil
}

func (m *mockTechDebtRepo) List(ctx context.Context) ([]*models.TechDebt, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return []*models.TechDebt{}, nil
}

func (m *mockTechDebtRepo) ListWithFilters(ctx context.Context, filters techdebt.TechDebtFilters) ([]*models.TechDebt, error) {
	if m.listWithFiltersFn != nil {
		return m.listWithFiltersFn(ctx, filters)
	}
	return []*models.TechDebt{}, nil
}

func (m *mockTechDebtRepo) CountByStatus(ctx context.Context) (map[string]int, error) {
	if m.countByStatusFn != nil {
		return m.countByStatusFn(ctx)
	}
	return map[string]int{}, nil
}

func (m *mockTechDebtRepo) CountByCategory(ctx context.Context) (map[string]int, error) {
	if m.countByCategoryFn != nil {
		return m.countByCategoryFn(ctx)
	}
	return map[string]int{}, nil
}

// mockTDEntityRepo is a minimal EntityRepository adapter for tech-debt tests.
type mockTDEntityRepo struct {
	tdRepo *mockTechDebtRepo
}

func (m *mockTDEntityRepo) GetByKey(ctx context.Context, key string) (models.Entity, error) {
	return m.tdRepo.GetByKey(ctx, key)
}

func (m *mockTDEntityRepo) GetByID(ctx context.Context, id int64) (models.Entity, error) {
	return m.tdRepo.GetByID(ctx, id)
}

func (m *mockTDEntityRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	return m.tdRepo.UpdateStatus(ctx, id, models.TechDebtStatus(status))
}

func (m *mockTDEntityRepo) Update(ctx context.Context, entity models.Entity) error {
	td, ok := entity.(*models.TechDebt)
	if !ok {
		return fmt.Errorf("expected *models.TechDebt, got %T", entity)
	}
	return m.tdRepo.Update(ctx, td)
}

func (m *mockTDEntityRepo) GetContextData(_ context.Context, _ int64) (*string, error) {
	return nil, nil
}

func (m *mockTDEntityRepo) UpdateContextData(_ context.Context, _ int64, _ *string) error {
	return nil
}

func newTDWorkflowSvc() *workflow.Service {
	return workflow.NewService("")
}

func newTechDebtService(repo *mockTechDebtRepo) *TechDebtService {
	wfSvc := newTDWorkflowSvc()
	entitySvc := NewEntityService(wfSvc)
	entityRepo := &mockTDEntityRepo{tdRepo: repo}
	return NewTechDebtService(repo, entitySvc, entityRepo, "", nil)
}

// --- CreateTechDebt tests ---

func TestTechDebtService_Create_Success(t *testing.T) {
	ctx := context.Background()

	repo := &mockTechDebtRepo{
		generateNextKeyFn: func(_ context.Context) (string, error) {
			return "TD-001", nil
		},
		createFn: func(_ context.Context, td *models.TechDebt) error {
			if td.Key != "TD-001" {
				t.Errorf("expected key TD-001, got %s", td.Key)
			}
			if td.Title != "Refactor database layer" {
				t.Errorf("expected title 'Refactor database layer', got %s", td.Title)
			}
			if td.Category != models.TechDebtCategoryArchitecture {
				t.Errorf("expected category architecture, got %s", td.Category)
			}
			if td.Severity != models.TechDebtSeverityHigh {
				t.Errorf("expected severity high, got %s", td.Severity)
			}
			td.ID = 1
			return nil
		},
	}

	svc := newTechDebtService(repo)

	td, _, err := svc.CreateTechDebt(ctx, CreateTechDebtInput{
		Title:    "Refactor database layer",
		Category: models.TechDebtCategoryArchitecture,
		Severity: models.TechDebtSeverityHigh,
	})
	if err != nil {
		t.Fatalf("CreateTechDebt() error = %v", err)
	}
	if td == nil {
		t.Fatal("expected tech-debt, got nil")
	}
	if td.Key != "TD-001" {
		t.Errorf("expected key TD-001, got %s", td.Key)
	}
	if td.Slug == nil || *td.Slug == "" {
		t.Error("expected non-empty slug")
	}
}

func TestTechDebtService_Create_ValidationError(t *testing.T) {
	ctx := context.Background()
	svc := newTechDebtService(&mockTechDebtRepo{})

	_, _, err := svc.CreateTechDebt(ctx, CreateTechDebtInput{
		Title:    "",
		Category: models.TechDebtCategoryCodeQuality,
		Severity: models.TechDebtSeverityMedium,
	})
	if err == nil {
		t.Fatal("expected error for empty title")
	}
	if !strings.Contains(err.Error(), "title") {
		t.Errorf("expected error about title, got: %v", err)
	}
}

func TestTechDebtService_Create_InvalidCategory(t *testing.T) {
	ctx := context.Background()
	svc := newTechDebtService(&mockTechDebtRepo{})

	_, _, err := svc.CreateTechDebt(ctx, CreateTechDebtInput{
		Title:    "Some debt",
		Category: "invalid-category",
		Severity: models.TechDebtSeverityMedium,
	})
	if err == nil {
		t.Fatal("expected error for invalid category")
	}
	if !strings.Contains(err.Error(), "invalid category") {
		t.Errorf("expected error about invalid category, got: %v", err)
	}
}

// --- GetTechDebt tests ---

func TestTechDebtService_Get_Success(t *testing.T) {
	ctx := context.Background()

	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test Debt"},
				Status:     "identified",
				Category:   models.TechDebtCategoryCodeQuality,
				Severity:   models.TechDebtSeverityMedium,
			}, nil
		},
	}

	svc := newTechDebtService(repo)

	td, err := svc.GetTechDebt(ctx, "TD-001")
	if err != nil {
		t.Fatalf("GetTechDebt() error = %v", err)
	}
	if td == nil {
		t.Fatal("expected tech-debt, got nil")
	}
	if td.Key != "TD-001" {
		t.Errorf("expected key TD-001, got %s", td.Key)
	}
}

func TestTechDebtService_Get_NotFound(t *testing.T) {
	ctx := context.Background()

	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.TechDebt, error) {
			return nil, fmt.Errorf("tech-debt not found with key %q", key)
		},
	}

	svc := newTechDebtService(repo)

	_, err := svc.GetTechDebt(ctx, "TD-999")
	if err == nil {
		t.Fatal("expected error for not found")
	}
	if !strings.Contains(err.Error(), "TD-999") {
		t.Errorf("expected error containing key, got: %v", err)
	}
}

// --- UpdateTechDebt tests ---

func TestTechDebtService_Update_Success(t *testing.T) {
	ctx := context.Background()

	var updatedTD *models.TechDebt

	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Old Title"},
				Status:     "identified",
				Category:   models.TechDebtCategoryCodeQuality,
				Severity:   models.TechDebtSeverityMedium,
			}, nil
		},
		updateFn: func(_ context.Context, td *models.TechDebt) error {
			updatedTD = td
			return nil
		},
	}

	svc := newTechDebtService(repo)

	newTitle := "New Title"
	newSeverity := models.TechDebtSeverityHigh
	td, err := svc.UpdateTechDebt(ctx, "TD-001", TechDebtUpdates{
		Title:    &newTitle,
		Severity: &newSeverity,
	})
	if err != nil {
		t.Fatalf("UpdateTechDebt() error = %v", err)
	}
	if td.Title != "New Title" {
		t.Errorf("expected title 'New Title', got %s", td.Title)
	}
	if updatedTD == nil {
		t.Fatal("expected update to be called")
	}
	if updatedTD.Severity != models.TechDebtSeverityHigh {
		t.Errorf("expected severity high, got %s", updatedTD.Severity)
	}
	// Category should remain unchanged
	if updatedTD.Category != models.TechDebtCategoryCodeQuality {
		t.Errorf("expected category unchanged (code-quality), got %s", updatedTD.Category)
	}
}

// --- Size on Create / Update ---

// TestTechDebtService_Create_WithSize verifies that a Size value passed in the
// CreateTechDebtInput is propagated to the persisted TechDebt model.
func TestTechDebtService_Create_WithSize(t *testing.T) {
	ctx := context.Background()

	var createdTD *models.TechDebt
	repo := &mockTechDebtRepo{
		generateNextKeyFn: func(_ context.Context) (string, error) { return "TD-001", nil },
		createFn: func(_ context.Context, td *models.TechDebt) error {
			createdTD = td
			td.ID = 1
			return nil
		},
	}

	svc := newTechDebtService(repo)
	five := 5
	td, _, err := svc.CreateTechDebt(ctx, CreateTechDebtInput{
		Title:    "Refactor auth",
		Category: models.TechDebtCategoryArchitecture,
		Severity: models.TechDebtSeverityMedium,
		Size:     &five,
	})
	if err != nil {
		t.Fatalf("CreateTechDebt() error = %v", err)
	}
	if td.Size == nil || *td.Size != 5 {
		t.Errorf("expected returned td.Size=5, got %v", td.Size)
	}
	if createdTD == nil || createdTD.Size == nil || *createdTD.Size != 5 {
		t.Errorf("expected persisted td.Size=5, got %v", createdTD)
	}
}

// TestTechDebtService_Update_SetSize verifies the three-branch dispatch:
// Size != nil → set; ClearSize=true → clear; both empty → no-op.
func TestTechDebtService_Update_SetSize(t *testing.T) {
	ctx := context.Background()
	existingSize := 3
	var updatedTD *models.TechDebt

	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Old", Size: &existingSize},
				Status:     "identified",
				Category:   models.TechDebtCategoryCodeQuality,
				Severity:   models.TechDebtSeverityMedium,
			}, nil
		},
		updateFn: func(_ context.Context, td *models.TechDebt) error {
			updatedTD = td
			return nil
		},
	}
	svc := newTechDebtService(repo)

	eight := 8
	td, err := svc.UpdateTechDebt(ctx, "TD-001", TechDebtUpdates{Size: &eight})
	if err != nil {
		t.Fatalf("UpdateTechDebt() error = %v", err)
	}
	if td.Size == nil || *td.Size != 8 {
		t.Errorf("expected td.Size=8 after update, got %v", td.Size)
	}
	if updatedTD == nil || updatedTD.Size == nil || *updatedTD.Size != 8 {
		t.Errorf("expected persisted td.Size=8, got %v", updatedTD)
	}
}

func TestTechDebtService_Update_ClearSize(t *testing.T) {
	ctx := context.Background()
	existingSize := 5
	var updatedTD *models.TechDebt

	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Old", Size: &existingSize},
				Status:     "identified",
				Category:   models.TechDebtCategoryCodeQuality,
				Severity:   models.TechDebtSeverityMedium,
			}, nil
		},
		updateFn: func(_ context.Context, td *models.TechDebt) error {
			updatedTD = td
			return nil
		},
	}
	svc := newTechDebtService(repo)

	td, err := svc.UpdateTechDebt(ctx, "TD-001", TechDebtUpdates{ClearSize: true})
	if err != nil {
		t.Fatalf("UpdateTechDebt() error = %v", err)
	}
	if td.Size != nil {
		t.Errorf("expected td.Size=nil after ClearSize, got %d", *td.Size)
	}
	if updatedTD == nil {
		t.Fatal("expected update to be called")
	}
	if updatedTD.Size != nil {
		t.Errorf("expected persisted td.Size=nil, got %d", *updatedTD.Size)
	}
}

func TestTechDebtService_Update_SizeNoOpPreservesExisting(t *testing.T) {
	ctx := context.Background()
	existingSize := 5
	var updatedTD *models.TechDebt

	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Old", Size: &existingSize},
				Status:     "identified",
				Category:   models.TechDebtCategoryCodeQuality,
				Severity:   models.TechDebtSeverityMedium,
			}, nil
		},
		updateFn: func(_ context.Context, td *models.TechDebt) error {
			updatedTD = td
			return nil
		},
	}
	svc := newTechDebtService(repo)

	newTitle := "New Title"
	// Touch only Title; Size and ClearSize both unset → existing size preserved.
	_, err := svc.UpdateTechDebt(ctx, "TD-001", TechDebtUpdates{Title: &newTitle})
	if err != nil {
		t.Fatalf("UpdateTechDebt() error = %v", err)
	}
	if updatedTD == nil || updatedTD.Size == nil || *updatedTD.Size != 5 {
		t.Errorf("expected td.Size to be preserved as 5, got %v", updatedTD)
	}
}

// --- DeleteTechDebt tests ---

func TestTechDebtService_Delete_Success(t *testing.T) {
	ctx := context.Background()

	var deletedID int64
	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, _ string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{ID: 42, Key: "TD-001", Title: "Test"},
				Status:     "identified",
				Category:   models.TechDebtCategoryCodeQuality,
				Severity:   models.TechDebtSeverityMedium,
			}, nil
		},
		deleteFn: func(_ context.Context, id int64) error {
			deletedID = id
			return nil
		},
	}

	svc := newTechDebtService(repo)

	err := svc.DeleteTechDebt(ctx, "TD-001")
	if err != nil {
		t.Fatalf("DeleteTechDebt() error = %v", err)
	}
	if deletedID != 42 {
		t.Errorf("expected delete with ID 42, got %d", deletedID)
	}
}

// --- AdvanceStatus tests ---

func TestTechDebtService_AdvanceStatus_Success(t *testing.T) {
	ctx := context.Background()

	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test"},
				Status:     "identified",
				Category:   models.TechDebtCategoryCodeQuality,
				Severity:   models.TechDebtSeverityMedium,
			}, nil
		},
		updateStatusFn: func(_ context.Context, _ int64, status models.TechDebtStatus) error {
			if status != "triaged" {
				t.Errorf("expected status triaged, got %s", status)
			}
			return nil
		},
	}

	svc := newTechDebtService(repo)

	info, err := svc.GetNextStatus(ctx, "TD-001")
	if err != nil {
		t.Fatalf("GetNextStatus() error = %v", err)
	}
	if len(info.AvailableTransitions) == 0 {
		t.Fatal("expected at least one transition from identified")
	}
}

func TestTechDebtService_AdvanceStatus_TerminalState(t *testing.T) {
	ctx := context.Background()

	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test"},
				Status:     "resolved",
				Category:   models.TechDebtCategoryCodeQuality,
				Severity:   models.TechDebtSeverityMedium,
			}, nil
		},
	}

	svc := newTechDebtService(repo)

	info, err := svc.GetNextStatus(ctx, "TD-001")
	if err != nil {
		t.Fatalf("GetNextStatus() error = %v", err)
	}
	if len(info.AvailableTransitions) != 0 {
		t.Errorf("expected no transitions from resolved, got %d", len(info.AvailableTransitions))
	}
}

// --- SetStatus tests ---

func TestTechDebtService_SetStatus_ValidTransition(t *testing.T) {
	ctx := context.Background()

	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test"},
				Status:     "identified",
				Category:   models.TechDebtCategoryCodeQuality,
				Severity:   models.TechDebtSeverityMedium,
			}, nil
		},
		updateStatusFn: func(_ context.Context, _ int64, status models.TechDebtStatus) error {
			return nil
		},
	}

	svc := newTechDebtService(repo)

	result, err := svc.TransitionStatus(ctx, "TD-001", "research", TransitionOptions{})
	if err != nil {
		t.Fatalf("TransitionStatus() error = %v", err)
	}
	if result.ToStatus != "research" {
		t.Errorf("expected to_status research, got %s", result.ToStatus)
	}
}

func TestTechDebtService_SetStatus_InvalidTransition(t *testing.T) {
	ctx := context.Background()

	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test"},
				Status:     "identified",
				Category:   models.TechDebtCategoryCodeQuality,
				Severity:   models.TechDebtSeverityMedium,
			}, nil
		},
	}

	svc := newTechDebtService(repo)

	// "resolved" is not a valid transition from "identified" (must go through triaged first)
	_, err := svc.TransitionStatus(ctx, "TD-001", "resolved", TransitionOptions{})
	if err == nil {
		t.Fatal("expected error for invalid transition")
	}
}

func TestTechDebtService_SetStatus_Force(t *testing.T) {
	ctx := context.Background()

	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test"},
				Status:     "identified",
				Category:   models.TechDebtCategoryCodeQuality,
				Severity:   models.TechDebtSeverityMedium,
			}, nil
		},
		updateStatusFn: func(_ context.Context, _ int64, _ models.TechDebtStatus) error {
			return nil
		},
	}

	svc := newTechDebtService(repo)

	// Force bypasses transition validation (requires reason)
	result, err := svc.TransitionStatus(ctx, "TD-001", "resolved", TransitionOptions{Force: true, Reason: "testing force"})
	if err != nil {
		t.Fatalf("TransitionStatus() with force error = %v", err)
	}
	if result.ToStatus != "resolved" {
		t.Errorf("expected to_status resolved, got %s", result.ToStatus)
	}
}

// --- Triage tests ---

func TestTechDebtService_Triage_FromIdentified(t *testing.T) {
	ctx := context.Background()

	var updatedTD *models.TechDebt
	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test"},
				Status:     "identified",
				Category:   models.TechDebtCategoryCodeQuality,
				Severity:   models.TechDebtSeverityMedium,
			}, nil
		},
		updateFn: func(_ context.Context, td *models.TechDebt) error {
			updatedTD = td
			return nil
		},
	}

	svc := newTechDebtService(repo)

	td, err := svc.TriageTechDebt(ctx, "TD-001", TriageTechDebtInput{
		Severity: "critical",
		Category: "architecture",
	})
	if err != nil {
		t.Fatalf("TriageTechDebt() error = %v", err)
	}
	if td.Status != "research" {
		t.Errorf("expected status research, got %s", td.Status)
	}
	if updatedTD == nil {
		t.Fatal("expected update to be called")
	}
	if updatedTD.Severity != models.TechDebtSeverityCritical {
		t.Errorf("expected severity critical, got %s", updatedTD.Severity)
	}
	if updatedTD.Category != models.TechDebtCategoryArchitecture {
		t.Errorf("expected category architecture, got %s", updatedTD.Category)
	}
}

func TestTechDebtService_Triage_AlreadyTriaged(t *testing.T) {
	ctx := context.Background()

	var updatedTD *models.TechDebt
	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test"},
				Status:     "in_progress",
				Category:   models.TechDebtCategoryCodeQuality,
				Severity:   models.TechDebtSeverityMedium,
			}, nil
		},
		updateFn: func(_ context.Context, td *models.TechDebt) error {
			updatedTD = td
			return nil
		},
	}

	svc := newTechDebtService(repo)

	td, err := svc.TriageTechDebt(ctx, "TD-001", TriageTechDebtInput{
		Severity: "high",
	})
	if err != nil {
		t.Fatalf("TriageTechDebt() error = %v", err)
	}
	// Status should NOT change from in_progress (already past identified)
	if td.Status != "in_progress" {
		t.Errorf("expected status in_progress (unchanged), got %s", td.Status)
	}
	if updatedTD.Severity != models.TechDebtSeverityHigh {
		t.Errorf("expected severity high, got %s", updatedTD.Severity)
	}
}

// --- ListTechDebts tests ---

func TestTechDebtService_List_WithFilters(t *testing.T) {
	ctx := context.Background()

	var capturedFilters techdebt.TechDebtFilters
	repo := &mockTechDebtRepo{
		listWithFiltersFn: func(_ context.Context, filters techdebt.TechDebtFilters) ([]*models.TechDebt, error) {
			capturedFilters = filters
			return []*models.TechDebt{
				{
					BaseEntity: models.BaseEntity{ID: 1, Key: "TD-001", Title: "Test"},
					Status:     "identified",
					Category:   models.TechDebtCategoryCodeQuality,
					Severity:   models.TechDebtSeverityMedium,
				},
			}, nil
		},
	}

	svc := newTechDebtService(repo)

	cat := models.TechDebtCategoryCodeQuality
	status := "identified"
	items, err := svc.ListTechDebts(ctx, TechDebtFilters{
		Status:   &status,
		Category: &cat,
	})
	if err != nil {
		t.Fatalf("ListTechDebts() error = %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
	if capturedFilters.Status == nil || *capturedFilters.Status != "identified" {
		t.Error("expected status filter to be passed to repository")
	}
	if capturedFilters.Category == nil || *capturedFilters.Category != models.TechDebtCategoryCodeQuality {
		t.Error("expected category filter to be passed to repository")
	}
}

// --- GetStatusOptions tests ---

func TestTechDebtService_GetStatusOptions(t *testing.T) {
	ctx := context.Background()

	repo := &mockTechDebtRepo{
		getByKeyFn: func(_ context.Context, key string) (*models.TechDebt, error) {
			return &models.TechDebt{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test"},
				Status:     "identified",
				Category:   models.TechDebtCategoryCodeQuality,
				Severity:   models.TechDebtSeverityMedium,
			}, nil
		},
	}

	svc := newTechDebtService(repo)

	options, err := svc.GetStatusOptions(ctx, "TD-001")
	if err != nil {
		t.Fatalf("GetStatusOptions() error = %v", err)
	}
	if len(options) == 0 {
		t.Error("expected at least one status option from identified")
	}
	// Verify "research" is a valid option from "identified"
	found := false
	for _, opt := range options {
		if opt == "research" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'research' in options, got %v", options)
	}
}
