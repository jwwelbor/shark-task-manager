package gatepersist

import (
	"context"
	"encoding/json"
	"fmt"

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
	Lease      LeaseReleaser
}

// NewCoordinator constructs a Coordinator. Every dependency is required —
// this coordinator is the only caller of note/kickback/transition/release
// APIs for a gate result (per its component-boundary contract), so a
// missing dependency is a wiring bug that must fail at construction, not
// silently degrade at persist time.
func NewCoordinator(notes NoteWriter, noteReader NoteReader, history HistoryReader, validator StatusValidator, transition Transitioner, lease LeaseReleaser) *Coordinator {
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
	case lease == nil:
		panic("gatepersist: NewCoordinator requires a non-nil LeaseReleaser")
	}
	return &Coordinator{
		Notes:      notes,
		NoteReader: noteReader,
		History:    history,
		Validator:  validator,
		Transition: transition,
		Lease:      lease,
	}
}

// Persist runs the full REQ-F-002/REQ-F-003 sequence for req: acquire the
// per-run lock, create-once the accepted result, initialize or verify the
// replay identity of the operation state, validate every kickback's
// target-entity workflow membership, reconcile and apply the six-step
// persistence order (skipping already-completed suboperations), apply the
// guarded main-entity transition exactly once, and release the lease only
// once RetirementConfirmed is set.
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

	kickbackEntityTypes, err := validateKickbacks(req.Result.Kickbacks, req.EntityKey, c.Validator)
	if err != nil {
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

	// Both "persistence just completed this call" and "already
	// persistence_complete from a prior call" resume here; both
	// "already transition_applied" also re-enters this idempotent call so
	// its only effect is the verify-no-write path EntityService.
	// TransitionStatus's own idempotency check guarantees (see the
	// Transitioner interface doc comment) — it never repeats a write.
	if state.PersistenceState == gaterun.PersistenceStateComplete || state.PersistenceState == gaterun.PersistenceStateTransitioned {
		reason := fmt.Sprintf("gate %s outcome %s", req.Gate, req.OutcomeKey)
		fromStatus, transitioned, err := c.Transition.Transition(ctx, req.EntityType, req.EntityKey, req.TargetStatus, reason, req.Session.Agent)
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
		released, err := c.Lease.Release(ctx, string(req.EntityType), req.EntityKey, req.Session.ID, req.OutcomeKey, false)
		if err != nil {
			return nil, fmt.Errorf("gatepersist: release lease: %w", err)
		}
		result.LeaseReleased = released
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
	reason := buildKickbackReason(k.Reason, subID)
	if _, _, err := c.Transition.Transition(ctx, entityType, k.EntityKey, k.TargetStatus, reason, req.Session.Agent); err != nil {
		return fmt.Errorf("gatepersist: apply kickback to %s: %w", k.EntityKey, err)
	}
	return nil
}
