package sync

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/discovery"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

func TestDiscoveryIntegration(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Setup test database
	dbPath := filepath.Join(tempDir, "test.db")
	db := setupTestDatabase(t, dbPath)
	defer db.Close()

	// Create engine with initialized database
	engine, err := NewSyncEngine(dbPath)
	if err != nil {
		t.Fatalf("Failed to create sync engine: %v", err)
	}
	defer engine.Close()

	// Create test epic-index.md
	docsDir := filepath.Join(tempDir, "docs", "plan")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatalf("Failed to create docs dir: %v", err)
	}

	indexContent := `# Epic Index

## Active Epics

- [Task Management CLI Core](./E04-task-mgmt-cli-core/)
  - [Task Creation](./E04-task-mgmt-cli-core/E04-F06-task-creation/)
  - [Task List View](./E04-task-mgmt-cli-core/E04-F07-task-list-view/)

- [Enhancements](./E07-enhancements/)
  - [Discovery Integration](./E07-enhancements/E07-F07-epic-index-discovery-integration/)
`
	indexPath := filepath.Join(docsDir, "epic-index.md")
	if err := os.WriteFile(indexPath, []byte(indexContent), 0644); err != nil {
		t.Fatalf("Failed to write epic-index.md: %v", err)
	}

	// Create matching folder structure
	epicDir := filepath.Join(docsDir, "E04-task-mgmt-cli-core")
	if err := os.MkdirAll(epicDir, 0755); err != nil {
		t.Fatalf("Failed to create epic dir: %v", err)
	}

	featureDir := filepath.Join(epicDir, "E04-F06-task-creation")
	if err := os.MkdirAll(featureDir, 0755); err != nil {
		t.Fatalf("Failed to create feature dir: %v", err)
	}

	// Update engine's docsRoot
	engine.docsRoot = tempDir

	// Test discovery
	opts := SyncOptions{
		DBPath:            dbPath,
		FolderPath:        "docs/plan",
		EnableDiscovery:   true,
		DiscoveryStrategy: DiscoveryStrategyMerge,
		ValidationLevel:   ValidationLevelBalanced,
		DryRun:            false,
	}

	report, err := engine.runDiscovery(context.Background(), opts)
	if err != nil {
		t.Fatalf("Discovery failed: %v", err)
	}

	// Verify results
	if report.EpicsDiscovered < 2 {
		t.Errorf("Expected at least 2 epics discovered, got %d", report.EpicsDiscovered)
	}

	if report.FeaturesDiscovered < 3 {
		t.Errorf("Expected at least 3 features discovered, got %d", report.FeaturesDiscovered)
	}

	if report.EpicsImported < 1 {
		t.Errorf("Expected at least 1 epic imported, got %d", report.EpicsImported)
	}

	if report.FeaturesImported < 1 {
		t.Errorf("Expected at least 1 feature imported, got %d", report.FeaturesImported)
	}

	// Verify database has entities
	ctx := context.Background()
	epics, err := engine.epicRepo.List(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get epics: %v", err)
	}

	if len(epics) < 1 {
		t.Errorf("Expected at least 1 epic in database, got %d", len(epics))
	}

	features, err := engine.featureRepo.List(ctx)
	if err != nil {
		t.Fatalf("Failed to get features: %v", err)
	}

	if len(features) < 1 {
		t.Errorf("Expected at least 1 feature in database, got %d", len(features))
	}
}

