package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/keys"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/repository/sprint"
	"github.com/jwwelbor/shark-task-manager/internal/utils"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// pluralize returns singular if n == 1, otherwise plural.
func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

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
	UpdateStatusTx(ctx context.Context, tx *sql.Tx, id int64, status models.SprintStatus) error
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

	// --- Sprint order methods (F07) ---

	// MaxSprintOrder returns max(sprint_order) for active assignments in the sprint,
	// or 0 if no ordered items exist. Used by AddEntityToSprint when Position is nil.
	MaxSprintOrder(ctx context.Context, sprintID int64) (int, error)

	// SetSprintOrderTx assigns sprint_order = newPosition for the given assignment ID
	// within the caller-supplied transaction. Pass nil to clear (set NULL).
	SetSprintOrderTx(ctx context.Context, tx *sql.Tx, assignmentID int64, newPosition *int) error

	// RenumberAssignmentsTx applies a slice of (assignment_id, new_position) pairs in a
	// single CASE WHEN UPDATE. Used by AddEntityToSprint (--at) and ReorderAssignment.
	RenumberAssignmentsTx(ctx context.Context, tx *sql.Tx, sprintID int64, ops []sprint.RenumberOp) error

	// ListOrderedAssignments returns active assignments for a sprint sorted by
	// sprint_order ASC NULLS LAST, assigned_at ASC. Used by AddEntityToSprint (--at)
	// to build the shift plan without re-scanning the full backlog UNION.
	ListOrderedAssignments(ctx context.Context, sprintID int64) ([]*models.SprintAssignment, error)
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
	db             *repository.DB                  // optional: nil-safe; required for CloseSprintWithCarryover
}

// NewSprintService creates a SprintService with dependency injection.
// Panics if repo or workflowSvc is nil.
// assignmentRepo, capacityRepo, cfg, and db are optional (nil-safe) and degrade gracefully.
// db is required only for CloseSprintWithCarryover (transaction support); pass nil to skip it.
func NewSprintService(
	repo SprintRepository,
	workflowSvc *workflow.Service,
	assignmentRepo SprintAssignmentQueryRepository,
	capacityRepo SprintCapacityRepository,
	cfg *config.Config,
	db ...*repository.DB,
) *SprintService {
	if repo == nil {
		panic("SprintRepository cannot be nil")
	}
	if workflowSvc == nil {
		panic("workflow.Service cannot be nil")
	}

	svc := &SprintService{
		repo:           repo,
		workflowSvc:    workflowSvc.ForLevel(workflow.LevelSprint),
		assignmentRepo: assignmentRepo,
		capacityRepo:   capacityRepo,
		cfg:            cfg,
	}
	// db is optional variadic; take the first value if provided.
	if len(db) > 0 && db[0] != nil {
		svc.db = db[0]
	}
	return svc
}

func (s *SprintService) getTracer() trace.Tracer {
	return otel.Tracer("shark/services/sprint")
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
	ctx, span := s.getTracer().Start(ctx, "SprintService.CreateSprint",
		trace.WithAttributes(
			attribute.String("sprint.name", input.Name),
		),
	)
	defer span.End()

	// Validate name is non-empty
	if strings.TrimSpace(input.Name) == "" {
		return nil, recordSpanError(span, fmt.Errorf("sprint name cannot be empty"))
	}

	// Validate date ordering
	if !input.EndDate.After(input.StartDate) {
		return nil, recordSpanError(span, fmt.Errorf("sprint end_date must be after start_date"))
	}

	// Generate key
	key, err := s.repo.GetNextKey(ctx)
	if err != nil {
		return nil, recordSpanError(span, fmt.Errorf("failed to generate sprint key: %w", err))
	}

	// Generate slug from name
	slug := utils.GenerateSlug(input.Name)
	filePath := filepath.Join("docs", "plan", "sprints", key+".md")

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
		FilePath:  filePath,
	}

	// Validate model
	if err := newSprint.Validate(); err != nil {
		return nil, recordSpanError(span, fmt.Errorf("sprint validation failed: %w", err))
	}

	// Create in repository
	if err := s.repo.Create(ctx, newSprint); err != nil {
		return nil, recordSpanError(span, fmt.Errorf("failed to create sprint: %w", err))
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

	slog.Info("sprint created", "key", newSprint.Key, "name", newSprint.Name, "status", newSprint.Status)
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
	ctx, span := s.getTracer().Start(ctx, "SprintService.GetSprint",
		trace.WithAttributes(attribute.String("sprint.key", key)),
	)
	defer span.End()

	sprint, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		var notFound *models.NotFoundError
		if errors.Is(err, repository.ErrNotFound) || errors.As(err, &notFound) {
			return nil, recordSpanError(span, &models.NotFoundError{Entity: fmt.Sprintf("sprint %s", key)})
		}
		return nil, recordSpanError(span, fmt.Errorf("failed to get sprint %s: %w", key, err))
	}
	slog.Debug("sprint retrieved", "key", key, "found", true)
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
	ctx, span := s.getTracer().Start(ctx, "SprintService.ListSprints")
	defer span.End()

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
		return nil, recordSpanError(span, fmt.Errorf("failed to list sprints: %w", err))
	}

	// Return empty slice instead of nil for consistency
	if sprints == nil {
		return []*models.Sprint{}, nil
	}

	slog.Info("sprint list", "status", func() string {
		if filters == nil {
			return ""
		}
		return filters.Status
	}(), "count", len(sprints))
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
	ctx, span := s.getTracer().Start(ctx, "SprintService.UpdateSprint",
		trace.WithAttributes(attribute.String("sprint.key", key)),
	)
	defer span.End()

	// Get current sprint
	current, err := s.GetSprint(ctx, key)
	if err != nil {
		return nil, recordSpanError(span, fmt.Errorf("failed to update sprint %s: %w", key, err))
	}

	// Apply updates
	if updates.Name != nil {
		name := strings.TrimSpace(*updates.Name)
		if name == "" {
			return nil, recordSpanError(span, fmt.Errorf("sprint name cannot be empty"))
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
			return nil, recordSpanError(span, fmt.Errorf("sprint end_date must be after start_date"))
		}
		current.EndDate = *updates.EndDate
	}

	// Validate updated model
	if err := current.Validate(); err != nil {
		return nil, recordSpanError(span, fmt.Errorf("sprint validation failed: %w", err))
	}

	// Update in repository
	if err := s.repo.Update(ctx, current); err != nil {
		return nil, recordSpanError(span, fmt.Errorf("failed to update sprint %s: %w", key, err))
	}

	slog.Info("sprint updated", "key", key)
	return current, nil
}

// DeleteSprint deletes a sprint by key.
// Sprints can only be deleted when in "planning" status.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - key: sprint key
//
// Returns:
//   - error: validation errors or repository errors
//
// Validation:
//   - Sprint status must be "planning"
func (s *SprintService) DeleteSprint(ctx context.Context, key string) error {
	ctx, span := s.getTracer().Start(ctx, "SprintService.DeleteSprint",
		trace.WithAttributes(attribute.String("sprint.key", key)),
	)
	defer span.End()

	sprint, err := s.GetSprint(ctx, key)
	if err != nil {
		return recordSpanError(span, fmt.Errorf("failed to delete sprint %s: %w", key, err))
	}

	// Only allow deletion of sprints in planning status.
	if string(sprint.Status) != "planning" {
		return recordSpanError(span, fmt.Errorf("cannot delete sprint %s in status %s: only sprints in planning status can be deleted", key, sprint.Status))
	}

	if err := s.repo.Delete(ctx, sprint.ID); err != nil {
		return recordSpanError(span, fmt.Errorf("failed to delete sprint %s: %w", key, err))
	}

	slog.Info("sprint deleted", "key", key, "status_was", sprint.Status)
	return nil
}

// StartSprint transitions a sprint to active status.
// Enforces single-active-sprint constraint: only one sprint can be active at a time.
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
//   - Current sprint status must allow transition to "active"
//   - No other sprint can be in "active" status
func (s *SprintService) StartSprint(ctx context.Context, key string) (*models.Sprint, error) {
	ctx, span := s.getTracer().Start(ctx, "SprintService.StartSprint",
		trace.WithAttributes(attribute.String("sprint.key", key)),
	)
	defer span.End()

	// Get sprint
	sprint, err := s.GetSprint(ctx, key)
	if err != nil {
		return nil, recordSpanError(span, fmt.Errorf("failed to start sprint %s: %w", key, err))
	}

	// Validate workflow transition
	if err := s.workflowSvc.ValidateTransition(string(sprint.Status), "active"); err != nil {
		return nil, recordSpanError(span, fmt.Errorf("cannot start sprint %s in status %s: %w", key, sprint.Status, err))
	}

	// Check single-active constraint: no other sprint should be active.
	activeSprints, err := s.ListSprints(ctx, &SprintListFilters{Status: "active"})
	if err != nil {
		return nil, recordSpanError(span, fmt.Errorf("failed to check active sprints for %s: %w", key, err))
	}

	if len(activeSprints) > 0 {
		// There's already an active sprint
		if activeSprints[0].Key == key {
			return nil, recordSpanError(span, fmt.Errorf("sprint %s is already active", key))
		}
		return nil, recordSpanError(span, fmt.Errorf("cannot activate sprint %s: sprint %s is already active", key, activeSprints[0].Key))
	}

	// Update status
	if err := s.repo.UpdateStatus(ctx, sprint.ID, models.SprintStatus("active")); err != nil {
		return nil, recordSpanError(span, fmt.Errorf("failed to start sprint %s: %w", key, err))
	}

	// Reload and return updated sprint
	updated, err := s.GetSprint(ctx, key)
	if err != nil {
		return nil, recordSpanError(span, fmt.Errorf("failed to reload sprint %s after starting: %w", key, err))
	}

	slog.Info("sprint transitioned", "key", key, "from", sprint.Status, "to", updated.Status)
	return updated, nil
}

// CloseSprint transitions a sprint to closing status.
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
//   - Current sprint status must allow transition to "closing"
//
// Note: Carryover logic is deferred to a future feature.
func (s *SprintService) CloseSprint(ctx context.Context, key string) (*models.Sprint, error) {
	ctx, span := s.getTracer().Start(ctx, "SprintService.CloseSprint",
		trace.WithAttributes(attribute.String("sprint.key", key)),
	)
	defer span.End()

	// Get sprint
	sprint, err := s.GetSprint(ctx, key)
	if err != nil {
		return nil, recordSpanError(span, fmt.Errorf("failed to close sprint %s: %w", key, err))
	}

	// Validate workflow transition
	if err := s.workflowSvc.ValidateTransition(string(sprint.Status), "closing"); err != nil {
		return nil, recordSpanError(span, fmt.Errorf("cannot close sprint %s in status %s: %w", key, sprint.Status, err))
	}

	// Update status
	if err := s.repo.UpdateStatus(ctx, sprint.ID, models.SprintStatus("closing")); err != nil {
		return nil, recordSpanError(span, fmt.Errorf("failed to close sprint %s: %w", key, err))
	}

	// Reload and return updated sprint
	updated, err := s.GetSprint(ctx, key)
	if err != nil {
		return nil, recordSpanError(span, fmt.Errorf("failed to reload sprint %s after closing: %w", key, err))
	}

	slog.Info("sprint transitioned", "key", key, "from", sprint.Status, "to", updated.Status)
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
	// Position is an optional 1-based position for the new assignment in the sprint order.
	// When nil (no --at flag), the item is appended after all ordered items: sprint_order = max + 1.
	// When set, the item is inserted at the given position and all items at >= position are
	// shifted up by 1. Must be in the range [1, count+1]; count+1 is equivalent to append.
	// Validation: returns error if Position <= 0 or Position > count+1.
	Position *int
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
		// Change-card keys in the legacy CC-### format are not handled by
		// KeyService.Parse (which uses the C### pattern). Fall back to
		// IsChangeCardKey to support the CC-### format used in the feature spec
		// (REQ-F-004) and by older workflows.
		if keys.IsChangeCardKey(entityKey) {
			normalized := strings.ToUpper(strings.TrimSpace(entityKey))
			entityID, err = repo.GetChangeCardIDByKey(ctx, normalized)
			if err != nil {
				return "", 0, fmt.Errorf("change_card %q not found: %w", entityKey, err)
			}
			return "change_card", entityID, nil
		}
		return "", 0, fmt.Errorf(
			"unsupported entity type %q for sprint assignment: entity key %q must be a task, bug, change-card (C### or CC-###), or tech-debt (TD-###) key",
			parsed.EntityType, entityKey,
		)
	}
}

