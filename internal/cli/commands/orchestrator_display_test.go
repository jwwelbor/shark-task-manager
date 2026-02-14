package commands

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

// TestDisplayOrchestratorAction_Nil verifies that calling displayOrchestratorAction
// with nil does not panic and outputs the "None configured" fallback.
func TestDisplayOrchestratorAction_Nil(t *testing.T) {
	// Should not panic when called with nil
	displayOrchestratorAction(nil)
}

// TestDisplayOrchestratorAction_WithAction verifies that a fully populated action
// is displayed without panicking.
func TestDisplayOrchestratorAction_WithAction(t *testing.T) {
	action := &config.PopulatedAction{
		Action:      config.ActionSpawnAgent,
		AgentType:   "developer",
		Skills:      []string{"backend", "testing"},
		Instruction: "Implement T-E99-F01-001 following the specification",
	}

	// Should not panic with a full action
	displayOrchestratorAction(action)
}

// TestDisplayOrchestratorAction_PartialAction verifies that a partial action
// (empty AgentType, no Skills) is displayed without panicking.
func TestDisplayOrchestratorAction_PartialAction(t *testing.T) {
	action := &config.PopulatedAction{
		Action:      config.ActionPause,
		AgentType:   "",
		Skills:      nil,
		Instruction: "Wait for manual review before proceeding",
	}

	// Should not panic with partial fields
	displayOrchestratorAction(action)
}

// NOTE: Tests for resolveTaskAction, resolveEpicAction, resolveFeatureAction have been
// removed as these functions were moved to DisplayService in internal/services/display_service.go.
// The orchestrator action resolution logic is now tested in display_service_test.go.
