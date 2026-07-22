package workflow

import (
	"encoding/json"
	"testing"
)

func TestGetWorkflowForLevel_EpicWithNil(t *testing.T) {
	m := &MultiLevelWorkflow{}
	wf := m.GetWorkflowForLevel("epic")
	if wf == nil {
		t.Fatal("expected non-nil workflow for epic level with nil Epic")
	}
	// Should return default epic workflow (route-based epic.yaml has 12 steps,
	// including the BUG-5 intake "assessment" triage step)
	if len(wf.StatusFlow) != 12 {
		t.Errorf("expected 12 statuses in default epic workflow, got %d", len(wf.StatusFlow))
	}
	if _, ok := wf.StatusFlow["draft"]; !ok {
		t.Error("expected 'draft' status in default epic workflow")
	}
	if _, ok := wf.StatusFlow["active"]; !ok {
		t.Error("expected 'active' status in default epic workflow")
	}
}

func TestGetWorkflowForLevel_FeatureWithNil(t *testing.T) {
	m := &MultiLevelWorkflow{}
	wf := m.GetWorkflowForLevel("feature")
	if wf == nil {
		t.Fatal("expected non-nil workflow for feature level with nil Feature")
	}
	// route-based feature.yaml has 15 steps
	if len(wf.StatusFlow) != 15 {
		t.Errorf("expected 15 statuses in default feature workflow, got %d", len(wf.StatusFlow))
	}
}

func TestGetWorkflowForLevel_TaskWithNil(t *testing.T) {
	m := &MultiLevelWorkflow{}
	wf := m.GetWorkflowForLevel("task")
	if wf == nil {
		t.Fatal("expected non-nil workflow for task level with nil Task")
	}
	// Default task workflow relies on feature-level research.
	if len(wf.StatusFlow) != 6 {
		t.Errorf("expected 6 statuses in default task workflow, got %d", len(wf.StatusFlow))
	}
	if _, ok := wf.StatusFlow["draft"]; !ok {
		t.Error("expected 'draft' status in default task workflow")
	}
}

func TestGetWorkflowForLevel_CustomEpic(t *testing.T) {
	customEpic := &WorkflowConfig{
		Version: "1.0",
		StatusFlow: map[string][]string{
			"draft":              {"ready_for_research"},
			"ready_for_research": {"active"},
			"active":             {"completed"},
			"completed":          {},
		},
	}
	m := &MultiLevelWorkflow{Epic: customEpic}
	wf := m.GetWorkflowForLevel("epic")
	if wf != customEpic {
		t.Error("expected custom epic workflow to be returned")
	}
	if _, ok := wf.StatusFlow["ready_for_research"]; !ok {
		t.Error("expected custom status 'ready_for_research' in epic workflow")
	}
}

func TestGetWorkflowForLevel_UnknownLevel(t *testing.T) {
	m := &MultiLevelWorkflow{}
	wf := m.GetWorkflowForLevel("unknown")
	if wf == nil {
		t.Fatal("expected non-nil workflow for unknown level")
	}
	// Should fall back to default task workflow
	if _, ok := wf.StatusFlow["draft"]; !ok {
		t.Error("expected default task workflow for unknown level")
	}
}

func TestGetWorkflowForLevel_Isolation(t *testing.T) {
	customEpic := &WorkflowConfig{
		StatusFlow: map[string][]string{
			"draft":  {"active"},
			"active": {},
		},
	}
	m := &MultiLevelWorkflow{Epic: customEpic}

	epicWf := m.GetWorkflowForLevel("epic")
	taskWf := m.GetWorkflowForLevel("task")

	// Epic workflow should have custom statuses
	if len(epicWf.StatusFlow) != 2 {
		t.Errorf("expected 2 statuses in custom epic workflow, got %d", len(epicWf.StatusFlow))
	}

	// Task workflow relies on feature-level research.
	if len(taskWf.StatusFlow) != 6 {
		t.Errorf("expected 6 statuses in default task workflow, got %d", len(taskWf.StatusFlow))
	}
}

