package discovery

import (
	"encoding/json"
	"testing"
)

// TestDiscoveryOptions tests the DiscoveryOptions struct
func TestDiscoveryOptions(t *testing.T) {
	tests := []struct {
		name string
		opts DiscoveryOptions
		want DiscoveryOptions
	}{
		{
			name: "default options",
			opts: DiscoveryOptions{
				DocsRoot:        "docs/plan",
				Strategy:        ConflictStrategyIndexPrecedence,
				ValidationLevel: ValidationLevelBalanced,
			},
			want: DiscoveryOptions{
				DocsRoot:        "docs/plan",
				Strategy:        ConflictStrategyIndexPrecedence,
				ValidationLevel: ValidationLevelBalanced,
			},
		},
		{
			name: "custom options",
			opts: DiscoveryOptions{
				DocsRoot:        "custom/docs",
				IndexPath:       "custom/docs/epic-index.md",
				Strategy:        ConflictStrategyMerge,
				DryRun:          true,
				ValidationLevel: ValidationLevelStrict,
			},
			want: DiscoveryOptions{
				DocsRoot:        "custom/docs",
				IndexPath:       "custom/docs/epic-index.md",
				Strategy:        ConflictStrategyMerge,
				DryRun:          true,
				ValidationLevel: ValidationLevelStrict,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.opts.DocsRoot != tt.want.DocsRoot {
				t.Errorf("DocsRoot = %v, want %v", tt.opts.DocsRoot, tt.want.DocsRoot)
			}
			if tt.opts.Strategy != tt.want.Strategy {
				t.Errorf("Strategy = %v, want %v", tt.opts.Strategy, tt.want.Strategy)
			}
			if tt.opts.ValidationLevel != tt.want.ValidationLevel {
				t.Errorf("ValidationLevel = %v, want %v", tt.opts.ValidationLevel, tt.want.ValidationLevel)
			}
		})
	}
}

// TestDiscoveryReportJSONMarshaling tests JSON marshaling of DiscoveryReport
func TestDiscoveryReportJSONMarshaling(t *testing.T) {
	report := DiscoveryReport{
		FoldersScanned:       47,
		FilesAnalyzed:        123,
		EpicsDiscovered:      15,
		EpicsFromIndex:       12,
		EpicsFromFolders:     5,
		FeaturesDiscovered:   87,
		FeaturesFromIndex:    80,
		FeaturesFromFolders:  10,
		RelatedDocsCataloged: 234,
		ConflictsDetected:    2,
		Conflicts: []Conflict{
			{
				Type:       ConflictTypeEpicFolderOnly,
				Key:        "E04",
				Path:       "docs/plan/E04-task-mgmt-cli-core/",
				Resolution: "skipped",
				Strategy:   "index-precedence",
				Suggestion: "Add E04 to epic-index.md or use merge strategy",
			},
		},
		Warnings: []string{"Warning: Feature E05-F03 listed in index but folder not found"},
		Errors:   []string{},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("Failed to marshal DiscoveryReport to JSON: %v", err)
	}

	// Unmarshal back
	var unmarshaled DiscoveryReport
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal JSON to DiscoveryReport: %v", err)
	}

	// Verify key fields
	if unmarshaled.FoldersScanned != report.FoldersScanned {
		t.Errorf("FoldersScanned = %v, want %v", unmarshaled.FoldersScanned, report.FoldersScanned)
	}
	if unmarshaled.EpicsDiscovered != report.EpicsDiscovered {
		t.Errorf("EpicsDiscovered = %v, want %v", unmarshaled.EpicsDiscovered, report.EpicsDiscovered)
	}
	if unmarshaled.ConflictsDetected != report.ConflictsDetected {
		t.Errorf("ConflictsDetected = %v, want %v", unmarshaled.ConflictsDetected, report.ConflictsDetected)
	}
	if len(unmarshaled.Conflicts) != len(report.Conflicts) {
		t.Errorf("len(Conflicts) = %v, want %v", len(unmarshaled.Conflicts), len(report.Conflicts))
	}
	if len(unmarshaled.Warnings) != len(report.Warnings) {
		t.Errorf("len(Warnings) = %v, want %v", len(unmarshaled.Warnings), len(report.Warnings))
	}
}

// TestConflictTypes tests all conflict type constants
func TestConflictTypes(t *testing.T) {
	tests := []struct {
		name         string
		conflictType ConflictType
		want         string
	}{
		{"epic_index_only", ConflictTypeEpicIndexOnly, "epic_index_only"},
		{"epic_folder_only", ConflictTypeEpicFolderOnly, "epic_folder_only"},
		{"feature_index_only", ConflictTypeFeatureIndexOnly, "feature_index_only"},
		{"feature_folder_only", ConflictTypeFeatureFolderOnly, "feature_folder_only"},
		{"relationship_mismatch", ConflictTypeRelationshipMismatch, "relationship_mismatch"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.conflictType) != tt.want {
				t.Errorf("ConflictType = %v, want %v", tt.conflictType, tt.want)
			}
		})
	}
}

// TestConflictStrategies tests all conflict strategy constants
func TestConflictStrategies(t *testing.T) {
	tests := []struct {
		name     string
		strategy ConflictStrategy
		want     string
	}{
		{"index-precedence", ConflictStrategyIndexPrecedence, "index-precedence"},
		{"folder-precedence", ConflictStrategyFolderPrecedence, "folder-precedence"},
		{"merge", ConflictStrategyMerge, "merge"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.strategy) != tt.want {
				t.Errorf("ConflictStrategy = %v, want %v", tt.strategy, tt.want)
			}
		})
	}
}

