package services

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

func TestDetermineDisplayMode_EpicPlanning(t *testing.T) {
	svc := &DisplayService{
		epicWorkflow: testEpicWorkflow(),
	}

	epic := &models.Epic{Status: "draft"}
	mode := svc.DetermineEpicDisplayMode(epic)
	if mode != DisplayModePlanning {
		t.Errorf("expected planning mode for draft epic, got %s", mode)
	}
}

func TestDetermineDisplayMode_EpicAggregation(t *testing.T) {
	svc := &DisplayService{
		epicWorkflow: testEpicWorkflow(),
	}

	epic := &models.Epic{Status: "active"}
	mode := svc.DetermineEpicDisplayMode(epic)
	if mode != DisplayModeAggregation {
		t.Errorf("expected aggregation mode for active epic, got %s", mode)
	}
}

func TestDetermineDisplayMode_EpicCompleted(t *testing.T) {
	svc := &DisplayService{
		epicWorkflow: testEpicWorkflow(),
	}

	epic := &models.Epic{Status: "completed"}
	mode := svc.DetermineEpicDisplayMode(epic)
	if mode != DisplayModeAggregation {
		t.Errorf("expected aggregation mode for completed epic, got %s", mode)
	}
}

func TestDetermineDisplayMode_FeaturePlanning(t *testing.T) {
	svc := &DisplayService{
		featureWorkflow: testFeatureWorkflow(),
	}

	feature := &models.Feature{Status: "draft"}
	mode := svc.DetermineFeatureDisplayMode(feature)
	if mode != DisplayModePlanning {
		t.Errorf("expected planning mode for draft feature, got %s", mode)
	}
}

func TestDetermineDisplayMode_FeatureAggregation(t *testing.T) {
	svc := &DisplayService{
		featureWorkflow: testFeatureWorkflow(),
	}

	feature := &models.Feature{Status: "active"}
	mode := svc.DetermineFeatureDisplayMode(feature)
	if mode != DisplayModeAggregation {
		t.Errorf("expected aggregation mode for active feature, got %s", mode)
	}
}

func TestDetermineDisplayMode_NilWorkflow(t *testing.T) {
	svc := &DisplayService{
		epicWorkflow: nil,
	}

	epic := &models.Epic{Status: "draft"}
	mode := svc.DetermineEpicDisplayMode(epic)
	if mode != DisplayModeAggregation {
		t.Errorf("expected aggregation mode when no workflow configured, got %s", mode)
	}
}

func TestDetermineDisplayMode_UnknownStatus(t *testing.T) {
	svc := &DisplayService{
		epicWorkflow: testEpicWorkflow(),
	}

	epic := &models.Epic{Status: "unknown_status"}
	mode := svc.DetermineEpicDisplayMode(epic)
	if mode != DisplayModeAggregation {
		t.Errorf("expected aggregation mode for unknown status, got %s", mode)
	}
}

func TestDetermineDisplayMode_CustomPlanningStatuses(t *testing.T) {
	// Test with a custom workflow that has multiple planning statuses
	wf := &config.WorkflowConfig{
		StatusFlow: map[string][]string{
			"draft":                   {"ready_for_refinement_ba"},
			"ready_for_refinement_ba": {"in_refinement_ba"},
			"in_refinement_ba":        {"active"},
			"active":                  {"completed"},
			"completed":               {},
		},
		StatusMetadata: map[string]config.StatusMetadata{
			"draft":                   {Phase: "planning", IsPlanning: true},
			"ready_for_refinement_ba": {Phase: "refinement", IsPlanning: true},
			"in_refinement_ba":        {Phase: "refinement", IsPlanning: true},
			"active":                  {Phase: "execution", AggregatesFrom: "features"},
			"completed":               {Phase: "done"},
		},
		SpecialStatuses: map[string][]string{
			config.StartStatusKey:       {"draft"},
			config.CompleteStatusKey:    {"completed"},
			config.AggregationStatusKey: {"active"},
		},
	}

	svc := &DisplayService{epicWorkflow: wf}

	tests := []struct {
		status string
		want   DisplayMode
	}{
		{"draft", DisplayModePlanning},
		{"ready_for_refinement_ba", DisplayModePlanning},
		{"in_refinement_ba", DisplayModePlanning},
		{"active", DisplayModeAggregation},
		{"completed", DisplayModeAggregation},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			epic := &models.Epic{Status: models.EpicStatus(tt.status)}
			got := svc.DetermineEpicDisplayMode(epic)
			if got != tt.want {
				t.Errorf("status %q: got %s, want %s", tt.status, got, tt.want)
			}
		})
	}
}

