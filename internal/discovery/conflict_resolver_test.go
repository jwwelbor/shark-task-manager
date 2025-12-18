package discovery

import (
	"testing"
)

func TestConflictResolver_Resolve_IndexPrecedence_OnlyIndexItems(t *testing.T) {
	resolver := NewConflictResolver()

	// Arrange: Items only from index (should succeed)
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
	conflicts := []Conflict{}

	// Act
	resultEpics, resultFeatures, warnings, err := resolver.Resolve(
		indexEpics, folderEpics, indexFeatures, folderFeatures,
		conflicts, ConflictStrategyIndexPrecedence)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(resultEpics) != 1 {
		t.Fatalf("expected 1 epic, got %d", len(resultEpics))
	}
	if resultEpics[0].Key != "E04" {
		t.Errorf("expected epic E04, got %s", resultEpics[0].Key)
	}

	if len(resultFeatures) != 1 {
		t.Fatalf("expected 1 feature, got %d", len(resultFeatures))
	}
	if resultFeatures[0].Key != "E04-F07" {
		t.Errorf("expected feature E04-F07, got %s", resultFeatures[0].Key)
	}

	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %d", len(warnings))
	}
}

func TestConflictResolver_Resolve_IndexPrecedence_FailsOnIndexOnlyEpic(t *testing.T) {
	resolver := NewConflictResolver()

	// Arrange: Epic in index but not in folder (should fail with index-precedence)
	indexEpics := []DiscoveredEpic{
		{Key: "E04", Title: "Epic Four", Source: SourceIndex},
	}
	folderEpics := []DiscoveredEpic{}
	indexFeatures := []DiscoveredFeature{}
	folderFeatures := []DiscoveredFeature{}
	conflicts := []Conflict{
		{Type: ConflictTypeEpicIndexOnly, Key: "E04"},
	}

	// Act
	_, _, _, err := resolver.Resolve(
		indexEpics, folderEpics, indexFeatures, folderFeatures,
		conflicts, ConflictStrategyIndexPrecedence)

	// Assert
	if err == nil {
		t.Fatal("expected error for index-only epic with index-precedence strategy")
	}
}

func TestConflictResolver_Resolve_IndexPrecedence_WarnsOnFolderOnlyItems(t *testing.T) {
	resolver := NewConflictResolver()

	// Arrange: Folder-only items should be skipped with warning
	indexEpics := []DiscoveredEpic{
		{Key: "E04", Title: "Epic Four", Source: SourceIndex},
	}
	folderEpics := []DiscoveredEpic{
		{Key: "E04", Title: "Epic Four", Source: SourceFolder},
		{Key: "E05", Title: "Epic Five", Source: SourceFolder},
	}
	indexFeatures := []DiscoveredFeature{}
	folderFeatures := []DiscoveredFeature{
		{Key: "E05-F02", EpicKey: "E05", Title: "Feature Two", Source: SourceFolder},
	}
	conflicts := []Conflict{
		{Type: ConflictTypeEpicFolderOnly, Key: "E05"},
		{Type: ConflictTypeFeatureFolderOnly, Key: "E05-F02"},
	}

	// Act
	resultEpics, resultFeatures, warnings, err := resolver.Resolve(
		indexEpics, folderEpics, indexFeatures, folderFeatures,
		conflicts, ConflictStrategyIndexPrecedence)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Only E04 should be included (from index)
	if len(resultEpics) != 1 {
		t.Fatalf("expected 1 epic, got %d", len(resultEpics))
	}
	if resultEpics[0].Key != "E04" {
		t.Errorf("expected epic E04, got %s", resultEpics[0].Key)
	}

	// No features should be included (none in index)
	if len(resultFeatures) != 0 {
		t.Fatalf("expected 0 features, got %d", len(resultFeatures))
	}

	// Should have warnings about skipped items
	if len(warnings) != 2 {
		t.Fatalf("expected 2 warnings, got %d", len(warnings))
	}
}

func TestConflictResolver_Resolve_FolderPrecedence_OnlyFolderItems(t *testing.T) {
	resolver := NewConflictResolver()

	// Arrange: Items from folders should be used
	indexEpics := []DiscoveredEpic{}
	folderEpics := []DiscoveredEpic{
		{Key: "E05", Title: "Epic Five", Source: SourceFolder},
	}
	indexFeatures := []DiscoveredFeature{}
	folderFeatures := []DiscoveredFeature{
		{Key: "E05-F02", EpicKey: "E05", Title: "Feature Two", Source: SourceFolder},
	}
	conflicts := []Conflict{
		{Type: ConflictTypeEpicFolderOnly, Key: "E05"},
		{Type: ConflictTypeFeatureFolderOnly, Key: "E05-F02"},
	}

	// Act
	resultEpics, resultFeatures, warnings, err := resolver.Resolve(
		indexEpics, folderEpics, indexFeatures, folderFeatures,
		conflicts, ConflictStrategyFolderPrecedence)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(resultEpics) != 1 {
		t.Fatalf("expected 1 epic, got %d", len(resultEpics))
	}
	if resultEpics[0].Key != "E05" {
		t.Errorf("expected epic E05, got %s", resultEpics[0].Key)
	}

	if len(resultFeatures) != 1 {
		t.Fatalf("expected 1 feature, got %d", len(resultFeatures))
	}
	if resultFeatures[0].Key != "E05-F02" {
		t.Errorf("expected feature E05-F02, got %s", resultFeatures[0].Key)
	}

	// Should warn about index-only items being skipped
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %d", len(warnings))
	}
}

