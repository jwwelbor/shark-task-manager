package commands

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// TestNormalizeWireAction_TableDriven covers every row of the verb-mapping
// table in next.go. Adding a new internal verb without updating this table
// will leave it routed through the "error" branch — which is the loudest
// possible signal to the next reviewer.
func TestNormalizeWireAction_TableDriven(t *testing.T) {
	// A nextInfo with one productive forward transition (in_development)
	// and several non-productive siblings, so the auto-advance picker
	// produces "in_development" deterministically.
	productiveNextInfo := &services.NextStatusInfo{
		AvailableTransitions: []services.TransitionInfoWithAction{
			{TransitionInfo: workflow.TransitionInfo{TargetStatus: "in_development"}},
			{TransitionInfo: workflow.TransitionInfo{TargetStatus: "blocked"}},
			{TransitionInfo: workflow.TransitionInfo{TargetStatus: "on_hold"}},
		},
	}

	// A nextInfo whose only available transitions are non-productive —
	// auto-advance has nothing safe to pick, so we expect autoAdvanceTarget="".
	stuckNextInfo := &services.NextStatusInfo{
		AvailableTransitions: []services.TransitionInfoWithAction{
			{TransitionInfo: workflow.TransitionInfo{TargetStatus: "blocked"}},
			{TransitionInfo: workflow.TransitionInfo{TargetStatus: "on_hold"}},
		},
	}

	tests := []struct {
		name           string
		internalAction string
		agentType      string
		nextInfo       *services.NextStatusInfo
		wantWire       string
		wantTarget     string
	}{
		{"spawn_agent passes through", "spawn_agent", "developer", productiveNextInfo, "spawn_agent", ""},
		{"check_or_resume becomes spawn_agent", "check_or_resume", "developer", productiveNextInfo, "spawn_agent", ""},
		{"advance_status with agent becomes spawn_agent", "advance_status", "developer", productiveNextInfo, "spawn_agent", ""},
		{"advance_status without agent triggers auto-advance", "advance_status", "", productiveNextInfo, "advance_and_recurse", "in_development"},
		{"advance_status with no safe target", "advance_status", "", stuckNextInfo, "advance_and_recurse", ""},
		{"pause stays pause", "pause", "", productiveNextInfo, "pause", ""},
		{"archive stays archive", "archive", "", productiveNextInfo, "archive", ""},
		{"empty verb with agent_type becomes spawn_agent", "", "developer", productiveNextInfo, "spawn_agent", ""},
		{"empty verb without agent_type pauses", "", "", productiveNextInfo, "pause", ""},
		{"unknown verb returns error sentinel", "frobnicate", "developer", productiveNextInfo, "error", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wire, target := normalizeWireAction(tt.internalAction, tt.agentType, tt.nextInfo)
			if wire != tt.wantWire {
				t.Errorf("wireAction=%q want %q", wire, tt.wantWire)
			}
			if target != tt.wantTarget {
				t.Errorf("autoAdvanceTarget=%q want %q", target, tt.wantTarget)
			}
		})
	}
}

func TestPickAutoAdvanceTarget_PrefersFirstProductive(t *testing.T) {
	info := &services.NextStatusInfo{
		AvailableTransitions: []services.TransitionInfoWithAction{
			// Authoring may list these in workflow-natural order.
			{TransitionInfo: workflow.TransitionInfo{TargetStatus: "ready_for_assessment"}},
			{TransitionInfo: workflow.TransitionInfo{TargetStatus: "cancelled"}},
			{TransitionInfo: workflow.TransitionInfo{TargetStatus: "on_hold"}},
		},
	}
	if got := pickAutoAdvanceTarget(info); got != "ready_for_assessment" {
		t.Errorf("expected ready_for_assessment, got %q", got)
	}
}

func TestPickAutoAdvanceTarget_NilSafe(t *testing.T) {
	if got := pickAutoAdvanceTarget(nil); got != "" {
		t.Errorf("nil nextInfo should return empty, got %q", got)
	}
}

// ─── B022 regression: legacy status graceful degradation ─────────────────────

// TestIsStatusNotFoundError_DetectsStatusNotFoundError verifies that the helper
// correctly identifies action.StatusNotFoundError so resolveNext can apply
// graceful degradation (pause) instead of propagating the error.
//
// Bug: B022 — "shark next exits 1 on legacy task statuses (in_approval,
// ready_for_approval) instead of degrading". When a task's current status is
// not defined in the workflow YAML, GetStatusActionPopulated returns a
// StatusNotFoundError; the fix converts that to a pause action.
func TestIsStatusNotFoundError_DetectsStatusNotFoundError(t *testing.T) {
	snfe := &action.StatusNotFoundError{Status: "in_approval"}

	if !isStatusNotFoundError(snfe) {
		t.Error("isStatusNotFoundError should return true for *action.StatusNotFoundError")
	}
}

func TestIsStatusNotFoundError_IgnoresOtherErrors(t *testing.T) {
	otherErr := errors.New("some other error")

	if isStatusNotFoundError(otherErr) {
		t.Error("isStatusNotFoundError should return false for a non-StatusNotFoundError")
	}
}

func TestIsStatusNotFoundError_NilReturnsFalse(t *testing.T) {
	if isStatusNotFoundError(nil) {
		t.Error("isStatusNotFoundError should return false for nil")
	}
}

func TestIsStatusNotFoundError_WrappedErrorDetected(t *testing.T) {
	// Wrapped StatusNotFoundError (e.g., from fmt.Errorf("...: %w", snfe))
	// should also be detected, since errors.As unwraps.
	snfe := &action.StatusNotFoundError{Status: "ready_for_approval"}
	wrapped := fmt.Errorf("failed to populate action for status %q: %w", "ready_for_approval", snfe)

	if !isStatusNotFoundError(wrapped) {
		t.Error("isStatusNotFoundError should return true for a wrapped *action.StatusNotFoundError")
	}
}
