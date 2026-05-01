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
	"github.com/jwwelbor/shark-task-manager/internal/repository/techdebt"
	"github.com/jwwelbor/shark-task-manager/internal/utils"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// TechDebtRepository defines the repository interface for tech-debt operations.
type TechDebtRepository interface {
	Create(ctx context.Context, td *models.TechDebt) error
	GetByKey(ctx context.Context, key string) (*models.TechDebt, error)
	GetByID(ctx context.Context, id int64) (*models.TechDebt, error)
	Update(ctx context.Context, td *models.TechDebt) error
	Delete(ctx context.Context, id int64) error
	UpdateStatus(ctx context.Context, id int64, status models.TechDebtStatus) error
	GenerateNextKey(ctx context.Context) (string, error)
	List(ctx context.Context) ([]*models.TechDebt, error)
	ListWithFilters(ctx context.Context, filters techdebt.TechDebtFilters) ([]*models.TechDebt, error)
	CountByStatus(ctx context.Context) (map[string]int, error)
	CountByCategory(ctx context.Context) (map[string]int, error)
}

// TechDebtService provides business logic for tech-debt operations.
type TechDebtService struct {
	repo        TechDebtRepository
	workflowSvc *workflow.Service
	entitySvc   *EntityService
	entityRepo  EntityRepository
	projectRoot string
	docSvc      *EntityDocumentService // shared document operations; built by SetWritableDocRepo
	// tagSvc is optional — nil disables tag integration on create/update.
	// Mirrors the bug/change-card pattern (E28-F04).
	tagSvc TagQuerier
}

// NewTechDebtService creates a new TechDebtService with injected dependencies.
// entitySvc and entityRepo are required for status transition delegation.
// tagSvc is optional (pass nil to disable tag integration).
//
// Panics:
//   - If repo is nil (required dependency)
//   - If entitySvc is nil (required dependency)
func NewTechDebtService(
	repo TechDebtRepository,
	entitySvc *EntityService,
	entityRepo EntityRepository,
	projectRoot string,
	tagSvc TagQuerier,
) *TechDebtService {
	requireNonNil(repo, "TechDebtService requires a non-nil TechDebtRepository")
	requireNonNil(entitySvc, "TechDebtService requires a non-nil EntityService")
	return &TechDebtService{
		repo:        repo,
		workflowSvc: entitySvc.GetWorkflowService().ForLevel(workflow.LevelTechDebt),
		entitySvc:   entitySvc.ForLevel(workflow.LevelTechDebt),
		entityRepo:  entityRepo,
		projectRoot: projectRoot,
		tagSvc:      tagSvc,
	}
}

