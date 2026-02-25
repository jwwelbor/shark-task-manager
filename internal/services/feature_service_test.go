package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// mockFeatureRepo implements FeatureRepository for testing.
type mockFeatureRepo struct {
	getByKeyFn                    func(ctx context.Context, key string) (*models.Feature, error)
	getByIDFn                     func(ctx context.Context, id int64) (*models.Feature, error)
	createFn                      func(ctx context.Context, feature *models.Feature) error
	updateFn                      func(ctx context.Context, feature *models.Feature) error
	deleteFn                      func(ctx context.Context, id int64) error
	listFn                        func(ctx context.Context) ([]*models.Feature, error)
	listByEpicFn                  func(ctx context.Context, epicID int64) ([]*models.Feature, error)
	getTaskStatusBreakdownFn      func(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error)
	cascadeStatusToTasksFn        func(ctx context.Context, featureID int64, targetTaskStatus models.TaskStatus) error
	updateKeyFn                   func(ctx context.Context, oldKey string, newKey string) error
	getTaskCountFn                func(ctx context.Context, featureID int64) (int, error)
	setStatusOverrideFn           func(ctx context.Context, featureID int64, override bool) error
	updateStatusIfNotOverriddenFn func(ctx context.Context, featureID int64, newStatus models.FeatureStatus) (bool, error)
	updateFilePathFn              func(ctx context.Context, featureKey string, newFilePath *string) error
	getByFilePathFn               func(ctx context.Context, filePath string) (*models.Feature, error)
}

// mockFeatureEpicLookup implements FeatureEpicLookup for testing.
type mockFeatureEpicLookup struct {
	getByKeyFn      func(ctx context.Context, key string) (*models.Epic, error)
	getByFilePathFn func(ctx context.Context, filePath string) (*models.Epic, error)
}

func (m *mockFeatureEpicLookup) GetByKey(ctx context.Context, key string) (*models.Epic, error) {
	if m.getByKeyFn != nil {
		return m.getByKeyFn(ctx, key)
	}
	return nil, nil
}

func (m *mockFeatureEpicLookup) GetByFilePath(ctx context.Context, filePath string) (*models.Epic, error) {
	if m.getByFilePathFn != nil {
		return m.getByFilePathFn(ctx, filePath)
	}
	return nil, nil
}

func (m *mockFeatureEpicLookup) UpdateFilePath(ctx context.Context, epicKey string, newFilePath *string) error {
	return nil
}

func (m *mockFeatureEpicLookup) List(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error) {
	return nil, nil
}

func (m *mockFeatureRepo) GetByKey(ctx context.Context, key string) (*models.Feature, error) {
	if m.getByKeyFn != nil {
		return m.getByKeyFn(ctx, key)
	}
	return nil, nil
}

func (m *mockFeatureRepo) GetByID(ctx context.Context, id int64) (*models.Feature, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockFeatureRepo) Update(ctx context.Context, feature *models.Feature) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, feature)
	}
	return nil
}

func (m *mockFeatureRepo) List(ctx context.Context) ([]*models.Feature, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return nil, nil
}

func (m *mockFeatureRepo) ListByEpic(ctx context.Context, epicID int64) ([]*models.Feature, error) {
	if m.listByEpicFn != nil {
		return m.listByEpicFn(ctx, epicID)
	}
	return nil, nil
}

func (m *mockFeatureRepo) GetTaskStatusBreakdown(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
	if m.getTaskStatusBreakdownFn != nil {
		return m.getTaskStatusBreakdownFn(ctx, featureID)
	}
	return nil, nil
}

func (m *mockFeatureRepo) Create(ctx context.Context, feature *models.Feature) error {
	if m.createFn != nil {
		return m.createFn(ctx, feature)
	}
	return nil
}

func (m *mockFeatureRepo) Delete(ctx context.Context, id int64) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockFeatureRepo) GetByFilePath(ctx context.Context, filePath string) (*models.Feature, error) {
	if m.getByFilePathFn != nil {
		return m.getByFilePathFn(ctx, filePath)
	}
	return nil, nil
}

func (m *mockFeatureRepo) UpdateFilePath(ctx context.Context, featureKey string, newFilePath *string) error {
	if m.updateFilePathFn != nil {
		return m.updateFilePathFn(ctx, featureKey, newFilePath)
	}
	return nil
}

func (m *mockFeatureRepo) ListByEpicAndStatus(ctx context.Context, epicID int64, status models.FeatureStatus) ([]*models.Feature, error) {
	return nil, nil
}

func (m *mockFeatureRepo) UpdateKey(ctx context.Context, oldKey string, newKey string) error {
	if m.updateKeyFn != nil {
		return m.updateKeyFn(ctx, oldKey, newKey)
	}
	return nil
}

func (m *mockFeatureRepo) GetTaskCount(ctx context.Context, featureID int64) (int, error) {
	if m.getTaskCountFn != nil {
		return m.getTaskCountFn(ctx, featureID)
	}
	return 0, nil
}

func (m *mockFeatureRepo) SetStatusOverride(ctx context.Context, featureID int64, override bool) error {
	if m.setStatusOverrideFn != nil {
		return m.setStatusOverrideFn(ctx, featureID, override)
	}
	return nil
}

func (m *mockFeatureRepo) UpdateStatusIfNotOverridden(ctx context.Context, featureID int64, newStatus models.FeatureStatus) (bool, error) {
	if m.updateStatusIfNotOverriddenFn != nil {
		return m.updateStatusIfNotOverriddenFn(ctx, featureID, newStatus)
	}
	return true, nil
}

func (m *mockFeatureRepo) CascadeStatusToTasks(ctx context.Context, featureID int64, targetTaskStatus models.TaskStatus) error {
	if m.cascadeStatusToTasksFn != nil {
		return m.cascadeStatusToTasksFn(ctx, featureID, targetTaskStatus)
	}
	return nil
}

// Suppress unused import warnings
var _ = time.Now

func newTestFeatureWorkflowService() *workflow.Service {
	return workflow.NewService("")
}