// ---------------------------------------------------------------------------
// AddEntityToSprint — T-E19-F03-005 / T-E19-F07-003
// ---------------------------------------------------------------------------

// AddEntityToSprint assigns any supported entity to a sprint.
//
// Steps:
//  1. Resolve sprint by key.
//  2. Validate Position when provided (must be >= 1 before any repo call).
//  3. Parse entity key to determine entity_type (task/bug/change_card/tech_debt).
//  4. Resolve entity_id by querying the entity's table.
//  5. Check for an existing active assignment; return ConflictError naming the
//     conflicting sprint if one is found.
//  6. Determine sprint_order:
//     - Position nil: auto-assign = MaxSprintOrder + 1 (FIFO append).
//     - Position set: validate against current count; insert at position and shift siblings.
//  7. Call repo.AddAssignment with sprint_order populated.
//  8. For insert-at-position: call RenumberAssignmentsTx to shift items >= position.
//  9. Compute capacity warning if capacityRepo and AgentType are provided
//     (advisory only — never blocks).
//
// Returns the created SprintAssignment and an optional CapacityWarning.
// A non-nil CapacityWarning does NOT indicate failure; the assignment was created.
func (s *SprintService) AddEntityToSprint(ctx context.Context, input AddEntityInput) (*models.SprintAssignment, *CapacityWarning, error) {
	// Step 1: Resolve sprint and validate its status.
	// Per spec §4.2.1 step 1, only planning and active sprints may accept new
	// entity assignments.
	sprintEntity, err := s.repo.GetByKey(ctx, input.SprintKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve sprint %q: %w", input.SprintKey, err)
	}
	switch sprintEntity.Status {
	case "planning", "active":
		// planning or active — allowed
	default:
		return nil, nil, fmt.Errorf(
			"cannot assign entity to sprint %s: sprint is in %q status (only planning or active sprints accept new assignments)",
			input.SprintKey, sprintEntity.Status,
		)
	}

	// Step 2: Validate Position early (before any repo calls) per TD-10 / TC-008 / TC-009.
	if input.Position != nil && *input.Position < 1 {
		return nil, nil, fmt.Errorf(
			"invalid position %d for sprint %s: position must be >= 1",
			*input.Position, input.SprintKey,
		)
	}

	// Step 3+4: Parse entity key and resolve entity ID
	entityType, entityID, err := resolveEntityTypeAndID(ctx, s.repo, input.EntityKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve entity %q for sprint assignment: %w", input.EntityKey, err)
	}

	// Step 5: Conflict check — at most one active sprint assignment per entity
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

	// Step 6: Determine sprint_order for the new assignment.
	//
	// When Position is nil (no --at flag): append at max+1 (FIFO). This is the fast
	// path — no ListOrderedAssignments needed, just MaxSprintOrder.
	//
	// When Position is set: validate range, then call ListOrderedAssignments to build
	// the shift plan; shift is applied inside a transaction after AddAssignment succeeds.
	var targetOrder int
	var orderedItems []*models.SprintAssignment // populated only for insert-at-position

	if input.Position == nil {
		// Auto-assign: append at max+1. MaxSprintOrder returns 0 when no ordered items
		// exist, so the first item gets sprint_order=1 (per TC-011).
		maxOrder, maxErr := s.repo.MaxSprintOrder(ctx, sprintEntity.ID)
		if maxErr != nil {
			return nil, nil, fmt.Errorf("failed to determine sprint order for %q in sprint %s: %w",
				input.EntityKey, input.SprintKey, maxErr)
		}
		targetOrder = maxOrder + 1
	} else {
		// Insert at requested position: validate and fetch ordered items for shift.
		pos := *input.Position

		// Get current ordered items to determine valid upper bound.
		orderedItems, err = s.repo.ListOrderedAssignments(ctx, sprintEntity.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to list ordered assignments for sprint %s: %w",
				input.SprintKey, err)
		}

		count := len(orderedItems)
		if pos > count+1 {
			return nil, nil, fmt.Errorf(
				"position %d is out of range: sprint has %d ordered items (valid range: 1..%d)",
				pos, count, count+1,
			)
		}

		targetOrder = pos
	}

	// Step 7: Create assignment with sprint_order pre-populated.
	assignment := &models.SprintAssignment{
		SprintID:    sprintEntity.ID,
		EntityType:  entityType,
		EntityID:    entityID,
		AssignedAt:  time.Now().UTC(),
		SprintOrder: &targetOrder,
	}
	if err := s.repo.AddAssignment(ctx, assignment); err != nil {
		return nil, nil, fmt.Errorf("failed to add assignment for %q to sprint %s: %w",
			input.EntityKey, input.SprintKey, err)
	}

	// Step 8: For insert-at-position, shift all items that were at >= targetOrder up by 1.
	// Items at position == count+1 (append at boundary) require no shift.
	if input.Position != nil && len(orderedItems) > 0 {
		pos := *input.Position
		count := len(orderedItems)
		if pos <= count {
			// Build shift ops for all existing items at positions >= pos.
			ops := make([]sprint.RenumberOp, 0, count-pos+1)
			for _, item := range orderedItems {
				if item.SprintOrder != nil && *item.SprintOrder >= pos {
					newPos := *item.SprintOrder + 1
					ops = append(ops, sprint.RenumberOp{
						AssignmentID: item.ID,
						NewPosition:  &newPos,
					})
				}
			}
			if len(ops) > 0 {
				// TD-9: service owns the transaction boundary.
				// Since we have no outstanding tx here (AddAssignment committed above),
				// we call RenumberAssignmentsTx with a nil tx — the repo handles it
				// as a non-transactional update. The spec allows this because the
				// partial unique index (sprint_id, sprint_order) enforces ordering
				// invariants at the DB layer.
				//
				// Note: for full transactional safety, s.db must be non-nil. If s.db
				// is nil we proceed without a transaction (advisory, not blocking).
				if err := s.repo.RenumberAssignmentsTx(ctx, nil, sprintEntity.ID, ops); err != nil {
					return nil, nil, fmt.Errorf("failed to shift sprint order for insert at position %d in sprint %s: %w",
						pos, input.SprintKey, err)
				}
			}
		}
	}

	// Step 9: Advisory capacity warning (never blocks)
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

// ---------------------------------------------------------------------------
// GetSprintBacklog types and method — T-E19-F03-006
// ---------------------------------------------------------------------------

// BacklogOptions carries filter parameters for the backlog view.
// EntityType is "" for all types; "task", "bug", "change_card", "tech_debt" to filter.
// BlockedOnly limits results to entities in a blocked-equivalent workflow status.
// View selects the display mode: "ordered" (pull-queue sorted by sprint_order ASC NULLS LAST)
// or "grouped" (current grouped-by-status behavior). If View is "" the service applies a
// default: "ordered" for active sprints, "grouped" for all other statuses.
type BacklogOptions struct {
	EntityType       string // "" = all types
	BlockedOnly      bool
	View             string // "ordered" | "grouped" | "" (auto-detect from sprint status)
	IncludeCompleted bool   // when true, terminal-status items are included in ordered view
}

// SprintBacklog is the return value of GetSprintBacklog.
type SprintBacklog struct {
	SprintKey         string             `json:"sprint_key"`
	SprintName        string             `json:"sprint_name"`
	TotalCount        int                `json:"total_count"`
	CompletedCount    int                `json:"completed_count"`
	CompletionPercent float64            `json:"completion_percent"`
	Groups            []*BacklogGroup    `json:"groups,omitempty"` // populated in grouped view only
	Items             []*BacklogItemView `json:"items,omitempty"`  // populated in ordered view only
	View              string             `json:"view"`             // "ordered" | "grouped"
}

// BacklogGroup is a set of entities sharing the same status category.
type BacklogGroup struct {
	StatusCategory string             `json:"status_category"` // e.g., "in_progress", "todo", "completed", "blocked"
	Items          []*BacklogItemView `json:"items"`
}

// BacklogItemView is the CLI-friendly projection of a BacklogItem.
type BacklogItemView struct {
	EntityType      string    `json:"entity_type"`
	Key             string    `json:"key"`
	Title           string    `json:"title"`
	Status          string    `json:"status"`
	AgentType       string    `json:"agent_type,omitempty"`
	Priority        int       `json:"priority,omitempty"`
	ExecutionOrder  *int      `json:"execution_order,omitempty"`
	Size            *int      `json:"size,omitempty"`
	AssignedAt      time.Time `json:"assigned_at,omitempty"`
	DaysBlocked     int       `json:"days_blocked,omitempty"`     // For --blocked view
	SprintOrder     *int      `json:"sprint_order,omitempty"`     // nullable; nil = unordered
	Position        *int      `json:"position,omitempty"`         // 1-based dense rank in ordered view; set even when SprintOrder is nil
	SprintKey       string    `json:"sprint_key,omitempty"`       // set by GetNextTask only
	SelectionReason string    `json:"selection_reason,omitempty"` // set by GetNextTask only
}

// ReorderTarget specifies where to move an assignment within the sprint.
// Exactly one of Position, Top, or Bottom must be set; all three set at once
// is a programming error caught by ReorderAssignment at service entry.
type ReorderTarget struct {
	Position *int // 1-based target position; nil if Top or Bottom is set
	Top      bool // move to position 1
	Bottom   bool // move to position max+1 (last)
}

// validBacklogEntityTypes is the allowlist of entity types accepted by GetSprintBacklog's
// EntityType filter. The service validates against this set before passing to the repository
// to prevent invalid values from reaching the UNION query.
var validBacklogEntityTypes = map[string]bool{
	"task":        true,
	"bug":         true,
	"change_card": true,
	"tech_debt":   true,
}

