package models

import (
	"encoding/json"
	"testing"
)

func TestContextData_RelatedFeatures(t *testing.T) {
	cd := &ContextData{
		RelatedFeatures: []string{"E07-F01", "E07-F05", "E10-F03"},
	}

	// Test ToJSON
	jsonStr, err := cd.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() failed: %v", err)
	}

	// Verify JSON contains related_features
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if _, ok := parsed["related_features"]; !ok {
		t.Error("JSON missing 'related_features' field")
	}

	// Test FromJSON
	cd2, err := FromJSON(jsonStr)
	if err != nil {
		t.Fatalf("FromJSON() failed: %v", err)
	}

	if len(cd2.RelatedFeatures) != 3 {
		t.Errorf("Expected 3 related features, got %d", len(cd2.RelatedFeatures))
	}

	if cd2.RelatedFeatures[0] != "E07-F01" {
		t.Errorf("Expected first feature 'E07-F01', got '%s'", cd2.RelatedFeatures[0])
	}
}

func TestContextData_RelatedEpics(t *testing.T) {
	cd := &ContextData{
		RelatedEpics: []string{"E01", "E03", "E07"},
	}

	// Test ToJSON
	jsonStr, err := cd.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() failed: %v", err)
	}

	// Verify JSON contains related_epics
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if _, ok := parsed["related_epics"]; !ok {
		t.Error("JSON missing 'related_epics' field")
	}

	// Test FromJSON
	cd2, err := FromJSON(jsonStr)
	if err != nil {
		t.Fatalf("FromJSON() failed: %v", err)
	}

	if len(cd2.RelatedEpics) != 3 {
		t.Errorf("Expected 3 related epics, got %d", len(cd2.RelatedEpics))
	}

	if cd2.RelatedEpics[1] != "E03" {
		t.Errorf("Expected second epic 'E03', got '%s'", cd2.RelatedEpics[1])
	}
}

func TestContextData_AllRelatedFields(t *testing.T) {
	cd := &ContextData{
		RelatedTasks:    []string{"E07-F01-001", "E10-F05-002"},
		RelatedFeatures: []string{"E07-F21", "E07-F28"},
		RelatedEpics:    []string{"E01", "E05"},
	}

	jsonStr, err := cd.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() failed: %v", err)
	}

	cd2, err := FromJSON(jsonStr)
	if err != nil {
		t.Fatalf("FromJSON() failed: %v", err)
	}

	if len(cd2.RelatedTasks) != 2 {
		t.Errorf("Expected 2 related tasks, got %d", len(cd2.RelatedTasks))
	}

	if len(cd2.RelatedFeatures) != 2 {
		t.Errorf("Expected 2 related features, got %d", len(cd2.RelatedFeatures))
	}

	if len(cd2.RelatedEpics) != 2 {
		t.Errorf("Expected 2 related epics, got %d", len(cd2.RelatedEpics))
	}
}

func TestContextData_MergeRelatedFeatures(t *testing.T) {
	cd1 := &ContextData{
		RelatedFeatures: []string{"E07-F01"},
	}

	cd2 := &ContextData{
		RelatedFeatures: []string{"E07-F05", "E10-F03"},
	}

	cd1.Merge(cd2)

	if len(cd1.RelatedFeatures) != 2 {
		t.Errorf("Expected 2 related features after merge, got %d", len(cd1.RelatedFeatures))
	}

	if cd1.RelatedFeatures[0] != "E07-F05" {
		t.Errorf("Expected merged features to replace original")
	}
}

func TestContextData_MergeRelatedEpics(t *testing.T) {
	cd1 := &ContextData{
		RelatedEpics: []string{"E01"},
	}

	cd2 := &ContextData{
		RelatedEpics: []string{"E03", "E07"},
	}

	cd1.Merge(cd2)

	if len(cd1.RelatedEpics) != 2 {
		t.Errorf("Expected 2 related epics after merge, got %d", len(cd1.RelatedEpics))
	}

	if cd1.RelatedEpics[1] != "E07" {
		t.Errorf("Expected merged epics to replace original")
	}
}

func TestContextData_EmptyRelatedFields(t *testing.T) {
	cd := &ContextData{}

	jsonStr, err := cd.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() failed: %v", err)
	}

	// Verify empty arrays are omitted from JSON (omitempty)
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if _, ok := parsed["related_features"]; ok {
		t.Error("Empty related_features should be omitted from JSON")
	}

	if _, ok := parsed["related_epics"]; ok {
		t.Error("Empty related_epics should be omitted from JSON")
	}
}

func TestContextData_BackwardCompatibility(t *testing.T) {
	// JSON without new fields should parse without error
	oldJSON := `{"related_tasks": ["E07-F01-001"]}`

	cd, err := FromJSON(oldJSON)
	if err != nil {
		t.Fatalf("FromJSON() failed for backward compatibility: %v", err)
	}

	if len(cd.RelatedTasks) != 1 {
		t.Errorf("Expected 1 related task, got %d", len(cd.RelatedTasks))
	}

	if len(cd.RelatedFeatures) != 0 {
		t.Errorf("Expected 0 related features, got %d", len(cd.RelatedFeatures))
	}

	if len(cd.RelatedEpics) != 0 {
		t.Errorf("Expected 0 related epics, got %d", len(cd.RelatedEpics))
	}
}
