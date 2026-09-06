package gaterun

import (
	"context"
	"errors"
	"testing"
)

type stubTargetRecordReader struct {
	ids []string
	err error
}

func (s stubTargetRecordReader) CompletedSuboperationIDs(_ context.Context, _ string) ([]string, error) {
	return s.ids, s.err
}

func TestDecideResume_NoResult(t *testing.T) {
	dir := newRunDir(t)
	_, err := DecideResume(dir)
	if !errors.Is(err, ErrNoDurableResult) {
		t.Fatalf("DecideResume on empty dir: err = %v, want ErrNoDurableResult", err)
	}
}

func TestDecideResume_ResultWithoutOperationState_ResumesNextOperation(t *testing.T) {
	// The crash window between create-once result.json and
	// operation-state.json initialization (task spec Testing Strategy).
	dir := newRunDir(t)
	if _, err := CreateResult(dir, []byte(`{"a":1}`)); err != nil {
		t.Fatalf("CreateResult: %v", err)
	}
	decision, err := DecideResume(dir)
	if err != nil {
		t.Fatalf("DecideResume: %v", err)
	}
	if decision.Action != ResumeActionResumeNextOperation {
		t.Errorf("action = %s, want resume_next_operation", decision.Action)
	}
	if decision.State != nil {
		t.Errorf("state = %+v, want nil (no operation-state.json yet)", decision.State)
	}
}

func TestDecideResume_PendingState_ResumesNextOperation(t *testing.T) {
	dir := newRunDir(t)
	if _, err := CreateResult(dir, []byte(`{"a":1}`)); err != nil {
		t.Fatalf("CreateResult: %v", err)
	}
	s := NewOperationState("run-1", "E01-F01-001", "task", "in_review", "code_review", "digest")
	s.AddCompletedSuboperation("sub-1")
	if err := s.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}

	decision, err := DecideResume(dir)
	if err != nil {
		t.Fatalf("DecideResume: %v", err)
	}
	if decision.Action != ResumeActionResumeNextOperation {
		t.Errorf("action = %s, want resume_next_operation", decision.Action)
	}
	if !decision.State.HasCompleted("sub-1") {
		t.Error("resumed state lost already-completed suboperation sub-1")
	}
}

func TestDecideResume_PersistenceComplete_ResumesTransition(t *testing.T) {
	dir := newRunDir(t)
	if _, err := CreateResult(dir, []byte(`{"a":1}`)); err != nil {
		t.Fatalf("CreateResult: %v", err)
	}
	s := NewOperationState("run-1", "E01-F01-001", "task", "in_review", "code_review", "digest")
	if err := s.MarkPersistenceComplete(); err != nil {
		t.Fatalf("mark: %v", err)
	}
	if err := s.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}

	decision, err := DecideResume(dir)
	if err != nil {
		t.Fatalf("DecideResume: %v", err)
	}
	if decision.Action != ResumeActionResumeTransition {
		t.Errorf("action = %s, want resume_transition", decision.Action)
	}
}

func TestDecideResume_TransitionApplied_AlreadyTransitioned(t *testing.T) {
	dir := newRunDir(t)
	if _, err := CreateResult(dir, []byte(`{"a":1}`)); err != nil {
		t.Fatalf("CreateResult: %v", err)
	}
	s := NewOperationState("run-1", "E01-F01-001", "task", "in_review", "code_review", "digest")
	if err := s.MarkPersistenceComplete(); err != nil {
		t.Fatalf("mark persistence: %v", err)
	}
	if err := s.MarkTransitionApplied(); err != nil {
		t.Fatalf("mark transition: %v", err)
	}
	if err := s.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}

	decision, err := DecideResume(dir)
	if err != nil {
		t.Fatalf("DecideResume: %v", err)
	}
	if decision.Action != ResumeActionAlreadyTransitioned {
		t.Errorf("action = %s, want already_transitioned", decision.Action)
	}
}

func TestVerifyResumeIdentity(t *testing.T) {
	s := NewOperationState("run-1", "E01-F01-001", "task", "in_review", "code_review", "digest-x")

	if err := VerifyResumeIdentity(s, "E01-F01-001", "task", "in_review", "digest-x"); err != nil {
		t.Errorf("matching identity: want nil, got %v", err)
	}

	mismatches := []struct {
		name                                        string
		entityKey, entityType, sourceStatus, digest string
	}{
		{"entity_key", "E01-F01-002", "task", "in_review", "digest-x"},
		{"entity_type", "E01-F01-001", "feature", "in_review", "digest-x"},
		{"source_status", "E01-F01-001", "task", "in_qa", "digest-x"},
		{"digest", "E01-F01-001", "task", "in_review", "digest-y"},
	}
	for _, m := range mismatches {
		if err := VerifyResumeIdentity(s, m.entityKey, m.entityType, m.sourceStatus, m.digest); err == nil {
			t.Errorf("%s mismatch: want error, got nil", m.name)
		}
	}
}

func TestReconcileCompletedSuboperations_MergesDurableRecords(t *testing.T) {
	s := NewOperationState("run-1", "E01-F01-001", "task", "in_review", "code_review", "digest")
	s.AddCompletedSuboperation("sub-1")

	reader := stubTargetRecordReader{ids: []string{"sub-1", "sub-2", "sub-3"}}
	changed, err := ReconcileCompletedSuboperations(context.Background(), reader, s)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if !changed {
		t.Error("changed = false, want true (sub-2/sub-3 were missing)")
	}
	for _, id := range []string{"sub-1", "sub-2", "sub-3"} {
		if !s.HasCompleted(id) {
			t.Errorf("state missing %s after reconciliation", id)
		}
	}
}

func TestReconcileCompletedSuboperations_NoOpWhenAlreadyComplete(t *testing.T) {
	s := NewOperationState("run-1", "E01-F01-001", "task", "in_review", "code_review", "digest")
	s.AddCompletedSuboperation("sub-1")

	reader := stubTargetRecordReader{ids: []string{"sub-1"}}
	changed, err := ReconcileCompletedSuboperations(context.Background(), reader, s)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if changed {
		t.Error("changed = true, want false when durable records add nothing new")
	}
}

func TestReconcileCompletedSuboperations_PropagatesReaderError(t *testing.T) {
	s := NewOperationState("run-1", "E01-F01-001", "task", "in_review", "code_review", "digest")
	reader := stubTargetRecordReader{err: errors.New("boom")}
	if _, err := ReconcileCompletedSuboperations(context.Background(), reader, s); err == nil {
		t.Fatal("reconcile with failing reader: want error, got nil")
	}
}

func TestReconcileCompletedSuboperations_RejectsNilArgs(t *testing.T) {
	s := NewOperationState("run-1", "E01-F01-001", "task", "in_review", "code_review", "digest")
	if _, err := ReconcileCompletedSuboperations(context.Background(), nil, s); err == nil {
		t.Fatal("nil reader: want error, got nil")
	}
	reader := stubTargetRecordReader{}
	if _, err := ReconcileCompletedSuboperations(context.Background(), reader, nil); err == nil {
		t.Fatal("nil state: want error, got nil")
	}
}
