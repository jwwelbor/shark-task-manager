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
