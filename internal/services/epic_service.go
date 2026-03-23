package services

import (
	"context"
	"fmt"
	"log/slog"
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

// EpicRepository defines the repository interface needed by EpicService.
// This interface is satisfied by *repository.EpicRepository.
type EpicRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Epic, error)
	GetByID(ctx context.Context, id int64) (*models.Epic, error)
	Create(ctx context.Context, epic *models.Epic) error
	Update(ctx context.Context, epic *models.Epic) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error)
	GetByFilePath(ctx context.Context, filePath string) (*models.Epic, error)
	UpdateFilePath(ctx context.Context, epicKey string, newFilePath *string) error
	UpdateKey(ctx context.Context, oldKey string, newKey string) error
	GetFeatureProgressDataByEpic(ctx context.Context, epicID int64) ([]repository.FeatureProgressData, error)
	GetFeatureStatusBreakdown(ctx context.Context, epicID int64) (map[models.FeatureStatus]int, error)
	GetFeatureStatusBreakdownByKey(ctx context.Context, epicKey string) (map[models.FeatureStatus]int, error)
	GetFeatureStatusRollup(ctx context.Context, epicID int64) (map[string]int, error)
	GetTaskStatusRollup(ctx context.Context, epicID int64) (map[string]int, error)
	UpdateStatus(ctx context.Context, epicID int64, status models.EpicStatus) error
	CascadeStatusToFeaturesAndTasks(ctx context.Context, epicID int64, targetFeatureStatus models.FeatureStatus, targetTaskStatus models.TaskStatus) error
	GetEpicDisplayDataRaw(ctx context.Context, epicID int64) (*repository.EpicDisplayDataRaw, error)
}

// EpicTaskLister defines the task repository interface needed by EpicService
// for querying blocked tasks across an epic and completing all tasks in an epic.
type EpicTaskLister interface {
	ListBlockedTasksByEpic(ctx context.Context, epicKey string) ([]*models.Task, error)
	ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error)
	UpdateStatusForced(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string, rejectionReason *string, documentPath *string, force bool) error
	GetStatusBreakdownMapBatch(ctx context.Context, featureIDs []int64) (map[int64]map[models.TaskStatus]int, error)
	GetTaskCountsForFeatures(ctx context.Context, featureIDs []int64) (map[int64]int, error)
}

// EpicWritableDocumentRepository defines the writable interface for document linking on epics.
// This interface is satisfied by *repository.DocumentRepository.
// (EpicWritableDocumentRepository removed -- replaced by EntityDocumentRepository + EntityDocumentLinkRepository)

// EpicFeatureCounter defines the feature counting interface needed by EpicService
// to count child features for backward transition warnings and epic completion.
type EpicFeatureCounter interface {
	ListByEpic(ctx context.Context, epicID int64) ([]*models.Feature, error)
	GetByID(ctx context.Context, id int64) (*models.Feature, error)
	Update(ctx context.Context, feature *models.Feature) error
	GetTaskStatusBreakdown(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error)
}

// EpicService provides business logic for epic operations.
type EpicService struct {
	repo             EpicRepository
	entitySvc        *EntityService
	entityRepo       EntityRepository
	featureRepo      EpicFeatureCounter
	taskRepo         EpicTaskLister
	docRepo          config.DocumentRepository
	relRepo          config.EpicRelationshipRepository
	docSvc           *EntityDocumentService // shared document operations; built by SetWritableDocRepo
	analyticsService *EpicAnalyticsService  // optional; lazy-initialized if nil
	enrichRepo       config.TemplateEnrichmentRepository
	tracer           trace.Tracer // optional; defaults to otel.Tracer("shark/services/epic") if nil
}

// NewEpicService creates a new EpicService.
// The entitySvc provides shared transition logic; it is automatically scoped to epic level.
// featureRepo and taskRepo can be nil for graceful degradation.
// Rejection note creation is handled by EntityService (via SetNoteRepo).
//
// Panics:
//   - If repo is nil (required dependency)
//   - If entitySvc is nil (required dependency)
func NewEpicService(repo EpicRepository, entitySvc *EntityService, entityRepo EntityRepository, featureRepo EpicFeatureCounter, taskRepo EpicTaskLister) *EpicService {
	requireNonNil(repo, "EpicService requires a non-nil EpicRepository")
	requireNonNil(entitySvc, "EpicService requires a non-nil EntityService")
	requireNonNil(entityRepo, "EpicService requires a non-nil EntityRepository")
	return &EpicService{
		repo:        repo,
		entitySvc:   entitySvc.ForLevel(workflow.LevelEpic),
		entityRepo:  entityRepo,
		featureRepo: featureRepo,
		taskRepo:    taskRepo,
		docRepo:     nil,
		relRepo:     nil,
	}
}

