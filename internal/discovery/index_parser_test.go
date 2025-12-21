package discovery

import (
	"os"
	"path/filepath"
	"strings"
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

func TestIndexParser_Parse_RealWorld_WormwoodGM(t *testing.T) {
	// Arrange
	// This test uses the real-world wormwoodGM-epic-index.md file
	// This file is interesting because it primarily uses FILE links (.md files)
	// rather than FOLDER links, which tests that the parser correctly filters
	// out file links and only processes folder links.
	indexPath := filepath.Join("testdata", "wormwoodGM-epic-index.md")

	parser := NewIndexParser()

	// Act
	epics, features, err := parser.Parse(indexPath)

	// Assert
	if err != nil {
		t.Fatalf("Parse failed on real-world file: %v", err)
	}

	// The wormwoodGM file has mostly file links, not folder links
	// Only the following FOLDER links exist in the file:
	// - ./bug-tracker/ (invalid: has hyphen)
	// - ./bug-tracker/open/ (ignored: too deep, 2 segments)
	// - ./bug-tracker/resolved/ (ignored: too deep, 2 segments)
	// - ./change-cards/ (valid special epic)
	// - ./tech-debt/ (valid special epic)
	// - ./E09-campaign-screen-redux/ (valid epic)
	// - ./FUTURE_SCOPE/ (invalid: not matching any pattern)
	// - ./E10-opentelemetry-observability/prps/ (ignored: too deep, 2 segments)

	// Expected: 3 epics (change-cards, tech-debt, E09)
	expectedEpicCount := 3
	if len(epics) != expectedEpicCount {
		t.Errorf("Expected %d epics, got %d", expectedEpicCount, len(epics))
		for i, epic := range epics {
			t.Logf("  Epic %d: %s (%s) - %s", i, epic.Key, epic.Path, epic.Title)
		}
	}

	// Verify specific epics are found
	epicKeys := make(map[string]IndexEpic)
	for _, epic := range epics {
		epicKeys[epic.Key] = epic
	}

	// Test that we found the expected epics
	expectedEpics := []struct {
		key   string
		title string
		path  string
	}{
		{key: "change-cards", title: "View Folder", path: "change-cards"},
		{key: "tech-debt", title: "View Folder", path: "tech-debt"},
		{key: "E09", title: "View Folder", path: "E09-campaign-screen-redux"},
	}

	for _, expected := range expectedEpics {
		epic, found := epicKeys[expected.key]
		if !found {
			t.Errorf("Expected to find epic key '%s', but it was not parsed", expected.key)
			continue
		}
		if epic.Title != expected.title {
			t.Errorf("Epic %s: expected title '%s', got '%s'", expected.key, expected.title, epic.Title)
		}
		if epic.Path != expected.path {
			t.Errorf("Epic %s: expected path '%s', got '%s'", expected.key, expected.path, epic.Path)
		}
	}

	// Verify Features
	// The file has NO folder links to features - all feature links are to README.md files
	// which should be filtered out by the parser
	expectedFeatureCount := 0
	if len(features) != expectedFeatureCount {
		t.Errorf("Expected %d features, got %d", expectedFeatureCount, len(features))
		for i, feature := range features {
			t.Logf("  Feature %d: %s (Epic: %s) - %s", i, feature.Key, feature.EpicKey, feature.Title)
		}
	}

	// Verify that file links (.md) are correctly ignored
	// This is the KEY test for this real-world file
	// The file contains many links like:
	// - ./E00-launchpad/epic.md
	// - ./E00-launchpad/F01-User_Authentication/README.md
	// These should all be filtered out

	// We can verify by checking that none of the standard epics E00-E08, E10 were parsed
	// (since they only appear as file links, not folder links)
	invalidEpicKeys := []string{"E00", "E01", "E02", "E03", "E04", "E05", "E06", "E07", "E08", "E10"}
	for _, invalidKey := range invalidEpicKeys {
		if _, found := epicKeys[invalidKey]; found {
			t.Errorf("Should not parse epic '%s' - it only appears as file link in this document", invalidKey)
		}
	}

	// Verify other invalid patterns are not parsed
	otherInvalidKeys := []string{"bug-tracker", "FUTURE_SCOPE"}
	for _, invalidKey := range otherInvalidKeys {
		if _, found := epicKeys[invalidKey]; found {
			t.Errorf("Should not parse '%s' as an epic (invalid pattern)", invalidKey)
		}
	}

	// Verify that deeper paths (2+ segments) are ignored for epics
	for _, epic := range epics {
		segmentCount := len(strings.Split(epic.Path, "/"))
		if segmentCount != 1 {
			t.Errorf("Epic %s has invalid path segment count: %d (path: %s)", epic.Key, segmentCount, epic.Path)
		}
	}

	// This test demonstrates that the parser correctly:
	// 1. Filters out file links (*.md files)
	// 2. Only processes folder links (ending in /)
	// 3. Validates epic patterns (E##-slug or special types)
	// 4. Ignores deeper paths (bug-tracker/open/, etc.)
	// 5. Handles mixed content with external links, markdown formatting, etc.
}
