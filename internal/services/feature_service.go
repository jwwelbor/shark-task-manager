package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/keys"
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
	// UpdateNoResequence updates a feature without cascading execution_order
	// changes to siblings. Used to preserve intentional duplicate-order
	// groups (parallel work). Wired from `--parallel` on `shark feature update`.
	UpdateNoResequence(ctx context.Context, feature *models.Feature) error
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

type featureAggregateTxRepository interface {
	BeginTx(ctx context.Context) (*sql.Tx, error)
	CreateWithTx(ctx context.Context, tx *sql.Tx, feature *models.Feature) error
	UpdateWithTx(ctx context.Context, tx *sql.Tx, feature *models.Feature, skipResequence bool) error
	DeleteWithTx(ctx context.Context, tx *sql.Tx, id int64) error
	UpdateStatusTx(ctx context.Context, tx *sql.Tx, id int64, status string, agent *string, notes *string) error
	UpdateStatusIfCurrentTx(ctx context.Context, tx *sql.Tx, featureID int64, expectedStatus models.FeatureStatus, newStatus models.FeatureStatus) (bool, error)
	UpdateStatusIfNotOverriddenWithTx(ctx context.Context, tx *sql.Tx, featureID int64, newStatus models.FeatureStatus) (bool, error)
	CascadeStatusToTasksWithTx(ctx context.Context, tx *sql.Tx, featureID int64, targetTaskStatus models.TaskStatus) error
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
	repo                 FeatureRepository
	entitySvc            *EntityService
	entityRepo           EntityRepository
	taskRepo             FeatureTaskCounter
	docRepo              DocumentRepository
	relRepo              FeatureRelationshipRepository
	epicLookupRepo       FeatureEpicLookup
	docSvc               *EntityDocumentService // shared document operations; built by SetWritableDocRepo
	progressService      *FeatureProgressService
	enrichRepo           config.TemplateEnrichmentRepository
	entityHistoryRepo    EntityHistoryRecorder // optional: records to entity_history table
	tracer               trace.Tracer          // optional; defaults to otel.Tracer("shark/services/feature") if nil
	searchIndexer        SearchIndexer
	aggregateCoordinator *AggregateMutationCoordinator

	// tagSvc is optional — nil disables tag integration.
	// TagQuerier extends TagAttacher with EntityIDsByTags for list filtering (F05).
	tagSvc TagQuerier

	// sizeCfg is optional — nil disables size enforcement on create.
	sizeCfg SizeEnforcementConfig

	// Cascade reopen dependencies (all optional; cascade fires only when all are non-nil).
	cascadeDB          txBeginner
	cascadeEpicRepo    CascadeEpicRepo
	cascadeHistQuerier ParentReopenHistoryQuerier
	cascadeHistTx      EntityHistoryTxRecorder
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

// SetTagService wires the optional TagQuerier dependency. When nil, tag
// hooks in CreateFeature, UpdateFeature, and ListFeatures are skipped silently.
// TagQuerier extends TagAttacher with EntityIDsByTags for list filtering (F05).
func (s *FeatureService) SetTagService(tagSvc TagQuerier) {
	s.tagSvc = tagSvc
}

// SetSearchIndexer wires the optional search indexer used after feature writes.
func (s *FeatureService) SetSearchIndexer(indexer SearchIndexer) {
	s.searchIndexer = indexer
}

// SetAggregateMutationCoordinator wires the transactional parent epic rollup
// used for feature membership mutations.
func (s *FeatureService) SetAggregateMutationCoordinator(coordinator *AggregateMutationCoordinator) {
	s.aggregateCoordinator = coordinator
}

// SetSizeEnforcement wires the optional SizeEnforcementConfig. When nil or
// when the config does not list "feature" in SizeRequiredFor, CreateFeature
// accepts nil Size silently.
func (s *FeatureService) SetSizeEnforcement(cfg SizeEnforcementConfig) {
	s.sizeCfg = cfg
}

// SetCascadeDeps wires the optional cascade reopen dependencies for FeatureService.
// All four parameters must be non-nil for the cascade to fire; any nil value
// disables the cascade silently (graceful degradation per AC-T5).
// Note: FeatureService does not need a CascadeFeatureRepo because the cascade from
// a feature transition starts directly at the epic leg.
func (s *FeatureService) SetCascadeDeps(db txBeginner, er CascadeEpicRepo, hq ParentReopenHistoryQuerier, ht EntityHistoryTxRecorder) {
	s.cascadeDB = db
	s.cascadeEpicRepo = er
	s.cascadeHistQuerier = hq
	s.cascadeHistTx = ht
}

// cascadeEnabled returns true iff all four cascade dependencies are non-nil.
func (s *FeatureService) cascadeEnabled() bool {
	return s.cascadeDB != nil && s.cascadeEpicRepo != nil && s.cascadeHistQuerier != nil && s.cascadeHistTx != nil
}

// cascadeDepsBundle packages the cascade dependencies into the cascadeDeps struct.
// The featureCascadeReadAdapter satisfies CascadeFeatureRepo for the feature+epic path
// (cascadeLegFeature), which still needs to look up the feature to get feature.EpicID.
// For the epic-only path (cascadeLegEpic + epicID != 0), the cascade skips featureRepo
// entirely, so the adapter is never called — it exists only to satisfy the interface.
func (s *FeatureService) cascadeDepsBundle() cascadeDeps {
	return cascadeDeps{
		db:             s.cascadeDB,
		featureRepo:    &featureCascadeReadAdapter{repo: s.repo},
		epicRepo:       s.cascadeEpicRepo,
		historyQuerier: s.cascadeHistQuerier,
		historyTx:      s.cascadeHistTx,
		workflowSvc:    &workflowProviderAdapter{svc: s.entitySvc.GetWorkflowService()},
	}
}

// SetWritableDocRepo sets the writable document repository on the service.
// This enables LinkDocument and UnlinkDocument operations on features.
func (s *FeatureService) SetWritableDocRepo(writableRepo EntityDocumentRepository, linkRepo EntityDocumentLinkRepository, projectRoot string) {
	s.docSvc = NewEntityDocumentService(
		writableRepo,
		linkRepo,
		EntityLookupFnFromRepo(s.entityRepo),
		projectRoot,
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

	var result *TransitionResult
	if s.aggregateCoordinator != nil {
		featureForAggregate, err := s.repo.GetByKey(ctx, featureKey)
		if err != nil {
			return nil, recordSpanError(span, fmt.Errorf("failed to get feature %s for aggregate transition: %w", featureKey, err))
		}
		if featureForAggregate == nil {
			return nil, recordSpanError(span, fmt.Errorf("feature not found: %s", featureKey))
		}
		txRepo, ok := s.repo.(featureAggregateTxRepository)
		if !ok {
			return nil, recordSpanError(span, fmt.Errorf("feature repository does not support aggregate transactions"))
		}
		tx, err := txRepo.BeginTx(ctx)
		if err != nil {
			return nil, recordSpanError(span, fmt.Errorf("failed to begin feature transition transaction: %w", err))
		}
		defer rollbackAfterAggregateMutation(tx)
		txEntityRepo := &transactionalEntityRepo{
			EntityRepository: s.entityRepo,
			updateStatus: func(ctx context.Context, id int64, status string) error {
				return txRepo.UpdateStatusTx(ctx, tx, id, status, nil, nil)
			},
			updateStatusIfCurrent: func(ctx context.Context, id int64, expected, status string) (bool, error) {
				return txRepo.UpdateStatusIfCurrentTx(ctx, tx, id, models.FeatureStatus(expected), models.FeatureStatus(status))
			},
		}
		result, err = s.entitySvc.TransitionStatusWithTx(
			ctx, tx, txEntityRepo, models.EntityTypeFeature, featureKey, targetStatus, opts,
			DefaultTransitionFeatures(), s.makeResolveActionFn(ctx),
		)
		if err != nil {
			return nil, recordSpanError(span, err)
		}
		triggerKind := "transition"
		featureWf := s.entitySvc.GetWorkflowService().ForLevel(workflow.LevelFeature)
		if result.Transitioned && featureWf.IsTerminalStatus(result.FromStatus) && !featureWf.IsTerminalStatus(result.ToStatus) {
			triggerKind = "regression"
		}
		if result.Transitioned {
			if err := s.aggregateCoordinator.RefreshEpicStatus(ctx, tx, featureForAggregate.EpicID, cascadeTrigger{
				triggerKey: featureKey, triggerKind: triggerKind, triggerType: models.EntityTypeFeature, startLeg: cascadeLegEpic,
			}); err != nil {
				return nil, recordSpanError(span, fmt.Errorf("failed to maintain feature aggregates: %w", err))
			}
		}
		if err := tx.Commit(); err != nil {
			return nil, recordSpanError(span, fmt.Errorf("failed to commit feature transition: %w", err))
		}
	} else {
		// Delegate shared logic to EntityService.
		var err error
		result, err = s.entitySvc.TransitionStatus(
			ctx, s.entityRepo, models.EntityTypeFeature, featureKey, targetStatus, opts,
			DefaultTransitionFeatures(), s.makeResolveActionFn(ctx),
		)
		if err != nil {
			return nil, recordSpanError(span, err)
		}
	}

	// Post-hook: count child tasks
	if s.taskRepo != nil {
		tasks, listErr := s.taskRepo.ListByFeature(ctx, result.EntityID)
		if listErr == nil {
			result.ChildCount = len(tasks)
		}
	}

	// Cascade post-hook: reopen terminal epic when a feature regresses from
	// a terminal status to a non-terminal status (AC-02 / REQ-F-001).
	if s.aggregateCoordinator == nil && s.cascadeEnabled() {
		featureWf := s.entitySvc.GetWorkflowService().ForLevel(workflow.LevelFeature)
		if featureWf.IsTerminalStatus(result.FromStatus) && !featureWf.IsTerminalStatus(result.ToStatus) {
			cascadeParentReopens(ctx, s.cascadeDepsBundle(), cascadeTrigger{
				triggerKey:  featureKey,
				triggerKind: "regression",
				triggerType: models.EntityTypeFeature,
				startLeg:    cascadeLegEpic,
				featureID:   result.EntityID,
			})
		}
	}
	if err := indexEntityIfConfigured(ctx, s.searchIndexer, models.EntityTypeFeature, result.EntityID); err != nil {
		return nil, recordSpanError(span, err)
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

// GetFeatureWithTags returns the feature and the sorted list of tag names
// attached to it. When tagSvc is nil the tags slice is nil (graceful
// degradation — consistent with F04 REQ-F-018). When ListTagsForEntity fails
// the method returns (nil, nil, wrappedErr) per AC-T3.
func (s *FeatureService) GetFeatureWithTags(ctx context.Context, key string) (*models.Feature, []string, error) {
	ctx, span := s.getTracer().Start(ctx, "FeatureService.GetFeatureWithTags",
		trace.WithAttributes(attribute.String("feature.key", key)),
	)
	defer span.End()

	feature, err := s.GetFeature(ctx, key)
	if err != nil {
		return nil, nil, err
	}
	if s.tagSvc == nil {
		return feature, nil, nil
	}
	names, err := s.tagSvc.ListTagsForEntity(ctx, models.EntityTypeFeature, feature.ID)
	if err != nil {
		return nil, nil, recordSpanError(span, fmt.Errorf("load tags for feature %s: %w", key, err))
	}
	return feature, names, nil
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

	// Block 1: pre-filter by tag IDs (E28-F05 §2.5.2).
	var taggedIDSet map[int64]struct{}
	if len(filters.Tags) > 0 {
		if s.tagSvc == nil {
			return nil, recordSpanError(span, &TagFilterUnavailableError{})
		}
		ids, err := s.tagSvc.EntityIDsByTags(ctx, models.EntityTypeFeature, filters.Tags, TagQueryOpAnd)
		if err != nil {
			return nil, recordSpanError(span, err)
		}
		if len(ids) == 0 {
			return []*models.Feature{}, nil // REQ-F-017 short-circuit
		}
		taggedIDSet = make(map[int64]struct{}, len(ids))
		for _, id := range ids {
			taggedIDSet[id] = struct{}{}
		}
	}

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

	// Block 2: post-filter in-memory (E28-F05 §2.5.2).
	features = filterByTagIDs(features, taggedIDSet, func(f *models.Feature) int64 { return f.ID })

	return features, nil
}

// ListFeaturesByEpicKey retrieves features for a specific epic using filters.EpicKey,
// with optional status filtering via filters.Status and tag filtering via filters.Tags.
// Unlike ListFeatures, this supports epic-scoped queries by resolving the epic key via epicLookupRepo.
// Returns an error if epicLookupRepo is nil or the epic does not exist.
//
// Tag filtering (E28-F05): when filters.Tags is non-empty, Block 1 pre-filters the entity ID
// set via tagSvc.EntityIDsByTags, and Block 2 post-filters the base-list result in-memory.
// This preserves the indexed ListByEpic / ListByEpicAndStatus repository path.
func (s *FeatureService) ListFeaturesByEpicKey(ctx context.Context, filters FeatureFilters) ([]*models.Feature, error) {
	ctx, span := s.getTracer().Start(ctx, "FeatureService.ListFeaturesByEpicKey")
	defer span.End()

	if s.epicLookupRepo == nil {
		return nil, recordSpanError(span, fmt.Errorf("epic lookup repository not available"))
	}
	epic, err := s.epicLookupRepo.GetByKey(ctx, filters.EpicKey)
	if err != nil {
		return nil, recordSpanError(span, fmt.Errorf("epic %s does not exist: %w", filters.EpicKey, err))
	}
	if epic == nil {
		return nil, recordSpanError(span, fmt.Errorf("epic %s not found", filters.EpicKey))
	}

	// Block 1: pre-filter by tag IDs (E28-F05 §2.5.2).
	var taggedIDSet map[int64]struct{}
	if len(filters.Tags) > 0 {
		if s.tagSvc == nil {
			return nil, recordSpanError(span, &TagFilterUnavailableError{})
		}
		ids, err := s.tagSvc.EntityIDsByTags(ctx, models.EntityTypeFeature, filters.Tags, TagQueryOpAnd)
		if err != nil {
			return nil, recordSpanError(span, err)
		}
		if len(ids) == 0 {
			return []*models.Feature{}, nil // REQ-F-017 short-circuit
		}
		taggedIDSet = make(map[int64]struct{}, len(ids))
		for _, id := range ids {
			taggedIDSet[id] = struct{}{}
		}
	}

	var features []*models.Feature
	if filters.Status != "" {
		features, err = s.repo.ListByEpicAndStatus(ctx, epic.ID, models.FeatureStatus(filters.Status))
	} else {
		features, err = s.repo.ListByEpic(ctx, epic.ID)
	}
	if err != nil {
		return nil, recordSpanError(span, fmt.Errorf("failed to list features for epic %s: %w", filters.EpicKey, err))
	}

	// Block 2: post-filter in-memory (E28-F05 §2.5.2).
	features = filterByTagIDs(features, taggedIDSet, func(f *models.Feature) int64 { return f.ID })

	return features, nil
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
	if s.searchIndexer == nil {
		return nil
	}
	feature, err := s.repo.GetByKey(ctx, newKey)
	if err != nil {
		return fmt.Errorf("failed to get renamed feature %s for indexing: %w", newKey, err)
	}
	if feature == nil {
		return fmt.Errorf("renamed feature not found: %s", newKey)
	}
	if err := indexEntityIfConfigured(ctx, s.searchIndexer, models.EntityTypeFeature, feature.ID); err != nil {
		return err
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
	var updated bool
	if s.aggregateCoordinator != nil {
		txRepo, ok := s.repo.(featureAggregateTxRepository)
		if !ok {
			return false, fmt.Errorf("feature repository does not support aggregate transactions")
		}
		tx, err := txRepo.BeginTx(ctx)
		if err != nil {
			return false, fmt.Errorf("failed to begin conditional feature status transaction: %w", err)
		}
		defer rollbackAfterAggregateMutation(tx)
		updated, err = txRepo.UpdateStatusIfNotOverriddenWithTx(ctx, tx, feature.ID, newStatus)
		if err != nil {
			return false, err
		}
		if updated {
			triggerKind := "status_update"
			featureWf := s.entitySvc.GetWorkflowService().ForLevel(workflow.LevelFeature)
			if featureWf.IsTerminalStatus(string(feature.Status)) && !featureWf.IsTerminalStatus(string(newStatus)) {
				triggerKind = "regression"
			}
			if err := s.aggregateCoordinator.RefreshEpicStatus(ctx, tx, feature.EpicID, cascadeTrigger{
				triggerKey: feature.Key, triggerKind: triggerKind, triggerType: models.EntityTypeFeature, startLeg: cascadeLegEpic, epicID: feature.EpicID,
			}); err != nil {
				return false, fmt.Errorf("failed to maintain feature aggregates: %w", err)
			}
		}
		if err := tx.Commit(); err != nil {
			return false, fmt.Errorf("failed to commit conditional feature status: %w", err)
		}
	} else {
		updated, err = s.repo.UpdateStatusIfNotOverridden(ctx, feature.ID, newStatus)
		if err != nil {
			return false, err
		}
	}
	if !updated {
		return false, nil
	}
	if err := indexEntityIfConfigured(ctx, s.searchIndexer, models.EntityTypeFeature, feature.ID); err != nil {
		return false, err
	}
	return true, nil
}

// CascadeFeatureStatusToTasks sets all tasks in a feature to the given status.
func (s *FeatureService) CascadeFeatureStatusToTasks(ctx context.Context, featureKey string, targetTaskStatus models.TaskStatus) error {
	feature, err := s.repo.GetByKey(ctx, featureKey)
	if err != nil {
		return fmt.Errorf("feature %s does not exist: %w", featureKey, err)
	}
	if s.aggregateCoordinator != nil {
		txRepo, ok := s.repo.(featureAggregateTxRepository)
		if !ok {
			return fmt.Errorf("feature repository does not support aggregate transactions")
		}
		tx, err := txRepo.BeginTx(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin feature cascade transaction: %w", err)
		}
		defer rollbackAfterAggregateMutation(tx)
		if err := txRepo.CascadeStatusToTasksWithTx(ctx, tx, feature.ID, targetTaskStatus); err != nil {
			return fmt.Errorf("failed to cascade status to tasks for feature %s: %w", featureKey, err)
		}
		if err := s.aggregateCoordinator.RefreshFeatureAndEpic(ctx, tx, feature.ID, cascadeTrigger{
			triggerKey: feature.Key, triggerKind: "task_cascade", triggerType: models.EntityTypeFeature, startLeg: cascadeLegFeature, featureID: feature.ID,
		}); err != nil {
			return fmt.Errorf("failed to maintain feature aggregates: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit feature cascade: %w", err)
		}
	} else if err := s.repo.CascadeStatusToTasks(ctx, feature.ID, targetTaskStatus); err != nil {
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
		if s.aggregateCoordinator != nil {
			txRepo, ok := s.repo.(featureAggregateTxRepository)
			if !ok {
				return nil, fmt.Errorf("feature repository does not support aggregate transactions")
			}
			tx, err := txRepo.BeginTx(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to begin feature completion transaction: %w", err)
			}
			defer rollbackAfterAggregateMutation(tx)
			if err := txRepo.UpdateStatusTx(ctx, tx, feature.ID, string(models.FeatureStatusCompleted), nil, nil); err != nil {
				return nil, fmt.Errorf("failed to complete feature %s: %w", featureKey, err)
			}
			if err := s.aggregateCoordinator.RefreshEpicStatus(ctx, tx, feature.EpicID, cascadeTrigger{
				triggerKey: feature.Key, triggerKind: "completion", triggerType: models.EntityTypeFeature, startLeg: cascadeLegEpic, epicID: feature.EpicID,
			}); err != nil {
				return nil, fmt.Errorf("failed to maintain feature aggregates: %w", err)
			}
			if err := tx.Commit(); err != nil {
				return nil, fmt.Errorf("failed to commit feature completion: %w", err)
			}
		} else {
			feature.Status = models.FeatureStatusCompleted
			if err := s.repo.Update(ctx, feature); err != nil {
				return nil, fmt.Errorf("failed to complete feature %s: %w", featureKey, err)
			}
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

	// Force-complete all incomplete tasks in one repository-owned transaction.
	affectedKeys := make([]string, 0)
	for _, task := range tasks {
		if task.Status == models.TaskStatus("completed") {
			continue
		}
		affectedKeys = append(affectedKeys, task.Key)
	}
	if s.aggregateCoordinator != nil {
		txRepo, ok := s.repo.(featureAggregateTxRepository)
		if !ok {
			return nil, fmt.Errorf("feature repository does not support aggregate transactions")
		}
		tx, err := txRepo.BeginTx(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to begin feature completion transaction: %w", err)
		}
		defer rollbackAfterAggregateMutation(tx)
		if len(affectedKeys) > 0 {
			if err := txRepo.CascadeStatusToTasksWithTx(ctx, tx, feature.ID, models.TaskStatus("completed")); err != nil {
				return nil, fmt.Errorf("failed to complete tasks for feature %s: %w", featureKey, err)
			}
		}
		if err := s.aggregateCoordinator.RefreshFeatureAndEpic(ctx, tx, feature.ID, cascadeTrigger{
			triggerKey: feature.Key, triggerKind: "completion", triggerType: models.EntityTypeFeature, startLeg: cascadeLegFeature, featureID: feature.ID,
		}); err != nil {
			return nil, fmt.Errorf("failed to maintain feature aggregates: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("failed to commit feature completion: %w", err)
		}
	} else {
		if len(affectedKeys) > 0 {
			if err := s.repo.CascadeStatusToTasks(ctx, feature.ID, models.TaskStatus("completed")); err != nil {
				return nil, fmt.Errorf("failed to complete tasks for feature %s: %w", featureKey, err)
			}
		}
		if err := s.RecalculateAndSetProgress(ctx, feature.ID); err != nil {
			return nil, fmt.Errorf("failed to update feature progress: %w", err)
		}
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
	if s.aggregateCoordinator == nil {
		return s.getProgressService().RecalculateAndSetProgress(ctx, featureID)
	}
	txRepo, ok := s.repo.(featureAggregateTxRepository)
	if !ok {
		return fmt.Errorf("feature repository does not support aggregate transactions")
	}
	tx, err := txRepo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin feature progress transaction: %w", err)
	}
	defer rollbackAfterAggregateMutation(tx)
	if err := s.aggregateCoordinator.RefreshFeatureAndEpic(ctx, tx, featureID, cascadeTrigger{
		triggerKind: "progress_refresh", triggerType: models.EntityTypeFeature, startLeg: cascadeLegFeature, featureID: featureID,
	}); err != nil {
		return fmt.Errorf("failed to refresh feature progress: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit feature progress: %w", err)
	}
	return nil
}

// RecalculateAndSetProgressByKey recalculates progress for a feature identified by key.
func (s *FeatureService) RecalculateAndSetProgressByKey(ctx context.Context, key string) error {
	if s.aggregateCoordinator == nil {
		return s.getProgressService().RecalculateAndSetProgressByKey(ctx, key)
	}
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

	if err := enforceTagsRequired(ctx, s.tagSvc, models.EntityTypeFeature, input.Tags); err != nil {
		return nil, err
	}
	if err := enforceSizeRequired(s.sizeCfg, models.EntityTypeFeature, input.Size); err != nil {
		return nil, err
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

	// Determine feature key (B063: honor a custom key when supplied, following
	// the same validation/uniqueness pattern as EpicService.CreateEpic).
	var featureKey string
	if input.CustomKey != "" {
		var keyErr error
		featureKey, keyErr = resolveFeatureCustomKey(input.CustomKey, epicKey)
		if keyErr != nil {
			return nil, keyErr
		}
		existing, err := s.repo.GetByKey(ctx, featureKey)
		if err == nil && existing != nil {
			if next := s.suggestNextFeatureKey(ctx, epic.ID, epicKey); next != "" {
				return nil, fmt.Errorf("feature with key '%s' already exists (next available: %s)", featureKey, next)
			}
			return nil, fmt.Errorf("feature with key '%s' already exists", featureKey)
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("failed to check existing feature key %s: %w", featureKey, err)
		}
	} else {
		featureKey, err = s.nextFeatureKey(ctx, epic.ID, epicKey)
		if err != nil {
			return nil, fmt.Errorf("failed to generate feature key: %w", err)
		}
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
		FilePath:    filePath,
		Size:        input.Size}, EpicID: epic.ID,

		Status:         models.FeatureStatus(statusStr),
		ExecutionOrder: input.ExecutionOrder,
	}

	if err := feature.Validate(); err != nil {
		return nil, fmt.Errorf("feature validation failed: %w", err)
	}

	if s.aggregateCoordinator != nil {
		txRepo, ok := s.repo.(featureAggregateTxRepository)
		if !ok {
			return nil, fmt.Errorf("feature repository does not support aggregate transactions")
		}
		tx, err := txRepo.BeginTx(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to begin feature aggregate transaction: %w", err)
		}
		defer rollbackAfterAggregateMutation(tx)
		if err := txRepo.CreateWithTx(ctx, tx, feature); err != nil {
			return nil, fmt.Errorf("failed to create feature %s: %w", featureKey, err)
		}
		if err := s.aggregateCoordinator.RefreshEpicStatus(ctx, tx, epic.ID, cascadeTrigger{
			triggerKey: feature.Key, triggerKind: "creation", triggerType: models.EntityTypeFeature, startLeg: cascadeLegEpic, epicID: epic.ID,
		}); err != nil {
			return nil, fmt.Errorf("failed to maintain epic aggregate: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("failed to commit feature aggregate transaction: %w", err)
		}
	} else if err := s.repo.Create(ctx, feature); err != nil {
		return nil, fmt.Errorf("failed to create feature %s: %w", featureKey, err)
	}

	if err := attachTagsIfAny(ctx, s.tagSvc, models.EntityTypeFeature, feature.ID, input.Tags); err != nil {
		return nil, err
	}
	if err := indexEntityIfConfigured(ctx, s.searchIndexer, models.EntityTypeFeature, feature.ID); err != nil {
		return nil, err
	}

	if s.aggregateCoordinator == nil {
		s.maybeReopenParentEpic(ctx, epic, feature.Key)
	}
	return feature, nil
}

// maybeReopenParentEpic checks if the parent epic is in a terminal status
// and reopens it using the cascade helper (history-based target resolution).
// When cascade deps are not wired (e.g., tests without full wiring), this is
// a no-op — the cascade helper guards itself via cascadeEnabled.
//
// Best-effort: errors are logged via slog.Warn, never propagated to the caller.
//
// Parameters:
//   - ctx: context for cancellation
//   - epic: the parent epic model (already retrieved during feature creation)
//   - featureKey: key of the newly created feature (for audit logging)
func (s *FeatureService) maybeReopenParentEpic(ctx context.Context, epic *models.Epic, featureKey string) {
	if !s.cascadeEnabled() {
		return
	}
	cascadeParentReopens(ctx, s.cascadeDepsBundle(), cascadeTrigger{
		triggerKey:  featureKey,
		triggerKind: "creation",
		triggerType: models.EntityTypeFeature,
		startLeg:    cascadeLegEpic,
		epicID:      epic.ID,
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
	previousStatus := feature.Status

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

	// Three-branch Size update logic (E07-F42 AC-T1).
	if updates.ClearSize {
		feature.Size = nil
	} else if updates.Size != nil {
		feature.Size = updates.Size
	}
	// else: leave feature.Size unchanged (no-op)

	if err := feature.Validate(); err != nil {
		return nil, fmt.Errorf("feature validation failed: %w", err)
	}

	// file_path is included in the Update query — single atomic operation
	if updates.FilePath != nil {
		feature.FilePath = updates.FilePath
	}

	var saveErr error
	if s.aggregateCoordinator != nil && updates.Status != nil {
		txRepo, ok := s.repo.(featureAggregateTxRepository)
		if !ok {
			return nil, fmt.Errorf("feature repository does not support aggregate transactions")
		}
		tx, err := txRepo.BeginTx(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to begin feature update transaction: %w", err)
		}
		defer rollbackAfterAggregateMutation(tx)
		if err := txRepo.UpdateWithTx(ctx, tx, feature, updates.SkipResequence); err != nil {
			return nil, fmt.Errorf("failed to update feature %s: %w", key, err)
		}
		triggerKind := "status_update"
		featureWf := s.entitySvc.GetWorkflowService().ForLevel(workflow.LevelFeature)
		if featureWf.IsTerminalStatus(string(previousStatus)) && !featureWf.IsTerminalStatus(string(feature.Status)) {
			triggerKind = "regression"
		}
		if err := s.aggregateCoordinator.RefreshEpicStatus(ctx, tx, feature.EpicID, cascadeTrigger{
			triggerKey: feature.Key, triggerKind: triggerKind, triggerType: models.EntityTypeFeature, startLeg: cascadeLegEpic, epicID: feature.EpicID,
		}); err != nil {
			return nil, fmt.Errorf("failed to maintain feature aggregates: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("failed to commit feature update: %w", err)
		}
	} else if updates.SkipResequence {
		saveErr = s.repo.UpdateNoResequence(ctx, feature)
	} else {
		saveErr = s.repo.Update(ctx, feature)
	}
	if saveErr != nil {
		return nil, fmt.Errorf("failed to update feature %s: %w", key, saveErr)
	}

	// `--tag` on update is additive only; detach goes through `shark feature tag rm`.
	if err := attachTagsIfAny(ctx, s.tagSvc, models.EntityTypeFeature, feature.ID, updates.Tags); err != nil {
		return nil, err
	}
	if err := indexEntityIfConfigured(ctx, s.searchIndexer, models.EntityTypeFeature, feature.ID); err != nil {
		return nil, err
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

	if s.aggregateCoordinator != nil {
		txRepo, ok := s.repo.(featureAggregateTxRepository)
		if !ok {
			return fmt.Errorf("feature repository does not support aggregate transactions")
		}
		tx, err := txRepo.BeginTx(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin feature aggregate transaction: %w", err)
		}
		defer rollbackAfterAggregateMutation(tx)
		if err := txRepo.DeleteWithTx(ctx, tx, feature.ID); err != nil {
			return fmt.Errorf("failed to delete feature %s: %w", key, err)
		}
		if err := s.aggregateCoordinator.RefreshEpicStatus(ctx, tx, feature.EpicID, cascadeTrigger{
			triggerKey: feature.Key, triggerKind: "deletion", triggerType: models.EntityTypeFeature, startLeg: cascadeLegEpic, epicID: feature.EpicID,
		}); err != nil {
			return fmt.Errorf("failed to maintain epic aggregate: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit feature aggregate transaction: %w", err)
		}
	} else if err := s.repo.Delete(ctx, feature.ID); err != nil {
		return fmt.Errorf("failed to delete feature %s: %w", key, err)
	}
	if err := removeEntityFromIndexIfConfigured(ctx, s.searchIndexer, models.EntityTypeFeature, feature.ID); err != nil {
		return err
	}

	return nil
}

// resolveFeatureCustomKey normalizes a caller-supplied custom feature key and
// verifies it names the epic the feature is actually being created under
// (B063 BLOCKER-1). A bare suffix ("F07") is derived against epicKey; a full
// key ("E01-F07") naming a *different* epic is rejected rather than silently
// squatting a globally-unique features.key row that the named epic's own
// auto-generation will collide with later.
func resolveFeatureCustomKey(customKey, epicKey string) (string, error) {
	trimmed := strings.ToUpper(strings.TrimSpace(customKey))

	if keys.IsFeatureKeySuffix(trimmed) {
		return epicKey + "-" + trimmed, nil
	}

	if keys.IsFeatureKey(trimmed) {
		parentEpic, _, err := keys.ParseFeatureKey(trimmed)
		if err != nil {
			return "", fmt.Errorf("invalid feature key %q: %w", customKey, err)
		}
		if parentEpic != epicKey {
			return "", fmt.Errorf("feature key %q does not belong to epic %q", trimmed, epicKey)
		}
		return trimmed, nil
	}

	return "", fmt.Errorf("invalid feature key %q: expected format F## or %s-F## (e.g. F07 or %s-F07)", customKey, epicKey, epicKey)
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

// suggestNextFeatureKey returns the next available feature key for use in
// duplicate-key error messages, or the empty string if it could not be
// computed. Best-effort: a failure here must not mask the original
// duplicate-key error (B063).
func (s *FeatureService) suggestNextFeatureKey(ctx context.Context, epicID int64, epicKey string) string {
	next, err := s.nextFeatureKey(ctx, epicID, epicKey)
	if err != nil {
		return ""
	}
	return next
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

// ============================================================
// featureCascadeReadAdapter — adapts FeatureRepository to CascadeFeatureRepo
// ============================================================

// featureCascadeReadAdapter wraps FeatureRepository and satisfies CascadeFeatureRepo.
// When a FeatureService cascade starts at cascadeLegEpic, the cascade helper still
// calls featureRepo.GetByID once to obtain feature.EpicID, but never calls
// GetByIDTx or UpdateStatusTx on the feature itself (those are only reachable when
// featureNeedsReopen is true, which requires startLeg == cascadeLegFeature).
// The Tx stubs are therefore unreachable in practice; they exist only to satisfy the
// CascadeFeatureRepo interface.
type featureCascadeReadAdapter struct {
	repo FeatureRepository
}

func (a *featureCascadeReadAdapter) GetByID(ctx context.Context, id int64) (*models.Feature, error) {
	return a.repo.GetByID(ctx, id)
}

// GetByIDTx is unreachable for cascadeLegEpic triggers but is required by the
// CascadeFeatureRepo interface. Delegates to the non-Tx variant.
func (a *featureCascadeReadAdapter) GetByIDTx(ctx context.Context, _ *sql.Tx, id int64) (*models.Feature, error) {
	return a.repo.GetByID(ctx, id)
}

// UpdateStatusTx is unreachable for cascadeLegEpic triggers but is required by the
// CascadeFeatureRepo interface. Returns an error to make any accidental call visible.
func (a *featureCascadeReadAdapter) UpdateStatusTx(_ context.Context, _ *sql.Tx, _ int64, _ string, _ *string, _ *string) error {
	return fmt.Errorf("featureCascadeReadAdapter.UpdateStatusTx: unreachable for cascadeLegEpic triggers")
}
