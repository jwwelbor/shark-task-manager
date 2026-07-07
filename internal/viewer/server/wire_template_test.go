package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	testutil "github.com/jwwelbor/shark-task-manager/internal/test"
)

func TestWireServices_AppliesWorkflowIndexTemplateDirectory(t *testing.T) {
	projectRoot := t.TempDir()
	fixture := testutil.WriteWorkflowIndexFixture(t)

	configBody := `{
  "color_enabled": false,
  "interactive_mode": false,
  "workflow_config": "` + fixture.WorkflowIndexPath + `"
}
`
	if err := os.WriteFile(filepath.Join(projectRoot, ".sharkconfig.json"), []byte(configBody), 0o644); err != nil {
		t.Fatalf("write .sharkconfig.json: %v", err)
	}

	sqlDB, err := db.InitDB(filepath.Join(projectRoot, "shark-tasks.db"))
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	repoDB := repository.NewDB(sqlDB)
	t.Cleanup(func() {
		_ = repoDB.Close()
	})

	config.ClearWorkflowCache()
	templates.SetConfiguredTemplateDir("")
	templates.SetConfiguredSharkDataPath(config.DefaultSharkDataPath)
	templates.ResetOrchestratorEngine()
	t.Cleanup(func() {
		config.ClearWorkflowCache()
		templates.SetConfiguredTemplateDir("")
		templates.SetConfiguredSharkDataPath(config.DefaultSharkDataPath)
		templates.ResetOrchestratorEngine()
	})

	_ = WireServices(repoDB, projectRoot)

	wantTemplateDir := fixture.ExpectedPromptsDir
	if got := templates.GetTemplateDirName(); got != wantTemplateDir {
		t.Fatalf("template dir = %q, want %q", got, wantTemplateDir)
	}

	rendered, err := templates.GetOrchestratorEngine().Render("epic/assessment.tmpl", map[string]string{
		"id":        "E01",
		"title":     "Epic prompt coverage",
		"file_path": "docs/plan/epic.md",
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(rendered, "ROUTE FIXTURE EPIC ASSESSMENT") {
		t.Fatalf("rendered prompt missing assessment instructions:\n%s", rendered)
	}
}
