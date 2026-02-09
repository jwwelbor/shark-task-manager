package services

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// FeatureRepository defines the repository interface needed by FeatureService.
// This interface is satisfied by *repository.FeatureRepository.
type FeatureRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Feature, error)
	Update(ctx context.Context, feature *models.Feature) error
}

// FeatureService provides business logic for feature operations.
type FeatureService struct {
	repo        FeatureRepository
	workflowSvc *workflow.Service
}

// NewFeatureService creates a new FeatureService.
// The workflow service is automatically scoped to the feature level.
func NewFeatureService(repo FeatureRepository, workflowSvc *workflow.Service) *FeatureService {
	return &FeatureService{
		repo:        repo,
		workflowSvc: workflowSvc.ForLevel(workflow.LevelFeature),
	}
}

// TransitionStatus validates and performs a status transition on a feature.
//
// Parameters:
//   - ctx: context
//   - featureKey: the feature key (e.g., "E16-F01")
//   - targetStatus: the desired new status
//   - force: if true, bypass workflow validation
//
// Returns:
//   - *TransitionResult: details of the transition
//   - error: validation or database errors
func (s *FeatureService) TransitionStatus(ctx context.Context, featureKey string, targetStatus string, force bool) (*TransitionResult, error) {
	feature, err := s.repo.GetByKey(ctx, featureKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature: %w", err)
	}
	if feature == nil {
		return nil, fmt.Errorf("feature not found: %s", featureKey)
	}

	currentStatus := string(feature.Status)

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
	feature.Status = models.FeatureStatus(targetStatus)
	if err := s.repo.Update(ctx, feature); err != nil {
		return nil, fmt.Errorf("failed to update feature status: %w", err)
	}

	action := s.resolveAction(featureKey, targetStatus)

	return &TransitionResult{
		EntityType:         "feature",
		EntityKey:          featureKey,
		FromStatus:         currentStatus,
		ToStatus:           targetStatus,
		Transitioned:       true,
		OrchestratorAction: action,
	}, nil
}

// GetNextStatus returns the available transitions for the current status of a feature.
func (s *FeatureService) GetNextStatus(ctx context.Context, featureKey string) (*NextStatusInfo, error) {
	feature, err := s.repo.GetByKey(ctx, featureKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature: %w", err)
	}
	if feature == nil {
		return nil, fmt.Errorf("feature not found: %s", featureKey)
	}

	currentStatus := string(feature.Status)
	transitions := s.workflowSvc.GetTransitionInfo(currentStatus)
	currentMeta := s.workflowSvc.GetStatusMetadata(currentStatus)

	// Wrap transitions with action support
	wrapped := make([]TransitionInfoWithAction, 0, len(transitions))
	for _, t := range transitions {
		wrapped = append(wrapped, TransitionInfoWithAction{
			TransitionInfo:     t,
			OrchestratorAction: s.resolveAction(featureKey, t.TargetStatus),
		})
	}

	return &NextStatusInfo{
		EntityType:           "feature",
		EntityKey:            featureKey,
		CurrentStatus:        currentStatus,
		CurrentPhase:         currentMeta.Phase,
		AvailableTransitions: wrapped,
		IsTerminal:           s.workflowSvc.IsTerminalStatus(currentStatus),
	}, nil
}

// ValidateStatus checks if a status is valid in the feature workflow.
func (s *FeatureService) ValidateStatus(status string) error {
	return s.workflowSvc.ValidateStatus(status)
}

// resolveAction returns a populated orchestrator action for the given status,
// or nil if no action is defined for that status.
func (s *FeatureService) resolveAction(entityKey string, status string) *config.PopulatedAction {
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
		Instruction: meta.OrchestratorAction.PopulateTemplate(entityKey),
	}
}
