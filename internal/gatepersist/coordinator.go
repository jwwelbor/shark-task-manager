package gatepersist

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
	"github.com/jwwelbor/shark-task-manager/internal/models"
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
	// Identity is the authoritative, repository-backed "same entity" check
	// validateKickbacks uses for its round-12 self-kickback comparison (see
	// IdentityResolver's doc comment). Required, not optional: unlike
	// ClaimVerifier's TOCTOU re-check (a NEW protection where nil safely
	// reproduces prior behavior), a nil Identity here would silently fall
	// back to the Normalize-only comparison round 12 proved insufficient —
	// a documented path back to the exact vulnerability being fixed.
	Identity IdentityResolver

	// ClaimVerifier, when set, re-verifies Request.Session's claim ownership
	// immediately after Persist acquires the per-run lock (UAT round-2
	// Finding 1). See its doc comment (interfaces.go) for why this is a
	// separate, optional field rather than a NewCoordinator parameter: it
	// defaults to nil so every existing caller/test keeps today's behavior
	// unchanged, and production wiring (internal/cli/commands.buildGateCoordinator)
	// sets it explicitly.
	ClaimVerifier ClaimVerifier
}

// NewCoordinator constructs a Coordinator. Every dependency is required —
// this coordinator is the only caller of note/kickback/transition/release
// APIs for a gate result (per its component-boundary contract), so a
// missing dependency is a wiring bug that must fail at construction, not
// silently degrade at persist time.
func NewCoordinator(notes NoteWriter, noteReader NoteReader, history HistoryReader, validator StatusValidator, transition Transitioner, status StatusReader, lease LeaseReleaser, identity IdentityResolver) *Coordinator {
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
	case identity == nil:
		panic("gatepersist: NewCoordinator requires a non-nil IdentityResolver")
	}
	return &Coordinator{
		Notes:      notes,
		NoteReader: noteReader,
		History:    history,
		Validator:  validator,
		Transition: transition,
		Status:     status,
		Lease:      lease,
		Identity:   identity,
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

	lock, err := c.acquireLock(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = lock.Release() }()

	// UAT round-2 Finding 1: re-verify the claim/lease session INSIDE the
	// per-run lock's critical section, immediately before any write below
	// (CreateResult, note/kickback writes, the guarded transition) — closing
	// the TOCTOU window between run.go's one-time verifyClaimSession check
	// and these actual mutating calls. See ClaimVerifier's doc comment for
	// why a nil ClaimVerifier skips this (back-compat for existing callers).
	if err := c.verifyClaimSession(ctx, req); err != nil {
		return nil, err
	}

	digest, kickbackEntityTypes, state, err := c.initializeRun(ctx, req)
	if err != nil {
		return nil, err
	}

	ops := buildOperations(req.Result, summaryFrom(req.Gate, req.Result.Summary))
	result := &Result{OperationDigest: digest}

	if state.PersistenceState == gaterun.PersistenceStatePending {
		if err := c.reconcileAndApplyOperations(ctx, req, state, ops, digest, kickbackEntityTypes); err != nil {
			return nil, err
		}
	}

	result.CompletedSuboperations = append([]string(nil), state.CompletedSuboperationIDs...)
	result.PersistenceComplete = state.PersistenceState == gaterun.PersistenceStateComplete || state.PersistenceState == gaterun.PersistenceStateTransitioned

	if err := c.applyGuardedTransition(ctx, req, state, result); err != nil {
		return nil, err
	}
	result.ToStatus = req.TargetStatus
	result.TransitionApplied = state.PersistenceState == gaterun.PersistenceStateTransitioned

	if err := c.retire(req, state, result, ctx); err != nil {
		return nil, err
	}

	return result, nil
}

// acquireLock acquires the per-run lock guarding the whole Persist critical
// section, applying Request.LockTimeout (or gaterun.DefaultLockTimeout when
// unset).
func (c *Coordinator) acquireLock(req Request) (*gaterun.RunLock, error) {
	lockTimeout := req.LockTimeout
	if lockTimeout <= 0 {
		lockTimeout = gaterun.DefaultLockTimeout
	}
	lock, err := gaterun.AcquireRunLock(req.RunDir, lockTimeout)
	if err != nil {
		return nil, fmt.Errorf("gatepersist: acquire run lock: %w", err)
	}
	return lock, nil
}

