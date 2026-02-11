package services

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
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

// Test helpers

func testEpicWorkflow() *config.WorkflowConfig {
	return config.DefaultEpicWorkflow()
}

func testFeatureWorkflow() *config.WorkflowConfig {
	return config.DefaultFeatureWorkflow()
}
