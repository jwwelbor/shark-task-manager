package services

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// mockChangeCardRepo implements ChangeCardRepository for testing.
type mockChangeCardRepo struct {
	createFn        func(ctx context.Context, card *models.ChangeCard) error
	getByKeyFn      func(ctx context.Context, key string) (*models.ChangeCard, error)
	getByIDFn       func(ctx context.Context, id int64) (*models.ChangeCard, error)
	updateFn        func(ctx context.Context, card *models.ChangeCard) error
	deleteFn        func(ctx context.Context, id int64) error
	updateStatusFn  func(ctx context.Context, id int64, status models.ChangeCardStatus) error
	listFn          func(ctx context.Context, filter *repository.ChangeCardRepoFilter) ([]*models.ChangeCard, error)
	listByEpicFn    func(ctx context.Context, epicID int64) ([]*models.ChangeCard, error)
	listByFeatureFn func(ctx context.Context, featureID int64) ([]*models.ChangeCard, error)
	countByStatusFn func(ctx context.Context) (map[string]int, error)
	getNextKeyFn    func(ctx context.Context) (string, error)
}

func (m *mockChangeCardRepo) Create(ctx context.Context, card *models.ChangeCard) error {
	if m.createFn != nil {
		return m.createFn(ctx, card)
	}
	return nil
}

func (m *mockChangeCardRepo) GetByKey(ctx context.Context, key string) (*models.ChangeCard, error) {
	if m.getByKeyFn != nil {
		return m.getByKeyFn(ctx, key)
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockChangeCardRepo) GetByID(ctx context.Context, id int64) (*models.ChangeCard, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockChangeCardRepo) Update(ctx context.Context, card *models.ChangeCard) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, card)
	}
	return nil
}

func (m *mockChangeCardRepo) Delete(ctx context.Context, id int64) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockChangeCardRepo) UpdateStatus(ctx context.Context, id int64, status models.ChangeCardStatus) error {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, id, status)
	}
	return nil
}

func (m *mockChangeCardRepo) List(ctx context.Context, filter *repository.ChangeCardRepoFilter) ([]*models.ChangeCard, error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return []*models.ChangeCard{}, nil
}

func (m *mockChangeCardRepo) ListByEpic(ctx context.Context, epicID int64) ([]*models.ChangeCard, error) {
	if m.listByEpicFn != nil {
		return m.listByEpicFn(ctx, epicID)
	}
	return []*models.ChangeCard{}, nil
}

func (m *mockChangeCardRepo) ListByFeature(ctx context.Context, featureID int64) ([]*models.ChangeCard, error) {
	if m.listByFeatureFn != nil {
		return m.listByFeatureFn(ctx, featureID)
	}
	return []*models.ChangeCard{}, nil
}

func (m *mockChangeCardRepo) CountByStatus(ctx context.Context) (map[string]int, error) {
	if m.countByStatusFn != nil {
		return m.countByStatusFn(ctx)
	}
	return map[string]int{}, nil
}

func (m *mockChangeCardRepo) GetNextKey(ctx context.Context) (string, error) {
	if m.getNextKeyFn != nil {
		return m.getNextKeyFn(ctx)
	}
	return "CC-001", nil
}

// changeCardEpicRepo implements EpicRepository for change card tests.
type changeCardEpicRepo struct {
	getByKeyFn func(ctx context.Context, key string) (*models.Epic, error)
}

func (m *changeCardEpicRepo) GetByKey(ctx context.Context, key string) (*models.Epic, error) {
	if m.getByKeyFn != nil {
		return m.getByKeyFn(ctx, key)
	}
	return nil, fmt.Errorf("not found")
}

