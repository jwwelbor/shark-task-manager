package templates

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"text/template"

	"github.com/jwwelbor/shark-task-manager/internal/sharkdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// E1 — {{include:}} directive tests
// ============================================================================

// setupSharkDataRoot lays down a temp shark-data tree and returns its path.
// Layout:
//
//	<root>/shark-data/
//	  skills/<files>
//	  prompts/<files>
//	  overrides/skills/<files>
func setupSharkDataRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	dataRoot := filepath.Join(root, "shark-data")
	require.NoError(t, os.MkdirAll(filepath.Join(dataRoot, "skills", "quality", "workflows"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(dataRoot, "skills", "tdd"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(dataRoot, "prompts", "task"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(dataRoot, "prompts", "_partials"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(dataRoot, "overrides", "skills", "quality", "workflows"), 0755))
	return dataRoot
}

func TestIncludeResolver_LegacyModeNoOp(t *testing.T) {
	r := NewIncludeResolver("")
	src := "Body before. {{include: skills/quality/SKILL.md}} Body after."
	got, err := r.Resolve(src)
	require.NoError(t, err)
	assert.Equal(t, src, got, "legacy mode (empty data root) should leave include directives intact")
}

func TestIncludeResolver_BasicInclude(t *testing.T) {
	dataRoot := setupSharkDataRoot(t)
	require.NoError(t, os.WriteFile(
		filepath.Join(dataRoot, "skills", "tdd", "SKILL.md"),
		[]byte("TDD: Write a failing test first."),
		0644,
	))

	r := NewIncludeResolver(dataRoot)
	got, err := r.Resolve("Pre. {{include: skills/tdd/SKILL.md}} Post.")
	require.NoError(t, err)
	assert.Equal(t, "Pre. TDD: Write a failing test first. Post.", got)
}

func TestIncludeResolver_OverrideWinsOverDefault(t *testing.T) {
	dataRoot := setupSharkDataRoot(t)
	require.NoError(t, os.WriteFile(
		filepath.Join(dataRoot, "skills", "quality", "workflows", "review-code.md"),
		[]byte("DEFAULT review-code"),
		0644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dataRoot, "overrides", "skills", "quality", "workflows", "review-code.md"),
		[]byte("OVERRIDE review-code"),
		0644,
	))

	r := NewIncludeResolver(dataRoot)
	got, err := r.Resolve("{{include: skills/quality/workflows/review-code.md}}")
	require.NoError(t, err)
	assert.Equal(t, "OVERRIDE review-code", got, "override under overrides/ must fully replace the default")
}

func TestIncludeResolver_NoOverrideFallsBackToDefault(t *testing.T) {
	// When overrides/<path> does NOT exist, resolvePath must return the default
	// path content. This is the complement of TestIncludeResolver_OverrideWinsOverDefault
	// and pins the no-override-fallback contract (ADR-3 §2).
	dataRoot := setupSharkDataRoot(t)
	require.NoError(t, os.WriteFile(
		filepath.Join(dataRoot, "skills", "quality", "workflows", "review-code.md"),
		[]byte("DEFAULT review-code"),
		0644,
	))
	// No override file placed under overrides/ — the default must be returned.

	r := NewIncludeResolver(dataRoot)
	got, err := r.Resolve("{{include: skills/quality/workflows/review-code.md}}")
	require.NoError(t, err)
	assert.Equal(t, "DEFAULT review-code", got, "when no override exists, default path content must be returned")
}

func TestIncludeResolver_AugmentSameAsIncludeForNow(t *testing.T) {
	// {{augment:}} accepts the same path syntax. Per F2 design, the divergent
	// semantics of augment vs include are TBD; for now both inline the file.
	dataRoot := setupSharkDataRoot(t)
	require.NoError(t, os.WriteFile(
		filepath.Join(dataRoot, "skills", "tdd", "SKILL.md"),
		[]byte("TDD body"),
		0644,
	))

	r := NewIncludeResolver(dataRoot)
	got, err := r.Resolve("{{augment: skills/tdd/SKILL.md}}")
	require.NoError(t, err)
	assert.Equal(t, "TDD body", got)
}

func TestIncludeResolver_NestedInclude(t *testing.T) {
	dataRoot := setupSharkDataRoot(t)
	require.NoError(t, os.WriteFile(
		filepath.Join(dataRoot, "prompts", "_partials", "_inner.md"),
		[]byte("INNER"),
		0644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dataRoot, "prompts", "_partials", "_outer.md"),
		[]byte("OUTER[ {{include: prompts/_partials/_inner.md}} ]"),
		0644,
	))

	r := NewIncludeResolver(dataRoot)
	got, err := r.Resolve("Top: {{include: prompts/_partials/_outer.md}}")
	require.NoError(t, err)
	assert.Equal(t, "Top: OUTER[ INNER ]", got)
}

func TestIncludeResolver_MdIncludeStripsFrontmatter(t *testing.T) {
	dataRoot := setupSharkDataRoot(t)
	require.NoError(t, os.WriteFile(
		filepath.Join(dataRoot, "skills", "tdd", "SKILL.md"),
		[]byte("---\ninputs:\n  - x\n---\nTDD body"),
		0644,
	))

	r := NewIncludeResolver(dataRoot)
	got, err := r.Resolve("{{include: skills/tdd/SKILL.md}}")
	require.NoError(t, err)
	assert.Equal(t, "TDD body", got, "frontmatter on .md includes must be stripped")
}

func TestIncludeResolver_DepthCapEnforced(t *testing.T) {
	dataRoot := setupSharkDataRoot(t)
	// Build a chain a -> b -> c -> d -> e -> f (depth 6 > IncludeDepthCap=5).
	for i, name := range []string{"a", "b", "c", "d", "e"} {
		next := []string{"b", "c", "d", "e", "f"}[i]
		body := "{{include: prompts/_partials/_" + next + ".md}}"
		require.NoError(t, os.WriteFile(
			filepath.Join(dataRoot, "prompts", "_partials", "_"+name+".md"),
			[]byte(body),
			0644,
		))
	}
	require.NoError(t, os.WriteFile(
		filepath.Join(dataRoot, "prompts", "_partials", "_f.md"),
		[]byte("LEAF"),
		0644,
	))

	r := NewIncludeResolver(dataRoot)
	_, err := r.Resolve("{{include: prompts/_partials/_a.md}}")
	require.Error(t, err)
	assert.True(t,
		strings.Contains(err.Error(), "depth cap"),
		"deep include chain should hit the depth cap; got: %v", err,
	)
}

func TestIncludeResolver_CycleDetected(t *testing.T) {
	dataRoot := setupSharkDataRoot(t)
	// a -> b -> a forms a cycle.
	require.NoError(t, os.WriteFile(
		filepath.Join(dataRoot, "prompts", "_partials", "_a.md"),
		[]byte("A: {{include: prompts/_partials/_b.md}}"),
		0644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dataRoot, "prompts", "_partials", "_b.md"),
		[]byte("B: {{include: prompts/_partials/_a.md}}"),
		0644,
	))

	r := NewIncludeResolver(dataRoot)
	_, err := r.Resolve("{{include: prompts/_partials/_a.md}}")
	require.Error(t, err)
	assert.True(t,
		strings.Contains(err.Error(), "cycle") || strings.Contains(err.Error(), "depth cap"),
		"cycle should be detected explicitly or via depth cap; got: %v", err,
	)
}

