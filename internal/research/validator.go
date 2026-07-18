package research

import (
	"fmt"
	"os"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"gopkg.in/yaml.v3"
)

type artifact struct {
	EntityKey   string   `yaml:"entity_key"`
	EntityType  string   `yaml:"entity_type"`
	Recipe      string   `yaml:"recipe"`
	Rigor       string   `yaml:"rigor"`
	Categories  []string `yaml:"categories"`
	SourceSet   []string `yaml:"source_set"`
	RelatedWork *bool    `yaml:"related_work"`
}

// ValidateEntity checks the declarative research artifact contract. It checks
// presence, front matter, selected recipe/tier/categories, sections, and
// source references only; it deliberately does not evaluate prose quality.
func ValidateEntity(projectRoot string, entity models.Entity) error {
	catalog, err := LoadCatalog(projectRoot)
	if err != nil {
		return err
	}
	paths, err := ArtifactPaths(projectRoot, entity)
	if err != nil {
		return err
	}
	plan, planBody, err := readArtifact(paths.Plan)
	if err != nil {
		return fmt.Errorf("research plan: %w", err)
	}
	report, reportBody, err := readArtifact(paths.Report)
	if err != nil {
		return fmt.Errorf("research report: %w", err)
	}
	recipe, err := validateArtifactMetadata(catalog, entity, plan, report)
	if err != nil {
		return err
	}
	return validateArtifactStructure(recipe, plan, planBody, report, reportBody)
}

func validateArtifactMetadata(catalog *Catalog, entity models.Entity, plan, report artifact) (Recipe, error) {
	if err := validateIdentity("research plan", plan, entity); err != nil {
		return Recipe{}, err
	}
	if err := validateIdentity("research report", report, entity); err != nil {
		return Recipe{}, err
	}
	if err := validateMatchingMetadata(plan, report); err != nil {
		return Recipe{}, err
	}
	recipe, ok := catalog.Recipes[plan.Recipe]
	if !ok {
		return Recipe{}, fmt.Errorf("unknown research recipe %q", plan.Recipe)
	}
	if err := validateRecipeSelection(recipe, plan, entity); err != nil {
		return Recipe{}, err
	}
	return recipe, nil
}

func validateMatchingMetadata(plan, report artifact) error {
	if plan.Recipe != report.Recipe || plan.Rigor != report.Rigor || !sameStrings(plan.Categories, report.Categories) {
		return fmt.Errorf("research plan and report select different recipe metadata")
	}
	if plan.RelatedWork == nil || report.RelatedWork == nil {
		return fmt.Errorf("research plan and report must include related_work")
	}
	return nil
}

func validateRecipeSelection(recipe Recipe, plan artifact, entity models.Entity) error {
	if !contains(recipe.EntityTypes, string(entity.GetEntityType())) {
		return fmt.Errorf("research recipe %q does not support entity type %q", plan.Recipe, entity.GetEntityType())
	}
	if _, ok := recipe.Rigor[plan.Rigor]; !ok {
		return fmt.Errorf("research recipe %q does not define rigor %q", plan.Recipe, plan.Rigor)
	}
	if len(plan.Categories) == 0 {
		return fmt.Errorf("research recipe must select at least one category")
	}
	for _, category := range plan.Categories {
		if !contains(recipe.Categories, category) {
			return fmt.Errorf("research recipe %q does not define category %q", plan.Recipe, category)
		}
	}
	return nil
}

func validateArtifactStructure(recipe Recipe, plan artifact, planBody string, report artifact, reportBody string) error {
	if len(plan.SourceSet) == 0 || len(report.SourceSet) == 0 {
		return fmt.Errorf("research plan and report must include source references")
	}
	if err := requireSections(planBody, recipe.RequiredPlanSections); err != nil {
		return fmt.Errorf("research plan: %w", err)
	}
	if err := requireSections(reportBody, recipe.RequiredReportSections); err != nil {
		return fmt.Errorf("research report: %w", err)
	}
	if (*plan.RelatedWork || *report.RelatedWork) && !capabilityMapHasDecision(reportBody) {
		return fmt.Errorf("research report: capability map is required when related work exists")
	}
	return nil
}

func capabilityMapHasDecision(body string) bool {
	needle := "## capability map"
	lower := strings.ToLower(body)
	start := strings.Index(lower, needle)
	if start < 0 {
		return false
	}
	rest := body[start+len(needle):]
	if next := strings.Index(strings.ToLower(rest), "\n## "); next >= 0 {
		rest = rest[:next]
	}
	for _, line := range strings.Split(rest, "\n") {
		upper := strings.ToUpper(line)
		if strings.Contains(line, "|") && (strings.Contains(upper, "REUSE") ||
			strings.Contains(upper, "EXTEND") || strings.Contains(upper, "NEW") || strings.Contains(upper, "CONTRADICTS")) {
			return true
		}
	}
	return false
}

func readArtifact(path string) (artifact, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return artifact{}, "", fmt.Errorf("read %s: %w", path, err)
	}
	text := string(data)
	if !strings.HasPrefix(text, "---\n") && !strings.HasPrefix(text, "---\r\n") {
		return artifact{}, "", fmt.Errorf("%s is missing YAML front matter", path)
	}
	lines := strings.Split(text, "\n")
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return artifact{}, "", fmt.Errorf("%s has unclosed YAML front matter", path)
	}
	var value artifact
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:end], "\n")), &value); err != nil {
		return artifact{}, "", fmt.Errorf("parse %s front matter: %w", path, err)
	}
	return value, strings.Join(lines[end+1:], "\n"), nil
}

func validateIdentity(name string, value artifact, entity models.Entity) error {
	if value.EntityKey != entity.GetKey() || value.EntityType != string(entity.GetEntityType()) {
		return fmt.Errorf("%s front matter must identify %s %s", name, entity.GetEntityType(), entity.GetKey())
	}
	if value.Recipe == "" || value.Rigor == "" {
		return fmt.Errorf("%s front matter must include recipe and rigor", name)
	}
	return nil
}

func requireSections(body string, sections []string) error {
	for _, section := range sections {
		if !sectionHasContent(body, section) {
			return fmt.Errorf("missing required section %q", section)
		}
	}
	return nil
}

func sectionHasContent(body, section string) bool {
	needle := "## " + strings.ToLower(section)
	lower := strings.ToLower(body)
	start := strings.Index(lower, needle)
	if start < 0 {
		return false
	}
	rest := body[start+len(needle):]
	next := strings.Index(strings.ToLower(rest), "\n## ")
	if next >= 0 {
		rest = rest[:next]
	}
	return strings.TrimSpace(rest) != ""
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
