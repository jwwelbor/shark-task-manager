package services

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

func TestTransitionResult_NilAction_OmittedFromJSON(t *testing.T) {
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
	if strings.Contains(jsonStr, "orchestrator_action") {
		t.Errorf("expected JSON to omit orchestrator_action when nil, got: %s", jsonStr)
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
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	jsonStr := string(data)
	if strings.Contains(jsonStr, "orchestrator_action") {
		t.Errorf("expected JSON to omit orchestrator_action when nil, got: %s", jsonStr)
	}
}

func TestNextStatusInfo_PopulatedAction_IncludedInJSON(t *testing.T) {
	info := NextStatusInfo{
		EntityType:    "feature",
		EntityKey:     "E16-F01",
		CurrentStatus: "ready_for_refinement_ba",
		OrchestratorAction: &config.PopulatedAction{
			Action:    "notify",
			AgentType: "ba",
		},
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	jsonStr := string(data)
	if !strings.Contains(jsonStr, "orchestrator_action") {
		t.Errorf("expected JSON to contain orchestrator_action when populated, got: %s", jsonStr)
	}
}
