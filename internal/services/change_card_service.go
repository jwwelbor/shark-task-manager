package services

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/fileops"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/utils"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ChangeCardRepository defines the data access interface needed by ChangeCardService.
// This interface is satisfied by *repository.ChangeCardRepository.
type ChangeCardRepository interface {
	Create(ctx context.Context, card *models.ChangeCard) error
	GetByKey(ctx context.Context, key string) (*models.ChangeCard, error)
	GetByID(ctx context.Context, id int64) (*models.ChangeCard, error)
	Update(ctx context.Context, card *models.ChangeCard) error
	Delete(ctx context.Context, id int64) error
	UpdateStatus(ctx context.Context, id int64, status models.ChangeCardStatus) error
	List(ctx context.Context, filter *repository.ChangeCardRepoFilter) ([]*models.ChangeCard, error)
	ListByEpic(ctx context.Context, epicID int64) ([]*models.ChangeCard, error)
	ListByFeature(ctx context.Context, featureID int64) ([]*models.ChangeCard, error)
	CountByStatus(ctx context.Context) (map[string]int, error)
	GetNextKey(ctx context.Context) (string, error)
}

// (ChangeCardWritableDocumentRepository removed -- replaced by EntityDocumentRepository + EntityDocumentLinkRepository)

// ChangeCardService provides business logic for change-card operations.
type ChangeCardService struct {
	repo        ChangeCardRepository
	workflowSvc *workflow.Service
	entitySvc   *EntityService
	entityRepo  EntityRepository
	epicRepo    EpicRepository
	featureRepo FeatureRepository
	projectRoot string
	docSvc      *EntityDocumentService // shared document operations; built by SetWritableDocRepo
	// tagSvc is optional — nil disables tag integration.
	// TagQuerier extends TagAttacher with EntityIDsByTags for list filtering (F05).
	tagSvc TagQuerier

	// sizeCfg is optional — nil disables size enforcement on create.
	sizeCfg SizeEnforcementConfig
}

// SetTagService wires the optional TagQuerier dependency. When nil, tag
// hooks in CreateChangeCard and UpdateChangeCard are skipped silently.
func (s *ChangeCardService) SetTagService(tagSvc TagQuerier) {
	s.tagSvc = tagSvc
}

// SetSizeEnforcement wires the optional SizeEnforcementConfig. When nil or
// when the config does not list "change" in SizeRequiredFor, CreateChangeCard
// accepts nil Size silently.
func (s *ChangeCardService) SetSizeEnforcement(cfg SizeEnforcementConfig) {
	s.sizeCfg = cfg
}

// NewChangeCardService creates a new ChangeCardService.
// entitySvc and entityRepo are required for status transition delegation.
//
// Panics:
//   - If repo is nil (required dependency)
//   - If entitySvc is nil (required dependency)
func NewChangeCardService(
	repo ChangeCardRepository,
	entitySvc *EntityService,
	entityRepo EntityRepository,
	epicRepo EpicRepository,
	featureRepo FeatureRepository,
	projectRoot string,
) *ChangeCardService {
	requireNonNil(repo, "ChangeCardService requires a non-nil ChangeCardRepository")
	requireNonNil(entitySvc, "ChangeCardService requires a non-nil EntityService")
	return &ChangeCardService{
		repo:        repo,
		workflowSvc: entitySvc.GetWorkflowService().ForLevel(workflow.LevelChange),
		entitySvc:   entitySvc.ForLevel(workflow.LevelChange),
		entityRepo:  entityRepo,
		epicRepo:    epicRepo,
		featureRepo: featureRepo,
		projectRoot: projectRoot,
	}
}

