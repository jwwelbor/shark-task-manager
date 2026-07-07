package commands

import (
	"os"
	"path/filepath"
	"testing"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// b028CustomWorkflowConfig is a minimal multi-level workflow whose task /
// feature / epic terminal status is renamed to "shipped" (not in the
// pre-B028 hardcoded list of {completed, cancelled, archived, done}).
const b028CustomWorkflowConfig = `{
  "task_workflow": {
    "statuses": ["todo", "in_progress", "shipped"],
    "status_flow": {
      "todo": ["in_progress"],
      "in_progress": ["shipped"],
      "shipped": []
    },
    "special_statuses": {
      "_start_": ["todo"],
      "_complete_": ["shipped"]
    },
    "status_metadata": {
      "todo": {"color": "gray", "phase": "planning"},
      "in_progress": {"color": "blue", "phase": "development"},
      "shipped": {"color": "green", "phase": "done"}
    }
  },
  "feature_workflow": {
    "statuses": ["draft", "active", "shipped"],
    "status_flow": {
      "draft": ["active"],
      "active": ["shipped"],
      "shipped": []
    },
    "special_statuses": {
      "_start_": ["draft"],
      "_complete_": ["shipped"]
    },
    "status_metadata": {
      "draft": {"color": "gray", "phase": "planning"},
      "active": {"color": "blue", "phase": "development"},
      "shipped": {"color": "green", "phase": "done"}
    }
  },
  "epic_workflow": {
    "statuses": ["draft", "active", "shipped"],
    "status_flow": {
      "draft": ["active"],
      "active": ["shipped"],
      "shipped": []
    },
    "special_statuses": {
      "_start_": ["draft"],
      "_complete_": ["shipped"]
    },
    "status_metadata": {
      "draft": {"color": "gray", "phase": "planning"},
      "active": {"color": "blue", "phase": "development"},
      "shipped": {"color": "green", "phase": "done"}
    }
  }
}`

func writeB028Config(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, ".sharkconfig.json"), []byte(b028CustomWorkflowConfig), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	return tmp
}

// TestB028_TerminalStatusDelegation pins the workflow-level terminal-status
// contract that cascade resolution depends on (see B028). A custom workflow
// that renames "completed" to "shipped" must still classify children
// correctly during cascade dispatch.
func TestB028_TerminalStatusDelegation(t *testing.T) {
	tmp := writeB028Config(t)
	taskWf := workflow.NewService(tmp).ForLevel(workflow.LevelTask)

	if !taskWf.IsTerminalStatus("shipped") {
		t.Error("expected IsTerminalStatus(\"shipped\") = true for custom workflow with renamed terminal")
	}
	if taskWf.IsTerminalStatus("in_progress") {
		t.Error("expected IsTerminalStatus(\"in_progress\") = false")
	}
	// "completed" is NOT a terminal for this workflow; the pre-B028
	// hardcoded list would have wrongly returned true.
	if taskWf.IsTerminalStatus("completed") {
		t.Error("expected IsTerminalStatus(\"completed\") = false; workflow.Service is consulting a hardcoded list")
	}

	featureWf := workflow.NewService(tmp).ForLevel(workflow.LevelFeature)
	if !featureWf.IsTerminalStatus("shipped") {
		t.Error("expected feature-level IsTerminalStatus(\"shipped\") = true")
	}
}

