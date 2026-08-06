// Package runner
//
// LivenessRecorder tests for T-E40-F04-001 (foundation): the D1
// stage-boundary state machine driven through Observe, and the D3 NDJSON
// stderr renderer. No controller, no database — same-package, real-stderr-io
// tests against t.TempDir(), matching transcript_test.go's convention.
//
// Covers (see docs/plan/E40-shark-bench-workflow-benchmarking-harness/
// E40-F04-shark-run-live-progress-and-per-run-log/test-plan.md):
//   - TC-011 (stderr-capture half only — the run.log half lands in
//     T-E40-F04-002)
//   - TC-015 (parent -> child -> parent cascade labeling)
//   - TC-020 (stage pairing, decision-table rows 1/2/5/6)
//   - TC-021 (stage pairing under a non-unique (entity_key, iteration)
//     re-dispatch, decision-table row 7)
//   - TC-029 (closed JSON field set)
package runner

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Shared test helpers
// ---------------------------------------------------------------------------

// wireLine mirrors spec.md D3's NDJSON schema. It is defined independently of
// production's ndjsonLine type (rather than reusing it) so that a typo'd json
// tag in the implementation cannot pass this test merely by agreeing with
// itself.
type wireLine struct {
	TS             string `json:"ts"`
	RunID          string `json:"run_id"`
	Event          string `json:"event"`
	EntityKey      string `json:"entity_key"`
	Iteration      int    `json:"iteration"`
	Status         string `json:"status"`
	Action         string `json:"action"`
	AgentType      string `json:"agent_type"`
	Provider       string `json:"provider"`
	StageElapsedMs int64  `json:"stage_elapsed_ms"`
	TotalElapsedMs int64  `json:"total_elapsed_ms"`
}

// captureStderrLines redirects os.Stderr for the duration of fn and returns
// the captured output split into non-empty lines. Mirrors the project's
// existing captureStderrOutput helper (internal/cli/commands/
// next_unresolved_placeholders_test.go), adapted to return lines directly
// since callers here parse each line as NDJSON.
func captureStderrLines(t *testing.T, fn func()) []string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w
	defer func() { os.Stderr = old }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}
	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, readErr := r.Read(buf)
		sb.Write(buf[:n])
		if readErr != nil {
			break
		}
	}
	raw := strings.TrimRight(sb.String(), "\n")
	if raw == "" {
		return nil
	}
	return strings.Split(raw, "\n")
}

// parseWireLines unmarshals each captured line into a wireLine, failing the
// test immediately if any line is not valid JSON.
func parseWireLines(t *testing.T, lines []string) []wireLine {
	t.Helper()
	out := make([]wireLine, len(lines))
	for i, raw := range lines {
		if err := json.Unmarshal([]byte(raw), &out[i]); err != nil {
			t.Fatalf("line %d: invalid JSON (%v): %s", i, err, raw)
		}
	}
	return out
}

// parseRawLines unmarshals each captured line into a raw key set, for tests
// that assert key presence/absence rather than field values.
func parseRawLines(t *testing.T, lines []string) []map[string]json.RawMessage {
	t.Helper()
	out := make([]map[string]json.RawMessage, len(lines))
	for i, raw := range lines {
		var m map[string]json.RawMessage
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			t.Fatalf("line %d: invalid JSON (%v): %s", i, err, raw)
		}
		out[i] = m
	}
	return out
}

// pairResult is one matched (stage_start, stage_end) pair.
type pairResult struct {
	start wireLine
	end   wireLine
}