// SetTracer sets the OpenTelemetry tracer for the service.
// When nil, getTracer falls back to the OTel global tracer (noop until provider is wired).
func (s *EpicService) SetTracer(t trace.Tracer) {
	s.tracer = t
}

// getTracer returns the configured tracer or falls back to the OTel global tracer.
func (s *EpicService) getTracer() trace.Tracer {
	if s.tracer != nil {
		return s.tracer
	}
	return otel.Tracer("shark/services/epic")
}

// SetRelRepo sets the epic relationship repository on the service.
// This enables related epic lookups.
func (s *EpicService) SetRelRepo(relRepo config.EpicRelationshipRepository) {
	s.relRepo = relRepo
}

// SetDocRepo sets the document repository on the service.
// This is used when the service is created via NewEpicService (which does not accept docRepo)
// but the caller needs document lookup functionality (e.g., GetRelatedDocuments).
func (s *EpicService) SetDocRepo(docRepo config.DocumentRepository) {
	s.docRepo = docRepo
}

// SetEnrichRepo sets the template enrichment repository on the service.
// This enables enrichment data population for template rendering.
func (s *EpicService) SetEnrichRepo(enrichRepo config.TemplateEnrichmentRepository) {
	s.enrichRepo = enrichRepo
}

// SetWritableDocRepo sets the writable document repository on the service.
// This enables LinkDocument and UnlinkDocument operations on epics.
// The *repository.DocumentRepository type satisfies the EpicWritableDocumentRepository interface.
func (s *EpicService) SetWritableDocRepo(writableRepo EntityDocumentRepository, linkRepo EntityDocumentLinkRepository) {
	s.docSvc = NewEntityDocumentService(
		writableRepo,
		linkRepo,
		EntityLookupFnFromRepo(s.entityRepo),
	)
}

// SetAnalyticsService sets the analytics sub-service on EpicService.
// When set, analytics methods (CalculateProgress, GetProgress, GetFeatureRollup,
// GetTaskStatusRollup, GetImpediments, GetBlockedTasks, GetHealth, GetEpicDisplayData)
// delegate to this service instead of being computed inline.
//
// If not set, a default EpicAnalyticsService is lazily created from s.repo and s.taskRepo
// on first analytics method call.
func (s *EpicService) SetAnalyticsService(svc *EpicAnalyticsService) {
	s.analyticsService = svc
}

// getAnalyticsService returns the analytics sub-service, creating one lazily if nil.
func (s *EpicService) getAnalyticsService() *EpicAnalyticsService {
	if s.analyticsService == nil {
		s.analyticsService = NewEpicAnalyticsService(s.repo, s.taskRepo)
	}
	return s.analyticsService
}

// LinkDocument creates or retrieves a document by its title and file path, then links it to an epic.
// Delegates to the shared EntityDocumentService.
func (s *EpicService) LinkDocument(ctx context.Context, epicKey, docTitle, docPath string) error {
	if s.docSvc == nil {
		return fmt.Errorf("writable document repository not configured")
	}
	_, err := s.docSvc.LinkDocumentByKey(ctx, epicKey, docTitle, docPath)
	return err
}

// UnlinkDocument removes a document link from an epic by document title.
// Delegates to the shared EntityDocumentService.
func (s *EpicService) UnlinkDocument(ctx context.Context, epicKey, docTitle string) error {
	if s.docSvc == nil {
		return fmt.Errorf("writable document repository not configured")
	}
	return s.docSvc.UnlinkDocumentByKey(ctx, epicKey, docTitle)
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
	ctx, span := s.getTracer().Start(ctx, "EpicService.TransitionStatus",
		trace.WithAttributes(
			attribute.String("epic.key", epicKey),
			attribute.String("epic.target_status", targetStatus),
		),
	)
	defer span.End()

	// Delegate shared logic to EntityService
	result, err := s.entitySvc.TransitionStatus(
		ctx, s.entityRepo, models.EntityTypeEpic, epicKey, targetStatus, opts,
		DefaultTransitionFeatures(),
		s.makeResolveActionFn(ctx),
	)
	if err != nil {
		return nil, recordSpanError(span, err)
	}

	// Post-hook: count child features
	if s.featureRepo != nil {
		features, listErr := s.featureRepo.ListByEpic(ctx, result.EntityID)
		if listErr == nil {
			result.ChildCount = len(features)
		}
	}

	return result, nil
}

