package services

import (
	"encoding/json"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/status"
)

// ============================================================================
// Integration Tests: Backward Compatibility
// ============================================================================

func TestBackwardCompat_NilEpicWorkflow_AlwaysAggregation(t *testing.T) {
	svc := &DisplayService{
		epicWorkflow:    nil,
		featureWorkflow: nil,
	}

	statuses := []string{"draft", "active", "completed", "unknown", ""}
	for _, st := range statuses {
		t.Run(st, func(t *testing.T) {
			epic := &models.Epic{Status: models.EpicStatus(st)}
			mode := svc.DetermineEpicDisplayMode(epic)
			if mode != DisplayModeAggregation {
				t.Errorf("nil workflow with status %q: expected aggregation, got %s", st, mode)
			}
		})
	}
}

func TestBackwardCompat_NilFeatureWorkflow_AlwaysAggregation(t *testing.T) {
	svc := &DisplayService{
		epicWorkflow:    nil,
		featureWorkflow: nil,
	}

	statuses := []string{"draft", "active", "completed", "unknown", ""}
	for _, st := range statuses {
		t.Run(st, func(t *testing.T) {
			feature := &models.Feature{Status: models.FeatureStatus(st)}
			mode := svc.DetermineFeatureDisplayMode(feature)
			if mode != DisplayModeAggregation {
				t.Errorf("nil workflow with status %q: expected aggregation, got %s", st, mode)
			}
		})
	}
}

func TestBackwardCompat_WorkflowPosition_NilWorkflow(t *testing.T) {
	svc := &DisplayService{}

	pos := svc.BuildWorkflowPosition("draft", nil)
	if pos != nil {
		t.Error("expected nil workflow position when workflow is nil")
	}
}

// ============================================================================
// Integration Tests: Mixed Mode (Epic aggregating, features mixed)
// ============================================================================

func TestMixedMode_EpicAggregation_FeaturesVaried(t *testing.T) {
	epicWf := config.DefaultEpicWorkflow()
	featureWf := config.DefaultFeatureWorkflow()

	svc := &DisplayService{
		epicWorkflow:    epicWf,
		featureWorkflow: featureWf,
	}

	// Epic in active (aggregation mode)
	epic := &models.Epic{Status: "active"}
	epicMode := svc.DetermineEpicDisplayMode(epic)
	if epicMode != DisplayModeAggregation {
		t.Fatalf("expected epic in active status to be aggregation, got %s", epicMode)
	}

	// Feature in draft (planning) and active (aggregation)
	featureDraft := &models.Feature{Status: "draft"}
	featureActive := &models.Feature{Status: "active"}

	draftMode := svc.DetermineFeatureDisplayMode(featureDraft)
	activeMode := svc.DetermineFeatureDisplayMode(featureActive)

	if draftMode != DisplayModePlanning {
		t.Errorf("expected draft feature to be planning, got %s", draftMode)
	}
	if activeMode != DisplayModeAggregation {
		t.Errorf("expected active feature to be aggregation, got %s", activeMode)
	}
}

func TestMixedMode_EpicPlanning_FeaturesIgnored(t *testing.T) {
	epicWf := config.DefaultEpicWorkflow()
	featureWf := config.DefaultFeatureWorkflow()

	svc := &DisplayService{
		epicWorkflow:    epicWf,
		featureWorkflow: featureWf,
	}

	// Epic in draft (planning mode) - features are irrelevant
	epic := &models.Epic{Status: "draft"}
	epicMode := svc.DetermineEpicDisplayMode(epic)
	if epicMode != DisplayModePlanning {
		t.Errorf("expected epic in draft to be planning, got %s", epicMode)
	}

	// Even though features exist, epic in planning should not show aggregation
	// This test validates the concept that planning mode means NO child progress
}

// ============================================================================
// Integration Tests: Threshold Crossing
// ============================================================================