func TestIncludeResolver_SiblingsCanIncludeSamePartial(t *testing.T) {
	// Two non-cyclic siblings include the same partial — must succeed.
	dataRoot := setupSharkDataRoot(t)
	require.NoError(t, os.WriteFile(
		filepath.Join(dataRoot, "prompts", "_partials", "_shared.md"),
		[]byte("S"),
		0644,
	))

	r := NewIncludeResolver(dataRoot)
	got, err := r.Resolve("{{include: prompts/_partials/_shared.md}}-{{include: prompts/_partials/_shared.md}}")
	require.NoError(t, err)
	assert.Equal(t, "S-S", got)
}

func TestIncludeResolver_MissingFileFailsWithLocations(t *testing.T) {
	dataRoot := setupSharkDataRoot(t)
	r := NewIncludeResolver(dataRoot)
	_, err := r.Resolve("{{include: skills/missing/whatever.md}}")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "skills/missing/whatever.md")
	assert.Contains(t, err.Error(), "overrides", "error message should mention both lookup locations")
}

func TestIncludeResolver_RejectsAbsolutePath(t *testing.T) {
	r := NewIncludeResolver(t.TempDir())
	_, err := r.Resolve("{{include: /etc/passwd}}")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be relative")
}

func TestIncludeResolver_RejectsParentTraversal(t *testing.T) {
	r := NewIncludeResolver(t.TempDir())
	_, err := r.Resolve("{{include: ../../../etc/passwd}}")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "..")
}