// GetNextStatus returns the available transitions for the current status of an epic.
func (s *EpicService) GetNextStatus(ctx context.Context, epicKey string) (*NextStatusInfo, error) {
	return s.entitySvc.GetNextStatus(ctx, s.entityRepo, models.EntityTypeEpic, epicKey,
		s.makeResolveActionFn(ctx))
}

// ValidateStatus checks if a status is valid in the epic workflow.
func (s *EpicService) ValidateStatus(status string) error {
	return s.entitySvc.GetWorkflowService().ValidateStatus(status)
}

// makeResolveActionFn returns a ResolveActionFn that generates Epic-specific
// placeholders including enrichment data, related documents, and related epics.
func (s *EpicService) makeResolveActionFn(ctx context.Context) ResolveActionFn {
	return func(entity models.Entity, status string) *config.PopulatedAction {
		epic, ok := entity.(*models.Epic)
		if !ok {
			return nil
		}

		// Fetch enrichment data (optional, graceful degradation)
		var enrichment *config.TemplateEnrichmentData
		if s.enrichRepo != nil {
			data, err := s.enrichRepo.GetEpicEnrichment(ctx, epic.ID)
			if err != nil {
				slog.Warn("Failed to fetch enrichment data for epic", "epic", epic.Key, "error", err)
			} else {
				enrichment = data
			}
		}

		// Determine which placeholder function to use based on available repositories
		var placeholders map[string]string
		if s.docRepo != nil && s.relRepo != nil {
			placeholders = config.EpicPlaceholdersWithRelated(epic, s.docRepo, s.relRepo, ctx, enrichment)
		} else {
			placeholders = config.EpicPlaceholders(epic)
			config.ApplyEnrichmentData(enrichment, placeholders)
		}

		return s.entitySvc.ResolveActionForStatus(status, placeholders)
	}
}

// GetEpic retrieves an epic by key.
func (s *EpicService) GetEpic(ctx context.Context, key string) (*models.Epic, error) {
	ctx, span := s.getTracer().Start(ctx, "EpicService.GetEpic",
		trace.WithAttributes(attribute.String("epic.key", key)),
	)
	defer span.End()

	epic, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, recordSpanError(span, fmt.Errorf("failed to get epic %s: %w", key, err))
	}
	if epic == nil {
		return nil, recordSpanError(span, fmt.Errorf("epic not found: %s", key))
	}
	return epic, nil
}

// ListEpics retrieves epics with optional filtering.
func (s *EpicService) ListEpics(ctx context.Context, filters EpicFilters) ([]*models.Epic, error) {
	ctx, span := s.getTracer().Start(ctx, "EpicService.ListEpics")
	defer span.End()

	var statusPtr *models.EpicStatus
	if filters.Status != "" {
		status := models.EpicStatus(filters.Status)
		statusPtr = &status
	}
	epics, err := s.repo.List(ctx, statusPtr)
	if err != nil {
		return nil, recordSpanError(span, fmt.Errorf("failed to list epics: %w", err))
	}
	return epics, nil
}

// CalculateProgress computes epic progress from raw feature data.
// Delegates to EpicAnalyticsService (lazily initialized if not set via SetAnalyticsService).
func (s *EpicService) CalculateProgress(ctx context.Context, epicID int64) (float64, error) {
	return s.getAnalyticsService().CalculateProgress(ctx, epicID)
}

// GetProgress retrieves progress metrics for an epic.
// Delegates to EpicAnalyticsService (lazily initialized if not set via SetAnalyticsService).
func (s *EpicService) GetProgress(ctx context.Context, key string) (*EpicProgressInfo, error) {
	return s.getAnalyticsService().GetProgress(ctx, key)
}

