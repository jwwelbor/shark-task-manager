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
