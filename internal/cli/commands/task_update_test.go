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
		// Updated to match new positional argument syntax
		if cmd.Use == "create [EPIC] [FEATURE] <title> [flags]" {
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

// TestTaskUpdateCommand_OrderFlag tests that the task update command has an --order flag
func TestTaskUpdateCommand_OrderFlag(t *testing.T) {
	// Verify the task update command is registered
	var found bool
	for _, cmd := range taskCmd.Commands() {
		if cmd.Use == "update <task-key>" {
			found = true

			// Verify it has the --order flag
			if cmd.Flags().Lookup("order") == nil {
				t.Error("task update command missing --order flag")
			}

			break
		}
	}

	if !found {
		t.Fatal("task update command not found in task subcommands")
	}
}

// TestTaskCreateCommand_OrderFlag tests that the task create command has an --order flag
func TestTaskCreateCommand_OrderFlag(t *testing.T) {
	// Verify the task create command is registered
	var found bool
	for _, cmd := range taskCmd.Commands() {
		// Updated to match new positional argument syntax
		if cmd.Use == "create [EPIC] [FEATURE] <title> [flags]" {
			found = true

			// Verify it has the --order flag
			if cmd.Flags().Lookup("order") == nil {
				t.Error("task create command missing --order flag")
			}

			break
		}
	}

	if !found {
		t.Fatal("task create command not found in task subcommands")
	}
}

// TestTaskUpdateCommand_StatusFlag tests that the task update command has a --status flag
func TestTaskUpdateCommand_StatusFlag(t *testing.T) {
	// Verify the task update command is registered
	var found bool
	for _, cmd := range taskCmd.Commands() {
		if cmd.Use == "update <task-key>" {
			found = true

			// Verify it has the --status flag
			if cmd.Flags().Lookup("status") == nil {
				t.Error("task update command missing --status flag")
			}

			break
		}
	}

	if !found {
		t.Fatal("task update command not found in task subcommands")
	}
}

// TestTaskReopenCommand_RejectionReasonFlag tests that the task reopen command has a --rejection-reason flag
func TestTaskReopenCommand_RejectionReasonFlag(t *testing.T) {
	// Verify the task reopen command is registered
	var found bool
	for _, cmd := range taskCmd.Commands() {
		if cmd.Use == "reopen <task-key>" {
			found = true

			// Verify it has the --rejection-reason flag
			if cmd.Flags().Lookup("rejection-reason") == nil {
				t.Error("task reopen command missing --rejection-reason flag")
			}

			break
		}
	}

	if !found {
		t.Fatal("task reopen command not found in task subcommands")
	}
}

// TestTaskApproveCommand_RejectionReasonFlag tests that the task approve command has a --rejection-reason flag
func TestTaskApproveCommand_RejectionReasonFlag(t *testing.T) {
	// Verify the task approve command is registered
	var found bool
	for _, cmd := range taskCmd.Commands() {
		if cmd.Use == "approve <task-key>" {
			found = true

			// Verify it has the --rejection-reason flag
			if cmd.Flags().Lookup("rejection-reason") == nil {
				t.Error("task approve command missing --rejection-reason flag")
			}

			break
		}
	}

	if !found {
		t.Fatal("task approve command not found in task subcommands")
	}
}

// TestTaskUpdateCommand_ReasonFlag tests that the task update command has a --reason flag
func TestTaskUpdateCommand_ReasonFlag(t *testing.T) {
	// Verify the task update command is registered
	var found bool
	for _, cmd := range taskCmd.Commands() {
		if cmd.Use == "update <task-key>" {
			found = true

			// Verify it has the --reason flag
			if cmd.Flags().Lookup("reason") == nil {
				t.Error("task update command missing --reason flag")
			}

			break
		}
	}

	if !found {
		t.Fatal("task update command not found in task subcommands")
	}
}

