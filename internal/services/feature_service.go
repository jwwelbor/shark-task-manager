package services

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/utils"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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
	UpdateStatus(ctx context.Context, featureID int64, status models.FeatureStatus) error
	UpdateStatusIfNotOverridden(ctx context.Context, featureID int64, newStatus models.FeatureStatus) (bool, error)
	CascadeStatusToTasks(ctx context.Context, featureID int64, targetTaskStatus models.TaskStatus) error
	GetFeatureDisplayDataRaw(ctx context.Context, featureID int64) (*repository.FeatureDisplayDataRaw, error)
}

// FeatureEpicLookup defines the minimal epic repository interface needed by FeatureService
// when creating features (to look up the parent epic by key) and auto-reopening epics.
type FeatureEpicLookup interface {
	GetByKey(ctx context.Context, key string) (*models.Epic, error)
	GetByFilePath(ctx context.Context, filePath string) (*models.Epic, error)
	UpdateFilePath(ctx context.Context, epicKey string, newFilePath *string) error
	List(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error)
	Update(ctx context.Context, epic *models.Epic) error
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
// (FeatureWritableDocumentRepository removed -- replaced by EntityDocumentRepository + EntityDocumentLinkRepository)

// FeatureService provides business logic for feature operations.
type FeatureService struct {
	repo              FeatureRepository
	entitySvc         *EntityService
	entityRepo        EntityRepository
	taskRepo          FeatureTaskCounter
	docRepo           DocumentRepository
	relRepo           FeatureRelationshipRepository
	epicLookupRepo    FeatureEpicLookup
	docSvc            *EntityDocumentService // shared document operations; built by SetWritableDocRepo
	progressService   *FeatureProgressService
	enrichRepo        config.TemplateEnrichmentRepository
	entityHistoryRepo EntityHistoryRecorder // optional: records to entity_history table
	tracer            trace.Tracer          // optional; defaults to otel.Tracer("shark/services/feature") if nil
}

// NewFeatureService creates a new FeatureService.
// The workflow service is automatically scoped to the feature level.
// taskRepo, epicLookupRepo can be nil for graceful degradation.
// Rejection note creation is handled by EntityService (via SetNoteRepo).
//
// Panics:
//   - If repo is nil (required dependency)
//   - If entitySvc is nil (required dependency)
func NewFeatureService(repo FeatureRepository, entitySvc *EntityService, entityRepo EntityRepository, taskRepo FeatureTaskCounter, epicLookupRepo FeatureEpicLookup) *FeatureService {
	requireNonNil(repo, "FeatureService requires a non-nil FeatureRepository")
	requireNonNil(entitySvc, "FeatureService requires a non-nil EntityService")
	requireNonNil(entityRepo, "FeatureService requires a non-nil EntityRepository")
	return &FeatureService{
		repo:           repo,
		entitySvc:      entitySvc.ForLevel(workflow.LevelFeature),
		entityRepo:     entityRepo,
		taskRepo:       taskRepo,
		docRepo:        nil,
		relRepo:        nil,
		epicLookupRepo: epicLookupRepo,
	}
}

// SetTracer sets the OpenTelemetry tracer for the service.
// When nil, getTracer falls back to the OTel global tracer (noop until provider is wired).
func (s *FeatureService) SetTracer(t trace.Tracer) {
	s.tracer = t
}

// getTracer returns the configured tracer or falls back to the OTel global tracer.
func (s *FeatureService) getTracer() trace.Tracer {
	if s.tracer != nil {
		return s.tracer
	}
	return otel.Tracer("shark/services/feature")
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

// SetEnrichRepo sets the template enrichment repository on the service.
// This enables enrichment data population for template rendering.
func (s *FeatureService) SetEnrichRepo(enrichRepo config.TemplateEnrichmentRepository) {
	s.enrichRepo = enrichRepo
}

// SetEntityHistoryRepo sets the entity history recorder for audit trail recording.
// When set, auto-reopen operations will create entity_history records.
// The *repository.EntityHistoryRepository satisfies EntityHistoryRecorder directly.
func (s *FeatureService) SetEntityHistoryRepo(repo EntityHistoryRecorder) {
	s.entityHistoryRepo = repo
}

// SetWritableDocRepo sets the writable document repository on the service.
// This enables LinkDocument and UnlinkDocument operations on features.
func (s *FeatureService) SetWritableDocRepo(writableRepo EntityDocumentRepository, linkRepo EntityDocumentLinkRepository) {
	s.docSvc = NewEntityDocumentService(
		writableRepo,
		linkRepo,
		EntityLookupFnFromRepo(s.entityRepo),
	)
}

// SetProgressService sets the progress sub-service on the feature service.
// When nil, progress methods fall back to lazy initialization using the stored repos.
func (s *FeatureService) SetProgressService(progressService *FeatureProgressService) {
	s.progressService = progressService
}

// getProgressService returns the configured progress sub-service, or lazily creates
// one from the stored repository fields if none has been set.
func (s *FeatureService) getProgressService() *FeatureProgressService {
	if s.progressService != nil {
		return s.progressService
	}
	// Lazy initialization: build from stored repositories.
	// workflowSvc is passed unscoped; FeatureProgressService uses ForLevel(LevelTask) internally.
	return NewFeatureProgressService(s.repo, s.taskRepo, s.entitySvc.GetWorkflowService())
}

// LinkDocument creates or retrieves a document by its title and file path, then links it to a feature.
// Delegates to the shared EntityDocumentService.
func (s *FeatureService) LinkDocument(ctx context.Context, featureKey, docTitle, docPath string) error {
	if s.docSvc == nil {
		return fmt.Errorf("writable document repository not configured")
	}
	_, err := s.docSvc.LinkDocumentByKey(ctx, featureKey, docTitle, docPath)
	return err
}

// UnlinkDocument removes a document link from a feature by document title.
// Delegates to the shared EntityDocumentService.
func (s *FeatureService) UnlinkDocument(ctx context.Context, featureKey, docTitle string) error {
	if s.docSvc == nil {
		return fmt.Errorf("writable document repository not configured")
	}
	return s.docSvc.UnlinkDocumentByKey(ctx, featureKey, docTitle)
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
	ctx, span := s.getTracer().Start(ctx, "FeatureService.TransitionStatus",
		trace.WithAttributes(
			attribute.String("feature.key", featureKey),
			attribute.String("feature.target_status", targetStatus),
		),
	)
	defer span.End()

	// Delegate shared logic to EntityService
	result, err := s.entitySvc.TransitionStatus(
		ctx, s.entityRepo, models.EntityTypeFeature, featureKey, targetStatus, opts,
		DefaultTransitionFeatures(),
		s.makeResolveActionFn(ctx),
	)
	if err != nil {
		return nil, recordSpanError(span, err)
	}

	// Post-hook: count child tasks
	if s.taskRepo != nil {
		tasks, listErr := s.taskRepo.ListByFeature(ctx, result.EntityID)
		if listErr == nil {
			result.ChildCount = len(tasks)
		}
	}

	return result, nil
}

// GetNextStatus returns the available transitions for the current status of a feature.
func (s *FeatureService) GetNextStatus(ctx context.Context, featureKey string) (*NextStatusInfo, error) {
	return s.entitySvc.GetNextStatus(ctx, s.entityRepo, models.EntityTypeFeature, featureKey,
		s.makeResolveActionFn(ctx))
}

// ValidateStatus checks if a status is valid in the feature workflow.
func (s *FeatureService) ValidateStatus(status string) error {
	return s.entitySvc.GetWorkflowService().ValidateStatus(status)
}

// makeResolveActionFn returns a ResolveActionFn that generates Feature-specific
// placeholders including enrichment data, related documents, and related features.
func (s *FeatureService) makeResolveActionFn(ctx context.Context) ResolveActionFn {
	return func(entity models.Entity, status string) *config.PopulatedAction {
		feature, ok := entity.(*models.Feature)
		if !ok {
			return nil
		}

		// Fetch enrichment data (optional, graceful degradation)
		var enrichment *config.TemplateEnrichmentData
		if s.enrichRepo != nil {
			data, err := s.enrichRepo.GetFeatureEnrichment(ctx, feature.ID)
			if err != nil {
				slog.Warn("Failed to fetch enrichment data for feature", "feature", feature.Key, "error", err)
			} else {
				enrichment = data
			}
		}

		var placeholders map[string]string
		if s.docRepo != nil && s.relRepo != nil {
			placeholders = config.FeaturePlaceholdersWithRelated(ctx, feature, s.docRepo, s.relRepo, enrichment)
		} else {
			placeholders = config.FeaturePlaceholders(feature)
			config.ApplyEnrichmentData(enrichment, placeholders)
		}

		// Fresh transition context: suppress RESUME CONTEXT preamble in templates.
		// is_resume="true" is reserved for shark get (display_service.go).
		placeholders["is_resume"] = "false"

		return s.entitySvc.ResolveActionForStatus(status, placeholders)
	}
}

// GetFeature retrieves a feature by key.
func (s *FeatureService) GetFeature(ctx context.Context, key string) (*models.Feature, error) {
	ctx, span := s.getTracer().Start(ctx, "FeatureService.GetFeature",
		trace.WithAttributes(attribute.String("feature.key", key)),
	)
	defer span.End()

	feature, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, recordSpanError(span, fmt.Errorf("failed to get feature %s: %w", key, err))
	}
	if feature == nil {
		return nil, recordSpanError(span, fmt.Errorf("feature not found: %s", key))
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
	ctx, span := s.getTracer().Start(ctx, "FeatureService.ListFeatures")
	defer span.End()

	features, err := s.repo.List(ctx)
	if err != nil {
		return nil, recordSpanError(span, fmt.Errorf("failed to list features: %w", err))
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
	return s.getProgressService().GetStatusBreakdownBatch(ctx, featureIDs)
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
// Delegates to the shared EntityDocumentService if available, otherwise falls back to
// ListRelatedDocuments. Returns an empty slice if no document repository is configured.
func (s *FeatureService) ListRelatedDocumentsByKey(ctx context.Context, featureKey string) ([]*models.Document, error) {
	if s.docSvc != nil {
		return s.docSvc.ListDocumentsByKey(ctx, featureKey)
	}
	// Fallback for when only read-only docRepo is set (no writable doc repo)
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
	ctx, span := s.getTracer().Start(ctx, "FeatureService.CompleteFeature",
		trace.WithAttributes(attribute.String("feature.key", featureKey)),
	)
	defer span.End()

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
	ctx, span := s.getTracer().Start(ctx, "FeatureService.GetProgress",
		trace.WithAttributes(attribute.String("feature.key", key)),
	)
	defer span.End()

	info, err := s.getProgressService().GetProgress(ctx, key)
	if err != nil {
		return nil, recordSpanError(span, err)
	}
	return info, nil
}

// RecalculateAndSetProgress recalculates the cached progress_pct for a feature
// and persists it. Automatically sets feature status to "completed" when weighted
// progress reaches 100% (all tasks completed).
func (s *FeatureService) RecalculateAndSetProgress(ctx context.Context, featureID int64) error {
	return s.getProgressService().RecalculateAndSetProgress(ctx, featureID)
}

// RecalculateAndSetProgressByKey recalculates progress for a feature identified by key.
func (s *FeatureService) RecalculateAndSetProgressByKey(ctx context.Context, key string) error {
	return s.getProgressService().RecalculateAndSetProgressByKey(ctx, key)
}

// GetTaskCounts returns the total task count for each of the given feature IDs in a
// single batch query. This is used by the feature list command to avoid N+1 queries.
//
// Returns a map from featureID to count. Feature IDs with no tasks will have a count
// of zero (not included in the map; callers should treat missing keys as zero).
// Degrades gracefully if taskRepo is nil (returns empty map).
func (s *FeatureService) GetTaskCounts(ctx context.Context, featureIDs []int64) (map[int64]int, error) {
	return s.getProgressService().GetTaskCounts(ctx, featureIDs)
}

// GetHealth analyzes the health of a feature based on blocked tasks and approval age.
// Degrades gracefully if taskRepo is nil (returns healthy).
func (s *FeatureService) GetHealth(ctx context.Context, key string) (*FeatureHealthInfo, error) {
	return s.getProgressService().GetHealth(ctx, key)
}

// GetWorkBreakdown categorizes remaining work by responsibility using workflow config.
func (s *FeatureService) GetWorkBreakdown(ctx context.Context, key string) (*WorkBreakdown, error) {
	return s.getProgressService().GetWorkBreakdown(ctx, key)
}

// GetActionItems returns tasks requiring immediate attention for a feature.
// Groups tasks into awaiting_approval, blocked, and in_progress categories.
// Degrades gracefully if taskRepo is nil (returns empty result).
func (s *FeatureService) GetActionItems(ctx context.Context, key string) (*FeatureActionItems, error) {
	return s.getProgressService().GetActionItems(ctx, key)
}

// GetEnrichedTaskStatusBreakdown returns task status counts for a feature,
// enriched with workflow metadata (phase, color, order) from the task-level workflow.
// Returns a []workflow.StatusCount ordered by workflow phase.
func (s *FeatureService) GetEnrichedTaskStatusBreakdown(ctx context.Context, key string) ([]workflow.StatusCount, error) {
	return s.getProgressService().GetEnrichedTaskStatusBreakdown(ctx, key)
}

// GetTaskStatusBreakdownByFeatureID returns the enriched task status breakdown for a feature
// using its database ID directly, avoiding a redundant key-based lookup when the caller
// already has the feature loaded.
func (s *FeatureService) GetTaskStatusBreakdownByFeatureID(ctx context.Context, featureID int64) ([]workflow.StatusCount, error) {
	return s.getProgressService().GetTaskStatusBreakdownByFeatureID(ctx, featureID)
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
	return s.getProgressService().GetTaskStatusBreakdown(ctx, key)
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
		// Default path: {epicDir}/{featureKey}-{featureSlug}/feature.md
		// Use the epic's existing file_path to determine the parent directory,
		// rather than generating a new folder name from the slug/title.
		featureSlug := utils.GenerateSlug(input.Title)
		var epicDir string
		if epic.FilePath != nil && *epic.FilePath != "" {
			epicDir = filepath.Dir(*epic.FilePath)
		} else if epic.Slug != nil && *epic.Slug != "" {
			epicDir = fmt.Sprintf("docs/plan/%s-%s", epicKey, *epic.Slug)
		} else {
			epicSlug := utils.GenerateSlug(epic.Title)
			epicDir = fmt.Sprintf("docs/plan/%s-%s", epicKey, epicSlug)
		}
		defaultPath := fmt.Sprintf("%s/%s-%s/feature.md", epicDir, featureKey, featureSlug)
		filePath = &defaultPath
	}

	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: featureKey,
		Title:       strings.TrimSpace(input.Title),
		Description: input.Description,

		FilePath: filePath}, EpicID: epic.ID,

		Status:         models.FeatureStatus(statusStr),
		ExecutionOrder: input.ExecutionOrder,
	}

	if err := feature.Validate(); err != nil {
		return nil, fmt.Errorf("feature validation failed: %w", err)
	}

	if err := s.repo.Create(ctx, feature); err != nil {
		return nil, fmt.Errorf("failed to create feature %s: %w", featureKey, err)
	}

	s.maybeReopenParentEpic(ctx, epic, feature.Key)
	return feature, nil
}

// maybeReopenParentEpic checks if the parent epic is in a terminal status
// and reopens it to the first aggregation status. Best-effort: logs a warning
// on failure, never fails the caller.
//
// Parameters:
//   - ctx: context for cancellation
//   - epic: the parent epic model (already retrieved during feature creation)
//   - featureKey: key of the newly created feature (for audit logging)
func (s *FeatureService) maybeReopenParentEpic(ctx context.Context, epic *models.Epic, featureKey string) {
	if s.epicLookupRepo == nil {
		return
	}

	epicWf := s.entitySvc.GetWorkflowService().ForLevel(workflow.LevelEpic)
	if !epicWf.IsTerminalStatus(string(epic.Status)) {
		return
	}

	aggStatuses := epicWf.GetAggregationStatuses()
	oldStatus := string(epic.Status)
	epic.Status = models.EpicStatus(aggStatuses[0])

	if err := s.epicLookupRepo.Update(ctx, epic); err != nil {
		slog.Warn("auto-reopen of epic failed", "epic", epic.Key, "error", err)
		return
	}

	// Record history for the auto-reopen
	notes := fmt.Sprintf("auto-reopened: new feature %s created under terminal epic", featureKey)
	recordEntityHistory(ctx, s.entityHistoryRepo, models.EntityTypeEpic, epic.ID,
		oldStatus, string(epic.Status), false, EntityHistoryOpts{
			Agent:  "system",
			Reason: notes,
		})
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
	if err := feature.Validate(); err != nil {
		return nil, fmt.Errorf("feature validation failed: %w", err)
	}

	// file_path is included in the Update query — single atomic operation
	if updates.FilePath != nil {
		feature.FilePath = updates.FilePath
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

// FeatureDisplayData holds all data needed to display a feature detail view.
// Populated by GetFeatureDisplayData from a single SQL query.
type FeatureDisplayData struct {
	Tasks           []*models.Task
	StatusBreakdown map[string]int
	RelatedDocs     []*models.Document
	Notes           []*models.EntityNote
	ContextData     *models.ContextData
	DirPath         string
	Filename        string
}

// featureTaskJSON is the JSON helper type for tasks in the feature_display_data view.
type featureTaskJSON struct {
	ID             int64   `json:"id"`
	Key            string  `json:"key"`
	Title          string  `json:"title"`
	Slug           *string `json:"slug"`
	Description    *string `json:"description"`
	Status         string  `json:"status"`
	AgentType      *string `json:"agent_type"`
	Priority       *int    `json:"priority"`
	ExecutionOrder *int    `json:"execution_order"`
	FilePath       *string `json:"file_path"`
	ContextData    *string `json:"context_data"`
	BlockedReason  *string `json:"blocked_reason"`
	BlockedAt      *string `json:"blocked_at"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

// featureTaskBreakdownJSON is the JSON helper for task status breakdown rows.
type featureTaskBreakdownJSON struct {
	Status string `json:"status"`
	Count  int    `json:"cnt"`
}

// GetFeatureDisplayData fetches all data needed to display a feature in a single SQL query
// via the feature_display_data view. This reduces round-trips from ~5 to 1, critical for
// Turso cloud databases where each round-trip costs ~150-200ms.
func (s *FeatureService) GetFeatureDisplayData(ctx context.Context, feature *models.Feature, projectRoot string) (*FeatureDisplayData, error) {
	result := &FeatureDisplayData{
		Tasks:           make([]*models.Task, 0),
		StatusBreakdown: make(map[string]int),
		RelatedDocs:     make([]*models.Document, 0),
		Notes:           make([]*models.EntityNote, 0),
	}

	// Resolve feature path without re-fetching
	relPath := s.ResolveFeaturePathFromFeature(feature, projectRoot)
	if relPath != "" {
		result.DirPath = filepath.Dir(relPath) + "/"
		result.Filename = filepath.Base(relPath)
	}

	// Single query via the feature_display_data view
	raw, err := s.repo.GetFeatureDisplayDataRaw(ctx, feature.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get display data for feature %s: %w", feature.Key, err)
	}

	// Unmarshal tasks
	tasksRaw, err := unmarshalJSONArray[featureTaskJSON](raw.TasksJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal tasks for feature %s: %w", feature.Key, err)
	}
	for _, tj := range tasksRaw {
		task := &models.Task{BaseEntity: models.BaseEntity{ID: tj.ID,
			Key:   tj.Key,
			Title: tj.Title,

			Slug:        tj.Slug,
			Description: tj.Description,

			FilePath: tj.FilePath,

			ContextData: tj.ContextData}, Status: models.TaskStatus(tj.Status),
			FeatureID: feature.ID,

			AgentType: tj.AgentType,

			BlockedReason:  tj.BlockedReason,
			ExecutionOrder: tj.ExecutionOrder,
		}
		if tj.Priority != nil {
			task.Priority = *tj.Priority
		}
		result.Tasks = append(result.Tasks, task)
	}

	// Unmarshal task status breakdown
	breakdownRaw, err := unmarshalJSONArray[featureTaskBreakdownJSON](raw.TaskBreakdownJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal task breakdown for feature %s: %w", feature.Key, err)
	}
	for _, b := range breakdownRaw {
		result.StatusBreakdown[b.Status] = b.Count
	}

	// Unmarshal related documents (reuse documentJSON from epic_service.go — same package)
	docsRaw, err := unmarshalJSONArray[documentJSON](raw.DocumentsJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal documents for feature %s: %w", feature.Key, err)
	}
	for _, d := range docsRaw {
		result.RelatedDocs = append(result.RelatedDocs, &models.Document{
			ID:       d.ID,
			Title:    d.Title,
			FilePath: d.FilePath,
		})
	}

	// Unmarshal notes (reuse noteJSON from epic_service.go — same package)
	notesRaw, err := unmarshalJSONArray[noteJSON](raw.NotesJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal notes for feature %s: %w", feature.Key, err)
	}
	for _, n := range notesRaw {
		result.Notes = append(result.Notes, &models.EntityNote{
			ID:         n.ID,
			EntityType: models.EntityTypeFeature,
			EntityID:   feature.ID,
			NoteType:   models.NoteType(n.NoteType),
			Content:    n.Content,
			CreatedBy:  n.CreatedBy,
			Metadata:   n.Metadata,
		})
	}

	// Parse context data from feature's stored context_data column
	if feature.ContextData != nil && *feature.ContextData != "" {
		cd, parseErr := models.FromJSON(*feature.ContextData)
		if parseErr == nil {
			result.ContextData = cd
		}
	}

	return result, nil
}
