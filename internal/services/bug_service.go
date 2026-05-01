package services

import (
	"context"
	"fmt"
	"log/slog"
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

// BugRepository defines the repository interface for bug operations.
type BugRepository interface {
	Create(ctx context.Context, bug *models.Bug) error
	GetByKey(ctx context.Context, key string) (*models.Bug, error)
	GetByID(ctx context.Context, id int64) (*models.Bug, error)
	Update(ctx context.Context, bug *models.Bug) error
	Delete(ctx context.Context, id int64) error
	UpdateStatus(ctx context.Context, id int64, status models.BugStatus) error
	GetNextKey(ctx context.Context) (string, error)
	List(ctx context.Context, filters *repository.BugListFilters) ([]*models.Bug, error)
	CountByStatus(ctx context.Context) (map[string]int, error)
	CountBySeverity(ctx context.Context) (map[string]int, error)
}

// LinkValidatorEpicRepo defines the interface for validating epic links.
type LinkValidatorEpicRepo interface {
	GetByKey(ctx context.Context, key string) (*models.Epic, error)
}

// LinkValidatorFeatureRepo defines the interface for validating feature links.
type LinkValidatorFeatureRepo interface {
	GetByKey(ctx context.Context, key string) (*models.Feature, error)
}

// LinkValidatorTaskRepo defines the interface for validating task links.
type LinkValidatorTaskRepo interface {
	GetByKey(ctx context.Context, key string) (*models.Task, error)
}

// TagAttacher is the narrow interface that entity services depend on for
// tag-related operations. Satisfied by *TagService in production and by
// test mocks.
//
// The fourth method, ListTagsForEntity, is used by GetXxxWithTags wrappers
// (REQ-F-014, spec §2.5.3) so a single interface covers both the
// Create/Update attach path and the Get-with-tags query path.
type TagAttacher interface {
	EnforceRequired(ctx context.Context, entityType models.EntityType, names []string) error
	AttachMany(ctx context.Context, entityType models.EntityType, entityID int64, names []string) error
	DetachOne(ctx context.Context, entityType models.EntityType, entityID int64, name string) error

	// ListTagsForEntity returns the sorted normalized tag names attached to
	// (entityType, entityID). Returns an empty non-nil slice when no tags
	// are attached. Used by GetXxxWithTags (spec §2.5.3, REQ-F-014).
	ListTagsForEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]string, error)
}

// Compile-time check: *TagService must satisfy TagAttacher (AC-T2).
var _ TagAttacher = (*TagService)(nil)

// (BugWritableDocumentRepository removed -- replaced by EntityDocumentRepository + EntityDocumentLinkRepository)

// BugService provides business logic for bug operations.
type BugService struct {
	repo        BugRepository
	workflowSvc *workflow.Service
	entitySvc   *EntityService
	entityRepo  EntityRepository
	epicRepo    LinkValidatorEpicRepo
	featureRepo LinkValidatorFeatureRepo
	taskRepo    LinkValidatorTaskRepo
	projectRoot string
	docSvc      *EntityDocumentService // shared document operations; built by SetWritableDocRepo
	// tagSvc is optional — nil disables tag integration.
	// TagQuerier extends TagAttacher with EntityIDsByTags for list filtering (F05).
	tagSvc TagQuerier
}

// NewBugService creates a BugService. tagSvc is optional (pass nil to
// disable tag integration). Panics if repo or entitySvc is nil.
func NewBugService(
	repo BugRepository,
	entitySvc *EntityService,
	entityRepo EntityRepository,
	epicRepo LinkValidatorEpicRepo,
	featureRepo LinkValidatorFeatureRepo,
	taskRepo LinkValidatorTaskRepo,
	projectRoot string,
	tagSvc TagQuerier,
) *BugService {
	requireNonNil(repo, "BugService requires a non-nil BugRepository")
	requireNonNil(entitySvc, "BugService requires a non-nil EntityService")
	return &BugService{
		repo:        repo,
		workflowSvc: entitySvc.GetWorkflowService().ForLevel(workflow.LevelBug),
		entitySvc:   entitySvc.ForLevel(workflow.LevelBug),
		entityRepo:  entityRepo,
		epicRepo:    epicRepo,
		featureRepo: featureRepo,
		taskRepo:    taskRepo,
		projectRoot: projectRoot,
		tagSvc:      tagSvc,
	}
}

