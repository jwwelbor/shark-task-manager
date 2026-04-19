// Package runner
//
// Stage-event emission tests: per-stage slog events emitted by RunController.
// These tests capture slog output during a run and assert that each expected
// event (run.stage.start / run.stage.dispatch / run.stage.complete /
// run.stage.transition / run.stage.error) is emitted with the documented
// attribute set.
//
// Test strategy:
//   - Install a custom slog.Handler for the duration of each test that writes
//     structured records into an in-memory buffer.
//   - Drive a RunController (with mocked deps) through representative paths:
//   - single successful stage (terminal after one transition)
//   - agent failure via AgentFailedError (error wraps result)
//   - agent failure via non-zero exit code without error (mock style)
//   - Parse the captured JSON lines and assert attributes are present /
//     absent / within the expected size envelopes.
//
// Golden rule: no real database, no real agent, no filesystem. Pure mocks.
package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// ---------------------------------------------------------------------------
// slog capture infrastructure
// ---------------------------------------------------------------------------

// captureSlog redirects the default slog logger to a JSON handler backed by
// buf for the duration of t. The handler is at LevelDebug so all records land
// in buf. Restoration is registered with t.Cleanup.
func captureSlog(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	handler := slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	prev := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return buf
}

// parseEvents returns the sequence of slog records captured in buf parsed as
// generic JSON maps. Each record is one JSON object on its own line.
func parseEvents(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimRight(buf.String(), "\n"), "\n") {
		if line == "" {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("parseEvents: invalid JSON line %q: %v", line, err)
		}
		out = append(out, rec)
	}
	return out
}

// eventsByMsg filters parsed events by their "msg" field. Order is preserved.
func eventsByMsg(events []map[string]any, msg string) []map[string]any {
	var out []map[string]any
	for _, e := range events {
		if s, _ := e["msg"].(string); s == msg {
			out = append(out, e)
		}
	}
	return out
}

// requireAttrs asserts that every key in want is present on ev with the
// expected value. Value comparison uses a tolerant approach that handles the
// JSON numeric encoding (int → float64 after unmarshalling).
func requireAttrs(t *testing.T, ev map[string]any, want map[string]any, context string) {
	t.Helper()
	for k, expected := range want {
		got, ok := ev[k]
		if !ok {
			t.Errorf("%s: attribute %q missing from event", context, k)
			continue
		}
		if !attrEqual(expected, got) {
			t.Errorf("%s: attribute %q = %v (%T), want %v (%T)", context, k, got, got, expected, expected)
		}
	}
}

// attrEqual compares expected/got with tolerance for JSON numeric decoding
// (integer literals decode as float64).
func attrEqual(want, got any) bool {
	switch w := want.(type) {
	case int:
		if g, ok := got.(float64); ok {
			return g == float64(w)
		}
	case int64:
		if g, ok := got.(float64); ok {
			return g == float64(w)
		}
	}
	return want == got
}

// ---------------------------------------------------------------------------
// Shared helpers for happy-path fixtures
// ---------------------------------------------------------------------------

// happyPathFixture returns mocks that drive exactly one successful stage
// (spawn_agent + transition to a terminal status). The dispatcher behaviour
// can be customised via dispatchFunc; pass nil for "ExitCode:0, Stdout: ..." default.
func happyPathFixture(t *testing.T, dispatchFunc func(ctx context.Context, input DispatchInput) (*DispatchResult, error)) *RunController {
	t.Helper()

	calls := 0
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			calls++
			if calls == 1 {
				return &services.NextStatusInfo{
					CurrentStatus: "in_development",
					IsTerminal:    false,
					AvailableTransitions: []services.TransitionInfoWithAction{
						{TransitionInfo: workflow.TransitionInfo{TargetStatus: "completed"}},
					},
				}, nil
			}
			return &services.NextStatusInfo{
				CurrentStatus: "completed",
				IsTerminal:    true,
			}, nil
		},
		TransitionStatusFunc: func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			return &services.TransitionResult{ToStatus: target}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:      config.ActionSpawnAgent,
				AgentType:   "developer",
				Provider:    "anthropic",
				Instruction: "do work",
			}, nil
		},
	}
	if dispatchFunc == nil {
		dispatchFunc = func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
			return &DispatchResult{
				ExitCode: 0,
				Stdout:   "agent stdout",
				Command:  "claude-mock --prompt ...",
			}, nil
		}
	}
	dispatchers := map[string]AgentDispatcher{
		"anthropic": &MockDispatcher{DispatchFunc: dispatchFunc},
	}
	return makeController(t, transitioner, actionSvc, dispatchers)
}

