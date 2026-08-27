package research

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/require"
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
				if filepath.Base(paths.Report) != "research-report.md" {
					t.Fatalf("expected directory-local report, got %+v", paths)
				}
				return
			}
			if filepath.Base(paths.Report) != entity.GetKey()+".research-report.md" {
				t.Fatalf("expected sidecar report for %s, got %s", entity.GetKey(), paths.Report)
			}
		})
	}
}

func TestValidateEntity_V2(t *testing.T) {
	root := t.TempDir()
	entity := taskEntity()
	paths, err := ArtifactPaths(root, entity)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(paths.Report), 0o755))

	tests := []struct {
		name    string
		report  string
		wantErr string
	}{
		{"valid simple task references parent map", validV2TaskReport(entity), ""},
		{"missing core module", strings.Replace(validV2TaskReport(entity), "- [x] `scope_vocabulary`", "", 1), "requires module \"scope_vocabulary\""},
		{"unchecked module", strings.Replace(validV2TaskReport(entity), "- [x] `scope_vocabulary`", "- [ ] `scope_vocabulary`", 1), "is unchecked"},
		{"missing evidence", strings.Replace(validV2TaskReport(entity), "Evidence: `tasks/T-E01-F01-001.md`.", "", 1), "missing evidence"},
		{"unknown module", strings.Replace(validV2TaskReport(entity), "scope_vocabulary", "invented_module", 1), "unknown module"},
		{"inapplicable module", strings.Replace(validV2TaskReport(entity), "affected_implementation_or_contract", "related_work", 1), "not applicable"},
		{"unknown category", strings.Replace(validV2TaskReport(entity), "categories: [backend]", "categories: [unknown]", 1), "does not define category"},
		{"unsupported schema", strings.Replace(validV2TaskReport(entity), "research_schema: 2", "research_schema: 3", 1), "unsupported research_schema"},
		{"standard lacks coverage", strings.Replace(validV2TaskReport(entity), "rigor: simple", "rigor: standard", 1), "requires pattern_contract or dependency_impact"},
		{"complex lacks risks", strings.Replace(validV2TaskReport(entity), "rigor: simple", "rigor: complex", 1), "requires pattern_contract or dependency_impact"},
		{"missing rigor", strings.Replace(validV2TaskReport(entity), "rigor: simple\n", "", 1), "front matter must include rigor"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestValidateEntity_V2AcceptsSoftWrappedChecklistEvidence(t *testing.T) {
	root := t.TempDir()
	entity := taskEntity()
	paths, err := ArtifactPaths(root, entity)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(paths.Report), 0o755))
	report := strings.Replace(validV2TaskReport(entity),
		"- [x] `scope_vocabulary` — Evidence: `tasks/T-E01-F01-001.md`.",
		"- [x] `scope_vocabulary` — This checklist item uses a readable wrapped line\n  Evidence: `tasks/T-E01-F01-001.md`.", 1)
	require.NoError(t, os.WriteFile(paths.Report, []byte(report), 0o644))
	require.NoError(t, ValidateEntity(root, entity))
}

func TestValidateEntity_V2DoesNotAbsorbOtherMarkdownBullets(t *testing.T) {
	for _, marker := range []string{"*", "1."} {
		t.Run(marker, func(t *testing.T) {
			root := t.TempDir()
			entity := taskEntity()
			paths, err := ArtifactPaths(root, entity)
			require.NoError(t, err)
			require.NoError(t, os.MkdirAll(filepath.Dir(paths.Report), 0o755))
			report := strings.Replace(validV2TaskReport(entity),
				"- [x] `scope_vocabulary` — Evidence: `tasks/T-E01-F01-001.md`.",
				"- [x] `scope_vocabulary` — This item has no evidence\n"+marker+" Evidence: `tasks/T-E01-F01-001.md`.", 1)
			require.NoError(t, os.WriteFile(paths.Report, []byte(report), 0o644))
			err = ValidateEntity(root, entity)
			require.ErrorContains(t, err, "missing evidence")
		})
	}
}