// pairStages implements REQ-F-009's pairing rule directly: scanning in
// stream order, each stage_end pairs with the nearest preceding unpaired
// stage_start (a stack). It fails the test if any stage_end has no preceding
// unpaired stage_start, or if any stage_start is left unpaired at the end —
// both are AC-08 violations. heartbeat lines (none emitted by this task's
// Observe) are ignored; any other event value fails immediately since D3's
// enum is closed to exactly three values.
func pairStages(t *testing.T, lines []wireLine) []pairResult {
	t.Helper()
	var stack []wireLine
	var pairs []pairResult
	for i, l := range lines {
		switch l.Event {
		case eventStageStart:
			stack = append(stack, l)
		case eventStageEnd:
			if len(stack) == 0 {
				t.Fatalf("line %d: unpaired stage_end with no preceding stage_start: %+v", i, l)
				continue
			}
			start := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			pairs = append(pairs, pairResult{start: start, end: l})
		case "heartbeat":
			// Not emitted by this task (no ticker yet); tolerated here so
			// this helper stays reusable once T-E40-F04-003 lands.
		default:
			t.Fatalf("line %d: event %q outside the closed 3-value enum", i, l.Event)
		}
	}
	if len(stack) != 0 {
		t.Fatalf("unpaired stage_start(s) remain: %+v", stack)
	}
	return pairs
}

// ---------------------------------------------------------------------------
// TC-011: NDJSON schema — field presence and closed enum (stderr half)
// ---------------------------------------------------------------------------

// TestLiveness_TC011_SchemaFieldPresence drives one full stage (reaches the
// action phase) and asserts every AC-01/AC-01-schema field spec.md D3
// requires: ts (RFC3339Nano, UTC), run_id, event, entity_key, iteration,
// status, action, agent_type, provider, stage_elapsed_ms, total_elapsed_ms —
// and that no line ever carries a "phase" key.
func TestLiveness_TC011_SchemaFieldPresence(t *testing.T) {
	start := time.Now()
	rec := NewLivenessRecorder(t.TempDir(), "run-tc011", "T-E40-F04-003", true, start)

	lines := captureStderrLines(t, func() {
		rec.Observe(RunProgress{Phase: "iteration", Iteration: 1, EntityKey: "T-E40-F04-003", Status: "in_development"})
		rec.Observe(RunProgress{
			Phase: "action", Iteration: 1, EntityKey: "T-E40-F04-003", Status: "in_development",
			Action: "spawn_agent", AgentType: "developer", Provider: "anthropic",
		})
		// Closes the stage above; also proves the recorder never emits
		// anything for the "iteration" phase itself.
		rec.Observe(RunProgress{Phase: "iteration", Iteration: 2, EntityKey: "T-E40-F04-003", Status: "code_review"})
	})
	if len(lines) != 2 {
		t.Fatalf("expected 2 emitted lines (stage_start, stage_end), got %d: %v", len(lines), lines)
	}

	raw := parseRawLines(t, lines)
	for i, m := range raw {
		for _, key := range []string{"ts", "run_id", "event", "entity_key", "iteration", "status", "action", "agent_type", "provider", "stage_elapsed_ms", "total_elapsed_ms"} {
			if _, ok := m[key]; !ok {
				t.Errorf("line %d: missing required key %q (raw=%s)", i, key, lines[i])
			}
		}
		if _, ok := m["phase"]; ok {
			t.Errorf("line %d: unexpected \"phase\" key present (D3: phase is deliberately absent): %s", i, lines[i])
		}
	}

	got := parseWireLines(t, lines)

	// Line 0: stage_start, enriched by the action phase.
	if got[0].Event != eventStageStart {
		t.Errorf("line 0 event = %q, want %q", got[0].Event, eventStageStart)
	}
	if got[0].RunID != "run-tc011" {
		t.Errorf("line 0 run_id = %q, want %q", got[0].RunID, "run-tc011")
	}
	if got[0].EntityKey != "T-E40-F04-003" || got[0].Iteration != 1 || got[0].Status != "in_development" {
		t.Errorf("line 0 identity fields wrong: %+v", got[0])
	}
	if got[0].Action != "spawn_agent" || got[0].AgentType != "developer" || got[0].Provider != "anthropic" {
		t.Errorf("line 0 action fields wrong: %+v", got[0])
	}
	if _, err := time.Parse(time.RFC3339Nano, got[0].TS); err != nil {
		t.Errorf("line 0 ts %q does not parse as RFC3339Nano: %v", got[0].TS, err)
	}
	if !strings.HasSuffix(got[0].TS, "Z") {
		t.Errorf("line 0 ts %q is not UTC (missing trailing Z)", got[0].TS)
	}

	// Line 1: stage_end for the same stage, carrying the same identity.
	if got[1].Event != eventStageEnd {
		t.Errorf("line 1 event = %q, want %q", got[1].Event, eventStageEnd)
	}
	if got[1].EntityKey != "T-E40-F04-003" || got[1].Iteration != 1 || got[1].Status != "in_development" {
		t.Errorf("line 1 identity fields wrong: %+v", got[1])
	}

	// Negative: no event value ever appears outside the closed 3-value set.
	for i, l := range got {
		if l.Event != eventStageStart && l.Event != eventStageEnd && l.Event != "heartbeat" {
			t.Errorf("line %d: event %q outside D3's closed enum", i, l.Event)
		}
	}
}

