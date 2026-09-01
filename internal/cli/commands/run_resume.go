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
// performs no ingestion — the transition is already durably applied, and
// gatepersist.Coordinator (not this command) owns lease release.
package commands

import (
	"context"
	"fmt"
	"time"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
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

	// Ingested/ToStatus/Transitioned are populated only when this call
	// performed a gate_result_v1 re-ingestion (T-E34-F05-004): the step's
	// resolved result_contract was gate_result_v1 and Action was not
	// already_transitioned.
	Ingested     bool   `json:"ingested,omitempty"`
	ToStatus     string `json:"to_status,omitempty"`
	Transitioned bool   `json:"transitioned,omitempty"`
}

// runResumeTransitionerOverride/runResumeCoordinatorOverride let tests
// inject a mocked runner.EntityTransitioner/gatepersist.Coordinator instead
// of the real cli.Get*Service()-backed ones (per the CLI-tests golden rule:
// never a real database in a CLI-command test). Production callers leave
// both nil.
var (
	runResumeTransitionerOverride runner.EntityTransitioner
	runResumeCoordinatorOverride  *gatepersist.Coordinator
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

	if decision.State != nil && decision.Action != gaterun.ResumeActionAlreadyTransitioned {
		if err := resumeGateIngestIfConfigured(ctx, projectRoot, entityType, entityKey, decision, out); err != nil {
			return fmt.Errorf("resume gate ingestion failed: %w", err)
		}
	}

	return cli.OutputJSON(out)
}

// resumeGateIngestIfConfigured re-ingests decision.Result (the durably
// stored envelope bytes) through runner.IngestGateResult when the entity's
// currently-dispatched step resolves to result_contract: gate_result_v1.
// A legacy step is left untouched — --resume-run's status/decision report
// is its only contribution, unchanged from before this task.
func resumeGateIngestIfConfigured(ctx context.Context, projectRoot, entityType, entityKey string, decision *gaterun.ResumeDecision, out *resumeRunOutput) error {
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
		SourceStatus:        decision.State.SourceStatus,
		Gate:                decision.State.Gate,
		Session:             gatepersist.Session{ID: runSession},
		OutcomeRoles:        nextInfo.OutcomeRoles,
		Outcomes:            nextInfo.Outcomes,
		RetirementConfirmed: true,
	})
	if err != nil {
		return err
	}

	out.Ingested = true
	out.ToStatus = result.ToStatus
	out.Transitioned = result.Transitioned
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
