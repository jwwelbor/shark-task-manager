package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/keys"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/sprint"
	"github.com/jwwelbor/shark-task-manager/internal/utils"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// SprintRepository defines the repository interface for sprint operations.
// Extended in T-E19-F03-005 to include assignment CRUD, backlog query, and
// entity-ID resolution methods required by AddEntityToSprint and RemoveEntityFromSprint.
type SprintRepository interface {
	// --- Core CRUD (from E19-F01/F02) ---
	Create(ctx context.Context, s *models.Sprint) error
	GetByKey(ctx context.Context, key string) (*models.Sprint, error)
	GetByID(ctx context.Context, id int64) (*models.Sprint, error)
	Update(ctx context.Context, s *models.Sprint) error
	Delete(ctx context.Context, id int64) error
	UpdateStatus(ctx context.Context, id int64, status models.SprintStatus) error
	GetNextKey(ctx context.Context) (string, error)
	List(ctx context.Context, filters *sprint.SprintListFilters) ([]*models.Sprint, error)

	// --- Assignment CRUD (F03) ---
	AddAssignment(ctx context.Context, assignment *models.SprintAssignment) error
	RemoveAssignment(ctx context.Context, sprintID int64, entityType string, entityID int64) error
	GetActiveAssignment(ctx context.Context, entityType string, entityID int64) (*models.SprintAssignment, error)
	ListAssignments(ctx context.Context, sprintID int64, entityType *string) ([]*models.SprintAssignment, error)
	ListAssignmentsForCarryover(ctx context.Context, sprintID int64, completedStatuses ...string) ([]*models.SprintAssignment, error)
	ReassignToSprintTx(ctx context.Context, tx *sql.Tx, assignmentIDs []int64, newSprintID int64) error
	DropAssignmentsTx(ctx context.Context, tx *sql.Tx, assignmentIDs []int64) error
	CreateCompletionTx(ctx context.Context, tx *sql.Tx, completion *models.SprintCompletion) error
	ListBacklog(ctx context.Context, sprintID int64, entityType *string, blockedOnly bool, blockedStatuses ...string) ([]*sprint.BacklogItem, error)

	// --- Entity ID resolution (F03) ---
	GetTaskIDByKey(ctx context.Context, key string) (int64, error)
	GetBugIDByKey(ctx context.Context, key string) (int64, error)
	GetChangeCardIDByKey(ctx context.Context, key string) (int64, error)
	GetTechDebtIDByKey(ctx context.Context, key string) (int64, error)
}

// SprintAssignmentQueryRepository handles assignment queries needed for sprint planning.
// Implemented by *sprint.SprintRepository — no separate type needed.
type SprintAssignmentQueryRepository interface {
	BulkAssign(ctx context.Context, sprintID int64, assignments []models.SprintAssignment) (int, error)
	ListUnassignedBacklog(ctx context.Context, entityTypes []string) ([]sprint.BacklogItem, error)
	GetAssignmentsWithSize(ctx context.Context, sprintID int64) ([]sprint.AssignmentWithSize, error)
}

// SprintCapacityRepository handles capacity CRUD for sprints.
// Implemented by *sprint.SprintRepository — no separate type needed.
type SprintCapacityRepository interface {
	GetCapacity(ctx context.Context, sprintID int64) ([]*models.SprintCapacity, error)
	SetCapacity(ctx context.Context, c *models.SprintCapacity) error
}

// CreateSprintInput contains parameters for creating a new sprint.
type CreateSprintInput struct {
	Name      string    // Required, non-empty
	Goal      string    // Optional
	StartDate time.Time // Required
	EndDate   time.Time // Required
}

// UpdateSprintInput contains parameters for updating a sprint.
// Fields are pointers to indicate optional updates.
type UpdateSprintInput struct {
	Name    *string    // Optional update
	Goal    *string    // Optional update
	EndDate *time.Time // Optional update
}

// SprintListFilters contains filter options for listing sprints.
type SprintListFilters struct {
	Status string // Optional filter, e.g. "active"
}

// SprintService provides business logic for sprint operations.
type SprintService struct {
	repo           SprintRepository
	workflowSvc    *workflow.Service
	assignmentRepo SprintAssignmentQueryRepository // optional: nil-safe
	capacityRepo   SprintCapacityRepository        // optional: nil-safe
	cfg            *config.Config                  // optional: nil-safe; used for sprint_defaults
}

