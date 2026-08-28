package dbinit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindProjectRoot_SharkConfig(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub", "dir")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Create .sharkconfig.json at the root.
	if err := os.WriteFile(filepath.Join(dir, ".sharkconfig.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	root, err := findProjectRootFrom(sub, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != dir {
		t.Errorf("root = %q, want %q", root, dir)
	}
}

func TestFindProjectRoot_SharkDB(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "shark-tasks.db"), []byte(""), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	root, err := findProjectRootFrom(sub, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != dir {
		t.Errorf("root = %q, want %q", root, dir)
	}
}

func TestFindProjectRoot_ConfigPreferredOverDB(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// DB in sub, config in root — config should win.
	if err := os.WriteFile(filepath.Join(sub, "shark-tasks.db"), []byte(""), 0o644); err != nil {
		t.Fatalf("write db: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".sharkconfig.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	root, err := findProjectRootFrom(sub, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != dir {
		t.Errorf("root = %q (config dir), got %q", dir, root)
	}
}

func TestFindProjectRoot_EmptyStartDir_UsesWorkDir(t *testing.T) {
	// Should not error when startDir is "".
	root, err := findProjectRoot("")
	if err != nil {
		t.Fatalf("unexpected error with empty startDir: %v", err)
	}
	if root == "" {
		t.Error("expected non-empty root when startDir is empty")
	}
}

// TestFindProjectRoot_EmptyGitDir_NotAcceptedAsMarker guards against B054: a
// stray, empty .git directory (no HEAD file, no objects/ dir) must NOT be
// accepted as a project-root marker. Without content validation, findProjectRoot
// would treat any directory containing a bare "mkdir .git" as a project root,
// which can point database initialization at the wrong location.
func TestFindProjectRoot_EmptyGitDir_NotAcceptedAsMarker(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub", "dir")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Empty .git dir — no HEAD, no objects/. Should NOT be treated as a marker.
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	// ceiling = dir keeps the walk from escaping into the host filesystem: without
	// it, this test would otherwise walk unbounded past dir, which could pick up a
	// real marker above it (e.g. in a sandbox where TMPDIR resolves under a real
	// project root) and fail/pass for the wrong reason.
	root, err := findProjectRootFrom(sub, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root == dir {
		t.Errorf("root = %q, empty .git dir must not be accepted as a marker (B054)", root)
	}
	if root != sub {
		t.Errorf("root = %q, want %q (fallback to start dir since no valid marker exists)", root, sub)
	}
}

// TestFindProjectRoot_GitDirWithHEAD_Accepted mirrors internal/cli's
// findProjectRootFrom regression coverage: a .git directory containing a HEAD
// file is a valid marker.
func TestFindProjectRoot_GitDirWithHEAD_Accepted(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub", "dir")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
		t.Fatalf("write HEAD: %v", err)
	}

	root, err := findProjectRootFrom(sub, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != dir {
		t.Errorf("root = %q, want %q (.git dir with HEAD should be accepted)", root, dir)
	}
}

// TestFindProjectRoot_GitDirWithObjects_Accepted mirrors internal/cli's
// findProjectRootFrom regression coverage: a .git directory containing an
// objects/ directory is a valid marker.
func TestFindProjectRoot_GitDirWithObjects_Accepted(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub", "dir")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(filepath.Join(gitDir, "objects"), 0o755); err != nil {
		t.Fatalf("mkdir .git/objects: %v", err)
	}

	root, err := findProjectRootFrom(sub, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != dir {
		t.Errorf("root = %q, want %q (.git dir with objects/ should be accepted)", root, dir)
	}
}

// TestFindProjectRoot_GitFile_WorktreePointer_AcceptedUnmodified mirrors
// internal/cli's findProjectRootFrom regression coverage: a .git file (git
// worktree pointer, e.g. "gitdir: /path/to/real/.git") is always accepted as
// a marker without content validation.
func TestFindProjectRoot_GitFile_WorktreePointer_AcceptedUnmodified(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub", "dir")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: /some/other/path/.git\n"), 0o644); err != nil {
		t.Fatalf("write .git file: %v", err)
	}

	root, err := findProjectRootFrom(sub, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != dir {
		t.Errorf("root = %q, want %q (.git worktree file should be accepted unmodified)", root, dir)
	}
}
