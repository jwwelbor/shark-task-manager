package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestFormatEntityCreationMessage tests the creation message formatting for human-readable output
func TestFormatEntityCreationMessage(t *testing.T) {
	// Get absolute path for testing
	projectRoot := "/home/user/projects/shark"
	filePath := filepath.Join(projectRoot, "docs/plan/E07-epic-name/epic.md")

	tests := []struct {
		name             string
		entityType       string
		entityKey        string
		entityTitle      string
		filePath         string
		projectRoot      string
		requiredSections []string
		wantContains     []string
		wantNotContains  []string
	}{
		{
			name:             "Epic creation message",
			entityType:       "epic",
			entityKey:        "E07",
			entityTitle:      "User Management System",
			filePath:         filePath,
			projectRoot:      projectRoot,
			requiredSections: []string{"Vision & Goals", "Success Criteria", "Features"},
			wantContains: []string{
				"Created epic E07",
				"User Management System",
				"PLACEHOLDER FILE CREATED - EDITING REQUIRED",
				filePath,
				"REQUIRED ACTIONS",
				"1. Edit the task file to add implementation details",
				"2. Fill in required sections",
				"Vision & Goals",
				"Success Criteria",
				"Features",
			},
			wantNotContains: []string{
				"File created at", // Old messaging
			},
		},
		{
			name:             "Feature creation message",
			entityType:       "feature",
			entityKey:        "E07-F01",
			entityTitle:      "Authentication & Authorization",
			filePath:         filepath.Join(projectRoot, "docs/plan/E07-epic/E07-F01-auth/feature.md"),
			projectRoot:      projectRoot,
			requiredSections: []string{"Requirements", "Design", "Test Plan"},
			wantContains: []string{
				"Created feature E07-F01",
				"Authentication & Authorization",
				"PLACEHOLDER FILE CREATED - EDITING REQUIRED",
				"REQUIRED ACTIONS",
				"Requirements",
				"Design",
				"Test Plan",
			},
		},
		{
			name:             "Task creation message",
			entityType:       "task",
			entityKey:        "T-E07-F01-001",
			entityTitle:      "Implement JWT validation",
			filePath:         filepath.Join(projectRoot, "docs/plan/E07/F01/tasks/T-E07-F01-001.md"),
			projectRoot:      projectRoot,
			requiredSections: []string{"Implementation Plan", "Acceptance Criteria", "Test Plan"},
			wantContains: []string{
				"Created task T-E07-F01-001",
				"Implement JWT validation",
				"PLACEHOLDER FILE CREATED - EDITING REQUIRED",
				"REQUIRED ACTIONS",
				"Implementation Plan",
				"Acceptance Criteria",
				"Test Plan",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the function (default to fileWasLinked=false for these legacy tests)
			message := FormatEntityCreationMessage(tt.entityType, tt.entityKey, tt.entityTitle, tt.filePath, tt.projectRoot, false, tt.requiredSections)

			// Check that all required strings are present
			for _, want := range tt.wantContains {
				if !strings.Contains(message, want) {
					t.Errorf("FormatEntityCreationMessage() message missing %q\nGot:\n%s", want, message)
				}
			}

			// Check that unwanted strings are not present
			for _, notWant := range tt.wantNotContains {
				if strings.Contains(message, notWant) {
					t.Errorf("FormatEntityCreationMessage() message should not contain %q\nGot:\n%s", notWant, message)
				}
			}

			// Verify absolute path is used (not relative)
			if !strings.Contains(message, tt.filePath) {
				t.Errorf("FormatEntityCreationMessage() should contain absolute path %q\nGot:\n%s", tt.filePath, message)
			}
		})
	}
}

