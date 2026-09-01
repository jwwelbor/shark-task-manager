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

// TransitionGuard carries the replay-protection tuple a parent-owned
// transition must supply so services.EntityService's advance_guard can
// engage (session id, the expected pre-transition status, and the semantic
// outcome driving the transition). It mirrors
// internal/runner/controller.go's guardedTransitionOptions helper for this
// package's own caller shape: gatepersist has no *services.NextStatusInfo
// to resolve an outcome name from, so callers supply Outcome directly
// (Request.OutcomeKey — the worker's own recommended_outcome, the same
// workflow-outcome-name role guardedTransitionOptions resolves for the
// runner's dispatch-loop transitions).
//
// GuardAdvance is set unconditionally true by every caller in this package,
// exactly as guardedTransitionOptions does — services.EntityService itself
// ANDs it with the configured advance_guard.enabled flag
// (shouldUseAdvanceGuard), so a legacy/unguarded deployment is unaffected
// and this package never needs its own copy of that config decision.
type TransitionGuard struct {
	SessionID  string
	FromStatus string
	Outcome    string
}

// Transitioner applies a workflow status transition to one entity, recording
// reason and agent on its audit trail. It must be idempotent: calling it
// again when the entity is already at targetStatus must succeed with
// transitioned=false rather than erroring (this is how the coordinator
// "verifies an already-applied identical target" without a second read
// path — see EntityService.TransitionStatus's step-2 idempotency check,
// which the default adapter delegates to).
type Transitioner interface {
	Transition(ctx context.Context, entityType models.EntityType, entityKey, targetStatus, reason, agent string, guard TransitionGuard) (fromStatus string, transitioned bool, err error)
}

// LeaseReleaser releases the parent's claim/lease session on an entity. Its
// signature matches *services.ClaimService.Release exactly.
type LeaseReleaser interface {
	Release(ctx context.Context, entityType, entityKey, sessionID, outcome string, force bool) (bool, error)
}

// StatusReader reads an entity's current live status. It is used only on
// the already-transitioned resume path (state.PersistenceState ==
// gaterun.PersistenceStateTransitioned): architecture.md step 8 requires
// that path to "verify the expected live target state" without repeating
// the transition call, so this coordinator never trusts Transitioner's own
// idempotency to stand in for that verification once a transition is
// already durably recorded applied.
type StatusReader interface {
	CurrentStatus(ctx context.Context, entityType models.EntityType, entityKey string) (string, error)
}