func TestFeatureService_TransitionStatus_Valid(t *testing.T) {
	var updatedFeature *models.Feature
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{
				Key:    "E16-F01",
				Status: models.FeatureStatusDraft,
			}, nil
		},
		updateFn: func(ctx context.Context, feature *models.Feature) error {
			updatedFeature = feature
			return nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	result, err := svc.TransitionStatus(ctx, "E16-F01", "active", TransitionOptions{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.EntityType != "feature" {
		t.Errorf("expected entity_type 'feature', got %q", result.EntityType)
	}
	if result.EntityKey != "E16-F01" {
		t.Errorf("expected entity_key 'E16-F01', got %q", result.EntityKey)
	}
	if result.FromStatus != "draft" {
		t.Errorf("expected from_status 'draft', got %q", result.FromStatus)
	}
	if result.ToStatus != "active" {
		t.Errorf("expected to_status 'active', got %q", result.ToStatus)
	}
	if !result.Transitioned {
		t.Error("expected transitioned=true")
	}
	if updatedFeature == nil {
		t.Fatal("expected Update to be called")
	}
	if string(updatedFeature.Status) != "active" {
		t.Errorf("expected feature status 'active', got %q", updatedFeature.Status)
	}
}

func TestFeatureService_TransitionStatus_Invalid(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{
				Key:    "E16-F01",
				Status: models.FeatureStatusDraft,
			}, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	// "draft" -> "completed" is not valid in default feature workflow
	_, err := svc.TransitionStatus(ctx, "E16-F01", "completed", TransitionOptions{})
	if err == nil {
		t.Fatal("expected error for invalid transition")
	}
}

func TestFeatureService_TransitionStatus_Force(t *testing.T) {
	var updatedFeature *models.Feature
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{
				Key:    "E16-F01",
				Status: models.FeatureStatusDraft,
			}, nil
		},
		updateFn: func(ctx context.Context, feature *models.Feature) error {
			updatedFeature = feature
			return nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	result, err := svc.TransitionStatus(ctx, "E16-F01", "custom_status", TransitionOptions{Force: true, Reason: "test force override"})
	if err != nil {
		t.Fatalf("expected no error with force, got: %v", err)
	}
	if result.ToStatus != "custom_status" {
		t.Errorf("expected to_status 'custom_status', got %q", result.ToStatus)
	}
	if updatedFeature == nil {
		t.Fatal("expected Update to be called")
	}
}

func TestFeatureService_TransitionStatus_NotFound(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return nil, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	_, err := svc.TransitionStatus(ctx, "E99-F01", "active", TransitionOptions{})
	if err == nil {
		t.Fatal("expected error for not-found feature")
	}
	if err.Error() != "feature not found: E99-F01" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestFeatureService_TransitionStatus_RepoError(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return nil, fmt.Errorf("db connection failed")
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	_, err := svc.TransitionStatus(ctx, "E16-F01", "active", TransitionOptions{})
	if err == nil {
		t.Fatal("expected error from repo failure")
	}
}

func TestFeatureService_TransitionStatus_UpdateError(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{
				Key:    "E16-F01",
				Status: models.FeatureStatusDraft,
			}, nil
		},
		updateFn: func(ctx context.Context, feature *models.Feature) error {
			return fmt.Errorf("update failed")
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	_, err := svc.TransitionStatus(ctx, "E16-F01", "active", TransitionOptions{})
	if err == nil {
		t.Fatal("expected error from update failure")
	}
}

func TestFeatureService_GetNextStatus(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{
				Key:    "E16-F01",
				Status: models.FeatureStatusDraft,
			}, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	info, err := svc.GetNextStatus(ctx, "E16-F01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if info.EntityType != "feature" {
		t.Errorf("expected entity_type 'feature', got %q", info.EntityType)
	}
	if info.CurrentStatus != "draft" {
		t.Errorf("expected current_status 'draft', got %q", info.CurrentStatus)
	}
	if info.IsTerminal {
		t.Error("expected IsTerminal=false for draft status")
	}
	if len(info.AvailableTransitions) == 0 {
		t.Error("expected available transitions for draft status")
	}
}

func TestFeatureService_GetNextStatus_Terminal(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{
				Key:    "E16-F01",
				Status: models.FeatureStatusArchived,
			}, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	info, err := svc.GetNextStatus(ctx, "E16-F01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !info.IsTerminal {
		t.Error("expected IsTerminal=true for archived status")
	}
	if len(info.AvailableTransitions) != 0 {
		t.Errorf("expected no transitions for terminal status, got %d", len(info.AvailableTransitions))
	}
}

func TestFeatureService_GetNextStatus_NotFound(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return nil, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	_, err := svc.GetNextStatus(ctx, "E99-F01")
	if err == nil {
		t.Fatal("expected error for not-found feature")
	}
}

func TestFeatureService_ValidateStatus(t *testing.T) {
	repo := &mockFeatureRepo{}
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	// Valid feature statuses
	for _, status := range []string{"draft", "active", "completed", "archived"} {
		if err := svc.ValidateStatus(status); err != nil {
			t.Errorf("expected %q to be valid, got error: %v", status, err)
		}
	}

	// Invalid status
	if err := svc.ValidateStatus("in_progress"); err == nil {
		t.Error("expected 'in_progress' to be invalid for feature workflow")
	}
}

// newTestFeatureWorkflowServiceWithActions creates a workflow.Service whose feature workflow
// has orchestrator_action defined on the "active" status but not on "draft".
// This enables tests for resolveAction with and without actions.
func newTestFeatureWorkflowServiceWithActions(t *testing.T) *workflow.Service {
	t.Helper()

	// Clear the workflow cache so our temp config is loaded fresh
	config.ClearWorkflowCache()

	tmpDir := t.TempDir()
	configContent := `{
  "feature_workflow": {
    "version": "1.0",
    "status_flow": {
      "draft": ["active", "archived"],
      "active": ["completed", "archived"],
      "completed": ["archived"],
      "archived": []
    },
    "status_metadata": {
      "draft": {
        "color": "gray",
        "description": "Feature created, not yet started",
        "phase": "planning"
      },
      "active": {
        "color": "blue",
        "description": "Feature in progress",
        "phase": "execution",
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "developer",
          "skills": ["implementation", "test-driven-development"],
          "instruction_template": "Implement feature {id} following TDD approach."
        }
      },
      "completed": {
        "color": "green",
        "description": "All tasks complete",
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
      "_complete_": ["completed", "archived"],
      "_aggregation_": ["active"]
    }
  }
}`
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	svc := workflow.NewService(tmpDir)

	// Clean up cache after test to avoid affecting other tests
	t.Cleanup(func() {
		config.ClearWorkflowCache()
	})

	return svc
}

func TestFeatureService_TransitionStatus_WithAction(t *testing.T) {
	var updatedFeature *models.Feature
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{
				Key:    "E16-F01",
				Status: models.FeatureStatusDraft,
			}, nil
		},
		updateFn: func(ctx context.Context, feature *models.Feature) error {
			updatedFeature = feature
			return nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowServiceWithActions(t), nil, nil, nil)
	ctx := context.Background()

	// Transition from draft -> active; "active" has an orchestrator_action defined
	result, err := svc.TransitionStatus(ctx, "E16-F01", "active", TransitionOptions{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.OrchestratorAction == nil {
		t.Fatal("expected OrchestratorAction to be populated for target status 'active'")
	}
	if result.OrchestratorAction.Action != "spawn_agent" {
		t.Errorf("expected action 'spawn_agent', got %q", result.OrchestratorAction.Action)
	}
	if result.OrchestratorAction.AgentType != "developer" {
		t.Errorf("expected agent_type 'developer', got %q", result.OrchestratorAction.AgentType)
	}
	if len(result.OrchestratorAction.Skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(result.OrchestratorAction.Skills))
	}
	// Verify the template was populated with the feature key
	expectedInstruction := "Implement feature E16-F01 following TDD approach."
	if result.OrchestratorAction.Instruction != expectedInstruction {
		t.Errorf("expected instruction %q, got %q", expectedInstruction, result.OrchestratorAction.Instruction)
	}

	if updatedFeature == nil {
		t.Fatal("expected Update to be called")
	}
}

func TestFeatureService_TransitionStatus_WithoutAction(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{
				Key:    "E16-F01",
				Status: models.FeatureStatusActive,
			}, nil
		},
		updateFn: func(ctx context.Context, feature *models.Feature) error {
			return nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowServiceWithActions(t), nil, nil, nil)
	ctx := context.Background()

	// Transition from active -> completed; "completed" has NO orchestrator_action
	result, err := svc.TransitionStatus(ctx, "E16-F01", "completed", TransitionOptions{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.OrchestratorAction != nil {
		t.Errorf("expected nil OrchestratorAction for 'completed', got: %+v", result.OrchestratorAction)
	}
}

func TestFeatureService_GetNextStatus_WithActions(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{
				Key:    "E16-F01",
				Status: models.FeatureStatusDraft,
			}, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowServiceWithActions(t), nil, nil, nil)
	ctx := context.Background()

	info, err := svc.GetNextStatus(ctx, "E16-F01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(info.AvailableTransitions) == 0 {
		t.Fatal("expected available transitions for draft status")
	}

	// Find the "active" transition - it should have an action
	var activeTransition *TransitionInfoWithAction
	var archivedTransition *TransitionInfoWithAction
	for i, tr := range info.AvailableTransitions {
		if tr.TargetStatus == "active" {
			activeTransition = &info.AvailableTransitions[i]
		}
		if tr.TargetStatus == "archived" {
			archivedTransition = &info.AvailableTransitions[i]
		}
	}

	if activeTransition == nil {
		t.Fatal("expected 'active' in available transitions")
	}
	if activeTransition.OrchestratorAction == nil {
		t.Fatal("expected OrchestratorAction on 'active' transition")
	}
	if activeTransition.OrchestratorAction.Action != "spawn_agent" {
		t.Errorf("expected action 'spawn_agent', got %q", activeTransition.OrchestratorAction.Action)
	}
	// Verify template is populated with feature key
	expectedInstruction := "Implement feature E16-F01 following TDD approach."
	if activeTransition.OrchestratorAction.Instruction != expectedInstruction {
		t.Errorf("expected instruction %q, got %q", expectedInstruction, activeTransition.OrchestratorAction.Instruction)
	}

	// "archived" should have no action
	if archivedTransition == nil {
		t.Fatal("expected 'archived' in available transitions")
	}
	if archivedTransition.OrchestratorAction != nil {
		t.Errorf("expected nil OrchestratorAction on 'archived' transition, got: %+v", archivedTransition.OrchestratorAction)
	}
}

func TestFeatureService_resolveAction_NilWorkflow(t *testing.T) {
	// Create a FeatureService with a workflow service that has a nil workflow.
	// Use empty string project root - this gives a default workflow (not nil).
	// To truly test nil, we test through the default workflow which has no actions.
	repo := &mockFeatureRepo{}
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	// The default feature workflow has no orchestrator_action on any status.
	// resolveAction should return nil without panicking.
	feature := &models.Feature{Key: "E16-F01", Title: "Test Feature", Status: "draft"}
	ctx := context.Background()
	action := svc.resolveAction(ctx, feature, "draft")
	if action != nil {
		t.Errorf("expected nil action for default workflow, got: %+v", action)
	}
}

func TestFeatureService_resolveAction_UnknownStatus(t *testing.T) {
	repo := &mockFeatureRepo{}
	svc := NewFeatureService(repo, newTestFeatureWorkflowServiceWithActions(t), nil, nil, nil)

	// Unknown status should return nil without panicking
	feature := &models.Feature{Key: "E16-F01", Title: "Test Feature", Status: "nonexistent_status"}
	ctx := context.Background()
	action := svc.resolveAction(ctx, feature, "nonexistent_status")
	if action != nil {
		t.Errorf("expected nil action for unknown status, got: %+v", action)
	}
}

func TestFeatureService_resolveAction_StatusWithAction(t *testing.T) {
	repo := &mockFeatureRepo{}
	svc := NewFeatureService(repo, newTestFeatureWorkflowServiceWithActions(t), nil, nil, nil)

	feature := &models.Feature{Key: "E16-F02", Title: "Test Feature", Status: "active"}
	ctx := context.Background()
	action := svc.resolveAction(ctx, feature, "active")
	if action == nil {
		t.Fatal("expected non-nil action for 'active' status")
	}
	if action.Action != "spawn_agent" {
		t.Errorf("expected action 'spawn_agent', got %q", action.Action)
	}
	if action.AgentType != "developer" {
		t.Errorf("expected agent_type 'developer', got %q", action.AgentType)
	}
	// Verify template is populated with the entity key
	expectedInstruction := "Implement feature E16-F02 following TDD approach."
	if action.Instruction != expectedInstruction {
		t.Errorf("expected instruction %q, got %q", expectedInstruction, action.Instruction)
	}
}

func TestFeatureService_resolveAction_StatusWithoutAction(t *testing.T) {
	repo := &mockFeatureRepo{}
	svc := NewFeatureService(repo, newTestFeatureWorkflowServiceWithActions(t), nil, nil, nil)

	feature := &models.Feature{Key: "E16-F01", Title: "Test Feature", Status: "completed"}
	ctx := context.Background()
	action := svc.resolveAction(ctx, feature, "completed")
	if action != nil {
		t.Errorf("expected nil action for 'completed' status, got: %+v", action)
	}
}

// =============================================================================
// Tests for GetFeature
// =============================================================================

func TestFeatureService_GetFeature_Success(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{
				ID:     1,
				Key:    "E16-F01",
				Title:  "Test Feature",
				Status: models.FeatureStatusActive,
			}, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	feature, err := svc.GetFeature(ctx, "E16-F01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if feature == nil {
		t.Fatal("expected feature, got nil")
	}
	if feature.Key != "E16-F01" {
		t.Errorf("expected key E16-F01, got %s", feature.Key)
	}
	if feature.Title != "Test Feature" {
		t.Errorf("expected title 'Test Feature', got %s", feature.Title)
	}
}

func TestFeatureService_GetFeature_NotFound(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return nil, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	_, err := svc.GetFeature(ctx, "E99-F01")
	if err == nil {
		t.Fatal("expected error for not-found feature")
	}
	if err.Error() != "feature not found: E99-F01" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFeatureService_GetFeature_RepoError(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return nil, fmt.Errorf("db connection failed")
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	_, err := svc.GetFeature(ctx, "E16-F01")
	if err == nil {
		t.Fatal("expected error from repo failure")
	}
}

// =============================================================================
// Tests for ListFeatures
// =============================================================================

func TestFeatureService_ListFeatures_NoFilter(t *testing.T) {
	repo := &mockFeatureRepo{
		listFn: func(ctx context.Context) ([]*models.Feature, error) {
			return []*models.Feature{
				{Key: "E16-F01", Status: models.FeatureStatusDraft},
				{Key: "E16-F02", Status: models.FeatureStatusActive},
				{Key: "E16-F03", Status: models.FeatureStatusCompleted},
			}, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	features, err := svc.ListFeatures(ctx, FeatureFilters{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(features) != 3 {
		t.Errorf("expected 3 features, got %d", len(features))
	}
}

func TestFeatureService_ListFeatures_StatusFilter(t *testing.T) {
	repo := &mockFeatureRepo{
		listFn: func(ctx context.Context) ([]*models.Feature, error) {
			return []*models.Feature{
				{Key: "E16-F01", Status: models.FeatureStatusDraft},
				{Key: "E16-F02", Status: models.FeatureStatusActive},
				{Key: "E16-F03", Status: models.FeatureStatusActive},
			}, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	features, err := svc.ListFeatures(ctx, FeatureFilters{Status: "active"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(features) != 2 {
		t.Errorf("expected 2 active features, got %d", len(features))
	}
	for _, f := range features {
		if string(f.Status) != "active" {
			t.Errorf("expected status 'active', got %q", f.Status)
		}
	}
}

func TestFeatureService_ListFeatures_StatusFilterNoMatch(t *testing.T) {
	repo := &mockFeatureRepo{
		listFn: func(ctx context.Context) ([]*models.Feature, error) {
			return []*models.Feature{
				{Key: "E16-F01", Status: models.FeatureStatusDraft},
			}, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	features, err := svc.ListFeatures(ctx, FeatureFilters{Status: "active"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(features) != 0 {
		t.Errorf("expected 0 features, got %d", len(features))
	}
}

func TestFeatureService_ListFeatures_RepoError(t *testing.T) {
	repo := &mockFeatureRepo{
		listFn: func(ctx context.Context) ([]*models.Feature, error) {
			return nil, fmt.Errorf("db error")
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	_, err := svc.ListFeatures(ctx, FeatureFilters{})
	if err == nil {
		t.Fatal("expected error from repo failure")
	}
}

// =============================================================================
// Tests for GetProgress
// =============================================================================

func TestFeatureService_GetProgress_Success(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E16-F01", Status: models.FeatureStatusActive}, nil
		},
		getTaskStatusBreakdownFn: func(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
			return map[models.TaskStatus]int{
				"draft":     2,
				"active":    1,
				"completed": 2,
			}, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	progress, err := svc.GetProgress(ctx, "E16-F01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if progress == nil {
		t.Fatal("expected progress info, got nil")
	}
	if progress.FeatureKey != "E16-F01" {
		t.Errorf("expected feature key E16-F01, got %s", progress.FeatureKey)
	}
	if progress.TotalTasks != 5 {
		t.Errorf("expected 5 total tasks, got %d", progress.TotalTasks)
	}
	if progress.CompletionProgress != 40.0 {
		t.Errorf("expected 40%% completion, got %.2f%%", progress.CompletionProgress)
	}
	if progress.CompletionRatio != "2/5" {
		t.Errorf("expected completion ratio '2/5', got %s", progress.CompletionRatio)
	}
}

func TestFeatureService_GetProgress_NoTasks(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E16-F01", Status: models.FeatureStatusDraft}, nil
		},
		getTaskStatusBreakdownFn: func(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
			return map[models.TaskStatus]int{}, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	progress, err := svc.GetProgress(ctx, "E16-F01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if progress.TotalTasks != 0 {
		t.Errorf("expected 0 total tasks, got %d", progress.TotalTasks)
	}
	if progress.WeightedProgress != 0 {
		t.Errorf("expected 0%% weighted progress, got %.2f%%", progress.WeightedProgress)
	}
}

func TestFeatureService_GetProgress_NotFound(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return nil, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	_, err := svc.GetProgress(ctx, "E99-F01")
	if err == nil {
		t.Fatal("expected error for not-found feature")
	}
}

func TestFeatureService_GetProgress_BreakdownError(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E16-F01"}, nil
		},
		getTaskStatusBreakdownFn: func(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
			return nil, fmt.Errorf("breakdown error")
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	_, err := svc.GetProgress(ctx, "E16-F01")
	if err == nil {
		t.Fatal("expected error from breakdown failure")
	}
}

// =============================================================================
// Tests for GetHealth
// =============================================================================

func TestFeatureService_GetHealth_Healthy(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E16-F01"}, nil
		},
	}
	taskRepo := &mockFeatureTaskCounter{
		listByFeatureFn: func(ctx context.Context, featureID int64) ([]*models.Task, error) {
			return []*models.Task{
				{Key: "T-E16-F01-001", Status: "draft", Priority: 5, UpdatedAt: time.Now()},
				{Key: "T-E16-F01-002", Status: "active", Priority: 5, UpdatedAt: time.Now()},
			}, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, taskRepo, nil)
	ctx := context.Background()

	health, err := svc.GetHealth(ctx, "E16-F01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if health.Status != "healthy" {
		t.Errorf("expected healthy, got %s", health.Status)
	}
	if len(health.Reasons) != 0 {
		t.Errorf("expected no reasons, got %v", health.Reasons)
	}
}

func TestFeatureService_GetHealth_Warning_OneBlocked(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E16-F01"}, nil
		},
	}
	taskRepo := &mockFeatureTaskCounter{
		listByFeatureFn: func(ctx context.Context, featureID int64) ([]*models.Task, error) {
			return []*models.Task{
				{Key: "T-E16-F01-001", Status: "blocked", Priority: 5, UpdatedAt: time.Now()},
				{Key: "T-E16-F01-002", Status: "active", Priority: 5, UpdatedAt: time.Now()},
			}, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, taskRepo, nil)
	ctx := context.Background()

	health, err := svc.GetHealth(ctx, "E16-F01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if health.Status != "warning" {
		t.Errorf("expected warning, got %s", health.Status)
	}
	if len(health.Reasons) != 1 {
		t.Errorf("expected 1 reason, got %d", len(health.Reasons))
	}
}

func TestFeatureService_GetHealth_Critical_MultipleBlocked(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E16-F01"}, nil
		},
	}
	taskRepo := &mockFeatureTaskCounter{
		listByFeatureFn: func(ctx context.Context, featureID int64) ([]*models.Task, error) {
			return []*models.Task{
				{Key: "T-E16-F01-001", Status: "blocked", Priority: 5, UpdatedAt: time.Now()},
				{Key: "T-E16-F01-002", Status: "blocked", Priority: 5, UpdatedAt: time.Now()},
			}, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, taskRepo, nil)
	ctx := context.Background()

	health, err := svc.GetHealth(ctx, "E16-F01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if health.Status != "critical" {
		t.Errorf("expected critical, got %s", health.Status)
	}
}

func TestFeatureService_GetHealth_Critical_HighPriorityBlocked(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E16-F01"}, nil
		},
	}
	taskRepo := &mockFeatureTaskCounter{
		listByFeatureFn: func(ctx context.Context, featureID int64) ([]*models.Task, error) {
			return []*models.Task{
				{Key: "T-E16-F01-001", Status: "blocked", Priority: 2, UpdatedAt: time.Now()}, // High priority
			}, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, taskRepo, nil)
	ctx := context.Background()

	health, err := svc.GetHealth(ctx, "E16-F01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// Single blocked task = warning, but high priority escalates to critical
	if health.Status != "critical" {
		t.Errorf("expected critical for high-priority blocked task, got %s", health.Status)
	}
}

func TestFeatureService_GetHealth_NilTaskRepo(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E16-F01"}, nil
		},
	}

	// No task repo - should degrade gracefully
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	health, err := svc.GetHealth(ctx, "E16-F01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if health.Status != "healthy" {
		t.Errorf("expected healthy when taskRepo is nil, got %s", health.Status)
	}
}

func TestFeatureService_GetHealth_NotFound(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return nil, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	_, err := svc.GetHealth(ctx, "E99-F01")
	if err == nil {
		t.Fatal("expected error for not-found feature")
	}
}

func TestFeatureService_GetHealth_TaskRepoError(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E16-F01"}, nil
		},
	}
	taskRepo := &mockFeatureTaskCounter{
		listByFeatureFn: func(ctx context.Context, featureID int64) ([]*models.Task, error) {
			return nil, fmt.Errorf("task repo error")
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, taskRepo, nil)
	ctx := context.Background()

	_, err := svc.GetHealth(ctx, "E16-F01")
	if err == nil {
		t.Fatal("expected error from task repo failure")
	}
}

// =============================================================================
// Tests for GetWorkBreakdown
// =============================================================================

func TestFeatureService_GetWorkBreakdown_Success(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E16-F01"}, nil
		},
		getTaskStatusBreakdownFn: func(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
			return map[models.TaskStatus]int{
				"draft":     2, // not started (no responsibility in default workflow)
				"active":    1, // agent work in some configs
				"completed": 3, // terminal
				"blocked":   1, // blocked
			}, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	wb, err := svc.GetWorkBreakdown(ctx, "E16-F01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if wb.FeatureKey != "E16-F01" {
		t.Errorf("expected feature key E16-F01, got %s", wb.FeatureKey)
	}
	if wb.TotalTasks != 7 {
		t.Errorf("expected 7 total tasks, got %d", wb.TotalTasks)
	}
	if wb.BlockedWork != 1 {
		t.Errorf("expected 1 blocked, got %d", wb.BlockedWork)
	}
	// Completed tasks counted via terminal status check
	// The default feature workflow has "completed" and "archived" as terminal
}

func TestFeatureService_GetWorkBreakdown_NoTasks(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E16-F01"}, nil
		},
		getTaskStatusBreakdownFn: func(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
			return map[models.TaskStatus]int{}, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	wb, err := svc.GetWorkBreakdown(ctx, "E16-F01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if wb.TotalTasks != 0 {
		t.Errorf("expected 0 total tasks, got %d", wb.TotalTasks)
	}
}

func TestFeatureService_GetWorkBreakdown_NotFound(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return nil, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	_, err := svc.GetWorkBreakdown(ctx, "E99-F01")
	if err == nil {
		t.Fatal("expected error for not-found feature")
	}
}

func TestFeatureService_GetWorkBreakdown_RepoError(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E16-F01"}, nil
		},
		getTaskStatusBreakdownFn: func(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
			return nil, fmt.Errorf("breakdown error")
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	_, err := svc.GetWorkBreakdown(ctx, "E16-F01")
	if err == nil {
		t.Fatal("expected error from breakdown failure")
	}
}

// =============================================================================
// Tests for GetActionItems
// =============================================================================

func TestFeatureService_GetActionItems_Success(t *testing.T) {
	now := time.Now()
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E16-F01"}, nil
		},
	}
	taskRepo := &mockFeatureTaskCounter{
		listByFeatureFn: func(ctx context.Context, featureID int64) ([]*models.Task, error) {
			return []*models.Task{
				{Key: "T-E16-F01-001", Title: "Blocked Task", Status: "blocked", Priority: 5, UpdatedAt: now.Add(-48 * time.Hour)},
				{Key: "T-E16-F01-002", Title: "Draft Task", Status: "draft", Priority: 5, UpdatedAt: now},
				{Key: "T-E16-F01-003", Title: "Completed Task", Status: "completed", Priority: 5, UpdatedAt: now},
			}, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, taskRepo, nil)
	ctx := context.Background()

	items, err := svc.GetActionItems(ctx, "E16-F01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if items.FeatureKey != "E16-F01" {
		t.Errorf("expected feature key E16-F01, got %s", items.FeatureKey)
	}
	if len(items.Blocked) != 1 {
		t.Errorf("expected 1 blocked item, got %d", len(items.Blocked))
	}
	if len(items.Blocked) > 0 && items.Blocked[0].TaskKey != "T-E16-F01-001" {
		t.Errorf("expected blocked task key T-E16-F01-001, got %s", items.Blocked[0].TaskKey)
	}
}

func TestFeatureService_GetActionItems_NilTaskRepo(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E16-F01"}, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	items, err := svc.GetActionItems(ctx, "E16-F01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if items.FeatureKey != "E16-F01" {
		t.Errorf("expected feature key E16-F01, got %s", items.FeatureKey)
	}
	if items.Blocked != nil {
		t.Errorf("expected nil blocked items when taskRepo is nil, got %v", items.Blocked)
	}
}

func TestFeatureService_GetActionItems_NotFound(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return nil, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	_, err := svc.GetActionItems(ctx, "E99-F01")
	if err == nil {
		t.Fatal("expected error for not-found feature")
	}
}

func TestFeatureService_GetActionItems_TaskRepoError(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E16-F01"}, nil
		},
	}
	taskRepo := &mockFeatureTaskCounter{
		listByFeatureFn: func(ctx context.Context, featureID int64) ([]*models.Task, error) {
			return nil, fmt.Errorf("list error")
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, taskRepo, nil)
	ctx := context.Background()

	_, err := svc.GetActionItems(ctx, "E16-F01")
	if err == nil {
		t.Fatal("expected error from task repo failure")
	}
}

// =============================================================================
// Tests for GetTaskStatusBreakdown
// =============================================================================

func TestFeatureService_GetTaskStatusBreakdown_Success(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E16-F01"}, nil
		},
		getTaskStatusBreakdownFn: func(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
			return map[models.TaskStatus]int{
				"draft":     3,
				"active":    2,
				"completed": 1,
			}, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	breakdown, err := svc.GetTaskStatusBreakdown(ctx, "E16-F01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(breakdown) != 3 {
		t.Errorf("expected 3 statuses, got %d", len(breakdown))
	}
	if breakdown["draft"] != 3 {
		t.Errorf("expected 3 draft tasks, got %d", breakdown["draft"])
	}
	if breakdown["active"] != 2 {
		t.Errorf("expected 2 active tasks, got %d", breakdown["active"])
	}
	if breakdown["completed"] != 1 {
		t.Errorf("expected 1 completed task, got %d", breakdown["completed"])
	}
}

func TestFeatureService_GetTaskStatusBreakdown_Empty(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E16-F01"}, nil
		},
		getTaskStatusBreakdownFn: func(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
			return map[models.TaskStatus]int{}, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	breakdown, err := svc.GetTaskStatusBreakdown(ctx, "E16-F01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(breakdown) != 0 {
		t.Errorf("expected empty breakdown, got %v", breakdown)
	}
}

func TestFeatureService_GetTaskStatusBreakdown_NotFound(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return nil, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	_, err := svc.GetTaskStatusBreakdown(ctx, "E99-F01")
	if err == nil {
		t.Fatal("expected error for not-found feature")
	}
}

func TestFeatureService_GetTaskStatusBreakdown_RepoError(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E16-F01"}, nil
		},
		getTaskStatusBreakdownFn: func(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
			return nil, fmt.Errorf("breakdown error")
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	_, err := svc.GetTaskStatusBreakdown(ctx, "E16-F01")
	if err == nil {
		t.Fatal("expected error from breakdown failure")
	}
}

// =============================================================================
// Tests for RecalculateAndSetProgress
// =============================================================================

func TestFeatureService_RecalculateAndSetProgress_Success(t *testing.T) {
	var updatedFeature *models.Feature
	repo := &mockFeatureRepo{
		getByIDFn: func(ctx context.Context, id int64) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E16-F01", Status: models.FeatureStatusActive}, nil
		},
		getTaskStatusBreakdownFn: func(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
			return map[models.TaskStatus]int{
				"draft":     2,
				"active":    1,
				"completed": 2,
			}, nil
		},
		updateFn: func(ctx context.Context, feature *models.Feature) error {
			updatedFeature = feature
			return nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	err := svc.RecalculateAndSetProgress(ctx, 1)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if updatedFeature == nil {
		t.Fatal("expected Update to be called")
	}
	// Progress should be set (exact value depends on workflow config weights)
	// Status should NOT be completed since not all tasks are done
	if updatedFeature.Status == models.FeatureStatusCompleted {
		t.Error("expected status to NOT be completed when tasks are not all done")
	}
}

func TestFeatureService_RecalculateAndSetProgress_AutoComplete(t *testing.T) {
	var updatedFeature *models.Feature
	repo := &mockFeatureRepo{
		getByIDFn: func(ctx context.Context, id int64) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E16-F01", Status: models.FeatureStatusActive}, nil
		},
		getTaskStatusBreakdownFn: func(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
			// All tasks completed - should trigger auto-complete
			return map[models.TaskStatus]int{
				"completed": 5,
			}, nil
		},
		updateFn: func(ctx context.Context, feature *models.Feature) error {
			updatedFeature = feature
			return nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	err := svc.RecalculateAndSetProgress(ctx, 1)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if updatedFeature == nil {
		t.Fatal("expected Update to be called")
	}
	if updatedFeature.ProgressPct < 100.0 {
		t.Errorf("expected progress >= 100%%, got %.2f%%", updatedFeature.ProgressPct)
	}
	if updatedFeature.Status != models.FeatureStatusCompleted {
		t.Errorf("expected auto-complete to set status to 'completed', got %q", updatedFeature.Status)
	}
}

func TestFeatureService_RecalculateAndSetProgress_NotFound(t *testing.T) {
	repo := &mockFeatureRepo{
		getByIDFn: func(ctx context.Context, id int64) (*models.Feature, error) {
			return nil, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	err := svc.RecalculateAndSetProgress(ctx, 999)
	if err == nil {
		t.Fatal("expected error for not-found feature")
	}
}

func TestFeatureService_RecalculateAndSetProgress_BreakdownError(t *testing.T) {
	repo := &mockFeatureRepo{
		getByIDFn: func(ctx context.Context, id int64) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E16-F01"}, nil
		},
		getTaskStatusBreakdownFn: func(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
			return nil, fmt.Errorf("breakdown error")
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	err := svc.RecalculateAndSetProgress(ctx, 1)
	if err == nil {
		t.Fatal("expected error from breakdown failure")
	}
}

func TestFeatureService_RecalculateAndSetProgress_UpdateError(t *testing.T) {
	repo := &mockFeatureRepo{
		getByIDFn: func(ctx context.Context, id int64) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E16-F01"}, nil
		},
		getTaskStatusBreakdownFn: func(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
			return map[models.TaskStatus]int{"draft": 1}, nil
		},
		updateFn: func(ctx context.Context, feature *models.Feature) error {
			return fmt.Errorf("update failed")
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	err := svc.RecalculateAndSetProgress(ctx, 1)
	if err == nil {
		t.Fatal("expected error from update failure")
	}
}

// =============================================================================
// Tests for RecalculateAndSetProgressByKey
// =============================================================================

func TestFeatureService_RecalculateAndSetProgressByKey_Success(t *testing.T) {
	var updatedFeature *models.Feature
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E16-F01", Status: models.FeatureStatusActive}, nil
		},
		getByIDFn: func(ctx context.Context, id int64) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E16-F01", Status: models.FeatureStatusActive}, nil
		},
		getTaskStatusBreakdownFn: func(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
			return map[models.TaskStatus]int{"draft": 3, "completed": 2}, nil
		},
		updateFn: func(ctx context.Context, feature *models.Feature) error {
			updatedFeature = feature
			return nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	err := svc.RecalculateAndSetProgressByKey(ctx, "E16-F01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if updatedFeature == nil {
		t.Fatal("expected Update to be called")
	}
}

func TestFeatureService_RecalculateAndSetProgressByKey_NotFound(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return nil, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	ctx := context.Background()

	err := svc.RecalculateAndSetProgressByKey(ctx, "E99-F01")
	if err == nil {
		t.Fatal("expected error for not-found feature")
	}
}

// ==================== CRUD Tests ====================

func TestFeatureService_CreateFeature_Success(t *testing.T) {
	var capturedFeature *models.Feature
	epicTitle := "Test Epic"
	repo := &mockFeatureRepo{
		listByEpicFn: func(ctx context.Context, epicID int64) ([]*models.Feature, error) {
			return []*models.Feature{}, nil
		},
		createFn: func(ctx context.Context, feature *models.Feature) error {
			capturedFeature = feature
			return nil
		},
	}
	epicLookup := &mockFeatureEpicLookup{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{ID: 1, Key: "E01", Title: epicTitle, Status: models.EpicStatusActive}, nil
		},
	}
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, epicLookup)

	feature, err := svc.CreateFeature(context.Background(), CreateFeatureInput{
		EpicKey: "E01",
		Title:   "My Feature",
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if feature == nil {
		t.Fatal("expected feature, got nil")
	}
	if feature.Key == "" {
		t.Error("expected non-empty key")
	}
	if capturedFeature == nil {
		t.Fatal("expected Create to be called")
	}
	if capturedFeature.Title != "My Feature" {
		t.Errorf("expected title 'My Feature', got %q", capturedFeature.Title)
	}
	if capturedFeature.EpicID != 1 {
		t.Errorf("expected EpicID 1, got %d", capturedFeature.EpicID)
	}
}

func TestFeatureService_CreateFeature_EpicNotFound(t *testing.T) {
	repo := &mockFeatureRepo{}
	epicLookup := &mockFeatureEpicLookup{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, nil
		},
	}
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, epicLookup)

	_, err := svc.CreateFeature(context.Background(), CreateFeatureInput{
		EpicKey: "E99",
		Title:   "My Feature",
	})
	if err == nil {
		t.Fatal("expected error for missing epic")
	}
}

func TestFeatureService_CreateFeature_EmptyTitle(t *testing.T) {
	repo := &mockFeatureRepo{}
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	_, err := svc.CreateFeature(context.Background(), CreateFeatureInput{
		EpicKey: "E01",
		Title:   "",
	})
	if err == nil {
		t.Fatal("expected validation error for empty title")
	}
}

func TestFeatureService_UpdateFeature_Success(t *testing.T) {
	var updatedFeature *models.Feature
	newTitle := "Updated Title"
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{
				ID:     1,
				Key:    "E01-F01",
				Title:  "Old Title",
				Status: models.FeatureStatusActive,
			}, nil
		},
		updateFn: func(ctx context.Context, feature *models.Feature) error {
			updatedFeature = feature
			return nil
		},
	}
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	feature, err := svc.UpdateFeature(context.Background(), "E01-F01", FeatureUpdates{
		Title: &newTitle,
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if feature == nil {
		t.Fatal("expected feature, got nil")
	}
	if feature.Title != "Updated Title" {
		t.Errorf("expected title 'Updated Title', got %q", feature.Title)
	}
	if updatedFeature == nil {
		t.Fatal("expected Update to be called")
	}
}

func TestFeatureService_UpdateFeature_NotFound(t *testing.T) {
	updateCalled := false
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return nil, nil
		},
		updateFn: func(ctx context.Context, feature *models.Feature) error {
			updateCalled = true
			return nil
		},
	}
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	_, err := svc.UpdateFeature(context.Background(), "E99-F01", FeatureUpdates{})
	if err == nil {
		t.Fatal("expected error for not-found feature")
	}
	if updateCalled {
		t.Error("expected Update not to be called for not-found feature")
	}
}

func TestFeatureService_UpdateFeature_EmptyTitle(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E01-F01", Title: "Title", Status: models.FeatureStatusActive}, nil
		},
	}
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	emptyTitle := ""
	_, err := svc.UpdateFeature(context.Background(), "E01-F01", FeatureUpdates{
		Title: &emptyTitle,
	})
	if err == nil {
		t.Fatal("expected validation error for empty title")
	}
}

func TestFeatureService_DeleteFeature_Success(t *testing.T) {
	var deletedID int64
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 42, Key: "E01-F01", Title: "Feature", Status: models.FeatureStatusActive}, nil
		},
		deleteFn: func(ctx context.Context, id int64) error {
			deletedID = id
			return nil
		},
	}
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	err := svc.DeleteFeature(context.Background(), "E01-F01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if deletedID != 42 {
		t.Errorf("expected Delete to be called with ID 42, got %d", deletedID)
	}
}

func TestFeatureService_DeleteFeature_NotFound(t *testing.T) {
	deleteCalled := false
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return nil, nil
		},
		deleteFn: func(ctx context.Context, id int64) error {
			deleteCalled = true
			return nil
		},
	}
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	err := svc.DeleteFeature(context.Background(), "E99-F01")
	if err == nil {
		t.Fatal("expected error for not-found feature")
	}
	if deleteCalled {
		t.Error("expected Delete not to be called for not-found feature")
	}
}

func TestFeatureService_GetFeatureByID_Success(t *testing.T) {
	repo := &mockFeatureRepo{
		getByIDFn: func(ctx context.Context, id int64) (*models.Feature, error) {
			return &models.Feature{ID: id, Key: "E01-F01", Title: "Feature", Status: models.FeatureStatusActive}, nil
		},
	}
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	feature, err := svc.GetFeatureByID(context.Background(), 5)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if feature == nil {
		t.Fatal("expected feature, got nil")
	}
	if feature.ID != 5 {
		t.Errorf("expected ID 5, got %d", feature.ID)
	}
}

func TestFeatureService_GetFeatureByID_NotFound(t *testing.T) {
	repo := &mockFeatureRepo{
		getByIDFn: func(ctx context.Context, id int64) (*models.Feature, error) {
			return nil, nil
		},
	}
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	feature, err := svc.GetFeatureByID(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for not-found ID, got nil")
	}
	if feature != nil {
		t.Error("expected nil feature on error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error message, got: %v", err)
	}
}

func TestFeatureService_ListFeaturesByEpicKey_Success(t *testing.T) {
	repo := &mockFeatureRepo{
		listByEpicFn: func(ctx context.Context, epicID int64) ([]*models.Feature, error) {
			return []*models.Feature{
				{ID: 1, Key: "E01-F01", Title: "Feature 1", Status: models.FeatureStatusActive},
				{ID: 2, Key: "E01-F02", Title: "Feature 2", Status: models.FeatureStatusDraft},
			}, nil
		},
	}
	epicLookup := &mockFeatureEpicLookup{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{ID: 1, Key: "E01", Title: "Epic", Status: models.EpicStatusActive}, nil
		},
	}
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, epicLookup)

	features, err := svc.ListFeaturesByEpicKey(context.Background(), "E01", "")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(features) != 2 {
		t.Errorf("expected 2 features, got %d", len(features))
	}
}

func TestFeatureService_ListFeaturesByEpicKey_EpicNotFound(t *testing.T) {
	repo := &mockFeatureRepo{}
	epicLookup := &mockFeatureEpicLookup{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, nil
		},
	}
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, epicLookup)

	_, err := svc.ListFeaturesByEpicKey(context.Background(), "E99", "")
	if err == nil {
		t.Fatal("expected error for missing epic")
	}
}

