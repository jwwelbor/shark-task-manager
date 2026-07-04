package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIndexParser_Parse_StandardEpicLinks(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "epic-index.md")

	content := `# Epic Index

## Active Epics

- [Task Management CLI Core](./E04-task-mgmt-cli-core/)
- [Advanced Querying](./E05-advanced-querying/)
- [Intelligent Scanning](./E06-intelligent-scanning/)
`

	err := os.WriteFile(indexPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewIndexParser()

	// Act
	epics, features, err := parser.Parse(indexPath)

	// Assert
	if err != nil {
		t.Errorf("Parse failed: %v", err)
	}

	if len(epics) != 3 {
		t.Errorf("Expected 3 epics, got %d", len(epics))
	}

	if len(features) != 0 {
		t.Errorf("Expected 0 features, got %d", len(features))
	}

	// Verify first epic
	if epics[0].Key != "E04" {
		t.Errorf("Expected epic key 'E04', got '%s'", epics[0].Key)
	}
	if epics[0].Title != "Task Management CLI Core" {
		t.Errorf("Expected title 'Task Management CLI Core', got '%s'", epics[0].Title)
	}
	if epics[0].Path != "E04-task-mgmt-cli-core" {
		t.Errorf("Expected path 'E04-task-mgmt-cli-core', got '%s'", epics[0].Path)
	}

	// Verify second epic
	if epics[1].Key != "E05" {
		t.Errorf("Expected epic key 'E05', got '%s'", epics[1].Key)
	}

	// Verify third epic
	if epics[2].Key != "E06" {
		t.Errorf("Expected epic key 'E06', got '%s'", epics[2].Key)
	}
}

func TestIndexParser_Parse_SpecialEpicTypes(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "epic-index.md")

	content := `# Epic Index

## Special Types

- [Technical Debt](./tech-debt/)
- [Bug Fixes](./bugs/)
- [Change Cards](./change-cards/)
`

	err := os.WriteFile(indexPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewIndexParser()

	// Act
	epics, features, err := parser.Parse(indexPath)

	// Assert
	if err != nil {
		t.Errorf("Parse failed: %v", err)
	}

	if len(epics) != 3 {
		t.Errorf("Expected 3 special epics, got %d", len(epics))
	}

	if len(features) != 0 {
		t.Errorf("Expected 0 features, got %d", len(features))
	}

	// Verify special epic keys
	if epics[0].Key != "tech-debt" {
		t.Errorf("Expected epic key 'tech-debt', got '%s'", epics[0].Key)
	}
	if epics[0].Title != "Technical Debt" {
		t.Errorf("Expected title 'Technical Debt', got '%s'", epics[0].Title)
	}

	if epics[1].Key != "bugs" {
		t.Errorf("Expected epic key 'bugs', got '%s'", epics[1].Key)
	}

	if epics[2].Key != "change-cards" {
		t.Errorf("Expected epic key 'change-cards', got '%s'", epics[2].Key)
	}
}

func TestIndexParser_Parse_FeatureLinks(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "epic-index.md")

	content := `# Epic Index

## Epics with Features

- [Task Management](./E04-task-mgmt-cli-core/)
  - [Initialization & Sync](./E04-task-mgmt-cli-core/E04-F07-initialization-sync/)
  - [Task CRUD Operations](./E04-task-mgmt-cli-core/E04-F01-task-crud/)
- [Advanced Querying](./E05-advanced-querying/)
  - [Pattern Library](./E05-advanced-querying/E05-F01-pattern-library/)
`

	err := os.WriteFile(indexPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewIndexParser()

	// Act
	epics, features, err := parser.Parse(indexPath)

	// Assert
	if err != nil {
		t.Errorf("Parse failed: %v", err)
	}

	if len(epics) != 2 {
		t.Errorf("Expected 2 epics, got %d", len(epics))
	}

	if len(features) != 3 {
		t.Errorf("Expected 3 features, got %d", len(features))
	}

	// Verify first feature
	if features[0].Key != "E04-F07" {
		t.Errorf("Expected feature key 'E04-F07', got '%s'", features[0].Key)
	}
	if features[0].EpicKey != "E04" {
		t.Errorf("Expected epic key 'E04', got '%s'", features[0].EpicKey)
	}
	if features[0].Title != "Initialization & Sync" {
		t.Errorf("Expected title 'Initialization & Sync', got '%s'", features[0].Title)
	}

	// Verify second feature
	if features[1].Key != "E04-F01" {
		t.Errorf("Expected feature key 'E04-F01', got '%s'", features[1].Key)
	}
	if features[1].EpicKey != "E04" {
		t.Errorf("Expected epic key 'E04', got '%s'", features[1].EpicKey)
	}

	// Verify third feature (different epic)
	if features[2].Key != "E05-F01" {
		t.Errorf("Expected feature key 'E05-F01', got '%s'", features[2].Key)
	}
	if features[2].EpicKey != "E05" {
		t.Errorf("Expected epic key 'E05', got '%s'", features[2].EpicKey)
	}
}

