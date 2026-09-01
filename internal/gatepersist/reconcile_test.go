package gatepersist

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

func TestReconciler_MergesDurableNoteAndHistoryRecords(t *testing.T) {
	world := newFakeWorld()
	digest := "digest-1"
	ops := buildOperations(fixtureResult(), "summary text")

	// Simulate a prior crash that already durably wrote the gate-summary
	// note and applied the kickback transition, but never updated
	// operation-state.json to record either.
	summaryOp := ops[0]
	kickbackOp := ops[len(ops)-1]

	summaryMeta := map[string]interface{}{}
	for k, v := range summaryOp.metadata {
		summaryMeta[k] = v
	}
	summaryMeta[metaRunID] = "run-x"
	summaryMeta[metaSuboperationID] = summaryOp.suboperationID(digest)
	summaryMeta[metaContentDigest] = summaryOp.contentDigest()
	encoded, _ := json.Marshal(summaryMeta)
	if _, err := world.AddNoteWithMetadata(context.Background(), models.EntityTypeTask, mainEntityKey, summaryOp.noteType, summaryOp.content, "agent", string(encoded)); err != nil {
		t.Fatalf("seed note: %v", err)
	}

	world.setStatus(models.EntityTypeTask, kickbackOp.kickback.EntityKey, "in_review")
	subID := kickbackOp.suboperationID(digest)
	reason := buildKickbackReason(kickbackOp.kickback.Reason, subID)
	if _, _, err := world.Transition(context.Background(), models.EntityTypeTask, kickbackOp.kickback.EntityKey, kickbackOp.kickback.TargetStatus, reason, "agent"); err != nil {
		t.Fatalf("seed kickback transition: %v", err)
	}

	rec := &reconciler{
		noteReader:     world,
		historyReader:  world,
		mainEntityType: models.EntityTypeTask,
		mainEntityKey:  mainEntityKey,
		kickbackEntities: map[string]models.EntityType{
			kickbackOp.kickback.EntityKey: models.EntityTypeTask,
		},
		ops:             ops,
		operationDigest: digest,
	}

	ids, err := rec.CompletedSuboperationIDs(context.Background(), "run-x")
	if err != nil {
		t.Fatalf("CompletedSuboperationIDs: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 reconciled suboperations (summary note + kickback), got %d: %v", len(ids), ids)
	}
	found := map[string]bool{}
	for _, id := range ids {
		found[id] = true
	}
	if !found[summaryOp.suboperationID(digest)] || !found[subID] {
		t.Fatalf("expected both the summary and kickback suboperation IDs, got %v", ids)
	}
}

func TestReconciler_ConflictingKickbackHistoryFailsClosed(t *testing.T) {
	world := newFakeWorld()
	digest := "digest-2"
	ops := buildOperations(fixtureResult(), "summary text")
	kickbackOp := ops[len(ops)-1]
	subID := kickbackOp.suboperationID(digest)

	// Durable history records the same suboperation ID but a DIFFERENT
	// target status than what this run's operation set would apply --
	// exactly the "conflicting target status ... on retry fails closed"
	// acceptance criterion.
	world.setStatus(models.EntityTypeTask, kickbackOp.kickback.EntityKey, "in_review")
	conflictingReason := buildKickbackReason("a different reason", subID)
	if _, _, err := world.Transition(context.Background(), models.EntityTypeTask, kickbackOp.kickback.EntityKey, "completed", conflictingReason, "agent"); err != nil {
		t.Fatalf("seed conflicting kickback transition: %v", err)
	}

	rec := &reconciler{
		noteReader:     world,
		historyReader:  world,
		mainEntityType: models.EntityTypeTask,
		mainEntityKey:  mainEntityKey,
		kickbackEntities: map[string]models.EntityType{
			kickbackOp.kickback.EntityKey: models.EntityTypeTask,
		},
		ops:             ops,
		operationDigest: digest,
	}

	_, err := rec.CompletedSuboperationIDs(context.Background(), "run-x")
	if err == nil {
		t.Fatalf("expected a conflicting kickback history record to fail closed")
	}
	var kerr *KickbackConflictError
	if !errors.As(err, &kerr) {
		t.Fatalf("expected *KickbackConflictError, got %T: %v", err, err)
	}
}

func TestReconciler_ConflictingNoteContentFailsClosed(t *testing.T) {
	world := newFakeWorld()
	digest := "digest-3"
	ops := buildOperations(fixtureResult(), "summary text")
	summaryOp := ops[0]

	meta := map[string]interface{}{
		metaRunID:          "run-x",
		metaSuboperationID: summaryOp.suboperationID(digest),
		metaContentDigest:  "not-the-real-digest",
	}
	encoded, _ := json.Marshal(meta)
	if _, err := world.AddNoteWithMetadata(context.Background(), models.EntityTypeTask, mainEntityKey, summaryOp.noteType, "different content than the operation expects", "agent", string(encoded)); err != nil {
		t.Fatalf("seed conflicting note: %v", err)
	}

	rec := &reconciler{
		noteReader:       world,
		historyReader:    world,
		mainEntityType:   models.EntityTypeTask,
		mainEntityKey:    mainEntityKey,
		kickbackEntities: map[string]models.EntityType{},
		ops:              ops,
		operationDigest:  digest,
	}

	_, err := rec.CompletedSuboperationIDs(context.Background(), "run-x")
	if err == nil {
		t.Fatalf("expected conflicting note content to fail closed")
	}
}

func TestReconciler_IgnoresUnrelatedNotesAndHistory(t *testing.T) {
	world := newFakeWorld()
	digest := "digest-4"
	ops := buildOperations(fixtureResult(), "summary text")

	// A note from a different run_id, and one with no metadata at all, must
	// both be ignored rather than mistaken for this run's records.
	if _, err := world.AddNoteWithMetadata(context.Background(), models.EntityTypeTask, mainEntityKey, "comment", "unrelated", "agent", ""); err != nil {
		t.Fatalf("seed unrelated note: %v", err)
	}
	otherRunMeta, _ := json.Marshal(map[string]interface{}{metaRunID: "some-other-run", metaSuboperationID: "whatever"})
	if _, err := world.AddNoteWithMetadata(context.Background(), models.EntityTypeTask, mainEntityKey, "review", "from another run", "agent", string(otherRunMeta)); err != nil {
		t.Fatalf("seed other-run note: %v", err)
	}

	rec := &reconciler{
		noteReader:       world,
		historyReader:    world,
		mainEntityType:   models.EntityTypeTask,
		mainEntityKey:    mainEntityKey,
		kickbackEntities: map[string]models.EntityType{},
		ops:              ops,
		operationDigest:  digest,
	}

	ids, err := rec.CompletedSuboperationIDs(context.Background(), "run-x")
	if err != nil {
		t.Fatalf("CompletedSuboperationIDs: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("expected no suboperations reconciled from unrelated notes, got %v", ids)
	}
}
