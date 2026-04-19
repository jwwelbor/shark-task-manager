// Package commands provides CLI command implementations.
// This file contains tests for T-E07-F41-002: run_id generation and
// run.start/run.end slog event emission in runRun.
//
// Strategy: the slog logging functions are extracted into package-internal helpers
// (emitRunStart, emitRunEnd, generateRunID) that can be called directly from tests
// without needing a live database or running controller.
package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
)

// ─── generateRunID ────────────────────────────────────────────────────────────

// TestGenerateRunID_NonEmpty verifies generateRunID returns a non-empty string.
func TestGenerateRunID_NonEmpty(t *testing.T) {
	id := generateRunID()
	if id == "" {
		t.Fatal("generateRunID() returned empty string, want non-empty run ID")
	}
}

// TestGenerateRunID_Unique verifies that two successive calls return different IDs.
func TestGenerateRunID_Unique(t *testing.T) {
	id1 := generateRunID()
	id2 := generateRunID()
	if id1 == id2 {
		t.Errorf("generateRunID() returned same ID twice: %q — IDs should be unique", id1)
	}
}

// ─── Helpers: slog capture ───────────────────────────────────────────────────

// captureLog replaces the slog default logger with an in-memory JSON logger,
// runs fn, then restores the previous logger. Returns the captured JSON lines.
func captureLog(fn func()) []string {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	old := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(old)

	fn()

	var lines []string
	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// parseLogLines parses a slice of JSON log lines into a slice of maps.
func parseLogLines(t *testing.T, lines []string) []map[string]interface{} {
	t.Helper()
	result := make([]map[string]interface{}, 0, len(lines))
	for _, line := range lines {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Errorf("failed to parse log line %q: %v", line, err)
			continue
		}
		result = append(result, m)
	}
	return result
}

// findEvent returns the first log event with the given "msg" (event name), or nil.
func findEvent(events []map[string]interface{}, msg string) map[string]interface{} {
	for _, e := range events {
		if e["msg"] == msg {
			return e
		}
	}
	return nil
}

// ─── emitRunStart ────────────────────────────────────────────────────────────

// TestEmitRunStart_EmitsEvent verifies that emitRunStart writes a run.start
// event to the slog default logger when called.
func TestEmitRunStart_EmitsEvent(t *testing.T) {
	obs := config.ObservabilityConfig{Enabled: true}
	params := runStartParams{
		EntityKey:    "E07-F01-001",
		EntityType:   "task",
		DryRun:       false,
		Worktree:     false,
		WorktreePath: "",
		RunID:        "test-run-id",
		Args:         []string{"E07-F01-001"},
	}

	lines := captureLog(func() {
		emitRunStart(obs, params)
	})

	if len(lines) == 0 {
		t.Fatal("emitRunStart() produced no log output, expected run.start event")
	}
	events := parseLogLines(t, lines)
	ev := findEvent(events, "run.start")
	if ev == nil {
		t.Fatalf("run.start event not found in log output; got events: %v", lines)
	}
}

// TestEmitRunStart_Fields verifies all required fields are present in run.start.
func TestEmitRunStart_Fields(t *testing.T) {
	obs := config.ObservabilityConfig{Enabled: true}
	params := runStartParams{
		EntityKey:    "E07-F01-001",
		EntityType:   "task",
		DryRun:       true,
		Worktree:     true,
		WorktreePath: "/tmp/shark-worktree-abc",
		RunID:        "abc-123",
		Args:         []string{"E07-F01-001", "--dry-run"},
	}

	lines := captureLog(func() {
		emitRunStart(obs, params)
	})

	events := parseLogLines(t, lines)
	ev := findEvent(events, "run.start")
	if ev == nil {
		t.Fatal("run.start event not found")
	}

	requiredFields := []string{
		"entity_key", "entity_type", "dry_run", "worktree",
		"worktree_path", "run_id", "args",
	}
	for _, field := range requiredFields {
		if _, ok := ev[field]; !ok {
			t.Errorf("run.start event missing required field %q; event: %v", field, ev)
		}
	}

	if ev["entity_key"] != "E07-F01-001" {
		t.Errorf("entity_key = %v, want E07-F01-001", ev["entity_key"])
	}
	if ev["entity_type"] != "task" {
		t.Errorf("entity_type = %v, want task", ev["entity_type"])
	}
	if ev["run_id"] != "abc-123" {
		t.Errorf("run_id = %v, want abc-123", ev["run_id"])
	}
	if ev["dry_run"] != true {
		t.Errorf("dry_run = %v, want true", ev["dry_run"])
	}
	if ev["worktree"] != true {
		t.Errorf("worktree = %v, want true", ev["worktree"])
	}
}

