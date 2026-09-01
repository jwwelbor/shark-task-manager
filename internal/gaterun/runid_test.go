package gaterun

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateRunID(t *testing.T) {
	valid := []string{"a", "run-1", "Run_ID.2", "0123456789"}
	for _, id := range valid {
		if err := ValidateRunID(id); err != nil {
			t.Errorf("ValidateRunID(%q) = %v, want nil", id, err)
		}
	}

	invalid := []string{
		"",
		".",
		"..",
		"/etc/passwd",
		"../escape",
		"a/b",
		"a\\b",
		"-leading-dash",
		".leading-dot",
	}
	for _, id := range invalid {
		if err := ValidateRunID(id); err == nil {
			t.Errorf("ValidateRunID(%q) = nil, want error", id)
		}
	}
}

func TestRunDir_RejectsUnsafeRunID(t *testing.T) {
	root := t.TempDir()
	if _, err := RunDir(root, "../escape"); err == nil {
		t.Fatal("RunDir with escaping run id: want error, got nil")
	}
}

func TestRunDir_CreatesOwnerOnlyLeaf(t *testing.T) {
	root := t.TempDir()
	dir, err := RunDir(root, "run-1")
	if err != nil {
		t.Fatalf("RunDir: %v", err)
	}
	fi, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat run dir: %v", err)
	}
	if !fi.IsDir() {
		t.Fatalf("run dir %s is not a directory", dir)
	}
	if perm := fi.Mode().Perm(); perm != 0o700 {
		t.Errorf("run dir mode = %o, want 0700", perm)
	}
}

func TestRunDir_IdempotentOnExistingRealDir(t *testing.T) {
	root := t.TempDir()
	dir1, err := RunDir(root, "run-1")
	if err != nil {
		t.Fatalf("RunDir first call: %v", err)
	}
	dir2, err := RunDir(root, "run-1")
	if err != nil {
		t.Fatalf("RunDir second call: %v", err)
	}
	if dir1 != dir2 {
		t.Errorf("RunDir not stable: %s vs %s", dir1, dir2)
	}
}

func TestRunDir_RejectsSymlinkedLeaf(t *testing.T) {
	root := t.TempDir()
	runsDir := filepath.Join(root, ".shark", "runs")
	if err := os.MkdirAll(runsDir, 0o755); err != nil {
		t.Fatalf("mkdir runs dir: %v", err)
	}
	elsewhere := t.TempDir()
	link := filepath.Join(runsDir, "run-1")
	if err := os.Symlink(elsewhere, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if _, err := RunDir(root, "run-1"); err == nil {
		t.Fatal("RunDir over symlinked leaf: want error, got nil")
	}
}

func TestRunDir_RejectsSymlinkedIntermediate(t *testing.T) {
	root := t.TempDir()
	elsewhere := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".shark"), 0o755); err != nil {
		t.Fatalf("mkdir .shark: %v", err)
	}
	link := filepath.Join(root, ".shark", "runs")
	if err := os.Symlink(elsewhere, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if _, err := RunDir(root, "run-1"); err == nil {
		t.Fatal("RunDir over symlinked intermediate: want error, got nil")
	}
}

func TestRunDir_RejectsNonDirectoryLeaf(t *testing.T) {
	root := t.TempDir()
	runsDir := filepath.Join(root, ".shark", "runs")
	if err := os.MkdirAll(runsDir, 0o755); err != nil {
		t.Fatalf("mkdir runs dir: %v", err)
	}
	leaf := filepath.Join(runsDir, "run-1")
	if err := os.WriteFile(leaf, []byte("not a dir"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := RunDir(root, "run-1"); err == nil {
		t.Fatal("RunDir over a regular-file leaf: want error, got nil")
	}
}

func TestRunDir_EmptyProjectRoot(t *testing.T) {
	if _, err := RunDir("", "run-1"); err == nil {
		t.Fatal("RunDir with empty project root: want error, got nil")
	}
}
