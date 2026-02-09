package services

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

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
		AvailableTransitions: []workflow.TransitionInfo{
			{TargetStatus: "active", Phase: "execution"},
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