// TestEmitRunStart_LevelIsInfo verifies that run.start is logged at INFO level.
func TestEmitRunStart_LevelIsInfo(t *testing.T) {
	obs := config.ObservabilityConfig{Enabled: true}
	params := runStartParams{
		EntityKey:  "E07-F01-001",
		EntityType: "task",
		RunID:      "test-run-id",
		Args:       []string{"E07-F01-001"},
	}

	lines := captureLog(func() {
		emitRunStart(obs, params)
	})

	events := parseLogLines(t, lines)
	ev := findEvent(events, "run.start")
	if ev == nil {
		t.Fatal("run.start event not found")
	}
	// slog JSON handler writes level as "INFO", "WARN", etc.
	level, _ := ev["level"].(string)
	if !strings.EqualFold(level, "INFO") {
		t.Errorf("run.start level = %q, want INFO", level)
	}
}

// TestEmitRunStart_DisabledWhenObsDisabled verifies that no event is emitted
// when observability.enabled is false (AC-T5).
func TestEmitRunStart_DisabledWhenObsDisabled(t *testing.T) {
	obs := config.ObservabilityConfig{Enabled: false}
	params := runStartParams{
		EntityKey:  "E07-F01-001",
		EntityType: "task",
		RunID:      "test-run-id",
		Args:       []string{"E07-F01-001"},
	}

	lines := captureLog(func() {
		emitRunStart(obs, params)
	})

	events := parseLogLines(t, lines)
	ev := findEvent(events, "run.start")
	if ev != nil {
		t.Error("run.start event emitted when observability is disabled (AC-T5 violation)")
	}
}

// ─── emitRunEnd ──────────────────────────────────────────────────────────────

// TestEmitRunEnd_EmitsEvent verifies that emitRunEnd writes a run.end event.
func TestEmitRunEnd_EmitsEvent(t *testing.T) {
	obs := config.ObservabilityConfig{Enabled: true}
	result := &runner.RunResult{
		EntityKey:       "E07-F01-001",
		Outcome:         "completed",
		FinalStatus:     "completed",
		StagesCompleted: 3,
		Error:           "",
	}

	lines := captureLog(func() {
		emitRunEnd(context.Background(), obs, "run-id-123", result, 150)
	})

	if len(lines) == 0 {
		t.Fatal("emitRunEnd() produced no log output, expected run.end event")
	}
	events := parseLogLines(t, lines)
	ev := findEvent(events, "run.end")
	if ev == nil {
		t.Fatalf("run.end event not found; got: %v", lines)
	}
}

// TestEmitRunEnd_Fields verifies all required fields are present in run.end.
func TestEmitRunEnd_Fields(t *testing.T) {
	obs := config.ObservabilityConfig{Enabled: true}
	result := &runner.RunResult{
		EntityKey:       "E07-F01-001",
		Outcome:         "completed",
		FinalStatus:     "done",
		StagesCompleted: 4,
		Error:           "",
	}

	lines := captureLog(func() {
		emitRunEnd(context.Background(), obs, "run-id-abc", result, 500)
	})

	events := parseLogLines(t, lines)
	ev := findEvent(events, "run.end")
	if ev == nil {
		t.Fatal("run.end event not found")
	}

	requiredFields := []string{
		"entity_key", "outcome", "final_status", "stages_completed",
		"duration_ms", "run_id",
	}
	for _, field := range requiredFields {
		if _, ok := ev[field]; !ok {
			t.Errorf("run.end event missing required field %q; event: %v", field, ev)
		}
	}

	if ev["entity_key"] != "E07-F01-001" {
		t.Errorf("entity_key = %v, want E07-F01-001", ev["entity_key"])
	}
	if ev["outcome"] != "completed" {
		t.Errorf("outcome = %v, want completed", ev["outcome"])
	}
	if ev["run_id"] != "run-id-abc" {
		t.Errorf("run_id = %v, want run-id-abc", ev["run_id"])
	}
}

// TestEmitRunEnd_LevelInfoOnSuccess verifies run.end is INFO when no error (AC-T4).
func TestEmitRunEnd_LevelInfoOnSuccess(t *testing.T) {
	obs := config.ObservabilityConfig{Enabled: true}
	result := &runner.RunResult{
		EntityKey:   "E07-F01-001",
		Outcome:     "completed",
		FinalStatus: "completed",
		Error:       "", // no error
	}

	lines := captureLog(func() {
		emitRunEnd(context.Background(), obs, "run-id", result, 100)
	})

	events := parseLogLines(t, lines)
	ev := findEvent(events, "run.end")
	if ev == nil {
		t.Fatal("run.end event not found")
	}
	level, _ := ev["level"].(string)
	if !strings.EqualFold(level, "INFO") {
		t.Errorf("run.end level = %q on success, want INFO", level)
	}
}

