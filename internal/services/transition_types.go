package services

import (
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

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
	EntityType         string                  `json:"entity_type"` // "epic", "feature", "task"
	EntityKey          string                  `json:"entity_key"`
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
	EntityType           string                     `json:"entity_type"`
	EntityKey            string                     `json:"entity_key"`
	CurrentStatus        string                     `json:"current_status"`
	CurrentPhase         string                     `json:"current_phase,omitempty"`
	AvailableTransitions []TransitionInfoWithAction `json:"available_transitions"`
	IsTerminal           bool                       `json:"is_terminal"`
}