// GetFeatureRollup aggregates feature statuses for an epic.
// Delegates to EpicAnalyticsService (lazily initialized if not set via SetAnalyticsService).
func (s *EpicService) GetFeatureRollup(ctx context.Context, key string) (*FeatureRollup, error) {
	return s.getAnalyticsService().GetFeatureRollup(ctx, key)
}

// GetTaskStatusRollup aggregates task statuses across all features in an epic.
// Delegates to EpicAnalyticsService (lazily initialized if not set via SetAnalyticsService).
func (s *EpicService) GetTaskStatusRollup(ctx context.Context, key string) (map[string]int, error) {
	return s.getAnalyticsService().GetTaskStatusRollup(ctx, key)
}

// GetImpediments returns blocked tasks that impede epic progress.
// Delegates to EpicAnalyticsService (lazily initialized if not set via SetAnalyticsService).
func (s *EpicService) GetImpediments(ctx context.Context, key string) ([]*Impediment, error) {
	return s.getAnalyticsService().GetImpediments(ctx, key)
}

// GetBlockedTasks returns the raw blocked tasks that impede epic progress.
// Delegates to EpicAnalyticsService (lazily initialized if not set via SetAnalyticsService).
func (s *EpicService) GetBlockedTasks(ctx context.Context, key string) ([]*models.Task, error) {
	return s.getAnalyticsService().GetBlockedTasks(ctx, key)
}

// GetHealth analyzes the health of an epic based on blocked tasks and feature status.
// Delegates to EpicAnalyticsService (lazily initialized if not set via SetAnalyticsService).
func (s *EpicService) GetHealth(ctx context.Context, key string) (*EpicHealthInfo, error) {
	return s.getAnalyticsService().GetHealth(ctx, key)
}

// GetEpicDisplayData fetches all data needed to display an epic in a single SQL query
// via the epic_display_data view.
// Delegates to EpicAnalyticsService (lazily initialized if not set via SetAnalyticsService).
func (s *EpicService) GetEpicDisplayData(ctx context.Context, epic *models.Epic, projectRoot string) (*EpicDisplayData, error) {
	return s.getAnalyticsService().GetEpicDisplayData(ctx, epic, projectRoot)
}

// GetRelatedDocuments returns the documents associated with an epic.
// Degrades gracefully if docRepo is nil (returns empty slice).
func (s *EpicService) GetRelatedDocuments(ctx context.Context, epicID int64) ([]*models.Document, error) {
	if s.docRepo == nil {
		return []*models.Document{}, nil
	}
	docs, err := s.docRepo.ListForEpic(ctx, epicID)
	if err != nil {
		return nil, fmt.Errorf("failed to get related documents for epic ID %d: %w", epicID, err)
	}
	if docs == nil {
		return []*models.Document{}, nil
	}
	return docs, nil
}

// ListRelatedDocumentsByKey returns the documents associated with an epic identified by key.
// Delegates to the shared EntityDocumentService if available, otherwise falls back to
// GetRelatedDocuments. Degrades gracefully if no document repository is configured.
func (s *EpicService) ListRelatedDocumentsByKey(ctx context.Context, epicKey string) ([]*models.Document, error) {
	if s.docSvc != nil {
		return s.docSvc.ListDocumentsByKey(ctx, epicKey)
	}
	// Fallback for when only read-only docRepo is set (no writable doc repo)
	epic, err := s.repo.GetByKey(ctx, epicKey)
	if err != nil {
		return nil, fmt.Errorf("epic not found: %w", err)
	}
	return s.GetRelatedDocuments(ctx, epic.ID)
}

