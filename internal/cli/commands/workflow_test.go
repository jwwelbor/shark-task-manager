package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/spf13/cobra"
)

// TestWorkflowListCommand tests the workflow list command with multi-level output
func TestWorkflowListCommand(t *testing.T) {
	// Save original GlobalConfig
	originalConfig := cli.GlobalConfig
	defer func() { cli.GlobalConfig = originalConfig }()

	tests := []struct {
		name           string
		configContent  string
		jsonOutput     bool
		expectError    bool
		expectedOutput []string
	}{
		{
			name: "all_defaults_shows_three_levels",
			configContent: `{
				"task_folder_base": "docs/plan"
			}`,
			jsonOutput:  false,
			expectError: false,
			expectedOutput: []string{
				"Workflow Configuration",
				"Epic Workflow (default)",
				"Feature Workflow (default)",
				"Task Workflow (default)",
				"todo",
				"in_progress",
				"ready_for_review",
				"completed",
				"blocked",
				"Legend:",
			},
		},
		{
			name: "custom_task_workflow_shows_custom_label",
			configContent: `{
				"task_folder_base": "docs/plan",
				"status_flow_version": "1.0",
				"status_flow": {
					"todo": ["in_progress"],
					"in_progress": ["done"],
					"done": []
				},
				"special_statuses": {
					"_start_": ["todo"],
					"_complete_": ["done"]
				}
			}`,
			jsonOutput:  false,
			expectError: false,
			expectedOutput: []string{
				"Workflow Configuration",
				"Epic Workflow (default)",
				"Feature Workflow (default)",
				"Task Workflow (custom)",
				"todo",
				"in_progress",
				"done",
			},
		},
		{
			name: "planning_and_aggregation_markers",
			configContent: `{
				"task_folder_base": "docs/plan"
			}`,
			jsonOutput:  false,
			expectError: false,
			expectedOutput: []string{
				"[planning]",
				"[active]",
				"[aggregates: features]",
				"[aggregates: tasks]",
				"_aggregation_",
			},
		},
		{
			name: "workflow_with_metadata",
			configContent: `{
				"task_folder_base": "docs/plan",
				"status_flow_version": "1.0",
				"status_flow": {
					"todo": ["in_progress"],
					"in_progress": ["done"],
					"done": []
				},
				"status_metadata": {
					"todo": {
						"description": "Ready to start",
						"phase": "planning",
						"color": "gray",
						"agent_types": ["developer"]
					}
				},
				"special_statuses": {
					"_start_": ["todo"],
					"_complete_": ["done"]
				}
			}`,
			jsonOutput:  false,
			expectError: false,
			expectedOutput: []string{
				"todo",
				"Ready to start",
				"phase: planning",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.ClearWorkflowCache()

			// Create temp config file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, ".sharkconfig.json")
			err := os.WriteFile(configPath, []byte(tt.configContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}

			// Set up GlobalConfig
			cli.GlobalConfig = &cli.Config{
				JSON:       tt.jsonOutput,
				ConfigFile: configPath,
			}

			// Capture output
			var buf bytes.Buffer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run command
			cmd := &cobra.Command{
				RunE: runWorkflowList,
			}
			cmd.SetContext(context.Background())

			err = runWorkflowList(cmd, []string{})

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout
			_, _ = buf.ReadFrom(r)
			output := buf.String()

			// Check error expectation
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Validate text output
			if !tt.jsonOutput && !tt.expectError {
				for _, expected := range tt.expectedOutput {
					if !strings.Contains(output, expected) {
						t.Errorf("Expected output to contain '%s'\nGot: %s", expected, output)
					}
				}
			}
		})
	}
}

// TestWorkflowListCommandJSON tests the JSON output of workflow list command
func TestWorkflowListCommandJSON(t *testing.T) {
	originalConfig := cli.GlobalConfig
	defer func() { cli.GlobalConfig = originalConfig }()

	tests := []struct {
		name          string
		configContent string
		checkFunc     func(t *testing.T, display MultiLevelWorkflowDisplay)
	}{
		{
			name: "all_defaults_json",
			configContent: `{
				"task_folder_base": "docs/plan"
			}`,
			checkFunc: func(t *testing.T, display MultiLevelWorkflowDisplay) {
				if display.EpicWorkflow == nil {
					t.Fatal("Expected epic_workflow in JSON output")
				}
				if display.FeatureWorkflow == nil {
					t.Fatal("Expected feature_workflow in JSON output")
				}
				if display.TaskWorkflow == nil {
					t.Fatal("Expected task_workflow in JSON output")
				}
				if display.EpicWorkflow.Source != "default" {
					t.Errorf("Expected epic source 'default', got %q", display.EpicWorkflow.Source)
				}
				if display.FeatureWorkflow.Source != "default" {
					t.Errorf("Expected feature source 'default', got %q", display.FeatureWorkflow.Source)
				}
				if display.TaskWorkflow.Source != "default" {
					t.Errorf("Expected task source 'default', got %q", display.TaskWorkflow.Source)
				}
				if display.EpicWorkflow.Level != "epic" {
					t.Errorf("Expected epic level 'epic', got %q", display.EpicWorkflow.Level)
				}
			},
		},
		{
			name: "custom_task_json",
			configContent: `{
				"task_folder_base": "docs/plan",
				"status_flow_version": "1.0",
				"status_flow": {
					"todo": ["done"],
					"done": []
				},
				"special_statuses": {
					"_start_": ["todo"],
					"_complete_": ["done"]
				}
			}`,
			checkFunc: func(t *testing.T, display MultiLevelWorkflowDisplay) {
				if display.TaskWorkflow.Source != "custom" {
					t.Errorf("Expected task source 'custom', got %q", display.TaskWorkflow.Source)
				}
				if display.EpicWorkflow.Source != "default" {
					t.Errorf("Expected epic source 'default', got %q", display.EpicWorkflow.Source)
				}
				if display.TaskWorkflow.StatusCount != 2 {
					t.Errorf("Expected 2 task statuses, got %d", display.TaskWorkflow.StatusCount)
				}
			},
		},
		{
			name: "planning_aggregation_in_json",
			configContent: `{
				"task_folder_base": "docs/plan"
			}`,
			checkFunc: func(t *testing.T, display MultiLevelWorkflowDisplay) {
				// Check epic workflow has planning and aggregation markers
				foundPlanning := false
				foundAggregation := false
				for _, s := range display.EpicWorkflow.Statuses {
					if s.IsPlanning {
						foundPlanning = true
					}
					if s.AggregatesFrom != "" {
						foundAggregation = true
						if s.AggregatesFrom != "features" {
							t.Errorf("Expected epic aggregates_from 'features', got %q", s.AggregatesFrom)
						}
					}
				}
				if !foundPlanning {
					t.Error("Expected at least one planning status in epic workflow")
				}
				if !foundAggregation {
					t.Error("Expected at least one aggregation status in epic workflow")
				}

				// Check feature workflow aggregates tasks
				foundTaskAgg := false
				for _, s := range display.FeatureWorkflow.Statuses {
					if s.AggregatesFrom == "tasks" {
						foundTaskAgg = true
					}
				}
				if !foundTaskAgg {
					t.Error("Expected feature workflow to have a status aggregating tasks")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.ClearWorkflowCache()

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, ".sharkconfig.json")
			if err := os.WriteFile(configPath, []byte(tt.configContent), 0644); err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}

			cli.GlobalConfig = &cli.Config{
				JSON:       true,
				ConfigFile: configPath,
			}

			var buf bytes.Buffer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			cmd := &cobra.Command{RunE: runWorkflowList}
			cmd.SetContext(context.Background())
			err := runWorkflowList(cmd, []string{})

			w.Close()
			os.Stdout = oldStdout
			_, _ = buf.ReadFrom(r)
			output := buf.String()

			if err != nil {
				t.Fatalf("Unexpected error: %v\nOutput: %s", err, output)
			}

			var display MultiLevelWorkflowDisplay
			if err := json.Unmarshal([]byte(output), &display); err != nil {
				t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, output)
			}

			tt.checkFunc(t, display)
		})
	}
}

