package gaterun

import (
	"path/filepath"
	"testing"
	"time"
)

func TestProjectStatus(t *testing.T) {
	s := NewOperationState("run-1", "E01-F01-001", "task", "in_review", "code_review", "digest")
	s.WorkerPhase = "dispatch"
	s.NestedOperation = "code_review:findings"
	s.RetirementState = RetirementPending
	s.ResultLocation = "/tmp/run-1/result.json"
	s.AddCompletedSuboperation("sub-1")
	s.AddCompletedSuboperation("sub-2")

	now := s.StartedAt.Add(30 * time.Second)
	proj := ProjectStatus(s, now)

	if proj.RunID != "run-1" || proj.EntityKey != "E01-F01-001" {
		t.Errorf("identity mismatch: %+v", proj)
	}
	if proj.WorkerPhase != "dispatch" {
		t.Errorf("WorkerPhase = %q, want dispatch", proj.WorkerPhase)
	}
	if proj.NestedOperation != "code_review:findings" {
		t.Errorf("NestedOperation = %q", proj.NestedOperation)
	}
	if proj.RetirementState != RetirementPending {
		t.Errorf("RetirementState = %q, want pending", proj.RetirementState)
	}
	if proj.ElapsedSeconds < 29.5 || proj.ElapsedSeconds > 30.5 {
		t.Errorf("ElapsedSeconds = %v, want ~30", proj.ElapsedSeconds)
	}
	if proj.ResultLocation != "/tmp/run-1/result.json" {
		t.Errorf("ResultLocation = %q", proj.ResultLocation)
	}
	if proj.CompletedCount != 2 {
		t.Errorf("CompletedCount = %d, want 2", proj.CompletedCount)
	}
}

// TestProjectStatus_FieldsPopulatedByRealSequence is the population
// counterpart of TestProjectStatus above: that test only proves ProjectStatus
// *copies* WorkerPhase/NestedOperation/ResultLocation when they are set by
// hand, which cannot fail if nothing ever populates them on a real run — the
// exact gap UAT's red-team found (these three fields were declared and
// projected but never populated by any ingest path). This test drives the
// actual sequence a caller performs (construct, apply a suboperation, mark
// persistence complete, mark the transition applied, save, reload from
// disk) and asserts all three land non-empty in the reloaded state's
// projection, with no test code assigning them directly.
func TestProjectStatus_FieldsPopulatedByRealSequence(t *testing.T) {
	dir := newRunDir(t)
	s := NewOperationState("run-1", "E01-F01-001", "task", "in_review", "code_review", "digest")

	if s.WorkerPhase == "" {
		t.Error("WorkerPhase is empty immediately after NewOperationState, want a non-empty default phase")
	}

	s.AddCompletedSuboperation("sub-1")
	if s.NestedOperation != "sub-1" {
		t.Errorf("NestedOperation after AddCompletedSuboperation = %q, want %q", s.NestedOperation, "sub-1")
	}

	if err := s.Save(dir); err != nil {
		t.Fatalf("save after first suboperation: %v", err)
	}
	if err := s.MarkPersistenceComplete(); err != nil {
		t.Fatalf("MarkPersistenceComplete: %v", err)
	}
	if err := s.Save(dir); err != nil {
		t.Fatalf("save after persistence complete: %v", err)
	}
	if err := s.MarkTransitionApplied(); err != nil {
		t.Fatalf("MarkTransitionApplied: %v", err)
	}
	if err := s.Save(dir); err != nil {
		t.Fatalf("save after transition applied: %v", err)
	}

	reloaded, exists, err := LoadOperationState(dir)
	if err != nil || !exists {
		t.Fatalf("LoadOperationState: exists=%v err=%v", exists, err)
	}

	proj := ProjectStatus(reloaded, time.Now())
	if proj.WorkerPhase == "" {
		t.Error("reloaded WorkerPhase is empty, want it populated by the real ingest sequence")
	}
	if proj.NestedOperation == "" {
		t.Error("reloaded NestedOperation is empty, want it populated by the real ingest sequence")
	}
	wantLocation := filepath.Join(dir, resultFileName)
	if proj.ResultLocation != wantLocation {
		t.Errorf("reloaded ResultLocation = %q, want %q", proj.ResultLocation, wantLocation)
	}
}
