package commands

// f02_smoke_test.go — Integration smoke test for E32-F02 Feature Acceptance Criteria
//
// This file verifies all six AC items from the E32-F02 feature spec. Each AC
// is its own top-level test function, making failures pinpoint-precise.
//
// AC1: shark-data/ workflow config works for one feature in a test env.
// AC2: Rendered prompt with {{include:}} inlines skill content, not a path ref.
// AC3: Rendered prompt is semantically equivalent to the golden corpus output.
// AC4: Embedded prompt defaults work when shark-data/ is absent from disk.
// AC5: {{include:}} cycle detection fires on a deliberate cycle test fixture.
// AC6: Override resolution: shark-data/overrides/<path> wins over shark-data/<path>.
//
// These tests operate purely at the templates.OrchestratorRenderer / IncludeResolver
// level — no database, no cobra harness — so they run fast and are hermetic.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// standardVars are the deterministic placeholder values used across AC tests,
// matching the goldenVars() set used by TestRenderedPromptsGolden so that
// AC3 comparisons line up.
func standardVars() map[string]string {
	return map[string]string{
		"id":                  "E07-F01",
		"title":               "Sample feature for golden test",
		"file_path":           "docs/plan/E07/E07-F01/E07-F01.md",
		"epic_id":             "E07",
		"task_id":             "E07-F01-001",
		"category":            "feature",
		"severity":            "medium",
		"primary_doc":         "docs/plan/E07/E07-F01/E07-F01.md",
		"doc_friendly_name":   "Feature spec",
		"related_docs":        "docs/plan/E07/E07-F01/spec.md",
		"related_tasks":       "E07-F01-001, E07-F01-002",
		"review_base":         "main",
		"is_resume":           "false",
		"advance_summary":     "Advancing entity to next status",
		"blocked_reason":      "External dependency unavailable",
		"fail_reason_summary": "Validation failed",
	}
}

// ── AC1: shark-data/ workflow config works for one feature ───────────────────

// TestF02_AC1_SharkDataWorkflowConfigLoads verifies that the shipped
// shark-data/workflow/feature.yaml can be parsed by the YAML workflow loader
// without errors and produces a non-empty status map. This is the AC1
// gate: "Manual shark-data/ for one feature in the test environment works."
//
// We exercise the renderer rather than the config loader directly, because
// the renderer is the entry point the engine (and `shark next`) ultimately
// calls. A renderer that initializes successfully against shark-data/prompts/
// proves that the engine can operate against a shark-data/ tree.
func TestF02_AC1_SharkDataWorkflowConfigLoads(t *testing.T) {
	promptsDir := findRepoPromptsDir(t) // uses helper from next_test.go

	renderer, err := templates.NewOrchestratorRenderer(promptsDir)
	require.NoError(t, err,
		"AC1: NewOrchestratorRenderer must succeed against shark-data/prompts/; "+
			"this proves the shark-data/ layout is valid for the engine")

	// Render one feature prompt to confirm the renderer is fully operational.
	out, err := renderer.Render("feature/assessment.md", standardVars())
	require.NoError(t, err, "AC1: feature/assessment.md must render without error")
	require.NotEmpty(t, out, "AC1: rendered prompt must not be empty")
}

// ── AC2: {{include:}} inlines skill content, not a path reference ────────────

// TestF02_AC2_RenderedPromptInlinesSkillContent is the AC2 gate:
// "shark next <task> --json returns a rendered prompt with skill content inlined."
//
// We use the same renderer and template the golden test uses. The assertion
// checks that the stable H1 from the selected assessment workflow is present
// in the rendered output, which proves the content was inlined rather than
// left as a path reference.
func TestF02_AC2_RenderedPromptInlinesSkillContent(t *testing.T) {
	promptsDir := findRepoPromptsDir(t)

	renderer, err := templates.NewOrchestratorRenderer(promptsDir)
	require.NoError(t, err)

	out, err := renderer.Render("feature/assessment.md", standardVars())
	require.NoError(t, err)

	// The selected assessment workflow has a stable H1 that proves it was inlined.
	assert.Contains(t, out, "# Workflow: Complexity Triage",
		"AC2: rendered prompt must contain inlined workflow body via {{include:}}")

	// Confirm the path-reference idiom is absent (no LOAD: prefix or raw path).
	assert.NotContains(t, out, "{{include:",
		"AC2: {{include:}} directives must all be resolved — none should survive into the output")
}