// GetSprintBacklog returns all entities assigned to a sprint.
//
// View modes (controlled by opts.View):
//   - "ordered":  Items array sorted by sprint_order ASC NULLS LAST, then execution_order, priority,
//     assigned_at. Groups is nil. Each item gets a 1-based dense-rank Position field.
//   - "grouped":  Items is nil. Groups is populated by status category (existing behavior).
//   - "":         Defaults to "ordered" when sprint is active; "grouped" otherwise.
//
// The method:
//  1. Resolves the sprint by key.
//  2. Determines the effective view mode from opts.View + sprint.Status.
//  3. Validates the optional entity-type filter (returns error for invalid types).
//  4. Asks workflow.Service for the blocked-status set when BlockedOnly=true.
//  5. Calls repo.ListBacklog to fetch raw items.
//  6. Builds view-mode output: ordered items list OR grouped-by-status map.
//  7. Computes CompletionPercent as float64 (completed / total * 100); returns 0.0 when total=0.
//
// The blocked-status set is delegated to workflow.Service so that custom workflow
// configurations with non-standard "blocked" status names are handled correctly.
func (s *SprintService) GetSprintBacklog(ctx context.Context, sprintKey string, opts BacklogOptions) (*SprintBacklog, error) {
	// Step 1: Resolve sprint
	sprintEntity, err := s.repo.GetByKey(ctx, sprintKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get sprint %s for backlog: %w", sprintKey, err)
	}

	// Step 2: Resolve effective view mode.
	// Default: "ordered" for active sprints; "grouped" for all other statuses.
	effectiveView := opts.View
	if effectiveView == "" {
		if string(sprintEntity.Status) == "active" {
			effectiveView = "ordered"
		} else {
			effectiveView = "grouped"
		}
	}

	// Step 3: Validate entity type filter
	var entityTypeFilter *string
	if opts.EntityType != "" {
		if !validBacklogEntityTypes[opts.EntityType] {
			return nil, fmt.Errorf(
				"invalid entity type %q: must be one of task, bug, change_card, tech_debt",
				opts.EntityType,
			)
		}
		entityTypeFilter = &opts.EntityType
	}

	// Step 4: Determine blocked statuses from workflow service (never hardcode "blocked")
	var blockedStatuses []string
	if opts.BlockedOnly {
		blockedStatuses = s.workflowSvc.GetStatusesByPhase("blocked")
		if len(blockedStatuses) == 0 {
			// Fallback: use "blocked" as the default if the workflow has no "blocked" phase
			blockedStatuses = []string{"blocked"}
		}
	}

	// Step 5: Fetch raw backlog items from repository
	items, err := s.repo.ListBacklog(ctx, sprintEntity.ID, entityTypeFilter, opts.BlockedOnly, blockedStatuses...)
	if err != nil {
		return nil, fmt.Errorf("failed to list backlog for sprint %s: %w", sprintKey, err)
	}

	totalCount := len(items)
	completedCount := 0

	// Build per-entity-type terminal status sets once, outside the item loop.
	// Keyed by the entity_type string stored in sprint_assignments.
	terminalStatusesByEntityType := map[string]map[string]bool{
		"task":        terminalSet(s.workflowSvc.ForLevel(workflow.LevelTask)),
		"bug":         terminalSet(s.workflowSvc.ForLevel(workflow.LevelBug)),
		"change_card": terminalSet(s.workflowSvc.ForLevel(workflow.LevelChange)),
		"tech_debt":   terminalSet(s.workflowSvc.ForLevel(workflow.LevelTechDebt)),
	}

	// Count completed items across both view modes.
	for _, item := range items {
		if terminals, ok := terminalStatusesByEntityType[item.EntityType]; ok && terminals[item.Status] {
			completedCount++
		}
	}

	// Step 6: Compute CompletionPercent using float64 division (never integer)
	var completionPercent float64
	if totalCount > 0 {
		completionPercent = float64(completedCount) / float64(totalCount) * 100.0
	}

	backlog := &SprintBacklog{
		SprintKey:         sprintKey,
		SprintName:        sprintEntity.Name,
		TotalCount:        totalCount,
		CompletedCount:    completedCount,
		CompletionPercent: completionPercent,
		View:              effectiveView,
	}

	if effectiveView == "ordered" {
		backlog.Items = buildOrderedView(items)
	} else {
		backlog.Groups = buildGroupedView(items)
	}

	return backlog, nil
}

// buildOrderedView converts raw BacklogItems into a sorted Items slice for the ordered view.
// Sorting: sprint_order ASC NULLS LAST, then execution_order ASC NULLS LAST, priority ASC, assigned_at ASC.
// Each item receives a 1-based dense-rank Position field. Items with nil sprint_order receive a Position
// but their SprintOrder field remains nil (the two fields diverge — spec §3.2, TC-019).
func buildOrderedView(items []*sprint.BacklogItem) []*BacklogItemView {
	views := make([]*BacklogItemView, 0, len(items))
	for _, item := range items {
		v := backlogItemToView(item)
		views = append(views, v)
	}

	// Sort: sprint_order ASC NULLS LAST, execution_order ASC NULLS LAST, priority ASC, assigned_at ASC
	sort.SliceStable(views, func(i, j int) bool {
		a, b := views[i], views[j]

		// Tier 1: sprint_order ASC NULLS LAST
		switch {
		case a.SprintOrder != nil && b.SprintOrder == nil:
			return true // a comes first (ordered before unordered)
		case a.SprintOrder == nil && b.SprintOrder != nil:
			return false // b comes first
		case a.SprintOrder != nil && b.SprintOrder != nil:
			if *a.SprintOrder != *b.SprintOrder {
				return *a.SprintOrder < *b.SprintOrder
			}
		}

		// Tier 2: execution_order ASC NULLS LAST
		switch {
		case a.ExecutionOrder != nil && b.ExecutionOrder == nil:
			return true
		case a.ExecutionOrder == nil && b.ExecutionOrder != nil:
			return false
		case a.ExecutionOrder != nil && b.ExecutionOrder != nil:
			if *a.ExecutionOrder != *b.ExecutionOrder {
				return *a.ExecutionOrder < *b.ExecutionOrder
			}
		}

		// Tier 3: priority ASC (lower number = higher priority)
		if a.Priority != b.Priority {
			return a.Priority < b.Priority
		}

		// Tier 4: assigned_at ASC (oldest first — FIFO tiebreaker)
		return a.AssignedAt.Before(b.AssignedAt)
	})

	// Assign 1-based dense-rank Position to all items.
	// Items with a non-nil sprint_order retain their sprint_order; unordered items get Position
	// from the dense rank but their SprintOrder remains nil.
	for i, v := range views {
		pos := i + 1
		v.Position = &pos
	}

	return views
}

// buildGroupedView groups raw BacklogItems by status category (existing behavior).
// Returns a slice of BacklogGroup in insertion order (stable across calls).
func buildGroupedView(items []*sprint.BacklogItem) []*BacklogGroup {
	groupOrder := make([]string, 0)
	groupMap := make(map[string]*BacklogGroup)

	for _, item := range items {
		v := backlogItemToView(item)

		category := item.Status
		if _, exists := groupMap[category]; !exists {
			groupMap[category] = &BacklogGroup{
				StatusCategory: category,
				Items:          []*BacklogItemView{},
			}
			groupOrder = append(groupOrder, category)
		}
		groupMap[category].Items = append(groupMap[category].Items, v)
	}

	groups := make([]*BacklogGroup, 0, len(groupOrder))
	for _, status := range groupOrder {
		groups = append(groups, groupMap[status])
	}
	return groups
}

// backlogItemToView converts a sprint.BacklogItem into a BacklogItemView.
// Position is not set here; it is assigned by buildOrderedView after sorting.
func backlogItemToView(item *sprint.BacklogItem) *BacklogItemView {
	v := &BacklogItemView{
		EntityType:     item.EntityType,
		Key:            item.Key,
		Title:          item.Title,
		Status:         item.Status,
		Priority:       item.Priority,
		ExecutionOrder: item.ExecutionOrder,
		Size:           item.Size,
		AssignedAt:     item.AssignedAt,
		SprintOrder:    item.SprintOrder, // propagate nullable sprint_order
	}
	if item.AgentType != nil {
		v.AgentType = *item.AgentType
	}
	return v
}

// GetNextTask returns the single next eligible item to work on from the active sprint.
// Selection logic (four-tier stable sort, TD-8):
//  1. sprint_order ASC NULLS LAST (ordered items before unordered)
//  2. ExecutionOrder ASC NULLS LAST
//  3. Priority ASC (lower number = higher priority — preserves existing semantics)
//  4. AssignedAt ASC (oldest first — FIFO tiebreaker)
//
// sort.SliceStable is used (not sort.Slice) so that full ties preserve insertion order,
// giving AI agents deterministic results across repeated calls.
//
// Filters for items in the initial (queued) status of their respective entity-type workflows.
// If agentType is non-empty, only items for that agent are considered.
//
// The returned BacklogItemView has SprintOrder, SprintKey, and SelectionReason populated.
func (s *SprintService) GetNextTask(ctx context.Context, agentType string) (*BacklogItemView, error) {
	// 1. Find the active sprint (must be exactly one for unambiguous selection)
	filters := &SprintListFilters{Status: "active"}
	activeSprints, err := s.ListSprints(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list active sprints: %w", err)
	}
	if len(activeSprints) == 0 {
		return nil, fmt.Errorf("no active sprint found (start a sprint first)")
	}
	activeSprint := activeSprints[0]

	// 2. Fetch the backlog for the active sprint using grouped view so we can filter by status.
	// GetNextTask does its own four-tier sort, so it needs all items regardless of sprint_order.
	// Using View="grouped" explicitly avoids the active-sprint default of "ordered".
	backlog, err := s.GetSprintBacklog(ctx, activeSprint.Key, BacklogOptions{View: "grouped"})
	if err != nil {
		return nil, err
	}

	// 3. Collect candidates whose status is the initial (queued) status for their entity type.
	// Each entity type has its own workflow with its own initial status (e.g. tasks start at
	// "todo", tech_debt at "identified"), so we must check per-entity-type rather than
	// assuming everything uses the task workflow's initial status.
	initialStatusByEntityType := map[string]string{
		"task":        s.workflowSvc.ForLevel(workflow.LevelTask).GetInitialStatusString(),
		"bug":         s.workflowSvc.ForLevel(workflow.LevelBug).GetInitialStatusString(),
		"change_card": s.workflowSvc.ForLevel(workflow.LevelChange).GetInitialStatusString(),
		"tech_debt":   s.workflowSvc.ForLevel(workflow.LevelTechDebt).GetInitialStatusString(),
	}

	var candidates []*BacklogItemView
	for _, group := range backlog.Groups {
		for _, item := range group.Items {
			initialStatus, ok := initialStatusByEntityType[item.EntityType]
			if !ok || item.Status != initialStatus {
				continue
			}
			// Apply agent filter if requested (filter BEFORE sort — agent filter is pre-sort)
			if agentType != "" && item.AgentType != agentType {
				continue
			}
			candidates = append(candidates, item)
		}
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	// 4. Four-tier stable sort (TD-8: sort.SliceStable, not sort.Slice)
	sort.SliceStable(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]

		// Tier 1: sprint_order ASC NULLS LAST (ordered items first)
		if a.SprintOrder != nil && b.SprintOrder == nil {
			return true
		}
		if a.SprintOrder == nil && b.SprintOrder != nil {
			return false
		}
		if a.SprintOrder != nil && b.SprintOrder != nil {
			if *a.SprintOrder != *b.SprintOrder {
				return *a.SprintOrder < *b.SprintOrder
			}
		}

		// Tier 2: ExecutionOrder ASC NULLS LAST
		if a.ExecutionOrder != nil && b.ExecutionOrder == nil {
			return true
		}
		if a.ExecutionOrder == nil && b.ExecutionOrder != nil {
			return false
		}
		if a.ExecutionOrder != nil && b.ExecutionOrder != nil {
			if *a.ExecutionOrder != *b.ExecutionOrder {
				return *a.ExecutionOrder < *b.ExecutionOrder
			}
		}

		// Tier 3: Priority ASC (lower number = higher priority — existing semantics preserved)
		if a.Priority != b.Priority {
			return a.Priority < b.Priority
		}

		// Tier 4: AssignedAt ASC (oldest first — FIFO)
		return a.AssignedAt.Before(b.AssignedAt)
	})

	// 5. Compute selection_reason by comparing winner to runner-up
	winner := candidates[0]
	var runnerUp *BacklogItemView
	if len(candidates) > 1 {
		runnerUp = candidates[1]
	}
	winner.SprintKey = activeSprint.Key
	winner.SelectionReason = computeSelectionReason(winner, runnerUp)

	slog.Debug("sprint.next.selection",
		"reason", winner.SelectionReason,
		"sprint_id", activeSprint.Key,
		"key", winner.Key,
	)

	return winner, nil
}