// ==================== Lifecycle Tests ====================

func TestFeatureService_CompleteFeature_AllTasksDone(t *testing.T) {
	var updatedFeature *models.Feature
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E01-F01", Title: "Feature", Status: models.FeatureStatusActive}, nil
		},
		getByIDFn: func(ctx context.Context, id int64) (*models.Feature, error) {
			return &models.Feature{ID: id, Key: "E01-F01", Title: "Feature", Status: models.FeatureStatusActive}, nil
		},
		getTaskStatusBreakdownFn: func(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
			return map[models.TaskStatus]int{
				models.TaskStatus("completed"): 3,
			}, nil
		},
		updateFn: func(ctx context.Context, feature *models.Feature) error {
			updatedFeature = feature
			return nil
		},
	}
	taskRepo := &mockFeatureTaskCounter{
		listByFeatureFn: func(ctx context.Context, featureID int64) ([]*models.Task, error) {
			return []*models.Task{
				{ID: 1, Key: "T-E01-F01-001", Status: models.TaskStatus("completed")},
				{ID: 2, Key: "T-E01-F01-002", Status: models.TaskStatus("completed")},
				{ID: 3, Key: "T-E01-F01-003", Status: models.TaskStatus("completed")},
			}, nil
		},
	}
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, taskRepo, nil)

	result, err := svc.CompleteFeature(context.Background(), "E01-F01", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.RequiresForce {
		t.Error("expected RequiresForce=false when all tasks complete")
	}
	if updatedFeature == nil {
		t.Fatal("expected Update to be called")
	}
}