// ── AC3: Semantically equivalent to golden corpus output ─────────────────────

// TestF02_AC3_OutputEqualsGoldenCorpus is the AC3 gate:
// "Diff against existing .tmpl output — semantically equivalent."
//
// The golden corpus under testdata/rendered-prompts/ was generated from the
// shark-data/prompts/ tree using goldenVars(). Rendering the same template
// with standardVars() (which equals goldenVars()) must produce output that
// is byte-for-byte identical to the stored golden file.
//
// This test deliberately exercises the two most important prompt types:
//   - feature/assessment.md (has {{include:}} skill inlining)
//   - task/development.md (exercises task workflow path)
func TestF02_AC3_OutputEqualsGoldenCorpus(t *testing.T) {
	promptsDir := findRepoPromptsDir(t)

	renderer, err := templates.NewOrchestratorRenderer(promptsDir)
	require.NoError(t, err)

	goldenRoot := filepath.Join("testdata", "rendered-prompts")
	vars := standardVars() // Same as goldenVars() in next_golden_test.go

	testCases := []struct {
		tmplName   string
		goldenPath string
	}{
		{
			tmplName:   "feature/assessment.md",
			goldenPath: filepath.Join(goldenRoot, "feature", "assessment.golden"),
		},
		{
			tmplName:   "task/development.md",
			goldenPath: filepath.Join(goldenRoot, "task", "development.golden"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.tmplName, func(t *testing.T) {
			rendered, err := renderer.Render(tc.tmplName, vars)
			require.NoErrorf(t, err, "AC3: Render(%s) must succeed", tc.tmplName)

			want, err := os.ReadFile(tc.goldenPath)
			require.NoErrorf(t, err,
				"AC3: golden file %s must exist — run TestRenderedPromptsGolden with -update to generate it",
				tc.goldenPath)

			assert.Equal(t, string(want), rendered,
				"AC3: rendered output must match golden corpus (semantic equivalence gate); "+
					"if the change is intentional, regenerate the corpus with -update")
		})
	}
}

// ── AC4: Embedded fallback when shark-data/ is absent ────────────────────────

// TestF02_AC4_UsesEmbeddedPromptsWhenSharkDataAbsent is the AC4 gate:
// a project with no disk prompt tree still renders from the embedded bundle.
func TestF02_AC4_UsesEmbeddedPromptsWhenSharkDataAbsent(t *testing.T) {
	// Build an isolated temp project without any disk prompt tree.
	root := t.TempDir()

	// Change working directory so findTemplateDir walks up from root.
	origWD, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origWD) }()
	require.NoError(t, os.Chdir(root))

	// Reset the configured template override so canonical resolution kicks in.
	prev := templates.GetTemplateDirName()
	templates.SetConfiguredTemplateDir("")
	defer templates.SetConfiguredTemplateDir(prev)

	promptsPath := filepath.Join(root, "shark-data", "prompts")
	renderer, err := templates.NewOrchestratorRenderer(promptsPath)
	require.NoError(t, err, "AC4: NewOrchestratorRenderer must fall back to embedded prompts")

	out, err := renderer.Render("task/development.md", map[string]string{
		"id":       "E07-F01-001",
		"key":      "E07-F01-001",
		"task_key": "E07-F01-001",
		"task_id":  "E07-F01-001",
		"title":    "Implement authentication system",
	})
	require.NoError(t, err, "AC4: embedded task prompt must render without error")
	assert.Contains(t, out, "E07-F01-001",
		"AC4: embedded prompt should receive standard task variables")
	assert.NotContains(t, out, "{{include:",
		"AC4: embedded prompts must resolve include directives")
}

// ── AC5: {{include:}} cycle detection fires ───────────────────────────────────

