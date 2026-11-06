package fileops

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAtomicWrite verifies atomic publication creates a complete file.
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

	actualContent, err := os.ReadFile(filepath.Join(tmpDir, "atomic.md"))
	require.NoError(t, err)
	assert.Equal(t, content, actualContent)
	actualInfo, err := os.Stat(filepath.Join(tmpDir, "atomic.md"))
	require.NoError(t, err)
	require.NotNil(t, result.CreatedFileInfo)
	assert.True(t, os.SameFile(result.CreatedFileInfo, actualInfo))
	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "atomic.md", entries[0].Name())
}

func TestAtomicWriteEmptyContent(t *testing.T) {
	tmpDir := t.TempDir()
	writer := NewEntityFileWriter()

	result, err := writer.WriteEntityFile(WriteOptions{
		Content:         []byte{},
		ProjectRoot:     tmpDir,
		FilePath:        "empty.md",
		EntityType:      "task",
		UseAtomicWrite:  true,
		CreateIfMissing: true,
	})

	require.NoError(t, err)
	assert.True(t, result.Written)
	actualContent, err := os.ReadFile(filepath.Join(tmpDir, "empty.md"))
	require.NoError(t, err)
	assert.Empty(t, actualContent)
	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "empty.md", entries[0].Name())
}

func TestAtomicPublicationPreservesRacedInFile(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "raced-in.md")
	replacement := []byte("created by the concurrent writer")
	writer := NewEntityFileWriter()
	writer.ioHooks = &fileWriterIOHooks{
		rename: func(stagedPath, destinationPath string) error {
			require.NoError(t, os.WriteFile(destinationPath, replacement, 0644))
			return os.Link(stagedPath, destinationPath)
		},
	}

	result, err := writer.WriteEntityFile(WriteOptions{
		Content:         []byte("content from this writer"),
		ProjectRoot:     tmpDir,
		FilePath:        "raced-in.md",
		UseAtomicWrite:  true,
		CreateIfMissing: true,
		EntityType:      "task",
	})

	require.NoError(t, err)
	assert.False(t, result.Written)
	assert.True(t, result.Linked)
	assert.Nil(t, result.CreatedFileInfo)
	actualContent, err := os.ReadFile(targetPath)
	require.NoError(t, err)
	assert.Equal(t, replacement, actualContent)
	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "raced-in.md", entries[0].Name())
}

func TestAtomicWriteLinksExistingTargetWithoutChangingOriginalContent(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "existing.md")
	originalContent := []byte("original document bytes\x00\xff")
	replacementContent := []byte("new content must not replace an existing file")
	require.NoError(t, os.WriteFile(targetPath, originalContent, 0644))

	result, err := NewEntityFileWriter().WriteEntityFile(WriteOptions{
		Content:         replacementContent,
		ProjectRoot:     tmpDir,
		FilePath:        "existing.md",
		UseAtomicWrite:  true,
		CreateIfMissing: true,
		EntityType:      "task",
	})

	require.NoError(t, err)
	assert.True(t, result.Linked)
	assert.False(t, result.Written)
	actualContent, err := os.ReadFile(targetPath)
	require.NoError(t, err)
	assert.Equal(t, originalContent, actualContent, "linking an existing regular file must preserve its original bytes")
}

func TestRenameNoReplacePreservesExistingTarget(t *testing.T) {
	root := t.TempDir()
	stagedPath := filepath.Join(root, "staged.md")
	targetPath := filepath.Join(root, "target.md")
	stagedContent := []byte("staged content")
	targetContent := []byte("existing content")
	require.NoError(t, os.WriteFile(stagedPath, stagedContent, 0600))
	require.NoError(t, os.WriteFile(targetPath, targetContent, 0600))

	err := renameNoReplace(stagedPath, targetPath)
	if errors.Is(err, ErrRenameUnsupported) {
		t.Skip("native no-replace rename is unavailable on this platform")
	}
	require.ErrorIs(t, err, os.ErrExist)

	actualTarget, readErr := os.ReadFile(targetPath)
	require.NoError(t, readErr)
	assert.Equal(t, targetContent, actualTarget, "a no-replace rename must preserve the existing target")
	actualStaged, readErr := os.ReadFile(stagedPath)
	require.NoError(t, readErr)
	assert.Equal(t, stagedContent, actualStaged, "a rejected rename must preserve its staged source")
}