func TestDiscoveryStrategyMapping(t *testing.T) {
	tests := []struct {
		name     string
		input    DiscoveryStrategy
		expected discovery.ConflictStrategy
	}{
		{
			name:     "index-only maps to index-precedence",
			input:    DiscoveryStrategyIndexOnly,
			expected: discovery.ConflictStrategyIndexPrecedence,
		},
		{
			name:     "folder-only maps to folder-precedence",
			input:    DiscoveryStrategyFolderOnly,
			expected: discovery.ConflictStrategyFolderPrecedence,
		},
		{
			name:     "merge maps to merge",
			input:    DiscoveryStrategyMerge,
			expected: discovery.ConflictStrategyMerge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapDiscoveryStrategy(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestValidationLevelMapping(t *testing.T) {
	tests := []struct {
		name     string
		input    ValidationLevel
		expected discovery.ValidationLevel
	}{
		{
			name:     "strict maps correctly",
			input:    ValidationLevelStrict,
			expected: discovery.ValidationLevelStrict,
		},
		{
			name:     "balanced maps correctly",
			input:    ValidationLevelBalanced,
			expected: discovery.ValidationLevelBalanced,
		},
		{
			name:     "permissive maps correctly",
			input:    ValidationLevelPermissive,
			expected: discovery.ValidationLevelPermissive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapValidationLevel(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestConvertIndexEpics(t *testing.T) {
	input := []discovery.IndexEpic{
		{
			Key:   "E04",
			Title: "Task Management",
			Path:  "E04-task-mgmt",
		},
		{
			Key:   "E07",
			Title: "Enhancements",
			Path:  "E07-enhancements",
		},
	}

	result := convertIndexEpics(input)

	if len(result) != 2 {
		t.Fatalf("Expected 2 epics, got %d", len(result))
	}

	if result[0].Key != "E04" {
		t.Errorf("Expected first epic key to be E04, got %s", result[0].Key)
	}

	if result[0].Source != discovery.SourceIndex {
		t.Errorf("Expected source to be index, got %s", result[0].Source)
	}
}

func TestConvertFolderEpics(t *testing.T) {
	epicMdPath := "/path/to/epic.md"
	input := []discovery.FolderEpic{
		{
			Key:        "E04",
			Slug:       "task-mgmt",
			Path:       "/path/to/E04",
			EpicMdPath: &epicMdPath,
		},
	}

	result := convertFolderEpics(input)

	if len(result) != 1 {
		t.Fatalf("Expected 1 epic, got %d", len(result))
	}

	if result[0].Key != "E04" {
		t.Errorf("Expected epic key to be E04, got %s", result[0].Key)
	}

	if result[0].Source != discovery.SourceFolder {
		t.Errorf("Expected source to be folder, got %s", result[0].Source)
	}

	if result[0].FilePath == nil || *result[0].FilePath != epicMdPath {
		t.Errorf("Expected file path to be %s", epicMdPath)
	}
}

func TestImportDiscoveredEntities(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Setup database
	db := setupTestDatabase(t, dbPath)
	defer db.Close()

	engine, err := NewSyncEngine(dbPath)
	if err != nil {
		t.Fatalf("Failed to create sync engine: %v", err)
	}
	defer engine.Close()

	// Test data
	epics := []discovery.DiscoveredEpic{
		{
			Key:    "E04",
			Title:  "Task Management",
			Source: discovery.SourceIndex,
		},
		{
			Key:    "E07",
			Title:  "Enhancements",
			Source: discovery.SourceFolder,
		},
	}

	features := []discovery.DiscoveredFeature{
		{
			Key:     "E04-F06",
			EpicKey: "E04",
			Title:   "Task Creation",
			Source:  discovery.SourceIndex,
		},
		{
			Key:     "E07-F07",
			EpicKey: "E07",
			Title:   "Discovery Integration",
			Source:  discovery.SourceMerged,
		},
	}

	ctx := context.Background()
	epicsImported, featuresImported, _, err := engine.importDiscoveredEntities(ctx, epics, features)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if epicsImported != 2 {
		t.Errorf("Expected 2 epics imported, got %d", epicsImported)
	}

	if featuresImported != 2 {
		t.Errorf("Expected 2 features imported, got %d", featuresImported)
	}

	// Verify in database
	dbEpics, err := engine.epicRepo.List(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get epics: %v", err)
	}

	if len(dbEpics) != 2 {
		t.Errorf("Expected 2 epics in database, got %d", len(dbEpics))
	}

	dbFeatures, err := engine.featureRepo.List(ctx)
	if err != nil {
		t.Fatalf("Failed to get features: %v", err)
	}

	if len(dbFeatures) != 2 {
		t.Errorf("Expected 2 features in database, got %d", len(dbFeatures))
	}

	// Verify feature relationships
	for _, feature := range dbFeatures {
		epic, err := engine.epicRepo.GetByID(ctx, feature.EpicID)
		if err != nil {
			t.Fatalf("Failed to get epic for feature %s: %v", feature.Key, err)
		}

		expectedEpicKey := feature.Key[:3] // E04 or E07
		if epic.Key != expectedEpicKey {
			t.Errorf("Feature %s has wrong epic: expected %s, got %s",
				feature.Key, expectedEpicKey, epic.Key)
		}
	}
}

func TestImportDiscoveredEntitiesUpdate(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Setup database
	db := setupTestDatabase(t, dbPath)
	defer db.Close()

	engine, err := NewSyncEngine(dbPath)
	if err != nil {
		t.Fatalf("Failed to create sync engine: %v", err)
	}
	defer engine.Close()

	ctx := context.Background()

	// Create initial epic
	initialEpic := &models.Epic{
		Key:      "E04",
		Title:    "Old Title",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	if err := engine.epicRepo.Create(ctx, initialEpic); err != nil {
		t.Fatalf("Failed to create initial epic: %v", err)
	}

	// Import with updated title
	epics := []discovery.DiscoveredEpic{
		{
			Key:    "E04",
			Title:  "New Title",
			Source: discovery.SourceIndex,
		},
	}

	epicsImported, _, _, err := engine.importDiscoveredEntities(ctx, epics, []discovery.DiscoveredFeature{})
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Should not count as imported (already exists)
	if epicsImported != 0 {
		t.Errorf("Expected 0 epics imported (update), got %d", epicsImported)
	}

	// Verify title was updated
	updatedEpic, err := engine.epicRepo.GetByKey(ctx, "E04")
	if err != nil {
		t.Fatalf("Failed to get epic: %v", err)
	}

	if updatedEpic.Title != "New Title" {
		t.Errorf("Expected title to be updated to 'New Title', got '%s'", updatedEpic.Title)
	}
}
