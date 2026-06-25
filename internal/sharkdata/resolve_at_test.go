package sharkdata

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// InitAt / UpgradeAt / ValidateAt are the resolution-aware entry points used
// when shark_data_path selects a non-default (or absolute) bundle root. These
// tests exercise those entry points directly against an explicit dest that is
// NOT <projectRoot>/shark-data, which is the case the CLI hits for a custom
// shark_data_path.

func TestInitAt_MaterializesAtExplicitRoot(t *testing.T) {
	// dest deliberately a custom name, not "shark-data".
	dest := filepath.Join(t.TempDir(), "custom-bundle")

	got, err := InitAt(dest)
	require.NoError(t, err)
	require.Equal(t, dest, got)

	for _, sub := range []string{"prompts", "skills", "agents", "workflow", "overrides"} {
		info, statErr := os.Stat(filepath.Join(dest, sub))
		require.NoError(t, statErr)
		assert.True(t, info.IsDir(), "%s should be a directory", sub)
	}
}

func TestInitAt_AlreadyInitialized(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "custom-bundle")
	_, err := InitAt(dest)
	require.NoError(t, err)

	got, err := InitAt(dest)
	require.ErrorIs(t, err, ErrAlreadyInitialized)
	assert.Equal(t, dest, got)
}

func TestUpgradeAt_RefreshesExplicitRoot(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "custom-bundle")
	_, err := InitAt(dest)
	require.NoError(t, err)

	summary, err := UpgradeAt(dest, true) // dry-run
	require.NoError(t, err)
	require.NotNil(t, summary)
	// A freshly-initialized bundle should report nothing to add and should
	// always skip the overrides/ subtree.
	assert.Empty(t, summary.Added, "fresh bundle has no files to add on upgrade")
	assert.NotEmpty(t, summary.SkippedOverrides, "overrides/ must be skipped")
}

func TestUpgradeAt_RequiresExistingRoot(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "missing-bundle")
	_, err := UpgradeAt(dest, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestValidateAt_ResolvedRoot(t *testing.T) {
	t.Run("valid bundle has no error-level issues", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "custom-bundle")
		_, err := InitAt(dest)
		require.NoError(t, err)

		report, err := ValidateAt(dest)
		require.NoError(t, err)
		assert.Equal(t, dest, report.Path)
		assert.False(t, report.HasErrors(), "freshly-initialized bundle should validate clean: %+v", report.Issues)
	})

	t.Run("missing root is an error-level issue", func(t *testing.T) {
		report, err := ValidateAt(filepath.Join(t.TempDir(), "nope"))
		require.NoError(t, err)
		assert.True(t, report.HasErrors(), "missing bundle root must be flagged")
	})

	t.Run("file (not dir) is an error-level issue", func(t *testing.T) {
		f := filepath.Join(t.TempDir(), "afile")
		require.NoError(t, os.WriteFile(f, []byte("x"), 0644))

		report, err := ValidateAt(f)
		require.NoError(t, err)
		assert.True(t, report.HasErrors(), "a regular file as bundle root must be flagged")
	})
}
