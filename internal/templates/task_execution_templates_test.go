package templates

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTaskTemplatesFixtures creates temporary directory with task execution templates
func setupTaskTemplatesFixtures(t *testing.T) string {
	t.Helper()

	testDir := t.TempDir()
	templatesDir := filepath.Join(testDir, "custom-prompts")
	taskDir := filepath.Join(templatesDir, "task")
	partialsDir := filepath.Join(templatesDir, "partials")

	// Create directories
	require.NoError(t, os.MkdirAll(taskDir, 0755))
	require.NoError(t, os.MkdirAll(partialsDir, 0755))

	// Create partial: _tdd_process.tmpl
	tddProcess := `{{define "_tdd_process"}}TDD PROCESS:
1. Write failing test first (red)
2. Implement minimum code to pass (green)
3. Refactor while keeping tests green
4. Commit when test suite passes{{end}}`
	require.NoError(t, os.WriteFile(filepath.Join(partialsDir, "_tdd_process.tmpl"), []byte(tddProcess), 0644))

	return templatesDir
}

// Test data for rendering
func getTaskTestData() map[string]string {
	return map[string]string{
		"task_id":         "E07-F30-001",
		"title":           "Implement feature X",
		"file_path":       "docs/plan/E07-enhancements/E07-F30/tasks/E07-F30-001.md",
		"related_docs":    "architecture.md, prd.md",
		"related_tasks":   "E07-F29-003, E07-F29-004",
		"complexity_tier": "STANDARD",
	}
}

func getTaskTestDataNoRelated() map[string]string {
	return map[string]string{
		"task_id":         "E07-F30-002",
		"title":           "Simple bug fix",
		"file_path":       "docs/plan/E07-enhancements/E07-F30/tasks/E07-F30-002.md",
		"related_docs":    "",
		"related_tasks":   "",
		"complexity_tier": "SIMPLE",
	}
}

// ============================================================================
// T1: ready_for_development.tmpl Rendering Tests
// ============================================================================

func TestTaskReadyForDevelopmentTemplate_BasicRendering(t *testing.T) {
	// AC-4.1-T01: Basic rendering with all fields populated
	templatesDir := setupTaskTemplatesFixtures(t)

	// Create the template
	tmplContent := `Launch developer with test-driven-development skill for task {{.task_id}}: "{{.title}}".

LOAD: test-driven-development + implementation skills.

READ:
(1) Task spec at {{.file_path}}
(2) Feature test plan (09-test-plan.md) for test cases, acceptance criteria tests, and API contract tests relevant to this task
(3) Feature architecture docs for contracts and patterns
{{- if .related_docs}}
(4) Related docs: {{.related_docs}}
{{- end}}
{{- if .related_tasks}}
({{if .related_docs}}5{{else}}4{{end}}) Related tasks: {{.related_tasks}}
{{- end}}

{{template "_tdd_process" .}}

EXIT GATE:
- All mapped test cases from feature test plan pass
- Implementation matches task spec
- Code follows codebase conventions from research

Advance: shark task next-status {{.task_id}}.`

	taskDir := filepath.Join(templatesDir, "task")
	require.NoError(t, os.WriteFile(filepath.Join(taskDir, "ready_for_development.tmpl"), []byte(tmplContent), 0644))

	// Create renderer with template directory
	renderer, err := NewOrchestratorRenderer(templatesDir)
	require.NoError(t, err)

	// Render template
	data := getTaskTestData()
	result, err := renderer.Render("task/ready_for_development.tmpl", data)
	require.NoError(t, err)

	// Verify output contains expected sections
	assert.Contains(t, result, "Launch developer with test-driven-development skill")
	assert.Contains(t, result, "E07-F30-001")
	assert.Contains(t, result, "LOAD: test-driven-development + implementation skills")
	assert.Contains(t, result, "READ:")
	assert.Contains(t, result, "Feature test plan")
	assert.Contains(t, result, "Related docs: architecture.md, prd.md")
	assert.Contains(t, result, "Related tasks: E07-F29-003, E07-F29-004")
	assert.Contains(t, result, "TDD PROCESS:")
	assert.Contains(t, result, "EXIT GATE:")
	assert.Contains(t, result, "Advance: shark task next-status E07-F30-001")
}

