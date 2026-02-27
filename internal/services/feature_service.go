package services

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/progress"
	"github.com/jwwelbor/shark-task-manager/internal/utils"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// FeatureRepository defines the repository interface needed by FeatureService.
// This interface is satisfied by *repository.FeatureRepository.
type FeatureRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Feature, error)
	GetByID(ctx context.Context, id int64) (*models.Feature, error)
	Create(ctx context.Context, feature *models.Feature) error
	Update(ctx context.Context, feature *models.Feature) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]*models.Feature, error)
	ListByEpic(ctx context.Context, epicID int64) ([]*models.Feature, error)
	ListByEpicAndStatus(ctx context.Context, epicID int64, status models.FeatureStatus) ([]*models.Feature, error)
	GetByFilePath(ctx context.Context, filePath string) (*models.Feature, error)
	UpdateFilePath(ctx context.Context, featureKey string, newFilePath *string) error
	UpdateKey(ctx context.Context, oldKey string, newKey string) error
	GetTaskStatusBreakdown(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error)
	GetTaskCount(ctx context.Context, featureID int64) (int, error)
	SetStatusOverride(ctx context.Context, featureID int64, override bool) error
	UpdateStatusIfNotOverridden(ctx context.Context, featureID int64, newStatus models.FeatureStatus) (bool, error)
	CascadeStatusToTasks(ctx context.Context, featureID int64, targetTaskStatus models.TaskStatus) error
}

// FeatureEpicLookup defines the minimal epic repository interface needed by FeatureService
// when creating features (to look up the parent epic by key).
type FeatureEpicLookup interface {
	GetByKey(ctx context.Context, key string) (*models.Epic, error)
	GetByFilePath(ctx context.Context, filePath string) (*models.Epic, error)
	UpdateFilePath(ctx context.Context, epicKey string, newFilePath *string) error
	List(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error)
}

// FeatureNoteRepository defines the note repo interface needed by FeatureService
// for creating rejection notes on backward transitions.
type FeatureNoteRepository interface {
	CreateRejectionNote(ctx context.Context, entityType string, entityID int64,
		historyID int64, fromStatus, toStatus, reason, rejectedBy, documentPath string) error
}

// FeatureTaskCounter defines the task counting interface needed by FeatureService
// to count child tasks for backward transition warnings and feature completion.
type FeatureTaskCounter interface {
	ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error)
	UpdateStatusForced(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string, rejectionReason *string, documentPath *string, force bool) error
	GetStatusBreakdownMapBatch(ctx context.Context, featureIDs []int64) (map[int64]map[models.TaskStatus]int, error)
	GetTaskCountsForFeatures(ctx context.Context, featureIDs []int64) (map[int64]int, error)
}

// DocumentRepository defines the interface for accessing documents linked to entities.
// This is satisfied by implementations from the config or repository packages.
type DocumentRepository = config.DocumentRepository

// FeatureRelationshipRepository defines the interface for accessing feature relationships.
// This is satisfied by implementations from the config or repository packages.
type FeatureRelationshipRepository = config.FeatureRelationshipRepository

// FeatureWritableDocumentRepository defines the writable interface for document linking on features.
// This interface is satisfied by *repository.DocumentRepository.
// The existing DocumentRepository (config.DocumentRepository) only exposes read-only List methods;
// this interface adds the write operations needed by LinkDocument and UnlinkDocument.
type FeatureWritableDocumentRepository interface {
	CreateOrGet(ctx context.Context, title, filePath string) (*models.Document, error)
	GetByTitle(ctx context.Context, title string) (*models.Document, error)
	LinkToFeature(ctx context.Context, featureID, documentID int64) error
	UnlinkFromFeature(ctx context.Context, featureID, documentID int64) error
}

// FeatureService provides business logic for feature operations.
type FeatureService struct {
	repo            FeatureRepository
	workflowSvc     *workflow.Service
	noteRepo        FeatureNoteRepository
	taskRepo        FeatureTaskCounter
	docRepo         DocumentRepository
	relRepo         FeatureRelationshipRepository
	epicLookupRepo  FeatureEpicLookup
	writableDocRepo FeatureWritableDocumentRepository
}

