package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// --- Epic backward transition tests ---

func newTestEpicWorkflowServiceForBackward(t *testing.T) *workflow.Service {
	t.Helper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	configJSON := `{
		"epic_workflow": {
			"status_flow_version": "1.0",
			"require_rejection_reason": true,
			"status_flow": {
				"draft": ["active"],
				"active": ["draft", "completed", "archived"],
				"completed": ["archived"],
				"archived": []
			},
			"status_metadata": {
				"draft": {
					"color": "gray",
					"description": "Epic created",
					"phase": "planning"
				},
				"active": {
					"color": "blue",
					"description": "Epic in progress",
					"phase": "development"
				},
				"completed": {
					"color": "green",
					"description": "All features complete",
					"phase": "done"
				},
				"archived": {
					"color": "gray",
					"description": "Epic archived",
					"phase": "done"
				}
			},
			"special_statuses": {
				"_start_": ["draft"],
				"_complete_": ["completed", "archived"]
			}
		}
	}`

	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	config.ClearWorkflowCache()
	t.Cleanup(func() {
		config.ClearWorkflowCache()
	})

	return workflow.NewService(tmpDir)
}

func TestEpicService_BackwardTransition_RequiresReason(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				Key:    "E16",
				Status: models.EpicStatus("active"),
			}, nil
		},
	}

	svc := NewEpicService(repo, newTestEpicWorkflowServiceForBackward(t), nil, nil, nil)
	ctx := context.Background()

	// active -> draft is backward (execution -> planning)
	_, err := svc.TransitionStatus(ctx, "E16", "draft", TransitionOptions{})
	if err == nil {
		t.Fatal("expected error for backward transition without reason")
	}
	if !errors.Is(err, ErrReasonRequired) {
		t.Errorf("expected ErrReasonRequired, got: %v", err)
	}
	var backwardErr *BackwardReasonError
	if !errors.As(err, &backwardErr) {
		t.Errorf("expected BackwardReasonError, got %T: %v", err, err)
	}
}

func TestEpicService_BackwardTransition_WithReason(t *testing.T) {
	var updatedEpic *models.Epic
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				Key:    "E16",
				Status: models.EpicStatus("active"),
			}, nil
		},
		updateFn: func(ctx context.Context, epic *models.Epic) error {
			updatedEpic = epic
			return nil
		},
	}

	svc := NewEpicService(repo, newTestEpicWorkflowServiceForBackward(t), nil, nil, nil)
	ctx := context.Background()

	// active -> draft with reason should succeed (valid backward transition)
	result, err := svc.TransitionStatus(ctx, "E16", "draft", TransitionOptions{
		Reason: "Requirements changed",
	})
	if err != nil {
		t.Fatalf("expected no error with reason, got: %v", err)
	}

	if updatedEpic == nil {
		t.Fatal("expected Update to be called")
	}
	if result.Reason != "Requirements changed" {
		t.Errorf("expected reason 'Requirements changed', got %q", result.Reason)
	}
	if !result.IsBackward {
		t.Error("expected IsBackward=true")
	}
}

func TestEpicService_ForceTransition_RequiresReason(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				Key:    "E16",
				Status: models.EpicStatusDraft,
			}, nil
		},
	}

	svc := NewEpicService(repo, newTestEpicWorkflowServiceForBackward(t), nil, nil, nil)
	ctx := context.Background()

	// Force without reason should fail
	_, err := svc.TransitionStatus(ctx, "E16", "anything", TransitionOptions{Force: true})
	if err == nil {
		t.Fatal("expected error for force without reason")
	}
	if !errors.Is(err, ErrForceReasonRequired) {
		t.Errorf("expected ErrForceReasonRequired, got: %v", err)
	}
}

func TestEpicService_ForwardTransition_NoReasonRequired(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				Key:    "E16",
				Status: models.EpicStatusDraft,
			}, nil
		},
		updateFn: func(ctx context.Context, epic *models.Epic) error {
			return nil
		},
	}

	svc := NewEpicService(repo, newTestEpicWorkflowServiceForBackward(t), nil, nil, nil)
	ctx := context.Background()

	// Forward transition (draft -> active) should not require reason
	result, err := svc.TransitionStatus(ctx, "E16", "active", TransitionOptions{})
	if err != nil {
		t.Fatalf("expected no error for forward transition, got: %v", err)
	}

	if result.IsBackward {
		t.Error("expected IsBackward=false for forward transition")
	}
}

