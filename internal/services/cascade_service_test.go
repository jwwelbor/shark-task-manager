package services_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// -------------------------------------------------------------------------
// Mocks (function-field pattern per .claude/rules/services/testing.md)
// -------------------------------------------------------------------------

type mockCascadeTaskRepo struct {
	ListByFeatureKeyFunc    func(ctx context.Context, featureKey string) ([]*models.Task, error)
	GetTaskDependenciesFunc func(ctx context.Context, taskKey string) ([]*models.Task, error)
}

func (m *mockCascadeTaskRepo) ListByFeatureKey(ctx context.Context, featureKey string) ([]*models.Task, error) {
	if m.ListByFeatureKeyFunc != nil {
		return m.ListByFeatureKeyFunc(ctx, featureKey)
	}
	return nil, fmt.Errorf("ListByFeatureKey not implemented in mock")
}

func (m *mockCascadeTaskRepo) GetTaskDependencies(ctx context.Context, taskKey string) ([]*models.Task, error) {
	if m.GetTaskDependenciesFunc != nil {
		return m.GetTaskDependenciesFunc(ctx, taskKey)
	}
	return []*models.Task{}, nil
}

type mockCascadeEpicLookup struct {
	GetByKeyFunc func(ctx context.Context, key string) (*models.Epic, error)
}

func (m *mockCascadeEpicLookup) GetByKey(ctx context.Context, key string) (*models.Epic, error) {
	if m.GetByKeyFunc != nil {
		return m.GetByKeyFunc(ctx, key)
	}
	return nil, fmt.Errorf("GetByKey not implemented in mock")
}

type mockCascadeFeatureLister struct {
	ListByEpicFunc func(ctx context.Context, epicID int64) ([]*models.Feature, error)
}

func (m *mockCascadeFeatureLister) ListByEpic(ctx context.Context, epicID int64) ([]*models.Feature, error) {
	if m.ListByEpicFunc != nil {
		return m.ListByEpicFunc(ctx, epicID)
	}
	return nil, fmt.Errorf("ListByEpic not implemented in mock")
}

// b029CustomWorkflowConfig is a multi-level workflow whose task / feature /
// epic terminal status is renamed to "shipped". Used to verify that
// CascadeService.DescribeDispatchableChildren consults
// workflow.Service.IsTerminalStatus rather than any hardcoded list.
const b029CustomWorkflowConfig = `{
  "task_workflow": {
    "statuses": ["todo", "in_progress", "shipped"],
    "status_flow": {
      "todo": ["in_progress"],
      "in_progress": ["shipped"],
      "shipped": []
    },
    "special_statuses": {
      "_start_": ["todo"],
      "_complete_": ["shipped"]
    },
    "status_metadata": {
      "todo": {"color": "gray", "phase": "planning"},
      "in_progress": {"color": "blue", "phase": "development"},
      "shipped": {"color": "green", "phase": "done"}
    }
  },
  "feature_workflow": {
    "statuses": ["draft", "active", "shipped"],
    "status_flow": {
      "draft": ["active"],
      "active": ["shipped"],
      "shipped": []
    },
    "special_statuses": {
      "_start_": ["draft"],
      "_complete_": ["shipped"]
    },
    "status_metadata": {
      "draft": {"color": "gray", "phase": "planning"},
      "active": {"color": "blue", "phase": "development"},
      "shipped": {"color": "green", "phase": "done"}
    }
  },
  "epic_workflow": {
    "statuses": ["draft", "active", "shipped"],
    "status_flow": {
      "draft": ["active"],
      "active": ["shipped"],
      "shipped": []
    },
    "special_statuses": {
      "_start_": ["draft"],
      "_complete_": ["shipped"]
    },
    "status_metadata": {
      "draft": {"color": "gray", "phase": "planning"},
      "active": {"color": "blue", "phase": "development"},
      "shipped": {"color": "green", "phase": "done"}
    }
  }
}`

func newWorkflowService(t *testing.T, configBody string) *workflow.Service {
	t.Helper()
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, ".sharkconfig.json"), []byte(configBody), 0o644); err != nil {
		t.Fatalf("failed to write workflow config: %v", err)
	}
	return workflow.NewService(tmp)
}

// -------------------------------------------------------------------------
// Tests
// -------------------------------------------------------------------------