// NewFeatureService creates a new FeatureService.
// The workflow service is automatically scoped to the feature level.
// noteRepo, taskRepo, epicLookupRepo can be nil for graceful degradation.
//
// Panics:
//   - If repo is nil (required dependency)
//   - If workflowSvc is nil (required dependency)
func NewFeatureService(repo FeatureRepository, workflowSvc *workflow.Service, noteRepo FeatureNoteRepository, taskRepo FeatureTaskCounter, epicLookupRepo FeatureEpicLookup) *FeatureService {
	requireNonNil(repo, "FeatureService requires a non-nil FeatureRepository")
	requireNonNil(workflowSvc, "FeatureService requires a non-nil workflow.Service")
	return &FeatureService{
		repo:           repo,
		workflowSvc:    workflowSvc.ForLevel(workflow.LevelFeature),
		noteRepo:       noteRepo,
		taskRepo:       taskRepo,
		docRepo:        nil,
		relRepo:        nil,
		epicLookupRepo: epicLookupRepo,
	}
}

// SetDocRepo sets the read-only document repository on the service.
// This enables GetRelatedDocuments and document listing operations on features.
func (s *FeatureService) SetDocRepo(docRepo DocumentRepository) {
	s.docRepo = docRepo
}

// SetRelRepo sets the feature relationship repository on the service.
// This enables related feature lookups.
func (s *FeatureService) SetRelRepo(relRepo FeatureRelationshipRepository) {
	s.relRepo = relRepo
}

// SetWritableDocRepo sets the writable document repository on the service.
// This enables LinkDocument and UnlinkDocument operations on features.
// The *repository.DocumentRepository type satisfies the FeatureWritableDocumentRepository interface.
func (s *FeatureService) SetWritableDocRepo(docRepo FeatureWritableDocumentRepository) {
	s.writableDocRepo = docRepo
}

// LinkDocument creates or retrieves a document by its title and file path, then links it to a feature.
// If the document already exists, it is reused (no duplicate created).
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - featureKey: the feature key (e.g., "E07-F01")
//   - docTitle: the title of the document to link
//   - docPath: the file path of the document
//
// Returns:
//   - error: FeatureNotFoundError if feature not found, or repository errors
//
// Errors:
//   - writable document repository not configured
//   - feature not found
//   - repository operation failed
func (s *FeatureService) LinkDocument(ctx context.Context, featureKey, docTitle, docPath string) error {
	if s.writableDocRepo == nil {
		return fmt.Errorf("writable document repository not configured")
	}

	feature, err := s.repo.GetByKey(ctx, featureKey)
	if err != nil {
		return fmt.Errorf("feature not found: %w", err)
	}

	_, err = linkDocumentToEntity(ctx, s.writableDocRepo, s.writableDocRepo.LinkToFeature,
		feature.ID, docTitle, docPath, "feature", featureKey)
	return err
}

