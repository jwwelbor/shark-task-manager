package gaterun

import (
	"encoding/json"
	"fmt"
	"time"
)

// PersistenceState is the REQ-F-003 durable phase of a run's operation
// state, distinguishing "notes/kickbacks durably written" from "the guarded
// workflow transition itself durably applied" so each crash window resumes
// exactly once.
type PersistenceState string

const (
	// PersistenceStatePending means no target write has been confirmed
	// durable yet for this run; resume must replay from the beginning of
	// the target-write sequence (reconciled against durable target records).
	PersistenceStatePending PersistenceState = "pending"
	// PersistenceStateComplete means every target write (notes,
	// kickbacks, sweeps, impacts) is durably recorded, but the guarded
	// workflow transition has not yet been confirmed applied.
	PersistenceStateComplete PersistenceState = "persistence_complete"
	// PersistenceStateTransitioned means the guarded workflow transition
	// itself is durably applied. Only lease release may remain.
	PersistenceStateTransitioned PersistenceState = "transition_applied"
)

// RetirementState is the worker-retirement evidence REQ-F-003/REQ-NF-001
// require independent of chat notifications (lease-session-agent-issues.md
// issues #1/#3).
type RetirementState string

const (
	RetirementUnknown RetirementState = "unknown"
	RetirementPending RetirementState = "pending"
	RetirementRetired RetirementState = "retired"
)

// OperationState is the REQ-F-003 replay journal persisted (atomically
// replaced, any number of times) as operation-state.json. It is the mutable
// counterpart to the immutable, create-once result.json.
type OperationState struct {
	RunID           string `json:"run_id"`
	EntityKey       string `json:"entity_key"`
	EntityType      string `json:"entity_type"`
	SourceStatus    string `json:"source_status"`
	Gate            string `json:"gate"`
	OperationDigest string `json:"operation_digest"`

	PersistenceState PersistenceState `json:"persistence_state"`
	// CompletedSuboperationIDs holds every suboperation ID (see
	// DeriveSuboperationID) whose target write has been confirmed durable,
	// in first-completed order. Membership, not order, is authoritative for
	// resume decisions.
	CompletedSuboperationIDs []string `json:"completed_suboperation_ids"`

	WorkerPhase     string          `json:"worker_phase,omitempty"`
	NestedOperation string          `json:"nested_operation,omitempty"`
	RetirementState RetirementState `json:"retirement_state"`
	ResultLocation  string          `json:"result_location,omitempty"`

	StartedAt time.Time `json:"started_at"`
	UpdatedAt time.Time `json:"updated_at"`

	completed map[string]struct{}
}

// NewOperationState constructs a fresh, PersistenceStatePending operation
// state for the given stable identity and digest.
func NewOperationState(runID, entityKey, entityType, sourceStatus, gate, operationDigest string) *OperationState {
	now := time.Now().UTC()
	return &OperationState{
		RunID:                    runID,
		EntityKey:                entityKey,
		EntityType:               entityType,
		SourceStatus:             sourceStatus,
		Gate:                     gate,
		OperationDigest:          operationDigest,
		PersistenceState:         PersistenceStatePending,
		CompletedSuboperationIDs: nil,
		RetirementState:          RetirementUnknown,
		StartedAt:                now,
		UpdatedAt:                now,
	}
}

// HasCompleted reports whether suboperationID is already recorded complete.
func (s *OperationState) HasCompleted(suboperationID string) bool {
	s.ensureIndex()
	_, ok := s.completed[suboperationID]
	return ok
}

// AddCompletedSuboperation records suboperationID as complete, returning
// true if it was newly added (false if it was already present — resuming an
// already-recorded suboperation is idempotent and never duplicates it).
func (s *OperationState) AddCompletedSuboperation(suboperationID string) bool {
	s.ensureIndex()
	if _, ok := s.completed[suboperationID]; ok {
		return false
	}
	s.completed[suboperationID] = struct{}{}
	s.CompletedSuboperationIDs = append(s.CompletedSuboperationIDs, suboperationID)
	s.UpdatedAt = time.Now().UTC()
	return true
}

func (s *OperationState) ensureIndex() {
	if s.completed != nil {
		return
	}
	s.completed = make(map[string]struct{}, len(s.CompletedSuboperationIDs))
	for _, id := range s.CompletedSuboperationIDs {
		s.completed[id] = struct{}{}
	}
}

// MarkPersistenceComplete transitions to PersistenceStateComplete. It is a
// no-op (returns nil) if already at or past that state, so repeated calls
// during a partial resume are safe.
func (s *OperationState) MarkPersistenceComplete() error {
	switch s.PersistenceState {
	case PersistenceStatePending:
		s.PersistenceState = PersistenceStateComplete
		s.UpdatedAt = time.Now().UTC()
		return nil
	case PersistenceStateComplete, PersistenceStateTransitioned:
		return nil
	default:
		return fmt.Errorf("gaterun: unknown persistence state %q", s.PersistenceState)
	}
}

// MarkTransitionApplied transitions to PersistenceStateTransitioned. It
// fails closed if persistence was never marked complete first, since a
// transition must not be recorded ahead of the evidence it depends on.
func (s *OperationState) MarkTransitionApplied() error {
	switch s.PersistenceState {
	case PersistenceStateComplete:
		s.PersistenceState = PersistenceStateTransitioned
		s.UpdatedAt = time.Now().UTC()
		return nil
	case PersistenceStateTransitioned:
		return nil
	default:
		return fmt.Errorf("gaterun: cannot mark transition applied from persistence state %q", s.PersistenceState)
	}
}

// Marshal serializes s to its canonical operation-state.json bytes.
func (s *OperationState) Marshal() ([]byte, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("gaterun: marshal operation state: %w", err)
	}
	return data, nil
}

// Save writes s to dir/operation-state.json via WriteOperationState
// (atomic replace).
func (s *OperationState) Save(dir string) error {
	data, err := s.Marshal()
	if err != nil {
		return err
	}
	return WriteOperationState(dir, data)
}

// UnmarshalOperationState decodes operation-state.json bytes.
func UnmarshalOperationState(data []byte) (*OperationState, error) {
	var s OperationState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("gaterun: decode operation state: %w", err)
	}
	s.ensureIndex()
	return &s, nil
}

// LoadOperationState reads and decodes dir/operation-state.json, or returns
// (nil, false, nil) if it does not exist yet.
func LoadOperationState(dir string) (*OperationState, bool, error) {
	data, exists, err := ReadOperationState(dir)
	if err != nil || !exists {
		return nil, exists, err
	}
	s, err := UnmarshalOperationState(data)
	if err != nil {
		return nil, true, err
	}
	return s, true, nil
}
