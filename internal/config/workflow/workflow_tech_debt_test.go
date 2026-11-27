package workflow

import "testing"

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

	assertTechDebtSpecialStatuses(t, wf)

	// Check identified metadata
	identifiedMeta := requireStatusMetadata(t, wf, "identified")
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
	resolvedMeta := requireStatusMetadata(t, wf, "resolved")
	if resolvedMeta.Color != "green" {
		t.Errorf("expected resolved color 'green', got %q", resolvedMeta.Color)
	}
	if resolvedMeta.ProgressWeight != 1.0 {
		t.Errorf("expected resolved progress weight 1.0, got %f", resolvedMeta.ProgressWeight)
	}

	assertNoTransitions(t, wf, "wont_fix")
	assertNoTransitions(t, wf, "cancelled")
}

func assertTechDebtSpecialStatuses(t *testing.T, wf *WorkflowConfig) {
	t.Helper()

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
}

func requireStatusMetadata(t *testing.T, wf *WorkflowConfig, status string) StatusMetadata {
	t.Helper()
	meta, ok := wf.StatusMetadata[status]
	if !ok {
		t.Fatalf("expected '%s' in status metadata", status)
	}
	return meta
}

// assertNoTransitions checks that a status is present and terminal.
func assertNoTransitions(t *testing.T, wf *WorkflowConfig, status string) {
	t.Helper()
	transitions, ok := wf.StatusFlow[status]
	if !ok {
		t.Errorf("expected '%s' in status flow", status)
		return
	}
	if len(transitions) != 0 {
		t.Errorf("expected %s to be terminal (0 transitions), got %d", status, len(transitions))
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
	if len(bugWf.StatusFlow) != 11 {
		t.Errorf("expected 11 statuses in default bug workflow, got %d", len(bugWf.StatusFlow))
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