func TestBuildWorkflowPosition(t *testing.T) {
	svc := &DisplayService{
		epicWorkflow: testEpicWorkflow(),
	}

	pos := svc.BuildWorkflowPosition("active", testEpicWorkflow())
	if pos == nil {
		t.Fatal("expected non-nil workflow position")
	}

	if pos.CurrentStatus != "active" {
		t.Errorf("expected current status 'active', got %q", pos.CurrentStatus)
	}

	if len(pos.Statuses) == 0 {
		t.Error("expected non-empty statuses list")
	}

	// Verify current status is in the list
	found := false
	for i, s := range pos.Statuses {
		if s == "active" {
			found = true
			if pos.CurrentIndex != i {
				t.Errorf("CurrentIndex %d doesn't match position %d of 'active'", pos.CurrentIndex, i)
			}
			break
		}
	}
	if !found {
		t.Error("current status 'active' not found in statuses list")
	}
}

func TestBuildWorkflowPosition_NilWorkflow(t *testing.T) {
	svc := &DisplayService{}

	pos := svc.BuildWorkflowPosition("draft", nil)
	if pos != nil {
		t.Error("expected nil workflow position for nil workflow")
	}
}

func TestBuildWorkflowPosition_CustomWorkflow(t *testing.T) {
	wf := &config.WorkflowConfig{
		StatusFlow: map[string][]string{
			"draft":                     {"ready_for_refinement_ba", "active"},
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
			"active":                    {Phase: "execution"},
			"completed":                 {Phase: "done"},
		},
		SpecialStatuses: map[string][]string{
			config.StartStatusKey:    {"draft"},
			config.CompleteStatusKey: {"completed"},
		},
	}

	svc := &DisplayService{}
	pos := svc.BuildWorkflowPosition("in_refinement_ba", wf)
	if pos == nil {
		t.Fatal("expected non-nil workflow position")
	}

	if pos.CurrentStatus != "in_refinement_ba" {
		t.Errorf("expected current status 'in_refinement_ba', got %q", pos.CurrentStatus)
	}

	// Verify the happy path follows first non-blocked transitions
	expectedOrder := []string{
		"draft", "ready_for_refinement_ba", "in_refinement_ba",
		"ready_for_refinement_tech", "in_refinement_tech", "active", "completed",
	}

	if len(pos.Statuses) != len(expectedOrder) {
		t.Errorf("expected %d statuses, got %d: %v", len(expectedOrder), len(pos.Statuses), pos.Statuses)
	} else {
		for i, expected := range expectedOrder {
			if pos.Statuses[i] != expected {
				t.Errorf("position %d: expected %q, got %q", i, expected, pos.Statuses[i])
			}
		}
	}
}

func TestBuildOrderedStatuses_DefaultEpicWorkflow(t *testing.T) {
	wf := config.DefaultEpicWorkflow()
	ordered := buildOrderedStatuses(wf)

	// Should follow: draft -> active -> completed
	if len(ordered) < 3 {
		t.Fatalf("expected at least 3 statuses, got %d: %v", len(ordered), ordered)
	}

	if ordered[0] != "draft" {
		t.Errorf("expected first status to be 'draft', got %q", ordered[0])
	}
}

func TestBuildOrderedStatuses_DefaultFeatureWorkflow(t *testing.T) {
	wf := config.DefaultFeatureWorkflow()
	ordered := buildOrderedStatuses(wf)

	if len(ordered) < 3 {
		t.Fatalf("expected at least 3 statuses, got %d: %v", len(ordered), ordered)
	}

	if ordered[0] != "draft" {
		t.Errorf("expected first status to be 'draft', got %q", ordered[0])
	}
}

