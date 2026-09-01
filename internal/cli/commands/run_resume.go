// This file implements `shark run <entity-key> --resume-run=<run_id>
// --session=<session-id>` (T-E34-F05-002/004, REQ-F-003/REQ-F-005
// deliverable): reports a durable GateResult sidecar's resume status and,
// for a `gate_result_v1` step whose persistence is not yet fully applied
// (resume_transition/resume_next_operation), re-ingests the durably stored
// envelope (result.json's own bytes — no new result bytes are ever
// accepted from the caller) through the same runner.IngestGateResult
// boundary the core dispatch loop and Rider's --apply-result surface call.
// It acquires no claim and dispatches no agent — those remain out of scope
// here; resuming the guarded transition itself is now in scope, delegated
// entirely to gatepersist.Coordinator's own resume-aware Persist (idempotent
// under gaterun.VerifyResumeIdentity). An already_transitioned decision
// performs no target-write/transition repeat — Persist's own
// PersistenceStateTransitioned branch skips both — but it still calls the
// coordinator so it can verify the entity's live status against the
// recorded target and release the lease exactly once
// (gaterun/resume.go's ResumeActionAlreadyTransitioned contract: "verify the
// expected live target state and release the lease"). gatepersist.Coordinator,
// not this command, owns both of those side effects.
package commands

import (
	"context"
	"fmt"
	"time"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
)

// resumeRunOutput is the JSON shape `shark run --resume-run` prints.
type resumeRunOutput struct {
	RunID      string                    `json:"run_id"`
	EntityKey  string                    `json:"entity_key"`
	EntityType string                    `json:"entity_type"`
	Action     gaterun.ResumeAction      `json:"resume_action"`
	Status     *gaterun.StatusProjection `json:"status,omitempty"`

	// Ingested/ToStatus/Transitioned are populated whenever this call reached
	// the coordinator for a gate_result_v1 step (T-E34-F05-004 rework, F-2):
	// the durably recorded gate step's resolved result_contract was
	// gate_result_v1, for ANY resume action — including already_transitioned,
	// which no longer skips this. Transitioned is false for
	// already_transitioned's own idempotent verify-only branch (the
	// transition already happened on a prior call); Ingested/ToStatus are
	// still set because the coordinator ran.
	Ingested     bool   `json:"ingested,omitempty"`
	ToStatus     string `json:"to_status,omitempty"`
	Transitioned bool   `json:"transitioned,omitempty"`

	// LeaseReleased reports gatepersist.Coordinator's own
	// Result.LeaseReleased (F-2/T-E34-F05-004 rework): true when this call
	// released the parent's claim/lease session, which happens for both a
	// freshly-applied transition and an already_transitioned decision whose
	// live status verification passed.
	LeaseReleased bool `json:"lease_released,omitempty"`
}

// gateStepResolver exposes the narrow *workflow.Service surface
// resumeGateIngestIfConfigured needs: the durably-recorded gate step's
// (decision.State.Gate — not the entity's possibly-since-advanced current
// status's) result_contract/outcomes/outcome_roles. *workflow.Service
// satisfies this directly via ForLevel(entityType). A dedicated interface
// (rather than reusing runner.EntityTransitioner, which only exposes
// GetNextStatus for the entity's *current* status) is required because an
// already_transitioned entity has moved off the gate step by the time
// --resume-run runs — see resumeGateIngestIfConfigured's doc comment.
type gateStepResolver interface {
	GetResultContract(status string) string
	GetOutcomes(status string) map[string]string
	GetOutcomeRoles(status string) map[string]gateresult.OutcomeRole
}

// runResumeWorkflowServiceOverride/runResumeCoordinatorOverride/
// runResumeTransitionerOverride let tests inject a mocked
// gateStepResolver/gatepersist.Coordinator/runner.EntityTransitioner
// instead of the real cli.Get*Service()-backed ones (per the CLI-tests
// golden rule: never a real database in a CLI-command test). Production
// callers leave all three nil. runResumeTransitionerOverride is consumed
// only by resumeGateIngestForUninitializedState — see its doc comment for
// why that one branch needs the entity's *current* status rather than a
// durably recorded gate step name.
var (
	runResumeWorkflowServiceOverride gateStepResolver
	runResumeCoordinatorOverride     *gatepersist.Coordinator
	runResumeTransitionerOverride    runner.EntityTransitioner
)

