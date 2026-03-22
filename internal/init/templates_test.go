package init

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyTemplates(t *testing.T) {
	tests := []struct {
		name      string
		force     bool
		setupFunc func(string) error
		wantMin   int // Minimum expected count
		wantErr   bool
	}{
		{
			name:    "copies templates to new directory",
			force:   false,
			wantMin: 10, // We have many templates now (entity + orchestrator + partials)
			wantErr: false,
		},
		{
			name:  "skips existing templates without force",
			force: false,
			setupFunc: func(baseDir string) error {
				// Create entity templates directory with existing file
				entityDir := filepath.Join(baseDir, "shark-templates", "entity")
				if err := os.MkdirAll(entityDir, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(entityDir, "task.md"), []byte("existing"), 0644)
			},
			wantMin: 10, // Still copies many new files, just skips the one that exists
			wantErr: false,
		},
		{
			name:  "overwrites existing templates with force",
			force: true,
			setupFunc: func(baseDir string) error {
				entityDir := filepath.Join(baseDir, "shark-templates", "entity")
				if err := os.MkdirAll(entityDir, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(entityDir, "task.md"), []byte("existing"), 0644)
			},
			wantMin: 10, // Overwrites existing + copies all others
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tempDir := t.TempDir()
			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get working directory: %v", err)
			}
			defer func() { _ = os.Chdir(originalDir) }()

			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}

			// Setup
			if tt.setupFunc != nil {
				if err := tt.setupFunc(tempDir); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			// Execute
			initializer := NewInitializer()
			count, err := initializer.copyTemplates(tt.force, "")

			// Assert
			if (err != nil) != tt.wantErr {
				t.Errorf("copyTemplates() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if count < tt.wantMin {
				t.Errorf("copyTemplates() count = %d, want at least %d", count, tt.wantMin)
			}

			// Verify templates directory exists
			templateDir := filepath.Join(tempDir, "shark-templates")
			if _, err := os.Stat(templateDir); os.IsNotExist(err) {
				t.Error("Templates directory does not exist")
			}
		})
	}
}

func TestCopyTemplatesCreatesDirectory(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	initializer := NewInitializer()
	_, err = initializer.copyTemplates(false, "")
	if err != nil {
		t.Fatalf("copyTemplates() failed: %v", err)
	}

	// Verify templates directory was created
	templateDir := filepath.Join(tempDir, "shark-templates")
	info, err := os.Stat(templateDir)
	if err != nil {
		t.Fatalf("Templates directory does not exist: %v", err)
	}

	if !info.IsDir() {
		t.Error("shark-templates is not a directory")
	}
}

func TestCopyTemplatesIncludesAllSubdirectories(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	initializer := NewInitializer()
	count, err := initializer.copyTemplates(false, "")
	if err != nil {
		t.Fatalf("copyTemplates() failed: %v", err)
	}

	if count == 0 {
		t.Fatal("No templates were copied")
	}

	// Verify key subdirectories were created
	templateDir := filepath.Join(tempDir, "shark-templates")
	expectedDirs := []string{
		"task_short",
		"feature_short",
		"epic_short",
		"bug",
		"change",
	}

	for _, dir := range expectedDirs {
		dirPath := filepath.Join(templateDir, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
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
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	initializer := NewInitializer()
	count, err := initializer.copyTemplates(false, "")
	if err != nil {
		t.Fatalf("copyTemplates() failed: %v", err)
	}

	if count == 0 {
		t.Skip("No templates embedded, skipping permission check")
	}

	// Check permissions on copied files in task_short/ subdirectory
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
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	customDir := "my-custom-templates"
	initializer := NewInitializer()
	count, err := initializer.copyTemplates(false, customDir)
	if err != nil {
		t.Fatalf("copyTemplates() failed: %v", err)
	}

	if count == 0 {
		t.Fatal("No templates were copied")
	}

	// Verify custom directory was used
	if _, err := os.Stat(filepath.Join(tempDir, customDir, "task_short")); os.IsNotExist(err) {
		t.Error("Task orchestrator templates not found in custom directory")
	}

	if _, err := os.Stat(filepath.Join(tempDir, customDir, "bug")); os.IsNotExist(err) {
		t.Error("Bug templates not found in custom directory")
	}
}
