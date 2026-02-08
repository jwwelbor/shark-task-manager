package commands

import (
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
)

// TestAliasCommandsRegistered verifies that all alias commands are registered with RootCmd
func TestAliasCommandsRegistered(t *testing.T) {
	aliasNames := []string{"next", "start", "done", "block", "unblock"}
	for _, name := range aliasNames {
		t.Run(name, func(t *testing.T) {
			cmd, _, err := cli.RootCmd.Find([]string{name})
			if err != nil {
				t.Errorf("alias %q not found: %v", name, err)
				return
			}
			if cmd == nil || cmd.Name() != name {
				t.Errorf("expected command %q, got %v", name, cmd)
			}
		})
	}
}

// TestAliasNextHasExpectedFlags verifies the next alias has the correct flags
func TestAliasNextHasExpectedFlags(t *testing.T) {
	cmd, _, err := cli.RootCmd.Find([]string{"next"})
	if err != nil {
		t.Fatalf("next command not found: %v", err)
	}
	if cmd == nil {
		t.Fatal("next command is nil")
	}

	expectedFlags := []string{"agent", "epic"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("next command missing expected flag: %s", flag)
		}
	}
}

// TestAliasStartHasExpectedFlags verifies the start alias has the correct flags
func TestAliasStartHasExpectedFlags(t *testing.T) {
	cmd, _, err := cli.RootCmd.Find([]string{"start"})
	if err != nil {
		t.Fatalf("start command not found: %v", err)
	}
	if cmd == nil {
		t.Fatal("start command is nil")
	}

	expectedFlags := []string{"agent", "force"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("start command missing expected flag: %s", flag)
		}
	}
}

// TestAliasDoneHasExpectedFlags verifies the done alias has all expected flags
func TestAliasDoneHasExpectedFlags(t *testing.T) {
	cmd, _, err := cli.RootCmd.Find([]string{"done"})
	if err != nil {
		t.Fatalf("done command not found: %v", err)
	}
	if cmd == nil {
		t.Fatal("done command is nil")
	}

	// Main flags
	expectedFlags := []string{"agent", "notes", "force"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("done command missing expected flag: %s", flag)
		}
	}

	// Completion metadata flags
	metadataFlags := []string{"files-created", "files-modified", "tests", "summary", "verified", "agent-id", "time-spent"}
	for _, flag := range metadataFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("done command missing metadata flag: %s", flag)
		}
	}
}

// TestAliasBlockHasExpectedFlags verifies the block alias has the correct flags
func TestAliasBlockHasExpectedFlags(t *testing.T) {
	cmd, _, err := cli.RootCmd.Find([]string{"block"})
	if err != nil {
		t.Fatalf("block command not found: %v", err)
	}
	if cmd == nil {
		t.Fatal("block command is nil")
	}

	expectedFlags := []string{"reason", "agent", "force"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("block command missing expected flag: %s", flag)
		}
	}
}

// TestAliasUnblockHasExpectedFlags verifies the unblock alias has the correct flags
func TestAliasUnblockHasExpectedFlags(t *testing.T) {
	cmd, _, err := cli.RootCmd.Find([]string{"unblock"})
	if err != nil {
		t.Fatalf("unblock command not found: %v", err)
	}
	if cmd == nil {
		t.Fatal("unblock command is nil")
	}

	expectedFlags := []string{"agent", "force"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("unblock command missing expected flag: %s", flag)
		}
	}
}

// TestAliasCommandsHaveCorrectGroupID verifies all aliases are in the essentials group
func TestAliasCommandsHaveCorrectGroupID(t *testing.T) {
	aliasNames := []string{"next", "start", "done", "block", "unblock"}
	for _, name := range aliasNames {
		t.Run(name, func(t *testing.T) {
			cmd, _, err := cli.RootCmd.Find([]string{name})
			if err != nil {
				t.Fatalf("%s command not found: %v", name, err)
			}
			if cmd == nil {
				t.Fatalf("%s command is nil", name)
			}

			if cmd.GroupID != "quick" {
				t.Errorf("%s command has GroupID %q, expected 'quick'", name, cmd.GroupID)
			}
		})
	}
}