// CreateEpic creates a new epic.
//
// The epic key is auto-generated unless input.CustomKey is provided.
// Status defaults to "draft" and priority defaults to "medium" if not specified.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - input: create parameters
//
// Returns the created epic.
func (s *EpicService) CreateEpic(ctx context.Context, input CreateEpicInput) (*models.Epic, error) {
	if strings.TrimSpace(input.Title) == "" {
		return nil, fmt.Errorf("epic title cannot be empty")
	}

	// Determine epic key
	epicKey := input.CustomKey
	if epicKey != "" {
		epicKey = strings.ToUpper(epicKey)
		// Validate key doesn't already exist
		existing, err := s.repo.GetByKey(ctx, epicKey)
		if err == nil && existing != nil {
			return nil, fmt.Errorf("epic with key '%s' already exists", epicKey)
		}
	} else {
		// Auto-generate next epic key
		var err error
		epicKey, err = s.nextEpicKey(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to generate epic key: %w", err)
		}
	}

	// Set default status
	statusStr := input.Status
	if statusStr == "" {
		statusStr = "draft"
	}

	// Set default priority
	priorityStr := input.Priority
	if priorityStr == "" {
		priorityStr = "medium"
	}

	// Build slug for default file path generation (if no custom file path)
	var filePath *string
	if input.FilePath != nil {
		// Handle custom file path - check for collision
		if err := s.resolveEpicFilePath(ctx, input.FilePath, nil, input.Force); err != nil {
			return nil, err
		}
		filePath = input.FilePath
	} else {
		// Default path: docs/plan/{key}-{slug}/epic.md
		slug := utils.GenerateSlug(input.Title)
		defaultPath := fmt.Sprintf("docs/plan/%s-%s/epic.md", epicKey, slug)
		filePath = &defaultPath
	}

	var businessValue *models.Priority
	if input.BusinessValue != nil {
		bv := models.Priority(*input.BusinessValue)
		businessValue = &bv
	}

	epic := &models.Epic{
		BaseEntity: models.BaseEntity{
			Key:         epicKey,
			Title:       strings.TrimSpace(input.Title),
			Description: input.Description,
			FilePath:    filePath,
		},
		Status:        models.EpicStatus(statusStr),
		Priority:      models.Priority(priorityStr),
		BusinessValue: businessValue,
	}

	if err := epic.Validate(); err != nil {
		return nil, fmt.Errorf("epic validation failed: %w", err)
	}

	if err := s.repo.Create(ctx, epic); err != nil {
		return nil, fmt.Errorf("failed to create epic %s: %w", epicKey, err)
	}

	return epic, nil
}

// UpdateEpic updates fields on an existing epic.
// Only non-nil fields in updates are applied.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: epic key (e.g., "E07")
//   - updates: fields to update
//
// Returns the updated epic.
func (s *EpicService) UpdateEpic(ctx context.Context, key string, updates EpicUpdates) (*models.Epic, error) {
	epic, err := s.repo.GetByKey(ctx, strings.ToUpper(key))
	if err != nil {
		return nil, fmt.Errorf("failed to get epic %s: %w", key, err)
	}
	if epic == nil {
		return nil, fmt.Errorf("epic not found: %s", key)
	}

	// Apply non-nil updates
	if updates.Title != nil {
		if strings.TrimSpace(*updates.Title) == "" {
			return nil, fmt.Errorf("epic title cannot be empty")
		}
		epic.Title = strings.TrimSpace(*updates.Title)
	}
	if updates.Description != nil {
		epic.Description = updates.Description
	}
	if updates.Status != nil {
		epic.Status = *updates.Status
	}
	if updates.Priority != nil {
		epic.Priority = *updates.Priority
	}
	if updates.BusinessValue != nil {
		epic.BusinessValue = updates.BusinessValue
	}
	if err := epic.Validate(); err != nil {
		return nil, fmt.Errorf("epic validation failed: %w", err)
	}

	if err := s.repo.Update(ctx, epic); err != nil {
		return nil, fmt.Errorf("failed to update epic %s: %w", key, err)
	}

	// Update file path separately since repo.Update doesn't include file_path
	if updates.FilePath != nil {
		if err := s.repo.UpdateFilePath(ctx, epic.Key, updates.FilePath); err != nil {
			return nil, fmt.Errorf("failed to update epic %s file path: %w", key, err)
		}
		epic.FilePath = updates.FilePath
	}

	return epic, nil
}

// DeleteEpic deletes an epic by key.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: epic key (e.g., "E07")
//
// Returns an error if the epic is not found or cannot be deleted.
func (s *EpicService) DeleteEpic(ctx context.Context, key string) error {
	epic, err := s.repo.GetByKey(ctx, strings.ToUpper(key))
	if err != nil {
		return fmt.Errorf("failed to get epic %s: %w", key, err)
	}
	if epic == nil {
		return fmt.Errorf("epic not found: %s", key)
	}

	if err := s.repo.Delete(ctx, epic.ID); err != nil {
		return fmt.Errorf("failed to delete epic %s: %w", key, err)
	}

	return nil
}

