package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// mockFeatureRepo implements FeatureRepository for testing.
type mockFeatureRepo struct {
	getByKeyFn               func(ctx context.Context, key string) (*models.Feature, error)
	getByIDFn                func(ctx context.Context, id int64) (*models.Feature, error)
	updateFn                 func(ctx context.Context, feature *models.Feature) error
	listFn                   func(ctx context.Context) ([]*models.Feature, error)
	listByEpicFn             func(ctx context.Context, epicID int64) ([]*models.Feature, error)
	getTaskStatusBreakdownFn func(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, taskRepo)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, taskRepo)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, taskRepo)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, taskRepo)
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
	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, taskRepo)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, taskRepo)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, taskRepo)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService(), nil, nil)
	ctx := context.Background()

	err := svc.RecalculateAndSetProgressByKey(ctx, "E99-F01")
	if err == nil {
		t.Fatal("expected error for not-found feature")
	}
}
