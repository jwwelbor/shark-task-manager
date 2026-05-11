package commands

import (
	"testing"

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
