package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetConfigPath tests the GetConfigPath function in various scenarios
func TestGetConfigPath(t *testing.T) {
	// Save and restore original working directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalWd)
	}()

	t.Run("finds config in project root when run from root", func(t *testing.T) {
		// Create temp directory structure
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, ".sharkconfig.json")
		require.NoError(t, os.WriteFile(configFile, []byte("{}"), 0644))

		// Change to project root
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(originalWd) }()

		// Get config path
		configPath, err := GetConfigPath()
		require.NoError(t, err)

		// Should return absolute path to config
		assert.Equal(t, configFile, configPath)
	})

	t.Run("finds config in project root when run from subdirectory", func(t *testing.T) {
		// Create temp directory structure
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, ".sharkconfig.json")
		require.NoError(t, os.WriteFile(configFile, []byte("{}"), 0644))

		// Create nested subdirectories
		subDir := filepath.Join(tmpDir, "docs", "plan", "E07-test")
		require.NoError(t, os.MkdirAll(subDir, 0755))

		// Change to subdirectory
		require.NoError(t, os.Chdir(subDir))
		defer func() { _ = os.Chdir(originalWd) }()

		// Get config path - should find config in tmpDir (project root)
		configPath, err := GetConfigPath()
		require.NoError(t, err)

		// Should return absolute path to config in project root
		assert.Equal(t, configFile, configPath)
	})

	t.Run("finds config in project root when run from deeply nested subdirectory", func(t *testing.T) {
		// Create temp directory structure
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, ".sharkconfig.json")
		require.NoError(t, os.WriteFile(configFile, []byte("{}"), 0644))

		// Create deeply nested subdirectories
		deepDir := filepath.Join(tmpDir, "a", "b", "c", "d", "e", "f")
		require.NoError(t, os.MkdirAll(deepDir, 0755))

		// Change to deep subdirectory
		require.NoError(t, os.Chdir(deepDir))
		defer func() { _ = os.Chdir(originalWd) }()

		// Get config path - should find config in tmpDir (project root)
		configPath, err := GetConfigPath()
		require.NoError(t, err)

		// Should return absolute path to config in project root
		assert.Equal(t, configFile, configPath)
	})

	t.Run("prefers .sharkconfig.json over .git directory", func(t *testing.T) {
		// Create temp directory with both .sharkconfig.json and .git
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, ".sharkconfig.json")
		require.NoError(t, os.WriteFile(configFile, []byte("{}"), 0644))

		gitDir := filepath.Join(tmpDir, ".git")
		require.NoError(t, os.MkdirAll(gitDir, 0755))

		// Create subdirectory
		subDir := filepath.Join(tmpDir, "subdir")
		require.NoError(t, os.MkdirAll(subDir, 0755))

		// Change to subdirectory
		require.NoError(t, os.Chdir(subDir))
		defer func() { _ = os.Chdir(originalWd) }()

		// Get config path
		configPath, err := GetConfigPath()
		require.NoError(t, err)

		// Should return config in tmpDir (where .sharkconfig.json is)
		assert.Equal(t, configFile, configPath)
	})

	t.Run("finds config with shark-tasks.db as marker", func(t *testing.T) {
		// Create temp directory with shark-tasks.db instead of .sharkconfig.json
		tmpDir := t.TempDir()
		dbFile := filepath.Join(tmpDir, "shark-tasks.db")
		require.NoError(t, os.WriteFile(dbFile, []byte{}, 0644))

		// Create subdirectory
		subDir := filepath.Join(tmpDir, "subdir")
		require.NoError(t, os.MkdirAll(subDir, 0755))

		// Change to subdirectory
		require.NoError(t, os.Chdir(subDir))
		defer func() { _ = os.Chdir(originalWd) }()

		// Get config path - should find project root via shark-tasks.db
		configPath, err := GetConfigPath()
		require.NoError(t, err)

		// Should return path to .sharkconfig.json in project root (even if it doesn't exist)
		expectedPath := filepath.Join(tmpDir, ".sharkconfig.json")
		assert.Equal(t, expectedPath, configPath)
	})

	t.Run("uses .git as fallback when no config or db file exists", func(t *testing.T) {
		// Skip test if /tmp has project markers (shark-tasks.db, .git, .sharkconfig.json)
		// This prevents test failures when running on systems where /tmp contains these files
		hasMarkers := false
		for _, marker := range []string{".sharkconfig.json", "shark-tasks.db", ".git"} {
			if _, err := os.Stat(filepath.Join(os.TempDir(), marker)); err == nil {
				hasMarkers = true
				break
			}
		}
		if hasMarkers {
			t.Skip("Skipping test: /tmp contains project markers that would interfere with test")
		}

		// Create temp directory with only .git
		tmpDir := t.TempDir()
		gitDir := filepath.Join(tmpDir, ".git")
		require.NoError(t, os.MkdirAll(gitDir, 0755))

		// Create subdirectory
		subDir := filepath.Join(tmpDir, "subdir")
		require.NoError(t, os.MkdirAll(subDir, 0755))

		// Change to subdirectory
		require.NoError(t, os.Chdir(subDir))
		defer func() { _ = os.Chdir(originalWd) }()

		// Get config path - should find project root via .git
		configPath, err := GetConfigPath()
		require.NoError(t, err)

		// Should return path to .sharkconfig.json in git root
		expectedPath := filepath.Join(tmpDir, ".sharkconfig.json")
		assert.Equal(t, expectedPath, configPath)
	})

	t.Run("returns current directory when no project root markers found", func(t *testing.T) {
		// Skip test if /tmp has project markers (shark-tasks.db, .git, .sharkconfig.json)
		// This prevents test failures when running on systems where /tmp contains these files
		hasMarkers := false
		for _, marker := range []string{".sharkconfig.json", "shark-tasks.db", ".git"} {
			if _, err := os.Stat(filepath.Join(os.TempDir(), marker)); err == nil {
				hasMarkers = true
				break
			}
		}
		if hasMarkers {
			t.Skip("Skipping test: /tmp contains project markers that would interfere with test")
		}

		// Create temp directory with no markers
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "subdir")
		require.NoError(t, os.MkdirAll(subDir, 0755))

		// Change to subdirectory
		require.NoError(t, os.Chdir(subDir))
		defer func() { _ = os.Chdir(originalWd) }()

		// Get config path - should use current directory as fallback
		configPath, err := GetConfigPath()
		require.NoError(t, err)

		// Should return path to .sharkconfig.json in current directory
		expectedPath := filepath.Join(subDir, ".sharkconfig.json")
		assert.Equal(t, expectedPath, configPath)
	})

	t.Run("uses explicit config path when GlobalConfig.ConfigFile is set", func(t *testing.T) {
		// Create temp directory
		tmpDir := t.TempDir()

		// Set explicit config path
		explicitConfig := filepath.Join(tmpDir, "custom-config.json")
		GlobalConfig.ConfigFile = explicitConfig

		// Cleanup
		defer func() {
			GlobalConfig.ConfigFile = ""
		}()

		// Get config path
		configPath, err := GetConfigPath()
		require.NoError(t, err)

		// Should return explicit path
		assert.Equal(t, explicitConfig, configPath)
	})
}

