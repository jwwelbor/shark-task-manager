package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The selector tests lean on designationFixture (validator_primary_test.go):
// every semantic selection has two candidates, the "aaa_wrong_*" twin always
// sorts first alphabetically, and the semantically right twin carries
// primary: true. A selector that regressed to a positional/alphabetical pick
// returns the aaa_wrong_* name and fails these tests.

func TestPrimaryAggregationStatus_PicksPrimaryNotAlphabetical(t *testing.T) {
	got, err := designationFixture().PrimaryAggregationStatus()
	require.NoError(t, err)
	assert.Equal(t, "reopen", got, "expected primary-tagged status")
}

func TestPrimaryAggregationStatus_SingleCandidate(t *testing.T) {
	cfg := designationFixture()
	// Drop one candidate's aggregation marker: a single candidate needs no tag.
	cfg.Steps["reopen"].AggregatesFrom = ""
	cfg.Steps["reopen"].Primary = false
	cfg.SpecialStatuses = nil
	cfg.DeriveLegacy()

	got, err := cfg.PrimaryAggregationStatus()
	require.NoError(t, err)
	assert.Equal(t, "aaa_wrong_reopen", got, "expected sole candidate")
}

func TestPrimaryAggregationStatus_Ambiguous(t *testing.T) {
	cfg := designationFixture()
	cfg.Steps["reopen"].Primary = false

	_, err := cfg.PrimaryAggregationStatus()
	var ambiguous *AmbiguousSelectionError
	require.ErrorAs(t, err, &ambiguous)
	// The error must name the candidates and the fix.
	msg := err.Error()
	for _, want := range []string{"aaa_wrong_reopen", "reopen", "primary: true", "shark admin workflow validate"} {
		assert.Contains(t, msg, want, "error message must name the candidate and fix")
	}
}

func TestPrimaryAggregationStatus_NoCandidate(t *testing.T) {
	cfg := designationFixture()
	cfg.Steps["reopen"].AggregatesFrom = ""
	cfg.Steps["aaa_wrong_reopen"].AggregatesFrom = ""
	cfg.SpecialStatuses = nil
	cfg.DeriveLegacy()

	_, err := cfg.PrimaryAggregationStatus()
	var noCandidate *NoCandidateError
	require.ErrorAs(t, err, &noCandidate)
}

func TestStatusForPhase_PicksPrimaryNotAlphabetical(t *testing.T) {
	cfg := designationFixture()
	for phase, want := range map[string]string{
		"execution": "active",
		"review":    "closing",
	} {
		got, err := cfg.StatusForPhase(phase)
		require.NoErrorf(t, err, "phase %s: unexpected error", phase)
		assert.Equalf(t, want, got, "phase %s: expected primary-tagged status", phase)
	}
}

func TestStatusForPhase_AmbiguousAndNoCandidate(t *testing.T) {
	cfg := designationFixture()
	cfg.Steps["active"].Primary = false

	_, err := cfg.StatusForPhase("execution")
	var ambiguous *AmbiguousSelectionError
	require.ErrorAs(t, err, &ambiguous)

	_, err = cfg.StatusForPhase("no_such_phase")
	var noCandidate *NoCandidateError
	require.ErrorAs(t, err, &noCandidate)
}

func TestCompletedSprintStatus_ExcludesTerminalsAndPicksPrimary(t *testing.T) {
	// The fixture's done phase holds four steps: two non-terminal candidates
	// (primary on "completed") and two terminals that must be excluded.
	got, err := designationFixture().CompletedSprintStatus()
	require.NoError(t, err)
	assert.Equal(t, "completed", got, "expected primary-tagged status")
}

func TestCompletedSprintStatus_Ambiguous(t *testing.T) {
	cfg := designationFixture()
	cfg.Steps["completed"].Primary = false

	_, err := cfg.CompletedSprintStatus()
	var ambiguous *AmbiguousSelectionError
	require.ErrorAs(t, err, &ambiguous)
	// The candidate list must name only the non-terminal done-phase steps —
	// terminals are a different selection (ArchiveTerminalStatus).
	want := []string{"aaa_wrong_completed", "completed"}
	require.Equal(t, want, ambiguous.Candidates, "expected non-terminal done-phase candidates")
	for i, name := range want {
		assert.Equalf(t, name, ambiguous.Candidates[i], "candidate %d", i)
	}
}