func TestIndexParser_Parse_MixedListFormats(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "epic-index.md")

	// Test both unordered and ordered lists
	content := `# Epic Index

## Unordered List

- [Epic One](./E01-epic-one/)
- [Epic Two](./E02-epic-two/)

## Ordered List

1. [Epic Three](./E03-epic-three/)
2. [Epic Four](./E04-epic-four/)
`

	err := os.WriteFile(indexPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewIndexParser()

	// Act
	epics, _, err := parser.Parse(indexPath)

	// Assert
	if err != nil {
		t.Errorf("Parse failed: %v", err)
	}

	if len(epics) != 4 {
		t.Errorf("Expected 4 epics from mixed list formats, got %d", len(epics))
	}
}

func TestIndexParser_Parse_RelativePathVariations(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "epic-index.md")

	// Test different path formats: ./, /, no prefix, trailing slash
	content := `# Epic Index

- [With Dot Slash](./E01-with-dot/)
- [Without Dot Slash](E02-without-dot/)
- [With Leading Slash](/E03-leading-slash/)
- [No Trailing Slash](./E04-no-trailing)
`

	err := os.WriteFile(indexPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewIndexParser()

	// Act
	epics, _, err := parser.Parse(indexPath)

	// Assert
	if err != nil {
		t.Errorf("Parse failed: %v", err)
	}

	if len(epics) != 4 {
		t.Errorf("Expected 4 epics with various path formats, got %d", len(epics))
	}

	// All paths should be normalized (no leading ./ or /, no trailing /)
	for i, epic := range epics {
		if epic.Path[0] == '/' || epic.Path[0] == '.' {
			t.Errorf("Epic %d path not normalized: '%s'", i, epic.Path)
		}
	}
}

func TestIndexParser_Parse_MalformedLinks(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "epic-index.md")

	// Include malformed links that should be skipped
	content := `# Epic Index

- [Valid Epic](./E01-valid/)
- [Broken Link (./E02-broken/
- Missing Closing [Bracket for E02
- [Invalid Pattern](./X99-invalid/)
- [Valid Epic Two](./E04-valid-two/)
- [Deep Path Should Be Ignored](./E04-valid-two/feature/task/)
`

	err := os.WriteFile(indexPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewIndexParser()

	// Act
	epics, features, err := parser.Parse(indexPath)

	// Assert - should not fail, just skip invalid links
	if err != nil {
		t.Errorf("Parse should not fail on malformed links: %v", err)
	}

	// Should only get the 2 valid epics (E01, E04)
	// - E01-valid: valid
	// - X99-invalid: invalid pattern, skipped
	// - E04-valid-two: valid
	// - E04-valid-two/feature/task: too deep, ignored
	if len(epics) != 2 {
		t.Errorf("Expected 2 valid epics (skipping malformed), got %d", len(epics))
	}

	if len(features) != 0 {
		t.Errorf("Expected 0 features, got %d", len(features))
	}
}

