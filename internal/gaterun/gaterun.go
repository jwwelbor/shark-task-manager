// Package gaterun implements the T-E34-F05-002 durable sidecar transport for
// GateResult replay: the create-once result.json / atomically-replaced
// operation-state.json protocol under .shark/runs/<run-id>/, the canonical
// operation-digest and suboperation-ID derivation functions, filesystem
// safety (owner-only modes, no-follow symlink rejection, fsync'd atomic
// writes), and the resume/reconciliation read-path contract.
//
// This package is a leaf: it imports only the standard library, so it can be
// consumed by internal/runner, internal/services, and internal/cli/commands
// alike without an import cycle (internal/runner already imports
// internal/services). It deliberately does not import internal/gateresult —
// the "validated envelope" this package digests is accepted as opaque JSON
// bytes, so this package never needs to know GateResult's shape.
//
// Scope boundaries (see docs/plan/E34-prompt-and-skill-improvements/E34-F05-.../tasks/T-E34-F05-002.md):
//   - This package owns the sidecar transport, digest, suboperation-ID
//     derivation, filesystem safety, and the resume decision/reconciliation
//     contract (via the injected TargetRecordReader interface).
//   - It does NOT write notes, task history, or perform transitions — that is
//     T-E34-F05-003 (persistence coordinator) and T-E34-F05-004 (core/Rider
//     ingestion parity).
//   - It does NOT implement `--apply-result` (new-result ingestion); only
//     `--resume-run` (recovery of an already-created result) is in scope
//     here, per the task spec's deliverables list.
package gaterun

import (
	"errors"
	"fmt"
)

// maxSidecarBytes bounds every sidecar file read/write (result.json and
// operation-state.json). It is generous enough for a bounded GateResult
// envelope (see internal/gateresult's own per-field/collection bounds) while
// still rejecting unbounded/hostile input before it reaches memory.
const maxSidecarBytes = 5 * 1024 * 1024 // 5 MiB

// ConflictError reports that an existing result.json holds different bytes
// than the caller attempted to create. Per REQ-F-003, this is a rejection,
// never an overwrite: the accepted (first-writer) bytes are left untouched.
type ConflictError struct {
	Path string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("gaterun: %s already holds a different accepted result; conflicting replay rejected", e.Path)
}

// IsConflict reports whether err is (or wraps) a *ConflictError.
func IsConflict(err error) bool {
	var c *ConflictError
	return errors.As(err, &c)
}

// UnsafePathError reports that a sidecar path component failed the no-follow
// / regular-file-only safety check (REQ-NF-001).
type UnsafePathError struct {
	Path   string
	Reason string
}

func (e *UnsafePathError) Error() string {
	return fmt.Sprintf("gaterun: unsafe path %s: %s", e.Path, e.Reason)
}
