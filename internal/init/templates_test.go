package init

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCopyTemplatesFreshCopy verifies that on a clean directory every embedded
// template is copied and counted as Copied (not Refreshed).
func TestCopyTemplatesFreshCopy(t *testing.T) {
	tempDir := t.TempDir()
	chdir(t, tempDir)

	initializer := NewInitializer()
	result, err := initializer.copyTemplates(false, "")
	if err != nil {
		t.Fatalf("copyTemplates() error = %v", err)
	}

	if result.Copied < 10 {
		t.Errorf("Copied = %d, want >= 10", result.Copied)
	}
	if result.Refreshed != 0 {
		t.Errorf("Refreshed = %d, want 0", result.Refreshed)
	}
	if len(result.Differed) != 0 {
		t.Errorf("Differed = %v, want empty", result.Differed)
	}

	if _, err := os.Stat(filepath.Join(tempDir, "shark-templates")); os.IsNotExist(err) {
		t.Error("shark-templates directory was not created")
	}
}

// TestCopyTemplatesSkipsIdentical verifies that running copy a second time
// against unmodified files reports zero copied/refreshed/differed.
func TestCopyTemplatesSkipsIdentical(t *testing.T) {
	tempDir := t.TempDir()
	chdir(t, tempDir)

	initializer := NewInitializer()
	if _, err := initializer.copyTemplates(false, ""); err != nil {
		t.Fatalf("first copy failed: %v", err)
	}

	result, err := initializer.copyTemplates(false, "")
	if err != nil {
		t.Fatalf("second copy failed: %v", err)
	}
	if result.Copied != 0 {
		t.Errorf("Copied on idempotent run = %d, want 0", result.Copied)
	}
	if result.Refreshed != 0 {
		t.Errorf("Refreshed on idempotent run = %d, want 0", result.Refreshed)
	}
	if len(result.Differed) != 0 {
		t.Errorf("Differed on idempotent run = %v, want empty", result.Differed)
	}
}

// TestCopyTemplatesDifferedWithoutForce verifies user customization is
// preserved and reported (but not overwritten) when force=false.
func TestCopyTemplatesDifferedWithoutForce(t *testing.T) {
	tempDir := t.TempDir()
	chdir(t, tempDir)

	// Plant a user-modified file ahead of the copy.
	entityDir := filepath.Join(tempDir, "shark-templates", "entity")
	if err := os.MkdirAll(entityDir, 0755); err != nil {
		t.Fatalf("setup mkdir failed: %v", err)
	}
	customPath := filepath.Join(entityDir, "task.md")
	customContent := []byte("user customized content")
	if err := os.WriteFile(customPath, customContent, 0644); err != nil {
		t.Fatalf("setup write failed: %v", err)
	}

	initializer := NewInitializer()
	result, err := initializer.copyTemplates(false, "")
	if err != nil {
		t.Fatalf("copyTemplates() error = %v", err)
	}

	if len(result.Differed) == 0 {
		t.Fatal("expected at least one differed file, got none")
	}

	// Differed paths are relative to the working directory (since templateDir
	// defaults to "shark-templates"). Match against the relative form.
	wantRel := filepath.Join("shark-templates", "entity", "task.md")
	foundCustom := false
	for _, p := range result.Differed {
		if p == wantRel {
			foundCustom = true
			break
		}
	}
	if !foundCustom {
		t.Errorf("Differed = %v, expected to contain %s", result.Differed, wantRel)
	}

	if result.Refreshed != 0 {
		t.Errorf("Refreshed = %d, want 0 without --force", result.Refreshed)
	}

	// Verify the user's content was NOT clobbered.
	got, err := os.ReadFile(customPath)
	if err != nil {
		t.Fatalf("failed to read custom file: %v", err)
	}
	if string(got) != string(customContent) {
		t.Errorf("user file was overwritten without --force\n  got:  %q\n  want: %q", got, customContent)
	}
}

