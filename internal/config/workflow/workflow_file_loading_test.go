package workflow

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Test helpers ---

// buildWorkflowBlock creates a minimal valid workflow block with a distinguishable version marker
// and a non-empty status_flow. The version is stored in "status_flow_version" to match the
// WorkflowConfig JSON tag.
func buildWorkflowBlock(version string) map[string]interface{} {
	return map[string]interface{}{
		"status_flow_version": version,
		"status_flow": map[string]interface{}{
			"draft":     []string{"active"},
			"active":    []string{"completed"},
			"completed": []string{},
		},
		"status_metadata": map[string]interface{}{
			"draft":     map[string]interface{}{"color": "gray", "description": "Draft"},
			"active":    map[string]interface{}{"color": "blue", "description": "Active"},
			"completed": map[string]interface{}{"color": "green", "description": "Done"},
		},
		"special_statuses": map[string]interface{}{
			"_start_":    []string{"draft"},
			"_complete_": []string{"completed"},
		},
	}
}

func writeWorkflowYAML(t *testing.T, workflowDir, filename, content string) {
	t.Helper()
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		t.Fatalf("failed to create workflow dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workflowDir, filename), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write workflow YAML: %v", err)
	}
}

// buildFullConfig creates a .sharkconfig.json body with all 5 entity workflow blocks
// using prefix in version strings for identification.
func buildFullConfig(prefix string) map[string]interface{} {
	return map[string]interface{}{
		"epic_workflow":    buildWorkflowBlock(prefix + "-epic"),
		"feature_workflow": buildWorkflowBlock(prefix + "-feature"),
		"task_workflow":    buildWorkflowBlock(prefix + "-task"),
		"bug_workflow":     buildWorkflowBlock(prefix + "-bug"),
		"change_workflow":  buildWorkflowBlock(prefix + "-change"),
	}
}

// writeJSON marshals data to JSON and writes it to path. Calls t.Fatal on error.
func writeJSON(t *testing.T, path string, data interface{}) {
	t.Helper()
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create directory %s: %v", dir, err)
	}
	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}

// --- AC Group 7: Task Workflow Block Precedence (ADR-F04-004) ---

