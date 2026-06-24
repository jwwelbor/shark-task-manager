package workflow

import "testing"

func TestValidateCoreOutcomes_AllPresent(t *testing.T) {
	cfg := sampleStepsConfig()
	// draft only defines pass -> it is workable, so missing fail/blocked.
	errs := cfg.ValidateCoreOutcomes()
	// draft is missing fail+blocked; qa is complete; on_hold/completed skipped.
	if len(errs) != 1 {
		t.Fatalf("expected 1 core-outcome error (draft), got %d: %v", len(errs), errs)
	}
	coErr, ok := errs[0].(*CoreOutcomeError)
	if !ok || coErr.Step != "draft" {
		t.Fatalf("expected CoreOutcomeError for draft, got %v", errs[0])
	}
	if len(coErr.Missing) != 2 {
		t.Errorf("draft missing = %v, want [blocked fail] (2)", coErr.Missing)
	}
}

func TestValidateCoreOutcomes_Complete(t *testing.T) {
	cfg := &WorkflowConfig{
		Start: "dev",
		Steps: map[string]*Step{
			"dev": {
				Phase:    "development",
				Outcomes: map[string]string{"pass": "done", "fail": "dev", "blocked": "hold"},
			},
			"hold": {Phase: "paused", Parking: true},
			"done": {Phase: "done", Terminal: true},
		},
	}
	if errs := cfg.ValidateCoreOutcomes(); len(errs) != 0 {
		t.Errorf("expected no errors for complete config, got %v", errs)
	}
}

func TestValidateCoreOutcomes_UnknownTarget(t *testing.T) {
	cfg := &WorkflowConfig{
		Start: "dev",
		Steps: map[string]*Step{
			"dev": {
				Phase:    "development",
				Outcomes: map[string]string{"pass": "ghost", "fail": "dev", "blocked": "dev"},
			},
		},
	}
	errs := cfg.ValidateCoreOutcomes()
	found := false
	for _, e := range errs {
		if _, ok := e.(*CoreOutcomeError); !ok {
			found = true // a non-core error => the unknown-target error
		}
	}
	if !found {
		t.Errorf("expected an unknown-target error, got %v", errs)
	}
}

func TestValidateCoreOutcomes_LegacyNoOp(t *testing.T) {
	cfg := &WorkflowConfig{StatusFlow: map[string][]string{"todo": {"done"}}}
	if errs := cfg.ValidateCoreOutcomes(); errs != nil {
		t.Errorf("legacy config should produce no core-outcome errors, got %v", errs)
	}
}
