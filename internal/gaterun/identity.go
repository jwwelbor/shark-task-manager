package gaterun

import (
	"encoding/json"
	"fmt"
)

// RunIdentity is the immutable run_id -> entity binding this package commits
// BEFORE result.json ever becomes durably recoverable (UAT-3-1 fix:
// "Crash-recovery artifacts are committed before immutable owner and entity
// identity, allowing an uninitialized run result to be rebound to a
// different entity"). It is deliberately narrower than OperationState (no
// SourceStatus/Gate/session — those can legitimately change across a
// resumed call with a fresh claim session) so that CreateIdentity's
// create-once, first-writer-wins content comparison only ever fails closed
// on a genuine entity mismatch, never on an unrelated field drifting between
// calls for the SAME entity.
type RunIdentity struct {
	RunID      string `json:"run_id"`
	EntityKey  string `json:"entity_key"`
	EntityType string `json:"entity_type"`
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
// binds rec; every later call for the SAME entity is idempotent success; a
// later call naming a DIFFERENT entity for the same run_id returns
// *ConflictError, exactly like a conflicting result.json replay — never
// silently rebinding the run to a new owner.
//
// Callers MUST call this before CreateResult (or any other operation that
// makes a run's result durably recoverable) for the same run_id, so a crash
// after this call but before the caller's next durable write can never leave
// a recoverable artifact whose owning entity is not yet bound.
func CreateIdentity(dir string, rec RunIdentity) (created bool, err error) {
	if rec.RunID == "" || rec.EntityKey == "" || rec.EntityType == "" {
		return false, fmt.Errorf("gaterun: run identity requires a non-empty run_id, entity_key, and entity_type")
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

// VerifyRunIdentity fails closed when rec's bound entity does not exactly
// match the caller's requested entityKey/entityType. Used both by
// gatepersist.Coordinator (defense in depth alongside CreateIdentity's own
// content-conflict check) and by any resume path that reads a durable
// artifact before a full OperationState exists to check identity against
// (see internal/cli/commands.resumeGateIngestForUninitializedState).
func VerifyRunIdentity(rec *RunIdentity, entityKey, entityType string) error {
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
