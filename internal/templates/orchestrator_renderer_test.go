package templates

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Fixtures Setup
func setupTestFixtures(t *testing.T) string {
	t.Helper()

	testDir := t.TempDir()
	fixturesDir := filepath.Join(testDir, "test_fixtures")

	// Create directory structure
	require.NoError(t, os.MkdirAll(filepath.Join(fixturesDir, "valid"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(fixturesDir, "partials"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(fixturesDir, "invalid"), 0755))

	// Create valid templates
	validBasic := `Task: {{.task_id}}
Title: {{.title}}
Status: {{.status}}`
	require.NoError(t, os.WriteFile(filepath.Join(fixturesDir, "valid", "basic.tmpl"), []byte(validBasic), 0644))

	validConditional := `{{if .related_docs}}Related: {{.related_docs}}{{else}}No related docs.{{end}}`
	require.NoError(t, os.WriteFile(filepath.Join(fixturesDir, "valid", "conditional.tmpl"), []byte(validConditional), 0644))

	validTier := `{{if eq .complexity_tier "SIMPLE"}}Brief instructions{{else if eq .complexity_tier "STANDARD"}}Focused instructions{{else}}Comprehensive instructions{{end}}`
	require.NoError(t, os.WriteFile(filepath.Join(fixturesDir, "valid", "tier.tmpl"), []byte(validTier), 0644))

	// Create partial template
	partial := `{{define "_test_partial"}}Partial content here.{{end}}`
	require.NoError(t, os.WriteFile(filepath.Join(fixturesDir, "partials", "_test_partial.tmpl"), []byte(partial), 0644))

	// Create template that uses partial
	withPartial := `Main template content.
{{template "_test_partial" .}}
End of template.`
	require.NoError(t, os.WriteFile(filepath.Join(fixturesDir, "valid", "with_partial.tmpl"), []byte(withPartial), 0644))

	// Create malformed template (missing end tag)
	malformed := `{{if .condition}}No closing end tag!`
	require.NoError(t, os.WriteFile(filepath.Join(fixturesDir, "invalid", "malformed.tmpl"), []byte(malformed), 0644))

	return fixturesDir
}

// ============================================================================
// Constructor Tests (API-01 to API-04)
// ============================================================================

func TestNewOrchestratorRenderer_ValidDirectory(t *testing.T) {
	// API-01: Valid template directory → returns renderer
	fixturesDir := setupTestFixtures(t)

	renderer, err := NewOrchestratorRenderer(fixturesDir)

	require.NoError(t, err)
	assert.NotNil(t, renderer)
	assert.NotNil(t, renderer.templates)
	assert.Equal(t, fixturesDir, renderer.templateDir)
}

func TestNewOrchestratorRenderer_MissingDirectory(t *testing.T) {
	// API-02: Missing directory → returns error
	renderer, err := NewOrchestratorRenderer("/nonexistent/path")

	assert.Error(t, err)
	assert.Nil(t, renderer)
	assert.Contains(t, err.Error(), "failed to parse templates")
}

func TestNewOrchestratorRenderer_MalformedTemplate(t *testing.T) {
	// API-03: Malformed template → returns parse error with file/line
	fixturesDir := setupTestFixtures(t)

	// Create a directory with ONLY malformed templates
	invalidDir := filepath.Join(fixturesDir, "invalid")

	renderer, err := NewOrchestratorRenderer(invalidDir)

	assert.Error(t, err)
	assert.Nil(t, renderer)
	assert.Contains(t, err.Error(), "failed to parse templates")
}

func TestNewOrchestratorRenderer_EmptyDirectory(t *testing.T) {
	// API-04: Empty directory → returns renderer with no templates
	emptyDir := t.TempDir()

	renderer, err := NewOrchestratorRenderer(emptyDir)

	// Empty directory is valid - no templates to parse
	require.NoError(t, err)
	assert.NotNil(t, renderer)
}

// ============================================================================
// Singleton Accessor Tests (API-05 to API-07)
// ============================================================================

func TestGetOrchestratorEngine_FirstCallCreates(t *testing.T) {
	// API-05: First call creates singleton
	// Reset singleton for testing
	resetSingleton()

	fixturesDir := setupTestFixtures(t)
	setTestTemplateDir(fixturesDir)

	engine := GetOrchestratorEngine()

	assert.NotNil(t, engine)
	assert.NotNil(t, engine.templates)
}

func TestGetOrchestratorEngine_SubsequentCallsReturnSame(t *testing.T) {
	// API-06: Subsequent calls return same instance (pointer equality)
	resetSingleton()

	fixturesDir := setupTestFixtures(t)
	setTestTemplateDir(fixturesDir)

	first := GetOrchestratorEngine()
	second := GetOrchestratorEngine()
	third := GetOrchestratorEngine()

	assert.Same(t, first, second, "Second call should return same instance")
	assert.Same(t, first, third, "Third call should return same instance")
}

func TestGetOrchestratorEngine_ConcurrentAccess(t *testing.T) {
	// API-07: Concurrent access safe (1000 goroutines, no race)
	resetSingleton()

	fixturesDir := setupTestFixtures(t)
	setTestTemplateDir(fixturesDir)

	const numGoroutines = 1000
	var wg sync.WaitGroup
	engines := make([]*OrchestratorRenderer, numGoroutines)

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			engines[idx] = GetOrchestratorEngine()
		}(i)
	}
	wg.Wait()

	// All should be the same instance
	first := engines[0]
	for i := 1; i < numGoroutines; i++ {
		assert.Same(t, first, engines[i], "Goroutine %d got different instance", i)
	}
}

