package gaterun

import (
	"context"
	"fmt"
)

// ResumeAction is the outcome of DecideResume's read of a run directory's
// sidecar state, per REQ-F-003's crash-window table: "a durable
// transition_applied state plus the expected live target returns success;
// persistence_complete resumes the guarded transition; a partial
// persistence state resumes its next operation."
type ResumeAction string

const (
	// ResumeActionAlreadyTransitioned: operation-state.json already records
	// transition_applied. The caller's only remaining duty is to verify the
	// expected live target state and release the lease; it must not repeat
	// the transition.
	ResumeActionAlreadyTransitioned ResumeAction = "already_transitioned"
	// ResumeActionResumeTransition: every target write is durably recorded
	// (persistence_complete) but the transition itself is not yet
	// confirmed. The caller must (re-)apply the guarded transition.
	ResumeActionResumeTransition ResumeAction = "resume_transition"
	// ResumeActionResumeNextOperation: persistence is only partially
	// complete (or not yet started). The caller must resume writing
	// remaining target operations, skipping any suboperation ID already
	// present in CompletedSuboperationIDs.
	ResumeActionResumeNextOperation ResumeAction = "resume_next_operation"
)

// ResumeDecision is DecideResume's result: the action to take, the loaded
// operation state (nil only if no result.json exists at all — see
// DecideResume's error return in that case), and the accepted result.json
// bytes for the caller to re-validate against its own expected envelope.
type ResumeDecision struct {
	Action ResumeAction
	State  *OperationState
	Result []byte
}

// ErrNoDurableResult is returned by DecideResume when dir has no
// result.json yet: there is nothing to resume. A resume request against
// such a run_id is a caller error (it should have used a fresh run, not
// --resume-run), not a recoverable resume path.
var ErrNoDurableResult = fmt.Errorf("gaterun: no durable result found for this run; nothing to resume")

// DecideResume reads dir's sidecar files and returns the REQ-F-003 resume
// decision. It does not itself perform reconciliation against durable
// target records (see ReconcileCompletedSuboperations) or verify caller-
// supplied identity/digest (see VerifyResumeIdentity) — callers compose
// those as needed.
func DecideResume(dir string) (*ResumeDecision, error) {
	result, resultExists, err := ReadResult(dir)
	if err != nil {
		return nil, err
	}
	if !resultExists {
		return nil, ErrNoDurableResult
	}

	state, stateExists, err := LoadOperationState(dir)
	if err != nil {
		return nil, err
	}
	if !stateExists {
		// result.json committed but operation-state.json was never
		// initialized: the crash window between create-once result and
		// operation-state initialization (task spec Testing Strategy).
		// There is nothing recorded complete yet, so resume from the
		// start of target-write persistence.
		return &ResumeDecision{
			Action: ResumeActionResumeNextOperation,
			State:  nil,
			Result: result,
		}, nil
	}

	var action ResumeAction
	switch state.PersistenceState {
	case PersistenceStateTransitioned:
		action = ResumeActionAlreadyTransitioned
	case PersistenceStateComplete:
		action = ResumeActionResumeTransition
	case PersistenceStatePending:
		action = ResumeActionResumeNextOperation
	default:
		return nil, fmt.Errorf("gaterun: unknown persistence state %q in operation-state.json", state.PersistenceState)
	}

	return &ResumeDecision{Action: action, State: state, Result: result}, nil
}

// VerifyResumeIdentity fails closed when the caller's expected bound
// identity and operation digest do not exactly match the durable operation
// state's — a `shark run --resume-run` must never resume a different
// entity, source status, or a differently-digested operation under the same
// run_id.
func VerifyResumeIdentity(state *OperationState, entityKey, entityType, sourceStatus, operationDigest string) error {
	if state == nil {
		return fmt.Errorf("gaterun: cannot verify resume identity against a nil operation state")
	}
	if state.EntityKey != entityKey {
		return fmt.Errorf("gaterun: resume entity_key mismatch: recorded %q, requested %q", state.EntityKey, entityKey)
	}
	if state.EntityType != entityType {
		return fmt.Errorf("gaterun: resume entity_type mismatch: recorded %q, requested %q", state.EntityType, entityType)
	}
	if state.SourceStatus != sourceStatus {
		return fmt.Errorf("gaterun: resume source_status mismatch: recorded %q, requested %q", state.SourceStatus, sourceStatus)
	}
	if state.OperationDigest != operationDigest {
		return fmt.Errorf("gaterun: resume operation digest mismatch: recorded %q, requested %q", state.OperationDigest, operationDigest)
	}
	return nil
}

// TargetRecordReader is implemented by the caller (T-E34-F05-003's
// persistence coordinator, or a test stub) to expose durable target-write
// records — notes and bounded entity-history reason metadata — keyed by
// their suboperation ID. This package never reads notes or task history
// itself; it only consumes this injected read path.
type TargetRecordReader interface {
	// CompletedSuboperationIDs returns every suboperation ID durably
	// recorded as written for runID, regardless of whether
	// operation-state.json already reflects it.
	CompletedSuboperationIDs(ctx context.Context, runID string) ([]string, error)
}

// ReconcileCompletedSuboperations closes the target-commit/sidecar-update
// crash window (REQ-F-003): a target write (note/history record) can commit
// durably and then the process can crash before operation-state.json is
// updated to reflect it. This merges every suboperation ID durably recorded
// by reader into state, without duplicating IDs state already has. It does
// not write state to disk — call state.Save(dir) afterward when changed is
// true.
func ReconcileCompletedSuboperations(ctx context.Context, reader TargetRecordReader, state *OperationState) (changed bool, err error) {
	if reader == nil {
		return false, fmt.Errorf("gaterun: reconciliation requires a non-nil TargetRecordReader")
	}
	if state == nil {
		return false, fmt.Errorf("gaterun: reconciliation requires a non-nil operation state")
	}
	ids, err := reader.CompletedSuboperationIDs(ctx, state.RunID)
	if err != nil {
		return false, fmt.Errorf("gaterun: read durable target records for run %s: %w", state.RunID, err)
	}
	for _, id := range ids {
		if state.AddCompletedSuboperation(id) {
			changed = true
		}
	}
	return changed, nil
}
