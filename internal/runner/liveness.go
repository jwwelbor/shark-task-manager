// Package runner — run-liveness recorder (stage-boundary state machine,
// NDJSON/plain-text renderers, and the per-run run.log file sink).
//
// This file implements the LivenessRecorder introduced by E40-F04 (spec.md
// D1/D3/D4): a single-open-stage state machine that derives stage_start /
// heartbeat / stage_end events from the existing RunProgress callback stream,
// plus both D3 renderers (jsonMode=true NDJSON on stderr; plain text on
// stderr in jsonMode=false and unconditionally in run.log) and the D4
// per-event durable file sink.
//
// Scope of THIS file as of T-E40-F04-002:
//   - NewLivenessRecorder / Observe, implementing D1's phase table for the
//     "iteration" and "action" phases (other phases are ignored).
//   - The jsonMode=true NDJSON stderr renderer (D3's 11-field schema).
//   - The plain-text renderer (D3's fixed-column format), used for stderr in
//     jsonMode=false and always for run.log.
//   - LogPath() and the run.log file sink: per-event open-append-write-close
//     (D4), fail-soft on any error (REQ-N-002, matching
//     internal/observability/file_jsonl_exporter.go), permissions matching
//     transcript.go (0o755 dir / 0o644 file), and a no-op file sink when
//     projectRoot is empty (mirroring NewFileJSONLExporter("")).
//
// Deliberately NOT yet implemented here — added by later F04 tasks, tracked
// so an absence here is never mistaken for a design decision:
//   - The 10s heartbeat ticker, Start()/Stop(), and the run.log path
//     announcement (T-E40-F04-003).
//   - Finish() and the run_end summary line (T-E40-F04-004).
//
// REQ-N-004: this file and its test never touch controller.go or
// transcript.go — the only seam used is RunOptions.Progress's existing
// callback signature (RunProgress, defined in controller.go).
package runner

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Event names for the D3 NDJSON schema. The enum is closed at exactly three
// values (spec.md D3); eventHeartbeat is added in T-E40-F04-003 once the
// ticker exists to emit it.
const (
	eventStageStart = "stage_start"
	eventStageEnd   = "stage_end"
)

// ndjsonLine is the exact D3 wire schema for a single stderr NDJSON line.
// agent_type/provider are omitted (not emitted as empty strings) when unset;
// every other field is always present, even at its zero value, per D3
// ("always; \"\" when ...", "always; 0 when ...").
type ndjsonLine struct {
	TS             string `json:"ts"`
	RunID          string `json:"run_id"`
	Event          string `json:"event"`
	EntityKey      string `json:"entity_key"`
	Iteration      int    `json:"iteration"`
	Status         string `json:"status"`
	Action         string `json:"action"`
	AgentType      string `json:"agent_type,omitempty"`
	Provider       string `json:"provider,omitempty"`
	StageElapsedMs int64  `json:"stage_elapsed_ms"`
	TotalElapsedMs int64  `json:"total_elapsed_ms"`
}

// stageSlot is the single open-stage slot D1 describes. LivenessRecorder
// holds at most one at a time; cascade children run strictly sequentially
// (spec.md D1 — controller.go's handleCascade child loop has no goroutine
// fan-out), so a single slot is always sufficient to represent "the entity
// currently executing."
type stageSlot struct {
	entityKey    string
	iteration    int
	status       string
	action       string
	agentType    string
	provider     string
	openedAt     time.Time
	startEmitted bool
}

// LivenessRecorder derives shark run's stage-boundary liveness stream
// (stage_start / heartbeat / stage_end) from the RunOptions.Progress callback
// stream and renders it. See the package doc comment above for this file's
// current scope.
type LivenessRecorder struct {
	mu sync.Mutex

	projectRoot string
	runID       string
	topLevelKey string
	jsonMode    bool
	start       time.Time

	// logPath is the resolved absolute run.log path, computed once at
	// construction time. Empty projectRoot resolves to "" here, disabling the
	// file sink only (D5 — mirrors NewFileJSONLExporter("")). Immutable after
	// construction, so reads need no mutex.
	logPath string

	open *stageSlot
}

// NewLivenessRecorder constructs a LivenessRecorder. projectRoot, runID, and
// topLevelKey are recorded verbatim for use by later pieces of this file
// (the no-stage-open heartbeat default, added in T-E40-F04-003); jsonMode
// selects the D3 stderr renderer; start is the run's wall-clock start time,
// used as the reference point for total_elapsed_ms.
//
// projectRoot also determines the run.log file sink path (LogPath()): an
// empty projectRoot disables the file sink only, per D5/D6 edit 1.
//
// There is deliberately no duration/interval parameter: REQ-N-001 fixes the
// heartbeat cadence (added in T-E40-F04-003) at a literal 10 seconds,
// independent of any config or claim/lease TTL input.
func NewLivenessRecorder(projectRoot, runID, topLevelKey string, jsonMode bool, start time.Time) *LivenessRecorder {
	return &LivenessRecorder{
		projectRoot: projectRoot,
		runID:       runID,
		topLevelKey: topLevelKey,
		jsonMode:    jsonMode,
		start:       start,
		logPath:     resolveLogPath(projectRoot, runID),
	}
}