// ============================================================================
// Tests: Notes and ContextData fields on display structs (T-E07-F32-002)
// ============================================================================

func TestEpicDisplayInfo_HasNotesField(t *testing.T) {
	info := &EpicDisplayInfo{
		Notes: []*models.EntityNote{
			{ID: 1, EntityType: models.EntityTypeEpic, Content: "test note"},
		},
	}
	if len(info.Notes) != 1 {
		t.Errorf("expected 1 note, got %d", len(info.Notes))
	}
	if info.Notes[0].Content != "test note" {
		t.Errorf("expected note content 'test note', got %q", info.Notes[0].Content)
	}
}

func TestEpicDisplayInfo_HasContextDataField(t *testing.T) {
	step := "Writing tests"
	info := &EpicDisplayInfo{
		ContextData: &models.ContextData{
			Progress: &models.ProgressContext{
				CurrentStep: &step,
			},
		},
	}
	if info.ContextData == nil {
		t.Fatal("expected non-nil ContextData")
	}
	if info.ContextData.Progress == nil || info.ContextData.Progress.CurrentStep == nil {
		t.Fatal("expected non-nil Progress.CurrentStep")
	}
	if *info.ContextData.Progress.CurrentStep != "Writing tests" {
		t.Errorf("expected current step 'Writing tests', got %q", *info.ContextData.Progress.CurrentStep)
	}
}

func TestFeatureDisplayInfo_HasNotesField(t *testing.T) {
	info := &FeatureDisplayInfo{
		Notes: []*models.EntityNote{
			{ID: 1, EntityType: models.EntityTypeFeature, Content: "feature note"},
		},
	}
	if len(info.Notes) != 1 {
		t.Errorf("expected 1 note, got %d", len(info.Notes))
	}
	if info.Notes[0].Content != "feature note" {
		t.Errorf("expected note content 'feature note', got %q", info.Notes[0].Content)
	}
}

func TestFeatureDisplayInfo_HasContextDataField(t *testing.T) {
	step := "Implementing API"
	info := &FeatureDisplayInfo{
		ContextData: &models.ContextData{
			Progress: &models.ProgressContext{
				CurrentStep: &step,
			},
		},
	}
	if info.ContextData == nil {
		t.Fatal("expected non-nil ContextData")
	}
	if *info.ContextData.Progress.CurrentStep != "Implementing API" {
		t.Errorf("expected current step 'Implementing API', got %q", *info.ContextData.Progress.CurrentStep)
	}
}

