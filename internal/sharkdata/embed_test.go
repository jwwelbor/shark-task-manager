package sharkdata

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readEmbeddedAll is a test-only convenience for reading a file from the
// embedded tree by its path relative to embedRootDir.
func readEmbeddedAll(rel string) ([]byte, error) {
	return embeddedFS.ReadFile(filepath.Join(embedRootDir, rel))
}

// ============================================================================
// ReadEmbedded — security-rejection paths
// ============================================================================

func TestReadEmbedded_AbsolutePath(t *testing.T) {
	_, err := ReadEmbedded("/etc/passwd")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be relative")
}

func TestReadEmbedded_ParentTraversal(t *testing.T) {
	_, err := ReadEmbedded("../etc/passwd")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must not escape")
}

func TestReadEmbedded_NotFound(t *testing.T) {
	_, err := ReadEmbedded("prompts/nonexistent/file-that-does-not-exist.md")
	require.Error(t, err)
	assert.True(t, errors.Is(err, fs.ErrNotExist), "expected fs.ErrNotExist, got %v", err)
}

func TestReadEmbedded_KnownFile(t *testing.T) {
	data, err := ReadEmbedded("README.md")
	require.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.Contains(t, string(data), "shark-data")
}

func TestReadEmbedded_KnownEntityTemplateFile(t *testing.T) {
	data, err := ReadEmbedded("templates/epic.md")
	require.NoError(t, err)
	assert.Contains(t, string(data), "epic_key")
}

// ============================================================================
// E6 — shark init
// ============================================================================

func TestInit_FreshProject(t *testing.T) {
	root := t.TempDir()
	dest, err := Init(root)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, SharkDataDirName), dest)

	// README.md is part of the embedded skeleton.
	readme, err := os.ReadFile(filepath.Join(dest, "README.md"))
	require.NoError(t, err)
	assert.Contains(t, string(readme), "shark-data")

	// All six subdirectories are present.
	for _, sub := range []string{"prompts", "templates", "skills", "agents", "workflow", "overrides"} {
		info, err := os.Stat(filepath.Join(dest, sub))
		require.NoError(t, err)
		assert.True(t, info.IsDir(), "%s should be a directory", sub)
	}

	epicTemplate, err := os.ReadFile(filepath.Join(dest, "templates", "epic.md"))
	require.NoError(t, err)
	assert.Contains(t, string(epicTemplate), "epic_key")
}

func TestInit_AlreadyInitialized(t *testing.T) {
	root := t.TempDir()
	// First init succeeds.
	_, err := Init(root)
	require.NoError(t, err)

	// Second init returns the sentinel error and does not overwrite.
	dest, err := Init(root)
	require.ErrorIs(t, err, ErrAlreadyInitialized)
	assert.Equal(t, filepath.Join(root, SharkDataDirName), dest)
}

// ============================================================================
// E7 — shark upgrade
// ============================================================================

