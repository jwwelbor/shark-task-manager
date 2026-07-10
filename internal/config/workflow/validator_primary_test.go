package workflow

import (
	"strings"
	"testing"
)

// pfb builds the core pass/fail/blocked outcome map used by the fixtures.
func pfb(pass, fail, blocked string) map[string]string {
	return map[string]string{"pass": pass, "fail": fail, "blocked": blocked}
}

// designationFixture returns a route-based workflow in which every semantic
// selection has two candidates and alphabetical order contradicts semantic
// order (the "aaa_wrong_*" twin always sorts first). The semantically right
// candidate of each pair carries primary: true, so the fixture validates
// clean; tests mutate it to produce the ambiguous variants.
func designationFixture() *WorkflowConfig {
	cfg := &WorkflowConfig{
		Start: "reopen",
		Steps: map[string]*Step{
			// Aggregation (reopen-target) pair — phase deliberately not one of
			// the validated phases so the dimensions stay independent.
			"reopen":           {Phase: "development", AggregatesFrom: "tasks", Primary: true, Outcomes: pfb("aaa_wrong_reopen", "reopen", "hold")},
			"aaa_wrong_reopen": {Phase: "development", AggregatesFrom: "tasks", Outcomes: pfb("active", "aaa_wrong_reopen", "hold")},

			// Execution-phase pair.
			"active":           {Phase: "execution", Primary: true, Outcomes: pfb("aaa_wrong_active", "active", "hold")},
			"aaa_wrong_active": {Phase: "execution", Outcomes: pfb("closing", "aaa_wrong_active", "hold")},

			// Review-phase pair.
			"closing":           {Phase: "review", Primary: true, Outcomes: pfb("aaa_wrong_closing", "closing", "hold")},
			"aaa_wrong_closing": {Phase: "review", Outcomes: pfb("completed", "aaa_wrong_closing", "hold")},

			// Done-phase non-terminal pair (the completed-sprint selection).
			"completed":           {Phase: "done", Primary: true, Outcomes: pfb("aaa_wrong_completed", "completed", "hold")},
			"aaa_wrong_completed": {Phase: "done", Outcomes: pfb("archived", "aaa_wrong_archived", "hold")},

			// Terminal pair — no action: archive, so the primary tag decides.
			"archived":           {Phase: "done", Terminal: true, Primary: true},
			"aaa_wrong_archived": {Phase: "done", Terminal: true},

			// Parking step for the blocked outcomes.
			"hold": {Phase: "paused", Parking: true, Action: "pause"},
		},
	}
	cfg.DeriveLegacy()
	return cfg
}

func TestValidateWorkflow_PrimaryDesignations_ValidFixture(t *testing.T) {
	if err := ValidateWorkflow(designationFixture()); err != nil {
		t.Fatalf("fixture with primary tags should validate clean, got: %v", err)
	}
}

func TestValidateWorkflow_AmbiguousAggregation_NoPrimary(t *testing.T) {
	cfg := designationFixture()
	cfg.Steps["reopen"].Primary = false
	err := ValidateWorkflow(cfg)
	if err == nil || !strings.Contains(err.Error(), "aggregation") || !strings.Contains(err.Error(), "primary") {
		t.Errorf("expected ambiguous-aggregation error naming the primary fix, got %v", err)
	}
	// The error must name the candidates so the fix is actionable.
	if err != nil && (!strings.Contains(err.Error(), "reopen") || !strings.Contains(err.Error(), "aaa_wrong_reopen")) {
		t.Errorf("expected error to name both candidates, got %v", err)
	}
}

func TestValidateWorkflow_AmbiguousAggregation_MultiplePrimary(t *testing.T) {
	cfg := designationFixture()
	cfg.Steps["aaa_wrong_reopen"].Primary = true
	err := ValidateWorkflow(cfg)
	if err == nil || !strings.Contains(err.Error(), "multiple steps tagged primary") {
		t.Errorf("expected multiple-primary error, got %v", err)
	}
}

func TestValidateWorkflow_AmbiguousPhase_NoPrimary(t *testing.T) {
	for _, tc := range []struct{ phase, step string }{
		{"execution", "active"},
		{"review", "closing"},
	} {
		cfg := designationFixture()
		cfg.Steps[tc.step].Primary = false
		err := ValidateWorkflow(cfg)
		if err == nil || !strings.Contains(err.Error(), tc.phase) || !strings.Contains(err.Error(), "primary") {
			t.Errorf("phase %s: expected ambiguous-phase error, got %v", tc.phase, err)
		}
	}
}

func TestValidateWorkflow_AmbiguousTerminals_NoArchiveNoPrimary(t *testing.T) {
	cfg := designationFixture()
	cfg.Steps["archived"].Primary = false
	err := ValidateWorkflow(cfg)
	if err == nil || !strings.Contains(err.Error(), "terminal") || !strings.Contains(err.Error(), "primary") {
		t.Errorf("expected ambiguous-terminal error, got %v", err)
	}
}

func TestValidateWorkflow_MultipleTerminals_ArchiveActionSuffices(t *testing.T) {
	// An action: archive terminal designates the archive endpoint, so no
	// primary tag is required (matches the shipped default workflows, which
	// have several archive-action terminals).
	cfg := designationFixture()
	cfg.Steps["archived"].Primary = false
	cfg.Steps["archived"].Action = "archive"
	if err := ValidateWorkflow(cfg); err != nil {
		t.Errorf("archive-action terminal should satisfy the terminal rule, got %v", err)
	}
}

func TestValidateWorkflow_ParkingStepWithAdvanceStatus(t *testing.T) {
	cfg := designationFixture()
	cfg.Steps["hold"].Action = "advance_status"
	err := ValidateWorkflow(cfg)
	if err == nil || !strings.Contains(err.Error(), "parking") || !strings.Contains(err.Error(), "advance_status") {
		t.Errorf("expected parking/advance_status validation error, got %v", err)
	}
}

func TestValidateWorkflow_SingleCandidates_NoPrimaryNeeded(t *testing.T) {
	// The plain valid route-based config has at most one candidate per
	// selection; no primary tags anywhere must stay valid.
	if err := ValidateWorkflow(validRouteBasedConfig()); err != nil {
		t.Errorf("single-candidate workflow must not require primary tags, got %v", err)
	}
}

// The embedded default workflows are covered by
// TestCanonicalWorkflows_AreRouteBased, which runs full ValidateWorkflow
// (including these designation rules) over every shipped YAML.
