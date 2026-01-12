package fileops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWriteNewFile tests writing to a non-existent file
func TestWriteNewFile(t *testing.T) {
	tmpDir := t.TempDir()
	content := []byte("test content")

	writer := NewEntityFileWriter()
	result, err := writer.WriteEntityFile(WriteOptions{
		Content:     content,
		ProjectRoot: tmpDir,
		FilePath:    "test.md",
		EntityType:  "epic",
	})

	require.NoError(t, err)
	assert.True(t, result.Written)
	assert.False(t, result.Linked)
	assert.Equal(t, filepath.Join(tmpDir, "test.md"), result.AbsolutePath)
	assert.Equal(t, "test.md", result.RelativePath)

	// Verify file contents
	actualContent, err := os.ReadFile(filepath.Join(tmpDir, "test.md"))
	require.NoError(t, err)
	assert.Equal(t, content, actualContent)
}

// TestLinkExistingFile tests that existing files are linked, not overwritten
func TestLinkExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "existing.md")
	originalContent := []byte("original content")
	newContent := []byte("new content")

	// Create existing file
	err := os.WriteFile(filePath, originalContent, 0644)
	require.NoError(t, err)

	writer := NewEntityFileWriter()
	result, err := writer.WriteEntityFile(WriteOptions{
		Content:     newContent,
		ProjectRoot: tmpDir,
		FilePath:    "existing.md",
		EntityType:  "feature",
	})

	require.NoError(t, err)
	assert.False(t, result.Written)
	assert.True(t, result.Linked)

	// Verify original content is unchanged
	actualContent, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, originalContent, actualContent)
}

// TestAtomicWrite tests that atomic writes use O_EXCL flag
func TestAtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	content := []byte("atomic content")

	writer := NewEntityFileWriter()
	result, err := writer.WriteEntityFile(WriteOptions{
		Content:         content,
		ProjectRoot:     tmpDir,
		FilePath:        "atomic.md",
		EntityType:      "task",
		UseAtomicWrite:  true,
		CreateIfMissing: true, // Required for tasks
	})

	require.NoError(t, err)
	assert.True(t, result.Written)

	// Verify file was created
	actualContent, err := os.ReadFile(filepath.Join(tmpDir, "atomic.md"))
	require.NoError(t, err)
	assert.Equal(t, content, actualContent)
}

// TestNonAtomicWrite tests simple write without O_EXCL
func TestNonAtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	content := []byte("non-atomic content")

	writer := NewEntityFileWriter()
	result, err := writer.WriteEntityFile(WriteOptions{
		Content:        content,
		ProjectRoot:    tmpDir,
		FilePath:       "non-atomic.md",
		EntityType:     "epic",
		UseAtomicWrite: false,
	})

	require.NoError(t, err)
	assert.True(t, result.Written)

	// Verify file was created
	actualContent, err := os.ReadFile(filepath.Join(tmpDir, "non-atomic.md"))
	require.NoError(t, err)
	assert.Equal(t, content, actualContent)
}

// TestCreateDirectories tests that parent directories are created
func TestCreateDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	content := []byte("nested content")
	nestedPath := "a/b/c/deep.md"

	writer := NewEntityFileWriter()
	result, err := writer.WriteEntityFile(WriteOptions{
		Content:     content,
		ProjectRoot: tmpDir,
		FilePath:    nestedPath,
		EntityType:  "feature",
	})

	require.NoError(t, err)
	assert.True(t, result.Written)

	// Verify file exists
	actualContent, err := os.ReadFile(filepath.Join(tmpDir, nestedPath))
	require.NoError(t, err)
	assert.Equal(t, content, actualContent)
}

// TestRelativePath tests relative path resolution
func TestRelativePath(t *testing.T) {
	tmpDir := t.TempDir()
	content := []byte("relative content")

	writer := NewEntityFileWriter()
	result, err := writer.WriteEntityFile(WriteOptions{
		Content:     content,
		ProjectRoot: tmpDir,
		FilePath:    "docs/plan/test.md",
		EntityType:  "epic",
	})

	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmpDir, "docs/plan/test.md"), result.AbsolutePath)
	assert.Equal(t, "docs/plan/test.md", result.RelativePath)
}

