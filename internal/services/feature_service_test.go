package services

import (
	"context"
	"fmt"
	"testing"

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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService())
	ctx := context.Background()

	result, err := svc.TransitionStatus(ctx, "E16-F01", "active", false)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService())
	ctx := context.Background()

	// "draft" -> "completed" is not valid in default feature workflow
	_, err := svc.TransitionStatus(ctx, "E16-F01", "completed", false)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService())
	ctx := context.Background()

	result, err := svc.TransitionStatus(ctx, "E16-F01", "custom_status", true)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService())
	ctx := context.Background()

	_, err := svc.TransitionStatus(ctx, "E99-F01", "active", false)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService())
	ctx := context.Background()

	_, err := svc.TransitionStatus(ctx, "E16-F01", "active", false)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService())
	ctx := context.Background()

	_, err := svc.TransitionStatus(ctx, "E16-F01", "active", false)
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService())
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService())
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

	svc := NewFeatureService(repo, newTestFeatureWorkflowService())
	ctx := context.Background()

	_, err := svc.GetNextStatus(ctx, "E99-F01")
	if err == nil {
		t.Fatal("expected error for not-found feature")
	}
}

func TestFeatureService_ValidateStatus(t *testing.T) {
	repo := &mockFeatureRepo{}
	svc := NewFeatureService(repo, newTestFeatureWorkflowService())

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