// TestAliasCommandsHaveCorrectArgsValidation verifies aliases have correct argument requirements
func TestAliasCommandsHaveCorrectArgsValidation(t *testing.T) {
	tests := []struct {
		name     string
		cmdName  string
		wantArgs int // 0 = no args required, 1 = exactly 1 arg required
	}{
		{"next accepts no args", "next", 0},
		{"start requires 1 arg", "start", 1},
		{"done requires 1 arg", "done", 1},
		{"block requires 1 arg", "block", 1},
		{"unblock requires 1 arg", "unblock", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, _, err := cli.RootCmd.Find([]string{tt.cmdName})
			if err != nil {
				t.Fatalf("%s command not found: %v", tt.cmdName, err)
			}
			if cmd == nil {
				t.Fatalf("%s command is nil", tt.cmdName)
			}

			// Verify Args validator exists for commands that require args
			if tt.wantArgs > 0 && cmd.Args == nil {
				t.Errorf("%s command should have Args validator", tt.cmdName)
			}
		})
	}
}

// TestUnblockAliasHelpTextUsesConfigDrivenDefaultStatus verifies that the unblock
// alias command's Long description uses the workflow config's default status
// instead of a hardcoded "todo" string.
func TestUnblockAliasHelpTextUsesConfigDrivenDefaultStatus(t *testing.T) {
	cmd, _, err := cli.RootCmd.Find([]string{"unblock"})
	if err != nil {
		t.Fatalf("unblock command not found: %v", err)
	}
	if cmd == nil {
		t.Fatal("unblock command is nil")
	}

	// The Long description should contain the default status from workflow config
	defaultStatus := cli.GetWorkflowService().GetDefaultStatus()
	if !strings.Contains(cmd.Long, defaultStatus) {
		t.Errorf("unblock Long description should contain default status %q, got: %s", defaultStatus, cmd.Long)
	}

	// Verify it does NOT contain a hardcoded "todo" that isn't the actual default
	// (This test passes trivially when default IS "todo", but catches regressions
	// if someone hardcodes a different status)
	if defaultStatus != "todo" && strings.Contains(cmd.Long, "to todo status") {
		t.Errorf("unblock Long description contains hardcoded 'todo' but default status is %q", defaultStatus)
	}
}

// TestTaskListStatusFlagIsConfigDriven verifies that the task list command's
// --status flag description is generated from the workflow config.
func TestTaskListStatusFlagIsConfigDriven(t *testing.T) {
	cmd, _, err := cli.RootCmd.Find([]string{"task", "list"})
	if err != nil {
		t.Fatalf("task list command not found: %v", err)
	}
	if cmd == nil {
		t.Fatal("task list command is nil")
	}

	statusFlag := cmd.Flags().Lookup("status")
	if statusFlag == nil {
		t.Fatal("task list command missing --status flag")
	}

	// The flag description should start with "Status filter ("
	if !strings.HasPrefix(statusFlag.Usage, "Status filter (") {
		t.Errorf("status flag usage should start with 'Status filter (', got: %s", statusFlag.Usage)
	}

	// It should contain at least some statuses from the workflow config
	allStatuses := cli.GetWorkflowService().GetAllStatusesOrdered()
	for _, status := range allStatuses {
		if !strings.Contains(statusFlag.Usage, status) {
			t.Errorf("status flag usage should contain workflow status %q, got: %s", status, statusFlag.Usage)
		}
	}

	// It should NOT be the old hardcoded string
	if statusFlag.Usage == "Filter by status (todo, in_progress, completed, blocked)" {
		t.Error("status flag usage still contains hardcoded status list instead of config-driven values")
	}
}

// TestTaskUnblockHelpTextUsesConfigDrivenDefaultStatus verifies that the task unblock
// command's Long description uses the workflow config's default status.
func TestTaskUnblockHelpTextUsesConfigDrivenDefaultStatus(t *testing.T) {
	cmd, _, err := cli.RootCmd.Find([]string{"task", "unblock"})
	if err != nil {
		t.Fatalf("task unblock command not found: %v", err)
	}
	if cmd == nil {
		t.Fatal("task unblock command is nil")
	}

	// The Long description should contain the default status from workflow config
	defaultStatus := cli.GetWorkflowService().GetDefaultStatus()
	if !strings.Contains(cmd.Long, defaultStatus) {
		t.Errorf("task unblock Long description should contain default status %q, got: %s", defaultStatus, cmd.Long)
	}
}
