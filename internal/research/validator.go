package research

import (
	"fmt"
	"os"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"gopkg.in/yaml.v3"
)

// artifact holds the frontmatter fields shark cannot derive on its own:
// Rigor and Categories are the researcher's classification of the work,
// and RelatedWork records whether related capability work exists. Identity
// (entity key/type) is not parsed here — shark already knows which entity
// it is validating (ArtifactPaths resolves the report path from the entity,
// not the reverse), so there is nothing for the agent to restate. Recipe is
// still parsed but optional: it defaults to "universal" in readArtifact
// since that is the only recipe that exists today.
type artifact struct {
	ResearchSchema int      `yaml:"research_schema"`
	Recipe         string   `yaml:"recipe"`
	Rigor          string   `yaml:"rigor"`
	Categories     []string `yaml:"categories"`
	SourceSet      []string `yaml:"source_set"`
	RelatedWork    *bool    `yaml:"related_work"`
}

// ValidateEntity checks the declarative research artifact contract. It checks
// presence, front matter, selected recipe/tier/categories, checked modules,
// and source evidence only; it deliberately does not evaluate prose quality.
func ValidateEntity(projectRoot string, entity models.Entity) error {
	catalog, err := LoadCatalog(projectRoot)
	if err != nil {
		return err
	}
	paths, err := ArtifactPaths(projectRoot, entity)
	if err != nil {
		return err
	}
	report, reportBody, err := readArtifact(paths.Report)
	if err != nil {
		return fmt.Errorf("research report: %w", err)
	}
	if report.ResearchSchema == 0 {
		return validateLegacyReport(catalog, entity, report, reportBody)
	}
	if report.ResearchSchema != 2 {
		return fmt.Errorf("research report has unsupported research_schema %d", report.ResearchSchema)
	}
	return validateV2Report(catalog, entity, report, reportBody)
}

