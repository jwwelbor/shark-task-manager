package init

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateFolders(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(string) error
		wantFolders   []string
		wantErr       bool
		expectedPerms os.FileMode
	}{
		{
			name:          "creates all folders successfully",
			setupFunc:     nil,
			wantFolders:   []string{"docs/plan", "shark-templates"},
			wantErr:       false,
			expectedPerms: 0755,
		},
		{
			name: "idempotent - folders already exist",
			setupFunc: func(baseDir string) error {
				// Create folders first
				folders := []string{"docs/plan", "shark-templates"}
				for _, folder := range folders {
					if err := os.MkdirAll(filepath.Join(baseDir, folder), 0755); err != nil {
						return err
					}
				}
				return nil
			},
			wantFolders: []string{}, // No new folders created
			wantErr:     false,
		},
		{
			name:          "creates nested folders",
			setupFunc:     nil,
			wantFolders:   []string{"docs/plan", "shark-templates"},
			wantErr:       false,
			expectedPerms: 0755,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory and change to it
			tempDir := t.TempDir()
			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get working directory: %v", err)
			}
			defer func() {
				_ = os.Chdir(originalDir)
			}()

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
			created, err := initializer.createFolders()

			// Assert
			if (err != nil) != tt.wantErr {
				t.Errorf("createFolders() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(created) != len(tt.wantFolders) {
				t.Errorf("createFolders() created %d folders, want %d", len(created), len(tt.wantFolders))
			}

			// Verify all expected folders exist
			expectedFolders := []string{"docs/plan", "shark-templates"}
			for _, folder := range expectedFolders {
				folderPath := filepath.Join(tempDir, folder)
				info, err := os.Stat(folderPath)
				if err != nil {
					t.Errorf("Folder %s does not exist: %v", folder, err)
					continue
				}

				if !info.IsDir() {
					t.Errorf("%s is not a directory", folder)
				}

				// Check permissions (only for newly created folders)
				if len(tt.wantFolders) > 0 {
					gotPerms := info.Mode().Perm()
					if gotPerms != tt.expectedPerms {
						t.Errorf("Folder %s permissions = %o, want %o", folder, gotPerms, tt.expectedPerms)
					}
				}
			}
		})
	}
}

func TestCreateFoldersInvalidPath(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() {

		_ = os.Chdir(originalDir)

	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create a file where docs/plan should be
	if err := os.WriteFile("docs", []byte("block"), 0644); err != nil {
		t.Fatalf("Failed to create blocking file: %v", err)
	}

	// Execute
	initializer := NewInitializer()
	_, err = initializer.createFolders()

	// Should fail because 'docs' is a file, not a directory
	if err == nil {
		t.Error("createFolders() expected error when path is blocked, got nil")
	}
}

func TestCreateFoldersAbsolutePaths(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() {

		_ = os.Chdir(originalDir)

	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Execute
	initializer := NewInitializer()
	created, err := initializer.createFolders()
	if err != nil {
		t.Fatalf("createFolders() failed: %v", err)
	}

	// Verify returned paths are absolute
	for _, path := range created {
		if !filepath.IsAbs(path) {
			t.Errorf("createFolders() returned relative path %s, want absolute", path)
		}
	}
}
