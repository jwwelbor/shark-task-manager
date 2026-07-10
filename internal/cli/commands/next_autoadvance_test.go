package commands

import (
	"os"
	"path/filepath"
	"testing"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// terminalPassWorkflowConfig defines a route-based task workflow whose
// auto-advance placeholder ("finalize") routes pass to a TERMINAL step and
// fail to a forward, non-terminal step. The pre-selector transition scan
// skipped the terminal pass target and returned the fail target ("rework") —
// the positional-selection defect this test pins.
const terminalPassWorkflowConfig = `{
	"task_workflow": {
		"start": "rework",
		"steps": {
			"rework": {
				"phase": "development",
				"outcomes": {"pass": "finalize", "fail": "rework", "blocked": "hold"}
			},
			"finalize": {
				"phase": "approval",
				"action": "advance_status",
				"outcomes": {"pass": "done", "fail": "rework", "blocked": "hold"}
			},
			"hold": {"phase": "blocked", "parking": true, "action": "pause"},
			"done": {"phase": "done", "terminal": true, "action": "archive"}
		}
	}
}`

// TestPickAutoAdvanceTarget_TerminalPassTargetWins pins the outcome-key
// selection: auto-advance follows the step's declared pass outcome even when
// that target is terminal, instead of scanning past it to the fail target.
//
// NOTE: Must run serially (no t.Parallel()). It os.Chdir()s into a temp
// project and resets the global workflow-service singleton.
func TestPickAutoAdvanceTarget_TerminalPassTargetWins(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, ".sharkconfig.json"), []byte(terminalPassWorkflowConfig), 0644); err != nil {
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

	// Transitions as production produces them (pass-first ordering); the old
	// scan skipped "done" (terminal) and "hold" (parking) and picked "rework".
	info := &services.NextStatusInfo{
		EntityType:    "task",
		CurrentStatus: "finalize",
		AvailableTransitions: []services.TransitionInfoWithAction{
			{TransitionInfo: workflow.TransitionInfo{TargetStatus: "done"}},
			{TransitionInfo: workflow.TransitionInfo{TargetStatus: "rework"}},
			{TransitionInfo: workflow.TransitionInfo{TargetStatus: "hold"}},
		},
	}

	if got := pickAutoAdvanceTarget(info); got != "done" {
		t.Errorf("expected auto-advance to the terminal pass target %q, got %q", "done", got)
	}
}

// TestPickAutoAdvanceTarget_PassSelfLoopPauses: a pass outcome that loops back
// to the current status declares no forward motion — auto-advance must pause
// ("") rather than fall through to another outcome's target.
func TestPickAutoAdvanceTarget_PassSelfLoopPauses(t *testing.T) {
	selfLoopConfig := `{
	"task_workflow": {
		"start": "spin",
		"steps": {
			"spin": {
				"phase": "development",
				"action": "advance_status",
				"outcomes": {"pass": "spin", "fail": "rework", "blocked": "spin"}
			},
			"rework": {
				"phase": "development",
				"outcomes": {"pass": "done", "fail": "rework", "blocked": "rework"}
			},
			"done": {"phase": "done", "terminal": true}
		}
	}
}`
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, ".sharkconfig.json"), []byte(selfLoopConfig), 0644); err != nil {
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

	info := &services.NextStatusInfo{
		EntityType:    "task",
		CurrentStatus: "spin",
		AvailableTransitions: []services.TransitionInfoWithAction{
			{TransitionInfo: workflow.TransitionInfo{TargetStatus: "spin"}},
			{TransitionInfo: workflow.TransitionInfo{TargetStatus: "rework"}},
		},
	}

	if got := pickAutoAdvanceTarget(info); got != "" {
		t.Errorf("expected pause (empty) for a pass self-loop, got %q", got)
	}
}
