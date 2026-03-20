package services

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// RejectionNoteCreator creates rejection notes during backward/forced transitions.
// Satisfied directly by *repository.EntityNoteRepository — no adapter needed.
type RejectionNoteCreator interface {
	CreateRejectionNote(ctx context.Context, entityType models.EntityType, entityID int64,
		historyID int64, fromStatus, toStatus, reason, rejectedBy string, documentPath *string) (*models.EntityNote, error)
}

// TransitionFeatures controls which optional steps of the TransitionStatus
// algorithm are active for a given entity type.
type TransitionFeatures struct {
	// DetectBackward enables backward transition detection and reason requirement.
	// Set to true for Epic, Feature, Task.
	// Set to false for Bug, ChangeCard.
	DetectBackward bool

	// CreateRejectionNotes enables rejection note creation during backward/forced
	// transitions. When true and EntityService.noteRepo is set, notes are created
	// directly by EntityService.TransitionStatus. No post-hook needed.
	CreateRejectionNotes bool

	// ResolveOrchestratorAction enables orchestrator action resolution.
	// Set to true for all entity types.
	ResolveOrchestratorAction bool
}

// DefaultTransitionFeatures returns the full feature set used by Epic, Feature, and Task.
func DefaultTransitionFeatures() TransitionFeatures {
	return TransitionFeatures{
		DetectBackward:            true,
		CreateRejectionNotes:      true,
		ResolveOrchestratorAction: true,
	}
}

// SimpleTransitionFeatures returns the reduced feature set used by Bug and ChangeCard.
func SimpleTransitionFeatures() TransitionFeatures {
	return TransitionFeatures{
		DetectBackward:            false,
		CreateRejectionNotes:      false,
		ResolveOrchestratorAction: true,
	}
}

// ResolveActionFn is a callback that generates a PopulatedAction for a given
// entity and target status. Entity-specific services provide this to include
// their enrichment data, document repos, and relationship repos in the
// placeholder generation.
type ResolveActionFn func(entity models.Entity, status string) *config.PopulatedAction

// EntityService provides shared status transition logic for all entity types.
// Entity-specific services compose this and delegate shared steps to it.
type EntityService struct {
	workflowSvc *workflow.Service
	noteRepo    RejectionNoteCreator // optional, for rejection notes during transitions
}

// NewEntityService creates an EntityService with the workflow service dependency.
// The provided workflow service should be the root (unscoped) service.
// Use ForLevel() to create level-scoped instances for specific entity types.
func NewEntityService(workflowSvc *workflow.Service) *EntityService {
	requireNonNil(workflowSvc, "EntityService requires a non-nil workflow.Service")
	return &EntityService{
		workflowSvc: workflowSvc,
	}
}

// ForLevel returns a new EntityService scoped to the specified workflow level.
// This is used by entity-specific services to get correct status flows for their entity type.
// Level constants: workflow.LevelEpic, workflow.LevelFeature, workflow.LevelTask.
func (s *EntityService) ForLevel(level string) *EntityService {
	return &EntityService{
		workflowSvc: s.workflowSvc.ForLevel(level),
		noteRepo:    s.noteRepo,
	}
}

// SetNoteRepo sets the rejection note repository. Optional — degrades gracefully.
// The *repository.EntityNoteRepository satisfies RejectionNoteCreator directly.
func (s *EntityService) SetNoteRepo(noteRepo RejectionNoteCreator) {
	s.noteRepo = noteRepo
}

// TransitionStatus performs a status transition on any entity via its
// EntityRepository adapter. The features parameter controls which optional
// steps are active.
//
// Steps performed:
//  1. Get entity by key via repo
//  2. Extract current status
//  3. Validate transition (unless forced)
//  4. Normalize target status (unless forced)
//  5. Enforce reason requirement for forced transitions
//  6. [opt-in] Detect backward transition and require reason
//  7. Update entity status via repo.UpdateStatus
//  8. [opt-in] Create rejection note (if backward/forced with reason and noteRepo set)
//  9. Resolve orchestrator action (opt-in, via resolveActionFn)
//
// 10. Build and return TransitionResult
func (s *EntityService) TransitionStatus(
	ctx context.Context,
	repo EntityRepository,
	entityType models.EntityType,
	key string,
	targetStatus string,
	opts TransitionOptions,
	features TransitionFeatures,
	resolveActionFn ResolveActionFn,
) (*TransitionResult, error) {
	// Step 1: Get entity
	entity, err := repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get %s: %w", entityType, err)
	}
	if entity == nil {
		return nil, fmt.Errorf("%s not found: %s", entityType, key)
	}

	currentStatus := entity.GetStatus()

	// Steps 3-4: Validate and normalize
	targetStatus, err = s.ValidateAndNormalize(currentStatus, targetStatus, opts.Force)
	if err != nil {
		return nil, err
	}

	// Step 5: Enforce reason for forced transitions
	if opts.Force && opts.Reason == "" {
		return nil, ErrForceReasonRequired
	}

	// Step 6: Backward detection (opt-in)
	var isBackward bool
	if features.DetectBackward {
		isBackward, err = s.DetectBackward(currentStatus, targetStatus, opts.Force, opts.Reason)
		if err != nil {
			return nil, err
		}
	}

	// Step 7: Update status via repo
	if err := repo.UpdateStatus(ctx, entity.GetID(), targetStatus); err != nil {
		return nil, fmt.Errorf("failed to update %s status: %w", entityType, err)
	}

	// Step 8: Create rejection note (opt-in, if backward/forced with reason)
	if features.CreateRejectionNotes && s.noteRepo != nil && (isBackward || opts.Force) && opts.Reason != "" {
		agent := opts.Agent
		var docPath *string
		if opts.DocumentPath != "" {
			docPath = &opts.DocumentPath
		}
		_, _ = s.noteRepo.CreateRejectionNote(ctx, entityType, entity.GetID(),
			0, currentStatus, targetStatus, opts.Reason, agent, docPath)
	}

	// Step 9: Resolve orchestrator action (opt-in)
	var action *config.PopulatedAction
	if features.ResolveOrchestratorAction && resolveActionFn != nil {
		action = resolveActionFn(entity, targetStatus)
	}

	// Step 10: Build result
	return &TransitionResult{
		EntityType:         entityType,
		EntityKey:          key,
		EntityID:           entity.GetID(),
		FromStatus:         currentStatus,
		ToStatus:           targetStatus,
		Transitioned:       true,
		OrchestratorAction: action,
		IsBackward:         isBackward,
		IsForced:           opts.Force,
		Reason:             opts.Reason,
		// ChildCount is set by the calling entity service in its post-hook
	}, nil
}

