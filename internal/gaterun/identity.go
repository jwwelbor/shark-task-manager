package gaterun

import (
	"encoding/json"
	"fmt"
)

// RunIdentity is the immutable run_id -> entity binding this package commits
// BEFORE result.json ever becomes durably recoverable (UAT-3-1 fix:
// "Crash-recovery artifacts are committed before immutable owner and entity
// identity, allowing an uninitialized run result to be rebound to a
// different entity").
//
// UAT round 3+4 finding (note #2926): the original, narrower shape (just
// RunID/EntityKey/EntityType, deliberately omitting SourceStatus/Gate on the
// theory that those "can legitimately change across a resumed call with a
// fresh claim session") was insufficient — it let a same-entity result.json
// recovery derive a brand-new replay context (source status, gate,
// operation digest) from whatever the entity's LIVE status happened to be
// at resume time, instead of the context the run was originally created
// under. That prior rationale is superseded: a resumed call for the SAME
// run_id must resume the SAME operation, not merely the same entity, so
// RunIdentity now carries the complete replay-identity contract —
// RunID/EntityKey/EntityType PLUS SourceStatus/Gate/OperationDigest, the
// same trio OperationState already uses for VerifyResumeIdentity. A later
// call for the same run_id that disagrees on ANY of these six fields is a
// genuine identity conflict, not legitimate resume drift, and must fail
// closed exactly like an entity mismatch always did.
type RunIdentity struct {
	RunID      string `json:"run_id"`
	EntityKey  string `json:"entity_key"`
	EntityType string `json:"entity_type"`

	// SourceStatus, Gate, and OperationDigest complete the replay-identity
	// contract (UAT round 3+4, note #2926): the same trio OperationState
	// records via NewOperationState, computed once by the caller (see
	// gatepersist.Coordinator.Persist, which reuses the single
	// gaterun.ComputeOperationDigest call it already makes for
	// OperationState so identity.json and operation-state.json can never
	// disagree on digest for the same run_id).
	SourceStatus    string `json:"source_status"`
	Gate            string `json:"gate"`
	OperationDigest string `json:"operation_digest"`
}

// Marshal serializes r to its canonical identity.json bytes.
func (r RunIdentity) Marshal() ([]byte, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("gaterun: marshal run identity: %w", err)
	}
	return data, nil
}

// CreateIdentity atomically creates dir/identity.json exactly once, using
// the same create-once/no-replace/first-writer-wins protocol as
// CreateResult (createOnceSidecar): a first call for a given run_id durably
// binds rec (now the complete RunID/EntityKey/EntityType/SourceStatus/
// Gate/OperationDigest replay-identity contract — UAT round 3+4, note
// #2926); every later call whose fields all agree with the bound rec is
// idempotent success; a later call disagreeing on ANY field for the same
// run_id returns *ConflictError, exactly like a conflicting result.json
// replay — never silently rebinding the run to a new owner or a new replay
// context.
//
// Callers MUST call this before CreateResult (or any other operation that
// makes a run's result durably recoverable) for the same run_id, so a crash
// after this call but before the caller's next durable write can never leave
// a recoverable artifact whose owning entity and replay context are not yet
// bound. Callers MUST compute rec.OperationDigest the same way they compute
// the OperationDigest they will later pass to NewOperationState/
// VerifyResumeIdentity for the same call (see gatepersist.Coordinator.
// Persist), so identity.json and operation-state.json never disagree.
func CreateIdentity(dir string, rec RunIdentity) (created bool, err error) {
	if rec.RunID == "" || rec.EntityKey == "" || rec.EntityType == "" {
		return false, fmt.Errorf("gaterun: run identity requires a non-empty run_id, entity_key, and entity_type")
	}
	if rec.Gate == "" || rec.OperationDigest == "" {
		return false, fmt.Errorf("gaterun: run identity requires a non-empty gate and operation_digest")
	}
	data, err := rec.Marshal()
	if err != nil {
		return false, err
	}
	return createOnceSidecar(dir, identityFileName, ".identity-", data)
}