func TestConflictResolver_Resolve_FolderPrecedence_WarnsOnIndexOnlyItems(t *testing.T) {
	resolver := NewConflictResolver()

	// Arrange: Index-only items should be skipped with warning
	indexEpics := []DiscoveredEpic{
		{Key: "E04", Title: "Epic Four", Source: SourceIndex},
	}
	folderEpics := []DiscoveredEpic{
		{Key: "E05", Title: "Epic Five", Source: SourceFolder},
	}
	indexFeatures := []DiscoveredFeature{
		{Key: "E04-F07", EpicKey: "E04", Title: "Feature Seven", Source: SourceIndex},
	}
	folderFeatures := []DiscoveredFeature{}
	conflicts := []Conflict{
		{Type: ConflictTypeEpicIndexOnly, Key: "E04"},
		{Type: ConflictTypeFeatureIndexOnly, Key: "E04-F07"},
	}

	// Act
	resultEpics, resultFeatures, warnings, err := resolver.Resolve(
		indexEpics, folderEpics, indexFeatures, folderFeatures,
		conflicts, ConflictStrategyFolderPrecedence)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Only E05 should be included (from folders)
	if len(resultEpics) != 1 {
		t.Fatalf("expected 1 epic, got %d", len(resultEpics))
	}
	if resultEpics[0].Key != "E05" {
		t.Errorf("expected epic E05, got %s", resultEpics[0].Key)
	}

	// No features should be included (none in folders)
	if len(resultFeatures) != 0 {
		t.Fatalf("expected 0 features, got %d", len(resultFeatures))
	}

	// Should have warnings about skipped items
	if len(warnings) != 2 {
		t.Fatalf("expected 2 warnings, got %d", len(warnings))
	}
}

func TestConflictResolver_Resolve_Merge_CombinesBothSources(t *testing.T) {
	resolver := NewConflictResolver()

	// Arrange: Items from both sources should be merged
	indexEpics := []DiscoveredEpic{
		{Key: "E04", Title: "Epic Four From Index", Source: SourceIndex},
	}
	folderEpics := []DiscoveredEpic{
		{Key: "E05", Title: "Epic Five From Folder", Source: SourceFolder},
	}
	indexFeatures := []DiscoveredFeature{
		{Key: "E04-F07", EpicKey: "E04", Title: "Feature Seven", Source: SourceIndex},
	}
	folderFeatures := []DiscoveredFeature{
		{Key: "E05-F02", EpicKey: "E05", Title: "Feature Two", Source: SourceFolder},
	}
	conflicts := []Conflict{
		{Type: ConflictTypeEpicIndexOnly, Key: "E04"},
		{Type: ConflictTypeEpicFolderOnly, Key: "E05"},
		{Type: ConflictTypeFeatureIndexOnly, Key: "E04-F07"},
		{Type: ConflictTypeFeatureFolderOnly, Key: "E05-F02"},
	}

	// Act
	resultEpics, resultFeatures, warnings, err := resolver.Resolve(
		indexEpics, folderEpics, indexFeatures, folderFeatures,
		conflicts, ConflictStrategyMerge)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Both epics should be included
	if len(resultEpics) != 2 {
		t.Fatalf("expected 2 epics, got %d", len(resultEpics))
	}

	// Both features should be included
	if len(resultFeatures) != 2 {
		t.Fatalf("expected 2 features, got %d", len(resultFeatures))
	}

	// Should have warnings about missing folders/index entries
	if len(warnings) < 2 {
		t.Errorf("expected at least 2 warnings, got %d", len(warnings))
	}
}

