package services

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

func TestTransitionResult_NilAction_ExplicitNullInJSON(t *testing.T) {
	result := TransitionResult{
		EntityType:   "epic",
		EntityKey:    "E16",
		FromStatus:   "draft",
		ToStatus:     "active",
		Transitioned: true,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	jsonStr := string(data)
	// Per BA requirement: orchestrator_action must be explicit null, not omitted
	if !strings.Contains(jsonStr, `"orchestrator_action":null`) {
		t.Errorf("expected JSON to contain explicit null orchestrator_action, got: %s", jsonStr)
	}
}

func TestTransitionResult_PopulatedAction_IncludedInJSON(t *testing.T) {
	result := TransitionResult{
		EntityType:   "task",
		EntityKey:    "E16-F01-001",
		FromStatus:   "ready_for_development",
		ToStatus:     "in_development",
		Transitioned: true,
		OrchestratorAction: &config.PopulatedAction{
			Action:      "dispatch",
			AgentType:   "developer",
			Skills:      []string{"test-driven-development"},
			Instruction: "Implement the task using TDD",
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	jsonStr := string(data)
	if !strings.Contains(jsonStr, "orchestrator_action") {
		t.Errorf("expected JSON to contain orchestrator_action when populated, got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, "developer") {
		t.Errorf("expected JSON to contain agent_type 'developer', got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, "test-driven-development") {
		t.Errorf("expected JSON to contain skill 'test-driven-development', got: %s", jsonStr)
	}

	// Verify it round-trips correctly
	var parsed TransitionResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if parsed.OrchestratorAction == nil {
		t.Fatal("expected OrchestratorAction to be non-nil after round-trip")
	}
	if parsed.OrchestratorAction.AgentType != "developer" {
		t.Errorf("expected agent_type 'developer', got %s", parsed.OrchestratorAction.AgentType)
	}
}

func TestNextStatusInfo_NilAction_OmittedFromJSON(t *testing.T) {
	info := NextStatusInfo{
		EntityType:    "feature",
		EntityKey:     "E16-F01",
		CurrentStatus: "draft",
		IsTerminal:    false,
		AvailableTransitions: []TransitionInfoWithAction{
			{
				TransitionInfo: workflow.TransitionInfo{
					TargetStatus: "active",
					Phase:        "execution",
				},
				// OrchestratorAction is nil - should be omitted
			},
		},
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	jsonStr := string(data)
	// The top-level NextStatusInfo no longer has orchestrator_action field
	// Verify the transition has no orchestrator_action in JSON
	if !strings.Contains(jsonStr, "available_transitions") {
		t.Errorf("expected JSON to contain available_transitions, got: %s", jsonStr)
	}
}

func TestNextStatusInfo_PopulatedAction_IncludedInJSON(t *testing.T) {
	info := NextStatusInfo{
		EntityType:    "feature",
		EntityKey:     "E16-F01",
		CurrentStatus: "ready_for_refinement_ba",
		AvailableTransitions: []TransitionInfoWithAction{
			{
				TransitionInfo: workflow.TransitionInfo{
					TargetStatus: "in_refinement_ba",
					Phase:        "planning",
				},
				OrchestratorAction: &config.PopulatedAction{
					Action:    "notify",
					AgentType: "ba",
				},
			},
		},
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	jsonStr := string(data)
	if !strings.Contains(jsonStr, "orchestrator_action") {
		t.Errorf("expected JSON to contain orchestrator_action in transition, got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, "ba") {
		t.Errorf("expected JSON to contain agent_type 'ba', got: %s", jsonStr)
	}
}

func TestTransitionInfoWithAction_EmbedsTransitionInfo(t *testing.T) {
	// Test that embedded TransitionInfo fields are accessible directly
	transition := TransitionInfoWithAction{
		TransitionInfo: workflow.TransitionInfo{
			TargetStatus: "in_development",
			Description:  "Start development",
			Phase:        "development",
			AgentTypes:   []string{"developer"},
			Color:        "yellow",
		},
	}

	// Verify embedded fields are accessible
	if transition.TargetStatus != "in_development" {
		t.Errorf("expected TargetStatus 'in_development', got %q", transition.TargetStatus)
	}
	if transition.Phase != "development" {
		t.Errorf("expected Phase 'development', got %q", transition.Phase)
	}

	// Verify JSON flattening
	data, err := json.Marshal(transition)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	jsonStr := string(data)
	if !strings.Contains(jsonStr, "target_status") {
		t.Errorf("expected JSON to contain flattened 'target_status', got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, "in_development") {
		t.Errorf("expected JSON to contain 'in_development', got: %s", jsonStr)
	}
	if strings.Contains(jsonStr, "TransitionInfo") {
		t.Errorf("expected JSON to NOT contain 'TransitionInfo' key (should be flattened), got: %s", jsonStr)
	}
}

func TestTransitionInfoWithAction_WithAction(t *testing.T) {
	// Test that orchestrator_action appears in JSON when non-nil
	transition := TransitionInfoWithAction{
		TransitionInfo: workflow.TransitionInfo{
			TargetStatus: "in_development",
			Phase:        "development",
		},
		OrchestratorAction: &config.PopulatedAction{
			Action:      "dispatch",
			AgentType:   "developer",
			Skills:      []string{"test-driven-development"},
			Instruction: "Implement using TDD",
		},
	}

	data, err := json.Marshal(transition)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	jsonStr := string(data)
	if !strings.Contains(jsonStr, "orchestrator_action") {
		t.Errorf("expected JSON to contain 'orchestrator_action', got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, "developer") {
		t.Errorf("expected JSON to contain agent_type 'developer', got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, "test-driven-development") {
		t.Errorf("expected JSON to contain skill 'test-driven-development', got: %s", jsonStr)
	}

	// Verify round-trip
	var parsed TransitionInfoWithAction
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if parsed.OrchestratorAction == nil {
		t.Fatal("expected OrchestratorAction to be non-nil after round-trip")
	}
	if parsed.OrchestratorAction.AgentType != "developer" {
		t.Errorf("expected agent_type 'developer', got %q", parsed.OrchestratorAction.AgentType)
	}
}

func TestNextStatusInfo_JSONWithMixedActions(t *testing.T) {
	// Test that some transitions can have actions while others don't
	info := NextStatusInfo{
		EntityType:    "task",
		EntityKey:     "E16-F02-002",
		CurrentStatus: "draft",
		AvailableTransitions: []TransitionInfoWithAction{
			{
				TransitionInfo: workflow.TransitionInfo{
					TargetStatus: "ready_for_refinement_ba",
					Phase:        "planning",
				},
				OrchestratorAction: &config.PopulatedAction{
					Action:    "notify",
					AgentType: "ba",
				},
			},
			{
				TransitionInfo: workflow.TransitionInfo{
					TargetStatus: "cancelled",
					Phase:        "done",
				},
				// No action for cancelled transition
			},
		},
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	jsonStr := string(data)

	// Verify both transitions are present
	if !strings.Contains(jsonStr, "ready_for_refinement_ba") {
		t.Errorf("expected JSON to contain 'ready_for_refinement_ba', got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, "cancelled") {
		t.Errorf("expected JSON to contain 'cancelled', got: %s", jsonStr)
	}

	// Verify round-trip preserves structure
	var parsed NextStatusInfo
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(parsed.AvailableTransitions) != 2 {
		t.Fatalf("expected 2 transitions after round-trip, got %d", len(parsed.AvailableTransitions))
	}

	// First transition should have action
	if parsed.AvailableTransitions[0].OrchestratorAction == nil {
		t.Error("expected first transition to have action")
	} else if parsed.AvailableTransitions[0].OrchestratorAction.AgentType != "ba" {
		t.Errorf("expected first transition agent_type 'ba', got %q",
			parsed.AvailableTransitions[0].OrchestratorAction.AgentType)
	}

	// Second transition should NOT have action
	if parsed.AvailableTransitions[1].OrchestratorAction != nil {
		t.Error("expected second transition to have nil action")
	}
}