// TC-060: task_workflow block takes precedence over legacy top-level keys
func TestLoadMultiLevelWorkflow_TaskWorkflowBlock(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	// Config with BOTH task_workflow block and legacy top-level keys
	configData := map[string]interface{}{
		"task_workflow": buildWorkflowBlock("block"),
		// Legacy top-level keys (should be ignored when task_workflow block exists)
		"status_flow": map[string]interface{}{
			"todo":        []string{"in_progress"},
			"in_progress": []string{"done"},
			"done":        []string{},
		},
		"status_metadata": map[string]interface{}{
			"todo":        map[string]interface{}{"color": "gray", "description": "To do"},
			"in_progress": map[string]interface{}{"color": "yellow", "description": "In progress"},
			"done":        map[string]interface{}{"color": "green", "description": "Done"},
		},
	}
	writeJSON(t, configPath, configData)

	ClearWorkflowCache()
	result, err := LoadMultiLevelWorkflow(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Task == nil {
		t.Fatal("expected non-nil Task workflow")
	}
	if result.Task.Version != "block" {
		t.Errorf("expected Task.Version == 'block' (from task_workflow block), got %q", result.Task.Version)
	}
	// Verify it has the block's statuses, not the legacy ones
	if _, ok := result.Task.StatusFlow["draft"]; !ok {
		t.Error("expected 'draft' status from task_workflow block, not legacy 'todo'")
	}
}

// TC-061: Legacy top-level keys used when no task_workflow block
func TestLoadMultiLevelWorkflow_LegacyTaskKeysOnly(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	// Config with only legacy top-level keys (no task_workflow block)
	configData := map[string]interface{}{
		"status_flow_version": "1.0",
		"status_flow": map[string]interface{}{
			"todo":        []string{"in_progress"},
			"in_progress": []string{"done"},
			"done":        []string{},
		},
		"status_metadata": map[string]interface{}{
			"todo":        map[string]interface{}{"color": "gray", "description": "To do"},
			"in_progress": map[string]interface{}{"color": "yellow", "description": "In progress"},
			"done":        map[string]interface{}{"color": "green", "description": "Done"},
		},
	}
	writeJSON(t, configPath, configData)

	ClearWorkflowCache()
	result, err := LoadMultiLevelWorkflow(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Task == nil {
		t.Fatal("expected non-nil Task workflow from legacy keys")
	}
	if _, ok := result.Task.StatusFlow["todo"]; !ok {
		t.Error("expected 'todo' status from legacy top-level keys")
	}
}

// TC-062: Workflow file task_workflow overrides both config block and legacy keys
func TestLoadMultiLevelWorkflow_WorkflowFileTaskOverridesConfigBlock(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	// Config with task_workflow block AND legacy top-level keys
	configData := map[string]interface{}{
		"task_workflow": buildWorkflowBlock("config-block"),
		"status_flow": map[string]interface{}{
			"todo":        []string{"in_progress"},
			"in_progress": []string{"done"},
			"done":        []string{},
		},
	}
	writeJSON(t, configPath, configData)

	// Workflow file with its own task_workflow
	workflowData := map[string]interface{}{
		"task_workflow": buildWorkflowBlock("file"),
	}
	writeJSON(t, filepath.Join(dir, ".sharkworkflow.json"), workflowData)

	ClearWorkflowCache()
	result, err := LoadMultiLevelWorkflow(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Task == nil {
		t.Fatal("expected non-nil Task workflow")
	}
	if result.Task.Version != "file" {
		t.Errorf("expected Task.Version == 'file' (from workflow file), got %q", result.Task.Version)
	}
}

// --- AC Group 1: Workflow File Detection (REQ-F04-001) ---

// TC-001: Both files exist, all 5 entities from workflow file
func TestLoadMultiLevelWorkflow_WorkflowFileDetected(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	// Minimal runtime config
	writeJSON(t, configPath, map[string]interface{}{})

	// Workflow file with all 5 entities
	workflowData := map[string]interface{}{
		"epic_workflow":    buildWorkflowBlock("file-epic"),
		"feature_workflow": buildWorkflowBlock("file-feature"),
		"task_workflow":    buildWorkflowBlock("file-task"),
		"bug_workflow":     buildWorkflowBlock("file-bug"),
		"change_workflow":  buildWorkflowBlock("file-change"),
	}
	writeJSON(t, filepath.Join(dir, ".sharkworkflow.json"), workflowData)

	ClearWorkflowCache()
	result, err := LoadMultiLevelWorkflow(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Epic == nil || result.Epic.Version != "file-epic" {
		t.Errorf("expected Epic.Version == 'file-epic', got %v", result.Epic)
	}
	if result.Feature == nil || result.Feature.Version != "file-feature" {
		t.Errorf("expected Feature.Version == 'file-feature', got %v", result.Feature)
	}
	if result.Task == nil || result.Task.Version != "file-task" {
		t.Errorf("expected Task.Version == 'file-task', got %v", result.Task)
	}
	if result.Bug == nil || result.Bug.Version != "file-bug" {
		t.Errorf("expected Bug.Version == 'file-bug', got %v", result.Bug)
	}
	if result.Change == nil || result.Change.Version != "file-change" {
		t.Errorf("expected Change.Version == 'file-change', got %v", result.Change)
	}
}

// TC-002: Workflow file with only epic_workflow
func TestLoadMultiLevelWorkflow_WorkflowFileOnlyPartialEntities(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	writeJSON(t, configPath, map[string]interface{}{})

	workflowData := map[string]interface{}{
		"epic_workflow": buildWorkflowBlock("file-epic"),
	}
	writeJSON(t, filepath.Join(dir, ".sharkworkflow.json"), workflowData)

	ClearWorkflowCache()
	result, err := LoadMultiLevelWorkflow(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Epic == nil || result.Epic.Version != "file-epic" {
		t.Errorf("expected Epic from workflow file, got %v", result.Epic)
	}
	// Other levels should be nil (defaults via GetWorkflowForLevel)
	if result.Task != nil {
		t.Errorf("expected Task to be nil, got %v", result.Task)
	}
	if result.Feature != nil {
		t.Errorf("expected Feature to be nil, got %v", result.Feature)
	}
}

// TC-003: Unknown keys in workflow file are silently ignored
func TestLoadMultiLevelWorkflow_UnknownKeysIgnored(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	writeJSON(t, configPath, map[string]interface{}{})

	workflowData := map[string]interface{}{
		"task_workflow":   buildWorkflowBlock("file-task"),
		"sprint_workflow": buildWorkflowBlock("unknown"),
		"unknown_key":     "some value",
	}
	writeJSON(t, filepath.Join(dir, ".sharkworkflow.json"), workflowData)

	ClearWorkflowCache()
	result, err := LoadMultiLevelWorkflow(configPath)
	if err != nil {
		t.Fatalf("unexpected error (unknown keys should be ignored): %v", err)
	}

	if result.Task == nil || result.Task.Version != "file-task" {
		t.Errorf("expected Task from workflow file, got %v", result.Task)
	}
}

// --- AC Group 2: Backward-Compatible Fallback (REQ-F04-002) ---

// TC-010: Config-only, all 5 entities from inline blocks
func TestLoadMultiLevelWorkflow_NoWorkflowFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	configData := buildFullConfig("config")
	writeJSON(t, configPath, configData)

	ClearWorkflowCache()
	result, err := LoadMultiLevelWorkflow(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Epic == nil || result.Epic.Version != "config-epic" {
		t.Errorf("expected Epic from config, got %v", result.Epic)
	}
	if result.Feature == nil || result.Feature.Version != "config-feature" {
		t.Errorf("expected Feature from config, got %v", result.Feature)
	}
	if result.Task == nil || result.Task.Version != "config-task" {
		t.Errorf("expected Task from config, got %v", result.Task)
	}
	if result.Bug == nil || result.Bug.Version != "config-bug" {
		t.Errorf("expected Bug from config, got %v", result.Bug)
	}
	if result.Change == nil || result.Change.Version != "config-change" {
		t.Errorf("expected Change from config, got %v", result.Change)
	}
}

// TC-012: Empty config, no workflow file, all defaults
func TestLoadMultiLevelWorkflow_NoConfigNoWorkflowFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	writeJSON(t, configPath, map[string]interface{}{})

	ClearWorkflowCache()
	result, err := LoadMultiLevelWorkflow(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All levels nil -- defaults via GetWorkflowForLevel()
	if result.Epic != nil {
		t.Errorf("expected nil Epic, got %v", result.Epic)
	}
	if result.Feature != nil {
		t.Errorf("expected nil Feature, got %v", result.Feature)
	}
	if result.Task != nil {
		t.Errorf("expected nil Task, got %v", result.Task)
	}
	if result.Bug != nil {
		t.Errorf("expected nil Bug, got %v", result.Bug)
	}
	if result.Change != nil {
		t.Errorf("expected nil Change, got %v", result.Change)
	}
}

// --- AC Group 3: Per-Entity Precedence (REQ-F04-003) ---

// TC-020: File defines epic+task, config defines all 5
func TestLoadMultiLevelWorkflow_PerEntityPrecedence(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	configData := buildFullConfig("config")
	writeJSON(t, configPath, configData)

	workflowData := map[string]interface{}{
		"epic_workflow": buildWorkflowBlock("file-epic"),
		"task_workflow": buildWorkflowBlock("file-task"),
	}
	writeJSON(t, filepath.Join(dir, ".sharkworkflow.json"), workflowData)

	ClearWorkflowCache()
	result, err := LoadMultiLevelWorkflow(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Workflow file wins for epic and task
	if result.Epic == nil || result.Epic.Version != "file-epic" {
		t.Errorf("expected Epic.Version == 'file-epic', got %v", result.Epic)
	}
	if result.Task == nil || result.Task.Version != "file-task" {
		t.Errorf("expected Task.Version == 'file-task', got %v", result.Task)
	}

	// Config wins for feature, bug, change
	if result.Feature == nil || result.Feature.Version != "config-feature" {
		t.Errorf("expected Feature.Version == 'config-feature', got %v", result.Feature)
	}
	if result.Bug == nil || result.Bug.Version != "config-bug" {
		t.Errorf("expected Bug.Version == 'config-bug', got %v", result.Bug)
	}
	if result.Change == nil || result.Change.Version != "config-change" {
		t.Errorf("expected Change.Version == 'config-change', got %v", result.Change)
	}
}

// TC-021: Three tiers: file, config, defaults
func TestLoadMultiLevelWorkflow_FullPrecedenceChain(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	// Config defines only feature_workflow
	configData := map[string]interface{}{
		"feature_workflow": buildWorkflowBlock("config-feature"),
	}
	writeJSON(t, configPath, configData)

	// Workflow file defines only epic_workflow
	workflowData := map[string]interface{}{
		"epic_workflow": buildWorkflowBlock("file-epic"),
	}
	writeJSON(t, filepath.Join(dir, ".sharkworkflow.json"), workflowData)

	ClearWorkflowCache()
	result, err := LoadMultiLevelWorkflow(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Epic from file
	if result.Epic == nil || result.Epic.Version != "file-epic" {
		t.Errorf("expected Epic from file, got %v", result.Epic)
	}
	// Feature from config
	if result.Feature == nil || result.Feature.Version != "config-feature" {
		t.Errorf("expected Feature from config, got %v", result.Feature)
	}
	// Task, Bug, Change from defaults (nil)
	if result.Task != nil {
		t.Errorf("expected nil Task (defaults), got %v", result.Task)
	}
	if result.Bug != nil {
		t.Errorf("expected nil Bug (defaults), got %v", result.Bug)
	}
	if result.Change != nil {
		t.Errorf("expected nil Change (defaults), got %v", result.Change)
	}
}

// TC-022: Both files define task_workflow, file wins
func TestLoadMultiLevelWorkflow_WorkflowFileOverridesInline(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	configData := map[string]interface{}{
		"task_workflow": buildWorkflowBlock("config-task"),
	}
	writeJSON(t, configPath, configData)

	workflowData := map[string]interface{}{
		"task_workflow": buildWorkflowBlock("file-task"),
	}
	writeJSON(t, filepath.Join(dir, ".sharkworkflow.json"), workflowData)

	ClearWorkflowCache()
	result, err := LoadMultiLevelWorkflow(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Task == nil || result.Task.Version != "file-task" {
		t.Errorf("expected Task.Version == 'file-task' (workflow file wins), got %v", result.Task)
	}
}

// TC-023: Empty entity block in workflow file treated as "not defined"
func TestLoadMultiLevelWorkflow_EmptyEntityInWorkflowFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	configData := map[string]interface{}{
		"task_workflow": buildWorkflowBlock("config-task"),
	}
	writeJSON(t, configPath, configData)

	// Workflow file has empty task_workflow block
	workflowData := map[string]interface{}{
		"task_workflow": map[string]interface{}{},
	}
	writeJSON(t, filepath.Join(dir, ".sharkworkflow.json"), workflowData)

	ClearWorkflowCache()
	result, err := LoadMultiLevelWorkflow(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Empty block treated as "not defined" -- config value used
	if result.Task == nil || result.Task.Version != "config-task" {
		t.Errorf("expected Task from config (empty block = not defined), got %v", result.Task)
	}
}

// TC-024: Empty status_flow in workflow file treated as nil
func TestLoadMultiLevelWorkflow_EmptyStatusFlowInWorkflowFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	configData := map[string]interface{}{
		"task_workflow": buildWorkflowBlock("config-task"),
	}
	writeJSON(t, configPath, configData)

	workflowData := map[string]interface{}{
		"task_workflow": map[string]interface{}{
			"version":     "file-task",
			"status_flow": map[string]interface{}{},
		},
	}
	writeJSON(t, filepath.Join(dir, ".sharkworkflow.json"), workflowData)

	ClearWorkflowCache()
	result, err := LoadMultiLevelWorkflow(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Empty status_flow treated as nil per parseWorkflowSection() behavior
	if result.Task == nil || result.Task.Version != "config-task" {
		t.Errorf("expected Task from config (empty status_flow = not defined), got %v", result.Task)
	}
}

// --- AC Group 4: Configurable Workflow File Path (REQ-F04-004) ---

// TC-030: Custom YAML directory via workflow_config
func TestLoadMultiLevelWorkflow_CustomPath(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	configData := map[string]interface{}{
		"workflow_config": "config/workflow",
	}
	writeJSON(t, configPath, configData)

	workflowDir := filepath.Join(dir, "config", "workflow")
	writeWorkflowYAML(t, workflowDir, "epic.yaml", epicYAML)

	ClearWorkflowCache()
	result, err := LoadMultiLevelWorkflow(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Epic == nil || result.Epic.Version != "1.0" {
		t.Errorf("expected Epic from custom path, got %v", result.Epic)
	}
}

// TC-030b: Explicit JSON workflow_config is rejected with a migration hint.
func TestLoadMultiLevelWorkflow_JSONWorkflowConfigRejected(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	configData := map[string]interface{}{
		"workflow_config": "config/workflows.json",
	}
	writeJSON(t, configPath, configData)

	workflowData := map[string]interface{}{
		"epic_workflow": buildWorkflowBlock("json-epic"),
	}
	writeJSON(t, filepath.Join(dir, "config", "workflows.json"), workflowData)

	ClearWorkflowCache()
	_, err := LoadMultiLevelWorkflow(configPath)
	if err == nil {
		t.Fatal("expected deprecated JSON workflow_config error, got nil")
	}
	if !errors.Is(err, ErrDeprecatedWorkflowConfigJSON) {
		t.Fatalf("error = %q; want ErrDeprecatedWorkflowConfigJSON", err.Error())
	}
}

func TestIsDeprecatedWorkflowConfigTarget_JSONOnly(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "json workflow", value: "config/workflows.json", want: true},
		{name: "legacy sharkworkflow json", value: ".sharkworkflow.json", want: true},
		{name: "yaml master index", value: "workflow.yaml", want: false},
		{name: "sharkworkflow yaml master index", value: ".sharkworkflow.yaml", want: false},
		{name: "directory", value: "shark-data/workflow", want: false},
		{name: "empty", value: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDeprecatedWorkflowConfigTarget(tt.value); got != tt.want {
				t.Errorf("IsDeprecatedWorkflowConfigTarget(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

// TC-031: Missing custom directory falls back to inline config
func TestLoadMultiLevelWorkflow_MissingCustomPath(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	configData := map[string]interface{}{
		"workflow_config": "nonexistent/workflow",
		"task_workflow":   buildWorkflowBlock("inline-task"),
	}
	writeJSON(t, configPath, configData)

	ClearWorkflowCache()
	result, err := LoadMultiLevelWorkflow(configPath)
	if err != nil {
		t.Fatalf("unexpected error (missing custom path should silently fall back): %v", err)
	}

	if result.Task == nil || result.Task.Version != "inline-task" {
		t.Errorf("expected Task from inline config, got %v", result.Task)
	}
}

// TC-032: Absent workflow_config key, default path used
func TestLoadMultiLevelWorkflow_AbsentWorkflowConfigKey(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	// No workflow_config key
	writeJSON(t, configPath, map[string]interface{}{})

	// Default .sharkworkflow.json in same dir
	workflowData := map[string]interface{}{
		"epic_workflow": buildWorkflowBlock("default-path-epic"),
	}
	writeJSON(t, filepath.Join(dir, ".sharkworkflow.json"), workflowData)

	ClearWorkflowCache()
	result, err := LoadMultiLevelWorkflow(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Epic == nil || result.Epic.Version != "default-path-epic" {
		t.Errorf("expected Epic from default path, got %v", result.Epic)
	}
}

// TC-033: Empty string workflow_config treated as absent
func TestLoadMultiLevelWorkflow_EmptyWorkflowConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	configData := map[string]interface{}{
		"workflow_config": "",
	}
	writeJSON(t, configPath, configData)

	workflowData := map[string]interface{}{
		"epic_workflow": buildWorkflowBlock("default-path-epic"),
	}
	writeJSON(t, filepath.Join(dir, ".sharkworkflow.json"), workflowData)

	ClearWorkflowCache()
	result, err := LoadMultiLevelWorkflow(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Epic == nil || result.Epic.Version != "default-path-epic" {
		t.Errorf("expected Epic from default path (empty config = absent), got %v", result.Epic)
	}
}

// TC-034: Absolute YAML directory via workflow_config (within project root)
func TestLoadMultiLevelWorkflow_AbsolutePath(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	// Use an absolute directory within the same project root directory
	absPath := filepath.Join(dir, "workflows", "custom")

	configData := map[string]interface{}{
		"workflow_config": absPath,
	}
	writeJSON(t, configPath, configData)

	writeWorkflowYAML(t, absPath, "epic.yaml", epicYAML)

	ClearWorkflowCache()
	result, err := LoadMultiLevelWorkflow(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Epic == nil || result.Epic.Version != "1.0" {
		t.Errorf("expected Epic from absolute path, got %v", result.Epic)
	}
}

// TC-034b: Absolute path outside project root is rejected (path traversal protection)
// TC-034b: Absolute path outside project root is trusted (user explicitly configured it).
// This supports shared workflow directories outside the project root.
func TestLoadMultiLevelWorkflow_AbsolutePathOutsideRoot(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	absDir := t.TempDir()
	absPath := filepath.Join(absDir, "workflows")

	configData := map[string]interface{}{
		"workflow_config": absPath,
	}
	writeJSON(t, configPath, configData)

	writeWorkflowYAML(t, absPath, "epic.yaml", epicYAML)

	ClearWorkflowCache()
	result, err := LoadMultiLevelWorkflow(configPath)
	if err != nil {
		t.Fatalf("unexpected error for absolute path: %v", err)
	}
	if result == nil || result.Epic == nil {
		t.Fatal("expected epic workflow to be loaded from absolute path")
	}
}

// TC-035: Relative YAML directory resolved relative to config directory
func TestLoadMultiLevelWorkflow_RelativePathResolution(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	configData := map[string]interface{}{
		"workflow_config": "subdir/workflow",
	}
	writeJSON(t, configPath, configData)

	writeWorkflowYAML(t, filepath.Join(dir, "subdir", "workflow"), "epic.yaml", epicYAML)

	ClearWorkflowCache()
	result, err := LoadMultiLevelWorkflow(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Epic == nil || result.Epic.Version != "1.0" {
		t.Errorf("expected Epic from subdir, got %v", result.Epic)
	}
}

// --- AC Group 5: Template Directory Precedence (REQ-F04-006) ---

// TC-040: template_directory from workflow file wins
func TestLoadMultiLevelWorkflow_TemplateDirFromWorkflowFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	configData := map[string]interface{}{
		"template_directory": "custom-prompts",
	}
	writeJSON(t, configPath, configData)

	workflowData := map[string]interface{}{
		"template_directory": "custom-templates",
	}
	writeJSON(t, filepath.Join(dir, ".sharkworkflow.json"), workflowData)

	ClearWorkflowCache()
	result, err := LoadMultiLevelWorkflow(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TemplateDirectory == nil || *result.TemplateDirectory != "custom-templates" {
		t.Errorf("expected TemplateDirectory == 'custom-templates' from workflow file, got %v", result.TemplateDirectory)
	}
}

// TC-041: No template_directory in workflow file, config value used
func TestLoadMultiLevelWorkflow_TemplateDirFallbackToConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	configData := map[string]interface{}{
		"template_directory": "custom-prompts",
	}
	writeJSON(t, configPath, configData)

	// Workflow file has no template_directory
	workflowData := map[string]interface{}{
		"epic_workflow": buildWorkflowBlock("file-epic"),
	}
	writeJSON(t, filepath.Join(dir, ".sharkworkflow.json"), workflowData)

	ClearWorkflowCache()
	result, err := LoadMultiLevelWorkflow(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// result.TemplateDirectory should be nil (Config layer handles fallback)
	if result.TemplateDirectory != nil {
		t.Errorf("expected nil TemplateDirectory (config handles fallback), got %v", *result.TemplateDirectory)
	}
}

// TC-042: Neither file has template_directory
func TestLoadMultiLevelWorkflow_TemplateDirNeitherFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	writeJSON(t, configPath, map[string]interface{}{})

	ClearWorkflowCache()
	result, err := LoadMultiLevelWorkflow(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TemplateDirectory != nil {
		t.Errorf("expected nil TemplateDirectory, got %v", *result.TemplateDirectory)
	}
}

// --- AC Group 6: Error Handling ---

// TC-050: Invalid JSON in workflow file returns error with file path
func TestLoadMultiLevelWorkflow_InvalidWorkflowFileJSON(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	writeJSON(t, configPath, map[string]interface{}{})

	// Write invalid JSON
	wfPath := filepath.Join(dir, ".sharkworkflow.json")
	if err := os.WriteFile(wfPath, []byte("{invalid}"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	ClearWorkflowCache()
	_, err := LoadMultiLevelWorkflow(configPath)
	if err == nil {
		t.Fatal("expected error for invalid JSON in workflow file")
	}
	if !strings.Contains(err.Error(), wfPath) {
		t.Errorf("error should contain file path %q, got: %v", wfPath, err)
	}
}

// TC-051: Invalid entity block returns error with entity name and file path
func TestLoadMultiLevelWorkflow_InvalidEntityBlock(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	writeJSON(t, configPath, map[string]interface{}{})

	// Write workflow file with invalid epic_workflow (not an object)
	wfPath := filepath.Join(dir, ".sharkworkflow.json")
	wfContent := `{"epic_workflow": "not-an-object"}`
	if err := os.WriteFile(wfPath, []byte(wfContent), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	ClearWorkflowCache()
	_, err := LoadMultiLevelWorkflow(configPath)
	if err == nil {
		t.Fatal("expected error for invalid entity block")
	}
	if !strings.Contains(err.Error(), "epic_workflow") {
		t.Errorf("error should mention 'epic_workflow', got: %v", err)
	}
	if !strings.Contains(err.Error(), wfPath) {
		t.Errorf("error should contain file path, got: %v", err)
	}
}

// TC-052: Empty workflow file (0 bytes) treated as empty JSON
func TestLoadMultiLevelWorkflow_EmptyWorkflowFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	configData := map[string]interface{}{
		"task_workflow": buildWorkflowBlock("config-task"),
	}
	writeJSON(t, configPath, configData)

	// Write empty file
	wfPath := filepath.Join(dir, ".sharkworkflow.json")
	if err := os.WriteFile(wfPath, []byte{}, 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	ClearWorkflowCache()
	result, err := LoadMultiLevelWorkflow(configPath)
	if err != nil {
		t.Fatalf("empty workflow file should not cause error: %v", err)
	}

	// Config values should be used
	if result.Task == nil || result.Task.Version != "config-task" {
		t.Errorf("expected Task from config (empty file = no overrides), got %v", result.Task)
	}
}

// TC-053: Circular JSON workflow_config is rejected as deprecated config
func TestLoadMultiLevelWorkflow_CircularWorkflowConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	configData := map[string]interface{}{
		"workflow_config": ".sharkconfig.json",
		"task_workflow":   buildWorkflowBlock("config-task"),
	}
	writeJSON(t, configPath, configData)

	ClearWorkflowCache()
	_, err := LoadMultiLevelWorkflow(configPath)
	if err == nil {
		t.Fatal("expected deprecated JSON workflow_config error, got nil")
	}
	if !errors.Is(err, ErrDeprecatedWorkflowConfigJSON) {
		t.Fatalf("error = %q; want ErrDeprecatedWorkflowConfigJSON", err.Error())
	}
}

// --- AC Group 8: Cache Behavior ---

// TC-070: Legacy cache sync
func TestLoadMultiLevelWorkflow_LegacyCacheSync(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	writeJSON(t, configPath, map[string]interface{}{})

	workflowData := map[string]interface{}{
		"task_workflow": buildWorkflowBlock("file-task"),
	}
	writeJSON(t, filepath.Join(dir, ".sharkworkflow.json"), workflowData)

	ClearWorkflowCache()
	result, err := LoadMultiLevelWorkflow(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// GetWorkflowOrDefault should return the same task workflow
	legacyWf := GetWorkflowOrDefault(configPath)
	if legacyWf.Version != result.Task.Version {
		t.Errorf("legacy cache out of sync: expected %q, got %q", result.Task.Version, legacyWf.Version)
	}
}

// TC-071: Cache cleared between loads
func TestLoadMultiLevelWorkflow_CacheClearedBetweenLoads(t *testing.T) {
	dir1 := t.TempDir()
	configPath1 := filepath.Join(dir1, ".sharkconfig.json")
	writeJSON(t, configPath1, map[string]interface{}{
		"task_workflow": buildWorkflowBlock("config-A"),
	})

	dir2 := t.TempDir()
	configPath2 := filepath.Join(dir2, ".sharkconfig.json")
	writeJSON(t, configPath2, map[string]interface{}{
		"task_workflow": buildWorkflowBlock("config-B"),
	})

	ClearWorkflowCache()
	result1, err := LoadMultiLevelWorkflow(configPath1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result1.Task.Version != "config-A" {
		t.Errorf("expected config-A, got %q", result1.Task.Version)
	}

	ClearWorkflowCache()
	result2, err := LoadMultiLevelWorkflow(configPath2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result2.Task.Version != "config-B" {
		t.Errorf("expected config-B after cache clear, got %q", result2.Task.Version)
	}
}

// --- AC Group 9: Performance ---

func BenchmarkLoadMultiLevelWorkflow_TwoFiles(b *testing.B) {
	dir := b.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	configData := buildFullConfig("config")
	jsonData, _ := json.MarshalIndent(configData, "", "  ")
	os.WriteFile(configPath, jsonData, 0644)

	workflowData := map[string]interface{}{
		"epic_workflow":    buildWorkflowBlock("file-epic"),
		"feature_workflow": buildWorkflowBlock("file-feature"),
		"task_workflow":    buildWorkflowBlock("file-task"),
		"bug_workflow":     buildWorkflowBlock("file-bug"),
		"change_workflow":  buildWorkflowBlock("file-change"),
	}
	wfData, _ := json.MarshalIndent(workflowData, "", "  ")
	os.WriteFile(filepath.Join(dir, ".sharkworkflow.json"), wfData, 0644)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ClearWorkflowCache()
		_, err := LoadMultiLevelWorkflow(configPath)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// --- Edge Case Tests for loadWorkflowFile ---

// TestLoadWorkflowFile_InvalidJSON verifies that invalid JSON content returns an error
// with file path and byte offset information.
func TestLoadWorkflowFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	wfPath := filepath.Join(dir, ".sharkworkflow.json")

	// Write syntactically invalid JSON
	if err := os.WriteFile(wfPath, []byte(`{"task_workflow": {invalid}`), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	result, err := loadWorkflowFile(wfPath)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result for invalid JSON, got %v", result)
	}
	// Error should contain the file path
	if !strings.Contains(err.Error(), wfPath) {
		t.Errorf("expected error to contain file path %q, got: %v", wfPath, err)
	}
	// Error should mention byte offset (syntax error)
	if !strings.Contains(err.Error(), "byte offset") {
		t.Errorf("expected error to mention 'byte offset', got: %v", err)
	}
}

// TestLoadWorkflowFile_ArrayInsteadOfObject verifies that a JSON array at the top level
// returns an error (expected object, got array).
func TestLoadWorkflowFile_ArrayInsteadOfObject(t *testing.T) {
	dir := t.TempDir()
	wfPath := filepath.Join(dir, ".sharkworkflow.json")

	// Write a valid JSON array instead of an object
	if err := os.WriteFile(wfPath, []byte(`[{"task_workflow": {}}]`), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	result, err := loadWorkflowFile(wfPath)
	if err == nil {
		t.Fatal("expected error for JSON array (not object), got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
	// Error should contain the file path
	if !strings.Contains(err.Error(), wfPath) {
		t.Errorf("expected error to contain file path %q, got: %v", wfPath, err)
	}
}

// TestLoadWorkflowFile_PermissionDenied verifies that a permission-denied error
// is returned (not silently ignored like file-not-found).
func TestLoadWorkflowFile_PermissionDenied(t *testing.T) {
	// Skip on environments where we can't test permissions (e.g., running as root)
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	wfPath := filepath.Join(dir, ".sharkworkflow.json")

	// Write a valid file, then remove read permissions
	if err := os.WriteFile(wfPath, []byte(`{}`), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.Chmod(wfPath, 0000); err != nil {
		t.Fatalf("failed to chmod file: %v", err)
	}
	// Restore permissions on cleanup so t.TempDir() can clean up
	t.Cleanup(func() {
		os.Chmod(wfPath, 0644)
	})

	result, err := loadWorkflowFile(wfPath)
	if err == nil {
		t.Fatal("expected error for permission-denied file, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result for permission-denied, got %v", result)
	}
	// Should NOT be silently ignored (that is only for IsNotExist)
	if !strings.Contains(err.Error(), wfPath) {
		t.Errorf("expected error to contain file path %q, got: %v", wfPath, err)
	}
}

// TestLoadWorkflowFile_NonExistentFile verifies that a missing file returns (nil, nil).
func TestLoadWorkflowFile_NonExistentFile(t *testing.T) {
	dir := t.TempDir()
	wfPath := filepath.Join(dir, "does-not-exist.json")

	result, err := loadWorkflowFile(wfPath)
	if err != nil {
		t.Fatalf("expected nil error for non-existent file, got: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for non-existent file, got %v", result)
	}
}

// TestLoadWorkflowFile_EmptyFile verifies that a 0-byte file returns an empty map (no error).
func TestLoadWorkflowFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	wfPath := filepath.Join(dir, ".sharkworkflow.json")

	if err := os.WriteFile(wfPath, []byte{}, 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	result, err := loadWorkflowFile(wfPath)
	if err != nil {
		t.Fatalf("expected nil error for empty file, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil (empty) map for empty file, got nil")
	}
	if len(result) != 0 {
		t.Errorf("expected empty map for empty file, got %d entries", len(result))
	}
}

// TestLoadWorkflowFile_WhitespaceOnlyContent verifies that a file with only whitespace
// is treated as invalid JSON (not as empty).
func TestLoadWorkflowFile_WhitespaceOnlyContent(t *testing.T) {
	dir := t.TempDir()
	wfPath := filepath.Join(dir, ".sharkworkflow.json")

	if err := os.WriteFile(wfPath, []byte("   \n\t  "), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	result, err := loadWorkflowFile(wfPath)
	if err == nil {
		t.Fatal("expected error for whitespace-only content, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}

func BenchmarkLoadMultiLevelWorkflow_ConfigOnly(b *testing.B) {
	dir := b.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	configData := buildFullConfig("config")
	jsonData, _ := json.MarshalIndent(configData, "", "  ")
	os.WriteFile(configPath, jsonData, 0644)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ClearWorkflowCache()
		_, err := LoadMultiLevelWorkflow(configPath)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// --- Path Traversal Validation Tests (E20-F08-003) ---

func TestValidateWorkflowFilePath(t *testing.T) {
	// Use a real temp directory as the project root so filepath.Abs works predictably
	projectRoot := t.TempDir()

	tests := []struct {
		name        string
		filePath    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid relative path in project root",
			filePath: filepath.Join(projectRoot, "config", "workflow.json"),
			wantErr:  false,
		},
		{
			name:     "valid default sharkworkflow.json",
			filePath: filepath.Join(projectRoot, ".sharkworkflow.json"),
			wantErr:  false,
		},
		{
			name:        "path traversal escaping project root",
			filePath:    filepath.Join(projectRoot, "..", "..", "..", "etc", "passwd"),
			wantErr:     true,
			errContains: "escapes project root",
		},
		{
			name:        "absolute path outside project root",
			filePath:    "/tmp/malicious.json",
			wantErr:     true,
			errContains: "escapes project root",
		},
		{
			name:     "relative path with dotdot that resolves within project",
			filePath: filepath.Join(projectRoot, "subdir", "..", "workflow.json"),
			wantErr:  false,
		},
		{
			name:     "file at project root itself",
			filePath: projectRoot,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWorkflowFilePath(projectRoot, tt.filePath)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestLoadMultiLevelWorkflow_PathTraversal verifies that LoadMultiLevelWorkflow
// rejects a workflow_config value that escapes the project root.
func TestLoadMultiLevelWorkflow_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")

	tests := []struct {
		name           string
		workflowConfig string
		wantErr        bool
		errContains    string
	}{
		{
			name:           "path traversal attack",
			workflowConfig: "../../../etc/passwd",
			wantErr:        true,
			errContains:    "escapes project root",
		},
		{
			name:           "absolute path outside project is trusted",
			workflowConfig: "/tmp/malicious-workflow",
			wantErr:        false, // absolute paths are trusted (user explicitly configured them)
		},
		{
			name:           "valid relative path",
			workflowConfig: "config/workflow",
			wantErr:        false,
		},
		{
			name:           "deprecated JSON workflow target",
			workflowConfig: ".sharkworkflow.json",
			wantErr:        true,
			errContains:    "deprecated workflow_config JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ClearWorkflowCache()

			configData := map[string]interface{}{
				"workflow_config": tt.workflowConfig,
			}
			writeJSON(t, configPath, configData)

			_, err := LoadMultiLevelWorkflow(configPath)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
