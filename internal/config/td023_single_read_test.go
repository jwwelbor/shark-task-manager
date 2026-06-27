package config

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config/workflow"
)

// TestTD023_DefaultWorkflowDataLoader_ReadsConfigFileOnce verifies that a
// single invocation of defaultWorkflowDataLoader hits .sharkconfig.json with
// exactly one os.ReadFile call.
//
// Before TD-023 the loader read the file 2-3× per call:
//   - once in resolveWorkflowDir
//   - once in readLegacyWorkflowConfigValue (for legacy-file errors only)
//   - once in workflow.LoadMultiLevelWorkflowOrDefault
//
// The fix reads bytes once at the top and threads them through every helper.
// This test pins that contract via the readConfigFile package-level seam so a
// future regression that reintroduces a redundant read fails CI.
func TestTD023_DefaultWorkflowDataLoader_ReadsConfigFileOnce(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".sharkconfig.json")

	// A minimal inline workflow so the loader has something real to parse
	// — exercises Pass 2 (LoadMultiLevelWorkflowOrDefaultFromBytes), which
	// previously triggered the third read.
	configJSON := `{
		"status_flow": {
			"todo": ["in_progress"],
			"in_progress": ["completed"],
			"completed": []
		},
		"status_metadata": {
			"todo": {"color": "gray"},
			"in_progress": {"color": "yellow"},
			"completed": {"color": "green"}
		}
	}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// Clear the workflow package cache so the loader can't short-circuit on
	// a hit from a previous test.
	workflow.ClearWorkflowCache()

	// Wrap readConfigFile with a counter. Only count reads that target the
	// .sharkconfig.json under test so unrelated reads (none expected, but
	// belt-and-braces) don't pollute the count.
	var reads int32
	orig := readConfigFile
	readConfigFile = func(name string) ([]byte, error) {
		if name == configPath {
			atomic.AddInt32(&reads, 1)
		}
		return orig(name)
	}
	t.Cleanup(func() { readConfigFile = orig })

	if _, err := defaultWorkflowDataLoader(configPath); err != nil {
		t.Fatalf("defaultWorkflowDataLoader: %v", err)
	}

	got := atomic.LoadInt32(&reads)
	if got != 1 {
		t.Fatalf("expected .sharkconfig.json to be read exactly once per loader call, got %d reads", got)
	}
}

// TestTD023_DefaultWorkflowDataLoader_LegacyFileErrorPathReadsOnce covers
// the second-read path that previously fired only when workflow_config
// pointed at a Shark 1.x JSON file (readLegacyWorkflowConfigValue). The fix
// reuses the already-loaded bytes for the error message, so even this branch
// must stay at exactly one read.
func TestTD023_DefaultWorkflowDataLoader_LegacyFileErrorPathReadsOnce(t *testing.T) {
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

	workflow.ClearWorkflowCache()

	var reads int32
	orig := readConfigFile
	readConfigFile = func(name string) ([]byte, error) {
		if name == configPath {
			atomic.AddInt32(&reads, 1)
		}
		return orig(name)
	}
	t.Cleanup(func() { readConfigFile = orig })

	// We expect an error here (legacy JSON file is rejected) — but the read
	// count must still be exactly one.
	_, err := defaultWorkflowDataLoader(configPath)
	if err == nil {
		t.Fatalf("expected legacy-file error, got nil")
	}

	got := atomic.LoadInt32(&reads)
	if got != 1 {
		t.Fatalf("expected .sharkconfig.json to be read exactly once on legacy-file error path, got %d reads", got)
	}
}