func TestFeatureService_CompleteFeature_TasksIncomplete(t *testing.T) {
	updateCalled := false
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E01-F01", Title: "Feature", Status: models.FeatureStatusActive}, nil
		},
		getTaskStatusBreakdownFn: func(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
			return map[models.TaskStatus]int{
				models.TaskStatus("completed"):   1,
				models.TaskStatus("in_progress"): 1,
			}, nil
		},
		updateFn: func(ctx context.Context, feature *models.Feature) error {
			updateCalled = true
			return nil
		},
	}
	taskRepo := &mockFeatureTaskCounter{
		listByFeatureFn: func(ctx context.Context, featureID int64) ([]*models.Task, error) {
			return []*models.Task{
				{ID: 1, Key: "T-E01-F01-001", Status: models.TaskStatus("completed")},
				{ID: 2, Key: "T-E01-F01-002", Status: models.TaskStatus("in_progress")},
			}, nil
		},
	}
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, taskRepo, nil)

	result, err := svc.CompleteFeature(context.Background(), "E01-F01", false)
	if err != nil {
		t.Fatalf("expected no error (RequiresForce signal, not error), got: %v", err)
	}
	if !result.RequiresForce {
		t.Error("expected RequiresForce=true for incomplete tasks with force=false")
	}
	if updateCalled {
		t.Error("expected Update not to be called when force=false and tasks incomplete")
	}
}