// obsEnabled returns an ObservabilityConfig with Enabled=true and the provided
// truncation byte budget (0 → default of 4096).
func obsEnabled(truncateBytes int) config.ObservabilityConfig {
	return config.ObservabilityConfig{
		Enabled:          true,
		LogTruncateBytes: truncateBytes,
	}
}

// ---------------------------------------------------------------------------
// run.stage.start — entity_key, status, iteration, run_id
// ---------------------------------------------------------------------------

// TestController_EmitsStageStart verifies that each loop iteration emits a
// run.stage.start event with the required attributes and a monotonically
// increasing iteration counter starting at 1.
func TestController_EmitsStageStart(t *testing.T) {
	buf := captureSlog(t)

	ctrl := happyPathFixture(t, nil)
	opts := RunOptions{
		RunID:         "run-123",
		Observability: obsEnabled(0),
	}
	result, err := ctrl.Run(context.Background(), "E07-F01-001", opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Outcome != "completed" {
		t.Fatalf("expected outcome=completed, got %s", result.Outcome)
	}

	events := eventsByMsg(parseEvents(t, buf), "run.stage.start")
	if len(events) < 1 {
		t.Fatalf("expected at least 1 run.stage.start event, got %d", len(events))
	}
	requireAttrs(t, events[0], map[string]any{
		"entity_key": "E07-F01-001",
		"status":     "in_development",
		"iteration":  1,
		"run_id":     "run-123",
	}, "stage.start[0]")
}

// ---------------------------------------------------------------------------
// run.stage.dispatch — entity_key, status, agent_type, provider, command, run_id
// ---------------------------------------------------------------------------

// TestController_EmitsStageDispatch verifies the dispatch event precedes
// agent invocation and carries the required attributes.
func TestController_EmitsStageDispatch(t *testing.T) {
	buf := captureSlog(t)

	var seenInput DispatchInput
	ctrl := happyPathFixture(t, func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
		seenInput = input
		return &DispatchResult{
			ExitCode: 0,
			Stdout:   "ok",
			Command:  "claude-mock --prompt x",
		}, nil
	})
	opts := RunOptions{
		RunID:         "run-abc",
		Observability: obsEnabled(0),
	}
	_, err := ctrl.Run(context.Background(), "E07-F01-001", opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	// Sanity: the dispatcher was reached at all.
	if seenInput.EntityKey == "" {
		t.Fatal("dispatcher was never called")
	}

	events := eventsByMsg(parseEvents(t, buf), "run.stage.dispatch")
	if len(events) != 1 {
		t.Fatalf("expected exactly 1 run.stage.dispatch event, got %d", len(events))
	}
	requireAttrs(t, events[0], map[string]any{
		"entity_key": "E07-F01-001",
		"status":     "in_development",
		"agent_type": "developer",
		"provider":   "anthropic",
		"run_id":     "run-abc",
	}, "stage.dispatch[0]")
	if _, ok := events[0]["command"]; !ok {
		t.Error("stage.dispatch: attribute \"command\" missing")
	}
}

// ---------------------------------------------------------------------------
// run.stage.complete — entity_key, status, exit_code, duration_ms, next_status, run_id
// (MUST NOT include stdout)
// ---------------------------------------------------------------------------

// TestController_EmitsStageComplete verifies the complete event carries the
// required attributes and MUST NOT carry stdout (the complete event is a hot
// path; transcript capture goes to a separate channel, not slog).
func TestController_EmitsStageComplete(t *testing.T) {
	buf := captureSlog(t)

	ctrl := happyPathFixture(t, func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
		return &DispatchResult{
			ExitCode: 0,
			Stdout:   "this stdout should NOT appear in slog",
			Command:  "claude-mock ...",
		}, nil
	})
	opts := RunOptions{
		RunID:         "run-xyz",
		Observability: obsEnabled(0),
	}
	_, err := ctrl.Run(context.Background(), "E07-F01-001", opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	events := eventsByMsg(parseEvents(t, buf), "run.stage.complete")
	if len(events) != 1 {
		t.Fatalf("expected exactly 1 run.stage.complete event, got %d", len(events))
	}
	requireAttrs(t, events[0], map[string]any{
		"entity_key":  "E07-F01-001",
		"status":      "in_development",
		"exit_code":   0,
		"next_status": "completed",
		"run_id":      "run-xyz",
	}, "stage.complete[0]")
	if _, ok := events[0]["duration_ms"]; !ok {
		t.Error("stage.complete: attribute \"duration_ms\" missing")
	}
	if _, ok := events[0]["stdout"]; ok {
		t.Error("stage.complete: attribute \"stdout\" must NOT be present")
	}
}

// ---------------------------------------------------------------------------
// run.stage.transition — entity_key, from_status, to_status, run_id
// ---------------------------------------------------------------------------

// TestController_EmitsStageTransition verifies that when a status transition
// is performed (after a successful spawn_agent, or inside advance_status), a
// run.stage.transition event is emitted with from_status, to_status, and run_id.
func TestController_EmitsStageTransition(t *testing.T) {
	buf := captureSlog(t)

	ctrl := happyPathFixture(t, nil)
	opts := RunOptions{
		RunID:         "run-trn",
		Observability: obsEnabled(0),
	}
	_, err := ctrl.Run(context.Background(), "E07-F01-001", opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	events := eventsByMsg(parseEvents(t, buf), "run.stage.transition")
	if len(events) < 1 {
		t.Fatalf("expected at least 1 run.stage.transition event, got %d", len(events))
	}
	requireAttrs(t, events[0], map[string]any{
		"entity_key":  "E07-F01-001",
		"from_status": "in_development",
		"to_status":   "completed",
		"run_id":      "run-trn",
	}, "stage.transition[0]")
}

// ---------------------------------------------------------------------------
// run.stage.error — exit_code, stderr (truncated), stdout_tail
//                   (truncated), command, truncated flag
// ---------------------------------------------------------------------------

// TestController_EmitsStageErrorOnAgentFailure verifies that a failed agent
// dispatch (non-zero exit) emits a run.stage.error event carrying diagnostic
// fields: exit_code, stderr, stdout_tail, command, and truncated.
//
// This covers the "dispatcher returns *AgentFailedError" path used by real
// Claude/Codex dispatchers (execAndCapture wraps non-zero exits).
func TestController_EmitsStageErrorOnAgentFailure(t *testing.T) {
	buf := captureSlog(t)

	// Simulate the real dispatcher contract: on non-zero exit, the dispatcher
	// returns BOTH a *DispatchResult and an *AgentFailedError wrapping the
	// exit code, stdout, stderr, and command.
	failStderr := "fatal: compilation failed"
	failStdout := "working ... still working ... still working"
	failCmd := "claude-mock --prompt x"
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{CurrentStatus: "in_development", IsTerminal: false}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:      config.ActionSpawnAgent,
				AgentType:   "developer",
				Provider:    "",
				Instruction: "do work",
			}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{
		"": &MockDispatcher{
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				return &DispatchResult{
						ExitCode: 2,
						Stdout:   failStdout,
						Stderr:   failStderr,
						Command:  failCmd,
					}, &AgentFailedError{
						ExitCode: 2,
						Stdout:   failStdout,
						Stderr:   failStderr,
						Command:  failCmd,
					}
			},
		},
	}
	ctrl := makeController(t, transitioner, actionSvc, dispatchers)

	opts := RunOptions{
		RunID:         "run-err",
		Observability: obsEnabled(0),
	}
	result, err := ctrl.Run(context.Background(), "E07-F01-001", opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Outcome != "failed" {
		t.Fatalf("expected outcome=failed, got %s", result.Outcome)
	}

	events := eventsByMsg(parseEvents(t, buf), "run.stage.error")
	if len(events) < 1 {
		t.Fatalf("expected at least 1 run.stage.error event, got %d", len(events))
	}
	ev := events[0]
	requireAttrs(t, ev, map[string]any{
		"exit_code": 2,
		"command":   failCmd,
		"run_id":    "run-err",
	}, "stage.error[0]")
	// stderr must be present (possibly truncated — short input so full).
	if s, _ := ev["stderr"].(string); s == "" {
		t.Error("stage.error: attribute \"stderr\" missing or empty")
	}
	// stdout_tail must be present (from the tail of stdout).
	if s, _ := ev["stdout_tail"].(string); s == "" {
		t.Error("stage.error: attribute \"stdout_tail\" missing or empty")
	}
	// truncated flag must be ABSENT for small payloads: the attribute is only
	// emitted when truncation actually occurred.
	if _, ok := ev["truncated"]; ok {
		t.Error("stage.error: attribute \"truncated\" must be omitted for small payloads")
	}
}

// TestController_EmitsStageErrorOnNonZeroExitWithoutError covers the test-mock
// path where Dispatch returns (result, nil) with a non-zero ExitCode. The
// controller's existing "gate on exit code 0" branch (line 447) currently
// handles this. The stage.error event must still be emitted here too, with
// the same attribute set built from DispatchResult.
func TestController_EmitsStageErrorOnNonZeroExitWithoutError(t *testing.T) {
	buf := captureSlog(t)

	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{CurrentStatus: "in_development", IsTerminal: false}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:    config.ActionSpawnAgent,
				AgentType: "developer",
				Provider:  "",
			}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{
		"": &MockDispatcher{
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				return &DispatchResult{
					ExitCode: 9,
					Stdout:   "some stdout",
					Stderr:   "some stderr",
					Command:  "fake-cmd",
				}, nil
			},
		},
	}
	ctrl := makeController(t, transitioner, actionSvc, dispatchers)

	opts := RunOptions{
		RunID:         "run-err2",
		Observability: obsEnabled(0),
	}
	result, err := ctrl.Run(context.Background(), "E07-F01-001", opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Outcome != "failed" {
		t.Fatalf("expected outcome=failed, got %s", result.Outcome)
	}

	events := eventsByMsg(parseEvents(t, buf), "run.stage.error")
	if len(events) < 1 {
		t.Fatalf("expected at least 1 run.stage.error event, got %d", len(events))
	}
	requireAttrs(t, events[0], map[string]any{
		"exit_code": 9,
		"command":   "fake-cmd",
		"run_id":    "run-err2",
	}, "stage.error[0]")
}

