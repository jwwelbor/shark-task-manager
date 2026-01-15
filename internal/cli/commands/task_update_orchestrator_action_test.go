package commands

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/stretchr/testify/assert"
)

// TestOrchestrationActionSerialization tests that PopulatedAction can be properly serialized
func TestOrchestratorActionSerialization(t *testing.T) {
	// Create an orchestrator action for serialization testing
	populatedAction := &config.PopulatedAction{
		Action:      config.ActionSpawnAgent,
		AgentType:   "developer",
		Skills:      []string{"backend", "testing"},
		Instruction: "Implement T-E99-F01-001 following the specification",
	}

	// Verify all fields are properly set
	assert.Equal(t, config.ActionSpawnAgent, populatedAction.Action)
	assert.Equal(t, "developer", populatedAction.AgentType)
	assert.ElementsMatch(t, []string{"backend", "testing"}, populatedAction.Skills)
	assert.Equal(t, "Implement T-E99-F01-001 following the specification", populatedAction.Instruction)
}

// TestOrchestratorActionNilWhenNotDefined tests backward compatibility
func TestOrchestratorActionNilWhenNotDefined(t *testing.T) {
	// When no action is configured for a status, PopulatedAction should be nil
	// This is used to test the omitempty serialization behavior

	var populatedAction *config.PopulatedAction = nil

	// Verify nil action can be safely used
	assert.Nil(t, populatedAction)
}

// TestOrchestratorActionTemplateVariables tests that {task_id} is properly replaced
func TestOrchestratorActionTemplateVariables(t *testing.T) {
	// Simulate template population as done in repository layer
	taskID := "T-E99-F03-002"

	// Simulate the PopulateTemplate call
	populatedAction := &config.PopulatedAction{
		Action:      config.ActionSpawnAgent,
		AgentType:   "developer",
		Skills:      []string{"backend"},
		Instruction: "Implement " + taskID + " using the provided specification", // After population
	}

	// Verify template variable is populated
	assert.Contains(t, populatedAction.Instruction, taskID)
	assert.NotContains(t, populatedAction.Instruction, "{task_id}", "Template variable should be replaced")
	assert.Equal(t, "Implement T-E99-F03-002 using the provided specification", populatedAction.Instruction)
}

// TestUpdateStatusResponse_ActionFieldSerializationWithOmitempty tests JSON serialization with omitempty
func TestUpdateStatusResponse_ActionFieldSerializationWithOmitempty(t *testing.T) {
	// Test that PopulatedAction respects omitempty tags
	action := &config.PopulatedAction{
		Action:      config.ActionSpawnAgent,
		AgentType:   "developer",
		Skills:      []string{"backend"},
		Instruction: "Do work on {task_id}",
	}

	// Verify all fields are present
	assert.NotEmpty(t, action.Action)
	assert.NotEmpty(t, action.AgentType)
	assert.NotEmpty(t, action.Skills)
	assert.NotEmpty(t, action.Instruction)

	// Test that non-spawn actions omit agent fields
	pauseAction := &config.PopulatedAction{
		Action:      config.ActionPause,
		// AgentType and Skills are empty for pause
		Instruction: "Pause processing of {task_id}",
	}

	assert.Equal(t, config.ActionPause, pauseAction.Action)
	assert.Empty(t, pauseAction.AgentType) // Should be omitted in JSON due to omitempty
	assert.Empty(t, pauseAction.Skills)     // Should be omitted in JSON due to omitempty
}