// CreateBug creates a new bug with auto-generated key and slug.
//
// Returns the created bug, a boolean indicating whether an existing markdown
// file was linked (vs. a fresh placeholder being written), and any error.
func (s *BugService) CreateBug(ctx context.Context, input CreateBugInput) (*models.Bug, bool, error) {
	if strings.TrimSpace(input.Title) == "" {
		return nil, false, fmt.Errorf("bug title cannot be empty")
	}

	if !models.ValidBugSeverities[input.Severity] {
		return nil, false, fmt.Errorf("invalid severity %q: must be one of critical, high, medium, low", input.Severity)
	}

	// Validate linked entity if provided
	if input.LinkedEntityType != "" || input.LinkedEntityKey != "" {
		if input.LinkedEntityType == "" || input.LinkedEntityKey == "" {
			return nil, false, fmt.Errorf("both linked_entity_type and linked_entity_key must be provided together")
		}
		if err := s.validateLinkedEntity(ctx, input.LinkedEntityType, input.LinkedEntityKey); err != nil {
			return nil, false, fmt.Errorf("linked entity validation failed: %w", err)
		}
	}

	// Enforce tag_required_for before key allocation or persistence.
	if err := enforceTagsRequired(ctx, s.tagSvc, models.EntityTypeBug, input.Tags); err != nil {
		return nil, false, err
	}

	// Generate key
	key, err := s.repo.GetNextKey(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("failed to generate bug key: %w", err)
	}

	// Get default status from workflow
	defaultStatus := s.workflowSvc.GetDefaultStatus()

	// Generate slug
	slug := utils.GenerateSlug(input.Title)

	bug := &models.Bug{BaseEntity: models.BaseEntity{Key: key,
		Title: input.Title,
		Slug:  &slug,
		Size:  input.Size}, Status: models.BugStatus(defaultStatus),
		Severity: input.Severity,
	}

	if input.Description != "" {
		bug.Description = &input.Description
	}

	if input.LinkedEntityType != "" {
		bug.LinkedEntityType = &input.LinkedEntityType
		bug.LinkedEntityKey = &input.LinkedEntityKey
	}

	// Resolve file path: use caller-supplied path or compute default
	var filePath string
	if input.FilePath != nil && *input.FilePath != "" {
		filePath = *input.FilePath
	} else {
		filePath = filepath.Join("docs", "plan", "bugs", key+".md")
	}
	bug.FilePath = &filePath

	if err := s.repo.Create(ctx, bug); err != nil {
		return nil, false, fmt.Errorf("failed to create bug: %w", err)
	}

	// Attach tags after insert so bug.ID is valid. Not wrapped in a
	// transaction — on failure the row is persisted with zero tags and
	// the user retries via `shark bug update --tag=...`.
	if err := attachTagsIfAny(ctx, s.tagSvc, models.EntityTypeBug, bug.ID, input.Tags); err != nil {
		return nil, false, err
	}

	// Generate and write markdown file (best-effort)
	content := s.generateMarkdown(bug)
	if input.Body != "" {
		content = fileops.ReplaceBodyAfterFrontmatter(content, input.Body)
	}
	writer := fileops.NewEntityFileWriter()
	writeResult, writeErr := writer.WriteEntityFile(fileops.WriteOptions{
		Content:        []byte(content),
		ProjectRoot:    s.projectRoot,
		FilePath:       filePath,
		EntityType:     "bug",
		UseAtomicWrite: !input.Force,
		Force:          input.Force,
	})
	if writeErr != nil {
		// Log warning but don't fail -- DB record is the source of truth
		slog.Warn("failed to write bug file", "path", filePath, "error", writeErr)
	}

	fileWasLinked := writeResult != nil && writeResult.Linked
	return bug, fileWasLinked, nil
}