// TestController_TruncatesStderr verifies that stderr/stdout_tail are truncated
// to LogTruncateBytes and that the "truncated" flag is set when truncation occurs.
func TestController_TruncatesStderr(t *testing.T) {
	buf := captureSlog(t)

	const limit = 128
	big := strings.Repeat("x", 4096) // much larger than limit

	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{CurrentStatus: "in_development", IsTerminal: false}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{Action: config.ActionSpawnAgent, AgentType: "developer"}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{
		"": &MockDispatcher{
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				return &DispatchResult{
						ExitCode: 1,
						Stdout:   big,
						Stderr:   big,
						Command:  "cmd",
					}, &AgentFailedError{
						ExitCode: 1,
						Stdout:   big,
						Stderr:   big,
						Command:  "cmd",
					}
			},
		},
	}
	ctrl := makeController(t, transitioner, actionSvc, dispatchers)

	opts := RunOptions{
		RunID:         "run-trunc",
		Observability: obsEnabled(limit),
	}
	_, err := ctrl.Run(context.Background(), "E07-F01-001", opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	events := eventsByMsg(parseEvents(t, buf), "run.stage.error")
	if len(events) < 1 {
		t.Fatalf("expected at least 1 run.stage.error event, got %d", len(events))
	}
	ev := events[0]

	// stderr must be truncated to at most `limit` bytes.
	stderr, _ := ev["stderr"].(string)
	if len(stderr) == 0 {
		t.Fatal("stage.error: stderr missing or empty")
	}
	if len(stderr) > limit {
		t.Errorf("stage.error: stderr len=%d exceeds truncate limit=%d", len(stderr), limit)
	}
	stdoutTail, _ := ev["stdout_tail"].(string)
	if len(stdoutTail) == 0 {
		t.Fatal("stage.error: stdout_tail missing or empty")
	}
	if len(stdoutTail) > limit {
		t.Errorf("stage.error: stdout_tail len=%d exceeds truncate limit=%d", len(stdoutTail), limit)
	}
	if tr, _ := ev["truncated"].(bool); !tr {
		t.Error("stage.error: truncated flag must be true when stderr/stdout exceed limit")
	}
}

