package gaterun

import (
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
