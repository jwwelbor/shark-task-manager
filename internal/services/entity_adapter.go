package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// entityAdapterRepo is the minimal typed repository seam every generic
// EntityRepository adapter requires: load by key or ID, persist, and an
// atomic conditional status swap. PT is the entity's pointer type (e.g.
// *models.Bug, which must satisfy models.Entity); S is its status enum
// (e.g. models.BugStatus).
//
// UpdateStatusIfCurrent stays a required typed method rather than a
// synthesized one: the typed repositories implement it as a single atomic
// "UPDATE ... WHERE status = ?" statement, and a generic load-compare-save
// fallback built from these four methods would race under concurrent
// writers.
type entityAdapterRepo[PT models.Entity, S ~string] interface {
	GetByKey(ctx context.Context, key string) (PT, error)
	GetByID(ctx context.Context, id int64) (PT, error)
	Update(ctx context.Context, entity PT) error
	UpdateStatusIfCurrent(ctx context.Context, id int64, expected, next S) (bool, error)
}

// entityStatusSetter is an optional capability: typed repositories that
// expose a direct, non-conditional status write. When the wrapped
// repository doesn't implement it (e.g. ideas), entityAdapter emulates the
// same effect via models.Entity.SetStatus + Update.
type entityStatusSetter[S ~string] interface {
	UpdateStatus(ctx context.Context, id int64, status S) error
}

// entityContextDataRepo is an optional capability: typed repositories
// backed by a table with a context_data column. Entities without one
// (ideas, sprints) report "not supported" for both methods instead of
// silently discarding writes -- models.Entity's GetContextData/
// SetContextData are no-ops for those types, so a generic load-mutate-save
// fallback would hide data loss rather than surface it.
type entityContextDataRepo interface {
	GetContextData(ctx context.Context, id int64) (*string, error)
	UpdateContextData(ctx context.Context, id int64, data *string) error
}

// entityAdapter lifts a typed repository into the generic EntityRepository
// contract used by cross-cutting services (NoteService, ContextService,
// dispatch/transition gates, ...). One instantiation per entity type
// replaces what used to be a hand-copied ~65-line adapter file; behavioral
// differences across entities (no direct UpdateStatus, no context_data
// column) are expressed as optional capabilities the wrapped repository
// either does or doesn't implement, not as separate hand-written adapters.
//
// Entities whose typed repository doesn't fit this shape at all (task's
// UpdateStatus additionally threads agent/notes and task_history side
// effects) embed entityAdapter and override the divergent method instead
// of instantiating it directly; see task_repo_adapter.go.
type entityAdapter[PT models.Entity, S ~string] struct {
	kind string
	repo entityAdapterRepo[PT, S]
	// emulateContextData is set for entities whose model carries a real
	// ContextData field but whose typed repository has no dedicated
	// GetContextData/UpdateContextData methods (e.g. questions, which
	// round-trip context data through ConfigureWorkflow/Resolve/Withdraw
	// instead). When false, entities whose model has no such field
	// (ideas, sprints) must not silently "succeed": models.Entity's
	// generic GetContextData/SetContextData are no-ops for those types,
	// so emulating through them would hide data loss instead of
	// reporting "not supported".
	emulateContextData bool
}

// newEntityAdapter constructs a generic EntityRepository adapter whose
// context-data methods report "not supported" when the wrapped repository
// has no dedicated GetContextData/UpdateContextData methods. kind names
// the entity type (e.g. "Bug") for adapter error messages.
func newEntityAdapter[PT models.Entity, S ~string](kind string, repo entityAdapterRepo[PT, S]) *entityAdapter[PT, S] {
	return &entityAdapter[PT, S]{kind: kind, repo: repo}
}

// newEntityAdapterWithEmulatedContextData is like newEntityAdapter, but
// falls back to GetByID + models.Entity's generic GetContextData/
// SetContextData + Update when the wrapped repository has no dedicated
// context-data methods, instead of reporting "not supported". Use only for
// entities whose model type genuinely persists a context_data field
// through that generic accessor.
func newEntityAdapterWithEmulatedContextData[PT models.Entity, S ~string](kind string, repo entityAdapterRepo[PT, S]) *entityAdapter[PT, S] {
	return &entityAdapter[PT, S]{kind: kind, repo: repo, emulateContextData: true}
}

func (a *entityAdapter[PT, S]) GetByKey(ctx context.Context, key string) (models.Entity, error) {
	return a.repo.GetByKey(ctx, key)
}

func (a *entityAdapter[PT, S]) GetByID(ctx context.Context, id int64) (models.Entity, error) {
	return a.repo.GetByID(ctx, id)
}

func (a *entityAdapter[PT, S]) Update(ctx context.Context, entity models.Entity) error {
	typed, ok := entity.(PT)
	if !ok {
		var zero PT
		return fmt.Errorf("%sRepositoryAdapter.Update: expected %T, got %T", a.kind, zero, entity)
	}
	return a.repo.Update(ctx, typed)
}

func (a *entityAdapter[PT, S]) UpdateStatus(ctx context.Context, id int64, status string) error {
	if setter, ok := a.repo.(entityStatusSetter[S]); ok {
		return setter.UpdateStatus(ctx, id, S(status))
	}
	entity, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("%sRepositoryAdapter.UpdateStatus: load: %w", a.kind, err)
	}
	entity.SetStatus(status)
	return a.repo.Update(ctx, entity)
}

func (a *entityAdapter[PT, S]) UpdateStatusIfCurrent(ctx context.Context, id int64, expectedCurrentStatus, newStatus string) (bool, error) {
	return a.repo.UpdateStatusIfCurrent(ctx, id, S(expectedCurrentStatus), S(newStatus))
}

func (a *entityAdapter[PT, S]) GetContextData(ctx context.Context, id int64) (*string, error) {
	if cd, ok := a.repo.(entityContextDataRepo); ok {
		return cd.GetContextData(ctx, id)
	}
	if !a.emulateContextData {
		return nil, a.contextDataUnsupported()
	}
	entity, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%sRepositoryAdapter.GetContextData: load: %w", a.kind, err)
	}
	return entity.GetContextData(), nil
}

func (a *entityAdapter[PT, S]) UpdateContextData(ctx context.Context, id int64, data *string) error {
	if cd, ok := a.repo.(entityContextDataRepo); ok {
		return cd.UpdateContextData(ctx, id, data)
	}
	if !a.emulateContextData {
		return a.contextDataUnsupported()
	}
	entity, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("%sRepositoryAdapter.UpdateContextData: load: %w", a.kind, err)
	}
	entity.SetContextData(data)
	if err := a.repo.Update(ctx, entity); err != nil {
		return fmt.Errorf("%sRepositoryAdapter.UpdateContextData: update: %w", a.kind, err)
	}
	return nil
}

func (a *entityAdapter[PT, S]) contextDataUnsupported() error {
	return fmt.Errorf("%sRepositoryAdapter: context_data is not supported for %ss", a.kind, strings.ToLower(a.kind))
}