// TestLiveness_TC011_UnsetAgentProviderOmitted covers D3's other explicit
// requirement: when a stage never reaches the action phase (closed by lazy
// stage_start immediately before stage_end), agent_type/provider are OMITTED
// from the JSON — not present as empty strings — and action is "" (present,
// per D3's "always; \"\" before the action phase").
func TestLiveness_TC011_UnsetAgentProviderOmitted(t *testing.T) {
	rec := NewLivenessRecorder(t.TempDir(), "run-tc011b", "PARENT", true, time.Now())

	lines := captureStderrLines(t, func() {
		rec.Observe(RunProgress{Phase: "iteration", Iteration: 1, EntityKey: "E1", Status: "s0"})
		// Closes E1's stage before it ever reached the action phase.
		rec.Observe(RunProgress{Phase: "iteration", Iteration: 1, EntityKey: "E2", Status: "s1"})
	})
	if len(lines) != 2 {
		t.Fatalf("expected 2 emitted lines (lazy stage_start, stage_end), got %d: %v", len(lines), lines)
	}

	raw := parseRawLines(t, lines)
	for i, m := range raw {
		if _, ok := m["agent_type"]; ok {
			t.Errorf("line %d: agent_type key present though never set (raw=%s)", i, lines[i])
		}
		if _, ok := m["provider"]; ok {
			t.Errorf("line %d: provider key present though never set (raw=%s)", i, lines[i])
		}
		var action string
		if err := json.Unmarshal(m["action"], &action); err != nil {
			t.Fatalf("line %d: action key missing or not a string: %v", i, err)
		}
		if action != "" {
			t.Errorf("line %d: action = %q, want empty (never reached action phase)", i, action)
		}
	}

	got := parseWireLines(t, lines)
	if got[0].Event != eventStageStart || got[1].Event != eventStageEnd {
		t.Fatalf("expected [stage_start, stage_end], got [%s, %s]", got[0].Event, got[1].Event)
	}
	if got[0].EntityKey != "E1" || got[1].EntityKey != "E1" {
		t.Errorf("lazy-start pair should both carry E1's key, got start=%s end=%s", got[0].EntityKey, got[1].EntityKey)
	}
}

// ---------------------------------------------------------------------------
// TC-015: cascade parent -> child -> parent labeling (state transition)
// ---------------------------------------------------------------------------

