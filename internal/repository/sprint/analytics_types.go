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

// AssignedEntity represents one row from the polymorphic sprint_assignments join.
// Used for burndown reconstruction and sprint summary calculations.
type AssignedEntity struct {
	EntityType string
	EntityID   int64
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