// generateMarkdown produces a markdown document for a newly created bug.
func (s *BugService) generateMarkdown(bug *models.Bug) string {
	var sb strings.Builder

	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("bug_key: %s\n", bug.Key))
	sb.WriteString(fmt.Sprintf("title: %s\n", bug.Title))
	sb.WriteString(fmt.Sprintf("status: %s\n", bug.Status))
	sb.WriteString(fmt.Sprintf("severity: %s\n", bug.Severity))
	if bug.Slug != nil {
		sb.WriteString(fmt.Sprintf("slug: %s\n", *bug.Slug))
	}
	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("# %s\n\n", bug.Title))
	sb.WriteString("## Description\n\n")

	if bug.Description != nil {
		sb.WriteString(*bug.Description + "\n\n")
	} else {
		sb.WriteString("[Describe the bug and how to reproduce it]\n\n")
	}

	sb.WriteString("## Expected Behavior\n\n")
	sb.WriteString("[What should happen?]\n\n")
	sb.WriteString("## Actual Behavior\n\n")
	sb.WriteString("[What actually happens?]\n\n")
	sb.WriteString("## Steps to Reproduce\n\n")
	sb.WriteString("1. \n\n")
	sb.WriteString("## Fix Notes\n\n")
	sb.WriteString("[Notes on the fix once resolved]\n")

	return sb.String()
}

// GetBug retrieves a bug by its key.
func (s *BugService) GetBug(ctx context.Context, key string) (*models.Bug, error) {
	bug, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get bug %s: %w", key, err)
	}
	return bug, nil
}

// GetBugWithTags returns the bug and the sorted list of tag names attached to
// it. When tagSvc is nil the tags slice is nil (graceful degradation —
// consistent with F04 REQ-F-018). When ListTagsForEntity fails the method
// returns (nil, nil, wrappedErr) per AC-T3.
func (s *BugService) GetBugWithTags(ctx context.Context, key string) (*models.Bug, []string, error) {
	ctx, span := otel.Tracer("shark/services/bug").Start(ctx, "BugService.GetBugWithTags",
		trace.WithAttributes(attribute.String("bug.key", key)),
	)
	defer span.End()

	bug, err := s.GetBug(ctx, key)
	if err != nil {
		return nil, nil, err
	}
	if s.tagSvc == nil {
		return bug, nil, nil
	}
	names, err := s.tagSvc.ListTagsForEntity(ctx, models.EntityTypeBug, bug.ID)
	if err != nil {
		return nil, nil, recordSpanError(span, fmt.Errorf("load tags for bug %s: %w", key, err))
	}
	return bug, names, nil
}

// UpdateBug applies partial updates to a bug.
func (s *BugService) UpdateBug(ctx context.Context, key string, updates BugUpdates) (*models.Bug, error) {
	bug, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get bug %s: %w", key, err)
	}

	if updates.Title != nil {
		if strings.TrimSpace(*updates.Title) == "" {
			return nil, fmt.Errorf("bug title cannot be empty")
		}
		bug.Title = *updates.Title
		slug := utils.GenerateSlug(bug.Title)
		bug.Slug = &slug
	}

	if updates.Description != nil {
		bug.Description = updates.Description
	}

	if updates.Severity != nil {
		if !models.ValidBugSeverities[*updates.Severity] {
			return nil, fmt.Errorf("invalid severity %q: must be one of critical, high, medium, low", *updates.Severity)
		}
		bug.Severity = *updates.Severity
	}

	if updates.FilePath != nil {
		bug.FilePath = updates.FilePath
	}

	// Three-branch Size update logic (E07-F42 AC-T1).
	if updates.ClearSize {
		bug.Size = nil
	} else if updates.Size != nil {
		bug.Size = updates.Size
	}
	// else: leave bug.Size unchanged (no-op)

	if updates.LinkedEntityType != nil || updates.LinkedEntityKey != nil {
		entityType := ""
		entityKey := ""
		if updates.LinkedEntityType != nil {
			entityType = *updates.LinkedEntityType
		}
		if updates.LinkedEntityKey != nil {
			entityKey = *updates.LinkedEntityKey
		}

		if entityType != "" || entityKey != "" {
			if entityType == "" || entityKey == "" {
				return nil, fmt.Errorf("both linked_entity_type and linked_entity_key must be provided together")
			}
			if err := s.validateLinkedEntity(ctx, entityType, entityKey); err != nil {
				return nil, fmt.Errorf("linked entity validation failed: %w", err)
			}
			bug.LinkedEntityType = &entityType
			bug.LinkedEntityKey = &entityKey
		} else {
			// Both empty means clear the link
			bug.LinkedEntityType = nil
			bug.LinkedEntityKey = nil
		}
	}

	if err := s.repo.Update(ctx, bug); err != nil {
		return nil, fmt.Errorf("failed to update bug %s: %w", key, err)
	}

	// `--tag` on update is additive only; detach goes through `shark bug tag rm`.
	if err := attachTagsIfAny(ctx, s.tagSvc, models.EntityTypeBug, bug.ID, updates.Tags); err != nil {
		return nil, err
	}

	return bug, nil
}