// TestLiveness_TC015_CascadeParentChildParentLabeling drives the sequence
// controller.go's sequential cascade loop actually produces (D1): parent
// iteration -> parent action -> child iteration -> child action -> parent's
// next iteration. It asserts REQ-F-005: every event tied to the child's
// stage carries the child's key, never the parent's, and that the parent's
// own cascade-stage stage_end fires exactly when the child's iteration
// arrives (D1's stated design point — this is the boundary event, not a
// labeling defect). Heartbeat coverage during the cascade window is added by
// T-E40-F04-003 (TC-012); this task has no ticker yet.
func TestLiveness_TC015_CascadeParentChildParentLabeling(t *testing.T) {
	const parent = "E40-F04"
	const child = "T-E40-F04-002"

	rec := NewLivenessRecorder(t.TempDir(), "run-tc015", parent, true, time.Now())

	lines := captureStderrLines(t, func() {
		rec.Observe(RunProgress{Phase: "iteration", Iteration: 1, EntityKey: parent, Status: "ready_for_cascade"})
		rec.Observe(RunProgress{Phase: "action", Iteration: 1, EntityKey: parent, Status: "ready_for_cascade", Action: "cascade"})
		rec.Observe(RunProgress{Phase: "iteration", Iteration: 1, EntityKey: child, Status: "todo"})
		rec.Observe(RunProgress{Phase: "action", Iteration: 1, EntityKey: child, Status: "todo", Action: "spawn_agent", AgentType: "developer", Provider: "anthropic"})
		rec.Observe(RunProgress{Phase: "iteration", Iteration: 2, EntityKey: parent, Status: "in_progress"})
	})

	got := parseWireLines(t, lines)
	if len(got) != 4 {
		t.Fatalf("expected 4 emitted lines, got %d: %+v", len(got), got)
	}

	// Line 0: parent's stage_start for its cascade stage.
	if got[0].Event != eventStageStart || got[0].EntityKey != parent {
		t.Errorf("line 0 = %+v, want parent stage_start", got[0])
	}
	// Line 1: parent's stage_end — the boundary event, fired exactly when the
	// child's iteration(1) arrives. Correctly labeled with the PARENT's key
	// (it is the parent's own stage ending), per D1.
	if got[1].Event != eventStageEnd || got[1].EntityKey != parent {
		t.Errorf("line 1 = %+v, want parent stage_end", got[1])
	}
	// Everything from here to (not including) the parent's next iteration
	// must carry the CHILD's key — this is REQ-F-005 and the Finding-1 defect
	// this feature exists to fix.
	if got[2].Event != eventStageStart || got[2].EntityKey != child {
		t.Errorf("line 2 = %+v, want child stage_start (REQ-F-005)", got[2])
	}
	if got[3].Event != eventStageEnd || got[3].EntityKey != child {
		t.Errorf("line 3 = %+v, want child stage_end (REQ-F-005)", got[3])
	}

	for i, l := range got[2:] {
		if l.EntityKey == parent {
			t.Errorf("line %d carries the PARENT's key during the child's open window: %+v", i+2, l)
		}
	}
}

// ---------------------------------------------------------------------------
// TC-020: stage pairing — decision-table rows 1/2/5/6
// ---------------------------------------------------------------------------