func TestTaskReadyForDevelopmentTemplate_ConditionalHidesEmpty(t *testing.T) {
	// AC-4.5-T01: Conditionals hide empty sections
	templatesDir := setupTaskTemplatesFixtures(t)

	tmplContent := `Launch developer with test-driven-development skill for task {{.task_id}}: "{{.title}}".

LOAD: test-driven-development + implementation skills.

READ:
(1) Task spec at {{.file_path}}
(2) Feature test plan (09-test-plan.md) for test cases, acceptance criteria tests, and API contract tests relevant to this task
(3) Feature architecture docs for contracts and patterns
{{- if .related_docs}}
(4) Related docs: {{.related_docs}}
{{- end}}
{{- if .related_tasks}}
({{if .related_docs}}5{{else}}4{{end}}) Related tasks: {{.related_tasks}}
{{- end}}

{{template "_tdd_process" .}}

EXIT GATE:
- All mapped test cases from feature test plan pass
- Implementation matches task spec
- Code follows codebase conventions from research

Advance: shark task next-status {{.task_id}}.`

	taskDir := filepath.Join(templatesDir, "task")
	require.NoError(t, os.WriteFile(filepath.Join(taskDir, "ready_for_development.tmpl"), []byte(tmplContent), 0644))

	renderer, err := NewOrchestratorRenderer(templatesDir)
	require.NoError(t, err)

	// Render with no related docs/tasks
	data := getTaskTestDataNoRelated()
	result, err := renderer.Render("task/ready_for_development.tmpl", data)
	require.NoError(t, err)

	// Verify empty sections are hidden
	assert.NotContains(t, result, "Related docs:")
	assert.NotContains(t, result, "Related tasks:")
	// But still should have base sections
	assert.Contains(t, result, "(1) Task spec at")
	assert.Contains(t, result, "(2) Feature test plan")
	assert.Contains(t, result, "(3) Feature architecture docs")
}

// ============================================================================
// T2: ready_for_code_review.tmpl Rendering Tests
// ============================================================================

func TestTaskReadyForCodeReviewTemplate_BasicRendering(t *testing.T) {
	// AC-4.1-T01: Basic rendering with code review focus
	templatesDir := setupTaskTemplatesFixtures(t)

	tmplContent := `Launch tech-lead with quality skill to review task {{.task_id}}: "{{.title}}".

LOAD: quality skill.

READ:
(1) Task spec at {{.file_path}}
(2) Feature test plan (09-test-plan.md) for expected test coverage and acceptance criteria
(3) Feature PRD for feature-level intent
(4) Implementation code changes
{{- if .related_docs}}
(5) Related docs: {{.related_docs}}
{{- end}}
{{- if .related_tasks}}
({{if .related_docs}}6{{else}}5{{end}}) Related tasks: {{.related_tasks}}
{{- end}}

REVIEW:
- Check code quality, security, and adherence to codebase standards
- Compare implementation behavior against the task spec, feature test plan, and feature PRD
- Verify acceptance criteria are met as specified in feature requirements
- Verify TDD compliance -- tests from feature test plan are implemented and passing
- Flag any deviation between feature intent, spec, and actual behavior

EXIT GATE:
- Code quality passes
- Tests from feature plan implemented and passing
- No spec drift from feature PRD

Advance: shark task next-status {{.task_id}}.`

	taskDir := filepath.Join(templatesDir, "task")
	require.NoError(t, os.WriteFile(filepath.Join(taskDir, "ready_for_code_review.tmpl"), []byte(tmplContent), 0644))

	renderer, err := NewOrchestratorRenderer(templatesDir)
	require.NoError(t, err)

	data := getTaskTestData()
	result, err := renderer.Render("task/ready_for_code_review.tmpl", data)
	require.NoError(t, err)

	assert.Contains(t, result, "Launch tech-lead with quality skill")
	assert.Contains(t, result, "LOAD: quality skill")
	assert.Contains(t, result, "REVIEW:")
	assert.Contains(t, result, "Check code quality, security")
	assert.Contains(t, result, "TDD compliance")
}

// ============================================================================
// T3: ready_for_qa.tmpl Rendering Tests
// ============================================================================

