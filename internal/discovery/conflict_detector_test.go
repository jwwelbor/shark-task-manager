package discovery

import (
	"testing"
)

func TestConflictDetector_Detect_EpicIndexOnly(t *testing.T) {
	detector := NewConflictDetector()

	// Arrange: Epic in index but not in folders
	indexEpics := []DiscoveredEpic{
		{Key: "E04", Title: "Epic Four", Source: SourceIndex},
	}
	folderEpics := []DiscoveredEpic{}
	indexFeatures := []DiscoveredFeature{}
	folderFeatures := []DiscoveredFeature{}

	// Act
	conflicts := detector.Detect(indexEpics, folderEpics, indexFeatures, folderFeatures)

	// Assert
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}

	conflict := conflicts[0]
	if conflict.Type != ConflictTypeEpicIndexOnly {
		t.Errorf("expected conflict type %s, got %s", ConflictTypeEpicIndexOnly, conflict.Type)
	}
	if conflict.Key != "E04" {
		t.Errorf("expected conflict key E04, got %s", conflict.Key)
	}
	if conflict.Suggestion == "" {
		t.Error("expected non-empty suggestion")
	}
}

func TestConflictDetector_Detect_EpicFolderOnly(t *testing.T) {
	detector := NewConflictDetector()

	// Arrange: Epic in folder but not in index
	indexEpics := []DiscoveredEpic{}
	folderEpics := []DiscoveredEpic{
		{Key: "E05", Title: "Epic Five", Source: SourceFolder},
	}
	indexFeatures := []DiscoveredFeature{}
	folderFeatures := []DiscoveredFeature{}

	// Act
	conflicts := detector.Detect(indexEpics, folderEpics, indexFeatures, folderFeatures)

	// Assert
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}

	conflict := conflicts[0]
	if conflict.Type != ConflictTypeEpicFolderOnly {
		t.Errorf("expected conflict type %s, got %s", ConflictTypeEpicFolderOnly, conflict.Type)
	}
	if conflict.Key != "E05" {
		t.Errorf("expected conflict key E05, got %s", conflict.Key)
	}
	if conflict.Suggestion == "" {
		t.Error("expected non-empty suggestion")
	}
}

func TestConflictDetector_Detect_FeatureIndexOnly(t *testing.T) {
	detector := NewConflictDetector()

	// Arrange: Feature in index but not in folder
	indexEpics := []DiscoveredEpic{}
	folderEpics := []DiscoveredEpic{}
	indexFeatures := []DiscoveredFeature{
		{Key: "E04-F07", EpicKey: "E04", Title: "Feature Seven", Source: SourceIndex},
	}
	folderFeatures := []DiscoveredFeature{}

	// Act
	conflicts := detector.Detect(indexEpics, folderEpics, indexFeatures, folderFeatures)

	// Assert
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}

	conflict := conflicts[0]
	if conflict.Type != ConflictTypeFeatureIndexOnly {
		t.Errorf("expected conflict type %s, got %s", ConflictTypeFeatureIndexOnly, conflict.Type)
	}
	if conflict.Key != "E04-F07" {
		t.Errorf("expected conflict key E04-F07, got %s", conflict.Key)
	}
	if conflict.Suggestion == "" {
		t.Error("expected non-empty suggestion")
	}
}

func TestConflictDetector_Detect_FeatureFolderOnly(t *testing.T) {
	detector := NewConflictDetector()

	// Arrange: Feature in folder but not in index
	indexEpics := []DiscoveredEpic{}
	folderEpics := []DiscoveredEpic{}
	indexFeatures := []DiscoveredFeature{}
	folderFeatures := []DiscoveredFeature{
		{Key: "E05-F02", EpicKey: "E05", Title: "Feature Two", Source: SourceFolder},
	}

	// Act
	conflicts := detector.Detect(indexEpics, folderEpics, indexFeatures, folderFeatures)

	// Assert
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}

	conflict := conflicts[0]
	if conflict.Type != ConflictTypeFeatureFolderOnly {
		t.Errorf("expected conflict type %s, got %s", ConflictTypeFeatureFolderOnly, conflict.Type)
	}
	if conflict.Key != "E05-F02" {
		t.Errorf("expected conflict key E05-F02, got %s", conflict.Key)
	}
	if conflict.Suggestion == "" {
		t.Error("expected non-empty suggestion")
	}
}

func TestConflictDetector_Detect_FeatureRelationshipMismatch(t *testing.T) {
	detector := NewConflictDetector()

	// Arrange: Feature with different parent epic in index vs folder
	indexEpics := []DiscoveredEpic{}
	folderEpics := []DiscoveredEpic{}
	indexFeatures := []DiscoveredFeature{
		{Key: "E04-F07", EpicKey: "E04", Title: "Feature Seven", Source: SourceIndex},
	}
	folderFeatures := []DiscoveredFeature{
		{Key: "E04-F07", EpicKey: "E05", Title: "Feature Seven", Source: SourceFolder},
	}

	// Act
	conflicts := detector.Detect(indexEpics, folderEpics, indexFeatures, folderFeatures)

	// Assert
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}

	conflict := conflicts[0]
	if conflict.Type != ConflictTypeRelationshipMismatch {
		t.Errorf("expected conflict type %s, got %s", ConflictTypeRelationshipMismatch, conflict.Type)
	}
	if conflict.Key != "E04-F07" {
		t.Errorf("expected conflict key E04-F07, got %s", conflict.Key)
	}
	if conflict.Suggestion == "" {
		t.Error("expected non-empty suggestion")
	}
}