func TestDefaultEpicWorkflow_PassesValidation(t *testing.T) {
	wf := DefaultEpicWorkflow()
	if err := ValidateWorkflow(wf); err != nil {
		t.Errorf("default epic workflow should pass validation: %v", err)
	}
}

func TestDefaultEpicWorkflow_ResearchPrecedesDesign(t *testing.T) {
	wf := DefaultEpicWorkflow()
	if got := wf.StatusFlow["research"][0]; got != "design" {
		t.Fatalf("research pass route = %q, want design", got)
	}
	if got := wf.StatusFlow["design"][0]; got != "decomposition" {
		t.Fatalf("design pass route = %q, want decomposition", got)
	}
}

func TestDefaultFeatureWorkflow_PassesValidation(t *testing.T) {
	wf := DefaultFeatureWorkflow()
	if err := ValidateWorkflow(wf); err != nil {
		t.Errorf("default feature workflow should pass validation: %v", err)
	}
}

func TestDefaultEpicWorkflow_HasCorrectMetadata(t *testing.T) {
	wf := DefaultEpicWorkflow()

	// Draft should be a planning status
	draftMeta, ok := wf.StatusMetadata["draft"]
	if !ok {
		t.Fatal("expected 'draft' in status metadata")
	}
	if !draftMeta.IsPlanning {
		t.Error("expected draft to have IsPlanning=true")
	}

	// Active should aggregate from features
	activeMeta, ok := wf.StatusMetadata["active"]
	if !ok {
		t.Fatal("expected 'active' in status metadata")
	}
	if activeMeta.AggregatesFrom != "features" {
		t.Errorf("expected active to aggregate from 'features', got %q", activeMeta.AggregatesFrom)
	}
	if activeMeta.IsPlanning {
		t.Error("expected active to have IsPlanning=false")
	}

	// Check special statuses
	aggStatuses, ok := wf.SpecialStatuses[AggregationStatusKey]
	if !ok {
		t.Fatal("expected _aggregation_ in special statuses")
	}
	if len(aggStatuses) != 1 || aggStatuses[0] != "active" {
		t.Errorf("expected _aggregation_ = ['active'], got %v", aggStatuses)
	}
}

func TestDefaultFeatureWorkflow_HasCorrectMetadata(t *testing.T) {
	wf := DefaultFeatureWorkflow()

	// Active should aggregate from tasks
	activeMeta, ok := wf.StatusMetadata["active"]
	if !ok {
		t.Fatal("expected 'active' in status metadata")
	}
	if activeMeta.AggregatesFrom != "tasks" {
		t.Errorf("expected active to aggregate from 'tasks', got %q", activeMeta.AggregatesFrom)
	}
}

func TestAggregationStatusKeyConstant(t *testing.T) {
	if AggregationStatusKey != "_aggregation_" {
		t.Errorf("expected AggregationStatusKey to be '_aggregation_', got %q", AggregationStatusKey)
	}
}