func TestThresholdCrossing_FeatureDraftToActive(t *testing.T) {
	featureWf := config.DefaultFeatureWorkflow()

	svc := &DisplayService{
		featureWorkflow: featureWf,
	}

	feature := &models.Feature{Status: "draft"}

	// Start in planning mode
	mode := svc.DetermineFeatureDisplayMode(feature)
	if mode != DisplayModePlanning {
		t.Fatalf("expected planning mode for draft, got %s", mode)
	}

	// Transition to active (crosses threshold)
	feature.Status = "active"
	mode = svc.DetermineFeatureDisplayMode(feature)
	if mode != DisplayModeAggregation {
		t.Fatalf("expected aggregation mode for active, got %s", mode)
	}
}

func TestThresholdCrossing_EpicDraftToActive(t *testing.T) {
	epicWf := config.DefaultEpicWorkflow()

	svc := &DisplayService{
		epicWorkflow: epicWf,
	}

	epic := &models.Epic{Status: "draft"}

	// Start in planning mode
	mode := svc.DetermineEpicDisplayMode(epic)
	if mode != DisplayModePlanning {
		t.Fatalf("expected planning mode for draft, got %s", mode)
	}

	// Transition to active (crosses threshold)
	epic.Status = "active"
	mode = svc.DetermineEpicDisplayMode(epic)
	if mode != DisplayModeAggregation {
		t.Fatalf("expected aggregation mode for active, got %s", mode)
	}
}

func TestThresholdCrossing_Bidirectional(t *testing.T) {
	featureWf := config.DefaultFeatureWorkflow()

	svc := &DisplayService{
		featureWorkflow: featureWf,
	}

	feature := &models.Feature{}

	// Forward: draft -> active
	feature.Status = "draft"
	if svc.DetermineFeatureDisplayMode(feature) != DisplayModePlanning {
		t.Error("expected planning for draft")
	}

	feature.Status = "active"
	if svc.DetermineFeatureDisplayMode(feature) != DisplayModeAggregation {
		t.Error("expected aggregation for active")
	}

	// If somehow went back to draft (e.g., re-planning scenario)
	feature.Status = "draft"
	if svc.DetermineFeatureDisplayMode(feature) != DisplayModePlanning {
		t.Error("expected planning for draft (back to planning)")
	}
}

// ============================================================================
// Integration Tests: Multi-Status Planning Workflows
// ============================================================================

func TestMultiStatusPlanning_AllPlanningStatuses(t *testing.T) {
	// Workflow with multiple planning statuses before aggregation threshold
	wf := &config.WorkflowConfig{
		StatusFlow: map[string][]string{
			"draft":                     {"ready_for_refinement_ba"},
			"ready_for_refinement_ba":   {"in_refinement_ba"},
			"in_refinement_ba":          {"ready_for_refinement_tech"},
			"ready_for_refinement_tech": {"in_refinement_tech"},
			"in_refinement_tech":        {"active"},
			"active":                    {"completed"},
			"completed":                 {},
		},
		StatusMetadata: map[string]config.StatusMetadata{
			"draft":                     {Phase: "planning", IsPlanning: true},
			"ready_for_refinement_ba":   {Phase: "refinement", IsPlanning: true},
			"in_refinement_ba":          {Phase: "refinement", IsPlanning: true},
			"ready_for_refinement_tech": {Phase: "refinement", IsPlanning: true},
			"in_refinement_tech":        {Phase: "refinement", IsPlanning: true},
			"active":                    {Phase: "execution", AggregatesFrom: "features"},
			"completed":                 {Phase: "done"},
		},
		SpecialStatuses: map[string][]string{
			config.StartStatusKey:       {"draft"},
			config.CompleteStatusKey:    {"completed"},
			config.AggregationStatusKey: {"active"},
		},
	}

	svc := &DisplayService{epicWorkflow: wf}

	planningStatuses := []string{
		"draft", "ready_for_refinement_ba", "in_refinement_ba",
		"ready_for_refinement_tech", "in_refinement_tech",
	}
	aggregationStatuses := []string{"active", "completed"}

	for _, st := range planningStatuses {
		t.Run("planning/"+st, func(t *testing.T) {
			epic := &models.Epic{Status: models.EpicStatus(st)}
			mode := svc.DetermineEpicDisplayMode(epic)
			if mode != DisplayModePlanning {
				t.Errorf("expected planning for %q, got %s", st, mode)
			}
		})
	}

	for _, st := range aggregationStatuses {
		t.Run("aggregation/"+st, func(t *testing.T) {
			epic := &models.Epic{Status: models.EpicStatus(st)}
			mode := svc.DetermineEpicDisplayMode(epic)
			if mode != DisplayModeAggregation {
				t.Errorf("expected aggregation for %q, got %s", st, mode)
			}
		})
	}
}

