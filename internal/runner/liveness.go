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
// Scope of THIS file as of T-E40-F04-004:
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
//   - Start()/Stop() and the fixed 10s heartbeat ticker (REQ-N-001,
//     REQ-F-006): a separately callable unexported tick() method emits a
//     heartbeat for the open stage, or a top-level-key heartbeat when none is
//     open (spec.md's "decisions the implementer does not get to make"
//     table). Start() also announces LogPath() on stderr exactly once,
//     before any event line (REQ-F-008/AC-10 — this task's resolved
//     D6-edit-2 deviation; see the task spec's "Deviation" note).
//   - Finish() and the file-only run_end summary line (D1's "run end" row,
//     D3): closes any open stage, then appends run_end sourced from a
//     *RunResult when one is available, including the zero-Observe fallback
//     (AC-08 row 8b) the task spec resolves explicitly.
//
// This file is now feature-complete for LivenessRecorder itself; wiring it
// into run.go is T-E40-F04-005/006.
//
// REQ-N-004: this file and its test never touch controller.go or
// transcript.go — the only seam used is RunOptions.Progress's existing
// callback signature (RunProgress, defined in controller.go), plus reading
// RunResult's already-defined fields in Finish (no new field, no shape
// change).
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
// values (spec.md D3).
const (
	eventStageStart = "stage_start"
	eventStageEnd   = "stage_end"
	eventHeartbeat  = "heartbeat"
)

// heartbeatInterval is REQ-N-001's fixed heartbeat cadence: a literal 10
// seconds, independent of any claim/lease duration config and never derived
// from the claim service's own lease-length accessor (see the "two
// heartbeats" conflation this spec calls out).
const heartbeatInterval = 10 * time.Second

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

	// observed records whether Observe has ever been called, regardless of
	// phase or whether it ever opened a stage. Finish (T-E40-F04-004) uses
	// this to distinguish AC-08 row 8b (Finish(nil), zero Observe calls ever
	// made — nothing else will satisfy AC-06's non-empty-file invariant) from
	// AC-T2 (Finish(nil) after at least one observed stage, whether or not it
	// is still open — the closing stage_start/stage_end already satisfy it).
	observed bool

	// stopCh/stopOnce/wg implement Start()/Stop()'s ticker-goroutine
	// lifecycle (T-E40-F04-003). stopCh is always initialized by the
	// constructor, so Stop() is safe even when Start() was never called;
	// stopOnce guards against a double-close if Stop() is called more than
	// once.
	stopCh   chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
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
		stopCh:      make(chan struct{}),
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

	r.observed = true
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

// Finish implements D1's "run end" row: close any open stage (lazy
// stage_start + stage_end, via the same closeOpenStage logic T-E40-F04-001
// already exercises ahead of this method), then append the file-only
// run_end summary line (D3) — never written to stderr in either mode.
// Production calls this exactly once per run via run.go's
// `defer func() { rec.Stop(); rec.Finish(runResult) }()` (D6 edit 2); this
// method does not itself guard against a second call.
//
// D3 ties the run_end line to RunResult availability:
//
//   - result != nil (AC-T3/AC-T4, AC-08 rows 5/6/8a): render
//     outcome/final_status/stages/total from the result's real fields. The
//     zero-stage row-8a shape (already_terminal/paused before the first
//     iteration) renders exactly like the primary multi-stage shape — both
//     are real RunResults, so both render real fields, never the fallback
//     text below.
//   - result == nil and at least one Observe call was ever made (AC-T2): no
//     run_end line at all. The closing stage_start/stage_end already
//     satisfy AC-06's non-empty-file invariant, and fabricating an
//     "unknown" summary here would misrepresent a run whose stage we did in
//     fact observe.
//   - result == nil and ZERO Observe calls were ever made (AC-T1 — the task
//     spec's resolved AC-08 row 8b decision: controller.Run returned a bare
//     Go error before its first iteration, so run.go's liveness-teardown
//     defer never received a RunResult): closeOpenStage is a no-op — nothing
//     was ever opened — so nothing else would satisfy AC-06. Finish writes
//     exactly one synthetic run_end line sourced from topLevelKey with
//     outcome/final_status "unknown" and stages 0. No stage_start/stage_end
//     pair is fabricated to "make the file look normal."
func (r *LivenessRecorder) Finish(result *RunResult) {
	r.mu.Lock()
	observed := r.observed
	r.mu.Unlock()

	r.closeOpenStage()

	now := time.Now()
	switch {
	case result != nil:
		r.writeLogLine(renderRunEndLine(now, result.EntityKey, result.Outcome, result.FinalStatus, result.StagesCompleted, result.TotalDuration))
	case !observed:
		r.writeLogLine(renderRunEndLine(now, r.topLevelKey, "unknown", "unknown", 0, now.Sub(r.start)))
	}
}

