package research

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

func TestArtifactPaths_AllWorkflowEntityTypes(t *testing.T) {
	root := t.TempDir()
	for _, entity := range researchEntities() {
		t.Run(string(entity.GetEntityType()), func(t *testing.T) {
			paths, err := ArtifactPaths(root, entity)
			if err != nil {
				t.Fatalf("ArtifactPaths() error = %v", err)
			}
			if entity.GetEntityType() == models.EntityTypeEpic || entity.GetEntityType() == models.EntityTypeFeature {
				if filepath.Base(paths.Plan) != "research-plan.md" || filepath.Base(paths.Report) != "research-report.md" {
					t.Fatalf("expected directory-local artifacts, got %+v", paths)
				}
				return
			}
			if !strings.Contains(filepath.Base(paths.Plan), entity.GetKey()+".research-plan.md") {
				t.Fatalf("expected sidecar plan for %s, got %s", entity.GetKey(), paths.Plan)
			}
		})
	}
}

func TestValidateEntity(t *testing.T) {
	root := t.TempDir()
	entity := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E01-F01-001", FilePath: stringPtr("docs/plan/E01/F01/tasks/T-E01-F01-001.md")}, Status: "research"}
	paths, err := ArtifactPaths(root, entity)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(paths.Plan), 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		plan    string
		report  string
		wantErr string
	}{
		{"valid", validPlan(entity), validReport(entity), ""},
		{"plan identity", strings.Replace(validPlan(entity), "entity_key: T-E01-F01-001", "entity_key: T-E99-F99-999", 1), validReport(entity), "must identify"},
		{"report entity type", validPlan(entity), strings.Replace(validReport(entity), "entity_type: task", "entity_type: feature", 1), "must identify"},
		{"unknown category", strings.Replace(validPlan(entity), "backend", "unknown", 1), strings.Replace(validReport(entity), "backend", "unknown", 1), "does not define category"},
		{"mismatched report metadata", validPlan(entity), strings.Replace(validReport(entity), "rigor: simple", "rigor: complex", 1), "different recipe metadata"},
		{"missing related work", strings.Replace(validPlan(entity), "related_work: true\n", "", 1), validReport(entity), "must include related_work"},
		{"missing capability map for related work", validPlan(entity), strings.Replace(validReport(entity), "| Existing capability | source | REUSE |", "", 1), "capability map is required"},
		{"missing source references", strings.Replace(validPlan(entity), "  - docs/plan/E01/F01/tasks/T-E01-F01-001.md\n", "", 1), validReport(entity), "must include source references"},
		{"missing report source references", validPlan(entity), strings.Replace(validReport(entity), "  - internal/services/task_service.go\n", "", 1), "must include source references"},
		{"missing plan section", strings.Replace(validPlan(entity), "## Steps\nInspect the service.", "## Steps\n", 1), validReport(entity), "missing required section \"Steps\""},
		{"missing report section", validPlan(entity), strings.Replace(validReport(entity), "## Decisions\nExtend the existing service.", "## Decisions\n", 1), "missing required section \"Decisions\""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.WriteFile(paths.Plan, []byte(tt.plan), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(paths.Report, []byte(tt.report), 0o644); err != nil {
				t.Fatal(err)
			}
			err := ValidateEntity(root, entity)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("ValidateEntity() error = %v", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("ValidateEntity() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoadCatalog_UsesConfiguredSharkDataPath(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".sharkconfig.json"), []byte(`{"shark_data_path":"custom-data"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	catalogPath := filepath.Join(root, "custom-data", "research", "recipes.yaml")
	if err := os.MkdirAll(filepath.Dir(catalogPath), 0o755); err != nil {
		t.Fatal(err)
	}
	data := "version: test\nrecipes:\n  custom:\n    entity_types: [task]\n    required_plan_sections: [Scope]\n    required_report_sections: [Scope]\n    categories: [backend]\n    rigor:\n      simple:\n        description: test\n"
	if err := os.WriteFile(catalogPath, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	catalog, err := LoadCatalog(root)
	if err != nil {
		t.Fatalf("LoadCatalog() error = %v", err)
	}
	if _, ok := catalog.Recipes["custom"]; !ok {
		t.Fatal("LoadCatalog() did not use the configured shark_data_path")
	}
}

func validPlan(entity models.Entity) string {
	return "---\nentity_key: " + entity.GetKey() + "\nentity_type: task\nrecipe: universal\nrigor: simple\ncategories:\n  - backend\nsource_set:\n  - docs/plan/E01/F01/tasks/T-E01-F01-001.md\nrelated_work: true\n---\n# Research plan\n\n## Scope\nTask scope.\n\n## Recipe\nUniversal.\n\n## Source set\nTask file.\n\n## Steps\nInspect the service.\n"
}

func validReport(entity models.Entity) string {
	return "---\nentity_key: " + entity.GetKey() + "\nentity_type: task\nrecipe: universal\nrigor: simple\ncategories:\n  - backend\nsource_set:\n  - internal/services/task_service.go\nrelated_work: true\n---\n# Research report\n\n## Scope\nTask scope.\n\n## Capability map\n| Capability | Source | Decision |\n| --- | --- | --- |\n| Existing capability | source | REUSE |\n\n## Ubiquitous vocabulary\nTask: atomic work.\n\n## Findings\nUse the existing service.\n\n## Decisions\nExtend the existing service.\n\n## Sources\ninternal/services/task_service.go\n"
}

func researchEntities() []models.Entity {
	file := stringPtr("docs/plan/entity.md")
	base := func(key string) models.BaseEntity { return models.BaseEntity{Key: key, FilePath: file} }
	return []models.Entity{
		&models.Epic{BaseEntity: base("E01")},
		&models.Feature{BaseEntity: base("E01-F01")},
		&models.Task{BaseEntity: base("T-E01-F01-001")},
		&models.Bug{BaseEntity: base("B001")},
		&models.ChangeCard{BaseEntity: base("CC-001")},
		&models.TechDebt{BaseEntity: base("TD-001")},
		&models.Sprint{Key: "S001", FilePath: "docs/plan/sprints/S001.md"},
	}
}

func stringPtr(value string) *string { return &value }