// CreateChangeCard creates a new change-card with optional entity linking.
//
// Returns the created card, a boolean indicating whether an existing markdown
// file was linked (vs. a fresh placeholder being written), and any error.
func (s *ChangeCardService) CreateChangeCard(ctx context.Context, input CreateChangeCardInput) (*models.ChangeCard, bool, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, false, fmt.Errorf("change-card title cannot be empty")
	}

	// Resolve epic link
	var epicID *int64
	if input.EpicKey != "" && s.epicRepo != nil {
		epic, err := s.epicRepo.GetByKey(ctx, input.EpicKey)
		if err != nil {
			return nil, false, fmt.Errorf("epic %s not found: %w", input.EpicKey, err)
		}
		epicID = &epic.ID
	}

	// Resolve feature link
	var featureID *int64
	if input.FeatureKey != "" && s.featureRepo != nil {
		feature, err := s.featureRepo.GetByKey(ctx, input.FeatureKey)
		if err != nil {
			return nil, false, fmt.Errorf("feature %s not found: %w", input.FeatureKey, err)
		}
		featureID = &feature.ID
	}

	if err := enforceSizeRequired(s.sizeCfg, models.EntityTypeChange, input.Size); err != nil {
		return nil, false, err
	}
	if err := enforceTagsRequired(ctx, s.tagSvc, models.EntityTypeChange, input.Tags); err != nil {
		return nil, false, err
	}

	// Generate next key
	nextKey, err := s.repo.GetNextKey(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("failed to generate change-card key: %w", err)
	}

	// Generate slug
	slugVal := utils.GenerateSlug(title)

	// Get default status from workflow
	defaultStatus := s.workflowSvc.GetDefaultStatus()

	// Default priority
	priority := input.Priority
	if priority == 0 {
		priority = 5
	}

	// Build model
	card := &models.ChangeCard{BaseEntity: models.BaseEntity{Key: nextKey,
		Title: title,
		Slug:  &slugVal,
		Size:  input.Size}, Status: models.ChangeCardStatus(defaultStatus),
		Priority: priority,

		EpicID:    epicID,
		FeatureID: featureID,
	}

	if input.Description != "" {
		desc := input.Description
		card.Description = &desc
	}
	if input.RequestedBy != "" {
		rb := input.RequestedBy
		card.RequestedBy = &rb
	}
	if input.Justification != "" {
		j := input.Justification
		card.Justification = &j
	}

	if err := card.Validate(); err != nil {
		return nil, false, fmt.Errorf("validation failed: %w", err)
	}

	// Set file path
	filePath := filepath.Join("docs", "plan", "changes", nextKey+".md")
	card.FilePath = &filePath

	// Create in database
	if err := s.repo.Create(ctx, card); err != nil {
		return nil, false, fmt.Errorf("failed to create change-card: %w", err)
	}

	if err := attachTagsIfAny(ctx, s.tagSvc, models.EntityTypeChange, card.ID, input.Tags); err != nil {
		return nil, false, err
	}

	// Generate and write markdown file (best-effort)
	content := s.generateMarkdown(card)
	if input.Body != "" {
		content = fileops.ReplaceBodyAfterFrontmatter(content, input.Body)
	}
	writer := fileops.NewEntityFileWriter()
	writeResult, writeErr := writer.WriteEntityFile(fileops.WriteOptions{
		Content:        []byte(content),
		ProjectRoot:    s.projectRoot,
		FilePath:       filePath,
		EntityType:     "change",
		UseAtomicWrite: true,
	})
	if writeErr != nil {
		// Log warning but don't fail -- DB record is the source of truth
		slog.Warn("failed to write change-card file", "path", filePath, "error", writeErr)
	}

	fileWasLinked := writeResult != nil && writeResult.Linked
	return card, fileWasLinked, nil
}

// GetChangeCard retrieves a change-card by key.
func (s *ChangeCardService) GetChangeCard(ctx context.Context, key string) (*models.ChangeCard, error) {
	card, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get change-card %s: %w", key, err)
	}
	return card, nil
}

