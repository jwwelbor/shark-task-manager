package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func TestRenderer_LegacyTmplPathHasNoIncludeRoot(t *testing.T) {
	// A legacy shark-templates/ layout has no detectable data root, so
	// {{include:}} in a .tmpl is left as-is. Author shouldn't use {{include:}}
	// in legacy templates anyway; this test pins the contract.
	root := t.TempDir() // legacy: dir base is NOT "prompts"
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