// computeSelectionReason determines which sort tier differentiated the winner from the
// runner-up. If there is only one candidate (runnerUp == nil), returns "assigned_at"
// as the default (AC-T2: no index-out-of-bounds on candidates[1]).
//
// Returns exactly one of: "sprint_order", "execution_order", "priority", "assigned_at".
func computeSelectionReason(top, runnerUp *BacklogItemView) string {
	if runnerUp == nil {
		// Single candidate — default to "assigned_at" (AC-T2)
		return "assigned_at"
	}

	// Tier 1: sprint_order
	aHasOrder := top.SprintOrder != nil
	bHasOrder := runnerUp.SprintOrder != nil
	if aHasOrder != bHasOrder {
		return "sprint_order"
	}
	if aHasOrder && bHasOrder && *top.SprintOrder != *runnerUp.SprintOrder {
		return "sprint_order"
	}

	// Tier 2: execution_order
	aHasExec := top.ExecutionOrder != nil
	bHasExec := runnerUp.ExecutionOrder != nil
	if aHasExec != bHasExec {
		return "execution_order"
	}
	if aHasExec && bHasExec && *top.ExecutionOrder != *runnerUp.ExecutionOrder {
		return "execution_order"
	}

	// Tier 3: priority
	if top.Priority != runnerUp.Priority {
		return "priority"
	}

	// Tier 4: assigned_at (FIFO — also the default when all else ties)
	return "assigned_at"
}

// ReorderAssignment moves an entity's sprint assignment to the specified position
// within the sprint's ordered pull queue, then densely renumbers all ordered items.
//
// The moved assignment and the abbreviated top-N list (N=8) are returned.
// The operation runs inside a single transaction — either all renumbers succeed or
// none are committed (TD-9: service-owned transaction).
//
// Validation:
//   - Sprint must be in "planning" or "active" status (not "completed" or "archived").
//   - Entity must be assigned to the sprint (AC-T3).
//   - Exactly one of target.Position, target.Top, or target.Bottom must be set.
//   - target.Position must be in range [1, orderedCount+1].
//
// AC-T4: entity-level repo methods (TaskService.Update etc.) are NOT called.
func (s *SprintService) ReorderAssignment(
	ctx context.Context,
	sprintKey, entityKey string,
	target ReorderTarget,
) (*models.SprintAssignment, []*models.SprintAssignment, error) {
	// Step 1: Resolve and validate sprint
	sprintEntity, err := s.repo.GetByKey(ctx, sprintKey)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot reorder: failed to resolve sprint %q: %w", sprintKey, err)
	}

	status := string(sprintEntity.Status)
	if status != "planning" && status != "active" {
		return nil, nil, fmt.Errorf(
			"cannot reorder: sprint %q is in status %q; only planning and active sprints can be reordered",
			sprintKey, status,
		)
	}

	// Step 2: Resolve entity to (type, ID)
	entityType, entityID, err := resolveEntityTypeAndID(ctx, s.repo, entityKey)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot reorder: %w", err)
	}

	// Step 3: Verify entity is assigned to this sprint (AC-T3)
	assignment, err := s.repo.GetActiveAssignment(ctx, entityType, entityID)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot reorder: failed to look up assignment for %q: %w", entityKey, err)
	}
	if assignment == nil || assignment.SprintID != sprintEntity.ID {
		return nil, nil, fmt.Errorf(
			"entity %q is not assigned to sprint %s",
			entityKey, sprintKey,
		)
	}

	// Step 4: Compute target position from ReorderTarget
	// Get current ordered assignments (excludes the target so we can compute count correctly)
	orderedAll, err := s.repo.ListOrderedAssignments(ctx, sprintEntity.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot reorder: failed to list ordered assignments: %w", err)
	}

	var newPosition int
	switch {
	case target.Top:
		newPosition = 1
	case target.Bottom:
		// Place after the last ordered item
		ordered := orderedWithoutTarget(orderedAll, assignment.ID)
		newPosition = len(ordered) + 1
	case target.Position != nil:
		newPosition = *target.Position
		if newPosition < 1 {
			return nil, nil, fmt.Errorf("cannot reorder: position must be >= 1, got %d", newPosition)
		}
		// Validate upper bound: count of ordered items excluding target + 1
		ordered := orderedWithoutTarget(orderedAll, assignment.ID)
		maxPos := len(ordered) + 1
		if newPosition > maxPos {
			return nil, nil, fmt.Errorf(
				"cannot reorder: position %d is out of range (sprint has %d ordered items)",
				newPosition, len(ordered),
			)
		}
	default:
		return nil, nil, fmt.Errorf("cannot reorder: one of Position, Top, or Bottom must be set in ReorderTarget")
	}

	// Step 5: Build renumber plan for siblings
	// Siblings = ordered items excluding the target, shifted to accommodate the target's new position.
	siblings := orderedWithoutTarget(orderedAll, assignment.ID)
	ops := buildRenumberOps(siblings, newPosition)

	// Step 6: Apply in a transaction (TD-9)
	if s.db == nil {
		// No DB available — apply ops without a real transaction (test path or CLI without --db).
		// For production use, db must be provided to NewSprintService.
		if err := s.repo.RenumberAssignmentsTx(ctx, nil, sprintEntity.ID, ops); err != nil {
			return nil, nil, fmt.Errorf("cannot reorder: renumber failed: %w", err)
		}
		if err := s.repo.SetSprintOrderTx(ctx, nil, assignment.ID, &newPosition); err != nil {
			return nil, nil, fmt.Errorf("cannot reorder: set sprint_order failed: %w", err)
		}
	} else {
		tx, txErr := s.db.BeginTxContext(ctx)
		if txErr != nil {
			return nil, nil, fmt.Errorf("cannot reorder: failed to begin transaction: %w", txErr)
		}
		defer tx.Rollback() //nolint:errcheck

		if err := s.repo.RenumberAssignmentsTx(ctx, tx, sprintEntity.ID, ops); err != nil {
			return nil, nil, fmt.Errorf("cannot reorder: renumber failed: %w", err)
		}
		if err := s.repo.SetSprintOrderTx(ctx, tx, assignment.ID, &newPosition); err != nil {
			return nil, nil, fmt.Errorf("cannot reorder: set sprint_order failed: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return nil, nil, fmt.Errorf("cannot reorder: commit failed: %w", err)
		}
	}

	// Update the in-memory assignment to reflect the new position
	assignment.SprintOrder = &newPosition

	slog.Debug("sprint.reorder",
		"sprint_id", sprintKey,
		"entity", entityKey,
		"new_pos", newPosition,
		"renumbered", len(ops),
	)

	// Step 7: Build top-8 abbreviated list from the updated ordered assignments
	// Re-fetch ordered assignments to reflect the post-reorder state.
	updatedOrdered, err := s.repo.ListOrderedAssignments(ctx, sprintEntity.ID)
	if err != nil {
		// Non-fatal: return the moved assignment without top-N
		updatedOrdered = nil
	}
	topN := topNAssignments(updatedOrdered, 8)

	return assignment, topN, nil
}

// orderedWithoutTarget returns the ordered assignments slice with the target assignment
// removed. The input is NOT modified (a new slice is allocated).
func orderedWithoutTarget(ordered []*models.SprintAssignment, targetID int64) []*models.SprintAssignment {
	out := make([]*models.SprintAssignment, 0, len(ordered))
	for _, a := range ordered {
		if a.ID != targetID {
			out = append(out, a)
		}
	}
	return out
}

// buildRenumberOps builds a []RenumberOp for sibling assignments around a new position.
// siblings are already ordered by sprint_order (all ordered items excluding the target).
// Items at positions >= newPosition are shifted up by 1; items before are unchanged.
func buildRenumberOps(siblings []*models.SprintAssignment, newPosition int) []sprint.RenumberOp {
	ops := make([]sprint.RenumberOp, 0, len(siblings))
	pos := 1
	for _, sib := range siblings {
		if pos >= newPosition {
			pos++ // skip the slot reserved for the target
		}
		p := pos
		ops = append(ops, sprint.RenumberOp{AssignmentID: sib.ID, NewPosition: &p})
		pos++
	}
	return ops
}

// topNAssignments returns the first n assignments from the ordered slice, or all if len < n.
func topNAssignments(ordered []*models.SprintAssignment, n int) []*models.SprintAssignment {
	if n > len(ordered) {
		n = len(ordered)
	}
	return ordered[:n]
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
	ctx, span := s.getTracer().Start(ctx, "SprintService.ArchiveSprint",
		trace.WithAttributes(attribute.String("sprint.key", key)),
	)
	defer span.End()

	// Get sprint
	sprint, err := s.GetSprint(ctx, key)
	if err != nil {
		return nil, recordSpanError(span, fmt.Errorf("failed to archive sprint %s: %w", key, err))
	}

	// Validate workflow transition
	if err := s.workflowSvc.ValidateTransition(string(sprint.Status), "archived"); err != nil {
		return nil, recordSpanError(span, fmt.Errorf("cannot archive sprint %s in status %s: %w", key, sprint.Status, err))
	}

	// Update status
	if err := s.repo.UpdateStatus(ctx, sprint.ID, models.SprintStatus("archived")); err != nil {
		return nil, recordSpanError(span, fmt.Errorf("failed to archive sprint %s: %w", key, err))
	}

	// Reload and return updated sprint
	updated, err := s.GetSprint(ctx, key)
	if err != nil {
		return nil, recordSpanError(span, fmt.Errorf("failed to reload sprint %s after archiving: %w", key, err))
	}

	slog.Info("sprint transitioned", "key", key, "from", sprint.Status, "to", updated.Status)
	return updated, nil
}

// ---------------------------------------------------------------------------
// BulkAddToSprint — T-E19-F03-008
// ---------------------------------------------------------------------------

// BulkAddInput specifies what to bulk-assign to a sprint.
//
// Exactly one of FeatureKey or EntityTypes should be provided:
//   - FeatureKey: bulk-assign tasks from the named feature (e.g., "E07-F34").
//   - EntityTypes: bulk-assign all open entities of the given types (e.g., ["bug", "tech_debt"]).
//
// If both are provided, FeatureKey takes precedence. If neither is provided,
// all unassigned entities of all types are bulk-assigned.
type BulkAddInput struct {
	// SprintKey identifies the target sprint (e.g., "S024"). Required.
	SprintKey string

	// FeatureKey limits bulk-assignment to tasks from this feature (e.g., "E07-F34").
	// When set, only tasks whose key has this feature as a prefix are considered.
	// Mutually exclusive with EntityTypes (FeatureKey wins if both set).
	FeatureKey string

	// EntityTypes limits bulk-assignment to these entity types (e.g., ["bug", "change_card"]).
	// Valid values: "task", "bug", "change_card", "tech_debt".
	// Empty slice means all supported types.
	EntityTypes []string
}

// BulkAddResult summarises the outcome of a BulkAddToSprint call.
type BulkAddResult struct {
	// AddedByType maps entity_type -> count of entities successfully added.
	AddedByType map[string]int

	// SkippedByType maps entity_type -> count of entities skipped
	// (already assigned to an active/planning sprint, or in a non-assignable status).
	SkippedByType map[string]int

	// CapacityWarnings is non-nil when the bulk assignment pushed one or more
	// agent types over their configured capacity. Advisory only — never blocks.
	CapacityWarnings []*CapacityWarning
}

