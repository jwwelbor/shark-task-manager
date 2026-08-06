// Package runner — run-liveness recorder (stage-boundary state machine and
// NDJSON stderr renderer).
//
// This file implements the LivenessRecorder introduced by E40-F04 (spec.md
// D1/D3): a single-open-stage state machine that derives stage_start /
// heartbeat / stage_end events from the existing RunProgress callback stream,
// plus the --json-mode NDJSON renderer for those events on stderr.
//
// Scope of THIS file as of T-E40-F04-001 (foundation only):
//   - NewLivenessRecorder / Observe, implementing D1's phase table for the
//     "iteration" and "action" phases (other phases are ignored).
//   - The jsonMode=true NDJSON stderr renderer (D3's 11-field schema).
//
// Deliberately NOT yet implemented here — added by later F04 tasks, tracked
// so an absence here is never mistaken for a design decision:
//   - Plain-mode (jsonMode=false) rendering and the run.log file sink
//     (T-E40-F04-002).
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
	"os"
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

	open *stageSlot
}

// NewLivenessRecorder constructs a LivenessRecorder. projectRoot, runID, and
// topLevelKey are recorded verbatim for use by later pieces of this file
// (the run.log file sink and the no-stage-open heartbeat default,
// respectively); jsonMode selects the D3 renderer; start is the run's
// wall-clock start time, used as the reference point for total_elapsed_ms.
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
	}
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

// emitLocked renders and writes one event for the currently open slot.
// Callers must hold r.mu and only call this when r.open != nil.
func (r *LivenessRecorder) emitLocked(now time.Time, event string) {
	if r.open == nil {
		return
	}
	if !r.jsonMode {
		// Plain-mode rendering lands in T-E40-F04-002.
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
	data, err := json.Marshal(line)
	if err != nil {
		// Fixed struct of only strings/ints; should never fail. Fail soft
		// rather than risk aborting the run over a display concern
		// (REQ-N-002).
		return
	}
	fmt.Fprintf(os.Stderr, "%s\n", data)
}