// TestFormatEntityCreationJSON tests the JSON output formatting
func TestFormatEntityCreationJSON(t *testing.T) {
	projectRoot := "/home/user/projects/shark"
	filePath := filepath.Join(projectRoot, "docs/plan/E07-epic/epic.md")

	tests := []struct {
		name             string
		entityType       string
		entityKey        string
		entityTitle      string
		filePath         string
		projectRoot      string
		requiredSections []string
		wantFields       map[string]interface{}
	}{
		{
			name:             "Task JSON output",
			entityType:       "task",
			entityKey:        "T-E07-F01-001",
			entityTitle:      "Implement JWT validation",
			filePath:         filePath,
			projectRoot:      projectRoot,
			requiredSections: []string{"Implementation Plan", "Acceptance Criteria", "Test Plan"},
			wantFields: map[string]interface{}{
				"status":           "created",
				"entity_type":      "task",
				"key":              "T-E07-F01-001",
				"title":            "Implement JWT validation",
				"file_state":       "placeholder",
				"requires_editing": true,
			},
		},
		{
			name:             "Epic JSON output",
			entityType:       "epic",
			entityKey:        "E07",
			entityTitle:      "User Management",
			filePath:         filePath,
			projectRoot:      projectRoot,
			requiredSections: []string{"Vision & Goals"},
			wantFields: map[string]interface{}{
				"status":           "created",
				"entity_type":      "epic",
				"key":              "E07",
				"file_state":       "placeholder",
				"requires_editing": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the function
			result := FormatEntityCreationJSON(tt.entityType, tt.entityKey, tt.entityTitle, tt.filePath, tt.projectRoot, tt.requiredSections)

			// Check required fields
			for key, expectedValue := range tt.wantFields {
				actualValue, ok := result[key]
				if !ok {
					t.Errorf("FormatEntityCreationJSON() missing field %q", key)
					continue
				}

				if actualValue != expectedValue {
					t.Errorf("FormatEntityCreationJSON() field %q = %v, want %v", key, actualValue, expectedValue)
				}
			}

			// Check file_path is absolute
			filePath, ok := result["file_path"].(string)
			if !ok {
				t.Error("FormatEntityCreationJSON() missing or invalid file_path field")
			} else if !filepath.IsAbs(filePath) {
				t.Errorf("FormatEntityCreationJSON() file_path should be absolute, got %q", filePath)
			}

			// Check required_actions array exists and has correct structure
			requiredActions, ok := result["required_actions"].([]map[string]interface{})
			if !ok {
				t.Error("FormatEntityCreationJSON() missing or invalid required_actions field")
			} else if len(requiredActions) == 0 {
				t.Error("FormatEntityCreationJSON() required_actions should not be empty")
			} else {
				// Check first action has required fields
				action := requiredActions[0]
				if action["action"] != "edit_file" {
					t.Errorf("FormatEntityCreationJSON() first action should be 'edit_file', got %v", action["action"])
				}
				if action["priority"] != "blocking" {
					t.Errorf("FormatEntityCreationJSON() action priority should be 'blocking', got %v", action["priority"])
				}

				// Check required_sections matches input
				sections, ok := action["required_sections"].([]string)
				if !ok {
					t.Error("FormatEntityCreationJSON() action missing required_sections")
				} else if len(sections) != len(tt.requiredSections) {
					t.Errorf("FormatEntityCreationJSON() action has %d sections, want %d", len(sections), len(tt.requiredSections))
				}
			}
		})
	}
}