// resolveLogPath computes the absolute run.log path for the given
// projectRoot/runID, or "" when projectRoot is empty (file sink disabled).
func resolveLogPath(projectRoot, runID string) string {
	if projectRoot == "" {
		return ""
	}
	p := filepath.Join(projectRoot, ".shark", "runs", runID, "run.log")
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}

// LogPath returns the resolved absolute run.log path used by the file sink,
// or "" when the file sink is disabled (empty projectRoot at construction).
func (r *LivenessRecorder) LogPath() string {
	return r.logPath
}

// Observe implements spec.md D1's phase table for the two phases this task
// covers:
//
//	"iteration" — closes any open slot (emitting its stage_end, with a lazy
//	              stage_start first if that slot never reached the action
//	              phase), then opens a new slot bound to p.EntityKey.
//	"action"    — enriches the open slot with action/agent/provider and emits
//	              stage_start (idempotent — a no-op if already emitted).
//	other       — ignored, per D1.
//
// Observe is bound verbatim as opts.Progress = rec.Observe (T-E40-F04-005) —
// no adapter layer sits between the controller and this method, so its
// signature must match RunOptions.Progress exactly. Mutex-guarded per
// REQ-N-003: the heartbeat ticker goroutine (T-E40-F04-003) and the
// controller's Progress callback touch this same slot state concurrently.
func (r *LivenessRecorder) Observe(p RunProgress) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	switch p.Phase {
	case "iteration":
		r.closeOpenLocked(now)
		r.open = &stageSlot{
			entityKey: p.EntityKey,
			iteration: p.Iteration,
			status:    p.Status,
			openedAt:  now,
		}
	case "action":
		if r.open == nil {
			// controller.go always emits "iteration" before "action" for the
			// same stage (Run()'s loop body, controller.go:423-491); guard
			// defensively rather than panic if that invariant is ever
			// violated by a future caller.
			return
		}
		r.open.action = p.Action
		r.open.agentType = p.AgentType
		r.open.provider = p.Provider
		r.emitStageStartLocked(now)
	default:
		// All other phases (placeholders, action_lookup, context, ...) are
		// ignored per D1's phase table.
	}
}

// closeOpenStage closes the currently open stage slot, if any, using the
// identical close logic Observe's "iteration" case uses. It exists so D1's
// "run end" row ("close the open slot" -> stage_end) can be exercised ahead
// of Finish() (T-E40-F04-004), which will call this same method rather than
// duplicate the close logic.
func (r *LivenessRecorder) closeOpenStage() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closeOpenLocked(time.Now())
}

// closeOpenLocked closes r.open, if set: emitting a lazy stage_start first
// when the slot never reached the action phase, then stage_end, then
// clearing r.open. A no-op when no slot is open. Callers must hold r.mu.
func (r *LivenessRecorder) closeOpenLocked(now time.Time) {
	if r.open == nil {
		return
	}
	r.emitStageStartLocked(now)
	r.emitLocked(now, eventStageEnd)
	r.open = nil
}

// emitStageStartLocked emits stage_start for the open slot exactly once:
// subsequent calls (e.g. the lazy-start call from closeOpenLocked, after an
// action phase already emitted it) are no-ops. Callers must hold r.mu.
func (r *LivenessRecorder) emitStageStartLocked(now time.Time) {
	if r.open == nil || r.open.startEmitted {
		return
	}
	r.open.startEmitted = true
	r.emitLocked(now, eventStageStart)
}