// TestCascadeService_FeatureChildren_FiltersTerminal verifies the regression
// fix for B029: cascade business logic now lives in CascadeService, repos are
// consulted via the service (not constructed from the CLI), and the terminal
// filter delegates to workflow.Service for B028 compliance.
func TestCascadeService_FeatureChildren_FiltersTerminal(t *testing.T) {
	wf := newWorkflowService(t, b029CustomWorkflowConfig)

	var capturedFeatureKey string
	taskRepo := &mockCascadeTaskRepo{
		ListByFeatureKeyFunc: func(ctx context.Context, featureKey string) ([]*models.Task, error) {
			capturedFeatureKey = featureKey
			return []*models.Task{
				{BaseEntity: models.BaseEntity{Key: "E07-F01-001"}, Status: models.TaskStatus("todo")},
				{BaseEntity: models.BaseEntity{Key: "E07-F01-002"}, Status: models.TaskStatus("shipped")}, // terminal — must be filtered
				{BaseEntity: models.BaseEntity{Key: "E07-F01-003"}, Status: models.TaskStatus("in_progress")},
				{BaseEntity: models.BaseEntity{Key: "E07-F01-004"}, Status: models.TaskStatus("shipped")}, // terminal — must be filtered
			}, nil
		},
	}
	svc := services.NewCascadeService(taskRepo, &mockCascadeEpicLookup{}, &mockCascadeFeatureLister{}, wf)

	state, err := svc.DescribeDispatchableChildren(context.Background(), "feature", "E07-F01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedFeatureKey != "E07-F01" {
		t.Errorf("expected feature key passed through to repo as %q, got %q", "E07-F01", capturedFeatureKey)
	}
	if state.TotalChildren != 4 {
		t.Errorf("expected TotalChildren=4 (every task counted), got %d", state.TotalChildren)
	}
	if state.NonTerminalChildren != 2 {
		t.Errorf("expected NonTerminalChildren=2 (terminals excluded), got %d", state.NonTerminalChildren)
	}
	out := state.Children
	if len(out) != 2 {
		t.Fatalf("expected 2 dispatchable tasks after filtering terminals, got %d: %+v", len(out), out)
	}
	if out[0].Key != "E07-F01-001" || out[0].EntityType != "task" {
		t.Errorf("expected first child = (E07-F01-001, task); got %+v", out[0])
	}
	if out[1].Key != "E07-F01-003" || out[1].EntityType != "task" {
		t.Errorf("expected second child = (E07-F01-003, task); got %+v", out[1])
	}
}