func TestUpgrade_RequiresExistingTree(t *testing.T) {
	root := t.TempDir()
	_, err := Upgrade(root, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestUpgrade_PreservesOverrides(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	// User adds a local override.
	overridePath := filepath.Join(root, SharkDataDirName, "overrides", "skills", "quality", "qa-testing.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(overridePath), 0755))
	overrideContent := []byte("USER OVERRIDE — do not touch")
	require.NoError(t, os.WriteFile(overridePath, overrideContent, 0644))

	summary, err := Upgrade(root, false)
	require.NoError(t, err)
	require.NotNil(t, summary)

	// Override file is byte-identical after upgrade.
	got, err := os.ReadFile(overridePath)
	require.NoError(t, err)
	assert.Equal(t, overrideContent, got, "upgrade must not touch files under overrides/")

	// Summary records that overrides were skipped.
	assert.NotEmpty(t, summary.SkippedOverrides, "summary should list at least the overrides/.gitkeep entries it skipped")
}

func TestUpgrade_DryRunWritesNothing(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	// Manually break the README to verify dry-run does NOT restore it.
	readmePath := filepath.Join(root, SharkDataDirName, "README.md")
	require.NoError(t, os.WriteFile(readmePath, []byte("LOCALLY EDITED"), 0644))

	summary, err := Upgrade(root, true)
	require.NoError(t, err)
	require.NotNil(t, summary)
	assert.Contains(t, summary.Updated, "README.md", "README.md should appear in Updated under dry-run")

	// File on disk still has the local edit.
	got, err := os.ReadFile(readmePath)
	require.NoError(t, err)
	assert.Equal(t, []byte("LOCALLY EDITED"), got, "dry-run must not write changes")
}

func TestUpgrade_ApplyRestoresCanonical(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	readmePath := filepath.Join(root, SharkDataDirName, "README.md")
	require.NoError(t, os.WriteFile(readmePath, []byte("LOCALLY EDITED"), 0644))

	_, err = Upgrade(root, false)
	require.NoError(t, err)

	got, err := os.ReadFile(readmePath)
	require.NoError(t, err)
	assert.NotEqual(t, []byte("LOCALLY EDITED"), got, "upgrade should restore the canonical README")
	assert.Contains(t, string(got), "shark-data", "restored content should be the embedded README")
}

// ============================================================================
// E9 — shark validate
// ============================================================================

func TestValidate_MissingTreeReportsError(t *testing.T) {
	root := t.TempDir()
	report, err := Validate(root)
	require.NoError(t, err)
	require.NotNil(t, report)
	require.NotEmpty(t, report.Issues)
	assert.True(t, report.HasErrors())
	assert.Contains(t, report.Issues[0].Message, "does not exist")
}

func TestValidate_FreshInitPasses(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	report, err := Validate(root)
	require.NoError(t, err)
	require.NotNil(t, report)
	for _, issue := range report.Issues {
		assert.NotEqual(t, IssueLevelError, issue.Level, "fresh init should produce zero error-level issues; got: %+v", issue)
	}
}

func TestValidate_BrokenInclude(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	// Add a prompt that references a non-existent file.
	promptDir := filepath.Join(root, SharkDataDirName, "prompts", "task")
	require.NoError(t, os.MkdirAll(promptDir, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(promptDir, "scratch.md"),
		[]byte("Header. {{include: skills/missing/whatever.md}}"),
		0644,
	))

	report, err := Validate(root)
	require.NoError(t, err)
	require.True(t, report.HasErrors(), "missing include target should be flagged as error")

	var found bool
	for _, issue := range report.Issues {
		if strings.Contains(issue.Message, "skills/missing/whatever.md") {
			found = true
			assert.Equal(t, IssueLevelError, issue.Level)
		}
	}
	assert.True(t, found, "report should mention the missing include path")
}

func TestValidate_AbsoluteIncludeRejected(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	promptDir := filepath.Join(root, SharkDataDirName, "prompts", "task")
	require.NoError(t, os.MkdirAll(promptDir, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(promptDir, "scratch.md"),
		[]byte("{{include: /etc/passwd}}"),
		0644,
	))

	report, err := Validate(root)
	require.NoError(t, err)
	require.True(t, report.HasErrors())

	var found bool
	for _, issue := range report.Issues {
		if strings.Contains(issue.Message, "absolute") {
			found = true
		}
	}
	assert.True(t, found, "validate must flag absolute include paths")
}

func TestValidate_ParentTraversalRejected(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	promptDir := filepath.Join(root, SharkDataDirName, "prompts", "task")
	require.NoError(t, os.MkdirAll(promptDir, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(promptDir, "scratch.md"),
		[]byte("{{include: ../../../etc/passwd}}"),
		0644,
	))

	report, err := Validate(root)
	require.NoError(t, err)
	require.True(t, report.HasErrors())
}

func TestValidate_OverrideOnlyFileWarns(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	// Override exists but no canonical default.
	overridePath := filepath.Join(root, SharkDataDirName, "overrides", "skills", "quality", "qa-testing.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(overridePath), 0755))
	require.NoError(t, os.WriteFile(overridePath, []byte("override only"), 0644))

	// Prompt references that path.
	promptDir := filepath.Join(root, SharkDataDirName, "prompts", "task")
	require.NoError(t, os.MkdirAll(promptDir, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(promptDir, "scratch.md"),
		[]byte("{{include: skills/quality/qa-testing.md}}"),
		0644,
	))

	report, err := Validate(root)
	require.NoError(t, err)
	// Override-only is a warning, not an error.
	assert.False(t, report.HasErrors())

	var foundWarning bool
	for _, issue := range report.Issues {
		if issue.Level == IssueLevelWarning && strings.Contains(issue.Message, "only an override") {
			foundWarning = true
		}
	}
	assert.True(t, foundWarning, "override-only include should produce a warning")
}

func TestValidate_BadWorkflowYAML(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	workflowDir := filepath.Join(root, SharkDataDirName, "workflow")
	require.NoError(t, os.WriteFile(
		filepath.Join(workflowDir, "task.yaml"),
		[]byte("status_flow: not-a-map\nspecial_statuses: also bad"),
		0644,
	))

	report, err := Validate(root)
	require.NoError(t, err)
	// The decode succeeds (it's still YAML) and both required top-level keys are
	// present, so the presence checks must NOT report status_flow as missing.
	// This documents the explicit gap: validation is presence-level, not
	// schema-level (a future tightening would catch "not-a-map"). Assert the
	// gap concretely rather than leaving the loop assertion-free.
	for _, issue := range report.Issues {
		assert.NotContains(t, issue.Message, "missing required key \"status_flow\"",
			"weak-but-present status_flow must not be flagged as missing")
	}
}

func TestValidate_WorkflowYAMLMissingRequiredKey(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	workflowDir := filepath.Join(root, SharkDataDirName, "workflow")
	require.NoError(t, os.WriteFile(
		filepath.Join(workflowDir, "task.yaml"),
		[]byte("status_flow_version: \"1.0\""), // status_flow missing
		0644,
	))

	report, err := Validate(root)
	require.NoError(t, err)
	require.True(t, report.HasErrors(), "missing status_flow should be an error")

	var found bool
	for _, issue := range report.Issues {
		if strings.Contains(issue.Message, "status_flow") {
			found = true
		}
	}
	assert.True(t, found)
}

// TestValidate_MissingAgentRef verifies that shark validate exits with an
// error-level issue when a workflow YAML references an agent_type that has no
// corresponding file under shark-data/agents/. This covers E9 AC6.
func TestValidate_MissingAgentRef(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	// Overwrite task.yaml to reference a nonexistent agent.
	workflowDir := filepath.Join(root, SharkDataDirName, "workflow")
	require.NoError(t, os.WriteFile(
		filepath.Join(workflowDir, "task.yaml"),
		[]byte(`version: '1.0'
status_flow:
  todo:
  - done
  done: []
special_statuses:
  terminal: [done]
status_metadata:
  todo:
    responsibility: agent
    orchestrator_action:
      action: spawn_agent
      agent_type: nonexistent_agent_xyz
`),
		0644,
	))

	report, err := Validate(root)
	require.NoError(t, err)
	require.True(t, report.HasErrors(), "missing agent file should be flagged as error")

	var found bool
	for _, issue := range report.Issues {
		if issue.Level == IssueLevelError && strings.Contains(issue.Message, "nonexistent_agent_xyz") {
			found = true
		}
	}
	assert.True(t, found, "report should mention the missing agent_type; got issues: %+v", report.Issues)
}

func TestValidate_MissingWorkflowPromptRef(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	workflowDir := filepath.Join(root, SharkDataDirName, "workflow")
	require.NoError(t, os.WriteFile(
		filepath.Join(workflowDir, "task.yaml"),
		[]byte(`version: '1.0'
start: draft
steps:
  draft:
    phase: planning
    action: advance_status
    prompt: task/missing-prompt.md
    outcomes:
      pass: completed
      fail: draft
      blocked: blocked
  completed:
    phase: done
    terminal: true
    prompt: task/completed.md
`),
		0644,
	))

	report, err := Validate(root)
	require.NoError(t, err)
	assertReportHasErrorContaining(t, report, "task/missing-prompt.md")
}

func TestValidate_MissingWorkflowSkillRef(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	workflowDir := filepath.Join(root, SharkDataDirName, "workflow")
	require.NoError(t, os.WriteFile(
		filepath.Join(workflowDir, "task.yaml"),
		[]byte(`version: '1.0'
start: draft
steps:
  draft:
    phase: planning
    action: spawn_agent
    agent: developer
    skills: [missing-skill-xyz]
    prompt: task/development.md
    outcomes:
      pass: completed
      fail: draft
      blocked: blocked
  completed:
    phase: done
    terminal: true
    prompt: task/completed.md
`),
		0644,
	))

	report, err := Validate(root)
	require.NoError(t, err)
	assertReportHasErrorContaining(t, report, "missing-skill-xyz")
}

func TestValidate_LegacyWorkflowAliasRejected(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	workflowDir := filepath.Join(root, SharkDataDirName, "workflow")
	require.NoError(t, os.WriteFile(
		filepath.Join(workflowDir, "task.yaml"),
		[]byte(`version: '1.0'
start: draft
steps:
  draft:
    phase: planning
    action: advance_status
    prompt: task/draft.md
    outcomes:
      pass: development
      fail: draft
      blocked: blocked
  development:
    phase: development
    action: spawn_agent
    agent: developer
    skills: [implementation]
    prompt: task/development.md
    outcomes:
      pass: completed
      fail: draft
      blocked: blocked
    aliases: [ready_for_development, in_development]
  completed:
    phase: done
    terminal: true
    prompt: task/completed.md
`),
		0644,
	))

	report, err := Validate(root)
	require.NoError(t, err)
	assertReportHasErrorContaining(t, report, "ready_for_development")
	assertReportHasErrorContaining(t, report, "in_development")
}

func TestValidate_LegacyPromptFilenameRejected(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	promptDir := filepath.Join(root, SharkDataDirName, "prompts", "task")
	require.NoError(t, os.WriteFile(filepath.Join(promptDir, "ready_for_qa.md"), []byte("legacy"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(promptDir, "in_qa.md"), []byte("legacy"), 0644))

	report, err := Validate(root)
	require.NoError(t, err)
	assertReportHasErrorContaining(t, report, "ready_for_qa.md")
	assertReportHasErrorContaining(t, report, "in_qa.md")
}

func TestValidate_LegacyStatusLiteralRejectedInActiveInstructions(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	agentPath := filepath.Join(root, SharkDataDirName, "agents", "qa.md")
	require.NoError(t, os.WriteFile(agentPath, []byte("Route failures to ready_for_development."), 0644))

	promptPath := filepath.Join(root, SharkDataDirName, "prompts", "task", "development.md")
	require.NoError(t, os.WriteFile(promptPath, []byte("Do not set in_development directly."), 0644))

	skillPath := filepath.Join(root, SharkDataDirName, "skills", "quality", "SKILL.md")
	require.NoError(t, os.WriteFile(skillPath, []byte("No embedded skill should mention in_progress task status."), 0644))

	report, err := Validate(root)
	require.NoError(t, err)
	assertReportHasErrorContaining(t, report, "ready_for_development")
	assertReportHasErrorContaining(t, report, "in_development")
	assertReportHasErrorContaining(t, report, "in_progress")
}

// TestValidate_MissingWorkflowYAML_SingleFile verifies that when one of the
// expected per-entity workflow YAML files is absent from shark-data/workflow/,
// shark validate surfaces an error-level issue (not silently falls back to
// hardcoded defaults).  This is the regression test for B023.
func TestValidate_MissingWorkflowYAML_SingleFile(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	// Remove feature.yaml — the same operation the B023 reproducer uses.
	featurePath := filepath.Join(root, SharkDataDirName, "workflow", "feature.yaml")
	require.NoError(t, os.Remove(featurePath))

	report, err := Validate(root)
	require.NoError(t, err)
	require.NotNil(t, report)

	// Validation must report at least one error (not just warnings).
	require.True(t, report.HasErrors(), "missing feature.yaml should produce an error-level issue")

	// The error message must name the missing file so the user can act on it.
	var found bool
	for _, issue := range report.Issues {
		if issue.Level == IssueLevelError && strings.Contains(issue.Path, "feature.yaml") {
			found = true
		}
	}
	assert.True(t, found, "error issue should reference workflow/feature.yaml; got issues: %+v", report.Issues)
}

// TestValidate_MissingWorkflowYAML_MultipleFiles verifies that each missing
// expected workflow file produces its own error issue (not just the first one).
func TestValidate_MissingWorkflowYAML_MultipleFiles(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	workflowDir := filepath.Join(root, SharkDataDirName, "workflow")
	require.NoError(t, os.Remove(filepath.Join(workflowDir, "feature.yaml")))
	require.NoError(t, os.Remove(filepath.Join(workflowDir, "epic.yaml")))

	report, err := Validate(root)
	require.NoError(t, err)
	require.True(t, report.HasErrors())

	// Both missing files must appear as separate error issues.
	missing := map[string]bool{"feature.yaml": false, "epic.yaml": false}
	for _, issue := range report.Issues {
		if issue.Level == IssueLevelError {
			for name := range missing {
				if strings.Contains(issue.Path, name) {
					missing[name] = true
				}
			}
		}
	}
	for name, found := range missing {
		assert.True(t, found, "expected error issue for missing %s; got issues: %+v", name, report.Issues)
	}
}

// TestValidate_SkipsExtractedSidecars verifies that shark validate does not
// inspect SKILL.md (or any other) files under a skill's _extracted/ directory.
// Those sidecars are F1 migration scaffolding, not canonical skill methodology,
// so they must be excluded from the skill-purity gate (E32-F04 AC-10). A
// scaffolding sidecar with malformed frontmatter must produce no issue.
func TestValidate_SkipsExtractedSidecars(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	// Drop a scaffolding sidecar whose frontmatter is NOT strict YAML. If the
	// validator walked _extracted/, this would surface a warning referencing
	// the file. It must not.
	extractedDir := filepath.Join(root, SharkDataDirName, "skills", "assessment", "_extracted")
	require.NoError(t, os.MkdirAll(extractedDir, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(extractedDir, "SKILL.md"),
		[]byte("---\n: this is not valid yaml : at all :\n---\n# scaffolding capture\n"),
		0644,
	))

	report, err := Validate(root)
	require.NoError(t, err)
	require.NotNil(t, report)

	for _, issue := range report.Issues {
		assert.NotContains(t, issue.Path, "_extracted",
			"validate must skip _extracted/ sidecars; got issue: %+v", issue)
	}
}

// TestValidate_SkipsExtractedInclude verifies that prompt-include validation
// also skips _extracted/ sidecars: a broken {{include:}} inside a scaffolding
// file must not fail the gate, since these files are never rendered.
func TestValidate_SkipsExtractedInclude(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	// Sidecars live under skills/, but the include scan walks prompts/. Place a
	// scaffolding _extracted/ tree under prompts/ to prove the walk skips it
	// regardless of which subtree it appears in.
	extractedDir := filepath.Join(root, SharkDataDirName, "prompts", "_extracted")
	require.NoError(t, os.MkdirAll(extractedDir, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(extractedDir, "capture.md"),
		[]byte("{{include: skills/does-not-exist/whatever.md}}"),
		0644,
	))

	report, err := Validate(root)
	require.NoError(t, err)
	require.NotNil(t, report)

	for _, issue := range report.Issues {
		assert.NotContains(t, issue.Path, "_extracted",
			"validate must skip _extracted/ in prompt-include scan; got issue: %+v", issue)
	}
}

// ============================================================================
// embed.FS sanity
// ============================================================================

func TestEmbedded_HasReadme(t *testing.T) {
	data, err := readEmbeddedAll("README.md")
	require.NoError(t, err)
	assert.Contains(t, string(data), "shark-data")
}

func TestEmbedded_AllExpectedDirectoriesPresent(t *testing.T) {
	paths, err := CopyEmbeddedTreeForTest()
	require.NoError(t, err)

	// Every top-level directory must be represented either by real content
	// or by a .gitkeep placeholder. Real content is preferred (means F4
	// populated the directory); .gitkeep is fallback during the F3 skeleton
	// phase.
	requiredDirs := []string{"README.md", "prompts/", "templates/", "skills/", "agents/", "workflow/", "overrides/"}
	for _, want := range requiredDirs {
		var found bool
		for _, got := range paths {
			if got == want || strings.HasPrefix(got, want) {
				found = true
				break
			}
		}
		assert.True(t, found, "embedded tree should include something at %s; got top: %s", want, firstN(paths, 5))
	}
}

// firstN returns the first n elements of paths as a slice for error messages.
func firstN(paths []string, n int) []string {
	if len(paths) < n {
		return paths
	}
	return paths[:n]
}

// TestEmbedded_SkillsContainNoBareSharkCLIRefs enforces the skill-purity rule
// (E32-F09 AC): no skill .md body outside _extracted/ sidecars may contain a
// bare "shark <verb>" CLI invocation.  _extracted/ files are migration
// scaffolding excluded by the same logic as TestValidate_SkipsExtractedSidecars.
//
// This turns the one-time acceptance-criterion grep into a permanent regression
// gate: if a future edit re-introduces a CLI ref into a canonical skill file,
// `make test` will fail here before the change can merge.
func TestEmbedded_SkillsContainNoBareSharkCLIRefs(t *testing.T) {
	const skillsPrefix = "skills/"
	const extractedDir = "/_extracted/"

	// Verbs that would constitute a bare CLI invocation.
	cliVerbs := []string{
		"shark status ", "shark get ", "shark task ", "shark feature ",
		"shark epic ", "shark list ", "shark create ", "shark delete ",
		"shark update ", "shark progress ", "shark analytics ", "shark cloud ",
		"shark admin ", "shark config ", "shark search ", "shark view ",
		"shark notes ", "shark idea ", "shark bug ", "shark td ",
	}

	var violations []string

	err := walkEmbedded(func(relPath string, data []byte, isDir bool) error {
		if isDir {
			return nil
		}
		if !strings.HasPrefix(relPath, skillsPrefix) {
			return nil
		}
		if !strings.HasSuffix(relPath, ".md") {
			return nil
		}
		// Skip _extracted/ sidecars — they are migration scaffolding, not
		// canonical skill content (E32-F04 AC-10 / E32-F09 scope exclusion).
		if strings.Contains(relPath, extractedDir) {
			return nil
		}
		content := string(data)
		for _, verb := range cliVerbs {
			if strings.Contains(content, verb) {
				violations = append(violations, relPath+": contains \""+verb+"\"")
			}
		}
		return nil
	})
	require.NoError(t, err)

	assert.Empty(t, violations,
		"skill files must not contain bare shark CLI invocations; found:\n%s",
		strings.Join(violations, "\n"))
}

func assertReportHasErrorContaining(t *testing.T, report *ValidationReport, needle string) {
	t.Helper()
	require.NotNil(t, report)
	for _, issue := range report.Issues {
		if issue.Level == IssueLevelError && strings.Contains(issue.Message, needle) {
			return
		}
		if issue.Level == IssueLevelError && strings.Contains(issue.Path, needle) {
			return
		}
	}
	t.Fatalf("expected error containing %q; got issues: %+v", needle, report.Issues)
}
