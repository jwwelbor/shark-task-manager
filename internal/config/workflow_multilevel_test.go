package config

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
	// Should return default epic workflow with 4 statuses
	if len(wf.StatusFlow) != 4 {
		t.Errorf("expected 4 statuses in default epic workflow, got %d", len(wf.StatusFlow))
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
	if len(wf.StatusFlow) != 4 {
		t.Errorf("expected 4 statuses in default feature workflow, got %d", len(wf.StatusFlow))
	}
}

func TestGetWorkflowForLevel_TaskWithNil(t *testing.T) {
	m := &MultiLevelWorkflow{}
	wf := m.GetWorkflowForLevel("task")
	if wf == nil {
		t.Fatal("expected non-nil workflow for task level with nil Task")
	}
	// Default task workflow has 5 statuses
	if len(wf.StatusFlow) != 5 {
		t.Errorf("expected 5 statuses in default task workflow, got %d", len(wf.StatusFlow))
	}
	if _, ok := wf.StatusFlow["todo"]; !ok {
		t.Error("expected 'todo' status in default task workflow")
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
	if _, ok := wf.StatusFlow["todo"]; !ok {
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

	// Task workflow should have default statuses (5)
	if len(taskWf.StatusFlow) != 5 {
		t.Errorf("expected 5 statuses in default task workflow, got %d", len(taskWf.StatusFlow))
	}
}

func TestDefaultEpicWorkflow_PassesValidation(t *testing.T) {
	wf := DefaultEpicWorkflow()
	if err := ValidateWorkflow(wf); err != nil {
		t.Errorf("default epic workflow should pass validation: %v", err)
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
