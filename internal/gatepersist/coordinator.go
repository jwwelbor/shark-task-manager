package gatepersist

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
)

// Coordinator is the T-E34-F05-003 parent-owned persistence coordinator. All
// side effects go through its injected interfaces; Coordinator itself never
// touches a database.
type Coordinator struct {
	Notes      NoteWriter
	NoteReader NoteReader
	History    HistoryReader
	Validator  StatusValidator
	Transition Transitioner
	Status     StatusReader
	Lease      LeaseReleaser
}

// NewCoordinator constructs a Coordinator. Every dependency is required —
// this coordinator is the only caller of note/kickback/transition/release
// APIs for a gate result (per its component-boundary contract), so a
// missing dependency is a wiring bug that must fail at construction, not
// silently degrade at persist time.
func NewCoordinator(notes NoteWriter, noteReader NoteReader, history HistoryReader, validator StatusValidator, transition Transitioner, status StatusReader, lease LeaseReleaser) *Coordinator {
	switch {
	case notes == nil:
		panic("gatepersist: NewCoordinator requires a non-nil NoteWriter")
	case noteReader == nil:
		panic("gatepersist: NewCoordinator requires a non-nil NoteReader")
	case history == nil:
		panic("gatepersist: NewCoordinator requires a non-nil HistoryReader")
	case validator == nil:
		panic("gatepersist: NewCoordinator requires a non-nil StatusValidator")
	case transition == nil:
		panic("gatepersist: NewCoordinator requires a non-nil Transitioner")
	case status == nil:
		panic("gatepersist: NewCoordinator requires a non-nil StatusReader")
	case lease == nil:
		panic("gatepersist: NewCoordinator requires a non-nil LeaseReleaser")
	}
	return &Coordinator{
		Notes:      notes,
		NoteReader: noteReader,
		History:    history,
		Validator:  validator,
		Transition: transition,
		Status:     status,
		Lease:      lease,
	}
}