func TestFeatureService_CompleteFeature_Force(t *testing.T) {
	forcedTaskIDs := make([]int64, 0)
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E01-F01", Title: "Feature", Status: models.FeatureStatusActive}, nil
		},
		getByIDFn: func(ctx context.Context, id int64) (*models.Feature, error) {
			return &models.Feature{ID: id, Key: "E01-F01", Status: models.FeatureStatusActive}, nil
		},
		getTaskStatusBreakdownFn: func(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
			return map[models.TaskStatus]int{
				models.TaskStatus("completed"):   1,
				models.TaskStatus("in_progress"): 1,
			}, nil
		},
		updateFn: func(ctx context.Context, feature *models.Feature) error {
			return nil
		},
	}
	taskRepo := &mockFeatureTaskCounter{
		listByFeatureFn: func(ctx context.Context, featureID int64) ([]*models.Task, error) {
			return []*models.Task{
				{ID: 1, Key: "T-E01-F01-001", Status: models.TaskStatus("completed")},
				{ID: 2, Key: "T-E01-F01-002", Status: models.TaskStatus("in_progress")},
			}, nil
		},
		updateStatusForcedFn: func(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string, rejectionReason *string, documentPath *string, force bool) error {
			forcedTaskIDs = append(forcedTaskIDs, taskID)
			return nil
		},
	}
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, taskRepo, nil)

	result, err := svc.CompleteFeature(context.Background(), "E01-F01", true)
	if err != nil {
		t.Fatalf("expected no error with force=true, got: %v", err)
	}
	if result.RequiresForce {
		t.Error("expected RequiresForce=false after forced completion")
	}
	// Only the in_progress task (ID 2) should be force-completed
	if len(forcedTaskIDs) != 1 || forcedTaskIDs[0] != 2 {
		t.Errorf("expected only task 2 to be force-completed, got: %v", forcedTaskIDs)
	}
}