// initializeRun computes this call's operation digest, durably binds the
// run's replay identity, validates every kickback's target-entity workflow
// membership, create-once-writes the accepted result, and loads (or
// initializes) the operation state — the full REQ-F-003 setup that must
// happen exactly once, in this order, before any suboperation is applied.
func (c *Coordinator) initializeRun(ctx context.Context, req Request) (digest string, kickbackEntityTypes map[string]models.EntityType, state *gaterun.OperationState, err error) {
	// The operation digest is computed here — BEFORE CreateIdentity — so
	// identity.json's OperationDigest and the OperationDigest this call will
	// later pass to NewOperationState/VerifyResumeIdentity are always the
	// SAME computed value for the same call. This is one computation reused
	// twice, not two independent digest computations that could drift.
	digest, err = gaterun.ComputeOperationDigest(req.EntityKey, string(req.EntityType), req.SourceStatus, req.Gate, req.EnvelopeJSON)
	if err != nil {
		return "", nil, nil, fmt.Errorf("gatepersist: compute operation digest: %w", err)
	}

	// UAT-3-1 fix, extended by UAT round 3+4 (note #2926): durably bind this
	// run_id's owning entity AND its full replay-identity context BEFORE
	// result.json (or anything derived from it) ever becomes recoverable.
	// gaterun.CreateIdentity uses the same create-once/first-writer-wins
	// protocol as CreateResult: a first call for run_id durably binds
	// req.EntityKey/EntityType/SourceStatus/Gate/digest; a later call whose
	// fields all agree is idempotent (the ordinary resume/replay case); a
	// later call disagreeing on ANY of those fields for the same run_id
	// fails closed with *ConflictError before any coordinator write —
	// closing the crash window where a result.json could exist with no
	// durable replay context to check a resume request's caller-supplied
	// identity against (see
	// internal/cli/commands.resumeGateIngestForUninitializedState's now-
	// resolved former accepted-risk comment; the full wiring of that
	// caller's own verification is T-E34-F05-004's scope).
	if _, err := gaterun.CreateIdentity(req.RunDir, gaterun.RunIdentity{
		RunID:           req.RunID,
		EntityKey:       req.EntityKey,
		EntityType:      string(req.EntityType),
		SourceStatus:    req.SourceStatus,
		Gate:            req.Gate,
		OperationDigest: digest,
	}); err != nil {
		return "", nil, nil, fmt.Errorf("gatepersist: bind run identity: %w", err)
	}

	// Kickback validation runs before this run_id ever accepts a durable
	// result: a run whose result would be rejected must not permanently burn
	// its run_id under gaterun.CreateResult's create-once/first-writer-wins
	// contract (there is no rewrite path — a corrected envelope needs a new
	// run_id otherwise). This is "rejected without partial mutation" applied
	// to the sidecar transport itself, not just target-store writes.
	kickbackEntityTypes, err = validateKickbacks(ctx, req.Result.Kickbacks, req.EntityType, req.EntityKey, c.Validator, c.Identity)
	if err != nil {
		return "", nil, nil, err
	}

	if _, err := gaterun.CreateResult(req.RunDir, []byte(req.EnvelopeJSON)); err != nil {
		return "", nil, nil, err
	}

	state, exists, err := gaterun.LoadOperationState(req.RunDir)
	if err != nil {
		return "", nil, nil, fmt.Errorf("gatepersist: load operation state: %w", err)
	}
	if !exists {
		// Entity identity here (req.EntityKey/req.EntityType) is no longer
		// established solely from THIS call's Request (the UAT-3-1 gap): the
		// CreateIdentity call above already durably bound req.RunID to
		// req.EntityKey/EntityType before CreateResult ever ran, and fails
		// closed with *ConflictError if a prior call already bound this
		// run_id to a DIFFERENT entity — so reaching this branch at all
		// means req.EntityKey/EntityType is either this run_id's first
		// binding or already matches its one durable owner. This closes the
		// window run_resume.go's resumeGateIngestForUninitializedState used
		// to accept as risk: a caller naming a DIFFERENT entity's run_id that
		// crashed in the create-once-result/before-state-init window now
		// fails at CreateIdentity, before this state is ever initialized.
		state = gaterun.NewOperationState(req.RunID, req.EntityKey, string(req.EntityType), req.SourceStatus, req.Gate, digest)
		if err := state.Save(req.RunDir); err != nil {
			return "", nil, nil, fmt.Errorf("gatepersist: initialize operation state: %w", err)
		}
	} else if err := gaterun.VerifyResumeIdentity(state, req.EntityKey, string(req.EntityType), req.SourceStatus, digest); err != nil {
		return "", nil, nil, err
	}

	return digest, kickbackEntityTypes, state, nil
}