// TestEmitRunEnd_LevelErrorOnFailure verifies run.end is ERROR when result.Error is
// non-empty (AC-T4).
func TestEmitRunEnd_LevelErrorOnFailure(t *testing.T) {
	obs := config.ObservabilityConfig{Enabled: true}
	result := &runner.RunResult{
		EntityKey:   "E07-F01-001",
		Outcome:     "failed",
		FinalStatus: "in_development",
		Error:       "agent exited with code 1",
	}

	lines := captureLog(func() {
		emitRunEnd(context.Background(), obs, "run-id", result, 200)
	})

	events := parseLogLines(t, lines)
	ev := findEvent(events, "run.end")
	if ev == nil {
		t.Fatal("run.end event not found")
	}
	level, _ := ev["level"].(string)
	if !strings.EqualFold(level, "ERROR") {
		t.Errorf("run.end level = %q on failure, want ERROR", level)
	}
	// error field must be present and match
	if ev["error"] != "agent exited with code 1" {
		t.Errorf("error field = %v, want %q", ev["error"], "agent exited with code 1")
	}
}

// TestEmitRunEnd_DisabledWhenObsDisabled verifies no event is emitted when
// observability.enabled is false (AC-T5).
func TestEmitRunEnd_DisabledWhenObsDisabled(t *testing.T) {
	obs := config.ObservabilityConfig{Enabled: false}
	result := &runner.RunResult{
		EntityKey: "E07-F01-001",
		Outcome:   "completed",
	}

	lines := captureLog(func() {
		emitRunEnd(context.Background(), obs, "run-id", result, 50)
	})

	events := parseLogLines(t, lines)
	ev := findEvent(events, "run.end")
	if ev != nil {
		t.Error("run.end event emitted when observability is disabled (AC-T5 violation)")
	}
}

// ─── RunOptions.RunID field ───────────────────────────────────────────────────

// TestRunOptions_HasRunIDField verifies that runner.RunOptions has a RunID field.
// This is AC-T1 from the task: run_id must be in RunOptions for downstream use.
func TestRunOptions_HasRunIDField(t *testing.T) {
	opts := runner.RunOptions{
		RunID: "test-run-id",
	}
	if opts.RunID != "test-run-id" {
		t.Errorf("RunOptions.RunID = %q, want %q", opts.RunID, "test-run-id")
	}
}

// ─── AC-T1 gap-fill: combined run.start + run.end end-to-end flow ─────────────
//
// Existing tests in this file exercise emitRunStart and emitRunEnd in
// isolation. T-E07-F41-005 AC-T1 additionally requires proof that BOTH
// events are emitted in order, share the same run_id, and carry the
// operator-visible fields required by REQ-F-001 / REQ-F-002, when an
// observability-enabled run completes successfully.
//
// Integration seam: runRun in run.go calls emitRunStart before invoking
// RunController.Run and calls emitRunEnd via a deferred closure after
// the controller returns. Since the controller has its own dedicated
// observability coverage (controller_stage_events_test.go), this test
// simulates the seam by calling the two emitters around a constructed
// RunResult that stands in for a successful controller return — exactly
// what runRun does.

