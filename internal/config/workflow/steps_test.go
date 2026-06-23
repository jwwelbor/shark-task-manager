package workflow

import (
	"sort"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
)

// sampleStepsConfig returns a small route-based config exercising the four step
// kinds: auto (advance_status), agent (spawn_agent with outcomes), parking, and
// terminal.
func sampleStepsConfig() *WorkflowConfig {
	return &WorkflowConfig{
		Version: "1.0",
		Start:   "draft",
		Steps: map[string]*Step{
			"draft": {
				Phase:          "planning",
				ProgressWeight: 0.0,
				Action:         action.ActionAdvanceStatus,
				Prompt:         "epic/draft.md",
				Outcomes:       map[string]string{"pass": "qa"},
			},
			"qa": {
				Phase:          "qa",
				Color:          "cyan",
				ProgressWeight: 0.85,
				Responsibility: "agent",
				Action:         action.ActionSpawnAgent,
				Agent:          "qa",
				Provider:       "anthropic",
				Model:          "sonnet",
				Skills:         []string{"quality"},
				Prompt:         "feature/qa.md",
				Aliases:        []string{"ready_for_qa", "in_qa"},
				Outcomes: map[string]string{
					"pass":    "completed",
					"fail":    "draft",
					"blocked": "on_hold",
				},
			},
			"on_hold": {
				Phase:   "paused",
				Parking: true,
			},
			"completed": {
				Phase:          "done",
				ProgressWeight: 1.0,
				Terminal:       true,
			},
		},
	}
}

func TestDeriveLegacyFromSteps_StatusFlow(t *testing.T) {
	cfg := sampleStepsConfig()
	deriveLegacyFromSteps(cfg)

	// draft -> [qa]
	if got := cfg.StatusFlow["draft"]; len(got) != 1 || got[0] != "qa" {
		t.Errorf("draft transitions = %v, want [qa]", got)
	}

	// qa -> sorted unique outcome targets {completed, draft, on_hold}
	wantQA := []string{"completed", "draft", "on_hold"}
	gotQA := append([]string(nil), cfg.StatusFlow["qa"]...)
	sort.Strings(gotQA)
	if !equalSlice(gotQA, wantQA) {
		t.Errorf("qa transitions = %v, want %v", gotQA, wantQA)
	}

	// terminal step has no transitions
	if got := cfg.StatusFlow["completed"]; len(got) != 0 {
		t.Errorf("completed transitions = %v, want []", got)
	}

	// parking step exposes workable steps as resume targets (draft, qa)
	wantPark := []string{"draft", "qa"}
	gotPark := append([]string(nil), cfg.StatusFlow["on_hold"]...)
	sort.Strings(gotPark)
	if !equalSlice(gotPark, wantPark) {
		t.Errorf("on_hold transitions = %v, want %v", gotPark, wantPark)
	}
}

func TestDeriveLegacyFromSteps_Metadata(t *testing.T) {
	cfg := sampleStepsConfig()
	deriveLegacyFromSteps(cfg)

	meta, ok := cfg.GetStatusMetadata("qa")
	if !ok {
		t.Fatal("qa metadata not derived")
	}
	if meta.Color != "cyan" || meta.Phase != "qa" || meta.ProgressWeight != 0.85 {
		t.Errorf("qa metadata wrong: %+v", meta)
	}
	if meta.Responsibility != "agent" {
		t.Errorf("qa responsibility = %q, want agent", meta.Responsibility)
	}
	// agent should surface as an agent type for --agent targeting
	if len(meta.AgentTypes) != 1 || meta.AgentTypes[0] != "qa" {
		t.Errorf("qa agent_types = %v, want [qa]", meta.AgentTypes)
	}
	// orchestrator action built from consolidated fields
	if meta.OrchestratorAction == nil {
		t.Fatal("qa orchestrator action not built")
	}
	oa := meta.OrchestratorAction
	if oa.Action != action.ActionSpawnAgent || oa.AgentType != "qa" {
		t.Errorf("qa action wrong: %+v", oa)
	}
	if oa.Provider != "anthropic" || oa.Model != "sonnet" {
		t.Errorf("qa provider/model wrong: %+v", oa)
	}
	if oa.InstructionTemplate != "feature/qa.md" {
		t.Errorf("qa prompt -> instruction_template = %q, want feature/qa.md", oa.InstructionTemplate)
	}
	if len(oa.Skills) != 1 || oa.Skills[0] != "quality" {
		t.Errorf("qa skills = %v, want [quality]", oa.Skills)
	}
}