func TestEpicService_ChildCount_WithFeatureRepo(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				ID:     42,
				Key:    "E16",
				Status: models.EpicStatusDraft,
			}, nil
		},
		updateFn: func(ctx context.Context, epic *models.Epic) error {
			return nil
		},
	}

	featureCounter := &mockEpicFeatureCounter{
		listByEpicFn: func(ctx context.Context, epicID int64) ([]*models.Feature, error) {
			return []*models.Feature{
				{Key: "E16-F01"},
				{Key: "E16-F02"},
				{Key: "E16-F03"},
			}, nil
		},
	}

	svc := NewEpicService(repo, newTestEpicWorkflowServiceForBackward(t), nil, featureCounter, nil)
	ctx := context.Background()

	result, err := svc.TransitionStatus(ctx, "E16", "active", TransitionOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ChildCount != 3 {
		t.Errorf("expected child_count=3, got %d", result.ChildCount)
	}
}

// --- Feature backward transition tests ---

func newTestFeatureWorkflowServiceForBackward(t *testing.T) *workflow.Service {
	t.Helper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	configJSON := `{
		"feature_workflow": {
			"version": "1.0",
			"require_rejection_reason": true,
			"status_flow": {
				"draft": ["active"],
				"active": ["draft", "completed", "archived"],
				"completed": ["archived"],
				"archived": []
			},
			"status_metadata": {
				"draft": {
					"color": "gray",
					"description": "Feature created",
					"phase": "planning"
				},
				"active": {
					"color": "blue",
					"description": "Feature in progress",
					"phase": "development"
				},
				"completed": {
					"color": "green",
					"description": "All tasks done",
					"phase": "done"
				},
				"archived": {
					"color": "gray",
					"description": "Feature archived",
					"phase": "done"
				}
			},
			"special_statuses": {
				"_start_": ["draft"],
				"_complete_": ["completed", "archived"]
			}
		}
	}`

	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	config.ClearWorkflowCache()
	t.Cleanup(func() {
		config.ClearWorkflowCache()
	})

	return workflow.NewService(tmpDir)
}

func TestFeatureService_BackwardTransition_RequiresReason(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{
				Key:    "E16-F01",
				Status: models.FeatureStatus("active"),
			}, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowServiceForBackward(t), nil, nil, nil)
	ctx := context.Background()

	// active -> draft is backward
	_, err := svc.TransitionStatus(ctx, "E16-F01", "draft", TransitionOptions{})
	if err == nil {
		t.Fatal("expected error for backward transition without reason")
	}
	if !errors.Is(err, ErrReasonRequired) {
		t.Errorf("expected ErrReasonRequired, got: %v", err)
	}
	var backwardErr *BackwardReasonError
	if !errors.As(err, &backwardErr) {
		t.Errorf("expected BackwardReasonError, got %T: %v", err, err)
	}
}

func TestFeatureService_ForceTransition_RequiresReason(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{
				Key:    "E16-F01",
				Status: models.FeatureStatusDraft,
			}, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowServiceForBackward(t), nil, nil, nil)
	ctx := context.Background()

	// Force without reason should fail
	_, err := svc.TransitionStatus(ctx, "E16-F01", "anything", TransitionOptions{Force: true})
	if err == nil {
		t.Fatal("expected error for force without reason")
	}
	if !errors.Is(err, ErrForceReasonRequired) {
		t.Errorf("expected ErrForceReasonRequired, got: %v", err)
	}
}

func TestFeatureService_ForwardTransition_NoReasonRequired(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{
				Key:    "E16-F01",
				Status: models.FeatureStatusDraft,
			}, nil
		},
		updateFn: func(ctx context.Context, feature *models.Feature) error {
			return nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowServiceForBackward(t), nil, nil, nil)
	ctx := context.Background()

	result, err := svc.TransitionStatus(ctx, "E16-F01", "active", TransitionOptions{})
	if err != nil {
		t.Fatalf("expected no error for forward transition, got: %v", err)
	}

	if result.IsBackward {
		t.Error("expected IsBackward=false for forward transition")
	}
}