// Stub out remaining EpicRepository interface methods.
func (m *changeCardEpicRepo) GetByID(context.Context, int64) (*models.Epic, error) { return nil, nil }
func (m *changeCardEpicRepo) Create(context.Context, *models.Epic) error           { return nil }
func (m *changeCardEpicRepo) Update(context.Context, *models.Epic) error           { return nil }
func (m *changeCardEpicRepo) Delete(context.Context, int64) error                  { return nil }
func (m *changeCardEpicRepo) List(context.Context, *models.EpicStatus) ([]*models.Epic, error) {
	return nil, nil
}
func (m *changeCardEpicRepo) GetByFilePath(context.Context, string) (*models.Epic, error) {
	return nil, nil
}
func (m *changeCardEpicRepo) UpdateFilePath(context.Context, string, *string) error { return nil }
func (m *changeCardEpicRepo) UpdateKey(context.Context, string, string) error       { return nil }
func (m *changeCardEpicRepo) GetFeatureProgressDataByEpic(context.Context, int64) ([]repository.FeatureProgressData, error) {
	return nil, nil
}
func (m *changeCardEpicRepo) GetFeatureStatusBreakdown(context.Context, int64) (map[models.FeatureStatus]int, error) {
	return nil, nil
}
func (m *changeCardEpicRepo) GetFeatureStatusBreakdownByKey(context.Context, string) (map[models.FeatureStatus]int, error) {
	return nil, nil
}
func (m *changeCardEpicRepo) GetFeatureStatusRollup(context.Context, int64) (map[string]int, error) {
	return nil, nil
}
func (m *changeCardEpicRepo) GetTaskStatusRollup(context.Context, int64) (map[string]int, error) {
	return nil, nil
}
func (m *changeCardEpicRepo) UpdateStatus(context.Context, int64, models.EpicStatus) error {
	return nil
}
func (m *changeCardEpicRepo) CascadeStatusToFeaturesAndTasks(context.Context, int64, models.FeatureStatus, models.TaskStatus) error {
	return nil
}
func (m *changeCardEpicRepo) CascadeStatusToFeaturesAndTasksWithTx(_ context.Context, _ *sql.Tx, _ int64, _ models.FeatureStatus, _ models.TaskStatus) error {
	return nil
}
func (m *changeCardEpicRepo) BeginTx(context.Context) (*sql.Tx, error) { return nil, nil }
func (m *changeCardEpicRepo) GetEpicDisplayDataRaw(context.Context, int64) (*repository.EpicDisplayDataRaw, error) {
	return nil, nil
}

// changeCardFeatureRepo implements FeatureRepository for change card tests.
type changeCardFeatureRepo struct {
	getByKeyFn func(ctx context.Context, key string) (*models.Feature, error)
}

func (m *changeCardFeatureRepo) GetByKey(ctx context.Context, key string) (*models.Feature, error) {
	if m.getByKeyFn != nil {
		return m.getByKeyFn(ctx, key)
	}
	return nil, fmt.Errorf("not found")
}

// Stub out remaining FeatureRepository interface methods.
func (m *changeCardFeatureRepo) GetByID(context.Context, int64) (*models.Feature, error) {
	return nil, nil
}
func (m *changeCardFeatureRepo) Create(context.Context, *models.Feature) error   { return nil }
func (m *changeCardFeatureRepo) Update(context.Context, *models.Feature) error   { return nil }
func (m *changeCardFeatureRepo) Delete(context.Context, int64) error             { return nil }
func (m *changeCardFeatureRepo) List(context.Context) ([]*models.Feature, error) { return nil, nil }
func (m *changeCardFeatureRepo) ListByEpic(context.Context, int64) ([]*models.Feature, error) {
	return nil, nil
}
func (m *changeCardFeatureRepo) ListByEpicAndStatus(context.Context, int64, models.FeatureStatus) ([]*models.Feature, error) {
	return nil, nil
}
func (m *changeCardFeatureRepo) GetByFilePath(context.Context, string) (*models.Feature, error) {
	return nil, nil
}
func (m *changeCardFeatureRepo) UpdateFilePath(context.Context, string, *string) error { return nil }
func (m *changeCardFeatureRepo) UpdateKey(context.Context, string, string) error       { return nil }
func (m *changeCardFeatureRepo) GetTaskStatusBreakdown(context.Context, int64) (map[models.TaskStatus]int, error) {
	return nil, nil
}
func (m *changeCardFeatureRepo) GetTaskCount(context.Context, int64) (int, error)     { return 0, nil }
func (m *changeCardFeatureRepo) SetStatusOverride(context.Context, int64, bool) error { return nil }
func (m *changeCardFeatureRepo) UpdateStatus(context.Context, int64, models.FeatureStatus) error {
	return nil
}
func (m *changeCardFeatureRepo) UpdateStatusIfNotOverridden(context.Context, int64, models.FeatureStatus) (bool, error) {
	return false, nil
}
func (m *changeCardFeatureRepo) CascadeStatusToTasks(context.Context, int64, models.TaskStatus) error {
	return nil
}
func (m *changeCardFeatureRepo) GetFeatureDisplayDataRaw(context.Context, int64) (*repository.FeatureDisplayDataRaw, error) {
	return nil, nil
}

