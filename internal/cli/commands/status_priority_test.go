package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
)

// testWorkflowConfig is a self-contained workflow config for tests.
// Tests must NOT rely on the project's .sharkconfig.json.
// This config uses the default epic/feature/task workflow statuses.
const testWorkflowConfig = `{
	"status_flow": {
		"todo": ["in_progress", "blocked"],
		"in_progress": ["ready_for_review", "blocked"],
		"ready_for_review": ["completed", "in_progress"],
		"completed": [],
		"blocked": ["todo", "in_progress"]
	},
	"status_metadata": {
		"todo": {"color": "gray", "phase": "planning", "description": "Not started"},
		"in_progress": {"color": "blue", "phase": "development", "description": "In progress"},
		"ready_for_review": {"color": "yellow", "phase": "review", "description": "Awaiting review"},
		"completed": {"color": "green", "phase": "done", "description": "Done"},
		"blocked": {"color": "red", "phase": "blocked", "description": "Blocked"}
	},
	"special_statuses": {
		"_start_": ["todo"],
		"_complete_": ["completed"]
	},
	"epic_workflow": {
		"version": "1.0",
		"status_flow": {
			"draft": ["active", "archived"],
			"active": ["completed", "archived"],
			"completed": ["archived"],
			"archived": []
		},
		"status_metadata": {
			"draft": {"color": "gray", "phase": "planning", "description": "Not started"},
			"active": {"color": "blue", "phase": "execution", "description": "In progress"},
			"completed": {"color": "green", "phase": "done", "description": "Done"},
			"archived": {"color": "gray", "phase": "done", "description": "Archived"}
		},
		"special_statuses": {
			"_start_": ["draft"],
			"_complete_": ["completed", "archived"]
		}
	},
	"feature_workflow": {
		"version": "1.0",
		"status_flow": {
			"draft": ["active", "archived"],
			"active": ["completed", "archived"],
			"completed": ["archived"],
			"archived": []
		},
		"status_metadata": {
			"draft": {"color": "gray", "phase": "planning", "description": "Not started"},
			"active": {"color": "blue", "phase": "execution", "description": "In progress"},
			"completed": {"color": "green", "phase": "done", "description": "Done"},
			"archived": {"color": "gray", "phase": "done", "description": "Archived"}
		},
		"special_statuses": {
			"_start_": ["draft"],
			"_complete_": ["completed", "archived"]
		}
	}
}`