// TestLiveness_TC020_PairingRows drives each of the AC-08 decision table's
// rows 1, 2, 5, and 6 and asserts REQ-F-009: pairing every stage_end with the
// nearest preceding unpaired stage_start leaves no unpaired stage_end.
func TestLiveness_TC020_PairingRows(t *testing.T) {
	t.Run("row1_reached_action_closed_by_next_iteration_same_entity", func(t *testing.T) {
		rec := NewLivenessRecorder(t.TempDir(), "run-r1", "TOP", true, time.Now())
		lines := captureStderrLines(t, func() {
			rec.Observe(RunProgress{Phase: "iteration", Iteration: 1, EntityKey: "A", Status: "s0"})
			rec.Observe(RunProgress{Phase: "action", Iteration: 1, EntityKey: "A", Status: "s0", Action: "spawn_agent"})
			rec.Observe(RunProgress{Phase: "iteration", Iteration: 2, EntityKey: "A", Status: "s1"})
		})
		got := parseWireLines(t, lines)
		pairs := pairStages(t, got)
		if len(pairs) != 1 {
			t.Fatalf("expected 1 pair, got %d: %+v", len(pairs), pairs)
		}
		if pairs[0].start.Action != "spawn_agent" {
			t.Errorf("expected stage_end to pair with its OWN stage_start (action=spawn_agent), got %+v", pairs[0])
		}
	})

	t.Run("row2_no_action_closed_by_next_iteration_same_entity_lazy_start", func(t *testing.T) {
		rec := NewLivenessRecorder(t.TempDir(), "run-r2", "TOP", true, time.Now())
		lines := captureStderrLines(t, func() {
			rec.Observe(RunProgress{Phase: "iteration", Iteration: 1, EntityKey: "A", Status: "s0"})
			rec.Observe(RunProgress{Phase: "iteration", Iteration: 2, EntityKey: "A", Status: "s1"})
		})
		got := parseWireLines(t, lines)
		if len(got) != 2 || got[0].Event != eventStageStart || got[1].Event != eventStageEnd {
			t.Fatalf("expected [lazy stage_start, stage_end], got %+v", got)
		}
		pairs := pairStages(t, got)
		if len(pairs) != 1 {
			t.Fatalf("expected 1 pair, got %d: %+v", len(pairs), pairs)
		}
		if pairs[0].start.Action != "" {
			t.Errorf("lazy stage_start should have action=\"\" (never reached action phase), got %q", pairs[0].start.Action)
		}
	})

	t.Run("row5_reached_action_closed_by_run_end", func(t *testing.T) {
		rec := NewLivenessRecorder(t.TempDir(), "run-r5", "TOP", true, time.Now())
		lines := captureStderrLines(t, func() {
			rec.Observe(RunProgress{Phase: "iteration", Iteration: 1, EntityKey: "A", Status: "s0"})
			rec.Observe(RunProgress{Phase: "action", Iteration: 1, EntityKey: "A", Status: "s0", Action: "spawn_agent"})
			rec.closeOpenStage() // simulates the "run end" row ahead of Finish() (T-E40-F04-004)
		})
		got := parseWireLines(t, lines)
		pairs := pairStages(t, got)
		if len(pairs) != 1 {
			t.Fatalf("expected 1 pair, got %d: %+v", len(pairs), pairs)
		}
		if pairs[0].start.Action != "spawn_agent" {
			t.Errorf("expected the final stage_end to pair with its own stage_start, got %+v", pairs[0])
		}
	})

	t.Run("row6_no_action_closed_by_run_end_lazy_start", func(t *testing.T) {
		rec := NewLivenessRecorder(t.TempDir(), "run-r6", "TOP", true, time.Now())
		lines := captureStderrLines(t, func() {
			rec.Observe(RunProgress{Phase: "iteration", Iteration: 1, EntityKey: "A", Status: "s0"})
			rec.closeOpenStage()
		})
		got := parseWireLines(t, lines)
		if len(got) != 2 || got[0].Event != eventStageStart || got[1].Event != eventStageEnd {
			t.Fatalf("expected [lazy stage_start, stage_end] at run end, got %+v", got)
		}
		pairs := pairStages(t, got)
		if len(pairs) != 1 {
			t.Fatalf("expected 1 pair, got %d: %+v", len(pairs), pairs)
		}
		if pairs[0].start.Action != "" {
			t.Errorf("lazy stage_start at run end should have action=\"\", got %q", pairs[0].start.Action)
		}
	})
}

// ---------------------------------------------------------------------------
// TC-021: pairing under a non-unique (entity_key, iteration) re-dispatch
// ---------------------------------------------------------------------------

