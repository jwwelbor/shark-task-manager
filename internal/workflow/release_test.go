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