// ---------------------------------------------------------------------------
// Successful dispatch event stays under ~1 KB
// ---------------------------------------------------------------------------

// TestController_SuccessfulDispatchUnder1KB verifies that the emitted
// run.stage.dispatch event stays under ~1 KB on a REALISTIC agent invocation
// (≥ 4 KB command including a large -p prompt argv token). Instruction is not
// a separate field on dispatch — it is embedded in the `command` string,
// which is why a 1024-byte hard cap is applied to the command attribute.
//
// The test asserts two things:
//  1. The emitted `command` attribute is ≤ dispatchCommandMaxBytes (1024
//     bytes), i.e. the cap actually fires on realistic input.
//  2. The raw JSON line for the dispatch event is bounded to a small,
//     operator-friendly envelope around the capped command — concretely,
//     the envelope overhead (JSON escaping + attrs) cannot exceed 512
//     bytes, so the full line stays under 1024+512 = 1536 bytes. This
//     codifies the "under ~1 KB" intent with a deterministic bound while
//     tolerating the slog JSON envelope.
func TestController_SuccessfulDispatchUnder1KB(t *testing.T) {
	buf := captureSlog(t)

	// Realistic ≥ 4 KB invocation. An actual `claude -p "<instruction>"`
	// call where the instruction carries detailed task context can easily
	// exceed 4 KB; the 1024-byte cap must hold regardless.
	const padSize = 4 * 1024
	bigCmd := "claude -p " + strings.Repeat("x", padSize) + " --output-format json"
	if len(bigCmd) <= 1024 {
		t.Fatalf("fixture setup: bigCmd length %d must exceed 1024-byte cap", len(bigCmd))
	}

	calls := 0
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			calls++
			if calls == 1 {
				return &services.NextStatusInfo{
					CurrentStatus: "in_development",
					IsTerminal:    false,
					AvailableTransitions: []services.TransitionInfoWithAction{
						{TransitionInfo: workflow.TransitionInfo{TargetStatus: "completed"}},
					},
				}, nil
			}
			return &services.NextStatusInfo{CurrentStatus: "completed", IsTerminal: true}, nil
		},
		TransitionStatusFunc: func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			return &services.TransitionResult{ToStatus: target}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:      config.ActionSpawnAgent,
				AgentType:   "developer",
				Provider:    "anthropic",
				Instruction: "do work",
			}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{
		"anthropic": &MockDispatcher{
			BuildCommandFunc: func(input DispatchInput) (string, error) { return bigCmd, nil },
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				return &DispatchResult{ExitCode: 0, Command: bigCmd}, nil
			},
		},
	}
	ctrl := makeController(t, transitioner, actionSvc, dispatchers)

	opts := RunOptions{
		RunID:         "run-1kb",
		Observability: obsEnabled(0),
	}
	if _, err := ctrl.Run(context.Background(), "E07-F01-001", opts); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	// (1) Emitted `command` attribute must be ≤ 1024 bytes.
	dispatchEvents := eventsByMsg(parseEvents(t, buf), "run.stage.dispatch")
	if len(dispatchEvents) != 1 {
		t.Fatalf("expected 1 run.stage.dispatch event, got %d", len(dispatchEvents))
	}
	gotCmd, _ := dispatchEvents[0]["command"].(string)
	if len(gotCmd) > 1024 {
		t.Errorf("stage.dispatch: emitted command is %d bytes, must be ≤ 1024", len(gotCmd))
	}

	// (2) Raw JSON line for the dispatch event must be within a small
	// operator-friendly envelope around the capped command. With a
	// 1024-byte command attribute and a slog JSON envelope (timestamp +
	// level + msg + run_id + entity_key + status + agent_type + provider +
	// truncated), the total line is slightly over 1 KB but bounded. We
	// assert a hard ceiling of 1536 bytes (1024 + 512 envelope budget) to
	// codify the intent while tolerating slog's JSON framing. Without the
	// command cap the line would be ≥ 4 KB (failure mode).
	const dispatchLineMaxBytes = 1024 + 512
	for _, line := range strings.Split(strings.TrimRight(buf.String(), "\n"), "\n") {
		if !strings.Contains(line, `"run.stage.dispatch"`) {
			continue
		}
		if n := len(line); n > dispatchLineMaxBytes {
			t.Errorf("run.stage.dispatch JSON line is %d bytes, must be ≤ %d (~1 KB envelope)", n, dispatchLineMaxBytes)
		}
		return
	}
	t.Fatal("no run.stage.dispatch event found in captured slog output")
}

