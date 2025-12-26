package commands

import (
	"testing"
)

// TestEpicUpdateCommand_Exists tests that the epic update command exists
func TestEpicUpdateCommand_Exists(t *testing.T) {
	// Verify the epic update command is registered
	var found bool
	for _, cmd := range epicCmd.Commands() {
		if cmd.Use == "update <epic-key>" {
			found = true

			// Verify it has the expected flags
			if cmd.Flags().Lookup("title") == nil {
				t.Error("epic update command missing --title flag")
			}
			if cmd.Flags().Lookup("description") == nil {
				t.Error("epic update command missing --description flag")
			}
			if cmd.Flags().Lookup("status") == nil {
				t.Error("epic update command missing --status flag")
			}
			if cmd.Flags().Lookup("priority") == nil {
				t.Error("epic update command missing --priority flag")
			}
			if cmd.Flags().Lookup("filename") == nil {
				t.Error("epic update command missing --filename flag")
			}
			if cmd.Flags().Lookup("path") == nil {
				t.Error("epic update command missing --path flag")
			}
			if cmd.Flags().Lookup("key") == nil {
				t.Error("epic update command missing --key flag")
			}

			break
		}
	}

	if !found {
		t.Fatal("epic update command not found in epic subcommands")
	}
}

// TestEpicCreateCommand_KeyFlag tests that the epic create command has a --key flag
func TestEpicCreateCommand_KeyFlag(t *testing.T) {
	// Verify the epic create command is registered
	var found bool
	for _, cmd := range epicCmd.Commands() {
		if cmd.Use == "create <title>" {
			found = true

			// Verify it has the --key flag
			if cmd.Flags().Lookup("key") == nil {
				t.Error("epic create command missing --key flag")
			}

			break
		}
	}

	if !found {
		t.Fatal("epic create command not found in epic subcommands")
	}
}