func TestRenameNoReplacePublishesStagedFile(t *testing.T) {
	if !supportsNativeNoReplaceOnCurrentOS() {
		t.Skip("native no-replace rename is not implemented for this GOOS")
	}
	root := t.TempDir()
	stagedPath := filepath.Join(root, "staged.md")
	targetPath := filepath.Join(root, "target.md")
	stagedContent := []byte("published by native no-replace rename")
	require.NoError(t, os.WriteFile(stagedPath, stagedContent, 0600))

	err := renameNoReplace(stagedPath, targetPath)
	require.NoError(t, err)

	actualTarget, readErr := os.ReadFile(targetPath)
	require.NoError(t, readErr)
	assert.Equal(t, stagedContent, actualTarget)
	_, statErr := os.Lstat(stagedPath)
	assert.ErrorIs(t, statErr, os.ErrNotExist, "native publication must move the staged file without using the link fallback")
}

func TestAtomicPublicationUsesNativeNoReplaceBeforeLinkFallback(t *testing.T) {
	if !supportsNativeNoReplaceOnCurrentOS() {
		t.Skip("native no-replace rename is not implemented for this GOOS")
	}
	root := t.TempDir()
	targetPath := filepath.Join(root, "entity.md")
	content := []byte("published by the native no-replace operation")
	linkCalled := false
	writer := NewEntityFileWriter()
	writer.ioHooks = &fileWriterIOHooks{
		link: func(string, string) error {
			linkCalled = true
			return errors.New("link fallback must not run when native publication succeeds")
		},
	}

	result, err := writer.WriteEntityFile(WriteOptions{
		Content:         content,
		ProjectRoot:     root,
		FilePath:        "entity.md",
		EntityType:      "task",
		UseAtomicWrite:  true,
		CreateIfMissing: true,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Written)
	assert.False(t, result.Linked)
	assert.False(t, linkCalled, "successful native publication must not use the hard-link fallback")
	actualContent, readErr := os.ReadFile(targetPath)
	require.NoError(t, readErr)
	assert.Equal(t, content, actualContent)
	targetInfo, statErr := os.Stat(targetPath)
	require.NoError(t, statErr)
	require.NotNil(t, result.CreatedFileInfo)
	assert.True(t, os.SameFile(result.CreatedFileInfo, targetInfo))
}

func supportsNativeNoReplaceOnCurrentOS() bool {
	return runtime.GOOS == "linux" || runtime.GOOS == "darwin" || runtime.GOOS == "windows"
}

func TestAtomicPublicationFailureDoesNotLeaveTarget(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "unsupported-publication.md")
	publicationErr := errors.New("hard-link publication is unsupported")
	writer := NewEntityFileWriter()
	writer.ioHooks = &fileWriterIOHooks{
		rename: func(string, string) error {
			return publicationErr
		},
	}

	_, err := writer.WriteEntityFile(WriteOptions{
		Content:         []byte("complete staged content"),
		ProjectRoot:     tmpDir,
		FilePath:        "unsupported-publication.md",
		UseAtomicWrite:  true,
		CreateIfMissing: true,
		EntityType:      "task",
	})

	require.ErrorIs(t, err, publicationErr)
	_, statErr := os.Stat(targetPath)
	require.ErrorIs(t, statErr, os.ErrNotExist)
	entries, readErr := os.ReadDir(tmpDir)
	require.NoError(t, readErr)
	assert.Empty(t, entries)
}

func TestAtomicPublicationRejectsSymlinkToRegularFile(t *testing.T) {
	for _, tc := range []struct {
		name  string
		raced bool
	}{{"preexisting", false}, {"raced", true}} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			targetPath := filepath.Join(root, "task-link.md")
			referentPath := filepath.Join(root, "existing.md")
			original := []byte("existing document must remain unchanged")
			require.NoError(t, os.WriteFile(referentPath, original, 0644))
			createSymlink := func() {
				if err := os.Symlink(referentPath, targetPath); err != nil {
					t.Skipf("symlink creation is unavailable: %v", err)
				}
			}
			writer := NewEntityFileWriter()
			if tc.raced {
				writer.ioHooks = &fileWriterIOHooks{rename: func(stagedPath, destinationPath string) error {
					createSymlink()
					return os.Link(stagedPath, destinationPath)
				}}
			} else {
				createSymlink()
			}

			result, err := writer.WriteEntityFile(WriteOptions{
				Content: []byte("task content"), ProjectRoot: root, FilePath: "task-link.md",
				UseAtomicWrite: true, CreateIfMissing: true, EntityType: "task",
			})

			require.ErrorIs(t, err, os.ErrInvalid)
			assert.Nil(t, result, "a symlink must not be reported as a linked regular file")
			assertPublicationSymlinkPreserved(t, root, targetPath, referentPath, original)
		})
	}
}

