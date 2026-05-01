package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestFormatEntityCreationMessage_Placeholder covers the not-linked branch:
// the message must announce that a placeholder file was created and instruct
// the user to edit it.
func TestFormatEntityCreationMessage_Placeholder(t *testing.T) {
	projectRoot := "/home/user/projects/shark"
	filePath := filepath.Join(projectRoot, "docs/plan/E07-epic-name/epic.md")

	tests := []struct {
		name        string
		entityType  string
		entityKey   string
		entityTitle string
	}{
		{"epic", "epic", "E07", "User Management System"},
		{"feature", "feature", "E07-F01", "Authentication & Authorization"},
		{"task", "task", "T-E07-F01-001", "Implement JWT validation"},
		{"bug", "bug", "B001", "Login page crashes"},
		{"change", "change", "CC-001", "Update auth flow"},
		{"tech-debt", "tech-debt", "TD-001", "Refactor module"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := FormatEntityCreationMessage(tt.entityType, tt.entityKey, tt.entityTitle, filePath, projectRoot, false)

			wantContains := []string{
				"Created " + tt.entityType + " " + tt.entityKey,
				tt.entityTitle,
				"PLACEHOLDER FILE CREATED - EDITING REQUIRED",
				filePath,
				"Edit the file",
			}
			for _, want := range wantContains {
				if !strings.Contains(msg, want) {
					t.Errorf("missing %q in:\n%s", want, msg)
				}
			}
			if strings.Contains(msg, "LINKED TO EXISTING FILE") {
				t.Errorf("did not expect linked banner in placeholder branch:\n%s", msg)
			}
		})
	}
}

// TestFormatEntityCreationMessage_Linked covers the file-linked branch.
func TestFormatEntityCreationMessage_Linked(t *testing.T) {
	projectRoot := "/home/user/projects/shark"
	filePath := filepath.Join(projectRoot, "docs/plan/E01-content/F08-indexer/prps/02-vision-api.md")

	msg := FormatEntityCreationMessage("task", "T-E01-F08-008", "Vision API Enhancement",
		filePath, projectRoot, true)

	wantContains := []string{
		"Created task T-E01-F08-008",
		"Vision API Enhancement",
		"LINKED TO EXISTING FILE",
		"No action required",
		filePath,
	}
	for _, want := range wantContains {
		if !strings.Contains(msg, want) {
			t.Errorf("missing %q in:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "PLACEHOLDER FILE CREATED") {
		t.Errorf("did not expect placeholder banner in linked branch:\n%s", msg)
	}
}

// TestFormatEntityCreationJSON_BasicFields verifies the structural fields the
// JSON output must always carry.
func TestFormatEntityCreationJSON_BasicFields(t *testing.T) {
	projectRoot := "/home/user/projects/shark"
	filePath := filepath.Join(projectRoot, "docs/plan/E07/epic.md")

	result := FormatEntityCreationJSON("epic", "E07", "User Management", filePath, projectRoot)

	want := map[string]interface{}{
		"status":           "created",
		"entity_type":      "epic",
		"key":              "E07",
		"title":            "User Management",
		"file_state":       "placeholder",
		"requires_editing": true,
	}
	for k, expected := range want {
		got, ok := result[k]
		if !ok {
			t.Errorf("missing field %q", k)
			continue
		}
		if got != expected {
			t.Errorf("field %q = %v, want %v", k, got, expected)
		}
	}
	if filePath := result["file_path"]; filePath == "" {
		t.Errorf("file_path should be set")
	}

	requiredActions, ok := result["required_actions"].([]map[string]interface{})
	if !ok {
		t.Fatalf("required_actions wrong type: %T", result["required_actions"])
	}
	if len(requiredActions) == 0 {
		t.Fatal("required_actions should not be empty")
	}
	if requiredActions[0]["action"] != "edit_file" {
		t.Errorf("first action should be edit_file, got %v", requiredActions[0]["action"])
	}
	if _, hasSections := requiredActions[0]["required_sections"]; hasSections {
		t.Errorf("required_sections should no longer be present in JSON output")
	}
}

// TestFormatEntityCreationJSON_NextCommands ensures every supported entity
// type produces a non-empty next_commands list and that unknown types still
// return a present-but-empty slice (never crash, never nil).
func TestFormatEntityCreationJSON_NextCommands(t *testing.T) {
	projectRoot := "/home/user/projects/shark"
	cases := []struct {
		entityType   string
		entityKey    string
		expectInList string
		expectEmpty  bool
	}{
		{"epic", "E07", "shark epic get E07", false},
		{"feature", "E07-F01", "shark feature get E07-F01", false},
		{"task", "T-E07-F01-001", "shark task next-status T-E07-F01-001", false},
		{"bug", "B001", "shark bug get B001", false},
		{"change", "CC-001", "shark change get CC-001", false},
		{"tech-debt", "TD-001", "shark td get TD-001", false},
		{"unknown", "X-1", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.entityType, func(t *testing.T) {
			result := FormatEntityCreationJSON(tc.entityType, tc.entityKey, "Title", "/file.md", projectRoot)
			cmds, ok := result["next_commands"].([]string)
			if !ok {
				t.Fatalf("next_commands missing or wrong type: %T", result["next_commands"])
			}
			if tc.expectEmpty {
				if len(cmds) != 0 {
					t.Errorf("expected empty next_commands for %q, got %v", tc.entityType, cmds)
				}
				return
			}
			found := false
			for _, c := range cmds {
				if strings.Contains(c, tc.expectInList) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected next_commands to contain %q, got %v", tc.expectInList, cmds)
			}
		})
	}
}