func TestStatusMetadata_IsPlanningJSONParsing(t *testing.T) {
	// Test that IsPlanning and AggregatesFrom parse correctly from JSON
	jsonStr := `{
		"color": "gray",
		"description": "test",
		"phase": "planning",
		"is_planning": true,
		"aggregates_from": "features"
	}`

	var meta StatusMetadata
	if err := json.Unmarshal([]byte(jsonStr), &meta); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if !meta.IsPlanning {
		t.Error("expected IsPlanning=true after JSON unmarshal")
	}
	if meta.AggregatesFrom != "features" {
		t.Errorf("expected AggregatesFrom='features', got %q", meta.AggregatesFrom)
	}
}

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
	if len(wf.StatusFlow) != 9 {
		t.Errorf("expected 9 statuses in default bug workflow, got %d", len(wf.StatusFlow))
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

	// Bug workflow should have 8 statuses
	if len(bugWf.StatusFlow) != 9 {
		t.Errorf("expected 9 statuses in default bug workflow, got %d", len(bugWf.StatusFlow))
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

func TestGetWorkflowForLevel_TechDebtWithNil(t *testing.T) {
	m := &MultiLevelWorkflow{}
	wf := m.GetWorkflowForLevel("tech_debt")
	if wf == nil {
		t.Fatal("expected non-nil workflow for tech_debt level with nil TechDebt")
	}
	// Should return default tech-debt workflow with the research gate.
	if len(wf.StatusFlow) != 7 {
		t.Errorf("expected 7 statuses in default tech-debt workflow, got %d", len(wf.StatusFlow))
	}
	if _, ok := wf.StatusFlow["identified"]; !ok {
		t.Error("expected 'identified' status in default tech-debt workflow")
	}
	if _, ok := wf.StatusFlow["triaged"]; !ok {
		t.Error("expected 'triaged' status in default tech-debt workflow")
	}
	if _, ok := wf.StatusFlow["in_progress"]; !ok {
		t.Error("expected 'in_progress' status in default tech-debt workflow")
	}
	if _, ok := wf.StatusFlow["resolved"]; !ok {
		t.Error("expected 'resolved' status in default tech-debt workflow")
	}
	if _, ok := wf.StatusFlow["wont_fix"]; !ok {
		t.Error("expected 'wont_fix' status in default tech-debt workflow")
	}
	if _, ok := wf.StatusFlow["cancelled"]; !ok {
		t.Error("expected 'cancelled' status in default tech-debt workflow")
	}
}

func TestGetWorkflowForLevel_CustomTechDebt(t *testing.T) {
	customTD := &WorkflowConfig{
		Version: "1.0",
		StatusFlow: map[string][]string{
			"new":           {"investigating"},
			"investigating": {"resolved"},
			"resolved":      {},
		},
	}
	m := &MultiLevelWorkflow{TechDebt: customTD}
	wf := m.GetWorkflowForLevel("tech_debt")
	if wf != customTD {
		t.Error("expected custom tech-debt workflow to be returned")
	}
	if _, ok := wf.StatusFlow["investigating"]; !ok {
		t.Error("expected custom status 'investigating' in tech-debt workflow")
	}
}

func TestDefaultTechDebtWorkflow_PassesValidation(t *testing.T) {
	wf := DefaultTechDebtWorkflow()
	if err := ValidateWorkflow(wf); err != nil {
		t.Errorf("default tech-debt workflow should pass validation: %v", err)
	}
}

func TestDefaultTechDebtWorkflow_HasCorrectMetadata(t *testing.T) {
	wf := DefaultTechDebtWorkflow()

	// Check start status
	startStatuses, ok := wf.SpecialStatuses[StartStatusKey]
	if !ok {
		t.Fatal("expected _start_ in special statuses")
	}
	if len(startStatuses) != 1 || startStatuses[0] != "identified" {
		t.Errorf("expected _start_ = ['identified'], got %v", startStatuses)
	}

	// Check complete statuses
	completeStatuses, ok := wf.SpecialStatuses[CompleteStatusKey]
	if !ok {
		t.Fatal("expected _complete_ in special statuses")
	}
	if len(completeStatuses) != 3 {
		t.Errorf("expected 3 complete statuses, got %d", len(completeStatuses))
	}

	// Check identified metadata
	identifiedMeta, ok := wf.StatusMetadata["identified"]
	if !ok {
		t.Fatal("expected 'identified' in status metadata")
	}
	if identifiedMeta.Color != "gray" {
		t.Errorf("expected identified color 'gray', got %q", identifiedMeta.Color)
	}
	if identifiedMeta.Phase != "planning" {
		t.Errorf("expected identified phase 'planning', got %q", identifiedMeta.Phase)
	}
	if identifiedMeta.ProgressWeight != 0.0 {
		t.Errorf("expected identified progress weight 0.0, got %f", identifiedMeta.ProgressWeight)
	}

	// Check resolved metadata
	resolvedMeta, ok := wf.StatusMetadata["resolved"]
	if !ok {
		t.Fatal("expected 'resolved' in status metadata")
	}
	if resolvedMeta.Color != "green" {
		t.Errorf("expected resolved color 'green', got %q", resolvedMeta.Color)
	}
	if resolvedMeta.ProgressWeight != 1.0 {
		t.Errorf("expected resolved progress weight 1.0, got %f", resolvedMeta.ProgressWeight)
	}

	// Check wont_fix is terminal (empty transitions)
	if transitions, ok := wf.StatusFlow["wont_fix"]; ok {
		if len(transitions) != 0 {
			t.Errorf("expected wont_fix to be terminal (0 transitions), got %d", len(transitions))
		}
	} else {
		t.Error("expected 'wont_fix' in status flow")
	}

	// Check cancelled is terminal
	if transitions, ok := wf.StatusFlow["cancelled"]; ok {
		if len(transitions) != 0 {
			t.Errorf("expected cancelled to be terminal (0 transitions), got %d", len(transitions))
		}
	} else {
		t.Error("expected 'cancelled' in status flow")
	}
}

// TestDefaultTechDebtWorkflow_StatusFlow checks the transitions derived from
// tech-debt.yaml's outcomes maps. Every workable step routes "blocked" and
// "fail" back to "identified" (a self-loop for "identified" itself) in
// addition to "cancelled" and "wont_fix", so each has 4 unique targets, not
// the single/handful the old hardcoded workflow had. The forward ("pass")
// target is always first (uniqueSortedOutcomeTargets in steps.go orders by
// outcome-key priority, pass=0 first).
func TestDefaultTechDebtWorkflow_StatusFlow(t *testing.T) {
	wf := DefaultTechDebtWorkflow()

	// identified -(pass)-> research, plus identified/cancelled/wont_fix.
	identifiedTransitions := wf.StatusFlow["identified"]
	if len(identifiedTransitions) != 4 {
		t.Errorf("expected 4 identified transitions, got %d: %v", len(identifiedTransitions), identifiedTransitions)
	}
	if len(identifiedTransitions) > 0 && identifiedTransitions[0] != "research" {
		t.Errorf("expected identified's forward transition to be 'research', got %v", identifiedTransitions)
	}

	// triaged -(pass)-> in_progress, plus identified/cancelled/wont_fix
	triagedTransitions := wf.StatusFlow["triaged"]
	if len(triagedTransitions) != 4 {
		t.Errorf("expected 4 triaged transitions, got %d: %v", len(triagedTransitions), triagedTransitions)
	}
	if len(triagedTransitions) > 0 && triagedTransitions[0] != "in_progress" {
		t.Errorf("expected triaged's forward transition to be 'in_progress', got %v", triagedTransitions)
	}

	// in_progress -(pass)-> resolved, plus identified/cancelled/wont_fix
	inProgressTransitions := wf.StatusFlow["in_progress"]
	if len(inProgressTransitions) != 4 {
		t.Errorf("expected 4 in_progress transitions, got %d: %v", len(inProgressTransitions), inProgressTransitions)
	}
	if len(inProgressTransitions) > 0 && inProgressTransitions[0] != "resolved" {
		t.Errorf("expected in_progress's forward transition to be 'resolved', got %v", inProgressTransitions)
	}
}

func TestGetWorkflowForLevel_TechDebtIsolation(t *testing.T) {
	m := &MultiLevelWorkflow{}

	tdWf := m.GetWorkflowForLevel("tech_debt")
	bugWf := m.GetWorkflowForLevel("bug")
	taskWf := m.GetWorkflowForLevel("task")

	// Tech-debt workflow should include the research gate.
	if len(tdWf.StatusFlow) != 7 {
		t.Errorf("expected 7 statuses in default tech-debt workflow, got %d", len(tdWf.StatusFlow))
	}

	// Bug workflow should include the research gate.
	if len(bugWf.StatusFlow) != 9 {
		t.Errorf("expected 9 statuses in default bug workflow, got %d", len(bugWf.StatusFlow))
	}

	// Task workflow relies on feature-level research.
	if len(taskWf.StatusFlow) != 6 {
		t.Errorf("expected 6 statuses in default task workflow, got %d", len(taskWf.StatusFlow))
	}

	// Verify no cross-contamination
	if _, ok := tdWf.StatusFlow["reported"]; ok {
		t.Error("tech-debt workflow should not contain 'reported' from bug workflow")
	}
	if _, ok := bugWf.StatusFlow["identified"]; ok {
		t.Error("bug workflow should not contain 'identified' from tech-debt workflow")
	}
}

func TestStatusMetadata_IsPlanningOmittedDefaults(t *testing.T) {
	jsonStr := `{"color": "blue", "description": "test"}`

	var meta StatusMetadata
	if err := json.Unmarshal([]byte(jsonStr), &meta); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if meta.IsPlanning {
		t.Error("expected IsPlanning=false when omitted")
	}
	if meta.AggregatesFrom != "" {
		t.Errorf("expected AggregatesFrom='' when omitted, got %q", meta.AggregatesFrom)
	}
}

// TestGetByType_AllSlotsAndUnknown verifies GetByType returns the slot pointer
// for each of the seven recognized entity types (without falling back to
// defaults) and nil for an unknown type. This locks in GetByType as the
// single dispatcher source so aliases.go and other call sites can rely on it.
func TestGetByType_AllSlotsAndUnknown(t *testing.T) {
	// Construct an MLW where every slot has a uniquely identifiable value.
	mark := func(label string) *WorkflowConfig {
		return &WorkflowConfig{
			Version: label,
			StatusFlow: map[string][]string{
				label: {},
			},
		}
	}
	m := &MultiLevelWorkflow{
		Epic:     mark("epic-cfg"),
		Feature:  mark("feature-cfg"),
		Task:     mark("task-cfg"),
		Sprint:   mark("sprint-cfg"),
		Bug:      mark("bug-cfg"),
		Change:   mark("change-cfg"),
		TechDebt: mark("tech_debt-cfg"),
	}

	tests := []struct {
		entityType string
		wantLabel  string // empty means expect nil
	}{
		{"epic", "epic-cfg"},
		{"feature", "feature-cfg"},
		{"task", "task-cfg"},
		{"sprint", "sprint-cfg"},
		{"bug", "bug-cfg"},
		{"change", "change-cfg"},
		{"change_card", "change-cfg"},
		{"change-card", "change-cfg"},
		{"change-cards", "change-cfg"},
		{"tech_debt", "tech_debt-cfg"},
		{"tech-debt", "tech_debt-cfg"},
		{"td", "tech_debt-cfg"},
		{"unknown", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.entityType, func(t *testing.T) {
			got := m.GetByType(tt.entityType)
			if tt.wantLabel == "" {
				if got != nil {
					t.Fatalf("GetByType(%q) = %+v, want nil", tt.entityType, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("GetByType(%q) = nil, want slot with version %q", tt.entityType, tt.wantLabel)
			}
			if got.Version != tt.wantLabel {
				t.Errorf("GetByType(%q).Version = %q, want %q", tt.entityType, got.Version, tt.wantLabel)
			}
		})
	}

	// Nil slot should return nil (no fallback). Use a fresh empty MLW so
	// every slot is unset.
	empty := &MultiLevelWorkflow{}
	for _, et := range EntityTypes() {
		if got := empty.GetByType(et); got != nil {
			t.Errorf("empty MLW: GetByType(%q) = %+v, want nil (no fallback)", et, got)
		}
	}

	// EntityTypes must cover every slot GetByType recognizes. Verify by
	// ensuring every entry resolves to a non-nil slot on the marked MLW.
	for _, et := range EntityTypes() {
		if got := m.GetByType(et); got == nil {
			t.Errorf("EntityTypes() lists %q but GetByType returned nil for marked MLW", et)
		}
	}
}
