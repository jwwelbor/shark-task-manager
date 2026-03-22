package services

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
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

// setupTestDisplayService creates a DisplayService backed by a temporary SQLite database.
// Returns the service, a cleanup function, and the raw *sql.DB for seeding test data.
func setupTestDisplayService(t *testing.T) (*DisplayService, func(), *repository.DB) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test-display.db")

	sqlDB, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("failed to init test db: %v", err)
	}

	repoDB := repository.NewDB(sqlDB)

	epicWf := config.DefaultEpicWorkflow()
	featureWf := config.DefaultFeatureWorkflow()

	svc := &DisplayService{
		deps: DisplayServiceDeps{
			EpicRepo:     repository.NewEpicRepository(repoDB),
			FeatureRepo:  repository.NewFeatureRepository(repoDB),
			TaskRepo:     repository.NewTaskRepositoryWithWorkflow(repoDB, nil),
			DocumentRepo: repository.NewPolymorphicDocRepoAdapter(repository.NewEntityDocumentRepository(repoDB)),
			NoteRepo:     repository.NewEntityNoteRepository(repoDB),
		},
		epicWorkflow:    epicWf,
		featureWorkflow: featureWf,
	}

	cleanup := func() {
		sqlDB.Close()
		os.Remove(dbPath)
	}

	return svc, cleanup, repoDB
}

// seedEpicWithDocs creates an epic in "draft" (planning) status and links documents to it.
// Returns the epic ID and document IDs.
func seedEpicWithDocs(t *testing.T, repoDB *repository.DB) (int64, []int64) {
	t.Helper()
	ctx := context.Background()

	// Create epic in planning status (draft)
	result, err := repoDB.ExecContext(ctx, `
		INSERT INTO epics (key, title, description, status, priority)
		VALUES ('E98', 'Test Planning Epic', 'Epic in draft status', 'draft', 'high')
	`)
	if err != nil {
		t.Fatalf("failed to insert epic: %v", err)
	}
	epicID, _ := result.LastInsertId()

	// Create documents and link to epic
	docRepo := repository.NewDocumentRepository(repoDB)
	entityDocRepo := repository.NewEntityDocumentRepository(repoDB)
	doc1, err := docRepo.CreateOrGet(ctx, "Architecture Doc", "docs/architecture.md")
	if err != nil {
		t.Fatalf("failed to create doc1: %v", err)
	}
	doc2, err := docRepo.CreateOrGet(ctx, "Test Plan", "docs/test-plan.md")
	if err != nil {
		t.Fatalf("failed to create doc2: %v", err)
	}

	if err := entityDocRepo.Link(ctx, models.EntityTypeEpic, epicID, doc1.ID, ""); err != nil {
		t.Fatalf("failed to link doc1 to epic: %v", err)
	}
	if err := entityDocRepo.Link(ctx, models.EntityTypeEpic, epicID, doc2.ID, ""); err != nil {
		t.Fatalf("failed to link doc2 to epic: %v", err)
	}

	return epicID, []int64{doc1.ID, doc2.ID}
}

// seedFeatureWithDocs creates a feature in "draft" (planning) status and links documents to it.
// Returns the feature ID and document IDs.
func seedFeatureWithDocs(t *testing.T, repoDB *repository.DB, epicID int64) (int64, []int64) {
	t.Helper()
	ctx := context.Background()

	// Create feature in planning status (draft)
	result, err := repoDB.ExecContext(ctx, `
		INSERT INTO features (epic_id, key, title, slug, description, status)
		VALUES (?, 'E98-F01', 'Test Planning Feature', 'test-planning-feature', 'Feature in draft status', 'draft')
	`, epicID)
	if err != nil {
		t.Fatalf("failed to insert feature: %v", err)
	}
	featureID, _ := result.LastInsertId()

	// Create documents and link to feature
	docRepo := repository.NewDocumentRepository(repoDB)
	entityDocRepo := repository.NewEntityDocumentRepository(repoDB)
	doc1, err := docRepo.CreateOrGet(ctx, "Feature Spec", "docs/feature-spec.md")
	if err != nil {
		t.Fatalf("failed to create doc1: %v", err)
	}

	if err := entityDocRepo.Link(ctx, models.EntityTypeFeature, featureID, doc1.ID, ""); err != nil {
		t.Fatalf("failed to link doc1 to feature: %v", err)
	}

	return featureID, []int64{doc1.ID}
}

