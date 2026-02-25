package models

import "database/sql"

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
}