func TestIndexParser_Parse_FileNotFound(t *testing.T) {
	// Arrange
	parser := NewIndexParser()
	nonexistentPath := "/nonexistent/epic-index.md"

	// Act
	_, _, err := parser.Parse(nonexistentPath)

	// Assert
	if err == nil {
		t.Error("Expected error when file doesn't exist")
	}
}

func TestIndexParser_Parse_EmptyFile(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "epic-index.md")

	err := os.WriteFile(indexPath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewIndexParser()

	// Act
	epics, features, err := parser.Parse(indexPath)

	// Assert
	if err != nil {
		t.Errorf("Parse should not fail on empty file: %v", err)
	}

	if len(epics) != 0 {
		t.Errorf("Expected 0 epics from empty file, got %d", len(epics))
	}

	if len(features) != 0 {
		t.Errorf("Expected 0 features from empty file, got %d", len(features))
	}
}

func TestIndexParser_Parse_NoMarkdownLinks(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "epic-index.md")

	content := `# Epic Index

This is a plain text file with no markdown links.
Just some regular text.
`

	err := os.WriteFile(indexPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewIndexParser()

	// Act
	epics, features, err := parser.Parse(indexPath)

	// Assert
	if err != nil {
		t.Errorf("Parse should not fail when no links found: %v", err)
	}

	if len(epics) != 0 {
		t.Errorf("Expected 0 epics when no links, got %d", len(epics))
	}

	if len(features) != 0 {
		t.Errorf("Expected 0 features when no links, got %d", len(features))
	}
}

func TestIndexParser_Parse_ComplexRealWorld(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "epic-index.md")

	// Realistic epic-index.md with mixed content
	content := `# Epic Index

This index tracks all epics and features in the Shark Task Manager project.

## Core Functionality

1. [Task Management CLI Core](./E04-task-mgmt-cli-core/)
   - [Initialization & Sync](./E04-task-mgmt-cli-core/E04-F07-initialization-sync/)
   - [Task CRUD Operations](./E04-task-mgmt-cli-core/E04-F01-task-crud/)

2. [Advanced Querying & Filtering](./E05-advanced-querying/)
   - [Pattern Configuration](./E05-advanced-querying/E05-F01-pattern-library/)

## Enhancements

- [Intelligent Documentation Scanning](./E06-intelligent-scanning/)
  - [Epic & Feature Discovery](./E06-intelligent-scanning/E06-F02-epic-feature-discovery/)

## Maintenance

- [Technical Debt](./tech-debt/)
- [Bug Fixes](./bugs/)

## Notes

Some links to external resources [Google](https://google.com) should be ignored.
Links to markdown files [README](./README.md) should also be ignored.
`

	err := os.WriteFile(indexPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewIndexParser()

	// Act
	epics, features, err := parser.Parse(indexPath)

	// Assert
	if err != nil {
		t.Errorf("Parse failed: %v", err)
	}

	// Should find 5 epics: E04, E05, E06, tech-debt, bugs
	if len(epics) != 5 {
		t.Errorf("Expected 5 epics, got %d", len(epics))
	}

	// Should find 3 features: E04-F07, E04-F01, E05-F01, E06-F02
	if len(features) != 4 {
		t.Errorf("Expected 4 features, got %d", len(features))
	}

	// Verify epic keys
	epicKeys := make(map[string]bool)
	for _, epic := range epics {
		epicKeys[epic.Key] = true
	}

	expectedEpicKeys := []string{"E04", "E05", "E06", "tech-debt", "bugs"}
	for _, expectedKey := range expectedEpicKeys {
		if !epicKeys[expectedKey] {
			t.Errorf("Expected to find epic key '%s'", expectedKey)
		}
	}

	// Verify feature keys
	featureKeys := make(map[string]bool)
	for _, feature := range features {
		featureKeys[feature.Key] = true
	}

	expectedFeatureKeys := []string{"E04-F07", "E04-F01", "E05-F01", "E06-F02"}
	for _, expectedKey := range expectedFeatureKeys {
		if !featureKeys[expectedKey] {
			t.Errorf("Expected to find feature key '%s'", expectedKey)
		}
	}
}