// Persist runs the full REQ-F-002/REQ-F-003 sequence for req: acquire the
// per-run lock, create-once the accepted result, initialize or verify the
// replay identity of the operation state, validate every kickback's
// target-entity workflow membership, reconcile and apply the six-step
// persistence order (skipping already-completed suboperations), apply the
// guarded main-entity transition exactly once, and release the lease only
// once BOTH RetirementConfirmed and RunConcluded are set (see their doc
// comments on Request: worker-process retirement and run-conclusion are
// deliberately distinct signals — this coordinator does not infer one from
// the other, nor trust a caller that conflates them).
//
// Persist is safe to call again for the same run_id (see gaterun's
// create-once/atomic-replace sidecar contract): a resumed call with an
// identical Request completes whatever remains; a call with a differently-
// digested Request under the same run_id fails closed
// (gaterun.VerifyResumeIdentity / gaterun.IsConflict).
func (c *Coordinator) Persist(ctx context.Context, req Request) (*Result, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}

	lockTimeout := req.LockTimeout
	if lockTimeout <= 0 {
		lockTimeout = gaterun.DefaultLockTimeout
	}
	lock, err := gaterun.AcquireRunLock(req.RunDir, lockTimeout)
	if err != nil {
		return nil, fmt.Errorf("gatepersist: acquire run lock: %w", err)
	}
	defer func() { _ = lock.Release() }()

	// Kickback validation runs before this run_id ever accepts a durable
	// result: a run whose result would be rejected must not permanently burn
	// its run_id under gaterun.CreateResult's create-once/first-writer-wins
	// contract (there is no rewrite path — a corrected envelope needs a new
	// run_id otherwise). This is "rejected without partial mutation" applied
	// to the sidecar transport itself, not just target-store writes.
	kickbackEntityTypes, err := validateKickbacks(req.Result.Kickbacks, req.EntityKey, c.Validator)
	if err != nil {
		return nil, err
	}

	digest, err := gaterun.ComputeOperationDigest(req.EntityKey, string(req.EntityType), req.SourceStatus, req.Gate, req.EnvelopeJSON)
	if err != nil {
		return nil, fmt.Errorf("gatepersist: compute operation digest: %w", err)
	}

	if _, err := gaterun.CreateResult(req.RunDir, []byte(req.EnvelopeJSON)); err != nil {
		return nil, err
	}

	state, exists, err := gaterun.LoadOperationState(req.RunDir)
	if err != nil {
		return nil, fmt.Errorf("gatepersist: load operation state: %w", err)
	}
	if !exists {
		state = gaterun.NewOperationState(req.RunID, req.EntityKey, string(req.EntityType), req.SourceStatus, req.Gate, digest)
		if err := state.Save(req.RunDir); err != nil {
			return nil, fmt.Errorf("gatepersist: initialize operation state: %w", err)
		}
	} else if err := gaterun.VerifyResumeIdentity(state, req.EntityKey, string(req.EntityType), req.SourceStatus, digest); err != nil {
		return nil, err
	}

	ops := buildOperations(req.Result, summaryFrom(req.Gate, req.Result.Summary))
	result := &Result{OperationDigest: digest}

	if state.PersistenceState == gaterun.PersistenceStatePending {
		rec := &reconciler{
			noteReader:       c.NoteReader,
			historyReader:    c.History,
			mainEntityType:   req.EntityType,
			mainEntityKey:    req.EntityKey,
			kickbackEntities: kickbackEntityTypes,
			ops:              ops,
			operationDigest:  digest,
		}
		changed, err := gaterun.ReconcileCompletedSuboperations(ctx, rec, state)
		if err != nil {
			return nil, err
		}
		if changed {
			if err := state.Save(req.RunDir); err != nil {
				return nil, fmt.Errorf("gatepersist: save reconciled operation state: %w", err)
			}
		}

		for _, op := range ops {
			subID := op.suboperationID(digest)
			if state.HasCompleted(subID) {
				continue
			}
			if err := c.applyOperation(ctx, op, subID, digest, req); err != nil {
				return nil, err
			}
			state.AddCompletedSuboperation(subID)
			if err := state.Save(req.RunDir); err != nil {
				return nil, fmt.Errorf("gatepersist: save operation state after suboperation %s: %w", subID, err)
			}
		}

		if err := state.MarkPersistenceComplete(); err != nil {
			return nil, err
		}
		if err := state.Save(req.RunDir); err != nil {
			return nil, fmt.Errorf("gatepersist: save operation state after persistence complete: %w", err)
		}
	}

	result.CompletedSuboperations = append([]string(nil), state.CompletedSuboperationIDs...)
	result.PersistenceComplete = state.PersistenceState == gaterun.PersistenceStateComplete || state.PersistenceState == gaterun.PersistenceStateTransitioned

	// "persistence just completed this call" and "already persistence_complete
	// from a prior call" both apply the guarded transition exactly once here.
	// "already transition_applied" (a prior call already recorded it) must
	// NOT repeat the transition call (architecture.md step 8: "it must not
	// repeat the transition") — it only verifies the expected live target
	// state via StatusReader and fails closed on any mismatch, since nothing
	// guarantees the entity is still where this run last left it (a human
	// `status set --force`, a cascade, or a concurrent run could have moved
	// it since).
	switch state.PersistenceState {
	case gaterun.PersistenceStateComplete:
		// F-3: verify the entity is still where this run left it before
		// transitioning "exactly once from the recorded source" (REQ-F-002).
		// Accept either SourceStatus (the normal case) or TargetStatus
		// (a resumed call landing in the window between Transition
		// returning below and MarkTransitionApplied's state.Save — the
		// transition already applied, the sidecar just hasn't caught up
		// yet) and fail closed on anything else, mirroring the
		// PersistenceStateTransitioned verification below rather than
		// trusting Transitioner's own idempotency to stand in for it.
		current, err := c.Status.CurrentStatus(ctx, req.EntityType, req.EntityKey)
		if err != nil {
			return nil, fmt.Errorf("gatepersist: verify source status before main transition: %w", err)
		}
		if !strings.EqualFold(current, req.SourceStatus) && !strings.EqualFold(current, req.TargetStatus) {
			return nil, fmt.Errorf("gatepersist: entity %s is recorded at source status %q but is currently at %q; refusing to transition from an unrecorded source", req.EntityKey, req.SourceStatus, current)
		}

		reason := fmt.Sprintf("gate %s outcome %s", req.Gate, req.OutcomeKey)
		guard := TransitionGuard{
			SessionID:  req.Session.ID,
			FromStatus: req.SourceStatus,
			Outcome:    req.OutcomeKey,
		}
		fromStatus, transitioned, err := c.Transition.Transition(ctx, req.EntityType, req.EntityKey, req.TargetStatus, reason, req.Session.Agent, guard)
		if err != nil {
			return nil, fmt.Errorf("gatepersist: apply main transition: %w", err)
		}
		result.FromStatus = fromStatus
		result.Transitioned = transitioned

		if err := state.MarkTransitionApplied(); err != nil {
			return nil, err
		}
		if err := state.Save(req.RunDir); err != nil {
			return nil, fmt.Errorf("gatepersist: save operation state after transition applied: %w", err)
		}
	case gaterun.PersistenceStateTransitioned:
		current, err := c.Status.CurrentStatus(ctx, req.EntityType, req.EntityKey)
		if err != nil {
			return nil, fmt.Errorf("gatepersist: verify already-applied transition target: %w", err)
		}
		if !strings.EqualFold(current, req.TargetStatus) {
			return nil, fmt.Errorf("gatepersist: entity %s is recorded transition_applied to %q but is currently at %q; refusing to repeat or silently diverge from the recorded transition", req.EntityKey, req.TargetStatus, current)
		}
		result.FromStatus = state.SourceStatus
	}
	result.ToStatus = req.TargetStatus
	result.TransitionApplied = state.PersistenceState == gaterun.PersistenceStateTransitioned

	if req.RetirementConfirmed {
		if state.RetirementState != gaterun.RetirementRetired {
			state.RetirementState = gaterun.RetirementRetired
			if err := state.Save(req.RunDir); err != nil {
				return nil, fmt.Errorf("gatepersist: save retirement state: %w", err)
			}
		}
		// The lease itself is released only once RunConcluded ALSO holds —
		// RetirementConfirmed alone only proves this call's dispatched
		// worker process exited, which is true for every stage of a
		// multi-stage run (see Request.RetirementConfirmed's doc comment).
		// A caller that (incorrectly) passes RetirementConfirmed per-stage
		// without ever setting RunConcluded simply never releases via this
		// coordinator — safe by construction, since a stale/leaked lease
		// still expires via the claim TTL backstop, whereas a premature
		// release let a concurrent claimant race the still-running loop.
		if req.RunConcluded {
			released, err := c.Lease.Release(ctx, string(req.EntityType), req.EntityKey, req.Session.ID, req.OutcomeKey, false)
			if err != nil {
				return nil, fmt.Errorf("gatepersist: release lease: %w", err)
			}
			result.LeaseReleased = released
		}
	} else if state.RetirementState == gaterun.RetirementUnknown {
		state.RetirementState = gaterun.RetirementPending
		if err := state.Save(req.RunDir); err != nil {
			return nil, fmt.Errorf("gatepersist: save retirement state: %w", err)
		}
	}

	return result, nil
}