// mockChangeCardEntityRepo adapts mockChangeCardRepo to the EntityRepository interface.
type mockChangeCardEntityRepo struct {
	ccRepo *mockChangeCardRepo
}

func (m *mockChangeCardEntityRepo) GetByKey(ctx context.Context, key string) (models.Entity, error) {
	return m.ccRepo.GetByKey(ctx, key)
}

func (m *mockChangeCardEntityRepo) GetByID(ctx context.Context, id int64) (models.Entity, error) {
	return m.ccRepo.GetByID(ctx, id)
}

func (m *mockChangeCardEntityRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	return m.ccRepo.UpdateStatus(ctx, id, models.ChangeCardStatus(status))
}

func (m *mockChangeCardEntityRepo) Update(ctx context.Context, entity models.Entity) error {
	card, ok := entity.(*models.ChangeCard)
	if !ok {
		return fmt.Errorf("expected *models.ChangeCard, got %T", entity)
	}
	return m.ccRepo.Update(ctx, card)
}

func (m *mockChangeCardEntityRepo) GetContextData(ctx context.Context, id int64) (*string, error) {
	return nil, nil
}

func (m *mockChangeCardEntityRepo) UpdateContextData(ctx context.Context, id int64, data *string) error {
	return nil
}

func newChangeCardService(repo *mockChangeCardRepo, epicRepo *changeCardEpicRepo, featureRepo *changeCardFeatureRepo) *ChangeCardService {
	wfSvc := workflow.NewService("")
	entitySvc := NewEntityService(wfSvc)
	entityRepo := &mockChangeCardEntityRepo{ccRepo: repo}
	if epicRepo == nil {
		epicRepo = &changeCardEpicRepo{}
	}
	if featureRepo == nil {
		featureRepo = &changeCardFeatureRepo{}
	}
	return NewChangeCardService(repo, entitySvc, entityRepo, epicRepo, featureRepo, "")
}

func TestChangeCardService_CreateChangeCard(t *testing.T) {
	ctx := context.Background()

	repo := &mockChangeCardRepo{
		getNextKeyFn: func(ctx context.Context) (string, error) {
			return "CC-001", nil
		},
		createFn: func(ctx context.Context, card *models.ChangeCard) error {
			if card.Key != "CC-001" {
				t.Errorf("expected key CC-001, got %s", card.Key)
			}
			if card.Title != "Add dark mode toggle" {
				t.Errorf("expected title 'Add dark mode toggle', got %s", card.Title)
			}
			card.ID = 1
			return nil
		},
	}

	svc := newChangeCardService(repo, nil, nil)

	card, err := svc.CreateChangeCard(ctx, CreateChangeCardInput{
		Title: "Add dark mode toggle",
	})
	if err != nil {
		t.Fatalf("CreateChangeCard() error = %v", err)
	}
	if card == nil {
		t.Fatal("expected card, got nil")
	}
	if card.Key != "CC-001" {
		t.Errorf("expected key C001, got %s", card.Key)
	}
	if card.Slug == nil || *card.Slug == "" {
		t.Error("expected non-empty slug")
	}
}

func TestChangeCardService_CreateChangeCard_EmptyTitle(t *testing.T) {
	ctx := context.Background()
	svc := newChangeCardService(&mockChangeCardRepo{}, nil, nil)

	_, err := svc.CreateChangeCard(ctx, CreateChangeCardInput{Title: ""})
	if err == nil {
		t.Fatal("expected error for empty title")
	}
}

