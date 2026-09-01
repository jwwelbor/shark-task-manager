package gaterun

import "testing"

func TestOperationState_AddCompletedSuboperation_Dedupes(t *testing.T) {
	s := NewOperationState("run-1", "E01-F01-001", "task", "in_review", "code_review", "digest")
	if !s.AddCompletedSuboperation("sub-1") {
		t.Fatal("first add: want true (newly added)")
	}
	if s.AddCompletedSuboperation("sub-1") {
		t.Fatal("second add of same id: want false (already present)")
	}
	if !s.HasCompleted("sub-1") {
		t.Error("HasCompleted(sub-1) = false, want true")
	}
	if s.HasCompleted("sub-2") {
		t.Error("HasCompleted(sub-2) = true, want false")
	}
	if len(s.CompletedSuboperationIDs) != 1 {
		t.Errorf("CompletedSuboperationIDs = %v, want exactly one entry", s.CompletedSuboperationIDs)
	}
}

func TestOperationState_PersistenceTransitionSequence(t *testing.T) {
	s := NewOperationState("run-1", "E01-F01-001", "task", "in_review", "code_review", "digest")
	if s.PersistenceState != PersistenceStatePending {
		t.Fatalf("initial state = %s, want pending", s.PersistenceState)
	}

	if err := s.MarkTransitionApplied(); err == nil {
		t.Fatal("MarkTransitionApplied before persistence complete: want error, got nil")
	}

	if err := s.MarkPersistenceComplete(); err != nil {
		t.Fatalf("MarkPersistenceComplete: %v", err)
	}
	if s.PersistenceState != PersistenceStateComplete {
		t.Fatalf("state = %s, want persistence_complete", s.PersistenceState)
	}
	// Idempotent re-call.
	if err := s.MarkPersistenceComplete(); err != nil {
		t.Fatalf("repeat MarkPersistenceComplete: %v", err)
	}

	if err := s.MarkTransitionApplied(); err != nil {
		t.Fatalf("MarkTransitionApplied: %v", err)
	}
	if s.PersistenceState != PersistenceStateTransitioned {
		t.Fatalf("state = %s, want transition_applied", s.PersistenceState)
	}
	// Idempotent re-call.
	if err := s.MarkTransitionApplied(); err != nil {
		t.Fatalf("repeat MarkTransitionApplied: %v", err)
	}
}

func TestOperationState_RoundTripSaveLoad(t *testing.T) {
	dir := newRunDir(t)
	s := NewOperationState("run-1", "E01-F01-001", "task", "in_review", "code_review", "digest")
	s.AddCompletedSuboperation("sub-1")
	if err := s.MarkPersistenceComplete(); err != nil {
		t.Fatalf("mark: %v", err)
	}
	if err := s.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, exists, err := LoadOperationState(dir)
	if err != nil || !exists {
		t.Fatalf("load: exists=%v err=%v", exists, err)
	}
	if loaded.PersistenceState != PersistenceStateComplete {
		t.Errorf("loaded state = %s, want persistence_complete", loaded.PersistenceState)
	}
	if !loaded.HasCompleted("sub-1") {
		t.Error("loaded state missing sub-1")
	}
	if loaded.RunID != "run-1" || loaded.EntityKey != "E01-F01-001" {
		t.Errorf("loaded identity mismatch: %+v", loaded)
	}
}

func TestLoadOperationState_NotExists(t *testing.T) {
	dir := newRunDir(t)
	_, exists, err := LoadOperationState(dir)
	if err != nil {
		t.Fatalf("load on empty dir: %v", err)
	}
	if exists {
		t.Error("exists = true, want false for a fresh run dir")
	}
}
