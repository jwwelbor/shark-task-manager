package workflow

import "testing"

// TestGetWorkflowForLevel_BugWithNil verifies the default bug workflow's new
// route-based shape (bug.yaml): draft/development/code_review/qa/completed/
// blocked/cancelled/on_hold, with "reported" surviving only as a backward-
// compat alias (draft.aliases: [reported]) resolved via ResolveAlias -- it is
// NOT a key in StatusFlow/StatusMetadata anymore.
func TestGetWorkflowForLevel_BugWithNil(t *testing.T) {
	m := &MultiLevelWorkflow{}
	wf := m.GetWorkflowForLevel("bug")
	if wf == nil {
		t.Fatal("expected non-nil workflow for bug level with nil Bug")
	}
	// Should return default bug workflow with the research gate.
	if len(wf.StatusFlow) != 11 {
		t.Errorf("expected 11 statuses in default bug workflow, got %d", len(wf.StatusFlow))
	}
	if _, ok := wf.StatusFlow["draft"]; !ok {
		t.Error("expected 'draft' status in default bug workflow")
	}
	if _, ok := wf.StatusFlow["development"]; !ok {
		t.Error("expected 'development' status in default bug workflow")
	}
	if _, ok := wf.StatusFlow["code_review"]; !ok {
		t.Error("expected 'code_review' status in default bug workflow")
	}
	if _, ok := wf.StatusFlow["completed"]; !ok {
		t.Error("expected 'completed' status in default bug workflow")
	}
	// Old status name "reported" resolves via alias to "draft".
	if got := wf.ResolveAlias("reported"); got != "draft" {
		t.Errorf("expected ResolveAlias(\"reported\") = \"draft\", got %q", got)
	}
}

// TestGetWorkflowForLevel_ChangeWithNil mirrors the bug test for change.yaml,
// which has the same route-based shape ("proposed"/"declined" survive only
// as aliases on draft/cancelled).
func TestGetWorkflowForLevel_ChangeWithNil(t *testing.T) {
	m := &MultiLevelWorkflow{}
	wf := m.GetWorkflowForLevel("change")
	if wf == nil {
		t.Fatal("expected non-nil workflow for change level with nil Change")
	}
	// Should return default change-card workflow with the research gate.
	if len(wf.StatusFlow) != 9 {
		t.Errorf("expected 9 statuses in default change-card workflow, got %d", len(wf.StatusFlow))
	}
	if _, ok := wf.StatusFlow["draft"]; !ok {
		t.Error("expected 'draft' status in default change-card workflow")
	}
	if _, ok := wf.StatusFlow["code_review"]; !ok {
		t.Error("expected 'code_review' status in default change-card workflow")
	}
	if _, ok := wf.StatusFlow["completed"]; !ok {
		t.Error("expected 'completed' status in default change-card workflow")
	}
	// Old status name "proposed" resolves via alias to "draft".
	if got := wf.ResolveAlias("proposed"); got != "draft" {
		t.Errorf("expected ResolveAlias(\"proposed\") = \"draft\", got %q", got)
	}
}

func TestGetWorkflowForLevel_CustomBug(t *testing.T) {
	customBug := &WorkflowConfig{
		Version: "1.0",
		StatusFlow: map[string][]string{
			"new":           {"investigating"},
			"investigating": {"fixed"},
			"fixed":         {},
		},
	}
	m := &MultiLevelWorkflow{Bug: customBug}
	wf := m.GetWorkflowForLevel("bug")
	if wf != customBug {
		t.Error("expected custom bug workflow to be returned")
	}
	if _, ok := wf.StatusFlow["investigating"]; !ok {
		t.Error("expected custom status 'investigating' in bug workflow")
	}
}

func TestGetWorkflowForLevel_CustomChange(t *testing.T) {
	customChange := &WorkflowConfig{
		Version: "1.0",
		StatusFlow: map[string][]string{
			"draft":     {"submitted"},
			"submitted": {"done"},
			"done":      {},
		},
	}
	m := &MultiLevelWorkflow{Change: customChange}
	wf := m.GetWorkflowForLevel("change")
	if wf != customChange {
		t.Error("expected custom change workflow to be returned")
	}
	if _, ok := wf.StatusFlow["submitted"]; !ok {
		t.Error("expected custom status 'submitted' in change workflow")
	}
}

func TestDefaultBugWorkflow_PassesValidation(t *testing.T) {
	wf := DefaultBugWorkflow()
	if err := ValidateWorkflow(wf); err != nil {
		t.Errorf("default bug workflow should pass validation: %v", err)
	}
}

func TestDefaultChangeCardWorkflow_PassesValidation(t *testing.T) {
	wf := DefaultChangeCardWorkflow()
	if err := ValidateWorkflow(wf); err != nil {
		t.Errorf("default change-card workflow should pass validation: %v", err)
	}
}