// TestFindProjectRoot tests the FindProjectRoot function behavior
func TestFindProjectRoot_Priority(t *testing.T) {
	// Save and restore original working directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalWd)
	}()

	t.Run("prioritizes .sharkconfig.json over .git", func(t *testing.T) {
		// Create nested structure:
		// tmpDir/.git
		// tmpDir/subdir/.sharkconfig.json
		tmpDir := t.TempDir()

		// Create .git in root
		gitDir := filepath.Join(tmpDir, ".git")
		require.NoError(t, os.MkdirAll(gitDir, 0755))

		// Create .sharkconfig.json in subdirectory
		subDir := filepath.Join(tmpDir, "subdir")
		require.NoError(t, os.MkdirAll(subDir, 0755))
		configFile := filepath.Join(subDir, ".sharkconfig.json")
		require.NoError(t, os.WriteFile(configFile, []byte("{}"), 0644))

		// Create deeper subdirectory
		deepDir := filepath.Join(subDir, "deep")
		require.NoError(t, os.MkdirAll(deepDir, 0755))

		// Change to deep subdirectory
		require.NoError(t, os.Chdir(deepDir))
		defer func() { _ = os.Chdir(originalWd) }()

		// Find project root
		projectRoot, err := FindProjectRoot()
		require.NoError(t, err)

		// Should return subDir (where .sharkconfig.json is) not tmpDir (where .git is)
		assert.Equal(t, subDir, projectRoot)
	})

	t.Run("prioritizes shark-tasks.db over .git", func(t *testing.T) {
		// Create nested structure:
		// tmpDir/.git
		// tmpDir/subdir/shark-tasks.db
		tmpDir := t.TempDir()

		// Create .git in root
		gitDir := filepath.Join(tmpDir, ".git")
		require.NoError(t, os.MkdirAll(gitDir, 0755))

		// Create shark-tasks.db in subdirectory
		subDir := filepath.Join(tmpDir, "subdir")
		require.NoError(t, os.MkdirAll(subDir, 0755))
		dbFile := filepath.Join(subDir, "shark-tasks.db")
		require.NoError(t, os.WriteFile(dbFile, []byte{}, 0644))

		// Create deeper subdirectory
		deepDir := filepath.Join(subDir, "deep")
		require.NoError(t, os.MkdirAll(deepDir, 0755))

		// Change to deep subdirectory
		require.NoError(t, os.Chdir(deepDir))
		defer func() { _ = os.Chdir(originalWd) }()

		// Find project root
		projectRoot, err := FindProjectRoot()
		require.NoError(t, err)

		// Should return subDir (where shark-tasks.db is) not tmpDir (where .git is)
		assert.Equal(t, subDir, projectRoot)
	})

	t.Run("returns .git directory when no strong markers found", func(t *testing.T) {
		// Skip test if /tmp has project markers (shark-tasks.db, .git, .sharkconfig.json)
		// This prevents test failures when running on systems where /tmp contains these files
		hasMarkers := false
		for _, marker := range []string{".sharkconfig.json", "shark-tasks.db", ".git"} {
			if _, err := os.Stat(filepath.Join(os.TempDir(), marker)); err == nil {
				hasMarkers = true
				break
			}
		}
		if hasMarkers {
			t.Skip("Skipping test: /tmp contains project markers that would interfere with test")
		}

		// Create directory with only .git
		tmpDir := t.TempDir()
		gitDir := filepath.Join(tmpDir, ".git")
		require.NoError(t, os.MkdirAll(gitDir, 0755))

		// Create subdirectory
		subDir := filepath.Join(tmpDir, "subdir")
		require.NoError(t, os.MkdirAll(subDir, 0755))

		// Change to subdirectory
		require.NoError(t, os.Chdir(subDir))
		defer func() { _ = os.Chdir(originalWd) }()

		// Find project root
		projectRoot, err := FindProjectRoot()
		require.NoError(t, err)

		// Should return tmpDir (where .git is)
		assert.Equal(t, tmpDir, projectRoot)
	})

	t.Run("prioritizes .sharkconfig.json in parent over shark-tasks.db in subdirectory", func(t *testing.T) {
		// This tests the exact bug that was reported:
		// tmpDir/.sharkconfig.json (project root)
		// tmpDir/docs/shark-tasks.db (subdirectory)
		// tmpDir/docs/plan/ (working directory)
		// Should find .sharkconfig.json in tmpDir, not stop at shark-tasks.db in docs
		tmpDir := t.TempDir()

		// Create .sharkconfig.json in root
		configFile := filepath.Join(tmpDir, ".sharkconfig.json")
		require.NoError(t, os.WriteFile(configFile, []byte("{}"), 0644))

		// Create docs subdirectory with shark-tasks.db
		docsDir := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(docsDir, 0755))
		dbFile := filepath.Join(docsDir, "shark-tasks.db")
		require.NoError(t, os.WriteFile(dbFile, []byte{}, 0644))

		// Create deeper subdirectory
		planDir := filepath.Join(docsDir, "plan")
		require.NoError(t, os.MkdirAll(planDir, 0755))

		// Change to plan directory
		require.NoError(t, os.Chdir(planDir))
		defer func() { _ = os.Chdir(originalWd) }()

		// Find project root
		projectRoot, err := FindProjectRoot()
		require.NoError(t, err)

		// Should return tmpDir (where .sharkconfig.json is) NOT docsDir (where shark-tasks.db is)
		assert.Equal(t, tmpDir, projectRoot)
	})
}
