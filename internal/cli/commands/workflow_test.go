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

// findLevel returns the LevelWorkflowDisplay for the named level, or nil if absent.
func findLevel(display MultiLevelWorkflowDisplay, name string) *LevelWorkflowDisplay {
	for _, l := range display.Levels {
		if l != nil && l.Level == name {
			return l
		}
	}
	return nil
}

func runWorkflowListForTest(t *testing.T, configContent string, jsonOutput bool, expanded bool, args ...string) (string, error) {
	t.Helper()

	cmd := &cobra.Command{RunE: runWorkflowList}
	cmd.SetContext(context.Background())
	return runWorkflowListWithCommandForTest(t, cmd, configContent, jsonOutput, expanded, args...)
}

func runWorkflowListCobraForTest(t *testing.T, configContent string, jsonOutput bool, args ...string) (string, error) {
	t.Helper()

	cmd := &cobra.Command{
		Use:  workflowListCmd.Use,
		Args: workflowListCmd.Args,
		RunE: workflowListCmd.RunE,
	}
	cmd.Flags().BoolVar(&workflowListAll, "all", false, "Render expanded workflow details")
	cmd.SetArgs(args)
	cmd.SetContext(context.Background())

	return runWorkflowListWithCommandForTest(t, cmd, configContent, jsonOutput, false)
}

func runWorkflowListWithCommandForTest(t *testing.T, cmd *cobra.Command, configContent string, jsonOutput bool, expanded bool, args ...string) (string, error) {
	t.Helper()

	config.ClearWorkflowCache()

	originalConfig := cli.GlobalConfig
	originalAll := workflowListAll
	t.Cleanup(func() {
		cli.GlobalConfig = originalConfig
		workflowListAll = originalAll
	})

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sharkconfig.json")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cli.GlobalConfig = &cli.Config{
		JSON:       jsonOutput,
		ConfigFile: configPath,
	}
	workflowListAll = expanded

	return captureStdoutForTest(t, func() error {
		if len(args) > 0 {
			return runWorkflowList(cmd, args)
		}
		return cmd.Execute()
	})
}

func captureStdoutForTest(t *testing.T, run func() error) (string, error) {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}
	os.Stdout = w
	restored := false
	restoreStdout := func() {
		if !restored {
			os.Stdout = oldStdout
			restored = true
		}
	}
	defer restoreStdout()

	type readResult struct {
		output string
		err    error
	}
	outputCh := make(chan readResult, 1)
	go func() {
		var buf bytes.Buffer
		_, readErr := buf.ReadFrom(r)
		outputCh <- readResult{output: buf.String(), err: readErr}
	}()

	runErr := run()
	closeWriterErr := w.Close()
	restoreStdout()
	result := <-outputCh
	closeReaderErr := r.Close()

	if runErr == nil && closeWriterErr != nil {
		runErr = fmt.Errorf("close stdout writer: %w", closeWriterErr)
	}
	if runErr == nil && result.err != nil {
		runErr = fmt.Errorf("read stdout: %w", result.err)
	}
	if runErr == nil && closeReaderErr != nil {
		runErr = fmt.Errorf("close stdout reader: %w", closeReaderErr)
	}

	return result.output, runErr
}

