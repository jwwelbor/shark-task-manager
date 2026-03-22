package commands

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// entityTransitioner is an interface for performing status transitions on entities.
type entityTransitioner interface {
	TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
}

// performEntityTransition executes a status transition via the service layer.
func performEntityTransition(ctx context.Context, svc entityTransitioner, entityKey string, targetStatus string, opts services.TransitionOptions, result *EntityNextStatusResult) error {
	if opts.Force {
		cli.Warning("Workflow validation bypassed with --force")
	}

	transResult, err := svc.TransitionStatus(ctx, entityKey, targetStatus, opts)
	if err != nil {
		return fmt.Errorf("failed to transition status: %w", err)
	}

	result.NewStatus = transResult.ToStatus
	result.Transitioned = true

	if cli.GlobalConfig.JSON {
		result.Message = fmt.Sprintf("Transitioned: %s -> %s", transResult.FromStatus, transResult.ToStatus)
		result.IsBackward = transResult.IsBackward
		result.Reason = transResult.Reason
		result.ChildCount = transResult.ChildCount
		return cli.OutputJSON(result)
	}

	cli.Success(fmt.Sprintf("Transitioned: %s -> %s", transResult.FromStatus, transResult.ToStatus))
	if transResult.IsBackward && transResult.Reason != "" {
		cli.Info(fmt.Sprintf("Reason: %s", transResult.Reason))
	}
	if transResult.ChildCount > 0 {
		cli.Warning(fmt.Sprintf("%d child entities remain in current states.", transResult.ChildCount))
	}
	displayOrchestratorAction(transResult.OrchestratorAction)
	return nil
}

// EntityNextStatusResult contains the result of a next-status operation for epics/features.
type EntityNextStatusResult struct {
	EntityType           string                   `json:"entity_type"`
	EntityKey            string                   `json:"entity_key"`
	CurrentStatus        string                   `json:"current_status"`
	CurrentPhase         string                   `json:"current_phase,omitempty"`
	AvailableTransitions []EntityTransitionChoice `json:"available_transitions"`
	NewStatus            string                   `json:"new_status,omitempty"`
	Transitioned         bool                     `json:"transitioned"`
	Message              string                   `json:"message,omitempty"`
	IsBackward           bool                     `json:"is_backward,omitempty"`
	Reason               string                   `json:"reason,omitempty"`
	ChildCount           int                      `json:"child_count,omitempty"`
}

// EntityTransitionChoice represents a valid status transition for display.
type EntityTransitionChoice struct {
	Number      int    `json:"number"`
	Status      string `json:"status"`
	Description string `json:"description,omitempty"`
	Phase       string `json:"phase,omitempty"`
}

// buildNextStatusResult constructs an EntityNextStatusResult from service info.
func buildNextStatusResult(entityType string, info *services.NextStatusInfo) *EntityNextStatusResult {
	result := &EntityNextStatusResult{
		EntityType:    entityType,
		EntityKey:     info.EntityKey,
		CurrentStatus: info.CurrentStatus,
		CurrentPhase:  info.CurrentPhase,
	}

	for i, t := range info.AvailableTransitions {
		result.AvailableTransitions = append(result.AvailableTransitions, EntityTransitionChoice{
			Number:      i + 1,
			Status:      t.TargetStatus,
			Description: t.Description,
			Phase:       t.Phase,
		})
	}

	return result
}

// printEntityTransitions prints available transitions in a formatted list.
func printEntityTransitions(transitions []EntityTransitionChoice) {
	for _, t := range transitions {
		fmt.Printf("  %d) %s", t.Number, t.Status)
		if t.Phase != "" {
			fmt.Printf(" (phase: %s)", t.Phase)
		}
		fmt.Println()
		if t.Description != "" {
			fmt.Printf("     \"%s\"\n", t.Description)
		}
		fmt.Println()
	}
}