// ============================================================================
// Render Method Tests (API-08 to API-12)
// ============================================================================

func TestRender_ExistingTemplate(t *testing.T) {
	// API-08: Existing template → renders correctly
	fixturesDir := setupTestFixtures(t)
	renderer, err := NewOrchestratorRenderer(fixturesDir)
	require.NoError(t, err)

	vars := map[string]string{
		"task_id": "E07-F30-001",
		"title":   "Test Task",
		"status":  "todo",
	}

	result, err := renderer.Render("valid/basic.tmpl", vars)

	require.NoError(t, err)
	assert.Contains(t, result, "Task: E07-F30-001")
	assert.Contains(t, result, "Title: Test Task")
	assert.Contains(t, result, "Status: todo")
}

func TestRender_MissingTemplate(t *testing.T) {
	// API-09: Missing template → returns error "template not found"
	fixturesDir := setupTestFixtures(t)
	renderer, err := NewOrchestratorRenderer(fixturesDir)
	require.NoError(t, err)

	vars := map[string]string{"task_id": "E07-F30-001"}

	result, err := renderer.Render("valid/nonexistent.tmpl", vars)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template not found")
	assert.Empty(t, result)
}

func TestRender_EmptyVars(t *testing.T) {
	// API-10: Empty vars → renders with empty placeholders
	fixturesDir := setupTestFixtures(t)
	renderer, err := NewOrchestratorRenderer(fixturesDir)
	require.NoError(t, err)

	result, err := renderer.Render("valid/basic.tmpl", map[string]string{})

	require.NoError(t, err)
	// Template should render with empty values
	assert.Contains(t, result, "Task:")
	assert.Contains(t, result, "Title:")
	assert.Contains(t, result, "Status:")
}

func TestRender_MissingVariable(t *testing.T) {
	// API-11: Missing variable → execution error
	fixturesDir := setupTestFixtures(t)
	renderer, err := NewOrchestratorRenderer(fixturesDir)
	require.NoError(t, err)

	// Template expects task_id, title, status but we only provide task_id
	vars := map[string]string{"task_id": "E07-F30-001"}

	result, err := renderer.Render("valid/basic.tmpl", vars)

	// Should render successfully with empty values for missing variables
	require.NoError(t, err)
	assert.Contains(t, result, "Task: E07-F30-001")
}

// ============================================================================
// Custom Function Tests (API-18 to API-26)
// ============================================================================