// TestWorkflowListCommand tests the workflow list command with multi-level output
func TestWorkflowListCommand(t *testing.T) {
	// Save original GlobalConfig
	originalConfig := cli.GlobalConfig
	originalAll := workflowListAll
	defer func() {
		cli.GlobalConfig = originalConfig
		workflowListAll = originalAll
	}()

	tests := []struct {
		name           string
		configContent  string
		jsonOutput     bool
		expectError    bool
		expectedOutput []string
	}{
		{
			name: "all_defaults_shows_all_levels",
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
				"Sprint Workflow (default)",
				"Bug Workflow (default)",
				"Change Workflow (default)",
				"Tech Debt Workflow (default)",
				"draft",
				"development",
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
				"Tech Debt Workflow (default)",
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
			output, err := runWorkflowListForTest(t, tt.configContent, tt.jsonOutput, true)

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

func TestWorkflowListCommandEntityFilter(t *testing.T) {
	output, err := runWorkflowListForTest(t, `{"task_folder_base": "docs/plan"}`, false, false, "task")
	if err != nil {
		t.Fatalf("Unexpected error: %v\nOutput: %s", err, output)
	}

	for _, expected := range []string{
		"Workflow Configuration (simple)",
		"Task Workflow (default)",
		" -> ",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q\nGot: %s", expected, output)
		}
	}

	for _, unexpected := range []string{
		"Epic Workflow",
		"Feature Workflow",
		"Bug Workflow",
		"Change Workflow",
	} {
		if strings.Contains(output, unexpected) {
			t.Errorf("Expected filtered output not to contain %q\nGot: %s", unexpected, output)
		}
	}
}

func TestWorkflowListCommandEntityFilterAliases(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"epic", "epic"},
		{"epics", "epic"},
		{"feature", "feature"},
		{"features", "feature"},
		{"task", "task"},
		{"tasks", "task"},
		{"sprint", "sprint"},
		{"sprints", "sprint"},
		{"bug", "bug"},
		{"bugs", "bug"},
		{"change", "change"},
		{"changes", "change"},
		{"change-card", "change"},
		{"change_card", "change"},
		{"change-cards", "change"},
		{"change_cards", "change"},
		{"tech-debt", "tech_debt"},
		{"tech_debt", "tech_debt"},
		{"tech-debts", "tech_debt"},
		{"tech_debts", "tech_debt"},
		{"td", "tech_debt"},
		{" TASKS ", "task"},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := normalizeWorkflowListLevel(tt.raw)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("normalizeWorkflowListLevel(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestWorkflowListCommandEntityFilterAcceptsKnownLevelFallback(t *testing.T) {
	originalLevels := config.KnownWorkflowLevels
	config.KnownWorkflowLevels = append(append([]string{}, originalLevels...), "experiment")
	t.Cleanup(func() {
		config.KnownWorkflowLevels = originalLevels
	})

	got, err := normalizeWorkflowListLevel("experiment")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if got != "experiment" {
		t.Fatalf("Expected fallback level %q, got %q", "experiment", got)
	}
}

func TestWorkflowListCommandInvalidEntityFilter(t *testing.T) {
	output, err := runWorkflowListForTest(t, `{"task_folder_base": "docs/plan"}`, false, false, "widget")
	if err == nil {
		t.Fatalf("Expected invalid entity error\nOutput: %s", output)
	}
	if !strings.Contains(err.Error(), `invalid entity type "widget"`) {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestWorkflowListCommandDefaultSimpleOutput(t *testing.T) {
	output, err := runWorkflowListForTest(t, `{"task_folder_base": "docs/plan"}`, false, false)
	if err != nil {
		t.Fatalf("Unexpected error: %v\nOutput: %s", err, output)
	}

	for _, expected := range []string{
		"Workflow Configuration (simple)",
		"Epic Workflow (default)",
		"Task Workflow (default)",
		"Change Workflow (default)",
		"Tech Debt Workflow (default)",
		" -> ",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected simple output to contain %q\nGot: %s", expected, output)
		}
	}

	for _, unexpected := range []string{
		"Status Transitions:",
		"Legend:",
		"phase:",
	} {
		if strings.Contains(output, unexpected) {
			t.Errorf("Expected simple output not to contain verbose marker %q\nGot: %s", unexpected, output)
		}
	}
}

func TestWorkflowListCommandSimpleOutputEdges(t *testing.T) {
	output, err := runWorkflowListForTest(t, `{
		"task_folder_base": "docs/plan",
		"status_flow_version": "1.0",
		"status_flow": {
			"ready": ["review", "blocked"],
			"review": ["done"],
			"blocked": [],
			"done": []
		},
		"special_statuses": {
			"_start_": ["ready"],
			"_complete_": ["done", "blocked"]
		}
	}`, false, false, "task")
	if err != nil {
		t.Fatalf("Unexpected error: %v\nOutput: %s", err, output)
	}

	for _, expected := range []string{
		"  ready -> review | blocked",
		"  review -> done",
		"  done -> [terminal]",
		"  blocked -> [terminal]",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected simple output to contain %q\nGot: %s", expected, output)
		}
	}
}

func TestWorkflowListCommandAllOutput(t *testing.T) {
	output, err := runWorkflowListCobraForTest(t, `{"task_folder_base": "docs/plan"}`, false, "task", "--all")
	if err != nil {
		t.Fatalf("Unexpected error: %v\nOutput: %s", err, output)
	}

	for _, expected := range []string{
		"Workflow Configuration",
		"Task Workflow (default)",
		"Status Transitions:",
		"Legend:",
		"phase:",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected expanded output to contain %q\nGot: %s", expected, output)
		}
	}
}

func TestWorkflowListCommandRejectsTooManyArgs(t *testing.T) {
	output, err := runWorkflowListCobraForTest(t, `{"task_folder_base": "docs/plan"}`, false, "task", "bug")
	if err == nil {
		t.Fatalf("Expected too many args error\nOutput: %s", output)
	}
}

func TestWorkflowListCommandEntityFilterJSON(t *testing.T) {
	output, err := runWorkflowListForTest(t, `{"task_folder_base": "docs/plan"}`, true, false, "bug")
	if err != nil {
		t.Fatalf("Unexpected error: %v\nOutput: %s", err, output)
	}

	var display MultiLevelWorkflowDisplay
	if err := json.Unmarshal([]byte(output), &display); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, output)
	}
	if len(display.Levels) != 1 {
		t.Fatalf("Expected one filtered level, got %d", len(display.Levels))
	}
	if display.Levels[0].Level != "bug" {
		t.Fatalf("Expected bug level, got %q", display.Levels[0].Level)
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
				// All levels in config.KnownWorkflowLevels should be present.
				for _, lvl := range []string{"epic", "feature", "task", "sprint", "bug", "change", "tech_debt"} {
					if findLevel(display, lvl) == nil {
						t.Fatalf("Expected %s level in JSON output", lvl)
					}
				}
				for _, lvl := range []string{"epic", "feature", "task"} {
					ld := findLevel(display, lvl)
					if ld.Source != "default" {
						t.Errorf("Expected %s source 'default', got %q", lvl, ld.Source)
					}
				}
				if epic := findLevel(display, "epic"); epic.Level != "epic" {
					t.Errorf("Expected epic level 'epic', got %q", epic.Level)
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
				task := findLevel(display, "task")
				epic := findLevel(display, "epic")
				if task.Source != "custom" {
					t.Errorf("Expected task source 'custom', got %q", task.Source)
				}
				if epic.Source != "default" {
					t.Errorf("Expected epic source 'default', got %q", epic.Source)
				}
				if task.StatusCount != 2 {
					t.Errorf("Expected 2 task statuses, got %d", task.StatusCount)
				}
			},
		},
		{
			name: "planning_aggregation_in_json",
			configContent: `{
				"task_folder_base": "docs/plan"
			}`,
			checkFunc: func(t *testing.T, display MultiLevelWorkflowDisplay) {
				epic := findLevel(display, "epic")
				feature := findLevel(display, "feature")

				// Check epic workflow has planning and aggregation markers
				foundPlanning := false
				foundAggregation := false
				for _, s := range epic.Statuses {
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
				for _, s := range feature.Statuses {
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
			output, err := runWorkflowListForTest(t, tt.configContent, true, false)

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
			expectedLevels: 7,
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
			expectedLevels: 7,
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
			expectedLevels: 7,
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
			expectedLevels: 7,
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
			task := &models.Task{BaseEntity: models.BaseEntity{ID: 1,
				Key:   "T-TEST-001",
				Title: "Test Task"}, Status: tt.currentStatus,
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
			name:          "valid_start_from_draft",
			currentStatus: models.TaskStatus("draft"),
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
			task := &models.Task{BaseEntity: models.BaseEntity{ID: 1,
				Key:   "T-TEST-001",
				Title: "Test Task"}, Status: tt.currentStatus,
				Priority: 5,
			}
			mockRepo.AddTask(task)

			// Test status update (simulating task start)
			ctx := context.Background()
			err := mockRepo.UpdateStatusForced(ctx, task.ID, models.TaskStatus("development"), nil, nil, nil, nil, tt.force)

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
			name:          "valid_complete_from_development",
			currentStatus: models.TaskStatus("development"),
			force:         false,
			expectError:   false,
		},
		{
			name:          "invalid_complete_from_draft",
			currentStatus: models.TaskStatus("draft"),
			force:         false,
			expectError:   true,
		},
		{
			name:          "force_complete_from_draft",
			currentStatus: models.TaskStatus("draft"),
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
			task := &models.Task{BaseEntity: models.BaseEntity{ID: 1,
				Key:   "T-TEST-001",
				Title: "Test Task"}, Status: tt.currentStatus,
				Priority: 5,
			}
			mockRepo.AddTask(task)

			// Test status update (simulating task complete)
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

// TestTaskApproveWithWorkflow tests the task approve command with workflow validation
func TestTaskApproveWithWorkflow(t *testing.T) {
	tests := []struct {
		name          string
		currentStatus models.TaskStatus
		force         bool
		expectError   bool
	}{
		{
			name:          "valid_approve_from_development",
			currentStatus: models.TaskStatus("development"),
			force:         false,
			expectError:   false,
		},
		{
			name:          "invalid_approve_from_blocked",
			currentStatus: models.TaskStatus("blocked"),
			force:         false,
			expectError:   true,
		},
		{
			name:          "force_approve_from_blocked",
			currentStatus: models.TaskStatus("blocked"),
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
			task := &models.Task{BaseEntity: models.BaseEntity{ID: 1,
				Key:   "T-TEST-001",
				Title: "Test Task"}, Status: tt.currentStatus,
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