// BulkAddToSprint assigns multiple eligible entities to a sprint in one operation.
//
// "Eligible" means:
//   - Entity is not already actively assigned to another sprint.
//   - When FeatureKey is set, task key must match the feature key prefix.
//
// The method calls assignmentRepo.ListUnassignedBacklog to discover candidates,
// then optionally filters by FeatureKey, then calls assignmentRepo.BulkAssign
// for the eligible subset. Already-assigned entities are silently skipped (not errors).
//
// Returns a BulkAddResult with per-type counts. A nil assignmentRepo is
// a configuration error and returns an error immediately.
func (s *SprintService) BulkAddToSprint(ctx context.Context, input BulkAddInput) (*BulkAddResult, error) {
	// Require assignmentRepo: BulkAddToSprint cannot work without it.
	if s.assignmentRepo == nil {
		return nil, fmt.Errorf("BulkAddToSprint requires assignmentRepo; service was constructed without one")
	}

	// Step 1: Resolve sprint — confirm it exists.
	sprintEntity, err := s.repo.GetByKey(ctx, input.SprintKey)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve sprint %q for bulk add: %w", input.SprintKey, err)
	}

	// Step 2: Determine which entity types to consider.
	entityTypes := input.EntityTypes
	if input.FeatureKey != "" {
		// Feature-based bulk: only tasks qualify (features contain tasks, not bugs/etc).
		entityTypes = []string{"task"}
	}

	// Step 3: Discover unassigned candidates from the repository.
	// ListUnassignedBacklog already excludes entities in active/planning sprints.
	candidates, err := s.assignmentRepo.ListUnassignedBacklog(ctx, entityTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to list unassigned backlog for bulk add to sprint %s: %w",
			input.SprintKey, err)
	}

	result := &BulkAddResult{
		AddedByType:   make(map[string]int),
		SkippedByType: make(map[string]int),
	}

	// Step 4: Apply FeatureKey filter if set.
	// A task key belonging to feature "E07-F34" has the form "T-E07-F34-NNN" or "E07-F34-NNN".
	// We normalise the feature key to uppercase and check that the entity key contains it.
	var filtered []sprint.BacklogItem
	if input.FeatureKey != "" {
		featurePrefix := strings.ToUpper(strings.TrimSpace(input.FeatureKey))
		for _, c := range candidates {
			upperKey := strings.ToUpper(c.Key)
			// Match keys of the form "T-E07-F34-001" or "E07-F34-001".
			// The feature prefix "E07-F34" matches when the task key contains "-E07-F34-"
			// or starts with "E07-F34-" (short format).
			if strings.Contains(upperKey, "-"+featurePrefix+"-") ||
				strings.HasPrefix(upperKey, featurePrefix+"-") {
				filtered = append(filtered, c)
			} else {
				result.SkippedByType[c.EntityType]++
			}
		}
	} else {
		filtered = candidates
	}

	// Step 5: Convert filtered items to SprintAssignment for BulkAssign.
	if len(filtered) == 0 {
		return result, nil
	}

	// Step 5a (F07): Fetch the current max sprint_order so bulk rows can be auto-numbered.
	// §3.6: new rows get sprint_order = max + 1, max + 2, … (dense per-row assignment).
	maxOrder, maxOrderErr := s.repo.MaxSprintOrder(ctx, sprintEntity.ID)
	if maxOrderErr != nil {
		return nil, fmt.Errorf("failed to determine sprint order for bulk add to sprint %s: %w",
			input.SprintKey, maxOrderErr)
	}

	toAssign := make([]models.SprintAssignment, 0, len(filtered))
	for i, item := range filtered {
		order := maxOrder + i + 1 // 1-based offset within the bulk batch
		toAssign = append(toAssign, models.SprintAssignment{
			SprintID:    sprintEntity.ID,
			EntityType:  item.EntityType,
			EntityID:    item.EntityID,
			SprintOrder: &order,
		})
	}

	// Step 6: Perform the bulk insert. BulkAssign uses INSERT OR IGNORE to skip
	// any entity that gained an active assignment between ListUnassignedBacklog and now.
	// sprint_order is passed per-row so the DB receives the intended ordering immediately.
	inserted, err := s.assignmentRepo.BulkAssign(ctx, sprintEntity.ID, toAssign)
	if err != nil {
		return nil, fmt.Errorf("failed to bulk assign to sprint %s: %w", input.SprintKey, err)
	}

	// Step 6a (F07 §3.6 ImplementationNote-1): If INSERT OR IGNORE skipped any rows,
	// gaps appear in the sprint_order sequence (e.g., items at 3, 5 after item 4 was skipped).
	// Call RenumberAssignmentsTx to repair the sequence densely.
	// We only need to renumber when skips occurred AND some rows were inserted.
	skippedFromBulk := len(filtered) - inserted
	if skippedFromBulk > 0 && inserted > 0 {
		// Fetch the current ordered state after the bulk insert to build a clean renumber plan.
		// We renumber only the newly inserted batch (sprint_order > maxOrder) to avoid disturbing
		// pre-existing items.
		orderedAfter, listErr := s.repo.ListOrderedAssignments(ctx, sprintEntity.ID)
		if listErr == nil {
			// Build renumber ops for inserted batch items (those with sprint_order > maxOrder).
			var batchItems []*models.SprintAssignment
			for _, a := range orderedAfter {
				if a.SprintOrder != nil && *a.SprintOrder > maxOrder {
					batchItems = append(batchItems, a)
				}
			}
			// Dense-renumber the batch starting at maxOrder+1.
			if len(batchItems) > 0 {
				ops := make([]sprint.RenumberOp, 0, len(batchItems))
				for i, a := range batchItems {
					newPos := maxOrder + i + 1
					ops = append(ops, sprint.RenumberOp{
						AssignmentID: a.ID,
						NewPosition:  &newPos,
					})
				}
				// Ignore renumber errors — advisory, non-blocking. The sprint is usable
				// even if the order has minor gaps (user can run sprint reorder to fix).
				_ = s.repo.RenumberAssignmentsTx(ctx, nil, sprintEntity.ID, ops)
			}
		}
	}

	// Step 7: Compute per-type counts from the inserted vs. filtered totals.
	// BulkAssign returns a total inserted count; we attribute per-type from the candidate list.
	filteredByType := make(map[string]int)
	for _, item := range filtered {
		filteredByType[item.EntityType]++
	}

	skippedTotal := len(filtered) - inserted
	for et, count := range filteredByType {
		if skippedTotal <= 0 {
			result.AddedByType[et] += count
		} else {
			// Conservative best-effort attribution when concurrent modifications occurred.
			skip := bulkMin(skippedTotal, count)
			result.AddedByType[et] += count - skip
			result.SkippedByType[et] += skip
			skippedTotal -= skip
		}
	}

	// Step 8: Optional capacity warnings (advisory only, never blocks).
	if s.capacityRepo != nil {
		// Aggregate new size by agent type across all assigned items.
		newSizeByAgent := make(map[string]int)
		for _, item := range filtered {
			if item.AgentType != nil && *item.AgentType != "" {
				sz := 0
				if item.Size != nil {
					sz = *item.Size
				}
				newSizeByAgent[*item.AgentType] += sz
			}
		}
		for agentType, newSize := range newSizeByAgent {
			if w := s.computeCapacityWarning(ctx, sprintEntity.ID, agentType, newSize); w != nil {
				result.CapacityWarnings = append(result.CapacityWarnings, w)
			}
		}
	}

	return result, nil
}

// GetCarryoverBehavior returns the configured default carryover behavior from
// sprint_defaults.carryover_behavior in .sharkconfig.json.
//
// Returns the configured value, or "" if not configured (callers should default
// to "next" per spec §4.5 when the result is "").
func (s *SprintService) GetCarryoverBehavior() string {
	if s.cfg == nil || s.cfg.SprintDefaults == nil {
		return ""
	}
	return s.cfg.SprintDefaults.CarryoverBehavior
}

// bulkMin returns the smaller of two ints. Named to avoid collision with
// Go 1.21+ built-in min in environments where toolchain version varies.
func bulkMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ---------------------------------------------------------------------------
// CloseSprintWithCarryover — T-E19-F03-007
// ---------------------------------------------------------------------------

// CarryoverMode controls what happens to incomplete assignments when a sprint is closed.
type CarryoverMode string

const (
	// CarryoverNext moves incomplete entities to the next planning sprint.
	// If no planning sprint exists, a new one is auto-created.
	CarryoverNext CarryoverMode = "next"

	// CarryoverBacklog soft-deletes incomplete assignments (sets removed_at),
	// returning entities to the unassigned backlog.
	CarryoverBacklog CarryoverMode = "backlog"
)

// SprintCloseResult summarizes what happened during CloseSprintWithCarryover.
type SprintCloseResult struct {
	// Sprint is the closed sprint (reloaded after status update).
	Sprint *models.Sprint
	// CompletedCount is the number of entities in terminal status at close time.
	CompletedCount int
	// CarriedOverCount is the number of incomplete entities moved to the next sprint (carryover=next).
	CarriedOverCount int
	// DroppedCount is the number of incomplete entities soft-deleted (carryover=backlog).
	DroppedCount int
	// NextSprintKey is the key of the sprint that received carryover entities. Empty when carryover=backlog.
	NextSprintKey string
	// CarryoverPreserved indicates whether the relative sprint_order of carried-over items was
	// preserved in the receiving sprint. True for CarryoverNext (items appended at M+1..M+K),
	// false for CarryoverBacklog (sprint_order cleared).
	CarryoverPreserved bool
}

