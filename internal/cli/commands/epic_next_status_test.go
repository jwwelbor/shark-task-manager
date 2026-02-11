package commands

import (
	"context"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// mockEpicServiceForTest wraps a mock EpicService for testing the next-status command.
type mockEpicServiceForTest struct {
	getNextStatusFn    func(ctx context.Context, epicKey string) (*services.NextStatusInfo, error)
	transitionStatusFn func(ctx context.Context, epicKey string, targetStatus string, force bool) (*services.TransitionResult, error)
}

func (m *mockEpicServiceForTest) GetNextStatus(ctx context.Context, epicKey string) (*services.NextStatusInfo, error) {
	if m.getNextStatusFn != nil {
		return m.getNextStatusFn(ctx, epicKey)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockEpicServiceForTest) TransitionStatus(ctx context.Context, epicKey string, targetStatus string, force bool) (*services.TransitionResult, error) {
	if m.transitionStatusFn != nil {
		return m.transitionStatusFn(ctx, epicKey, targetStatus, force)
	}
	return nil, fmt.Errorf("not implemented")
}

func TestBuildNextStatusResult_Epic(t *testing.T) {
	info := &services.NextStatusInfo{
		EntityType:    "epic",
		EntityKey:     "E16",
		CurrentStatus: "draft",
		CurrentPhase:  "planning",
		AvailableTransitions: []services.TransitionInfoWithAction{
			{
				TransitionInfo: workflow.TransitionInfo{
					TargetStatus: "active",
					Description:  "Activate epic",
					Phase:        "execution",
				},
			},
			{
				TransitionInfo: workflow.TransitionInfo{
					TargetStatus: "archived",
					Description:  "Archive epic",
					Phase:        "done",
				},
			},
		},
		IsTerminal: false,
	}

	result := buildNextStatusResult("epic", info)

	if result.EntityType != "epic" {
		t.Errorf("expected entity_type 'epic', got %q", result.EntityType)
	}
	if result.EntityKey != "E16" {
		t.Errorf("expected entity_key 'E16', got %q", result.EntityKey)
	}
	if result.CurrentStatus != "draft" {
		t.Errorf("expected current_status 'draft', got %q", result.CurrentStatus)
	}
	if result.CurrentPhase != "planning" {
		t.Errorf("expected current_phase 'planning', got %q", result.CurrentPhase)
	}
	if len(result.AvailableTransitions) != 2 {
		t.Fatalf("expected 2 transitions, got %d", len(result.AvailableTransitions))
	}
	if result.AvailableTransitions[0].Status != "active" {
		t.Errorf("expected first transition status 'active', got %q", result.AvailableTransitions[0].Status)
	}
	if result.AvailableTransitions[0].Number != 1 {
		t.Errorf("expected first transition number 1, got %d", result.AvailableTransitions[0].Number)
	}
	if result.AvailableTransitions[1].Status != "archived" {
		t.Errorf("expected second transition status 'archived', got %q", result.AvailableTransitions[1].Status)
	}
}

func TestBuildNextStatusResult_Terminal(t *testing.T) {
	info := &services.NextStatusInfo{
		EntityType:           "epic",
		EntityKey:            "E16",
		CurrentStatus:        "archived",
		CurrentPhase:         "done",
		AvailableTransitions: []services.TransitionInfoWithAction{},
		IsTerminal:           true,
	}

	result := buildNextStatusResult("epic", info)

	if len(result.AvailableTransitions) != 0 {
		t.Errorf("expected 0 transitions for terminal status, got %d", len(result.AvailableTransitions))
	}
}

func TestBuildNextStatusResult_Feature(t *testing.T) {
	info := &services.NextStatusInfo{
		EntityType:    "feature",
		EntityKey:     "E16-F01",
		CurrentStatus: "draft",
		CurrentPhase:  "planning",
		AvailableTransitions: []services.TransitionInfoWithAction{
			{
				TransitionInfo: workflow.TransitionInfo{
					TargetStatus: "active",
					Description:  "Activate feature",
					Phase:        "execution",
				},
			},
		},
		IsTerminal: false,
	}

	result := buildNextStatusResult("feature", info)

	if result.EntityType != "feature" {
		t.Errorf("expected entity_type 'feature', got %q", result.EntityType)
	}
	if result.EntityKey != "E16-F01" {
		t.Errorf("expected entity_key 'E16-F01', got %q", result.EntityKey)
	}
	if len(result.AvailableTransitions) != 1 {
		t.Fatalf("expected 1 transition, got %d", len(result.AvailableTransitions))
	}
}

func TestPerformEntityTransition_Success(t *testing.T) {
	mock := &mockEpicServiceForTest{
		transitionStatusFn: func(ctx context.Context, epicKey string, targetStatus string, force bool) (*services.TransitionResult, error) {
			return &services.TransitionResult{
				EntityType:   "epic",
				EntityKey:    epicKey,
				FromStatus:   "draft",
				ToStatus:     targetStatus,
				Transitioned: true,
			}, nil
		},
	}

	result := &EntityNextStatusResult{
		EntityType:    "epic",
		EntityKey:     "E16",
		CurrentStatus: "draft",
	}

	ctx := context.Background()
	err := performEntityTransition(ctx, mock, nil, "E16", "active", false, result)
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

func TestPerformEntityTransition_Error(t *testing.T) {
	mock := &mockEpicServiceForTest{
		transitionStatusFn: func(ctx context.Context, epicKey string, targetStatus string, force bool) (*services.TransitionResult, error) {
			return nil, fmt.Errorf("invalid transition: draft -> completed")
		},
	}

	result := &EntityNextStatusResult{
		EntityType:    "epic",
		EntityKey:     "E16",
		CurrentStatus: "draft",
	}

	ctx := context.Background()
	err := performEntityTransition(ctx, mock, nil, "E16", "completed", false, result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if result.Transitioned {
		t.Error("expected transitioned=false on error")
	}
}

func TestPerformEntityTransition_Force(t *testing.T) {
	var forcePassed bool
	mock := &mockEpicServiceForTest{
		transitionStatusFn: func(ctx context.Context, epicKey string, targetStatus string, force bool) (*services.TransitionResult, error) {
			forcePassed = force
			return &services.TransitionResult{
				EntityType:   "epic",
				EntityKey:    epicKey,
				FromStatus:   "draft",
				ToStatus:     targetStatus,
				Transitioned: true,
			}, nil
		},
	}

	result := &EntityNextStatusResult{
		EntityType:    "epic",
		EntityKey:     "E16",
		CurrentStatus: "draft",
	}

	ctx := context.Background()
	err := performEntityTransition(ctx, mock, nil, "E16", "custom_status", true, result)
	if err != nil {
		t.Fatalf("expected no error with force, got: %v", err)
	}

	if !forcePassed {
		t.Error("expected force=true to be passed to service")
	}
	if result.NewStatus != "custom_status" {
		t.Errorf("expected new_status 'custom_status', got %q", result.NewStatus)
	}
}

func TestEntityNextStatusResult_JSONFields(t *testing.T) {
	result := &EntityNextStatusResult{
		EntityType:    "epic",
		EntityKey:     "E16",
		CurrentStatus: "draft",
		CurrentPhase:  "planning",
		AvailableTransitions: []EntityTransitionChoice{
			{Number: 1, Status: "active", Description: "Activate epic", Phase: "execution"},
		},
		NewStatus:    "active",
		Transitioned: true,
		Message:      "Transitioned: draft -> active",
	}

	if result.EntityType != "epic" {
		t.Errorf("expected entity_type 'epic', got %q", result.EntityType)
	}
	if result.NewStatus != "active" {
		t.Errorf("expected new_status 'active', got %q", result.NewStatus)
	}
	if !result.Transitioned {
		t.Error("expected transitioned=true")
	}
	if result.Message != "Transitioned: draft -> active" {
		t.Errorf("unexpected message: %q", result.Message)
	}
}