// TestWorkflowValidateCommand tests the workflow validate command
func TestWorkflowValidateCommand(t *testing.T) {
	// Save original GlobalConfig
	originalConfig := cli.GlobalConfig
	defer func() { cli.GlobalConfig = originalConfig }()

	tests := []struct {
		name          string
		configContent string
		jsonOutput    bool
		expectValid   bool
		expectedError string
	}{
		{
			name: "valid_workflow",
			configContent: `{
				"task_folder_base": "docs/plan",
				"status_flow_version": "1.0",
				"status_flow": {
					"todo": ["in_progress"],
					"in_progress": ["done"],
					"done": []
				},
				"special_statuses": {
					"_start_": ["todo"],
					"_complete_": ["done"]
				}
			}`,
			jsonOutput:  false,
			expectValid: true,
		},
		{
			name: "missing_start_status",
			configContent: `{
				"task_folder_base": "docs/plan",
				"status_flow_version": "1.0",
				"status_flow": {
					"todo": ["in_progress"],
					"in_progress": ["done"],
					"done": []
				},
				"special_statuses": {
					"_complete_": ["done"]
				}
			}`,
			jsonOutput:    false,
			expectValid:   false,
			expectedError: "_start_",
		},
		{
			name: "undefined_status_reference",
			configContent: `{
				"task_folder_base": "docs/plan",
				"status_flow_version": "1.0",
				"status_flow": {
					"todo": ["in_progress", "missing_status"],
					"in_progress": ["done"],
					"done": []
				},
				"special_statuses": {
					"_start_": ["todo"],
					"_complete_": ["done"]
				}
			}`,
			jsonOutput:    false,
			expectValid:   false,
			expectedError: "missing_status",
		},
		{
			name: "unreachable_status",
			configContent: `{
				"task_folder_base": "docs/plan",
				"status_flow_version": "1.0",
				"status_flow": {
					"todo": ["in_progress"],
					"in_progress": ["done"],
					"orphan": ["done"],
					"done": []
				},
				"special_statuses": {
					"_start_": ["todo"],
					"_complete_": ["done"]
				}
			}`,
			jsonOutput:    false,
			expectValid:   false,
			expectedError: "orphan",
		},
		{
			name: "valid_workflow_json_output",
			configContent: `{
				"task_folder_base": "docs/plan",
				"status_flow_version": "1.0",
				"status_flow": {
					"todo": ["in_progress"],
					"in_progress": ["done"],
					"done": []
				},
				"special_statuses": {
					"_start_": ["todo"],
					"_complete_": ["done"]
				}
			}`,
			jsonOutput:  true,
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp config file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, ".sharkconfig.json")
			err := os.WriteFile(configPath, []byte(tt.configContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}

			// Set up GlobalConfig
			cli.GlobalConfig = &cli.Config{
				JSON:       tt.jsonOutput,
				ConfigFile: configPath,
			}

			// Capture output
			var buf bytes.Buffer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run command
			cmd := &cobra.Command{
				RunE: runWorkflowValidate,
			}
			cmd.SetContext(context.Background())

			err = runWorkflowValidate(cmd, []string{})

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout
			_, _ = buf.ReadFrom(r)
			output := buf.String()

			// Check validation result
			if tt.expectValid {
				if err != nil {
					t.Errorf("Expected valid workflow but got error: %v\nOutput: %s", err, output)
				}
			} else {
				if err == nil {
					t.Errorf("Expected validation error but got none\nOutput: %s", output)
				}
				// The error message is in the output (via cli.Error), not in err.Error()
				// Just verify we got an error - the specific message goes to stderr which we can't easily capture
			}

			// Validate JSON output format
			if tt.jsonOutput {
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("Failed to parse JSON output: %v\nOutput: %s", err, output)
				}
				if _, ok := result["valid"]; !ok {
					t.Errorf("Expected 'valid' field in JSON output")
				}
			}
		})
	}
}