// CompleteEpic completes all tasks and features in an epic.
//
// If force is false and there are incomplete tasks, returns an EpicCompleteResult
// with RequiresForce=true and IncompleteDetails populated so the caller can display
// a warning and prompt the user to retry with --force.
//
// If force is true, forces all non-completed tasks to "completed" status, marks all
// features as completed, and marks the epic as completed.
//
// The agentID parameter is used as the agent identifier for forced task completions.
// Pass an empty string to use the current user/system agent.
//
// NOTE: Progress recalculation (RecalculateAndSetProgress) for each feature is NOT
// performed here — callers should invoke FeatureService.RecalculateAndSetProgress
// for each feature after this call if they want accurate progress_pct values.
func (s *EpicService) CompleteEpic(ctx context.Context, epicKey string, force bool, agentID string) (*EpicCompleteResult, error) {
	ctx, span := s.getTracer().Start(ctx, "EpicService.CompleteEpic",
		trace.WithAttributes(attribute.String("epic.key", epicKey)),
	)
	defer span.End()

	epic, err := s.repo.GetByKey(ctx, strings.ToUpper(epicKey))
	if err != nil {
		return nil, fmt.Errorf("failed to get epic %s: %w", epicKey, err)
	}
	if epic == nil {
		return nil, fmt.Errorf("epic not found: %s", epicKey)
	}

	features, err := s.featureRepo.ListByEpic(ctx, epic.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list features for epic %s: %w", epicKey, err)
	}

	if len(features) == 0 {
		// No features — nothing to do, return early result
		return &EpicCompleteResult{
			EpicKey:         epic.Key,
			FeatureCount:    0,
			TotalCount:      0,
			CompletedCount:  0,
			AffectedTasks:   []string{},
			StatusBreakdown: map[string]int{},
		}, nil
	}

	var allTasks []*models.Task
	totalStatusBreakdown := make(map[models.TaskStatus]int)
	featureTaskBreakdown := make(map[string]map[models.TaskStatus]int)
	featureTaskCounts := make(map[string]int)

	for _, feature := range features {
		tasks, err := s.taskRepo.ListByFeature(ctx, feature.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to list tasks for feature %s: %w", feature.Key, err)
		}
		allTasks = append(allTasks, tasks...)
		featureTaskCounts[feature.Key] = len(tasks)

		statusBreakdown, err := s.featureRepo.GetTaskStatusBreakdown(ctx, feature.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get task status breakdown for feature %s: %w", feature.Key, err)
		}
		featureTaskBreakdown[feature.Key] = statusBreakdown
		for st, count := range statusBreakdown {
			totalStatusBreakdown[st] += count
		}
	}

	if len(allTasks) == 0 {
		// Features exist but no tasks — still return result
		return &EpicCompleteResult{
			EpicKey:         epic.Key,
			FeatureCount:    len(features),
			TotalCount:      0,
			CompletedCount:  0,
			AffectedTasks:   []string{},
			StatusBreakdown: map[string]int{},
		}, nil
	}

	completedCount := totalStatusBreakdown[models.TaskStatus("completed")]
	reviewedCount := totalStatusBreakdown[models.TaskStatus("ready_for_review")]
	allDoneCount := completedCount + reviewedCount
	hasIncomplete := allDoneCount < len(allTasks)

	if hasIncomplete && !force {
		// Build incomplete details per feature
		incompleteDetails := make(map[string]FeatureIncompleteDetails)
		for _, feature := range features {
			breakdown := featureTaskBreakdown[feature.Key]
			total := featureTaskCounts[feature.Key]
			completed := breakdown[models.TaskStatus("completed")] + breakdown[models.TaskStatus("ready_for_review")]
			incomplete := total - completed
			if incomplete > 0 {
				breakdownStr := make(map[string]int)
				for st, count := range breakdown {
					if st != models.TaskStatus("completed") && st != models.TaskStatus("ready_for_review") {
						breakdownStr[string(st)] = count
					}
				}
				incompleteDetails[feature.Key] = FeatureIncompleteDetails{
					TotalTasks:      total,
					CompletedTasks:  completed,
					IncompleteCount: incomplete,
					StatusBreakdown: breakdownStr,
				}
			}
		}

		statusBreakdownMap := make(map[string]int)
		for st, count := range totalStatusBreakdown {
			statusBreakdownMap[string(st)] = count
		}

		return &EpicCompleteResult{
			EpicKey:           epic.Key,
			FeatureCount:      len(features),
			TotalCount:        len(allTasks),
			CompletedCount:    completedCount,
			AffectedTasks:     []string{},
			StatusBreakdown:   statusBreakdownMap,
			RequiresForce:     true,
			IncompleteDetails: incompleteDetails,
		}, nil
	}

	// Force-complete all tasks or complete all-done tasks
	var affectedTaskKeys []string
	newCompletedCount := completedCount

	for _, task := range allTasks {
		if task.Status == models.TaskStatus("completed") {
			continue
		}
		agentPtr := &agentID
		if err := s.taskRepo.UpdateStatusForced(ctx, task.ID, models.TaskStatus("completed"), agentPtr, nil, nil, nil, true); err != nil {
			return nil, fmt.Errorf("failed to complete task %s: %w", task.Key, err)
		}
		newCompletedCount++
		affectedTaskKeys = append(affectedTaskKeys, task.Key)
	}

	// Mark all features as completed
	for _, feature := range features {
		updatedFeature, err := s.featureRepo.GetByID(ctx, feature.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get updated feature %s: %w", feature.Key, err)
		}
		if updatedFeature.Status != models.FeatureStatusCompleted {
			updatedFeature.Status = models.FeatureStatusCompleted
			if err := s.featureRepo.Update(ctx, updatedFeature); err != nil {
				return nil, fmt.Errorf("failed to complete feature %s: %w", updatedFeature.Key, err)
			}
		}
	}

	// Mark epic as completed
	epic.Status = models.EpicStatusCompleted
	if err := s.repo.Update(ctx, epic); err != nil {
		return nil, fmt.Errorf("failed to complete epic %s: %w", epicKey, err)
	}

	statusBreakdownMap := make(map[string]int)
	for st, count := range totalStatusBreakdown {
		statusBreakdownMap[string(st)] = count
	}
	statusBreakdownMap["completed"] = newCompletedCount

	if affectedTaskKeys == nil {
		affectedTaskKeys = []string{}
	}

	return &EpicCompleteResult{
		EpicKey:         epic.Key,
		FeatureCount:    len(features),
		TotalCount:      len(allTasks),
		CompletedCount:  newCompletedCount,
		AffectedTasks:   affectedTaskKeys,
		StatusBreakdown: statusBreakdownMap,
		ForceCompleted:  force && hasIncomplete,
	}, nil
}

