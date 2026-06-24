package workflow

import "testing"

func aliasConfig() *WorkflowConfig {
	return &WorkflowConfig{
		Start: "refinement",
		Steps: map[string]*Step{
			"refinement": {
				Phase:    "refinement",
				Aliases:  []string{"ready_for_refinement", "in_refinement"},
				Outcomes: map[string]string{"pass": "qa", "fail": "refinement", "blocked": "blocked"},
			},
			"qa": {
				Phase:    "qa",
				Aliases:  []string{"ready_for_qa", "in_qa"},
				Outcomes: map[string]string{"pass": "completed", "fail": "refinement", "blocked": "blocked"},
			},
			"blocked":   {Phase: "blocked", Parking: true},
			"completed": {Phase: "done", Terminal: true},
		},
	}
}

func TestAliasMap(t *testing.T) {
	cfg := aliasConfig()
	m, errs := cfg.AliasMap()
	if len(errs) != 0 {
		t.Fatalf("unexpected alias errors: %v", errs)
	}
	want := map[string]string{
		"ready_for_refinement": "refinement",
		"in_refinement":        "refinement",
		"ready_for_qa":         "qa",
		"in_qa":                "qa",
	}
	for old, exp := range want {
		if m[old] != exp {
			t.Errorf("AliasMap[%q] = %q, want %q", old, m[old], exp)
		}
	}
}

func TestAliasMap_Collision(t *testing.T) {
	cfg := &WorkflowConfig{
		Steps: map[string]*Step{
			"a": {Aliases: []string{"dup"}, Outcomes: map[string]string{"pass": "a"}},
			"b": {Aliases: []string{"dup"}, Outcomes: map[string]string{"pass": "b"}},
		},
	}
	_, errs := cfg.AliasMap()
	if len(errs) == 0 {
		t.Error("expected a collision error for alias 'dup'")
	}
}

func TestResolveAlias(t *testing.T) {
	cfg := aliasConfig()
	cases := []struct{ in, want string }{
		{"ready_for_qa", "qa"}, // old alias -> new step
		{"in_refinement", "refinement"},
		{"qa", "qa"},                         // already a step -> unchanged
		{"unknown_status", "unknown_status"}, // unknown -> unchanged
		{"READY_FOR_QA", "qa"},               // case-insensitive
	}
	for _, c := range cases {
		if got := cfg.ResolveAlias(c.in); got != c.want {
			t.Errorf("ResolveAlias(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