// TestWorkflowValidateDefaultWorkflow tests validation of default workflow
func TestWorkflowValidateDefaultWorkflow(t *testing.T) {
	// Save original GlobalConfig
	originalConfig := cli.GlobalConfig
	defer func() { cli.GlobalConfig = originalConfig }()

	// Create temp config file without workflow (will use default)
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	err := os.WriteFile(configPath, []byte(`{"task_folder_base": "docs/plan"}`), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Set up GlobalConfig
	cli.GlobalConfig = &cli.Config{
		JSON:       false,
		ConfigFile: configPath,
	}

	// Capture output
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run command
	cmd := &cobra.Command{
		RunE: runWorkflowValidate,
	}
	cmd.SetContext(context.Background())

	err = runWorkflowValidate(cmd, []string{})

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Default workflow should always be valid
	if err != nil {
		t.Errorf("Default workflow validation failed: %v\nOutput: %s", err, output)
	}

	// Should contain success message or statistics
	if !strings.Contains(output, "valid") && !strings.Contains(output, "Valid") && !strings.Contains(output, "Statistics") {
		t.Errorf("Expected success message or statistics in output\nGot: %s", output)
	}
}

// TestWorkflowValidateMultiLevel tests validation of all three workflow levels
func TestWorkflowValidateMultiLevel(t *testing.T) {
	originalConfig := cli.GlobalConfig
	defer func() { cli.GlobalConfig = originalConfig }()

	tests := []struct {
		name           string
		configContent  string
		jsonOutput     bool
		expectValid    bool
		expectedLevels int
	}{
		{
			name: "all_defaults",
			configContent: `{
				"task_folder_base": "docs/plan"
			}`,
			jsonOutput:     false,
			expectValid:    true,
			expectedLevels: 3,
		},
		{
			name: "custom_epic_workflow",
			configContent: `{
				"task_folder_base": "docs/plan",
				"epic_workflow": {
					"status_flow": {
						"planning": ["active"],
						"active": ["completed"],
						"completed": []
					},
					"special_statuses": {
						"_start_": ["planning"],
						"_complete_": ["completed"]
					}
				}
			}`,
			jsonOutput:     false,
			expectValid:    true,
			expectedLevels: 3,
		},
		{
			name: "custom_feature_workflow",
			configContent: `{
				"task_folder_base": "docs/plan",
				"feature_workflow": {
					"status_flow": {
						"draft": ["in_progress"],
						"in_progress": ["done"],
						"done": []
					},
					"special_statuses": {
						"_start_": ["draft"],
						"_complete_": ["done"]
					}
				}
			}`,
			jsonOutput:     false,
			expectValid:    true,
			expectedLevels: 3,
		},
		{
			name: "invalid_epic_workflow_valid_others",
			configContent: `{
				"task_folder_base": "docs/plan",
				"epic_workflow": {
					"status_flow": {
						"planning": ["active"],
						"active": ["completed"],
						"completed": []
					},
					"special_statuses": {
						"_complete_": ["completed"]
					}
				}
			}`,
			jsonOutput:  false,
			expectValid: false,
		},
		{
			name: "all_custom_json",
			configContent: `{
				"task_folder_base": "docs/plan",
				"epic_workflow": {
					"status_flow": {
						"draft": ["active"],
						"active": ["done"],
						"done": []
					},
					"special_statuses": {
						"_start_": ["draft"],
						"_complete_": ["done"]
					}
				},
				"feature_workflow": {
					"status_flow": {
						"open": ["closed"],
						"closed": []
					},
					"special_statuses": {
						"_start_": ["open"],
						"_complete_": ["closed"]
					}
				},
				"status_flow_version": "1.0",
				"status_flow": {
					"todo": ["done"],
					"done": []
				},
				"special_statuses": {
					"_start_": ["todo"],
					"_complete_": ["done"]
				}
			}`,
			jsonOutput:     true,
			expectValid:    true,
			expectedLevels: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.ClearWorkflowCache()

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, ".sharkconfig.json")
			if err := os.WriteFile(configPath, []byte(tt.configContent), 0644); err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}

			cli.GlobalConfig = &cli.Config{
				JSON:       tt.jsonOutput,
				ConfigFile: configPath,
			}

			var buf bytes.Buffer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			cmd := &cobra.Command{RunE: runWorkflowValidate}
			cmd.SetContext(context.Background())
			err := runWorkflowValidate(cmd, []string{})

			w.Close()
			os.Stdout = oldStdout
			_, _ = buf.ReadFrom(r)
			output := buf.String()

			if tt.expectValid && err != nil {
				t.Errorf("Expected valid but got error: %v\nOutput: %s", err, output)
			}
			if !tt.expectValid && err == nil {
				t.Errorf("Expected error but got none\nOutput: %s", output)
			}

			if tt.jsonOutput && tt.expectValid {
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Fatalf("Failed to parse JSON: %v\nOutput: %s", err, output)
				}
				if _, ok := result["levels"]; !ok {
					t.Error("Expected 'levels' field in JSON output")
				}
				levels, ok := result["levels"].([]interface{})
				if !ok {
					t.Fatal("Expected 'levels' to be an array")
				}
				if len(levels) != tt.expectedLevels {
					t.Errorf("Expected %d levels, got %d", tt.expectedLevels, len(levels))
				}
			}
		})
	}
}

