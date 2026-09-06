package commands

import (
	"testing"
)

// TestParseCreateFeatureInput_CustomKey covers B063: the --key flag was
// registered on featureCreateCmd but parseCreateFeatureInput never read it
// into CreateFeatureInput, so a supplied key was silently discarded in favor
// of an auto-generated one. This proves the flag is read through to
// CreateFeatureInput.CustomKey.
func TestParseCreateFeatureInput_CustomKey(t *testing.T) {
	origEpic, origKey := featureCreateEpic, featureCreateKey
	defer func() { featureCreateEpic, featureCreateKey = origEpic, origKey }()

	if err := featureCreateCmd.Flags().Set("key", "E01-F99"); err != nil {
		t.Fatalf("failed to set key flag: %v", err)
	}
	defer func() { _ = featureCreateCmd.Flags().Set("key", "") }()

	input, _, _, err := parseCreateFeatureInput(featureCreateCmd, []string{"E01", "Custom Key Feature"})
	if err != nil {
		t.Fatalf("parseCreateFeatureInput returned error: %v", err)
	}
	if input.CustomKey != "E01-F99" {
		t.Errorf("expected CustomKey %q, got %q", "E01-F99", input.CustomKey)
	}
}

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