func TestIncludeResolver_NoDirectivesIsNoop(t *testing.T) {
	r := NewIncludeResolver(t.TempDir())
	body := "no includes here, just {{.task_id}} and {{template \"_x\" .}}"
	got, err := r.Resolve(body)
	require.NoError(t, err)
	assert.Equal(t, body, got, "Go-template directives like {{.foo}} or {{template ...}} must pass through untouched")
}

func TestIncludeResolver_SizeWarningFires(t *testing.T) {
	dataRoot := setupSharkDataRoot(t)
	big := strings.Repeat("x", IncludeSizeWarnBytes+10)
	require.NoError(t, os.WriteFile(
		filepath.Join(dataRoot, "skills", "tdd", "SKILL.md"),
		[]byte(big),
		0644,
	))

	r := NewIncludeResolver(dataRoot)
	var warned bool
	r.warnFn = func(path string, size int) { warned = true }

	_, err := r.Resolve("{{include: skills/tdd/SKILL.md}}")
	require.NoError(t, err)
	assert.True(t, warned, "size warning should fire when included file exceeds threshold")
}

func TestIncludeResolver_FirstErrStopsSubsequentMatches(t *testing.T) {
	// When two directives appear in the same content and the first fails, the
	// error from the first must propagate and the second must not mask it.
	dataRoot := setupSharkDataRoot(t)
	require.NoError(t, os.WriteFile(
		filepath.Join(dataRoot, "skills", "tdd", "SKILL.md"),
		[]byte("TDD body"),
		0644,
	))

	r := NewIncludeResolver(dataRoot)
	// First include is missing; second include exists. Error from the first
	// must win — this exercises the `if firstErr != nil { return match }` guard.
	_, err := r.Resolve("{{include: skills/missing/gone.md}} {{include: skills/tdd/SKILL.md}}")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "skills/missing/gone.md", "error should identify the failing include")
}

func TestIncludeResolver_ReadFileFails(t *testing.T) {
	// Verify the os.ReadFile error path (lines 121-124) by making the resolved
	// path a directory — os.Stat succeeds but os.ReadFile fails on a directory.
	dataRoot := t.TempDir()
	// Create a directory where a file is expected. The override lookup will find
	// no override; the default lookup resolves to a directory that exists.
	dirPath := filepath.Join(dataRoot, "skills")
	require.NoError(t, os.MkdirAll(dirPath, 0755))

	r := NewIncludeResolver(dataRoot)
	// "skills" is a directory — stat will succeed, ReadFile will fail.
	_, err := r.Resolve("{{include: skills}}")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read include", "read error must surface with context")
}

// ============================================================================
// E1 — integration with renderer (end-to-end via NewOrchestratorRenderer)
// ============================================================================

func TestRenderer_IncludeResolvedAtParseTime(t *testing.T) {
	dataRoot := setupSharkDataRoot(t)

	require.NoError(t, os.WriteFile(
		filepath.Join(dataRoot, "skills", "quality", "workflows", "qa-testing.md"),
		[]byte("---\ninputs:\n  - x\n---\nQA craft body"),
		0644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dataRoot, "prompts", "task", "in_qa.md"),
		[]byte("PROMPT[ {{include: skills/quality/workflows/qa-testing.md}} ]-{{.task_id}}"),
		0644,
	))

	promptsDir := filepath.Join(dataRoot, "prompts")
	renderer, err := NewOrchestratorRenderer(promptsDir)
	require.NoError(t, err)

	out, err := renderer.Render("task/in_qa.md", map[string]string{"task_id": "T1"})
	require.NoError(t, err)
	assert.Equal(t, "PROMPT[ QA craft body ]-T1", out)
}