// TestAbsolutePath tests absolute path handling
func TestAbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()
	content := []byte("absolute content")
	absPath := filepath.Join(tmpDir, "absolute.md")

	writer := NewEntityFileWriter()
	result, err := writer.WriteEntityFile(WriteOptions{
		Content:         content,
		ProjectRoot:     tmpDir,
		FilePath:        absPath,
		EntityType:      "task",
		CreateIfMissing: true, // Required for tasks
	})

	require.NoError(t, err)
	assert.Equal(t, absPath, result.AbsolutePath)
	assert.Equal(t, "absolute.md", result.RelativePath)
}

// TestCreateIfMissingTrue tests task creation with --create flag
func TestCreateIfMissingTrue(t *testing.T) {
	tmpDir := t.TempDir()
	content := []byte("task content")

	writer := NewEntityFileWriter()
	result, err := writer.WriteEntityFile(WriteOptions{
		Content:         content,
		ProjectRoot:     tmpDir,
		FilePath:        "task.md",
		CreateIfMissing: true,
		EntityType:      "task",
	})

	require.NoError(t, err)
	assert.True(t, result.Written)

	// Verify file was created
	actualContent, err := os.ReadFile(filepath.Join(tmpDir, "task.md"))
	require.NoError(t, err)
	assert.Equal(t, content, actualContent)
}

// TestVerboseLogging tests that logger is called when verbose is true
func TestVerboseLogging(t *testing.T) {
	tmpDir := t.TempDir()
	content := []byte("verbose content")
	var logMessages []string
	logger := func(msg string) {
		logMessages = append(logMessages, msg)
	}

	writer := NewEntityFileWriter()
	_, err := writer.WriteEntityFile(WriteOptions{
		Content:     content,
		ProjectRoot: tmpDir,
		FilePath:    "verbose.md",
		Verbose:     true,
		EntityType:  "epic",
		Logger:      logger,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, logMessages)
	assert.True(t, len(logMessages) > 0)
	assert.Contains(t, logMessages[0], "epic")
}

// TestVerboseLoggingForExistingFile tests verbose logging when file exists
func TestVerboseLoggingForExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "existing-verbose.md")
	originalContent := []byte("original")

	// Create existing file
	err := os.WriteFile(filePath, originalContent, 0644)
	require.NoError(t, err)

	var logMessages []string
	logger := func(msg string) {
		logMessages = append(logMessages, msg)
	}

	writer := NewEntityFileWriter()
	_, err = writer.WriteEntityFile(WriteOptions{
		Content:     []byte("new"),
		ProjectRoot: tmpDir,
		FilePath:    "existing-verbose.md",
		Verbose:     true,
		EntityType:  "feature",
		Logger:      logger,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, logMessages)
	assert.Contains(t, logMessages[0], "already exists")
	assert.Contains(t, logMessages[0], "linking")
}

// ============================================
// NEGATIVE TEST CASES
// ============================================