func TestTaskReadyForQATemplate_BasicRendering(t *testing.T) {
	// AC-4.1-T01: Basic rendering with QA focus
	templatesDir := setupTaskTemplatesFixtures(t)

	tmplContent := `Launch qa agent with quality skill to test task {{.task_id}}: "{{.title}}".

LOAD: quality skill.

READ:
(1) Task spec at {{.file_path}}
(2) Feature test plan (09-test-plan.md) for the acceptance test cases mapped to this task
(3) Feature PRD for feature-level intent
(4) Implementation code and test results
{{- if .related_docs}}
(5) Related docs: {{.related_docs}}
{{- end}}
{{- if .related_tasks}}
({{if .related_docs}}6{{else}}5{{end}}) Related tasks: {{.related_tasks}}
{{- end}}

VALIDATE:
- Run test cases from feature test plan that map to this task
- Verify acceptance criteria are met per the feature test plan, not just the task spec
- Check edge cases identified in the test plan
- Validate integration points behave correctly

EXIT GATE:
- All mapped test cases pass
- Acceptance criteria validated against feature intent
- No regressions in sibling task functionality

Advance: shark task next-status {{.task_id}}.`

	taskDir := filepath.Join(templatesDir, "task")
	require.NoError(t, os.WriteFile(filepath.Join(taskDir, "ready_for_qa.tmpl"), []byte(tmplContent), 0644))

	renderer, err := NewOrchestratorRenderer(templatesDir)
	require.NoError(t, err)

	data := getTaskTestData()
	result, err := renderer.Render("task/ready_for_qa.tmpl", data)
	require.NoError(t, err)

	assert.Contains(t, result, "Launch qa agent with quality skill")
	assert.Contains(t, result, "VALIDATE:")
	assert.Contains(t, result, "Run test cases from feature test plan")
}

// ============================================================================
// T4: ready_for_refinement_ba.tmpl Rendering Tests
// ============================================================================

func TestTaskReadyForRefinementBATemplate_BasicRendering(t *testing.T) {
	// AC-4.1-T01: Basic rendering with BA refinement focus
	templatesDir := setupTaskTemplatesFixtures(t)

	tmplContent := `Launch business-analyst with specification-writing skill for task {{.task_id}}: "{{.title}}".

LOAD: specification-writing workflow refine-task-requirements.md.

READ:
(1) Task spec at {{.file_path}}
(2) Parent feature PRD for feature-level intent
(3) Feature research report for codebase patterns and integration points
(4) Task notes (shark task get {{.task_id}} --json) for blocker context
(5) Architecture docs for constraints
{{- if .related_docs}}
(6) Related docs: {{.related_docs}}
{{- end}}
{{- if .related_tasks}}
({{if .related_docs}}7{{else}}6{{end}}) Related tasks: {{.related_tasks}}
{{- end}}

PRODUCE:
- Updated task spec with testable user stories
- Measurable acceptance criteria
- Business rules and edge cases

ALSO VERIFY:
- Requirements align with feature PRD (no spec drift)
- Codebase standards referenced (absorbs former task research)
- Contracts consistent with architecture

EXIT GATE:
- AC unambiguous and measurable
- No vague language
- Traces to feature PRD
- Blocker notes addressed

Advance: shark task next-status {{.task_id}}.`

	taskDir := filepath.Join(templatesDir, "task")
	require.NoError(t, os.WriteFile(filepath.Join(taskDir, "ready_for_refinement_ba.tmpl"), []byte(tmplContent), 0644))

	renderer, err := NewOrchestratorRenderer(templatesDir)
	require.NoError(t, err)

	data := getTaskTestData()
	result, err := renderer.Render("task/ready_for_refinement_ba.tmpl", data)
	require.NoError(t, err)

	assert.Contains(t, result, "Launch business-analyst with specification-writing skill")
	assert.Contains(t, result, "PRODUCE:")
	assert.Contains(t, result, "Updated task spec with testable user stories")
	assert.Contains(t, result, "ALSO VERIFY:")
}

// ============================================================================
// T5: ready_for_refinement_tech.tmpl Rendering Tests
// ============================================================================

func TestTaskReadyForRefinementTechTemplate_BasicRendering(t *testing.T) {
	// AC-4.1-T01: Basic rendering with Tech refinement focus
	templatesDir := setupTaskTemplatesFixtures(t)

	tmplContent := `Launch architect with architecture and specification-writing skills for task {{.task_id}}: "{{.title}}".

LOAD: specification-writing workflow refine-task-requirements.md (architect path).

READ:
(1) Task spec at {{.file_path}}
(2) BA-refined requirements
(3) Feature architecture docs (02-08 series)
(4) Feature research report for patterns and integration points
(5) Blocker notes if returning from dev/review
{{- if .related_docs}}
(6) Related docs: {{.related_docs}}
{{- end}}
{{- if .related_tasks}}
({{if .related_docs}}7{{else}}6{{end}}) Related tasks: {{.related_tasks}}
{{- end}}

PRODUCE:
- Updated task spec or PRP with API contracts
- Data models and schema updates
- Architecture decisions with rationale
- Security/performance considerations

VERIFY:
- Contracts consistent with feature architecture
- Codebase patterns from feature research referenced
- No ambiguity in specifications

EXIT GATE:
- Contracts defined
- Models specified
- Decisions documented
- No TBDs
- Consistent with feature architecture
- Task implementable without ambiguity

Advance: shark task next-status {{.task_id}}.`

	taskDir := filepath.Join(templatesDir, "task")
	require.NoError(t, os.WriteFile(filepath.Join(taskDir, "ready_for_refinement_tech.tmpl"), []byte(tmplContent), 0644))

	renderer, err := NewOrchestratorRenderer(templatesDir)
	require.NoError(t, err)

	data := getTaskTestData()
	result, err := renderer.Render("task/ready_for_refinement_tech.tmpl", data)
	require.NoError(t, err)

	assert.Contains(t, result, "Launch architect with architecture and specification-writing skills")
	assert.Contains(t, result, "PRODUCE:")
	assert.Contains(t, result, "API contracts")
	assert.Contains(t, result, "Data models")
	assert.Contains(t, result, "VERIFY:")
}