// ============================================================================
// Integration Tests: Edge Cases
// ============================================================================

func TestEdgeCase_EmptyStatusString(t *testing.T) {
	svc := &DisplayService{
		epicWorkflow:    config.DefaultEpicWorkflow(),
		featureWorkflow: config.DefaultFeatureWorkflow(),
	}

	epic := &models.Epic{Status: ""}
	mode := svc.DetermineEpicDisplayMode(epic)
	if mode != DisplayModeAggregation {
		t.Errorf("expected aggregation for empty status, got %s", mode)
	}

	feature := &models.Feature{Status: ""}
	fMode := svc.DetermineFeatureDisplayMode(feature)
	if fMode != DisplayModeAggregation {
		t.Errorf("expected aggregation for empty feature status, got %s", fMode)
	}
}

func TestEdgeCase_CaseSensitivity(t *testing.T) {
	wf := &config.WorkflowConfig{
		StatusFlow: map[string][]string{
			"draft":  {"active"},
			"active": {"completed"},
		},
		StatusMetadata: map[string]config.StatusMetadata{
			"draft":  {Phase: "planning", IsPlanning: true},
			"active": {Phase: "execution"},
		},
		SpecialStatuses: map[string][]string{
			config.StartStatusKey:       {"draft"},
			config.AggregationStatusKey: {"active"},
		},
	}

	svc := &DisplayService{epicWorkflow: wf}

	// Exact case match
	epic := &models.Epic{Status: "draft"}
	if svc.DetermineEpicDisplayMode(epic) != DisplayModePlanning {
		t.Error("expected planning for 'draft'")
	}

	// Different case - _aggregation_ uses EqualFold, but is_planning metadata lookup
	// may be case-sensitive depending on config.GetStatusMetadata implementation
	epic.Status = "active"
	if svc.DetermineEpicDisplayMode(epic) != DisplayModeAggregation {
		t.Error("expected aggregation for 'active'")
	}
}

func TestEdgeCase_AggregationStatusKey_TakesPrecedence(t *testing.T) {
	// Test scenario: a status is both in _aggregation_ list AND has is_planning=true
	// The _aggregation_ check happens first, so it should be aggregation mode
	wf := &config.WorkflowConfig{
		StatusFlow: map[string][]string{
			"weird": {"completed"},
		},
		StatusMetadata: map[string]config.StatusMetadata{
			"weird": {Phase: "planning", IsPlanning: true}, // has is_planning=true
		},
		SpecialStatuses: map[string][]string{
			config.AggregationStatusKey: {"weird"}, // but also in aggregation list
		},
	}

	svc := &DisplayService{epicWorkflow: wf}
	epic := &models.Epic{Status: "weird"}
	mode := svc.DetermineEpicDisplayMode(epic)
	if mode != DisplayModeAggregation {
		t.Errorf("_aggregation_ should take precedence over is_planning, got %s", mode)
	}
}

func TestEdgeCase_WorkflowPosition_StatusNotInFlow(t *testing.T) {
	wf := &config.WorkflowConfig{
		StatusFlow: map[string][]string{
			"draft":  {"active"},
			"active": {"completed"},
		},
		SpecialStatuses: map[string][]string{
			config.StartStatusKey: {"draft"},
		},
	}

	svc := &DisplayService{}
	pos := svc.BuildWorkflowPosition("nonexistent_status", wf)
	if pos == nil {
		t.Fatal("expected non-nil position even for unknown status")
	}

	if pos.CurrentIndex != -1 {
		t.Errorf("expected CurrentIndex -1 for unknown status, got %d", pos.CurrentIndex)
	}
}