// UnlinkDocument removes a document link from a feature by document title.
// If the document does not exist, it returns an error.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - featureKey: the feature key (e.g., "E07-F01")
//   - docTitle: the title of the document to unlink
//
// Returns:
//   - error: FeatureNotFoundError if feature not found, or repository errors
//
// Errors:
//   - writable document repository not configured
//   - feature not found
//   - document not found
//   - repository operation failed
func (s *FeatureService) UnlinkDocument(ctx context.Context, featureKey, docTitle string) error {
	if s.writableDocRepo == nil {
		return fmt.Errorf("writable document repository not configured")
	}

	feature, err := s.repo.GetByKey(ctx, featureKey)
	if err != nil {
		return fmt.Errorf("feature not found: %w", err)
	}

	return unlinkDocumentFromEntity(ctx, s.writableDocRepo, s.writableDocRepo.UnlinkFromFeature,
		feature.ID, docTitle, "feature", featureKey)
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

// GetFeatureByID retrieves a feature by its database ID.
// This is used when iterating over features that were returned with IDs but no keys,
// such as after RecalculateAndSetProgress to get the updated feature record.
func (s *FeatureService) GetFeatureByID(ctx context.Context, id int64) (*models.Feature, error) {
	feature, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature by ID %d: %w", id, err)
	}
	if feature == nil {
		return nil, fmt.Errorf("feature not found for ID %d", id)
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

// ListFeaturesByEpicKey retrieves features for a specific epic, with optional status filtering.
// Unlike ListFeatures, this supports epic-scoped queries by resolving the epic key via epicLookupRepo.
// Returns an error if epicLookupRepo is nil or the epic does not exist.
func (s *FeatureService) ListFeaturesByEpicKey(ctx context.Context, epicKey string, statusFilter string) ([]*models.Feature, error) {
	if s.epicLookupRepo == nil {
		return nil, fmt.Errorf("epic lookup repository not available")
	}
	epic, err := s.epicLookupRepo.GetByKey(ctx, epicKey)
	if err != nil {
		return nil, fmt.Errorf("epic %s does not exist: %w", epicKey, err)
	}
	if epic == nil {
		return nil, fmt.Errorf("epic %s not found", epicKey)
	}
	if statusFilter != "" {
		return s.repo.ListByEpicAndStatus(ctx, epic.ID, models.FeatureStatus(statusFilter))
	}
	return s.repo.ListByEpic(ctx, epic.ID)
}

// GetTaskCount returns the number of tasks for a feature.
func (s *FeatureService) GetTaskCount(ctx context.Context, featureID int64) (int, error) {
	count, err := s.repo.GetTaskCount(ctx, featureID)
	if err != nil {
		return 0, fmt.Errorf("failed to get task count for feature %d: %w", featureID, err)
	}
	return count, nil
}

// GetStatusBreakdownBatch fetches task status breakdowns for multiple features in one query.
// Returns a map of featureID -> (taskStatus -> count).
// Returns nil, nil if taskRepo is not available (graceful degradation).
func (s *FeatureService) GetStatusBreakdownBatch(ctx context.Context, featureIDs []int64) (map[int64]map[models.TaskStatus]int, error) {
	if s.taskRepo == nil {
		return nil, nil
	}
	return s.taskRepo.GetStatusBreakdownMapBatch(ctx, featureIDs)
}

// UpdateFeatureKey renames a feature key. Returns an error if the new key is already in use.
func (s *FeatureService) UpdateFeatureKey(ctx context.Context, oldKey string, newKey string) error {
	// Check if new key already exists
	existing, err := s.repo.GetByKey(ctx, newKey)
	if err == nil && existing != nil {
		return fmt.Errorf("feature with key '%s' already exists", newKey)
	}
	if err := s.repo.UpdateKey(ctx, oldKey, newKey); err != nil {
		return fmt.Errorf("failed to update feature key from %s to %s: %w", oldKey, newKey, err)
	}
	return nil
}

// SetFeatureStatusOverride sets or clears the manual status override flag for a feature.
func (s *FeatureService) SetFeatureStatusOverride(ctx context.Context, featureKey string, override bool) error {
	feature, err := s.repo.GetByKey(ctx, featureKey)
	if err != nil {
		return fmt.Errorf("feature %s does not exist: %w", featureKey, err)
	}
	if err := s.repo.SetStatusOverride(ctx, feature.ID, override); err != nil {
		return fmt.Errorf("failed to set status override for feature %s: %w", featureKey, err)
	}
	return nil
}

// UpdateFeatureStatusIfNotOverridden updates the feature status only if the status_override flag is false.
// Returns true if the status was updated, false if skipped due to override.
func (s *FeatureService) UpdateFeatureStatusIfNotOverridden(ctx context.Context, featureKey string, newStatus models.FeatureStatus) (bool, error) {
	feature, err := s.repo.GetByKey(ctx, featureKey)
	if err != nil {
		return false, fmt.Errorf("feature %s does not exist: %w", featureKey, err)
	}
	return s.repo.UpdateStatusIfNotOverridden(ctx, feature.ID, newStatus)
}

// CascadeFeatureStatusToTasks sets all tasks in a feature to the given status.
func (s *FeatureService) CascadeFeatureStatusToTasks(ctx context.Context, featureKey string, targetTaskStatus models.TaskStatus) error {
	feature, err := s.repo.GetByKey(ctx, featureKey)
	if err != nil {
		return fmt.Errorf("feature %s does not exist: %w", featureKey, err)
	}
	if err := s.repo.CascadeStatusToTasks(ctx, feature.ID, targetTaskStatus); err != nil {
		return fmt.Errorf("failed to cascade status to tasks for feature %s: %w", featureKey, err)
	}
	return nil
}

// ResolveFeaturePath resolves the file path for a feature relative to projectRoot.
// If the feature has an explicit FilePath stored, uses that.
// Otherwise constructs the default path from epic key and feature key.
// Returns an empty string if the path cannot be determined.
func (s *FeatureService) ResolveFeaturePath(ctx context.Context, featureKey string, projectRoot string) string {
	feature, err := s.repo.GetByKey(ctx, featureKey)
	if err != nil || feature == nil {
		return ""
	}
	if feature.FilePath != nil && *feature.FilePath != "" {
		return *feature.FilePath
	}
	// Construct default relative path using feature key
	// Feature key format: E07-F01, path: docs/plan/E07/E07-F01/feature.md
	parts := strings.SplitN(featureKey, "-", 2)
	if len(parts) == 2 {
		return fmt.Sprintf("docs/plan/%s/%s/feature.md", parts[0], featureKey)
	}
	return fmt.Sprintf("docs/plan/%s/feature.md", featureKey)
}

// UpdateFeatureFilePath updates the stored file path for a feature.
func (s *FeatureService) UpdateFeatureFilePath(ctx context.Context, featureKey string, filePath *string) error {
	if err := s.repo.UpdateFilePath(ctx, featureKey, filePath); err != nil {
		return fmt.Errorf("failed to update file path for feature %s: %w", featureKey, err)
	}
	return nil
}

// ListTasksForFeature returns all tasks belonging to a feature by feature ID.
// Returns an empty slice if the task repository is not available.
func (s *FeatureService) ListTasksForFeature(ctx context.Context, featureID int64) ([]*models.Task, error) {
	if s.taskRepo == nil {
		return []*models.Task{}, nil
	}
	tasks, err := s.taskRepo.ListByFeature(ctx, featureID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks for feature ID %d: %w", featureID, err)
	}
	return tasks, nil
}

// ListRelatedDocuments returns all documents linked to a feature.
// Returns an empty slice if the document repository is not available.
func (s *FeatureService) ListRelatedDocuments(ctx context.Context, featureID int64) ([]*models.Document, error) {
	if s.docRepo == nil {
		return []*models.Document{}, nil
	}
	docs, err := s.docRepo.ListForFeature(ctx, featureID)
	if err != nil {
		return nil, fmt.Errorf("failed to list related documents for feature ID %d: %w", featureID, err)
	}
	return docs, nil
}

// ListRelatedDocumentsByKey returns all documents linked to a feature identified by key.
// Returns an empty slice if the document repository is not available.
func (s *FeatureService) ListRelatedDocumentsByKey(ctx context.Context, featureKey string) ([]*models.Document, error) {
	feature, err := s.repo.GetByKey(ctx, featureKey)
	if err != nil {
		return nil, fmt.Errorf("feature not found: %w", err)
	}
	return s.ListRelatedDocuments(ctx, feature.ID)
}

// CompleteFeature marks all tasks in a feature as completed and sets the feature status to completed.
// If force is false and there are incomplete tasks, returns an error with details.
// If force is true, all incomplete tasks are force-completed before marking the feature done.
//
// Returns a FeatureCompleteResult with details about what was completed.
func (s *FeatureService) CompleteFeature(ctx context.Context, featureKey string, force bool) (*FeatureCompleteResult, error) {
	feature, err := s.repo.GetByKey(ctx, featureKey)
	if err != nil {
		return nil, fmt.Errorf("feature %s does not exist: %w", featureKey, err)
	}

	// Get all tasks for this feature
	var tasks []*models.Task
	if s.taskRepo != nil {
		tasks, err = s.taskRepo.ListByFeature(ctx, feature.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to list tasks for feature %s: %w", featureKey, err)
		}
	}

	// No tasks: just mark feature completed
	if len(tasks) == 0 {
		feature.Status = models.FeatureStatusCompleted
		if err := s.repo.Update(ctx, feature); err != nil {
			return nil, fmt.Errorf("failed to complete feature %s: %w", featureKey, err)
		}
		return &FeatureCompleteResult{
			FeatureKey:      featureKey,
			TotalCount:      0,
			CompletedCount:  0,
			AffectedTasks:   []string{},
			StatusBreakdown: map[string]int{},
		}, nil
	}

	// Build status breakdown from tasks
	statusBreakdown, err := s.repo.GetTaskStatusBreakdown(ctx, feature.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task status breakdown for feature %s: %w", featureKey, err)
	}
	breakdownStr := make(map[string]int, len(statusBreakdown))
	for k, v := range statusBreakdown {
		breakdownStr[string(k)] = v
	}

	// Find incomplete tasks
	var incompleteTasks []*models.Task
	for _, task := range tasks {
		if task.Status != models.TaskStatus("completed") && task.Status != models.TaskStatus("ready_for_review") {
			incompleteTasks = append(incompleteTasks, task)
		}
	}

	if len(incompleteTasks) > 0 && !force {
		// Return result indicating force is required, without completing
		incompleteKeys := make([]string, len(incompleteTasks))
		for i, t := range incompleteTasks {
			incompleteKeys[i] = t.Key
		}
		return &FeatureCompleteResult{
			FeatureKey:      featureKey,
			TotalCount:      len(tasks),
			CompletedCount:  breakdownStr["completed"] + breakdownStr["ready_for_review"],
			AffectedTasks:   incompleteKeys,
			StatusBreakdown: breakdownStr,
			RequiresForce:   true,
		}, nil
	}

	// Force-complete all incomplete tasks
	agent := "shark-cli"
	numCompleted := 0
	affectedKeys := make([]string, 0)
	for _, task := range tasks {
		if task.Status == models.TaskStatus("completed") {
			continue
		}
		if err := s.taskRepo.UpdateStatusForced(ctx, task.ID, models.TaskStatus("completed"), &agent, nil, nil, nil, true); err != nil {
			return nil, fmt.Errorf("failed to complete task %s: %w", task.Key, err)
		}
		numCompleted++
		affectedKeys = append(affectedKeys, task.Key)
	}

	// Recalculate progress (which may auto-complete the feature)
	if err := s.RecalculateAndSetProgress(ctx, feature.ID); err != nil {
		return nil, fmt.Errorf("failed to update feature progress: %w", err)
	}

	return &FeatureCompleteResult{
		FeatureKey:      featureKey,
		TotalCount:      len(tasks),
		CompletedCount:  len(tasks),
		AffectedTasks:   affectedKeys,
		StatusBreakdown: breakdownStr,
		RequiresForce:   false,
	}, nil
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

// GetTaskCounts returns the total task count for each of the given feature IDs in a
// single batch query. This is used by the feature list command to avoid N+1 queries.
//
// Returns a map from featureID to count. Feature IDs with no tasks will have a count
// of zero (not included in the map; callers should treat missing keys as zero).
// Degrades gracefully if taskRepo is nil (returns empty map).
func (s *FeatureService) GetTaskCounts(ctx context.Context, featureIDs []int64) (map[int64]int, error) {
	if s.taskRepo == nil || len(featureIDs) == 0 {
		return map[int64]int{}, nil
	}
	return s.taskRepo.GetTaskCountsForFeatures(ctx, featureIDs)
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

// GetTaskStatusBreakdownByFeatureID returns the enriched task status breakdown for a feature
// using its database ID directly, avoiding a redundant key-based lookup when the caller
// already has the feature loaded.
func (s *FeatureService) GetTaskStatusBreakdownByFeatureID(ctx context.Context, featureID int64) ([]workflow.StatusCount, error) {
	rawBreakdown, err := s.repo.GetTaskStatusBreakdown(ctx, featureID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task status breakdown for feature ID %d: %w", featureID, err)
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

// ResolveFeaturePathFromFeature resolves the file path for a feature using an already-loaded
// feature model, avoiding a redundant key-based lookup when the caller already has the feature.
func (s *FeatureService) ResolveFeaturePathFromFeature(feature *models.Feature, projectRoot string) string {
	if feature == nil {
		return ""
	}
	if feature.FilePath != nil && *feature.FilePath != "" {
		return *feature.FilePath
	}
	// Construct default relative path using feature key
	// Feature key format: E07-F01, path: docs/plan/E07/E07-F01/feature.md
	parts := strings.SplitN(feature.Key, "-", 2)
	if len(parts) == 2 {
		return fmt.Sprintf("docs/plan/%s/%s/feature.md", parts[0], feature.Key)
	}
	return fmt.Sprintf("docs/plan/%s/feature.md", feature.Key)
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

// CreateFeature creates a new feature under the specified epic.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - input: feature creation parameters
//
// Returns the created feature.
func (s *FeatureService) CreateFeature(ctx context.Context, input CreateFeatureInput) (*models.Feature, error) {
	if strings.TrimSpace(input.Title) == "" {
		return nil, fmt.Errorf("feature title cannot be empty")
	}
	if strings.TrimSpace(input.EpicKey) == "" {
		return nil, fmt.Errorf("epic key cannot be empty")
	}

	epicKey := strings.ToUpper(strings.TrimSpace(input.EpicKey))

	// Validate epic exists (requires epicLookupRepo)
	if s.epicLookupRepo == nil {
		return nil, fmt.Errorf("epic lookup not available: epicLookupRepo is nil")
	}

	epic, err := s.epicLookupRepo.GetByKey(ctx, epicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get epic %s: %w", epicKey, err)
	}
	if epic == nil {
		return nil, fmt.Errorf("epic not found: %s", epicKey)
	}

	// Generate next feature key
	featureKey, err := s.nextFeatureKey(ctx, epic.ID, epicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate feature key: %w", err)
	}

	// Resolve status
	statusStr := strings.TrimSpace(input.Status)
	if statusStr == "" {
		statusStr = "draft"
	}

	// Resolve file path
	var filePath *string
	if input.FilePath != nil {
		if err := s.resolveFeatureFilePath(ctx, input.FilePath, nil, input.Force); err != nil {
			return nil, err
		}
		filePath = input.FilePath
	} else {
		// Default path: docs/plan/{epicKey}-{epicSlug}/{featureKey}-{featureSlug}/feature.md
		featureSlug := utils.GenerateSlug(input.Title)
		var defaultPath string
		if epic.Slug != nil && *epic.Slug != "" {
			defaultPath = fmt.Sprintf("docs/plan/%s-%s/%s-%s/feature.md", epicKey, *epic.Slug, featureKey, featureSlug)
		} else {
			epicSlug := utils.GenerateSlug(epic.Title)
			defaultPath = fmt.Sprintf("docs/plan/%s-%s/%s-%s/feature.md", epicKey, epicSlug, featureKey, featureSlug)
		}
		filePath = &defaultPath
	}

	feature := &models.Feature{
		EpicID:         epic.ID,
		Key:            featureKey,
		Title:          strings.TrimSpace(input.Title),
		Description:    input.Description,
		Status:         models.FeatureStatus(statusStr),
		ExecutionOrder: input.ExecutionOrder,
		FilePath:       filePath,
	}

	if err := feature.Validate(); err != nil {
		return nil, fmt.Errorf("feature validation failed: %w", err)
	}

	if err := s.repo.Create(ctx, feature); err != nil {
		return nil, fmt.Errorf("failed to create feature %s: %w", featureKey, err)
	}

	return feature, nil
}

// UpdateFeature updates fields on an existing feature.
// Only non-nil fields in updates are applied.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: feature key (e.g., "E07-F01")
//   - updates: fields to update
//
// Returns the updated feature.
func (s *FeatureService) UpdateFeature(ctx context.Context, key string, updates FeatureUpdates) (*models.Feature, error) {
	feature, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature %s: %w", key, err)
	}
	if feature == nil {
		return nil, fmt.Errorf("feature not found: %s", key)
	}

	// Apply non-nil updates
	if updates.Title != nil {
		if strings.TrimSpace(*updates.Title) == "" {
			return nil, fmt.Errorf("feature title cannot be empty")
		}
		feature.Title = strings.TrimSpace(*updates.Title)
	}
	if updates.Description != nil {
		feature.Description = updates.Description
	}
	if updates.Status != nil {
		feature.Status = *updates.Status
	}
	if updates.ExecutionOrder != nil {
		feature.ExecutionOrder = updates.ExecutionOrder
	}
	if updates.FilePath != nil {
		feature.FilePath = updates.FilePath
	}

	if err := feature.Validate(); err != nil {
		return nil, fmt.Errorf("feature validation failed: %w", err)
	}

	if err := s.repo.Update(ctx, feature); err != nil {
		return nil, fmt.Errorf("failed to update feature %s: %w", key, err)
	}

	return feature, nil
}

// DeleteFeature deletes a feature by key.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: feature key (e.g., "E07-F01")
//
// Returns an error if the feature is not found or cannot be deleted.
func (s *FeatureService) DeleteFeature(ctx context.Context, key string) error {
	feature, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get feature %s: %w", key, err)
	}
	if feature == nil {
		return fmt.Errorf("feature not found: %s", key)
	}

	if err := s.repo.Delete(ctx, feature.ID); err != nil {
		return fmt.Errorf("failed to delete feature %s: %w", key, err)
	}

	return nil
}

// nextFeatureKey generates the next available feature key (E##-F##) for a given epic.
func (s *FeatureService) nextFeatureKey(ctx context.Context, epicID int64, epicKey string) (string, error) {
	features, err := s.repo.ListByEpic(ctx, epicID)
	if err != nil {
		return "", fmt.Errorf("failed to list features for epic %s: %w", epicKey, err)
	}

	maxNum := 0
	for _, f := range features {
		// Parse feature number from key like "E07-F03"
		var num int
		if _, err := fmt.Sscanf(f.Key, epicKey+"-F%d", &num); err == nil {
			if num > maxNum {
				maxNum = num
			}
		}
	}
	return fmt.Sprintf("%s-F%02d", epicKey, maxNum+1), nil
}

// resolveFeatureFilePath checks for file path collisions and handles force reassignment.
// currentFeatureKey is the key of the feature being updated (nil for new features).
func (s *FeatureService) resolveFeatureFilePath(ctx context.Context, filePath *string, currentFeatureKey *string, force bool) error {
	if filePath == nil {
		return nil
	}

	fp := *filePath

	// Check if another feature claims this file
	existingFeature, err := s.repo.GetByFilePath(ctx, fp)
	if err != nil {
		return fmt.Errorf("failed to check feature file path collision: %w", err)
	}

	if existingFeature != nil {
		// If it's the same feature being updated, no collision
		if currentFeatureKey != nil && existingFeature.Key == *currentFeatureKey {
			return nil
		}
		if !force {
			return fmt.Errorf("file '%s' is already claimed by feature %s ('%s'). Use force to reassign",
				fp, existingFeature.Key, existingFeature.Title)
		}
		// Force: release the file from the existing feature
		if err := s.repo.UpdateFilePath(ctx, existingFeature.Key, nil); err != nil {
			return fmt.Errorf("failed to release file path from feature %s: %w", existingFeature.Key, err)
		}
	}

	return nil
}
