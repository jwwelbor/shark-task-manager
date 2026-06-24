package templates

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Shark 2.0 (.md prompts + frontmatter + shark-data/ resolution) tests
// ============================================================================

func TestStripFrontmatter_NoFrontmatter(t *testing.T) {
	body := "# Just a heading\n\nNo frontmatter here.\n"
	got := stripFrontmatter(body)
	assert.Equal(t, body, got, "content without frontmatter should be unchanged")
}

func TestStripFrontmatter_WithFrontmatter(t *testing.T) {
	content := "---\nname: foo\nstatus: in_qa\n---\n# Body heading\n\nBody content.\n"
	want := "# Body heading\n\nBody content.\n"
	got := stripFrontmatter(content)
	assert.Equal(t, want, got)
}

func TestStripFrontmatter_FrontmatterCRLF(t *testing.T) {
	content := "---\r\nname: foo\r\n---\r\nBody.\r\n"
	got := stripFrontmatter(content)
	assert.Equal(t, "Body.\n", got, "CRLF frontmatter should be stripped")
}

func TestStripFrontmatter_UnclosedFrontmatter(t *testing.T) {
	// No closing --- → treat as if there's no frontmatter (avoid silent data loss).
	content := "---\nname: foo\nstill no close\nbody"
	got := stripFrontmatter(content)
	assert.Equal(t, content, got)
}

func TestStripFrontmatter_FrontmatterOnly(t *testing.T) {
	// Frontmatter present, body is empty.
	content := "---\nname: foo\n---\n"
	got := stripFrontmatter(content)
	assert.Equal(t, "", got)
}

func TestNewOrchestratorRenderer_MdFile(t *testing.T) {
	// E4: .md prompt files render correctly.
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "task"), 0755))

	mdContent := "Task: {{.task_id}} — {{.title}}"
	require.NoError(t, os.WriteFile(
		filepath.Join(root, "task", "in_qa.md"),
		[]byte(mdContent),
		0644,
	))

	renderer, err := NewOrchestratorRenderer(root)
	require.NoError(t, err)

	out, err := renderer.Render("task/in_qa.md", map[string]string{
		"task_id": "E01-F02-001",
		"title":   "Auth flow",
	})
	require.NoError(t, err)
	assert.Equal(t, "Task: E01-F02-001 — Auth flow", out)
}

func TestNewOrchestratorRenderer_MdFileWithFrontmatter(t *testing.T) {
	// E4: .md prompt with YAML frontmatter — frontmatter is stripped before
	// the body is parsed as a Go template.
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "task"), 0755))

	content := `---
entity_type: task
status: in_qa
agent_type: qa
includes:
  - skills/quality/workflows/qa-testing.md
---
Task: {{.task_id}}
Status: in_qa
`
	require.NoError(t, os.WriteFile(
		filepath.Join(root, "task", "in_qa.md"),
		[]byte(content),
		0644,
	))

	renderer, err := NewOrchestratorRenderer(root)
	require.NoError(t, err)

	out, err := renderer.Render("task/in_qa.md", map[string]string{
		"task_id": "E01-F02-001",
	})
	require.NoError(t, err)
	assert.Equal(t, "Task: E01-F02-001\nStatus: in_qa\n", out)
}

func TestNewOrchestratorRenderer_MixedTmplAndMd(t *testing.T) {
	// .tmpl and .md files coexist in the same templateDir during the F4
	// migration window.
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "task"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "feature"), 0755))

	// Legacy .tmpl
	require.NoError(t, os.WriteFile(
		filepath.Join(root, "task", "in_qa.tmpl"),
		[]byte("LEGACY-{{.task_id}}"),
		0644,
	))
	// New .md without frontmatter
	require.NoError(t, os.WriteFile(
		filepath.Join(root, "feature", "in_design.md"),
		[]byte("NEW-{{.feature_id}}"),
		0644,
	))

	renderer, err := NewOrchestratorRenderer(root)
	require.NoError(t, err)

	legacy, err := renderer.Render("task/in_qa.tmpl", map[string]string{"task_id": "T1"})
	require.NoError(t, err)
	assert.Equal(t, "LEGACY-T1", legacy)

	newMd, err := renderer.Render("feature/in_design.md", map[string]string{"feature_id": "F1"})
	require.NoError(t, err)
	assert.Equal(t, "NEW-F1", newMd)
}