// runResumeRun is the RunE-called entry point for the --resume-run branch.
// It validates the required --session flag, resolves the run directory, and
// prints the resume status (plus any gate re-ingestion outcome) as JSON
// (this surface is inherently machine/parent-facing, per the task spec's
// "operator polling independent of chat notifications" — it always emits
// JSON regardless of the global --json flag).
func runResumeRun(ctx context.Context, entityType, entityKey string) error {
	if runSession == "" {
		return fmt.Errorf("--resume-run requires --session=<authorized-session-id>")
	}

	// REQ-F-002 authorization gate (UAT CRITICAL finding #1): --session must
	// name the ACTIVE claim/lease session on this entity, not merely be a
	// non-empty string. Verified before any coordinator call/mutation so a
	// mismatched/nonexistent/expired session produces zero writes.
	if err := verifyClaimSession(ctx, entityType, entityKey, runSession); err != nil {
		return fmt.Errorf("resume-run authorization failed: %w", err)
	}

	projectRoot, err := cli.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("resolve project root for --resume-run: %w", err)
	}

	out, decision, err := resolveResumeStatusAndDecision(projectRoot, runResumeID, entityType, entityKey, time.Now())
	if err != nil {
		return err
	}

	// F-2/T-E34-F05-004 rework: an already_transitioned decision must still
	// reach the coordinator — it owns the live-status verification and
	// lease release gaterun/resume.go's ResumeActionAlreadyTransitioned
	// contract requires (see resumeGateIngestIfConfigured's doc comment) —
	// so this no longer short-circuits on decision.Action.
	if decision.State != nil {
		if err := resumeGateIngestIfConfigured(ctx, projectRoot, entityType, entityKey, decision, out); err != nil {
			return fmt.Errorf("resume gate ingestion failed: %w", err)
		}
	} else {
		// Sibling of the F-2 defect class, swept in the same pass: the
		// create-once-result/before-state-init crash window
		// (gaterun.DecideResume's nil-State ResumeActionResumeNextOperation
		// case — result.json committed, operation-state.json never
		// written) must also reach the coordinator, or a crash in this
		// narrower window leaves the lease held forever exactly like F-2.
		// See resumeGateIngestForUninitializedState's doc comment for why
		// it derives SourceStatus/Gate differently from
		// resumeGateIngestIfConfigured.
		if err := resumeGateIngestForUninitializedState(ctx, projectRoot, entityType, entityKey, decision, out); err != nil {
			return fmt.Errorf("resume gate ingestion failed: %w", err)
		}
	}

	return cli.OutputJSON(out)
}

