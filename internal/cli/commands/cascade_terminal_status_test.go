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

// TestB028_IsTerminalStatus_DelegatesToWorkflowService is the regression
// test for B028. cascade.go's isTerminalStatus must consult
// workflow.Service.IsTerminalStatus rather than a hardcoded literal list,
// so a custom workflow that renames "completed" to "shipped" still
// classifies children correctly during cascade resolution.
//
// Before the B028 fix this helper was:
//
//	switch s { case "completed", "cancelled", "archived", "done": return true }
//
// …which silently misclassified "shipped" as non-terminal and recursed
// into entities that should have been filtered out.
func TestB028_IsTerminalStatus_DelegatesToWorkflowService(t *testing.T) {
	tmp := writeB028Config(t)
	taskWf := workflow.NewService(tmp).ForLevel(workflow.LevelTask)

	if !isTerminalStatus(taskWf, "shipped") {
		t.Error("expected isTerminalStatus(\"shipped\") = true for custom workflow with renamed terminal; cascade.go regressed to hardcoded list")
	}
	if isTerminalStatus(taskWf, "in_progress") {
		t.Error("expected isTerminalStatus(\"in_progress\") = false")
	}
	// "completed" is NOT a terminal for this workflow; the pre-fix
	// hardcoded list would have wrongly returned true.
	if isTerminalStatus(taskWf, "completed") {
		t.Error("expected isTerminalStatus(\"completed\") = false; cascade.go is still consulting a hardcoded list")
	}

	featureWf := workflow.NewService(tmp).ForLevel(workflow.LevelFeature)
	if !isTerminalStatus(featureWf, "shipped") {
		t.Error("expected feature-level isTerminalStatus(\"shipped\") = true")
	}
}

// TestB028_IsTerminalStatus_NilWorkflowSafe guards against panics if the
// helper is ever called with a nil workflow.
func TestB028_IsTerminalStatus_NilWorkflowSafe(t *testing.T) {
	if isTerminalStatus(nil, "anything") {
		t.Error("expected isTerminalStatus with nil wf to return false")
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
