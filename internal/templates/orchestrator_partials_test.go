package templates

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

const canonicalPromptsDir = "../../internal/sharkdata/default_data/prompts"

func canonicalPromptPath(elem ...string) string {
	parts := append([]string{canonicalPromptsDir}, elem...)
	return filepath.Join(parts...)
}

func corePartialFiles() []string {
	return []string{
		canonicalPromptPath("_partials", "_tdd_process.md"),
		canonicalPromptPath("_partials", "_sizing.md"),
		canonicalPromptPath("_partials", "_commands.md"),
	}
}

func parseCanonicalPartials(t *testing.T) *template.Template {
	t.Helper()

	tmpl, err := template.New("partials").Funcs(orchestratorFuncs()).ParseFiles(corePartialFiles()...)
	if err != nil {
		t.Fatalf("Failed to load canonical partials: %v", err)
	}

	return tmpl
}

func parseCanonicalPartialsInto(t *testing.T, tmpl *template.Template) *template.Template {
	t.Helper()

	tmpl, err := tmpl.Funcs(orchestratorFuncs()).ParseFiles(corePartialFiles()...)
	if err != nil {
		t.Fatalf("Failed to load canonical partials: %v", err)
	}

	return tmpl
}

// TestPartialsDirectoryStructure validates the directory structure matches the architecture
func TestPartialsDirectoryStructure(t *testing.T) {
	expectedDirs := []string{
		canonicalPromptsDir,
		canonicalPromptPath("epic"),
		canonicalPromptPath("feature"),
		canonicalPromptPath("task"),
		canonicalPromptPath("_partials"),
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
		canonicalPromptPath("_partials", "_tdd_process.md"),
		canonicalPromptPath("_partials", "_sizing.md"),
		canonicalPromptPath("_partials", "_commands.md"),
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
	tmpl := parseCanonicalPartials(t)

	expectedPartials := []string{"_tdd_process", "_sizing_scale", "advance"}
	for _, partialName := range expectedPartials {
		partial := tmpl.Lookup(partialName)
		if partial == nil {
			t.Fatalf("Partial %s not found in precompiled templates", partialName)
		}
	}
}

// TestTDDProcessPartialRenders validates the TDD process partial renders correctly
func TestTDDProcessPartialRenders(t *testing.T) {
	tmpl := parseCanonicalPartials(t)

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
	tmpl = parseCanonicalPartialsInto(t, tmpl)

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
	entries, err := os.ReadDir(canonicalPromptPath("_partials"))
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

		// Check that file starts with underscore and ends with .md
		name := entry.Name()
		if filepath.Ext(name) != ".md" {
			t.Errorf("File %s does not have .md extension", name)
		}
		// Naming convention check
		if !strings.HasPrefix(name, "_") {
			t.Errorf("File %s does not follow _prefix naming convention", name)
		}
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