func TestRenderer_NonBundleTmplPathHasNoIncludeRoot(t *testing.T) {
	// A non-bundle prompt directory has no detectable data root. Embed-aware
	// include resolution still works for bundled files; this test pins the
	// IncludeRoot contract for custom .tmpl paths.
	root := t.TempDir() // dir base is NOT "prompts"
	require.NoError(t, os.MkdirAll(filepath.Join(root, "task"), 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(root, "task", "x.tmpl"),
		[]byte("LEGACY: {{.task_id}}"),
		0644,
	))

	renderer, err := NewOrchestratorRenderer(root)
	require.NoError(t, err)
	assert.Equal(t, "", renderer.includeRoot, "legacy templateDir should have empty includeRoot")

	out, err := renderer.Render("task/x.tmpl", map[string]string{"task_id": "T1"})
	require.NoError(t, err)
	assert.Equal(t, "LEGACY: T1", out)
}

// ============================================================================
// CC-039 — Hybrid embed/disk resolution
// ============================================================================

// TestIncludeResolverWithEmbed_FallsBackToEmbed verifies that when the include
// file is absent from disk, the embed-aware resolver reads from the embedded FS.
func TestIncludeResolverWithEmbed_FallsBackToEmbed(t *testing.T) {
	// Use an empty data root — no files on disk.
	dataRoot := t.TempDir()
	r := NewIncludeResolverWithEmbed(dataRoot)

	// "prompts/_partials/_commands.md" is a real file in the embedded FS.
	// We verify resolution succeeds and produces non-empty content.
	out, err := r.Resolve("{{include: prompts/_partials/_commands.md}}")
	require.NoError(t, err, "embed fallback must resolve a known embedded file")
	assert.NotEqual(t, "{{include: prompts/_partials/_commands.md}}", out,
		"resolved output must differ from the original directive")
	assert.NotEmpty(t, strings.TrimSpace(out), "embedded file content must be non-empty")
}

// TestIncludeResolverWithEmbed_DiskWinsOverEmbed verifies that a disk file
// takes precedence over the embedded canonical when both exist.
func TestIncludeResolverWithEmbed_DiskWinsOverEmbed(t *testing.T) {
	dataRoot := t.TempDir()

	// Create a local disk file that shadows an embedded path.
	partialDir := filepath.Join(dataRoot, "prompts", "_partials")
	require.NoError(t, os.MkdirAll(partialDir, 0755))
	localContent := "LOCAL DISK CONTENT"
	require.NoError(t, os.WriteFile(
		filepath.Join(partialDir, "_commands.md"),
		[]byte(localContent),
		0644,
	))

	r := NewIncludeResolverWithEmbed(dataRoot)
	out, err := r.Resolve("{{include: prompts/_partials/_commands.md}}")
	require.NoError(t, err)
	assert.Equal(t, localContent, out,
		"disk file must win over embedded canonical")
}

// TestIncludeResolverWithEmbed_MissingFromBoth verifies that when a file is
// absent from both disk and embed, an error is returned.
func TestIncludeResolverWithEmbed_MissingFromBoth(t *testing.T) {
	dataRoot := t.TempDir()
	r := NewIncludeResolverWithEmbed(dataRoot)

	_, err := r.Resolve("{{include: prompts/nonexistent/definitely-not-there.md}}")
	require.Error(t, err, "missing from both disk and embed must return an error")
	assert.Contains(t, err.Error(), "nonexistent/definitely-not-there.md")
}

// TestIncludeResolverWithEmbed_DefectClassSweepRenders verifies that the
// checked-in defect-class-sweep workflow file (E34-F06) renders cleanly
// through the production renderer and carries every section named in
// spec.md REQ-F-001: class naming, search-scope declaration, enumeration,
// zero-result reporting, instance evidence, guard selection, closure rule,
// and the three-part re-verification procedure.
func TestIncludeResolverWithEmbed_DefectClassSweepRenders(t *testing.T) {
	r := NewIncludeResolverWithEmbed("")

	out, err := r.Resolve("{{include: skills/quality/workflows/defect-class-sweep.md}}")
	require.NoError(t, err, "defect-class-sweep.md must render through the production renderer with no errors")

	for _, section := range []string{
		"## Class naming",
		"## Search-scope declaration",
		"## Enumeration procedure",
		"## Zero-result reporting",
		"## Instance evidence",
		"## Guard selection",
		"## Structural guard closure",
		"## Full-class re-verification",
	} {
		assert.Contains(t, out, section, "defect-class-sweep.md must contain section %q", section)
	}
}