// ---------------------------------------------------------------------------
// Correlation: every stage event on a single run must share the same run_id.
// ---------------------------------------------------------------------------

// TestController_AllEventsShareRunID verifies that every stage.* event emitted
// during a single Run call carries the same run_id attribute. This enables
// grep-based correlation in shark.log.
func TestController_AllEventsShareRunID(t *testing.T) {
	buf := captureSlog(t)

	ctrl := happyPathFixture(t, nil)
	const runID = "run-correlate"
	opts := RunOptions{
		RunID:         runID,
		Observability: obsEnabled(0),
	}
	if _, err := ctrl.Run(context.Background(), "E07-F01-001", opts); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	events := parseEvents(t, buf)
	if len(events) == 0 {
		t.Fatal("no slog events captured")
	}
	stageCount := 0
	for _, ev := range events {
		msg, _ := ev["msg"].(string)
		if !strings.HasPrefix(msg, "run.stage.") {
			continue
		}
		stageCount++
		got, _ := ev["run_id"].(string)
		if got != runID {
			t.Errorf("event %q has run_id=%q, want %q", msg, got, runID)
		}
	}
	if stageCount < 4 {
		// Expect: stage.start, stage.dispatch, stage.complete, stage.transition
		t.Errorf("expected at least 4 stage.* events in a happy path run, got %d", stageCount)
	}
}