// DeleteBug deletes a bug by its key.
func (s *BugService) DeleteBug(ctx context.Context, key string) error {
	bug, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get bug %s: %w", key, err)
	}

	if err := s.repo.Delete(ctx, bug.ID); err != nil {
		return fmt.Errorf("failed to delete bug %s: %w", key, err)
	}

	return nil
}

// ListBugs retrieves bugs with optional filters.
func (s *BugService) ListBugs(ctx context.Context, filters BugFilters) ([]*models.Bug, error) {
	// Block 1: pre-filter by tag IDs (E28-F05 §2.5.2).
	var taggedIDSet map[int64]struct{}
	if len(filters.Tags) > 0 {
		if s.tagSvc == nil {
			return nil, &TagFilterUnavailableError{}
		}
		ids, err := s.tagSvc.EntityIDsByTags(ctx, models.EntityTypeBug, filters.Tags, TagQueryOpAnd)
		if err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			return []*models.Bug{}, nil // REQ-F-017 short-circuit
		}
		taggedIDSet = make(map[int64]struct{}, len(ids))
		for _, id := range ids {
			taggedIDSet[id] = struct{}{}
		}
	}

	repoFilters := &repository.BugListFilters{
		Status:           filters.Status,
		Severity:         filters.Severity,
		LinkedEntityKey:  filters.LinkedEntityKey,
		IncludeTerminal:  filters.ShowAll,
		TerminalStatuses: s.workflowSvc.GetTerminalStatuses(),
	}

	bugs, err := s.repo.List(ctx, repoFilters)
	if err != nil {
		return nil, fmt.Errorf("failed to list bugs: %w", err)
	}

	// Block 2: post-filter in-memory (E28-F05 §2.5.2).
	bugs = filterByTagIDs(bugs, taggedIDSet, func(b *models.Bug) int64 { return b.ID })

	return bugs, nil
}

// TransitionStatus transitions a bug to a specific status with workflow validation.
// Delegates to EntityService.TransitionStatus for shared transition logic.
func (s *BugService) TransitionStatus(ctx context.Context, key string, targetStatus string, opts TransitionOptions) (*TransitionResult, error) {
	return s.entitySvc.TransitionStatus(
		ctx, s.entityRepo, models.EntityTypeBug, key, targetStatus, opts,
		SimpleTransitionFeatures(),
		s.makeResolveActionFn(),
	)
}

// GetNextStatus returns the available transitions for the current status of a bug.
func (s *BugService) GetNextStatus(ctx context.Context, key string) (*NextStatusInfo, error) {
	return s.entitySvc.GetNextStatus(
		ctx, s.entityRepo, models.EntityTypeBug, key,
		s.makeResolveActionFn(),
	)
}

// GetNextStatusForBug returns available transitions for a pre-fetched bug, avoiding a DB re-fetch.
func (s *BugService) GetNextStatusForBug(bug *models.Bug) *NextStatusInfo {
	return s.entitySvc.GetNextStatusForEntity(
		models.EntityTypeBug, bug.Key, bug,
		s.makeResolveActionFn(),
	)
}

// TriageBug triages a bug by setting its severity and optionally assigning an agent.
// It advances the bug status from "reported" to "triaged" (if currently in "reported" status).
func (s *BugService) TriageBug(ctx context.Context, key string, input TriageBugInput) (*models.Bug, error) {
	if !models.ValidBugSeverities[models.BugSeverity(input.Severity)] {
		return nil, fmt.Errorf("invalid severity %q: must be one of critical, high, medium, low", input.Severity)
	}

	bug, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get bug %s: %w", key, err)
	}

	// Update severity
	bug.Severity = models.BugSeverity(input.Severity)

	// Advance to triaged status if currently reported
	validTransitions := s.workflowSvc.GetValidTransitions(string(bug.Status))
	for _, t := range validTransitions {
		if t == "triaged" {
			bug.Status = "triaged"
			break
		}
	}

	if err := s.repo.Update(ctx, bug); err != nil {
		return nil, fmt.Errorf("failed to triage bug %s: %w", key, err)
	}

	return bug, nil
}

