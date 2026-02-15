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

// FeatureNoteRepository defines the note repo interface needed by FeatureService
// for creating rejection notes on backward transitions.
type FeatureNoteRepository interface {
	CreateRejectionNote(ctx context.Context, entityType string, entityID int64,
		historyID int64, fromStatus, toStatus, reason, rejectedBy, documentPath string) error
}

// FeatureTaskCounter defines the task counting interface needed by FeatureService
// to count child tasks for backward transition warnings.
type FeatureTaskCounter interface {
	ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error)
}

// DocumentRepository defines the interface for accessing documents linked to entities.
// This is satisfied by implementations from the config or repository packages.
type DocumentRepository = config.DocumentRepository

// FeatureRelationshipRepository defines the interface for accessing feature relationships.
// This is satisfied by implementations from the config or repository packages.
type FeatureRelationshipRepository = config.FeatureRelationshipRepository

// FeatureService provides business logic for feature operations.
type FeatureService struct {
	repo        FeatureRepository
	workflowSvc *workflow.Service
	noteRepo    FeatureNoteRepository
	taskRepo    FeatureTaskCounter
	docRepo     DocumentRepository
	relRepo     FeatureRelationshipRepository
}

// NewFeatureService creates a new FeatureService.
// The workflow service is automatically scoped to the feature level.
// noteRepo, taskRepo, docRepo, and relRepo can be nil for graceful degradation.
func NewFeatureService(repo FeatureRepository, workflowSvc *workflow.Service, noteRepo FeatureNoteRepository, taskRepo FeatureTaskCounter) *FeatureService {
	return &FeatureService{
		repo:        repo,
		workflowSvc: workflowSvc.ForLevel(workflow.LevelFeature),
		noteRepo:    noteRepo,
		taskRepo:    taskRepo,
		docRepo:     nil,
		relRepo:     nil,
	}
}

// NewFeatureServiceWithRelationships creates a new FeatureService with document and relationship repositories.
// Use this constructor when orchestrator actions need to populate related documents and features.
func NewFeatureServiceWithRelationships(repo FeatureRepository, workflowSvc *workflow.Service, noteRepo FeatureNoteRepository, taskRepo FeatureTaskCounter, docRepo DocumentRepository, relRepo FeatureRelationshipRepository) *FeatureService {
	return &FeatureService{
		repo:        repo,
		workflowSvc: workflowSvc.ForLevel(workflow.LevelFeature),
		noteRepo:    noteRepo,
		taskRepo:    taskRepo,
		docRepo:     docRepo,
		relRepo:     relRepo,
	}
}

// TransitionStatus validates and performs a status transition on a feature.
//
// Parameters:
//   - ctx: context
//   - featureKey: the feature key (e.g., "E16-F01")
//   - targetStatus: the desired new status
//   - opts: transition options (force, reason, etc.)
//
// Returns:
//   - *TransitionResult: details of the transition
//   - error: validation or database errors
func (s *FeatureService) TransitionStatus(ctx context.Context, featureKey string, targetStatus string, opts TransitionOptions) (*TransitionResult, error) {
	feature, err := s.repo.GetByKey(ctx, featureKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature: %w", err)
	}
	if feature == nil {
		return nil, fmt.Errorf("feature not found: %s", featureKey)
	}

	currentStatus := string(feature.Status)

	// Validate transition (unless forced)
	if !opts.Force {
		if err := s.workflowSvc.ValidateTransition(currentStatus, targetStatus); err != nil {
			return nil, err
		}
	}

	// Normalize target status (unless forcing, where we accept any string)
	if !opts.Force {
		targetStatus = s.workflowSvc.NormalizeStatus(targetStatus)
	}

	// Enforce reason requirement for forced transitions
	if opts.Force && opts.Reason == "" {
		return nil, ErrForceReasonRequired
	}

	// Detect backward transition
	isBackward, err := s.workflowSvc.IsBackwardTransition(currentStatus, targetStatus)
	if err != nil {
		// If forcing, we might be transitioning to a status not in the workflow.
		// In this case, we can't determine if it's backward, so we assume it's not.
		if !opts.Force {
			return nil, fmt.Errorf("could not determine transition direction: %w", err)
		}
		isBackward = false
	}
	if isBackward && !opts.Force {
		wf := s.workflowSvc.GetWorkflow()
		requireReason := wf == nil || wf.RequireRejectionReason
		if requireReason && opts.Reason == "" {
			return nil, &BackwardReasonError{FromStatus: currentStatus, ToStatus: targetStatus}
		}
	}

	// Perform update
	feature.Status = models.FeatureStatus(targetStatus)
	if err := s.repo.Update(ctx, feature); err != nil {
		return nil, fmt.Errorf("failed to update feature status: %w", err)
	}

	// Log rejection note for backward transitions with reason
	if (isBackward || opts.Force) && opts.Reason != "" && s.noteRepo != nil {
		_ = s.noteRepo.CreateRejectionNote(ctx, "feature", feature.ID,
			0, currentStatus, targetStatus,
			opts.Reason, opts.Agent, opts.DocumentPath)
	}

	// Count child tasks for warning
	var childCount int
	if s.taskRepo != nil {
		tasks, listErr := s.taskRepo.ListByFeature(ctx, feature.ID)
		if listErr == nil {
			childCount = len(tasks)
		}
	}

	action := s.resolveAction(ctx, feature, targetStatus)

	return &TransitionResult{
		EntityType:         "feature",
		EntityKey:          featureKey,
		FromStatus:         currentStatus,
		ToStatus:           targetStatus,
		Transitioned:       true,
		OrchestratorAction: action,
		IsBackward:         isBackward,
		IsForced:           opts.Force,
		Reason:             opts.Reason,
		ChildCount:         childCount,
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
			OrchestratorAction: s.resolveAction(ctx, feature, t.TargetStatus),
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
// Uses FeaturePlaceholdersWithRelated to populate related documents and features if repositories are available.
func (s *FeatureService) resolveAction(ctx context.Context, feature *models.Feature, status string) *config.PopulatedAction {
	wf := s.workflowSvc.GetWorkflow()
	if wf == nil || wf.StatusMetadata == nil {
		return nil
	}
	meta, exists := wf.StatusMetadata[status]
	if !exists || meta.OrchestratorAction == nil {
		return nil
	}

	// Determine which placeholder function to use based on available repositories
	var placeholders map[string]string
	if s.docRepo != nil && s.relRepo != nil {
		// Use the new function that includes related documents and features
		placeholders = config.FeaturePlaceholdersWithRelated(ctx, feature, s.docRepo, s.relRepo)
	} else {
		// Fall back to basic placeholders (backward compatible)
		placeholders = config.FeaturePlaceholders(feature)
	}

	return &config.PopulatedAction{
		Action:      meta.OrchestratorAction.Action,
		AgentType:   meta.OrchestratorAction.AgentType,
		Skills:      meta.OrchestratorAction.Skills,
		Instruction: meta.OrchestratorAction.PopulateTemplate(placeholders),
	}
}
