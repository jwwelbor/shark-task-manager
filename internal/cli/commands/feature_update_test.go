package commands

import (
	"testing"
)

// TestFeatureUpdateCommand_Exists tests that the feature update command exists
func TestFeatureUpdateCommand_Exists(t *testing.T) {
	// Verify the feature update command is registered
	var found bool
	for _, cmd := range featureCmd.Commands() {
		if cmd.Use == "update <feature-key>" {
			found = true

			// Verify it has the expected flags
			if cmd.Flags().Lookup("title") == nil {
				t.Error("feature update command missing --title flag")
			}
			if cmd.Flags().Lookup("description") == nil {
				t.Error("feature update command missing --description flag")
			}
			if cmd.Flags().Lookup("status") == nil {
				t.Error("feature update command missing --status flag")
			}
			if cmd.Flags().Lookup("execution-order") == nil {
				t.Error("feature update command missing --execution-order flag")
			}
			if cmd.Flags().Lookup("filename") == nil {
				t.Error("feature update command missing --filename flag")
			}
			if cmd.Flags().Lookup("path") == nil {
				t.Error("feature update command missing --path flag")
			}
			if cmd.Flags().Lookup("key") == nil {
				t.Error("feature update command missing --key flag")
			}

			break
		}
	}

	if !found {
		t.Fatal("feature update command not found in feature subcommands")
	}
}

// TestFeatureCreateCommand_KeyFlag tests that the feature create command has a --key flag
func TestFeatureCreateCommand_KeyFlag(t *testing.T) {
	// Verify the feature create command is registered
	var found bool
	for _, cmd := range featureCmd.Commands() {
		// Updated to match new positional argument syntax
		if cmd.Use == "create [EPIC] <title> [flags]" {
			found = true

			// Verify it has the --key flag
			if cmd.Flags().Lookup("key") == nil {
				t.Error("feature create command missing --key flag")
			}

			break
		}
	}

	if !found {
		t.Fatal("feature create command not found in feature subcommands")
	}
}
