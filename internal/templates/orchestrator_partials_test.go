package templates

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

// TestPartialsDirectoryStructure validates the directory structure matches the architecture
func TestPartialsDirectoryStructure(t *testing.T) {
	expectedDirs := []string{
		"../../templates",
		"../../templates/epic",
		"../../templates/feature",
		"../../templates/task",
		"../../templates/partials",
	}

	for _, dir := range expectedDirs {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("Directory %s does not exist: %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("%s is not a directory", dir)
		}
	}
}

// TestPartialsExist validates all required partial templates exist
func TestPartialsExist(t *testing.T) {
	expectedPartials := []string{
		"../../templates/partials/_tdd_process.tmpl",
		"../../templates/partials/_exit_gate.tmpl",
		"../../templates/partials/_read_section.tmpl",
	}

	for _, partial := range expectedPartials {
		info, err := os.Stat(partial)
		if err != nil {
			t.Fatalf("Partial %s does not exist: %v", partial, err)
		}
		if info.IsDir() {
			t.Fatalf("%s is a directory, expected file", partial)
		}
	}
}

// TestPartialsLoadViaParseGlob validates templates can be loaded via ParseGlob
func TestPartialsLoadViaParseGlob(t *testing.T) {
	tmpl, err := template.ParseGlob("../../templates/*/*.tmpl")
	if err != nil {
		t.Fatalf("Failed to load templates via ParseGlob: %v", err)
	}

	expectedPartials := []string{"_tdd_process", "_exit_gate", "_read_section"}
	for _, partialName := range expectedPartials {
		partial := tmpl.Lookup(partialName)
		if partial == nil {
			t.Fatalf("Partial %s not found in precompiled templates", partialName)
		}
	}
}

// TestTDDProcessPartialRenders validates the TDD process partial renders correctly
func TestTDDProcessPartialRenders(t *testing.T) {
	tmpl, err := template.ParseGlob("../../templates/*/*.tmpl")
	if err != nil {
		t.Fatalf("Failed to load templates: %v", err)
	}

	partial := tmpl.Lookup("_tdd_process")
	if partial == nil {
		t.Fatal("_tdd_process partial not found")
	}

	result, err := executeTemplate(partial, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to render _tdd_process: %v", err)
	}

	// Verify output contains expected TDD steps
	expectedPhrases := []string{
		"TDD PROCESS:",
		"Write failing test",
		"red",
		"Implement minimum",
		"green",
		"Refactor",
		"Commit",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(result, phrase) {
			t.Errorf("Expected phrase '%s' not found in rendered output", phrase)
		}
	}
}

// TestExitGatePartialRenders validates the exit gate partial renders correctly
func TestExitGatePartialRenders(t *testing.T) {
	tmpl, err := template.ParseGlob("../../templates/*/*.tmpl")
	if err != nil {
		t.Fatalf("Failed to load templates: %v", err)
	}

	partial := tmpl.Lookup("_exit_gate")
	if partial == nil {
		t.Fatal("_exit_gate partial not found")
	}

	result, err := executeTemplate(partial, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to render _exit_gate: %v", err)
	}

	// Verify output contains expected exit gate items
	expectedPhrases := []string{
		"EXIT GATE:",
		"acceptance criteria",
		"Tests passing",
		"Code reviewed",
		"Documentation",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(result, phrase) {
			t.Errorf("Expected phrase '%s' not found in rendered output", phrase)
		}
	}
}

// TestReadSectionPartialWithMinimalData validates _read_section with only primary doc
func TestReadSectionPartialWithMinimalData(t *testing.T) {
	tmpl, err := template.ParseGlob("../../templates/*/*.tmpl")
	if err != nil {
		t.Fatalf("Failed to load templates: %v", err)
	}

	partial := tmpl.Lookup("_read_section")
	if partial == nil {
		t.Fatal("_read_section partial not found")
	}

	data := map[string]interface{}{
		"primary_doc": "docs/task.md",
	}

	result, err := executeTemplate(partial, data)
	if err != nil {
		t.Fatalf("Failed to render _read_section: %v", err)
	}

	// With only primary_doc, should have (1) but not (2) or (3)
	if !strings.Contains(result, "(1)") {
		t.Error("Expected '(1)' in output")
	}
	if strings.Contains(result, "(2)") || strings.Contains(result, "(3)") {
		t.Error("Should not have (2) or (3) with only primary_doc")
	}
	if !strings.Contains(result, "docs/task.md") {
		t.Error("Expected primary_doc value in output")
	}
}

// TestReadSectionPartialWithRelatedDocs validates _read_section with related docs
func TestReadSectionPartialWithRelatedDocs(t *testing.T) {
	tmpl, err := template.ParseGlob("../../templates/*/*.tmpl")
	if err != nil {
		t.Fatalf("Failed to load templates: %v", err)
	}

	partial := tmpl.Lookup("_read_section")
	if partial == nil {
		t.Fatal("_read_section partial not found")
	}

	data := map[string]interface{}{
		"primary_doc":  "docs/task.md",
		"related_docs": "docs/api.md, docs/design.md",
	}

	result, err := executeTemplate(partial, data)
	if err != nil {
		t.Fatalf("Failed to render _read_section: %v", err)
	}

	// Should have (1) and (2) but not (3)
	if !strings.Contains(result, "(1)") || !strings.Contains(result, "(2)") {
		t.Error("Expected '(1)' and '(2)' in output")
	}
	if strings.Contains(result, "(3)") {
		t.Error("Should not have (3) with related_docs but no related_tasks")
	}
	if !strings.Contains(result, "docs/api.md") {
		t.Error("Expected related_docs value in output")
	}
}

