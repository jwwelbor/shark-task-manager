package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// EpicRepository defines the repository interface needed by EpicService.
// This interface is satisfied by *repository.EpicRepository.
type EpicRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Epic, error)
	Update(ctx context.Context, epic *models.Epic) error
	List(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error)
	GetFeatureProgressDataByEpic(ctx context.Context, epicID int64) ([]repository.FeatureProgressData, error)
	GetFeatureStatusBreakdownByKey(ctx context.Context, epicKey string) (map[models.FeatureStatus]int, error)
	GetFeatureStatusRollup(ctx context.Context, epicID int64) (map[string]int, error)
	GetTaskStatusRollup(ctx context.Context, epicID int64) (map[string]int, error)
}

// EpicTaskLister defines the task repository interface needed by EpicService
// for querying blocked tasks across an epic.
type EpicTaskLister interface {
	ListBlockedTasksByEpic(ctx context.Context, epicKey string) ([]*models.Task, error)
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
	taskRepo    EpicTaskLister
	docRepo     config.DocumentRepository
	relRepo     config.EpicRelationshipRepository
}

// NewEpicService creates a new EpicService.
// The workflow service is automatically scoped to the epic level.
// noteRepo and featureRepo can be nil for graceful degradation.
//
// Panics:
//   - If repo is nil (required dependency)
//   - If workflowSvc is nil (required dependency)
func NewEpicService(repo EpicRepository, workflowSvc *workflow.Service, noteRepo EpicNoteRepository, featureRepo EpicFeatureCounter, taskRepo EpicTaskLister) *EpicService {
	if repo == nil {
		panic("EpicService requires a non-nil EpicRepository")
	}
	if workflowSvc == nil {
		panic("EpicService requires a non-nil workflow.Service")
	}
	return &EpicService{
		repo:        repo,
		workflowSvc: workflowSvc.ForLevel(workflow.LevelEpic),
		noteRepo:    noteRepo,
		featureRepo: featureRepo,
		taskRepo:    taskRepo,
		docRepo:     nil,
		relRepo:     nil,
	}
}

// NewEpicServiceWithRelationships creates a new EpicService with document and relationship repositories.
// Use this constructor when orchestrator actions need to populate related documents and epics.
//
// Panics:
//   - If repo is nil (required dependency)
//   - If workflowSvc is nil (required dependency)
func NewEpicServiceWithRelationships(repo EpicRepository, workflowSvc *workflow.Service, noteRepo EpicNoteRepository, featureRepo EpicFeatureCounter, docRepo config.DocumentRepository, relRepo config.EpicRelationshipRepository) *EpicService {
	if repo == nil {
		panic("EpicService requires a non-nil EpicRepository")
	}
	if workflowSvc == nil {
		panic("EpicService requires a non-nil workflow.Service")
	}
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

// GetEpic retrieves an epic by key.
func (s *EpicService) GetEpic(ctx context.Context, key string) (*models.Epic, error) {
	epic, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get epic %s: %w", key, err)
	}
	if epic == nil {
		return nil, fmt.Errorf("epic not found: %s", key)
	}
	return epic, nil
}

// ListEpics retrieves epics with optional filtering.
func (s *EpicService) ListEpics(ctx context.Context, filters EpicFilters) ([]*models.Epic, error) {
	var statusPtr *models.EpicStatus
	if filters.Status != "" {
		status := models.EpicStatus(filters.Status)
		statusPtr = &status
	}
	epics, err := s.repo.List(ctx, statusPtr)
	if err != nil {
		return nil, fmt.Errorf("failed to list epics: %w", err)
	}
	return epics, nil
}

// CalculateProgress computes epic progress from raw feature data.
// Business rule: completed/archived features count as 100% progress regardless
// of their stored progress_pct value. All other features use their stored
// progress_pct. Epic progress is the average across all features.
// Returns 0 if the epic has no features.
func (s *EpicService) CalculateProgress(ctx context.Context, epicID int64) (float64, error) {
	data, err := s.repo.GetFeatureProgressDataByEpic(ctx, epicID)
	if err != nil {
		return 0, fmt.Errorf("failed to get feature progress data: %w", err)
	}

	if len(data) == 0 {
		return 0, nil
	}

	var totalProgress float64
	for _, d := range data {
		if d.Status == "completed" || d.Status == "archived" {
			totalProgress += 100.0
		} else {
			totalProgress += d.ProgressPct
		}
	}

	return totalProgress / float64(len(data)), nil
}

