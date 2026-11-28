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
	// Should return default epic workflow (route-based epic.yaml has 13 steps,
	// including the BUG-5 intake "assessment" triage step and T-E34-F08-009's
	// "integration_review" step between active and completed)
	if len(wf.StatusFlow) != 13 {
		t.Errorf("expected 13 statuses in default epic workflow, got %d", len(wf.StatusFlow))
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

// TC-103: Question's shipped workflow contains the complete F02 serial route.
// The responder dispatch path, resolution pause, and every terminal provenance
// state must be present in the embedded fallback bundle.
func TestDefaultQuestionWorkflow_TC103(t *testing.T) {
	m := &MultiLevelWorkflow{}
	wf := m.GetWorkflowForLevel("question")
	if wf == nil {
		t.Fatal("GetWorkflowForLevel(question) = nil")
	}
	for _, status := range []string{"open", "answering", "ready_for_resolution", "resolved", "withdrawn", "superseded"} {
		if _, ok := wf.StatusFlow[status]; !ok {
			t.Fatalf("Question workflow missing %s", status)
		}
	}
	terminals := map[string]bool{}
	for _, status := range wf.SpecialStatuses[CompleteStatusKey] {
		terminals[status] = true
	}
	for _, status := range []string{"resolved", "withdrawn", "superseded"} {
		if !terminals[status] {
			t.Fatalf("Question %s status must be terminal", status)
		}
	}
	if got := wf.StatusMetadata["open"].OrchestratorAction.Action; got != "spawn_agent" {
		t.Fatalf("Question open action = %q, want spawn_agent", got)
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

// markedMultiLevelWorkflow returns an MLW where every slot has a uniquely
// identifiable value.
func markedMultiLevelWorkflow() *MultiLevelWorkflow {
	mark := func(label string) *WorkflowConfig {
		return &WorkflowConfig{
			Version: label,
			StatusFlow: map[string][]string{
				label: {},
			},
		}
	}
	return &MultiLevelWorkflow{
		Epic:     mark("epic-cfg"),
		Feature:  mark("feature-cfg"),
		Task:     mark("task-cfg"),
		Sprint:   mark("sprint-cfg"),
		Bug:      mark("bug-cfg"),
		Change:   mark("change-cfg"),
		TechDebt: mark("tech_debt-cfg"),
		Question: mark("question-cfg"),
	}
}

// getByTypeLabels maps each entity type spelling to the version label of the
// slot GetByType must return. An empty label means GetByType must return nil.
var getByTypeLabels = []struct {
	entityType string
	wantLabel  string
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
	{"question", "question-cfg"},
	{"questions", "question-cfg"},
	{"unknown", ""},
	{"", ""},
}

func assertSlotLabel(t *testing.T, m *MultiLevelWorkflow, entityType, wantLabel string) {
	t.Helper()
	got := m.GetByType(entityType)
	if wantLabel == "" {
		if got != nil {
			t.Fatalf("GetByType(%q) = %+v, want nil", entityType, got)
		}
		return
	}
	if got == nil {
		t.Fatalf("GetByType(%q) = nil, want slot with version %q", entityType, wantLabel)
	}
	if got.Version != wantLabel {
		t.Errorf("GetByType(%q).Version = %q, want %q", entityType, got.Version, wantLabel)
	}
}

// TestGetByType_AllSlotsAndUnknown verifies GetByType returns the slot pointer
// for each of the seven recognized entity types (without falling back to
// defaults) and nil for an unknown type. This locks in GetByType as the
// single dispatcher source so aliases.go and other call sites can rely on it.
func TestGetByType_AllSlotsAndUnknown(t *testing.T) {
	m := markedMultiLevelWorkflow()

	for _, tt := range getByTypeLabels {
		t.Run(tt.entityType, func(t *testing.T) {
			assertSlotLabel(t, m, tt.entityType, tt.wantLabel)
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