// renderRunEndLine renders D3's file-only run_end summary line:
//
//	<ts>  run_end  <entity_key>  outcome=<outcome> final_status=<final_status> stages=<n> total=<elapsed>
//
// This is deliberately NOT an ndjsonLine render (D3: "not an event-struct
// render") — it carries a fixed, different key set (outcome/final_status/
// stages/total, never status/action/agent/provider/stage), and none of its
// four keys is ever omitted, even when the value is the literal placeholder
// "unknown" (Finish's row-8b fallback) or zero. It reuses renderPlainLine's
// column-spacing convention (two spaces between each fixed column, two
// spaces before the key=value block, single space between pairs) purely for
// visual consistency with the stage-event lines already in run.log.
func renderRunEndLine(ts time.Time, entityKey, outcome, finalStatus string, stages int, total time.Duration) string {
	return fmt.Sprintf("%s  run_end  %s  outcome=%s final_status=%s stages=%d total=%s",
		ts.UTC().Format(time.RFC3339Nano), entityKey, outcome, finalStatus, stages, formatElapsed(total))
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

	r.emitLine(ndjsonLine{
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
	})
}

// emitLine renders and writes one already-constructed event line to both
// sinks: stderr (NDJSON in jsonMode, plain text otherwise — D3) and run.log
// (always plain text — D3, "run.log in both modes"). Shared by emitLocked
// (open-slot events) and tick (the no-open-stage heartbeat, which has no
// slot to read from). Callers must hold r.mu.
func (r *LivenessRecorder) emitLine(line ndjsonLine) {
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

// Start announces the run.log path on stderr exactly once, before any event
// line (REQ-F-008/AC-10 — the task spec's resolved D6-edit-2 deviation: this
// prints from Start(), not run.go, per test-plan.md's "Recommendation: move
// the LogPath() print into Start()"), then launches the REQ-F-006 heartbeat
// ticker goroutine at the fixed heartbeatInterval. Production calls this
// exactly once per run (run.go D6 edit 2); a second call would start a
// second ticker goroutine and is not guarded against, matching Start()'s
// single-call contract.
func (r *LivenessRecorder) Start() {
	fmt.Fprintf(os.Stderr, "run.log: %s\n", r.logPath)

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		ticker := time.NewTicker(heartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-r.stopCh:
				return
			case <-ticker.C:
				r.tick()
			}
		}
	}()
}

// Stop signals the heartbeat goroutine to exit and waits for it to finish.
// Safe to call even when Start() was never called — stopCh is always
// initialized by the constructor, so closing it here is a harmless no-op
// with no goroutine listening — and safe to call more than once (stopOnce
// guards the channel close). T-E40-F04-004's teardown calls `rec.Stop();
// rec.Finish(...)` unconditionally, regardless of whether Start() ran.
func (r *LivenessRecorder) Stop() {
	r.stopOnce.Do(func() { close(r.stopCh) })
	r.wg.Wait()
}

// tick performs one heartbeat action (REQ-F-006/REQ-N-001): emits a
// heartbeat for the currently open stage, or — when no stage is open — a
// heartbeat carrying the top-level key with iteration 0 and empty
// status/action, per spec.md's "decisions the implementer does not get to
// make" table (a stall during that window would otherwise be invisible).
// Exposed as a separately callable unexported method (AC-T2) so tests can
// drive cadence deterministically without a real 10-second sleep; Start()'s
// ticker loop is the only production caller.
func (r *LivenessRecorder) tick() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if r.open != nil {
		r.emitLocked(now, eventHeartbeat)
		return
	}

	elapsed := now.Sub(r.start).Milliseconds()
	r.emitLine(ndjsonLine{
		TS:             now.UTC().Format(time.RFC3339Nano),
		RunID:          r.runID,
		Event:          eventHeartbeat,
		EntityKey:      r.topLevelKey,
		Iteration:      0,
		Status:         "",
		Action:         "",
		StageElapsedMs: elapsed,
		TotalElapsedMs: elapsed,
	})
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
