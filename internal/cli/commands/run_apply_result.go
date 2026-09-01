// This file implements `shark run <entity-key> --apply-result=<result-file>
// --run-id=<run_id> --session=<authorized-session-id>` (T-E34-F05-004,
// REQ-F-005): Rider's initial-ingestion CLI surface. It reads a candidate
// worker-control envelope from result-file and calls the exact same shared
// boundary (runner.IngestGateResult) the core runner calls directly from
// internal/runner/controller.go's gate_result_v1 branch — Rider is not a
// second implementation, it is a second caller of one function. Do not add
// a Rider-only parser or persistence sequence here.
package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/jwwelbor/shark-task-manager/internal/workercontrol"
)

var (
	runApplyResultPath string
	runApplyRunID      string
)

func init() {
	runCmd.Flags().StringVar(&runApplyResultPath, "apply-result", "", "Path to a worker-control envelope file to ingest for this entity (T-E34-F05-004); requires --run-id and --session")
	runCmd.Flags().StringVar(&runApplyRunID, "run-id", "", "Durable run_id for --apply-result")
}

// applyResultOutcomeRolesOverride lets tests inject a resolved outcome_roles
// map without a real workflow config source. Production callers leave this
// nil so applyResultIngest falls back to nextInfo.OutcomeRoles — the
// workflow-resolved map from the step's `outcome_roles` YAML field
// (T-E34-F05-005) — instead of guessing a role.
var applyResultOutcomeRolesOverride map[string]gateresult.OutcomeRole

// runApplyResultSet reports whether --apply-result was provided, so runRun
// can branch before any claim/dispatch state is touched (mirroring
// --resume-run's short-circuit).
func runApplyResultSet() bool {
	return runApplyResultPath != ""
}

// readBoundedEnvelopeFile reads path bounded at
// workercontrol.MaxEnvelopeBytes+1 bytes (UAT round-2 Finding 2): the prior
// unconditional os.ReadFile buffered the ENTIRE file into memory before
// workercontrol.Decode ever evaluated MaxEnvelopeBytes, so a maliciously
// huge --apply-result file was still fully read into memory first, defeating
// the point of that bound.
//
// It delegates to gaterun.ReadBoundedRegularFile (code-review round-6
// finding), which gives this CLI-flag-supplied path the exact same
// no-follow-open + fstat-regular-file-check safety internal/gaterun's own
// readRegularBounded uses for the sidecar transport's file reads (fsio.go).
// The prior plain os.Open here silently followed a symlink target, and would
// hang indefinitely if pointed at a FIFO with no writer connected — the size
// bound via io.LimitReader doesn't help there, since it still blocks reading
// from an open FIFO below the limit. See fsio_path.go's doc comment for why
// this is a shared entry point rather than a duplicated implementation.
func readBoundedEnvelopeFile(path string) ([]byte, error) {
	data, err := gaterun.ReadBoundedRegularFile(path, workercontrol.MaxEnvelopeBytes)
	if err != nil {
		if strings.Contains(err.Error(), "byte bound") {
			return nil, fmt.Errorf("exceeds the maximum envelope size of %d bytes", workercontrol.MaxEnvelopeBytes)
		}
		return nil, err
	}
	return data, nil
}

