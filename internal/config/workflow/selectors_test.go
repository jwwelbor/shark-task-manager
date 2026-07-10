package workflow

import (
	"errors"
	"strings"
	"testing"
)

// The selector tests lean on designationFixture (validator_primary_test.go):
// every semantic selection has two candidates, the "aaa_wrong_*" twin always
// sorts first alphabetically, and the semantically right twin carries
// primary: true. A selector that regressed to a positional/alphabetical pick
// returns the aaa_wrong_* name and fails these tests.

func TestPrimaryAggregationStatus_PicksPrimaryNotAlphabetical(t *testing.T) {
	got, err := designationFixture().PrimaryAggregationStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "reopen" {
		t.Errorf("expected primary-tagged %q, got %q", "reopen", got)
	}
}

func TestPrimaryAggregationStatus_SingleCandidate(t *testing.T) {
	cfg := designationFixture()
	// Drop one candidate's aggregation marker: a single candidate needs no tag.
	cfg.Steps["reopen"].AggregatesFrom = ""
	cfg.Steps["reopen"].Primary = false
	cfg.SpecialStatuses = nil
	cfg.DeriveLegacy()

	got, err := cfg.PrimaryAggregationStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "aaa_wrong_reopen" {
		t.Errorf("expected sole candidate %q, got %q", "aaa_wrong_reopen", got)
	}
}

func TestPrimaryAggregationStatus_Ambiguous(t *testing.T) {
	cfg := designationFixture()
	cfg.Steps["reopen"].Primary = false

	_, err := cfg.PrimaryAggregationStatus()
	var ambiguous *AmbiguousSelectionError
	if !errors.As(err, &ambiguous) {
		t.Fatalf("expected AmbiguousSelectionError, got %v", err)
	}
	// The error must name the candidates and the fix.
	msg := err.Error()
	for _, want := range []string{"aaa_wrong_reopen", "reopen", "primary: true", "shark admin workflow validate"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error message missing %q: %s", want, msg)
		}
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
	if !errors.As(err, &noCandidate) {
		t.Fatalf("expected NoCandidateError, got %v", err)
	}
}

func TestStatusForPhase_PicksPrimaryNotAlphabetical(t *testing.T) {
	cfg := designationFixture()
	for phase, want := range map[string]string{
		"execution": "active",
		"review":    "closing",
	} {
		got, err := cfg.StatusForPhase(phase)
		if err != nil {
			t.Fatalf("phase %s: unexpected error: %v", phase, err)
		}
		if got != want {
			t.Errorf("phase %s: expected primary-tagged %q, got %q", phase, want, got)
		}
	}
}

func TestStatusForPhase_AmbiguousAndNoCandidate(t *testing.T) {
	cfg := designationFixture()
	cfg.Steps["active"].Primary = false

	_, err := cfg.StatusForPhase("execution")
	var ambiguous *AmbiguousSelectionError
	if !errors.As(err, &ambiguous) {
		t.Fatalf("expected AmbiguousSelectionError, got %v", err)
	}

	_, err = cfg.StatusForPhase("no_such_phase")
	var noCandidate *NoCandidateError
	if !errors.As(err, &noCandidate) {
		t.Fatalf("expected NoCandidateError, got %v", err)
	}
}

func TestCompletedSprintStatus_ExcludesTerminalsAndPicksPrimary(t *testing.T) {
	// The fixture's done phase holds four steps: two non-terminal candidates
	// (primary on "completed") and two terminals that must be excluded.
	got, err := designationFixture().CompletedSprintStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "completed" {
		t.Errorf("expected primary-tagged %q, got %q", "completed", got)
	}
}

func TestCompletedSprintStatus_NoCandidate(t *testing.T) {
	cfg := designationFixture()
	cfg.Steps["completed"].Phase = "development"
	cfg.Steps["aaa_wrong_completed"].Phase = "development"
	cfg.DeriveLegacy()

	_, err := cfg.CompletedSprintStatus()
	var noCandidate *NoCandidateError
	if !errors.As(err, &noCandidate) {
		t.Fatalf("expected NoCandidateError, got %v", err)
	}
}

func TestArchiveTerminalStatus_PrimaryBreaksTie(t *testing.T) {
	// No archive-action terminals in the fixture: the primary tag decides.
	got, err := designationFixture().ArchiveTerminalStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "archived" {
		t.Errorf("expected primary-tagged %q, got %q", "archived", got)
	}
}

func TestArchiveTerminalStatus_ArchiveActionTakesPrecedence(t *testing.T) {
	cfg := designationFixture()
	// The alphabetically-first terminal gets the archive action; it must win
	// even though the other one is tagged primary (the action is the stronger,
	// operation-specific designation).
	cfg.Steps["aaa_wrong_archived"].Action = "archive"
	cfg.DeriveLegacy()

	got, err := cfg.ArchiveTerminalStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "aaa_wrong_archived" {
		t.Errorf("expected archive-action terminal %q, got %q", "aaa_wrong_archived", got)
	}
}

func TestArchiveTerminalStatus_MultipleArchiveActions_PrimaryDecides(t *testing.T) {
	cfg := designationFixture()
	cfg.Steps["aaa_wrong_archived"].Action = "archive"
	cfg.Steps["archived"].Action = "archive"
	cfg.DeriveLegacy()

	// primary: true on "archived" breaks the tie within the archive subset.
	got, err := cfg.ArchiveTerminalStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "archived" {
		t.Errorf("expected primary-tagged archive terminal %q, got %q", "archived", got)
	}

	// Without the tag the tie is unbreakable: ambiguous, never alphabetical.
	cfg.Steps["archived"].Primary = false
	_, err = cfg.ArchiveTerminalStatus()
	var ambiguous *AmbiguousSelectionError
	if !errors.As(err, &ambiguous) {
		t.Fatalf("expected AmbiguousSelectionError, got %v", err)
	}
}

func TestArchiveTerminalStatus_NoCandidate(t *testing.T) {
	cfg := &WorkflowConfig{StatusFlow: map[string][]string{"todo": {}}}
	_, err := cfg.ArchiveTerminalStatus()
	var noCandidate *NoCandidateError
	if !errors.As(err, &noCandidate) {
		t.Fatalf("expected NoCandidateError, got %v", err)
	}
}

func TestDefaultTransition_PassFirstContract(t *testing.T) {
	cfg := validRouteBasedConfig()
	// todo's outcomes: pass→qa, fail→todo, blocked→blocked. The default
	// transition is the pass target, not the alphabetical first.
	got, err := cfg.DefaultTransition("todo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "qa" {
		t.Errorf("expected pass target %q, got %q", "qa", got)
	}
}

func TestDefaultTransition_TerminalAndUnknown(t *testing.T) {
	cfg := validRouteBasedConfig()
	var noCandidate *NoCandidateError

	_, err := cfg.DefaultTransition("done")
	if !errors.As(err, &noCandidate) {
		t.Fatalf("expected NoCandidateError for terminal status, got %v", err)
	}

	_, err = cfg.DefaultTransition("ghost")
	if !errors.As(err, &noCandidate) {
		t.Fatalf("expected NoCandidateError for unknown status, got %v", err)
	}
}