func TestEpicDisplayInfo_NotesAndContextInJSON(t *testing.T) {
	step := "Step 1"
	info := &EpicDisplayInfo{
		Epic: &models.Epic{BaseEntity: models.BaseEntity{Key: "E07", Title: "Test"}, Status: "draft"},
		Mode: DisplayModePlanning,
		Notes: []*models.EntityNote{
			{ID: 1, Content: "a note"},
		},
		ContextData: &models.ContextData{
			Progress: &models.ProgressContext{CurrentStep: &step},
		},
		StatusSource: "workflow",
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	notes, ok := result["notes"].([]interface{})
	if !ok {
		t.Fatal("expected notes to be an array in JSON")
	}
	if len(notes) != 1 {
		t.Errorf("expected 1 note in JSON, got %d", len(notes))
	}

	cd, ok := result["context_data"].(map[string]interface{})
	if !ok {
		t.Fatal("expected context_data to be an object in JSON")
	}
	progress, ok := cd["progress"].(map[string]interface{})
	if !ok {
		t.Fatal("expected context_data.progress to be an object")
	}
	if progress["current_step"] != "Step 1" {
		t.Errorf("expected current_step 'Step 1', got %v", progress["current_step"])
	}
}

func TestFeatureDisplayInfo_NotesAndContextInJSON(t *testing.T) {
	step := "Building UI"
	info := &FeatureDisplayInfo{
		Feature: &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F01", Title: "Auth"}, Status: "draft"},
		Mode:    DisplayModePlanning,
		Notes: []*models.EntityNote{
			{ID: 1, Content: "feature note"},
		},
		ContextData: &models.ContextData{
			Progress: &models.ProgressContext{CurrentStep: &step},
		},
		StatusSource: "workflow",
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	notes, ok := result["notes"].([]interface{})
	if !ok {
		t.Fatal("expected notes to be an array in JSON")
	}
	if len(notes) != 1 {
		t.Errorf("expected 1 note in JSON, got %d", len(notes))
	}

	cd, ok := result["context_data"].(map[string]interface{})
	if !ok {
		t.Fatal("expected context_data to be an object in JSON")
	}
	progress, ok := cd["progress"].(map[string]interface{})
	if !ok {
		t.Fatal("expected context_data.progress to be an object")
	}
	if progress["current_step"] != "Building UI" {
		t.Errorf("expected current_step 'Building UI', got %v", progress["current_step"])
	}
}

func TestEpicDisplayInfo_NoNotesOmitsFromJSON(t *testing.T) {
	info := &EpicDisplayInfo{
		Epic:         &models.Epic{BaseEntity: models.BaseEntity{Key: "E07", Title: "Test"}, Status: "draft"},
		Mode:         DisplayModePlanning,
		StatusSource: "workflow",
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if _, exists := result["notes"]; exists {
		t.Error("expected notes to be omitted from JSON when nil")
	}
	if _, exists := result["context_data"]; exists {
		t.Error("expected context_data to be omitted from JSON when nil")
	}
}

func TestDisplayServiceDeps_HasNoteRepo(t *testing.T) {
	deps := DisplayServiceDeps{
		NoteRepo: nil,
	}
	if deps.NoteRepo != nil {
		t.Error("expected nil NoteRepo")
	}
}

// Test helpers

func testEpicWorkflow() *config.WorkflowConfig {
	return config.DefaultEpicWorkflow()
}

func testFeatureWorkflow() *config.WorkflowConfig {
	return config.DefaultFeatureWorkflow()
}

// ============================================================================
// Mock implementations for DisplayService repository interfaces
// ============================================================================

type mockDisplayEpicRepo struct {
	getByKeyFunc                 func(ctx context.Context, key string) (*models.Epic, error)
	getFeatureProgressDataByEpic func(ctx context.Context, epicID int64) ([]repository.FeatureProgressData, error)
	getFeatureStatusRollupFunc   func(ctx context.Context, epicID int64) (map[string]int, error)
	getTaskStatusRollupFunc      func(ctx context.Context, epicID int64) (map[string]int, error)
}

func (m *mockDisplayEpicRepo) GetByKey(ctx context.Context, key string) (*models.Epic, error) {
	if m.getByKeyFunc != nil {
		return m.getByKeyFunc(ctx, key)
	}
	return nil, fmt.Errorf("GetByKey not implemented in mock")
}

func (m *mockDisplayEpicRepo) GetFeatureProgressDataByEpic(ctx context.Context, epicID int64) ([]repository.FeatureProgressData, error) {
	if m.getFeatureProgressDataByEpic != nil {
		return m.getFeatureProgressDataByEpic(ctx, epicID)
	}
	return nil, nil
}

func (m *mockDisplayEpicRepo) GetFeatureStatusRollup(ctx context.Context, epicID int64) (map[string]int, error) {
	if m.getFeatureStatusRollupFunc != nil {
		return m.getFeatureStatusRollupFunc(ctx, epicID)
	}
	return map[string]int{}, nil
}

func (m *mockDisplayEpicRepo) GetTaskStatusRollup(ctx context.Context, epicID int64) (map[string]int, error) {
	if m.getTaskStatusRollupFunc != nil {
		return m.getTaskStatusRollupFunc(ctx, epicID)
	}
	return map[string]int{}, nil
}

type mockDisplayFeatureRepo struct {
	getByKeyFunc           func(ctx context.Context, key string) (*models.Feature, error)
	listByEpicFunc         func(ctx context.Context, epicID int64) ([]*models.Feature, error)
	getTaskStatusBreakdown func(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error)
}

func (m *mockDisplayFeatureRepo) GetByKey(ctx context.Context, key string) (*models.Feature, error) {
	if m.getByKeyFunc != nil {
		return m.getByKeyFunc(ctx, key)
	}
	return nil, fmt.Errorf("GetByKey not implemented in mock")
}

func (m *mockDisplayFeatureRepo) ListByEpic(ctx context.Context, epicID int64) ([]*models.Feature, error) {
	if m.listByEpicFunc != nil {
		return m.listByEpicFunc(ctx, epicID)
	}
	return nil, nil
}

func (m *mockDisplayFeatureRepo) GetTaskStatusBreakdown(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
	if m.getTaskStatusBreakdown != nil {
		return m.getTaskStatusBreakdown(ctx, featureID)
	}
	return map[models.TaskStatus]int{}, nil
}

type mockDisplayTaskRepo struct {
	getTaskCountForFeatureFunc func(ctx context.Context, featureID int64) (int, error)
	listBlockedTasksByEpicFunc func(ctx context.Context, epicKey string, blockedStatuses []string) ([]*models.Task, error)
	listByFeatureFunc          func(ctx context.Context, featureID int64) ([]*models.Task, error)
}

func (m *mockDisplayTaskRepo) GetTaskCountForFeature(ctx context.Context, featureID int64) (int, error) {
	if m.getTaskCountForFeatureFunc != nil {
		return m.getTaskCountForFeatureFunc(ctx, featureID)
	}
	return 0, nil
}

func (m *mockDisplayTaskRepo) ListBlockedTasksByEpic(ctx context.Context, epicKey string, blockedStatuses []string) ([]*models.Task, error) {
	if m.listBlockedTasksByEpicFunc != nil {
		return m.listBlockedTasksByEpicFunc(ctx, epicKey, blockedStatuses)
	}
	return nil, nil
}

func (m *mockDisplayTaskRepo) ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error) {
	if m.listByFeatureFunc != nil {
		return m.listByFeatureFunc(ctx, featureID)
	}
	return nil, nil
}

type mockDisplayNoteRepo struct {
	getByEntityFunc func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityNote, error)
}

func (m *mockDisplayNoteRepo) GetByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityNote, error) {
	if m.getByEntityFunc != nil {
		return m.getByEntityFunc(ctx, entityType, entityID)
	}
	return nil, nil
}

