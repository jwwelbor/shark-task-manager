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
			templateFile: "../../shark-templates/partials/_tdd_process.tmpl",
			templateName: "_tdd_process",
			expectedText: "TDD PROCESS:",
		},
		{
			name:         "Exit gate partial",
			templateFile: "../../shark-templates/partials/_exit_gate.tmpl",
			templateName: "_exit_gate",
			expectedText: "EXIT GATE:",
		},
		{
			name:         "Read section partial",
			templateFile: "../../shark-templates/partials/_read_section.tmpl",
			templateName: "_read_section",
			expectedText: "READ:",
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
	tmpl, err := template.ParseFiles("../../shark-templates/partials/_tdd_process.tmpl")
	require.NoError(t, err)

	definedTemplate := tmpl.Lookup("_tdd_process")
	require.NotNil(t, definedTemplate)

	var buf bytes.Buffer
	err = definedTemplate.Execute(&buf, nil)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "1. Write failing test first (red)")
	assert.Contains(t, output, "2. Implement minimum code to pass (green)")
	assert.Contains(t, output, "3. Refactor while keeping tests green")
	assert.Contains(t, output, "4. Commit when test suite passes")
}

func TestPartialTemplates_ExitGate(t *testing.T) {
	tmpl, err := template.ParseFiles("../../shark-templates/partials/_exit_gate.tmpl")
	require.NoError(t, err)

	definedTemplate := tmpl.Lookup("_exit_gate")
	require.NotNil(t, definedTemplate)

	var buf bytes.Buffer
	err = definedTemplate.Execute(&buf, nil)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "All acceptance criteria met")
	assert.Contains(t, output, "Tests passing (unit + integration)")
	assert.Contains(t, output, "Code reviewed and approved")
	assert.Contains(t, output, "Documentation updated")
}

func TestPartialTemplates_ReadSection_SmartNumbering(t *testing.T) {
	tmpl, err := template.ParseFiles("../../shark-templates/partials/_read_section.tmpl")
	require.NoError(t, err)

	definedTemplate := tmpl.Lookup("_read_section")
	require.NotNil(t, definedTemplate)

	tests := []struct {
		name           string
		data           map[string]string
		expectedOutput []string
	}{
		{
			name: "Only primary doc",
			data: map[string]string{
				"primary_doc":   "task.md",
				"related_docs":  "",
				"related_tasks": "",
			},
			expectedOutput: []string{
				"(1) task.md",
			},
		},
		{
			name: "Primary doc and related docs",
			data: map[string]string{
				"primary_doc":   "task.md",
				"related_docs":  "prd.md, arch.md",
				"related_tasks": "",
			},
			expectedOutput: []string{
				"(1) task.md",
				"(2) Related docs: prd.md, arch.md",
			},
		},
		{
			name: "All fields populated",
			data: map[string]string{
				"primary_doc":   "task.md",
				"related_docs":  "prd.md",
				"related_tasks": "E07-F29-003",
			},
			expectedOutput: []string{
				"(1) task.md",
				"(2) Related docs: prd.md",
				"(3) Related tasks: E07-F29-003",
			},
		},
		{
			name: "Primary doc and related tasks (no docs)",
			data: map[string]string{
				"primary_doc":   "task.md",
				"related_docs":  "",
				"related_tasks": "E07-F29-003",
			},
			expectedOutput: []string{
				"(1) task.md",
				"(2) Related tasks: E07-F29-003",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err = definedTemplate.Execute(&buf, tt.data)
			require.NoError(t, err)

			output := buf.String()
			for _, expected := range tt.expectedOutput {
				assert.Contains(t, output, expected, "Output should contain: %s", expected)
			}
		})
	}
}

func TestPartialTemplates_LoadAll(t *testing.T) {
	// Load all partials at once to verify they can coexist
	tmpl, err := template.ParseGlob("../../shark-templates/partials/_*.tmpl")
	require.NoError(t, err, "All partials should parse without errors")

	// Verify all three partials are defined
	assert.NotNil(t, tmpl.Lookup("_tdd_process"), "TDD process partial should be defined")
	assert.NotNil(t, tmpl.Lookup("_exit_gate"), "Exit gate partial should be defined")
	assert.NotNil(t, tmpl.Lookup("_read_section"), "Read section partial should be defined")
}

func TestPartialTemplates_Include(t *testing.T) {
	// Test that partials can be included in other templates
	mainTemplate := `Main content here.
{{template "_tdd_process" .}}
End of main content.`

	tmpl, err := template.New("main").Parse(mainTemplate)
	require.NoError(t, err)

	// Parse and add the partial
	_, err = tmpl.ParseFiles("../../shark-templates/partials/_tdd_process.tmpl")
	require.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, nil)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Main content here.")
	assert.Contains(t, output, "TDD PROCESS:")
	assert.Contains(t, output, "1. Write failing test first")
	assert.Contains(t, output, "End of main content.")
}
