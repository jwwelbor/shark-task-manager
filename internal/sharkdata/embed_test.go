package sharkdata

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
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

	// All content-bundle subdirectories are present.
	for _, sub := range []string{"prompts", "skills", "agents", "workflow", "overrides", "file_templates"} {
		info, err := os.Stat(filepath.Join(dest, sub))
		require.NoError(t, err)
		assert.True(t, info.IsDir(), "%s should be a directory", sub)
	}

	for _, rel := range []string{"file_templates/epic.md", "file_templates/feature.md", "file_templates/task.md", "file_templates/sprint.md"} {
		content, err := os.ReadFile(filepath.Join(dest, filepath.FromSlash(rel)))
		require.NoError(t, err, "%s should be materialized by init", rel)
		assert.NotEmpty(t, content, "%s should not be empty", rel)
	}
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

func TestUpgrade_PreservesFileTemplateOverrides(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	overridePath := filepath.Join(root, SharkDataDirName, "overrides", "file_templates", "task.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(overridePath), 0755))
	overrideContent := []byte("USER TASK TEMPLATE OVERRIDE")
	require.NoError(t, os.WriteFile(overridePath, overrideContent, 0644))

	summary, err := Upgrade(root, false)
	require.NoError(t, err)
	require.NotNil(t, summary)

	got, err := os.ReadFile(overridePath)
	require.NoError(t, err)
	assert.Equal(t, overrideContent, got, "upgrade must not touch file template overrides")
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

func TestUpgrade_AddsMissingFileTemplates(t *testing.T) {
	root := t.TempDir()
	dest, err := Init(root)
	require.NoError(t, err)

	require.NoError(t, os.RemoveAll(filepath.Join(dest, "file_templates")))

	summary, err := Upgrade(root, false)
	require.NoError(t, err)
	require.NotNil(t, summary)
	assert.Contains(t, summary.Added, "file_templates/task.md")

	for _, rel := range []string{"file_templates/epic.md", "file_templates/feature.md", "file_templates/task.md", "file_templates/sprint.md"} {
		content, readErr := os.ReadFile(filepath.Join(dest, filepath.FromSlash(rel)))
		require.NoError(t, readErr, "%s should be restored by upgrade", rel)
		assert.NotEmpty(t, content, "%s should not be empty", rel)
	}
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

func TestValidate_CanonicalQuestionStatusLiteralAllowedInInstructions(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	workflowPath := filepath.Join(root, SharkDataDirName, "workflow", "question.yaml")
	workflow, err := os.ReadFile(workflowPath)
	require.NoError(t, err)
	workflow = bytes.ReplaceAll(workflow, []byte("ready_for_resolution"), []byte("ready_for_question_owner"))
	require.NoError(t, os.WriteFile(workflowPath, workflow, 0644))

	skillPath := filepath.Join(root, SharkDataDirName, "skills", "quality", "SKILL.md")
	require.NoError(t, os.WriteFile(skillPath, []byte("Review Questions in ready_for_question_owner."), 0644))

	report, err := Validate(root)
	require.NoError(t, err)
	for _, issue := range report.Issues {
		assert.NotContains(t, issue.Message, "ready_for_question_owner")
	}
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

// TestE34F02DemoScriptBundle_TC003_TC004_TC005_TC006_TC007 guards the
// content-only contract for the portable demo-script bundle. It reads the
// embedded delivery surface rather than inventing a readiness runtime.
func TestE34F02DemoScriptBundle_TC003_TC004_TC005_TC006_TC007(t *testing.T) {
	files := map[string]string{
		"manifest": readEmbeddedString(t, "manifest.yaml"),
		"index":    readEmbeddedString(t, "skills/README.md"),
		"skill":    readEmbeddedString(t, "skills/demo-script/SKILL.md"),
		"template": readEmbeddedString(t, "skills/demo-script/context/demo-script-template.md"),
	}

	for name, want := range map[string][]string{
		"manifest": {
			"name: demo-script",
			"ownership: canonical",
		},
		"index": {
			"`demo-script`",
			"Portable, evidence-based demo scenario maps",
		},
		"skill": {
			"name: demo-script",
			"E34-interaction-map.md#i-01-readiness-evidence-shape",
			"TC-I-01-READINESS-SYMMETRY",
			"E34-F03",
			"E34-F02",
			"assessor_verdict",
			"owner_decision",
			"open_conditions",
			"gate_mode",
			"activation_owner",
			"closure_key",
			"counterpart_status",
			"review_basis",
			"demonstrability_disposition",
			"contract-only",
			"pending-integration",
			"override-accept",
			"and risk. Do not treat an override as demonstrated delivery.",
			"Demonstrated now",
			"Not demonstrated / pending integration",
			"Accepted risks and overrides",
			"Do not invent commands, credentials, deployments, endpoints, or proof.",
			"deduplication and user confirmation",
		},
		"template": {
			"Stakeholder value",
			"Source requirement or acceptance criterion",
			"Prerequisites and demo data",
			"Presenter actions",
			"Expected observable result",
			"Evidence type and path",
			"Evidence environment and date",
			"Acceptance/readiness classification",
			"Reset or recovery instructions",
			"Known limitations",
			"UI capture or recording",
			"CLI transcript",
			"API request/response plus resulting state",
			"SDK runnable example",
			"Pipeline artifact or data",
			"Infrastructure health or metric evidence",
			"Background trigger/log/result evidence",
		},
	} {
		t.Run(name, func(t *testing.T) {
			for _, expected := range want {
				assert.Contains(t, files[name], expected)
			}
		})
	}
	for _, forbidden := range []string{
		"shark question resolve", "shark question withdraw", "shark question supersede",
	} {
		assert.NotContains(t, files["skill"], forbidden)
	}
}

// TestE34F04QuestionManagementBundle_TC001_TC005 guards the embedded
// Question-management procedure and its content-only delivery boundary.
func TestE34F04QuestionManagementBundle_TC001_TC005(t *testing.T) {
	files := map[string]string{
		"manifest": readEmbeddedString(t, "manifest.yaml"),
		"index":    readEmbeddedString(t, "skills/README.md"),
		"skill":    readEmbeddedString(t, "skills/question-management/SKILL.md"),
	}

	for name, want := range map[string][]string{
		"manifest": {"name: question-management", "ownership: canonical"},
		"index":    {"`question-management`", "durable Question lifecycle"},
		"skill": {
			"name: question-management",
			"materiality test",
			"non-material",
			"deduplication",
			"Search Questions for the decision phrase",
			"shark question create",
			"shark question configure-workflow",
			"shark related-docs add",
			"shark link Q### <entity-key>",
			"--type=question_blocks",
			"before adding a block",
			"Shark Attack",
			"solution-walkthrough",
			"owns Question creation",
			"local_clarification",
			"feature_change",
			"product_decision",
			"architecture_decision",
			"follow_up_work",
			"no_lasting_consequence",
			"Do not put credentials",
			"rendered prompts",
			"full transcripts",
			"unbounded chat history",
			"shark claim Q### --by=<current-responder> --json",
			"shark question respond Q### --session=<session-id>",
			"--responder=<current-responder>",
			"--summary=\"<bounded response>\"",
			"--evidence-pointer=<durable-record-path>",
			"shark release Q### --session=<session-id>",
			"shark question resolve Q### --owner=<resolution-owner>",
			"--resolution-kind=<resolution-kind>",
			"--resolution-pointer=<durable-record-path>",
			"--resolution-kind=no_lasting_consequence",
		},
	} {
		t.Run(name, func(t *testing.T) {
			for _, expected := range want {
				assert.Contains(t, files[name], expected)
			}
		})
	}

	assert.Contains(t, files["skill"], "| `no_lasting_consequence` | No pointer")
	assert.Contains(t, files["skill"], "For `no_lasting_consequence`, omit `--resolution-pointer`")
	assert.NotContains(t, files["skill"], "--resolution-kind=no_lasting_consequence \\\n+     --resolution-pointer=")
}

func TestSolutionWalkthroughBundle(t *testing.T) {
	files := map[string]string{
		"manifest": readEmbeddedString(t, "manifest.yaml"),
		"index":    readEmbeddedString(t, "skills/README.md"),
		"skill":    readEmbeddedString(t, "skills/solution-walkthrough/SKILL.md"),
		"template": readEmbeddedString(t, "skills/solution-walkthrough/context/decision-record-template.md"),
	}

	for name, want := range map[string][]string{
		"manifest": {"name: solution-walkthrough", "ownership: canonical"},
		"index":    {"`solution-walkthrough`", "decision-by-decision solution walkthroughs"},
		"skill": {
			"name: solution-walkthrough",
			"authoritative project document",
			"outstanding Question entities",
			"ready_for_resolution",
			"increasing `--offset` until a page is short",
			"reviewed-but-out-of-scope Questions",
			"shark next <question-key>",
			"current_responder",
			"shark question respond <question-key>",
			"shark claim",
			"--by=<current-responder> --json",
			"returned",
			"Release the claim afterward",
			"never infer or impersonate a responder",
			"docs/product/progress.md",
			"docs/architecture/adr/ADR-<next-number>-<slug>.md",
			"entity-local `decisions.md`",
			"one decision at a time",
			"Record only after a response.",
			"Reviewed and confirmed",
			"reference` note",
			"no lifecycle state",
		},
		"template": {"# Decision Record — <entity key>", "## Context", "## Decision", "## Rationale and alternatives", "## Consequences and follow-through", "## Evidence and references"},
	} {
		t.Run(name, func(t *testing.T) {
			for _, expected := range want {
				assert.Contains(t, files[name], expected)
			}
		})
	}
}

// TestDefectClassSweepBundleIndexRegistration verifies the E34-F06 workflow
// file (defect-class-sweep.md) is named in both discovery indices: the
// quality skill's own "Workflow Selection" section and the top-level skills
// README's quality row file list (test-plan.md TC-003).
func TestDefectClassSweepBundleIndexRegistration(t *testing.T) {
	skill := readEmbeddedString(t, "skills/quality/SKILL.md")
	assert.Contains(t, skill, "workflows/defect-class-sweep.md",
		"quality SKILL.md Workflow Selection must name the new workflow file")

	readme := readEmbeddedString(t, "skills/README.md")
	assert.Contains(t, readme, "workflows/defect-class-sweep.md",
		"skills/README.md quality row must name the new workflow file")
}

// TestDefectClassSweepConsolidatedNotDuplicated is test-plan.md TC-002: the
// three gate call sites (code_review.md, approval.md, redteam-rubric.md)
// must reference the canonical defect-class-sweep.md workflow instead of
// restating its sweep procedure inline. This guards against the exact prose
// drift T-E34-F06-002 fixed re-creeping back in.
func TestDefectClassSweepConsolidatedNotDuplicated(t *testing.T) {
	oldDuplicatedProse := []string{
		// The old inline kickback-reason template restated in both
		// code_review.md and approval.md before consolidation.
		"Before fixing the cited instance, sweep the touched module(s) for every other instance of this defect class",
		// The old three-part re-verification list restated in
		// redteam-rubric.md before consolidation.
		"Full-rubric sanity pass** — re-run the verification checks above over the feature surface",
	}

	files := []string{
		"prompts/feature/code_review.md",
		"prompts/feature/approval.md",
		"skills/uat/references/redteam-rubric.md",
	}

	for _, rel := range files {
		content := readEmbeddedString(t, rel)

		for _, prose := range oldDuplicatedProse {
			assert.NotContains(t, content, prose,
				"%s must not restate the old duplicated sweep prose %q", rel, prose)
		}

		assert.Contains(t, content, "skills/quality/workflows/defect-class-sweep.md",
			"%s must reference the canonical defect-class-sweep.md workflow", rel)
	}
}

// TestDefectClassSweepNoWWGMOrToolLeakage is test-plan.md TC-004: the new
// workflow file and the three edited call sites must not leak WWGM defect
// names, Python tooling references, or local filesystem paths — the content
// must stay project-agnostic and render from `{{project guidance}}` inputs.
func TestDefectClassSweepNoWWGMOrToolLeakage(t *testing.T) {
	leakPattern := regexp.MustCompile(`(?i)WWGM|\.py\b|/home/|/Users/`)

	files := []string{
		"skills/quality/workflows/defect-class-sweep.md",
		"prompts/feature/code_review.md",
		"prompts/feature/approval.md",
		"skills/uat/references/redteam-rubric.md",
		"prompts/feature/qa.md",
		"prompts/task/development.md",
		"skills/uat/SKILL.md",
	}

	for _, rel := range files {
		content := readEmbeddedString(t, rel)
		if match := leakPattern.FindString(content); match != "" {
			t.Errorf("%s contains leaked WWGM/tool/path reference: %q", rel, match)
		}
	}
}

// TestDefectClassSweepNoGoPersistenceIntroduced is test-plan.md TC-010
// (task T-E34-F06-003, AC-T2): confirms REQ-NF-001 — this feature stayed
// content-only and did not introduce a Go persistence layer (a new type,
// struct field, table, or repository method) for the I-03 DefectClassSweep
// shape. Equivalent to `grep -rln "DefectClassSweep\|class_key" internal/`
// restricted to non-test Go source; any match there signals scope creep into
// a real Go persistence layer.
//
// Test files are excluded from the scan by design, not oversight: TC-001/
// TC-002's own tests (`internal/templates/includes_test.go`'s
// TestIncludeResolverWithEmbed_DefectClassSweepRenders and this file's
// TestDefectClassSweep* functions, added by T-E34-F06-001/002) legitimately
// name the workflow in Go identifiers and comments while verifying it
// renders/is referenced — that is expected test coverage, not a persistence
// layer, and matching REQ-NF-001's "signals scope creep into a Go
// persistence layer" against test-only identifiers would be a false
// positive on the very tests this task's own scope requires.
func TestDefectClassSweepNoGoPersistenceIntroduced(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller(0) failed; cannot locate repo root")
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	internalDir := filepath.Join(repoRoot, "internal")

	pattern := regexp.MustCompile(`DefectClassSweep|class_key`)
	var hits []string

	err := filepath.WalkDir(internalDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if pattern.Match(data) {
			rel, relErr := filepath.Rel(repoRoot, path)
			if relErr != nil {
				rel = path
			}
			hits = append(hits, rel)
		}
		return nil
	})
	require.NoError(t, err)

	assert.Empty(t, hits,
		"no non-test Go source under internal/ should reference DefectClassSweep or class_key (found: %v) — REQ-NF-001 requires this feature to stay content-only", hits)
}

// TestDefectClassSweepQAAndDevelopmentPromptsReference is the UAT-kickback
// (HIGH-1) regression guard: REQ-F-001 requires review, QA, UAT, AND
// development prompts to reference the canonical defect-class-sweep.md
// workflow rather than duplicate its procedure. T-E34-F06-002 only wired
// code_review.md/approval.md/redteam-rubric.md; this asserts the QA
// (feature/qa.md) and development (task/development.md) prompts, plus
// uat/SKILL.md's own rejection-routing prose, also reference it.
func TestDefectClassSweepQAAndDevelopmentPromptsReference(t *testing.T) {
	files := []string{
		"prompts/feature/qa.md",
		"prompts/task/development.md",
		"skills/uat/SKILL.md",
	}

	for _, rel := range files {
		content := readEmbeddedString(t, rel)
		assert.Contains(t, content, "skills/quality/workflows/defect-class-sweep.md",
			"%s must reference the canonical defect-class-sweep.md workflow", rel)
	}
}

// TestDefectClassSweepCallingGateIncludesQA is the UAT-kickback (HIGH-1)
// regression guard for the workflow's own `calling_gate` input enum: it must
// name `qa` alongside `code_review | approval | uat_redteam` so a QA-gate
// kickback is a valid caller, and the "When to invoke" section must name a
// QA gate as a trigger.
func TestDefectClassSweepCallingGateIncludesQA(t *testing.T) {
	content := readEmbeddedString(t, "skills/quality/workflows/defect-class-sweep.md")
	assert.Contains(t, content, "code_review | approval | uat_redteam | qa",
		"calling_gate enum must include qa")
	assert.Contains(t, content, "code-review, QA, or approval gate",
		"When to invoke section must name QA as a trigger gate")
}

// normalizeWhitespace collapses all runs of whitespace (including newlines
// introduced by markdown line-wrapping) to a single space, so a test
// assertion doesn't depend on exact line-wrap columns in prose files.
func normalizeWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// TestDefectClassSweepGuardCounterfactualDirectionCorrect is the UAT-kickback
// (HIGH-2) regression guard: the "Structural guard closure" counterfactual
// must require the guard to CATCH the reintroduced defect (not miss it) and
// not false-positive when the defect is absent — the inverse of the
// originally-shipped (backwards) wording.
func TestDefectClassSweepGuardCounterfactualDirectionCorrect(t *testing.T) {
	content := normalizeWhitespace(readEmbeddedString(t, "skills/quality/workflows/defect-class-sweep.md"))

	assert.NotContains(t, content,
		normalizeWhitespace("it fails to detect the class when the defect is deliberately re-introduced"),
		"the guard counterfactual must not require the guard to MISS the reintroduced defect")

	assert.Contains(t, content,
		normalizeWhitespace("it catches (flags, fails the build, or otherwise blocks) the class when the defect is deliberately re-introduced"),
		"the guard counterfactual must require the guard to CATCH the reintroduced defect")
}

// TestDefectClassSweepRecurrenceRequiresClassKey is the UAT-kickback (HIGH-3)
// regression guard: REQ-F-005 requires a new fingerprint to be classified as
// recurrence only when it shares the same class_key as a previously
// completed sweep AND lies inside that sweep's search_scope — not scope
// membership alone.
func TestDefectClassSweepRecurrenceRequiresClassKey(t *testing.T) {
	content := normalizeWhitespace(readEmbeddedString(t, "skills/quality/workflows/defect-class-sweep.md"))

	assert.NotContains(t, content,
		normalizeWhitespace("a new fingerprint appears inside a previously `status: complete` class's `search_scope`"),
		"recurrence classification must not accept scope membership alone (missing class_key discriminator)")

	assert.Contains(t, content, normalizeWhitespace("same `class_key`** as a previously"),
		"recurrence classification must require the same class_key")
	assert.Contains(t, content, "Both conjuncts are required",
		"recurrence classification must state both class_key and scope membership are required")
}

// TestDefectClassSweepSeverityConflictAtFindingLevel is the UAT-kickback
// (HIGH-4) regression guard: per architecture.md's I-02 GateResult schema,
// `severity_conflict` is an outer `GateResult.Finding.disposition` value, not
// an I-03 instance disposition. The sweep's own instances[].disposition must
// stay within {fixed, dispositioned, open} so
// fixed_count+dispositioned_count+open_count=matching_count keeps
// reconciling.
func TestDefectClassSweepSeverityConflictAtFindingLevel(t *testing.T) {
	content := normalizeWhitespace(readEmbeddedString(t, "skills/quality/workflows/defect-class-sweep.md"))

	assert.NotContains(t, content, "Mark the instance's disposition `severity_conflict`",
		"severity_conflict must not be assigned as an I-03 instance disposition")

	assert.Contains(t, content, "outer `GateResult.Finding.disposition` value",
		"the workflow must state severity_conflict belongs to the outer Finding schema")
	assert.Contains(t, content, normalizeWhitespace("instances[].disposition` stays within `{fixed, dispositioned, open}`"),
		"the workflow must keep the I-03 instance disposition enum closed to fixed/dispositioned/open")
}

func readEmbeddedString(t *testing.T, rel string) string {
	t.Helper()
	data, err := readEmbeddedAll(rel)
	require.NoError(t, err, "%s should be embedded", rel)
	return string(data)
}

func TestEmbedded_AllExpectedDirectoriesPresent(t *testing.T) {
	paths, err := CopyEmbeddedTreeForTest()
	require.NoError(t, err)

	// Every top-level directory must be represented either by real content
	// or by a .gitkeep placeholder. Real content is preferred (means F4
	// populated the directory); .gitkeep is fallback during the F3 skeleton
	// phase.
	requiredDirs := []string{"README.md", "prompts/", "skills/", "agents/", "workflow/", "overrides/", "file_templates/"}
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

// scanEmbeddedMarkdown walks canonical markdown files under prefix once and
// lets each bundle-purity gate declare only its content-specific check.
func scanEmbeddedMarkdown(t *testing.T, prefix string, scan func(relPath, content string, violations *[]string), skip func(string) bool) []string {
	t.Helper()
	var violations []string
	err := walkEmbedded(func(relPath string, data []byte, isDir bool) error {
		if isDir || !strings.HasPrefix(relPath, prefix) || !strings.HasSuffix(relPath, ".md") || (skip != nil && skip(relPath)) {
			return nil
		}
		scan(relPath, string(data), &violations)
		return nil
	})
	require.NoError(t, err)
	return violations
}

// TC-090: TestEmbedded_SkillsContainNoBareSharkCLIRefs enforces the E32-F09
// skill-purity rule for the embedded craft-skill set that the feature owns.
// Workflow and orchestration skills intentionally retain CLI instructions and
// are outside this feature's scope. The retired triage skill is also absent
// from the canonical bundle, so it is not a scan target.
//
// This turns the one-time acceptance-criterion grep into a permanent regression
// gate: if a future edit re-introduces a CLI ref into a canonical skill file,
// `make test` will fail here before the change can merge.
func TestEmbedded_SkillsContainNoBareSharkCLIRefs(t *testing.T) {
	const extractedDir = "/_extracted/"
	craftSkillPrefixes := []string{
		"skills/specification-writing/",
		"skills/uat/",
		"skills/assessment/",
		"skills/research/",
		"skills/quality/",
	}

	violations := scanEmbeddedMarkdown(t, "skills/", func(relPath, content string, violations *[]string) {
		inScope := false
		for _, prefix := range craftSkillPrefixes {
			if strings.HasPrefix(relPath, prefix) {
				inScope = true
				break
			}
		}
		if !inScope {
			return
		}
		*violations = append(*violations, craftSkillPlatformRefs(relPath, content)...)
	}, func(relPath string) bool { return strings.Contains(relPath, extractedDir) })

	assert.Empty(t, violations,
		"E32-F09 craft skill files must not contain bare shark CLI invocations; found:\n%s",
		strings.Join(violations, "\n"))
}

var craftSkillPlatformRef = regexp.MustCompile(`(?i)(?:/shark-rider\b|shark)`)

// craftSkillPlatformRefs enforces the feature policy rather than maintaining a
// brittle list of current CLI subcommands. Any platform name or command form in
// an owned craft skill is a violation; workflow/orchestration skills are scoped
// out by the caller.
func craftSkillPlatformRefs(relPath, content string) []string {
	match := craftSkillPlatformRef.FindString(content)
	if match == "" {
		return nil
	}
	return []string{relPath + `: contains platform-specific reference "` + match + `"`}
}

// TC-091: TestEmbedded_SkillPurityGateDetectsPlatformCommandFamilies verifies
// that the regression gate covers complete platform-command families.
func TestEmbedded_SkillPurityGateDetectsPlatformCommandFamilies(t *testing.T) {
	assert.Equal(t,
		[]string{`skills/research/SKILL.md: contains platform-specific reference "shark"`},
		craftSkillPlatformRefs("skills/research/SKILL.md", "Run shark related-docs E32 before returning."))
	assert.Equal(t,
		[]string{`skills/research/SKILL.md: contains platform-specific reference "shark"`},
		craftSkillPlatformRefs("skills/research/SKILL.md", "Run shark sprint E32 before returning."))
	assert.Equal(t,
		[]string{`skills/research/SKILL.md: contains platform-specific reference "shark"`},
		craftSkillPlatformRefs("skills/research/SKILL.md", "Read .sharkconfig.json before returning."))
	assert.Equal(t,
		[]string{`skills/research/SKILL.md: contains platform-specific reference "/shark-rider"`},
		craftSkillPlatformRefs("skills/research/SKILL.md", "Use /shark-rider project bootstrap."))
}

// TC-094: TestEmbedded_OutcomeReturningCraftSkillsDeclareThreeWayOutcome
// verifies the host-routable assessment and quality outcome contract.
func TestEmbedded_OutcomeReturningCraftSkillsDeclareThreeWayOutcome(t *testing.T) {
	for _, relPath := range []string{
		"skills/assessment/SKILL.md",
		"skills/quality/SKILL.md",
	} {
		content, err := readEmbeddedAll(relPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "outcome: pass | fail | blocked", relPath)
	}
}

// TestEmbedded_AgentsDescribeRoleNotWorkflow enforces the agent-persona purity
// rule: an agent .md file describes WHO the agent is (role identity, judgment
// posture, communication style), not HOW to run the workflow. Workflow
// mechanics belong in workflow YAMLs and prompts; craft methodology belongs in
// skills.
//
// This is a permanent regression gate: if a future edit re-introduces a status-
// management block, a stale harness path, or a workflow-node / skills-to-use /
// workflow-integration section into an agent persona, `make test` fails here
// before the change can merge.
func TestEmbedded_AgentsDescribeRoleNotWorkflow(t *testing.T) {
	const agentsPrefix = "agents/"

	// Substrings banned in EVERY agent file.
	banned := []string{
		"Shark Status Management",   // the CRITICAL status-management block
		"shark/SKILL.md",            // stale harness entry-point path
		"docs/workflow/state.json",  // runtime state file removed when shark became the state source
		"Workflow Nodes You Handle", // workflow-node tables belong in workflow YAMLs
		"Skills to Use",             // skill-listing sections belong in skills/prompts
		"Workflow Integration",      // workflow-time wiring belongs in prompts
		"/docs/tasks/created",       // stale task-lifecycle directory convention
		"docs/chatGPT",              // host-only path, never embedded
	}

	violations := scanEmbeddedMarkdown(t, agentsPrefix, func(relPath, content string, violations *[]string) {
		for _, b := range banned {
			if strings.Contains(content, b) {
				*violations = append(*violations, relPath+`: contains "`+b+`"`)
			}
		}
		// `shark status advance` is a workflow-mechanics instruction that
		// specialist agents must not carry. product-manager is the coordinator
		// exception (it may retain coordinator-level shark awareness).
		if relPath != agentsPrefix+"product-manager.md" && strings.Contains(content, "shark status advance") {
			*violations = append(*violations, relPath+`: contains "shark status advance" (only product-manager may)`)
		}
	}, nil)

	assert.Empty(t, violations,
		"agent personas must describe role identity, not workflow mechanics; found:\n%s",
		strings.Join(violations, "\n"))
}

// TestEmbedded_SkillsHaveNoStaleAgentSlugs enforces that skill files reference
// only current embedded agents. The specialized agents that predated
// consolidation (api-developer, frontend-developer, devops-engineer, and the
// per-domain architect personas) no longer exist; their work now lives in the
// `developer`, `devops`, and `architect` agents and the architecture skill's
// design-* workflows.
//
// `general-purpose` is a harness subagent type, not an embedded specialist, so
// it must not appear as an agent assignment. Plain prose ("a general-purpose
// choice") is fine — only slug usage (backtick-wrapped, or in an
// assigned_agent line) is flagged.
func TestEmbedded_SkillsHaveNoStaleAgentSlugs(t *testing.T) {
	const skillsPrefix = "skills/"

	staleSlugs := []string{
		"api-developer", "frontend-developer", "devops-engineer",
		"backend-architect", "frontend-architect", "db-admin",
		"security-architect", "principal-architect", "feature-architect",
	}

	violations := scanEmbeddedMarkdown(t, skillsPrefix, func(relPath, content string, violations *[]string) {
		for _, slug := range staleSlugs {
			if strings.Contains(content, slug) {
				*violations = append(*violations, relPath+`: contains stale agent slug "`+slug+`"`)
			}
		}
		// `general-purpose` only when used as an agent slug, not in prose.
		for _, line := range strings.Split(content, "\n") {
			if !strings.Contains(line, "general-purpose") {
				continue
			}
			if strings.Contains(line, "`general-purpose`") || strings.Contains(line, "assigned_agent") {
				*violations = append(*violations, relPath+`: uses "general-purpose" as an agent slug: `+strings.TrimSpace(line))
			}
		}
	}, nil)

	assert.Empty(t, violations,
		"skill files must reference current embedded agents (developer/devops/architect/qa), not consolidated-away slugs; found:\n%s",
		strings.Join(violations, "\n"))
}

// TestEmbedded_SkillsHaveNoDeadCollaborationLink guards against the
// `skills/collaboration/remembering-conversations` reference, which points at a
// capability that was never a tracked embedded path. The intended behavior is
// expressed as generic prose instead ("review available decision records and
// conversation history if the host exposes it").
func TestEmbedded_SkillsHaveNoDeadCollaborationLink(t *testing.T) {
	const skillsPrefix = "skills/"
	const deadLink = "skills/collaboration/remembering-conversations"

	violations := scanEmbeddedMarkdown(t, skillsPrefix, func(relPath, content string, violations *[]string) {
		if strings.Contains(content, deadLink) {
			*violations = append(*violations, relPath)
		}
	}, nil)

	assert.Empty(t, violations,
		"skill files must not reference the untracked collaboration path %q; found in:\n%s",
		deadLink, strings.Join(violations, "\n"))
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

func assertReportHasWarningContaining(t *testing.T, report *ValidationReport, needle string) {
	t.Helper()
	require.NotNil(t, report)
	for _, issue := range report.Issues {
		if issue.Level == IssueLevelWarning && strings.Contains(issue.Message, needle) {
			return
		}
		if issue.Level == IssueLevelWarning && strings.Contains(issue.Path, needle) {
			return
		}
	}
	t.Fatalf("expected warning containing %q; got issues: %+v", needle, report.Issues)
}

// ============================================================================
// New validator tests — Validators #1–#4
// ============================================================================

// TestValidate_FreshBundle_NoNewErrors is the regression guard: after Init,
// Validate must report NO errors from any of the four new validators. The
// cleaned bundle is expected to be structurally sound.
func TestValidate_FreshBundle_NoNewErrors(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	report, err := Validate(root)
	require.NoError(t, err)
	require.NotNil(t, report)
	for _, issue := range report.Issues {
		assert.NotEqual(t, IssueLevelError, issue.Level,
			"fresh init must produce zero error-level issues (new validators included); got: %+v", issue)
	}
}

// TestValidate_CrossEntityPromptPrefix verifies that a workflow referencing a
// prompt from another entity's namespace triggers an error.
func TestValidate_CrossEntityPromptPrefix(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	// Overwrite change.yaml to reference a bug/* prompt (cross-entity drift).
	workflowDir := filepath.Join(root, SharkDataDirName, "workflow")
	require.NoError(t, os.WriteFile(
		filepath.Join(workflowDir, "change.yaml"),
		[]byte(`version: "1.0"
start: draft
steps:
  draft:
    phase: planning
    action: advance_status
    prompt: bug/blocked.md
    outcomes:
      pass: completed
      fail: draft
      blocked: blocked
  completed:
    phase: done
    terminal: true
    prompt: change/completed.md
`),
		0644,
	))

	report, err := Validate(root)
	require.NoError(t, err)
	assertReportHasErrorContaining(t, report, "bug/blocked.md")
}

// TestValidate_CrossEntityPromptPrefix_SharedExempt verifies that a workflow
// referencing a _shared/ prompt is exempt from the cross-entity check.
func TestValidate_CrossEntityPromptPrefix_SharedExempt(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	// Write a task.yaml referencing _shared/qa.md — this is a legitimate
	// shared reference and must not produce a cross-entity error.
	workflowDir := filepath.Join(root, SharkDataDirName, "workflow")
	require.NoError(t, os.WriteFile(
		filepath.Join(workflowDir, "task.yaml"),
		[]byte(`version: "1.0"
start: draft
steps:
  draft:
    phase: planning
    action: advance_status
    prompt: task/draft.md
    outcomes:
      pass: qa
      fail: draft
      blocked: blocked
  qa:
    phase: qa
    action: spawn_agent
    agent: qa
    prompt: _shared/qa.md
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

	// Must not error on the _shared/qa.md reference.
	for _, issue := range report.Issues {
		if issue.Level == IssueLevelError && strings.Contains(issue.Message, "_shared/qa.md") {
			t.Errorf("_shared/qa.md should be exempt from cross-entity check; got: %+v", issue)
		}
	}
}

// TestValidate_HostLocalPath verifies that a file containing a host-local path
// token (e.g. /home/) is flagged as an error.
func TestValidate_HostLocalPath(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	// Write a host-local /home/ path into an agent file.
	agentPath := filepath.Join(root, SharkDataDirName, "agents", "local-agent.md")
	require.NoError(t, os.WriteFile(agentPath, []byte("See /home/developer/setup.sh for config.\n"), 0644))

	report, err := Validate(root)
	require.NoError(t, err)

	var found bool
	for _, issue := range report.Issues {
		if issue.Level == IssueLevelError && strings.Contains(issue.Message, "/home/") {
			found = true
			break
		}
	}
	assert.True(t, found, "host-local /home/ path should be flagged as error; got: %+v", report.Issues)
}

// TestValidate_HostLocalPath_DotfileToken verifies that ~/.claude triggers an error.
func TestValidate_HostLocalPath_DotfileToken(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	promptPath := filepath.Join(root, SharkDataDirName, "prompts", "task", "scratch.md")
	require.NoError(t, os.WriteFile(promptPath, []byte("Config lives at ~/.claude/settings.json\n"), 0644))

	report, err := Validate(root)
	require.NoError(t, err)

	var found bool
	for _, issue := range report.Issues {
		if issue.Level == IssueLevelError && strings.Contains(issue.Message, "~/.claude") {
			found = true
			break
		}
	}
	assert.True(t, found, "~/.claude path should be flagged as error; got: %+v", report.Issues)
}

// TestValidate_HostBinaryPath_RealPathFlagged_URLExempt verifies that an
// absolute path to a host binary (e.g. /usr/local/bin/codex) is flagged, while
// a URL whose path merely ends in /node or /codex (e.g. github.com/nodejs/node)
// is NOT a false positive.
func TestValidate_HostBinaryPath_RealPathFlagged_URLExempt(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	// Real absolute path to a host binary — must be flagged.
	binPath := filepath.Join(root, SharkDataDirName, "agents", "bin-agent.md")
	require.NoError(t, os.WriteFile(binPath, []byte("Run /usr/local/bin/codex to start.\n"), 0644))

	// URL ending in /node and /codex — must NOT be flagged.
	urlPath := filepath.Join(root, SharkDataDirName, "prompts", "task", "links.md")
	require.NoError(t, os.WriteFile(urlPath, []byte("See https://github.com/nodejs/node and https://example.com/codex\n"), 0644))

	report, err := Validate(root)
	require.NoError(t, err)

	assertReportHasErrorContaining(t, report, "/usr/local/bin/codex")
	for _, issue := range report.Issues {
		if strings.Contains(issue.Message, "host binary") &&
			(strings.Contains(issue.Message, "nodejs/node") || strings.Contains(issue.Path, "links.md")) {
			t.Fatalf("URL path should not be flagged as a host binary; got: %+v", issue)
		}
	}
}

// TestValidate_ManifestUnparseable_Flagged verifies that a present-but-invalid
// manifest.yaml surfaces as a validation error rather than being silently
// swallowed (which would let the cross-entity validators degrade to defaults
// without anyone noticing the manifest was broken).
func TestValidate_ManifestUnparseable_Flagged(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	manifestPath := filepath.Join(root, SharkDataDirName, "manifest.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte("prompt_namespaces: [unterminated\n"), 0644))

	report, err := Validate(root)
	require.NoError(t, err)

	assertReportHasErrorContaining(t, report, "manifest.yaml")
}

// TestValidate_UnreferencedPrompt verifies that a prompt file with no
// corresponding workflow reference is reported as a warning.
func TestValidate_UnreferencedPrompt(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	// Add a prompt file not referenced by any workflow YAML.
	orphanPath := filepath.Join(root, SharkDataDirName, "prompts", "bug", "orphan_xyz.md")
	require.NoError(t, os.WriteFile(orphanPath, []byte("# Orphan prompt\nThis is never dispatched.\n"), 0644))

	report, err := Validate(root)
	require.NoError(t, err)

	assertReportHasWarningContaining(t, report, "orphan_xyz.md")
	assert.False(t, report.HasErrors(), "unreferenced prompt must produce a warning, not an error")
}

// TestValidate_UnreferencedPrompt_PartialsExempt verifies that files under
// _partials/ are not flagged as unreferenced even when no workflow references them.
func TestValidate_UnreferencedPrompt_PartialsExempt(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	// Add a file under _partials/ — these are template fragments, never
	// dispatched directly.
	partialPath := filepath.Join(root, SharkDataDirName, "prompts", "_partials", "_custom_fragment.md")
	require.NoError(t, os.WriteFile(partialPath, []byte("{{define \"custom\"}}partial content{{end}}\n"), 0644))

	report, err := Validate(root)
	require.NoError(t, err)

	for _, issue := range report.Issues {
		assert.NotContains(t, issue.Path, "_custom_fragment.md",
			"_partials/ files must be exempt from the unreferenced-prompt check; got: %+v", issue)
	}
}

// TestValidate_SkillFrontmatterLegacyKey verifies that a skill using the
// removed skill_name: key triggers an error.
func TestValidate_SkillFrontmatterLegacyKey(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	skillPath := filepath.Join(root, SharkDataDirName, "skills", "implementation", "SKILL.md")
	require.NoError(t, os.WriteFile(skillPath,
		[]byte("---\nskill_name: implementation\ndescription: Replaces name with legacy key.\n---\n# Implementation\n"),
		0644,
	))

	report, err := Validate(root)
	require.NoError(t, err)
	assertReportHasErrorContaining(t, report, "skill_name")
}

// TestValidate_SkillFrontmatterWrongSlug verifies that a skill whose name:
// field does not match the directory basename triggers an error.
func TestValidate_SkillFrontmatterWrongSlug(t *testing.T) {
	root := t.TempDir()
	_, err := Init(root)
	require.NoError(t, err)

	skillPath := filepath.Join(root, SharkDataDirName, "skills", "implementation", "SKILL.md")
	require.NoError(t, os.WriteFile(skillPath,
		[]byte("---\nname: wrong-skill-name\ndescription: Mismatched slug.\n---\n# Implementation\n"),
		0644,
	))

	report, err := Validate(root)
	require.NoError(t, err)
	assertReportHasErrorContaining(t, report, "wrong-skill-name")
}