// ValidateAndNormalize validates a transition and normalizes the target status.
// If force is true, validation is skipped and the target is returned unchanged.
// Returns the (possibly normalized) target status and any validation error.
func (s *EntityService) ValidateAndNormalize(currentStatus, targetStatus string, force bool) (string, error) {
	if !force {
		if err := s.workflowSvc.ValidateTransition(currentStatus, targetStatus); err != nil {
			return "", err
		}
		targetStatus = s.workflowSvc.NormalizeStatus(targetStatus)
	}
	return targetStatus, nil
}

// DetectBackward checks if a transition is backward and enforces reason requirements.
// Returns isBackward flag and any error (BackwardReasonError if reason missing).
// If force is true and IsBackwardTransition errors, isBackward is set to false (graceful).
func (s *EntityService) DetectBackward(currentStatus, targetStatus string, force bool, reason string) (bool, error) {
	isBackward, err := s.workflowSvc.IsBackwardTransition(currentStatus, targetStatus)
	if err != nil {
		if !force {
			return false, fmt.Errorf("could not determine transition direction: %w", err)
		}
		return false, nil
	}
	if isBackward && !force {
		wf := s.workflowSvc.GetWorkflow()
		requireReason := wf == nil || wf.RequireRejectionReason
		if requireReason && reason == "" {
			return true, &BackwardReasonError{FromStatus: currentStatus, ToStatus: targetStatus}
		}
	}
	return isBackward, nil
}

// GetWorkflowService returns the underlying workflow service.
// Used by entity-specific services that need direct access for
// ValidateStatus, IsTerminalStatus, GetTransitionInfo, etc.
func (s *EntityService) GetWorkflowService() *workflow.Service {
	return s.workflowSvc
}

// ResolveActionForStatus resolves the orchestrator action for a target status.
// Uses the provided placeholders map to populate the action instruction template.
// Returns nil gracefully for nil workflow, missing metadata, or nil OrchestratorAction.
func (s *EntityService) ResolveActionForStatus(status string, placeholders map[string]string) *config.PopulatedAction {
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
		Instruction: meta.OrchestratorAction.PopulateTemplate(placeholders),
	}
}

// GetNextStatus returns available status transitions for an entity.
// The resolveActionFn callback generates entity-specific orchestrator actions
// per transition target.
func (s *EntityService) GetNextStatus(
	ctx context.Context,
	repo EntityRepository,
	entityType models.EntityType,
	key string,
	resolveActionFn ResolveActionFn,
) (*NextStatusInfo, error) {
	entity, err := repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get %s: %w", entityType, err)
	}
	if entity == nil {
		return nil, fmt.Errorf("%s not found: %s", entityType, key)
	}

	currentStatus := entity.GetStatus()
	transitions := s.workflowSvc.GetTransitionInfo(currentStatus)
	currentMeta := s.workflowSvc.GetStatusMetadata(currentStatus)

	wrapped := make([]TransitionInfoWithAction, 0, len(transitions))
	for _, t := range transitions {
		var action *config.PopulatedAction
		if resolveActionFn != nil {
			action = resolveActionFn(entity, t.TargetStatus)
		}
		wrapped = append(wrapped, TransitionInfoWithAction{
			TransitionInfo:     t,
			OrchestratorAction: action,
		})
	}

	return &NextStatusInfo{
		EntityType:           entityType,
		EntityKey:            key,
		CurrentStatus:        currentStatus,
		CurrentPhase:         currentMeta.Phase,
		AvailableTransitions: wrapped,
		IsTerminal:           s.workflowSvc.IsTerminalStatus(currentStatus),
	}, nil
}