// NewSprintService creates a SprintService with dependency injection.
// Panics if repo or workflowSvc is nil.
// assignmentRepo, capacityRepo, and cfg are optional (nil-safe) and degrade gracefully.
// Existing callers that pass only (repo, workflowSvc) must be updated to the 5-argument form;
// pass nil for the optional parameters to preserve previous behaviour.
func NewSprintService(
	repo SprintRepository,
	workflowSvc *workflow.Service,
	assignmentRepo SprintAssignmentQueryRepository,
	capacityRepo SprintCapacityRepository,
	cfg *config.Config,
) *SprintService {
	if repo == nil {
		panic("SprintRepository cannot be nil")
	}
	if workflowSvc == nil {
		panic("workflow.Service cannot be nil")
	}

	return &SprintService{
		repo:           repo,
		workflowSvc:    workflowSvc.ForLevel("sprint"),
		assignmentRepo: assignmentRepo,
		capacityRepo:   capacityRepo,
		cfg:            cfg,
	}
}

// CreateSprint creates a new sprint with validation.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - input: CreateSprintInput with name, goal, and dates
//
// Returns:
//   - *models.Sprint: the created sprint with ID populated
//   - error: validation errors or repository errors
//
// Validation:
//   - Name must be non-empty after trimming whitespace
//   - StartDate must be before EndDate
func (s *SprintService) CreateSprint(ctx context.Context, input CreateSprintInput) (*models.Sprint, error) {
	// Validate name is non-empty
	if strings.TrimSpace(input.Name) == "" {
		return nil, fmt.Errorf("sprint name cannot be empty")
	}

	// Validate date ordering
	if !input.EndDate.After(input.StartDate) {
		return nil, fmt.Errorf("sprint end_date must be after start_date")
	}

	// Generate key
	key, err := s.repo.GetNextKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate sprint key: %w", err)
	}

	// Generate slug from name
	slug := utils.GenerateSlug(input.Name)

	// Get default status from workflow
	initialStatus := s.workflowSvc.GetInitialStatusString()

	// Create sprint model
	newSprint := &models.Sprint{
		Key:       key,
		Name:      input.Name,
		Goal:      input.Goal,
		StartDate: input.StartDate,
		EndDate:   input.EndDate,
		Status:    models.SprintStatus(initialStatus),
		Slug:      slug,
	}

	// Validate model
	if err := newSprint.Validate(); err != nil {
		return nil, fmt.Errorf("sprint validation failed: %w", err)
	}

	// Create in repository
	if err := s.repo.Create(ctx, newSprint); err != nil {
		return nil, fmt.Errorf("failed to create sprint: %w", err)
	}

	// Apply sprint_defaults.capacity if configured and capacityRepo is available.
	// Per spec §2.8: non-fatal — log but do NOT fail sprint creation if capacity insert fails.
	if s.cfg != nil && s.cfg.SprintDefaults != nil &&
		len(s.cfg.SprintDefaults.Capacity) > 0 && s.capacityRepo != nil {
		for agentType, points := range s.cfg.SprintDefaults.Capacity {
			_ = s.capacityRepo.SetCapacity(ctx, &models.SprintCapacity{
				SprintID:       newSprint.ID,
				AgentType:      agentType,
				CapacityPoints: points,
			})
		}
	}

	return newSprint, nil
}

// GetSprint retrieves a sprint by key.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: sprint key (e.g., "S001")
//
// Returns:
//   - *models.Sprint: the sprint, or error if not found
//   - error: NotFoundError or repository errors
func (s *SprintService) GetSprint(ctx context.Context, key string) (*models.Sprint, error) {
	sprint, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get sprint %s: %w", key, err)
	}
	return sprint, nil
}

