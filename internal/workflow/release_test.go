package workflow

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

// routeBasedService builds a Service backed by a route-based (steps:) config
// directly, exercising the F02 release/outcome methods without file I/O.
func routeBasedService() *Service {
	cfg := &config.WorkflowConfig{
		Version: "1.0",
		Start:   "draft",
		Steps: map[string]*config.Step{
			"draft": {
				Phase:    "planning",
				Action:   "advance_status",
				Outcomes: map[string]string{"pass": "review"},
			},
			"review": {
				Phase:  "review",
				Action: "spawn_agent",
				Agent:  "reviewer",
				Skills: []string{"code-review"},
				Prompt: "task/review.md",
				Outcomes: map[string]string{
					"pass":    "done",
					"fail":    "draft",
					"blocked": "on_hold",
				},
			},
			"on_hold": {Phase: "paused", Parking: true},
			"done":    {Phase: "done", Terminal: true},
		},
	}
	// Release/GetOutcomes read the Steps map directly; the derived legacy maps
	// (populated by the loader in real use) are not needed for these methods.
	return &Service{workflow: cfg, level: LevelTask}
}

func TestService_IsRouteBased(t *testing.T) {
	if !routeBasedService().IsRouteBased() {
		t.Error("expected route-based service")
	}
	legacy := &Service{workflow: &config.WorkflowConfig{
		StatusFlow: map[string][]string{"todo": {"done"}},
	}, level: LevelTask}
	if legacy.IsRouteBased() {
		t.Error("legacy service should not be route-based")
	}
}

func TestService_Release(t *testing.T) {
	svc := routeBasedService()

	tests := []struct {
		from, outcome, want string
		wantErr             bool
	}{
		{"review", "pass", "done", false},
		{"review", "fail", "draft", false},
		{"review", "blocked", "on_hold", false},
		{"review", "PASS", "done", false}, // case-insensitive
		{"draft", "pass", "review", false},
		{"review", "dead-end", "", true}, // undefined outcome
		{"done", "pass", "", true},       // terminal: no outcomes
		{"on_hold", "pass", "", true},    // parking: no outcomes
	}
	for _, tt := range tests {
		got, err := svc.Release(tt.from, tt.outcome)
		if (err != nil) != tt.wantErr {
			t.Errorf("Release(%q,%q) err = %v, wantErr %v", tt.from, tt.outcome, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("Release(%q,%q) = %q, want %q", tt.from, tt.outcome, got, tt.want)
		}
	}
}

// aliasedRouteService builds a route-based config where the "review" step
// collapses the legacy "in_review"/"ready_for_review" names via aliases.
func aliasedRouteService() *Service {
	cfg := &config.WorkflowConfig{
		Version: "1.0",
		Start:   "draft",
		Steps: map[string]*config.Step{
			"draft": {
				Phase:    "planning",
				Action:   "advance_status",
				Outcomes: map[string]string{"pass": "review"},
			},
			"review": {
				Phase:   "review",
				Action:  "spawn_agent",
				Agent:   "reviewer",
				Aliases: []string{"in_review", "ready_for_review"},
				Outcomes: map[string]string{
					"pass":    "done",
					"fail":    "draft",
					"blocked": "on_hold",
				},
			},
			"on_hold": {Phase: "paused", Parking: true},
			"done":    {Phase: "done", Terminal: true},
		},
	}
	cfg.DeriveLegacy()
	return &Service{workflow: cfg, level: LevelTask}
}

// TestService_Release_ResolvesAlias verifies that an aliased (pre-migration)
// status routes its outcomes correctly: Release and GetOutcomes must resolve
// the old name to its new step before consulting the outcomes map (WS1-B).
func TestService_Release_ResolvesAlias(t *testing.T) {
	svc := aliasedRouteService()

	// Outcome routing from the old status name.
	got, err := svc.Release("in_review", "pass")
	if err != nil {
		t.Fatalf("Release(in_review, pass) err = %v", err)
	}
	if got != "done" {
		t.Errorf("Release(in_review, pass) = %q, want done", got)
	}
	if got, err := svc.Release("ready_for_review", "fail"); err != nil || got != "draft" {
		t.Errorf("Release(ready_for_review, fail) = %q, %v; want draft, nil", got, err)
	}

	// GetOutcomes for the old status name returns the new step's outcomes.
	outcomes := svc.GetOutcomes("in_review")
	if outcomes["pass"] != "done" || outcomes["blocked"] != "on_hold" {
		t.Errorf("GetOutcomes(in_review) = %v, want review step's outcomes", outcomes)
	}
}

func TestService_Release_LegacyError(t *testing.T) {
	legacy := &Service{workflow: &config.WorkflowConfig{
		StatusFlow: map[string][]string{"todo": {"in_progress"}},
	}, level: LevelTask}
	if _, err := legacy.Release("todo", "pass"); err == nil {
		t.Error("expected error releasing outcome on legacy workflow")
	}
}

func TestService_GetValidOutcomes(t *testing.T) {
	svc := routeBasedService()
	got := svc.GetValidOutcomes("review")
	want := []string{"blocked", "fail", "pass"} // sorted
	if len(got) != len(want) {
		t.Fatalf("GetValidOutcomes(review) = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("GetValidOutcomes(review)[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	if len(svc.GetValidOutcomes("done")) != 0 {
		t.Error("terminal step should have no outcomes")
	}
}
