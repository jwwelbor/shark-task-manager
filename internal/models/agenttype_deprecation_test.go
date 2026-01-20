package models

import (
	"testing"
)

// TestAgentTypeDeprecationBackwardCompatibility verifies that deprecated constants still work
func TestAgentTypeDeprecationBackwardCompatibility(t *testing.T) {
	// Test that all deprecated constants are still functional
	agents := []AgentType{
		AgentTypeFrontend,
		AgentTypeBackend,
		AgentTypeAPI,
		AgentTypeTesting,
		AgentTypeDevOps,
		AgentTypeGeneral,
	}

	for _, agent := range agents {
		if string(agent) == "" {
			t.Errorf("AgentType constant is empty: %v", agent)
		}

		// Verify ValidateAgentType still accepts deprecated constants
		if err := ValidateAgentType(string(agent)); err != nil {
			t.Errorf("ValidateAgentType rejected deprecated constant %s: %v", agent, err)
		}
	}
}

// TestAgentTypeCustomTypes verifies that ValidateAgentType accepts any non-empty string
func TestAgentTypeCustomTypes(t *testing.T) {
	customTypes := []string{
		"architect",
		"business-analyst",
		"qa",
		"custom-agent-123",
		"frontend", // Old constant still works as string
		"product-manager",
		"tech-lead",
	}

	for _, agentType := range customTypes {
		err := ValidateAgentType(agentType)
		if err != nil {
			t.Errorf("ValidateAgentType rejected custom type '%s': %v", agentType, err)
		}
	}
}

// TestAgentTypeValidationRejectsEmpty verifies empty strings are rejected
func TestAgentTypeValidationRejectsEmpty(t *testing.T) {
	err := ValidateAgentType("")
	if err == nil {
		t.Error("ValidateAgentType should reject empty string")
	}
}