type mockDocumentRepo struct {
	listForEpicFunc    func(ctx context.Context, epicID int64) ([]*models.Document, error)
	listForFeatureFunc func(ctx context.Context, featureID int64) ([]*models.Document, error)
}

func (m *mockDocumentRepo) ListForEpic(ctx context.Context, epicID int64) ([]*models.Document, error) {
	if m.listForEpicFunc != nil {
		return m.listForEpicFunc(ctx, epicID)
	}
	return []*models.Document{}, nil
}

func (m *mockDocumentRepo) ListForFeature(ctx context.Context, featureID int64) ([]*models.Document, error) {
	if m.listForFeatureFunc != nil {
		return m.listForFeatureFunc(ctx, featureID)
	}
	return []*models.Document{}, nil
}

func (m *mockDocumentRepo) ListForTask(ctx context.Context, taskID int64) ([]*models.Document, error) {
	return []*models.Document{}, nil
}

// ============================================================================
// Mock-based tests for populateEpicPlanningInfo / populateFeaturePlanningInfo
// ============================================================================

func TestPopulateEpicPlanningInfo_FetchesRelatedDocs(t *testing.T) {
	doc1 := &models.Document{ID: 1, Title: "Architecture Doc", FilePath: "docs/architecture.md"}
	doc2 := &models.Document{ID: 2, Title: "Test Plan", FilePath: "docs/test-plan.md"}

	svc := &DisplayService{
		epicWorkflow:    config.DefaultEpicWorkflow(),
		featureWorkflow: config.DefaultFeatureWorkflow(),
		workflowSvc:     workflow.NewService(""),
		deps: DisplayServiceDeps{
			FeatureRepo: &mockDisplayFeatureRepo{
				listByEpicFunc: func(ctx context.Context, epicID int64) ([]*models.Feature, error) {
					return []*models.Feature{}, nil
				},
			},
			TaskRepo: &mockDisplayTaskRepo{},
			NoteRepo: &mockDisplayNoteRepo{
				getByEntityFunc: func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityNote, error) {
					return []*models.EntityNote{}, nil
				},
			},
			DocumentRepo: &mockDocumentRepo{
				listForEpicFunc: func(ctx context.Context, epicID int64) ([]*models.Document, error) {
					return []*models.Document{doc1, doc2}, nil
				},
			},
		},
	}

	info := &EpicDisplayInfo{
		Epic: &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E98"}, Status: "draft"},
		Mode: DisplayModePlanning,
	}

	ctx := context.Background()
	err := svc.populateEpicPlanningInfo(ctx, info)
	if err != nil {
		t.Fatalf("populateEpicPlanningInfo failed: %v", err)
	}

	if len(info.RelatedDocs) != 2 {
		t.Errorf("expected 2 related docs, got %d", len(info.RelatedDocs))
	}
}

