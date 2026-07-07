package templates

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPartialTemplates_Syntax(t *testing.T) {
	tests := []struct {
		name         string
		templateFile string
		templateName string
		expectedText string
	}{
		{
			name:         "TDD process partial",
			templateFile: canonicalPromptPath("_partials", "_tdd_process.md"),
			templateName: "_tdd_process",
			expectedText: "TDD PROCESS:",
		},
		{
			name:         "Sizing scale partial",
			templateFile: canonicalPromptPath("_partials", "_sizing.md"),
			templateName: "_sizing_scale",
			expectedText: "SIZE SCALE",
		},
		{
			name:         "Advance command partial",
			templateFile: canonicalPromptPath("_partials", "_commands.md"),
			templateName: "advance",
			expectedText: "shark status advance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the partial template
			tmpl, err := template.ParseFiles(tt.templateFile)
			require.NoError(t, err, "Template should parse without errors")
			require.NotNil(t, tmpl, "Template should not be nil")

			// Verify the template defines the expected name
			definedTemplate := tmpl.Lookup(tt.templateName)
			require.NotNil(t, definedTemplate, "Template should define %s", tt.templateName)

			// Execute the template to verify syntax
			var buf bytes.Buffer
			data := map[string]string{
				"primary_doc":   "test.md",
				"related_docs":  "doc1.md, doc2.md",
				"related_tasks": "E07-F30-001",
			}
			err = definedTemplate.Execute(&buf, data)
			require.NoError(t, err, "Template should execute without errors")

			// Verify output contains expected text
			output := buf.String()
			assert.Contains(t, output, tt.expectedText, "Output should contain expected text")
		})
	}
}

func TestPartialTemplates_TDDProcess(t *testing.T) {
	tmpl, err := template.ParseFiles(canonicalPromptPath("_partials", "_tdd_process.md"))
	require.NoError(t, err)

	definedTemplate := tmpl.Lookup("_tdd_process")
	require.NotNil(t, definedTemplate)

	var buf bytes.Buffer
	err = definedTemplate.Execute(&buf, nil)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Write failing test first (red)")
	assert.Contains(t, output, "Implement minimum code to pass (green)")
	assert.Contains(t, output, "Refactor while keeping tests green")
	assert.Contains(t, output, "Commit when test suite passes")
}

func TestPartialTemplates_LoadAll(t *testing.T) {
	// Load the core Go-template partials at once to verify they can coexist.
	tmpl := parseCanonicalPartials(t)

	// Verify all three partials are defined
	assert.NotNil(t, tmpl.Lookup("_tdd_process"), "TDD process partial should be defined")
	assert.NotNil(t, tmpl.Lookup("_sizing_scale"), "Sizing scale partial should be defined")
	assert.NotNil(t, tmpl.Lookup("advance"), "Advance command partial should be defined")
}

func TestPartialTemplates_Include(t *testing.T) {
	// Test that partials can be included in other templates
	mainTemplate := `Main content here.
{{template "_tdd_process" .}}
End of main content.`

	tmpl, err := template.New("main").Parse(mainTemplate)
	require.NoError(t, err)

	// Parse and add the partial
	_, err = tmpl.ParseFiles(canonicalPromptPath("_partials", "_tdd_process.md"))
	require.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, nil)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Main content here.")
	assert.Contains(t, output, "TDD PROCESS:")
	assert.Contains(t, output, "Write failing test first")
	assert.Contains(t, output, "End of main content.")
}
