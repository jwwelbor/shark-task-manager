// Package runner
//
// LivenessRecorder tests for T-E40-F04-001 (foundation): the D1
// stage-boundary state machine driven through Observe, and the D3 NDJSON
// stderr renderer. No controller, no database — same-package, real-stderr-io
// tests against t.TempDir(), matching transcript_test.go's convention.
//
// Covers (see docs/plan/E40-shark-bench-workflow-benchmarking-harness/
// E40-F04-shark-run-live-progress-and-per-run-log/test-plan.md):
//   - TC-011 (stderr NDJSON half from T-E40-F04-001, plus the run.log
//     file-content half completed by T-E40-F04-002 below)
//   - TC-014 (plain-mode rendering, T-E40-F04-002)
//   - TC-015 (parent -> child -> parent cascade labeling)
//   - TC-019 (read-before-close durability — T-E40-F04-002's stage_start
//     half; the heartbeat half is added by T-E40-F04-003's ticker)
//   - TC-020 (stage pairing, decision-table rows 1/2/5/6)
//   - TC-021 (stage pairing under a non-unique (entity_key, iteration)
//     re-dispatch, decision-table row 7)
//   - TC-022 (empty projectRoot disables the file sink only)
//   - TC-023 (EACCES-class file sink failure — fail-soft, slog.Debug)
//   - TC-028 (run.log / parent-dir permission bits)
//   - TC-029 (closed JSON field set)
//
// T-E40-F04-003 (fixed 10s ticker and heartbeat cadence) adds:
//   - TC-012 (heartbeat cadence via the directly callable tick() method)
//   - TC-024 (LogPath() announced once on stderr, before any event line,
//     both jsonMode values)
//   - TC-026 (constructor signature + TTL-identifier source scan)
//   - TC-027 (concurrent Observe + tick under -race)
package runner

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"
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

// ---------------------------------------------------------------------------
// Plain-line parsing helper (shared by TC-011's file-content half, TC-014,
// TC-019, TC-022, TC-023)
// ---------------------------------------------------------------------------

// plainLineRe matches D3's fixed-column plain-text format:
//
//	<ts>  <event>  <entity_key>  key=value key=value ...
//
// Group 4 captures whatever follows entity_key VERBATIM (no whitespace
// consumed by the regex) so parsePlainLine below can itself assert the
// two-space separator D3's worked example requires between entity_key and
// the key=value block — rather than silently tolerating one space via a
// TrimSpace that would make a missing-space regression invisible to callers.
var plainLineRe = regexp.MustCompile(`^(\S+)  (\S+)  (\S+)(.*)$`)

// parsedPlainLine is a plain-text line broken into its fixed columns and its
// key=value tail.
type parsedPlainLine struct {
	ts        string
	event     string
	entityKey string
	kv        map[string]string
}

// parsePlainLine parses one plain-text line produced by renderPlainLine,
// failing the test if it does not match D3's fixed-column shape, does not use
// exactly two spaces between entity_key and the key=value block, or contains
// a malformed/double-spaced key=value token.
func parsePlainLine(t *testing.T, raw string) parsedPlainLine {
	t.Helper()
	m := plainLineRe.FindStringSubmatch(raw)
	if m == nil {
		t.Fatalf("line does not match D3's plain-text format <ts>  <event>  <entity_key>  k=v...: %q", raw)
	}
	kv := map[string]string{}
	rest := m[4]
	if rest != "" {
		if !strings.HasPrefix(rest, "  ") {
			t.Fatalf("expected exactly two spaces between entity_key and the key=value block, got %q in line %q", rest, raw)
		}
		tail := strings.TrimPrefix(rest, "  ")
		if tail == "" {
			t.Fatalf("key=value block separator present but no tokens follow: %q", raw)
		}
		for _, pair := range strings.Split(tail, " ") {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) != 2 {
				t.Fatalf("malformed key=value token %q in line %q", pair, raw)
			}
			kv[parts[0]] = parts[1]
		}
	}
	return parsedPlainLine{ts: m[1], event: m[2], entityKey: m[3], kv: kv}
}

// ---------------------------------------------------------------------------
// TC-011 file-content half (deferred from T-E40-F04-001)
// ---------------------------------------------------------------------------