func TestNewOrchestratorRenderer_TmplFrontmatterPreserved(t *testing.T) {
	// .tmpl files do NOT get frontmatter-stripped — backward compat. If a
	// legacy .tmpl happens to start with `---`, that's part of its rendered
	// output, not metadata to strip.
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "task"), 0755))

	content := "---\nthis-line: stays-in-output\n---\nBody {{.x}}"
	require.NoError(t, os.WriteFile(
		filepath.Join(root, "task", "weird.tmpl"),
		[]byte(content),
		0644,
	))

	renderer, err := NewOrchestratorRenderer(root)
	require.NoError(t, err)

	out, err := renderer.Render("task/weird.tmpl", map[string]string{"x": "v"})
	require.NoError(t, err)
	// Full content (including the --- lines) is rendered as template body.
	assert.Equal(t, "---\nthis-line: stays-in-output\n---\nBody v", out)
}

func TestHasPromptFiles(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "task"), 0755))

	assert.False(t, hasPromptFiles(root), "empty subdir should report no prompts")

	require.NoError(t, os.WriteFile(
		filepath.Join(root, "task", "x.tmpl"),
		[]byte("ok"),
		0644,
	))
	assert.True(t, hasPromptFiles(root), ".tmpl prompt should be detected")

	root2 := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root2, "task"), 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(root2, "task", "x.md"),
		[]byte("ok"),
		0644,
	))
	assert.True(t, hasPromptFiles(root2), ".md prompt should be detected")
}

func TestFindTemplateDir_PrefersSharkData(t *testing.T) {
	// E5: when both shark-data/prompts/ and shark-templates/ exist, prefer
	// shark-data/prompts/.
	root := t.TempDir()
	sharkData := filepath.Join(root, sharkDataPromptsSubdir, "task")
	sharkTemplates := filepath.Join(root, defaultTemplateDir, "task")
	require.NoError(t, os.MkdirAll(sharkData, 0755))
	require.NoError(t, os.MkdirAll(sharkTemplates, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sharkData, "in_qa.md"), []byte("MD"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(sharkTemplates, "in_qa.tmpl"), []byte("TMPL"), 0644))

	cwd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(cwd) }()
	require.NoError(t, os.Chdir(root))

	// Reset configured override so default resolution kicks in.
	prevConfigured := configuredTemplateDir
	configuredTemplateDir = ""
	defer func() { configuredTemplateDir = prevConfigured }()

	got := findTemplateDir()
	assert.Equal(t, filepath.Join(root, sharkDataPromptsSubdir), got,
		"findTemplateDir should prefer shark-data/prompts/ over shark-templates/")
}

func TestFindTemplateDir_FallsBackToSharkTemplates(t *testing.T) {
	// E5: when only shark-templates/ exists, fall back to it.
	root := t.TempDir()
	sharkTemplates := filepath.Join(root, defaultTemplateDir, "task")
	require.NoError(t, os.MkdirAll(sharkTemplates, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sharkTemplates, "in_qa.tmpl"), []byte("TMPL"), 0644))

	cwd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(cwd) }()
	require.NoError(t, os.Chdir(root))

	prevConfigured := configuredTemplateDir
	configuredTemplateDir = ""
	defer func() { configuredTemplateDir = prevConfigured }()

	got := findTemplateDir()
	assert.Equal(t, filepath.Join(root, defaultTemplateDir), got,
		"findTemplateDir should fall back to shark-templates/ when shark-data/prompts/ is absent")
}

func TestFindTemplateDir_AbsoluteOverrideUsedDirectly(t *testing.T) {
	// E5: absolute configured override skips the walk-up entirely.
	root := t.TempDir()
	abs := filepath.Join(root, "custom-templates")
	require.NoError(t, os.MkdirAll(filepath.Join(abs, "task"), 0755))

	prevConfigured := configuredTemplateDir
	configuredTemplateDir = abs
	defer func() { configuredTemplateDir = prevConfigured }()

	got := findTemplateDir()
	assert.Equal(t, abs, got, "absolute configured override should be returned directly")
}

func TestFindTemplateDir_NoLayoutFallsBackToConfiguredName(t *testing.T) {
	// When neither layout exists, return the configured directory name as a
	// relative path (the loader will then return an empty renderer for it).
	root := t.TempDir()
	cwd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(cwd) }()
	require.NoError(t, os.Chdir(root))

	prevConfigured := configuredTemplateDir
	configuredTemplateDir = ""
	defer func() { configuredTemplateDir = prevConfigured }()

	got := findTemplateDir()
	assert.Equal(t, defaultTemplateDir, got)
}
