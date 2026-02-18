package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/progress"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// FeatureRepository defines the repository interface needed by FeatureService.
// This interface is satisfied by *repository.FeatureRepository.
type FeatureRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Feature, error)
	GetByID(ctx context.Context, id int64) (*models.Feature, error)
	Update(ctx context.Context, feature *models.Feature) error
	List(ctx context.Context) ([]*models.Feature, error)
	ListByEpic(ctx context.Context, epicID int64) ([]*models.Feature, error)
	GetTaskStatusBreakdown(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error)
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
//
// Panics:
//   - If repo is nil (required dependency)
//   - If workflowSvc is nil (required dependency)
func NewFeatureService(repo FeatureRepository, workflowSvc *workflow.Service, noteRepo FeatureNoteRepository, taskRepo FeatureTaskCounter) *FeatureService {
	if repo == nil {
		panic("FeatureService requires a non-nil FeatureRepository")
	}
	if workflowSvc == nil {
		panic("FeatureService requires a non-nil workflow.Service")
	}
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
//
// Panics:
//   - If repo is nil (required dependency)
//   - If workflowSvc is nil (required dependency)
func NewFeatureServiceWithRelationships(repo FeatureRepository, workflowSvc *workflow.Service, noteRepo FeatureNoteRepository, taskRepo FeatureTaskCounter, docRepo DocumentRepository, relRepo FeatureRelationshipRepository) *FeatureService {
	if repo == nil {
		panic("FeatureService requires a non-nil FeatureRepository")
	}
	if workflowSvc == nil {
		panic("FeatureService requires a non-nil workflow.Service")
	}
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

// GetFeature retrieves a feature by key.
func (s *FeatureService) GetFeature(ctx context.Context, key string) (*models.Feature, error) {
	feature, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature %s: %w", key, err)
	}
	if feature == nil {
		return nil, fmt.Errorf("feature not found: %s", key)
	}
	return feature, nil
}

// ListFeatures retrieves features with optional filtering by status.
// Note: EpicKey filtering in FeatureFilters is not supported at the service level
// because FeatureService does not have an epic repository to resolve epic keys to IDs.
// Callers needing epic-scoped feature lists should resolve the epicID externally.
func (s *FeatureService) ListFeatures(ctx context.Context, filters FeatureFilters) ([]*models.Feature, error) {
	features, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list features: %w", err)
	}

	// Apply status filter in-memory
	if filters.Status != "" {
		filtered := make([]*models.Feature, 0)
		for _, f := range features {
			if string(f.Status) == filters.Status {
				filtered = append(filtered, f)
			}
		}
		features = filtered
	}

	return features, nil
}

// GetProgress retrieves progress metrics for a feature.
// Computes both weighted progress (using workflow config progress weights)
// and completion progress (raw task completion percentage).
// All progress calculation is done in the service layer using the progress package.
func (s *FeatureService) GetProgress(ctx context.Context, key string) (*FeatureProgressInfo, error) {
	feature, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature %s: %w", key, err)
	}
	if feature == nil {
		return nil, fmt.Errorf("feature not found: %s", key)
	}

	return s.calculateProgressForFeature(ctx, key, feature.ID)
}

// calculateProgressForFeature computes progress metrics for a feature by its ID.
// This is the single source of truth for feature progress calculation.
// Uses GetTaskStatusBreakdown from the repository and progress.CalculateProgress
// with workflow config weights.
func (s *FeatureService) calculateProgressForFeature(ctx context.Context, key string, featureID int64) (*FeatureProgressInfo, error) {
	statusBreakdown, err := s.repo.GetTaskStatusBreakdown(ctx, featureID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task status breakdown for feature %s: %w", key, err)
	}

	// Convert map[models.TaskStatus]int to map[string]int for progress package
	statusCounts := make(map[string]int, len(statusBreakdown))
	for k, v := range statusBreakdown {
		statusCounts[string(k)] = v
	}

	// Calculate progress using the progress package with task-level workflow config.
	// We must use the task-level workflow (not feature-level) because statusCounts
	// contains task statuses, and task status weights are defined in the task workflow.
	taskWorkflowSvc := s.workflowSvc.ForLevel(workflow.LevelTask)
	wf := taskWorkflowSvc.GetWorkflow()
	progressInfo := progress.CalculateProgress(statusCounts, wf)

	// Count completed tasks using terminal status check (task-level)
	totalTasks := 0
	completedTasks := 0
	for status, count := range statusBreakdown {
		totalTasks += count
		if taskWorkflowSvc.IsTerminalStatus(string(status)) {
			completedTasks += count
		}
	}

	completionPct := 0.0
	if totalTasks > 0 {
		completionPct = (float64(completedTasks) / float64(totalTasks)) * 100.0
	}

	// Build ratio strings
	completionRatio := fmt.Sprintf("%d/%d", completedTasks, totalTasks)

	return &FeatureProgressInfo{
		FeatureKey:         key,
		WeightedProgress:   math.Round(progressInfo.WeightedPct*100) / 100,
		CompletionProgress: math.Round(completionPct*100) / 100,
		TotalTasks:         totalTasks,
		CompletedTasks:     completedTasks,
		WeightedRatio:      progressInfo.WeightedRatio,
		CompletionRatio:    completionRatio,
	}, nil
}