func runApplyResult(cmd *cobra.Command, entityType, entityKey string) error {
	if runApplyRunID == "" {
		return fmt.Errorf("--apply-result requires --run-id=<run_id>")
	}
	if runSession == "" {
		return fmt.Errorf("--apply-result requires --session=<authorized-session-id>")
	}

	// REQ-F-002 authorization gate (UAT CRITICAL finding #1): --session must
	// name the ACTIVE claim/lease session on this entity, not merely be a
	// non-empty string. Verified before any file read or coordinator
	// construction so a mismatched/nonexistent/expired session produces zero
	// writes.
	if err := verifyClaimSession(cmd.Context(), entityType, entityKey, runSession); err != nil {
		return fmt.Errorf("apply-result authorization failed: %w", err)
	}

	envelopeBytes, err := readBoundedEnvelopeFile(runApplyResultPath)
	if err != nil {
		return fmt.Errorf("read --apply-result file %q: %w", runApplyResultPath, err)
	}

	projectRoot, err := cli.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("resolve project root for --apply-result: %w", err)
	}

	transitioner, err := buildTransitioner(cmd.Context(), entityType)
	if err != nil {
		return fmt.Errorf("failed to build transitioner for %s: %w", entityType, err)
	}

	coordinator, err := buildGateCoordinator(cmd.Context())
	if err != nil {
		return fmt.Errorf("build GateResult persistence coordinator: %w", err)
	}

	result, err := applyResultIngest(cmd.Context(), applyResultDeps{
		Transitioner: transitioner,
		Coordinator:  coordinator,
		ProjectRoot:  projectRoot,
		RunID:        runApplyRunID,
		EntityType:   entityType,
		EntityKey:    entityKey,
		SessionID:    runSession,
		OutcomeRoles: applyResultOutcomeRolesOverride,
		WorkflowSvc:  cli.GetWorkflowService(),
	}, envelopeBytes)
	if err != nil {
		return fmt.Errorf("apply-result ingestion failed: %w", err)
	}

	return cli.OutputJSON(applyResultOutput{
		RunID:        runApplyRunID,
		EntityKey:    entityKey,
		EntityType:   entityType,
		OutcomeKey:   result.OutcomeKey,
		Role:         string(result.Role),
		ToStatus:     result.ToStatus,
		Transitioned: result.Transitioned,
		Status:       result.Status,
	})
}

// applyResultDeps bundles applyResultIngest's dependencies so tests can
// inject a mocked runner.EntityTransitioner and gatepersist.Coordinator
// (per the CLI-tests golden rule: never a real database in a CLI-command
// test). runApplyResult builds real ones from cli's global service
// accessors and buildTransitioner.
type applyResultDeps struct {
	Transitioner runner.EntityTransitioner
	Coordinator  *gatepersist.Coordinator
	ProjectRoot  string
	RunID        string
	EntityType   string
	EntityKey    string
	SessionID    string
	OutcomeRoles map[string]gateresult.OutcomeRole

	// WorkflowSvc resolves whether the ingested gate result's target status
	// is terminal, mirroring internal/runner/controller.go's
	// ingestGateResultForDispatch resolve-terminal-then-re-ingest-with-both-
	// flags pattern (code-review round-7 Finding 2). Required: applyResultIngest
	// fails loud rather than silently never confirming retirement when nil.
	WorkflowSvc terminalStatusChecker
}

// terminalStatusChecker is the narrow interface applyResultIngest needs to
// decide whether an ingested gate result's resolved target status is
// terminal. Defined at point of use per project convention
// (.claude/rules/go/patterns.md); *workflow.Service satisfies it.
type terminalStatusChecker interface {
	IsTerminalStatus(status string) bool
}