func TestCompletedSprintStatus_NoCandidate(t *testing.T) {
	cfg := designationFixture()
	cfg.Steps["completed"].Phase = "development"
	cfg.Steps["aaa_wrong_completed"].Phase = "development"
	cfg.DeriveLegacy()

	_, err := cfg.CompletedSprintStatus()
	var noCandidate *NoCandidateError
	require.ErrorAs(t, err, &noCandidate)
}

func TestArchiveTerminalStatus_PrimaryBreaksTie(t *testing.T) {
	// No archive-action terminals in the fixture: the primary tag decides.
	got, err := designationFixture().ArchiveTerminalStatus()
	require.NoError(t, err)
	assert.Equal(t, "archived", got, "expected primary-tagged archive terminal")
}

func TestArchiveTerminalStatus_ArchiveActionTakesPrecedence(t *testing.T) {
	cfg := designationFixture()
	// The alphabetically-first terminal gets the archive action; it must win
	// even though the other one is tagged primary (the action is the stronger,
	// operation-specific designation).
	cfg.Steps["aaa_wrong_archived"].Action = "archive"
	cfg.DeriveLegacy()

	got, err := cfg.ArchiveTerminalStatus()
	require.NoError(t, err)
	assert.Equal(t, "aaa_wrong_archived", got, "expected archive-action terminal")
}

func TestArchiveTerminalStatus_MultipleArchiveActions_PrimaryDecides(t *testing.T) {
	cfg := designationFixture()
	cfg.Steps["aaa_wrong_archived"].Action = "archive"
	cfg.Steps["archived"].Action = "archive"
	cfg.DeriveLegacy()

	// primary: true on "archived" breaks the tie within the archive subset.
	got, err := cfg.ArchiveTerminalStatus()
	require.NoError(t, err)
	assert.Equal(t, "archived", got, "expected primary-tagged archive terminal")

	// Without the tag the tie is unbreakable: ambiguous, never alphabetical.
	cfg.Steps["archived"].Primary = false
	_, err = cfg.ArchiveTerminalStatus()
	var ambiguous *AmbiguousSelectionError
	require.ErrorAs(t, err, &ambiguous)
}

func TestArchiveTerminalStatus_NoCandidate(t *testing.T) {
	cfg := &WorkflowConfig{StatusFlow: map[string][]string{"todo": {}}}
	_, err := cfg.ArchiveTerminalStatus()
	var noCandidate *NoCandidateError
	require.ErrorAs(t, err, &noCandidate)
}

func TestDesignate_LegacySchemaFallsBackToFirstCandidate(t *testing.T) {
	// A legacy (status_flow) config cannot express primary: true, so the
	// designation rule preserves the pre-2.x behavior: first candidate wins
	// (declaration order for special_statuses arrays). Only route-based
	// (steps:) workflows get the strict ambiguity error.
	cfg := &WorkflowConfig{
		StatusFlow: map[string][]string{
			"draft":     {"zz_active"},
			"zz_active": {"done"},
			"aa_other":  {"done"},
			"done":      {},
		},
		SpecialStatuses: map[string][]string{
			StartStatusKey:       {"draft"},
			CompleteStatusKey:    {"done"},
			AggregationStatusKey: {"zz_active", "aa_other"}, // declaration order, not alphabetical
		},
	}

	got, err := cfg.PrimaryAggregationStatus()
	require.NoError(t, err, "unexpected error for legacy config")
	assert.Equal(t, "zz_active", got, "expected declaration-order first status")
}

// TestGetStatusesByAgentType_Sorted pins the determinism guarantee added with
// the selectors: StatusMetadata is a map, so without the sort the output
// order would vary between calls (an intermittent regression if removed).
func TestGetStatusesByAgentType_Sorted(t *testing.T) {
	cfg := &WorkflowConfig{
		StatusMetadata: map[string]StatusMetadata{
			"zebra": {AgentTypes: []string{"qa"}},
			"alpha": {AgentTypes: []string{"qa"}},
			"mango": {AgentTypes: []string{"qa"}},
			"other": {AgentTypes: []string{"dev"}},
		},
	}
	want := []string{"alpha", "mango", "zebra"}
	for i := 0; i < 20; i++ {
		got := cfg.GetStatusesByAgentType("qa")
		require.Len(t, got, len(want))
		for j := range want {
			assert.Equalf(t, want[j], got[j], "iteration %d: expected sorted statuses", i)
		}
	}
}