func TestChangeCardService_CreateChangeCard_WithEpicLink(t *testing.T) {
	ctx := context.Background()

	epicRepo := &changeCardEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			if key == "E07" {
				return &models.Epic{BaseEntity: models.BaseEntity{ID: 42, Key: "E07", Title: "Test Epic"}}, nil
			}
			return nil, fmt.Errorf("not found")
		},
	}

	var capturedCard *models.ChangeCard
	repo := &mockChangeCardRepo{
		getNextKeyFn: func(ctx context.Context) (string, error) { return "CC-002", nil },
		createFn: func(ctx context.Context, card *models.ChangeCard) error {
			capturedCard = card
			card.ID = 2
			return nil
		},
	}

	svc := newChangeCardService(repo, epicRepo, nil)

	card, err := svc.CreateChangeCard(ctx, CreateChangeCardInput{
		Title:   "Linked to epic",
		EpicKey: "E07",
	})
	if err != nil {
		t.Fatalf("CreateChangeCard() error = %v", err)
	}
	if card == nil {
		t.Fatal("expected card, got nil")
	}
	if capturedCard.EpicID == nil || *capturedCard.EpicID != 42 {
		t.Errorf("expected epic_id=42, got %v", capturedCard.EpicID)
	}
}

func TestChangeCardService_GetChangeCard(t *testing.T) {
	ctx := context.Background()

	repo := &mockChangeCardRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			if key == "CC-001" {
				return &models.ChangeCard{BaseEntity: models.BaseEntity{ID: 1, Key: "CC-001", Title: "Test Card"}, Status: "proposed"}, nil
			}
			return nil, fmt.Errorf("not found: %s", key)
		},
	}

	svc := newChangeCardService(repo, nil, nil)

	card, err := svc.GetChangeCard(ctx, "CC-001")
	if err != nil {
		t.Fatalf("GetChangeCard() error = %v", err)
	}
	if card.Key != "CC-001" {
		t.Errorf("expected key CC-001, got %s", card.Key)
	}
}

func TestChangeCardService_GetChangeCard_NotFound(t *testing.T) {
	ctx := context.Background()

	repo := &mockChangeCardRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			return nil, fmt.Errorf("not found: %s", key)
		},
	}

	svc := newChangeCardService(repo, nil, nil)

	_, err := svc.GetChangeCard(ctx, "CC-999")
	if err == nil {
		t.Fatal("expected error for non-existent card")
	}
}

func TestChangeCardService_ListChangeCards(t *testing.T) {
	ctx := context.Background()

	repo := &mockChangeCardRepo{
		listFn: func(ctx context.Context, filter *repository.ChangeCardRepoFilter) ([]*models.ChangeCard, error) {
			return []*models.ChangeCard{
				{BaseEntity: models.BaseEntity{ID: 1, Key: "CC-001", Title: "Card 1"}, Status: "proposed"},
				{BaseEntity: models.BaseEntity{ID: 2, Key: "CC-002", Title: "Card 2"}, Status: "approved"},
			}, nil
		},
	}

	svc := newChangeCardService(repo, nil, nil)

	cards, err := svc.ListChangeCards(ctx, ChangeCardFilters{})
	if err != nil {
		t.Fatalf("ListChangeCards() error = %v", err)
	}
	if len(cards) != 2 {
		t.Errorf("expected 2 cards, got %d", len(cards))
	}
}

func TestChangeCardService_ListChangeCards_WithEpicFilter(t *testing.T) {
	ctx := context.Background()

	epicRepo := &changeCardEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 10, Key: "E07"}}, nil
		},
	}

	var capturedFilter *repository.ChangeCardRepoFilter
	repo := &mockChangeCardRepo{
		listFn: func(ctx context.Context, filter *repository.ChangeCardRepoFilter) ([]*models.ChangeCard, error) {
			capturedFilter = filter
			return []*models.ChangeCard{}, nil
		},
	}

	svc := newChangeCardService(repo, epicRepo, nil)

	_, err := svc.ListChangeCards(ctx, ChangeCardFilters{EpicKey: "E07"})
	if err != nil {
		t.Fatalf("ListChangeCards() error = %v", err)
	}
	if capturedFilter == nil || capturedFilter.EpicID == nil || *capturedFilter.EpicID != 10 {
		t.Error("expected EpicID filter to be set to 10")
	}
}

