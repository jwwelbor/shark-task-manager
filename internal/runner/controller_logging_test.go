// Package runner
//
// Controller-level disabled-observability gap-fill tests.
//
// These tests complement (without duplicating) the stage-event coverage in
// controller_stage_events_test.go and the transcript coverage in
// transcript_test.go. Specifically they assert:
//
//	When observability.enabled is false, NO run.* or run.stage.* slog
//	events are emitted at all — not just the stage.* subset verified by
//	TestController_NoEventsWhenObservabilityDisabled.
//
// The existing controller disabled-path test only asserts the absence of
// "run.stage.*" messages. This file tightens that contract to "nothing with
// the run. prefix at all" at the controller boundary, guarding against
// regressions where a new log site (e.g. run.transcript.*) is added without
// honoring the observability gate.
//
// Golden rule: mocks only — no real database, dispatcher, or filesystem.
package runner

import (
	"context"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

// TestController_NoRunOrStageEventsWhenObservabilityDisabled is the
// full-run disabled-observability assertion at the controller boundary.
//
// It drives a complete happy-path Run through RunController with
// Observability.Enabled=false and asserts that the captured slog stream
// contains ZERO events whose "msg" begins with "run." — covering
// run.stage.start, run.stage.dispatch, run.stage.complete,
// run.stage.transition, run.stage.error, and any future run.*/run.stage.*
// emission site. This is strictly stronger than the existing
// TestController_NoEventsWhenObservabilityDisabled, which only rejects
// "run.stage.*".
func TestController_NoRunOrStageEventsWhenObservabilityDisabled(t *testing.T) {
	buf := captureSlog(t)

	ctrl := happyPathFixture(t, nil)
	opts := RunOptions{
		RunID:         "run-disabled-e2e",
		Observability: config.ObservabilityConfig{Enabled: false},
	}
	result, err := ctrl.Run(context.Background(), "E07-F01-001", opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil RunResult even with observability disabled")
	}
	// Confirm the controller itself still executed the happy path — the
	// behavioural surface must be identical regardless of logging toggles.
	if result.Outcome != "completed" {
		t.Errorf("expected outcome=completed (observability toggle must not alter behaviour), got %q", result.Outcome)
	}

	for _, ev := range parseEvents(t, buf) {
		msg, _ := ev["msg"].(string)
		if strings.HasPrefix(msg, "run.") {
			t.Errorf("expected NO run.* events when Observability.Enabled=false, got %q (attrs=%v)", msg, ev)
		}
	}
}