func assertPublicationSymlinkPreserved(t *testing.T, root, targetPath, referentPath string, original []byte) {
	t.Helper()
	linkTarget, err := os.Readlink(targetPath)
	require.NoError(t, err)
	assert.Equal(t, referentPath, linkTarget)
	actual, err := os.ReadFile(referentPath)
	require.NoError(t, err)
	assert.Equal(t, original, actual)
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	assert.Len(t, entries, 2, "rejection must preserve both paths and remove any staged file")
}

func TestAtomicPublicationRejectsDanglingSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "dangling-link.md")
	if err := os.Symlink("missing-target.md", targetPath); err != nil {
		t.Skipf("symlink creation is unavailable: %v", err)
	}
	writer := NewEntityFileWriter()

	result, err := writer.WriteEntityFile(WriteOptions{
		Content:         []byte("task content"),
		ProjectRoot:     tmpDir,
		FilePath:        "dangling-link.md",
		UseAtomicWrite:  true,
		CreateIfMissing: true,
		EntityType:      "task",
	})

	require.ErrorIs(t, err, os.ErrInvalid)
	assert.Nil(t, result)
	_, err = os.Lstat(targetPath)
	require.NoError(t, err, "the writer must preserve the path that caused the collision")
	linkTarget, err := os.Readlink(targetPath)
	require.NoError(t, err)
	assert.Equal(t, "missing-target.md", linkTarget)
	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	require.Len(t, entries, 1, "failed publication must remove the staged file")
	assert.Equal(t, "dangling-link.md", entries[0].Name())
}

func TestAtomicPublicationRejectsRacedDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "raced-directory.md")
	writer := NewEntityFileWriter()
	writer.ioHooks = &fileWriterIOHooks{
		rename: func(stagedPath, destinationPath string) error {
			require.NoError(t, os.Mkdir(destinationPath, 0755))
			return os.Link(stagedPath, destinationPath)
		},
	}

	result, err := writer.WriteEntityFile(WriteOptions{
		Content:         []byte("task content"),
		ProjectRoot:     tmpDir,
		FilePath:        "raced-directory.md",
		UseAtomicWrite:  true,
		CreateIfMissing: true,
		EntityType:      "task",
	})

	require.ErrorIs(t, err, os.ErrInvalid)
	assert.Nil(t, result)
	info, err := os.Stat(targetPath)
	require.NoError(t, err, "the writer must preserve the path that caused the collision")
	assert.True(t, info.IsDir())
	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	require.Len(t, entries, 1, "failed publication must remove the staged file")
	assert.Equal(t, "raced-directory.md", entries[0].Name())
}

func TestWriteFailureDoesNotLeavePartialTarget(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "partial.md")
	writeErr := errors.New("injected partial write failure")
	writer := NewEntityFileWriter()
	writer.ioHooks = &fileWriterIOHooks{
		write: func(file *os.File, content []byte) (int, error) {
			n, err := file.Write(content[:4])
			require.NoError(t, err)
			return n, writeErr
		},
	}

	_, err := writer.WriteEntityFile(WriteOptions{
		Content:         []byte("partial content"),
		ProjectRoot:     tmpDir,
		FilePath:        "partial.md",
		UseAtomicWrite:  true,
		CreateIfMissing: true,
		EntityType:      "task",
	})

	require.ErrorIs(t, err, writeErr)
	_, statErr := os.Stat(targetPath)
	require.ErrorIs(t, statErr, os.ErrNotExist)
	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestShortWriteWithoutErrorDoesNotLeaveTarget(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "short-write.md")
	writer := NewEntityFileWriter()
	writer.ioHooks = &fileWriterIOHooks{
		write: func(_ *os.File, content []byte) (int, error) {
			return len(content) - 1, nil
		},
	}

	_, err := writer.WriteEntityFile(WriteOptions{
		Content:         []byte("short write"),
		ProjectRoot:     tmpDir,
		FilePath:        "short-write.md",
		UseAtomicWrite:  true,
		CreateIfMissing: true,
		EntityType:      "task",
	})

	require.ErrorIs(t, err, io.ErrShortWrite)
	_, statErr := os.Stat(targetPath)
	require.ErrorIs(t, statErr, os.ErrNotExist)
	entries, readErr := os.ReadDir(tmpDir)
	require.NoError(t, readErr)
	assert.Empty(t, entries)
}