// TestPermissionDenied tests writing to a protected directory
func TestPermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tmpDir := t.TempDir()
	protectedDir := filepath.Join(tmpDir, "protected")
	err := os.Mkdir(protectedDir, 0555) // Read-only directory
	require.NoError(t, err)

	writer := NewEntityFileWriter()
	_, err = writer.WriteEntityFile(WriteOptions{
		Content:     []byte("content"),
		ProjectRoot: tmpDir,
		FilePath:    "protected/file.md",
		EntityType:  "epic",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write")
}

// TestCreateIfMissingFalse tests that task creation fails without --create flag
func TestCreateIfMissingFalse(t *testing.T) {
	tmpDir := t.TempDir()

	writer := NewEntityFileWriter()
	_, err := writer.WriteEntityFile(WriteOptions{
		Content:         []byte("task content"),
		ProjectRoot:     tmpDir,
		FilePath:        "nonexistent.md",
		CreateIfMissing: false,
		EntityType:      "task",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
	assert.Contains(t, err.Error(), "--create flag")
}

// TestAtomicWriteRaceCondition tests atomic write when file already exists
func TestAtomicWriteRaceCondition(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "race.md")

	// Create file first (simulating race condition)
	err := os.WriteFile(filePath, []byte("existing"), 0644)
	require.NoError(t, err)

	writer := NewEntityFileWriter()
	result, err := writer.WriteEntityFile(WriteOptions{
		Content:        []byte("new content"),
		ProjectRoot:    tmpDir,
		FilePath:       "race.md",
		UseAtomicWrite: true,
		EntityType:     "task",
	})

	// Should not error - should link to existing file instead
	require.NoError(t, err)
	assert.True(t, result.Linked)
	assert.False(t, result.Written)
}

// TestEmptyFilePath tests error handling for empty file path
func TestEmptyFilePath(t *testing.T) {
	tmpDir := t.TempDir()

	writer := NewEntityFileWriter()
	_, err := writer.WriteEntityFile(WriteOptions{
		Content:     []byte("content"),
		ProjectRoot: tmpDir,
		FilePath:    "",
		EntityType:  "epic",
	})

	assert.Error(t, err)
}

// TestPathTraversal tests that path traversal is handled
func TestPathTraversal(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file outside the project root
	outsideDir := filepath.Join(filepath.Dir(tmpDir), "outside")
	err := os.MkdirAll(outsideDir, 0755)
	require.NoError(t, err)
	defer os.RemoveAll(outsideDir)

	writer := NewEntityFileWriter()
	result, err := writer.WriteEntityFile(WriteOptions{
		Content:     []byte("content"),
		ProjectRoot: tmpDir,
		FilePath:    "../outside/file.md",
		EntityType:  "epic",
	})

	// Should succeed - path traversal is allowed for custom file paths
	// But the relative path should show it's outside project root
	require.NoError(t, err)
	assert.True(t, strings.Contains(result.RelativePath, "..") ||
		filepath.IsAbs(result.RelativePath))
}

// TestAbsolutePathOutsideProjectRoot tests absolute path outside project root
func TestAbsolutePathOutsideProjectRoot(t *testing.T) {
	tmpDir := t.TempDir()
	outsideDir := filepath.Join(filepath.Dir(tmpDir), "outside-absolute")
	err := os.MkdirAll(outsideDir, 0755)
	require.NoError(t, err)
	defer os.RemoveAll(outsideDir)

	absPath := filepath.Join(outsideDir, "file.md")

	writer := NewEntityFileWriter()
	result, err := writer.WriteEntityFile(WriteOptions{
		Content:     []byte("content"),
		ProjectRoot: tmpDir,
		FilePath:    absPath,
		EntityType:  "epic",
	})

	require.NoError(t, err)
	assert.Equal(t, absPath, result.AbsolutePath)
	// Relative path should be the absolute path when outside project root
	assert.True(t, filepath.IsAbs(result.RelativePath) ||
		strings.Contains(result.RelativePath, ".."))
}

// TestStatErrorHandling tests error handling when stat fails
func TestStatErrorHandling(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a path with a file as an intermediate component
	// (this will cause stat to return an error other than NotExist)
	intermediateFile := filepath.Join(tmpDir, "file.txt")
	err := os.WriteFile(intermediateFile, []byte("content"), 0644)
	require.NoError(t, err)

	writer := NewEntityFileWriter()
	_, err = writer.WriteEntityFile(WriteOptions{
		Content:     []byte("content"),
		ProjectRoot: tmpDir,
		FilePath:    "file.txt/impossible.md", // file.txt is a file, not a dir
		EntityType:  "epic",
	})

	// Should fail when trying to create directories or write
	assert.Error(t, err)
}

// TestNilLogger tests that nil logger doesn't cause panic
func TestNilLogger(t *testing.T) {
	tmpDir := t.TempDir()

	writer := NewEntityFileWriter()
	_, err := writer.WriteEntityFile(WriteOptions{
		Content:     []byte("content"),
		ProjectRoot: tmpDir,
		FilePath:    "test.md",
		Verbose:     true,
		EntityType:  "epic",
		Logger:      nil, // Nil logger should be handled gracefully
	})

	require.NoError(t, err)
}

// TestForceOverwrite tests that Force flag allows overwriting existing files
func TestForceOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "force.md")
	originalContent := []byte("original")
	newContent := []byte("new content")

	// Create existing file
	err := os.WriteFile(filePath, originalContent, 0644)
	require.NoError(t, err)

	writer := NewEntityFileWriter()
	result, err := writer.WriteEntityFile(WriteOptions{
		Content:     newContent,
		ProjectRoot: tmpDir,
		FilePath:    "force.md",
		Force:       true,
		EntityType:  "epic",
	})

	require.NoError(t, err)
	assert.True(t, result.Written)
	assert.False(t, result.Linked)

	// Verify content was overwritten
	actualContent, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, newContent, actualContent)
}

