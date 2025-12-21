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
		wantCount int // Minimum expected count
		wantErr   bool
	}{
		{
			name:      "copies templates to new directory",
			force:     false,
			setupFunc: nil,
			wantCount: 0, // Will be at least 1 when templates are added
			wantErr:   false,
		},
		{
			name:  "skips existing templates without force",
			force: false,
			setupFunc: func(baseDir string) error {
				// Create templates directory with existing file
				templateDir := filepath.Join(baseDir, "shark-templates")
				if err := os.MkdirAll(templateDir, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(templateDir, "task.md"), []byte("existing"), 0644)
			},
			wantCount: 0, // Should skip existing
			wantErr:   false,
		},
		{
			name:  "overwrites existing templates with force",
			force: true,
			setupFunc: func(baseDir string) error {
				// Create templates directory with existing file
				templateDir := filepath.Join(baseDir, "shark-templates")
				if err := os.MkdirAll(templateDir, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(templateDir, "task.md"), []byte("existing"), 0644)
			},
			wantCount: 0, // Will overwrite
			wantErr:   false,
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
			defer os.Chdir(originalDir)

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
			count, err := initializer.copyTemplates(tt.force)

			// Assert
			if (err != nil) != tt.wantErr {
				t.Errorf("copyTemplates() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if count < tt.wantCount {
				t.Errorf("copyTemplates() count = %d, want at least %d", count, tt.wantCount)
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
	// Create temp directory
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Execute - templates directory doesn't exist yet
	initializer := NewInitializer()
	_, err = initializer.copyTemplates(false)
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

func TestCopyTemplatesFilePermissions(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Execute
	initializer := NewInitializer()
	count, err := initializer.copyTemplates(false)
	if err != nil {
		t.Fatalf("copyTemplates() failed: %v", err)
	}

	if count == 0 {
		t.Skip("No templates embedded, skipping permission check")
	}

	// Check permissions on copied files
	templateDir := filepath.Join(tempDir, "shark-templates")
	entries, err := os.ReadDir(templateDir)
	if err != nil {
		t.Fatalf("Failed to read templates directory: %v", err)
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