func TestConflictDetector_Detect_MultipleConflicts(t *testing.T) {
	detector := NewConflictDetector()

	// Arrange: Multiple conflict types in same discovery
	indexEpics := []DiscoveredEpic{
		{Key: "E04", Title: "Epic Four", Source: SourceIndex},
	}
	folderEpics := []DiscoveredEpic{
		{Key: "E05", Title: "Epic Five", Source: SourceFolder},
	}
	indexFeatures := []DiscoveredFeature{
		{Key: "E04-F07", EpicKey: "E04", Title: "Feature Seven", Source: SourceIndex},
	}
	folderFeatures := []DiscoveredFeature{
		{Key: "E05-F02", EpicKey: "E05", Title: "Feature Two", Source: SourceFolder},
	}

	// Act
	conflicts := detector.Detect(indexEpics, folderEpics, indexFeatures, folderFeatures)

	// Assert
	// Expected conflicts:
	// 1. E04 in index but not in folders
	// 2. E05 in folders but not in index
	// 3. E04-F07 in index but not in folders
	// 4. E05-F02 in folders but not in index
	if len(conflicts) != 4 {
		t.Fatalf("expected 4 conflicts, got %d", len(conflicts))
	}

	// Verify we have one of each type
	typeCount := make(map[ConflictType]int)
	for _, conflict := range conflicts {
		typeCount[conflict.Type]++
	}

	expectedTypes := []ConflictType{
		ConflictTypeEpicIndexOnly,
		ConflictTypeEpicFolderOnly,
		ConflictTypeFeatureIndexOnly,
		ConflictTypeFeatureFolderOnly,
	}

	for _, expectedType := range expectedTypes {
		if typeCount[expectedType] != 1 {
			t.Errorf("expected 1 conflict of type %s, got %d", expectedType, typeCount[expectedType])
		}
	}
}

func TestConflictDetector_Detect_NoConflicts(t *testing.T) {
	detector := NewConflictDetector()

	// Arrange: Matching epics and features in both sources
	indexEpics := []DiscoveredEpic{
		{Key: "E04", Title: "Epic Four", Source: SourceIndex},
	}
	folderEpics := []DiscoveredEpic{
		{Key: "E04", Title: "Epic Four", Source: SourceFolder},
	}
	indexFeatures := []DiscoveredFeature{
		{Key: "E04-F07", EpicKey: "E04", Title: "Feature Seven", Source: SourceIndex},
	}
	folderFeatures := []DiscoveredFeature{
		{Key: "E04-F07", EpicKey: "E04", Title: "Feature Seven", Source: SourceFolder},
	}

	// Act
	conflicts := detector.Detect(indexEpics, folderEpics, indexFeatures, folderFeatures)

	// Assert
	if len(conflicts) != 0 {
		t.Fatalf("expected 0 conflicts, got %d", len(conflicts))
	}
}

func TestConflictDetector_Detect_EmptyInputs(t *testing.T) {
	detector := NewConflictDetector()

	// Arrange: All empty slices
	indexEpics := []DiscoveredEpic{}
	folderEpics := []DiscoveredEpic{}
	indexFeatures := []DiscoveredFeature{}
	folderFeatures := []DiscoveredFeature{}

	// Act
	conflicts := detector.Detect(indexEpics, folderEpics, indexFeatures, folderFeatures)

	// Assert
	if len(conflicts) != 0 {
		t.Fatalf("expected 0 conflicts, got %d", len(conflicts))
	}
}

func TestConflictDetector_Detect_ConflictSuggestionsAreActionable(t *testing.T) {
	detector := NewConflictDetector()

	testCases := []struct {
		name                 string
		indexEpics           []DiscoveredEpic
		folderEpics          []DiscoveredEpic
		indexFeatures        []DiscoveredFeature
		folderFeatures       []DiscoveredFeature
		expectedConflictType ConflictType
		mustContain          string
	}{
		{
			name:                 "Epic index only suggests creating folder",
			indexEpics:           []DiscoveredEpic{{Key: "E04", Title: "Epic Four", Source: SourceIndex}},
			folderEpics:          []DiscoveredEpic{},
			indexFeatures:        []DiscoveredFeature{},
			folderFeatures:       []DiscoveredFeature{},
			expectedConflictType: ConflictTypeEpicIndexOnly,
			mustContain:          "folder",
		},
		{
			name:                 "Epic folder only suggests adding to index",
			indexEpics:           []DiscoveredEpic{},
			folderEpics:          []DiscoveredEpic{{Key: "E05", Title: "Epic Five", Source: SourceFolder}},
			indexFeatures:        []DiscoveredFeature{},
			folderFeatures:       []DiscoveredFeature{},
			expectedConflictType: ConflictTypeEpicFolderOnly,
			mustContain:          "epic-index.md",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			conflicts := detector.Detect(tc.indexEpics, tc.folderEpics, tc.indexFeatures, tc.folderFeatures)

			// Assert
			if len(conflicts) != 1 {
				t.Fatalf("expected 1 conflict, got %d", len(conflicts))
			}

			conflict := conflicts[0]
			if conflict.Type != tc.expectedConflictType {
				t.Errorf("expected conflict type %s, got %s", tc.expectedConflictType, conflict.Type)
			}

			// Check suggestion contains expected keyword
			if !contains(conflict.Suggestion, tc.mustContain) {
				t.Errorf("expected suggestion to contain '%s', got: %s", tc.mustContain, conflict.Suggestion)
			}
		})
	}
}

// Helper function to check if string contains substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(findSubstring(s, substr) != -1))
}

func findSubstring(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