// TestCopyTemplatesForceOverwritesDiffered verifies --force replaces user-modified
// files with the embedded version.
func TestCopyTemplatesForceOverwritesDiffered(t *testing.T) {
	tempDir := t.TempDir()
	chdir(t, tempDir)

	entityDir := filepath.Join(tempDir, "shark-templates", "entity")
	if err := os.MkdirAll(entityDir, 0755); err != nil {
		t.Fatalf("setup mkdir failed: %v", err)
	}
	customPath := filepath.Join(entityDir, "task.md")
	if err := os.WriteFile(customPath, []byte("user customized content"), 0644); err != nil {
		t.Fatalf("setup write failed: %v", err)
	}

	initializer := NewInitializer()
	result, err := initializer.copyTemplates(true, "")
	if err != nil {
		t.Fatalf("copyTemplates() error = %v", err)
	}

	if result.Refreshed < 1 {
		t.Errorf("Refreshed = %d, want >= 1", result.Refreshed)
	}
	if len(result.Differed) != 0 {
		t.Errorf("Differed with --force = %v, want empty", result.Differed)
	}

	got, err := os.ReadFile(customPath)
	if err != nil {
		t.Fatalf("failed to read overwritten file: %v", err)
	}
	if string(got) == "user customized content" {
		t.Error("--force did not overwrite the user's customized file")
	}
}

func TestCopyTemplatesIncludesAllSubdirectories(t *testing.T) {
	tempDir := t.TempDir()
	chdir(t, tempDir)

	initializer := NewInitializer()
	result, err := initializer.copyTemplates(false, "")
	if err != nil {
		t.Fatalf("copyTemplates() failed: %v", err)
	}
	if result.Copied == 0 {
		t.Fatal("No templates were copied")
	}

	templateDir := filepath.Join(tempDir, "shark-templates")
	expectedDirs := []string{
		"task_short",
		"feature_short",
		"epic_short",
		"bug",
		"change",
	}
	for _, dir := range expectedDirs {
		if _, err := os.Stat(filepath.Join(templateDir, dir)); os.IsNotExist(err) {
			t.Errorf("Expected subdirectory %s does not exist", dir)
		}
	}

	// Verify at least one .tmpl file exists in task_short/
	taskDir := filepath.Join(templateDir, "task_short")
	entries, err := os.ReadDir(taskDir)
	if err != nil {
		t.Fatalf("Failed to read task_short directory: %v", err)
	}
	hasTmplFile := false
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".tmpl" {
			hasTmplFile = true
			break
		}
	}
	if !hasTmplFile {
		t.Error("No .tmpl files found in task_short/ directory")
	}
}

func TestCopyTemplatesFilePermissions(t *testing.T) {
	tempDir := t.TempDir()
	chdir(t, tempDir)

	initializer := NewInitializer()
	result, err := initializer.copyTemplates(false, "")
	if err != nil {
		t.Fatalf("copyTemplates() failed: %v", err)
	}
	if result.Copied == 0 {
		t.Skip("No templates embedded, skipping permission check")
	}

	taskShortDir := filepath.Join(tempDir, "shark-templates", "task_short")
	entries, err := os.ReadDir(taskShortDir)
	if err != nil {
		t.Fatalf("Failed to read task_short directory: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			t.Errorf("Failed to get info for %s: %v", entry.Name(), err)
			continue
		}
		gotPerms := info.Mode().Perm()
		wantPerms := os.FileMode(0644)
		if gotPerms != wantPerms {
			t.Errorf("Template %s permissions = %o, want %o", entry.Name(), gotPerms, wantPerms)
		}
	}
}

func TestCopyTemplatesCustomDir(t *testing.T) {
	tempDir := t.TempDir()
	chdir(t, tempDir)

	customDir := "my-custom-templates"
	initializer := NewInitializer()
	result, err := initializer.copyTemplates(false, customDir)
	if err != nil {
		t.Fatalf("copyTemplates() failed: %v", err)
	}
	if result.Copied == 0 {
		t.Fatal("No templates were copied")
	}

	if _, err := os.Stat(filepath.Join(tempDir, customDir, "task_short")); os.IsNotExist(err) {
		t.Error("Task orchestrator templates not found in custom directory")
	}
	if _, err := os.Stat(filepath.Join(tempDir, customDir, "bug")); os.IsNotExist(err) {
		t.Error("Bug templates not found in custom directory")
	}
}

// chdir cd's into dir and registers a cleanup that restores the original wd.
func chdir(t *testing.T, dir string) {
	t.Helper()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDir) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
}
