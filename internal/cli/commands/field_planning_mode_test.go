package commands

// field_planning_mode_test.go — BUG-2
//
// `shark get <planning-mode-epic|feature> --field status|title|description`
// returned NOT_FOUND because field.go's lookupField does a flat top-level
// JSON-key lookup, but planning-mode JSON nests the entity's scalar fields
// under "epic"/"feature" (EpicDisplayInfo.Epic / FeatureDisplayInfo.Feature).
// Aggregation mode already promotes these to the top level; planning mode did
// not.
//
// These tests exercise the exact marshal/unmarshal/inject code path executed
// by runEpicGet and runFeatureGet for planning-mode entities (mirroring the
// B032 size-field integration tests in size_output_test.go), asserting that
// "key", "title", "status", and "description" land at the TOP LEVEL of the
// resulting map so --field extraction resolves them.

import (
	"encoding/json"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// simulateEpicPlanningModeScalarJSON replicates the planning-mode JSON
// construction path from runEpicGet, including the BUG-2 scalar promotion.
func simulateEpicPlanningModeScalarJSON(t *testing.T, info *services.EpicDisplayInfo, epic *models.Epic) map[string]interface{} {
	t.Helper()
	infoJSON, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal EpicDisplayInfo: %v", err)
	}
	var infoMap map[string]interface{}
	if err := json.Unmarshal(infoJSON, &infoMap); err != nil {
		t.Fatalf("unmarshal EpicDisplayInfo: %v", err)
	}
	promoteEntityScalarFields(infoMap, epic)
	return infoMap
}

// simulateFeaturePlanningModeScalarJSON replicates the planning-mode JSON
// construction path from runFeatureGet, including the BUG-2 scalar promotion.
func simulateFeaturePlanningModeScalarJSON(t *testing.T, info *services.FeatureDisplayInfo, feature *models.Feature) map[string]interface{} {
	t.Helper()
	infoJSON, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal FeatureDisplayInfo: %v", err)
	}
	var infoMap map[string]interface{}
	if err := json.Unmarshal(infoJSON, &infoMap); err != nil {
		t.Fatalf("unmarshal FeatureDisplayInfo: %v", err)
	}
	promoteEntityScalarFields(infoMap, feature)
	return infoMap
}

func TestEpicPlanningModeJSON_ScalarFieldsAtTopLevel(t *testing.T) {
	desc := "Planning mode epic description"
	epic := &models.Epic{
		BaseEntity: models.BaseEntity{
			ID:          1,
			Key:         "E42",
			Title:       "Planning Mode Epic",
			Description: &desc,
		},
		Status: models.EpicStatus("ready_for_decomposition"),
	}
	info := &services.EpicDisplayInfo{
		Epic:         epic,
		Mode:         services.DisplayModePlanning,
		StatusSource: "workflow",
	}

	result := simulateEpicPlanningModeScalarJSON(t, info, epic)

	if got := result["key"]; got != "E42" {
		t.Errorf("expected top-level key=E42, got %v", got)
	}
	if got := result["title"]; got != "Planning Mode Epic" {
		t.Errorf("expected top-level title, got %v", got)
	}
	if got := result["status"]; got != "ready_for_decomposition" {
		t.Errorf("expected top-level status, got %v", got)
	}
	if got := result["description"]; got != desc {
		t.Errorf("expected top-level description=%q, got %v", desc, got)
	}
	// Nested "epic" key must still be present — aggregation-mode consumers
	// and existing callers rely on it.
	if _, ok := result["epic"]; !ok {
		t.Error("expected nested 'epic' key to still be present")
	}
}

func TestEpicPlanningModeJSON_DescriptionEmptyStringWhenNil(t *testing.T) {
	epic := &models.Epic{
		BaseEntity: models.BaseEntity{ID: 2, Key: "E43", Title: "No Description Epic"},
		Status:     models.EpicStatus("draft"),
	}
	info := &services.EpicDisplayInfo{
		Epic:         epic,
		Mode:         services.DisplayModePlanning,
		StatusSource: "workflow",
	}

	result := simulateEpicPlanningModeScalarJSON(t, info, epic)

	// The key must still be present (as "") so `--field description` resolves
	// instead of 404ing — omitting the key entirely reintroduces BUG-2.
	got, ok := result["description"]
	if !ok {
		t.Fatal("expected top-level 'description' key to be present (as \"\") when Description is nil")
	}
	if got != "" {
		t.Errorf("expected top-level description=\"\", got %v", got)
	}
}

func TestFeaturePlanningModeJSON_ScalarFieldsAtTopLevel(t *testing.T) {
	desc := "Planning mode feature description"
	feature := &models.Feature{
		BaseEntity: models.BaseEntity{
			ID:          1,
			Key:         "E42-F01",
			Title:       "Planning Mode Feature",
			Description: &desc,
		},
		Status: models.FeatureStatus("ready_for_refinement_ba"),
	}
	info := &services.FeatureDisplayInfo{
		Feature:      feature,
		Mode:         services.DisplayModePlanning,
		StatusSource: "workflow",
	}

	result := simulateFeaturePlanningModeScalarJSON(t, info, feature)

	if got := result["key"]; got != "E42-F01" {
		t.Errorf("expected top-level key=E42-F01, got %v", got)
	}
	if got := result["title"]; got != "Planning Mode Feature" {
		t.Errorf("expected top-level title, got %v", got)
	}
	if got := result["status"]; got != "ready_for_refinement_ba" {
		t.Errorf("expected top-level status, got %v", got)
	}
	if got := result["description"]; got != desc {
		t.Errorf("expected top-level description=%q, got %v", desc, got)
	}
	if _, ok := result["feature"]; !ok {
		t.Error("expected nested 'feature' key to still be present")
	}
}

func TestFeaturePlanningModeJSON_DescriptionEmptyStringWhenNil(t *testing.T) {
	feature := &models.Feature{
		BaseEntity: models.BaseEntity{ID: 2, Key: "E42-F02", Title: "No Description Feature"},
		Status:     models.FeatureStatus("draft"),
	}
	info := &services.FeatureDisplayInfo{
		Feature:      feature,
		Mode:         services.DisplayModePlanning,
		StatusSource: "workflow",
	}

	result := simulateFeaturePlanningModeScalarJSON(t, info, feature)

	got, ok := result["description"]
	if !ok {
		t.Fatal("expected top-level 'description' key to be present (as \"\") when Description is nil")
	}
	if got != "" {
		t.Errorf("expected top-level description=\"\", got %v", got)
	}
}

// TestPromoteEntityScalarFields_NilEntityIsNoop guards against a nil
// interface value causing a nil-pointer panic inside a JSON-output hot path.
func TestPromoteEntityScalarFields_NilEntityIsNoop(t *testing.T) {
	result := map[string]interface{}{"existing": "value"}
	promoteEntityScalarFields(result, nil)

	if len(result) != 1 {
		t.Errorf("expected promoteEntityScalarFields(nil) to be a no-op, got %v", result)
	}
}

// TestPromoteEntityScalarFields_TypedNilEntityIsNoop guards against a typed-nil
// concrete pointer (e.g. `var epic *models.Epic`), which satisfies the
// models.Entity interface without being == nil — a plain `entity == nil`
// check does not catch this case and would panic on GetKey().
func TestPromoteEntityScalarFields_TypedNilEntityIsNoop(t *testing.T) {
	result := map[string]interface{}{"existing": "value"}
	var epic *models.Epic
	promoteEntityScalarFields(result, epic)

	if len(result) != 1 {
		t.Errorf("expected promoteEntityScalarFields(typed-nil) to be a no-op, got %v", result)
	}
}