// GetChangeCardWithTags returns the change-card and the sorted list of tag
// names attached to it. When tagSvc is nil the tags slice is nil (graceful
// degradation — consistent with F04 REQ-F-018). When ListTagsForEntity fails
// the method returns (nil, nil, wrappedErr) per AC-T3.
func (s *ChangeCardService) GetChangeCardWithTags(ctx context.Context, key string) (*models.ChangeCard, []string, error) {
	ctx, span := otel.Tracer("shark/services/change_card").Start(ctx, "ChangeCardService.GetChangeCardWithTags",
		trace.WithAttributes(attribute.String("change_card.key", key)),
	)
	defer span.End()

	card, err := s.GetChangeCard(ctx, key)
	if err != nil {
		return nil, nil, err
	}
	if s.tagSvc == nil {
		return card, nil, nil
	}
	names, err := s.tagSvc.ListTagsForEntity(ctx, models.EntityTypeChange, card.ID)
	if err != nil {
		return nil, nil, recordSpanError(span, fmt.Errorf("load tags for change-card %s: %w", key, err))
	}
	return card, names, nil
}

// ListChangeCards retrieves change-cards with optional filtering.
func (s *ChangeCardService) ListChangeCards(ctx context.Context, filters ChangeCardFilters) ([]*models.ChangeCard, error) {
	// Block 1: pre-filter by tag IDs (E28-F05 §2.5.2).
	var taggedIDSet map[int64]struct{}
	if len(filters.Tags) > 0 {
		if s.tagSvc == nil {
			return nil, &TagFilterUnavailableError{}
		}
		ids, err := s.tagSvc.EntityIDsByTags(ctx, models.EntityTypeChange, filters.Tags, TagQueryOpAnd)
		if err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			return []*models.ChangeCard{}, nil // REQ-F-017 short-circuit
		}
		taggedIDSet = make(map[int64]struct{}, len(ids))
		for _, id := range ids {
			taggedIDSet[id] = struct{}{}
		}
	}

	repoFilter := &repository.ChangeCardRepoFilter{
		IncludeTerminal:  filters.ShowAll,
		TerminalStatuses: s.workflowSvc.GetTerminalStatuses(),
	}

	if filters.Status != "" {
		status := models.ChangeCardStatus(filters.Status)
		repoFilter.Status = &status
	}

	// Resolve epic key to ID for filtering
	if filters.EpicKey != "" && s.epicRepo != nil {
		epic, err := s.epicRepo.GetByKey(ctx, filters.EpicKey)
		if err != nil {
			return nil, fmt.Errorf("epic %s not found: %w", filters.EpicKey, err)
		}
		repoFilter.EpicID = &epic.ID
	}

	// Resolve feature key to ID for filtering
	if filters.FeatureKey != "" && s.featureRepo != nil {
		feature, err := s.featureRepo.GetByKey(ctx, filters.FeatureKey)
		if err != nil {
			return nil, fmt.Errorf("feature %s not found: %w", filters.FeatureKey, err)
		}
		repoFilter.FeatureID = &feature.ID
	}

	cards, err := s.repo.List(ctx, repoFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to list change-cards: %w", err)
	}

	if cards == nil {
		cards = []*models.ChangeCard{}
	}

	// Block 2: post-filter in-memory (E28-F05 §2.5.2).
	cards = filterByTagIDs(cards, taggedIDSet, func(c *models.ChangeCard) int64 { return c.ID })

	return cards, nil
}

