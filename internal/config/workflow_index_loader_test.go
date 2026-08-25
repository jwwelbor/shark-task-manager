package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
	workflowpkg "github.com/jwwelbor/shark-task-manager/internal/config/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests cover the ActionService workflow loader (defaultWorkflowDataLoader)
// when workflow_config points at a master index file (entities: map), per
// docs/guides/route-based-workflow.md §3.
//
// Regression context: the index-file form was implemented and tested in the
// `workflow` package (LoadWorkflowIndexFile) and wired into the parser path,
// but the *second* loader used by ActionService rejected every regular file as
// a legacy Shark 1.x JSON workflow before the index could be detected. No test
// exercised defaultWorkflowDataLoader with an index, so the gap went unnoticed.

// minimalEpicWorkflow is a route-based (steps:) epic workflow whose sole status
// carries a distinctive orchestrator action, so a passing assertion proves the
// data came from the index-referenced file and not a hardcoded default.
const minimalEpicWorkflow = `version: "1.0"
start: index_marker
steps:
  index_marker:
    phase: planning
    action: spawn_agent
    agent: business-analyst
    outcomes:
      pass: done
  done:
    phase: done
    action: archive
    terminal: true
`

func workflowWithAction(status, agent string) string {
	return `version: "1.0"
start: ` + status + `
steps:
  ` + status + `:
    phase: planning
    action: spawn_agent
    agent: ` + agent + `
    outcomes:
      pass: done
  done:
    phase: done
    action: archive
    terminal: true
`
}

