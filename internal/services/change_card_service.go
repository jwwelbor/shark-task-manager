package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/fileops"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/utils"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
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

// ChangeCardService provides business logic for change-card operations.
type ChangeCardService struct {
	repo        ChangeCardRepository
	workflowSvc *workflow.Service
	epicRepo    EpicRepository
	featureRepo FeatureRepository
	projectRoot string
}

// NewChangeCardService creates a new ChangeCardService.
// The workflow service is automatically scoped to the change level.
//
// Required: repo, workflowSvc.
// Optional (degrade gracefully when nil): epicRepo, featureRepo.
func NewChangeCardService(
	repo ChangeCardRepository,
	workflowSvc *workflow.Service,
	epicRepo EpicRepository,
	featureRepo FeatureRepository,
	projectRoot string,
) *ChangeCardService {
	requireNonNil(repo, "ChangeCardService requires a non-nil ChangeCardRepository")
	requireNonNil(workflowSvc, "ChangeCardService requires a non-nil workflow.Service")
	return &ChangeCardService{
		repo:        repo,
		workflowSvc: workflowSvc.ForLevel(workflow.LevelChange),
		epicRepo:    epicRepo,
		featureRepo: featureRepo,
		projectRoot: projectRoot,
	}
}

// CreateChangeCard creates a new change-card with optional entity linking.
func (s *ChangeCardService) CreateChangeCard(ctx context.Context, input CreateChangeCardInput) (*models.ChangeCard, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, fmt.Errorf("change-card title cannot be empty")
	}

	// Resolve epic link
	var epicID *int64
	if input.EpicKey != "" && s.epicRepo != nil {
		epic, err := s.epicRepo.GetByKey(ctx, input.EpicKey)
		if err != nil {
			return nil, fmt.Errorf("epic %s not found: %w", input.EpicKey, err)
		}
		epicID = &epic.ID
	}

	// Resolve feature link
	var featureID *int64
	if input.FeatureKey != "" && s.featureRepo != nil {
		feature, err := s.featureRepo.GetByKey(ctx, input.FeatureKey)
		if err != nil {
			return nil, fmt.Errorf("feature %s not found: %w", input.FeatureKey, err)
		}
		featureID = &feature.ID
	}

	// Generate next key
	nextKey, err := s.repo.GetNextKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate change-card key: %w", err)
	}

	// Generate slug
	slug := utils.GenerateSlug(title)

	// Get default status from workflow
	defaultStatus := s.workflowSvc.GetDefaultStatus()

	// Default priority
	priority := input.Priority
	if priority == 0 {
		priority = 5
	}

	// Build model
	card := &models.ChangeCard{
		Key:       nextKey,
		Title:     title,
		Status:    models.ChangeCardStatus(defaultStatus),
		Priority:  priority,
		Slug:      slug,
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
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Set file path
	filePath := filepath.Join("docs", "plan", "changes", nextKey+".md")
	card.FilePath = filePath

	// Create in database
	if err := s.repo.Create(ctx, card); err != nil {
		return nil, fmt.Errorf("failed to create change-card: %w", err)
	}

	// Generate and write markdown file (best-effort)
	content := s.generateMarkdown(card)
	writer := fileops.NewEntityFileWriter()
	_, writeErr := writer.WriteEntityFile(fileops.WriteOptions{
		Content:        []byte(content),
		ProjectRoot:    s.projectRoot,
		FilePath:       filePath,
		EntityType:     "change",
		UseAtomicWrite: true,
	})
	if writeErr != nil {
		// Log warning but don't fail -- DB record is the source of truth
		fmt.Fprintf(os.Stderr, "warning: failed to write change-card file %s: %v\n", filePath, writeErr)
	}

	return card, nil
}

// GetChangeCard retrieves a change-card by key.
func (s *ChangeCardService) GetChangeCard(ctx context.Context, key string) (*models.ChangeCard, error) {
	card, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get change-card %s: %w", key, err)
	}
	return card, nil
}