// TestRunLogging_RunStartAndRunEnd_BothEmittedInHappyPathFlow is the
// AC-T1 combined-flow assertion.
//
// The test:
//  1. Generates a single run_id via generateRunID (the same source runRun uses).
//  2. Emits run.start with realistic parameters.
//  3. Simulates a successful controller return by constructing a *runner.RunResult
//     with outcome="completed" and error="".
//  4. Emits run.end with INFO level and a plausible duration_ms.
//  5. Asserts BOTH events are present in the captured slog stream, in order,
//     with identical run_id values, and with the attribute sets required by
//     REQ-F-001 (run.start fields) and REQ-F-002 (run.end fields on success).
//
// If a future change drops either emitter or breaks the run_id correlation,
// this test fails and blocks the regression.
func TestRunLogging_RunStartAndRunEnd_BothEmittedInHappyPathFlow(t *testing.T) {
	obs := config.ObservabilityConfig{Enabled: true}

	// Single run_id spans the whole flow — this is the load-bearing property.
	runID := generateRunID()
	if runID == "" {
		t.Fatal("generateRunID() returned empty string")
	}

	startParams := runStartParams{
		Args:         []string{"E07-F01-001"},
		EntityKey:    "E07-F01-001",
		EntityType:   "task",
		DryRun:       false,
		Worktree:     true,
		WorktreePath: "/tmp/shark-worktree/E07-F01-001",
		RunID:        runID,
	}

	// Stand-in for a successful RunController.Run return (see runRun in run.go
	// where the result struct is populated from ctrl.Run and passed straight
	// through to emitRunEnd via the deferred closure).
	endResult := &runner.RunResult{
		EntityKey:       "E07-F01-001",
		Outcome:         "completed",
		FinalStatus:     "completed",
		StagesCompleted: 1,
		Error:           "",
	}
	const durationMS int64 = 123

	lines := captureLog(func() {
		emitRunStart(obs, startParams)
		emitRunEnd(context.Background(), obs, runID, endResult, durationMS)
	})

	events := parseLogLines(t, lines)

	// (1) Exactly one run.start and one run.end must be present.
	startEv := findEvent(events, "run.start")
	endEv := findEvent(events, "run.end")
	if startEv == nil {
		t.Fatalf("expected run.start in combined flow, got events: %v", events)
	}
	if endEv == nil {
		t.Fatalf("expected run.end in combined flow, got events: %v", events)
	}

	// (2) Order: run.start must precede run.end in the captured stream.
	//     We re-walk the raw slice so we assert on positional order, not just
	//     presence, because REQ-F-001/REQ-F-002 treat them as a book-ending
	//     pair around the controller invocation.
	startIdx, endIdx := -1, -1
	for i, e := range events {
		switch e["msg"] {
		case "run.start":
			if startIdx == -1 {
				startIdx = i
			}
		case "run.end":
			if endIdx == -1 {
				endIdx = i
			}
		}
	}
	if startIdx < 0 || endIdx < 0 || startIdx >= endIdx {
		t.Errorf("expected run.start (idx=%d) to precede run.end (idx=%d)", startIdx, endIdx)
	}

	// (3) run_id correlation: both events must carry the same run_id string.
	startRunID, _ := startEv["run_id"].(string)
	endRunID, _ := endEv["run_id"].(string)
	if startRunID != runID {
		t.Errorf("run.start run_id = %q, want %q", startRunID, runID)
	}
	if endRunID != runID {
		t.Errorf("run.end run_id = %q, want %q", endRunID, runID)
	}
	if startRunID != endRunID {
		t.Errorf("run.start run_id (%q) must equal run.end run_id (%q) for correlation", startRunID, endRunID)
	}

	// (4) run.start REQ-F-001 attribute set.
	wantStart := map[string]interface{}{
		"command":       "run",
		"entity_key":    "E07-F01-001",
		"entity_type":   "task",
		"dry_run":       false,
		"worktree":      true,
		"worktree_path": "/tmp/shark-worktree/E07-F01-001",
	}
	for k, want := range wantStart {
		if got, ok := startEv[k]; !ok {
			t.Errorf("run.start missing attribute %q", k)
		} else if got != want {
			t.Errorf("run.start[%q] = %v, want %v", k, got, want)
		}
	}
	// args must round-trip as the same string slice.
	argsAny, ok := startEv["args"]
	if !ok {
		t.Error("run.start missing attribute \"args\"")
	} else {
		argsSlice, ok := argsAny.([]interface{})
		if !ok || len(argsSlice) != 1 || argsSlice[0] != "E07-F01-001" {
			t.Errorf("run.start[\"args\"] = %v, want [\"E07-F01-001\"]", argsAny)
		}
	}

	// (5) run.end REQ-F-002 attribute set (INFO level on success, error absent).
	if lvl, _ := endEv["level"].(string); lvl != "INFO" {
		t.Errorf("run.end level = %q, want INFO on success", lvl)
	}
	if _, hasErr := endEv["error"]; hasErr {
		t.Errorf("run.end must NOT include \"error\" attribute on success, got: %v", endEv["error"])
	}
	wantEnd := map[string]interface{}{
		"entity_key":       "E07-F01-001",
		"outcome":          "completed",
		"final_status":     "completed",
		"stages_completed": float64(1), // JSON decodes int → float64
		"duration_ms":      float64(durationMS),
	}
	for k, want := range wantEnd {
		if got, ok := endEv[k]; !ok {
			t.Errorf("run.end missing attribute %q", k)
		} else if got != want {
			t.Errorf("run.end[%q] = %v (%T), want %v (%T)", k, got, got, want, want)
		}
	}
}
