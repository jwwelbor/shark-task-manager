package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
)

func TestApplyTemplateConfig_PrefersWorkflowIndexTemplateDirectory(t *testing.T) {
	project := t.TempDir()
	bundle := filepath.Join(project, "bundle")

	writeFile := func(path, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeFile(filepath.Join(bundle, "workflow", "task.yaml"), `version: "1.0"
start: draft
steps:
  draft:
    phase: planning
    action: advance_status
    outcomes: { pass: completed }
  completed:
    phase: done
    terminal: true
`)
	writeFile(filepath.Join(bundle, "workflow.yaml"), "template_directory: custom-prompts\nentities:\n  task: workflow/task.yaml\n")
	writeFile(filepath.Join(project, ".sharkconfig.json"), `{
  "shark_data_path": "shark-data",
  "workflow_config": "bundle/workflow.yaml"
}`)

	cfgPath := filepath.Join(project, ".sharkconfig.json")
	mgr := config.NewManager(cfgPath)
	loadedCfg, err := mgr.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	config.ClearWorkflowCache()
	t.Cleanup(config.ClearWorkflowCache)
	t.Cleanup(func() {
		templates.SetConfiguredTemplateDir("")
		templates.SetConfiguredSharkDataPath(config.DefaultSharkDataPath)
		templates.ResetOrchestratorEngine()
	})

	applyTemplateConfig(cfgPath, loadedCfg)

	want := filepath.Join(bundle, "custom-prompts")
	if got := templates.GetTemplateDirName(); got != want {
		t.Fatalf("template dir = %q, want %q", got, want)
	}
}