// TestProductCriticalPathGuardPartial_RendersStandalone verifies that the
// checked-in product-critical-path guard partial (E34-F10) defines exactly
// one named template (spec.md AC-1, test-plan.md TC-001) and renders cleanly
// through the production template engine (text/template.Parse + Funcs +
// ExecuteTemplate, the same engine NewOrchestratorRenderer uses to compile
// every prompt) with no data supplied — this repository's actual state has
// none of the four REQ-F-032 source files, so the guard's static reporting
// text must name each one as an unresolved prerequisite (D-F10-03), name the
// five REQ-F-034 disqualified evidence classes verbatim, and contain no bare
// workflow status name or `shark` CLI verb (REQ-NF-002/003).
func TestProductCriticalPathGuardPartial_RendersStandalone(t *testing.T) {
	partialPath := canonicalPromptPath("_partials", "_product_critical_path_guard.md")

	raw, err := os.ReadFile(partialPath)
	require.NoError(t, err, "product-critical-path guard partial must exist at %s", partialPath)
	source := string(raw)

	// AC-T1: exactly one {{define}} block, naming the guard template.
	assert.Equal(t, 1, strings.Count(source, "{{define"),
		"guard partial must contain exactly one {{define}} block")
	assert.Contains(t, source, `{{define "_product_critical_path_guard"}}`,
		"guard partial must define the _product_critical_path_guard template")

	tmpl, err := template.New("guard").Funcs(orchestratorFuncs()).Parse(source)
	require.NoError(t, err, "guard partial must parse as a valid Go template")

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "_product_critical_path_guard", nil)
	require.NoError(t, err, "guard partial must render cleanly through the production template engine")
	out := buf.String()

	// AC-T2 / D-F10-03: all four REQ-F-032 source files are absent from this
	// repository today, and the guard must report each one, by its exact
	// path, as an unresolved prerequisite.
	for _, sourceFile := range []string{
		"docs/product/D01-vision-statement.md",
		"docs/product/D02-success-criteria.md",
		"docs/plan/product-delivery-roadmap.md",
		"docs/plan/product-critical-path.md",
	} {
		assert.Contains(t, out, "unresolved prerequisite: "+sourceFile+" missing",
			"guard must report %s as an unresolved prerequisite when absent", sourceFile)
	}

	// AC-T3 / REQ-F-034: the five disqualified evidence classes, reusing
	// E34-F02's evidence-authenticity vocabulary verbatim (research-report.md
	// finding 5).
	for _, term := range []string{
		"fixture data",
		"a captured/recorded run",
		"a hand-authored test actor",
		"a contract-only test",
		"a component-level test suite",
	} {
		assert.Contains(t, out, term, "guard must name disqualified evidence class %q", term)
	}

	// AC-T3 / REQ-NF-002/003: no bare workflow status name (from any entity
	// workflow YAML's steps: keys) or shark CLI verb anywhere in the guard.
	forbiddenStatusNames := []string{
		"blocked", "cancelled", "code_review", "completed", "development",
		"draft", "on_hold", "qa", "research", "active", "assessment",
		"decomposition", "design", "feature_review", "integration_review",
		"refinement", "approval", "task_generation", "task_review",
		"test_planning", "open", "answering", "ready_for_resolution",
		"resolved", "withdrawn", "superseded", "archived", "planning",
		"closing", "identified", "in_progress", "triaged", "wont_fix",
	}
	wordBoundary := regexp.MustCompile(`\bshark\b`)
	assert.False(t, wordBoundary.MatchString(out), "guard must not name the `shark` CLI")
	for _, status := range forbiddenStatusNames {
		re := regexp.MustCompile(`\b` + regexp.QuoteMeta(status) + `\b`)
		assert.False(t, re.MatchString(out), "guard must not contain bare workflow status name %q", status)
	}
}