// ---------------------------------------------------------------------------
// Disabled observability: no stage.* events should be emitted at all.
// ---------------------------------------------------------------------------

// TestController_NoEventsWhenObservabilityDisabled verifies that when
// Observability.Enabled is false, no run.stage.* events are emitted.
func TestController_NoEventsWhenObservabilityDisabled(t *testing.T) {
	buf := captureSlog(t)

	ctrl := happyPathFixture(t, nil)
	opts := RunOptions{
		RunID:         "run-off",
		Observability: config.ObservabilityConfig{Enabled: false},
	}
	if _, err := ctrl.Run(context.Background(), "E07-F01-001", opts); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	for _, ev := range parseEvents(t, buf) {
		msg, _ := ev["msg"].(string)
		if strings.HasPrefix(msg, "run.stage.") {
			t.Errorf("expected no stage.* events when Observability.Enabled=false, got %q", msg)
		}
	}
}

// ---------------------------------------------------------------------------
// run.stage.dispatch MUST truncate the command field and emit truncated=true
// when truncation occurs.
//
// Successful dispatches must produce log events under ~1 KB; without
// truncation a pathological instruction (e.g. a 20 KB prompt with
// placeholders expanded to full file contents) would blow the budget on
// every stage.
// ---------------------------------------------------------------------------

