package gatepersist

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// conflictingDurableRecordError reports that a durable note already exists
// under a suboperation ID this run would also write, but with different
// content — a conflicting replay (REQ-F-003's "a different digest for the
// same stable identity is a conflict", applied at suboperation granularity
// as defense in depth alongside the whole-envelope gaterun.ConflictError
// check).
type conflictingDurableRecordError struct {
	Kind     string
	Identity string
}

func (e *conflictingDurableRecordError) Error() string {
	return fmt.Sprintf("gatepersist: durable %s record for %q already exists with different content; conflicting replay rejected", e.Kind, e.Identity)
}

// reconciler implements gaterun.TargetRecordReader (see reconcile's single
// method, CompletedSuboperationIDs) by reading durable notes on the bound
// main entity and durable kickback history on every kickback target entity,
// per architecture.md step 7: "Notes store it in typed metadata; kickback
// reasons/history store its bounded machine token." This closes the
// target-commit/sidecar-update crash window (REQ-F-003) without a second
// store: notes and entity history remain the reconciliation authority.
type reconciler struct {
	noteReader    NoteReader
	historyReader HistoryReader

	mainEntityType models.EntityType
	mainEntityKey  string

	// kickbackEntities maps a kickback target entity key to its resolved
	// entity type, for every kickback in the current operation set.
	kickbackEntities map[string]models.EntityType

	ops             []operation
	operationDigest string
}

// CompletedSuboperationIDs implements gaterun.TargetRecordReader.
func (r *reconciler) CompletedSuboperationIDs(ctx context.Context, runID string) ([]string, error) {
	opByID := make(map[string]operation, len(r.ops))
	for _, op := range r.ops {
		opByID[op.suboperationID(r.operationDigest)] = op
	}

	var completed []string

	notes, err := r.noteReader.ListNotes(ctx, r.mainEntityType, r.mainEntityKey, nil)
	if err != nil {
		return nil, fmt.Errorf("gatepersist: list notes for reconciliation: %w", err)
	}
	for _, n := range notes {
		if n.Metadata == nil {
			continue
		}
		meta, ok := decodeMetadata(*n.Metadata)
		if !ok {
			continue
		}
		if runIDVal, _ := meta[metaRunID].(string); runIDVal != runID {
			continue
		}
		subID, _ := meta[metaSuboperationID].(string)
		if subID == "" {
			continue
		}
		op, ok := opByID[subID]
		if !ok || op.kind == kindKickback {
			// Not part of the current operation set (or a stale/foreign
			// note); reconciliation only merges IDs this run would itself
			// derive, so an unrelated note can never be misread as complete.
			continue
		}
		wantDigest := op.contentDigest()
		gotDigest, _ := meta[metaContentDigest].(string)
		if gotDigest != wantDigest {
			return nil, &conflictingDurableRecordError{Kind: op.kind, Identity: op.itemIdentity}
		}
		completed = append(completed, subID)
	}

	for entityKey, entityType := range r.kickbackEntities {
		history, err := r.historyReader.GetHistory(ctx, entityType, entityKey)
		if err != nil {
			return nil, fmt.Errorf("gatepersist: list history for reconciliation: %w", err)
		}
		for _, h := range history {
			if h.Notes == nil {
				continue
			}
			subID, gotDigest, gotRunID, ok := parseKickbackToken(*h.Notes)
			if !ok {
				continue
			}
			if gotRunID != runID {
				// Mirrors the notes branch's run_id filter above: a
				// suboperation ID alone does not identify which run
				// produced it (ComputeOperationDigest never includes
				// run_id), so a history record from a DIFFERENT run must
				// never be misread as this run's own completed
				// suboperation (code-review round 11 finding).
				continue
			}
			op, ok := opByID[subID]
			if !ok || op.kind != kindKickback || op.kickback.EntityKey != entityKey {
				continue
			}
			// Compare both target status AND content digest (which itself
			// covers entity key, target status, and reason — operations.go's
			// contentDigest). Comparing status alone let a same-status,
			// different-reason replay through silently (round-2 UAT
			// rejection of TD-178's gap): REQ-F-003 requires failing closed
			// on a conflicting target status OR reason, not status alone.
			wantDigest := op.contentDigest()
			if !strings.EqualFold(h.ToStatus, op.kickback.TargetStatus) || gotDigest != wantDigest {
				return nil, &KickbackConflictError{EntityKey: entityKey}
			}
			completed = append(completed, subID)
		}
	}

	return completed, nil
}

// decodeMetadata parses a note's metadata JSON string into a generic map.
// Malformed or foreign metadata (from a note this package did not write) is
// tolerated by returning ok=false rather than erroring the whole
// reconciliation pass.
func decodeMetadata(raw string) (map[string]interface{}, bool) {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, false
	}
	return m, true
}
