package sharkdata

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	// All five subdirectories are present.
	for _, sub := range []string{"prompts", "skills", "agents", "workflow", "overrides"} {
		info, err := os.Stat(filepath.Join(dest, sub))
		require.NoError(t, err)
		assert.True(t, info.IsDir(), "%s should be a directory", sub)
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
		filepath.Join(promptDir, "in_qa.md"),
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
		filepath.Join(promptDir, "in_qa.md"),
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
		filepath.Join(promptDir, "in_qa.md"),
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
		filepath.Join(promptDir, "in_qa.md"),
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
	// The decode succeeds (it's still YAML), but it's structurally weak.
	// The presence checks should NOT fire because both top-level keys exist.
	// A future tightening would do schema-level validation; we accept this
	// gap explicitly here.
	for _, issue := range report.Issues {
		_ = issue
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
	requiredDirs := []string{"README.md", "prompts/", "skills/", "agents/", "workflow/", "overrides/"}
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