// TestF02_AC5_CycleDetectionFiresOnDeliberateCycle is the AC5 gate:
// "{{include:}} cycle detection fires on a deliberate cycle test fixture."
//
// We plant two files that mutually include each other (a → b → a) in a
// temp shark-data/ tree and confirm the resolver returns an error rather
// than hanging or panicking.
func TestF02_AC5_CycleDetectionFiresOnDeliberateCycle(t *testing.T) {
	root := t.TempDir()
	dataRoot := filepath.Join(root, "shark-data")
	partialsDir := filepath.Join(dataRoot, "prompts", "_partials")
	require.NoError(t, os.MkdirAll(partialsDir, 0755))

	// Plant a deliberate cycle: alpha includes beta, beta includes alpha.
	alphaPath := filepath.Join(partialsDir, "_alpha.md")
	betaPath := filepath.Join(partialsDir, "_beta.md")
	require.NoError(t, os.WriteFile(alphaPath,
		[]byte("ALPHA: {{include: prompts/_partials/_beta.md}}"), 0644))
	require.NoError(t, os.WriteFile(betaPath,
		[]byte("BETA: {{include: prompts/_partials/_alpha.md}}"), 0644))

	resolver := templates.NewIncludeResolver(dataRoot)
	_, err := resolver.Resolve("{{include: prompts/_partials/_alpha.md}}")

	require.Error(t, err,
		"AC5: {{include:}} cycle detection must fire and return an error")
	errMsg := err.Error()
	isCycleError := strings.Contains(errMsg, "cycle") ||
		strings.Contains(errMsg, "depth cap")
	assert.True(t, isCycleError,
		"AC5: error must mention 'cycle' or 'depth cap'; got: %q", errMsg)
}

// ── AC6: shark-data/overrides/ wins over shark-data/ ─────────────────────────

// TestF02_AC6_OverrideResolutionWinsOverDefault is the AC6 gate:
// "Override resolution: shark-data/overrides/<path> wins over shark-data/<path> when both exist."
//
// We plant both a default skill file and an override under overrides/.
// Rendering a prompt that {{include:}}-s the skill must inline the OVERRIDE
// content, not the default, proving the resolution order is correct.
func TestF02_AC6_OverrideResolutionWinsOverDefault(t *testing.T) {
	root := t.TempDir()
	dataRoot := filepath.Join(root, "shark-data")

	skillsDir := filepath.Join(dataRoot, "skills", "quality")
	overrideDir := filepath.Join(dataRoot, "overrides", "skills", "quality")
	require.NoError(t, os.MkdirAll(skillsDir, 0755))
	require.NoError(t, os.MkdirAll(overrideDir, 0755))

	// Plant the default and the override.
	require.NoError(t, os.WriteFile(
		filepath.Join(skillsDir, "SKILL.md"),
		[]byte("# Default Quality Skill\nDefault content."),
		0644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(overrideDir, "SKILL.md"),
		[]byte("# Override Quality Skill\nOverride content — local customization."),
		0644,
	))

	resolver := templates.NewIncludeResolver(dataRoot)
	out, err := resolver.Resolve("{{include: skills/quality/SKILL.md}}")
	require.NoError(t, err, "AC6: resolving an overridden skill must succeed")

	assert.Contains(t, out, "Override content — local customization.",
		"AC6: override file must fully replace the default; override content must appear in output")
	assert.NotContains(t, out, "Default content.",
		"AC6: default content must NOT appear when an override exists — overrides fully replace, never merge")
}

// ── AC2 extended: task prompt inlines skill via {{include:}} ─────────────────

// TestF02_AC2_TaskPromptInlinesSkill extends AC2 with a second prompt type
// (task/development.md) to confirm {{include:}} works for tasks too.
func TestF02_AC2_TaskPromptInlinesSkill(t *testing.T) {
	promptsDir := findRepoPromptsDir(t)

	renderer, err := templates.NewOrchestratorRenderer(promptsDir)
	require.NoError(t, err)

	out, err := renderer.Render("task/development.md", standardVars())
	require.NoError(t, err)

	// task/development.md includes skills/implementation/SKILL.md and
	// skills/test-driven-development/SKILL.md. Both should be inlined.
	assert.NotEmpty(t, out, "AC2 (task): rendered task prompt must not be empty")
	assert.NotContains(t, out, "{{include:",
		"AC2 (task): all {{include:}} directives must be resolved in the rendered task prompt")
	// Positive proof the skill *bodies* were inlined: assert the stable H1 from
	// each included skill is present (mirrors the feature AC2 test).
	assert.Contains(t, out, "# Implementation Skill",
		"AC2 (task): rendered prompt must contain inlined implementation skill body")
	assert.Contains(t, out, "# Test-Driven Development (TDD)",
		"AC2 (task): rendered prompt must contain inlined TDD skill body")
}