// CloseSprintWithCarryover atomically closes a sprint and handles incomplete entity assignments.
//
// Steps:
//  1. Validates sprint is in "active" status (TC-C12).
//  2. Resolves carryover mode: uses config default when carryoverMode == "" (TC-C09, TC-C10).
//  3. Fetches ALL active assignments (for total count) and INCOMPLETE assignments (for carryover).
//  4. Begins a database transaction.
//  5. Based on carryoverMode:
//     a. "next": finds or auto-creates the next planning sprint; calls ReassignToSprintTx.
//     b. "backlog": calls DropAssignmentsTx on incomplete entities.
//  6. Advances sprint status to "completed" via UpdateStatusTx (inside transaction).
//  7. Inserts a sprint_completions row via CreateCompletionTx (inside transaction).
//  8. Commits. Any failure causes a full rollback via defer tx.Rollback() (TC-C11).
func (s *SprintService) CloseSprintWithCarryover(ctx context.Context, sprintKey string, carryoverMode CarryoverMode) (*SprintCloseResult, error) {
	// Step 1: Resolve sprint and validate it is active — TC-C12
	sprintEntity, err := s.GetSprint(ctx, sprintKey)
	if err != nil {
		return nil, fmt.Errorf("failed to close sprint %s: %w", sprintKey, err)
	}

	if string(sprintEntity.Status) != "active" {
		return nil, fmt.Errorf("cannot close sprint %s: current status is %q, must be %q",
			sprintKey, sprintEntity.Status, "active")
	}

	// Step 2: Resolve carryover mode from config when empty (TC-C09, TC-C10)
	resolvedMode := carryoverMode
	if resolvedMode == "" {
		resolvedMode = s.resolveCarryoverMode()
	}

	// Step 3: Fetch assignments
	// All active assignments — for total count (TC-C08: PlannedEntityCount)
	allAssignments, err := s.repo.ListAssignments(ctx, sprintEntity.ID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list assignments for sprint %s: %w", sprintKey, err)
	}
	totalCount := len(allAssignments)

	// Incomplete assignments — uses workflow-aware terminal status detection (TC-C07).
	// Collect terminal statuses from each entity level that can be assigned to sprints.
	// Using s.workflowSvc.GetTerminalStatuses() would return sprint-level terminal statuses
	// (e.g. "completed", "archived"), not entity-level ones (e.g. "resolved" for tech_debt).
	terminalSet := make(map[string]struct{})
	for _, level := range []string{workflow.LevelTask, workflow.LevelBug, workflow.LevelChange, workflow.LevelTechDebt} {
		for _, st := range s.workflowSvc.ForLevel(level).GetTerminalStatuses() {
			terminalSet[st] = struct{}{}
		}
	}
	terminalStatuses := make([]string, 0, len(terminalSet))
	for st := range terminalSet {
		terminalStatuses = append(terminalStatuses, st)
	}
	incompleteAssignments, err := s.repo.ListAssignmentsForCarryover(ctx, sprintEntity.ID, terminalStatuses...)
	if err != nil {
		return nil, fmt.Errorf("failed to list incomplete assignments for sprint %s: %w", sprintKey, err)
	}

	completedCount := totalCount - len(incompleteAssignments)
	if completedCount < 0 {
		completedCount = 0
	}

	// Collect IDs of incomplete assignments for Tx methods
	incompleteIDs := make([]int64, 0, len(incompleteAssignments))
	for _, a := range incompleteAssignments {
		incompleteIDs = append(incompleteIDs, a.ID)
	}

	// Step 4: Begin transaction (TC-C11: defer tx.Rollback() ensures rollback on any error)
	if s.db == nil {
		return nil, fmt.Errorf("cannot close sprint %s: database connection not available for transaction", sprintKey)
	}
	tx, err := s.db.BeginTxContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction for sprint close %s: %w", sprintKey, err)
	}
	defer tx.Rollback() //nolint:errcheck // intentional: no-op after Commit; rolls back on any error path

	// Step 5a/b: Handle carryover
	var nextSprintKey string
	var nextSprintID *int64
	carriedOverCount := 0
	droppedCount := 0
	carryoverPreserved := false

	switch resolvedMode {
	case CarryoverNext:
		// Find an existing planning sprint.
		planningFilter := &sprint.SprintListFilters{Status: closeSprintStatusPtr("planning")}
		planningSprints, listErr := s.repo.List(ctx, planningFilter)
		if listErr != nil {
			return nil, fmt.Errorf("failed to find next planning sprint: %w", listErr)
		}

		var nextSprint *models.Sprint
		if len(planningSprints) > 0 {
			// TC-C01: existing planning sprint found
			nextSprint = planningSprints[0]
		} else {
			// TC-C02, TC-C04: auto-create sprint with start = closed.EndDate + 1 day, same duration
			duration := sprintEntity.EndDate.Sub(sprintEntity.StartDate)
			newStart := sprintEntity.EndDate.AddDate(0, 0, 1)
			newEnd := newStart.Add(duration)

			nextKey, keyErr := s.repo.GetNextKey(ctx)
			if keyErr != nil {
				return nil, fmt.Errorf("failed to generate key for auto-created sprint: %w", keyErr)
			}

			autoSprint := &models.Sprint{
				Key:       nextKey,
				Name:      "Sprint " + nextKey,
				StartDate: newStart,
				EndDate:   newEnd,
				Status:    models.SprintStatus("planning"),
				Slug:      utils.GenerateSlug("Sprint " + nextKey),
			}
			if createErr := s.repo.Create(ctx, autoSprint); createErr != nil {
				return nil, fmt.Errorf("failed to auto-create next sprint: %w", createErr)
			}
			nextSprint = autoSprint
		}

		nextSprintKey = nextSprint.Key
		id := nextSprint.ID
		nextSprintID = &id

		// Reassign incomplete assignments to next sprint (TC-C01, TC-C07, TC-C03: no-op when empty)
		if reassignErr := s.repo.ReassignToSprintTx(ctx, tx, incompleteIDs, nextSprint.ID); reassignErr != nil {
			return nil, fmt.Errorf("failed to reassign incomplete assignments to sprint %s: %w", nextSprintKey, reassignErr)
		}
		carriedOverCount = len(incompleteIDs)

		// Append carried-over items after existing ordered items in the receiving sprint (TC-021).
		// Sort carried assignments by (sprint_order ASC NULLS LAST, assigned_at ASC, id ASC) so that
		// items that had an explicit pull priority in the closed sprint are appended in that order.
		if len(incompleteAssignments) > 0 {
			maxOrder, maxErr := s.repo.MaxSprintOrder(ctx, nextSprint.ID)
			if maxErr != nil {
				return nil, fmt.Errorf("failed to get max sprint_order for receiving sprint %s: %w", nextSprintKey, maxErr)
			}

			// Sort carried assignments for deterministic append order.
			sortedCarried := make([]*models.SprintAssignment, len(incompleteAssignments))
			copy(sortedCarried, incompleteAssignments)
			sort.SliceStable(sortedCarried, func(i, j int) bool {
				a, b := sortedCarried[i], sortedCarried[j]
				// Tier 1: sprint_order ASC NULLS LAST
				switch {
				case a.SprintOrder != nil && b.SprintOrder == nil:
					return true
				case a.SprintOrder == nil && b.SprintOrder != nil:
					return false
				case a.SprintOrder != nil && b.SprintOrder != nil && *a.SprintOrder != *b.SprintOrder:
					return *a.SprintOrder < *b.SprintOrder
				}
				// Tier 2: assigned_at ASC
				if !a.AssignedAt.Equal(b.AssignedAt) {
					return a.AssignedAt.Before(b.AssignedAt)
				}
				// Tier 3: id ASC (stable tiebreaker)
				return a.ID < b.ID
			})

			// Build renumber ops: assign positions maxOrder+1, maxOrder+2, ...
			ops := make([]sprint.RenumberOp, 0, len(sortedCarried))
			for k, a := range sortedCarried {
				pos := maxOrder + k + 1
				ops = append(ops, sprint.RenumberOp{
					AssignmentID: a.ID,
					NewPosition:  &pos,
				})
			}
			if renumErr := s.repo.RenumberAssignmentsTx(ctx, tx, nextSprint.ID, ops); renumErr != nil {
				return nil, fmt.Errorf("failed to renumber carried-over assignments in sprint %s: %w", nextSprintKey, renumErr)
			}
		}
		carryoverPreserved = true

	case CarryoverBacklog:
		// Soft-delete incomplete assignments (TC-C05, TC-C06, no-op when empty).
		// sprint_order is cleared atomically in the same UPDATE as removed_at (TC-023).
		if dropErr := s.repo.DropAssignmentsTx(ctx, tx, incompleteIDs); dropErr != nil {
			return nil, fmt.Errorf("failed to drop incomplete assignments for sprint %s: %w", sprintKey, dropErr)
		}
		droppedCount = len(incompleteIDs)
		carryoverPreserved = false

	default:
		return nil, fmt.Errorf("unsupported carryover mode %q: must be %q or %q", resolvedMode, CarryoverNext, CarryoverBacklog)
	}

	// Step 6: Advance sprint status to completed inside the transaction (TC-C11)
	if statusErr := s.repo.UpdateStatusTx(ctx, tx, sprintEntity.ID, models.SprintStatus("completed")); statusErr != nil {
		return nil, fmt.Errorf("failed to update sprint %s status in transaction: %w", sprintKey, statusErr)
	}

	// Step 7: Insert sprint_completions row (TC-C08)
	completion := &models.SprintCompletion{
		SprintID:             sprintEntity.ID,
		CompletedAt:          time.Now().UTC(),
		PlannedEntityCount:   totalCount,
		CompletedEntityCount: completedCount,
		CarriedOverCount:     carriedOverCount,
		DroppedCount:         droppedCount,
		CarryoverMode:        string(resolvedMode),
		NextSprintID:         nextSprintID,
	}
	if completionErr := s.repo.CreateCompletionTx(ctx, tx, completion); completionErr != nil {
		return nil, fmt.Errorf("failed to create sprint_completions record for %s: %w", sprintKey, completionErr)
	}

	// Step 8: Commit
	if commitErr := tx.Commit(); commitErr != nil {
		return nil, fmt.Errorf("failed to commit sprint close transaction for %s: %w", sprintKey, commitErr)
	}

	// Reload sprint after commit (status is now "completed")
	closedSprint, err := s.GetSprint(ctx, sprintKey)
	if err != nil {
		return nil, fmt.Errorf("failed to reload sprint %s after close: %w", sprintKey, err)
	}

	return &SprintCloseResult{
		Sprint:             closedSprint,
		CompletedCount:     completedCount,
		CarriedOverCount:   carriedOverCount,
		DroppedCount:       droppedCount,
		NextSprintKey:      nextSprintKey,
		CarryoverPreserved: carryoverPreserved,
	}, nil
}

// resolveCarryoverMode returns the effective CarryoverMode from config,
// defaulting to CarryoverNext when the config key is absent (TC-C10).
func (s *SprintService) resolveCarryoverMode() CarryoverMode {
	if s.cfg != nil && s.cfg.SprintDefaults != nil && s.cfg.SprintDefaults.CarryoverBehavior != "" {
		return CarryoverMode(s.cfg.SprintDefaults.CarryoverBehavior)
	}
	return CarryoverNext
}

// closeSprintStatusPtr returns a pointer to a SprintStatus value (filter helper).
// Named distinctively to avoid collision with any future package-level helper.
func closeSprintStatusPtr(status string) *models.SprintStatus {
	v := models.SprintStatus(status)
	return &v
}

// ---------------------------------------------------------------------------
// SetSprintCapacity and GetSprintCapacity — T-E19-F05-006
// ---------------------------------------------------------------------------

// CapacityRow is one row in the sprint capacity display, computed at query time.
// AllocatedPoints is Σ size of assigned entities for this agent type.
// Remaining = CapacityPoints - AllocatedPoints (can be negative, indicating overcommit).
// UnsizedAssigned is the count of assigned entities with size IS NULL for this agent type.
type CapacityRow struct {
	AgentType       string  `json:"agent_type"`
	CapacityPoints  float64 `json:"capacity_points"`
	AllocatedPoints float64 `json:"allocated_points"`
	Remaining       float64 `json:"remaining"`
	UnsizedAssigned int     `json:"unsized_assigned"`
}

// SetSprintCapacityInput contains parameters for setting sprint capacity.
type SetSprintCapacityInput struct {
	// SprintKey identifies the sprint (e.g., "S024"). Required.
	SprintKey string
	// AgentType is the agent type bucket (e.g., "backend"). Required.
	AgentType string
	// Points is the capacity in story points. Must be > 0.
	Points float64
}