func TestSyncFailureDoesNotLeaveTarget(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "sync-failure.md")
	syncErr := errors.New("injected sync failure")
	writer := NewEntityFileWriter()
	writer.ioHooks = &fileWriterIOHooks{
		sync: func(*os.File) error {
			return syncErr
		},
	}

	_, err := writer.WriteEntityFile(WriteOptions{
		Content:         []byte("complete but unsynced content"),
		ProjectRoot:     tmpDir,
		FilePath:        "sync-failure.md",
		UseAtomicWrite:  true,
		CreateIfMissing: true,
		EntityType:      "task",
	})

	require.ErrorIs(t, err, syncErr)
	_, statErr := os.Stat(targetPath)
	require.ErrorIs(t, statErr, os.ErrNotExist)
	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestCloseFailureDoesNotLeaveTarget(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "close-failure.md")
	closeErr := errors.New("injected close failure")
	writer := NewEntityFileWriter()
	writer.ioHooks = &fileWriterIOHooks{
		close: func(file *os.File) error {
			if err := file.Close(); err != nil {
				return err
			}
			return closeErr
		},
	}

	result, err := writer.WriteEntityFile(WriteOptions{
		Content:         []byte("staged content"),
		ProjectRoot:     tmpDir,
		FilePath:        "close-failure.md",
		UseAtomicWrite:  true,
		CreateIfMissing: true,
		EntityType:      "task",
	})

	require.ErrorIs(t, err, closeErr)
	assert.Nil(t, result)
	_, statErr := os.Lstat(targetPath)
	assert.ErrorIs(t, statErr, os.ErrNotExist)
	entries, readDirErr := os.ReadDir(tmpDir)
	require.NoError(t, readDirErr)
	assert.Empty(t, entries, "a close failure must remove its staged sibling")
}

func TestStagingFailuresPreserveRacedInTarget(t *testing.T) {
	for _, failurePoint := range []string{"write", "sync", "close"} {
		t.Run(failurePoint, func(t *testing.T) {
			tmpDir := t.TempDir()
			targetPath := filepath.Join(tmpDir, "raced-in.md")
			concurrentContent := []byte("created by the concurrent writer")
			writeErr := errors.New("injected " + failurePoint + " failure")
			writer := NewEntityFileWriter()
			writer.ioHooks = &fileWriterIOHooks{}

			createConcurrentTarget := func() {
				require.NoError(t, os.WriteFile(targetPath, concurrentContent, 0644))
			}
			if failurePoint == "write" {
				writer.ioHooks.write = func(file *os.File, content []byte) (int, error) {
					createConcurrentTarget()
					n, err := file.Write(content[:4])
					require.NoError(t, err)
					return n, writeErr
				}
			} else {
				if failurePoint == "sync" {
					writer.ioHooks.sync = func(*os.File) error {
						createConcurrentTarget()
						return writeErr
					}
				} else {
					writer.ioHooks.close = func(file *os.File) error {
						if err := file.Close(); err != nil {
							return err
						}
						createConcurrentTarget()
						return writeErr
					}
				}
			}

			result, err := writer.WriteEntityFile(WriteOptions{
				Content:         []byte("staged content"),
				ProjectRoot:     tmpDir,
				FilePath:        "raced-in.md",
				UseAtomicWrite:  true,
				CreateIfMissing: true,
				EntityType:      "task",
			})

			require.ErrorIs(t, err, writeErr)
			assert.Nil(t, result)
			actualContent, readErr := os.ReadFile(targetPath)
			require.NoError(t, readErr)
			assert.Equal(t, concurrentContent, actualContent)
			entries, readDirErr := os.ReadDir(tmpDir)
			require.NoError(t, readDirErr)
			require.Len(t, entries, 1, "failed staging must remove only its staged sibling")
			assert.Equal(t, "raced-in.md", entries[0].Name())
		})
	}
}

