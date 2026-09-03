package gatepersist

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// *services.NoteService already satisfies NoteWriter and NoteReader
// structurally (see interfaces.go) — no adapter needed for notes.
// *services.EntityHistoryService already satisfies HistoryReader
// structurally — no adapter needed for history.
// *services.ClaimService already satisfies LeaseReleaser structurally — no
// adapter needed for lease release.
//
// Transitioner and StatusValidator need a small amount of glue over
// *services.EntityService/*services.EntityRegistry and *workflow.Service
// respectively, because their existing methods take more parameters than
// this coordinator's narrow interfaces need (repo lookup, transition
// features, orchestrator-action resolution). Both adapters below are pure
// wiring: they add no business logic of their own, per "Keep persistence
// within the entity and note services" (REQ-F-002).

// WorkflowStatusValidator adapts a root *workflow.Service into
// StatusValidator by scoping to entityType's workflow level for each check.
// IsValidStatus alias-resolves (workflow.Service.IsValidStatus), so a
// pre-migration status name still counts as a defined status.
type WorkflowStatusValidator struct {
	workflowSvc *workflow.Service
}

// NewWorkflowStatusValidator constructs a WorkflowStatusValidator over the
// root (unscoped) workflow service.
func NewWorkflowStatusValidator(workflowSvc *workflow.Service) *WorkflowStatusValidator {
	if workflowSvc == nil {
		panic("gatepersist: NewWorkflowStatusValidator requires a non-nil workflow.Service")
	}
	return &WorkflowStatusValidator{workflowSvc: workflowSvc}
}

// IsValidStatus implements StatusValidator.
func (v *WorkflowStatusValidator) IsValidStatus(entityType models.EntityType, status string) bool {
	return v.workflowSvc.ForLevel(string(entityType)).IsValidStatus(status)
}

// EntityServiceTransitioner adapts *services.EntityService plus a
// *services.EntityRegistry (for per-entity-type repository lookup) into
// Transitioner. It always applies the transition with
// services.SimpleTransitionFeatures() (no backward-transition detection, no
// rejection-note side effect) because this coordinator writes its own
// bounded notes and encodes its own machine-readable kickback token in the
// transition reason — a second, differently-shaped rejection note would
// duplicate that record under a different identity.
type EntityServiceTransitioner struct {
	entitySvc   *services.EntityService
	registry    *services.EntityRegistry
	workflowSvc *workflow.Service
}

// NewEntityServiceTransitioner constructs an EntityServiceTransitioner.
func NewEntityServiceTransitioner(entitySvc *services.EntityService, registry *services.EntityRegistry, workflowSvc *workflow.Service) *EntityServiceTransitioner {
	if entitySvc == nil {
		panic("gatepersist: NewEntityServiceTransitioner requires a non-nil EntityService")
	}
	if registry == nil {
		panic("gatepersist: NewEntityServiceTransitioner requires a non-nil EntityRegistry")
	}
	if workflowSvc == nil {
		panic("gatepersist: NewEntityServiceTransitioner requires a non-nil workflow.Service")
	}
	return &EntityServiceTransitioner{entitySvc: entitySvc, registry: registry, workflowSvc: workflowSvc}
}

// CurrentStatus implements StatusReader, reusing the same registry lookup
// Transition uses. EntityServiceTransitioner therefore satisfies both
// Transitioner and StatusReader, so callers can wire one instance to both
// Coordinator fields.
//
// The raw stored status is alias-resolved to its canonical step name
// (route-based-workflow.md §5 "resolve on read") before returning, mirroring
// EntityService.GetNextStatus's own NormalizeStatus call. Coordinator's
// pre-transition and already-transitioned verification branches compare
// this value against Request.SourceStatus/TargetStatus, which callers
// always populate from NextStatusInfo.CurrentStatus — itself already
// alias-resolved. Without this, an entity still parked under a
// pre-migration alias (e.g. "ready_for_qa" for the "qa" step) would compare
// its raw alias against the canonical name and fail closed on an
// otherwise-healthy entity.
func (t *EntityServiceTransitioner) CurrentStatus(ctx context.Context, entityType models.EntityType, entityKey string) (string, error) {
	repo, err := t.registry.GetRepository(entityType)
	if err != nil {
		return "", fmt.Errorf("gatepersist: resolve repository for %s: %w", entityType, err)
	}
	entity, err := repo.GetByKey(ctx, entityKey)
	if err != nil {
		return "", fmt.Errorf("gatepersist: get %s %s: %w", entityType, entityKey, err)
	}
	return t.workflowSvc.ForLevel(string(entityType)).NormalizeStatus(entity.GetStatus()), nil
}

// ResolveEntityID implements IdentityResolver, reusing the same registry
// lookup CurrentStatus/Transition use. Returning the resolved entity's raw
// database ID (rather than its key) makes this the authoritative "same
// entity" check: GetByKey's alias/suffix-match resolution (e.g.
// FeatureRepository's bare "F05" suffix match) determines the ID, not any
// syntactic property of entityKey itself.
func (t *EntityServiceTransitioner) ResolveEntityID(ctx context.Context, entityType models.EntityType, entityKey string) (int64, error) {
	repo, err := t.registry.GetRepository(entityType)
	if err != nil {
		return 0, fmt.Errorf("gatepersist: resolve repository for %s: %w", entityType, err)
	}
	entity, err := repo.GetByKey(ctx, entityKey)
	if err != nil {
		return 0, fmt.Errorf("gatepersist: get %s %s: %w", entityType, entityKey, err)
	}
	return entity.GetID(), nil
}

// Transition implements Transitioner. guard's SessionID/FromStatus/Outcome
// are wired into services.TransitionOptions alongside GuardAdvance: true, so
// EntityService's advance_guard replay/CAS protection engages for this
// coordinator's parent-owned transitions whenever advance_guard.enabled is
// configured — mirroring internal/runner/controller.go's
// guardedTransitionOptions pattern for the runner's own dispatch-loop
// transitions. EntityService ANDs GuardAdvance with the configured
// advance_guard.enabled flag itself (shouldUseAdvanceGuard), so passing it
// unconditionally is safe for legacy/unguarded deployments.
func (t *EntityServiceTransitioner) Transition(ctx context.Context, entityType models.EntityType, entityKey, targetStatus, reason, agent string, guard TransitionGuard) (string, bool, error) {
	repo, err := t.registry.GetRepository(entityType)
	if err != nil {
		return "", false, fmt.Errorf("gatepersist: resolve repository for %s: %w", entityType, err)
	}
	scoped := t.entitySvc.ForLevel(string(entityType))
	result, err := scoped.TransitionStatus(ctx, repo, entityType, entityKey, targetStatus,
		services.TransitionOptions{
			Reason:       reason,
			Agent:        agent,
			SessionID:    guard.SessionID,
			FromStatus:   guard.FromStatus,
			Outcome:      guard.Outcome,
			GuardAdvance: true,
		},
		services.SimpleTransitionFeatures(),
		nil,
	)
	if err != nil {
		return "", false, err
	}
	return result.FromStatus, result.Transitioned, nil
}