// SetSprintCapacity creates or updates a capacity row for a (sprint, agent_type) pair.
//
// Steps:
//  1. Validates that Points > 0 (returns error before any repo call).
//  2. Resolves sprint key to ID via SprintRepository.GetByKey.
//  3. Calls SprintCapacityRepository.SetCapacity (upsert via ON CONFLICT DO UPDATE).
//  4. Returns the upserted SprintCapacity model.
//
// Returns error if capacityRepo is nil, points <= 0, sprint not found, or repo fails.
func (s *SprintService) SetSprintCapacity(ctx context.Context, input SetSprintCapacityInput) (*models.SprintCapacity, error) {
	// Require capacityRepo: SetSprintCapacity cannot work without it.
	if s.capacityRepo == nil {
		return nil, fmt.Errorf("SetSprintCapacity requires capacityRepo; service was constructed without one")
	}

	// Validate points > 0 before any repo call (TC-014-03: SetCapacity must NOT be called).
	if input.Points <= 0 {
		return nil, fmt.Errorf("capacity points must be > 0; got %v", input.Points)
	}

	// Resolve sprint key → ID.
	sprintEntity, err := s.repo.GetByKey(ctx, input.SprintKey)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve sprint %q for capacity set: %w", input.SprintKey, err)
	}

	// Upsert capacity row.
	cap := &models.SprintCapacity{
		SprintID:       sprintEntity.ID,
		AgentType:      input.AgentType,
		CapacityPoints: input.Points,
	}
	if err := s.capacityRepo.SetCapacity(ctx, cap); err != nil {
		return nil, fmt.Errorf("failed to set capacity for sprint %s agent %s: %w",
			input.SprintKey, input.AgentType, err)
	}

	return cap, nil
}

// GetSprintCapacity returns capacity vs. allocation for all agent types in a sprint.
//
// Steps:
//  1. Resolves sprint key to ID.
//  2. Fetches capacity rows via SprintCapacityRepository.GetCapacity (one query).
//  3. Fetches active assignments with size via SprintAssignmentQueryRepository.GetAssignmentsWithSize
//     (one query, issued even when capacity rows are absent to satisfy the two-query contract).
//  4. Computes AllocatedPoints and UnsizedAssigned per agent type in-memory.
//  5. Returns a CapacityRow per configured agent type. Remaining may be negative.
//
// Returns empty slice (not error) when no capacity rows exist for the sprint (AC-6).
// Exactly two repository calls are issued regardless of data presence (spec §2.4).
// Returns error if sprint not found or repository calls fail.
func (s *SprintService) GetSprintCapacity(ctx context.Context, key string) ([]CapacityRow, error) {
	// Resolve sprint key → ID.
	sprintEntity, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve sprint %q for capacity show: %w", key, err)
	}

	// Fetch capacity rows — always issue this query (one of the two guaranteed queries).
	var capacities []*models.SprintCapacity
	if s.capacityRepo != nil {
		capacities, err = s.capacityRepo.GetCapacity(ctx, sprintEntity.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get capacity for sprint %s: %w", key, err)
		}
	}

	// Fetch assignments with size — always issue this query (second of the two guaranteed queries).
	// Issued even when capacity rows are absent so callers can rely on a fixed two-query pattern.
	var assignments []sprint.AssignmentWithSize
	if s.assignmentRepo != nil {
		assignments, err = s.assignmentRepo.GetAssignmentsWithSize(ctx, sprintEntity.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get assignments for sprint %s capacity view: %w", key, err)
		}
	}

	// No capacity rows configured — return empty slice (not error, per AC-6).
	// NOTE: both queries above are always issued; the early return here is correct because
	// there is no capacity configuration to build rows from, but the assignment query
	// was still executed (two-query contract is satisfied before this check).
	if len(capacities) == 0 {
		return []CapacityRow{}, nil
	}

	return buildCapacityRows(assignments, capacities), nil
}

// ---------------------------------------------------------------------------
// GetSprintReadiness — T-E19-F05-005
// ---------------------------------------------------------------------------

// ReadinessFactor is a single factor in the sprint readiness score.
type ReadinessFactor struct {
	// Name is the human-readable label (e.g., "Capacity utilization").
	Name string `json:"name"`
	// Score is the factor's individual score within [0, MaxScore].
	Score int `json:"score"`
	// MaxScore is the maximum possible score for this factor.
	MaxScore int `json:"max_score"`
	// Detail is a one-line explanation of this factor's result.
	Detail string `json:"detail"`
}

// SprintReadiness is the output of GetSprintReadiness.
// It contains the overall score (0-100), per-factor breakdown, and entity lists.
type SprintReadiness struct {
	// OverallScore is the sum of all factor scores, capped at 100.
	OverallScore int `json:"overall_score"`
	// Factors contains exactly 6 ReadinessFactor entries.
	Factors []ReadinessFactor `json:"factors"`
	// UnsizedEntities lists assigned entities with size IS NULL.
	UnsizedEntities []sprint.BacklogItem `json:"unsized_entities"`
	// OversizedEntities lists assigned entities with size >= 8.
	OversizedEntities []sprint.BacklogItem `json:"oversized_entities"`
}

// GetSprintReadiness computes the 0-100 readiness score for a sprint.
//
// Fetches data via exactly two repository calls:
//  1. assignmentRepo.GetAssignmentsWithSize — all active assignments with size, title, depends_on
//  2. capacityRepo.GetCapacity — all capacity rows for the sprint
//
// All six factor scores are then computed in-memory — no additional DB queries per factor.
// The result is deterministic: identical inputs always produce the same output.
//
// Six factors (total max: 100):
//  1. Capacity utilization     (0-25)
//  2. Dependency satisfaction  (0-20)
//  3. Task count               (0-15)
//  4. Agent balance            (0-15)
//  5. Sizing coverage          (0-15)
//  6. Oversized-entity flag    (0-10)
func (s *SprintService) GetSprintReadiness(ctx context.Context, key string) (*SprintReadiness, error) {
	// Require assignmentRepo: readiness scoring requires assignment data.
	if s.assignmentRepo == nil {
		return nil, fmt.Errorf("GetSprintReadiness requires assignmentRepo; service was constructed without one")
	}

	// Resolve sprint key → ID (one repo call via SprintRepository).
	sprintEntity, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve sprint %q for readiness: %w", key, err)
	}

	// ─── Query 1 of 2: assignments with size, title, depends_on ────────────
	assignments, err := s.assignmentRepo.GetAssignmentsWithSize(ctx, sprintEntity.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignments for sprint %s readiness: %w", key, err)
	}

	// ─── Query 2 of 2: capacity rows ───────────────────────────────────────
	var capacities []*models.SprintCapacity
	if s.capacityRepo != nil {
		capacities, err = s.capacityRepo.GetCapacity(ctx, sprintEntity.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get capacity for sprint %s readiness: %w", key, err)
		}
	}

	// ─── In-memory computation only from here ──────────────────────────────
	return computeReadinessFromData(assignments, capacities), nil
}

// computeCapacityUtilizationFactor computes Factor 1 (Capacity utilization, 0-25).
//
// Algorithm from spec §2.4:
//
//	if totalCapacity == 0: score = 0
//	utilization = totalAllocated / totalCapacity
//	if 0.5 <= utilization <= 1.0: score = 25
//	elif utilization > 1.0: score = max(0, 25 - int((utilization-1.0)*50))
//	else: score = int(utilization / 0.5 * 25)
func computeCapacityUtilizationFactor(totalCapacity, totalAllocated float64) ReadinessFactor {
	const maxScore = 25
	name := "Capacity utilization"

	if totalCapacity == 0 {
		return ReadinessFactor{
			Name:     name,
			Score:    0,
			MaxScore: maxScore,
			Detail:   "No capacity configured for this sprint",
		}
	}

	utilization := totalAllocated / totalCapacity
	var score int
	var detail string

	switch {
	case utilization >= 0.5 && utilization <= 1.0:
		score = 25
		detail = fmt.Sprintf("Utilization %.0f%% is in the optimal range (50-100%%)", utilization*100)
	case utilization > 1.0:
		penalty := int((utilization - 1.0) * 50)
		score = maxScore - penalty
		if score < 0 {
			score = 0
		}
		detail = fmt.Sprintf("Overcommitted at %.0f%% — penalty applied", utilization*100)
	default:
		// utilization < 0.5
		score = int(utilization / 0.5 * 25)
		detail = fmt.Sprintf("Utilization %.0f%% is below 50%% — sprint is under-loaded", utilization*100)
	}

	return ReadinessFactor{Name: name, Score: score, MaxScore: maxScore, Detail: detail}
}

// computeDependencySatisfactionFactor computes Factor 2 (Dependency satisfaction, 0-20).
//
// For each task in the sprint, parses depends_on JSON ([]string of task keys).
// A dependency is "satisfied" if the dependency key is also assigned to the sprint.
// Unsatisfied = dependencies not in the sprint's assigned key set.
// score = max(0, 20 - unsatisfied_count)
//
// Non-task entities are excluded (bugs/changes/tech-debts have no depends_on).
func computeDependencySatisfactionFactor(assignments []sprint.AssignmentWithSize, assignedKeys map[string]bool) ReadinessFactor {
	const maxScore = 20
	name := "Dependency satisfaction"

	unsatisfied := 0
	malformed := 0
	for _, a := range assignments {
		if a.EntityType != "task" || a.DependsOn == "" || a.DependsOn == "[]" || a.DependsOn == "null" {
			continue
		}
		// Parse depends_on as []string
		var deps []string
		if err := json.Unmarshal([]byte(a.DependsOn), &deps); err != nil {
			// Malformed JSON — treat as no dependencies (graceful degradation) but track count
			malformed++
			continue
		}
		for _, dep := range deps {
			if !assignedKeys[strings.ToUpper(dep)] {
				unsatisfied++
			}
		}
	}

	score := maxScore - unsatisfied
	if score < 0 {
		score = 0
	}

	var detail string
	if unsatisfied == 0 {
		detail = "All task dependencies are satisfied (assigned or already completed)"
	} else {
		detail = fmt.Sprintf("%d unsatisfied external task dependenc%s", unsatisfied,
			pluralize(unsatisfied, "y", "ies"))
	}
	if malformed > 0 {
		detail += fmt.Sprintf(" (%d %s had malformed depends_on)",
			malformed, pluralize(malformed, "entity", "entities"))
	}

	return ReadinessFactor{Name: name, Score: score, MaxScore: maxScore, Detail: detail}
}

// computeTaskCountFactor computes Factor 3 (Task count, 0-15).
//
// Algorithm from spec §2.4:
//
//	if totalEntities == 0: score = 0
//	elif totalEntities >= 3: score = 15
//	else: score = int(totalEntities / 3.0 * 15)
func computeTaskCountFactor(totalEntities int) ReadinessFactor {
	const maxScore = 15
	name := "Task count"

	var score int
	var detail string
	switch {
	case totalEntities == 0:
		score = 0
		detail = "Sprint has no assigned entities"
	case totalEntities >= 3:
		score = 15
		detail = fmt.Sprintf("%d entities assigned — at or above the minimum of 3", totalEntities)
	default:
		score = int(float64(totalEntities) / 3.0 * 15)
		detail = fmt.Sprintf("%d %s assigned — below the recommended minimum of 3",
			totalEntities, pluralize(totalEntities, "entity", "entities"))
	}

	return ReadinessFactor{Name: name, Score: score, MaxScore: maxScore, Detail: detail}
}

// computeAgentBalanceFactor computes Factor 4 (Agent balance, 0-15).
//
// 15 pts if >=2 distinct non-nil agent_type values are present; 0 pts otherwise.
func computeAgentBalanceFactor(assignments []sprint.AssignmentWithSize) ReadinessFactor {
	const maxScore = 15
	name := "Agent balance"

	distinct := make(map[string]bool)
	for _, a := range assignments {
		if a.AgentType != nil && *a.AgentType != "" {
			distinct[*a.AgentType] = true
		}
	}

	var score int
	var detail string
	if len(distinct) >= 2 {
		score = 15
		detail = fmt.Sprintf("%d distinct agent types present — sprint is well balanced", len(distinct))
	} else if len(distinct) == 1 {
		score = 0
		for at := range distinct {
			detail = fmt.Sprintf("Only one agent type (%s) — consider adding cross-functional work", at)
		}
	} else {
		score = 0
		detail = "No agent types assigned — sprint has no attributed work"
	}

	return ReadinessFactor{Name: name, Score: score, MaxScore: maxScore, Detail: detail}
}