// TestValidationLevels tests all validation level constants
func TestValidationLevels(t *testing.T) {
	tests := []struct {
		name  string
		level ValidationLevel
		want  string
	}{
		{"strict", ValidationLevelStrict, "strict"},
		{"balanced", ValidationLevelBalanced, "balanced"},
		{"permissive", ValidationLevelPermissive, "permissive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.level) != tt.want {
				t.Errorf("ValidationLevel = %v, want %v", tt.level, tt.want)
			}
		})
	}
}

// TestIndexEpic tests the IndexEpic struct
func TestIndexEpic(t *testing.T) {
	epic := IndexEpic{
		Key:   "E04",
		Title: "Task Management CLI Core",
		Path:  "./E04-task-mgmt-cli-core/",
		Features: []IndexFeature{
			{
				Key:     "E04-F07",
				EpicKey: "E04",
				Title:   "Initialization Sync",
				Path:    "./E04-task-mgmt-cli-core/E04-F07-initialization-sync/",
			},
		},
	}

	if epic.Key != "E04" {
		t.Errorf("Key = %v, want E04", epic.Key)
	}
	if len(epic.Features) != 1 {
		t.Errorf("len(Features) = %v, want 1", len(epic.Features))
	}
	if epic.Features[0].EpicKey != "E04" {
		t.Errorf("Features[0].EpicKey = %v, want E04", epic.Features[0].EpicKey)
	}
}

// TestFolderEpic tests the FolderEpic struct
func TestFolderEpic(t *testing.T) {
	epicMdPath := "docs/plan/E04-task-mgmt-cli-core/epic.md"
	epic := FolderEpic{
		Key:        "E04",
		Slug:       "task-mgmt-cli-core",
		Path:       "docs/plan/E04-task-mgmt-cli-core/",
		EpicMdPath: &epicMdPath,
		Features: []FolderFeature{
			{
				Key:         "E04-F07",
				EpicKey:     "E04",
				Slug:        "initialization-sync",
				Path:        "docs/plan/E04-task-mgmt-cli-core/E04-F07-initialization-sync/",
				PrdPath:     strPtr("docs/plan/E04-task-mgmt-cli-core/E04-F07-initialization-sync/prd.md"),
				RelatedDocs: []string{"docs/plan/E04-task-mgmt-cli-core/E04-F07-initialization-sync/02-architecture.md"},
			},
		},
	}

	if epic.Key != "E04" {
		t.Errorf("Key = %v, want E04", epic.Key)
	}
	if epic.Slug != "task-mgmt-cli-core" {
		t.Errorf("Slug = %v, want task-mgmt-cli-core", epic.Slug)
	}
	if epic.EpicMdPath == nil {
		t.Error("EpicMdPath should not be nil")
	}
	if len(epic.Features) != 1 {
		t.Errorf("len(Features) = %v, want 1", len(epic.Features))
	}
}

// TestDiscoveredEpic tests the DiscoveredEpic struct (merged result)
func TestDiscoveredEpic(t *testing.T) {
	epic := DiscoveredEpic{
		Key:         "E04",
		Title:       "Task Management CLI Core",
		Description: strPtr("Task management CLI core functionality"),
		FilePath:    strPtr("docs/plan/E04-task-mgmt-cli-core/epic.md"),
		Source:      SourceMerged,
		Features: []DiscoveredFeature{
			{
				Key:         "E04-F07",
				EpicKey:     "E04",
				Title:       "Initialization Sync",
				Description: strPtr("Initialization and sync functionality"),
				FilePath:    strPtr("docs/plan/E04-task-mgmt-cli-core/E04-F07-initialization-sync/prd.md"),
				RelatedDocs: []string{"docs/plan/E04-task-mgmt-cli-core/E04-F07-initialization-sync/02-architecture.md"},
				Source:      SourceMerged,
			},
		},
	}

	if epic.Key != "E04" {
		t.Errorf("Key = %v, want E04", epic.Key)
	}
	if epic.Source != SourceMerged {
		t.Errorf("Source = %v, want %v", epic.Source, SourceMerged)
	}
	if len(epic.Features) != 1 {
		t.Errorf("len(Features) = %v, want 1", len(epic.Features))
	}
}

// TestDiscoverySource tests all discovery source constants
func TestDiscoverySource(t *testing.T) {
	tests := []struct {
		name   string
		source DiscoverySource
		want   string
	}{
		{"index", SourceIndex, "index"},
		{"folder", SourceFolder, "folder"},
		{"merged", SourceMerged, "merged"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.source) != tt.want {
				t.Errorf("DiscoverySource = %v, want %v", tt.source, tt.want)
			}
		})
	}
}

// TestConflictJSONMarshaling tests JSON marshaling of Conflict
func TestConflictJSONMarshaling(t *testing.T) {
	conflict := Conflict{
		Type:       ConflictTypeEpicFolderOnly,
		Key:        "E04",
		Path:       "docs/plan/E04-task-mgmt-cli-core/",
		Resolution: "skipped",
		Strategy:   "index-precedence",
		Suggestion: "Add E04 to epic-index.md or use merge strategy",
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(conflict)
	if err != nil {
		t.Fatalf("Failed to marshal Conflict to JSON: %v", err)
	}

	// Unmarshal back
	var unmarshaled Conflict
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal JSON to Conflict: %v", err)
	}

	// Verify fields
	if unmarshaled.Type != conflict.Type {
		t.Errorf("Type = %v, want %v", unmarshaled.Type, conflict.Type)
	}
	if unmarshaled.Key != conflict.Key {
		t.Errorf("Key = %v, want %v", unmarshaled.Key, conflict.Key)
	}
	if unmarshaled.Path != conflict.Path {
		t.Errorf("Path = %v, want %v", unmarshaled.Path, conflict.Path)
	}
}

// Helper function to create string pointer
func strPtr(s string) *string {
	return &s
}
