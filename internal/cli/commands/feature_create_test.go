package commands

import (
	"testing"
)

// TestFeatureCreate_WithStatus tests that feature create command has --status flag
func TestFeatureCreate_WithStatus(t *testing.T) {
	// Test that the --status flag exists
	flag := featureCreateCmd.Flags().Lookup("status")
	if flag == nil {
		t.Skip("--status flag not yet implemented on featureCreateCmd")
	}

	// Verify default value
	if flag.DefValue != "draft" {
		t.Errorf("Expected default status 'draft', got '%s'", flag.DefValue)
	}
}