// TestLiveness_TC011_FileContentHalf completes TC-011's file-content half:
// even in jsonMode=true, run.log's lines are D3's plain-text format, never
// NDJSON — "run.log in both modes" per D3's renderer table.
func TestLiveness_TC011_FileContentHalf(t *testing.T) {
	root := t.TempDir()
	rec := NewLivenessRecorder(root, "run-tc011c", "T-E40-F04-003", true, time.Now())

	// stderr output isn't asserted here (covered by TC-011's stderr half in
	// T-E40-F04-001); suppress it so it doesn't pollute test output.
	captureStderrLines(t, func() {
		rec.Observe(RunProgress{Phase: "iteration", Iteration: 1, EntityKey: "T-E40-F04-003", Status: "in_development"})
		rec.Observe(RunProgress{
			Phase: "action", Iteration: 1, EntityKey: "T-E40-F04-003", Status: "in_development",
			Action: "spawn_agent", AgentType: "developer", Provider: "anthropic",
		})
		rec.Observe(RunProgress{Phase: "iteration", Iteration: 2, EntityKey: "T-E40-F04-003", Status: "code_review"})
	})

	content, err := os.ReadFile(rec.LogPath())
	if err != nil {
		t.Fatalf("read run.log: %v", err)
	}
	rawLines := strings.Split(strings.TrimRight(string(content), "\n"), "\n")
	if len(rawLines) != 2 {
		t.Fatalf("expected 2 run.log lines, got %d: %v", len(rawLines), rawLines)
	}
	for i, raw := range rawLines {
		if strings.HasPrefix(strings.TrimSpace(raw), "{") {
			t.Errorf("run.log line %d looks like JSON, want D3 plain text: %q", i, raw)
		}
	}
	start := parsePlainLine(t, rawLines[0])
	if start.event != eventStageStart || start.entityKey != "T-E40-F04-003" {
		t.Errorf("run.log line 0 = %+v, want stage_start/T-E40-F04-003", start)
	}
	end := parsePlainLine(t, rawLines[1])
	if end.event != eventStageEnd || end.entityKey != "T-E40-F04-003" {
		t.Errorf("run.log line 1 = %+v, want stage_end/T-E40-F04-003", end)
	}
}

// ---------------------------------------------------------------------------
// TC-014: plain-mode line rendering — required fields, omitted-when-empty
// ---------------------------------------------------------------------------

