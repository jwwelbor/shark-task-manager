// This file implements `shark run <entity-key> --resume-run=<run_id>
// --session=<session-id>` (T-E34-F05-002, REQ-F-003 deliverable): a
// read-only report of a durable GateResult sidecar's resume status. It
// acquires no claim, dispatches no agent, and applies no transition — those
// remain the persistence coordinator's job (T-E34-F05-003/004). This
// command's contribution is limited to what T-E34-F05-002 owns: locating
// the run directory, reading result.json/operation-state.json, and
// reporting the REQ-F-003 resume decision (already_transitioned /
// resume_transition / resume_next_operation) plus the operator status
// projection (worker phase, nested operation, elapsed time, retirement
// state, result location).
package commands

import (
	"fmt"
	"time"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
)

// resumeRunOutput is the JSON shape `shark run --resume-run` prints.
type resumeRunOutput struct {
	RunID      string                    `json:"run_id"`
	EntityKey  string                    `json:"entity_key"`
	EntityType string                    `json:"entity_type"`
	Action     gaterun.ResumeAction      `json:"resume_action"`
	Status     *gaterun.StatusProjection `json:"status,omitempty"`
}

// runResumeRun is the RunE-called entry point for the --resume-run branch.
// It validates the required --session flag, resolves the run directory, and
// prints resolveResumeStatus's result as JSON (this surface is inherently
// machine/parent-facing, per the task spec's "operator polling independent
// of chat notifications" — it always emits JSON regardless of the global
// --json flag).
func runResumeRun(entityType, entityKey string) error {
	if runSession == "" {
		return fmt.Errorf("--resume-run requires --session=<authorized-session-id>")
	}

	projectRoot, err := cli.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("resolve project root for --resume-run: %w", err)
	}

	out, err := resolveResumeStatus(projectRoot, runResumeID, entityType, entityKey, time.Now())
	if err != nil {
		return err
	}
	return cli.OutputJSON(out)
}

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
func resolveResumeStatus(projectRoot, runID, entityType, entityKey string, now time.Time) (*resumeRunOutput, error) {
	dir, err := gaterun.RunDir(projectRoot, runID)
	if err != nil {
		return nil, fmt.Errorf("resolve run directory for run_id %q: %w", runID, err)
	}

	lock, err := gaterun.AcquireRunLock(dir, gaterun.DefaultLockTimeout)
	if err != nil {
		return nil, fmt.Errorf("acquire run lock for run_id %q: %w", runID, err)
	}
	defer func() { _ = lock.Release() }()

	decision, err := gaterun.DecideResume(dir)
	if err != nil {
		return nil, fmt.Errorf("decide resume for run_id %q: %w", runID, err)
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
		return out, nil
	}

	if decision.State.EntityKey != entityKey || decision.State.EntityType != entityType {
		return nil, fmt.Errorf(
			"resume identity mismatch for run_id %q: recorded entity %s/%s does not match requested entity %s/%s",
			runID, decision.State.EntityType, decision.State.EntityKey, entityType, entityKey,
		)
	}

	projection := gaterun.ProjectStatus(decision.State, now)
	out.Status = &projection
	return out, nil
}