func TestPopulateFeaturePlanningInfo_FetchesRelatedDocs(t *testing.T) {
	doc1 := &models.Document{ID: 1, Title: "Feature Spec", FilePath: "docs/feature-spec.md"}

	svc := &DisplayService{
		featureWorkflow: config.DefaultFeatureWorkflow(),
		deps: DisplayServiceDeps{
			TaskRepo: &mockDisplayTaskRepo{
				listByFeatureFunc: func(ctx context.Context, featureID int64) ([]*models.Task, error) {
					return []*models.Task{}, nil
				},
			},
			NoteRepo: &mockDisplayNoteRepo{
				getByEntityFunc: func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityNote, error) {
					return []*models.EntityNote{}, nil
				},
			},
			DocumentRepo: &mockDocumentRepo{
				listForFeatureFunc: func(ctx context.Context, featureID int64) ([]*models.Document, error) {
					return []*models.Document{doc1}, nil
				},
			},
		},
	}

	info := &FeatureDisplayInfo{
		Feature: &models.Feature{BaseEntity: models.BaseEntity{ID: 1, Key: "E98-F01"}, Status: "draft"},
		Mode:    DisplayModePlanning,
	}

	ctx := context.Background()
	err := svc.populateFeaturePlanningInfo(ctx, info)
	if err != nil {
		t.Fatalf("populateFeaturePlanningInfo failed: %v", err)
	}

	if len(info.RelatedDocs) != 1 {
		t.Errorf("expected 1 related doc, got %d", len(info.RelatedDocs))
	}
}

func TestPopulateEpicPlanningInfo_NoDocs_EmptySlice(t *testing.T) {
	svc := &DisplayService{
		epicWorkflow:    config.DefaultEpicWorkflow(),
		featureWorkflow: config.DefaultFeatureWorkflow(),
		deps: DisplayServiceDeps{
			FeatureRepo: &mockDisplayFeatureRepo{
				listByEpicFunc: func(ctx context.Context, epicID int64) ([]*models.Feature, error) {
					return []*models.Feature{}, nil
				},
			},
			TaskRepo: &mockDisplayTaskRepo{},
			NoteRepo: &mockDisplayNoteRepo{},
			DocumentRepo: &mockDocumentRepo{
				listForEpicFunc: func(ctx context.Context, epicID int64) ([]*models.Document, error) {
					return []*models.Document{}, nil
				},
			},
		},
	}

	info := &EpicDisplayInfo{
		Epic: &models.Epic{BaseEntity: models.BaseEntity{ID: 2, Key: "E97"}, Status: "draft"},
		Mode: DisplayModePlanning,
	}

	ctx := context.Background()
	err := svc.populateEpicPlanningInfo(ctx, info)
	if err != nil {
		t.Fatalf("populateEpicPlanningInfo failed: %v", err)
	}

	if info.RelatedDocs == nil {
		t.Error("expected non-nil RelatedDocs (empty slice), got nil")
	}
	if len(info.RelatedDocs) != 0 {
		t.Errorf("expected 0 related docs, got %d", len(info.RelatedDocs))
	}
}

