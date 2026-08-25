package config

import (
	"os"
	"path/filepath"
	"testing"
)

// readConfig is a small helper that mirrors the once-per-call os.ReadFile
// performed by defaultWorkflowDataLoader. Tests for resolveWorkflowDir pass
// the resulting bytes in directly (TD-023).
func readConfig(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	return data
}

// TestResolveWorkflowDir_DefaultsToSharkDataWorkflow verifies the default
// path when workflow_config is not set in .sharkconfig.json.
func TestResolveWorkflowDir_DefaultsToSharkDataWorkflow(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".sharkconfig.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}
	// Create the directory so the directory branch fires.
	if err := os.MkdirAll(filepath.Join(tmp, "shark-data", "workflow"), 0755); err != nil {
		t.Fatal(err)
	}

	dir, overrides := resolveWorkflowDir(tmp, readConfig(t, configPath))
	want := filepath.Join(tmp, "shark-data", "workflow")
	if dir != want {
		t.Errorf("workflowDir=%q want %q", dir, want)
	}
	if overrides != filepath.Join(tmp, "shark-data", "overrides", "workflow") {
		t.Errorf("unexpected overrides path: %q", overrides)
	}
}

// TestResolveWorkflowDir_CustomRelativePath verifies a project-relative
// path in workflow_config is resolved against the project root.
func TestResolveWorkflowDir_CustomRelativePath(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "custom", "workflow"), 0755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(tmp, ".sharkconfig.json")
	cfg := `{"workflow_config": "custom/workflow"}`
	if err := os.WriteFile(configPath, []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}

	dir, _ := resolveWorkflowDir(tmp, readConfig(t, configPath))
	want := filepath.Join(tmp, "custom", "workflow")
	if dir != want {
		t.Errorf("workflowDir=%q want %q", dir, want)
	}
}

// TestResolveWorkflowDir_AbsolutePath verifies an absolute workflow_config
// is honored verbatim.
func TestResolveWorkflowDir_AbsolutePath(t *testing.T) {
	tmp := t.TempDir()
	abs := filepath.Join(tmp, "etc", "shark", "workflow")
	if err := os.MkdirAll(abs, 0755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(tmp, ".sharkconfig.json")
	cfg := `{"workflow_config": "` + abs + `"}`
	if err := os.WriteFile(configPath, []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}

	dir, _ := resolveWorkflowDir(tmp, readConfig(t, configPath))
	if dir != abs {
		t.Errorf("workflowDir=%q want %q", dir, abs)
	}
}

// TestResolveWorkflowDir_ExplicitFilePathPreserved verifies that file targets
// remain visible to the strict workflow parser, which owns their validation.
func TestResolveWorkflowDir_ExplicitFilePathPreserved(t *testing.T) {
	t.Skip("legacy fallback signal retired by E32-F06")
	tmp := t.TempDir()
	jsonWorkflow := filepath.Join(tmp, "legacy", "workflow.json")
	if err := os.MkdirAll(filepath.Dir(jsonWorkflow), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jsonWorkflow, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(tmp, ".sharkconfig.json")
	cfg := `{"workflow_config": "legacy/workflow.json"}`
	if err := os.WriteFile(configPath, []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}

	dir, _ := resolveWorkflowDir(tmp, readConfig(t, configPath))
	if dir != jsonWorkflow {
		t.Errorf("workflowDir=%q; want explicit file path %q", dir, jsonWorkflow)
	}
}

// TestResolveWorkflowDir_DefaultMissingReturnsCanonicalPath verifies that a
// missing disk bundle remains a canonical embedded-default case.
func TestResolveWorkflowDir_DefaultMissingReturnsCanonicalPath(t *testing.T) {
	t.Skip("legacy fallback signal retired by E32-F06")
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".sharkconfig.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}
	// shark-data/workflow/ does NOT exist.

	dir, _ := resolveWorkflowDir(tmp, readConfig(t, configPath))
	if want := filepath.Join(tmp, "shark-data", "workflow"); dir != want {
		t.Errorf("workflowDir=%q; want canonical path %q", dir, want)
	}
}
