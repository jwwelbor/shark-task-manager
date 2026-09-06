package services

import (
	"errors"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// Sentinel errors for status transitions.
var (
	// ErrReasonRequired indicates a reason is required for the transition.
	ErrReasonRequired = errors.New("reason is required for this transition")

	// ErrForceReasonRequired indicates --force requires --reason.
	ErrForceReasonRequired = errors.New("--force requires --reason to document why validation was bypassed")

	// ErrAdvanceGuardSessionRequired indicates guarded advances need a session id.
	ErrAdvanceGuardSessionRequired = errors.New("advance guard requires --session for guarded advances")

	// ErrAdvanceGuardFromStatusRequired indicates guarded advances need an expected from-status.
	ErrAdvanceGuardFromStatusRequired = errors.New("advance guard requires --from-status for guarded advances")

	// ErrAdvanceGuardRepeatRejected indicates the guarded advance was already consumed.
	ErrAdvanceGuardRepeatRejected = errors.New("guarded advance replay rejected")

	// ErrAdvanceGuardStaleFromStatus indicates the entity moved away from the expected status.
	ErrAdvanceGuardStaleFromStatus = errors.New("guarded advance rejected because the entity is no longer at --from-status")

	// ErrAdvanceGuardForceRepeatNotAllowed indicates the operator attempted an override while disabled in config.
	ErrAdvanceGuardForceRepeatNotAllowed = errors.New("advance guard override is disabled by config")

	// ErrAdvanceGuardForceRepeatReasonRequired indicates replay overrides must be auditable.
	ErrAdvanceGuardForceRepeatReasonRequired = errors.New("--force-repeat requires --reason")
)

// BackwardReasonError is returned when a backward transition is missing a reason.
type BackwardReasonError struct {
	FromStatus string
	ToStatus   string
}

func (e *BackwardReasonError) Error() string {
	return fmt.Sprintf("backward transition from '%s' to '%s' requires --reason flag", e.FromStatus, e.ToStatus)
}

func (e *BackwardReasonError) Is(target error) bool {
	return target == ErrReasonRequired
}

// TransitionOptions controls behavior of status transitions.
// Used by EpicService.TransitionStatus() and FeatureService.TransitionStatus().
type TransitionOptions struct {
	Force        bool   `json:"force,omitempty"`
	Reason       string `json:"reason,omitempty"`
	DocumentPath string `json:"document_path,omitempty"`
	Agent        string `json:"agent,omitempty"`
	SessionID    string `json:"session_id,omitempty"`
	FromStatus   string `json:"from_status,omitempty"`
	Outcome      string `json:"outcome,omitempty"`
	ForceRepeat  bool   `json:"force_repeat,omitempty"`
	GuardAdvance bool   `json:"guard_advance,omitempty"`
}

// TransitionResult represents the outcome of a status transition.
type TransitionResult struct {
	EntityType         models.EntityType       `json:"entity_type"` // "epic", "feature", "task"
	EntityKey          string                  `json:"entity_key"`
	EntityID           int64                   `json:"entity_id,omitempty"`
	FromStatus         string                  `json:"from_status"`
	ToStatus           string                  `json:"to_status"`
	Transitioned       bool                    `json:"transitioned"`
	Message            string                  `json:"message,omitempty"`
	OrchestratorAction *config.PopulatedAction `json:"orchestrator_action"`
	IsBackward         bool                    `json:"is_backward,omitempty"`
	IsForced           bool                    `json:"is_forced,omitempty"`
	Reason             string                  `json:"reason,omitempty"`
	ChildCount         int                     `json:"child_count,omitempty"`
}

// TransitionInfoWithAction wraps a TransitionInfo with an optional orchestrator action.
// The embedded TransitionInfo fields are flattened in JSON serialization.
type TransitionInfoWithAction struct {
	workflow.TransitionInfo
	OrchestratorAction *config.PopulatedAction `json:"orchestrator_action"`
}

// NextStatusInfo contains the available transitions for an entity.
type NextStatusInfo struct {
	EntityType           models.EntityType          `json:"entity_type"`
	EntityKey            string                     `json:"entity_key"`
	CurrentStatus        string                     `json:"current_status"`
	CurrentPhase         string                     `json:"current_phase,omitempty"`
	AvailableTransitions []TransitionInfoWithAction `json:"available_transitions"`
	IsTerminal           bool                       `json:"is_terminal"`

	// IsClaimed distinguishes a live lease from a terminal workflow state. A
	// keyed dispatcher must not issue a second worker while this is true, but
	// the parent holding that lease must still be able to advance its workflow
	// status before releasing it.
	IsClaimed bool `json:"is_claimed,omitempty"`

	// Outcomes is the route-based outcome→target map for the current step
	// (E35-F02). Empty for legacy (status_flow) workflows and for
	// terminal/parking steps. When present, callers may release a semantic
	// outcome (pass/fail/blocked/…) instead of naming a target status.
	Outcomes map[string]string `json:"outcomes,omitempty"`

	// ResultContract is the REQ-F-006 resolved worker-result contract for
	// the current step: "legacy" or "gate_result_v1". Always populated
	// ("legacy" is the default for omission/legacy workflows) so both the
	// core runner and Rider (via `shark next --json`) consume the exact
	// same resolved value instead of deriving it independently.
	ResultContract string `json:"result_contract"`

	// OutcomeRoles maps each key in Outcomes to its REQ-F-006 semantic role
	// (success, route_rework, kickback_rework, blocked, hold, cancelled).
	// Empty/nil for a "legacy" step.
	OutcomeRoles map[string]gateresult.OutcomeRole `json:"outcome_roles,omitempty"`
}

// TargetStatuses returns the list of target status strings from AvailableTransitions.
func (info *NextStatusInfo) TargetStatuses() []string {
	statuses := make([]string, len(info.AvailableTransitions))
	for i, t := range info.AvailableTransitions {
		statuses[i] = t.TargetStatus
	}
	return statuses
}
