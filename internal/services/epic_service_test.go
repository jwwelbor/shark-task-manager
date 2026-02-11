package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// mockEpicRepo implements EpicRepository for testing.
type mockEpicRepo struct {
	getByKeyFn func(ctx context.Context, key string) (*models.Epic, error)
	updateFn   func(ctx context.Context, epic *models.Epic) error
}

func (m *mockEpicRepo) GetByKey(ctx context.Context, key string) (*models.Epic, error) {
	if m.getByKeyFn != nil {
		return m.getByKeyFn(ctx, key)
	}
	return nil, nil
}

func (m *mockEpicRepo) Update(ctx context.Context, epic *models.Epic) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, epic)
	}
	return nil
}

// newTestEpicWorkflowService creates a workflow.Service with default config for testing.
func newTestEpicWorkflowService() *workflow.Service {
	return workflow.NewService("")
}

// newTestEpicWorkflowServiceWithActions creates a workflow.Service backed by a temp config
// that has orchestrator_action defined on the "active" status of the epic workflow.
// The "draft" status has no orchestrator_action, allowing tests to verify both branches.
func newTestEpicWorkflowServiceWithActions(t *testing.T) *workflow.Service {
	t.Helper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")

	configJSON := `{
		"epic_workflow": {
			"status_flow_version": "1.0",
			"status_flow": {
				"draft": ["active", "archived"],
				"active": ["completed", "archived"],
				"completed": ["archived"],
				"archived": []
			},
			"status_metadata": {
				"draft": {
					"color": "gray",
					"description": "Epic created, not yet started",
					"phase": "planning"
				},
				"active": {
					"color": "blue",
					"description": "Epic in progress",
					"phase": "execution",
					"orchestrator_action": {
						"action": "spawn_agent",
						"agent_type": "developer",
						"skills": ["implementation", "testing"],
						"instruction_template": "Work on epic {id} features"
					}
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

	// Clear workflow cache so fresh config is loaded
	config.ClearWorkflowCache()
	t.Cleanup(func() {
		config.ClearWorkflowCache()
	})

	return workflow.NewService(tmpDir)
}

func TestEpicService_TransitionStatus_Valid(t *testing.T) {
	var updatedEpic *models.Epic
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				Key:    "E16",
				Status: models.EpicStatusDraft,
			}, nil
		},
		updateFn: func(ctx context.Context, epic *models.Epic) error {
			updatedEpic = epic
			return nil
		},
	}

	svc := NewEpicService(repo, newTestEpicWorkflowService())
	ctx := context.Background()

	result, err := svc.TransitionStatus(ctx, "E16", "active", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.EntityType != "epic" {
		t.Errorf("expected entity_type 'epic', got %q", result.EntityType)
	}
	if result.EntityKey != "E16" {
		t.Errorf("expected entity_key 'E16', got %q", result.EntityKey)
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
	if updatedEpic == nil {
		t.Fatal("expected Update to be called")
	}
	if string(updatedEpic.Status) != "active" {
		t.Errorf("expected epic status 'active', got %q", updatedEpic.Status)
	}
}

func TestEpicService_TransitionStatus_Invalid(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				Key:    "E16",
				Status: models.EpicStatusDraft,
			}, nil
		},
	}

	svc := NewEpicService(repo, newTestEpicWorkflowService())
	ctx := context.Background()

	// "draft" -> "completed" is not a valid direct transition in default epic workflow
	_, err := svc.TransitionStatus(ctx, "E16", "completed", false)
	if err == nil {
		t.Fatal("expected error for invalid transition, got nil")
	}
}

func TestEpicService_TransitionStatus_Force(t *testing.T) {
	var updatedEpic *models.Epic
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				Key:    "E16",
				Status: models.EpicStatusDraft,
			}, nil
		},
		updateFn: func(ctx context.Context, epic *models.Epic) error {
			updatedEpic = epic
			return nil
		},
	}

	svc := NewEpicService(repo, newTestEpicWorkflowService())
	ctx := context.Background()

	// Force should bypass validation - even arbitrary strings should work
	result, err := svc.TransitionStatus(ctx, "E16", "custom_status", true)
	if err != nil {
		t.Fatalf("expected no error with force, got: %v", err)
	}
	if result.ToStatus != "custom_status" {
		t.Errorf("expected to_status 'custom_status', got %q", result.ToStatus)
	}
	if updatedEpic == nil {
		t.Fatal("expected Update to be called")
	}
	if string(updatedEpic.Status) != "custom_status" {
		t.Errorf("expected epic status 'custom_status', got %q", updatedEpic.Status)
	}
}

func TestEpicService_TransitionStatus_NotFound(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, nil
		},
	}

	svc := NewEpicService(repo, newTestEpicWorkflowService())
	ctx := context.Background()

	_, err := svc.TransitionStatus(ctx, "E99", "active", false)
	if err == nil {
		t.Fatal("expected error for not-found epic")
	}
	if err.Error() != "epic not found: E99" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestEpicService_TransitionStatus_RepoError(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, fmt.Errorf("db connection failed")
		},
	}

	svc := NewEpicService(repo, newTestEpicWorkflowService())
	ctx := context.Background()

	_, err := svc.TransitionStatus(ctx, "E16", "active", false)
	if err == nil {
		t.Fatal("expected error from repo failure")
	}
}

func TestEpicService_TransitionStatus_UpdateError(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				Key:    "E16",
				Status: models.EpicStatusDraft,
			}, nil
		},
		updateFn: func(ctx context.Context, epic *models.Epic) error {
			return fmt.Errorf("update failed")
		},
	}

	svc := NewEpicService(repo, newTestEpicWorkflowService())
	ctx := context.Background()

	_, err := svc.TransitionStatus(ctx, "E16", "active", false)
	if err == nil {
		t.Fatal("expected error from update failure")
	}
}

func TestEpicService_GetNextStatus(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				Key:    "E16",
				Status: models.EpicStatusDraft,
			}, nil
		},
	}

	svc := NewEpicService(repo, newTestEpicWorkflowService())
	ctx := context.Background()

	info, err := svc.GetNextStatus(ctx, "E16")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if info.EntityType != "epic" {
		t.Errorf("expected entity_type 'epic', got %q", info.EntityType)
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

func TestEpicService_GetNextStatus_Terminal(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				Key:    "E16",
				Status: models.EpicStatusArchived,
			}, nil
		},
	}

	svc := NewEpicService(repo, newTestEpicWorkflowService())
	ctx := context.Background()

	info, err := svc.GetNextStatus(ctx, "E16")
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

func TestEpicService_GetNextStatus_NotFound(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, nil
		},
	}

	svc := NewEpicService(repo, newTestEpicWorkflowService())
	ctx := context.Background()

	_, err := svc.GetNextStatus(ctx, "E99")
	if err == nil {
		t.Fatal("expected error for not-found epic")
	}
}

func TestEpicService_ValidateStatus(t *testing.T) {
	repo := &mockEpicRepo{}
	svc := NewEpicService(repo, newTestEpicWorkflowService())

	// Valid epic statuses
	for _, status := range []string{"draft", "active", "completed", "archived"} {
		if err := svc.ValidateStatus(status); err != nil {
			t.Errorf("expected %q to be valid, got error: %v", status, err)
		}
	}

	// Invalid status
	if err := svc.ValidateStatus("in_progress"); err == nil {
		t.Error("expected 'in_progress' to be invalid for epic workflow")
	}
}

func TestTransitionResult_JSONSerialization(t *testing.T) {
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

	var parsed TransitionResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.EntityType != "epic" {
		t.Errorf("expected entity_type 'epic', got %q", parsed.EntityType)
	}
	if parsed.FromStatus != "draft" {
		t.Errorf("expected from_status 'draft', got %q", parsed.FromStatus)
	}
	if parsed.ToStatus != "active" {
		t.Errorf("expected to_status 'active', got %q", parsed.ToStatus)
	}
}

func TestNextStatusInfo_JSONSerialization(t *testing.T) {
	info := &NextStatusInfo{
		EntityType:    "epic",
		EntityKey:     "E16",
		CurrentStatus: "draft",
		CurrentPhase:  "planning",
		AvailableTransitions: []TransitionInfoWithAction{
			{
				TransitionInfo: workflow.TransitionInfo{
					TargetStatus: "active",
					Phase:        "execution",
				},
			},
		},
		IsTerminal: false,
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed NextStatusInfo
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.EntityType != "epic" {
		t.Errorf("expected entity_type 'epic', got %q", parsed.EntityType)
	}
	if len(parsed.AvailableTransitions) != 1 {
		t.Fatalf("expected 1 transition, got %d", len(parsed.AvailableTransitions))
	}
	if parsed.AvailableTransitions[0].TargetStatus != "active" {
		t.Errorf("expected target_status 'active', got %q", parsed.AvailableTransitions[0].TargetStatus)
	}
}

// --- Tests for resolveAction integration ---

func TestEpicService_TransitionStatus_WithAction(t *testing.T) {
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

	svc := NewEpicService(repo, newTestEpicWorkflowServiceWithActions(t))
	ctx := context.Background()

	// Transition to "active" which has an orchestrator_action defined
	result, err := svc.TransitionStatus(ctx, "E16", "active", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.OrchestratorAction == nil {
		t.Fatal("expected OrchestratorAction to be populated for 'active' status")
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
	// Verify template was populated with entity key
	expectedInstruction := "Work on epic E16 features"
	if result.OrchestratorAction.Instruction != expectedInstruction {
		t.Errorf("expected instruction %q, got %q", expectedInstruction, result.OrchestratorAction.Instruction)
	}
}

func TestEpicService_TransitionStatus_WithoutAction(t *testing.T) {
	// Start from "active" to transition to "completed" (no action defined on completed)
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				Key:    "E16",
				Status: models.EpicStatusActive,
			}, nil
		},
		updateFn: func(ctx context.Context, epic *models.Epic) error {
			return nil
		},
	}

	svc := NewEpicService(repo, newTestEpicWorkflowServiceWithActions(t))
	ctx := context.Background()

	// Transition to "completed" which has no orchestrator_action
	result, err := svc.TransitionStatus(ctx, "E16", "completed", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.OrchestratorAction != nil {
		t.Errorf("expected nil OrchestratorAction for 'completed' status, got %+v", result.OrchestratorAction)
	}
}

func TestEpicService_GetNextStatus_WithActions(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				Key:    "E16",
				Status: models.EpicStatusDraft,
			}, nil
		},
	}

	svc := NewEpicService(repo, newTestEpicWorkflowServiceWithActions(t))
	ctx := context.Background()

	info, err := svc.GetNextStatus(ctx, "E16")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// From draft, transitions are: active (has action), archived (no action)
	if len(info.AvailableTransitions) == 0 {
		t.Fatal("expected available transitions for draft status")
	}

	// Find the "active" transition
	var activeTransition *TransitionInfoWithAction
	var archivedTransition *TransitionInfoWithAction
	for i := range info.AvailableTransitions {
		if info.AvailableTransitions[i].TargetStatus == "active" {
			activeTransition = &info.AvailableTransitions[i]
		}
		if info.AvailableTransitions[i].TargetStatus == "archived" {
			archivedTransition = &info.AvailableTransitions[i]
		}
	}

	if activeTransition == nil {
		t.Fatal("expected transition to 'active' status")
	}
	if activeTransition.OrchestratorAction == nil {
		t.Fatal("expected OrchestratorAction on 'active' transition")
	}
	if activeTransition.OrchestratorAction.Action != "spawn_agent" {
		t.Errorf("expected action 'spawn_agent', got %q", activeTransition.OrchestratorAction.Action)
	}
	// Template should be populated with the entity key
	expectedInstruction := "Work on epic E16 features"
	if activeTransition.OrchestratorAction.Instruction != expectedInstruction {
		t.Errorf("expected instruction %q, got %q", expectedInstruction, activeTransition.OrchestratorAction.Instruction)
	}

	if archivedTransition == nil {
		t.Fatal("expected transition to 'archived' status")
	}
	if archivedTransition.OrchestratorAction != nil {
		t.Errorf("expected nil OrchestratorAction on 'archived' transition, got %+v", archivedTransition.OrchestratorAction)
	}
}

func TestEpicService_resolveAction_NilWorkflow(t *testing.T) {
	// Create an EpicService with a default workflow (no actions defined)
	// resolveAction should return nil gracefully
	repo := &mockEpicRepo{}
	svc := NewEpicService(repo, newTestEpicWorkflowService())

	epic := &models.Epic{Key: "E16", Title: "Test Epic", Status: "draft"}
	action := svc.resolveAction(epic, "draft")
	if action != nil {
		t.Errorf("expected nil action for default workflow (no actions defined), got %+v", action)
	}
}

func TestEpicService_resolveAction_NonexistentStatus(t *testing.T) {
	// resolveAction with a status not in metadata should return nil
	repo := &mockEpicRepo{}
	svc := NewEpicService(repo, newTestEpicWorkflowServiceWithActions(t))

	epic := &models.Epic{Key: "E16", Title: "Test Epic", Status: "nonexistent_status"}
	action := svc.resolveAction(epic, "nonexistent_status")
	if action != nil {
		t.Errorf("expected nil action for nonexistent status, got %+v", action)
	}
}

func TestEpicService_resolveAction_StatusWithoutAction(t *testing.T) {
	// resolveAction for a status that exists but has no orchestrator_action
	repo := &mockEpicRepo{}
	svc := NewEpicService(repo, newTestEpicWorkflowServiceWithActions(t))

	epic := &models.Epic{Key: "E16", Title: "Test Epic", Status: "draft"}
	action := svc.resolveAction(epic, "draft")
	if action != nil {
		t.Errorf("expected nil action for 'draft' status (no action defined), got %+v", action)
	}
}

func TestEpicService_resolveAction_StatusWithAction(t *testing.T) {
	repo := &mockEpicRepo{}
	svc := NewEpicService(repo, newTestEpicWorkflowServiceWithActions(t))

	epic := &models.Epic{Key: "E16", Title: "Test Epic", Status: "active"}
	action := svc.resolveAction(epic, "active")
	if action == nil {
		t.Fatal("expected non-nil action for 'active' status")
	}
	if action.Action != "spawn_agent" {
		t.Errorf("expected action 'spawn_agent', got %q", action.Action)
	}
	if action.AgentType != "developer" {
		t.Errorf("expected agent_type 'developer', got %q", action.AgentType)
	}
	if len(action.Skills) != 2 || action.Skills[0] != "implementation" || action.Skills[1] != "testing" {
		t.Errorf("expected skills [implementation, testing], got %v", action.Skills)
	}
	expectedInstruction := "Work on epic E16 features"
	if action.Instruction != expectedInstruction {
		t.Errorf("expected instruction %q, got %q", expectedInstruction, action.Instruction)
	}
}

func TestEpicService_TransitionStatus_ActionJSON(t *testing.T) {
	// Verify the action is properly serialized in JSON output
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

	svc := NewEpicService(repo, newTestEpicWorkflowServiceWithActions(t))
	ctx := context.Background()

	result, err := svc.TransitionStatus(ctx, "E16", "active", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed TransitionResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.OrchestratorAction == nil {
		t.Fatal("expected orchestrator_action in JSON output")
	}
	if parsed.OrchestratorAction.Action != "spawn_agent" {
		t.Errorf("expected action 'spawn_agent' in JSON, got %q", parsed.OrchestratorAction.Action)
	}
}