// ListChangeCards retrieves change-cards with optional filtering.
func (s *ChangeCardService) ListChangeCards(ctx context.Context, filters ChangeCardFilters) ([]*models.ChangeCard, error) {
	repoFilter := &repository.ChangeCardRepoFilter{
		IncludeTerminal: filters.ShowAll,
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
		card.Slug = utils.GenerateSlug(card.Title)
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

	if err := card.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if err := s.repo.Update(ctx, card); err != nil {
		return nil, fmt.Errorf("failed to update change-card %s: %w", key, err)
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
	if card.FilePath != "" && s.projectRoot != "" {
		absPath := filepath.Join(s.projectRoot, card.FilePath)
		if removeErr := os.Remove(absPath); removeErr != nil && !os.IsNotExist(removeErr) {
			fmt.Fprintf(os.Stderr, "warning: failed to delete change-card file %s: %v\n", absPath, removeErr)
		}
	}

	return nil
}

// ApproveChangeCard transitions a change-card from proposed to approved.
func (s *ChangeCardService) ApproveChangeCard(ctx context.Context, key string) (*models.ChangeCard, error) {
	card, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get change-card %s: %w", key, err)
	}

	if err := s.workflowSvc.ValidateTransition(string(card.Status), "approved"); err != nil {
		return nil, fmt.Errorf("cannot approve change-card %s: current status is '%s': %w", key, card.Status, err)
	}

	if err := s.repo.UpdateStatus(ctx, card.ID, models.ChangeCardStatus("approved")); err != nil {
		return nil, fmt.Errorf("failed to update change-card %s status: %w", key, err)
	}

	card.Status = models.ChangeCardStatus("approved")
	return card, nil
}

// AdvanceChangeCardStatus advances a change-card to the next workflow status.
func (s *ChangeCardService) AdvanceChangeCardStatus(ctx context.Context, key string) (*models.ChangeCard, error) {
	card, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get change-card %s: %w", key, err)
	}

	validTransitions := s.workflowSvc.GetValidTransitions(string(card.Status))
	if len(validTransitions) == 0 {
		return nil, fmt.Errorf("change-card %s is in terminal status '%s'; no further transitions available", key, card.Status)
	}
	nextStatus := validTransitions[0]

	if err := s.workflowSvc.ValidateTransition(string(card.Status), nextStatus); err != nil {
		return nil, fmt.Errorf("cannot advance change-card %s: %w", key, err)
	}

	if err := s.repo.UpdateStatus(ctx, card.ID, models.ChangeCardStatus(nextStatus)); err != nil {
		return nil, fmt.Errorf("failed to update change-card %s status: %w", key, err)
	}

	card.Status = models.ChangeCardStatus(nextStatus)
	return card, nil
}

// SetChangeCardStatus sets a change-card to a specific status.
func (s *ChangeCardService) SetChangeCardStatus(ctx context.Context, key string, targetStatus string) (*models.ChangeCard, error) {
	card, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get change-card %s: %w", key, err)
	}

	if err := s.workflowSvc.ValidateTransition(string(card.Status), targetStatus); err != nil {
		return nil, fmt.Errorf("cannot set change-card %s status to '%s': %w", key, targetStatus, err)
	}

	if err := s.repo.UpdateStatus(ctx, card.ID, models.ChangeCardStatus(targetStatus)); err != nil {
		return nil, fmt.Errorf("failed to update change-card %s status: %w", key, err)
	}

	card.Status = models.ChangeCardStatus(targetStatus)
	return card, nil
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
	sb.WriteString(fmt.Sprintf("slug: %s\n", card.Slug))
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

// resolveAction looks up the orchestrator action for a given change-card status.
func (s *ChangeCardService) resolveAction(card *models.ChangeCard, status string) *config.PopulatedAction {
	wf := s.workflowSvc.GetWorkflow()
	if wf == nil || wf.StatusMetadata == nil {
		return nil
	}
	meta, exists := wf.StatusMetadata[status]
	if !exists || meta.OrchestratorAction == nil {
		return nil
	}
	placeholders := config.ChangeCardPlaceholders(card)
	return &config.PopulatedAction{
		Action:      meta.OrchestratorAction.Action,
		AgentType:   meta.OrchestratorAction.AgentType,
		Skills:      meta.OrchestratorAction.Skills,
		Instruction: meta.OrchestratorAction.PopulateTemplate(placeholders),
	}
}

// GetOrchestratorAction returns the orchestrator action for the change-card's current status.
func (s *ChangeCardService) GetOrchestratorAction(card *models.ChangeCard) *config.PopulatedAction {
	return s.resolveAction(card, string(card.Status))
}

// GetValidTransitions returns the valid next statuses for the given status key.
func (s *ChangeCardService) GetValidTransitions(status string) []string {
	wf := s.workflowSvc.GetWorkflow()
	if wf == nil {
		return []string{}
	}
	transitions, ok := wf.StatusFlow[status]
	if !ok {
		return []string{}
	}
	return transitions
}