// TestFormatEntityCreationMessageWithFileLinked tests the message when a file is linked to existing content
func TestFormatEntityCreationMessageWithFileLinked(t *testing.T) {
	projectRoot := "/home/user/projects/shark"
	filePath := filepath.Join(projectRoot, "docs/plan/E01-content/F08-indexer/prps/02-vision-api.md")

	tests := []struct {
		name             string
		entityType       string
		entityKey        string
		entityTitle      string
		filePath         string
		projectRoot      string
		fileWasLinked    bool
		requiredSections []string
		wantContains     []string
		wantNotContains  []string
	}{
		{
			name:             "Task with newly created file",
			entityType:       "task",
			entityKey:        "T-E01-F08-001",
			entityTitle:      "Implement API",
			filePath:         filePath,
			projectRoot:      projectRoot,
			fileWasLinked:    false,
			requiredSections: []string{"Implementation Plan", "Acceptance Criteria", "Test Plan"},
			wantContains: []string{
				"Created task T-E01-F08-001",
				"Implement API",
				"PLACEHOLDER FILE CREATED - EDITING REQUIRED",
				"REQUIRED ACTIONS",
				"1. Edit the task file to add implementation details",
				"2. Fill in required sections",
				"Implementation Plan",
				"Acceptance Criteria",
				"Test Plan",
			},
			wantNotContains: []string{
				"LINKED TO EXISTING FILE",
				"No action required",
			},
		},
		{
			name:             "Task with linked existing file",
			entityType:       "task",
			entityKey:        "T-E01-F08-008",
			entityTitle:      "Vision API Enhancement",
			filePath:         filePath,
			projectRoot:      projectRoot,
			fileWasLinked:    true,
			requiredSections: []string{"Implementation Plan", "Acceptance Criteria", "Test Plan"},
			wantContains: []string{
				"Created task T-E01-F08-008",
				"Vision API Enhancement",
				"LINKED TO EXISTING FILE",
				"No action required - using existing file content",
			},
			wantNotContains: []string{
				"PLACEHOLDER FILE CREATED",
				"EDITING REQUIRED",
				"REQUIRED ACTIONS",
				"1. Edit the task file",
			},
		},
		{
			name:             "Epic with linked existing file",
			entityType:       "epic",
			entityKey:        "E07",
			entityTitle:      "User Management",
			filePath:         filepath.Join(projectRoot, "docs/plan/E07-user-mgmt/epic.md"),
			projectRoot:      projectRoot,
			fileWasLinked:    true,
			requiredSections: []string{"Vision & Goals", "Success Criteria", "Features"},
			wantContains: []string{
				"Created epic E07",
				"User Management",
				"LINKED TO EXISTING FILE",
				"No action required",
			},
			wantNotContains: []string{
				"PLACEHOLDER",
				"EDITING REQUIRED",
				"REQUIRED ACTIONS",
			},
		},
		{
			name:             "Feature with newly created file",
			entityType:       "feature",
			entityKey:        "E07-F01",
			entityTitle:      "Authentication",
			filePath:         filepath.Join(projectRoot, "docs/plan/E07/F01/feature.md"),
			projectRoot:      projectRoot,
			fileWasLinked:    false,
			requiredSections: []string{"Requirements", "Design", "Test Plan"},
			wantContains: []string{
				"Created feature E07-F01",
				"Authentication",
				"PLACEHOLDER FILE CREATED - EDITING REQUIRED",
				"REQUIRED ACTIONS",
				"Requirements",
				"Design",
				"Test Plan",
			},
			wantNotContains: []string{
				"LINKED TO EXISTING FILE",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the function with the new fileWasLinked parameter
			message := FormatEntityCreationMessage(tt.entityType, tt.entityKey, tt.entityTitle, tt.filePath, tt.projectRoot, tt.fileWasLinked, tt.requiredSections)

			// Check that all required strings are present
			for _, want := range tt.wantContains {
				if !strings.Contains(message, want) {
					t.Errorf("FormatEntityCreationMessage() message missing %q\nGot:\n%s", want, message)
				}
			}

			// Check that unwanted strings are not present
			for _, notWant := range tt.wantNotContains {
				if strings.Contains(message, notWant) {
					t.Errorf("FormatEntityCreationMessage() message should not contain %q\nGot:\n%s", notWant, message)
				}
			}

			// Verify absolute path is used
			if !strings.Contains(message, tt.filePath) {
				t.Errorf("FormatEntityCreationMessage() should contain absolute path %q\nGot:\n%s", tt.filePath, message)
			}
		})
	}
}