// TestLiveness_TC021_NonUniqueEntityIterationRedispatch drives two full
// cascade passes over the SAME child entity key, with the child's iteration
// counter restarting at 1 in the second pass — the exact Q004 shape spec.md
// AC-08 row 7 describes as "not a unique key." It asserts the recorder's
// pairing is stream-order-based, not (entity_key, iteration)-keyed: each
// pass's stage_end must pair with THAT pass's own stage_start, proven by a
// distinguishing agent_type value per pass that must not cross-contaminate.
func TestLiveness_TC021_NonUniqueEntityIterationRedispatch(t *testing.T) {
	const parent = "PARENT"
	const child = "CHILD"

	rec := NewLivenessRecorder(t.TempDir(), "run-tc021", parent, true, time.Now())

	lines := captureStderrLines(t, func() {
		// Pass 1: parent dispatches the child once.
		rec.Observe(RunProgress{Phase: "iteration", Iteration: 1, EntityKey: parent, Status: "s0"})
		rec.Observe(RunProgress{Phase: "action", Iteration: 1, EntityKey: parent, Status: "s0", Action: "cascade"})
		rec.Observe(RunProgress{Phase: "iteration", Iteration: 1, EntityKey: child, Status: "cs0"})
		rec.Observe(RunProgress{Phase: "action", Iteration: 1, EntityKey: child, Status: "cs0", Action: "spawn_agent", AgentType: "dev-pass1"})
		rec.Observe(RunProgress{Phase: "iteration", Iteration: 2, EntityKey: parent, Status: "s1"})

		// Pass 2: a SECOND cascade dispatch of the same child; its iteration
		// counter restarts at 1, colliding with pass 1's (entity_key, iteration).
		rec.Observe(RunProgress{Phase: "action", Iteration: 2, EntityKey: parent, Status: "s1", Action: "cascade"})
		rec.Observe(RunProgress{Phase: "iteration", Iteration: 1, EntityKey: child, Status: "cs0"})
		rec.Observe(RunProgress{Phase: "action", Iteration: 1, EntityKey: child, Status: "cs0", Action: "spawn_agent", AgentType: "dev-pass2"})
		rec.Observe(RunProgress{Phase: "iteration", Iteration: 3, EntityKey: parent, Status: "s2"})

		rec.closeOpenStage() // flush the trailing open parent slot
	})

	got := parseWireLines(t, lines)
	pairs := pairStages(t, got)
	if len(pairs) != 5 {
		t.Fatalf("expected 5 pairs (parent#1, child-pass1, parent#2, child-pass2, parent#3), got %d: %+v", len(pairs), pairs)
	}

	childPairs := make([]pairResult, 0, 2)
	for _, p := range pairs {
		if p.start.EntityKey == child {
			childPairs = append(childPairs, p)
		}
	}
	if len(childPairs) != 2 {
		t.Fatalf("expected 2 child pairs, got %d: %+v", len(childPairs), childPairs)
	}

	// Both child pairs share the same (entity_key, iteration) composite —
	// the collision this test exists to exercise.
	for i, p := range childPairs {
		if p.start.Iteration != 1 {
			t.Errorf("child pair %d: start.Iteration = %d, want 1 (both passes restart at 1)", i, p.start.Iteration)
		}
		if p.start.EntityKey != p.end.EntityKey || p.start.Iteration != p.end.Iteration {
			t.Errorf("child pair %d: start/end identity mismatch: %+v", i, p)
		}
	}

	// The distinguishing field proves the two passes were NOT merged: an
	// implementation keying its pairing state by (entity_key, iteration)
	// would either cross-pair or lose one pass's agent_type.
	if childPairs[0].start.AgentType != "dev-pass1" {
		t.Errorf("first child pair agent_type = %q, want %q", childPairs[0].start.AgentType, "dev-pass1")
	}
	if childPairs[1].start.AgentType != "dev-pass2" {
		t.Errorf("second child pair agent_type = %q, want %q", childPairs[1].start.AgentType, "dev-pass2")
	}
	if childPairs[0].start.AgentType == childPairs[1].start.AgentType {
		t.Fatalf("both child pairs report the same agent_type — passes were merged")
	}
}

// ---------------------------------------------------------------------------
// TC-029: closed JSON field set — no prompt/output/credential leakage
// ---------------------------------------------------------------------------

// TestLiveness_TC029_ClosedFieldSet drives a sequence that reaches the action
// phase (D3: "where a prompt-body leak would most plausibly be introduced")
// and asserts every emitted line's key set is a subset of D3's closed
// 11-field list.
func TestLiveness_TC029_ClosedFieldSet(t *testing.T) {
	allowed := map[string]bool{
		"ts": true, "run_id": true, "event": true, "entity_key": true,
		"iteration": true, "status": true, "action": true,
		"agent_type": true, "provider": true,
		"stage_elapsed_ms": true, "total_elapsed_ms": true,
	}

	rec := NewLivenessRecorder(t.TempDir(), "run-tc029", "TOP", true, time.Now())
	lines := captureStderrLines(t, func() {
		rec.Observe(RunProgress{Phase: "iteration", Iteration: 1, EntityKey: "E1", Status: "in_development"})
		rec.Observe(RunProgress{
			Phase: "action", Iteration: 1, EntityKey: "E1", Status: "in_development",
			Action: "spawn_agent", AgentType: "developer", Provider: "anthropic",
		})
		rec.Observe(RunProgress{Phase: "iteration", Iteration: 2, EntityKey: "E1", Status: "code_review"})
	})
	if len(lines) == 0 {
		t.Fatal("expected at least one emitted line")
	}

	for i, m := range parseRawLines(t, lines) {
		for key := range m {
			if !allowed[key] {
				t.Errorf("line %d: unexpected key %q outside D3's closed field set (raw=%s)", i, key, lines[i])
			}
		}
	}
}
