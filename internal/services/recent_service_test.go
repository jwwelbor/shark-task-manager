package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// --- Mock repositories for RecentService ---

type mockRecentTaskRepo struct {
	GetRecentFunc func(ctx context.Context, limit int) ([]*models.Task, error)
	called        bool
}

func (m *mockRecentTaskRepo) GetRecent(ctx context.Context, limit int) ([]*models.Task, error) {
	m.called = true
	if m.GetRecentFunc != nil {
		return m.GetRecentFunc(ctx, limit)
	}
	return []*models.Task{}, nil
}

type mockRecentFeatureRepo struct {
	GetRecentFunc func(ctx context.Context, limit int) ([]*models.Feature, error)
	called        bool
}

func (m *mockRecentFeatureRepo) GetRecent(ctx context.Context, limit int) ([]*models.Feature, error) {
	m.called = true
	if m.GetRecentFunc != nil {
		return m.GetRecentFunc(ctx, limit)
	}
	return []*models.Feature{}, nil
}

type mockRecentEpicRepo struct {
	GetRecentFunc func(ctx context.Context, limit int) ([]*models.Epic, error)
	called        bool
}

func (m *mockRecentEpicRepo) GetRecent(ctx context.Context, limit int) ([]*models.Epic, error) {
	m.called = true
	if m.GetRecentFunc != nil {
		return m.GetRecentFunc(ctx, limit)
	}
	return []*models.Epic{}, nil
}

// --- Helpers for building test entities ---

func makeTask(key, title string, createdAt time.Time) *models.Task {
	return &models.Task{
		BaseEntity: models.BaseEntity{
			Key:       key,
			Title:     title,
			CreatedAt: createdAt,
		},
		Status: models.TaskStatus("todo"),
	}
}

func makeFeature(key, title string, createdAt time.Time) *models.Feature {
	return &models.Feature{
		BaseEntity: models.BaseEntity{
			Key:       key,
			Title:     title,
			CreatedAt: createdAt,
		},
		Status: models.FeatureStatusActive,
	}
}

func makeEpic(key, title string, createdAt time.Time) *models.Epic {
	return &models.Epic{
		BaseEntity: models.BaseEntity{
			Key:       key,
			Title:     title,
			CreatedAt: createdAt,
		},
		Status: models.EpicStatusActive,
	}
}

// --- Tests ---

