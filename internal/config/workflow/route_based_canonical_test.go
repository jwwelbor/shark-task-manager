package workflow

import (
	"testing"
)

// TestCanonicalWorkflows_AreRouteBased guards the E35-F01 migration: every
// embedded default workflow YAML must load as a valid route-based (steps:)
// config. For each entity slot the file must:
//   - parse and populate its slot,
//   - use the consolidated steps: schema (HasSteps),
//   - pass full ValidateWorkflow (legacy reachability/terminal-path on the
//     derived maps AND the route-based core-outcome/start/alias checks),
//   - have a collision-free alias map.
//
// This locks the default shipped workflows onto the route-based shape so a
// regression back to the two-map form (or a malformed steps: block) fails fast.
func TestCanonicalWorkflows_AreRouteBased(t *testing.T) {
	mlw, err := LoadMultiLevelWorkflowFromYAMLDir(canonicalWorkflowDir(t), "")
	if err != nil {
		t.Fatalf("failed to load canonical workflows: %v", err)
	}

	slots := map[string]*WorkflowConfig{
		"epic":     mlw.Epic,
		"feature":  mlw.Feature,
		"task":     mlw.Task,
		"bug":      mlw.Bug,
		"change":   mlw.Change,
		"techDebt": mlw.TechDebt,
		"sprint":   mlw.Sprint,
	}

	loaded := 0
	for slot, cfg := range slots {
		if cfg == nil {
			t.Errorf("canonical %s workflow did not load (nil slot)", slot)
			continue
		}
		loaded++

		if !cfg.HasSteps() {
			t.Errorf("canonical %s workflow is not route-based (no steps: block)", slot)
		}
		if err := ValidateWorkflow(cfg); err != nil {
			t.Errorf("canonical %s workflow failed validation: %v", slot, err)
		}
		if _, aliasErrs := cfg.AliasMap(); len(aliasErrs) > 0 {
			for _, e := range aliasErrs {
				t.Errorf("canonical %s workflow alias collision: %v", slot, e)
			}
		}
		// Every workable step must define the core outcome vocabulary with
		// targets that resolve to real steps.
		if errs := cfg.ValidateCoreOutcomes(); len(errs) > 0 {
			for _, e := range errs {
				t.Errorf("canonical %s workflow core-outcome error: %v", slot, e)
			}
		}
	}

	// epic/feature/task/bug/change must always be present; tech-debt and sprint
	// are also shipped as route-based here, so we expect all seven.
	if loaded < 7 {
		t.Fatalf("expected 7 canonical route-based workflows, loaded %d", loaded)
	}
}