// resumeGateIngestIfConfigured re-ingests decision.Result (the durably
// stored envelope bytes) through runner.IngestGateResult when the durably
// recorded gate step (decision.State.Gate) resolves to result_contract:
// gate_result_v1. A legacy step is left untouched — --resume-run's
// status/decision report is its only contribution, unchanged from before
// this task.
//
// This resolves result_contract/outcomes/outcome_roles by the *gate* step's
// name (decision.State.Gate), never by the entity's current live status:
// for a resume_transition/resume_next_operation decision the entity is
// still parked at the gate step, so the two happen to agree, but for an
// already_transitioned decision the entity has already moved onto the
// step *after* the gate — a `transitioner.GetNextStatus(entityKey)` lookup
// there would silently resolve the wrong step's (often legacy, non-gate)
// contract and skip this function's already_transitioned finalization
// entirely (the exact F-2 defect). gateStepResolver's
// GetResultContract/GetOutcomes/GetOutcomeRoles all take an explicit status
// argument for this reason.
//
// Calling IngestGateResult again for an already_transitioned decision is
// safe and non-duplicating: gatepersist.Coordinator.Persist's own
// PersistenceStateTransitioned branch (coordinator.go) skips the six-step
// write loop and the transition call entirely, and only (a) verifies the
// entity's live status against the recorded target, failing closed on
// divergence, and (b) — via RetirementConfirmed: true — marks retirement
// and releases the lease exactly once (ClaimService.Release is itself
// idempotent: a second release against an already-closed session returns
// released=false, not an error). This is what closes the "transition
// succeeded, crashed before release" crash window feature.md's "Replay a
// committed result" acceptance scenario names (F-2).
func resumeGateIngestIfConfigured(ctx context.Context, projectRoot, entityType, entityKey string, decision *gaterun.ResumeDecision, out *resumeRunOutput) error {
	resolver := runResumeWorkflowServiceOverride
	if resolver == nil {
		resolver = cli.GetWorkflowService().ForLevel(normalizeEntityTypeForWorkflow(entityType))
	}

	if resolver.GetResultContract(decision.State.Gate) != resultContractGateResultV1 {
		return nil
	}

	coordinator := runResumeCoordinatorOverride
	if coordinator == nil {
		var err error
		coordinator, err = buildGateCoordinator(ctx)
		if err != nil {
			return fmt.Errorf("build GateResult persistence coordinator: %w", err)
		}
	}

	result, err := runner.IngestGateResult(ctx, runner.GateIngestRequest{
		EnvelopeBytes:       decision.Result,
		Coordinator:         coordinator,
		ProjectRoot:         projectRoot,
		RunID:               runResumeID,
		EntityKey:           entityKey,
		EntityType:          models.EntityType(entityType),
		SourceStatus:        decision.State.SourceStatus,
		Gate:                decision.State.Gate,
		Session:             gatepersist.Session{ID: runSession},
		OutcomeRoles:        resolver.GetOutcomeRoles(decision.State.Gate),
		Outcomes:            resolver.GetOutcomes(decision.State.Gate),
		RetirementConfirmed: true,
		// RunConcluded: true — --resume-run acquires no claim and dispatches
		// no agent itself (see this file's package doc comment); this call
		// is the parent's one and only action for this entity/session, so it
		// is always the run's last action, matching gatepersist.Coordinator's
		// requirement (T-E34-F05-003 rework) that release needs BOTH
		// RetirementConfirmed AND RunConcluded, not RetirementConfirmed alone.
		RunConcluded: true,
	})
	if err != nil {
		return err
	}

	out.Ingested = true
	out.ToStatus = result.ToStatus
	out.Transitioned = result.Transitioned
	out.LeaseReleased = result.LeaseReleased
	return nil
}