// TestRecentService_DefaultFilters_IncludesAllTypes verifies that when no Include*
// flag is true in the filters, all three repos are called (AC-T1).
func TestRecentService_DefaultFilters_IncludesAllTypes(t *testing.T) {
	taskRepo := &mockRecentTaskRepo{}
	featureRepo := &mockRecentFeatureRepo{}
	epicRepo := &mockRecentEpicRepo{}

	svc := NewRecentService(taskRepo, featureRepo, epicRepo)

	filters := RecentFilters{
		Limit:           5,
		IncludeTasks:    false,
		IncludeFeatures: false,
		IncludeEpics:    false,
	}

	items, err := svc.ListRecent(context.Background(), filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !taskRepo.called {
		t.Error("expected task repo to be called when no type filter is set")
	}
	if !featureRepo.called {
		t.Error("expected feature repo to be called when no type filter is set")
	}
	if !epicRepo.called {
		t.Error("expected epic repo to be called when no type filter is set")
	}
	if items == nil {
		t.Error("expected non-nil slice, got nil")
	}
}

// TestRecentService_SingleTypeFilter_SkipsOtherRepos verifies that when only
// IncludeTasks=true, feature and epic repos are NOT called (AC-T2).
func TestRecentService_SingleTypeFilter_SkipsOtherRepos(t *testing.T) {
	now := time.Now().UTC()

	taskRepo := &mockRecentTaskRepo{
		GetRecentFunc: func(ctx context.Context, limit int) ([]*models.Task, error) {
			return []*models.Task{makeTask("E01-F01-001", "Task A", now)}, nil
		},
	}
	featureRepo := &mockRecentFeatureRepo{}
	epicRepo := &mockRecentEpicRepo{}

	svc := NewRecentService(taskRepo, featureRepo, epicRepo)

	filters := RecentFilters{
		Limit:           5,
		IncludeTasks:    true,
		IncludeFeatures: false,
		IncludeEpics:    false,
	}

	items, err := svc.ListRecent(context.Background(), filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !taskRepo.called {
		t.Error("expected task repo to be called")
	}
	if featureRepo.called {
		t.Error("expected feature repo NOT to be called when only IncludeTasks=true")
	}
	if epicRepo.called {
		t.Error("expected epic repo NOT to be called when only IncludeTasks=true")
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
	if items[0].Type != "task" {
		t.Errorf("expected type 'task', got %q", items[0].Type)
	}
}

// TestRecentService_MergesAndSortsAcrossTypes verifies that results from all three
// repos are merged and sorted by CreatedAt DESC (AC-T3).
func TestRecentService_MergesAndSortsAcrossTypes(t *testing.T) {
	base := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	t1 := base.Add(3 * time.Second) // newest
	t2 := base.Add(2 * time.Second)
	t3 := base.Add(1 * time.Second) // oldest

	taskRepo := &mockRecentTaskRepo{
		GetRecentFunc: func(ctx context.Context, limit int) ([]*models.Task, error) {
			return []*models.Task{makeTask("E01-F01-001", "Task A", t3)}, nil
		},
	}
	featureRepo := &mockRecentFeatureRepo{
		GetRecentFunc: func(ctx context.Context, limit int) ([]*models.Feature, error) {
			return []*models.Feature{makeFeature("E01-F01", "Feature B", t1)}, nil
		},
	}
	epicRepo := &mockRecentEpicRepo{
		GetRecentFunc: func(ctx context.Context, limit int) ([]*models.Epic, error) {
			return []*models.Epic{makeEpic("E01", "Epic C", t2)}, nil
		},
	}

	svc := NewRecentService(taskRepo, featureRepo, epicRepo)

	filters := RecentFilters{Limit: 10}
	items, err := svc.ListRecent(context.Background(), filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	// Should be sorted DESC: t1 (feature), t2 (epic), t3 (task)
	if !items[0].CreatedAt.Equal(t1) {
		t.Errorf("expected first item CreatedAt %v, got %v", t1, items[0].CreatedAt)
	}
	if !items[1].CreatedAt.Equal(t2) {
		t.Errorf("expected second item CreatedAt %v, got %v", t2, items[1].CreatedAt)
	}
	if !items[2].CreatedAt.Equal(t3) {
		t.Errorf("expected third item CreatedAt %v, got %v", t3, items[2].CreatedAt)
	}
}

// TestRecentService_AppliesFinalLimitAfterMerge verifies that the final limit is
// applied AFTER merging all repos, not per-repo (AC-T5).
func TestRecentService_AppliesFinalLimitAfterMerge(t *testing.T) {
	base := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)

	// 5 tasks, 5 features, 5 epics = 15 total; limit=4 should return 4
	makeTasks := func() []*models.Task {
		tasks := make([]*models.Task, 5)
		for i := 0; i < 5; i++ {
			tasks[i] = makeTask("E01-F01-00"+string(rune('1'+i)), "Task", base.Add(time.Duration(i)*time.Second))
		}
		return tasks
	}
	makeFeatures := func() []*models.Feature {
		features := make([]*models.Feature, 5)
		for i := 0; i < 5; i++ {
			features[i] = makeFeature("E01-F0"+string(rune('1'+i)), "Feature", base.Add(time.Duration(i+5)*time.Second))
		}
		return features
	}
	makeEpics := func() []*models.Epic {
		epics := make([]*models.Epic, 5)
		for i := 0; i < 5; i++ {
			epics[i] = makeEpic("E0"+string(rune('1'+i)), "Epic", base.Add(time.Duration(i+10)*time.Second))
		}
		return epics
	}

	taskRepo := &mockRecentTaskRepo{
		GetRecentFunc: func(ctx context.Context, limit int) ([]*models.Task, error) {
			return makeTasks(), nil
		},
	}
	featureRepo := &mockRecentFeatureRepo{
		GetRecentFunc: func(ctx context.Context, limit int) ([]*models.Feature, error) {
			return makeFeatures(), nil
		},
	}
	epicRepo := &mockRecentEpicRepo{
		GetRecentFunc: func(ctx context.Context, limit int) ([]*models.Epic, error) {
			return makeEpics(), nil
		},
	}

	svc := NewRecentService(taskRepo, featureRepo, epicRepo)

	filters := RecentFilters{Limit: 4}
	items, err := svc.ListRecent(context.Background(), filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 4 {
		t.Errorf("expected exactly 4 items after limit, got %d", len(items))
	}

	// The 4 most recent items should be from epics (they have the highest timestamps)
	for _, item := range items {
		if item.Type != "epic" {
			t.Errorf("expected top 4 items to be epics (highest timestamps), got type %q for key %q", item.Type, item.Key)
		}
	}
}

// TestRecentService_RepositoryErrorIsWrapped verifies that a single-repo error
// aborts the whole operation with the expected error wrap format (AC-T6).
func TestRecentService_RepositoryErrorIsWrapped(t *testing.T) {
	dbErr := errors.New("database unavailable")

	taskRepo := &mockRecentTaskRepo{
		GetRecentFunc: func(ctx context.Context, limit int) ([]*models.Task, error) {
			return nil, dbErr
		},
	}
	featureRepo := &mockRecentFeatureRepo{}
	epicRepo := &mockRecentEpicRepo{}

	svc := NewRecentService(taskRepo, featureRepo, epicRepo)

	filters := RecentFilters{Limit: 5}
	_, err := svc.ListRecent(context.Background(), filters)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	const wantSubstr = "failed to list recent task"
	if !strings.Contains(err.Error(), wantSubstr) {
		t.Errorf("expected error to contain %q, got: %q", wantSubstr, err.Error())
	}

	if !errors.Is(err, dbErr) {
		t.Errorf("expected wrapped error to unwrap to original dbErr")
	}
}

// TestRecentService_EmptyReposReturnEmptySlice verifies that when all repos return
// empty slices, the service returns []RecentItem{} (not nil) (AC-T7).
func TestRecentService_EmptyReposReturnEmptySlice(t *testing.T) {
	taskRepo := &mockRecentTaskRepo{
		GetRecentFunc: func(ctx context.Context, limit int) ([]*models.Task, error) {
			return []*models.Task{}, nil
		},
	}
	featureRepo := &mockRecentFeatureRepo{
		GetRecentFunc: func(ctx context.Context, limit int) ([]*models.Feature, error) {
			return []*models.Feature{}, nil
		},
	}
	epicRepo := &mockRecentEpicRepo{
		GetRecentFunc: func(ctx context.Context, limit int) ([]*models.Epic, error) {
			return []*models.Epic{}, nil
		},
	}

	svc := NewRecentService(taskRepo, featureRepo, epicRepo)

	filters := RecentFilters{Limit: 5}
	items, err := svc.ListRecent(context.Background(), filters)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if items == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

// TestRecentService_TieBreakByTypeThenKey verifies that items with identical
// CreatedAt are broken by type order (epic > feature > task) then key asc (AC-T4).
func TestRecentService_TieBreakByTypeThenKey(t *testing.T) {
	sameTime := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)

	taskRepo := &mockRecentTaskRepo{
		GetRecentFunc: func(ctx context.Context, limit int) ([]*models.Task, error) {
			return []*models.Task{makeTask("E01-F01-002", "Task B", sameTime)}, nil
		},
	}
	featureRepo := &mockRecentFeatureRepo{
		GetRecentFunc: func(ctx context.Context, limit int) ([]*models.Feature, error) {
			return []*models.Feature{makeFeature("E01-F01", "Feature A", sameTime)}, nil
		},
	}
	epicRepo := &mockRecentEpicRepo{
		GetRecentFunc: func(ctx context.Context, limit int) ([]*models.Epic, error) {
			return []*models.Epic{
				makeEpic("E01", "Epic Z", sameTime),
				makeEpic("E02", "Epic A", sameTime),
			}, nil
		},
	}

	svc := NewRecentService(taskRepo, featureRepo, epicRepo)

	filters := RecentFilters{Limit: 10}
	items, err := svc.ListRecent(context.Background(), filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 4 {
		t.Fatalf("expected 4 items, got %d", len(items))
	}

	// Order should be: E01 (epic), E02 (epic), E01-F01 (feature), E01-F01-002 (task)
	wantOrder := []struct {
		typ string
		key string
	}{
		{"epic", "E01"},
		{"epic", "E02"},
		{"feature", "E01-F01"},
		{"task", "E01-F01-002"},
	}

	for i, want := range wantOrder {
		if items[i].Type != want.typ {
			t.Errorf("item[%d]: expected type %q, got %q", i, want.typ, items[i].Type)
		}
		if items[i].Key != want.key {
			t.Errorf("item[%d]: expected key %q, got %q", i, want.key, items[i].Key)
		}
	}
}
