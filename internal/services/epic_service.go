package services

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// EpicRepository defines the repository interface needed by EpicService.
// This interface is satisfied by *repository.EpicRepository.
type EpicRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Epic, error)
	Update(ctx context.Context, epic *models.Epic) error
}

// EpicService provides business logic for epic operations.
type EpicService struct {
	repo        EpicRepository
	workflowSvc *workflow.Service
}

// NewEpicService creates a new EpicService.
// The workflow service is automatically scoped to the epic level.
func NewEpicService(repo EpicRepository, workflowSvc *workflow.Service) *EpicService {
	return &EpicService{
		repo:        repo,
		workflowSvc: workflowSvc.ForLevel(workflow.LevelEpic),
	}
}

// TransitionStatus validates and performs a status transition on an epic.
//
// Parameters:
//   - ctx: context
//   - epicKey: the epic key (e.g., "E16")
//   - targetStatus: the desired new status
//   - force: if true, bypass workflow validation
//
// Returns:
//   - *TransitionResult: details of the transition
//   - error: validation or database errors
func (s *EpicService) TransitionStatus(ctx context.Context, epicKey string, targetStatus string, force bool) (*TransitionResult, error) {
	epic, err := s.repo.GetByKey(ctx, epicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get epic: %w", err)
	}
	if epic == nil {
		return nil, fmt.Errorf("epic not found: %s", epicKey)
	}

	currentStatus := string(epic.Status)

	// Validate transition (unless forced)
	if !force {
		if err := s.workflowSvc.ValidateTransition(currentStatus, targetStatus); err != nil {
			return nil, err
		}
	}

	// Normalize target status (unless forcing, where we accept any string)
	if !force {
		targetStatus = s.workflowSvc.NormalizeStatus(targetStatus)
	}

	// Perform update
	epic.Status = models.EpicStatus(targetStatus)
	if err := s.repo.Update(ctx, epic); err != nil {
		return nil, fmt.Errorf("failed to update epic status: %w", err)
	}

	action := s.resolveAction(epic, targetStatus)

	return &TransitionResult{
		EntityType:         "epic",
		EntityKey:          epicKey,
		FromStatus:         currentStatus,
		ToStatus:           targetStatus,
		Transitioned:       true,
		OrchestratorAction: action,
	}, nil
}

// GetNextStatus returns the available transitions for the current status of an epic.
func (s *EpicService) GetNextStatus(ctx context.Context, epicKey string) (*NextStatusInfo, error) {
	epic, err := s.repo.GetByKey(ctx, epicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get epic: %w", err)
	}
	if epic == nil {
		return nil, fmt.Errorf("epic not found: %s", epicKey)
	}

	currentStatus := string(epic.Status)
	transitions := s.workflowSvc.GetTransitionInfo(currentStatus)
	currentMeta := s.workflowSvc.GetStatusMetadata(currentStatus)

	// Wrap transitions with action support
	wrapped := make([]TransitionInfoWithAction, 0, len(transitions))
	for _, t := range transitions {
		wrapped = append(wrapped, TransitionInfoWithAction{
			TransitionInfo:     t,
			OrchestratorAction: s.resolveAction(epic, t.TargetStatus),
		})
	}

	return &NextStatusInfo{
		EntityType:           "epic",
		EntityKey:            epicKey,
		CurrentStatus:        currentStatus,
		CurrentPhase:         currentMeta.Phase,
		AvailableTransitions: wrapped,
		IsTerminal:           s.workflowSvc.IsTerminalStatus(currentStatus),
	}, nil
}

// ValidateStatus checks if a status is valid in the epic workflow.
func (s *EpicService) ValidateStatus(status string) error {
	return s.workflowSvc.ValidateStatus(status)
}

// resolveAction looks up the orchestrator action for a given status in the workflow config.
// Returns nil if no action is defined for the status, or if the workflow config is nil.
func (s *EpicService) resolveAction(epic *models.Epic, status string) *config.PopulatedAction {
	wf := s.workflowSvc.GetWorkflow()
	if wf == nil || wf.StatusMetadata == nil {
		return nil
	}
	meta, exists := wf.StatusMetadata[status]
	if !exists || meta.OrchestratorAction == nil {
		return nil
	}
	return &config.PopulatedAction{
		Action:      meta.OrchestratorAction.Action,
		AgentType:   meta.OrchestratorAction.AgentType,
		Skills:      meta.OrchestratorAction.Skills,
		Instruction: meta.OrchestratorAction.PopulateTemplate(config.EpicPlaceholders(epic)),
	}
}