func TestFeatureService_ChildCount_WithTaskRepo(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{
				ID:     10,
				Key:    "E16-F01",
				Status: models.FeatureStatusDraft,
			}, nil
		},
		updateFn: func(ctx context.Context, feature *models.Feature) error {
			return nil
		},
	}

	taskCounter := &mockFeatureTaskCounter{
		listByFeatureFn: func(ctx context.Context, featureID int64) ([]*models.Task, error) {
			return []*models.Task{
				{Key: "E16-F01-001"},
				{Key: "E16-F01-002"},
			}, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowServiceForBackward(t), nil, taskCounter, nil)
	ctx := context.Background()

	result, err := svc.TransitionStatus(ctx, "E16-F01", "active", TransitionOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ChildCount != 2 {
		t.Errorf("expected child_count=2, got %d", result.ChildCount)
	}
}

// --- JSON serialization tests ---

func TestTransitionResult_BackwardFields_JSON(t *testing.T) {
	result := &TransitionResult{
		EntityType:   "epic",
		EntityKey:    "E16",
		FromStatus:   "active",
		ToStatus:     "draft",
		Transitioned: true,
		IsBackward:   true,
		IsForced:     true,
		Reason:       "Requirements changed",
		ChildCount:   5,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed TransitionResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if !parsed.IsBackward {
		t.Error("expected is_backward=true in JSON")
	}
	if !parsed.IsForced {
		t.Error("expected is_forced=true in JSON")
	}
	if parsed.Reason != "Requirements changed" {
		t.Errorf("expected reason 'Requirements changed', got %q", parsed.Reason)
	}
	if parsed.ChildCount != 5 {
		t.Errorf("expected child_count=5, got %d", parsed.ChildCount)
	}

	// Verify JSON field names
	jsonStr := string(data)
	expectedFields := []string{`"is_backward":true`, `"is_forced":true`, `"reason":"Requirements changed"`, `"child_count":5`}
	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("expected JSON to contain %s, got: %s", field, jsonStr)
		}
	}
}

func TestTransitionOptions_JSON(t *testing.T) {
	opts := TransitionOptions{
		Force:  true,
		Reason: "Test reason",
		Agent:  "developer",
	}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed TransitionOptions
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if !parsed.Force {
		t.Error("expected Force=true")
	}
	if parsed.Reason != "Test reason" {
		t.Errorf("expected reason 'Test reason', got %q", parsed.Reason)
	}
	if parsed.Agent != "developer" {
		t.Errorf("expected agent 'developer', got %q", parsed.Agent)
	}
}

func TestTransitionResult_OmitsEmptyBackwardFields(t *testing.T) {
	// Forward transition should omit backward-specific fields
	result := &TransitionResult{
		EntityType:   "epic",
		EntityKey:    "E16",
		FromStatus:   "draft",
		ToStatus:     "active",
		Transitioned: true,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(data)
	// These fields should be omitted (omitempty)
	omittedFields := []string{`"is_backward"`, `"is_forced"`, `"reason"`, `"child_count"`}
	for _, field := range omittedFields {
		if strings.Contains(jsonStr, field) {
			t.Errorf("expected JSON to omit %s for forward transition, got: %s", field, jsonStr)
		}
	}
}

// --- Mock helpers ---

type mockEpicFeatureCounter struct {
	listByEpicFn             func(ctx context.Context, epicID int64) ([]*models.Feature, error)
	getByIDFn                func(ctx context.Context, id int64) (*models.Feature, error)
	updateFn                 func(ctx context.Context, feature *models.Feature) error
	getTaskStatusBreakdownFn func(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error)
}

func (m *mockEpicFeatureCounter) ListByEpic(ctx context.Context, epicID int64) ([]*models.Feature, error) {
	if m.listByEpicFn != nil {
		return m.listByEpicFn(ctx, epicID)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockEpicFeatureCounter) GetByID(ctx context.Context, id int64) (*models.Feature, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockEpicFeatureCounter) Update(ctx context.Context, feature *models.Feature) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, feature)
	}
	return nil
}

func (m *mockEpicFeatureCounter) GetTaskStatusBreakdown(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
	if m.getTaskStatusBreakdownFn != nil {
		return m.getTaskStatusBreakdownFn(ctx, featureID)
	}
	return nil, nil
}

type mockFeatureTaskCounter struct {
	listByFeatureFn      func(ctx context.Context, featureID int64) ([]*models.Task, error)
	updateStatusForcedFn func(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string, rejectionReason *string, documentPath *string, force bool) error
}

func (m *mockFeatureTaskCounter) ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error) {
	if m.listByFeatureFn != nil {
		return m.listByFeatureFn(ctx, featureID)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockFeatureTaskCounter) UpdateStatusForced(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string, rejectionReason *string, documentPath *string, force bool) error {
	if m.updateStatusForcedFn != nil {
		return m.updateStatusForcedFn(ctx, taskID, newStatus, agent, notes, rejectionReason, documentPath, force)
	}
	return nil
}

func (m *mockFeatureTaskCounter) GetStatusBreakdownMapBatch(ctx context.Context, featureIDs []int64) (map[int64]map[models.TaskStatus]int, error) {
	return nil, nil
}