// resumeGateIngestForUninitializedState handles the sibling of F-2's crash
// window: gaterun.DecideResume's nil-State ResumeActionResumeNextOperation
// case, where result.json committed durably but operation-state.json was
// never written at all (the create-once-result/before-state-init window).
// There is no durably recorded gate/source_status to key off of yet — but
// unlike resumeGateIngestIfConfigured's already_transitioned case, the
// entity has NOT transitioned in this window (operation-state.json never
// even reached PersistenceStateComplete, let alone the transition), so its
// live current status IS the source status: a
// transitioner.GetNextStatus(entityKey) lookup is correct here, the one
// case where it would be wrong for resumeGateIngestIfConfigured.
// gatepersist.Coordinator.Persist's own !exists branch
// (coordinator.go) initializes a fresh OperationState from the
// SourceStatus/Gate/digest this call supplies, exactly closing this window.
//
// Accepted risk (code-review round-9 Finding 1, investigated — not fixed):
// unlike resumeGateIngestIfConfigured's decision.State != nil path, there is
// no durably recorded entity identity for this run_id to check the caller's
// entityType/entityKey against — that is exactly what this call is
// initializing. A caller holding a genuinely valid, live claim/session on
// entity Y who also names a DIFFERENT entity X's run_id (one that crashed in
// this exact create-once-result/before-state-init window) will have
// gatepersist.Coordinator.Persist bind Y's identity to X's already-durable
// result.json bytes (decision.Result — never new caller-supplied bytes; see
// this file's package doc comment) and apply X's envelope under Y's own
// outcome/kickback role mapping.
//
// This is accepted rather than fixed here because:
//  1. There is no durable record to check against pre-init — closing it for
//     real requires a schema change (recording run_id -> entity/claim binding
//     at run-creation time, e.g. on the claim itself) verified independently
//     of this call, which is a design change out of scope for this fix-only
//     rework round, not a local guard this function can add.
//  2. run_id is a cryptographically unguessable uuid.New() minted once per
//     top-level `shark run` invocation (internal/gaterun/runid.go) — this is
//     the same capability-token trust model this feature already relies on
//     elsewhere (claim SessionID). Reaching this window at all requires
//     either possessing that exact run_id or filesystem read access to
//     .shark/runs/ to enumerate it; either implies access roughly equivalent
//     to what a caller already has by holding a live claim/session in this
//     single-project, same-user CLI (not a multi-tenant boundary).
//  3. A successful cross-application still requires more than access alone:
//     X's envelope outcome_key must resolve against Y's own currently-valid
//     Outcomes map (nextInfo.Outcomes, derived from Y's live status below) —
//     not a coincidental match, since pass/fail/blocked are the mandatory
//     core outcome vocabulary every workable step defines (see the
//     route-based workflow guide's outcome-routing section), so this
//     condition is easily satisfied rather than a meaningful additional
//     barrier — and any kickbacks in X's envelope must independently pass
//     validateKickbacks against Y's own kickback-eligible workflow
//     membership — gateresult/gatepersist validation fails closed otherwise.
//  4. The far more likely real-world trigger is operator error (a stale or
//     mistyped --resume-run=<run_id> against the wrong entity), which this
//     reasoning does not make safe to ignore going forward — a future
//     rework that adds the run_id->entity binding from point 1 should remove
//     this comment and add the real check.
//
// Sibling swept (code-review round-9 rework protocol): run_apply_result.go's
// applyResultIngest reaches the same gatepersist.Coordinator.Persist !exists
// branch through runner.IngestGateResult, but its EnvelopeBytes come from a
// caller-supplied --apply-result file path, not a durably-read result.json —
// so gaterun.CreateResult's create-once/first-writer-wins content-hash check
// (fsio.go) fails closed on any foreign run_id whose result.json already
// holds different bytes. The same residual risk exists there only if the
// caller reproduces a foreign entity's exact original envelope bytes
// byte-for-byte, which needs the same read access as this window's exposure
// — not fixed separately for the same reasons as points 1-2 above.
func resumeGateIngestForUninitializedState(ctx context.Context, projectRoot, entityType, entityKey string, decision *gaterun.ResumeDecision, out *resumeRunOutput) error {
	transitioner := runResumeTransitionerOverride
	if transitioner == nil {
		var err error
		transitioner, err = buildTransitioner(ctx, entityType)
		if err != nil {
			return fmt.Errorf("build transitioner: %w", err)
		}
	}

	nextInfo, err := transitioner.GetNextStatus(ctx, entityKey)
	if err != nil {
		return fmt.Errorf("get status for %s: %w", entityKey, err)
	}
	if nextInfo.ResultContract != resultContractGateResultV1 {
		return nil
	}

	coordinator := runResumeCoordinatorOverride
	if coordinator == nil {
		var err error
		coordinator, err = buildGateCoordinator(ctx)
		if err != nil {
			return fmt.Errorf("build GateResult persistence coordinator: %w", err)
		}
	}

	result, err := runner.IngestGateResult(ctx, runner.GateIngestRequest{
		EnvelopeBytes:       decision.Result,
		Coordinator:         coordinator,
		ProjectRoot:         projectRoot,
		RunID:               runResumeID,
		EntityKey:           entityKey,
		EntityType:          models.EntityType(entityType),
		SourceStatus:        nextInfo.CurrentStatus,
		Gate:                nextInfo.CurrentStatus,
		Session:             gatepersist.Session{ID: runSession},
		OutcomeRoles:        nextInfo.OutcomeRoles,
		Outcomes:            nextInfo.Outcomes,
		RetirementConfirmed: true,
		// RunConcluded: true — --resume-run acquires no claim and dispatches
		// no agent itself (see this file's package doc comment); this call
		// is the parent's one and only action for this entity/session, so it
		// is always the run's last action, matching gatepersist.Coordinator's
		// requirement (T-E34-F05-003 rework) that release needs BOTH
		// RetirementConfirmed AND RunConcluded, not RetirementConfirmed alone.
		RunConcluded: true,
	})
	if err != nil {
		return err
	}

	out.Ingested = true
	out.ToStatus = result.ToStatus
	out.Transitioned = result.Transitioned
	out.LeaseReleased = result.LeaseReleased
	return nil
}

