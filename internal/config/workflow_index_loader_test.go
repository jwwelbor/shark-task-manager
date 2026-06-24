package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
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
	legacy := filepath.Join(tmp, "shark-templates", ".sharkworkflow.json")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, []byte(`{"status_flow": {}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(tmp, ".sharkconfig.json")
	if err := os.WriteFile(configPath,
		[]byte(`{"workflow_config": "shark-templates/.sharkworkflow.json"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := defaultWorkflowDataLoader(configPath)
	if err == nil {
		t.Fatal("expected rejection for genuine legacy JSON workflow file, got nil")
	}
	if !strings.Contains(err.Error(), "Shark 1.x") {
		t.Errorf("error = %q; want Shark 1.x migration hint", err.Error())
	}
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