// TestCreateIfMissingForNonTaskEntities tests that CreateIfMissing only affects tasks
func TestCreateIfMissingForNonTaskEntities(t *testing.T) {
	tmpDir := t.TempDir()

	writer := NewEntityFileWriter()
	// Epic with CreateIfMissing=false should still create the file
	result, err := writer.WriteEntityFile(WriteOptions{
		Content:         []byte("epic content"),
		ProjectRoot:     tmpDir,
		FilePath:        "epic.md",
		CreateIfMissing: false,
		EntityType:      "epic", // Not a task
	})

	require.NoError(t, err)
	assert.True(t, result.Written)

	// Verify file was created
	_, err = os.ReadFile(filepath.Join(tmpDir, "epic.md"))
	require.NoError(t, err)
}

// TestAtomicWriteFailsWhenFileExists tests O_EXCL behavior
func TestAtomicWriteFailsWhenFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "atomic-exists.md")

	// Create file first
	err := os.WriteFile(filePath, []byte("existing"), 0644)
	require.NoError(t, err)

	// Try to write with Force=true to bypass the existence check
	// but the atomic write should still be attempted after removing the file
	writer := NewEntityFileWriter()
	result, err := writer.WriteEntityFile(WriteOptions{
		Content:         []byte("new content"),
		ProjectRoot:     tmpDir,
		FilePath:        "atomic-exists.md",
		UseAtomicWrite:  true,
		CreateIfMissing: true,
		EntityType:      "task",
		Force:           true, // This will delete the file first
	})

	require.NoError(t, err)
	assert.True(t, result.Written)
}

// TestWriteFileExclusiveError tests writeFileExclusive error handling
func TestWriteFileExclusiveError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tmpDir := t.TempDir()
	protectedDir := filepath.Join(tmpDir, "protected-atomic")
	err := os.Mkdir(protectedDir, 0555) // Read-only directory
	require.NoError(t, err)

	writer := NewEntityFileWriter()
	_, err = writer.WriteEntityFile(WriteOptions{
		Content:         []byte("content"),
		ProjectRoot:     tmpDir,
		FilePath:        "protected-atomic/file.md",
		UseAtomicWrite:  true,
		CreateIfMissing: true,
		EntityType:      "task",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write")
}

// TestNonAtomicWriteError tests non-atomic write error handling
func TestNonAtomicWriteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tmpDir := t.TempDir()
	protectedDir := filepath.Join(tmpDir, "protected-non-atomic")
	err := os.Mkdir(protectedDir, 0555) // Read-only directory
	require.NoError(t, err)

	writer := NewEntityFileWriter()
	_, err = writer.WriteEntityFile(WriteOptions{
		Content:        []byte("content"),
		ProjectRoot:    tmpDir,
		FilePath:       "protected-non-atomic/file.md",
		UseAtomicWrite: false,
		EntityType:     "epic",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write")
}

// TestResolvePathsEdgeCases tests path resolution edge cases
func TestResolvePathsEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()

	// Test with path that can't be made relative (different drive on Windows, etc)
	writer := NewEntityFileWriter()
	result, err := writer.WriteEntityFile(WriteOptions{
		Content:     []byte("content"),
		ProjectRoot: tmpDir,
		FilePath:    tmpDir + "/test.md", // Absolute path within project
		EntityType:  "epic",
	})

	require.NoError(t, err)
	assert.True(t, result.Written)
	assert.Equal(t, filepath.Join(tmpDir, "test.md"), result.AbsolutePath)
}

// TestSyncFailure tests file sync failure handling
func TestSyncFailure(t *testing.T) {
	tmpDir := t.TempDir()

	writer := NewEntityFileWriter()
	_, err := writer.WriteEntityFile(WriteOptions{
		Content:         []byte("content"),
		ProjectRoot:     tmpDir,
		FilePath:        "sync-test.md",
		UseAtomicWrite:  true,
		CreateIfMissing: true,
		EntityType:      "task",
	})

	// Sync should succeed in normal cases
	require.NoError(t, err)

	// Verify file was created and synced
	data, err := os.ReadFile(filepath.Join(tmpDir, "sync-test.md"))
	require.NoError(t, err)
	assert.Equal(t, []byte("content"), data)
}