// TestDefaultBugWorkflow_HasCorrectMetadata checks the start status resolves
// (via alias) to the correct step and that step's metadata is right. The old
// hardcoded "reported" status is now an alias on "draft" (bug.yaml), so
// _start_/StatusMetadata are keyed by "draft", not "reported".
func TestDefaultBugWorkflow_HasCorrectMetadata(t *testing.T) {
	wf := DefaultBugWorkflow()

	// Check start status
	startStatuses, ok := wf.SpecialStatuses[StartStatusKey]
	if !ok {
		t.Fatal("expected _start_ in special statuses")
	}
	if len(startStatuses) != 1 || startStatuses[0] != "draft" {
		t.Errorf("expected _start_ = ['draft'], got %v", startStatuses)
	}

	// Check complete statuses (terminal steps: cancelled, completed)
	completeStatuses, ok := wf.SpecialStatuses[CompleteStatusKey]
	if !ok {
		t.Fatal("expected _complete_ in special statuses")
	}
	if len(completeStatuses) != 2 {
		t.Errorf("expected 2 complete statuses, got %d", len(completeStatuses))
	}

	// "reported" is a backward-compat alias for "draft", not a StatusMetadata key.
	if got := wf.ResolveAlias("reported"); got != "draft" {
		t.Errorf("expected ResolveAlias(\"reported\") = \"draft\", got %q", got)
	}

	// Check draft (the actual start step) metadata
	draftMeta, ok := wf.StatusMetadata["draft"]
	if !ok {
		t.Fatal("expected 'draft' in status metadata")
	}
	if draftMeta.Color != "gray" {
		t.Errorf("expected draft color 'gray', got %q", draftMeta.Color)
	}
	if draftMeta.Phase != "planning" {
		t.Errorf("expected draft phase 'planning', got %q", draftMeta.Phase)
	}
}

// TestDefaultChangeCardWorkflow_HasCorrectMetadata mirrors the bug test:
// "proposed" is now an alias for "draft" (change.yaml), not a StatusMetadata key.
func TestDefaultChangeCardWorkflow_HasCorrectMetadata(t *testing.T) {
	wf := DefaultChangeCardWorkflow()

	// Check start status
	startStatuses, ok := wf.SpecialStatuses[StartStatusKey]
	if !ok {
		t.Fatal("expected _start_ in special statuses")
	}
	if len(startStatuses) != 1 || startStatuses[0] != "draft" {
		t.Errorf("expected _start_ = ['draft'], got %v", startStatuses)
	}

	// Check complete statuses (terminal steps: cancelled, completed)
	completeStatuses, ok := wf.SpecialStatuses[CompleteStatusKey]
	if !ok {
		t.Fatal("expected _complete_ in special statuses")
	}
	if len(completeStatuses) != 2 {
		t.Errorf("expected 2 complete statuses, got %d", len(completeStatuses))
	}

	// "proposed" is a backward-compat alias for "draft", not a StatusMetadata key.
	if got := wf.ResolveAlias("proposed"); got != "draft" {
		t.Errorf("expected ResolveAlias(\"proposed\") = \"draft\", got %q", got)
	}

	// Check draft (the actual start step) metadata
	draftMeta, ok := wf.StatusMetadata["draft"]
	if !ok {
		t.Fatal("expected 'draft' in status metadata")
	}
	if draftMeta.Color != "gray" {
		t.Errorf("expected draft color 'gray', got %q", draftMeta.Color)
	}
}

func TestGetWorkflowForLevel_BugChangeIsolation(t *testing.T) {
	m := &MultiLevelWorkflow{}

	bugWf := m.GetWorkflowForLevel("bug")
	changeWf := m.GetWorkflowForLevel("change")
	taskWf := m.GetWorkflowForLevel("task")

	// Bug workflow includes research and repair planning gates
	if len(bugWf.StatusFlow) != 11 {
		t.Errorf("expected 11 statuses in default bug workflow, got %d", len(bugWf.StatusFlow))
	}

	// Change workflow should have 8 statuses
	if len(changeWf.StatusFlow) != 9 {
		t.Errorf("expected 9 statuses in default change workflow, got %d", len(changeWf.StatusFlow))
	}

	// Task workflow should still have 6 statuses.
	if len(taskWf.StatusFlow) != 6 {
		t.Errorf("expected 6 statuses in default task workflow, got %d", len(taskWf.StatusFlow))
	}

	// Verify no cross-contamination (both aliases still resolve on their own workflow only)
	if _, ok := bugWf.StatusFlow["proposed"]; ok {
		t.Error("bug workflow should not contain 'proposed' from change workflow")
	}
	if _, ok := changeWf.StatusFlow["reported"]; ok {
		t.Error("change workflow should not contain 'reported' from bug workflow")
	}
	if got := bugWf.ResolveAlias("proposed"); got != "proposed" {
		t.Errorf("bug workflow should not resolve 'proposed' (change-only alias), got %q", got)
	}
	if got := changeWf.ResolveAlias("reported"); got != "reported" {
		t.Errorf("change workflow should not resolve 'reported' (bug-only alias), got %q", got)
	}
}