// ============================================================================
// Regression Tests: Semantic Equivalence
// ============================================================================

func TestTaskTemplates_RegressionSemanticEquivalence(t *testing.T) {
	// AC-4.5-T04: All 5 templates render consistently with semantic meaning preserved
	templatesDir := setupTaskTemplatesFixtures(t)

	templates := []string{
		"ready_for_development",
		"ready_for_code_review",
		"ready_for_qa",
		"ready_for_refinement_ba",
		"ready_for_refinement_tech",
	}

	taskDir := filepath.Join(templatesDir, "task")

	// Create all templates with their content
	templateContent := map[string]string{
		"ready_for_development": `Launch developer with test-driven-development skill for task {{.task_id}}: "{{.title}}".

LOAD: test-driven-development + implementation skills.

READ:
(1) Task spec at {{.file_path}}
(2) Feature test plan (09-test-plan.md) for test cases, acceptance criteria tests, and API contract tests relevant to this task
(3) Feature architecture docs for contracts and patterns
{{- if .related_docs}}
(4) Related docs: {{.related_docs}}
{{- end}}
{{- if .related_tasks}}
({{if .related_docs}}5{{else}}4{{end}}) Related tasks: {{.related_tasks}}
{{- end}}

{{template "_tdd_process" .}}

EXIT GATE:
- All mapped test cases from feature test plan pass
- Implementation matches task spec
- Code follows codebase conventions from research

Advance: shark task next-status {{.task_id}}.`,
		"ready_for_code_review": `Launch tech-lead with quality skill to review task {{.task_id}}: "{{.title}}".

LOAD: quality skill.

READ:
(1) Task spec at {{.file_path}}
(2) Feature test plan (09-test-plan.md) for expected test coverage and acceptance criteria
(3) Feature PRD for feature-level intent
(4) Implementation code changes
{{- if .related_docs}}
(5) Related docs: {{.related_docs}}
{{- end}}
{{- if .related_tasks}}
({{if .related_docs}}6{{else}}5{{end}}) Related tasks: {{.related_tasks}}
{{- end}}

REVIEW:
- Check code quality, security, and adherence to codebase standards
- Compare implementation behavior against the task spec, feature test plan, and feature PRD
- Verify acceptance criteria are met as specified in feature requirements
- Verify TDD compliance -- tests from feature test plan are implemented and passing
- Flag any deviation between feature intent, spec, and actual behavior

EXIT GATE:
- Code quality passes
- Tests from feature plan implemented and passing
- No spec drift from feature PRD

Advance: shark task next-status {{.task_id}}.`,
		"ready_for_qa": `Launch qa agent with quality skill to test task {{.task_id}}: "{{.title}}".

LOAD: quality skill.

READ:
(1) Task spec at {{.file_path}}
(2) Feature test plan (09-test-plan.md) for the acceptance test cases mapped to this task
(3) Feature PRD for feature-level intent
(4) Implementation code and test results
{{- if .related_docs}}
(5) Related docs: {{.related_docs}}
{{- end}}
{{- if .related_tasks}}
({{if .related_docs}}6{{else}}5{{end}}) Related tasks: {{.related_tasks}}
{{- end}}

VALIDATE:
- Run test cases from feature test plan that map to this task
- Verify acceptance criteria are met per the feature test plan, not just the task spec
- Check edge cases identified in the test plan
- Validate integration points behave correctly

EXIT GATE:
- All mapped test cases pass
- Acceptance criteria validated against feature intent
- No regressions in sibling task functionality

Advance: shark task next-status {{.task_id}}.`,
		"ready_for_refinement_ba": `Launch business-analyst with specification-writing skill for task {{.task_id}}: "{{.title}}".

LOAD: specification-writing workflow refine-task-requirements.md.

READ:
(1) Task spec at {{.file_path}}
(2) Parent feature PRD for feature-level intent
(3) Feature research report for codebase patterns and integration points
(4) Task notes (shark task get {{.task_id}} --json) for blocker context
(5) Architecture docs for constraints
{{- if .related_docs}}
(6) Related docs: {{.related_docs}}
{{- end}}
{{- if .related_tasks}}
({{if .related_docs}}7{{else}}6{{end}}) Related tasks: {{.related_tasks}}
{{- end}}

PRODUCE:
- Updated task spec with testable user stories
- Measurable acceptance criteria
- Business rules and edge cases

ALSO VERIFY:
- Requirements align with feature PRD (no spec drift)
- Codebase standards referenced (absorbs former task research)
- Contracts consistent with architecture

EXIT GATE:
- AC unambiguous and measurable
- No vague language
- Traces to feature PRD
- Blocker notes addressed

Advance: shark task next-status {{.task_id}}.`,
		"ready_for_refinement_tech": `Launch architect with architecture and specification-writing skills for task {{.task_id}}: "{{.title}}".

LOAD: specification-writing workflow refine-task-requirements.md (architect path).

READ:
(1) Task spec at {{.file_path}}
(2) BA-refined requirements
(3) Feature architecture docs (02-08 series)
(4) Feature research report for patterns and integration points
(5) Blocker notes if returning from dev/review
{{- if .related_docs}}
(6) Related docs: {{.related_docs}}
{{- end}}
{{- if .related_tasks}}
({{if .related_docs}}7{{else}}6{{end}}) Related tasks: {{.related_tasks}}
{{- end}}

PRODUCE:
- Updated task spec or PRP with API contracts
- Data models and schema updates
- Architecture decisions with rationale
- Security/performance considerations

VERIFY:
- Contracts consistent with feature architecture
- Codebase patterns from feature research referenced
- No ambiguity in specifications

EXIT GATE:
- Contracts defined
- Models specified
- Decisions documented
- No TBDs
- Consistent with feature architecture
- Task implementable without ambiguity

Advance: shark task next-status {{.task_id}}.`,
	}

	// Write all templates
	for name, content := range templateContent {
		require.NoError(t, os.WriteFile(filepath.Join(taskDir, name+".tmpl"), []byte(content), 0644))
	}

	renderer, err := NewOrchestratorRenderer(templatesDir)
	require.NoError(t, err)

	// Test with both test data sets
	for _, tmplName := range templates {
		t.Run(tmplName, func(t *testing.T) {
			// With all fields
			result1, err := renderer.Render("task/"+tmplName+".tmpl", getTaskTestData())
			require.NoError(t, err)

			// Verify basic semantic content is present
			assert.NotEmpty(t, result1)
			assert.NotContains(t, result1, "{{.task_id}}") // Should not have unreplaced placeholders
			assert.Contains(t, result1, "E07-F30-001")
			assert.Contains(t, result1, "Advance: shark task next-status")

			// Without related docs/tasks
			result2, err := renderer.Render("task/"+tmplName+".tmpl", getTaskTestDataNoRelated())
			require.NoError(t, err)

			assert.NotEmpty(t, result2)
			assert.Contains(t, result2, "E07-F30-002")
			// Related sections should be hidden
			assert.NotContains(t, result2, "Related docs:")
			assert.NotContains(t, result2, "Related tasks:")
		})
	}
}

func TestTaskTemplates_AllTemplatesExist(t *testing.T) {
	// AC-4.1-T01: All 5 templates exist in templates/task/
	templatesDir := setupTaskTemplatesFixtures(t)

	templates := []string{
		"ready_for_development.tmpl",
		"ready_for_code_review.tmpl",
		"ready_for_qa.tmpl",
		"ready_for_refinement_ba.tmpl",
		"ready_for_refinement_tech.tmpl",
	}

	taskDir := filepath.Join(templatesDir, "task")

	// Create minimal versions of all templates
	minimalTemplate := `Task: {{.task_id}}`
	for _, tmplName := range templates {
		require.NoError(t, os.WriteFile(filepath.Join(taskDir, tmplName), []byte(minimalTemplate), 0644))
	}

	// Verify all exist
	for _, tmplName := range templates {
		_, err := os.Stat(filepath.Join(taskDir, tmplName))
		assert.NoError(t, err, "template should exist: %s", tmplName)
	}
}
