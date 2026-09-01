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
	"os"

	"github.com/spf13/cobra"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
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

func runApplyResult(cmd *cobra.Command, entityType, entityKey string) error {
	if runApplyRunID == "" {
		return fmt.Errorf("--apply-result requires --run-id=<run_id>")
	}
	if runSession == "" {
		return fmt.Errorf("--apply-result requires --session=<authorized-session-id>")
	}

	envelopeBytes, err := os.ReadFile(runApplyResultPath)
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
}

// applyResultIngest is the pure, directly-testable core of --apply-result:
// read the entity's current status/outcomes, then call the same
// runner.IngestGateResult boundary the core runner calls. It is also what a
// parity test compares against a direct runner.IngestGateResult call for
// the same fixture (T-E34-F05-004's REQ-F-005 acceptance criterion).
func applyResultIngest(ctx context.Context, deps applyResultDeps, envelopeBytes []byte) (*runner.GateIngestResult, error) {
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

	return runner.IngestGateResult(ctx, runner.GateIngestRequest{
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
	})
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
	claimSvc := cli.GetClaimService()

	return gatepersist.NewCoordinator(noteSvc, noteSvc, history, validator, transitioner, transitioner, claimSvc), nil
}
