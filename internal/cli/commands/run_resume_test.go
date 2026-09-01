package commands

import (
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
)

func TestResolveResumeStatus_NoDurableResult(t *testing.T) {
	root := t.TempDir()
	_, err := resolveResumeStatus(root, "run-1", "task", "E01-F01-001", time.Now())
	if err == nil {
		t.Fatal("resolveResumeStatus with no result.json: want error, got nil")
	}
}

func TestResolveResumeStatus_ResultWithoutStateReportsResumeNextOperation(t *testing.T) {
	root := t.TempDir()
	dir, err := gaterun.RunDir(root, "run-1")
	if err != nil {
		t.Fatalf("RunDir: %v", err)
	}
	if _, err := gaterun.CreateResult(dir, []byte(`{"a":1}`)); err != nil {
		t.Fatalf("CreateResult: %v", err)
	}

	out, err := resolveResumeStatus(root, "run-1", "task", "E01-F01-001", time.Now())
	if err != nil {
		t.Fatalf("resolveResumeStatus: %v", err)
	}
	if out.Action != gaterun.ResumeActionResumeNextOperation {
		t.Errorf("Action = %s, want resume_next_operation", out.Action)
	}
	if out.Status != nil {
		t.Errorf("Status = %+v, want nil (no operation-state.json yet)", out.Status)
	}
	if out.RunID != "run-1" || out.EntityKey != "E01-F01-001" || out.EntityType != "task" {
		t.Errorf("identity echo mismatch: %+v", out)
	}
}

func TestResolveResumeStatus_MatchingIdentityNoWarning(t *testing.T) {
	root := t.TempDir()
	dir, err := gaterun.RunDir(root, "run-1")
	if err != nil {
		t.Fatalf("RunDir: %v", err)
	}
	if _, err := gaterun.CreateResult(dir, []byte(`{"a":1}`)); err != nil {
		t.Fatalf("CreateResult: %v", err)
	}
	s := gaterun.NewOperationState("run-1", "E01-F01-001", "task", "in_review", "code_review", "digest")
	if err := s.MarkPersistenceComplete(); err != nil {
		t.Fatalf("mark: %v", err)
	}
	if err := s.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}

	out, err := resolveResumeStatus(root, "run-1", "task", "E01-F01-001", time.Now())
	if err != nil {
		t.Fatalf("resolveResumeStatus: %v", err)
	}
	if out.Action != gaterun.ResumeActionResumeTransition {
		t.Errorf("Action = %s, want resume_transition", out.Action)
	}
	if out.Status == nil {
		t.Fatal("Status = nil, want a projection once operation-state.json exists")
	}
}

func TestResolveResumeStatus_MismatchedIdentityFailsClosed(t *testing.T) {
	root := t.TempDir()
	dir, err := gaterun.RunDir(root, "run-1")
	if err != nil {
		t.Fatalf("RunDir: %v", err)
	}
	if _, err := gaterun.CreateResult(dir, []byte(`{"a":1}`)); err != nil {
		t.Fatalf("CreateResult: %v", err)
	}
	s := gaterun.NewOperationState("run-1", "E01-F01-999", "task", "in_review", "code_review", "digest")
	if err := s.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}

	if _, err := resolveResumeStatus(root, "run-1", "task", "E01-F01-001", time.Now()); err == nil {
		t.Fatal("resolveResumeStatus with mismatched recorded entity_key: want error, got nil")
	}
}

func TestResolveResumeStatus_AlreadyTransitioned(t *testing.T) {
	root := t.TempDir()
	dir, err := gaterun.RunDir(root, "run-1")
	if err != nil {
		t.Fatalf("RunDir: %v", err)
	}
	if _, err := gaterun.CreateResult(dir, []byte(`{"a":1}`)); err != nil {
		t.Fatalf("CreateResult: %v", err)
	}
	s := gaterun.NewOperationState("run-1", "E01-F01-001", "task", "in_review", "code_review", "digest")
	if err := s.MarkPersistenceComplete(); err != nil {
		t.Fatalf("mark persistence: %v", err)
	}
	if err := s.MarkTransitionApplied(); err != nil {
		t.Fatalf("mark transition: %v", err)
	}
	if err := s.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}

	out, err := resolveResumeStatus(root, "run-1", "task", "E01-F01-001", time.Now())
	if err != nil {
		t.Fatalf("resolveResumeStatus: %v", err)
	}
	if out.Action != gaterun.ResumeActionAlreadyTransitioned {
		t.Errorf("Action = %s, want already_transitioned", out.Action)
	}
}

func TestRunResumeRun_RequiresSession(t *testing.T) {
	origResumeID, origSession := runResumeID, runSession
	defer func() { runResumeID, runSession = origResumeID, origSession }()

	runResumeID = "run-1"
	runSession = ""
	if err := runResumeRun("task", "E01-F01-001"); err == nil {
		t.Fatal("runResumeRun without --session: want error, got nil")
	}
}
