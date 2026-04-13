package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// TC-F07-001: EditService.WriteFile — happy path
func TestEditService_WriteFile_HappyPath(t *testing.T) {
	root := t.TempDir()

	// Create file to overwrite
	docsDir := filepath.Join(root, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("setup: failed to create docs dir: %v", err)
	}
	filePath := filepath.Join(docsDir, "spec.md")
	if err := os.WriteFile(filePath, []byte("original content"), 0o644); err != nil {
		t.Fatalf("setup: failed to create test file: %v", err)
	}

	svc := NewEditService(root)
	content := "# Updated\n"
	result, err := svc.WriteFile(context.Background(), "docs/spec.md", content)
	if err != nil {
		t.Fatalf("WriteFile() unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("WriteFile() returned nil result")
	}
	if result.Path != "docs/spec.md" {
		t.Errorf("result.Path = %q, want %q", result.Path, "docs/spec.md")
	}
	if result.BytesWritten != len(content) {
		t.Errorf("result.BytesWritten = %d, want %d", result.BytesWritten, len(content))
	}

	// Verify file on disk
	got, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != content {
		t.Errorf("disk content = %q, want %q", string(got), content)
	}
}

// TC-F07-001 edge case: empty content (zero-byte write)
func TestEditService_WriteFile_EmptyContent(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "empty.md")
	if err := os.WriteFile(filePath, []byte("some data"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	svc := NewEditService(root)
	result, err := svc.WriteFile(context.Background(), "empty.md", "")
	if err != nil {
		t.Fatalf("WriteFile() unexpected error: %v", err)
	}
	if result.BytesWritten != 0 {
		t.Errorf("BytesWritten = %d, want 0", result.BytesWritten)
	}

	got, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("file content len = %d, want 0", len(got))
	}
}

// TC-F07-003: EditService.WriteFile — permission denied returns error
func TestEditService_WriteFile_PermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root; permission checks are not enforced")
	}

	root := t.TempDir()

	// Create a subdirectory that we'll make read-only to prevent writes.
	roDir := filepath.Join(root, "readonly-dir")
	if err := os.MkdirAll(roDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	filePath := filepath.Join(roDir, "target.md")
	if err := os.WriteFile(filePath, []byte("original"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Make the parent directory read-only so no new files (e.g. .tmp) can be created there.
	if err := os.Chmod(roDir, 0o555); err != nil {
		t.Fatalf("setup: chmod dir: %v", err)
	}
	defer os.Chmod(roDir, 0o755) // restore for cleanup

	svc := NewEditService(root)
	_, err := svc.WriteFile(context.Background(), "readonly-dir/target.md", "new content")
	if err == nil {
		t.Fatal("WriteFile() expected error for read-only directory, got nil")
	}

	// Must NOT be a SecurityError
	var secErr *SecurityError
	if asErr(err, &secErr) {
		t.Errorf("error is SecurityError, expected filesystem error; got: %v", err)
	}
}

// TC-F07-005: EditService.WriteFile — absolute path rejected immediately
func TestEditService_WriteFile_AbsolutePath(t *testing.T) {
	root := t.TempDir()
	svc := NewEditService(root)

	_, err := svc.WriteFile(context.Background(), "/etc/passwd", "evil")
	if err == nil {
		t.Fatal("WriteFile() expected SecurityError for absolute path, got nil")
	}

	var secErr *SecurityError
	if !asErr(err, &secErr) {
		t.Errorf("expected *SecurityError, got %T: %v", err, err)
	}
}

// TC-F07-006: EditService.WriteFile — `../` traversal outside root rejected
func TestEditService_WriteFile_TraversalOutsideRoot(t *testing.T) {
	root := t.TempDir()
	svc := NewEditService(root)

	// ../../outside.md resolves above the temp root
	_, err := svc.WriteFile(context.Background(), "../../outside.md", "evil")
	if err == nil {
		t.Fatal("WriteFile() expected SecurityError for traversal path, got nil")
	}

	var secErr *SecurityError
	if !asErr(err, &secErr) {
		t.Errorf("expected *SecurityError, got %T: %v", err, err)
	}
}

// TC-F07-006b: EditService.WriteFile — deep traversal to non-existent dir outside root
// Regression: before the pre-flight check, ../../../etc/passwd returned a generic error
// (500) instead of SecurityError (400) because EvalSymlinks on a non-existent parent
// returned os.ErrNotExist which was wrapped as a plain error.
func TestEditService_WriteFile_TraversalNonExistentParentOutsideRoot(t *testing.T) {
	root := t.TempDir()
	svc := NewEditService(root)

	// This path resolves above the root, and the intermediate directories don't exist.
	_, err := svc.WriteFile(context.Background(), "../../../etc/passwd", "evil")
	if err == nil {
		t.Fatal("WriteFile() expected SecurityError for deep traversal, got nil")
	}

	var secErr *SecurityError
	if !asErr(err, &secErr) {
		t.Errorf("expected *SecurityError, got %T: %v", err, err)
	}
}

// TC-F07-007: EditService.WriteFile — symlink resolving outside root rejected
func TestEditService_WriteFile_SymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outsideDir := t.TempDir()

	// Create a file outside root that the symlink would point to
	outsideFile := filepath.Join(outsideDir, "target.md")
	if err := os.WriteFile(outsideFile, []byte("outside"), 0o644); err != nil {
		t.Fatalf("setup: create outside file: %v", err)
	}

	// Create symlink inside root pointing to outside file
	symlinkPath := filepath.Join(root, "link.md")
	if err := os.Symlink(outsideFile, symlinkPath); err != nil {
		t.Fatalf("setup: create symlink: %v", err)
	}

	svc := NewEditService(root)
	_, err := svc.WriteFile(context.Background(), "link.md", "evil")
	if err == nil {
		t.Fatal("WriteFile() expected SecurityError for symlink escape, got nil")
	}

	var secErr *SecurityError
	if !asErr(err, &secErr) {
		t.Errorf("expected *SecurityError, got %T: %v", err, err)
	}

	// Verify the outside file was NOT modified
	got, readErr := os.ReadFile(outsideFile)
	if readErr != nil {
		t.Fatalf("ReadFile outside: %v", readErr)
	}
	if string(got) != "outside" {
		t.Errorf("outside file was modified; content = %q", string(got))
	}
}

// TC-F07-010: EditService.WriteFile — subdirectory path (standalone doc) succeeds
func TestEditService_WriteFile_SubdirectoryPath(t *testing.T) {
	root := t.TempDir()

	subDir := filepath.Join(root, "docs", "guide")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	introPath := filepath.Join(subDir, "intro.md")
	if err := os.WriteFile(introPath, []byte("old"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	svc := NewEditService(root)
	content := "# Guide\n"
	result, err := svc.WriteFile(context.Background(), "docs/guide/intro.md", content)
	if err != nil {
		t.Fatalf("WriteFile() unexpected error: %v", err)
	}

	if result.Path != "docs/guide/intro.md" {
		t.Errorf("result.Path = %q, want %q", result.Path, "docs/guide/intro.md")
	}

	got, err := os.ReadFile(introPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != content {
		t.Errorf("disk content = %q, want %q", string(got), content)
	}
}

// TC-F07-016: EditService.WriteFile — atomic: `.tmp` file does not persist
func TestEditService_WriteFile_AtomicCleanup(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "target.md")
	if err := os.WriteFile(filePath, []byte("original"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	svc := NewEditService(root)
	_, err := svc.WriteFile(context.Background(), "target.md", "updated content")
	if err != nil {
		t.Fatalf("WriteFile() unexpected error: %v", err)
	}

	// No .tmp file should remain after a successful write.
	// The temp file uses a random pattern so we scan the directory.
	entries, readErr := os.ReadDir(root)
	if readErr != nil {
		t.Fatalf("ReadDir: %v", readErr)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf(".tmp file still exists after successful write: %q", e.Name())
		}
	}
}

// TC-F07-017: EditService.WriteFile — concurrent writes do not corrupt file
func TestEditService_WriteFile_ConcurrentWrites(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "concurrent.md")
	if err := os.WriteFile(filePath, []byte("initial"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	svc := NewEditService(root)

	var wg sync.WaitGroup
	errs := make([]error, 10)
	for i := 0; i < 10; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			content := "goroutine content " + string(rune('0'+i)) + "\n"
			_, errs[i] = svc.WriteFile(context.Background(), "concurrent.md", content)
		}()
	}
	wg.Wait()

	// No goroutine should have returned an error
	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d: WriteFile() error: %v", i, err)
		}
	}

	// File should be readable and contain exactly one goroutine's content
	got, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile after concurrent writes: %v", err)
	}
	content := string(got)
	if len(content) == 0 {
		t.Error("file is empty after concurrent writes")
	}
	// File must end with newline (each writer appends \n), confirming no partial interleave
	if content[len(content)-1] != '\n' {
		t.Errorf("file content does not end with newline (possible corruption): %q", content)
	}
}

// asErr is a helper because errors.As requires a pointer-to-pointer but the
// *SecurityError type is already a pointer receiver. We use a local helper
// to avoid importing "errors" just for this check in the test.
func asErr(err error, target **SecurityError) bool {
	if err == nil {
		return false
	}
	if se, ok := err.(*SecurityError); ok {
		*target = se
		return true
	}
	// Also unwrap
	type unwrapper interface{ Unwrap() error }
	if u, ok := err.(unwrapper); ok {
		return asErr(u.Unwrap(), target)
	}
	return false
}