// ListSprints retrieves sprints, optionally filtered by status.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - filters: SprintListFilters with optional status filter
//
// Returns:
//   - []*models.Sprint: list of sprints (empty slice if none found)
//   - error: repository errors
func (s *SprintService) ListSprints(ctx context.Context, filters *SprintListFilters) ([]*models.Sprint, error) {
	// Convert service-level filter to repository filter
	var repoFilters *sprint.SprintListFilters
	if filters != nil && filters.Status != "" {
		status := models.SprintStatus(filters.Status)
		repoFilters = &sprint.SprintListFilters{
			Status: &status,
		}
	}

	sprints, err := s.repo.List(ctx, repoFilters)
	if err != nil {
		return nil, fmt.Errorf("failed to list sprints: %w", err)
	}

	// Return empty slice instead of nil for consistency
	if sprints == nil {
		return []*models.Sprint{}, nil
	}

	return sprints, nil
}

// UpdateSprint updates an existing sprint.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: sprint key
//   - updates: UpdateSprintInput with optional fields
//
// Returns:
//   - *models.Sprint: the updated sprint
//   - error: validation errors or repository errors
//
// Validation:
//   - Name (if provided) must be non-empty
//   - EndDate (if provided) must be after StartDate
func (s *SprintService) UpdateSprint(ctx context.Context, key string, updates UpdateSprintInput) (*models.Sprint, error) {
	// Get current sprint
	current, err := s.GetSprint(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to update sprint %s: %w", key, err)
	}

	// Apply updates
	if updates.Name != nil {
		name := strings.TrimSpace(*updates.Name)
		if name == "" {
			return nil, fmt.Errorf("sprint name cannot be empty")
		}
		current.Name = name
		current.Slug = utils.GenerateSlug(name)
	}

	if updates.Goal != nil {
		current.Goal = *updates.Goal
	}

	if updates.EndDate != nil {
		// Validate date ordering
		if !updates.EndDate.After(current.StartDate) {
			return nil, fmt.Errorf("sprint end_date must be after start_date")
		}
		current.EndDate = *updates.EndDate
	}

	// Validate updated model
	if err := current.Validate(); err != nil {
		return nil, fmt.Errorf("sprint validation failed: %w", err)
	}

	// Update in repository
	if err := s.repo.Update(ctx, current); err != nil {
		return nil, fmt.Errorf("failed to update sprint %s: %w", key, err)
	}

	return current, nil
}

// DeleteSprint deletes a sprint by key.
// Sprints can only be deleted when in "todo" status.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: sprint key
//
// Returns:
//   - error: validation errors or repository errors
//
// Validation:
//   - Sprint status must be "todo"
func (s *SprintService) DeleteSprint(ctx context.Context, key string) error {
	sprint, err := s.GetSprint(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete sprint %s: %w", key, err)
	}

	// Only allow deletion of sprints in todo status
	if string(sprint.Status) != "todo" {
		return fmt.Errorf("cannot delete sprint %s in status %s: only sprints in todo status can be deleted", key, sprint.Status)
	}

	if err := s.repo.Delete(ctx, sprint.ID); err != nil {
		return fmt.Errorf("failed to delete sprint %s: %w", key, err)
	}

	return nil
}

// StartSprint transitions a sprint to in_progress status.
// Enforces single-active-sprint constraint: only one sprint can be in_progress at a time.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: sprint key
//
// Returns:
//   - *models.Sprint: the updated sprint with new status
//   - error: validation errors or repository errors
//
// Validation:
//   - Current sprint status must allow transition to "in_progress"
//   - No other sprint can be in "in_progress" status
func (s *SprintService) StartSprint(ctx context.Context, key string) (*models.Sprint, error) {
	// Get sprint
	sprint, err := s.GetSprint(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to start sprint %s: %w", key, err)
	}

	// Validate workflow transition
	if err := s.workflowSvc.ValidateTransition(string(sprint.Status), "in_progress"); err != nil {
		return nil, fmt.Errorf("cannot start sprint %s in status %s: %w", key, sprint.Status, err)
	}

	// Check single-active constraint: no other sprint should be in_progress
	activeSprints, err := s.ListSprints(ctx, &SprintListFilters{Status: "in_progress"})
	if err != nil {
		return nil, fmt.Errorf("failed to check active sprints for %s: %w", key, err)
	}

	if len(activeSprints) > 0 {
		// There's already an active sprint
		return nil, fmt.Errorf("cannot activate sprint %s: sprint %s is already active", key, activeSprints[0].Key)
	}

	// Update status
	if err := s.repo.UpdateStatus(ctx, sprint.ID, models.SprintStatus("in_progress")); err != nil {
		return nil, fmt.Errorf("failed to start sprint %s: %w", key, err)
	}

	// Reload and return updated sprint
	updated, err := s.GetSprint(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to reload sprint %s after starting: %w", key, err)
	}

	return updated, nil
}