func TestAtomicPublicationSucceedsWithoutHardLinks(t *testing.T) {
	root := t.TempDir()
	writer := NewEntityFileWriter()
	writer.ioHooks = &fileWriterIOHooks{
		rename: func(from, to string) error { return os.Rename(from, to) },
		link:   func(string, string) error { return errors.New("hard links unavailable") },
	}
	result, err := writer.WriteEntityFile(WriteOptions{Content: []byte("complete"), ProjectRoot: root, FilePath: "entity.md", EntityType: "task", UseAtomicWrite: true, CreateIfMissing: true})
	require.NoError(t, err)
	require.True(t, result.Written)
	content, err := os.ReadFile(filepath.Join(root, "entity.md"))
	require.NoError(t, err)
	require.Equal(t, "complete", string(content))
}

func TestAtomicPublicationFallsBackWhenRenameUnsupported(t *testing.T) {
	root := t.TempDir()
	writer := NewEntityFileWriter()
	usedLink := false
	writer.ioHooks = &fileWriterIOHooks{rename: func(string, string) error { return ErrRenameUnsupported }, link: func(from, to string) error { usedLink = true; return os.Link(from, to) }}
	result, err := writer.WriteEntityFile(WriteOptions{Content: []byte("complete"), ProjectRoot: root, FilePath: "entity.md", EntityType: "task", UseAtomicWrite: true, CreateIfMissing: true})
	require.NoError(t, err)
	require.True(t, usedLink)
	require.True(t, result.Written)
	assert.False(t, result.Linked)
	content, err := os.ReadFile(filepath.Join(root, "entity.md"))
	require.NoError(t, err)
	require.Equal(t, "complete", string(content))
	targetInfo, err := os.Stat(filepath.Join(root, "entity.md"))
	require.NoError(t, err)
	require.NotNil(t, result.CreatedFileInfo)
	assert.True(t, os.SameFile(result.CreatedFileInfo, targetInfo))
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	require.Len(t, entries, 1)
}

func TestAtomicPublicationPreservesRacedInFileDuringLinkFallback(t *testing.T) {
	root := t.TempDir()
	targetPath := filepath.Join(root, "raced-in.md")
	concurrentContent := []byte("created by the concurrent writer\x00\xff")
	writer := NewEntityFileWriter()
	writer.ioHooks = &fileWriterIOHooks{
		rename: func(string, string) error { return ErrRenameUnsupported },
		link: func(stagedPath, destinationPath string) error {
			require.NoError(t, os.WriteFile(destinationPath, concurrentContent, 0644))
			return os.Link(stagedPath, destinationPath)
		},
	}

	result, err := writer.WriteEntityFile(WriteOptions{
		Content:         []byte("content from this writer"),
		ProjectRoot:     root,
		FilePath:        "raced-in.md",
		UseAtomicWrite:  true,
		CreateIfMissing: true,
		EntityType:      "task",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Written)
	assert.True(t, result.Linked)
	assert.Nil(t, result.CreatedFileInfo)
	actualContent, err := os.ReadFile(targetPath)
	require.NoError(t, err)
	assert.Equal(t, concurrentContent, actualContent)
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	require.Len(t, entries, 1, "fallback collision must remove only the staged sibling")
	assert.Equal(t, "raced-in.md", entries[0].Name())
}

func TestAtomicPublicationFailsClosedWhenBothMethodsUnsupported(t *testing.T) {
	root := t.TempDir()
	linkErr := errors.New("link unsupported")
	writer := NewEntityFileWriter()
	writer.ioHooks = &fileWriterIOHooks{rename: func(string, string) error { return ErrRenameUnsupported }, link: func(string, string) error { return linkErr }}
	_, err := writer.WriteEntityFile(WriteOptions{Content: []byte("complete"), ProjectRoot: root, FilePath: "entity.md", EntityType: "task", UseAtomicWrite: true, CreateIfMissing: true})
	require.ErrorIs(t, err, linkErr)
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	require.Empty(t, entries)
}