func TestChangeCardService_ListChangeCards_ShowAllTrue_IncludesTerminal(t *testing.T) {
	ctx := context.Background()

	var capturedFilter *repository.ChangeCardRepoFilter
	repo := &mockChangeCardRepo{
		listFn: func(ctx context.Context, filter *repository.ChangeCardRepoFilter) ([]*models.ChangeCard, error) {
			capturedFilter = filter
			return []*models.ChangeCard{}, nil
		},
	}

	svc := newChangeCardService(repo, nil, nil)

	_, err := svc.ListChangeCards(ctx, ChangeCardFilters{ShowAll: true})
	if err != nil {
		t.Fatalf("ListChangeCards() error = %v", err)
	}
	if capturedFilter == nil {
		t.Fatal("expected filter to be passed to repository")
	}
	if !capturedFilter.IncludeTerminal {
		t.Error("expected IncludeTerminal to be true when ShowAll is true")
	}
}

func TestChangeCardService_ListChangeCards_ShowAllFalse_ExcludesTerminal(t *testing.T) {
	ctx := context.Background()

	var capturedFilter *repository.ChangeCardRepoFilter
	repo := &mockChangeCardRepo{
		listFn: func(ctx context.Context, filter *repository.ChangeCardRepoFilter) ([]*models.ChangeCard, error) {
			capturedFilter = filter
			return []*models.ChangeCard{}, nil
		},
	}

	svc := newChangeCardService(repo, nil, nil)

	_, err := svc.ListChangeCards(ctx, ChangeCardFilters{ShowAll: false})
	if err != nil {
		t.Fatalf("ListChangeCards() error = %v", err)
	}
	if capturedFilter == nil {
		t.Fatal("expected filter to be passed to repository")
	}
	if capturedFilter.IncludeTerminal {
		t.Error("expected IncludeTerminal to be false when ShowAll is false")
	}
}

func TestChangeCardService_UpdateChangeCard(t *testing.T) {
	ctx := context.Background()

	repo := &mockChangeCardRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			return &models.ChangeCard{BaseEntity: models.BaseEntity{ID: 1, Key: "CC-001", Title: "Old Title"}, Status: "proposed"}, nil
		},
		updateFn: func(ctx context.Context, card *models.ChangeCard) error {
			if card.Title != "New Title" {
				t.Errorf("expected title 'New Title', got %s", card.Title)
			}
			return nil
		},
	}

	svc := newChangeCardService(repo, nil, nil)

	newTitle := "New Title"
	card, err := svc.UpdateChangeCard(ctx, "CC-001", ChangeCardUpdates{Title: &newTitle})
	if err != nil {
		t.Fatalf("UpdateChangeCard() error = %v", err)
	}
	if card.Title != "New Title" {
		t.Errorf("expected title 'New Title', got %s", card.Title)
	}
}

func TestChangeCardService_UpdateChangeCard_FilePath(t *testing.T) {
	ctx := context.Background()

	var capturedCard *models.ChangeCard
	repo := &mockChangeCardRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			return &models.ChangeCard{BaseEntity: models.BaseEntity{ID: 1, Key: "CC-001", Title: "Change Card"}, Status: "proposed"}, nil
		},
		updateFn: func(ctx context.Context, card *models.ChangeCard) error {
			capturedCard = card
			return nil
		},
	}

	svc := newChangeCardService(repo, nil, nil)

	newPath := "docs/plan/changes/CC-001-custom.md"
	card, err := svc.UpdateChangeCard(ctx, "CC-001", ChangeCardUpdates{FilePath: &newPath})
	if err != nil {
		t.Fatalf("UpdateChangeCard() error = %v", err)
	}
	if card.FilePath == nil || *card.FilePath != newPath {
		t.Errorf("expected file_path %q, got %v", newPath, card.FilePath)
	}
	if capturedCard == nil {
		t.Fatal("expected Update to be called")
	}
	if capturedCard.FilePath == nil || *capturedCard.FilePath != newPath {
		t.Errorf("expected captured file_path %q, got %v", newPath, capturedCard.FilePath)
	}
}