func TestTemplateFuncs_Eq(t *testing.T) {
	// API-18: eq function
	funcs := orchestratorFuncs()
	eqFunc := funcs["eq"].(func(a, b interface{}) bool)

	tests := []struct {
		name string
		a, b interface{}
		want bool
	}{
		{"equal strings", "a", "a", true},
		{"different strings", "a", "b", false},
		{"equal ints", 1, 1, true},
		{"different ints", 1, 2, false},
		{"different types", 1, "1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := eqFunc(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTemplateFuncs_Ne(t *testing.T) {
	// API-20: ne function
	funcs := orchestratorFuncs()
	neFunc := funcs["ne"].(func(a, b interface{}) bool)

	tests := []struct {
		name string
		a, b interface{}
		want bool
	}{
		{"equal strings", "a", "a", false},
		{"different strings", "a", "b", true},
		{"equal ints", 1, 1, false},
		{"different types", 1, "1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := neFunc(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTemplateFuncs_IsEmpty(t *testing.T) {
	// API-21, API-22, API-23: isEmpty function
	funcs := orchestratorFuncs()
	isEmptyFunc := funcs["isEmpty"].(func(s string) bool)

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty string", "", true},
		{"whitespace only", "   ", true},
		{"tab and newline", "\t\n", true},
		{"non-empty", "text", false},
		{"whitespace with text", "  text  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isEmptyFunc(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTemplateFuncs_TierHelpers(t *testing.T) {
	// API-24, API-25, API-26: isSimple, isStandard, isComplex functions
	funcs := orchestratorFuncs()

	isSimpleFunc := funcs["isSimple"].(func(tier string) bool)
	isStandardFunc := funcs["isStandard"].(func(tier string) bool)
	isComplexFunc := funcs["isComplex"].(func(tier string) bool)

	tests := []struct {
		tier       string
		wantSimple bool
		wantStd    bool
		wantCplx   bool
	}{
		{"SIMPLE", true, false, false},
		{"STANDARD", false, true, false},
		{"COMPLEX", false, false, true},
		{"simple", false, false, false}, // Case sensitive
		{"OTHER", false, false, false},
		{"", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.tier, func(t *testing.T) {
			assert.Equal(t, tt.wantSimple, isSimpleFunc(tt.tier), "isSimple")
			assert.Equal(t, tt.wantStd, isStandardFunc(tt.tier), "isStandard")
			assert.Equal(t, tt.wantCplx, isComplexFunc(tt.tier), "isComplex")
		})
	}
}

// ============================================================================
// Integration Tests: Conditionals (AC-2.1 to AC-2.4)
// ============================================================================

func TestRender_Conditionals_HideEmpty(t *testing.T) {
	// AC-2.1-T01: Empty string hides section
	fixturesDir := setupTestFixtures(t)
	renderer, err := NewOrchestratorRenderer(fixturesDir)
	require.NoError(t, err)

	vars := map[string]string{"related_docs": ""}

	result, err := renderer.Render("valid/conditional.tmpl", vars)

	require.NoError(t, err)
	assert.Equal(t, "No related docs.", result)
}

func TestRender_Conditionals_ShowPopulated(t *testing.T) {
	// AC-2.1-T02: Populated string shows section
	fixturesDir := setupTestFixtures(t)
	renderer, err := NewOrchestratorRenderer(fixturesDir)
	require.NoError(t, err)

	vars := map[string]string{"related_docs": "prd.md, arch.md"}

	result, err := renderer.Render("valid/conditional.tmpl", vars)

	require.NoError(t, err)
	assert.Equal(t, "Related: prd.md, arch.md", result)
}

func TestRender_ComplexityTier_SIMPLE(t *testing.T) {
	// AC-2.2-T01: SIMPLE tier branch
	fixturesDir := setupTestFixtures(t)
	renderer, err := NewOrchestratorRenderer(fixturesDir)
	require.NoError(t, err)

	vars := map[string]string{"complexity_tier": "SIMPLE"}

	result, err := renderer.Render("valid/tier.tmpl", vars)

	require.NoError(t, err)
	assert.Equal(t, "Brief instructions", result)
}

func TestRender_ComplexityTier_STANDARD(t *testing.T) {
	// AC-2.2-T02, AC-2.3-T02: STANDARD tier (middle branch)
	fixturesDir := setupTestFixtures(t)
	renderer, err := NewOrchestratorRenderer(fixturesDir)
	require.NoError(t, err)

	vars := map[string]string{"complexity_tier": "STANDARD"}

	result, err := renderer.Render("valid/tier.tmpl", vars)

	require.NoError(t, err)
	assert.Equal(t, "Focused instructions", result)
}

func TestRender_ComplexityTier_COMPLEX(t *testing.T) {
	// AC-2.3-T03: COMPLEX tier (else branch)
	fixturesDir := setupTestFixtures(t)
	renderer, err := NewOrchestratorRenderer(fixturesDir)
	require.NoError(t, err)

	vars := map[string]string{"complexity_tier": "COMPLEX"}

	result, err := renderer.Render("valid/tier.tmpl", vars)

	require.NoError(t, err)
	assert.Equal(t, "Comprehensive instructions", result)
}

func TestRender_ComplexityTier_Empty(t *testing.T) {
	// AC-2.3-T04: Empty tier falls to else
	fixturesDir := setupTestFixtures(t)
	renderer, err := NewOrchestratorRenderer(fixturesDir)
	require.NoError(t, err)

	vars := map[string]string{"complexity_tier": ""}

	result, err := renderer.Render("valid/tier.tmpl", vars)

	require.NoError(t, err)
	assert.Equal(t, "Comprehensive instructions", result)
}

// ============================================================================
// Partial Template Tests (AC-3.2)
// ============================================================================

func TestRender_IncludePartial(t *testing.T) {
	// AC-3.2-T01: Include partial by name
	fixturesDir := setupTestFixtures(t)
	renderer, err := NewOrchestratorRenderer(fixturesDir)
	require.NoError(t, err)

	vars := map[string]string{}

	result, err := renderer.Render("valid/with_partial.tmpl", vars)

	require.NoError(t, err)
	assert.Contains(t, result, "Main template content.")
	assert.Contains(t, result, "Partial content here.")
	assert.Contains(t, result, "End of template.")
}

func TestRender_PartialWithContext(t *testing.T) {
	// AC-3.2-T02: Pass context to partial
	fixturesDir := setupTestFixtures(t)

	// Create a partial that uses task_id
	partialContent := `{{define "_task_partial"}}Task ID: {{.task_id}}{{end}}`
	require.NoError(t, os.WriteFile(
		filepath.Join(fixturesDir, "partials", "_task_partial.tmpl"),
		[]byte(partialContent),
		0644,
	))

	// Create template that uses the partial
	templateContent := `{{template "_task_partial" .}}`
	require.NoError(t, os.WriteFile(
		filepath.Join(fixturesDir, "valid", "use_task_partial.tmpl"),
		[]byte(templateContent),
		0644,
	))

	renderer, err := NewOrchestratorRenderer(fixturesDir)
	require.NoError(t, err)

	vars := map[string]string{"task_id": "E07-F30-001"}

	result, err := renderer.Render("valid/use_task_partial.tmpl", vars)

	require.NoError(t, err)
	assert.Equal(t, "Task ID: E07-F30-001", result)
}

// ============================================================================
// Performance Benchmark (API-12)
// ============================================================================

func BenchmarkOrchestratorRenderer_Render(b *testing.B) {
	// API-12: Render performance benchmark
	// Create fixtures in a temp directory
	tmpDir := b.TempDir()
	fixturesDir := filepath.Join(tmpDir, "fixtures")
	os.MkdirAll(filepath.Join(fixturesDir, "valid"), 0755)

	templateContent := `Task: {{.task_id}}
Title: {{.title}}
Status: {{.status}}
{{if .related_docs}}Related: {{.related_docs}}{{end}}`
	os.WriteFile(
		filepath.Join(fixturesDir, "valid", "benchmark.tmpl"),
		[]byte(templateContent),
		0644,
	)

	renderer, _ := NewOrchestratorRenderer(fixturesDir)
	vars := map[string]string{
		"task_id":      "E07-F30-001",
		"title":        "Test Task",
		"status":       "in_progress",
		"related_docs": "prd.md, arch.md",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = renderer.Render("valid/benchmark.tmpl", vars)
	}
}

// ============================================================================
// Test Helpers
// ============================================================================

// resetSingleton resets the singleton for testing
func resetSingleton() {
	engineOnce = sync.Once{}
	engineInstance = nil
	engineError = nil
	testTemplateDir = ""
}

// setTestTemplateDir sets the template directory for testing
func setTestTemplateDir(dir string) {
	testTemplateDir = dir
}