// CreateTechDebt creates a new tech-debt item with auto-generated key and slug.
func (s *TechDebtService) CreateTechDebt(ctx context.Context, input CreateTechDebtInput) (*models.TechDebt, error) {
	if strings.TrimSpace(input.Title) == "" {
		return nil, fmt.Errorf("tech-debt title cannot be empty")
	}

	if !models.ValidTechDebtCategories[input.Category] {
		return nil, fmt.Errorf("invalid category %q: must be one of code-quality, architecture, dependency, testing, performance, documentation", input.Category)
	}

	if !models.ValidTechDebtSeverities[input.Severity] {
		return nil, fmt.Errorf("invalid severity %q: must be one of critical, high, medium, low", input.Severity)
	}

	// Enforce tag_required_for before key allocation or persistence.
	if err := enforceTagsRequired(ctx, s.tagSvc, models.EntityTypeTechDebt, input.Tags); err != nil {
		return nil, err
	}

	// Generate key
	key, err := s.repo.GenerateNextKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tech-debt key: %w", err)
	}

	// Get default status from workflow
	defaultStatus := s.workflowSvc.GetDefaultStatus()

	// Generate slug
	slug := utils.GenerateSlug(input.Title)

	td := &models.TechDebt{
		BaseEntity: models.BaseEntity{
			Key:   key,
			Title: input.Title,
			Slug:  &slug,
			Size:  input.Size,
		},
		Status:   models.TechDebtStatus(defaultStatus),
		Category: input.Category,
		Severity: input.Severity,
	}

	if input.Description != "" {
		td.Description = &input.Description
	}

	if input.EffortEstimate != "" {
		td.EffortEstimate = &input.EffortEstimate
	}

	// Resolve file path: use caller-supplied path or compute default
	var filePath string
	if input.FilePath != nil && *input.FilePath != "" {
		filePath = *input.FilePath
	} else {
		filePath = filepath.Join("docs", "plan", "tech-debt", key+".md")
	}
	td.FilePath = &filePath

	if err := s.repo.Create(ctx, td); err != nil {
		return nil, fmt.Errorf("failed to create tech-debt: %w", err)
	}

	// Attach tags after insert so td.ID is valid. Not wrapped in a
	// transaction — on failure the row is persisted with zero tags and
	// the user retries via `shark td update --tag=...`.
	if err := attachTagsIfAny(ctx, s.tagSvc, models.EntityTypeTechDebt, td.ID, input.Tags); err != nil {
		return nil, err
	}

	// Generate and write markdown file (best-effort)
	content := s.generateMarkdown(td)
	writer := fileops.NewEntityFileWriter()
	_, writeErr := writer.WriteEntityFile(fileops.WriteOptions{
		Content:        []byte(content),
		ProjectRoot:    s.projectRoot,
		FilePath:       filePath,
		EntityType:     "tech_debt",
		UseAtomicWrite: !input.Force,
		Force:          input.Force,
	})
	if writeErr != nil {
		// Log warning but don't fail -- DB record is the source of truth
		slog.Warn("failed to write tech-debt file", "path", filePath, "error", writeErr)
	}

	return td, nil
}

// generateMarkdown produces a markdown document for a newly created tech-debt item.
func (s *TechDebtService) generateMarkdown(td *models.TechDebt) string {
	var sb strings.Builder

	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("tech_debt_key: %s\n", td.Key))
	sb.WriteString(fmt.Sprintf("title: %s\n", td.Title))
	sb.WriteString(fmt.Sprintf("status: %s\n", td.Status))
	sb.WriteString(fmt.Sprintf("category: %s\n", td.Category))
	sb.WriteString(fmt.Sprintf("severity: %s\n", td.Severity))
	if td.EffortEstimate != nil {
		sb.WriteString(fmt.Sprintf("effort_estimate: %s\n", *td.EffortEstimate))
	}
	if td.Slug != nil {
		sb.WriteString(fmt.Sprintf("slug: %s\n", *td.Slug))
	}
	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("# %s\n\n", td.Title))
	sb.WriteString("## Description\n\n")

	if td.Description != nil {
		sb.WriteString(*td.Description + "\n\n")
	} else {
		sb.WriteString("[Describe the technical debt and its impact]\n\n")
	}

	sb.WriteString("## Impact\n\n")
	sb.WriteString("[What is the impact of this tech debt?]\n\n")
	sb.WriteString("## Resolution Plan\n\n")
	sb.WriteString("[How should this be resolved?]\n\n")
	sb.WriteString("## Resolution Notes\n\n")
	sb.WriteString("[Notes on the resolution once complete]\n")

	return sb.String()
}

// GetTechDebt retrieves a tech-debt item by its key.
func (s *TechDebtService) GetTechDebt(ctx context.Context, key string) (*models.TechDebt, error) {
	td, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get tech-debt %s: %w", key, err)
	}
	return td, nil
}

