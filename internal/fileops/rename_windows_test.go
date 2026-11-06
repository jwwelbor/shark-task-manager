//go:build windows

package fileops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeWindowsPathMatchesGoLongPathHandling(t *testing.T) {
	t.Run("short relative path stays relative", func(t *testing.T) {
		path := filepath.Join("docs", "plan", "task.md")
		got, err := normalizeWindowsPath(path, false)
		require.NoError(t, err)
		require.Equal(t, filepath.Clean(path), got)
	})

	t.Run("long drive path receives extended prefix", func(t *testing.T) {
		path := filepath.Join(`C:\project`, strings.Repeat(`segment\`, 32), "task.md")
		got, err := normalizeWindowsPath(path, false)
		require.NoError(t, err)
		require.True(t, strings.HasPrefix(got, `\\?\`), "long paths need Go-compatible extended-length syntax")
		require.Equal(t, filepath.Clean(path), strings.TrimPrefix(got, `\\?\`))
	})

	t.Run("long UNC path receives UNC extended prefix", func(t *testing.T) {
		path := `\\server\share\` + strings.Repeat(`segment\`, 32) + "task.md"
		got, err := normalizeWindowsPath(path, false)
		require.NoError(t, err)
		require.True(t, strings.HasPrefix(got, `\\?\UNC\`), "long UNC paths need the extended UNC prefix")
		require.Equal(t, strings.TrimPrefix(filepath.Clean(path), `\\`), strings.TrimPrefix(got, `\\?\UNC\`))
	})

	t.Run("already extended path remains unchanged", func(t *testing.T) {
		path := `\\?\C:\project\task.md`
		got, err := normalizeWindowsPath(path, false)
		require.NoError(t, err)
		require.Equal(t, path, got)
	})

	t.Run("long device path remains unchanged", func(t *testing.T) {
		path := `\\.\pipe\` + strings.Repeat(`segment\`, 32) + "task.md"
		got, err := normalizeWindowsPath(path, false)
		require.NoError(t, err)
		require.Equal(t, path, got)
	})

	t.Run("enabled long-path capability keeps path unmodified", func(t *testing.T) {
		path := filepath.Join(`C:\project`, strings.Repeat(`segment\`, 32), "task.md")
		got, err := normalizeWindowsPath(path, true)
		require.NoError(t, err)
		require.Equal(t, path, got)
	})
}

func TestRenameNoReplacePublishesLongLocalPath(t *testing.T) {
	root := t.TempDir()
	deepDirectory := filepath.Join(root, strings.Repeat(`longdir\`, 32))
	stagedPath := filepath.Join(deepDirectory, "staged.md")
	targetPath := filepath.Join(deepDirectory, "target.md")
	stagedContent := []byte("published through the Windows native adapter")
	require.Greater(t, len(targetPath), windowsLegacyPathLimit, "test path must exercise Windows long-path normalization")
	require.NoError(t, os.MkdirAll(deepDirectory, 0755))
	require.NoError(t, os.WriteFile(stagedPath, stagedContent, 0600))

	err := renameNoReplace(stagedPath, targetPath)
	require.NoError(t, err)

	actualTarget, readErr := os.ReadFile(targetPath)
	require.NoError(t, readErr)
	require.Equal(t, stagedContent, actualTarget)
	_, statErr := os.Lstat(stagedPath)
	require.ErrorIs(t, statErr, os.ErrNotExist, "native publication must move the staged file without using the link fallback")
}
