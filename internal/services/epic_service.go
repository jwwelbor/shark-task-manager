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

// EpicNoteRepository defines the note repo interface needed by EpicService
// for creating rejection notes on backward transitions.
type EpicNoteRepository interface {
	CreateRejectionNote(ctx context.Context, entityType string, entityID int64,
		historyID int64, fromStatus, toStatus, reason, rejectedBy, documentPath string) error
}

// EpicFeatureCounter defines the feature counting interface needed by EpicService
// to count child features for backward transition warnings.
type EpicFeatureCounter interface {
	ListByEpic(ctx context.Context, epicID int64) ([]*models.Feature, error)
}

// EpicService provides business logic for epic operations.
type EpicService struct {
	repo        EpicRepository
	workflowSvc *workflow.Service
	noteRepo    EpicNoteRepository
	featureRepo EpicFeatureCounter
	docRepo     config.DocumentRepository
	relRepo     config.EpicRelationshipRepository
}

// NewEpicService creates a new EpicService.
// The workflow service is automatically scoped to the epic level.
// noteRepo and featureRepo can be nil for graceful degradation.
func NewEpicService(repo EpicRepository, workflowSvc *workflow.Service, noteRepo EpicNoteRepository, featureRepo EpicFeatureCounter) *EpicService {
	return &EpicService{
		repo:        repo,
		workflowSvc: workflowSvc.ForLevel(workflow.LevelEpic),
		noteRepo:    noteRepo,
		featureRepo: featureRepo,
		docRepo:     nil,
		relRepo:     nil,
	}
}

// NewEpicServiceWithRelationships creates a new EpicService with document and relationship repositories.
// Use this constructor when orchestrator actions need to populate related documents and epics.
func NewEpicServiceWithRelationships(repo EpicRepository, workflowSvc *workflow.Service, noteRepo EpicNoteRepository, featureRepo EpicFeatureCounter, docRepo config.DocumentRepository, relRepo config.EpicRelationshipRepository) *EpicService {
	return &EpicService{
		repo:        repo,
		workflowSvc: workflowSvc.ForLevel(workflow.LevelEpic),
		noteRepo:    noteRepo,
		featureRepo: featureRepo,
		docRepo:     docRepo,
		relRepo:     relRepo,
	}
}

// TransitionStatus validates and performs a status transition on an epic.
//
// Parameters:
//   - ctx: context
//   - epicKey: the epic key (e.g., "E16")
//   - targetStatus: the desired new status
//   - opts: transition options (force, reason, etc.)
//
// Returns:
//   - *TransitionResult: details of the transition
//   - error: validation or database errors
func (s *EpicService) TransitionStatus(ctx context.Context, epicKey string, targetStatus string, opts TransitionOptions) (*TransitionResult, error) {
	epic, err := s.repo.GetByKey(ctx, epicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get epic: %w", err)
	}
	if epic == nil {
		return nil, fmt.Errorf("epic not found: %s", epicKey)
	}

	currentStatus := string(epic.Status)

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
	epic.Status = models.EpicStatus(targetStatus)
	if err := s.repo.Update(ctx, epic); err != nil {
		return nil, fmt.Errorf("failed to update epic status: %w", err)
	}

	// Log rejection note for backward transitions with reason
	if (isBackward || opts.Force) && opts.Reason != "" && s.noteRepo != nil {
		_ = s.noteRepo.CreateRejectionNote(ctx, "epic", epic.ID,
			0, currentStatus, targetStatus,
			opts.Reason, opts.Agent, opts.DocumentPath)
	}

	// Count child features for warning
	var childCount int
	if s.featureRepo != nil {
		features, listErr := s.featureRepo.ListByEpic(ctx, epic.ID)
		if listErr == nil {
			childCount = len(features)
		}
	}

	action := s.resolveAction(ctx, epic, targetStatus)

	return &TransitionResult{
		EntityType:         "epic",
		EntityKey:          epicKey,
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
			OrchestratorAction: s.resolveAction(ctx, epic, t.TargetStatus),
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
// Uses EpicPlaceholdersWithRelated if document and relationship repositories are available,
// otherwise falls back to basic EpicPlaceholders.
func (s *EpicService) resolveAction(ctx context.Context, epic *models.Epic, status string) *config.PopulatedAction {
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
		// Use the new function that includes related documents and epics
		placeholders = config.EpicPlaceholdersWithRelated(epic, s.docRepo, s.relRepo, ctx)
	} else {
		// Fall back to basic placeholders (backward compatible)
		placeholders = config.EpicPlaceholders(epic)
	}

	return &config.PopulatedAction{
		Action:      meta.OrchestratorAction.Action,
		AgentType:   meta.OrchestratorAction.AgentType,
		Skills:      meta.OrchestratorAction.Skills,
		Instruction: meta.OrchestratorAction.PopulateTemplate(placeholders),
	}
}
