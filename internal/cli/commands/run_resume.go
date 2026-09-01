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

// runResumeWorkflowServiceOverride/runResumeCoordinatorOverride let tests
// inject a mocked gateStepResolver/gatepersist.Coordinator instead of the
// real cli.Get*Service()-backed ones (per the CLI-tests golden rule: never a
// real database in a CLI-command test). Production callers leave both nil.
var (
	runResumeWorkflowServiceOverride gateStepResolver
	runResumeCoordinatorOverride     *gatepersist.Coordinator
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