// computeSizingCoverageFactor computes Factor 5 (Sizing coverage, 0-15).
//
// 15 pts if all entities have non-nil size; 1 pt deducted per unsized entity, floor 0.
func computeSizingCoverageFactor(assignments []sprint.AssignmentWithSize) ReadinessFactor {
	const maxScore = 15
	name := "Sizing coverage"

	unsizedCount := 0
	for _, a := range assignments {
		if a.Size == nil {
			unsizedCount++
		}
	}

	score := maxScore - unsizedCount
	if score < 0 {
		score = 0
	}

	var detail string
	if unsizedCount == 0 {
		detail = "All assigned entities have a size estimate"
	} else {
		detail = fmt.Sprintf("%d unsized %s — add size estimates to improve planning accuracy",
			unsizedCount, pluralize(unsizedCount, "entity", "entities"))
	}

	return ReadinessFactor{Name: name, Score: score, MaxScore: maxScore, Detail: detail}
}

// computeOversizedEntityFactor computes Factor 6 (Oversized-entity flag, 0-10).
//
// 10 pts if no assigned entity has size >= 8; 0 pts if any such entity exists.
// Per spec §1.1.3 AC-7 and .claude/rules/development-workflows.md: L/XL/XXL = size 8+.
func computeOversizedEntityFactor(assignments []sprint.AssignmentWithSize) ReadinessFactor {
	const maxScore = 10
	name := "Oversized-entity flag"

	oversizedCount := 0
	for _, a := range assignments {
		if a.Size != nil && *a.Size >= 8 {
			oversizedCount++
		}
	}

	var score int
	var detail string
	if oversizedCount == 0 {
		score = 10
		detail = "No oversized entities (size >= 8) — all work items are appropriately sized"
	} else {
		score = 0
		detail = fmt.Sprintf("%d oversized %s (size >= 8) — consider breaking them down before sprint start",
			oversizedCount, pluralize(oversizedCount, "entity", "entities"))
	}

	return ReadinessFactor{Name: name, Score: score, MaxScore: maxScore, Detail: detail}
}

// ---------------------------------------------------------------------------
// PlanSprint — T-E19-F05-004
// ---------------------------------------------------------------------------

// SprintPlanView is the composite output of PlanSprint.
// It aggregates the unassigned backlog, capacity utilization, and readiness score
// so CLI formatters can render the three planning sections without additional calls.
type SprintPlanView struct {
	Sprint    *models.Sprint       `json:"sprint"`
	Backlog   []sprint.BacklogItem `json:"backlog"`   // unassigned entities eligible for assignment
	Capacity  []CapacityRow        `json:"capacity"`  // per agent-type capacity vs. allocation
	Readiness *SprintReadiness     `json:"readiness"` // 0-100 readiness score with factor breakdown
}

// PlanSprint returns the composite planning view for a sprint.
//
// The view contains three sections rendered by the CLI:
//  1. Backlog: unassigned entities eligible for assignment (all entity types)
//  2. Capacity: per-agent-type capacity vs. allocated story points
//  3. Readiness: 0-100 readiness score with 6-factor breakdown
//
// Implementation strategy:
//   - Step 1: Resolve sprint key → entity (one GetByKey call)
//   - Step 2: List unassigned backlog from assignmentRepo (all types)
//   - Step 3: GetAssignmentsWithSize for the resolved sprint ID (capacity + readiness)
//   - Step 4: GetCapacity for the resolved sprint ID (capacity + readiness)
//   - Steps 5-6: Compute CapacityRow slice and SprintReadiness in-memory
//
// When assignmentRepo or capacityRepo is nil, the corresponding sections
// degrade gracefully to empty slices and a score-0 readiness.
func (s *SprintService) PlanSprint(ctx context.Context, key string) (*SprintPlanView, error) {
	// Step 1: Resolve sprint key → entity.
	sprintEntity, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve sprint %q for plan: %w", key, err)
	}

	// Step 2: Fetch unassigned backlog (all entity types).
	var backlog []sprint.BacklogItem
	if s.assignmentRepo != nil {
		allTypes := []string{"task", "bug", "change_card", "tech_debt"}
		backlog, err = s.assignmentRepo.ListUnassignedBacklog(ctx, allTypes)
		if err != nil {
			return nil, fmt.Errorf("failed to list unassigned backlog for sprint %s plan: %w", key, err)
		}
	}
	if backlog == nil {
		backlog = []sprint.BacklogItem{}
	}

	// Step 3: Fetch assignments with size (shared by capacity computation and readiness).
	var assignments []sprint.AssignmentWithSize
	if s.assignmentRepo != nil {
		assignments, err = s.assignmentRepo.GetAssignmentsWithSize(ctx, sprintEntity.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get assignments for sprint %s plan: %w", key, err)
		}
	}

	// Step 4: Fetch capacity rows (shared by capacity computation and readiness).
	var capacityModels []*models.SprintCapacity
	if s.capacityRepo != nil {
		capacityModels, err = s.capacityRepo.GetCapacity(ctx, sprintEntity.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get capacity for sprint %s plan: %w", key, err)
		}
	}

	// Step 5: Compute CapacityRow slice in-memory.
	capacity := buildCapacityRows(assignments, capacityModels)

	// Step 6: Compute SprintReadiness in-memory (uses the same factor algorithms as GetSprintReadiness).
	readiness := planComputeReadiness(assignments, capacityModels)

	return &SprintPlanView{
		Sprint:    sprintEntity,
		Backlog:   backlog,
		Capacity:  capacity,
		Readiness: readiness,
	}, nil
}

// buildCapacityRows builds a CapacityRow slice from in-memory assignment and capacity data.
// Shared by GetSprintCapacity (after its two guaranteed queries) and PlanSprint (which
// fetches the same data as part of its broader composite query set).
func buildCapacityRows(assignments []sprint.AssignmentWithSize, capacityModels []*models.SprintCapacity) []CapacityRow {
	if len(capacityModels) == 0 {
		return []CapacityRow{}
	}
	allocatedByAgent := make(map[string]float64)
	unsizedByAgent := make(map[string]int)
	for _, a := range assignments {
		if a.AgentType == nil || *a.AgentType == "" {
			continue
		}
		agent := *a.AgentType
		if a.Size == nil {
			unsizedByAgent[agent]++
		} else {
			allocatedByAgent[agent] += float64(*a.Size)
		}
	}
	rows := make([]CapacityRow, 0, len(capacityModels))
	for _, c := range capacityModels {
		alloc := allocatedByAgent[c.AgentType]
		rows = append(rows, CapacityRow{
			AgentType:       c.AgentType,
			CapacityPoints:  c.CapacityPoints,
			AllocatedPoints: alloc,
			Remaining:       c.CapacityPoints - alloc,
			UnsizedAssigned: unsizedByAgent[c.AgentType],
		})
	}
	return rows
}

// planComputeReadiness computes SprintReadiness from in-memory assignment and capacity data.
// Delegates to computeReadinessFromData so both paths produce identical output for
// identical inputs (determinism guarantee).
func planComputeReadiness(assignments []sprint.AssignmentWithSize, capacities []*models.SprintCapacity) *SprintReadiness {
	return computeReadinessFromData(assignments, capacities)
}

// computeReadinessFromData is the shared in-memory computation kernel used by both
// GetSprintReadiness (post-query) and planComputeReadiness (plan path).
//
// It aggregates capacity/allocation totals, builds the dependency-check index,
// collects unsized/oversized lists, and then calls each of the six factor helpers.
// Identical inputs always produce identical output (deterministic).
func computeReadinessFromData(assignments []sprint.AssignmentWithSize, capacities []*models.SprintCapacity) *SprintReadiness {
	// ─── Zero-entity degenerate case (spec AC-12) ──────────────────────────
	// When no entities are assigned, the overall score is 0 and all factor
	// scores are also 0, regardless of what individual formulae would produce.
	if len(assignments) == 0 {
		emptyFactors := []ReadinessFactor{
			{Name: "Capacity utilization", Score: 0, MaxScore: 25, Detail: "Sprint has no assigned entities"},
			{Name: "Dependency satisfaction", Score: 0, MaxScore: 20, Detail: "Sprint has no assigned entities"},
			{Name: "Task count", Score: 0, MaxScore: 15, Detail: "Sprint has no assigned entities"},
			{Name: "Agent balance", Score: 0, MaxScore: 15, Detail: "Sprint has no assigned entities"},
			{Name: "Sizing coverage", Score: 0, MaxScore: 15, Detail: "Sprint has no assigned entities"},
			{Name: "Oversized-entity flag", Score: 0, MaxScore: 10, Detail: "Sprint has no assigned entities"},
		}
		return &SprintReadiness{
			OverallScore:      0,
			Factors:           emptyFactors,
			UnsizedEntities:   []sprint.BacklogItem{},
			OversizedEntities: []sprint.BacklogItem{},
		}
	}

	totalEntities := len(assignments)

	// Build an index of assigned task keys for Factor 2 dependency check.
	assignedKeys := make(map[string]bool, totalEntities)
	for _, a := range assignments {
		assignedKeys[strings.ToUpper(a.Key)] = true
	}

	// Aggregate capacity totals for Factor 1.
	var totalCapacity, totalAllocated float64
	for _, c := range capacities {
		totalCapacity += c.CapacityPoints
	}
	for _, a := range assignments {
		if a.Size != nil {
			totalAllocated += float64(*a.Size)
		}
	}

	// Build UnsizedEntities and OversizedEntities lists (for JSON output).
	var unsized, oversized []sprint.BacklogItem
	for _, a := range assignments {
		if a.Size == nil {
			unsized = append(unsized, sprint.BacklogItem{Key: a.Key, Title: a.Title})
		} else if *a.Size >= 8 {
			oversized = append(oversized, sprint.BacklogItem{Key: a.Key, Title: a.Title})
		}
	}
	if unsized == nil {
		unsized = []sprint.BacklogItem{}
	}
	if oversized == nil {
		oversized = []sprint.BacklogItem{}
	}

	// ─── Factor 1: Capacity utilization (0-25) ─────────────────────────────
	f1 := computeCapacityUtilizationFactor(totalCapacity, totalAllocated)

	// ─── Factor 2: Dependency satisfaction (0-20) ──────────────────────────
	f2 := computeDependencySatisfactionFactor(assignments, assignedKeys)

	// ─── Factor 3: Task count (0-15) ───────────────────────────────────────
	f3 := computeTaskCountFactor(totalEntities)

	// ─── Factor 4: Agent balance (0-15) ────────────────────────────────────
	f4 := computeAgentBalanceFactor(assignments)

	// ─── Factor 5: Sizing coverage (0-15) ──────────────────────────────────
	f5 := computeSizingCoverageFactor(assignments)

	// ─── Factor 6: Oversized-entity flag (0-10) ────────────────────────────
	f6 := computeOversizedEntityFactor(assignments)

	factors := []ReadinessFactor{f1, f2, f3, f4, f5, f6}

	overall := 0
	for _, f := range factors {
		overall += f.Score
	}
	if overall > 100 {
		overall = 100
	}

	return &SprintReadiness{
		OverallScore:      overall,
		Factors:           factors,
		UnsizedEntities:   unsized,
		OversizedEntities: oversized,
	}
}

// terminalSet returns a set of terminal status strings for the given workflow level.
func terminalSet(svc *workflow.Service) map[string]bool {
	result := make(map[string]bool)
	for _, s := range svc.GetTerminalStatuses() {
		result[s] = true
	}
	return result
}