// GetFeatures returns all features belonging to an epic.
// This is used by CLI commands that need feature IDs for further processing
// (e.g., progress recalculation after completing an epic).
func (s *EpicService) GetFeatures(ctx context.Context, epicKey string) ([]*models.Feature, error) {
	epic, err := s.repo.GetByKey(ctx, strings.ToUpper(epicKey))
	if err != nil {
		return nil, fmt.Errorf("failed to get epic %s: %w", epicKey, err)
	}
	if epic == nil {
		return nil, fmt.Errorf("epic not found: %s", epicKey)
	}
	if s.featureRepo == nil {
		return nil, nil
	}
	return s.featureRepo.ListByEpic(ctx, epic.ID)
}

// RecalculateStatus recalculates and persists the derived status for an epic
// based on the statuses of its child features.
//
// Returns the previous and new status values. If the status did not change
// or if the epic is in planning mode, the result reflects that without error.
func (s *EpicService) RecalculateStatus(ctx context.Context, epicID int64) (*EpicRecalcResult, error) {
	epic, err := s.repo.GetByID(ctx, epicID)
	if err != nil {
		return nil, fmt.Errorf("failed to get epic for status recalculation: %w", err)
	}
	if epic == nil {
		return nil, fmt.Errorf("epic not found for ID %d", epicID)
	}

	featureCounts, err := s.repo.GetFeatureStatusBreakdown(ctx, epicID)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature status breakdown for epic %s: %w", epic.Key, err)
	}

	// Derive new status from feature breakdown
	newStatus := deriveEpicStatusFromFeatures(featureCounts, epic.Status)
	previousStatus := string(epic.Status)

	result := &EpicRecalcResult{
		EpicKey:        epic.Key,
		PreviousStatus: previousStatus,
		NewStatus:      string(newStatus),
		WasChanged:     string(newStatus) != previousStatus,
	}

	if result.WasChanged {
		if err := s.repo.UpdateStatus(ctx, epicID, newStatus); err != nil {
			return nil, fmt.Errorf("failed to update epic %s status: %w", epic.Key, err)
		}
	}

	return result, nil
}