// UpdateTechDebt applies partial updates to a tech-debt item.
func (s *TechDebtService) UpdateTechDebt(ctx context.Context, key string, updates TechDebtUpdates) (*models.TechDebt, error) {
	td, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get tech-debt %s: %w", key, err)
	}

	if updates.Title != nil {
		if strings.TrimSpace(*updates.Title) == "" {
			return nil, fmt.Errorf("tech-debt title cannot be empty")
		}
		td.Title = *updates.Title
		slug := utils.GenerateSlug(td.Title)
		td.Slug = &slug
	}

	if updates.Description != nil {
		td.Description = updates.Description
	}

	if updates.Category != nil {
		if !models.ValidTechDebtCategories[*updates.Category] {
			return nil, fmt.Errorf("invalid category %q: must be one of code-quality, architecture, dependency, testing, performance, documentation", *updates.Category)
		}
		td.Category = *updates.Category
	}

	if updates.Severity != nil {
		if !models.ValidTechDebtSeverities[*updates.Severity] {
			return nil, fmt.Errorf("invalid severity %q: must be one of critical, high, medium, low", *updates.Severity)
		}
		td.Severity = *updates.Severity
	}

	if updates.EffortEstimate != nil {
		td.EffortEstimate = updates.EffortEstimate
	}

	if updates.FilePath != nil {
		td.FilePath = updates.FilePath
	}

	// Three-branch Size update logic:
	//   ClearSize=true       → set to NULL
	//   Size != nil          → set to value
	//   neither              → leave unchanged (no-op)
	if updates.ClearSize {
		td.Size = nil
	} else if updates.Size != nil {
		td.Size = updates.Size
	}

	if err := s.repo.Update(ctx, td); err != nil {
		return nil, fmt.Errorf("failed to update tech-debt %s: %w", key, err)
	}

	// `--tag` on update is additive only; detach goes through `shark td tag rm`.
	if err := attachTagsIfAny(ctx, s.tagSvc, models.EntityTypeTechDebt, td.ID, updates.Tags); err != nil {
		return nil, err
	}

	return td, nil
}

// DeleteTechDebt deletes a tech-debt item by its key.
func (s *TechDebtService) DeleteTechDebt(ctx context.Context, key string) error {
	td, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get tech-debt %s: %w", key, err)
	}

	if err := s.repo.Delete(ctx, td.ID); err != nil {
		return fmt.Errorf("failed to delete tech-debt %s: %w", key, err)
	}

	return nil
}

// ListTechDebts retrieves tech-debt items with optional filters.
func (s *TechDebtService) ListTechDebts(ctx context.Context, filters TechDebtFilters) ([]*models.TechDebt, error) {
	repoFilters := techdebt.TechDebtFilters{
		Status:          filters.Status,
		Category:        filters.Category,
		Severity:        filters.Severity,
		IncludeTerminal: filters.ShowAll,
	}

	items, err := s.repo.ListWithFilters(ctx, repoFilters)
	if err != nil {
		return nil, fmt.Errorf("failed to list tech-debts: %w", err)
	}

	return items, nil
}

// TransitionStatus transitions a tech-debt item to a specific status with workflow validation.
// Delegates to EntityService.TransitionStatus for shared transition logic.
func (s *TechDebtService) TransitionStatus(ctx context.Context, key string, targetStatus string, opts TransitionOptions) (*TransitionResult, error) {
	return s.entitySvc.TransitionStatus(
		ctx, s.entityRepo, models.EntityTypeTechDebt, key, targetStatus, opts,
		SimpleTransitionFeatures(),
		s.makeResolveActionFn(),
	)
}

// GetNextStatus returns the available transitions for the current status of a tech-debt item.
func (s *TechDebtService) GetNextStatus(ctx context.Context, key string) (*NextStatusInfo, error) {
	return s.entitySvc.GetNextStatus(
		ctx, s.entityRepo, models.EntityTypeTechDebt, key,
		s.makeResolveActionFn(),
	)
}

// GetNextStatusForTechDebt returns available transitions for a pre-fetched tech-debt item,
// avoiding a DB re-fetch.
func (s *TechDebtService) GetNextStatusForTechDebt(td *models.TechDebt) *NextStatusInfo {
	return s.entitySvc.GetNextStatusForEntity(
		models.EntityTypeTechDebt, td.Key, td,
		s.makeResolveActionFn(),
	)
}