// TestIncludeResolverWithEmbed_StateSpaceCoverageRenders verifies that the
// checked-in state-space-coverage.md workflow file (E34-F07) renders cleanly
// through the production renderer and carries every one of the six required
// sections (spec.md REQ-F-001-004/006/007), asserting each section's required
// substantive clauses per test-plan.md TC-001 — not just heading presence.
func TestIncludeResolverWithEmbed_StateSpaceCoverageRenders(t *testing.T) {
	r := NewIncludeResolverWithEmbed("")

	out, err := r.Resolve("{{include: skills/quality/workflows/state-space-coverage.md}}")
	require.NoError(t, err, "state-space-coverage.md must render through the production renderer with no errors")

	// All six required section headings (REQ-F-001-004/006/007).
	for _, section := range []string{
		"## Closed lifecycle tables",
		"## Technique selection from state shape",
		"## Dependency discovery by interaction and caller path",
		"## Shipped consumer re-verification",
		"## I-04 propagation",
		"## Design divergence",
	} {
		assert.Contains(t, out, section, "state-space-coverage.md must contain section %q", section)
	}

	// Closed lifecycle tables: the detection heuristic and all six required
	// table columns (TC-001 clause 1).
	assert.Contains(t, out, "behavior-bearing", "must state the lifecycle-field detection heuristic")
	for _, column := range []string{
		"value", "meaning", "entry transition", "exit transition",
		"terminal", "invalid-transition", "failure/recovery",
	} {
		assert.Contains(t, out, column, "closed-table section must name column %q", column)
	}

	// Technique selection: trigger condition and technique name (TC-001 clause 2).
	assert.Contains(t, out, "state-transition", "must name the state-transition/decision-table technique")
	assert.Contains(t, out, "decision-table", "must name the decision-table technique")

	// Dependency discovery: priority-ordered sources and per-axis rationale
	// recording (TC-001 clause 3).
	assert.Contains(t, out, "interaction-map", "must name interaction-map rows as a discovery source")
	assert.Contains(t, out, "Caller-Path Contract", "must name production caller paths via the Caller-Path Contract concept")
	assert.Contains(t, out, "persistence reader", "must name persistence readers as a discovery source")
	assert.Contains(t, out, "inclusion/exclusion rationale", "must require per-axis inclusion/exclusion rationale")

	// Shipped consumer re-verification: all four required fields (TC-001 clause 4).
	for _, field := range []string{
		"caller path", "owning feature key", "affected AC ID", "regression-test pointer",
	} {
		assert.Contains(t, out, field, "shipped-consumer section must name field %q", field)
	}

	// I-04 propagation: shape reference and the no-silent-omission language
	// (TC-001 clause 5).
	assert.Contains(t, out, "ChangeImpactSet", "must reference the ChangeImpactSet shape")
	assert.Contains(t, out, "never a completion record that omits an affected artifact without a stated disposition",
		"must carry the exact no-silent-omission language")

	// Design divergence: references (does not restate) defect-class-sweep.md's
	// Backward-looking rework section (TC-001 clause 6 / REQ-F-007).
	assert.Contains(t, out, "defect-class-sweep.md", "design-divergence section must reference defect-class-sweep.md")
	assert.Contains(t, out, "Backward-looking rework", "must reference defect-class-sweep.md's Backward-looking rework section by name")
}

// TestNewIncludeResolverWithEmbed_EmptyDataRoot verifies that the embed
// backstop works even with an empty data root (zero-config consumer mode).
func TestNewIncludeResolverWithEmbed_EmptyDataRoot(t *testing.T) {
	r := NewIncludeResolverWithEmbed("")

	out, err := r.Resolve("{{include: prompts/_partials/_commands.md}}")
	require.NoError(t, err, "embed-only mode (empty dataRoot) must resolve from embed")
	assert.NotEmpty(t, strings.TrimSpace(out))
}

// TestOrchestratorRenderer_LoadsFromEmbed verifies the zero-config path:
// when no disk prompts exist, the renderer loads templates from the embedded FS.
func TestOrchestratorRenderer_LoadsFromEmbed(t *testing.T) {
	// Point at a directory that exists but contains no prompt files.
	emptyPromptsDir := t.TempDir()

	renderer, err := NewOrchestratorRenderer(emptyPromptsDir)
	require.NoError(t, err, "renderer must initialize even with no disk prompts")
	require.NotNil(t, renderer)

	// Embedded prompts include task/development.md and others.
	// Verify at least one template is loaded from the embed.
	out, err := renderer.Render("task/development.md", map[string]string{
		"task_title":      "Test Task",
		"task_key":        "E01-F01-001",
		"feature_key":     "E01-F01",
		"epic_key":        "E01",
		"task_id":         "E01-F01-001",
		"task_status":     "development",
		"complexity_tier": "SIMPLE",
	})
	require.NoError(t, err, "embedded task/development.md must render successfully")
	assert.NotEmpty(t, strings.TrimSpace(out), "rendered output must be non-empty")
}

