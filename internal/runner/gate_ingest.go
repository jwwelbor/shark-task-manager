// This file implements T-E34-F05-004's shared GateResult ingestion boundary
// (REQ-F-005 "Rider and core-runner parity"). Both the core runner (this
// package's handleSpawnAgent, for a gate_result_v1 step) and Rider's
// `shark run <entity-key> --apply-result=...` CLI surface
// (internal/cli/commands/run_apply_result.go) call IngestGateResult with the
// same envelope bytes and the same injected gatepersist.Coordinator, so a
// parity test comparing the two callers' output is comparing two thin
// wrappers over one function rather than two independently-implemented
// pipelines that merely happen to agree today.
//
// IngestGateResult owns exactly: decoding/validating the outer
// worker-control envelope (internal/workercontrol), requiring kind: final,
// decoding/validating the nested gate_result payload
// (internal/gateresult), resolving its semantic role via the caller-supplied
// outcome_roles map (REQ-F-006, resolved per step from the workflow's
// `outcome_roles` YAML field — T-E34-F05-005 — see resultContractFor),
// resolving the configured target status for the
// recommended outcome, and delegating persistence + the guarded transition
// to internal/gatepersist.Coordinator. It never falls back to the legacy
// recommendedOutcome parser on any failure (REQ-F-006: "must not fall back
// silently").
package runner

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workercontrol"
)

// GateIngestRequest binds a candidate worker-control envelope to the
// parent-observed run/entity/route context it must be validated and
// persisted against. Every non-envelope field is parent-observed, mirroring
// gatepersist.Request's own contract — none of them are derived from worker
// output.
type GateIngestRequest struct {
	// EnvelopeBytes is the worker's raw final response. The ENTIRE trimmed
	// value must be the worker-control envelope JSON object (mirroring
	// recommendedOutcome's "only a whole trimmed line/object" safety
	// property for the legacy path) — prose merely mentioning envelope-shaped
	// JSON must not be accepted.
	EnvelopeBytes []byte

	Coordinator *gatepersist.Coordinator

	ProjectRoot  string
	RunID        string
	EntityKey    string
	EntityType   models.EntityType
	SourceStatus string
	Gate         string
	Session      gatepersist.Session

	// OutcomeRoles maps each configured outcome key to its parent-owned
	// semantic role (REQ-F-006 outcome_roles). Populated by the caller from
	// the dispatched step's workflow configuration (T-E34-F05-005); an empty
	// map correctly fails closed for any recommended_outcome (no configured
	// role to validate against).
	OutcomeRoles map[string]gateresult.OutcomeRole
	// Outcomes maps each configured outcome key to its resolved target
	// status (services.NextStatusInfo.Outcomes).
	Outcomes map[string]string

	RetirementConfirmed bool
}

// GateIngestResult reports what IngestGateResult did, for the caller's own
// diagnostics/logging and for parity comparison in tests.
type GateIngestResult struct {
	*gatepersist.Result
	OutcomeKey string
	Role       gateresult.OutcomeRole
}

// IngestGateResult is the one exported ingestion boundary both execution
// paths call (REQ-F-005). It fails closed — returning an error with no
// coordinator call and no transition — on any envelope/payload/role/outcome
// validation failure, per REQ-F-006's "a gate explicitly configured for
// structured results must fail closed when the envelope is absent or
// malformed."
func IngestGateResult(ctx context.Context, req GateIngestRequest) (*GateIngestResult, error) {
	if req.Coordinator == nil {
		return nil, fmt.Errorf("gate ingestion: coordinator is required")
	}

	env, err := workercontrol.Decode(req.EnvelopeBytes)
	if err != nil {
		return nil, fmt.Errorf("gate ingestion: decode worker-control envelope: %w", err)
	}
	if env.Kind != workercontrol.KindFinal {
		return nil, fmt.Errorf("gate ingestion: expected worker-control envelope kind %q for a gate_result_v1 step, got %q", workercontrol.KindFinal, env.Kind)
	}
	if len(env.GateResult) == 0 {
		return nil, fmt.Errorf("gate ingestion: worker-control envelope is missing the required nested gate_result payload")
	}

	result, err := gateresult.Decode(env.GateResult)
	if err != nil {
		return nil, fmt.Errorf("gate ingestion: decode gate_result payload: %w", err)
	}

	role, ok := req.OutcomeRoles[env.RecommendedOutcome]
	if !ok {
		return nil, fmt.Errorf("gate ingestion: recommended_outcome %q has no configured outcome role", env.RecommendedOutcome)
	}
	if err := gateresult.ValidateRole(role, result, req.EntityKey); err != nil {
		return nil, fmt.Errorf("gate ingestion: validate gate_result against outcome role %q: %w", role, err)
	}

	targetStatus, ok := req.Outcomes[env.RecommendedOutcome]
	if !ok {
		return nil, fmt.Errorf("gate ingestion: recommended_outcome %q has no configured target status", env.RecommendedOutcome)
	}

	runDir, err := gaterun.RunDir(req.ProjectRoot, req.RunID)
	if err != nil {
		return nil, fmt.Errorf("gate ingestion: resolve run directory: %w", err)
	}

	evidenceJSON, err := marshalEvidence(env.Evidence)
	if err != nil {
		return nil, fmt.Errorf("gate ingestion: encode evidence: %w", err)
	}

	persisted, err := req.Coordinator.Persist(ctx, gatepersist.Request{
		RunDir:              runDir,
		RunID:               req.RunID,
		EntityKey:           req.EntityKey,
		EntityType:          req.EntityType,
		SourceStatus:        req.SourceStatus,
		Gate:                req.Gate,
		Session:             req.Session,
		EnvelopeJSON:        req.EnvelopeBytes,
		Result:              result,
		Role:                role,
		OutcomeKey:          env.RecommendedOutcome,
		Evidence:            evidenceJSON,
		TargetStatus:        targetStatus,
		RetirementConfirmed: req.RetirementConfirmed,
	})
	if err != nil {
		return nil, fmt.Errorf("gate ingestion: persist gate_result: %w", err)
	}

	return &GateIngestResult{Result: persisted, OutcomeKey: env.RecommendedOutcome, Role: role}, nil
}

// marshalEvidence encodes the outer envelope's common EvidenceRef collection
// as opaque JSON for gatepersist.Request.Evidence — gatepersist
// intentionally has no EvidenceRef type of its own (see its Request.Evidence
// doc comment), so this is the one place that shape is marshaled for it.
func marshalEvidence(evidence []workercontrol.EvidenceRef) ([]byte, error) {
	if len(evidence) == 0 {
		return nil, nil
	}
	return json.Marshal(evidence)
}
