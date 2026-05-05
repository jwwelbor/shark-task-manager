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

	root, err := findProjectRoot(sub)
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

	root, err := findProjectRoot(sub)
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

	root, err := findProjectRoot(sub)
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
