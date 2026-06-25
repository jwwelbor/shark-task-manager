package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

// readRawConfig reads .sharkconfig.json from dir and returns it as a map.
func readRawConfig(t *testing.T, dir string) map[string]interface{} {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, ".sharkconfig.json"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	return raw
}

// TestEnsureWorkflowConfigField_AddsSharkDataPathPreservingCustomWorkflowConfig
// covers the load-bearing migration path: an existing custom workflow_config
// with no shark_data_path must gain the default shark_data_path while leaving
// the user's workflow_config untouched, and must report updated=true.
func TestEnsureWorkflowConfigField_AddsSharkDataPathPreservingCustomWorkflowConfig(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".sharkconfig.json")
	if err := os.WriteFile(cfgPath, []byte(`{"workflow_config": "custom/wf"}`), 0644); err != nil {
		t.Fatal(err)
	}

	updated, migratedFrom, err := ensureWorkflowConfigField(tmp, "shark-data/workflow")
	if err != nil {
		t.Fatalf("ensureWorkflowConfigField: %v", err)
	}
	if !updated {
		t.Errorf("updated = false; want true (shark_data_path should be added)")
	}
	if migratedFrom != "" {
		t.Errorf("migratedFrom = %q; want empty (custom workflow_config is not a legacy value)", migratedFrom)
	}

	raw := readRawConfig(t, tmp)
	if got := raw["shark_data_path"]; got != config.DefaultSharkDataPath {
		t.Errorf("shark_data_path = %v; want %q", got, config.DefaultSharkDataPath)
	}
	if got := raw["workflow_config"]; got != "custom/wf" {
		t.Errorf("workflow_config = %v; want %q (custom value must be preserved)", got, "custom/wf")
	}
}

// TestEnsureWorkflowConfigField_PreservesExistingSharkDataPath ensures a
// user-set custom shark_data_path is never overwritten on a re-run, and that
// no spurious update is reported when both fields are already custom.
func TestEnsureWorkflowConfigField_PreservesExistingSharkDataPath(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, ".sharkconfig.json")
	if err := os.WriteFile(cfgPath, []byte(`{"workflow_config": "custom/wf", "shark_data_path": "my-bundle"}`), 0644); err != nil {
		t.Fatal(err)
	}

	updated, _, err := ensureWorkflowConfigField(tmp, "shark-data/workflow")
	if err != nil {
		t.Fatalf("ensureWorkflowConfigField: %v", err)
	}
	if updated {
		t.Errorf("updated = true; want false (nothing to change)")
	}

	raw := readRawConfig(t, tmp)
	if got := raw["shark_data_path"]; got != "my-bundle" {
		t.Errorf("shark_data_path = %v; want %q (custom value must not be overwritten)", got, "my-bundle")
	}
}
