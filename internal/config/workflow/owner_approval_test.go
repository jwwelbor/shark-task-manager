package workflow

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
)

func TestNormalizeOwnerApprovalLevels(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    []string
		wantErr bool
	}{
		{name: "true gates all known levels", input: true, want: KnownLevels},
		{name: "false disables", input: false, want: nil},
		{name: "nil disables", input: nil, want: nil},
		{name: "single string", input: "feature", want: []string{"feature"}},
		{name: "string list", input: []interface{}{"feature", "task"}, want: []string{"feature", "task"}},
		{name: "alias normalized", input: []interface{}{"tech-debt"}, want: []string{"tech_debt"}},
		{name: "unknown level errors", input: []interface{}{"gizmo"}, wantErr: true},
		{name: "non-string entry errors", input: []interface{}{42}, wantErr: true},
		{name: "number errors", input: 3.14, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeOwnerApprovalLevels(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInjectOwnerApprovalGate_DefaultFeatureWorkflow(t *testing.T) {
	wf := DefaultFeatureWorkflow()
	if err := wf.InjectOwnerApprovalGate(); err != nil {
		t.Fatalf("InjectOwnerApprovalGate: %v", err)
	}

	terminal, err := wf.ArchiveTerminalStatus()
	if err != nil {
		t.Fatalf("ArchiveTerminalStatus: %v", err)
	}

	gate, ok := wf.GetStep(OwnerApprovalStepName)
	if !ok {
		t.Fatalf("owner_approval step not injected")
	}
	if gate.Action != action.ActionPause {
		t.Errorf("gate action = %q, want %q", gate.Action, action.ActionPause)
	}
	if gate.Responsibility != "human" {
		t.Errorf("gate responsibility = %q, want human", gate.Responsibility)
	}
	if got := gate.Outcomes[OutcomePass]; got != terminal {
		t.Errorf("gate pass -> %q, want %q", got, terminal)
	}
	if gate.Outcomes[OutcomeFail] == "" || gate.Outcomes[OutcomeFail] == OwnerApprovalStepName {
		t.Errorf("gate fail -> %q, want a rework step", gate.Outcomes[OutcomeFail])
	}
	if gate.Outcomes[OutcomeBlocked] == "" || gate.Outcomes[OutcomeBlocked] == OwnerApprovalStepName {
		t.Errorf("gate blocked -> %q, want a parking/rework step", gate.Outcomes[OutcomeBlocked])
	}

	// No workable step may route directly to the primary terminal anymore;
	// everything must pass through the injected gate.
	for name, st := range wf.Steps {
		if st == nil || st.Terminal || st.Parking || name == OwnerApprovalStepName {
			continue
		}
		for outcome, target := range st.Outcomes {
			if target == terminal {
				t.Errorf("step %q outcome %q still routes directly to terminal %q", name, outcome, terminal)
			}
		}
	}

	// The derived legacy view must know the injected step and its transitions.
	if _, ok := wf.StatusMetadata[OwnerApprovalStepName]; !ok {
		t.Errorf("StatusMetadata missing %s after DeriveLegacy", OwnerApprovalStepName)
	}
	foundTerminal := false
	for _, target := range wf.StatusFlow[OwnerApprovalStepName] {
		if target == terminal {
			foundTerminal = true
		}
	}
	if !foundTerminal {
		t.Errorf("StatusFlow[%s] = %v, missing terminal %q", OwnerApprovalStepName, wf.StatusFlow[OwnerApprovalStepName], terminal)
	}

	// The gated workflow must still satisfy core-outcome validation.
	if errs := wf.ValidateCoreOutcomes(); len(errs) > 0 {
		t.Errorf("ValidateCoreOutcomes after injection: %v", errs)
	}

	// Idempotent: a second injection is a no-op.
	before := gate.Outcomes[OutcomePass]
	if err := wf.InjectOwnerApprovalGate(); err != nil {
		t.Fatalf("second InjectOwnerApprovalGate: %v", err)
	}
	gate2, _ := wf.GetStep(OwnerApprovalStepName)
	if gate2.Outcomes[OutcomePass] != before {
		t.Errorf("second injection changed gate routing")
	}
}

func TestInjectOwnerApprovalGate_AllDefaultWorkflows(t *testing.T) {
	// `require_owner_approval: true` gates every known level; each embedded
	// default must inject without error.
	for _, level := range KnownLevels {
		wf := defaultForType(level)
		if err := wf.InjectOwnerApprovalGate(); err != nil {
			t.Errorf("level %s: InjectOwnerApprovalGate: %v", level, err)
			continue
		}
		if errs := wf.ValidateCoreOutcomes(); len(errs) > 0 {
			t.Errorf("level %s: ValidateCoreOutcomes after injection: %v", level, errs)
		}
	}
}

func TestLoadMultiLevelWorkflow_RequireOwnerApproval(t *testing.T) {
	configPath := func(t *testing.T) string {
		return filepath.Join(t.TempDir(), ".sharkconfig.json")
	}

	t.Run("list gates only the listed level", func(t *testing.T) {
		ResetMultiLevelCache()
		defer ResetMultiLevelCache()
		data := []byte(`{"require_owner_approval": ["feature"]}`)
		mlw, err := LoadMultiLevelWorkflowFromBytes(configPath(t), data)
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		if mlw.Feature == nil {
			t.Fatalf("feature slot not materialized for gating")
		}
		if _, ok := mlw.Feature.GetStep(OwnerApprovalStepName); !ok {
			t.Errorf("feature workflow missing injected %s step", OwnerApprovalStepName)
		}
		// Task was not listed: slot stays nil, default stays ungated.
		if mlw.Task != nil {
			if _, ok := mlw.Task.GetStep(OwnerApprovalStepName); ok {
				t.Errorf("task workflow unexpectedly gated")
			}
		}
		if _, ok := mlw.GetWorkflowForLevel("task").GetStep(OwnerApprovalStepName); ok {
			t.Errorf("task default workflow unexpectedly gated")
		}
	})

	t.Run("true gates all levels", func(t *testing.T) {
		ResetMultiLevelCache()
		defer ResetMultiLevelCache()
		data := []byte(`{"require_owner_approval": true}`)
		mlw, err := LoadMultiLevelWorkflowFromBytes(configPath(t), data)
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		for _, level := range KnownLevels {
			wf := mlw.GetWorkflowForLevel(level)
			if _, ok := wf.GetStep(OwnerApprovalStepName); !ok {
				t.Errorf("level %s missing injected %s step", level, OwnerApprovalStepName)
			}
		}
	})

	t.Run("false is a no-op", func(t *testing.T) {
		ResetMultiLevelCache()
		defer ResetMultiLevelCache()
		data := []byte(`{"require_owner_approval": false}`)
		mlw, err := LoadMultiLevelWorkflowFromBytes(configPath(t), data)
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		if mlw.Feature != nil {
			t.Errorf("feature slot materialized despite disabled flag")
		}
	})

	t.Run("unknown level fails the load", func(t *testing.T) {
		ResetMultiLevelCache()
		defer ResetMultiLevelCache()
		data := []byte(`{"require_owner_approval": ["gizmo"]}`)
		if _, err := LoadMultiLevelWorkflowFromBytes(configPath(t), data); err == nil {
			t.Fatalf("expected error for unknown entity type")
		}
	})

	t.Run("existing owner_approval step is respected", func(t *testing.T) {
		ResetMultiLevelCache()
		defer ResetMultiLevelCache()
		// Inline route-based feature workflow that already defines its own
		// owner_approval step routing pass -> done directly from work.
		data := []byte(`{
			"require_owner_approval": ["feature"],
			"feature_workflow": {
				"version": "1.0",
				"start": "work",
				"steps": {
					"work": {
						"action": "spawn_agent",
						"agent": "dev",
						"outcomes": {"pass": "done", "fail": "work", "blocked": "work"}
					},
					"owner_approval": {
						"action": "pause",
						"responsibility": "human",
						"outcomes": {"pass": "done", "fail": "work", "blocked": "work"}
					},
					"done": {"terminal": true, "primary": true, "action": "archive"}
				}
			}
		}`)
		mlw, err := LoadMultiLevelWorkflowFromBytes(configPath(t), data)
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		work, ok := mlw.Feature.GetStep("work")
		if !ok {
			t.Fatalf("work step missing")
		}
		if got := work.Outcomes["pass"]; got != "done" {
			t.Errorf("work pass -> %q; user-defined owner_approval must leave routing untouched", got)
		}
	})
}