// TestGetRequiredSectionsForEntityType tests section determination by entity type
func TestGetRequiredSectionsForEntityType(t *testing.T) {
	tests := []struct {
		name       string
		entityType string
		want       []string
	}{
		{
			name:       "Epic sections",
			entityType: "epic",
			want:       []string{"Vision & Goals", "Success Criteria", "Features"},
		},
		{
			name:       "Feature sections",
			entityType: "feature",
			want:       []string{"Requirements", "Design", "Test Plan"},
		},
		{
			name:       "Task sections",
			entityType: "task",
			want:       []string{"Implementation Plan", "Acceptance Criteria", "Test Plan"},
		},
		{
			name:       "Bug sections",
			entityType: "bug",
			want:       []string{"Description", "Steps to Reproduce", "Expected Behavior"},
		},
		{
			name:       "Change sections",
			entityType: "change",
			want:       []string{"Description", "Justification", "Impact Analysis"},
		},
		{
			name:       "Unknown entity defaults to task sections",
			entityType: "unknown",
			want:       []string{"Implementation Plan", "Acceptance Criteria", "Test Plan"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetRequiredSectionsForEntityType(tt.entityType)

			if len(got) != len(tt.want) {
				t.Errorf("GetRequiredSectionsForEntityType() returned %d sections, want %d", len(got), len(tt.want))
				return
			}

			for i, section := range tt.want {
				if got[i] != section {
					t.Errorf("GetRequiredSectionsForEntityType() section[%d] = %q, want %q", i, got[i], section)
				}
			}
		})
	}
}

// TestFormatEntityCreationJSON_BugNextCommands tests that bug entity type
// produces correct next_commands in JSON output.
func TestFormatEntityCreationJSON_BugNextCommands(t *testing.T) {
	projectRoot := "/home/user/projects/shark"
	filePath := filepath.Join(projectRoot, "docs/plan/bugs/B001.md")

	result := FormatEntityCreationJSON("bug", "B001", "Login page crash", filePath, projectRoot,
		[]string{"Description", "Steps to Reproduce", "Expected Behavior"})

	// Verify entity_type
	if result["entity_type"] != "bug" {
		t.Errorf("FormatEntityCreationJSON() entity_type = %v, want %q", result["entity_type"], "bug")
	}

	// Verify next_commands contains bug get command
	nextCommands, ok := result["next_commands"].([]string)
	if !ok {
		t.Fatalf("FormatEntityCreationJSON() next_commands missing or wrong type, got %T", result["next_commands"])
	}

	if len(nextCommands) == 0 {
		t.Fatal("FormatEntityCreationJSON() next_commands should not be empty for bug")
	}

	foundGetCmd := false
	for _, cmd := range nextCommands {
		if strings.Contains(cmd, "shark bug get B001") {
			foundGetCmd = true
			break
		}
	}
	if !foundGetCmd {
		t.Errorf("FormatEntityCreationJSON() next_commands should contain 'shark bug get B001', got %v", nextCommands)
	}
}

// TestFormatEntityCreationJSON_ChangeNextCommands tests that change entity type
// produces correct next_commands in JSON output.
func TestFormatEntityCreationJSON_ChangeNextCommands(t *testing.T) {
	projectRoot := "/home/user/projects/shark"
	filePath := filepath.Join(projectRoot, "docs/plan/changes/C001.md")

	result := FormatEntityCreationJSON("change", "C001", "Update auth flow", filePath, projectRoot,
		[]string{"Description", "Justification", "Impact Analysis"})

	// Verify entity_type
	if result["entity_type"] != "change" {
		t.Errorf("FormatEntityCreationJSON() entity_type = %v, want %q", result["entity_type"], "change")
	}

	// Verify next_commands contains change get command
	nextCommands, ok := result["next_commands"].([]string)
	if !ok {
		t.Fatalf("FormatEntityCreationJSON() next_commands missing or wrong type, got %T", result["next_commands"])
	}

	if len(nextCommands) == 0 {
		t.Fatal("FormatEntityCreationJSON() next_commands should not be empty for change")
	}

	foundGetCmd := false
	for _, cmd := range nextCommands {
		if strings.Contains(cmd, "shark change get C001") {
			foundGetCmd = true
			break
		}
	}
	if !foundGetCmd {
		t.Errorf("FormatEntityCreationJSON() next_commands should contain 'shark change get C001', got %v", nextCommands)
	}
}