// RecalculateAndSetProgress recalculates the cached progress_pct for a feature
// and persists it. Automatically sets feature status to "completed" when weighted
// progress reaches 100% (all tasks completed).
//
// This method replaces the former FeatureRepository.UpdateProgress business logic
// that was incorrectly placed in the repository layer.
func (s *FeatureService) RecalculateAndSetProgress(ctx context.Context, featureID int64) error {
	feature, err := s.repo.GetByID(ctx, featureID)
	if err != nil {
		return fmt.Errorf("failed to get feature by ID %d: %w", featureID, err)
	}
	if feature == nil {
		return fmt.Errorf("feature not found with id %d", featureID)
	}

	progressInfo, err := s.calculateProgressForFeature(ctx, feature.Key, featureID)
	if err != nil {
		return fmt.Errorf("failed to calculate progress for feature %d: %w", featureID, err)
	}

	feature.ProgressPct = progressInfo.WeightedProgress

	// Auto-complete feature when all tasks are completed (weighted progress >= 100%)
	if progressInfo.WeightedProgress >= 100.0 {
		feature.Status = models.FeatureStatusCompleted
	}

	if err := s.repo.Update(ctx, feature); err != nil {
		return fmt.Errorf("failed to update feature progress: %w", err)
	}

	return nil
}

// RecalculateAndSetProgressByKey recalculates progress for a feature identified by key.
func (s *FeatureService) RecalculateAndSetProgressByKey(ctx context.Context, key string) error {
	feature, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get feature %s: %w", key, err)
	}
	if feature == nil {
		return fmt.Errorf("feature not found: %s", key)
	}

	return s.RecalculateAndSetProgress(ctx, feature.ID)
}

// GetHealth analyzes the health of a feature based on blocked tasks and approval age.
// Degrades gracefully if taskRepo is nil (returns healthy).
func (s *FeatureService) GetHealth(ctx context.Context, key string) (*FeatureHealthInfo, error) {
	feature, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature %s: %w", key, err)
	}
	if feature == nil {
		return nil, fmt.Errorf("feature not found: %s", key)
	}

	health := &FeatureHealthInfo{
		FeatureKey: key,
		Status:     "healthy",
	}

	if s.taskRepo == nil {
		return health, nil
	}

	tasks, err := s.taskRepo.ListByFeature(ctx, feature.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks for feature %s: %w", key, err)
	}

	// Count blocked tasks
	var blockedTasks []*models.Task
	for _, t := range tasks {
		if string(t.Status) == "blocked" {
			blockedTasks = append(blockedTasks, t)
		}
	}

	if len(blockedTasks) >= 2 {
		health.Status = "critical"
		health.Reasons = append(health.Reasons, fmt.Sprintf("%d blocked tasks", len(blockedTasks)))
	} else if len(blockedTasks) == 1 {
		health.Status = "warning"
		health.Reasons = append(health.Reasons, "1 blocked task")
	}

	// Check for high-priority blocked tasks (priority 1-3 is high)
	for _, t := range blockedTasks {
		if t.Priority <= 3 && health.Status != "critical" {
			health.Status = "critical"
			health.Reasons = append(health.Reasons, fmt.Sprintf("high-priority task %s is blocked", t.Key))
		}
	}

	// Check for old approval tasks
	now := time.Now()
	for _, t := range tasks {
		meta := s.workflowSvc.GetStatusMetadata(string(t.Status))
		if meta.Phase == "approval" || meta.Phase == "review" {
			ageDays := int(now.Sub(t.UpdatedAt).Hours() / 24)
			if ageDays > 3 && health.Status == "healthy" {
				health.Status = "warning"
				health.Reasons = append(health.Reasons, fmt.Sprintf("task %s awaiting approval for %d days", t.Key, ageDays))
			}
		}
	}

	return health, nil
}

