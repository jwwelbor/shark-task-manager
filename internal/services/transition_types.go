package services

import (
	"errors"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// Sentinel errors for status transitions.
var (
	// ErrReasonRequired indicates a reason is required for the transition.
	ErrReasonRequired = errors.New("reason is required for this transition")

	// ErrForceReasonRequired indicates --force requires --reason.
	ErrForceReasonRequired = errors.New("--force requires --reason to document why validation was bypassed")
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

	// Outcomes is the route-based outcome→target map for the current step
	// (E35-F02). Empty for legacy (status_flow) workflows and for
	// terminal/parking steps. When present, callers may release a semantic
	// outcome (pass/fail/blocked/…) instead of naming a target status.
	Outcomes map[string]string `json:"outcomes,omitempty"`
}

// TargetStatuses returns the list of target status strings from AvailableTransitions.
func (info *NextStatusInfo) TargetStatuses() []string {
	statuses := make([]string, len(info.AvailableTransitions))
	for i, t := range info.AvailableTransitions {
		statuses[i] = t.TargetStatus
	}
	return statuses
}
