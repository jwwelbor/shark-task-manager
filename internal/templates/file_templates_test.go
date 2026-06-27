package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFileTemplate_UsesDiskBeforeEmbedded(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "shark-data", "file_templates"), 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(root, "shark-data", "file_templates", "feature.md"),
		[]byte("CUSTOM FEATURE {{.Title}}"),
		0644,
	))

	restoreTemplateGlobals(t)
	SetConfiguredTemplateDir("")
	SetConfiguredSharkDataPath("shark-data")
	chdir(t, root)

	content, err := ReadFileTemplate("feature.md")
	require.NoError(t, err)
	assert.Equal(t, "CUSTOM FEATURE {{.Title}}", string(content))
}

func TestReadFileTemplate_UsesOverrideBeforeDisk(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "shark-data", "file_templates"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "shark-data", "overrides", "file_templates"), 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(root, "shark-data", "file_templates", "task.md"),
		[]byte("DISK TASK {{.Title}}"),
		0644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(root, "shark-data", "overrides", "file_templates", "task.md"),
		[]byte("OVERRIDE TASK {{.Title}}"),
		0644,
	))

	restoreTemplateGlobals(t)
	SetConfiguredTemplateDir("")
	SetConfiguredSharkDataPath("shark-data")
	chdir(t, filepath.Join(root))

	content, err := ReadFileTemplate("task.md")
	require.NoError(t, err)
	assert.Equal(t, "OVERRIDE TASK {{.Title}}", string(content))
}

func TestReadFileTemplate_FallsBackToEmbeddedFileTemplates(t *testing.T) {
	root := t.TempDir()

	restoreTemplateGlobals(t)
	SetConfiguredTemplateDir("")
	SetConfiguredSharkDataPath("shark-data")
	chdir(t, root)

	content, err := ReadFileTemplate("epic.md")
	require.NoError(t, err)
	text := string(content)
	assert.Contains(t, text, "Epic Key")
	assert.Contains(t, text, "{{.Title}}")
}

func TestReadFileTemplate_RejectsTraversal(t *testing.T) {
	_, err := ReadFileTemplate("../epic.md")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be relative")
}

func TestTaskLoader_UsesConsolidatedFileTemplate(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "custom-data", "file_templates"), 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(root, "custom-data", "file_templates", "task.md"),
		[]byte("CUSTOM TASK {{.Title}}"),
		0644,
	))

	restoreTemplateGlobals(t)
	SetConfiguredTemplateDir("")
	SetConfiguredSharkDataPath("custom-data")
	chdir(t, root)

	loader := NewLoader("")
	content, err := loader.LoadTemplate("backend")
	require.NoError(t, err)
	assert.Equal(t, "CUSTOM TASK {{.Title}}", strings.TrimSpace(content))
}

func restoreTemplateGlobals(t *testing.T) {
	t.Helper()
	oldTemplateDir := configuredTemplateDir
	oldSharkDataPath := configuredSharkDataPath
	t.Cleanup(func() {
		configuredTemplateDir = oldTemplateDir
		configuredSharkDataPath = oldSharkDataPath
	})
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	oldWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})
}
