package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// E3 — `shark next` agent-body auto-inline (per 2026-05-10 rendering decision)
// ============================================================================

// setupAgentFixture lays down a minimal shark-data/ tree with one agent file
// (and optionally an override) and returns the data root.
func setupAgentFixture(t *testing.T, agentType, body string, overrideBody string) string {
	t.Helper()
	root := t.TempDir()
	dataRoot := filepath.Join(root, "shark-data")
	require.NoError(t, os.MkdirAll(filepath.Join(dataRoot, "agents"), 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dataRoot, "agents", agentType+".md"),
		[]byte(body),
		0644,
	))
	if overrideBody != "" {
		require.NoError(t, os.MkdirAll(filepath.Join(dataRoot, "overrides", "agents"), 0755))
		require.NoError(t, os.WriteFile(
			filepath.Join(dataRoot, "overrides", "agents", agentType+".md"),
			[]byte(overrideBody),
			0644,
		))
	}
	return dataRoot
}

func TestLoadAgentBodyForInline_LegacyModeReturnsFalse(t *testing.T) {
	// Empty root signals legacy mode (no shark-data/). The function must not
	// attempt resolution and must report "not inlined" without erroring.
	got, ok := LoadAgentBodyForInline("", "qa")
	assert.False(t, ok, "legacy mode should return ok=false")
	assert.Equal(t, "", got)
}

func TestLoadAgentBodyForInline_EmptyAgentTypeReturnsFalse(t *testing.T) {
	root := t.TempDir()
	got, ok := LoadAgentBodyForInline(root, "")
	assert.False(t, ok)
	assert.Equal(t, "", got)
}

func TestLoadAgentBodyForInline_AgentFileFound(t *testing.T) {
	body := "You are the QA agent. Tools: Read, Bash, Grep."
	root := setupAgentFixture(t, "qa", body, "")

	got, ok := LoadAgentBodyForInline(root, "qa")
	require.True(t, ok, "agent file should be resolved")
	assert.Equal(t, body, got)
}

func TestLoadAgentBodyForInline_FrontmatterStripped(t *testing.T) {
	// Authors may put YAML frontmatter on agent files (model, allowed-tools,
	// description). The resolver strips it before returning the body — the
	// frontmatter is metadata, not content for the inlined prompt.
	body := "---\nname: qa\nmodel: opus\nallowed-tools: Read, Bash\n---\nQA agent persona body."
	root := setupAgentFixture(t, "qa", body, "")

	got, ok := LoadAgentBodyForInline(root, "qa")
	require.True(t, ok)
	assert.Equal(t, "QA agent persona body.", got)
	assert.NotContains(t, got, "name: qa", "frontmatter must be stripped")
	assert.NotContains(t, got, "---", "frontmatter delimiters must be stripped")
}

func TestLoadAgentBodyForInline_OverrideWins(t *testing.T) {
	defaultBody := "DEFAULT qa agent body"
	overrideBody := "OVERRIDE qa agent body"
	root := setupAgentFixture(t, "qa", defaultBody, overrideBody)

	got, ok := LoadAgentBodyForInline(root, "qa")
	require.True(t, ok)
	assert.Equal(t, overrideBody, got, "override under overrides/agents/ must fully replace the default")
}

func TestLoadAgentBodyForInline_AgentMissingReturnsFalse(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "shark-data", "agents"), 0755))

	got, ok := LoadAgentBodyForInline(filepath.Join(root, "shark-data"), "ghost-agent-that-doesnt-exist")
	assert.False(t, ok, "missing agent should return ok=false (non-fatal)")
	assert.Equal(t, "", got)
}

func TestLoadAgentBodyForInline_EmptyBodyTreatedAsMissing(t *testing.T) {
	// An agent file with only frontmatter (no body content) shouldn't
	// produce an empty inline — return ok=false so callers don't prepend
	// useless whitespace + separator.
	body := "---\nname: stub\n---\n"
	root := setupAgentFixture(t, "stub", body, "")

	got, ok := LoadAgentBodyForInline(root, "stub")
	assert.False(t, ok, "agent file with no body after frontmatter strip should report not inlined")
	assert.Equal(t, "", got)
}

func TestLoadAgentBodyForInline_PathTraversalRejected(t *testing.T) {
	root := t.TempDir()
	got, ok := LoadAgentBodyForInline(root, "../../../etc/passwd")
	assert.False(t, ok, "path-traversal agent type must not resolve")
	assert.Equal(t, "", got)
}

func TestLoadAgentBodyForInline_PrependFormatExample(t *testing.T) {
	// Documents the prepend format used in runNext so future reviewers see
	// the contract: agent body, blank line, --- separator, blank line, then
	// the action prompt.
	body := "QA persona"
	root := setupAgentFixture(t, "qa", body, "")

	got, ok := LoadAgentBodyForInline(root, "qa")
	require.True(t, ok)

	actionPrompt := "Run QA on E07-F02-001..."
	combined := got + "\n\n---\n\n" + actionPrompt

	assert.True(t, strings.HasPrefix(combined, "QA persona"))
	assert.Contains(t, combined, "\n\n---\n\n")
	assert.True(t, strings.HasSuffix(combined, actionPrompt))
}

// findRepoPromptsDir walks up from the test working directory looking for the
// canonical prompts directory. It checks the deployed shark-data/prompts tree
// first (present after `shark admin init`), then falls back to the embedded
// canonical at internal/sharkdata/default_data/prompts (always present in the
// repo). This lets the suite run in a clean checkout without a materialized
// shark-data/ on disk.
func findRepoPromptsDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	dir := wd
	for {
		// Prefer the deployed copy (shark-data/prompts) when present.
		if candidate := filepath.Join(dir, "shark-data", "prompts"); isDirExist(candidate) {
			return candidate
		}
		// Fall back to the embedded canonical (always present in the repo).
		if candidate := filepath.Join(dir, "internal", "sharkdata", "default_data", "prompts"); isDirExist(candidate) {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate prompts directory (shark-data/prompts or internal/sharkdata/default_data/prompts) walking up from %s", wd)
		}
		dir = parent
	}
}

// isDirExist returns true when path exists and is a directory.
func isDirExist(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// TestRunNext_InlinesSkillContent is the F02 AC #2 end-to-end check: the
// shipped feature/ready_for_assessment.md prompt must produce a rendered
// output that contains the body of skills/assessment/SKILL.md inlined via
// {{include:}}, not a path reference.
func TestRunNext_InlinesSkillContent(t *testing.T) {
	promptsDir := findRepoPromptsDir(t)

	renderer, err := templates.NewOrchestratorRenderer(promptsDir)
	require.NoError(t, err, "shipped prompts must parse with includes resolved")

	out, err := renderer.Render("feature/ready_for_assessment.md", map[string]string{
		"id":        "E32-F02",
		"title":     "Engine — includes",
		"file_path": "docs/plan/E32/E32-F02/E32-F02.md",
		"epic_id":   "E32",
		"is_resume": "false",
	})
	require.NoError(t, err)

	// The skill body has a stable H1 that proves the file was inlined,
	// not merely referenced by path.
	assert.Contains(t, out, "# Assessment Skill (craft)",
		"rendered prompt must inline skill body via {{include:}}")
	assert.NotContains(t, out, "Load skill: ",
		"path-reference idiom should not appear in the rendered prompt")
}
