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
	entitySvc *services.EntityService
	registry  *services.EntityRegistry
}

// NewEntityServiceTransitioner constructs an EntityServiceTransitioner.
func NewEntityServiceTransitioner(entitySvc *services.EntityService, registry *services.EntityRegistry) *EntityServiceTransitioner {
	if entitySvc == nil {
		panic("gatepersist: NewEntityServiceTransitioner requires a non-nil EntityService")
	}
	if registry == nil {
		panic("gatepersist: NewEntityServiceTransitioner requires a non-nil EntityRegistry")
	}
	return &EntityServiceTransitioner{entitySvc: entitySvc, registry: registry}
}

// CurrentStatus implements StatusReader, reusing the same registry lookup
// Transition uses. EntityServiceTransitioner therefore satisfies both
// Transitioner and StatusReader, so callers can wire one instance to both
// Coordinator fields.
func (t *EntityServiceTransitioner) CurrentStatus(ctx context.Context, entityType models.EntityType, entityKey string) (string, error) {
	repo, err := t.registry.GetRepository(entityType)
	if err != nil {
		return "", fmt.Errorf("gatepersist: resolve repository for %s: %w", entityType, err)
	}
	entity, err := repo.GetByKey(ctx, entityKey)
	if err != nil {
		return "", fmt.Errorf("gatepersist: get %s %s: %w", entityType, entityKey, err)
	}
	return entity.GetStatus(), nil
}

// Transition implements Transitioner.
func (t *EntityServiceTransitioner) Transition(ctx context.Context, entityType models.EntityType, entityKey, targetStatus, reason, agent string) (string, bool, error) {
	repo, err := t.registry.GetRepository(entityType)
	if err != nil {
		return "", false, fmt.Errorf("gatepersist: resolve repository for %s: %w", entityType, err)
	}
	scoped := t.entitySvc.ForLevel(string(entityType))
	result, err := scoped.TransitionStatus(ctx, repo, entityType, entityKey, targetStatus,
		services.TransitionOptions{Reason: reason, Agent: agent},
		services.SimpleTransitionFeatures(),
		nil,
	)
	if err != nil {
		return "", false, err
	}
	return result.FromStatus, result.Transitioned, nil
}