// TestLiveness_TC014_PlainModeRendering drives jsonMode=false and asserts
// AC-04/AC-T1: each stderr line matches D3's fixed-column format, carries
// entity key/status/action/agent/provider/stage/total when non-empty, and
// omits any empty-valued key entirely (never key= with no value).
func TestLiveness_TC014_PlainModeRendering(t *testing.T) {
	t.Run("full_stage_all_fields_present", func(t *testing.T) {
		rec := NewLivenessRecorder(t.TempDir(), "run-tc014a", "T-E40-F04-002", false, time.Now())

		lines := captureStderrLines(t, func() {
			rec.Observe(RunProgress{Phase: "iteration", Iteration: 1, EntityKey: "T-E40-F04-002", Status: "in_development"})
			rec.Observe(RunProgress{
				Phase: "action", Iteration: 1, EntityKey: "T-E40-F04-002", Status: "in_development",
				Action: "spawn_agent", AgentType: "developer", Provider: "anthropic",
			})
			rec.Observe(RunProgress{Phase: "iteration", Iteration: 2, EntityKey: "T-E40-F04-002", Status: "code_review"})
		})
		if len(lines) != 2 {
			t.Fatalf("expected 2 emitted lines (stage_start, stage_end), got %d: %v", len(lines), lines)
		}

		start := parsePlainLine(t, lines[0])
		if start.event != eventStageStart || start.entityKey != "T-E40-F04-002" {
			t.Fatalf("line 0 = %+v, want stage_start/T-E40-F04-002", start)
		}
		if _, err := time.Parse(time.RFC3339Nano, start.ts); err != nil {
			t.Errorf("line 0 ts %q does not parse as RFC3339Nano: %v", start.ts, err)
		}
		for _, key := range []string{"status", "action", "agent", "provider", "stage", "total"} {
			if _, ok := start.kv[key]; !ok {
				t.Errorf("line 0: missing key %q (raw=%s)", key, lines[0])
			}
		}
		if start.kv["status"] != "in_development" {
			t.Errorf("status = %q, want in_development", start.kv["status"])
		}
		if start.kv["action"] != "spawn_agent" {
			t.Errorf("action = %q, want spawn_agent", start.kv["action"])
		}
		if start.kv["agent"] != "developer" {
			t.Errorf("agent = %q, want developer", start.kv["agent"])
		}
		if start.kv["provider"] != "anthropic" {
			t.Errorf("provider = %q, want anthropic", start.kv["provider"])
		}
		if _, ok := start.kv["iteration"]; ok {
			t.Errorf("plain-mode line unexpectedly carries an iteration= key (not part of D3's plain field list): %s", lines[0])
		}

		end := parsePlainLine(t, lines[1])
		if end.event != eventStageEnd || end.entityKey != "T-E40-F04-002" {
			t.Fatalf("line 1 = %+v, want stage_end/T-E40-F04-002", end)
		}

		// AC-T1's "stderr and run.log, same renderer" parenthetical: run.log
		// must carry the EXACT same lines just captured on stderr, not a
		// second independently-formatted rendering.
		fileContent, err := os.ReadFile(rec.LogPath())
		if err != nil {
			t.Fatalf("read run.log: %v", err)
		}
		fileLines := strings.Split(strings.TrimRight(string(fileContent), "\n"), "\n")
		if len(fileLines) != len(lines) {
			t.Fatalf("run.log has %d lines, want %d (matching stderr): %v", len(fileLines), len(lines), fileLines)
		}
		for i := range lines {
			if fileLines[i] != lines[i] {
				t.Errorf("run.log line %d = %q, want exact match with stderr line %q", i, fileLines[i], lines[i])
			}
		}
	})

	t.Run("no_action_phase_empty_keys_omitted", func(t *testing.T) {
		rec := NewLivenessRecorder(t.TempDir(), "run-tc014b", "PARENT", false, time.Now())

		lines := captureStderrLines(t, func() {
			rec.Observe(RunProgress{Phase: "iteration", Iteration: 1, EntityKey: "E1", Status: "s0"})
			// Closes E1's stage before it ever reached the action phase.
			rec.Observe(RunProgress{Phase: "iteration", Iteration: 1, EntityKey: "E2", Status: "s1"})
		})
		if len(lines) != 2 {
			t.Fatalf("expected 2 emitted lines (lazy stage_start, stage_end), got %d: %v", len(lines), lines)
		}

		for i, raw := range lines {
			p := parsePlainLine(t, raw)
			if _, ok := p.kv["action"]; ok {
				t.Errorf("line %d: action= present though stage never reached action phase (raw=%s)", i, raw)
			}
			if _, ok := p.kv["agent"]; ok {
				t.Errorf("line %d: agent= present though never set (raw=%s)", i, raw)
			}
			if _, ok := p.kv["provider"]; ok {
				t.Errorf("line %d: provider= present though never set (raw=%s)", i, raw)
			}
			if _, ok := p.kv["stage"]; !ok {
				t.Errorf("line %d: stage= missing (always present, even for a lazy-start stage)", i)
			}
			if _, ok := p.kv["total"]; !ok {
				t.Errorf("line %d: total= missing (always present)", i)
			}
		}
	})
}

