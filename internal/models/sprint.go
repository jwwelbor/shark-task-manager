package models

import (
	"errors"
	"strings"
	"time"
)

// SprintStatus represents the workflow status of a sprint.
//
// Valid values are determined by the workflow configuration and validated at
// the service layer (mirrors the TaskStatus / BugStatus pattern). The model
// layer intentionally does NOT enumerate valid values — doing so would force
// the models package to depend on the workflow package and would couple the
// type to a specific workflow profile.
type SprintStatus string

// Sprint represents a time-boxed iteration of work containing assigned tasks,
// bugs, change-cards, and tech-debt items.
//
// This struct is the foundational data type used by the future SprintRepository
// and SprintService (introduced in E19-F02). It carries no progress, velocity,
// or capacity-allocation metrics — those are derived at the service layer from
// SprintAssignment and SprintCapacity rows.
type Sprint struct {
	ID        int64        `json:"id" db:"id"`
	Key       string       `json:"key" db:"key"`
	Name      string       `json:"name" db:"name"`
	Goal      string       `json:"goal,omitempty" db:"goal"`
	StartDate time.Time    `json:"start_date" db:"start_date"`
	EndDate   time.Time    `json:"end_date" db:"end_date"`
	Status    SprintStatus `json:"status" db:"status"`
	Slug      string       `json:"slug,omitempty" db:"slug"`
	FilePath  string       `json:"file_path,omitempty" db:"file_path"`
	CreatedAt time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt time.Time    `json:"updated_at" db:"updated_at"`
}

// SprintAssignment is a polymorphic association row linking a sprint to a
// task, bug, change-card, or tech-debt item.
//
// EntityType is constrained at the app layer only (see
// ValidateSprintAssignmentEntityType in validation.go); per the post-B018
// convention, the underlying sprint_assignments table does NOT carry a
// CHECK constraint on entity_type. Adding a fifth assignable entity type
// later requires updating only the Go validator — no DB migration.
//
// RemovedAt is nullable: a NULL value means the assignment is currently
// active, and the partial unique index (entity_type, entity_id) WHERE
// removed_at IS NULL enforces the "one active sprint per entity" rule at
// the database layer.
type SprintAssignment struct {
	ID          int64      `json:"id" db:"id"`
	SprintID    int64      `json:"sprint_id" db:"sprint_id"`
	EntityType  string     `json:"entity_type" db:"entity_type"`
	EntityID    int64      `json:"entity_id" db:"entity_id"`
	AssignedAt  time.Time  `json:"assigned_at" db:"assigned_at"`
	RemovedAt   *time.Time `json:"removed_at,omitempty" db:"removed_at"`
	SprintOrder *int       `json:"sprint_order,omitempty" db:"sprint_order"` // nullable; nil = unordered
}

// SprintCapacity records how many story points an agent type can deliver in
// a given sprint. AllocatedPoints is nullable in this feature because no
// triggers maintain it yet — capacity allocation queries arrive in E19-F05.
type SprintCapacity struct {
	ID              int64     `json:"id" db:"id"`
	SprintID        int64     `json:"sprint_id" db:"sprint_id"`
	AgentType       string    `json:"agent_type" db:"agent_type"`
	CapacityPoints  float64   `json:"capacity_points" db:"capacity_points"`
	AllocatedPoints *float64  `json:"allocated_points,omitempty" db:"allocated_points"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// Validate performs structural validation on the Sprint model.
//
// It does NOT check workflow status validity — that is the service layer's
// responsibility (mirrors how Task / Bug / TechDebt structs delegate status
// checks to workflow.Service).
func (s *Sprint) Validate() error {
	if err := ValidateSprintKey(s.Key); err != nil {
		return err
	}
	if strings.TrimSpace(s.Name) == "" {
		// Reuse ErrEmptyTitle: semantically "name cannot be empty" — sprints
		// use Name rather than Title, but the empty-string sentinel is shared.
		return ErrEmptyTitle
	}
	if !s.EndDate.After(s.StartDate) {
		return errors.New("sprint end_date must be after start_date")
	}
	if strings.TrimSpace(string(s.Status)) == "" {
		return errors.New("sprint status cannot be empty")
	}
	return nil
}

// SprintCompletion records the summary statistics generated when a sprint is
// closed via CloseSprintWithCarryover. One row is created per sprint close
// operation and is the primary input for velocity analytics in E19-F04.
//
// PlannedSizeSum and CompletedSizeSum are nullable (pointer-to-float64) because
// not all entities carry a size value — if all assigned entities are unsized the
// sums are nil rather than 0.0 to distinguish "unsized" from "zero velocity".
//
// NextSprintID is populated only when CarryoverMode == "next". When
// CarryoverMode == "backlog" it is nil (incomplete entities were released to
// the backlog rather than moved to another sprint).
type SprintCompletion struct {
	ID                   int64     `json:"id" db:"id"`
	SprintID             int64     `json:"sprint_id" db:"sprint_id"`
	CompletedAt          time.Time `json:"completed_at" db:"completed_at"`
	PlannedEntityCount   int       `json:"planned_entity_count" db:"planned_entity_count"`
	CompletedEntityCount int       `json:"completed_entity_count" db:"completed_entity_count"`
	CarriedOverCount     int       `json:"carried_over_count" db:"carried_over_count"`
	DroppedCount         int       `json:"dropped_count" db:"dropped_count"`
	PlannedSizeSum       *float64  `json:"planned_size_sum,omitempty" db:"planned_size_sum"`     // nil if all entities are unsized
	CompletedSizeSum     *float64  `json:"completed_size_sum,omitempty" db:"completed_size_sum"` // nil if all entities are unsized
	CarryoverMode        string    `json:"carryover_mode" db:"carryover_mode"`                   // "next" | "backlog"
	NextSprintID         *int64    `json:"next_sprint_id,omitempty" db:"next_sprint_id"`
}

// Validate enforces structural invariants on a SprintAssignment row:
//   - SprintID and EntityID must be greater than 0 (FK targets must exist).
//   - EntityType must be in the {task, bug, change_card, tech_debt} allowlist.
//
// See ValidateSprintAssignmentEntityType for the rationale behind keeping
// this check at the app layer only (no DB CHECK constraint).
func (sa *SprintAssignment) Validate() error {
	if sa.SprintID <= 0 {
		return errors.New("sprint_id must be greater than 0")
	}
	if sa.EntityID <= 0 {
		return errors.New("entity_id must be greater than 0")
	}
	return ValidateSprintAssignmentEntityType(sa.EntityType)
}