func TestCascadeService_FeatureChildren_SkipsUnmetDependencies(t *testing.T) {
	wf := newWorkflowService(t, b029CustomWorkflowConfig)

	taskRepo := &mockCascadeTaskRepo{
		ListByFeatureKeyFunc: func(ctx context.Context, featureKey string) ([]*models.Task, error) {
			return []*models.Task{
				{BaseEntity: models.BaseEntity{Key: "E07-F01-001"}, Status: models.TaskStatus("todo")},
				{BaseEntity: models.BaseEntity{Key: "E07-F01-002"}, Status: models.TaskStatus("todo")},
				{BaseEntity: models.BaseEntity{Key: "E07-F01-003"}, Status: models.TaskStatus("todo")},
			}, nil
		},
		GetTaskDependenciesFunc: func(ctx context.Context, taskKey string) ([]*models.Task, error) {
			switch taskKey {
			case "E07-F01-001":
				return []*models.Task{{BaseEntity: models.BaseEntity{Key: "E07-F01-000"}, Status: models.TaskStatus("todo")}}, nil
			case "E07-F01-002":
				return []*models.Task{{BaseEntity: models.BaseEntity{Key: "E07-F01-000"}, Status: models.TaskStatus("completed")}}, nil
			case "E07-F01-003":
				return []*models.Task{{BaseEntity: models.BaseEntity{Key: "E07-F01-000"}, Status: models.TaskStatus("archived")}}, nil
			default:
				return nil, fmt.Errorf("unexpected task key %s", taskKey)
			}
		},
	}
	svc := services.NewCascadeService(taskRepo, &mockCascadeEpicLookup{}, &mockCascadeFeatureLister{}, wf)

	state, err := svc.DescribeDispatchableChildren(context.Background(), "feature", "E07-F01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The critical distinction: a dependency-blocked task is NOT dispatchable
	// but IS non-terminal — its work is unfinished, so the parent must not be
	// treated as complete (counts feed tryCascade's auto-advance decision).
	if state.TotalChildren != 3 {
		t.Errorf("expected TotalChildren=3, got %d", state.TotalChildren)
	}
	if state.NonTerminalChildren != 3 {
		t.Errorf("expected NonTerminalChildren=3 (dependency-blocked tasks still count), got %d", state.NonTerminalChildren)
	}
	out := state.Children
	if len(out) != 2 {
		t.Fatalf("expected 2 dependency-ready tasks, got %d: %+v", len(out), out)
	}
	if out[0].Key != "E07-F01-002" {
		t.Errorf("expected first ready child E07-F01-002, got %+v", out[0])
	}
	if out[1].Key != "E07-F01-003" {
		t.Errorf("expected second ready child E07-F01-003, got %+v", out[1])
	}
}

// TestCascadeService_EpicChildren_FiltersTerminal verifies the epic branch:
// resolve key → list features → filter terminals using the feature-level
// workflow.
func TestCascadeService_EpicChildren_FiltersTerminal(t *testing.T) {
	wf := newWorkflowService(t, b029CustomWorkflowConfig)

	epicRepo := &mockCascadeEpicLookup{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Epic, error) {
			if key != "E07" {
				t.Errorf("expected epic key %q, got %q", "E07", key)
			}
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 42, Key: "E07"}}, nil
		},
	}
	var capturedEpicID int64
	featureRepo := &mockCascadeFeatureLister{
		ListByEpicFunc: func(ctx context.Context, epicID int64) ([]*models.Feature, error) {
			capturedEpicID = epicID
			return []*models.Feature{
				{BaseEntity: models.BaseEntity{Key: "E07-F01"}, Status: models.FeatureStatus("active")},
				{BaseEntity: models.BaseEntity{Key: "E07-F02"}, Status: models.FeatureStatus("shipped")}, // terminal — must be filtered
				{BaseEntity: models.BaseEntity{Key: "E07-F03"}, Status: models.FeatureStatus("draft")},
			}, nil
		},
	}
	svc := services.NewCascadeService(&mockCascadeTaskRepo{}, epicRepo, featureRepo, wf)

	state, err := svc.DescribeDispatchableChildren(context.Background(), "epic", "E07")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedEpicID != 42 {
		t.Errorf("expected ListByEpic called with id=42 (from GetByKey), got %d", capturedEpicID)
	}
	if state.TotalChildren != 3 {
		t.Errorf("expected TotalChildren=3, got %d", state.TotalChildren)
	}
	if state.NonTerminalChildren != 2 {
		t.Errorf("expected NonTerminalChildren=2, got %d", state.NonTerminalChildren)
	}
	out := state.Children
	if len(out) != 2 {
		t.Fatalf("expected 2 dispatchable features after filtering terminals, got %d: %+v", len(out), out)
	}
	if out[0].Key != "E07-F01" || out[0].EntityType != "feature" {
		t.Errorf("expected first child = (E07-F01, feature); got %+v", out[0])
	}
	if out[1].Key != "E07-F03" || out[1].EntityType != "feature" {
		t.Errorf("expected second child = (E07-F03, feature); got %+v", out[1])
	}
}

// TestCascadeService_LeafEntity_NoChildren verifies that leaf entity types
// (task, bug, change-card, tech-debt) return (nil, nil) rather than an error
// so the caller can treat "no children" as a pause.
func TestCascadeService_LeafEntity_NoChildren(t *testing.T) {
	wf := newWorkflowService(t, b029CustomWorkflowConfig)
	svc := services.NewCascadeService(&mockCascadeTaskRepo{}, &mockCascadeEpicLookup{}, &mockCascadeFeatureLister{}, wf)

	for _, et := range []string{"task", "bug", "change", "tech-debt", "unknown"} {
		t.Run(et, func(t *testing.T) {
			state, err := svc.DescribeDispatchableChildren(context.Background(), et, "X")
			if err != nil {
				t.Errorf("expected no error for leaf entity type %q, got %v", et, err)
			}
			if state.Children != nil {
				t.Errorf("expected nil children for leaf entity type %q, got %+v", et, state.Children)
			}
			if state.TotalChildren != 0 || state.NonTerminalChildren != 0 {
				t.Errorf("expected zero counts for leaf entity type %q, got %+v", et, state)
			}
		})
	}
}

