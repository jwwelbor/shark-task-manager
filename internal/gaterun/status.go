package gaterun

import "time"

// StatusProjection is the REQ-F-003/Deliverables "live parent/operator
// progress and retirement visibility" surface: worker phase, nested
// operation, elapsed time, retirement state, and result location, derived
// read-only from an OperationState. It is independent of chat notifications
// (lease-session-agent-issues.md issues #1/#3) — anything that can read the
// run directory can compute this without depending on a live worker
// connection.
type StatusProjection struct {
	RunID            string           `json:"run_id"`
	EntityKey        string           `json:"entity_key"`
	PersistenceState PersistenceState `json:"persistence_state"`
	WorkerPhase      string           `json:"worker_phase,omitempty"`
	NestedOperation  string           `json:"nested_operation,omitempty"`
	RetirementState  RetirementState  `json:"retirement_state"`
	ElapsedSeconds   float64          `json:"elapsed_seconds"`
	ResultLocation   string           `json:"result_location,omitempty"`
	CompletedCount   int              `json:"completed_suboperation_count"`
}

// ProjectStatus derives a StatusProjection from state as of now.
func ProjectStatus(state *OperationState, now time.Time) StatusProjection {
	return StatusProjection{
		RunID:            state.RunID,
		EntityKey:        state.EntityKey,
		PersistenceState: state.PersistenceState,
		WorkerPhase:      state.WorkerPhase,
		NestedOperation:  state.NestedOperation,
		RetirementState:  state.RetirementState,
		ElapsedSeconds:   now.Sub(state.StartedAt).Seconds(),
		ResultLocation:   state.ResultLocation,
		CompletedCount:   len(state.CompletedSuboperationIDs),
	}
}
