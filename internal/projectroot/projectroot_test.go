package projectroot

import (
	"os"
	"path/filepath"
	"testing"
)

// These cases moved here (not copied) from internal/cli/root_test.go when
// FindProjectRootFrom's implementation was extracted into this package
// (T-E34-F08-008's cascade wiring needed internal/integration to resolve a
// project root without importing internal/cli — see this package's doc
// comment). internal/cli.findProjectRootFrom now delegates here; keeping
// these cases only at the delegate would let a future refactor quietly
// re-inline the walk without any test noticing.

func TestFindProjectRootFrom_EmptyGitDirNotAccepted(t *testing.T) {
	// B054: a stray, empty .git directory (no HEAD file, no objects/ dir) must
	// NOT be accepted as a project-root marker. Without content validation,
	// FindProjectRootFrom wrongly treats any directory named ".git" as valid.
	//
	// Start the search from a subdirectory below the empty .git so that
	// "accepted" (root == tmpDir) and "not accepted" (root == startDir, the
	// no-markers-found fallback) are distinguishable outcomes.
	tmpDir := t.TempDir()

	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create empty .git directory: %v", err)
	}

	startDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(startDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// ceiling = tmpDir keeps the walk from escaping into the host filesystem.
	root, err := FindProjectRootFrom(startDir, tmpDir)
	if err != nil {
		t.Fatalf("FindProjectRootFrom() error = %v", err)
	}

	if root != startDir {
		t.Errorf("FindProjectRootFrom() = %q, want %q (empty .git dir must not be accepted as a marker, so no markers are found and startDir is returned)", root, startDir)
	}
}

func TestFindProjectRootFrom_GitDirWithHEADAccepted(t *testing.T) {
	// Regression guard: a .git directory containing a HEAD file (the common
	// case for a normal git checkout) must still be accepted.
	tmpDir := t.TempDir()

	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0644); err != nil {
		t.Fatalf("Failed to create HEAD file: %v", err)
	}

	root, err := FindProjectRootFrom(tmpDir, tmpDir)
	if err != nil {
		t.Fatalf("FindProjectRootFrom() error = %v", err)
	}

	if root != tmpDir {
		t.Errorf("FindProjectRootFrom() = %q, want %q (.git dir with HEAD file must be accepted)", root, tmpDir)
	}
}

func TestFindProjectRootFrom_GitDirWithObjectsAccepted(t *testing.T) {
	// Regression guard: a .git directory containing an objects/ subdirectory
	// (e.g. bare repo style) must still be accepted.
	tmpDir := t.TempDir()

	gitDir := filepath.Join(tmpDir, ".git")
	objectsDir := filepath.Join(gitDir, "objects")
	if err := os.MkdirAll(objectsDir, 0755); err != nil {
		t.Fatalf("Failed to create .git/objects directory: %v", err)
	}

	root, err := FindProjectRootFrom(tmpDir, tmpDir)
	if err != nil {
		t.Fatalf("FindProjectRootFrom() error = %v", err)
	}

	if root != tmpDir {
		t.Errorf("FindProjectRootFrom() = %q, want %q (.git dir with objects/ must be accepted)", root, tmpDir)
	}
}

func TestFindProjectRootFrom_GitFileWorktreeAccepted(t *testing.T) {
	// Critical regression guard: in a git worktree, ".git" is a FILE (not a
	// directory) containing a "gitdir: <path>" pointer to the real repo's
	// worktree metadata. This must be accepted as-is, unmodified by the fix.
	tmpDir := t.TempDir()

	gitFile := filepath.Join(tmpDir, ".git")
	content := "gitdir: /path/to/real/.git/worktrees/branch-name\n"
	if err := os.WriteFile(gitFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create .git worktree file: %v", err)
	}

	root, err := FindProjectRootFrom(tmpDir, tmpDir)
	if err != nil {
		t.Fatalf("FindProjectRootFrom() error = %v", err)
	}

	if root != tmpDir {
		t.Errorf("FindProjectRootFrom() = %q, want %q (.git worktree file must be accepted unmodified)", root, tmpDir)
	}
}

func TestFindProjectRootFrom_EmptyGitFileNotAccepted(t *testing.T) {
	// B054: a stray, empty (or garbage-content) .git FILE must not be
	// accepted as a project-root marker. Without content validation,
	// FindProjectRootFrom wrongly treats any file named ".git" as a valid
	// worktree pointer regardless of content (e.g. `touch .git`).
	tmpDir := t.TempDir()

	gitFile := filepath.Join(tmpDir, ".git")
	if err := os.WriteFile(gitFile, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create empty .git file: %v", err)
	}

	startDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(startDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	root, err := FindProjectRootFrom(startDir, tmpDir)
	if err != nil {
		t.Fatalf("FindProjectRootFrom() error = %v", err)
	}

	if root != startDir {
		t.Errorf("FindProjectRootFrom() = %q, want %q (empty .git file must not be accepted as a marker)", root, startDir)
	}
}

func TestFindProjectRootFrom_GarbageGitFileNotAccepted(t *testing.T) {
	// B054: a .git file with garbage content (not "gitdir: <path>") must not
	// be accepted as a project-root marker.
	tmpDir := t.TempDir()

	gitFile := filepath.Join(tmpDir, ".git")
	if err := os.WriteFile(gitFile, []byte("not a real git marker\n"), 0644); err != nil {
		t.Fatalf("Failed to create garbage .git file: %v", err)
	}

	startDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(startDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	root, err := FindProjectRootFrom(startDir, tmpDir)
	if err != nil {
		t.Fatalf("FindProjectRootFrom() error = %v", err)
	}

	if root != startDir {
		t.Errorf("FindProjectRootFrom() = %q, want %q (garbage .git file must not be accepted as a marker)", root, startDir)
	}
}

func TestFindProjectRootFrom_NoMarkers(t *testing.T) {
	// Use a deeply nested subdir with no markers. Pass tmpDir as the ceiling so
	// the walk cannot escape into host-environment ancestors (e.g. /tmp/.sharkconfig.json).
	tmpDir := t.TempDir()
	subdir := filepath.Join(tmpDir, "some", "random", "path")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectories: %v", err)
	}

	root, err := FindProjectRootFrom(subdir, tmpDir)
	if err != nil {
		t.Fatalf("FindProjectRootFrom() error = %v", err)
	}

	if root != subdir {
		t.Errorf("FindProjectRootFrom() = %q, want %q (startDir when no markers found)", root, subdir)
	}
}