func TestConflictResolver_Resolve_Merge_IndexMetadataWins(t *testing.T) {
	resolver := NewConflictResolver()

	// Arrange: Same epic in both sources, index metadata should win
	indexEpics := []DiscoveredEpic{
		{Key: "E04", Title: "Epic Four From Index", Description: stringPtr("Index description"), Source: SourceIndex},
	}
	folderEpics := []DiscoveredEpic{
		{Key: "E04", Title: "Epic Four From Folder", Description: stringPtr("Folder description"), Source: SourceFolder},
	}
	indexFeatures := []DiscoveredFeature{}
	folderFeatures := []DiscoveredFeature{}
	conflicts := []Conflict{}

	// Act
	resultEpics, _, _, err := resolver.Resolve(
		indexEpics, folderEpics, indexFeatures, folderFeatures,
		conflicts, ConflictStrategyMerge)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(resultEpics) != 1 {
		t.Fatalf("expected 1 epic, got %d", len(resultEpics))
	}

	epic := resultEpics[0]
	if epic.Title != "Epic Four From Index" {
		t.Errorf("expected index title to win, got: %s", epic.Title)
	}
	if epic.Description == nil || *epic.Description != "Index description" {
		t.Errorf("expected index description to win, got: %v", epic.Description)
	}
	if epic.Source != SourceMerged {
		t.Errorf("expected source to be merged, got: %s", epic.Source)
	}
}

func TestConflictResolver_Resolve_Merge_PreservesFolderFilePath(t *testing.T) {
	resolver := NewConflictResolver()

	// Arrange: Epic in both sources, should preserve folder's file path
	folderFilePath := "/path/to/folder/epic.md"
	indexEpics := []DiscoveredEpic{
		{Key: "E04", Title: "Epic Four From Index", Source: SourceIndex},
	}
	folderEpics := []DiscoveredEpic{
		{Key: "E04", Title: "Epic Four From Folder", FilePath: &folderFilePath, Source: SourceFolder},
	}
	indexFeatures := []DiscoveredFeature{}
	folderFeatures := []DiscoveredFeature{}
	conflicts := []Conflict{}

	// Act
	resultEpics, _, _, err := resolver.Resolve(
		indexEpics, folderEpics, indexFeatures, folderFeatures,
		conflicts, ConflictStrategyMerge)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(resultEpics) != 1 {
		t.Fatalf("expected 1 epic, got %d", len(resultEpics))
	}

	epic := resultEpics[0]
	if epic.FilePath == nil {
		t.Fatal("expected file path to be preserved")
	}
	if *epic.FilePath != folderFilePath {
		t.Errorf("expected folder file path to be preserved, got: %s", *epic.FilePath)
	}
}

func TestConflictResolver_Resolve_UnknownStrategy_ReturnsError(t *testing.T) {
	resolver := NewConflictResolver()

	// Arrange
	indexEpics := []DiscoveredEpic{}
	folderEpics := []DiscoveredEpic{}
	indexFeatures := []DiscoveredFeature{}
	folderFeatures := []DiscoveredFeature{}
	conflicts := []Conflict{}

	// Act
	_, _, _, err := resolver.Resolve(
		indexEpics, folderEpics, indexFeatures, folderFeatures,
		conflicts, ConflictStrategy("unknown-strategy"))

	// Assert
	if err == nil {
		t.Fatal("expected error for unknown strategy")
	}
}

func TestConflictResolver_Resolve_Merge_HandlesRelationshipMismatch(t *testing.T) {
	resolver := NewConflictResolver()

	// Arrange: Feature with different parent epics (index parent should win)
	indexEpics := []DiscoveredEpic{
		{Key: "E04", Title: "Epic Four", Source: SourceIndex},
		{Key: "E05", Title: "Epic Five", Source: SourceIndex},
	}
	folderEpics := []DiscoveredEpic{
		{Key: "E04", Title: "Epic Four", Source: SourceFolder},
		{Key: "E05", Title: "Epic Five", Source: SourceFolder},
	}
	indexFeatures := []DiscoveredFeature{
		{Key: "E04-F07", EpicKey: "E04", Title: "Feature Seven", Source: SourceIndex},
	}
	folderFeatures := []DiscoveredFeature{
		{Key: "E04-F07", EpicKey: "E05", Title: "Feature Seven", Source: SourceFolder},
	}
	conflicts := []Conflict{
		{Type: ConflictTypeRelationshipMismatch, Key: "E04-F07"},
	}

	// Act
	_, resultFeatures, warnings, err := resolver.Resolve(
		indexEpics, folderEpics, indexFeatures, folderFeatures,
		conflicts, ConflictStrategyMerge)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(resultFeatures) != 1 {
		t.Fatalf("expected 1 feature, got %d", len(resultFeatures))
	}

	// Index parent should win
	if resultFeatures[0].EpicKey != "E04" {
		t.Errorf("expected index epic key E04 to win, got: %s", resultFeatures[0].EpicKey)
	}

	// Should have warning about relationship mismatch
	if len(warnings) < 1 {
		t.Errorf("expected at least 1 warning, got %d", len(warnings))
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
