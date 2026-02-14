package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// mockFeatureRepo implements FeatureRepository for testing.
type mockFeatureRepo struct {
	getByKeyFn func(ctx context.Context, key string) (*models.Feature, error)
	updateFn   func(ctx context.Context, feature *models.Feature) error
}

func (m *mockFeatureRepo) GetByKey(ctx context.Context, key string) (*models.Feature, error) {
	if m.getByKeyFn != nil {
		return m.getByKeyFn(ctx, key)
	}
	return nil, nil
}

func (m *mockFeatureRepo) Update(ctx context.Context, feature *models.Feature) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, feature)
	}
	return nil
}

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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
	ctx := context.Background()

	_, err := svc.GetNextStatus(ctx, "E99-F01")
	if err == nil {
		t.Fatal("expected error for not-found feature")
	}
}

func TestFeatureService_ValidateStatus(t *testing.T) {
	repo := &mockFeatureRepo{}
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)

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

	svc := NewFeatureService(repo, newTestFeatureWorkflowServiceWithActions(t), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowServiceWithActions(t), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowServiceWithActions(t), nil, nil)
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
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)

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
	svc := NewFeatureService(repo, newTestFeatureWorkflowServiceWithActions(t), nil, nil)

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
	svc := NewFeatureService(repo, newTestFeatureWorkflowServiceWithActions(t), nil, nil)

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
	svc := NewFeatureService(repo, newTestFeatureWorkflowServiceWithActions(t), nil, nil)

	feature := &models.Feature{Key: "E16-F01", Title: "Test Feature", Status: "completed"}
	ctx := context.Background()
	action := svc.resolveAction(ctx, feature, "completed")
	if action != nil {
		t.Errorf("expected nil action for 'completed' status, got: %+v", action)
	}
}
