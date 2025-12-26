package commands

import (
	"testing"
)

// TestTaskUpdateCommand_Exists tests that the task update command exists
func TestTaskUpdateCommand_Exists(t *testing.T) {
	// Verify the task update command is registered
	var found bool
	for _, cmd := range taskCmd.Commands() {
		if cmd.Use == "update <task-key>" {
			found = true

			// Verify it has the expected flags
			if cmd.Flags().Lookup("title") == nil {
				t.Error("task update command missing --title flag")
			}
			if cmd.Flags().Lookup("description") == nil {
				t.Error("task update command missing --description flag")
			}
			if cmd.Flags().Lookup("priority") == nil {
				t.Error("task update command missing --priority flag")
			}
			if cmd.Flags().Lookup("agent") == nil {
				t.Error("task update command missing --agent flag")
			}
			if cmd.Flags().Lookup("filename") == nil {
				t.Error("task update command missing --filename flag")
			}
			if cmd.Flags().Lookup("depends-on") == nil {
				t.Error("task update command missing --depends-on flag")
			}
			if cmd.Flags().Lookup("key") == nil {
				t.Error("task update command missing --key flag")
			}

			break
		}
	}

	if !found {
		t.Fatal("task update command not found in task subcommands")
	}
}

// TestTaskCreateCommand_KeyFlag tests that the task create command has a --key flag
func TestTaskCreateCommand_KeyFlag(t *testing.T) {
	// Verify the task create command is registered
	var found bool
	for _, cmd := range taskCmd.Commands() {
		if cmd.Use == "create <title> [flags]" {
			found = true

			// Verify it has the --key flag
			if cmd.Flags().Lookup("key") == nil {
				t.Error("task create command missing --key flag")
			}

			break
		}
	}

	if !found {
		t.Fatal("task create command not found in task subcommands")
	}
}