// TestReadSectionPartialWithAllData validates _read_section with all data
func TestReadSectionPartialWithAllData(t *testing.T) {
	tmpl, err := template.ParseGlob("../../templates/*/*.tmpl")
	if err != nil {
		t.Fatalf("Failed to load templates: %v", err)
	}

	partial := tmpl.Lookup("_read_section")
	if partial == nil {
		t.Fatal("_read_section partial not found")
	}

	data := map[string]interface{}{
		"primary_doc":   "docs/task.md",
		"related_docs":  "docs/api.md",
		"related_tasks": "E07-F30-001",
	}

	result, err := executeTemplate(partial, data)
	if err != nil {
		t.Fatalf("Failed to render _read_section: %v", err)
	}

	// Should have (1), (2), and (3)
	if !strings.Contains(result, "(1)") || !strings.Contains(result, "(2)") || !strings.Contains(result, "(3)") {
		t.Error("Expected '(1)', '(2)', and '(3)' in output")
	}
	if !strings.Contains(result, "E07-F30-001") {
		t.Error("Expected related_tasks value in output")
	}
}

// TestReadSectionSmartNumbering validates smart numbering with only related_tasks
func TestReadSectionSmartNumbering(t *testing.T) {
	tmpl, err := template.ParseGlob("../../templates/*/*.tmpl")
	if err != nil {
		t.Fatalf("Failed to load templates: %v", err)
	}

	partial := tmpl.Lookup("_read_section")
	if partial == nil {
		t.Fatal("_read_section partial not found")
	}

	data := map[string]interface{}{
		"primary_doc":   "docs/task.md",
		"related_tasks": "E07-F30-001",
	}

	result, err := executeTemplate(partial, data)
	if err != nil {
		t.Fatalf("Failed to render _read_section: %v", err)
	}

	// Should have (1) and (2), not (3)
	if !strings.Contains(result, "(1)") || !strings.Contains(result, "(2)") {
		t.Error("Expected '(1)' and '(2)' in output when no related_docs")
	}
	if strings.Contains(result, "(3)") {
		t.Error("Should not have (3) when related_docs is empty")
	}
}

// TestPartialsCanBeIncludedInTemplates validates partials work in template includes
func TestPartialsCanBeIncludedInTemplates(t *testing.T) {
	// Create a test template that includes the _tdd_process partial
	testTemplateContent := `{{define "test_with_partial"}}Main content here.
{{template "_tdd_process" .}}
End of content.{{end}}`

	tmpl, err := template.New("test").Parse(testTemplateContent)
	if err != nil {
		t.Fatalf("Failed to parse test template: %v", err)
	}

	// Load all partials
	tmpl, err = tmpl.ParseGlob("../../templates/*/*.tmpl")
	if err != nil {
		t.Fatalf("Failed to load partials: %v", err)
	}

	result, err := executeTemplate(tmpl.Lookup("test_with_partial"), map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to render test template with partial: %v", err)
	}

	// Verify main content and partial content both present
	if !strings.Contains(result, "Main content here") {
		t.Error("Main template content not found")
	}
	if !strings.Contains(result, "TDD PROCESS:") {
		t.Error("Partial content not included in template")
	}
	if !strings.Contains(result, "Write failing test") {
		t.Error("Partial implementation not found in output")
	}
}

// TestPartialNamingConvention validates partials use _prefix naming
func TestPartialNamingConvention(t *testing.T) {
	entries, err := os.ReadDir("../../templates/partials")
	if err != nil {
		t.Fatalf("Failed to read partials directory: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("No files found in templates/partials directory")
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !entry.Type().IsRegular() {
			continue
		}

		// Check that file starts with underscore and ends with .tmpl
		name := entry.Name()
		if filepath.Ext(name) != ".tmpl" {
			t.Errorf("File %s does not have .tmpl extension", name)
		}
		// Naming convention check
		if !strings.HasPrefix(name, "_") {
			t.Errorf("File %s does not follow _prefix naming convention", name)
		}
	}
}

// TestReadSectionNoEmptyLines validates _read_section doesn't create empty lines
func TestReadSectionNoEmptyLines(t *testing.T) {
	tmpl, err := template.ParseGlob("../../templates/*/*.tmpl")
	if err != nil {
		t.Fatalf("Failed to load templates: %v", err)
	}

	partial := tmpl.Lookup("_read_section")
	if partial == nil {
		t.Fatal("_read_section partial not found")
	}

	// Test with only primary doc
	data := map[string]interface{}{
		"primary_doc": "docs/task.md",
	}

	result, err := executeTemplate(partial, data)
	if err != nil {
		t.Fatalf("Failed to render _read_section: %v", err)
	}

	// Count blank lines - should be minimal
	lines := strings.Split(result, "\n")
	blankCount := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			blankCount++
		}
	}

	// Allow 1-2 blank lines max (beginning/end)
	if blankCount > 2 {
		t.Errorf("Too many blank lines in output: %d, output:\n%s", blankCount, result)
	}
}

// Helper function to execute a template
func executeTemplate(tmpl *template.Template, data map[string]interface{}) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