// CloseSprint transitions a sprint to ready_for_review status.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: sprint key
//
// Returns:
//   - *models.Sprint: the updated sprint with new status
//   - error: validation errors or repository errors
//
// Validation:
//   - Current sprint status must allow transition to "ready_for_review"
//
// Note: Carryover logic is deferred to a future feature.
func (s *SprintService) CloseSprint(ctx context.Context, key string) (*models.Sprint, error) {
	// Get sprint
	sprint, err := s.GetSprint(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to close sprint %s: %w", key, err)
	}

	// Validate workflow transition
	if err := s.workflowSvc.ValidateTransition(string(sprint.Status), "ready_for_review"); err != nil {
		return nil, fmt.Errorf("cannot close sprint %s in status %s: %w", key, sprint.Status, err)
	}

	// Update status
	if err := s.repo.UpdateStatus(ctx, sprint.ID, models.SprintStatus("ready_for_review")); err != nil {
		return nil, fmt.Errorf("failed to close sprint %s: %w", key, err)
	}

	// Reload and return updated sprint
	updated, err := s.GetSprint(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to reload sprint %s after closing: %w", key, err)
	}

	return updated, nil
}

// ---------------------------------------------------------------------------
// AddEntityInput — DTO for AddEntityToSprint (T-E19-F03-005)
// ---------------------------------------------------------------------------

// AddEntityInput contains the parameters for assigning an entity to a sprint.
//
// SprintKey and EntityKey are required. AgentType and EstimatedSize are optional
// hints used only for the advisory capacity-warning check; if omitted the check
// is skipped gracefully.
type AddEntityInput struct {
	// SprintKey is the sprint to assign to (e.g., "S024").
	SprintKey string
	// EntityKey is the entity to assign (e.g., "E07-F01-001", "B001", "C001", "TD-001").
	EntityKey string
	// AgentType is an optional hint for capacity-warning computation (e.g., "backend").
	// When empty the capacity check is skipped.
	AgentType string
	// EstimatedSize is the entity's size (story points) used in the capacity check.
	// Zero means "unknown size"; the capacity check treats unknown-size entities as
	// contributing 0 points (advisory only — does not block).
	EstimatedSize int
}

// CapacityWarning is returned alongside the assignment when assigning an entity
// would push an agent type over its configured capacity. It is advisory only —
// the assignment still succeeds. Enforcement is deferred to E19-F05.
type CapacityWarning struct {
	// AgentType is the agent-type bucket that exceeded capacity (e.g., "backend").
	AgentType string
	// Capacity is the configured capacity in story points.
	Capacity float64
	// Allocated is the projected allocated points after this assignment.
	// Always > Capacity when a warning is emitted.
	Allocated float64
}

// ---------------------------------------------------------------------------
// resolveEntityTypeAndID maps an entity key to (sprintEntityType, entityID).
//
// Sprint entity_type strings ("task", "bug", "change_card", "tech_debt") differ
// from the keys package EntityType constants, so this helper translates between
// the two representations. Tech-debt keys (TD-###) are not handled by
// keys.KeyService.Parse — IsTechDebtKey is used as a fallback.
// ---------------------------------------------------------------------------