// applyOperation performs one target write: a typed note (gate summary,
// finding, sweep, or impact) or a kickback transition.
func (c *Coordinator) applyOperation(ctx context.Context, op operation, subID, digest string, req Request) error {
	if op.kind == kindKickback {
		return c.applyKickback(ctx, op, subID, req)
	}
	return c.writeNote(ctx, op, subID, digest, req)
}

func (c *Coordinator) writeNote(ctx context.Context, op operation, subID, digest string, req Request) error {
	meta := make(map[string]interface{}, len(op.metadata)+6)
	for k, v := range op.metadata {
		meta[k] = v
	}
	meta[metaRunID] = req.RunID
	meta[metaSuboperationID] = subID
	meta[metaOperationDigest] = digest
	meta[metaGate] = req.Gate
	meta[metaParentSession] = req.Session.ID
	meta[metaContentDigest] = op.contentDigest()
	if op.kind == kindGateSummary {
		meta[metaOutcomeKey] = req.OutcomeKey
		meta[metaRole] = string(req.Role)
		if len(req.Evidence) > 0 {
			var evidence interface{}
			if err := json.Unmarshal(req.Evidence, &evidence); err != nil {
				return fmt.Errorf("gatepersist: decode evidence for gate-summary note: %w", err)
			}
			meta[metaEvidence] = evidence
		}
	}

	encoded, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("gatepersist: encode note metadata for %s %q: %w", op.kind, op.itemIdentity, err)
	}

	if _, err := c.Notes.AddNoteWithMetadata(ctx, req.EntityType, req.EntityKey, op.noteType, op.content, req.Session.Agent, string(encoded)); err != nil {
		return fmt.Errorf("gatepersist: write %s note for %q: %w", op.kind, op.itemIdentity, err)
	}
	return nil
}

func (c *Coordinator) applyKickback(ctx context.Context, op operation, subID string, req Request) error {
	k := op.kickback
	entityType, err := kickbackEntityType(k.EntityKey)
	if err != nil {
		return err
	}
	// The kickback target (often a different entity than req.EntityKey) has
	// no parent-observed SourceStatus the way the main transition does, so
	// its expected pre-transition status is read fresh here, immediately
	// before the guarded Transition call — the same narrow
	// observe-then-transition window guardedTransitionOptions accepts for
	// the runner's own dispatch-loop transitions.
	fromStatus, err := c.Status.CurrentStatus(ctx, entityType, k.EntityKey)
	if err != nil {
		return fmt.Errorf("gatepersist: read kickback target status for %s: %w", k.EntityKey, err)
	}
	reason := buildKickbackReason(k.Reason, subID, op.contentDigest())
	guard := TransitionGuard{
		SessionID:  req.Session.ID,
		FromStatus: fromStatus,
		Outcome:    req.OutcomeKey,
	}
	if _, _, err := c.Transition.Transition(ctx, entityType, k.EntityKey, k.TargetStatus, reason, req.Session.Agent, guard); err != nil {
		return fmt.Errorf("gatepersist: apply kickback to %s: %w", k.EntityKey, err)
	}
	return nil
}