// UpdateChangeCard updates a change-card with the provided updates.
func (s *ChangeCardService) UpdateChangeCard(ctx context.Context, key string, updates ChangeCardUpdates) (*models.ChangeCard, error) {
	card, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get change-card %s: %w", key, err)
	}

	// Apply non-nil updates
	if updates.Title != nil {
		card.Title = strings.TrimSpace(*updates.Title)
		newSlug := utils.GenerateSlug(card.Title)
		card.Slug = &newSlug
	}
	if updates.Description != nil {
		card.Description = updates.Description
	}
	if updates.Priority != nil {
		card.Priority = *updates.Priority
	}
	if updates.RequestedBy != nil {
		card.RequestedBy = updates.RequestedBy
	}
	if updates.AssignedTo != nil {
		card.AssignedTo = updates.AssignedTo
	}
	if updates.Justification != nil {
		card.Justification = updates.Justification
	}
	if updates.ImpactAnalysis != nil {
		card.ImpactAnalysis = updates.ImpactAnalysis
	}
	if updates.RollbackPlan != nil {
		card.RollbackPlan = updates.RollbackPlan
	}
	if updates.FilePath != nil {
		card.FilePath = updates.FilePath
	}

	// Three-branch Size update logic (E07-F42 AC-T1).
	if updates.ClearSize {
		card.Size = nil
	} else if updates.Size != nil {
		card.Size = updates.Size
	}
	// else: leave card.Size unchanged (no-op)

	if err := card.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if err := s.repo.Update(ctx, card); err != nil {
		return nil, fmt.Errorf("failed to update change-card %s: %w", key, err)
	}

	// `--tag` on update is additive only; detach goes through `shark change tag rm`.
	if err := attachTagsIfAny(ctx, s.tagSvc, models.EntityTypeChange, card.ID, updates.Tags); err != nil {
		return nil, err
	}

	return card, nil
}

// DeleteChangeCard deletes a change-card by key.
func (s *ChangeCardService) DeleteChangeCard(ctx context.Context, key string) error {
	card, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get change-card %s: %w", key, err)
	}

	if err := s.repo.Delete(ctx, card.ID); err != nil {
		return fmt.Errorf("failed to delete change-card %s: %w", key, err)
	}

	// Best-effort file deletion
	if card.FilePath != nil && *card.FilePath != "" && s.projectRoot != "" {
		absPath := filepath.Join(s.projectRoot, *card.FilePath)
		if removeErr := os.Remove(absPath); removeErr != nil && !os.IsNotExist(removeErr) {
			slog.Warn("failed to delete change-card file", "path", absPath, "error", removeErr)
		}
	}

	return nil
}

// TransitionStatus transitions a change-card to a specific status with workflow validation.
// Delegates to EntityService.TransitionStatus for shared transition logic.
func (s *ChangeCardService) TransitionStatus(ctx context.Context, key string, targetStatus string, opts TransitionOptions) (*TransitionResult, error) {
	return s.entitySvc.TransitionStatus(
		ctx, s.entityRepo, models.EntityTypeChange, key, targetStatus, opts,
		SimpleTransitionFeatures(),
		s.makeResolveActionFn(),
	)
}

// GetNextStatus returns the available transitions for the current status of a change-card.
func (s *ChangeCardService) GetNextStatus(ctx context.Context, key string) (*NextStatusInfo, error) {
	return s.entitySvc.GetNextStatus(
		ctx, s.entityRepo, models.EntityTypeChange, key,
		s.makeResolveActionFn(),
	)
}

// GetNextStatusForCard returns available transitions for a pre-fetched change-card, avoiding a DB re-fetch.
func (s *ChangeCardService) GetNextStatusForCard(card *models.ChangeCard) *NextStatusInfo {
	return s.entitySvc.GetNextStatusForEntity(
		models.EntityTypeChange, card.Key, card,
		s.makeResolveActionFn(),
	)
}

// CountByStatus returns counts of change-cards grouped by status.
func (s *ChangeCardService) CountByStatus(ctx context.Context) (map[string]int, error) {
	return s.repo.CountByStatus(ctx)
}