// GetProgress retrieves progress metrics for an epic.
func (s *EpicService) GetProgress(ctx context.Context, key string) (*EpicProgressInfo, error) {
	epic, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get epic %s: %w", key, err)
	}
	if epic == nil {
		return nil, fmt.Errorf("epic not found: %s", key)
	}

	progressPct, err := s.CalculateProgress(ctx, epic.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate progress for epic %s: %w", key, err)
	}

	featureRollup, err := s.repo.GetFeatureStatusRollup(ctx, epic.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature rollup for epic %s: %w", key, err)
	}

	taskRollup, err := s.repo.GetTaskStatusRollup(ctx, epic.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task rollup for epic %s: %w", key, err)
	}

	totalFeatures := 0
	for _, count := range featureRollup {
		totalFeatures += count
	}

	return &EpicProgressInfo{
		EpicKey:       key,
		ProgressPct:   math.Round(progressPct*100) / 100,
		TotalFeatures: totalFeatures,
		TaskRollup:    taskRollup,
	}, nil
}

// GetFeatureRollup aggregates feature statuses for an epic.
func (s *EpicService) GetFeatureRollup(ctx context.Context, key string) (*FeatureRollup, error) {
	epic, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get epic %s: %w", key, err)
	}
	if epic == nil {
		return nil, fmt.Errorf("epic not found: %s", key)
	}

	statusCounts, err := s.repo.GetFeatureStatusRollup(ctx, epic.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature rollup for epic %s: %w", key, err)
	}

	totalFeatures := 0
	for _, count := range statusCounts {
		totalFeatures += count
	}

	return &FeatureRollup{
		EpicKey:       key,
		TotalFeatures: totalFeatures,
		StatusCounts:  statusCounts,
	}, nil
}

// GetTaskStatusRollup aggregates task statuses across all features in an epic.
func (s *EpicService) GetTaskStatusRollup(ctx context.Context, key string) (map[string]int, error) {
	epic, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get epic %s: %w", key, err)
	}
	if epic == nil {
		return nil, fmt.Errorf("epic not found: %s", key)
	}

	rollup, err := s.repo.GetTaskStatusRollup(ctx, epic.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task status rollup for epic %s: %w", key, err)
	}
	return rollup, nil
}

// GetImpediments returns blocked tasks that impede epic progress.
// Degrades gracefully if taskRepo is nil (returns empty slice).
func (s *EpicService) GetImpediments(ctx context.Context, key string) ([]*Impediment, error) {
	if s.taskRepo == nil {
		return []*Impediment{}, nil
	}

	blockedTasks, err := s.taskRepo.ListBlockedTasksByEpic(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get impediments for epic %s: %w", key, err)
	}

	impediments := make([]*Impediment, 0, len(blockedTasks))
	now := time.Now()
	for _, task := range blockedTasks {
		ageDays := 0
		if task.BlockedAt.Valid {
			ageDays = int(now.Sub(task.BlockedAt.Time).Hours() / 24)
		} else {
			ageDays = int(now.Sub(task.UpdatedAt).Hours() / 24)
		}
		impediments = append(impediments, &Impediment{
			TaskKey:  task.Key,
			Title:    task.Title,
			Status:   string(task.Status),
			Priority: task.Priority,
			AgeDays:  ageDays,
		})
	}

	return impediments, nil
}

// GetHealth analyzes the health of an epic based on blocked tasks and feature status.
// Degrades gracefully if taskRepo is nil (returns healthy).
func (s *EpicService) GetHealth(ctx context.Context, key string) (*EpicHealthInfo, error) {
	epic, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get epic %s: %w", key, err)
	}
	if epic == nil {
		return nil, fmt.Errorf("epic not found: %s", key)
	}

	health := &EpicHealthInfo{
		EpicKey: key,
		Status:  "healthy",
	}

	// Check for blocked tasks
	impediments, err := s.GetImpediments(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze health for epic %s: %w", key, err)
	}

	if len(impediments) >= 2 {
		health.Status = "critical"
		health.Reasons = append(health.Reasons, fmt.Sprintf("%d blocked tasks", len(impediments)))
	} else if len(impediments) == 1 {
		health.Status = "warning"
		health.Reasons = append(health.Reasons, "1 blocked task")
	}

	// Check for high-priority blocked tasks
	for _, imp := range impediments {
		if imp.Priority <= 3 && health.Status != "critical" {
			health.Status = "critical"
			health.Reasons = append(health.Reasons, fmt.Sprintf("high-priority task %s is blocked", imp.TaskKey))
		}
	}

	return health, nil
}