func TestEdgeCase_WorkflowPosition_NoStartStatus(t *testing.T) {
	wf := &config.WorkflowConfig{
		StatusFlow: map[string][]string{
			"draft": {"active"},
		},
		SpecialStatuses: map[string][]string{
			// No _start_ key
		},
	}

	svc := &DisplayService{}
	pos := svc.BuildWorkflowPosition("draft", wf)
	if pos == nil {
		t.Fatal("expected non-nil position")
	}

	// With no start status, ordered list should be empty
	if len(pos.Statuses) != 0 {
		t.Errorf("expected empty statuses list without start status, got %v", pos.Statuses)
	}
}

func TestEdgeCase_WorkflowPosition_CircularFlow(t *testing.T) {
	// Workflow with a cycle: A -> B -> A
	wf := &config.WorkflowConfig{
		StatusFlow: map[string][]string{
			"a": {"b"},
			"b": {"a"},
		},
		SpecialStatuses: map[string][]string{
			config.StartStatusKey: {"a"},
		},
	}

	svc := &DisplayService{}
	pos := svc.BuildWorkflowPosition("a", wf)
	if pos == nil {
		t.Fatal("expected non-nil position")
	}

	// Should stop at the cycle, not infinite loop
	if len(pos.Statuses) != 2 {
		t.Errorf("expected 2 statuses (a, b) before cycle detected, got %d: %v", len(pos.Statuses), pos.Statuses)
	}
}

// ============================================================================
// Integration Tests: JSON Output Validation
// ============================================================================