// reconcileAndApplyOperations reconciles already-completed suboperations
// (a resumed call may find some already applied at the target store) and
// then applies every suboperation still pending, saving state after each
// step so a crash mid-loop resumes from the last completed suboperation
// rather than repeating already-applied writes.
func (c *Coordinator) reconcileAndApplyOperations(ctx context.Context, req Request, state *gaterun.OperationState, ops []operation, digest string, kickbackEntityTypes map[string]models.EntityType) error {
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
		return err
	}
	if changed {
		if err := state.Save(req.RunDir); err != nil {
			return fmt.Errorf("gatepersist: save reconciled operation state: %w", err)
		}
	}

	for _, op := range ops {
		subID := op.suboperationID(digest)
		if state.HasCompleted(subID) {
			continue
		}
		if err := c.applyOperation(ctx, op, subID, digest, req); err != nil {
			return err
		}
		state.AddCompletedSuboperation(subID)
		if err := state.Save(req.RunDir); err != nil {
			return fmt.Errorf("gatepersist: save operation state after suboperation %s: %w", subID, err)
		}
	}

	if err := state.MarkPersistenceComplete(); err != nil {
		return err
	}
	if err := state.Save(req.RunDir); err != nil {
		return fmt.Errorf("gatepersist: save operation state after persistence complete: %w", err)
	}
	return nil
}

// applyGuardedTransition applies the main-entity transition exactly once —
// "persistence just completed this call" and "already persistence_complete
// from a prior call" both apply it here. "already transition_applied" (a
// prior call already recorded it) must NOT repeat the transition call
// (architecture.md step 8: "it must not repeat the transition") — it only
// verifies the expected live target state via StatusReader and fails closed
// on any mismatch, since nothing guarantees the entity is still where this
// run last left it (a human `status set --force`, a cascade, or a
// concurrent run could have moved it since).
func (c *Coordinator) applyGuardedTransition(ctx context.Context, req Request, state *gaterun.OperationState, result *Result) error {
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
			return fmt.Errorf("gatepersist: verify source status before main transition: %w", err)
		}
		if !strings.EqualFold(current, req.SourceStatus) && !strings.EqualFold(current, req.TargetStatus) {
			return fmt.Errorf("gatepersist: entity %s is recorded at source status %q but is currently at %q; refusing to transition from an unrecorded source", req.EntityKey, req.SourceStatus, current)
		}

		reason := fmt.Sprintf("gate %s outcome %s", req.Gate, req.OutcomeKey)
		guard := TransitionGuard{
			SessionID:  req.Session.ID,
			FromStatus: req.SourceStatus,
			Outcome:    req.OutcomeKey,
		}
		fromStatus, transitioned, err := c.Transition.Transition(ctx, req.EntityType, req.EntityKey, req.TargetStatus, reason, req.Session.Agent, guard)
		if err != nil {
			return fmt.Errorf("gatepersist: apply main transition: %w", err)
		}
		result.FromStatus = fromStatus
		result.Transitioned = transitioned

		if err := state.MarkTransitionApplied(); err != nil {
			return err
		}
		if err := state.Save(req.RunDir); err != nil {
			return fmt.Errorf("gatepersist: save operation state after transition applied: %w", err)
		}
	case gaterun.PersistenceStateTransitioned:
		current, err := c.Status.CurrentStatus(ctx, req.EntityType, req.EntityKey)
		if err != nil {
			return fmt.Errorf("gatepersist: verify already-applied transition target: %w", err)
		}
		if !strings.EqualFold(current, req.TargetStatus) {
			return fmt.Errorf("gatepersist: entity %s is recorded transition_applied to %q but is currently at %q; refusing to repeat or silently diverge from the recorded transition", req.EntityKey, req.TargetStatus, current)
		}
		result.FromStatus = state.SourceStatus
	}
	return nil
}