// emitLocked renders and writes one event for the currently open slot, to
// both sinks: stderr (NDJSON in jsonMode, plain text otherwise — D3) and
// run.log (always plain text — D3, "run.log in both modes"). Callers must
// hold r.mu and only call this when r.open != nil.
func (r *LivenessRecorder) emitLocked(now time.Time, event string) {
	if r.open == nil {
		return
	}

	line := ndjsonLine{
		TS:             now.UTC().Format(time.RFC3339Nano),
		RunID:          r.runID,
		Event:          event,
		EntityKey:      r.open.entityKey,
		Iteration:      r.open.iteration,
		Status:         r.open.status,
		Action:         r.open.action,
		AgentType:      r.open.agentType,
		Provider:       r.open.provider,
		StageElapsedMs: now.Sub(r.open.openedAt).Milliseconds(),
		TotalElapsedMs: now.Sub(r.start).Milliseconds(),
	}

	plain := renderPlainLine(line)

	if r.jsonMode {
		// Fixed struct of only strings/ints; json.Marshal should never fail
		// here. Fail soft (skip the stderr write) rather than risk aborting
		// the run over a display concern (REQ-N-002).
		if data, err := json.Marshal(line); err == nil {
			fmt.Fprintf(os.Stderr, "%s\n", data)
		}
	} else {
		fmt.Fprintf(os.Stderr, "%s\n", plain)
	}

	r.writeLogLine(plain)
}

// renderPlainLine renders one event in D3's plain-text format:
//
//	<ts>  <event>  <entity_key>  key=value key=value ...
//
// Fixed leading columns (ts, event, entity_key) are each separated by two
// spaces, INCLUDING between entity_key and the key=value block itself — D3's
// worked example (`...T-E40-F04-003  status=in_development...`) confirms two
// spaces there too, not one. key=value pairs are then single-space-separated
// among themselves, in the order status, action, agent, provider, stage,
// total. A key whose value is empty is omitted entirely (D3) — this differs
// from the NDJSON renderer, which always includes status/action even when
// "".
func renderPlainLine(line ndjsonLine) string {
	var sb strings.Builder
	sb.WriteString(line.TS)
	sb.WriteString("  ")
	sb.WriteString(line.Event)
	sb.WriteString("  ")
	sb.WriteString(line.EntityKey)

	tokens := make([]string, 0, 6)
	tokens = appendPlainToken(tokens, "status", line.Status)
	tokens = appendPlainToken(tokens, "action", line.Action)
	tokens = appendPlainToken(tokens, "agent", line.AgentType)
	tokens = appendPlainToken(tokens, "provider", line.Provider)
	tokens = appendPlainToken(tokens, "stage", formatElapsed(time.Duration(line.StageElapsedMs)*time.Millisecond))
	tokens = appendPlainToken(tokens, "total", formatElapsed(time.Duration(line.TotalElapsedMs)*time.Millisecond))

	if len(tokens) > 0 {
		sb.WriteString("  ")
		sb.WriteString(strings.Join(tokens, " "))
	}

	return sb.String()
}

// appendPlainToken appends "key=value" to tokens, unless value is empty, in
// which case the key is omitted entirely (D3's empty-valued-key rule).
func appendPlainToken(tokens []string, key, value string) []string {
	if value == "" {
		return tokens
	}
	return append(tokens, key+"="+value)
}

// formatElapsed renders a duration truncated to whole seconds as
// "<h>h<mm>m<ss>s" / "<m>m<ss>s" / "<s>s", zero-padding minutes/seconds to
// two digits whenever a larger unit is present (matching spec.md D3's
// worked example: 74213ms -> "1m14s", 181940ms -> "3m01s"). Never empty, so
// stage=/total= are always present.
func formatElapsed(d time.Duration) string {
	total := int64(d / time.Second)
	if total < 0 {
		total = 0
	}
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	switch {
	case h > 0:
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
	case m > 0:
		return fmt.Sprintf("%dm%02ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
	}
}

// writeLogLine appends one already-rendered plain-text line to run.log,
// open-append-write-close per event (D4 — no bufio.Writer, no deferred
// flush): this is what survives SIGKILL (REQ-F-010). A no-op when the file
// sink is disabled (r.logPath == "", empty projectRoot at construction).
//
// Fail-soft per REQ-N-002/REQ-F-011: every failure (MkdirAll, OpenFile,
// Write, Close) is logged at slog.Debug and swallowed, matching
// internal/observability/file_jsonl_exporter.go's convention exactly. A
// liveness sink failure never surfaces an error to Observe's caller.
func (r *LivenessRecorder) writeLogLine(line string) {
	if r.logPath == "" {
		return
	}

	dir := filepath.Dir(r.logPath)
	if err := os.MkdirAll(dir, transcriptDirMode); err != nil {
		slog.Debug("liveness: failed to create run.log dir", "dir", dir, "err", err)
		return
	}

	f, err := os.OpenFile(r.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, transcriptFileMode)
	if err != nil {
		slog.Debug("liveness: failed to open run.log", "path", r.logPath, "err", err)
		return
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			slog.Debug("liveness: failed to close run.log", "path", r.logPath, "err", cerr)
		}
	}()

	if _, err := fmt.Fprintf(f, "%s\n", line); err != nil {
		slog.Debug("liveness: failed to write run.log line", "path", r.logPath, "err", err)
	}
}