// TriageTechDebt triages a tech-debt item by setting category, severity, and effort estimate.
// It advances the status from "identified" to "triaged" if currently in "identified" status.
// If already past "identified", it updates fields without changing status.
func (s *TechDebtService) TriageTechDebt(ctx context.Context, key string, input TriageTechDebtInput) (*models.TechDebt, error) {
	td, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get tech-debt %s: %w", key, err)
	}

	// Apply triage fields if provided
	if input.Severity != "" {
		sev := models.TechDebtSeverity(input.Severity)
		if !models.ValidTechDebtSeverities[sev] {
			return nil, fmt.Errorf("invalid severity %q: must be one of critical, high, medium, low", input.Severity)
		}
		td.Severity = sev
	}

	if input.Category != "" {
		cat := models.TechDebtCategory(input.Category)
		if !models.ValidTechDebtCategories[cat] {
			return nil, fmt.Errorf("invalid category %q: must be one of code-quality, architecture, dependency, testing, performance, documentation", input.Category)
		}
		td.Category = cat
	}

	if input.EffortEstimate != "" {
		td.EffortEstimate = &input.EffortEstimate
	}

	// Advance to triaged status if currently identified
	validTransitions := s.workflowSvc.GetValidTransitions(string(td.Status))
	for _, t := range validTransitions {
		if t == "triaged" {
			td.Status = "triaged"
			break
		}
	}

	if err := s.repo.Update(ctx, td); err != nil {
		return nil, fmt.Errorf("failed to triage tech-debt %s: %w", key, err)
	}

	return td, nil
}

// GetStatusOptions returns valid next statuses for a tech-debt item.
func (s *TechDebtService) GetStatusOptions(ctx context.Context, key string) ([]string, error) {
	td, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get tech-debt %s: %w", key, err)
	}

	transitions := s.workflowSvc.GetValidTransitions(string(td.Status))
	return transitions, nil
}

// makeResolveActionFn returns a ResolveActionFn callback that generates
// TechDebt-specific placeholders for orchestrator action resolution.
func (s *TechDebtService) makeResolveActionFn() ResolveActionFn {
	return func(entity models.Entity, status string) *config.PopulatedAction {
		td, ok := entity.(*models.TechDebt)
		if !ok {
			return nil
		}
		placeholders := config.TechDebtPlaceholders(td)
		// Fresh transition context: suppress RESUME CONTEXT preamble in templates.
		// is_resume="true" is reserved for shark get (GetOrchestratorAction).
		placeholders["is_resume"] = "false"
		return s.entitySvc.ResolveActionForStatus(status, placeholders)
	}
}

// GetOrchestratorAction returns the orchestrator action for the tech-debt item's current status.
// Used by shark get — entity is already in this status, so RESUME CONTEXT preamble is shown.
func (s *TechDebtService) GetOrchestratorAction(td *models.TechDebt) *config.PopulatedAction {
	placeholders := config.TechDebtPlaceholders(td)
	placeholders["is_resume"] = "true"
	return s.entitySvc.ResolveActionForStatus(string(td.Status), placeholders)
}

// SetWritableDocRepo sets the writable document repository on the service.
// This enables LinkDocument, UnlinkDocument, and ListRelatedDocumentsByKey operations.
func (s *TechDebtService) SetWritableDocRepo(writableRepo EntityDocumentRepository, linkRepo EntityDocumentLinkRepository) {
	s.docSvc = NewEntityDocumentService(
		writableRepo,
		linkRepo,
		EntityLookupFnFromRepo(s.entityRepo),
	)
}

// LinkDocument links a document to a tech-debt item identified by key.
func (s *TechDebtService) LinkDocument(ctx context.Context, tdKey, docTitle, docPath string) error {
	if s.docSvc == nil {
		return fmt.Errorf("writable document repository not configured")
	}
	_, err := s.docSvc.LinkDocumentByKey(ctx, tdKey, docTitle, docPath)
	return err
}

// UnlinkDocument removes a document link from a tech-debt item by document title.
func (s *TechDebtService) UnlinkDocument(ctx context.Context, tdKey, docTitle string) error {
	if s.docSvc == nil {
		return fmt.Errorf("writable document repository not configured")
	}
	return s.docSvc.UnlinkDocumentByKey(ctx, tdKey, docTitle)
}

// ListRelatedDocumentsByKey returns all documents linked to a tech-debt item identified by key.
func (s *TechDebtService) ListRelatedDocumentsByKey(ctx context.Context, tdKey string) ([]*models.Document, error) {
	if s.docSvc == nil {
		return []*models.Document{}, nil
	}
	return s.docSvc.ListDocumentsByKey(ctx, tdKey)
}