// resultContractGateResultV1 mirrors internal/runner's own unexported
// constant of the same value — REQ-F-006's gate_result_v1 result_contract
// literal. Duplicated here (rather than exported from internal/runner)
// because this package must not reach into runner's dispatch-internal
// naming; the literal itself is stable API surface (T-E34-F05-005's
// workflow YAML field value).
const resultContractGateResultV1 = "gate_result_v1"

// resolveResumeStatus is the pure, filesystem-only (no DB, no claim
// service) core of --resume-run: given a project root and run_id, it
// acquires the per-run lock for the duration of the read (REQ-F-003:
// "acquires the run lock ... before resuming"), loads the durable sidecar,
// decides the REQ-F-003 resume action, fails closed on the bound identity it
// can check from CLI-supplied arguments alone (entity_key, entity_type —
// source_status and operation digest verification require the caller's
// current entity state and validated envelope, which belong to
// T-E34-F05-003/004's fuller resume path, not this read-only status
// report), and projects operator status.
//
// Extracted from runResumeRun so it is directly unit-testable against a
// t.TempDir() run directory without any CLI/service plumbing.
//
// resolveResumeStatus is a thin backward-compatible wrapper that discards
// the ResumeDecision; use resolveResumeStatusAndDecision when the caller
// also needs decision.Result (the durable envelope bytes) or
// decision.State (source_status/gate) to re-ingest a gate_result_v1 step.
func resolveResumeStatus(projectRoot, runID, entityType, entityKey string, now time.Time) (*resumeRunOutput, error) {
	out, _, err := resolveResumeStatusAndDecision(projectRoot, runID, entityType, entityKey, now)
	return out, err
}

func resolveResumeStatusAndDecision(projectRoot, runID, entityType, entityKey string, now time.Time) (*resumeRunOutput, *gaterun.ResumeDecision, error) {
	dir, err := gaterun.RunDir(projectRoot, runID)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve run directory for run_id %q: %w", runID, err)
	}

	lock, err := gaterun.AcquireRunLock(dir, gaterun.DefaultLockTimeout)
	if err != nil {
		return nil, nil, fmt.Errorf("acquire run lock for run_id %q: %w", runID, err)
	}
	defer func() { _ = lock.Release() }()

	decision, err := gaterun.DecideResume(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("decide resume for run_id %q: %w", runID, err)
	}

	out := &resumeRunOutput{
		RunID:      runID,
		EntityKey:  entityKey,
		EntityType: entityType,
		Action:     decision.Action,
	}

	if decision.State == nil {
		// result.json exists but operation-state.json was never
		// initialized (the create-once-result/before-state-init crash
		// window) — there is no durable state to identity-check yet or to
		// project status from.
		return out, decision, nil
	}

	if decision.State.EntityKey != entityKey || decision.State.EntityType != entityType {
		return nil, nil, fmt.Errorf(
			"resume identity mismatch for run_id %q: recorded entity %s/%s does not match requested entity %s/%s",
			runID, decision.State.EntityType, decision.State.EntityKey, entityType, entityKey,
		)
	}

	projection := gaterun.ProjectStatus(decision.State, now)
	out.Status = &projection
	return out, decision, nil
}