func TestFeatureService_CascadeFeatureStatusToTasks_Success(t *testing.T) {
	var cascadeFeatureID int64
	var cascadeTargetStatus models.TaskStatus
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 10, Key: "E01-F01", Title: "Feature", Status: models.FeatureStatusActive}, nil
		},
		cascadeStatusToTasksFn: func(ctx context.Context, featureID int64, targetTaskStatus models.TaskStatus) error {
			cascadeFeatureID = featureID
			cascadeTargetStatus = targetTaskStatus
			return nil
		},
	}
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	err := svc.CascadeFeatureStatusToTasks(context.Background(), "E01-F01", models.TaskStatus("completed"))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cascadeFeatureID != 10 {
		t.Errorf("expected cascade called with featureID 10, got %d", cascadeFeatureID)
	}
	if cascadeTargetStatus != models.TaskStatus("completed") {
		t.Errorf("expected cascade target status 'completed', got %q", cascadeTargetStatus)
	}
}

func TestFeatureService_CascadeFeatureStatusToTasks_NotFound(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return nil, fmt.Errorf("feature not found")
		},
	}
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	err := svc.CascadeFeatureStatusToTasks(context.Background(), "E99-F01", models.TaskStatus("completed"))
	if err == nil {
		t.Fatal("expected error for not-found feature")
	}
}

func TestFeatureService_CascadeFeatureStatusToTasks_ZeroTasks(t *testing.T) {
	// When feature exists but has no tasks, cascade should succeed with no error.
	cascadeCalled := false
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 10, Key: key, Title: "Empty Feature"}, nil
		},
		cascadeStatusToTasksFn: func(ctx context.Context, featureID int64, targetTaskStatus models.TaskStatus) error {
			cascadeCalled = true
			if featureID != 10 {
				t.Errorf("expected featureID 10, got %d", featureID)
			}
			// Simulate successful cascade with zero tasks (no error)
			return nil
		},
	}
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	err := svc.CascadeFeatureStatusToTasks(context.Background(), "E15-F09", models.TaskStatus("completed"))
	if err != nil {
		t.Fatalf("expected no error for zero-task cascade, got: %v", err)
	}
	if !cascadeCalled {
		t.Error("expected CascadeStatusToTasks to be called even for zero tasks")
	}
}

// ============================================================================
// MockFeatureWritableDocumentRepository
// ============================================================================

// mockFeatureWritableDocRepo implements FeatureWritableDocumentRepository for testing.
type mockFeatureWritableDocRepo struct {
	createOrGetFn       func(ctx context.Context, title, filePath string) (*models.Document, error)
	getByTitleFn        func(ctx context.Context, title string) (*models.Document, error)
	linkToFeatureFn     func(ctx context.Context, featureID, documentID int64) error
	unlinkFromFeatureFn func(ctx context.Context, featureID, documentID int64) error
}

func (m *mockFeatureWritableDocRepo) CreateOrGet(ctx context.Context, title, filePath string) (*models.Document, error) {
	if m.createOrGetFn != nil {
		return m.createOrGetFn(ctx, title, filePath)
	}
	return nil, fmt.Errorf("CreateOrGet not implemented in mockFeatureWritableDocRepo")
}

func (m *mockFeatureWritableDocRepo) GetByTitle(ctx context.Context, title string) (*models.Document, error) {
	if m.getByTitleFn != nil {
		return m.getByTitleFn(ctx, title)
	}
	return nil, fmt.Errorf("GetByTitle not implemented in mockFeatureWritableDocRepo")
}

func (m *mockFeatureWritableDocRepo) LinkToFeature(ctx context.Context, featureID, documentID int64) error {
	if m.linkToFeatureFn != nil {
		return m.linkToFeatureFn(ctx, featureID, documentID)
	}
	return fmt.Errorf("LinkToFeature not implemented in mockFeatureWritableDocRepo")
}

func (m *mockFeatureWritableDocRepo) UnlinkFromFeature(ctx context.Context, featureID, documentID int64) error {
	if m.unlinkFromFeatureFn != nil {
		return m.unlinkFromFeatureFn(ctx, featureID, documentID)
	}
	return fmt.Errorf("UnlinkFromFeature not implemented in mockFeatureWritableDocRepo")
}

// ============================================================================
// FeatureService.LinkDocument Tests
// ============================================================================

func TestFeatureService_LinkDocument_Happy_Path(t *testing.T) {
	var capturedFeatureID, capturedDocID int64

	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E07-F01"}, nil
		},
	}

	docRepo := &mockFeatureWritableDocRepo{
		createOrGetFn: func(ctx context.Context, title, filePath string) (*models.Document, error) {
			if title != "API Spec" {
				t.Errorf("expected title 'API Spec', got %q", title)
			}
			if filePath != "docs/api-spec.md" {
				t.Errorf("expected filePath 'docs/api-spec.md', got %q", filePath)
			}
			return &models.Document{ID: 55, Title: title, FilePath: filePath}, nil
		},
		linkToFeatureFn: func(ctx context.Context, featureID, documentID int64) error {
			capturedFeatureID = featureID
			capturedDocID = documentID
			return nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	svc.SetWritableDocRepo(docRepo)

	err := svc.LinkDocument(context.Background(), "E07-F01", "API Spec", "docs/api-spec.md")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if capturedFeatureID != 1 {
		t.Errorf("expected feature ID 1, got %d", capturedFeatureID)
	}
	if capturedDocID != 55 {
		t.Errorf("expected doc ID 55, got %d", capturedDocID)
	}
}

func TestFeatureService_LinkDocument_NoWritableDocRepo(t *testing.T) {
	repo := &mockFeatureRepo{}
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	// writableDocRepo not set

	err := svc.LinkDocument(context.Background(), "E07-F01", "API Spec", "docs/api-spec.md")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "writable document repository not configured") {
		t.Errorf("expected error to contain 'writable document repository not configured', got: %v", err)
	}
}

func TestFeatureService_LinkDocument_FeatureNotFound(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return nil, fmt.Errorf("feature not found")
		},
	}

	docRepo := &mockFeatureWritableDocRepo{}
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	svc.SetWritableDocRepo(docRepo)

	err := svc.LinkDocument(context.Background(), "E07-F99", "API Spec", "docs/api-spec.md")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "feature not found") {
		t.Errorf("expected error to contain 'feature not found', got: %v", err)
	}
}

func TestFeatureService_LinkDocument_CreateOrGetError(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E07-F01"}, nil
		},
	}

	docRepo := &mockFeatureWritableDocRepo{
		createOrGetFn: func(ctx context.Context, title, filePath string) (*models.Document, error) {
			return nil, fmt.Errorf("database error")
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	svc.SetWritableDocRepo(docRepo)

	err := svc.LinkDocument(context.Background(), "E07-F01", "API Spec", "docs/api-spec.md")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create or get document") {
		t.Errorf("expected error to contain 'failed to create or get document', got: %v", err)
	}
}

func TestFeatureService_LinkDocument_LinkError(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E07-F01"}, nil
		},
	}

	docRepo := &mockFeatureWritableDocRepo{
		createOrGetFn: func(ctx context.Context, title, filePath string) (*models.Document, error) {
			return &models.Document{ID: 55, Title: title}, nil
		},
		linkToFeatureFn: func(ctx context.Context, featureID, documentID int64) error {
			return fmt.Errorf("link failed")
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	svc.SetWritableDocRepo(docRepo)

	err := svc.LinkDocument(context.Background(), "E07-F01", "API Spec", "docs/api-spec.md")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to link document to feature") {
		t.Errorf("expected error to contain 'failed to link document to feature', got: %v", err)
	}
}

// ============================================================================
// FeatureService.UnlinkDocument Tests
// ============================================================================

func TestFeatureService_UnlinkDocument_Happy_Path(t *testing.T) {
	var capturedFeatureID, capturedDocID int64

	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E07-F01"}, nil
		},
	}

	docRepo := &mockFeatureWritableDocRepo{
		getByTitleFn: func(ctx context.Context, title string) (*models.Document, error) {
			if title != "API Spec" {
				t.Errorf("expected title 'API Spec', got %q", title)
			}
			return &models.Document{ID: 55, Title: title}, nil
		},
		unlinkFromFeatureFn: func(ctx context.Context, featureID, documentID int64) error {
			capturedFeatureID = featureID
			capturedDocID = documentID
			return nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	svc.SetWritableDocRepo(docRepo)

	err := svc.UnlinkDocument(context.Background(), "E07-F01", "API Spec")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if capturedFeatureID != 1 {
		t.Errorf("expected feature ID 1, got %d", capturedFeatureID)
	}
	if capturedDocID != 55 {
		t.Errorf("expected doc ID 55, got %d", capturedDocID)
	}
}

func TestFeatureService_UnlinkDocument_NoWritableDocRepo(t *testing.T) {
	repo := &mockFeatureRepo{}
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	err := svc.UnlinkDocument(context.Background(), "E07-F01", "API Spec")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "writable document repository not configured") {
		t.Errorf("expected error to contain 'writable document repository not configured', got: %v", err)
	}
}

func TestFeatureService_UnlinkDocument_FeatureNotFound(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return nil, fmt.Errorf("feature not found")
		},
	}

	docRepo := &mockFeatureWritableDocRepo{}
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	svc.SetWritableDocRepo(docRepo)

	err := svc.UnlinkDocument(context.Background(), "E07-F99", "API Spec")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "feature not found") {
		t.Errorf("expected error to contain 'feature not found', got: %v", err)
	}
}

func TestFeatureService_UnlinkDocument_DocumentNotFound(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E07-F01"}, nil
		},
	}

	docRepo := &mockFeatureWritableDocRepo{
		getByTitleFn: func(ctx context.Context, title string) (*models.Document, error) {
			return nil, fmt.Errorf("document not found")
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	svc.SetWritableDocRepo(docRepo)

	err := svc.UnlinkDocument(context.Background(), "E07-F01", "Missing Doc")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "document not found") {
		t.Errorf("expected error to contain 'document not found', got: %v", err)
	}
}

func TestFeatureService_UnlinkDocument_UnlinkError(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: "E07-F01"}, nil
		},
	}

	docRepo := &mockFeatureWritableDocRepo{
		getByTitleFn: func(ctx context.Context, title string) (*models.Document, error) {
			return &models.Document{ID: 55, Title: title}, nil
		},
		unlinkFromFeatureFn: func(ctx context.Context, featureID, documentID int64) error {
			return fmt.Errorf("unlink failed")
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	svc.SetWritableDocRepo(docRepo)

	err := svc.UnlinkDocument(context.Background(), "E07-F01", "API Spec")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to unlink document from feature") {
		t.Errorf("expected error to contain 'failed to unlink document from feature', got: %v", err)
	}
}

