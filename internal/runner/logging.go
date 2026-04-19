// Package runner
//
// Per-stage structured logging.
//
// This file defines the slog emit helpers used by RunController to produce a
// complete per-stage execution trace in shark.log. Five distinct event types
// are emitted over the life of a single run:
//
//   - run.stage.start       — emitted at the top of each loop iteration
//   - run.stage.dispatch    — emitted before an agent process is spawned
//   - run.stage.complete    — emitted after a successful agent dispatch
//   - run.stage.transition  — emitted after a status transition succeeds
//   - run.stage.error       — emitted on any failure (dispatch, transition, ...)
//
// Every event carries a run_id attribute so that a single run can be grepped
// out of shark.log. When ObservabilityConfig.Enabled is false, all emit
// helpers are no-ops — the run.start / run.end contract is preserved.
package runner

import (
	"context"
	"log/slog"
	"unicode/utf8"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

// dispatchCommandMaxBytes is the HARD per-event byte cap on the `command`
// attribute emitted on run.stage.dispatch. The successful-dispatch budget is
// independent of the error-path budget (`log_truncate_bytes`, default 4096,
// which governs only run.stage.error stderr/stdout_tail) and is intentionally
// NOT configurable: it is a spec-baked ceiling so operator-visible dispatch
// events remain within a ~1 KB envelope regardless of agent-instruction size.
const dispatchCommandMaxBytes = 1024

// stageStartParams bundles the fields emitted on run.stage.start.
type stageStartParams struct {
	EntityKey string
	Status    string
	Iteration int
	RunID     string
}

// emitStageStart emits the run.stage.start slog event at INFO level.
// No-ops when obs.Enabled is false.
func emitStageStart(ctx context.Context, obs config.ObservabilityConfig, p stageStartParams) {
	if !obs.Enabled {
		return
	}
	slog.InfoContext(ctx, "run.stage.start",
		"entity_key", p.EntityKey,
		"status", p.Status,
		"iteration", p.Iteration,
		"run_id", p.RunID,
	)
}

// stageDispatchParams bundles the fields emitted on run.stage.dispatch.
// Command is the full shell-equivalent invocation string including the
// agent-instruction argv token; it can be many KB. emitStageDispatch
// truncates it to dispatchCommandMaxBytes (1024) so the event stays under
// the ~1 KB budget, setting a conditional `truncated` attribute when
// truncation occurs. This cap is independent of
// `observability.log_truncate_bytes`, which governs only the error-path
// stderr/stdout_tail budget.
type stageDispatchParams struct {
	EntityKey string
	Status    string
	AgentType string
	Provider  string
	Command   string
	RunID     string
}

// emitStageDispatch emits the run.stage.dispatch slog event at INFO level.
// It applies a HARD byte-budget truncation to the command string using the
// package-level constant dispatchCommandMaxBytes (1024 bytes) — NOT
// obs.GetLogTruncateBytes(), which is reserved for the error-path stderr
// and stdout_tail budgets. The prefix of the command is preserved
// (limitPrefix) because the most operator-useful information — the binary
// name and key flags — sits at the beginning of the string. The
// `truncated` attribute is emitted (= true) ONLY when truncation actually
// occurred; when no truncation happens, the attribute is omitted entirely.
// No-ops when obs.Enabled is false.
func emitStageDispatch(ctx context.Context, obs config.ObservabilityConfig, p stageDispatchParams) {
	if !obs.Enabled {
		return
	}
	command, truncated := limitPrefix(p.Command, dispatchCommandMaxBytes)

	attrs := []any{
		"entity_key", p.EntityKey,
		"status", p.Status,
		"agent_type", p.AgentType,
		"provider", p.Provider,
		"command", command,
		"run_id", p.RunID,
	}
	if truncated {
		attrs = append(attrs, "truncated", true)
	}
	slog.InfoContext(ctx, "run.stage.dispatch", attrs...)
}

// stageCompleteParams bundles the fields emitted on run.stage.complete.
// Stdout is deliberately NOT included — the complete event is a hot path and
// agent transcripts ride on a separate transcript-capture channel.
//
// TranscriptPath is the PROJECT-RELATIVE path to the per-dispatch transcript
// file that was successfully written. Empty string means "no transcript was
// written for this dispatch" — either capture is disabled, the run-scoped
// disable flag has tripped, or this dispatch's write failed. When non-empty,
// it is emitted as the `transcript_path` attribute; when empty the attribute
// is omitted entirely.
type stageCompleteParams struct {
	EntityKey      string
	Status         string
	AgentType      string
	Provider       string
	ExitCode       int
	DurationMS     int64
	NextStatus     string
	RunID          string
	TranscriptPath string
}

// emitStageComplete emits the run.stage.complete slog event at INFO level.
// No-ops when obs.Enabled is false.
func emitStageComplete(ctx context.Context, obs config.ObservabilityConfig, p stageCompleteParams) {
	if !obs.Enabled {
		return
	}
	attrs := []any{
		"entity_key", p.EntityKey,
		"status", p.Status,
		"agent_type", p.AgentType,
		"provider", p.Provider,
		"exit_code", p.ExitCode,
		"duration_ms", p.DurationMS,
		"next_status", p.NextStatus,
		"run_id", p.RunID,
	}
	if p.TranscriptPath != "" {
		attrs = append(attrs, "transcript_path", p.TranscriptPath)
	}
	slog.InfoContext(ctx, "run.stage.complete", attrs...)
}

// stageTransitionParams bundles the fields emitted on run.stage.transition.
type stageTransitionParams struct {
	EntityKey  string
	FromStatus string
	ToStatus   string
	RunID      string
}

// emitStageTransition emits the run.stage.transition slog event at INFO level.
// No-ops when obs.Enabled is false.
func emitStageTransition(ctx context.Context, obs config.ObservabilityConfig, p stageTransitionParams) {
	if !obs.Enabled {
		return
	}
	slog.InfoContext(ctx, "run.stage.transition",
		"entity_key", p.EntityKey,
		"from_status", p.FromStatus,
		"to_status", p.ToStatus,
		"run_id", p.RunID,
	)
}

// stageErrorParams bundles the fields emitted on run.stage.error.
// Stdout / Stderr carry the RAW captured output from the agent — truncation
// is applied inside emitStageError using obs.GetLogTruncateBytes().
//
// Phase is a mandatory semantic label identifying where in the run loop the
// error occurred (e.g. "dispatch", "transition", "advance_status",
// "action_lookup", "context", "placeholders", "unknown_action",
// "dispatcher_selection", "post_dispatch"). Error is the error message
// string — also mandatory — which provides a machine-readable failure reason.
type stageErrorParams struct {
	EntityKey      string
	Status         string
	Phase          string
	Error          string
	ExitCode       int
	Stderr         string
	Stdout         string
	Command        string
	RunID          string
	TranscriptPath string
}

// emitStageError emits the run.stage.error slog event at ERROR level. It
// applies byte-budget truncation to stderr (head) and stdout_tail (tail of
// stdout) using obs.GetLogTruncateBytes(), and emits the "truncated" attribute
// (= true) ONLY when truncation actually occurred. When no truncation
// happens, the "truncated" attribute is omitted entirely.
// No-ops when obs.Enabled is false.
func emitStageError(ctx context.Context, obs config.ObservabilityConfig, p stageErrorParams) {
	if !obs.Enabled {
		return
	}
	limit := obs.GetLogTruncateBytes()
	stderr, tErr := limitPrefix(p.Stderr, limit)
	stdoutTail, tOut := limitSuffix(p.Stdout, limit)

	attrs := []any{
		"entity_key", p.EntityKey,
		"status", p.Status,
		"phase", p.Phase,
		"error", p.Error,
		"exit_code", p.ExitCode,
		"stderr", stderr,
		"stdout_tail", stdoutTail,
		"command", p.Command,
		"run_id", p.RunID,
	}
	if tErr || tOut {
		attrs = append(attrs, "truncated", true)
	}
	if p.TranscriptPath != "" {
		attrs = append(attrs, "transcript_path", p.TranscriptPath)
	}
	slog.ErrorContext(ctx, "run.stage.error", attrs...)
}

// emitTranscriptWarning emits the run.transcript.warning slog event at WARN
// level. This event is emitted at most ONCE per run whenever writing a
// per-dispatch transcript file fails. Subsequent write failures in the same
// run are suppressed by the caller's run-scoped disable flag, so this
// helper does not deduplicate on its own.
//
// Attributes:
//   - run_id: the run identifier so operators can correlate with other events.
//   - path:   the PROJECT-RELATIVE path that the write attempt targeted (may
//     still be useful even when MkdirAll failed before the file itself was
//     touched).
//   - error:  the error message returned by the failing write.
//
// No-ops when obs.Enabled is false.
func emitTranscriptWarning(ctx context.Context, obs config.ObservabilityConfig, runID, path string, err error) {
	if !obs.Enabled {
		return
	}
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	slog.WarnContext(ctx, "run.transcript.warning",
		"run_id", runID,
		"path", path,
		"error", errMsg,
	)
}

// limitPrefix returns up to the first `limit` bytes of s and a flag that is
// true when truncation occurred. When limit <= 0 or s is shorter than limit,
// s is returned unchanged and the flag is false. The result is guaranteed to
// be valid UTF-8: if the byte-cut lands inside a multi-byte rune, the partial
// rune at the end is dropped.
func limitPrefix(s string, limit int) (string, bool) {
	if limit <= 0 || len(s) <= limit {
		return s, false
	}
	i := limit
	for i > 0 && !utf8.RuneStart(s[i]) {
		i--
	}
	return s[:i], true
}

// limitSuffix returns up to the last `limit` bytes of s and a flag that is
// true when truncation occurred. When limit <= 0 or s is shorter than limit,
// s is returned unchanged and the flag is false. The result is guaranteed to
// be valid UTF-8: if the byte-cut lands inside a multi-byte rune, the partial
// rune at the beginning is dropped.
func limitSuffix(s string, limit int) (string, bool) {
	if limit <= 0 || len(s) <= limit {
		return s, false
	}
	i := len(s) - limit
	for i < len(s) && !utf8.RuneStart(s[i]) {
		i++
	}
	return s[i:], true
}
