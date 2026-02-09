package commands

import (
	"context"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// mockFeatureServiceForTest wraps a mock FeatureService for testing the next-status command.
type mockFeatureServiceForTest struct {
	getNextStatusFn    func(ctx context.Context, featureKey string) (*services.NextStatusInfo, error)
	transitionStatusFn func(ctx context.Context, featureKey string, targetStatus string, force bool) (*services.TransitionResult, error)
}

func (m *mockFeatureServiceForTest) GetNextStatus(ctx context.Context, featureKey string) (*services.NextStatusInfo, error) {
	if m.getNextStatusFn != nil {
		return m.getNextStatusFn(ctx, featureKey)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockFeatureServiceForTest) TransitionStatus(ctx context.Context, featureKey string, targetStatus string, force bool) (*services.TransitionResult, error) {
	if m.transitionStatusFn != nil {
		return m.transitionStatusFn(ctx, featureKey, targetStatus, force)
	}
	return nil, fmt.Errorf("not implemented")
}

func TestBuildNextStatusResult_FeatureWithMultipleTransitions(t *testing.T) {
	info := &services.NextStatusInfo{
		EntityType:    "feature",
		EntityKey:     "E16-F01",
		CurrentStatus: "draft",
		CurrentPhase:  "planning",
		AvailableTransitions: []workflow.TransitionInfo{
			{TargetStatus: "active", Description: "Activate feature", Phase: "execution"},
			{TargetStatus: "archived", Description: "Archive feature", Phase: "done"},
		},
		IsTerminal: false,
	}

	result := buildNextStatusResult("feature", info)

	if result.EntityType != "feature" {
		t.Errorf("expected entity_type 'feature', got %q", result.EntityType)
	}
	if len(result.AvailableTransitions) != 2 {
		t.Fatalf("expected 2 transitions, got %d", len(result.AvailableTransitions))
	}
	if result.AvailableTransitions[0].Number != 1 {
		t.Errorf("expected first transition number 1, got %d", result.AvailableTransitions[0].Number)
	}
	if result.AvailableTransitions[1].Number != 2 {
		t.Errorf("expected second transition number 2, got %d", result.AvailableTransitions[1].Number)
	}
}

func TestBuildNextStatusResult_FeatureTerminal(t *testing.T) {
	info := &services.NextStatusInfo{
		EntityType:           "feature",
		EntityKey:            "E16-F01",
		CurrentStatus:        "archived",
		CurrentPhase:         "done",
		AvailableTransitions: []workflow.TransitionInfo{},
		IsTerminal:           true,
	}

	result := buildNextStatusResult("feature", info)

	if len(result.AvailableTransitions) != 0 {
		t.Errorf("expected 0 transitions for terminal status, got %d", len(result.AvailableTransitions))
	}
	if result.CurrentStatus != "archived" {
		t.Errorf("expected current_status 'archived', got %q", result.CurrentStatus)
	}
}

func TestPerformEntityTransition_FeatureSuccess(t *testing.T) {
	mock := &mockFeatureServiceForTest{
		transitionStatusFn: func(ctx context.Context, featureKey string, targetStatus string, force bool) (*services.TransitionResult, error) {
			return &services.TransitionResult{
				EntityType:   "feature",
				EntityKey:    featureKey,
				FromStatus:   "draft",
				ToStatus:     targetStatus,
				Transitioned: true,
			}, nil
		},
	}

	result := &EntityNextStatusResult{
		EntityType:    "feature",
		EntityKey:     "E16-F01",
		CurrentStatus: "draft",
	}

	ctx := context.Background()
	err := performEntityTransition(ctx, mock, nil, "E16-F01", "active", false, result)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !result.Transitioned {
		t.Error("expected transitioned=true")
	}
	if result.NewStatus != "active" {
		t.Errorf("expected new_status 'active', got %q", result.NewStatus)
	}
}

func TestPerformEntityTransition_FeatureError(t *testing.T) {
	mock := &mockFeatureServiceForTest{
		transitionStatusFn: func(ctx context.Context, featureKey string, targetStatus string, force bool) (*services.TransitionResult, error) {
			return nil, fmt.Errorf("feature not found: E99-F01")
		},
	}

	result := &EntityNextStatusResult{
		EntityType:    "feature",
		EntityKey:     "E99-F01",
		CurrentStatus: "draft",
	}

	ctx := context.Background()
	err := performEntityTransition(ctx, mock, nil, "E99-F01", "active", false, result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result.Transitioned {
		t.Error("expected transitioned=false on error")
	}
}

func TestPerformEntityTransition_FeatureForce(t *testing.T) {
	var forcePassed bool
	mock := &mockFeatureServiceForTest{
		transitionStatusFn: func(ctx context.Context, featureKey string, targetStatus string, force bool) (*services.TransitionResult, error) {
			forcePassed = force
			return &services.TransitionResult{
				EntityType:   "feature",
				EntityKey:    featureKey,
				FromStatus:   "draft",
				ToStatus:     targetStatus,
				Transitioned: true,
			}, nil
		},
	}

	result := &EntityNextStatusResult{
		EntityType:    "feature",
		EntityKey:     "E16-F01",
		CurrentStatus: "draft",
	}

	ctx := context.Background()
	err := performEntityTransition(ctx, mock, nil, "E16-F01", "custom_status", true, result)
	if err != nil {
		t.Fatalf("expected no error with force, got: %v", err)
	}

	if !forcePassed {
		t.Error("expected force=true to be passed to service")
	}
}

func TestEntityTransitionChoice_Fields(t *testing.T) {
	choice := EntityTransitionChoice{
		Number:      1,
		Status:      "active",
		Description: "Activate feature",
		Phase:       "execution",
	}

	if choice.Number != 1 {
		t.Errorf("expected number 1, got %d", choice.Number)
	}
	if choice.Status != "active" {
		t.Errorf("expected status 'active', got %q", choice.Status)
	}
	if choice.Description != "Activate feature" {
		t.Errorf("expected description 'Activate feature', got %q", choice.Description)
	}
	if choice.Phase != "execution" {
		t.Errorf("expected phase 'execution', got %q", choice.Phase)
	}
}