func resolveEntityTypeAndID(ctx context.Context, repo SprintRepository, entityKey string) (entityType string, entityID int64, err error) {
	keySvc := keys.NewKeyService()
	parsed := keySvc.Parse(entityKey)

	switch parsed.EntityType {
	case keys.EntityTypeTask:
		entityID, err = repo.GetTaskIDByKey(ctx, parsed.Normalized)
		if err != nil {
			return "", 0, fmt.Errorf("task %q not found: %w", entityKey, err)
		}
		return "task", entityID, nil

	case keys.EntityTypeBug:
		entityID, err = repo.GetBugIDByKey(ctx, parsed.Normalized)
		if err != nil {
			return "", 0, fmt.Errorf("bug %q not found: %w", entityKey, err)
		}
		return "bug", entityID, nil

	case keys.EntityTypeChange:
		entityID, err = repo.GetChangeCardIDByKey(ctx, parsed.Normalized)
		if err != nil {
			return "", 0, fmt.Errorf("change_card %q not found: %w", entityKey, err)
		}
		return "change_card", entityID, nil

	default:
		// Tech-debt keys (TD-###) are not handled by KeyService.Parse.
		// Fall back to IsTechDebtKey before returning an unsupported-type error.
		if keys.IsTechDebtKey(entityKey) {
			normalized := strings.ToUpper(strings.TrimSpace(entityKey))
			entityID, err = repo.GetTechDebtIDByKey(ctx, normalized)
			if err != nil {
				return "", 0, fmt.Errorf("tech_debt %q not found: %w", entityKey, err)
			}
			return "tech_debt", entityID, nil
		}
		return "", 0, fmt.Errorf(
			"unsupported entity type %q for sprint assignment: entity key %q must be a task, bug, change-card (C###), or tech-debt (TD-###) key",
			parsed.EntityType, entityKey,
		)
	}
}

// ---------------------------------------------------------------------------
// AddEntityToSprint — T-E19-F03-005
// ---------------------------------------------------------------------------

// AddEntityToSprint assigns any supported entity to a sprint.
//
// Steps:
//  1. Resolve sprint by key.
//  2. Parse entity key to determine entity_type (task/bug/change_card/tech_debt).
//  3. Resolve entity_id by querying the entity's table.
//  4. Check for an existing active assignment; return ConflictError naming the
//     conflicting sprint if one is found.
//  5. Call repo.AddAssignment.
//  6. Compute capacity warning if capacityRepo and AgentType are provided
//     (advisory only — never blocks).
//
// Returns the created SprintAssignment and an optional CapacityWarning.
// A non-nil CapacityWarning does NOT indicate failure; the assignment was created.
func (s *SprintService) AddEntityToSprint(ctx context.Context, input AddEntityInput) (*models.SprintAssignment, *CapacityWarning, error) {
	// Step 1: Resolve sprint
	sprintEntity, err := s.repo.GetByKey(ctx, input.SprintKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve sprint %q: %w", input.SprintKey, err)
	}

	// Step 2+3: Parse entity key and resolve entity ID
	entityType, entityID, err := resolveEntityTypeAndID(ctx, s.repo, input.EntityKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve entity %q for sprint assignment: %w", input.EntityKey, err)
	}

	// Step 4: Conflict check — at most one active sprint assignment per entity
	existing, err := s.repo.GetActiveAssignment(ctx, entityType, entityID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check existing assignment for %q: %w", input.EntityKey, err)
	}
	if existing != nil {
		// Resolve the conflicting sprint key for the error message (AC-2)
		conflictingKey := fmt.Sprintf("sprint id %d", existing.SprintID)
		if conflictingSprint, lookupErr := s.repo.GetByID(ctx, existing.SprintID); lookupErr == nil {
			conflictingKey = conflictingSprint.Key
		}
		return nil, nil, fmt.Errorf(
			"entity %q is already assigned to sprint %s: remove it first before reassigning",
			input.EntityKey, conflictingKey,
		)
	}

	// Step 5: Create assignment
	assignment := &models.SprintAssignment{
		SprintID:   sprintEntity.ID,
		EntityType: entityType,
		EntityID:   entityID,
		AssignedAt: time.Now().UTC(),
	}
	if err := s.repo.AddAssignment(ctx, assignment); err != nil {
		return nil, nil, fmt.Errorf("failed to add assignment for %q to sprint %s: %w",
			input.EntityKey, input.SprintKey, err)
	}

	// Step 6: Advisory capacity warning (never blocks)
	var warning *CapacityWarning
	if s.capacityRepo != nil && input.AgentType != "" {
		warning = s.computeCapacityWarning(ctx, sprintEntity.ID, input.AgentType, input.EstimatedSize)
	}

	return assignment, warning, nil
}