// CascadeStatusToFeaturesAndTasks cascades a status change from this epic to all
// child features and their tasks. This is used when force-completing an epic via
// the --status=completed --force flags.
func (s *EpicService) CascadeStatusToFeaturesAndTasks(ctx context.Context, epicID int64, featureStatus models.FeatureStatus, taskStatus models.TaskStatus) error {
	if err := s.repo.CascadeStatusToFeaturesAndTasks(ctx, epicID, featureStatus, taskStatus); err != nil {
		return fmt.Errorf("failed to cascade status to features and tasks for epic ID %d: %w", epicID, err)
	}
	return nil
}

// RenameKey updates the key of an existing epic.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - oldKey: current epic key (e.g., "E07")
//   - newKey: desired new epic key (e.g., "E08")
//
// Returns an error if the old key does not exist or if the update fails.
func (s *EpicService) RenameKey(ctx context.Context, oldKey, newKey string) error {
	if err := s.repo.UpdateKey(ctx, oldKey, newKey); err != nil {
		return fmt.Errorf("failed to rename epic key from %s to %s: %w", oldKey, newKey, err)
	}
	return nil
}

// ResolveEpicPath returns the relative file path for an epic's planning document.
// If the epic has an explicit FilePath set, that is returned.
// Otherwise, the default slug-based path is computed:
//
//	docs/plan/{EpicKey}-{slug}/epic.md
//
// The projectRoot parameter is the absolute path to the project root directory.
// It is used to compute the relative path from the absolute path.
func (s *EpicService) ResolveEpicPath(ctx context.Context, epicKey string, projectRoot string) (string, error) {
	epic, err := s.repo.GetByKey(ctx, strings.ToUpper(epicKey))
	if err != nil {
		return "", fmt.Errorf("failed to get epic %s: %w", epicKey, err)
	}
	if epic == nil {
		return "", fmt.Errorf("epic not found: %s", epicKey)
	}

	return resolveEpicPathFromLoaded(epic, projectRoot), nil
}

// nextEpicKey generates the next available epic key (E##) by inspecting existing epics.
func (s *EpicService) nextEpicKey(ctx context.Context) (string, error) {
	epics, err := s.repo.List(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to list epics: %w", err)
	}

	maxNum := 0
	for _, e := range epics {
		var num int
		if _, err := fmt.Sscanf(e.Key, "E%d", &num); err == nil {
			if num > maxNum {
				maxNum = num
			}
		}
	}
	return fmt.Sprintf("E%02d", maxNum+1), nil
}

// resolveEpicFilePath checks for file path collisions and handles force reassignment.
// currentEpicKey is the key of the epic being updated (nil for new epics).
func (s *EpicService) resolveEpicFilePath(ctx context.Context, filePath *string, currentEpicKey *string, force bool) error {
	if filePath == nil {
		return nil
	}

	fp := *filePath

	// Check if another epic claims this file
	existingEpic, err := s.repo.GetByFilePath(ctx, fp)
	if err != nil {
		return fmt.Errorf("failed to check epic file path collision: %w", err)
	}

	if existingEpic != nil {
		// If it's the same epic being updated, no collision
		if currentEpicKey != nil && existingEpic.Key == *currentEpicKey {
			return nil
		}
		if !force {
			return fmt.Errorf("file '%s' is already claimed by epic %s ('%s'). Use force to reassign",
				fp, existingEpic.Key, existingEpic.Title)
		}
		// Force: release the file from the existing epic
		if err := s.repo.UpdateFilePath(ctx, existingEpic.Key, nil); err != nil {
			return fmt.Errorf("failed to release file path from epic %s: %w", existingEpic.Key, err)
		}
	}

	return nil
}