// ReadIdentity reads and decodes dir/identity.json, or returns (nil, false,
// nil) if it does not exist yet.
func ReadIdentity(dir string) (*RunIdentity, bool, error) {
	data, exists, err := readOnceSidecar(dir, identityFileName)
	if err != nil || !exists {
		return nil, exists, err
	}
	var rec RunIdentity
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, true, fmt.Errorf("gaterun: decode run identity: %w", err)
	}
	return &rec, true, nil
}

// VerifyRunIdentityOwner fails closed when rec's bound entity does not
// exactly match the caller's requested entityKey/entityType. This is the
// OWNER-ONLY subset of the replay-identity contract (entity binding alone,
// not the full SourceStatus/Gate/OperationDigest replay context) — see
// VerifyRunIdentity for the complete check required before deriving replay
// context from an uninitialized run.
//
// This narrower check is kept only for a caller that has not yet been
// wired to supply the full expectation (internal/cli/commands/run_resume.go
// resumeGateIngestForUninitializedState, T-E34-F05-004's scope per the
// coordinated fix for note #2926); every NEW caller must use
// VerifyRunIdentity instead.
func VerifyRunIdentityOwner(rec *RunIdentity, entityKey, entityType string) error {
	if rec == nil {
		return fmt.Errorf("gaterun: cannot verify run identity against a nil record")
	}
	if rec.EntityKey != entityKey {
		return fmt.Errorf("gaterun: run identity entity_key mismatch: recorded %q, requested %q", rec.EntityKey, entityKey)
	}
	if rec.EntityType != entityType {
		return fmt.Errorf("gaterun: run identity entity_type mismatch: recorded %q, requested %q", rec.EntityType, entityType)
	}
	return nil
}

// VerifyRunIdentity fails closed when rec does not exactly match want on
// every field of the replay-identity contract — RunID is intentionally
// excluded from the comparison (rec and want are always looked up BY the
// same run_id already, via ReadIdentity(dir) for that run's directory, so
// comparing it again would be redundant, not an additional check) but
// EntityKey, EntityType, SourceStatus, Gate, and OperationDigest must all
// agree.
//
// UAT round 3+4 finding (note #2926): the prior EntityKey/EntityType-only
// check let a same-entity result.json recovery derive a NEW replay context
// (source status, gate, operation digest) from whatever the entity's live
// status happened to be at resume time, instead of the context the run was
// originally created under. Callers MUST call this — with the caller's
// CURRENT SourceStatus/Gate and a freshly computed OperationDigest — before
// deriving any replay context (e.g. a live-status lookup) from an
// uninitialized run's durable result. Used both by gatepersist.Coordinator
// (defense in depth alongside CreateIdentity's own content-conflict check)
// and by any resume path that reads a durable artifact before a full
// OperationState exists to check identity against (see
// internal/cli/commands.resumeGateIngestForUninitializedState, once wired
// by T-E34-F05-004 — see VerifyRunIdentityOwner's doc comment for the
// interim narrower check that caller uses today).
func VerifyRunIdentity(rec *RunIdentity, want RunIdentity) error {
	if rec == nil {
		return fmt.Errorf("gaterun: cannot verify run identity against a nil record")
	}
	if rec.EntityKey != want.EntityKey {
		return fmt.Errorf("gaterun: run identity entity_key mismatch: recorded %q, requested %q", rec.EntityKey, want.EntityKey)
	}
	if rec.EntityType != want.EntityType {
		return fmt.Errorf("gaterun: run identity entity_type mismatch: recorded %q, requested %q", rec.EntityType, want.EntityType)
	}
	if rec.SourceStatus != want.SourceStatus {
		return fmt.Errorf("gaterun: run identity source_status mismatch: recorded %q, requested %q", rec.SourceStatus, want.SourceStatus)
	}
	if rec.Gate != want.Gate {
		return fmt.Errorf("gaterun: run identity gate mismatch: recorded %q, requested %q", rec.Gate, want.Gate)
	}
	if rec.OperationDigest != want.OperationDigest {
		return fmt.Errorf("gaterun: run identity operation_digest mismatch: recorded %q, requested %q", rec.OperationDigest, want.OperationDigest)
	}
	return nil
}