// computeCapacityWarning returns a non-nil CapacityWarning if adding an entity
// with the given agentType and size would exceed the sprint's configured capacity
// for that agent type. Returns nil if capacity is not configured or not exceeded.
//
// This method is advisory — it never blocks the assignment.
func (s *SprintService) computeCapacityWarning(ctx context.Context, sprintID int64, agentType string, newSize int) *CapacityWarning {
	if s.capacityRepo == nil || agentType == "" {
		return nil
	}

	capacities, err := s.capacityRepo.GetCapacity(ctx, sprintID)
	if err != nil {
		// Capacity lookup failure is non-fatal: skip the warning rather than
		// failing the assignment.
		return nil
	}

	// Find capacity record for this agent type
	var agentCap *models.SprintCapacity
	for _, cap := range capacities {
		if cap.AgentType == agentType {
			agentCap = cap
			break
		}
	}
	if agentCap == nil {
		// No capacity configured for this agent type — no warning possible.
		return nil
	}

	// Compute projected allocation after adding the new entity.
	currentAlloc := 0.0
	if agentCap.AllocatedPoints != nil {
		currentAlloc = *agentCap.AllocatedPoints
	}
	projected := currentAlloc + float64(newSize)

	if projected <= agentCap.CapacityPoints {
		return nil
	}

	return &CapacityWarning{
		AgentType: agentType,
		Capacity:  agentCap.CapacityPoints,
		Allocated: projected,
	}
}

// ---------------------------------------------------------------------------
// RemoveEntityFromSprint — T-E19-F03-005
// ---------------------------------------------------------------------------

// RemoveEntityFromSprint soft-deletes the active assignment for an entity from a sprint.
//
// Steps:
//  1. Resolve sprint by key.
//  2. Parse entity key and resolve entity ID.
//  3. Verify an active assignment exists (return not-found error if none).
//  4. Call repo.RemoveAssignment to soft-delete (sets removed_at).
//
// Returns an error if the entity is not currently assigned to any sprint.
func (s *SprintService) RemoveEntityFromSprint(ctx context.Context, sprintKey, entityKey string) error {
	// Step 1: Resolve sprint
	sprintEntity, err := s.repo.GetByKey(ctx, sprintKey)
	if err != nil {
		return fmt.Errorf("failed to resolve sprint %q: %w", sprintKey, err)
	}

	// Step 2: Parse entity key and resolve entity ID
	entityType, entityID, err := resolveEntityTypeAndID(ctx, s.repo, entityKey)
	if err != nil {
		return fmt.Errorf("failed to resolve entity %q for sprint removal: %w", entityKey, err)
	}

	// Step 3: Verify active assignment exists (TC-R08 counter-factual: must not skip this check)
	existing, err := s.repo.GetActiveAssignment(ctx, entityType, entityID)
	if err != nil {
		return fmt.Errorf("failed to check active assignment for %q: %w", entityKey, err)
	}
	if existing == nil {
		return fmt.Errorf("entity %q is not assigned to any active sprint", entityKey)
	}

	// Step 4: Soft-delete the assignment
	if err := s.repo.RemoveAssignment(ctx, sprintEntity.ID, entityType, entityID); err != nil {
		return fmt.Errorf("failed to remove %q from sprint %s: %w", entityKey, sprintKey, err)
	}

	return nil
}

// ArchiveSprint transitions a sprint to completed status.
// This marks the sprint as done and archives it from active consideration.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: sprint key
//
// Returns:
//   - *models.Sprint: the updated sprint with new status
//   - error: validation errors or repository errors
//
// Validation:
//   - Current sprint status must allow transition to "completed"
func (s *SprintService) ArchiveSprint(ctx context.Context, key string) (*models.Sprint, error) {
	// Get sprint
	sprint, err := s.GetSprint(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to archive sprint %s: %w", key, err)
	}

	// Validate workflow transition
	if err := s.workflowSvc.ValidateTransition(string(sprint.Status), "completed"); err != nil {
		return nil, fmt.Errorf("cannot archive sprint %s in status %s: %w", key, sprint.Status, err)
	}

	// Update status
	if err := s.repo.UpdateStatus(ctx, sprint.ID, models.SprintStatus("completed")); err != nil {
		return nil, fmt.Errorf("failed to archive sprint %s: %w", key, err)
	}

	// Reload and return updated sprint
	updated, err := s.GetSprint(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to reload sprint %s after archiving: %w", key, err)
	}

	return updated, nil
}