func TestChangeCardService_DeleteChangeCard(t *testing.T) {
	ctx := context.Background()

	deleteCalled := false
	repo := &mockChangeCardRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			return &models.ChangeCard{BaseEntity: models.BaseEntity{ID: 1, Key: "CC-001", Title: "To Delete"}, Status: "proposed"}, nil
		},
		deleteFn: func(ctx context.Context, id int64) error {
			deleteCalled = true
			if id != 1 {
				t.Errorf("expected delete id=1, got %d", id)
			}
			return nil
		},
	}

	svc := newChangeCardService(repo, nil, nil)

	err := svc.DeleteChangeCard(ctx, "CC-001")
	if err != nil {
		t.Fatalf("DeleteChangeCard() error = %v", err)
	}
	if !deleteCalled {
		t.Error("expected delete to be called")
	}
}

func TestChangeCardService_TransitionStatus(t *testing.T) {
	ctx := context.Background()

	currentStatus := models.ChangeCardStatus("proposed")
	repo := &mockChangeCardRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			return &models.ChangeCard{BaseEntity: models.BaseEntity{ID: 1, Key: "CC-001", Title: "Test"}, Status: currentStatus}, nil
		},
		updateStatusFn: func(ctx context.Context, id int64, status models.ChangeCardStatus) error {
			currentStatus = status
			return nil
		},
	}

	svc := newChangeCardService(repo, nil, nil)

	result, err := svc.TransitionStatus(ctx, "CC-001", "approved", TransitionOptions{})
	if err != nil {
		t.Fatalf("TransitionStatus() error = %v", err)
	}
	if result.ToStatus != "approved" {
		t.Errorf("expected status 'approved', got %s", result.ToStatus)
	}
}

func TestChangeCardService_TransitionStatus_InvalidTransition(t *testing.T) {
	ctx := context.Background()

	repo := &mockChangeCardRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			return &models.ChangeCard{BaseEntity: models.BaseEntity{ID: 1, Key: "CC-001", Title: "Test"}, Status: "completed"}, nil
		},
	}

	svc := newChangeCardService(repo, nil, nil)

	// completed -> proposed should be invalid
	_, err := svc.TransitionStatus(ctx, "CC-001", "proposed", TransitionOptions{})
	if err == nil {
		t.Fatal("expected error for invalid transition")
	}
}

func TestChangeCardService_CountByStatus(t *testing.T) {
	ctx := context.Background()

	repo := &mockChangeCardRepo{
		countByStatusFn: func(ctx context.Context) (map[string]int, error) {
			return map[string]int{
				"proposed": 3,
				"approved": 2,
			}, nil
		},
	}

	svc := newChangeCardService(repo, nil, nil)

	counts, err := svc.CountByStatus(ctx)
	if err != nil {
		t.Fatalf("CountByStatus() error = %v", err)
	}
	if counts["proposed"] != 3 {
		t.Errorf("expected proposed=3, got %d", counts["proposed"])
	}
	if counts["approved"] != 2 {
		t.Errorf("expected approved=2, got %d", counts["approved"])
	}
}

func TestChangeCardService_GetNextStatus(t *testing.T) {
	ctx := context.Background()

	repo := &mockChangeCardRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			return &models.ChangeCard{BaseEntity: models.BaseEntity{ID: 1, Key: "CC-001", Title: "Test"}, Status: "proposed"}, nil
		},
	}

	svc := newChangeCardService(repo, nil, nil)

	info, err := svc.GetNextStatus(ctx, "CC-001")
	if err != nil {
		t.Fatalf("GetNextStatus() error = %v", err)
	}
	if info.CurrentStatus != "proposed" {
		t.Errorf("expected current status 'proposed', got %s", info.CurrentStatus)
	}
	if len(info.AvailableTransitions) == 0 {
		t.Error("expected at least one available transition from 'proposed'")
	}
}

func TestChangeCardService_GetNextStatus_Terminal(t *testing.T) {
	ctx := context.Background()

	repo := &mockChangeCardRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			return &models.ChangeCard{BaseEntity: models.BaseEntity{ID: 1, Key: "CC-001", Title: "Test"}, Status: "completed"}, nil
		},
	}

	svc := newChangeCardService(repo, nil, nil)

	info, err := svc.GetNextStatus(ctx, "CC-001")
	if err != nil {
		t.Fatalf("GetNextStatus() error = %v", err)
	}
	if len(info.AvailableTransitions) != 0 {
		t.Errorf("expected no available transitions for terminal status, got %d", len(info.AvailableTransitions))
	}
}