// TestController_StageDispatch_CommandTruncatedToBudget verifies that when
// the dispatcher's BuildCommand returns a command string larger than the
// dispatch command cap, run.stage.dispatch emits:
//   - `command` attribute whose byte-length is exactly dispatchCommandMaxBytes
//     (1024 — the tail of the original string, because the back of the
//     string carries the most operator-useful content: flags and the
//     beginning of the instruction argv token).
//   - `truncated` attribute present and equal to true.
//
// This exercises the 1024-byte hard cap on the successful-dispatch `command`
// attribute, independent of `observability.log_truncate_bytes` which
// governs only the error-path budget for stderr / stdout_tail.
func TestController_StageDispatch_CommandTruncatedToBudget(t *testing.T) {
	buf := captureSlog(t)

	// Build a 20 KB command string. We prefix it with a marker that MUST
	// NOT appear in the logged command (truncation keeps the tail), and
	// suffix it with a marker that MUST appear (it survives tail preservation).
	const (
		prefixMarker = "PREFIX-THAT-MUST-BE-DROPPED"
		suffixMarker = "SUFFIX-THAT-MUST-BE-KEPT"
		padSize      = 20 * 1024
	)
	bigCmd := prefixMarker + strings.Repeat("x", padSize) + suffixMarker
	if len(bigCmd) <= 1024 {
		t.Fatalf("fixture setup: bigCmd length %d must exceed 1024-byte cap", len(bigCmd))
	}

	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "in_development",
				IsTerminal:    false,
				AvailableTransitions: []services.TransitionInfoWithAction{
					{TransitionInfo: workflow.TransitionInfo{TargetStatus: "completed"}},
				},
			}, nil
		},
		TransitionStatusFunc: func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			return &services.TransitionResult{ToStatus: target}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:      config.ActionSpawnAgent,
				AgentType:   "developer",
				Provider:    "anthropic",
				Instruction: "do-work",
			}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{
		"anthropic": &MockDispatcher{
			BuildCommandFunc: func(input DispatchInput) (string, error) { return bigCmd, nil },
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				return &DispatchResult{ExitCode: 0, Command: bigCmd}, nil
			},
		},
	}
	ctrl := makeController(t, transitioner, actionSvc, dispatchers)

	// Use default budget (0 → 4096 inside GetLogTruncateBytes()).
	opts := RunOptions{RunID: "run-trunc", Observability: obsEnabled(0)}
	if _, err := ctrl.Run(context.Background(), "E07-F01-001", opts); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	dispatchEvents := eventsByMsg(parseEvents(t, buf), "run.stage.dispatch")
	if len(dispatchEvents) != 1 {
		t.Fatalf("expected 1 run.stage.dispatch event, got %d", len(dispatchEvents))
	}
	ev := dispatchEvents[0]

	gotCmd, ok := ev["command"].(string)
	if !ok {
		t.Fatalf("stage.dispatch: command attr missing or not a string: %v", ev["command"])
	}

	// The hard cap is 1024 bytes (dispatchCommandMaxBytes constant in
	// logging.go — independent of observability.log_truncate_bytes, which
	// defaults to 4096 and governs the error-path budget).
	const wantLen = 1024
	if len(gotCmd) != wantLen {
		t.Errorf("stage.dispatch: len(command) = %d, want %d", len(gotCmd), wantLen)
	}

	// Tail preservation: suffix marker MUST be present.
	if !strings.Contains(gotCmd, suffixMarker) {
		t.Errorf("stage.dispatch: command should preserve tail suffix %q, not found in logged command", suffixMarker)
	}
	// Head-dropped: prefix marker MUST be absent (bigCmd > budget and
	// truncateTail keeps the last `limit` bytes).
	if strings.Contains(gotCmd, prefixMarker) {
		t.Errorf("stage.dispatch: command should drop head prefix %q when truncated", prefixMarker)
	}

	// Truncated attribute MUST be present and true when truncation occurs.
	truncRaw, present := ev["truncated"]
	if !present {
		t.Errorf("stage.dispatch: truncated attr missing (required when truncation occurs)")
	} else if truncRaw != true {
		t.Errorf("stage.dispatch: truncated = %v (%T), want true", truncRaw, truncRaw)
	}
}

// TestController_StageDispatch_NoTruncationAttrWhenUnderBudget verifies the
// inverse of the above: when the command is under the byte budget, the
// `truncated` attribute is OMITTED entirely. This prevents downstream log
// consumers from having to reason about a truncated=false signal.
func TestController_StageDispatch_NoTruncationAttrWhenUnderBudget(t *testing.T) {
	buf := captureSlog(t)

	const smallCmd = "claude -p do-work --output-format json"

	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "in_development",
				IsTerminal:    false,
				AvailableTransitions: []services.TransitionInfoWithAction{
					{TransitionInfo: workflow.TransitionInfo{TargetStatus: "completed"}},
				},
			}, nil
		},
		TransitionStatusFunc: func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			return &services.TransitionResult{ToStatus: target}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:      config.ActionSpawnAgent,
				AgentType:   "developer",
				Provider:    "anthropic",
				Instruction: "do-work",
			}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{
		"anthropic": &MockDispatcher{
			BuildCommandFunc: func(input DispatchInput) (string, error) { return smallCmd, nil },
			DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
				return &DispatchResult{ExitCode: 0, Command: smallCmd}, nil
			},
		},
	}
	ctrl := makeController(t, transitioner, actionSvc, dispatchers)

	opts := RunOptions{RunID: "run-small", Observability: obsEnabled(0)}
	if _, err := ctrl.Run(context.Background(), "E07-F01-001", opts); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	dispatchEvents := eventsByMsg(parseEvents(t, buf), "run.stage.dispatch")
	if len(dispatchEvents) != 1 {
		t.Fatalf("expected 1 run.stage.dispatch event, got %d", len(dispatchEvents))
	}
	ev := dispatchEvents[0]

	gotCmd, _ := ev["command"].(string)
	if gotCmd != smallCmd {
		t.Errorf("stage.dispatch: command = %q, want unchanged %q", gotCmd, smallCmd)
	}
	if _, present := ev["truncated"]; present {
		t.Errorf("stage.dispatch: truncated attr MUST be absent when command fits budget, got %v", ev["truncated"])
	}
}