func TestDeriveLegacyFromSteps_SpecialStatuses(t *testing.T) {
	cfg := sampleStepsConfig()
	deriveLegacyFromSteps(cfg)

	if got := cfg.SpecialStatuses[StartStatusKey]; len(got) != 1 || got[0] != "draft" {
		t.Errorf("_start_ = %v, want [draft]", got)
	}
	if got := cfg.SpecialStatuses[CompleteStatusKey]; len(got) != 1 || got[0] != "completed" {
		t.Errorf("_complete_ = %v, want [completed]", got)
	}
}

func TestDeriveLegacyFromSteps_Aggregation(t *testing.T) {
	cfg := &WorkflowConfig{
		Start: "draft",
		Steps: map[string]*Step{
			"draft":     {Phase: "planning", Outcomes: map[string]string{"pass": "active"}},
			"active":    {Phase: "development", AggregatesFrom: "features", Outcomes: map[string]string{"pass": "completed"}},
			"completed": {Phase: "done", Terminal: true},
		},
	}
	deriveLegacyFromSteps(cfg)
	if got := cfg.SpecialStatuses[AggregationStatusKey]; len(got) != 1 || got[0] != "active" {
		t.Errorf("_aggregation_ = %v, want [active]", got)
	}
}

func TestResolveOutcome(t *testing.T) {
	cfg := sampleStepsConfig()

	tests := []struct {
		from, outcome, want string
		ok                  bool
	}{
		{"qa", "pass", "completed", true},
		{"qa", "fail", "draft", true},
		{"qa", "blocked", "on_hold", true},
		{"qa", "PASS", "completed", true}, // case-insensitive
		{"qa", "dead-end", "", false},     // undefined outcome
		{"completed", "pass", "", false},  // terminal: no outcomes
		{"nonexistent", "pass", "", false},
	}
	for _, tt := range tests {
		got, ok := cfg.ResolveOutcome(tt.from, tt.outcome)
		if ok != tt.ok || got != tt.want {
			t.Errorf("ResolveOutcome(%q,%q) = (%q,%v), want (%q,%v)",
				tt.from, tt.outcome, got, ok, tt.want, tt.ok)
		}
	}
}

func TestDeriveLegacyFromSteps_NoOpWhenEmpty(t *testing.T) {
	cfg := &WorkflowConfig{
		StatusFlow:     map[string][]string{"todo": {"done"}},
		StatusMetadata: map[string]StatusMetadata{"todo": {Phase: "planning"}},
	}
	deriveLegacyFromSteps(cfg)
	if len(cfg.StatusFlow) != 1 || cfg.StatusFlow["todo"][0] != "done" {
		t.Errorf("legacy-only config was mutated: %v", cfg.StatusFlow)
	}
}

func TestParseWorkflowYAML_StepsShape(t *testing.T) {
	yaml := []byte(`
version: "1.0"
start: draft
steps:
  draft:
    phase: planning
    action: advance_status
    outcomes: { pass: review }
  review:
    phase: review
    color: yellow
    action: spawn_agent
    agent: reviewer
    skills: [code-review]
    prompt: task/review.md
    aliases: [ready_for_review, in_review]
    outcomes:
      pass: done
      fail: draft
      blocked: on_hold
  on_hold:
    phase: paused
    parking: true
  done:
    phase: done
    terminal: true
`)
	cfg, err := parseWorkflowYAML(yaml, "test.yaml")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if !cfg.HasSteps() {
		t.Fatal("HasSteps() = false, want true")
	}
	// Derived legacy maps must be present so existing readers work.
	if got := cfg.StatusFlow["draft"]; len(got) != 1 || got[0] != "review" {
		t.Errorf("draft transitions = %v, want [review]", got)
	}
	target, ok := cfg.ResolveOutcome("review", "fail")
	if !ok || target != "draft" {
		t.Errorf("ResolveOutcome(review,fail) = (%q,%v), want (draft,true)", target, ok)
	}
	meta, ok := cfg.GetStatusMetadata("review")
	if !ok || meta.Color != "yellow" {
		t.Errorf("review color not derived: %+v", meta)
	}
}

func equalSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