// retire records worker-process retirement and, once RunConcluded ALSO
// holds, releases the claim/lease. RetirementConfirmed alone only proves
// this call's dispatched worker process exited, which is true for every
// stage of a multi-stage run (see Request.RetirementConfirmed's doc
// comment); a caller that (incorrectly) passes RetirementConfirmed per-stage
// without ever setting RunConcluded simply never releases via this
// coordinator — safe by construction, since a stale/leaked lease still
// expires via the claim TTL backstop, whereas a premature release let a
// concurrent claimant race the still-running loop.
func (c *Coordinator) retire(req Request, state *gaterun.OperationState, result *Result, ctx context.Context) error {
	if req.RetirementConfirmed {
		if state.RetirementState != gaterun.RetirementRetired {
			state.RetirementState = gaterun.RetirementRetired
			if err := state.Save(req.RunDir); err != nil {
				return fmt.Errorf("gatepersist: save retirement state: %w", err)
			}
		}
		if req.RunConcluded {
			released, err := c.Lease.Release(ctx, string(req.EntityType), req.EntityKey, req.Session.ID, req.OutcomeKey, false)
			if err != nil {
				return fmt.Errorf("gatepersist: release lease: %w", err)
			}
			result.LeaseReleased = released
		}
	} else if state.RetirementState == gaterun.RetirementUnknown {
		state.RetirementState = gaterun.RetirementPending
		if err := state.Save(req.RunDir); err != nil {
			return fmt.Errorf("gatepersist: save retirement state: %w", err)
		}
	}
	return nil
}

// verifyClaimSession re-checks req.Session.ID against the live claim/lease
// state, mirroring internal/cli/commands.verifyClaimSession's four checks
// exactly (non-empty session, an active claim exists, the session matches
// it, and it has not TTL-expired) so a caller cannot distinguish which of
// the two checks rejected it. A nil c.ClaimVerifier is a deliberate no-op
// (see the field's doc comment).
func (c *Coordinator) verifyClaimSession(ctx context.Context, req Request) error {
	if c.ClaimVerifier == nil {
		return nil
	}
	sessionID := strings.TrimSpace(req.Session.ID)
	if sessionID == "" {
		return fmt.Errorf("gatepersist: re-verify claim session: a session id is required")
	}
	claim, err := c.ClaimVerifier.Get(ctx, string(req.EntityType), req.EntityKey)
	if err != nil {
		return fmt.Errorf("gatepersist: re-verify claim ownership for %s %s: %w", req.EntityType, req.EntityKey, err)
	}
	if claim == nil {
		return fmt.Errorf("gatepersist: no active claim on %s %s: refusing to persist without a live claim", req.EntityType, req.EntityKey)
	}
	if claim.SessionID != sessionID {
		return fmt.Errorf("gatepersist: session no longer matches the active claim session on %s %s", req.EntityType, req.EntityKey)
	}
	if claim.IsExpired(time.Now().UTC(), c.ClaimVerifier.TTL()) {
		return fmt.Errorf("gatepersist: the claim session on %s %s has expired; refusing to persist", req.EntityType, req.EntityKey)
	}
	return nil
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
	reason := buildKickbackReason(k.Reason, subID, op.contentDigest(), req.RunID)
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