func TestValidateEntity_V2StandardFeatureRequiresCapabilityMapDecision(t *testing.T) {
	root := t.TempDir()
	entity := &models.Feature{BaseEntity: models.BaseEntity{Key: "E01-F01", FilePath: stringPtr("docs/plan/E01/F01/feature.md")}}
	paths, err := ArtifactPaths(root, entity)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(paths.Report), 0o755); err != nil {
		t.Fatal(err)
	}
	report := validV2FeatureReport(entity)
	if err := os.WriteFile(paths.Report, []byte(report), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateEntity(root, entity); err != nil {
		t.Fatalf("ValidateEntity() error = %v", err)
	}
	withoutRelatedWork := strings.Replace(report, "- [x] `related_work` — Evidence: `docs/plan/E01/F02/research-report.md`.\n", "", 1)
	if err := os.WriteFile(paths.Report, []byte(withoutRelatedWork), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateEntity(root, entity); err == nil || !strings.Contains(err.Error(), "requires module \"related_work\"") {
		t.Fatalf("ValidateEntity() error = %v, want missing related_work module", err)
	}
	withoutDecision := strings.Replace(report, "| Existing capability | `docs/plan/E01/F02/research-report.md` | EXTEND |", "No related capability was found.", 1)
	if err := os.WriteFile(paths.Report, []byte(withoutDecision), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateEntity(root, entity); err == nil || !strings.Contains(err.Error(), "capability map must contain") {
		t.Fatalf("ValidateEntity() error = %v, want missing capability-map decision", err)
	}
}

func TestValidateEntity_V2ComplexResearchRequiresRiskAndAlternativeAnalysis(t *testing.T) {
	root := t.TempDir()
	entity := taskEntity()
	paths, err := ArtifactPaths(root, entity)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(paths.Report), 0o755); err != nil {
		t.Fatal(err)
	}
	report := strings.Replace(validV2TaskReport(entity), "rigor: simple", "rigor: complex", 1)
	report = strings.Replace(report, "## Findings", "- [x] `dependency_impact` — Evidence: `internal/runner/controller.go` calls the transition service.\n- [x] `cross_boundary_risks` — Evidence: `internal/runner` crosses the workflow boundary.\n- [x] `alternatives` — Evidence: `internal/runner/controller.go`; a direct repository call would bypass validation.\n\n## Findings", 1)
	if err := os.WriteFile(paths.Report, []byte(report), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateEntity(root, entity); err != nil {
		t.Fatalf("ValidateEntity() error = %v", err)
	}
}

func TestValidateEntity_AcceptsLegacyReportWithoutPlan(t *testing.T) {
	root := t.TempDir()
	entity := taskEntity()
	paths, err := ArtifactPaths(root, entity)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(paths.Report), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.Report, []byte(legacyReport(entity)), 0o644); err != nil {
		t.Fatal(err)
	}
	legacyPlan := "---\nentity_key: " + entity.GetKey() + "\nentity_type: task\nrecipe: universal\nrigor: simple\ncategories: [backend]\nsource_set: [internal/runner]\nrelated_work: false\n---\n# Historical research plan\n"
	if err := os.WriteFile(filepath.Join(filepath.Dir(paths.Report), entity.GetKey()+".research-plan.md"), []byte(legacyPlan), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateEntity(root, entity); err != nil {
		t.Fatalf("ValidateEntity() legacy report/pair error = %v", err)
	}
}

func TestLoadCatalog_UsesConfiguredV1SharkDataPath(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".sharkconfig.json"), []byte(`{"shark_data_path":"custom-data"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	catalogPath := filepath.Join(root, "custom-data", "research", "recipes.yaml")
	if err := os.MkdirAll(filepath.Dir(catalogPath), 0o755); err != nil {
		t.Fatal(err)
	}
	data := "version: 1.0\nrecipes:\n  custom:\n    entity_types: [task]\n    required_plan_sections: [Scope]\n    required_report_sections: [Scope]\n    categories: [backend]\n    rigor:\n      simple:\n        description: test\n"
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

func taskEntity() models.Entity {
	return &models.Task{BaseEntity: models.BaseEntity{Key: "T-E01-F01-001", FilePath: stringPtr("docs/plan/E01/F01/tasks/T-E01-F01-001.md")}, Status: "research"}
}

func validV2TaskReport(entity models.Entity) string {
	return "---\nresearch_schema: 2\nrigor: simple\ncategories: [backend]\nrelated_work: false\n---\n# Research report\n\n## Scope\nAdvance this task through the existing runner transition.\n\n## Research checklist\n- [x] `scope_vocabulary` — Evidence: `tasks/T-E01-F01-001.md`.\n- [x] `affected_implementation_or_contract` — Evidence: `internal/runner/controller.go` transition path.\n\n## Findings\nThe task uses the established transition service and references the parent Capability map at `docs/plan/E01/F01/research-report.md`.\n\n## Decisions\nExtend the established runner transition instead of creating another status path.\n\n## Sources\n- `tasks/T-E01-F01-001.md`\n- `internal/runner/controller.go`\n- `docs/plan/E01/F01/research-report.md` (parent Capability map)\n"
}

func validV2FeatureReport(entity models.Entity) string {
	return "---\nresearch_schema: 2\nrigor: standard\ncategories: [backend]\nrelated_work: true\n---\n# Research report\n\n## Scope\nExtend the existing feature capability.\n\n## Research checklist\n- [x] `scope_vocabulary` — Evidence: `docs/plan/E01/F01/feature.md`.\n- [x] `affected_implementation_or_contract` — Evidence: `internal/services/feature_service.go`.\n- [x] `related_work` — Evidence: `docs/plan/E01/F02/research-report.md`.\n- [x] `pattern_contract` — Evidence: `internal/services/feature_service.go` service boundary.\n\n## Capability map\n| Capability | Source | Decision |\n| --- | --- | --- |\n| Existing capability | `docs/plan/E01/F02/research-report.md` | EXTEND |\n\n## Findings\nThe feature extends the established service contract.\n\n## Decisions\nReuse the service boundary and extend its capability.\n\n## Sources\n- `docs/plan/E01/F01/feature.md`\n- `internal/services/feature_service.go`\n"
}

func legacyReport(entity models.Entity) string {
	return "---\nentity_key: " + entity.GetKey() + "\nentity_type: task\nrecipe: universal\nrigor: simple\ncategories: [backend]\nsource_set: [internal/runner]\nrelated_work: false\n---\n# Research report\n\n## Scope\nLegacy task scope.\n\n## Capability map\nNo related work applies.\n\n## Ubiquitous vocabulary\nTask: atomic work.\n\n## Findings\nUse the established runner.\n\n## Decisions\nExtend the established runner.\n\n## Sources\n- `internal/runner`\n"
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