// TestFormatElapsed drives the plain-mode duration formatter directly
// (white-box, same package), including spec.md D3's own worked example
// values (74213ms -> "1m14s", 181940ms -> "3m01s").
func TestFormatElapsed(t *testing.T) {
	tests := []struct {
		name string
		ms   int64
		want string
	}{
		{"zero", 0, "0s"},
		{"seconds_only", 14_000, "14s"},
		{"d3_example_stage_elapsed", 74213, "1m14s"},
		{"d3_example_total_elapsed", 181940, "3m01s"},
		{"exact_minute_zero_pads_seconds", 60_000, "1m00s"},
		{"hour_boundary", 3_661_000, "1h01m01s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatElapsed(time.Duration(tt.ms) * time.Millisecond)
			if got != tt.want {
				t.Errorf("formatElapsed(%dms) = %q, want %q", tt.ms, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TC-019 (partial): read-before-close per-event durability
// ---------------------------------------------------------------------------

// TestLiveness_TC019_ReadBeforeCloseDurability drives one stage through the
// action phase (which emits stage_start) and reads run.log directly from
// disk BEFORE calling closeOpenStage/Finish/Stop — proving REQ-F-010's
// per-event durability (D4's open-append-write-close) rather than a
// flush-at-end property. The heartbeat leg of TC-019 is added once the
// ticker exists (T-E40-F04-003); this task covers the stage_start half.
func TestLiveness_TC019_ReadBeforeCloseDurability(t *testing.T) {
	root := t.TempDir()
	rec := NewLivenessRecorder(root, "run-tc019", "T-E40-F04-002", true, time.Now())

	// stderr output isn't asserted here (covered by TC-011); suppress it so
	// it doesn't pollute test output.
	captureStderrLines(t, func() {
		rec.Observe(RunProgress{Phase: "iteration", Iteration: 1, EntityKey: "T-E40-F04-002", Status: "in_development"})
		rec.Observe(RunProgress{
			Phase: "action", Iteration: 1, EntityKey: "T-E40-F04-002", Status: "in_development",
			Action: "spawn_agent", AgentType: "developer", Provider: "anthropic",
		})
	})
	// Deliberately no Finish()/Stop()/closeOpenStage() call: the stage is
	// still open when we read the file below.

	content, err := os.ReadFile(rec.LogPath())
	if err != nil {
		t.Fatalf("run.log not readable before any close/flush call: %v", err)
	}
	got := string(content)
	if !strings.Contains(got, eventStageStart) {
		t.Errorf("run.log missing stage_start before close: %q", got)
	}
	if !strings.Contains(got, "T-E40-F04-002") {
		t.Errorf("run.log missing entity key: %q", got)
	}
	if !strings.Contains(got, "status=in_development") {
		t.Errorf("run.log missing status: %q", got)
	}
	if !strings.Contains(got, "agent=developer") {
		t.Errorf("run.log missing agent: %q", got)
	}
	if !strings.Contains(got, "provider=anthropic") {
		t.Errorf("run.log missing provider: %q", got)
	}
}

// ---------------------------------------------------------------------------
// LogPath(): real getter, resolved absolute path
// ---------------------------------------------------------------------------

// TestLiveness_LogPath_ResolvesAbsolutePath asserts AC-T3's "real getter, not
// a stub" requirement: LogPath() returns the exact absolute path the file
// sink writes to.
func TestLiveness_LogPath_ResolvesAbsolutePath(t *testing.T) {
	root := t.TempDir()
	rec := NewLivenessRecorder(root, "run-logpath", "TOP", true, time.Now())

	want := filepath.Join(root, ".shark", "runs", "run-logpath", "run.log")
	if got := rec.LogPath(); got != want {
		t.Errorf("LogPath() = %q, want %q", got, want)
	}
	if !filepath.IsAbs(rec.LogPath()) {
		t.Errorf("LogPath() = %q, want an absolute path", rec.LogPath())
	}
}

// ---------------------------------------------------------------------------
// TC-022: empty projectRoot — file sink disabled, no error, stderr unaffected
// ---------------------------------------------------------------------------

// TestLiveness_TC022_EmptyProjectRootDisablesFileSinkOnly asserts AC-09's
// empty-projectRoot boundary: LogPath() resolves empty, no panic/error from
// any recorder method, and stderr events still fire for every Observe call —
// mirroring NewFileJSONLExporter("")'s silent-skip semantics.
func TestLiveness_TC022_EmptyProjectRootDisablesFileSinkOnly(t *testing.T) {
	rec := NewLivenessRecorder("", "run-tc022", "TOP", true, time.Now())

	if got := rec.LogPath(); got != "" {
		t.Fatalf("LogPath() = %q, want empty for empty projectRoot", got)
	}

	lines := captureStderrLines(t, func() {
		rec.Observe(RunProgress{Phase: "iteration", Iteration: 1, EntityKey: "E1", Status: "s0"})
		rec.Observe(RunProgress{
			Phase: "action", Iteration: 1, EntityKey: "E1", Status: "s0",
			Action: "spawn_agent", AgentType: "developer", Provider: "anthropic",
		})
		rec.Observe(RunProgress{Phase: "iteration", Iteration: 2, EntityKey: "E1", Status: "s1"})
	})
	if len(lines) != 2 {
		t.Fatalf("expected 2 stderr lines despite disabled file sink, got %d: %v", len(lines), lines)
	}
	got := parseWireLines(t, lines)
	if got[0].Event != eventStageStart || got[1].Event != eventStageEnd {
		t.Fatalf("unexpected event sequence: %+v", got)
	}
}

// ---------------------------------------------------------------------------
// TC-023: file sink EACCES-class failure — fail-soft, slog.Debug evidence
// ---------------------------------------------------------------------------

// TestLiveness_TC023_EACCESFailSoft pre-creates .shark/runs/ and chmods it
// 0o555 (blocks creating the <run_id> subdirectory beneath it, matching
// transcript_test.go's/edit_service_test.go's fault-injection convention).
// Asserts REQ-F-011/REQ-N-002: every stderr event still fires, no error
// reaches the caller (Observe has no error return to begin with — the
// assertion is "no panic"), no run.log is ever created, and at least one
// slog.Debug record documents the write failure.
func TestLiveness_TC023_EACCESFailSoft(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-bit fault injection is not meaningful on windows")
	}
	if os.Getuid() == 0 {
		t.Skip("running as root; permission checks are not enforced")
	}

	drive := func(t *testing.T, root string) (stderrLines []string, slogBuf *bytes.Buffer) {
		t.Helper()
		buf := captureSlog(t)
		rec := NewLivenessRecorder(root, "run-tc023", "TOP", true, time.Now())
		lines := captureStderrLines(t, func() {
			rec.Observe(RunProgress{Phase: "iteration", Iteration: 1, EntityKey: "E1", Status: "s0"})
			rec.Observe(RunProgress{
				Phase: "action", Iteration: 1, EntityKey: "E1", Status: "s0",
				Action: "spawn_agent", AgentType: "developer", Provider: "anthropic",
			})
			rec.Observe(RunProgress{Phase: "iteration", Iteration: 2, EntityKey: "E1", Status: "s1"})
		})
		return lines, buf
	}

	assertFailSoft := func(t *testing.T, lines []string, buf *bytes.Buffer, runLogPath string) {
		t.Helper()
		if len(lines) != 2 {
			t.Fatalf("expected 2 stderr lines despite an unwritable run.log dir, got %d: %v", len(lines), lines)
		}
		if _, err := os.Stat(runLogPath); !os.IsNotExist(err) {
			t.Errorf("expected run.log NOT to be created under an unwritable dir, stat err = %v", err)
		}
		found := false
		for _, ev := range parseEvents(t, buf) {
			if msg, _ := ev["msg"].(string); strings.Contains(msg, "liveness") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected at least one slog.Debug record describing the run.log write failure")
		}
	}

	// mkdir_blocked: .shark/runs itself is unwritable, so the recorder's own
	// os.MkdirAll(".shark/runs/<run_id>") fails and os.OpenFile is never
	// reached.
	t.Run("mkdir_blocked", func(t *testing.T) {
		root := t.TempDir()
		runsDir := filepath.Join(root, ".shark", "runs")
		if err := os.MkdirAll(runsDir, 0o755); err != nil {
			t.Fatalf("setup: mkdir runs dir: %v", err)
		}
		if err := os.Chmod(runsDir, 0o555); err != nil {
			t.Fatalf("setup: chmod runs dir: %v", err)
		}
		t.Cleanup(func() {
			if err := os.Chmod(runsDir, 0o755); err != nil {
				t.Errorf("cleanup: restore runs dir perms: %v", err)
			}
		})

		lines, buf := drive(t, root)
		assertFailSoft(t, lines, buf, filepath.Join(runsDir, "run-tc023", "run.log"))
	})

	// openfile_blocked: the <run_id> directory already exists (so MkdirAll
	// succeeds — it is a no-op on an existing directory regardless of its
	// mode), but the directory itself is unwritable, so os.OpenFile(run.log,
	// O_CREATE, ...) is what fails. Exercises the sibling fail-soft branch
	// mkdir_blocked never reaches.
	t.Run("openfile_blocked", func(t *testing.T) {
		root := t.TempDir()
		runDir := filepath.Join(root, ".shark", "runs", "run-tc023")
		if err := os.MkdirAll(runDir, 0o755); err != nil {
			t.Fatalf("setup: mkdir run dir: %v", err)
		}
		if err := os.Chmod(runDir, 0o555); err != nil {
			t.Fatalf("setup: chmod run dir: %v", err)
		}
		t.Cleanup(func() {
			if err := os.Chmod(runDir, 0o755); err != nil {
				t.Errorf("cleanup: restore run dir perms: %v", err)
			}
		})

		lines, buf := drive(t, root)
		assertFailSoft(t, lines, buf, filepath.Join(runDir, "run.log"))
	})
}

// ---------------------------------------------------------------------------
// TC-028: run.log / parent-dir permission bits match the transcript
// convention
// ---------------------------------------------------------------------------

// TestLiveness_TC028_PermissionBits asserts REQ-N-005: run.log is written
// with file mode 0o644 and its parent directory with 0o755, matching
// internal/runner/transcript.go exactly (not the umask-dependent defaults of
// os.Create).
func TestLiveness_TC028_PermissionBits(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not meaningful on windows")
	}

	root := t.TempDir()
	rec := NewLivenessRecorder(root, "run-tc028", "TOP", true, time.Now())

	// stderr output isn't asserted here (covered by TC-011); suppress it so
	// it doesn't pollute test output.
	captureStderrLines(t, func() {
		rec.Observe(RunProgress{Phase: "iteration", Iteration: 1, EntityKey: "E1", Status: "s0"})
		rec.Observe(RunProgress{
			Phase: "action", Iteration: 1, EntityKey: "E1", Status: "s0",
			Action: "spawn_agent", AgentType: "developer", Provider: "anthropic",
		})
	})

	path := rec.LogPath()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat run.log: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Errorf("run.log perm = %o, want 0644", got)
	}

	parent, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat run.log parent dir: %v", err)
	}
	if got := parent.Mode().Perm(); got != 0o755 {
		t.Errorf("run.log parent dir perm = %o, want 0755", got)
	}
}

// ---------------------------------------------------------------------------
// TC-012: heartbeat cadence via the directly callable tick() method
// ---------------------------------------------------------------------------

// TestLiveness_TC012_HeartbeatCadenceViaTick drives the unexported tick()
// method directly (AC-T2's hard requirement: a separately callable method,
// not a real 10s sleep) and asserts REQ-F-006's cadence: every tick against
// an open stage emits exactly one heartbeat line carrying that stage's
// identity, with no window producing zero heartbeats.
func TestLiveness_TC012_HeartbeatCadenceViaTick(t *testing.T) {
	rec := NewLivenessRecorder(t.TempDir(), "run-tc012", "T-E40-F04-003", true, time.Now())

	lines := captureStderrLines(t, func() {
		rec.Observe(RunProgress{Phase: "iteration", Iteration: 1, EntityKey: "T-E40-F04-003", Status: "in_development"})
		rec.Observe(RunProgress{
			Phase: "action", Iteration: 1, EntityKey: "T-E40-F04-003", Status: "in_development",
			Action: "spawn_agent", AgentType: "developer", Provider: "anthropic",
		})
		// Three simulated 10s windows, driven directly via tick() — no real
		// sleep, per AC-T2/TC-012's caller-path contract.
		rec.tick()
		rec.tick()
		rec.tick()
	})
	// 1 stage_start (from the action phase) + 3 heartbeats.
	if len(lines) != 4 {
		t.Fatalf("expected 4 emitted lines (stage_start + 3 heartbeats), got %d: %v", len(lines), lines)
	}

	got := parseWireLines(t, lines)
	if got[0].Event != eventStageStart {
		t.Fatalf("line 0 = %+v, want stage_start", got[0])
	}
	for i := 1; i <= 3; i++ {
		l := got[i]
		if l.Event != "heartbeat" {
			t.Errorf("line %d event = %q, want heartbeat", i, l.Event)
		}
		if l.EntityKey != "T-E40-F04-003" || l.Iteration != 1 || l.Status != "in_development" {
			t.Errorf("line %d identity fields wrong: %+v", i, l)
		}
		if l.Action != "spawn_agent" || l.AgentType != "developer" || l.Provider != "anthropic" {
			t.Errorf("line %d action fields wrong: %+v", i, l)
		}
	}
}

// TestLiveness_TC012_NoOpenStageHeartbeat covers AC-T3 and spec.md's
// "decisions the implementer does not get to make" table: a tick with no
// open stage (before the first iteration, or during a cascade lookup gap)
// still emits a heartbeat carrying the TOP-LEVEL key, iteration 0, and empty
// status/action, rather than going silent in the exact window a stall would
// otherwise be invisible.
func TestLiveness_TC012_NoOpenStageHeartbeat(t *testing.T) {
	rec := NewLivenessRecorder(t.TempDir(), "run-tc012b", "TOP-LEVEL", true, time.Now())

	lines := captureStderrLines(t, func() {
		rec.tick()
	})
	if len(lines) != 1 {
		t.Fatalf("expected 1 emitted line, got %d: %v", len(lines), lines)
	}

	raw := parseRawLines(t, lines)[0]
	got := parseWireLines(t, lines)[0]

	if got.Event != "heartbeat" {
		t.Errorf("event = %q, want heartbeat", got.Event)
	}
	if got.EntityKey != "TOP-LEVEL" {
		t.Errorf("entity_key = %q, want top-level key %q", got.EntityKey, "TOP-LEVEL")
	}
	if got.Iteration != 0 {
		t.Errorf("iteration = %d, want 0", got.Iteration)
	}
	if got.Status != "" {
		t.Errorf("status = %q, want empty", got.Status)
	}
	if got.Action != "" {
		t.Errorf("action = %q, want empty", got.Action)
	}
	if _, ok := raw["agent_type"]; ok {
		t.Errorf("agent_type key present though never set: %s", lines[0])
	}
	if _, ok := raw["provider"]; ok {
		t.Errorf("provider key present though never set: %s", lines[0])
	}
}

// ---------------------------------------------------------------------------
// TC-024: run.log path printed on stderr exactly once, before the first
// event, in both modes
// ---------------------------------------------------------------------------

// TestLiveness_TC024_LogPathAnnouncedOnceBeforeFirstEvent drives Start() —
// AC-T4's chosen entrypoint per test-plan.md's "Recommendation: move the
// LogPath() print into Start()" — followed by an Observe sequence, in both
// jsonMode values, and asserts REQ-F-008/AC-10: exactly one stderr line
// carries the absolute LogPath() value, and it precedes every event line.
func TestLiveness_TC024_LogPathAnnouncedOnceBeforeFirstEvent(t *testing.T) {
	for _, jsonMode := range []bool{true, false} {
		name := "json_mode"
		if !jsonMode {
			name = "plain_mode"
		}
		t.Run(name, func(t *testing.T) {
			rec := NewLivenessRecorder(t.TempDir(), "run-tc024", "TOP", jsonMode, time.Now())

			lines := captureStderrLines(t, func() {
				rec.Start()
				rec.Observe(RunProgress{Phase: "iteration", Iteration: 1, EntityKey: "TOP", Status: "s0"})
				rec.Observe(RunProgress{Phase: "action", Iteration: 1, EntityKey: "TOP", Status: "s0", Action: "spawn_agent"})
				rec.Observe(RunProgress{Phase: "iteration", Iteration: 2, EntityKey: "TOP", Status: "s1"})
			})
			rec.Stop()

			if len(lines) != 3 {
				t.Fatalf("expected 3 lines total (path + stage_start + stage_end), got %d: %v", len(lines), lines)
			}
			if !strings.Contains(lines[0], rec.LogPath()) {
				t.Fatalf("first stderr line = %q, want it to contain LogPath() %q", lines[0], rec.LogPath())
			}

			count := 0
			for _, l := range lines {
				if strings.Contains(l, rec.LogPath()) {
					count++
				}
			}
			if count != 1 {
				t.Fatalf("expected exactly 1 line containing LogPath(), got %d: %v", count, lines)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TC-026: heartbeat interval is a literal constant, independent of any
// TTL/config input
// ---------------------------------------------------------------------------

// TestLiveness_TC026_FixedIntervalNoTTLInput asserts AC-T1/REQ-N-001:
// NewLivenessRecorder's signature carries no time.Duration parameter, and
// liveness.go's source text never references claim/lease TTL identifiers —
// the interval must come from nowhere but a literal 10 * time.Second.
func TestLiveness_TC026_FixedIntervalNoTTLInput(t *testing.T) {
	fnType := reflect.TypeOf(NewLivenessRecorder)
	if fnType.NumIn() != 5 {
		t.Fatalf("NewLivenessRecorder has %d params, want 5 (projectRoot, runID, topLevelKey, jsonMode, start)", fnType.NumIn())
	}
	durationType := reflect.TypeOf(time.Duration(0))
	for i := 0; i < fnType.NumIn(); i++ {
		if fnType.In(i) == durationType {
			t.Errorf("param %d is time.Duration — REQ-N-001 forbids a configurable interval", i)
		}
	}

	src, err := os.ReadFile("liveness.go")
	if err != nil {
		t.Fatalf("read liveness.go: %v", err)
	}
	text := string(src)
	for _, forbidden := range []string{"TTL(", "claim_ttl_seconds", "DefaultClaimTTL"} {
		if strings.Contains(text, forbidden) {
			t.Errorf("liveness.go source contains forbidden TTL identifier %q (REQ-N-001)", forbidden)
		}
	}
	if !strings.Contains(text, "10 * time.Second") {
		t.Errorf("liveness.go source does not contain the literal heartbeat interval %q (AC-T1)", "10 * time.Second")
	}
}

// ---------------------------------------------------------------------------
// TC-027: concurrent Observe + tick under the race detector
// ---------------------------------------------------------------------------

// TestLiveness_TC027_ConcurrentObserveAndTick drives Observe from many
// goroutines concurrently with direct tick() calls (REQ-N-003) and asserts
// no torn writes: every captured stderr line still parses as valid JSON and
// carries an event within the closed enum. The mutex-guard property itself
// is enforced by running this test under `go test -race`.
func TestLiveness_TC027_ConcurrentObserveAndTick(t *testing.T) {
	rec := NewLivenessRecorder(t.TempDir(), "run-tc027", "TOP", true, time.Now())

	const n = 25
	lines := captureStderrLines(t, func() {
		var wg sync.WaitGroup
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				rec.Observe(RunProgress{Phase: "iteration", Iteration: i, EntityKey: "E1", Status: "s0"})
				rec.Observe(RunProgress{Phase: "action", Iteration: i, EntityKey: "E1", Status: "s0", Action: "spawn_agent"})
			}(i)
		}
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				rec.tick()
			}()
		}
		wg.Wait()
	})

	// parseWireLines fails the test immediately on any line that doesn't
	// parse as valid JSON — that is what a torn/interleaved write would
	// produce, and it can only NOT happen if every emitLine call is
	// serialized by r.mu (REQ-N-003).
	got := parseWireLines(t, lines)
	for i, l := range got {
		if l.Event != eventStageStart && l.Event != eventStageEnd && l.Event != "heartbeat" {
			t.Errorf("line %d: event %q outside the closed 3-value enum", i, l.Event)
		}
	}
}

// ---------------------------------------------------------------------------
// Stop(): safe without a prior Start(), and idempotent
// ---------------------------------------------------------------------------

// TestLiveness_StopSafeWithoutStart asserts the task's "Notes for Agent"
// requirement: Stop() must be callable unconditionally, even when Start()
// was never called — T-E40-F04-004's teardown does `rec.Stop();
// rec.Finish(...)` regardless of whether the run ever reached Start().
func TestLiveness_StopSafeWithoutStart(t *testing.T) {
	rec := NewLivenessRecorder(t.TempDir(), "run-stop-safe", "TOP", true, time.Now())
	rec.Stop() // must not panic or block
}

// TestLiveness_StopIsIdempotent asserts Stop() can be called more than once
// without panicking (e.g. closing an already-closed channel).
func TestLiveness_StopIsIdempotent(t *testing.T) {
	rec := NewLivenessRecorder(t.TempDir(), "run-stop-idem", "TOP", true, time.Now())
	rec.Start()
	rec.Stop()
	rec.Stop()
}