func TestPopulateFeaturePlanningInfo_NoDocs_EmptySlice(t *testing.T) {
	svc := &DisplayService{
		featureWorkflow: config.DefaultFeatureWorkflow(),
		deps: DisplayServiceDeps{
			TaskRepo: &mockDisplayTaskRepo{
				listByFeatureFunc: func(ctx context.Context, featureID int64) ([]*models.Task, error) {
					return []*models.Task{}, nil
				},
			},
			NoteRepo: &mockDisplayNoteRepo{},
			DocumentRepo: &mockDocumentRepo{
				listForFeatureFunc: func(ctx context.Context, featureID int64) ([]*models.Document, error) {
					return []*models.Document{}, nil
				},
			},
		},
	}

	info := &FeatureDisplayInfo{
		Feature: &models.Feature{BaseEntity: models.BaseEntity{ID: 3, Key: "E96-F01"}, Status: "draft"},
		Mode:    DisplayModePlanning,
	}

	ctx := context.Background()
	err := svc.populateFeaturePlanningInfo(ctx, info)
	if err != nil {
		t.Fatalf("populateFeaturePlanningInfo failed: %v", err)
	}

	if info.RelatedDocs == nil {
		t.Error("expected non-nil RelatedDocs (empty slice), got nil")
	}
	if len(info.RelatedDocs) != 0 {
		t.Errorf("expected 0 related docs, got %d", len(info.RelatedDocs))
	}
}

func TestPopulateEpicPlanningInfo_NotesPopulated(t *testing.T) {
	notes := []*models.EntityNote{
		{ID: 1, EntityType: models.EntityTypeEpic, Content: "epic note"},
	}

	svc := &DisplayService{
		epicWorkflow:    config.DefaultEpicWorkflow(),
		featureWorkflow: config.DefaultFeatureWorkflow(),
		deps: DisplayServiceDeps{
			FeatureRepo: &mockDisplayFeatureRepo{
				listByEpicFunc: func(ctx context.Context, epicID int64) ([]*models.Feature, error) {
					return []*models.Feature{}, nil
				},
			},
			TaskRepo: &mockDisplayTaskRepo{},
			NoteRepo: &mockDisplayNoteRepo{
				getByEntityFunc: func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityNote, error) {
					return notes, nil
				},
			},
			DocumentRepo: &mockDocumentRepo{},
		},
	}

	info := &EpicDisplayInfo{
		Epic: &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}, Status: "draft"},
		Mode: DisplayModePlanning,
	}

	ctx := context.Background()
	if err := svc.populateEpicPlanningInfo(ctx, info); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(info.Notes) != 1 {
		t.Errorf("expected 1 note, got %d", len(info.Notes))
	}
	if info.Notes[0].Content != "epic note" {
		t.Errorf("expected note content 'epic note', got %q", info.Notes[0].Content)
	}
}

func TestPopulateFeaturePlanningInfo_NotesPopulated(t *testing.T) {
	notes := []*models.EntityNote{
		{ID: 2, EntityType: models.EntityTypeFeature, Content: "feature note"},
	}

	svc := &DisplayService{
		featureWorkflow: config.DefaultFeatureWorkflow(),
		deps: DisplayServiceDeps{
			TaskRepo: &mockDisplayTaskRepo{
				listByFeatureFunc: func(ctx context.Context, featureID int64) ([]*models.Task, error) {
					return []*models.Task{}, nil
				},
			},
			NoteRepo: &mockDisplayNoteRepo{
				getByEntityFunc: func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityNote, error) {
					return notes, nil
				},
			},
			DocumentRepo: &mockDocumentRepo{},
		},
	}

	info := &FeatureDisplayInfo{
		Feature: &models.Feature{BaseEntity: models.BaseEntity{ID: 1, Key: "E01-F01"}, Status: "draft"},
		Mode:    DisplayModePlanning,
	}

	ctx := context.Background()
	if err := svc.populateFeaturePlanningInfo(ctx, info); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(info.Notes) != 1 {
		t.Errorf("expected 1 note, got %d", len(info.Notes))
	}
	if info.Notes[0].Content != "feature note" {
		t.Errorf("expected note content 'feature note', got %q", info.Notes[0].Content)
	}
}