// writeIndexProject scaffolds a temp project whose workflow_config points at a
// master index file. entityRel is the path written into .sharkconfig.json for
// the index; epicPath is where the epic workflow YAML is written and what the
// index maps `epic:` to (caller controls relative vs absolute).
func writeIndexProject(t *testing.T, entityRel, indexEpicRef, epicAbsPath string) string {
	t.Helper()
	tmp := t.TempDir()

	indexPath := filepath.Join(tmp, entityRel)
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		t.Fatal(err)
	}
	index := "entities:\n  epic: " + indexEpicRef + "\n"
	if err := os.WriteFile(indexPath, []byte(index), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Dir(epicAbsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(epicAbsPath, []byte(minimalEpicWorkflow), 0o644); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(tmp, ".sharkconfig.json")
	cfg := `{"workflow_config": "` + entityRel + `"}`
	if err := os.WriteFile(configPath, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	return configPath
}

func TestDefaultWorkflowDataLoader_MasterIndexRelativePaths(t *testing.T) {
	// Index at route/workflow.yaml mapping epic -> workflow/epic.yaml (relative
	// to the index's bundle directory, i.e. route/).
	tmp := t.TempDir()
	bundle := filepath.Join(tmp, "route")
	epicAbs := filepath.Join(bundle, "workflow", "epic.yaml")
	if err := os.MkdirAll(filepath.Dir(epicAbs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(epicAbs, []byte(minimalEpicWorkflow), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, "workflow.yaml"),
		[]byte("entities:\n  epic: workflow/epic.yaml\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(tmp, ".sharkconfig.json")
	if err := os.WriteFile(configPath,
		[]byte(`{"workflow_config": "route/workflow.yaml"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := defaultWorkflowDataLoader(configPath)
	if err != nil {
		t.Fatalf("defaultWorkflowDataLoader returned error for master index: %v", err)
	}

	epic, ok := out["epic"]
	if !ok {
		t.Fatalf("epic workflow not loaded from index; out keys=%v", keysOf(out))
	}
	marker, ok := epic["index_marker"]
	if !ok {
		t.Fatalf("index_marker status not present; epic statuses=%v", statusKeys(epic))
	}
	if marker.OrchestratorAction == nil || marker.OrchestratorAction.Action != "spawn_agent" {
		t.Errorf("index_marker action=%+v; want Action=spawn_agent (proves index drove the load)",
			marker.OrchestratorAction)
	}
}

func TestDefaultWorkflowDataLoader_MasterIndexAbsolutePath(t *testing.T) {
	// epic mapped to an absolute path outside the bundle (shared-bundle case).
	shared := t.TempDir()
	epicAbs := filepath.Join(shared, "epic.yaml")
	configPath := writeIndexProject(t, "route/workflow.yaml", epicAbs, epicAbs)

	out, err := defaultWorkflowDataLoader(configPath)
	if err != nil {
		t.Fatalf("defaultWorkflowDataLoader error for absolute-path index entry: %v", err)
	}
	if _, ok := out["epic"]["index_marker"]; !ok {
		t.Fatalf("epic workflow not loaded from absolute index entry; statuses=%v",
			statusKeys(out["epic"]))
	}
}

func TestDefaultWorkflowDataLoader_GenuineLegacyJSONStillRejected(t *testing.T) {
	// A regular file that is NOT a master index (no entities: map) must still be
	// rejected with the migration hint — the fix must not loosen that guard.
	tmp := t.TempDir()
	configuredPath := "legacy/workflow.json"
	workflowPath := filepath.Join(tmp, configuredPath)
	if err := os.MkdirAll(filepath.Dir(workflowPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(workflowPath, []byte(`{"status_flow": {}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(tmp, ".sharkconfig.json")
	if err := os.WriteFile(configPath,
		[]byte(`{"workflow_config": "`+configuredPath+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := defaultWorkflowDataLoader(configPath)
	if err == nil {
		t.Fatal("expected rejection for genuine legacy JSON workflow file, got nil")
	}
	if !errors.Is(err, workflowpkg.ErrDeprecatedWorkflowConfigJSON) {
		t.Errorf("error = %q; want ErrDeprecatedWorkflowConfigJSON", err.Error())
	}
}

func TestDefaultWorkflowDataLoader_MissingLegacyJSONStillRejected(t *testing.T) {
	tmp := t.TempDir()
	configuredPath := "legacy/missing-workflow.json"
	configPath := filepath.Join(tmp, ".sharkconfig.json")
	if err := os.WriteFile(configPath,
		[]byte(`{"workflow_config": "`+configuredPath+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := defaultWorkflowDataLoader(configPath)
	if err == nil {
		t.Fatal("expected rejection for missing legacy JSON workflow file, got nil")
	}
	if !errors.Is(err, workflowpkg.ErrDeprecatedWorkflowConfigJSON) {
		t.Errorf("error = %q; want ErrDeprecatedWorkflowConfigJSON", err.Error())
	}
}

func TestDefaultWorkflowDataLoader_UsesStrictSharedParser(t *testing.T) {
	workflowpkg.ClearWorkflowCache()
	t.Cleanup(workflowpkg.ClearWorkflowCache)

	configPath := filepath.Join(t.TempDir(), ".sharkconfig.json")
	if err := os.WriteFile(configPath, []byte(`{"task_workflow":`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := defaultWorkflowDataLoader(configPath)
	if err == nil {
		t.Fatal("expected invalid configuration to be returned by the strict shared parser")
	}
}

func TestDefaultWorkflowDataLoader_UsesSharkDataPathAndQuestionWorkflow(t *testing.T) {
	workflowpkg.ClearWorkflowCache()
	t.Cleanup(workflowpkg.ClearWorkflowCache)

	root := t.TempDir()
	workflowDir := filepath.Join(root, "custom-bundle", "workflow")
	require.NoError(t, os.MkdirAll(workflowDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(workflowDir, "question.yaml"), []byte(workflowWithAction("awaiting_input", "question-specialist")), 0o644))
	configPath := filepath.Join(root, ".sharkconfig.json")
	require.NoError(t, os.WriteFile(configPath, []byte(`{"shark_data_path":"custom-bundle"}`), 0o644))

	out, err := defaultWorkflowDataLoader(configPath)
	require.NoError(t, err)
	questionAction := out["question"]["awaiting_input"].OrchestratorAction
	require.NotNil(t, questionAction)
	assert.Equal(t, "question-specialist", questionAction.AgentType)
}

func TestDefaultWorkflowDataLoader_LoadsDefaultDiskWorkflowWithoutConfigFile(t *testing.T) {
	workflowpkg.ClearWorkflowCache()
	t.Cleanup(workflowpkg.ClearWorkflowCache)

	root := t.TempDir()
	workflowDir := filepath.Join(root, "shark-data", "workflow")
	require.NoError(t, os.MkdirAll(workflowDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(workflowDir, "task.yaml"), []byte(workflowWithAction("disk_default", "disk-agent")), 0o644))

	out, err := defaultWorkflowDataLoader(filepath.Join(root, ".sharkconfig.json"))
	require.NoError(t, err)
	action := out["task"]["disk_default"].OrchestratorAction
	require.NotNil(t, action)
	assert.Equal(t, "disk-agent", action.AgentType)
}

func TestDefaultWorkflowDataLoader_ReloadsChangedWorkflowDirectory(t *testing.T) {
	workflowpkg.ClearWorkflowCache()
	t.Cleanup(workflowpkg.ClearWorkflowCache)

	root := t.TempDir()
	workflowDir := filepath.Join(root, "bundle", "workflow")
	require.NoError(t, os.MkdirAll(workflowDir, 0o755))
	taskPath := filepath.Join(workflowDir, "task.yaml")
	require.NoError(t, os.WriteFile(taskPath, []byte(workflowWithAction("queued", "first-agent")), 0o644))
	configPath := filepath.Join(root, ".sharkconfig.json")
	require.NoError(t, os.WriteFile(configPath, []byte(`{"shark_data_path":"bundle"}`), 0o644))

	svc, err := action.NewActionService(configPath, defaultWorkflowDataLoader)
	require.NoError(t, err)
	initial, err := svc.GetStatusAction(context.Background(), "queued")
	require.NoError(t, err)
	require.NotNil(t, initial)
	assert.Equal(t, "first-agent", initial.AgentType)

	require.NoError(t, os.WriteFile(taskPath, []byte(workflowWithAction("queued", "second-agent")), 0o644))
	require.NoError(t, svc.Reload(context.Background()))
	reloaded, err := svc.GetStatusAction(context.Background(), "queued")
	require.NoError(t, err)
	require.NotNil(t, reloaded)
	assert.Equal(t, "second-agent", reloaded.AgentType)
}

func TestWorkflowParserDefaultDirectoryModeDoesNotContaminateDefaultCache(t *testing.T) {
	t.Skip("fixture relies on retired root JSON fallback; E32-F06 replaces it with refusal coverage")
	workflowpkg.ClearWorkflowCache()
	t.Cleanup(workflowpkg.ClearWorkflowCache)

	root := t.TempDir()
	configPath := filepath.Join(root, ".sharkconfig.json")
	configBytes := []byte(`{"shark_data_path":"bundle"}`)
	require.NoError(t, os.WriteFile(configPath, configBytes, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".sharkworkflow.json"), []byte(`{"task_workflow":{"status_flow":{"root":["done"],"done":[]},"status_metadata":{"root":{"orchestrator_action":{"action":"spawn_agent","agent_type":"root-agent"}}}}}`), 0o644))
	customDir := filepath.Join(root, "bundle", "workflow")
	require.NoError(t, os.MkdirAll(customDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(customDir, "task.yaml"), []byte(workflowWithAction("custom", "custom-agent")), 0o644))

	assertParserModesIndependent := func() {
		t.Helper()
		standard, err := workflowpkg.LoadMultiLevelWorkflowFromBytes(configPath, configBytes)
		require.NoError(t, err)
		assert.NotNil(t, standard.Task.StatusMetadata["root"].OrchestratorAction)
		assert.Nil(t, standard.Task.StatusMetadata["custom"].OrchestratorAction)

		custom, err := workflowpkg.LoadMultiLevelWorkflowFromBytesWithDefaultWorkflowDir(configPath, configBytes, customDir)
		require.NoError(t, err)
		assert.NotNil(t, custom.Task.StatusMetadata["custom"].OrchestratorAction)
		assert.Nil(t, custom.Task.StatusMetadata["root"].OrchestratorAction)
	}

	// Run both orders: either parser mode may initialize first in production.
	assertParserModesIndependent()
	workflowpkg.ClearWorkflowCache()
	custom, err := workflowpkg.LoadMultiLevelWorkflowFromBytesWithDefaultWorkflowDir(configPath, configBytes, customDir)
	require.NoError(t, err)
	assert.NotNil(t, custom.Task.StatusMetadata["custom"].OrchestratorAction)
	legacy, err := workflowpkg.LoadWorkflowConfig(configPath)
	require.NoError(t, err)
	assert.Nil(t, legacy, "the custom default-directory parse must not populate the legacy path-only cache")
	standard, err := workflowpkg.LoadMultiLevelWorkflowFromBytes(configPath, configBytes)
	require.NoError(t, err)
	assert.NotNil(t, standard.Task.StatusMetadata["root"].OrchestratorAction)
}

func keysOf(m map[string]map[string]action.StatusActionData) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func statusKeys(m map[string]action.StatusActionData) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