// TestB028_IsArchivedStatus_DelegatesToWorkflowService verifies next.go's
// isArchivedStatus consults workflow.Service for the canonical terminal
// set rather than a hardcoded literal list, while still preserving the
// loose `_archived` suffix match.
// NOTE: This test MUST run serially (no t.Parallel()). It os.Chdir()s into a
// temp project and mutates the global workflow-service singleton via
// cli.ResetWorkflowService(); running it concurrently with any other test that
// depends on the current working directory or the global workflow service would
// race. The whole package therefore relies on the default serial test execution.
func TestB028_IsArchivedStatus_DelegatesToWorkflowService(t *testing.T) {
	tmp := writeB028Config(t)

	// Point the global workflow service at our tempdir.
	origCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origCwd)
		cli.ResetWorkflowService()
	})
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	cli.ResetWorkflowService()

	// Pre-fix hardcoded list: {"archived", "completed", "cancelled", "done"}.
	// "shipped" was not in it; this assertion fails on the regressed code.
	if !isArchivedStatus("task", "shipped") {
		t.Error("expected isArchivedStatus(\"task\", \"shipped\") = true for custom workflow with renamed terminal; next.go regressed to hardcoded list")
	}

	// Suffix match for *_archived must still work (the bug-fix notes
	// explicitly preserve this).
	if !isArchivedStatus("task", "in_qa_archived") {
		t.Error("expected isArchivedStatus suffix match for *_archived to still work")
	}

	// Non-terminal status must remain non-archived.
	if isArchivedStatus("task", "in_progress") {
		t.Error("expected isArchivedStatus(\"task\", \"in_progress\") = false")
	}

	// Case-insensitive match (IsTerminalStatus uses strings.EqualFold).
	if !isArchivedStatus("task", "SHIPPED") {
		t.Error("expected case-insensitive terminal match")
	}

	// Feature-level terminal must also be recognized when entityType=feature.
	if !isArchivedStatus("feature", "shipped") {
		t.Error("expected feature-level terminal \"shipped\" to be classified as archived")
	}
}

// b034ChangeCardWorkflowConfig defines distinct terminal statuses for task
// ("completed") and change ("resolved") so a lookup that fails to normalize
// "change_card" -> "change" and silently falls back to the task workflow is
// observable: it would treat "completed" (task's terminal, not change's) as
// archived for a change-card and fail to recognize "resolved" (change's own
// terminal) at all.
const b034ChangeCardWorkflowConfig = `{
  "task_workflow": {
    "statuses": ["todo", "in_progress", "completed"],
    "status_flow": {
      "todo": ["in_progress"],
      "in_progress": ["completed"],
      "completed": []
    },
    "special_statuses": {
      "_start_": ["todo"],
      "_complete_": ["completed"]
    },
    "status_metadata": {
      "todo": {"color": "gray", "phase": "planning"},
      "in_progress": {"color": "blue", "phase": "development"},
      "completed": {"color": "green", "phase": "done"}
    }
  },
  "change_workflow": {
    "statuses": ["draft", "development", "resolved"],
    "status_flow": {
      "draft": ["development"],
      "development": ["resolved"],
      "resolved": []
    },
    "special_statuses": {
      "_start_": ["draft"],
      "_complete_": ["resolved"]
    },
    "status_metadata": {
      "draft": {"color": "gray", "phase": "planning"},
      "development": {"color": "blue", "phase": "development"},
      "resolved": {"color": "green", "phase": "done"}
    }
  }
}`

// TestB034_IsArchivedStatus_NormalizesChangeCardEntityType verifies
// isArchivedStatus normalizes "change_card" -> "change" before narrowing
// wf.ForLevel, mirroring the ActionService.ForEntity fix for B034. Without
// the fix, GetWorkflowForLevel("change_card") falls through defaultForType's
// default branch to the TASK workflow instead of the change workflow.
// NOTE: Must run serially (no t.Parallel()) — see
// TestB028_IsArchivedStatus_DelegatesToWorkflowService above for why.
func TestB034_IsArchivedStatus_NormalizesChangeCardEntityType(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, ".sharkconfig.json"), []byte(b034ChangeCardWorkflowConfig), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	origCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origCwd)
		cli.ResetWorkflowService()
	})
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	cli.ResetWorkflowService()

	// change's own terminal status. Pre-fix, entityType="change_card" resolves
	// to the task workflow (which has no "resolved" status at all), so this
	// assertion fails on the regressed code.
	if !isArchivedStatus("change_card", "resolved") {
		t.Error("expected isArchivedStatus(\"change_card\", \"resolved\") = true; entityType not normalized to \"change\" before ForLevel lookup")
	}

	// task's terminal, NOT change's. Pre-fix, entityType="change_card" falls
	// back to the task workflow, which *does* recognize "completed" as
	// terminal, so this assertion is true on the regressed code and false
	// once the fix correctly resolves the change workflow instead.
	if isArchivedStatus("change_card", "completed") {
		t.Error("expected isArchivedStatus(\"change_card\", \"completed\") = false; change workflow does not define this status as terminal")
	}

	// Non-terminal change status remains non-archived.
	if isArchivedStatus("change_card", "development") {
		t.Error("expected isArchivedStatus(\"change_card\", \"development\") = false")
	}
}