// generateMarkdown generates the markdown content for a change-card file.
func (s *ChangeCardService) generateMarkdown(card *models.ChangeCard) string {
	var sb strings.Builder

	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("change_card_key: %s\n", card.Key))
	sb.WriteString(fmt.Sprintf("title: %s\n", card.Title))
	sb.WriteString(fmt.Sprintf("status: %s\n", card.Status))
	sb.WriteString(fmt.Sprintf("priority: %d\n", card.Priority))
	if card.Slug != nil {
		sb.WriteString(fmt.Sprintf("slug: %s\n", *card.Slug))
	}
	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("# %s\n\n", card.Title))
	sb.WriteString("## Description\n\n")

	if card.Description != nil {
		sb.WriteString(*card.Description + "\n\n")
	} else {
		sb.WriteString("[Describe the proposed change]\n\n")
	}

	sb.WriteString("## Justification\n\n")
	if card.Justification != nil {
		sb.WriteString(*card.Justification + "\n\n")
	} else {
		sb.WriteString("[Why is this change needed?]\n\n")
	}

	sb.WriteString("## Impact Analysis\n\n")
	sb.WriteString("[Describe impact of this change]\n\n")
	sb.WriteString("## Rollback Plan\n\n")
	sb.WriteString("[Describe how to revert this change if needed]\n")

	return sb.String()
}

// makeResolveActionFn returns a ResolveActionFn callback that generates
// ChangeCard-specific placeholders for orchestrator action resolution.
func (s *ChangeCardService) makeResolveActionFn() ResolveActionFn {
	return func(entity models.Entity, status string) *config.PopulatedAction {
		card, ok := entity.(*models.ChangeCard)
		if !ok {
			return nil
		}
		placeholders := config.ChangeCardPlaceholders(card)
		// Fresh transition context: suppress RESUME CONTEXT preamble in templates.
		// is_resume="true" is reserved for shark get (GetOrchestratorAction).
		placeholders["is_resume"] = "false"
		return s.entitySvc.ResolveActionForStatus(status, placeholders)
	}
}

// GetOrchestratorAction returns the orchestrator action for the change-card's current status.
// Used by shark get — entity is already in this status, so RESUME CONTEXT preamble is shown.
func (s *ChangeCardService) GetOrchestratorAction(card *models.ChangeCard) *config.PopulatedAction {
	placeholders := config.ChangeCardPlaceholders(card)
	placeholders["is_resume"] = "true"
	return s.entitySvc.ResolveActionForStatus(string(card.Status), placeholders)
}

// SetWritableDocRepo sets the writable document repository on the service.
// This enables LinkDocument, UnlinkDocument, and ListRelatedDocumentsByKey operations on change-cards.
func (s *ChangeCardService) SetWritableDocRepo(writableRepo EntityDocumentRepository, linkRepo EntityDocumentLinkRepository) {
	s.docSvc = NewEntityDocumentService(
		writableRepo,
		linkRepo,
		EntityLookupFnFromRepo(s.entityRepo),
	)
}

// LinkDocument links a document to a change-card identified by key.
// Delegates to the shared EntityDocumentService.
func (s *ChangeCardService) LinkDocument(ctx context.Context, cardKey, docTitle, docPath string) error {
	if s.docSvc == nil {
		return fmt.Errorf("writable document repository not configured")
	}
	_, err := s.docSvc.LinkDocumentByKey(ctx, cardKey, docTitle, docPath)
	return err
}

// UnlinkDocument removes a document link from a change-card by document title.
// Delegates to the shared EntityDocumentService.
func (s *ChangeCardService) UnlinkDocument(ctx context.Context, cardKey, docTitle string) error {
	if s.docSvc == nil {
		return fmt.Errorf("writable document repository not configured")
	}
	return s.docSvc.UnlinkDocumentByKey(ctx, cardKey, docTitle)
}

// ListRelatedDocumentsByKey returns all documents linked to a change-card identified by key.
// Delegates to the shared EntityDocumentService.
func (s *ChangeCardService) ListRelatedDocumentsByKey(ctx context.Context, cardKey string) ([]*models.Document, error) {
	if s.docSvc == nil {
		return []*models.Document{}, nil
	}
	return s.docSvc.ListDocumentsByKey(ctx, cardKey)
}