func TestPopulateEpicPlanningInfo_NilNoteRepo_DoesNotPanic(t *testing.T) {
	svc := &DisplayService{
		epicWorkflow:    config.DefaultEpicWorkflow(),
		featureWorkflow: config.DefaultFeatureWorkflow(),
		deps: DisplayServiceDeps{
			FeatureRepo: &mockDisplayFeatureRepo{
				listByEpicFunc: func(ctx context.Context, epicID int64) ([]*models.Feature, error) {
					return []*models.Feature{}, nil
				},
			},
			TaskRepo:     &mockDisplayTaskRepo{},
			NoteRepo:     nil, // explicitly nil
			DocumentRepo: &mockDocumentRepo{},
		},
	}

	info := &EpicDisplayInfo{
		Epic: &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}, Status: "draft"},
		Mode: DisplayModePlanning,
	}

	ctx := context.Background()
	if err := svc.populateEpicPlanningInfo(ctx, info); err != nil {
		t.Fatalf("unexpected error with nil NoteRepo: %v", err)
	}

	// Notes should be initialized to empty slice even if NoteRepo is nil
	if info.Notes == nil {
		t.Error("expected non-nil Notes even when NoteRepo is nil")
	}
}

func TestPopulateEpicAggregationInfo_BlockedTasks(t *testing.T) {
	blockedTask := &models.Task{BaseEntity: models.BaseEntity{ID: 1, Key: "E01-F01-001"}, Status: "blocked"}

	svc := &DisplayService{
		epicWorkflow:    config.DefaultEpicWorkflow(),
		featureWorkflow: config.DefaultFeatureWorkflow(),
		workflowSvc:     workflow.NewService(""),
		deps: DisplayServiceDeps{
			EpicRepo: &mockDisplayEpicRepo{
				getFeatureProgressDataByEpic: func(ctx context.Context, epicID int64) ([]repository.FeatureProgressData, error) {
					return []repository.FeatureProgressData{}, nil
				},
				getFeatureStatusRollupFunc: func(ctx context.Context, epicID int64) (map[string]int, error) {
					return map[string]int{}, nil
				},
				getTaskStatusRollupFunc: func(ctx context.Context, epicID int64) (map[string]int, error) {
					return map[string]int{"blocked": 1}, nil
				},
			},
			FeatureRepo: &mockDisplayFeatureRepo{
				listByEpicFunc: func(ctx context.Context, epicID int64) ([]*models.Feature, error) {
					return []*models.Feature{}, nil
				},
			},
			TaskRepo: &mockDisplayTaskRepo{
				listBlockedTasksByEpicFunc: func(ctx context.Context, epicKey string, _ []string) ([]*models.Task, error) {
					return []*models.Task{blockedTask}, nil
				},
			},
			DocumentRepo: &mockDocumentRepo{},
		},
	}

	info := &EpicDisplayInfo{
		Epic: &models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01"}, Status: "active"},
		Mode: DisplayModeAggregation,
	}

	ctx := context.Background()
	if err := svc.populateEpicAggregationInfo(ctx, info); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(info.BlockedTasks) != 1 {
		t.Errorf("expected 1 blocked task, got %d", len(info.BlockedTasks))
	}
	if info.BlockedTasks[0].Key != "E01-F01-001" {
		t.Errorf("expected blocked task key 'E01-F01-001', got %q", info.BlockedTasks[0].Key)
	}
}