func TestIndexParser_parseEpicLink_StandardFormat(t *testing.T) {
	// Arrange
	parser := NewIndexParser()

	testCases := []struct {
		linkText      string
		path          string
		expectedKey   string
		expectedTitle string
		shouldSucceed bool
	}{
		{
			linkText:      "Task Management",
			path:          "E04-task-mgmt-cli-core",
			expectedKey:   "E04",
			expectedTitle: "Task Management",
			shouldSucceed: true,
		},
		{
			linkText:      "Advanced Querying",
			path:          "E05-advanced-querying",
			expectedKey:   "E05",
			expectedTitle: "Advanced Querying",
			shouldSucceed: true,
		},
		{
			linkText:      "Technical Debt",
			path:          "tech-debt",
			expectedKey:   "tech-debt",
			expectedTitle: "Technical Debt",
			shouldSucceed: true,
		},
		{
			linkText:      "Bug Fixes",
			path:          "bugs",
			expectedKey:   "bugs",
			expectedTitle: "Bug Fixes",
			shouldSucceed: true,
		},
		{
			linkText:      "Invalid Pattern",
			path:          "X99-invalid",
			shouldSucceed: false,
		},
		{
			linkText:      "Feature Not Epic",
			path:          "E04-F01-something",
			shouldSucceed: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			// Act
			epic, err := parser.parseEpicLink(tc.linkText, tc.path)

			// Assert
			if tc.shouldSucceed {
				if err != nil {
					t.Errorf("Expected success, got error: %v", err)
				}
				if epic.Key != tc.expectedKey {
					t.Errorf("Expected key '%s', got '%s'", tc.expectedKey, epic.Key)
				}
				if epic.Title != tc.expectedTitle {
					t.Errorf("Expected title '%s', got '%s'", tc.expectedTitle, epic.Title)
				}
			} else {
				if err == nil {
					t.Error("Expected error for invalid pattern")
				}
			}
		})
	}
}

func TestIndexParser_parseFeatureLink_StandardFormat(t *testing.T) {
	// Arrange
	parser := NewIndexParser()

	testCases := []struct {
		linkText        string
		path            string
		expectedKey     string
		expectedEpicKey string
		expectedTitle   string
		shouldSucceed   bool
	}{
		{
			linkText:        "Initialization & Sync",
			path:            "E04-task-mgmt-cli-core/E04-F07-initialization-sync",
			expectedKey:     "E04-F07",
			expectedEpicKey: "E04",
			expectedTitle:   "Initialization & Sync",
			shouldSucceed:   true,
		},
		{
			linkText:        "Pattern Library",
			path:            "E05-advanced-querying/E05-F01-pattern-library",
			expectedKey:     "E05-F01",
			expectedEpicKey: "E05",
			expectedTitle:   "Pattern Library",
			shouldSucceed:   true,
		},
		{
			linkText:      "Too Deep",
			path:          "E04-epic/E04-F01-feature/tasks/T-001",
			shouldSucceed: false,
		},
		{
			linkText:      "Not a Feature",
			path:          "E04-epic-slug",
			shouldSucceed: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			// Act
			feature, err := parser.parseFeatureLink(tc.linkText, tc.path)

			// Assert
			if tc.shouldSucceed {
				if err != nil {
					t.Errorf("Expected success, got error: %v", err)
				}
				if feature.Key != tc.expectedKey {
					t.Errorf("Expected key '%s', got '%s'", tc.expectedKey, feature.Key)
				}
				if feature.EpicKey != tc.expectedEpicKey {
					t.Errorf("Expected epic key '%s', got '%s'", tc.expectedEpicKey, feature.EpicKey)
				}
				if feature.Title != tc.expectedTitle {
					t.Errorf("Expected title '%s', got '%s'", tc.expectedTitle, feature.Title)
				}
			} else {
				if err == nil {
					t.Error("Expected error for invalid pattern")
				}
			}
		})
	}
}