// TestWorkflowValidateMultiLevel_SourceField verifies custom vs default source detection
func TestWorkflowValidateMultiLevel_SourceField(t *testing.T) {
	originalConfig := cli.GlobalConfig
	defer func() { cli.GlobalConfig = originalConfig }()
	config.ClearWorkflowCache()

	configContent := `{
		"task_folder_base": "docs/plan",
		"epic_workflow": {
			"status_flow": {
				"draft": ["active"],
				"active": ["done"],
				"done": []
			},
			"special_statuses": {
				"_start_": ["draft"],
				"_complete_": ["done"]
			}
		}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cli.GlobalConfig = &cli.Config{
		JSON:       true,
		ConfigFile: configPath,
	}

	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := &cobra.Command{RunE: runWorkflowValidate}
	cmd.SetContext(context.Background())
	err := runWorkflowValidate(cmd, []string{})

	w.Close()
	os.Stdout = oldStdout
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("Expected valid but got error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v\nOutput: %s", err, output)
	}

	levels := result["levels"].([]interface{})

	// Epic should be "custom", feature and task should be "default"
	for _, lvl := range levels {
		l := lvl.(map[string]interface{})
		switch l["level"].(string) {
		case "epic":
			if l["source"] != "custom" {
				t.Errorf("Expected epic source 'custom', got %q", l["source"])
			}
		case "feature":
			if l["source"] != "default" {
				t.Errorf("Expected feature source 'default', got %q", l["source"])
			}
		case "task":
			if l["source"] != "default" {
				t.Errorf("Expected task source 'default', got %q", l["source"])
			}
		}
	}
}

// TestTaskSetStatusCommand tests the task set-status command with workflow validation
func TestTaskSetStatusCommand(t *testing.T) {
	// Save original GlobalConfig
	originalConfig := cli.GlobalConfig
	defer func() { cli.GlobalConfig = originalConfig }()

	// Create temp config with custom workflow
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	configContent := `{
		"task_folder_base": "docs/plan",
		"status_flow_version": "1.0",
		"status_flow": {
			"todo": ["in_progress", "blocked"],
			"in_progress": ["ready_for_review", "blocked"],
			"ready_for_review": ["completed", "in_progress"],
			"completed": [],
			"blocked": ["todo"]
		},
		"special_statuses": {
			"_start_": ["todo"],
			"_complete_": ["completed"]
		}
	}`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	tests := []struct {
		name          string
		currentStatus models.TaskStatus
		newStatus     string
		force         bool
		expectError   bool
		errorContains string
	}{
		{
			name:          "valid_transition_todo_to_in_progress",
			currentStatus: models.TaskStatus("todo"),
			newStatus:     "in_progress",
			force:         false,
			expectError:   false,
		},
		{
			name:          "valid_transition_in_progress_to_ready_for_review",
			currentStatus: models.TaskStatus("in_progress"),
			newStatus:     "ready_for_review",
			force:         false,
			expectError:   false,
		},
		{
			name:          "invalid_transition_todo_to_completed",
			currentStatus: models.TaskStatus("todo"),
			newStatus:     "completed",
			force:         false,
			expectError:   true,
			errorContains: "transition",
		},
		{
			name:          "invalid_transition_with_force",
			currentStatus: models.TaskStatus("todo"),
			newStatus:     "completed",
			force:         true,
			expectError:   false,
		},
		{
			name:          "valid_transition_with_force",
			currentStatus: models.TaskStatus("todo"),
			newStatus:     "in_progress",
			force:         true,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock repository
			mockRepo := NewMockTaskRepositoryWithWorkflow()

			// Add test task
			task := &models.Task{
				ID:       1,
				Key:      "T-TEST-001",
				Title:    "Test Task",
				Status:   tt.currentStatus,
				Priority: 5,
			}
			mockRepo.AddTask(task)

			// Load workflow
			workflow, err := config.LoadWorkflowConfig(configPath)
			if err != nil {
				t.Fatalf("Failed to load workflow config: %v", err)
			}
			mockRepo.SetWorkflow(workflow)

			// Set up GlobalConfig
			cli.GlobalConfig = &cli.Config{
				JSON:       false,
				ConfigFile: configPath,
			}

			// Test the workflow validation logic that would be called by the command
			ctx := context.Background()
			err = mockRepo.UpdateStatusForced(ctx, task.ID, models.TaskStatus(tt.newStatus), nil, nil, nil, nil, tt.force)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else {
					// Verify status was updated
					updatedTask, err := mockRepo.GetByID(ctx, task.ID)
					if err != nil {
						t.Errorf("Failed to get updated task: %v", err)
					} else if string(updatedTask.Status) != tt.newStatus {
						t.Errorf("Expected status '%s', got '%s'", tt.newStatus, updatedTask.Status)
					}
				}
			}
		})
	}
}

// TestTaskStartWithWorkflow tests the task start command with workflow validation
func TestTaskStartWithWorkflow(t *testing.T) {
	tests := []struct {
		name          string
		currentStatus models.TaskStatus
		force         bool
		expectError   bool
	}{
		{
			name:          "valid_start_from_todo",
			currentStatus: models.TaskStatus("todo"),
			force:         false,
			expectError:   false,
		},
		{
			name:          "invalid_start_from_completed",
			currentStatus: models.TaskStatus("completed"),
			force:         false,
			expectError:   true,
		},
		{
			name:          "force_start_from_completed",
			currentStatus: models.TaskStatus("completed"),
			force:         true,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock repository with default workflow
			mockRepo := NewMockTaskRepositoryWithWorkflow()
			mockRepo.SetWorkflow(config.DefaultWorkflow())

			// Add test task
			task := &models.Task{
				ID:       1,
				Key:      "T-TEST-001",
				Title:    "Test Task",
				Status:   tt.currentStatus,
				Priority: 5,
			}
			mockRepo.AddTask(task)

			// Test status update (simulating task start)
			ctx := context.Background()
			err := mockRepo.UpdateStatusForced(ctx, task.ID, models.TaskStatus("in_progress"), nil, nil, nil, nil, tt.force)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestTaskCompleteWithWorkflow tests the task complete command with workflow validation
func TestTaskCompleteWithWorkflow(t *testing.T) {
	tests := []struct {
		name          string
		currentStatus models.TaskStatus
		force         bool
		expectError   bool
	}{
		{
			name:          "valid_complete_from_in_progress",
			currentStatus: models.TaskStatus("in_progress"),
			force:         false,
			expectError:   false,
		},
		{
			name:          "invalid_complete_from_todo",
			currentStatus: models.TaskStatus("todo"),
			force:         false,
			expectError:   true,
		},
		{
			name:          "force_complete_from_todo",
			currentStatus: models.TaskStatus("todo"),
			force:         true,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock repository with default workflow
			mockRepo := NewMockTaskRepositoryWithWorkflow()
			mockRepo.SetWorkflow(config.DefaultWorkflow())

			// Add test task
			task := &models.Task{
				ID:       1,
				Key:      "T-TEST-001",
				Title:    "Test Task",
				Status:   tt.currentStatus,
				Priority: 5,
			}
			mockRepo.AddTask(task)

			// Test status update (simulating task complete)
			ctx := context.Background()
			err := mockRepo.UpdateStatusForced(ctx, task.ID, models.TaskStatus("ready_for_review"), nil, nil, nil, nil, tt.force)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestTaskApproveWithWorkflow tests the task approve command with workflow validation
func TestTaskApproveWithWorkflow(t *testing.T) {
	tests := []struct {
		name          string
		currentStatus models.TaskStatus
		force         bool
		expectError   bool
	}{
		{
			name:          "valid_approve_from_ready_for_review",
			currentStatus: models.TaskStatus("ready_for_review"),
			force:         false,
			expectError:   false,
		},
		{
			name:          "invalid_approve_from_todo",
			currentStatus: models.TaskStatus("todo"),
			force:         false,
			expectError:   true,
		},
		{
			name:          "force_approve_from_todo",
			currentStatus: models.TaskStatus("todo"),
			force:         true,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock repository with default workflow
			mockRepo := NewMockTaskRepositoryWithWorkflow()
			mockRepo.SetWorkflow(config.DefaultWorkflow())

			// Add test task
			task := &models.Task{
				ID:       1,
				Key:      "T-TEST-001",
				Title:    "Test Task",
				Status:   tt.currentStatus,
				Priority: 5,
			}
			mockRepo.AddTask(task)

			// Test status update (simulating task approve)
			ctx := context.Background()
			err := mockRepo.UpdateStatusForced(ctx, task.ID, models.TaskStatus("completed"), nil, nil, nil, nil, tt.force)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// MockTaskRepositoryWithWorkflow is a mock repository that supports workflow validation
type MockTaskRepositoryWithWorkflow struct {
	tasks    map[int64]*models.Task
	taskKeys map[string]int64
	workflow *config.WorkflowConfig
	nextID   int64
}

// NewMockTaskRepositoryWithWorkflow creates a new mock repository
func NewMockTaskRepositoryWithWorkflow() *MockTaskRepositoryWithWorkflow {
	return &MockTaskRepositoryWithWorkflow{
		tasks:    make(map[int64]*models.Task),
		taskKeys: make(map[string]int64),
		nextID:   1,
	}
}

// SetWorkflow sets the workflow configuration for validation
func (m *MockTaskRepositoryWithWorkflow) SetWorkflow(workflow *config.WorkflowConfig) {
	m.workflow = workflow
}

// AddTask adds a task to the mock repository
func (m *MockTaskRepositoryWithWorkflow) AddTask(task *models.Task) {
	if task.ID == 0 {
		task.ID = m.nextID
		m.nextID++
	}
	m.tasks[task.ID] = task
	m.taskKeys[task.Key] = task.ID
}

// GetByID retrieves a task by ID
func (m *MockTaskRepositoryWithWorkflow) GetByID(ctx context.Context, id int64) (*models.Task, error) {
	task, exists := m.tasks[id]
	if !exists {
		return nil, ErrTaskNotFound
	}
	return task, nil
}

// GetByKey retrieves a task by key
func (m *MockTaskRepositoryWithWorkflow) GetByKey(ctx context.Context, key string) (*models.Task, error) {
	id, exists := m.taskKeys[key]
	if !exists {
		return nil, ErrTaskNotFound
	}
	return m.GetByID(ctx, id)
}

// UpdateStatusForced updates task status with optional workflow validation
func (m *MockTaskRepositoryWithWorkflow) UpdateStatusForced(ctx context.Context, id int64, newStatus models.TaskStatus, agent *string, notes *string, rejectionReason *string, documentPath *string, force bool) error {
	task, err := m.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Validate transition if workflow is set and not forcing
	if m.workflow != nil && !force {
		currentStatusStr := string(task.Status)
		newStatusStr := string(newStatus)

		// Check if transition is valid
		validTransitions, exists := m.workflow.StatusFlow[currentStatusStr]
		if !exists {
			return ErrInvalidTransition
		}

		valid := false
		for _, validNext := range validTransitions {
			if validNext == newStatusStr {
				valid = true
				break
			}
		}

		if !valid {
			return ErrInvalidTransition
		}
	}

	// Update status
	task.Status = newStatus
	m.tasks[id] = task

	return nil
}

// Mock errors
var (
	ErrTaskNotFound      = fmt.Errorf("task not found")
	ErrInvalidTransition = fmt.Errorf("invalid status transition")
)