// applyResultIngest is the pure, directly-testable core of --apply-result:
// read the entity's current status/outcomes, then call the same
// runner.IngestGateResult boundary the core runner calls. It is also what a
// parity test compares against a direct runner.IngestGateResult call for
// the same fixture (T-E34-F05-004's REQ-F-005 acceptance criterion).
func applyResultIngest(ctx context.Context, deps applyResultDeps, envelopeBytes []byte) (*runner.GateIngestResult, error) {
	if deps.WorkflowSvc == nil {
		return nil, fmt.Errorf("apply-result ingestion: WorkflowSvc dependency is required to resolve terminal status")
	}

	nextInfo, err := deps.Transitioner.GetNextStatus(ctx, deps.EntityKey)
	if err != nil {
		return nil, fmt.Errorf("get status for %s before --apply-result: %w", deps.EntityKey, err)
	}

	// T-E34-F05-005: prefer the workflow-resolved outcome_roles map
	// (nextInfo.OutcomeRoles) so Rider's --apply-result path and the core
	// runner's gate_result_v1 branch enforce the exact same per-step role
	// map. deps.OutcomeRoles remains available for tests/operators that need
	// to override the resolved map explicitly.
	outcomeRoles := deps.OutcomeRoles
	if len(outcomeRoles) == 0 {
		outcomeRoles = nextInfo.OutcomeRoles
	}

	baseReq := runner.GateIngestRequest{
		EnvelopeBytes: envelopeBytes,
		Coordinator:   deps.Coordinator,
		ProjectRoot:   deps.ProjectRoot,
		RunID:         deps.RunID,
		EntityKey:     deps.EntityKey,
		EntityType:    models.EntityType(deps.EntityType),
		SourceStatus:  nextInfo.CurrentStatus,
		Gate:          nextInfo.CurrentStatus,
		Session:       gatepersist.Session{ID: deps.SessionID},
		OutcomeRoles:  outcomeRoles,
		Outcomes:      nextInfo.Outcomes,
	}

	req := baseReq
	req.RetirementConfirmed = false
	result, err := runner.IngestGateResult(ctx, req)
	if err != nil {
		return nil, err
	}

	// code-review round-7 Finding 2: --apply-result previously left
	// RetirementConfirmed/RunConcluded at their zero value (false)
	// unconditionally, unlike internal/runner/controller.go's
	// ingestGateResultForDispatch, so the claim/lease was never released on
	// this path — even on a terminal outcome — until TTL expiry. Mirror that
	// method's resolve-terminal-then-re-ingest-with-both-flags pattern:
	// --apply-result is a standalone, single-shot invocation for exactly one
	// gate stage (unlike the core runner's multi-stage Run() loop), so
	// RunConcluded is unconditionally true alongside RetirementConfirmed
	// once the resolved target status is terminal. The second call is safe
	// and non-duplicating — see ingestGateResultForDispatch's doc comment.
	if deps.WorkflowSvc.IsTerminalStatus(result.ToStatus) {
		retireReq := baseReq
		retireReq.RetirementConfirmed = true
		retireReq.RunConcluded = true
		retireResult, retireErr := runner.IngestGateResult(ctx, retireReq)
		if retireErr != nil {
			return nil, retireErr
		}
		result = retireResult
	}

	return result, nil
}

type applyResultOutput struct {
	RunID        string `json:"run_id"`
	EntityKey    string `json:"entity_key"`
	EntityType   string `json:"entity_type"`
	OutcomeKey   string `json:"outcome_key"`
	Role         string `json:"role"`
	ToStatus     string `json:"to_status"`
	Transitioned bool   `json:"transitioned"`

	// Status is the T-E34-F05-004 REQ-F-005 operator status projection
	// (worker phase, nested operation, elapsed time, retirement state,
	// result location), the same shape --resume-run reports.
	Status *gaterun.StatusProjection `json:"status,omitempty"`
}

// buildGateCoordinator wires a real gatepersist.Coordinator from the CLI's
// global service accessors, via the same adapters (gatepersist.adapters.go)
// used everywhere else a coordinator is constructed. It is the CLI-side
// mirror of whatever construction the core runner's own entry point
// (run.go's runRun) will use once T-E34-F05-005 lands and gate_result_v1
// steps become reachable through real dispatch.
func buildGateCoordinator(ctx context.Context) (*gatepersist.Coordinator, error) {
	noteSvc, err := cli.GetNoteService(ctx)
	if err != nil {
		return nil, fmt.Errorf("get note service: %w", err)
	}
	entitySvc := cli.GetEntityService()
	registry := cli.GetEntityRegistry()
	workflowSvc := cli.GetWorkflowService()
	transitioner := gatepersist.NewEntityServiceTransitioner(entitySvc, registry, workflowSvc)
	validator := gatepersist.NewWorkflowStatusValidator(workflowSvc)
	history := cli.GetEntityHistoryService()
	// getRunClaimService() (not cli.GetClaimService() directly) so tests can
	// inject a mocked claim store via runClaimSvcOverride (the CLI-tests
	// golden rule: never a real database in a CLI-command test) and so this
	// coordinator's re-verification (ClaimVerifier, below) reads from the
	// exact same claim state run.go's initial verifyClaimSession checked.
	claimSvc := getRunClaimService()

	coordinator := gatepersist.NewCoordinator(noteSvc, noteSvc, history, validator, transitioner, transitioner, claimSvc)
	// UAT round-2 Finding 1: fold a second claim-ownership check into
	// Persist's own critical section (the per-run lock), immediately before
	// its mutating writes — closing the TOCTOU window between run.go's
	// one-time verifyClaimSession call (above, before this coordinator is
	// even built) and Persist's actual writes. See
	// gatepersist.ClaimVerifier's doc comment.
	coordinator.ClaimVerifier = claimSvc
	return coordinator, nil
}
