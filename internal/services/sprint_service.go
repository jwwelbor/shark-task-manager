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
	"github.com/jwwelbor/shark-task-manager/internal/entitytype"
	"github.com/jwwelbor/shark-task-manager/internal/keys"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/repository/sprint"
	"github.com/jwwelbor/shark-task-manager/internal/research"
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

	// CountNullSprintOrder returns the number of active assignments for the sprint
	// that have sprint_order = NULL. Used by StartSprint to surface a soft warning
	// (REQ-F-009) without blocking the start transition.
	CountNullSprintOrder(ctx context.Context, sprintID int64) (int, error)
}

// SprintAssignmentQueryRepository handles assignment queries needed for sprint planning.
// Implemented by *sprint.SprintRepository — no separate type needed.
type SprintAssignmentQueryRepository interface {
	BulkAssign(ctx context.Context, sprintID int64, assignments []models.SprintAssignment) (int, error)
	ListUnassignedBacklog(ctx context.Context, entityTypes []string, assignedSprintStatuses ...string) ([]sprint.BacklogItem, error)
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
	entitySvc      *EntityService                  // optional: nil-safe; wired via EnableWorkflowDispatch, required for TransitionStatus/GetNextStatus (B059)
	entityRepo     EntityRepository                // optional: nil-safe; wired via EnableWorkflowDispatch
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

// EnableWorkflowDispatch wires the shared EntityService/EntityRepository so
// SprintService satisfies runner.EntityTransitioner (TransitionStatus,
// GetNextStatus), enabling `shark next`/`shark status set|advance` for
// sprints (B059). Optional: TransitionStatus and GetNextStatus return an
// error if called before this is set, matching the nil-safe degrade pattern
// used for assignmentRepo/capacityRepo above.
func (s *SprintService) EnableWorkflowDispatch(entitySvc *EntityService, entityRepo EntityRepository) {
	if entitySvc == nil || entityRepo == nil {
		return
	}
	s.entitySvc = entitySvc.ForLevel(workflow.LevelSprint)
	s.entityRepo = entityRepo
}

// TransitionStatus transitions a sprint to a specific status with workflow
// validation. Delegates to EntityService.TransitionStatus for shared
// transition logic (same pattern as BugService/TechDebtService/QuestionService).
// Requires EnableWorkflowDispatch to have been called; returns an error otherwise.
func (s *SprintService) TransitionStatus(ctx context.Context, key string, targetStatus string, opts TransitionOptions) (*TransitionResult, error) {
	if s.entitySvc == nil || s.entityRepo == nil {
		return nil, fmt.Errorf("sprint workflow dispatch not configured: EnableWorkflowDispatch was not called")
	}
	return s.entitySvc.TransitionStatus(
		ctx, s.entityRepo, models.EntityTypeSprint, key, targetStatus, opts,
		SimpleTransitionFeatures(), s.makeResolveActionFn(),
	)
}

// GetNextStatus returns the available transitions for the current status of a sprint.
// Requires EnableWorkflowDispatch to have been called; returns an error otherwise.
func (s *SprintService) GetNextStatus(ctx context.Context, key string) (*NextStatusInfo, error) {
	if s.entitySvc == nil || s.entityRepo == nil {
		return nil, fmt.Errorf("sprint workflow dispatch not configured: EnableWorkflowDispatch was not called")
	}
	return s.entitySvc.GetNextStatus(ctx, s.entityRepo, models.EntityTypeSprint, key, s.makeResolveActionFn())
}

// makeResolveActionFn returns a ResolveActionFn callback that generates
// Sprint-specific placeholders for route-based workflow actions.
func (s *SprintService) makeResolveActionFn() ResolveActionFn {
	return func(entity models.Entity, status string) *config.PopulatedAction {
		sprint, ok := entity.(*models.Sprint)
		if !ok {
			return nil
		}
		placeholders := config.EntityPlaceholders(sprint)
		placeholders["is_resume"] = "false"
		return s.entitySvc.ResolveActionForStatus(status, placeholders)
	}
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
	initialStatus := s.workflowSvc.NormalizeStatus(s.workflowSvc.GetInitialStatusString())

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

	var sprints []*models.Sprint
	var err error
	if filters != nil && filters.Status != "" {
		sprints, err = s.listSprintsByExactStatuses(
			ctx,
			workflowStatusVocabulary(s.workflowSvc, []string{filters.Status}),
		)
	} else {
		sprints, err = s.repo.List(ctx, nil)
	}
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

func (s *SprintService) listSprintsByExactStatuses(ctx context.Context, statuses []string) ([]*models.Sprint, error) {
	var result []*models.Sprint
	seenSprintIDs := make(map[int64]bool)
	for _, status := range statuses {
		statusValue := models.SprintStatus(status)
		statusSprints, err := s.repo.List(ctx, &sprint.SprintListFilters{Status: &statusValue})
		if err != nil {
			return nil, err
		}
		for _, candidate := range statusSprints {
			if !seenSprintIDs[candidate.ID] {
				seenSprintIDs[candidate.ID] = true
				result = append(result, candidate)
			}
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func (s *SprintService) listSprintsByWorkflowStatuses(ctx context.Context, statuses []string) ([]*models.Sprint, error) {
	return s.listSprintsByExactStatuses(ctx, workflowStatusVocabulary(s.workflowSvc, statuses))
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

	// Only allow deletion of sprints in the initial (planning) status.
	initialStatus := s.workflowSvc.GetInitialStatusString()
	if !workflowStatusMatchesAny(s.workflowSvc, string(sprint.Status), []string{initialStatus}) {
		return recordSpanError(span, fmt.Errorf("cannot delete sprint %s in status %s: only sprints in %s status can be deleted", key, sprint.Status, initialStatus))
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

	// Validate workflow transition. Use the designated execution-phase status
	// so custom workflows with renamed statuses work without code changes.
	activeStatus, err := s.executionPhaseStatus()
	if err != nil {
		return nil, recordSpanError(span, fmt.Errorf("cannot start sprint %s: %w", key, err))
	}
	// The default sprint workflow routes planning directly to its execution
	// status. startTransitionTarget also supports custom planning workflows
	// whose pass route includes one explicit intermediate gate.
	targetStatus := s.startTransitionTarget(string(sprint.Status), activeStatus)
	if err := s.workflowSvc.ValidateTransition(string(sprint.Status), targetStatus); err != nil {
		return nil, recordSpanError(span, fmt.Errorf("cannot start sprint %s in status %s: %w", key, sprint.Status, err))
	}
	if s.workflowSvc.ProjectRoot() != "" && workflowStatusMatchesAny(
		s.workflowSvc,
		string(sprint.Status),
		s.workflowSvc.GetStatusesByPhase("research"),
	) {
		if err := research.ValidateEntity(s.workflowSvc.ProjectRoot(), sprint); err != nil {
			return nil, recordSpanError(span, fmt.Errorf("cannot start sprint %s from research: %w", key, err))
		}
	}

	// Update status
	if err := s.repo.UpdateStatus(ctx, sprint.ID, models.SprintStatus(targetStatus)); err != nil {
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

// CountNullSprintOrder returns the number of active assignments in the sprint with
// sprint_order = NULL. Called by runSprintStart after a successful start to surface
// the soft warning required by REQ-F-009. Non-fatal: callers should log or warn on
// error rather than blocking the operation.
func (s *SprintService) CountNullSprintOrder(ctx context.Context, sprintKey string) (int, error) {
	sprintEntity, err := s.GetSprint(ctx, sprintKey)
	if err != nil {
		return 0, fmt.Errorf("CountNullSprintOrder: failed to resolve sprint %q: %w", sprintKey, err)
	}
	n, err := s.repo.CountNullSprintOrder(ctx, sprintEntity.ID)
	if err != nil {
		return 0, fmt.Errorf("CountNullSprintOrder: failed to count for sprint %q: %w", sprintKey, err)
	}
	return n, nil
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

	// Validate workflow transition. Use the designated review-phase status so
	// custom workflows with renamed statuses work without code changes.
	closingStatus, err := s.reviewPhaseStatus()
	if err != nil {
		return nil, recordSpanError(span, fmt.Errorf("cannot close sprint %s: %w", key, err))
	}
	if err := s.workflowSvc.ValidateTransition(string(sprint.Status), closingStatus); err != nil {
		return nil, recordSpanError(span, fmt.Errorf("cannot close sprint %s in status %s: %w", key, sprint.Status, err))
	}

	// Update status
	if err := s.repo.UpdateStatus(ctx, sprint.ID, models.SprintStatus(closingStatus)); err != nil {
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
		// Keep the legacy predicate as a defensive fallback for older callers
		// even though KeyService.Parse now handles the accepted change aliases.
		if keys.IsChangeCardKey(entityKey) {
			normalized := strings.ToUpper(strings.TrimSpace(entityKey))
			entityID, err = repo.GetChangeCardIDByKey(ctx, normalized)
			if err != nil {
				return "", 0, fmt.Errorf("change_card %q not found: %w", entityKey, err)
			}
			return "change_card", entityID, nil
		}
		return "", 0, fmt.Errorf(
			"unsupported entity type %q for sprint assignment: entity key %q must be a task, bug, change-card (canonical CC-###; C###/CC### aliases accepted), or tech-debt (TD-###) key",
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
	// Per spec §4.2.1 step 1, only sprints in the planning or execution phases
	// may accept new entity assignments. We delegate phase membership to
	// workflow.Service so custom sprint workflows (e.g. renamed "draft" instead
	// of "planning") are honored without code changes here.
	sprintEntity, err := s.repo.GetByKey(ctx, input.SprintKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve sprint %q: %w", input.SprintKey, err)
	}
	if !s.sprintAcceptsAssignments(string(sprintEntity.Status)) {
		return nil, nil, fmt.Errorf(
			"cannot assign entity to sprint %s: sprint is in %q status (only sprints in the planning or execution phases accept new assignments; valid statuses: %s)",
			input.SprintKey, sprintEntity.Status,
			strings.Join(s.assignableSprintStatuses(), ", "),
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

		orderedCount := 0
		for _, item := range orderedItems {
			if item.SprintOrder != nil {
				orderedCount++
			}
		}
		if pos > orderedCount+1 {
			return nil, nil, fmt.Errorf(
				"position %d is out of range: sprint has %d ordered items (valid range: 1..%d)",
				pos, orderedCount, orderedCount+1,
			)
		}

		targetOrder = pos
	}

	// Step 7: Determine whether this insert needs to shift existing siblings.
	// "Shift needed" = at least one ordered item currently sits at sprint_order >= targetOrder.
	// Append-at-end (targetOrder == count+1 or no ordered items yet) never collides.
	var shiftOps []sprint.RenumberOp
	if input.Position != nil {
		pos := *input.Position
		if len(orderedItems) > 0 && pos <= len(orderedItems) {
			shiftOps = make([]sprint.RenumberOp, 0, len(orderedItems)-pos+1)
			for _, item := range orderedItems {
				if item.SprintOrder != nil && *item.SprintOrder >= pos {
					newPos := *item.SprintOrder + 1
					shiftOps = append(shiftOps, sprint.RenumberOp{
						AssignmentID: item.ID,
						NewPosition:  &newPos,
					})
				}
			}
		}
	}

	// Step 8: Create the assignment.
	//
	// Fast path (no shift): INSERT with sprint_order = targetOrder. No siblings
	// occupy that slot, so the partial unique index won't fire.
	//
	// Shift path: INSERT with sprint_order = NULL first. The partial unique
	// index has WHERE sprint_order IS NOT NULL, so a NULL insert never collides
	// even though existing rows still occupy positions [pos, count]. The final
	// position is assigned in step 9 via a single renumber UPDATE that also
	// shifts the siblings — but only after the siblings are NULL-cleared, so
	// the per-row unique-index check during that UPDATE never sees a duplicate.
	insertOrder := &targetOrder
	if len(shiftOps) > 0 {
		insertOrder = nil
	}
	assignment := &models.SprintAssignment{
		SprintID:    sprintEntity.ID,
		EntityType:  entityType,
		EntityID:    entityID,
		AssignedAt:  time.Now().UTC(),
		SprintOrder: insertOrder,
	}
	if err := s.repo.AddAssignment(ctx, assignment); err != nil {
		return nil, nil, fmt.Errorf("failed to add assignment for %q to sprint %s: %w",
			input.EntityKey, input.SprintKey, err)
	}

	// Step 9: For the shift path, NULL-clear the siblings, then apply final
	// positions (target + shifted siblings) in one renumber statement.
	if len(shiftOps) > 0 {
		clearOps := make([]sprint.RenumberOp, 0, len(shiftOps))
		for _, op := range shiftOps {
			clearOps = append(clearOps, sprint.RenumberOp{AssignmentID: op.AssignmentID, NewPosition: nil})
		}

		finalOps := make([]sprint.RenumberOp, 0, len(shiftOps)+1)
		finalOps = append(finalOps, sprint.RenumberOp{AssignmentID: assignment.ID, NewPosition: &targetOrder})
		finalOps = append(finalOps, shiftOps...)

		// TD-9: AddAssignment doesn't accept a *sql.Tx today, so we can't make
		// this fully atomic without expanding the repo API. The worst-case
		// mid-failure leaves the new row with sprint_order=NULL (unordered),
		// which is recoverable via a subsequent reorder.
		if err := s.repo.RenumberAssignmentsTx(ctx, nil, sprintEntity.ID, clearOps); err != nil {
			return nil, nil, fmt.Errorf("failed to clear sprint_order for insert at position %d in sprint %s: %w",
				*input.Position, input.SprintKey, err)
		}
		if err := s.repo.RenumberAssignmentsTx(ctx, nil, sprintEntity.ID, finalOps); err != nil {
			return nil, nil, fmt.Errorf("failed to shift sprint order for insert at position %d in sprint %s: %w",
				*input.Position, input.SprintKey, err)
		}
		assignment.SprintOrder = &targetOrder
	}

	// Step 9: Advisory capacity warning (never blocks)
	var warning *CapacityWarning
	if s.capacityRepo != nil && input.AgentType != "" {
		warning = s.computeCapacityWarning(ctx, sprintEntity.ID, input.AgentType, input.EstimatedSize)
	}

	return assignment, warning, nil
}

// assignableSprintStatuses returns the set of sprint statuses (from the
// configured sprint workflow) that accept new entity assignments. Computed by
// asking workflow.Service which statuses live in the planning and execution
// phases. The result is deduplicated and ordered: planning phase first, then
// execution phase, with within-phase order preserved from workflow.Service.
// "planning" and "execution" are YAML phase labels, not status names.
func (s *SprintService) assignableSprintStatuses() []string {
	return s.sprintStatusesForPhases("planning", "execution")
}

func (s *SprintService) sprintStatusesForPhases(phases ...string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, phase := range phases {
		for _, status := range canonicalWorkflowStatuses(s.workflowSvc, s.workflowSvc.GetStatusesByPhase(phase)) {
			if !seen[status] {
				seen[status] = true
				result = append(result, status)
			}
		}
	}
	return result
}

func (s *SprintService) assignmentOccupyingSprintStatuses() []string {
	return workflowStatusVocabulary(
		s.workflowSvc,
		s.sprintStatusesForPhases("planning", "research", "execution"),
	)
}

// statusInPhase returns the designated status of the configured sprint
// workflow's named phase (the sole candidate, or the one tagged
// primary: true), falling back to the given literal if the workflow defines
// none, so custom workflows with renamed statuses work without code changes.
// A phase with several candidates and no primary designation is an error —
// the sprint lifecycle never picks one arbitrarily.
func (s *SprintService) statusInPhase(phase, fallback string) (string, error) {
	status, err := s.workflowSvc.StatusForPhase(phase)
	return s.canonicalSelectorWithFallback(fallback, status, err)
}

func (s *SprintService) canonicalSelectorWithFallback(fallback, status string, err error) (string, error) {
	selected, selectErr := selectorWithFallback(fallback, status, err)
	if selectErr != nil {
		return "", selectErr
	}
	return s.workflowSvc.NormalizeStatus(selected), nil
}

// selectorWithFallback resolves a named-selector result: the selected status
// on success, the literal fallback when the workflow defines no candidate,
// and the error itself when the selection is ambiguous (never an arbitrary
// pick).
func selectorWithFallback(fallback, status string, err error) (string, error) {
	if err == nil {
		return status, nil
	}
	var noCandidate *config.NoCandidateError
	if errors.As(err, &noCandidate) {
		return fallback, nil
	}
	return "", err
}

// executionPhaseStatus returns the designated status of the configured sprint
// workflow's "execution" phase (falling back to "active").
func (s *SprintService) executionPhaseStatus() (string, error) {
	return s.statusInPhase("execution", "active")
}

// startTransitionTarget resolves the status StartSprint should move a sprint
// to from currentStatus, given the workflow's designated execution-phase
// status (executionStatus).
//
// It prefers a direct hop to executionStatus. When a custom route-based
// planning workflow defines an explicit intermediate pass route instead, it
// advances one hop at a time rather than failing or skipping that gate.
func (s *SprintService) startTransitionTarget(currentStatus, executionStatus string) string {
	if s.workflowSvc.ValidateTransition(currentStatus, executionStatus) == nil {
		return executionStatus
	}
	if s.workflowSvc.IsRouteBased() && workflowStatusMatchesAny(
		s.workflowSvc, currentStatus, s.workflowSvc.GetStatusesByPhase("planning"),
	) {
		if target := s.workflowSvc.GetOutcomes(currentStatus)["pass"]; target != "" {
			return target
		}
	}
	return executionStatus
}

// reviewPhaseStatus returns the designated status of the configured sprint
// workflow's "review" phase (falling back to "closing").
func (s *SprintService) reviewPhaseStatus() (string, error) {
	return s.statusInPhase("review", "closing")
}

// terminalSprintStatus returns the terminal status ArchiveSprint should
// transition into. This specifically wants "the" archive endpoint, so
// terminals whose orchestrator action is "archive" take precedence and the
// primary: true tag breaks any remaining tie (see ArchiveTerminalStatus).
// Falls back to the literal "archived" when the workflow defines no
// terminals, so custom workflows with renamed statuses still work without
// code changes.
func (s *SprintService) terminalSprintStatus() (string, error) {
	status, err := s.workflowSvc.ArchiveTerminalStatus()
	return s.canonicalSelectorWithFallback("archived", status, err)
}

// completedSprintStatus returns the "done"-phase status a sprint moves to
// once its carryover has been processed but before it is archived (e.g.
// "completed"), falling back to "completed" if the workflow defines none.
// Phase alone can't disambiguate this from the terminal status, since both
// typically share the "done" phase, so terminal statuses are excluded (see
// CompletedSprintStatus).
func (s *SprintService) completedSprintStatus() (string, error) {
	status, err := s.workflowSvc.CompletedSprintStatus()
	return s.canonicalSelectorWithFallback("completed", status, err)
}

// sprintAcceptsAssignments reports whether a sprint in the given status may
// accept new entity assignments. Delegates to workflow.Service to discover
// which statuses live in the planning and execution phases, so custom sprint
// workflows with renamed statuses work without code changes.
//
// Comparison is case-insensitive, matching workflow.Service's other status
// comparison helpers (IsTerminalStatus, IsValidTransition).
func (s *SprintService) sprintAcceptsAssignments(status string) bool {
	return workflowStatusMatchesAny(s.workflowSvc, status, s.assignableSprintStatuses())
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

var sprintAssignableWorkflowLevels = []string{
	entitytype.WorkflowTask,
	entitytype.WorkflowBug,
	entitytype.WorkflowChange,
	entitytype.WorkflowTechDebt,
}

// validBacklogEntityTypes is the allowlist of storage entity types accepted by
// GetSprintBacklog's EntityType filter. The service validates against this set
// before passing to the repository to prevent invalid values from reaching the
// UNION query.
var validBacklogEntityTypes = func() map[string]bool {
	result := make(map[string]bool, len(sprintAssignableWorkflowLevels))
	for _, level := range sprintAssignableWorkflowLevels {
		result[normalizeBacklogEntityType(level)] = true
	}
	return result
}()

func normalizeBacklogEntityType(raw string) string {
	normalized := entitytype.WorkflowLevelOrSelf(raw)
	if normalized == entitytype.WorkflowChange {
		return "change_card"
	}
	return normalized
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
	// Default: "ordered" for execution-phase sprints; "grouped" for all other statuses.
	effectiveView := opts.View
	if effectiveView == "" {
		executionStatuses := s.workflowSvc.GetStatusesByPhase("execution")
		isExecution := workflowStatusMatchesAny(s.workflowSvc, string(sprintEntity.Status), executionStatuses)
		if isExecution {
			effectiveView = "ordered"
		} else {
			effectiveView = "grouped"
		}
	}

	// Step 3: Validate entity type filter
	var entityTypeFilter *string
	if opts.EntityType != "" {
		normalizedEntityType := normalizeBacklogEntityType(opts.EntityType)
		if !validBacklogEntityTypes[normalizedEntityType] {
			return nil, fmt.Errorf(
				"invalid entity type %q: must be one of task, bug, change, change_card, tech_debt",
				opts.EntityType,
			)
		}
		entityTypeFilter = &normalizedEntityType
	}

	// Step 4: Determine blocked statuses from workflow service (never hardcode "blocked")
	entityWorkflowIndex := newSprintEntityWorkflowIndex(s.workflowSvc)
	var blockedStatuses []string
	if opts.BlockedOnly {
		blockedStatuses = entityWorkflowIndex.blockedStatusVocabulary()
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
	if opts.BlockedOnly {
		filtered := make([]*sprint.BacklogItem, 0, len(items))
		for _, item := range items {
			if entityWorkflowIndex.isBlocked(item.EntityType, item.Status) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	totalCount := len(items)
	completedCount := 0

	// Count completed items across both view modes.
	for _, item := range items {
		if entityWorkflowIndex.isTerminal(item.EntityType, item.Status) {
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

type sprintEntityWorkflowIndex struct {
	workflows        map[string]*workflow.Service
	terminalStatuses map[string]map[string]bool
	blockedStatuses  map[string]map[string]bool
}

func newSprintEntityWorkflowIndex(sprintWorkflow *workflow.Service) sprintEntityWorkflowIndex {
	index := sprintEntityWorkflowIndex{
		workflows:        make(map[string]*workflow.Service, len(sprintAssignableWorkflowLevels)),
		terminalStatuses: make(map[string]map[string]bool),
		blockedStatuses:  make(map[string]map[string]bool),
	}
	for _, level := range sprintAssignableWorkflowLevels {
		index.workflows[normalizeBacklogEntityType(level)] = sprintWorkflow.ForLevel(level)
	}
	for entityType, entityWorkflow := range index.workflows {
		index.terminalStatuses[entityType] = terminalSet(entityWorkflow)
		blockedStatuses := entityWorkflow.GetStatusesByPhase("blocked")
		if len(blockedStatuses) == 0 {
			blockedStatuses = []string{"blocked"}
		}
		index.blockedStatuses[entityType] = normalizedStatusSet(entityWorkflow, blockedStatuses)
	}
	return index
}

func (i sprintEntityWorkflowIndex) workflowFor(entityType string) (*workflow.Service, string, bool) {
	storageEntityType := normalizeBacklogEntityType(entityType)
	entityWorkflow, ok := i.workflows[storageEntityType]
	return entityWorkflow, storageEntityType, ok
}

func (i sprintEntityWorkflowIndex) isTerminal(entityType, status string) bool {
	entityWorkflow, storageEntityType, ok := i.workflowFor(entityType)
	return ok && i.terminalStatuses[storageEntityType][workflowStatusKey(entityWorkflow, status)]
}

func (i sprintEntityWorkflowIndex) isBlocked(entityType, status string) bool {
	entityWorkflow, storageEntityType, ok := i.workflowFor(entityType)
	return ok && i.blockedStatuses[storageEntityType][workflowStatusKey(entityWorkflow, status)]
}

func (i sprintEntityWorkflowIndex) openBacklogItems(items []sprint.BacklogItem) []sprint.BacklogItem {
	result := make([]sprint.BacklogItem, 0, len(items))
	for _, item := range items {
		if !i.isTerminal(item.EntityType, item.Status) {
			result = append(result, item)
		}
	}
	return result
}

func (i sprintEntityWorkflowIndex) blockedStatusVocabulary() []string {
	seen := make(map[string]bool)
	var result []string
	for _, level := range sprintAssignableWorkflowLevels {
		entityType := normalizeBacklogEntityType(level)
		entityWorkflow := i.workflows[entityType]
		blockedStatuses := entityWorkflow.GetStatusesByPhase("blocked")
		if len(blockedStatuses) == 0 {
			blockedStatuses = []string{"blocked"}
		}
		for _, status := range workflowStatusVocabulary(entityWorkflow, blockedStatuses) {
			if !seen[status] {
				seen[status] = true
				result = append(result, status)
			}
		}
	}
	return result
}

type sprintPullWorkflowIndex struct {
	entityWorkflows  sprintEntityWorkflowIndex
	requestedRole    string
	eligibleStatuses map[string]map[string]bool
}

func newSprintPullWorkflowIndex(sprintWorkflow *workflow.Service, requestedRole string) sprintPullWorkflowIndex {
	index := sprintPullWorkflowIndex{
		entityWorkflows:  newSprintEntityWorkflowIndex(sprintWorkflow),
		requestedRole:    requestedRole,
		eligibleStatuses: make(map[string]map[string]bool),
	}
	for entityType, entityWorkflow := range index.entityWorkflows.workflows {
		index.eligibleStatuses[entityType] = normalizedStatusSet(entityWorkflow, entityWorkflow.GetStatusesByAgentType(requestedRole))
	}
	return index
}

func (i sprintPullWorkflowIndex) allows(entityType, status string) bool {
	entityWorkflow, storageEntityType, ok := i.entityWorkflows.workflowFor(entityType)
	if !ok {
		return false
	}
	canonicalStatus := workflowStatusKey(entityWorkflow, status)
	if i.entityWorkflows.terminalStatuses[storageEntityType][canonicalStatus] {
		return false
	}
	return i.requestedRole == "" || i.eligibleStatuses[storageEntityType][canonicalStatus]
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
// Filters out terminal items for each entity-type workflow so the sprint queue can return
// any open assigned entity regardless of whether it is a task, bug, change card, or
// tech-debt item. If agentType is non-empty, only items for that agent are considered.
//
// The returned BacklogItemView has SprintOrder, SprintKey, and SelectionReason populated.
func (s *SprintService) GetNextTask(ctx context.Context, agentType string) (*BacklogItemView, error) {
	// 1. Find all execution-phase sprints. Tolerate multiple active sprints by iterating
	// through all execution statuses and collecting all sprints in any of them. The
	// ListSprints ordering (from the repository) provides a deterministic first-match.
	executionStatuses := s.workflowSvc.GetStatusesByPhase("execution")
	if len(executionStatuses) == 0 {
		// Fallback: use "active" as the default if the workflow has no "execution" phase
		executionStatuses = []string{"active"}
	}

	executionSprints, err := s.listSprintsByWorkflowStatuses(ctx, executionStatuses)
	if err != nil {
		return nil, fmt.Errorf("failed to list sprints in execution phase: %w", err)
	}
	if len(executionSprints) == 0 {
		return nil, fmt.Errorf("no active sprint found (start a sprint first)")
	}

	// 2. Collect candidates from all execution-phase sprints.
	// GetNextTask does its own four-tier sort, so it needs all items regardless of sprint_order.
	// Using View="grouped" explicitly avoids the active-sprint default of "ordered".

	// 3. Collect candidates whose status is non-terminal for their entity type.
	// Sprint execution order is an explicit pull queue across assigned items, so selection
	// must not be limited to workflow-initial statuses.
	workflowIndex := newSprintPullWorkflowIndex(s.workflowSvc, agentType)

	var candidates []*BacklogItemView
	for _, sp := range executionSprints {
		backlog, err := s.GetSprintBacklog(ctx, sp.Key, BacklogOptions{View: "grouped"})
		if err != nil {
			return nil, err
		}

		for _, group := range backlog.Groups {
			for _, item := range group.Items {
				// Apply the requested workflow role before sorting. BacklogItem.AgentType
				// is persisted planning/display data; the workflow step for the item's
				// current status is the authorization source for a role-aware pull.
				if !workflowIndex.allows(item.EntityType, item.Status) {
					continue
				}
				item.SprintKey = sp.Key
				candidates = append(candidates, item)
			}
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
	winner.SelectionReason = computeSelectionReason(winner, runnerUp)

	slog.Debug("sprint.next.selection",
		"reason", winner.SelectionReason,
		"sprint_id", winner.SprintKey,
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
	if !s.sprintAcceptsAssignments(status) {
		return nil, nil, fmt.Errorf(
			"cannot reorder: sprint %q is in status %q; only sprints in %s status can be reordered",
			sprintKey, status, strings.Join(s.assignableSprintStatuses(), " or "),
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
				"cannot reorder: position %d is out of range (sprint has %d ordered items, valid range: 1..%d)",
				newPosition, len(orderedAll), maxPos,
			)
		}
	default:
		return nil, nil, fmt.Errorf("cannot reorder: one of Position, Top, or Bottom must be set in ReorderTarget")
	}

	// Step 5: Build the renumber plan.
	//
	// siblings = ordered items excluding the target; buildRenumberOps assigns
	// them dense positions around the slot reserved for the target. The target
	// gets folded into finalOps so the entire post-reorder state lands in a
	// single CASE WHEN UPDATE.
	//
	// clearOps is a NULL pre-pass over every row that will change. Without it,
	// the renumber UPDATE transiently collides with
	// idx_sprint_assignments_order_unique — SQLite enforces partial unique
	// indexes per row as the statement executes, so shifting a value into a
	// slot still occupied by an unprocessed row trips the index.
	siblings := orderedWithoutTarget(orderedAll, assignment.ID)
	siblingOps := buildRenumberOps(siblings, newPosition)
	clearOps := buildReorderClearOps(assignment.ID, siblingOps)
	finalOps := make([]sprint.RenumberOp, 0, len(siblingOps)+1)
	finalOps = append(finalOps, sprint.RenumberOp{AssignmentID: assignment.ID, NewPosition: &newPosition})
	finalOps = append(finalOps, siblingOps...)

	// Step 6: Apply the two UPDATEs (clear, then assign).
	//
	// When s.db is nil (test path / CLI without --db) we drop the tx wrapper.
	// Otherwise we wrap both UPDATEs in one tx so a mid-failure rolls back to
	// the pre-reorder state.
	apply := func(tx *sql.Tx) error {
		if err := s.repo.RenumberAssignmentsTx(ctx, tx, sprintEntity.ID, clearOps); err != nil {
			return fmt.Errorf("cannot reorder: clear failed: %w", err)
		}
		if err := s.repo.RenumberAssignmentsTx(ctx, tx, sprintEntity.ID, finalOps); err != nil {
			return fmt.Errorf("cannot reorder: renumber failed: %w", err)
		}
		return nil
	}

	if s.db == nil {
		if err := apply(nil); err != nil {
			return nil, nil, err
		}
	} else {
		tx, txErr := s.db.BeginTxContext(ctx)
		if txErr != nil {
			return nil, nil, fmt.Errorf("cannot reorder: failed to begin transaction: %w", txErr)
		}
		defer tx.Rollback() //nolint:errcheck

		if err := apply(tx); err != nil {
			return nil, nil, err
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
		"renumbered", len(siblingOps),
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

// buildReorderClearOps returns ops that set sprint_order=NULL for the target
// assignment plus every sibling in renumberOps. Used as a pre-pass so the
// subsequent renumber UPDATE never transiently violates
// idx_sprint_assignments_order_unique.
func buildReorderClearOps(targetID int64, renumberOps []sprint.RenumberOp) []sprint.RenumberOp {
	clear := make([]sprint.RenumberOp, 0, len(renumberOps)+1)
	clear = append(clear, sprint.RenumberOp{AssignmentID: targetID, NewPosition: nil})
	for _, op := range renumberOps {
		clear = append(clear, sprint.RenumberOp{AssignmentID: op.AssignmentID, NewPosition: nil})
	}
	return clear
}

// buildRenumberOps builds a []RenumberOp for sibling assignments around a new position.
// siblings are already ordered by sprint_order (all ordered items excluding the target).
// The slot at newPosition is reserved for the target; siblings fill 1..newPosition-1
// and newPosition+1..len(siblings)+1 densely.
func buildRenumberOps(siblings []*models.SprintAssignment, newPosition int) []sprint.RenumberOp {
	ops := make([]sprint.RenumberOp, 0, len(siblings))
	pos := 1
	for _, sib := range siblings {
		if pos == newPosition {
			pos++ // skip the slot reserved for the target (exactly once)
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

	// Validate workflow transition. Use the designated archive terminal so
	// custom workflows with renamed statuses work without code changes.
	archivedStatus, err := s.terminalSprintStatus()
	if err != nil {
		return nil, recordSpanError(span, fmt.Errorf("cannot archive sprint %s: %w", key, err))
	}
	if err := s.workflowSvc.ValidateTransition(string(sprint.Status), archivedStatus); err != nil {
		return nil, recordSpanError(span, fmt.Errorf("cannot archive sprint %s in status %s: %w", key, sprint.Status, err))
	}

	// Update status
	if err := s.repo.UpdateStatus(ctx, sprint.ID, models.SprintStatus(archivedStatus)); err != nil {
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
	candidates, err := s.assignmentRepo.ListUnassignedBacklog(ctx, entityTypes, s.assignmentOccupyingSprintStatuses()...)
	if err != nil {
		return nil, fmt.Errorf("failed to list unassigned backlog for bulk add to sprint %s: %w",
			input.SprintKey, err)
	}

	result := &BulkAddResult{
		AddedByType:   make(map[string]int),
		SkippedByType: make(map[string]int),
	}
	entityWorkflowIndex := newSprintEntityWorkflowIndex(s.workflowSvc)
	openCandidates := make([]sprint.BacklogItem, 0, len(candidates))
	for _, candidate := range candidates {
		if entityWorkflowIndex.isTerminal(candidate.EntityType, candidate.Status) {
			result.SkippedByType[candidate.EntityType]++
			continue
		}
		openCandidates = append(openCandidates, candidate)
	}
	candidates = openCandidates

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
				if err := s.repo.RenumberAssignmentsTx(ctx, nil, sprintEntity.ID, ops); err != nil {
					return nil, fmt.Errorf("failed to repair sprint order after bulk add for sprint %s: %w", input.SprintKey, err)
				}
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
//  3. Fetches active assignments and backlog statuses, then classifies incomplete work per entity workflow.
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

	activeStatus, err := s.executionPhaseStatus()
	if err != nil {
		return nil, fmt.Errorf("cannot close sprint %s: %w", sprintKey, err)
	}
	if !workflowStatusMatchesAny(s.workflowSvc, string(sprintEntity.Status), []string{activeStatus}) {
		return nil, fmt.Errorf("cannot close sprint %s: current status is %q, must be %q",
			sprintKey, sprintEntity.Status, activeStatus)
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

	// Classify each assignment through its own entity workflow. Missing backlog
	// projections remain incomplete conservatively so an orphaned join can never
	// cause work to disappear during close.
	backlogItems, err := s.repo.ListBacklog(ctx, sprintEntity.ID, nil, false)
	if err != nil {
		return nil, fmt.Errorf("failed to list assignment statuses for sprint %s: %w", sprintKey, err)
	}
	entityWorkflowIndex := newSprintEntityWorkflowIndex(s.workflowSvc)
	incompleteByID := make(map[int64]bool, len(allAssignments))
	for _, assignment := range allAssignments {
		incompleteByID[assignment.ID] = true
	}
	for _, item := range backlogItems {
		if entityWorkflowIndex.isTerminal(item.EntityType, item.Status) {
			delete(incompleteByID, item.AssignmentID)
		}
	}
	incompleteAssignments := make([]*models.SprintAssignment, 0, len(incompleteByID))
	for _, assignment := range allAssignments {
		if incompleteByID[assignment.ID] {
			incompleteAssignments = append(incompleteAssignments, assignment)
		}
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
		nextSprint, nextErr := s.nextPlanningSprint(ctx, sprintEntity)
		if nextErr != nil {
			return nil, nextErr
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

	if err := s.recordSprintCloseTx(ctx, tx, sprintEntity, totalCount, completedCount, carriedOverCount, droppedCount, resolvedMode, nextSprintID); err != nil {
		return nil, err
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

// nextPlanningSprint finds the next planning sprint or creates the next
// date-contiguous one. It intentionally runs before reassignment so failures
// leave the close transaction untouched.
func (s *SprintService) nextPlanningSprint(ctx context.Context, closed *models.Sprint) (*models.Sprint, error) {
	planningStatus, err := s.statusInPhase("planning", "planning")
	if err != nil {
		return nil, fmt.Errorf("failed to find next planning sprint: %w", err)
	}
	planningSprints, err := s.listSprintsByWorkflowStatuses(ctx, []string{planningStatus})
	if err != nil {
		return nil, fmt.Errorf("failed to find next planning sprint: %w", err)
	}
	if len(planningSprints) > 0 {
		return planningSprints[0], nil
	}

	nextKey, err := s.repo.GetNextKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key for auto-created sprint: %w", err)
	}
	duration := closed.EndDate.Sub(closed.StartDate)
	start := closed.EndDate.AddDate(0, 0, 1)
	next := &models.Sprint{
		Key:       nextKey,
		Name:      "Sprint " + nextKey,
		StartDate: start,
		EndDate:   start.Add(duration),
		Status:    models.SprintStatus(s.workflowSvc.NormalizeStatus(s.workflowSvc.GetInitialStatusString())),
		Slug:      utils.GenerateSlug("Sprint " + nextKey),
	}
	if err := s.repo.Create(ctx, next); err != nil {
		return nil, fmt.Errorf("failed to auto-create next sprint: %w", err)
	}
	return next, nil
}

// recordSprintCloseTx performs the terminal status write and completion
// insert as one transaction-owned operation (TC-C08 and TC-C11).
func (s *SprintService) recordSprintCloseTx(ctx context.Context, tx *sql.Tx, sprintEntity *models.Sprint, totalCount, completedCount, carriedOverCount, droppedCount int, mode CarryoverMode, nextSprintID *int64) error {
	completedStatus, err := s.completedSprintStatus()
	if err != nil {
		return fmt.Errorf("failed to close sprint %s: %w", sprintEntity.Key, err)
	}
	if err := s.repo.UpdateStatusTx(ctx, tx, sprintEntity.ID, models.SprintStatus(completedStatus)); err != nil {
		return fmt.Errorf("failed to update sprint %s status in transaction: %w", sprintEntity.Key, err)
	}
	completion := &models.SprintCompletion{
		SprintID:             sprintEntity.ID,
		CompletedAt:          time.Now().UTC(),
		PlannedEntityCount:   totalCount,
		CompletedEntityCount: completedCount,
		CarriedOverCount:     carriedOverCount,
		DroppedCount:         droppedCount,
		CarryoverMode:        string(mode),
		NextSprintID:         nextSprintID,
	}
	if err := s.repo.CreateCompletionTx(ctx, tx, completion); err != nil {
		return fmt.Errorf("failed to create sprint_completions record for %s: %w", sprintEntity.Key, err)
	}
	return nil
}

// resolveCarryoverMode returns the effective CarryoverMode from config,
// defaulting to CarryoverNext when the config key is absent (TC-C10).
func (s *SprintService) resolveCarryoverMode() CarryoverMode {
	if s.cfg != nil && s.cfg.SprintDefaults != nil && s.cfg.SprintDefaults.CarryoverBehavior != "" {
		return CarryoverMode(s.cfg.SprintDefaults.CarryoverBehavior)
	}
	return CarryoverNext
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
		detail = "All task dependencies are satisfied (assigned to this sprint)"
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
	Catalog   *SprintCatalog       `json:"catalog,omitempty"`
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
		allTypes := make([]string, 0, len(sprintAssignableWorkflowLevels))
		for _, level := range sprintAssignableWorkflowLevels {
			allTypes = append(allTypes, normalizeBacklogEntityType(level))
		}
		backlog, err = s.assignmentRepo.ListUnassignedBacklog(ctx, allTypes, s.assignmentOccupyingSprintStatuses()...)
		if err != nil {
			return nil, fmt.Errorf("failed to list unassigned backlog for sprint %s plan: %w", key, err)
		}
		backlog = newSprintEntityWorkflowIndex(s.workflowSvc).openBacklogItems(backlog)
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

func normalizedStatusSet(svc *workflow.Service, statuses []string) map[string]bool {
	result := make(map[string]bool, len(statuses))
	for _, status := range statuses {
		result[workflowStatusKey(svc, status)] = true
	}
	return result
}

func workflowStatusKey(svc *workflow.Service, status string) string {
	return strings.ToLower(svc.NormalizeStatus(status))
}

func canonicalWorkflowStatuses(svc *workflow.Service, statuses []string) []string {
	seen := make(map[string]bool, len(statuses))
	result := make([]string, 0, len(statuses))
	for _, status := range statuses {
		canonical := svc.NormalizeStatus(status)
		key := strings.ToLower(canonical)
		if !seen[key] {
			seen[key] = true
			result = append(result, canonical)
		}
	}
	return result
}

func workflowStatusMatchesAny(svc *workflow.Service, status string, candidates []string) bool {
	return normalizedStatusSet(svc, candidates)[workflowStatusKey(svc, status)]
}

// workflowStatusVocabulary returns every persisted spelling equivalent to the
// supplied workflow statuses: the declared keys, their canonical steps, and
// every compatibility alias targeting those steps.
func workflowStatusVocabulary(svc *workflow.Service, statuses []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(statuses))
	appendStatus := func(status string) {
		if status != "" && !seen[status] {
			seen[status] = true
			result = append(result, status)
		}
	}
	canonicalTargets := normalizedStatusSet(svc, statuses)
	for _, status := range statuses {
		appendStatus(status)
		appendStatus(svc.NormalizeStatus(status))
	}

	aliases := svc.StatusAliasMap()
	aliasNames := make([]string, 0, len(aliases))
	for alias := range aliases {
		aliasNames = append(aliasNames, alias)
	}
	sort.Strings(aliasNames)
	for _, alias := range aliasNames {
		if canonicalTargets[workflowStatusKey(svc, aliases[alias])] {
			appendStatus(alias)
		}
	}
	return result
}

// terminalSet returns canonical terminal statuses for the given workflow level.
func terminalSet(svc *workflow.Service) map[string]bool {
	return normalizedStatusSet(svc, svc.GetTerminalStatuses())
}