// GetWorkBreakdown categorizes remaining work by responsibility using workflow config.
func (s *FeatureService) GetWorkBreakdown(ctx context.Context, key string) (*WorkBreakdown, error) {
	feature, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature %s: %w", key, err)
	}
	if feature == nil {
		return nil, fmt.Errorf("feature not found: %s", key)
	}

	statusBreakdown, err := s.repo.GetTaskStatusBreakdown(ctx, feature.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task status breakdown for feature %s: %w", key, err)
	}

	wf := s.workflowSvc.GetWorkflow()

	wb := &WorkBreakdown{FeatureKey: key}

	for status, count := range statusBreakdown {
		statusStr := string(status)
		wb.TotalTasks += count

		// Determine responsibility from workflow config
		responsibility := "none"
		if wf != nil && wf.StatusMetadata != nil {
			if meta, found := wf.StatusMetadata[statusStr]; found {
				if meta.Responsibility != "" {
					responsibility = meta.Responsibility
				}
			}
		}

		// Check if terminal (completed)
		if s.workflowSvc.IsTerminalStatus(statusStr) {
			wb.CompletedTasks += count
			continue
		}

		// Check if blocked
		if statusStr == "blocked" {
			wb.BlockedWork += count
			continue
		}

		// Categorize by responsibility
		switch responsibility {
		case "agent":
			wb.AgentWork += count
		case "human", "qa_team":
			wb.HumanWork += count
		default:
			wb.NotStarted += count
		}
	}

	return wb, nil
}

// GetActionItems returns tasks requiring immediate attention for a feature.
// Groups tasks into awaiting_approval, blocked, and in_progress categories.
// Degrades gracefully if taskRepo is nil (returns empty result).
func (s *FeatureService) GetActionItems(ctx context.Context, key string) (*FeatureActionItems, error) {
	feature, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature %s: %w", key, err)
	}
	if feature == nil {
		return nil, fmt.Errorf("feature not found: %s", key)
	}

	result := &FeatureActionItems{FeatureKey: key}

	if s.taskRepo == nil {
		return result, nil
	}

	tasks, err := s.taskRepo.ListByFeature(ctx, feature.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks for feature %s: %w", key, err)
	}

	now := time.Now()
	for _, t := range tasks {
		statusStr := string(t.Status)
		meta := s.workflowSvc.GetStatusMetadata(statusStr)

		item := &ActionTaskItem{
			TaskKey:   t.Key,
			Title:     t.Title,
			Status:    statusStr,
			UpdatedAt: t.UpdatedAt,
		}

		// Categorize by status phase and characteristics
		if statusStr == "blocked" {
			ageDays := int(now.Sub(t.UpdatedAt).Hours() / 24)
			item.AgeDays = &ageDays
			result.Blocked = append(result.Blocked, item)
		} else if meta.Phase == "approval" || meta.Phase == "review" {
			ageDays := int(now.Sub(t.UpdatedAt).Hours() / 24)
			item.AgeDays = &ageDays
			result.AwaitingApproval = append(result.AwaitingApproval, item)
		} else if meta.Phase == "development" || meta.Phase == "execution" {
			result.InProgress = append(result.InProgress, item)
		}
	}

	return result, nil
}

// GetEnrichedTaskStatusBreakdown returns task status counts for a feature,
// enriched with workflow metadata (phase, color, order) from the task-level workflow.
// Returns a []workflow.StatusCount ordered by workflow phase.
func (s *FeatureService) GetEnrichedTaskStatusBreakdown(ctx context.Context, key string) ([]workflow.StatusCount, error) {
	feature, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature %s: %w", key, err)
	}
	if feature == nil {
		return nil, fmt.Errorf("feature not found: %s", key)
	}

	rawBreakdown, err := s.repo.GetTaskStatusBreakdown(ctx, feature.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task status breakdown for feature %s: %w", key, err)
	}

	// Convert to map[string]int for NewStatusBreakdown
	counts := make(map[string]int, len(rawBreakdown))
	for k, v := range rawBreakdown {
		counts[string(k)] = v
	}

	// Use task-level workflow service to enrich with phase/color/order metadata
	taskWorkflowSvc := s.workflowSvc.ForLevel(workflow.LevelTask)
	breakdown := workflow.NewStatusBreakdown(counts, taskWorkflowSvc)
	return breakdown.Counts, nil
}

// GetTaskStatusBreakdown returns the count of tasks per status for a feature.
func (s *FeatureService) GetTaskStatusBreakdown(ctx context.Context, key string) (map[string]int, error) {
	feature, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature %s: %w", key, err)
	}
	if feature == nil {
		return nil, fmt.Errorf("feature not found: %s", key)
	}

	breakdown, err := s.repo.GetTaskStatusBreakdown(ctx, feature.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task status breakdown for feature %s: %w", key, err)
	}

	// Convert map[models.TaskStatus]int to map[string]int
	result := make(map[string]int, len(breakdown))
	for k, v := range breakdown {
		result[string(k)] = v
	}
	return result, nil
}