func validateV2Report(catalog *Catalog, entity models.Entity, report artifact, body string) error {
	if err := validateIdentity("research report", report); err != nil {
		return err
	}
	recipe, ok := catalog.Recipes[report.Recipe]
	if !ok {
		return fmt.Errorf("unknown research recipe %q", report.Recipe)
	}
	if err := validateRecipeSelection(recipe, report, entity); err != nil {
		return err
	}
	if report.RelatedWork == nil {
		return fmt.Errorf("research report must include related_work")
	}
	if err := requireSections(body, recipe.RequiredReportSections); err != nil {
		return fmt.Errorf("research report: %w", err)
	}
	entries, err := parseChecklist(body)
	if err != nil {
		return fmt.Errorf("research report: %w", err)
	}
	if err := validateSelectedModules(recipe, entity, report, entries); err != nil {
		return err
	}
	if requiresCapabilityMap(entity, report, entries) && !capabilityMapHasDecision(body) {
		return fmt.Errorf("research report: capability map must contain a REUSE, EXTEND, NEW, or CONTRADICTS decision")
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

type checklistEntry struct {
	ID        string
	Completed bool
	Evidence  string
}

func parseChecklist(body string) ([]checklistEntry, error) {
	section, ok := sectionBody(body, "Research checklist")
	if !ok || strings.TrimSpace(section) == "" {
		return nil, fmt.Errorf("missing required section \"Research checklist\"")
	}
	var entries []checklistEntry
	lines := strings.Split(section, "\n")
	for index := 0; index < len(lines); index++ {
		line := strings.TrimSpace(lines[index])
		if !strings.HasPrefix(line, "- [") {
			continue
		}
		for index+1 < len(lines) {
			continuation := strings.TrimSpace(lines[index+1])
			if continuation == "" || isMarkdownListItem(continuation) {
				break
			}
			line += " " + continuation
			index++
		}
		if len(line) < 6 || line[4] != ']' {
			return nil, fmt.Errorf("invalid checklist entry %q", line)
		}
		rest := strings.TrimSpace(line[5:])
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			return nil, fmt.Errorf("checklist entry is missing a module ID")
		}
		id := strings.Trim(fields[0], "`")
		evidenceAt := strings.Index(strings.ToLower(rest), "evidence:")
		if evidenceAt < 0 || strings.TrimSpace(rest[evidenceAt+len("evidence:"):]) == "" {
			return nil, fmt.Errorf("checklist module %q is missing evidence", id)
		}
		entries = append(entries, checklistEntry{ID: id, Completed: strings.EqualFold(line[3:4], "x"), Evidence: strings.TrimSpace(rest[evidenceAt+len("evidence:"):])})
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("research checklist must select at least one module")
	}
	for _, entry := range entries {
		if !entry.Completed {
			return nil, fmt.Errorf("checklist module %q is unchecked", entry.ID)
		}
	}
	return entries, nil
}

func isMarkdownListItem(line string) bool {
	if len(line) >= 2 && (line[0] == '-' || line[0] == '*' || line[0] == '+') && line[1] == ' ' {
		return true
	}
	index := 0
	for index < len(line) && line[index] >= '0' && line[index] <= '9' {
		index++
	}
	if index > 0 && index+1 < len(line) && (line[index] == '.' || line[index] == ')') && line[index+1] == ' ' {
		return true
	}
	return false
}

func validateSelectedModules(recipe Recipe, entity models.Entity, report artifact, entries []checklistEntry) error {
	selected := make(map[string]bool, len(entries))
	for _, entry := range entries {
		if selected[entry.ID] {
			return fmt.Errorf("research checklist selects module %q more than once", entry.ID)
		}
		selected[entry.ID] = true
		module, ok := recipe.Modules[entry.ID]
		if !ok {
			return fmt.Errorf("research checklist selects unknown module %q", entry.ID)
		}
		if !moduleApplies(module, entity, report.Categories) {
			return fmt.Errorf("research checklist module %q is not applicable to the selected entity or categories", entry.ID)
		}
	}
	for _, core := range []string{"scope_vocabulary", "affected_implementation_or_contract"} {
		if !selected[core] {
			return fmt.Errorf("research rigor %q requires module %q", report.Rigor, core)
		}
	}
	if entity.GetEntityType() == models.EntityTypeEpic || entity.GetEntityType() == models.EntityTypeFeature {
		if !selected["related_work"] {
			return fmt.Errorf("research %s requires module %q", entity.GetEntityType(), "related_work")
		}
	}
	switch report.Rigor {
	case "simple":
		return nil
	case "standard":
		if selected["pattern_contract"] || selected["dependency_impact"] {
			return nil
		}
		return fmt.Errorf("research rigor standard requires pattern_contract or dependency_impact")
	case "complex":
		if !selected["pattern_contract"] && !selected["dependency_impact"] {
			return fmt.Errorf("research rigor complex requires pattern_contract or dependency_impact")
		}
		for _, module := range []string{"cross_boundary_risks", "alternatives"} {
			if !selected[module] {
				return fmt.Errorf("research rigor complex requires module %q", module)
			}
		}
		return nil
	default:
		return fmt.Errorf("research recipe %q does not define rigor %q", report.Recipe, report.Rigor)
	}
}

func moduleApplies(module Module, entity models.Entity, categories []string) bool {
	if len(module.EntityTypes) > 0 && !contains(module.EntityTypes, string(entity.GetEntityType())) {
		return false
	}
	if len(module.Categories) == 0 {
		return true
	}
	for _, category := range categories {
		if contains(module.Categories, category) {
			return true
		}
	}
	return false
}

func requiresCapabilityMap(entity models.Entity, report artifact, entries []checklistEntry) bool {
	if entity.GetEntityType() == models.EntityTypeEpic || entity.GetEntityType() == models.EntityTypeFeature || *report.RelatedWork {
		return true
	}
	for _, entry := range entries {
		if entry.ID == "related_work" {
			return true
		}
	}
	return false
}

func validateLegacyReport(catalog *Catalog, entity models.Entity, report artifact, body string) error {
	if err := validateIdentity("legacy research report", report); err != nil {
		return err
	}
	recipe, ok := catalog.Recipes[report.Recipe]
	if !ok {
		return fmt.Errorf("unknown research recipe %q", report.Recipe)
	}
	if err := validateRecipeSelection(recipe, report, entity); err != nil {
		return err
	}
	if len(report.SourceSet) == 0 {
		return fmt.Errorf("legacy research report must include source references")
	}
	sections := recipe.RequiredReportSections
	if len(sections) == 0 || contains(sections, "Research checklist") {
		sections = []string{"Scope", "Capability map", "Ubiquitous vocabulary", "Findings", "Decisions", "Sources"}
	}
	if err := requireSections(body, sections); err != nil {
		return fmt.Errorf("legacy research report: %w", err)
	}
	if report.RelatedWork != nil && *report.RelatedWork && !capabilityMapHasDecision(body) {
		return fmt.Errorf("legacy research report: capability map is required when related work exists")
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
	if value.Recipe == "" {
		value.Recipe = "universal"
	}
	return value, strings.Join(lines[end+1:], "\n"), nil
}

func validateIdentity(name string, value artifact) error {
	if value.Rigor == "" {
		return fmt.Errorf("%s front matter must include rigor", name)
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
	rest, ok := sectionBody(body, section)
	return ok && strings.TrimSpace(rest) != ""
}

func sectionBody(body, section string) (string, bool) {
	needle := "## " + strings.ToLower(section)
	lower := strings.ToLower(body)
	start := strings.Index(lower, needle)
	if start < 0 {
		return "", false
	}
	rest := body[start+len(needle):]
	if next := strings.Index(strings.ToLower(rest), "\n## "); next >= 0 {
		rest = rest[:next]
	}
	return rest, true
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
