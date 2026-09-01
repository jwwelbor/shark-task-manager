package gatepersist

import (
	"context"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// NoteWriter creates one typed, metadata-bearing note on an entity. Its
// signature matches *services.NoteService.AddNoteWithMetadata exactly, so
// that concrete service satisfies this interface with no adapter needed.
type NoteWriter interface {
	AddNoteWithMetadata(ctx context.Context, entityType models.EntityType, entityKey string, noteType string, content string, createdBy string, metadata string) (*models.EntityNote, error)
}

// NoteReader lists notes on an entity, optionally filtered by type. Its
// signature matches *services.NoteService.ListNotes exactly.
type NoteReader interface {
	ListNotes(ctx context.Context, entityType models.EntityType, entityKey string, noteTypes []string) ([]*models.EntityNote, error)
}

// HistoryReader lists an entity's status-transition history. Its signature
// matches *services.EntityHistoryService.GetHistory exactly.
type HistoryReader interface {
	GetHistory(ctx context.Context, entityType models.EntityType, entityKey string) ([]*models.EntityHistory, error)
}

// StatusValidator reports whether status is a status/outcome defined in
// entityType's own configured workflow. This is the target-entity
// workflow-membership check for Kickback.TargetStatus that
// internal/gateresult's package doc explicitly defers to this coordinator.
type StatusValidator interface {
	IsValidStatus(entityType models.EntityType, status string) bool
}

// Transitioner applies a workflow status transition to one entity, recording
// reason and agent on its audit trail. It must be idempotent: calling it
// again when the entity is already at targetStatus must succeed with
// transitioned=false rather than erroring (this is how the coordinator
// "verifies an already-applied identical target" without a second read
// path — see EntityService.TransitionStatus's step-2 idempotency check,
// which the default adapter delegates to).
type Transitioner interface {
	Transition(ctx context.Context, entityType models.EntityType, entityKey, targetStatus, reason, agent string) (fromStatus string, transitioned bool, err error)
}

// LeaseReleaser releases the parent's claim/lease session on an entity. Its
// signature matches *services.ClaimService.Release exactly.
type LeaseReleaser interface {
	Release(ctx context.Context, entityType, entityKey, sessionID, outcome string, force bool) (bool, error)
}