// ============================================================================
// T-E34-F08-002 / TC-001 — tier-matrix.md single-source-of-truth
// ============================================================================

// tierMatrixCanonicalPath is the exact bundle path the five consuming
// prompts must reference. TC-001's edge case ("a prompt that references the
// file by a stale/renamed path fails ... not just the grep") is why this
// constant, not a loose substring like "tier-matrix", is what the grep test
// below matches against.
const tierMatrixCanonicalPath = "skills/quality/context/tier-matrix.md"

// TestTierMatrixRendersThroughProductionRenderer renders tier-matrix.md
// through the production include resolver (embed-aware, matching how the
// five consuming prompts would pull it in) and asserts the SIMPLE/STANDARD/
// COMPLEX tier contract and the "missing artifacts are failures only when
// the selected tier requires them" paragraph (REQ-F-001) are present.
func TestTierMatrixRendersThroughProductionRenderer(t *testing.T) {
	r := NewIncludeResolverWithEmbed("")

	out, err := r.Resolve("{{include: " + tierMatrixCanonicalPath + "}}")
	require.NoError(t, err, "tier-matrix.md must render through the production renderer with no error")

	assert.Contains(t, out, "## Tier contract")
	assert.Contains(t, out, "## Executable gate evidence")
	for _, tier := range []string{"SIMPLE", "STANDARD", "COMPLEX"} {
		assert.Contains(t, out, tier, "tier contract table must name tier %q", tier)
	}
	assert.Contains(t, out, "Missing artifacts are failures only when the selected tier requires them.")
}

// TestTierMatrixIncludeResolvesFromStalePathFails is TC-001's explicit edge
// case: a stale or renamed path must fail include resolution, not just a
// grep for a loose substring.
func TestTierMatrixIncludeResolvesFromStalePathFails(t *testing.T) {
	r := NewIncludeResolverWithEmbed("")
	_, err := r.Resolve("{{include: skills/quality/context/tier-matrix-renamed.md}}")
	require.Error(t, err, "a stale/renamed tier-matrix.md path must fail include resolution")
}

// tierMatrixConsumingPrompts is the exact five-prompt set T-E34-F08-002's
// task scope names (REQ-F-001, AC-1/AC-T2).
var tierMatrixConsumingPrompts = []string{
	"feature/assessment.md",
	"feature/task_generation.md",
	"feature/code_review.md",
	"feature/qa.md",
	"feature/approval.md",
}

// tierMatrixTableHeaderCells is tier-matrix.md's own table header. A
// restated (not referenced) copy of the tier table in a consuming prompt
// would reproduce this same header row — this is the structural signal
// TC-001 requires ("a table with the same three tier names and a 'gate'
// column, not an exact-string blacklist").
var tierMatrixTableHeaderCells = []string{
	"Planning source", "Test source", "Same-model gate", "Separate QA", "Final UAT",
}

// TestFivePromptsReferenceTierMatrixWithoutRestatingIt is TC-001's
// structural grep/regex check: each of the five consuming prompts contains a
// textual pointer to tier-matrix.md's exact canonical path, and none of them
// restate the tier table itself.
func TestFivePromptsReferenceTierMatrixWithoutRestatingIt(t *testing.T) {
	for _, prompt := range tierMatrixConsumingPrompts {
		t.Run(prompt, func(t *testing.T) {
			body, err := sharkdata.ReadEmbedded("prompts/" + prompt)
			require.NoError(t, err, "embedded prompt %s must exist", prompt)
			content := string(body)

			assert.Contains(t, content, tierMatrixCanonicalPath,
				"%s must reference tier-matrix.md by its exact canonical path", prompt)

			restated := true
			for _, cell := range tierMatrixTableHeaderCells {
				if !strings.Contains(content, cell) {
					restated = false
					break
				}
			}
			assert.False(t, restated,
				"%s must not restate the tier table's own header cells (%v)", prompt, tierMatrixTableHeaderCells)
		})
	}
}