// setupTestWorkflowConfig creates a temp directory with a known .sharkconfig.json,
// changes the working directory so cli.GetWorkflowService() picks it up,
// and returns a cleanup function that restores everything.
func setupTestWorkflowConfig(t *testing.T) {
	t.Helper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	if err := os.WriteFile(configPath, []byte(testWorkflowConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to chdir to temp dir: %v", err)
	}

	config.ClearWorkflowCache()
	cli.ResetWorkflowService()

	t.Cleanup(func() {
		cli.ResetWorkflowService()
		config.ClearWorkflowCache()
		if err := os.Chdir(origWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	})
}

// TestParseEpicStatus tests parsing and validation of epic status values.
// Uses a test-specific workflow config with the default epic statuses:
// draft, active, completed, archived.
func TestParseEpicStatus(t *testing.T) {
	setupTestWorkflowConfig(t)

	tests := []struct {
		name      string
		input     string
		wantValue string
		wantErr   bool
	}{
		// Valid status values (from test config epic_workflow)
		{name: "valid draft status", input: "draft", wantValue: "draft", wantErr: false},
		{name: "valid active status", input: "active", wantValue: "active", wantErr: false},
		{name: "valid completed status", input: "completed", wantValue: "completed", wantErr: false},
		{name: "valid archived status", input: "archived", wantValue: "archived", wantErr: false},
		// Case-insensitive parsing
		{name: "uppercase DRAFT", input: "DRAFT", wantValue: "draft", wantErr: false},
		{name: "mixed case Active", input: "Active", wantValue: "active", wantErr: false},
		{name: "mixed case CoMpLeTeD", input: "CoMpLeTeD", wantValue: "completed", wantErr: false},
		// Invalid status values
		{name: "empty string", input: "", wantValue: "", wantErr: true},
		{name: "invalid status value", input: "invalid", wantValue: "", wantErr: true},
		{name: "task status in epic context", input: "in_progress", wantValue: "", wantErr: true},
		{name: "whitespace only", input: "   ", wantValue: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseEpicStatus(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseEpicStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantValue {
				t.Errorf("ParseEpicStatus() = %v, want %v", got, tt.wantValue)
			}
		})
	}
}

// TestParseFeatureStatus tests parsing and validation of feature status values.
// Uses a test-specific workflow config with the default feature statuses:
// draft, active, completed, archived.
func TestParseFeatureStatus(t *testing.T) {
	setupTestWorkflowConfig(t)

	tests := []struct {
		name      string
		input     string
		wantValue string
		wantErr   bool
	}{
		// Valid status values (from test config feature_workflow)
		{name: "valid draft status", input: "draft", wantValue: "draft", wantErr: false},
		{name: "valid active status", input: "active", wantValue: "active", wantErr: false},
		{name: "valid completed status", input: "completed", wantValue: "completed", wantErr: false},
		{name: "valid archived status", input: "archived", wantValue: "archived", wantErr: false},
		// Case-insensitive parsing
		{name: "uppercase ACTIVE", input: "ACTIVE", wantValue: "active", wantErr: false},
		{name: "mixed case Archived", input: "Archived", wantValue: "archived", wantErr: false},
		// Invalid status values
		{name: "empty string", input: "", wantValue: "", wantErr: true},
		{name: "invalid status value", input: "unknown", wantValue: "", wantErr: true},
		{name: "task status in feature context", input: "ready_for_review", wantValue: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFeatureStatus(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFeatureStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantValue {
				t.Errorf("ParseFeatureStatus() = %v, want %v", got, tt.wantValue)
			}
		})
	}
}

// TestParseTaskStatus tests parsing and validation of task status values.
// Uses a test-specific workflow config with statuses:
// todo, in_progress, ready_for_review, completed, blocked.
func TestParseTaskStatus(t *testing.T) {
	setupTestWorkflowConfig(t)

	tests := []struct {
		name      string
		input     string
		wantValue string
		wantErr   bool
	}{
		// Valid statuses from the test workflow config
		{name: "valid todo status", input: "todo", wantValue: "todo", wantErr: false},
		{name: "valid in_progress status", input: "in_progress", wantValue: "in_progress", wantErr: false},
		{name: "valid ready_for_review status", input: "ready_for_review", wantValue: "ready_for_review", wantErr: false},
		{name: "valid completed status", input: "completed", wantValue: "completed", wantErr: false},
		{name: "valid blocked status", input: "blocked", wantValue: "blocked", wantErr: false},
		// Case-insensitive parsing
		{name: "uppercase TODO", input: "TODO", wantValue: "todo", wantErr: false},
		{name: "mixed case In_Progress", input: "In_Progress", wantValue: "in_progress", wantErr: false},
		{name: "mixed case READY_FOR_REVIEW", input: "READY_FOR_REVIEW", wantValue: "ready_for_review", wantErr: false},
		// Invalid status values
		{name: "empty string", input: "", wantValue: "", wantErr: true},
		{name: "invalid status value", input: "invalid_status", wantValue: "", wantErr: true},
		{name: "epic status in task context", input: "active", wantValue: "", wantErr: true},
		{name: "whitespace only", input: "   ", wantValue: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTaskStatus(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTaskStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantValue {
				t.Errorf("ParseTaskStatus() = %v, want %v", got, tt.wantValue)
			}
		})
	}
}

// TestParseEpicPriority tests parsing and validation of epic priority values
func TestParseEpicPriority(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValue string
		wantErr   bool
	}{
		// Valid priority values
		{name: "valid low priority", input: "low", wantValue: "low", wantErr: false},
		{name: "valid medium priority", input: "medium", wantValue: "medium", wantErr: false},
		{name: "valid high priority", input: "high", wantValue: "high", wantErr: false},
		// Case-insensitive parsing
		{name: "uppercase LOW", input: "LOW", wantValue: "low", wantErr: false},
		{name: "mixed case Medium", input: "Medium", wantValue: "medium", wantErr: false},
		{name: "uppercase HIGH", input: "HIGH", wantValue: "high", wantErr: false},
		// Invalid priority values
		{name: "empty string", input: "", wantValue: "", wantErr: true},
		{name: "invalid priority value", input: "critical", wantValue: "", wantErr: true},
		{name: "numeric priority", input: "5", wantValue: "", wantErr: true},
		{name: "whitespace only", input: "   ", wantValue: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseEpicPriority(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseEpicPriority() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantValue {
				t.Errorf("ParseEpicPriority() = %v, want %v", got, tt.wantValue)
			}
		})
	}
}

// TestParseTaskPriority tests parsing and validation of task priority values (1-10)
func TestParseTaskPriority(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValue int
		wantErr   bool
	}{
		// Valid priority values
		{name: "valid priority 1", input: "1", wantValue: 1, wantErr: false},
		{name: "valid priority 5", input: "5", wantValue: 5, wantErr: false},
		{name: "valid priority 10", input: "10", wantValue: 10, wantErr: false},
		{name: "valid priority with whitespace", input: "  7  ", wantValue: 7, wantErr: false},
		// Invalid priority values - out of range
		{name: "priority 0 (below minimum)", input: "0", wantValue: 0, wantErr: true},
		{name: "priority 11 (above maximum)", input: "11", wantValue: 0, wantErr: true},
		{name: "negative priority", input: "-1", wantValue: 0, wantErr: true},
		{name: "priority 100", input: "100", wantValue: 0, wantErr: true},
		// Invalid priority values - format
		{name: "empty string", input: "", wantValue: 0, wantErr: true},
		{name: "non-numeric value", input: "high", wantValue: 0, wantErr: true},
		{name: "decimal value", input: "5.5", wantValue: 0, wantErr: true},
		{name: "whitespace only", input: "   ", wantValue: 0, wantErr: true},
		{name: "alphanumeric value", input: "p5", wantValue: 0, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTaskPriority(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTaskPriority() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantValue {
				t.Errorf("ParseTaskPriority() = %v, want %v", got, tt.wantValue)
			}
		})
	}
}

// TestParseTaskPriorityErrorMessages tests that error messages are user-friendly
func TestParseTaskPriorityErrorMessages(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantErrSubstr string
	}{
		{name: "out of range error includes range", input: "15", wantErrSubstr: "1-10"},
		{name: "non-numeric error is clear", input: "abc", wantErrSubstr: "invalid"},
		{name: "empty string error", input: "", wantErrSubstr: "empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseTaskPriority(tt.input)
			if err == nil {
				t.Errorf("ParseTaskPriority() expected error, got nil")
				return
			}
			if tt.wantErrSubstr != "" {
				errMsg := err.Error()
				if !containsSubstring(errMsg, tt.wantErrSubstr) {
					t.Errorf("ParseTaskPriority() error = %q, want substring %q", errMsg, tt.wantErrSubstr)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring (case-insensitive)
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				hasSubstring(s, substr)))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
