package config

import (
	"os"
	"path/filepath"
	"testing"
)

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

	dir, overrides, isFile := resolveWorkflowDir(tmp, configPath)
	want := filepath.Join(tmp, "shark-data", "workflow")
	if dir != want {
		t.Errorf("workflowDir=%q want %q", dir, want)
	}
	if isFile {
		t.Errorf("isLegacyFile=true; want false for directory")
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

	dir, _, isFile := resolveWorkflowDir(tmp, configPath)
	want := filepath.Join(tmp, "custom", "workflow")
	if dir != want {
		t.Errorf("workflowDir=%q want %q", dir, want)
	}
	if isFile {
		t.Errorf("isLegacyFile=true; want false")
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

	dir, _, isFile := resolveWorkflowDir(tmp, configPath)
	if dir != abs {
		t.Errorf("workflowDir=%q want %q", dir, abs)
	}
	if isFile {
		t.Errorf("isLegacyFile=true; want false")
	}
}

// TestResolveWorkflowDir_LegacyFileFlag verifies that pointing
// workflow_config at a file (rather than a directory) flags the legacy
// fallback so the caller falls back to JSON loading. This is the back-compat
// shim for projects still on `.sharkworkflow.json`.
func TestResolveWorkflowDir_LegacyFileFlag(t *testing.T) {
	tmp := t.TempDir()
	legacy := filepath.Join(tmp, "shark-templates", ".sharkworkflow.json")
	if err := os.MkdirAll(filepath.Dir(legacy), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(tmp, ".sharkconfig.json")
	cfg := `{"workflow_config": "shark-templates/.sharkworkflow.json"}`
	if err := os.WriteFile(configPath, []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}

	dir, _, isFile := resolveWorkflowDir(tmp, configPath)
	if !isFile {
		t.Errorf("isLegacyFile=false; want true when workflow_config points at a file")
	}
	if dir != "" {
		t.Errorf("workflowDir=%q; want empty when signaling legacy fallback", dir)
	}
}

// TestResolveWorkflowDir_DefaultMissingFallsBackToLegacy verifies that
// when the default shark-data/workflow/ doesn't exist AND the user didn't
// explicitly configure workflow_config, the resolver flags legacy fallback
// so projects without `shark init` keep using their inline-JSON workflow.
func TestResolveWorkflowDir_DefaultMissingFallsBackToLegacy(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".sharkconfig.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}
	// shark-data/workflow/ does NOT exist.

	_, _, isFile := resolveWorkflowDir(tmp, configPath)
	if !isFile {
		t.Errorf("expected legacy fallback when default dir is missing and no workflow_config is set")
	}
}