// TestCascadeService_FeatureChildren_RepoError ensures repository errors are
// wrapped with business context (feature key) rather than swallowed.
func TestCascadeService_FeatureChildren_RepoError(t *testing.T) {
	wf := newWorkflowService(t, b029CustomWorkflowConfig)
	sentinel := errors.New("boom")
	taskRepo := &mockCascadeTaskRepo{
		ListByFeatureKeyFunc: func(ctx context.Context, featureKey string) ([]*models.Task, error) {
			return nil, sentinel
		},
	}
	svc := services.NewCascadeService(taskRepo, &mockCascadeEpicLookup{}, &mockCascadeFeatureLister{}, wf)

	_, err := svc.DescribeDispatchableChildren(context.Background(), "feature", "E07-F01")
	if err == nil {
		t.Fatal("expected error from repo failure, got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("expected wrapped sentinel error, got %v", err)
	}
}

// TestCascadeService_EpicChildren_GetByKeyError ensures GetByKey errors are
// wrapped and propagated.
func TestCascadeService_EpicChildren_GetByKeyError(t *testing.T) {
	wf := newWorkflowService(t, b029CustomWorkflowConfig)
	sentinel := errors.New("epic not found")
	epicRepo := &mockCascadeEpicLookup{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, sentinel
		},
	}
	svc := services.NewCascadeService(&mockCascadeTaskRepo{}, epicRepo, &mockCascadeFeatureLister{}, wf)

	_, err := svc.DescribeDispatchableChildren(context.Background(), "epic", "E99")
	if err == nil {
		t.Fatal("expected error from GetByKey failure, got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("expected wrapped sentinel error, got %v", err)
	}
}

// TestCascadeService_EpicChildren_ListByEpicError ensures ListByEpic errors
// are wrapped and propagated.
func TestCascadeService_EpicChildren_ListByEpicError(t *testing.T) {
	wf := newWorkflowService(t, b029CustomWorkflowConfig)
	epicRepo := &mockCascadeEpicLookup{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 7, Key: "E07"}}, nil
		},
	}
	sentinel := errors.New("list failed")
	featureRepo := &mockCascadeFeatureLister{
		ListByEpicFunc: func(ctx context.Context, epicID int64) ([]*models.Feature, error) {
			return nil, sentinel
		},
	}
	svc := services.NewCascadeService(&mockCascadeTaskRepo{}, epicRepo, featureRepo, wf)

	_, err := svc.DescribeDispatchableChildren(context.Background(), "epic", "E07")
	if err == nil {
		t.Fatal("expected error from ListByEpic failure, got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("expected wrapped sentinel error, got %v", err)
	}
}

// TestCascadeService_FeatureChildren_OrderingPreserved asserts the contract
// that the repository's query order is passed through untouched.
func TestCascadeService_FeatureChildren_OrderingPreserved(t *testing.T) {
	wf := newWorkflowService(t, b029CustomWorkflowConfig)
	taskRepo := &mockCascadeTaskRepo{
		ListByFeatureKeyFunc: func(ctx context.Context, featureKey string) ([]*models.Task, error) {
			// Simulate repo's "execution_order ASC NULLS LAST" ordering by
			// returning a specific sequence and expecting the service to
			// preserve it.
			return []*models.Task{
				{BaseEntity: models.BaseEntity{Key: "E07-F01-003"}, Status: models.TaskStatus("in_progress")},
				{BaseEntity: models.BaseEntity{Key: "E07-F01-001"}, Status: models.TaskStatus("todo")},
				{BaseEntity: models.BaseEntity{Key: "E07-F01-002"}, Status: models.TaskStatus("todo")},
			}, nil
		},
	}
	svc := services.NewCascadeService(taskRepo, &mockCascadeEpicLookup{}, &mockCascadeFeatureLister{}, wf)

	state, err := svc.DescribeDispatchableChildren(context.Background(), "feature", "E07-F01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := state.Children
	wantOrder := []string{"E07-F01-003", "E07-F01-001", "E07-F01-002"}
	if len(out) != len(wantOrder) {
		t.Fatalf("expected %d children, got %d", len(wantOrder), len(out))
	}
	for i, want := range wantOrder {
		if out[i].Key != want {
			t.Errorf("position %d: expected key %q, got %q", i, want, out[i].Key)
		}
	}
}