// makeResolveActionFn returns a ResolveActionFn callback that generates
// Bug-specific placeholders for orchestrator action resolution.
func (s *BugService) makeResolveActionFn() ResolveActionFn {
	return func(entity models.Entity, status string) *config.PopulatedAction {
		bug, ok := entity.(*models.Bug)
		if !ok {
			return nil
		}
		placeholders := config.BugPlaceholders(bug)
		// Fresh transition context: suppress RESUME CONTEXT preamble in templates.
		// is_resume="true" is reserved for shark get (GetOrchestratorAction).
		placeholders["is_resume"] = "false"
		return s.entitySvc.ResolveActionForStatus(status, placeholders)
	}
}

// GetOrchestratorAction returns the orchestrator action for the bug's current status.
// Used by shark get — entity is already in this status, so RESUME CONTEXT preamble is shown.
func (s *BugService) GetOrchestratorAction(bug *models.Bug) *config.PopulatedAction {
	placeholders := config.BugPlaceholders(bug)
	placeholders["is_resume"] = "true"
	return s.entitySvc.ResolveActionForStatus(string(bug.Status), placeholders)
}

// validateLinkedEntity validates that a linked entity exists.
func (s *BugService) validateLinkedEntity(ctx context.Context, entityType, entityKey string) error {
	switch entityType {
	case "epic":
		if s.epicRepo == nil {
			return fmt.Errorf("epic repository not available for link validation")
		}
		_, err := s.epicRepo.GetByKey(ctx, entityKey)
		if err != nil {
			return fmt.Errorf("epic %q not found: %w", entityKey, err)
		}
	case "feature":
		if s.featureRepo == nil {
			return fmt.Errorf("feature repository not available for link validation")
		}
		_, err := s.featureRepo.GetByKey(ctx, entityKey)
		if err != nil {
			return fmt.Errorf("feature %q not found: %w", entityKey, err)
		}
	case "task":
		if s.taskRepo == nil {
			return fmt.Errorf("task repository not available for link validation")
		}
		_, err := s.taskRepo.GetByKey(ctx, entityKey)
		if err != nil {
			return fmt.Errorf("task %q not found: %w", entityKey, err)
		}
	default:
		return fmt.Errorf("invalid linked entity type %q: must be epic, feature, or task", entityType)
	}
	return nil
}

// SetWritableDocRepo sets the writable document repository on the service.
// This enables LinkDocument, UnlinkDocument, and ListRelatedDocumentsByKey operations on bugs.
func (s *BugService) SetWritableDocRepo(writableRepo EntityDocumentRepository, linkRepo EntityDocumentLinkRepository) {
	s.docSvc = NewEntityDocumentService(
		writableRepo,
		linkRepo,
		EntityLookupFnFromRepo(s.entityRepo),
	)
}

// LinkDocument links a document to a bug identified by key.
// Delegates to the shared EntityDocumentService.
func (s *BugService) LinkDocument(ctx context.Context, bugKey, docTitle, docPath string) error {
	if s.docSvc == nil {
		return fmt.Errorf("writable document repository not configured")
	}
	_, err := s.docSvc.LinkDocumentByKey(ctx, bugKey, docTitle, docPath)
	return err
}

// UnlinkDocument removes a document link from a bug by document title.
// Delegates to the shared EntityDocumentService.
func (s *BugService) UnlinkDocument(ctx context.Context, bugKey, docTitle string) error {
	if s.docSvc == nil {
		return fmt.Errorf("writable document repository not configured")
	}
	return s.docSvc.UnlinkDocumentByKey(ctx, bugKey, docTitle)
}

// ListRelatedDocumentsByKey returns all documents linked to a bug identified by key.
// Delegates to the shared EntityDocumentService.
func (s *BugService) ListRelatedDocumentsByKey(ctx context.Context, bugKey string) ([]*models.Document, error) {
	if s.docSvc == nil {
		return []*models.Document{}, nil
	}
	return s.docSvc.ListDocumentsByKey(ctx, bugKey)
}
