package services

import (
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// TransitionResult represents the outcome of a status transition.
type TransitionResult struct {
	EntityType         string                 `json:"entity_type"`   // "epic", "feature", "task"
	EntityKey          string                 `json:"entity_key"`
	FromStatus         string                 `json:"from_status"`
	ToStatus           string                 `json:"to_status"`
	Transitioned       bool                   `json:"transitioned"`
	Message            string                 `json:"message,omitempty"`
	OrchestratorAction *config.PopulatedAction `json:"orchestrator_action,omitempty"`
}

// NextStatusInfo contains the available transitions for an entity.
type NextStatusInfo struct {
	EntityType           string                   `json:"entity_type"`
	EntityKey            string                   `json:"entity_key"`
	CurrentStatus        string                   `json:"current_status"`
	CurrentPhase         string                   `json:"current_phase,omitempty"`
	AvailableTransitions []workflow.TransitionInfo `json:"available_transitions"`
	IsTerminal           bool                     `json:"is_terminal"`
	OrchestratorAction   *config.PopulatedAction  `json:"orchestrator_action,omitempty"`
}
