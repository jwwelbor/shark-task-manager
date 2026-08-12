package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		Start: "planning",
		Steps: map[string]*Step{
			// Planning-phase pair.
			"planning":           {Phase: "planning", Primary: true, Outcomes: pfb("reopen", "planning", "hold")},
			"aaa_wrong_planning": {Phase: "planning", Outcomes: pfb("reopen", "aaa_wrong_planning", "hold")},

			// Aggregation (reopen-target) pair — phase deliberately not one of
			// the sprint-specific phases so the dimensions stay independent.
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
	require.NoError(t, ValidateWorkflow(designationFixture()), "fixture with primary tags should validate clean")
}

func TestValidateWorkflow_AmbiguousAggregation_NoPrimary(t *testing.T) {
	cfg := designationFixture()
	cfg.Steps["reopen"].Primary = false
	err := ValidateWorkflow(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "aggregation")
	assert.Contains(t, err.Error(), "primary")
	// The error must name the candidates so the fix is actionable.
	assert.Contains(t, err.Error(), "reopen")
	assert.Contains(t, err.Error(), "aaa_wrong_reopen")
}

func TestValidateWorkflow_AmbiguousAggregation_MultiplePrimary(t *testing.T) {
	cfg := designationFixture()
	cfg.Steps["aaa_wrong_reopen"].Primary = true
	err := ValidateWorkflow(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple steps tagged primary")
}

func TestValidateWorkflow_AmbiguousPhase_NoPrimary(t *testing.T) {
	for _, tc := range []struct{ phase, step string }{
		{"planning", "planning"},
		{"execution", "active"},
		{"review", "closing"},
	} {
		cfg := designationFixture()
		cfg.Steps[tc.step].Primary = false
		err := ValidateWorkflow(cfg)
		require.Error(t, err, "phase %s: expected ambiguous-phase error", tc.phase)
		assert.Contains(t, err.Error(), tc.phase)
		assert.Contains(t, err.Error(), "primary")
	}
}

func TestValidateWorkflow_AmbiguousCompletedSprintStatus_NoPrimary(t *testing.T) {
	cfg := designationFixture()
	cfg.Steps["completed"].Primary = false
	err := ValidateWorkflow(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "completed (done-phase, non-terminal)")
	assert.Contains(t, err.Error(), "primary")
}

func TestValidateWorkflow_AmbiguousTerminals_NoArchiveNoPrimary(t *testing.T) {
	cfg := designationFixture()
	cfg.Steps["archived"].Primary = false
	err := ValidateWorkflow(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "terminal")
	assert.Contains(t, err.Error(), "primary")
}

func TestValidateWorkflow_MultipleTerminals_ArchiveActionSuffices(t *testing.T) {
	// A single action: archive terminal designates the archive endpoint, so
	// no primary tag is required.
	cfg := designationFixture()
	cfg.Steps["archived"].Primary = false
	cfg.Steps["archived"].Action = "archive"
	cfg.DeriveLegacy()
	assert.NoError(t, ValidateWorkflow(cfg), "archive-action terminal should satisfy the terminal rule")
}

func TestValidateWorkflow_MultipleArchiveTerminals_RequirePrimary(t *testing.T) {
	// Several action: archive terminals narrow the candidate set but still
	// need a primary tag — mirroring ArchiveTerminalStatus, so a workflow that
	// validates clean can never hard-fail the runtime selector (and vice
	// versa).
	cfg := designationFixture()
	cfg.Steps["archived"].Primary = false
	cfg.Steps["archived"].Action = "archive"
	cfg.Steps["aaa_wrong_archived"].Action = "archive"
	cfg.DeriveLegacy()

	err := ValidateWorkflow(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "archive terminal")

	// Tagging one of the archive terminals satisfies both validator and
	// runtime selector.
	cfg.Steps["archived"].Primary = true
	assert.NoError(t, ValidateWorkflow(cfg), "primary-tagged archive terminal should validate clean")
	got, selErr := cfg.ArchiveTerminalStatus()
	assert.NoError(t, selErr)
	assert.Equal(t, "archived", got, "expected runtime selector to agree")
}

func TestValidateWorkflow_ParkingStepWithAdvanceStatus(t *testing.T) {
	cfg := designationFixture()
	cfg.Steps["hold"].Action = "advance_status"
	err := ValidateWorkflow(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parking")
	assert.Contains(t, err.Error(), "advance_status")
}

func TestValidateWorkflow_SingleCandidates_NoPrimaryNeeded(t *testing.T) {
	// The plain valid route-based config has at most one candidate per
	// selection; no primary tags anywhere must stay valid.
	assert.NoError(t, ValidateWorkflow(validRouteBasedConfig()), "single-candidate workflow must not require primary tags")
}

// The embedded default workflows are covered by
// TestCanonicalWorkflows_AreRouteBased, which runs full ValidateWorkflow
// (including these designation rules) over every shipped YAML.
