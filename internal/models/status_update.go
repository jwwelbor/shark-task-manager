package models

import (
	"database/sql"
	"errors"
)

// ErrGuardedUpdateStale is returned by TaskRepository.StatusUpdateRaw/
// StatusUpdateRawWithTx when params.Guarded is true and the row's current
// status no longer matches params.OldStatus at the time of the UPDATE — i.e.
// the compare-and-swap lost the race or the caller's view of the current
// status was stale. Defined here (not in internal/repository/task) so the
// service layer can check for it via errors.Is without importing the
// concrete repository package.
var ErrGuardedUpdateStale = errors.New("task status update: current status no longer matches expected (guarded update stale)")

// ErrAdvanceGuardAlreadyConsumed reports that a replay-protection tuple was
// recorded already. It lives in models so services can use errors.Is without
// importing a concrete repository package.
var ErrAdvanceGuardAlreadyConsumed = errors.New("guarded advance already consumed")

// StatusUpdateParams contains all parameters for a raw (no-validation) status update.
// This struct is defined in the models package to avoid circular imports between
// the services and repository packages.
//
// Used by TaskRepository.StatusUpdateRaw() which performs an atomic status update
// without any business-logic validation (transitions, backward checks, etc.).
// All validation is expected to be done by the service layer before calling this.
type StatusUpdateParams struct {
	// TaskID is the database ID of the task to update.
	TaskID int64

	// NewStatus is the target status string.
	NewStatus TaskStatus

	// Agent is the optional agent performing the transition.
	Agent *string

	// Notes is optional notes for the transition.
	Notes *string

	// RejectionReason is optional reason for backward transitions.
	RejectionReason *string

	// DocumentPath is optional path to a related document.
	DocumentPath *string

	// Force indicates whether this is a forced transition (bypassing validation).
	Force bool

	// OldStatus is the current status before the transition (set by service layer).
	// This is used for history records and rejection notes.
	OldStatus string

	// TaskKey is the task key (e.g., "T-E07-F01-001"), used for auto-unblock logic.
	TaskKey string

	// StartedAt, CompletedAt, BlockedAt are current timestamp values from the task.
	// Used to determine whether timestamp columns need updating.
	StartedAt   sql.NullTime
	CompletedAt sql.NullTime
	BlockedAt   sql.NullTime

	// TerminalStatuses is the authoritative list of terminal task statuses,
	// used to decide (a) whether this transition finishes the task and should
	// therefore auto-unblock its dependents, and (b) whether each dependency of
	// a blocked dependent is satisfied. Callers should populate this from
	// workflow.Service.ForLevel(workflow.LevelTask).GetTerminalStatuses() so
	// custom workflows that rename terminal statuses keep working. When empty
	// the repository falls back to the historical hardcoded pair
	// (completed/archived) so callers that do not yet supply it keep working —
	// same contract as BugListFilters.TerminalStatuses and
	// ChangeCardRepoFilter.TerminalStatuses.
	TerminalStatuses []string

	// Guarded, when true, makes the UPDATE conditional on the row's current
	// status still matching OldStatus (case-insensitive), evaluated atomically
	// as part of the single UPDATE statement rather than a separate read. Used
	// for advance_guard compare-and-swap semantics; see
	// TaskRepository.StatusUpdateRawWithTx and ErrGuardedUpdateStale.
	Guarded bool
}
