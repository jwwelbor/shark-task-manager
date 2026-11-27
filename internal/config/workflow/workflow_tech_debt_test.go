package workflow

import (
	"maps"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetWorkflowForLevel_TechDebtWithNil(t *testing.T) {
	m := &MultiLevelWorkflow{}
	wf := m.GetWorkflowForLevel("tech_debt")
	if wf == nil {
		t.Fatal("expected non-nil workflow for tech_debt level with nil TechDebt")
	}
	// Should return default tech-debt workflow with the research gate.
	if len(wf.StatusFlow) != 8 {
		t.Errorf("expected 8 statuses in default tech-debt workflow, got %d", len(wf.StatusFlow))
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
// tech-debt.yaml's outcomes maps. Every workable step routes "fail" back to
// "identified" (a self-loop for "identified" itself) and "blocked" to the
// "blocked" parking step, in addition to "cancelled" and "wont_fix", so each
// has 5 unique targets. The forward ("pass") target is always first
// (uniqueSortedOutcomeTargets in steps.go orders by outcome-key priority,
// pass=0 first).
func TestDefaultTechDebtWorkflow_StatusFlow(t *testing.T) {
	wf := DefaultTechDebtWorkflow()

	// identified -(pass)-> research, plus identified/blocked/cancelled/wont_fix.
	identifiedTransitions := wf.StatusFlow["identified"]
	if len(identifiedTransitions) != 5 {
		t.Errorf("expected 5 identified transitions, got %d: %v", len(identifiedTransitions), identifiedTransitions)
	}
	if len(identifiedTransitions) > 0 && identifiedTransitions[0] != "research" {
		t.Errorf("expected identified's forward transition to be 'research', got %v", identifiedTransitions)
	}

	// triaged -(pass)-> in_progress, plus identified/blocked/cancelled/wont_fix
	triagedTransitions := wf.StatusFlow["triaged"]
	if len(triagedTransitions) != 5 {
		t.Errorf("expected 5 triaged transitions, got %d: %v", len(triagedTransitions), triagedTransitions)
	}
	if len(triagedTransitions) > 0 && triagedTransitions[0] != "in_progress" {
		t.Errorf("expected triaged's forward transition to be 'in_progress', got %v", triagedTransitions)
	}

	// in_progress -(pass)-> resolved, plus identified/blocked/cancelled/wont_fix
	inProgressTransitions := wf.StatusFlow["in_progress"]
	if len(inProgressTransitions) != 5 {
		t.Errorf("expected 5 in_progress transitions, got %d: %v", len(inProgressTransitions), inProgressTransitions)
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
	if len(tdWf.StatusFlow) != 8 {
		t.Errorf("expected 8 statuses in default tech-debt workflow, got %d", len(tdWf.StatusFlow))
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

// techDebtWorkableSteps are the steps that route outcomes, in sorted order.
var techDebtWorkableSteps = []string{"identified", "in_progress", "research", "triaged"}

// techDebtOutcomes is every outcome name the tech-debt workflow defines.
var techDebtOutcomes = []string{"pass", "fail", "blocked", "cancelled", "wont_fix"}

func TestE31F01_TC020a_TechDebtStatusSet(t *testing.T) {
	wf := DefaultTechDebtWorkflow()

	require.NoError(t, ValidateWorkflow(wf))
	assert.ElementsMatch(t, []string{
		"blocked", "cancelled", "identified", "in_progress",
		"research", "resolved", "triaged", "wont_fix",
	}, slices.Collect(maps.Keys(wf.StatusFlow)))
	assert.Equal(t, []string{"identified"}, wf.SpecialStatuses[StartStatusKey])
	assert.ElementsMatch(t, []string{"cancelled", "resolved", "wont_fix"}, wf.SpecialStatuses[CompleteStatusKey])
	for name, step := range wf.Steps {
		assert.Emptyf(t, step.ReworkOutcomes, "step %s must define no rework_outcomes", name)
	}
}

func TestE31F01_TC020a_TechDebtBlockedParkingStep(t *testing.T) {
	step, ok := DefaultTechDebtWorkflow().GetStep("blocked")
	require.True(t, ok, "tech-debt workflow must define a blocked step")

	assert.Equal(t, "blocked", step.Phase)
	assert.Equal(t, "red", step.Color)
	assert.Equal(t, "BLK", step.DisplayToken)
	assert.Equal(t, "human", step.Responsibility)
	assert.Equal(t, "pause", step.Action)
	assert.True(t, step.Parking)
	assert.Equal(t, "tech_debt/blocked.md", step.Prompt)
	assert.Empty(t, step.Outcomes)
}

func TestE31F01_TC020b_TechDebtOutcomeRoutes(t *testing.T) {
	wf := DefaultTechDebtWorkflow()
	forward := map[string]string{
		"identified":  "research",
		"research":    "triaged",
		"triaged":     "in_progress",
		"in_progress": "resolved",
	}

	for _, from := range techDebtWorkableSteps {
		t.Run(from, func(t *testing.T) {
			routes := map[string]string{
				"pass": forward[from], "fail": "identified", "blocked": "blocked",
				"cancelled": "cancelled", "wont_fix": "wont_fix",
			}
			assertOutcomesResolve(t, wf, from, routes)
			assert.ElementsMatch(t, slices.Collect(maps.Values(routes)), wf.StatusFlow[from])
			require.NotEmpty(t, wf.StatusFlow[from])
			assert.Equal(t, forward[from], wf.StatusFlow[from][0], "forward target must be first")
		})
	}
	assert.Equal(t, techDebtWorkableSteps, wf.StatusFlow["blocked"], "parking resume targets")
}

// assertOutcomesResolve checks each route of one step as its own subtest.
func assertOutcomesResolve(t *testing.T, wf *WorkflowConfig, from string, routes map[string]string) {
	t.Helper()
	for outcome, want := range routes {
		t.Run(outcome, func(t *testing.T) {
			got, ok := wf.ResolveOutcome(from, outcome)
			assert.True(t, ok, "outcome must resolve")
			assert.Equal(t, want, got)
		})
	}
}

// assertNoOutcomeResolves checks that a step routes no worker outcome.
func assertNoOutcomeResolves(t *testing.T, wf *WorkflowConfig, from string) {
	t.Helper()
	for _, outcome := range techDebtOutcomes {
		t.Run(outcome, func(t *testing.T) {
			got, ok := wf.ResolveOutcome(from, outcome)
			assert.False(t, ok, "outcome must not resolve, got %q", got)
		})
	}
}

func TestE31F01_TC020c_TechDebtInvalidTransitions(t *testing.T) {
	wf := DefaultTechDebtWorkflow()

	for _, from := range techDebtWorkableSteps {
		t.Run(from, func(t *testing.T) {
			got, _ := wf.ResolveOutcome(from, "blocked")
			assert.NotEqual(t, "identified", got, "blocked must not loop back to identified")
		})
	}
	for _, from := range []string{"blocked", "resolved", "wont_fix", "cancelled"} {
		t.Run(from, func(t *testing.T) {
			require.Contains(t, wf.StatusFlow, from)
			assertNoOutcomeResolves(t, wf, from)
			if from != "blocked" {
				assert.Empty(t, wf.StatusFlow[from])
			}
		})
	}
}
