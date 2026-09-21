package sprint

import "time"

// VelocityRow is one sprint's velocity data, returned by GetVelocityData.
// Entities with size IS NULL contribute 0 to CompletedSize and are counted in UnsizedCompleted.
type VelocityRow struct {
	SprintKey        string
	SprintName       string
	CompletedSize    int
	UnsizedCompleted int
}

// VelocitySprint identifies a sprint selected for velocity calculation. Item
// completion is intentionally calculated in the service layer from each
// entity's workflow, rather than by this repository projection.
type VelocitySprint struct {
	ID   int64
	Key  string
	Name string
}

// AssignedEntity represents one row from the polymorphic sprint_assignments join.
// Used for burndown reconstruction and sprint summary calculations.
type AssignedEntity struct {
	Key        string
	EntityType string
	EntityID   int64
	Status     string
	AssignedAt time.Time
	RemovedAt  *time.Time
	Size       *int // from entity table; nil when size IS NULL
}

// TaskCompletionEvent is a status transition that constitutes "entity completed" for burndown.
// Terminal states are determined by the service layer (which owns workflow knowledge).
type TaskCompletionEvent struct {
	EntityID   int64
	EntityType string
	NewStatus  string
	Timestamp  time.Time
}

// PhaseTimeRow is one phase's average duration, derived from task_history transitions.
// GetCycleTimeByPhase returns a slice of PhaseTimeRow, or an empty slice (not error)
// when work_sessions is empty.
type PhaseTimeRow struct {
	Phase       string // old_status of the task_history transition
	AverageDays float64
}
