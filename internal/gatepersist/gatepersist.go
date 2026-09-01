// Package gatepersist implements the T-E34-F05-003 parent-owned GateResult
// persistence coordinator (REQ-F-002): it turns a validated
// internal/gateresult.GateResult into durable Shark notes, sweep/impact
// references, and task kickbacks, in the feature-defined idempotent order,
// before any workflow transition. See
// docs/plan/E34-prompt-and-skill-improvements/architecture.md "Gate result
// flow" (steps 5-8) and E34-F05's feature.md REQ-F-002/REQ-F-003.
//
// Scope boundaries (see the task spec's Notes/Dependencies):
//   - This package owns entity/run/session binding, replay state, the
//     six-step persistence order, kickback application (including the
//     target-entity workflow-membership check gateresult's package doc
//     defers here), the guarded main-entity transition, and lease release
//     gated on worker-retirement evidence.
//   - It does NOT own gate reasoning, workflow outcome-role definitions
//     (that is internal/gateresult.ValidateRole, called by the caller before
//     Persist), the sidecar transport/digest/suboperation-ID primitives
//     (internal/gaterun, consumed here, not reimplemented), or Rider/core
//     runner CLI wiring (`shark run --apply-result`/`--resume-run` is
//     T-E34-F05-004's job).
//   - It accepts internal/gateresult.GateResult as its only input shape; it
//     never re-derives or trusts unvalidated worker output.
//
// Every side effect this package performs (notes, kickback transitions, the
// main transition, lease release) goes through a small caller-injected
// interface (see interfaces.go) so tests can substitute fakes instead of a
// real database, and so the concrete adapters (adapters.go) can be reused by
// whichever caller wires this coordinator to `internal/services` without
// this package importing `internal/cli`.
package gatepersist

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// Session identifies the parent-owned claim/lease session driving this
// persistence run. The session ID is associated provenance (recorded on
// writes for audit), never the replay identity (architecture.md step 4).
type Session struct {
	// ID is the authorized claim/lease session ID (e.g. from `shark claim`).
	ID string
	// Agent is the acting agent identity recorded on notes/history.
	Agent string
}

// Request binds one validated GateResult to the parent-observed entity,
// route, and durable run identity it must be persisted and transitioned
// against. Every field here is parent-observed; none are trusted from
// worker output (architecture.md step 4).
type Request struct {
	// RunDir is the durable run directory (see gaterun.RunDir) this run's
	// result.json/operation-state.json sidecars live under.
	RunDir string
	// RunID is the durable run identity (see gaterun.ValidateRunID).
	RunID string

	// EntityKey and EntityType identify the bound main entity the parent
	// observed this gate result for.
	EntityKey  string
	EntityType models.EntityType
	// SourceStatus is the entity's status at dispatch time (the guarded
	// source of the eventual transition).
	SourceStatus string
	// Gate names the observed gate (e.g. "code_review", "qa", "uat").
	Gate string

	// Session is the current authorized claim/lease session.
	Session Session

	// EnvelopeJSON is the exact accepted outer worker-control envelope bytes
	// (kind: final, including recommended_outcome, evidence, and the nested
	// gate_result). It is opaque to this package except as digest/result.json
	// input — see gaterun.ComputeOperationDigest and gaterun.CreateResult.
	EnvelopeJSON json.RawMessage

	// Result is the already-decoded, already-validated (Decode + Validate +
	// ValidateRole) nested GateResult payload from EnvelopeJSON. It MUST
	// correspond exactly to EnvelopeJSON; a caller resuming a prior run must
	// reconstruct both fields from the accepted result.json, never from a
	// fresh worker response (see gaterun.CreateResult's no-replace,
	// first-writer-wins conflict semantics, which this package relies on to
	// reject a non-identical replay).
	Result *gateresult.GateResult
	// Role is the semantic outcome role the parent's workflow configuration
	// assigned to OutcomeKey (REQ-F-006's outcome_roles map).
	Role gateresult.OutcomeRole
	// OutcomeKey is the opaque configured outcome the worker returned.
	OutcomeKey string

	// Evidence is the outer worker-control envelope's common EvidenceRef
	// collection (architecture.md "The outer final envelope's EvidenceRef
	// contains kind, pointer, and an optional bounded summary..."), already
	// decoded and bounded by the caller's envelope validation (this package
	// does not parse the outer envelope — see gatepersist.go's package doc).
	// When non-empty it is folded into the gate-summary note's metadata, so
	// the persisted "review" note carries the summary AND its evidence per
	// REQ-F-002's "gate-summary/evidence review note." It is opaque JSON
	// here rather than a typed slice because this package intentionally has
	// no EvidenceRef type of its own to avoid duplicating the outer
	// envelope's shape ahead of T-E34-F05-004's parity work.
	Evidence json.RawMessage

	// TargetStatus is the resolved main-entity transition target for
	// OutcomeKey, already resolved by the caller from workflow configuration
	// (this package never selects a status from an opaque outcome key).
	TargetStatus string

	// RetirementConfirmed attests that terminal worker-retirement evidence
	// (independent of chat notifications) has been observed for this run.
	// The lease is released only when true (architecture.md step 8);
	// otherwise Persist completes the transition (if due) and leaves release
	// pending for a later call once retirement is confirmed.
	RetirementConfirmed bool

	// LockTimeout overrides the default run-lock acquisition timeout
	// (gaterun.DefaultLockTimeout) when non-zero.
	LockTimeout time.Duration
}

// Result reports what Persist actually did, for the caller's own
// diagnostics/logging. It is not itself part of the replay contract — the
// durable operation-state.json is.
type Result struct {
	OperationDigest        string
	CompletedSuboperations []string
	PersistenceComplete    bool
	TransitionApplied      bool
	FromStatus             string
	ToStatus               string
	Transitioned           bool
	LeaseReleased          bool
}

// validate checks the structural preconditions Persist requires before
// touching the filesystem or any injected side effect.
func (r *Request) validate() error {
	if r.RunDir == "" {
		return fmt.Errorf("gatepersist: run dir is required")
	}
	if r.RunID == "" {
		return fmt.Errorf("gatepersist: run_id is required")
	}
	if r.EntityKey == "" {
		return fmt.Errorf("gatepersist: entity key is required")
	}
	if r.EntityType == "" {
		return fmt.Errorf("gatepersist: entity type is required")
	}
	if r.Gate == "" {
		return fmt.Errorf("gatepersist: gate is required")
	}
	if len(r.EnvelopeJSON) == 0 {
		return fmt.Errorf("gatepersist: envelope JSON is required")
	}
	if r.Result == nil {
		return fmt.Errorf("gatepersist: a decoded GateResult is required")
	}
	if r.TargetStatus == "" {
		return fmt.Errorf("gatepersist: target status is required")
	}
	if r.Session.ID == "" {
		return fmt.Errorf("gatepersist: session id is required")
	}
	return nil
}