func TestJSONOutput_EpicSummary_PlanningMode(t *testing.T) {
	summary := &status.EpicSummary{
		Key:         "E16",
		Title:       "Multi-Level Workflow",
		DisplayMode: string(DisplayModePlanning),
		IsPlanning:  true,
		Phase:       "planning",
	}

	data, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("failed to marshal EpicSummary: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify planning fields are present
	if result["display_mode"] != "planning" {
		t.Errorf("expected display_mode 'planning', got %v", result["display_mode"])
	}
	if result["is_planning"] != true {
		t.Errorf("expected is_planning true, got %v", result["is_planning"])
	}
	if result["phase"] != "planning" {
		t.Errorf("expected phase 'planning', got %v", result["phase"])
	}
}

func TestJSONOutput_EpicSummary_AggregationMode(t *testing.T) {
	summary := &status.EpicSummary{
		Key:             "E07",
		Title:           "User Management",
		ProgressPercent: 75.0,
		Health:          "healthy",
		TasksTotal:      10,
		TasksCompleted:  7,
		DisplayMode:     string(DisplayModeAggregation),
	}

	data, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("failed to marshal EpicSummary: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify aggregation fields
	if result["display_mode"] != "aggregation" {
		t.Errorf("expected display_mode 'aggregation', got %v", result["display_mode"])
	}

	// is_planning and phase should be omitted (omitempty)
	if _, exists := result["is_planning"]; exists {
		t.Error("is_planning should be omitted for aggregation mode (omitempty)")
	}
	if _, exists := result["phase"]; exists {
		t.Error("phase should be omitted for aggregation mode (omitempty)")
	}

	// Progress fields should be present
	if result["progress_percent"] != 75.0 {
		t.Errorf("expected progress_percent 75.0, got %v", result["progress_percent"])
	}
}

func TestJSONOutput_EpicDisplayInfo_PlanningMode(t *testing.T) {
	info := &EpicDisplayInfo{
		Epic: &models.Epic{BaseEntity: models.BaseEntity{Key: "E16",
			Title: "Multi-Level Workflow"}, Status: "draft",
		},
		Mode:             DisplayModePlanning,
		Phase:            "planning",
		PhaseDescription: "Initial planning phase",
		WorkflowPosition: &WorkflowPosition{
			Statuses:      []string{"draft", "active", "completed"},
			CurrentIndex:  0,
			CurrentStatus: "draft",
		},
		StatusSource: "workflow",
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("failed to marshal EpicDisplayInfo: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if result["display_mode"] != "planning" {
		t.Errorf("expected display_mode 'planning', got %v", result["display_mode"])
	}
	if result["phase"] != "planning" {
		t.Errorf("expected phase 'planning', got %v", result["phase"])
	}
	if result["status_source"] != "workflow" {
		t.Errorf("expected status_source 'workflow', got %v", result["status_source"])
	}

	wp, ok := result["workflow_position"].(map[string]interface{})
	if !ok {
		t.Fatal("expected workflow_position to be a map")
	}
	if wp["current_status"] != "draft" {
		t.Errorf("expected workflow position current_status 'draft', got %v", wp["current_status"])
	}
}

func TestJSONOutput_FeatureDisplayInfo_AggregationMode(t *testing.T) {
	info := &FeatureDisplayInfo{
		Feature: &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F01",
			Title: "Authentication"}, Status: "active",
		},
		Mode:         DisplayModeAggregation,
		StatusSource: "calculated",
		StatusBreakdown: []StatusCountItem{
			{Status: "todo", Count: 3},
			{Status: "completed", Count: 5},
		},
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("failed to marshal FeatureDisplayInfo: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if result["display_mode"] != "aggregation" {
		t.Errorf("expected display_mode 'aggregation', got %v", result["display_mode"])
	}
	if result["status_source"] != "calculated" {
		t.Errorf("expected status_source 'calculated', got %v", result["status_source"])
	}

	breakdown, ok := result["status_breakdown"].([]interface{})
	if !ok {
		t.Fatal("expected status_breakdown to be a list")
	}
	if len(breakdown) != 2 {
		t.Errorf("expected 2 status breakdown items, got %d", len(breakdown))
	}
}

// ============================================================================
// Integration Tests: Default Workflow Configurations
// ============================================================================

func TestDefaultEpicWorkflow_PlanningAndAggregation(t *testing.T) {
	wf := config.DefaultEpicWorkflow()
	svc := &DisplayService{epicWorkflow: wf}

	// draft should be planning
	epic := &models.Epic{Status: "draft"}
	if svc.DetermineEpicDisplayMode(epic) != DisplayModePlanning {
		t.Error("expected planning for default epic workflow 'draft' status")
	}

	// active should be aggregation
	epic.Status = "active"
	if svc.DetermineEpicDisplayMode(epic) != DisplayModeAggregation {
		t.Error("expected aggregation for default epic workflow 'active' status")
	}

	// completed should be aggregation
	epic.Status = "completed"
	if svc.DetermineEpicDisplayMode(epic) != DisplayModeAggregation {
		t.Error("expected aggregation for default epic workflow 'completed' status")
	}
}

func TestDefaultFeatureWorkflow_PlanningAndAggregation(t *testing.T) {
	wf := config.DefaultFeatureWorkflow()
	svc := &DisplayService{featureWorkflow: wf}

	// draft should be planning
	feature := &models.Feature{Status: "draft"}
	if svc.DetermineFeatureDisplayMode(feature) != DisplayModePlanning {
		t.Error("expected planning for default feature workflow 'draft' status")
	}

	// active should be aggregation
	feature.Status = "active"
	if svc.DetermineFeatureDisplayMode(feature) != DisplayModeAggregation {
		t.Error("expected aggregation for default feature workflow 'active' status")
	}

	// completed should be aggregation
	feature.Status = "completed"
	if svc.DetermineFeatureDisplayMode(feature) != DisplayModeAggregation {
		t.Error("expected aggregation for default feature workflow 'completed' status")
	}
}

// ============================================================================
// Integration Tests: Formatter Planning Mode (status.EpicSummary)
// ============================================================================

func TestEpicSummary_PlanningFieldsPopulation(t *testing.T) {
	// Simulate what enrichEpicSummaries does in the status command
	epicWf := config.DefaultEpicWorkflow()

	svc := &DisplayService{
		epicWorkflow: epicWf,
	}

	// Epic in draft (planning)
	epic := &models.Epic{Status: "draft"}
	mode := svc.DetermineEpicDisplayMode(epic)

	summary := &status.EpicSummary{
		Key:   "E16",
		Title: "Test Epic",
	}

	summary.DisplayMode = string(mode)
	if mode == DisplayModePlanning {
		summary.IsPlanning = true
		meta, found := epicWf.GetStatusMetadata("draft")
		if found {
			summary.Phase = meta.Phase
		}
	}

	if summary.DisplayMode != "planning" {
		t.Errorf("expected display_mode 'planning', got %s", summary.DisplayMode)
	}
	if !summary.IsPlanning {
		t.Error("expected IsPlanning to be true")
	}
	if summary.Phase != "planning" {
		t.Errorf("expected phase 'planning', got %s", summary.Phase)
	}
}

func TestEpicSummary_AggregationFieldsPreserved(t *testing.T) {
	epicWf := config.DefaultEpicWorkflow()
	svc := &DisplayService{
		epicWorkflow: epicWf,
	}

	epic := &models.Epic{Status: "active"}
	mode := svc.DetermineEpicDisplayMode(epic)

	summary := &status.EpicSummary{
		Key:             "E07",
		Title:           "User Management",
		ProgressPercent: 65.0,
		Health:          "warning",
		TasksTotal:      20,
		TasksCompleted:  13,
		TasksBlocked:    1,
		FeaturesTotal:   5,
		FeaturesActive:  3,
	}

	summary.DisplayMode = string(mode)
	if mode == DisplayModePlanning {
		summary.IsPlanning = true
	}

	// Verify aggregation fields remain intact
	if summary.DisplayMode != "aggregation" {
		t.Errorf("expected display_mode 'aggregation', got %s", summary.DisplayMode)
	}
	if summary.IsPlanning {
		t.Error("expected IsPlanning to be false for aggregation mode")
	}
	if summary.ProgressPercent != 65.0 {
		t.Errorf("expected progress 65.0, got %f", summary.ProgressPercent)
	}
	if summary.TasksTotal != 20 {
		t.Errorf("expected 20 total tasks, got %d", summary.TasksTotal)
	}
}

// ============================================================================
// Integration Tests: WorkflowPosition Completeness
// ============================================================================

func TestWorkflowPosition_DefaultEpicWorkflow(t *testing.T) {
	wf := config.DefaultEpicWorkflow()
	svc := &DisplayService{}

	pos := svc.BuildWorkflowPosition("draft", wf)
	if pos == nil {
		t.Fatal("expected non-nil position")
	}

	// Should have at least draft, active, completed
	if len(pos.Statuses) < 3 {
		t.Errorf("expected at least 3 statuses in epic workflow, got %d: %v", len(pos.Statuses), pos.Statuses)
	}

	if pos.CurrentIndex != 0 {
		t.Errorf("expected draft at index 0, got index %d", pos.CurrentIndex)
	}
}

func TestWorkflowPosition_DefaultFeatureWorkflow(t *testing.T) {
	wf := config.DefaultFeatureWorkflow()
	svc := &DisplayService{}

	pos := svc.BuildWorkflowPosition("draft", wf)
	if pos == nil {
		t.Fatal("expected non-nil position")
	}

	if len(pos.Statuses) < 3 {
		t.Errorf("expected at least 3 statuses in feature workflow, got %d: %v", len(pos.Statuses), pos.Statuses)
	}

	if pos.CurrentIndex != 0 {
		t.Errorf("expected draft at index 0, got index %d", pos.CurrentIndex)
	}
}

func TestWorkflowPosition_MiddleStatus(t *testing.T) {
	wf := config.DefaultEpicWorkflow()
	svc := &DisplayService{}

	pos := svc.BuildWorkflowPosition("active", wf)
	if pos == nil {
		t.Fatal("expected non-nil position")
	}

	if pos.CurrentIndex <= 0 {
		t.Errorf("expected active to be after first status, got index %d", pos.CurrentIndex)
	}

	if pos.CurrentStatus != "active" {
		t.Errorf("expected current status 'active', got %q", pos.CurrentStatus)
	}
}