// mockDocumentRepository implements DocumentRepository (read-only) for testing.
type mockDocumentRepository struct {
	listForFeatureFn func(ctx context.Context, featureID int64) ([]*models.Document, error)
}

func (m *mockDocumentRepository) ListForTask(ctx context.Context, taskID int64) ([]*models.Document, error) {
	return nil, nil
}

func (m *mockDocumentRepository) ListForFeature(ctx context.Context, featureID int64) ([]*models.Document, error) {
	if m.listForFeatureFn != nil {
		return m.listForFeatureFn(ctx, featureID)
	}
	return nil, nil
}

func (m *mockDocumentRepository) ListForEpic(ctx context.Context, epicID int64) ([]*models.Document, error) {
	return nil, nil
}

// TestFeatureService_SetDocRepo verifies the setter
// properly wires the optional document repository dependency.
func TestFeatureService_SetDocRepo(t *testing.T) {
	repo := &mockFeatureRepo{}
	workflowSvc := newTestFeatureWorkflowService()
	docRepo := &mockDocumentRepository{
		listForFeatureFn: func(ctx context.Context, featureID int64) ([]*models.Document, error) {
			return []*models.Document{}, nil
		},
	}

	svc := NewFeatureService(repo, workflowSvc, nil, nil, nil)
	svc.SetDocRepo(docRepo)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	// Verify docRepo was wired (ListRelatedDocuments will call the docRepo without error)
	docs, err := svc.ListRelatedDocuments(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// docRepo was called (not nil path), so result is non-nil empty slice
	if docs == nil {
		t.Error("expected non-nil slice from docRepo (even if empty)")
	}
}

func TestFeatureService_GetTaskCount_Success(t *testing.T) {
	repo := &mockFeatureRepo{
		getTaskCountFn: func(ctx context.Context, featureID int64) (int, error) {
			if featureID != 42 {
				t.Errorf("expected featureID 42, got %d", featureID)
			}
			return 7, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	count, err := svc.GetTaskCount(context.Background(), 42)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if count != 7 {
		t.Errorf("expected count 7, got %d", count)
	}
}

func TestFeatureService_GetTaskCount_Error(t *testing.T) {
	repo := &mockFeatureRepo{
		getTaskCountFn: func(ctx context.Context, featureID int64) (int, error) {
			return 0, fmt.Errorf("db error")
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	_, err := svc.GetTaskCount(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to get task count") {
		t.Errorf("expected error to contain 'failed to get task count', got: %v", err)
	}
}

func TestFeatureService_GetStatusBreakdownBatch_NilTaskRepo(t *testing.T) {
	repo := &mockFeatureRepo{}

	// No taskRepo passed → graceful degradation
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	result, err := svc.GetStatusBreakdownBatch(context.Background(), []int64{1, 2})
	if err != nil {
		t.Fatalf("expected no error with nil taskRepo, got: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result with nil taskRepo, got: %v", result)
	}
}

// mockFeatureTaskCounterWithBatch is a local variant that supports
// configurable GetStatusBreakdownMapBatch for TestFeatureService_GetStatusBreakdownBatch_WithTaskRepo.
type mockFeatureTaskCounterWithBatch struct {
	listByFeatureFn              func(ctx context.Context, featureID int64) ([]*models.Task, error)
	getStatusBreakdownMapBatchFn func(ctx context.Context, featureIDs []int64) (map[int64]map[models.TaskStatus]int, error)
}

func (m *mockFeatureTaskCounterWithBatch) ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error) {
	if m.listByFeatureFn != nil {
		return m.listByFeatureFn(ctx, featureID)
	}
	return nil, nil
}

func (m *mockFeatureTaskCounterWithBatch) UpdateStatusForced(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string, rejectionReason *string, documentPath *string, force bool) error {
	return nil
}

func (m *mockFeatureTaskCounterWithBatch) GetStatusBreakdownMapBatch(ctx context.Context, featureIDs []int64) (map[int64]map[models.TaskStatus]int, error) {
	if m.getStatusBreakdownMapBatchFn != nil {
		return m.getStatusBreakdownMapBatchFn(ctx, featureIDs)
	}
	return nil, nil
}

func (m *mockFeatureTaskCounterWithBatch) GetTaskCountsForFeatures(ctx context.Context, featureIDs []int64) (map[int64]int, error) {
	return map[int64]int{}, nil
}

func TestFeatureService_GetStatusBreakdownBatch_WithTaskRepo(t *testing.T) {
	repo := &mockFeatureRepo{}
	taskRepo := &mockFeatureTaskCounterWithBatch{
		getStatusBreakdownMapBatchFn: func(ctx context.Context, featureIDs []int64) (map[int64]map[models.TaskStatus]int, error) {
			result := map[int64]map[models.TaskStatus]int{
				1: {models.TaskStatus("todo"): 3},
			}
			return result, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, taskRepo, nil)

	result, err := svc.GetStatusBreakdownBatch(context.Background(), []int64{1})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result[1][models.TaskStatus("todo")] != 3 {
		t.Errorf("expected 3 todo tasks for feature 1, got %d", result[1][models.TaskStatus("todo")])
	}
}

func TestFeatureService_UpdateFeatureKey_Success(t *testing.T) {
	callCount := 0
	repo := &mockFeatureRepo{
		// GetByKey called to check if newKey exists - return error (not found)
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			if key == "E07-F02" {
				return nil, fmt.Errorf("not found")
			}
			return nil, fmt.Errorf("unexpected key: %s", key)
		},
		updateKeyFn: func(ctx context.Context, oldKey string, newKey string) error {
			callCount++
			if oldKey != "E07-F01" {
				t.Errorf("expected oldKey 'E07-F01', got %q", oldKey)
			}
			if newKey != "E07-F02" {
				t.Errorf("expected newKey 'E07-F02', got %q", newKey)
			}
			return nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	err := svc.UpdateFeatureKey(context.Background(), "E07-F01", "E07-F02")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected UpdateKey called once, got %d", callCount)
	}
}

func TestFeatureService_UpdateFeatureKey_KeyAlreadyExists(t *testing.T) {
	repo := &mockFeatureRepo{
		// GetByKey for newKey returns an existing feature (key collision)
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			if key == "E07-F02" {
				return &models.Feature{ID: 99, Key: "E07-F02"}, nil
			}
			return nil, fmt.Errorf("not found")
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	err := svc.UpdateFeatureKey(context.Background(), "E07-F01", "E07-F02")
	if err == nil {
		t.Fatal("expected error for key collision, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected error to contain 'already exists', got: %v", err)
	}
}

func TestFeatureService_SetFeatureStatusOverride_Success(t *testing.T) {
	callCount := 0
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 5, Key: key}, nil
		},
		setStatusOverrideFn: func(ctx context.Context, featureID int64, override bool) error {
			callCount++
			if featureID != 5 {
				t.Errorf("expected featureID 5, got %d", featureID)
			}
			if !override {
				t.Errorf("expected override=true")
			}
			return nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	err := svc.SetFeatureStatusOverride(context.Background(), "E07-F01", true)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected SetStatusOverride called once, got %d", callCount)
	}
}

func TestFeatureService_SetFeatureStatusOverride_FeatureNotFound(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return nil, fmt.Errorf("not found")
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	err := svc.SetFeatureStatusOverride(context.Background(), "E07-F99", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("expected error to contain 'does not exist', got: %v", err)
	}
}

func TestFeatureService_ResolveFeaturePath_StoredFilePath(t *testing.T) {
	storedPath := "docs/custom/E07-F01/feature.md"
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 1, Key: key, FilePath: &storedPath}, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	path := svc.ResolveFeaturePath(context.Background(), "E07-F01", "/project/root")
	if path != storedPath {
		t.Errorf("expected %q, got %q", storedPath, path)
	}
}

func TestFeatureService_ResolveFeaturePath_DefaultPath(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			// No FilePath set
			return &models.Feature{ID: 1, Key: key, FilePath: nil}, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	path := svc.ResolveFeaturePath(context.Background(), "E07-F01", "/project/root")
	expected := "docs/plan/E07/E07-F01/feature.md"
	if path != expected {
		t.Errorf("expected default path %q, got %q", expected, path)
	}
}

func TestFeatureService_ResolveFeaturePath_NotFound(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return nil, fmt.Errorf("not found")
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	path := svc.ResolveFeaturePath(context.Background(), "E07-F99", "/project/root")
	if path != "" {
		t.Errorf("expected empty path for not-found feature, got %q", path)
	}
}

func TestFeatureService_UpdateFeatureFilePath_Success(t *testing.T) {
	newPath := "docs/custom/feature.md"
	callCount := 0
	repo := &mockFeatureRepo{
		updateFilePathFn: func(ctx context.Context, featureKey string, fp *string) error {
			callCount++
			if featureKey != "E07-F01" {
				t.Errorf("expected key 'E07-F01', got %q", featureKey)
			}
			if fp == nil || *fp != newPath {
				t.Errorf("expected path %q, got %v", newPath, fp)
			}
			return nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	err := svc.UpdateFeatureFilePath(context.Background(), "E07-F01", &newPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected UpdateFilePath called once, got %d", callCount)
	}
}

func TestFeatureService_UpdateFeatureFilePath_Error(t *testing.T) {
	repo := &mockFeatureRepo{
		updateFilePathFn: func(ctx context.Context, featureKey string, fp *string) error {
			return fmt.Errorf("db write error")
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	err := svc.UpdateFeatureFilePath(context.Background(), "E07-F01", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to update file path") {
		t.Errorf("expected error to contain 'failed to update file path', got: %v", err)
	}
}

func TestFeatureService_ListTasksForFeature_NilTaskRepo(t *testing.T) {
	repo := &mockFeatureRepo{}

	// No taskRepo - graceful degradation
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	tasks, err := svc.ListTasksForFeature(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected no error with nil taskRepo, got: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected empty slice, got %d tasks", len(tasks))
	}
}

func TestFeatureService_ListTasksForFeature_Success(t *testing.T) {
	repo := &mockFeatureRepo{}
	taskRepo := &mockFeatureTaskCounter{
		listByFeatureFn: func(ctx context.Context, featureID int64) ([]*models.Task, error) {
			return []*models.Task{
				{ID: 1, Key: "E07-F01-001"},
				{ID: 2, Key: "E07-F01-002"},
			}, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, taskRepo, nil)

	tasks, err := svc.ListTasksForFeature(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}
}

func TestFeatureService_ListTasksForFeature_Error(t *testing.T) {
	repo := &mockFeatureRepo{}
	taskRepo := &mockFeatureTaskCounter{
		listByFeatureFn: func(ctx context.Context, featureID int64) ([]*models.Task, error) {
			return nil, fmt.Errorf("db error")
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, taskRepo, nil)

	_, err := svc.ListTasksForFeature(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to list tasks for feature") {
		t.Errorf("expected error to contain 'failed to list tasks for feature', got: %v", err)
	}
}

func TestFeatureService_ListRelatedDocuments_NilDocRepo(t *testing.T) {
	repo := &mockFeatureRepo{}

	// No docRepo - graceful degradation
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	docs, err := svc.ListRelatedDocuments(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected no error with nil docRepo, got: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("expected empty slice, got %d docs", len(docs))
	}
}

func TestFeatureService_ListRelatedDocuments_Success(t *testing.T) {
	repo := &mockFeatureRepo{}
	docRepo := &mockDocumentRepository{
		listForFeatureFn: func(ctx context.Context, featureID int64) ([]*models.Document, error) {
			return []*models.Document{
				{ID: 10, Title: "API Spec"},
				{ID: 11, Title: "Design Doc"},
			}, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	svc.SetDocRepo(docRepo)

	docs, err := svc.ListRelatedDocuments(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("expected 2 documents, got %d", len(docs))
	}
}

func TestFeatureService_ListRelatedDocuments_Error(t *testing.T) {
	repo := &mockFeatureRepo{}
	docRepo := &mockDocumentRepository{
		listForFeatureFn: func(ctx context.Context, featureID int64) ([]*models.Document, error) {
			return nil, fmt.Errorf("db error listing docs")
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	svc.SetDocRepo(docRepo)

	_, err := svc.ListRelatedDocuments(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to list related documents") {
		t.Errorf("expected error to contain 'failed to list related documents', got: %v", err)
	}
}

func TestFeatureService_ListRelatedDocumentsByKey_Success(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 3, Key: key}, nil
		},
	}
	docRepo := &mockDocumentRepository{
		listForFeatureFn: func(ctx context.Context, featureID int64) ([]*models.Document, error) {
			if featureID != 3 {
				return nil, fmt.Errorf("unexpected featureID %d", featureID)
			}
			return []*models.Document{{ID: 20, Title: "Spec"}}, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
	svc.SetDocRepo(docRepo)

	docs, err := svc.ListRelatedDocumentsByKey(context.Background(), "E07-F01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 document, got %d", len(docs))
	}
}

func TestFeatureService_ListRelatedDocumentsByKey_FeatureNotFound(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return nil, fmt.Errorf("not found")
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	_, err := svc.ListRelatedDocumentsByKey(context.Background(), "E07-F99")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "feature not found") {
		t.Errorf("expected error to contain 'feature not found', got: %v", err)
	}
}

func TestFeatureService_GetEnrichedTaskStatusBreakdown_Success(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 10, Key: key}, nil
		},
		getTaskStatusBreakdownFn: func(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
			return map[models.TaskStatus]int{
				models.TaskStatus("todo"):        2,
				models.TaskStatus("in_progress"): 1,
				models.TaskStatus("completed"):   3,
			}, nil
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	counts, err := svc.GetEnrichedTaskStatusBreakdown(context.Background(), "E07-F01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// The result is a []workflow.StatusCount; verify we got entries
	if len(counts) == 0 {
		t.Error("expected non-empty status counts")
	}
}

func TestFeatureService_GetEnrichedTaskStatusBreakdown_FeatureNotFound(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return nil, fmt.Errorf("not found")
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	_, err := svc.GetEnrichedTaskStatusBreakdown(context.Background(), "E07-F99")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to get feature") {
		t.Errorf("expected error to contain 'failed to get feature', got: %v", err)
	}
}

func TestFeatureService_GetEnrichedTaskStatusBreakdown_BreakdownError(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{ID: 10, Key: key}, nil
		},
		getTaskStatusBreakdownFn: func(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
			return nil, fmt.Errorf("breakdown db error")
		},
	}

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)

	_, err := svc.GetEnrichedTaskStatusBreakdown(context.Background(), "E07-F01")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to get task status breakdown") {
		t.Errorf("expected error to contain 'failed to get task status breakdown', got: %v", err)
	}
}

// TestFeatureService_resolveFeatureFilePath tests indirectly via CreateFeature with a non-nil FilePath.

func TestFeatureService_CreateFeature_WithFilePath_NoCollision(t *testing.T) {
	customPath := "docs/custom/feature.md"
	repo := &mockFeatureRepo{
		listByEpicFn: func(ctx context.Context, epicID int64) ([]*models.Feature, error) {
			return []*models.Feature{}, nil
		},
		createFn: func(ctx context.Context, feature *models.Feature) error {
			return nil
		},
		// GetByFilePath returns nil (no collision)
		getByFilePathFn: func(ctx context.Context, filePath string) (*models.Feature, error) {
			return nil, nil
		},
	}
	epicLookup := &mockFeatureEpicLookup{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{ID: 1, Key: "E01", Title: "Test Epic", Status: models.EpicStatusActive}, nil
		},
	}
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, epicLookup)

	feature, err := svc.CreateFeature(context.Background(), CreateFeatureInput{
		EpicKey:  "E01",
		Title:    "My Feature",
		FilePath: &customPath,
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if feature == nil {
		t.Fatal("expected non-nil feature")
	}
	if feature.FilePath == nil || *feature.FilePath != customPath {
		t.Errorf("expected FilePath %q, got %v", customPath, feature.FilePath)
	}
}

func TestFeatureService_CreateFeature_WithFilePath_Collision_NoForce(t *testing.T) {
	customPath := "docs/custom/feature.md"
	repo := &mockFeatureRepo{
		listByEpicFn: func(ctx context.Context, epicID int64) ([]*models.Feature, error) {
			return []*models.Feature{}, nil
		},
		// GetByFilePath returns an existing feature (collision)
		getByFilePathFn: func(ctx context.Context, filePath string) (*models.Feature, error) {
			return &models.Feature{ID: 99, Key: "E01-F02", Title: "Other Feature"}, nil
		},
	}
	epicLookup := &mockFeatureEpicLookup{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{ID: 1, Key: "E01", Title: "Test Epic", Status: models.EpicStatusActive}, nil
		},
	}
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, epicLookup)

	_, err := svc.CreateFeature(context.Background(), CreateFeatureInput{
		EpicKey:  "E01",
		Title:    "My Feature",
		FilePath: &customPath,
		Force:    false,
	})
	if err == nil {
		t.Fatal("expected error for file path collision without force, got nil")
	}
	if !strings.Contains(err.Error(), "already claimed") {
		t.Errorf("expected error to contain 'already claimed', got: %v", err)
	}
}

func TestFeatureService_CreateFeature_WithFilePath_Collision_WithForce(t *testing.T) {
	customPath := "docs/custom/feature.md"
	updateFilePathCalled := false
	repo := &mockFeatureRepo{
		listByEpicFn: func(ctx context.Context, epicID int64) ([]*models.Feature, error) {
			return []*models.Feature{}, nil
		},
		createFn: func(ctx context.Context, feature *models.Feature) error {
			return nil
		},
		// GetByFilePath returns an existing feature (collision)
		getByFilePathFn: func(ctx context.Context, filePath string) (*models.Feature, error) {
			return &models.Feature{ID: 99, Key: "E01-F02", Title: "Other Feature"}, nil
		},
		// UpdateFilePath called to release path from old feature
		updateFilePathFn: func(ctx context.Context, featureKey string, fp *string) error {
			updateFilePathCalled = true
			if featureKey != "E01-F02" {
				return fmt.Errorf("unexpected key: %s", featureKey)
			}
			if fp != nil {
				return fmt.Errorf("expected nil path to release, got %v", fp)
			}
			return nil
		},
	}
	epicLookup := &mockFeatureEpicLookup{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{ID: 1, Key: "E01", Title: "Test Epic", Status: models.EpicStatusActive}, nil
		},
	}
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, epicLookup)

	feature, err := svc.CreateFeature(context.Background(), CreateFeatureInput{
		EpicKey:  "E01",
		Title:    "My Feature",
		FilePath: &customPath,
		Force:    true,
	})
	if err != nil {
		t.Fatalf("expected no error with force=true, got: %v", err)
	}
	if feature == nil {
		t.Fatal("expected non-nil feature")
	}
	if !updateFilePathCalled {
		t.Error("expected UpdateFilePath to be called to release old path")
	}
}

func TestFeatureService_UpdateFeatureStatusIfNotOverridden(t *testing.T) {
	t.Run("success - status updated", func(t *testing.T) {
		repo := &mockFeatureRepo{
			getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
				return &models.Feature{ID: 42, Key: "E01-F01", Status: models.FeatureStatusDraft}, nil
			},
			updateStatusIfNotOverriddenFn: func(ctx context.Context, featureID int64, newStatus models.FeatureStatus) (bool, error) {
				if featureID != 42 {
					t.Errorf("expected featureID 42, got %d", featureID)
				}
				if newStatus != models.FeatureStatusActive {
					t.Errorf("expected status active, got %s", newStatus)
				}
				return true, nil
			},
		}

		svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
		updated, err := svc.UpdateFeatureStatusIfNotOverridden(context.Background(), "E01-F01", models.FeatureStatusActive)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if !updated {
			t.Error("expected updated to be true")
		}
	})

	t.Run("success - skipped due to override", func(t *testing.T) {
		repo := &mockFeatureRepo{
			getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
				return &models.Feature{ID: 42, Key: "E01-F01", Status: models.FeatureStatusDraft}, nil
			},
			updateStatusIfNotOverriddenFn: func(ctx context.Context, featureID int64, newStatus models.FeatureStatus) (bool, error) {
				return false, nil
			},
		}

		svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
		updated, err := svc.UpdateFeatureStatusIfNotOverridden(context.Background(), "E01-F01", models.FeatureStatusActive)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if updated {
			t.Error("expected updated to be false")
		}
	})

	t.Run("error - feature not found", func(t *testing.T) {
		repo := &mockFeatureRepo{
			getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
				return nil, fmt.Errorf("feature not found")
			},
		}

		svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil, nil)
		_, err := svc.UpdateFeatureStatusIfNotOverridden(context.Background(), "E01-F99", models.FeatureStatusActive)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "does not exist") {
			t.Errorf("expected 'does not exist' in error, got: %v", err)
		}
	})
}
