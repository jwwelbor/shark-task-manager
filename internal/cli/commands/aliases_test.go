package commands

import (
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
)

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