func TestPopulateEpicPlanningInfo_FetchesRelatedDocs(t *testing.T) {
	svc, cleanup, repoDB := setupTestDisplayService(t)
	defer cleanup()

	epicID, _ := seedEpicWithDocs(t, repoDB)

	info := &EpicDisplayInfo{
		Epic: &models.Epic{BaseEntity: models.BaseEntity{ID: epicID,
			Key: "E98"}, Status: "draft",
		},
		Mode: DisplayModePlanning,
	}

	ctx := context.Background()
	err := svc.populateEpicPlanningInfo(ctx, info)
	if err != nil {
		t.Fatalf("populateEpicPlanningInfo failed: %v", err)
	}

	// Verify RelatedDocs is populated (this is the bug fix assertion)
	if len(info.RelatedDocs) != 2 {
		t.Errorf("expected 2 related docs, got %d", len(info.RelatedDocs))
	}
}

func TestPopulateFeaturePlanningInfo_FetchesRelatedDocs(t *testing.T) {
	svc, cleanup, repoDB := setupTestDisplayService(t)
	defer cleanup()

	epicID, _ := seedEpicWithDocs(t, repoDB)
	featureID, _ := seedFeatureWithDocs(t, repoDB, epicID)

	info := &FeatureDisplayInfo{
		Feature: &models.Feature{BaseEntity: models.BaseEntity{ID: featureID,
			Key: "E98-F01"}, Status: "draft",
		},
		Mode: DisplayModePlanning,
	}

	ctx := context.Background()
	err := svc.populateFeaturePlanningInfo(ctx, info)
	if err != nil {
		t.Fatalf("populateFeaturePlanningInfo failed: %v", err)
	}

	// Verify RelatedDocs is populated (this is the bug fix assertion)
	if len(info.RelatedDocs) != 1 {
		t.Errorf("expected 1 related doc, got %d", len(info.RelatedDocs))
	}
}

func TestPopulateEpicPlanningInfo_NoDocs_EmptySlice(t *testing.T) {
	svc, cleanup, repoDB := setupTestDisplayService(t)
	defer cleanup()

	ctx := context.Background()
	// Create epic with no linked docs
	result, err := repoDB.ExecContext(ctx, `
		INSERT INTO epics (key, title, description, status, priority)
		VALUES ('E97', 'No Docs Epic', 'Epic with no docs', 'draft', 'medium')
	`)
	if err != nil {
		t.Fatalf("failed to insert epic: %v", err)
	}
	epicID, _ := result.LastInsertId()

	info := &EpicDisplayInfo{
		Epic: &models.Epic{BaseEntity: models.BaseEntity{ID: epicID,
			Key: "E97"}, Status: "draft",
		},
		Mode: DisplayModePlanning,
	}

	err = svc.populateEpicPlanningInfo(ctx, info)
	if err != nil {
		t.Fatalf("populateEpicPlanningInfo failed: %v", err)
	}

	// Should be empty slice (not nil) for clean JSON serialization
	if info.RelatedDocs == nil {
		t.Error("expected non-nil RelatedDocs (empty slice), got nil")
	}
	if len(info.RelatedDocs) != 0 {
		t.Errorf("expected 0 related docs, got %d", len(info.RelatedDocs))
	}
}

func TestPopulateFeaturePlanningInfo_NoDocs_EmptySlice(t *testing.T) {
	svc, cleanup, repoDB := setupTestDisplayService(t)
	defer cleanup()

	ctx := context.Background()
	// Create epic first (needed for feature FK)
	result, err := repoDB.ExecContext(ctx, `
		INSERT INTO epics (key, title, description, status, priority)
		VALUES ('E96', 'Parent Epic', 'Parent', 'draft', 'medium')
	`)
	if err != nil {
		t.Fatalf("failed to insert epic: %v", err)
	}
	epicID, _ := result.LastInsertId()

	// Create feature with no linked docs
	result, err = repoDB.ExecContext(ctx, `
		INSERT INTO features (epic_id, key, title, slug, description, status)
		VALUES (?, 'E96-F01', 'No Docs Feature', 'no-docs-feature', 'Feature with no docs', 'draft')
	`, epicID)
	if err != nil {
		t.Fatalf("failed to insert feature: %v", err)
	}
	featureID, _ := result.LastInsertId()

	info := &FeatureDisplayInfo{
		Feature: &models.Feature{BaseEntity: models.BaseEntity{ID: featureID,
			Key: "E96-F01"}, Status: "draft",
		},
		Mode: DisplayModePlanning,
	}

	err = svc.populateFeaturePlanningInfo(ctx, info)
	if err != nil {
		t.Fatalf("populateFeaturePlanningInfo failed: %v", err)
	}

	// Should be empty slice (not nil) for clean JSON serialization
	if info.RelatedDocs == nil {
		t.Error("expected non-nil RelatedDocs (empty slice), got nil")
	}
	if len(info.RelatedDocs) != 0 {
		t.Errorf("expected 0 related docs, got %d", len(info.RelatedDocs))
	}
}
