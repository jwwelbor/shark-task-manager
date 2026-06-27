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
			wantFolders:   []string{"docs/plan"},
			wantErr:       false,
			expectedPerms: 0755,
		},
		{
			name: "idempotent - folders already exist",
			setupFunc: func(baseDir string) error {
				if err := os.MkdirAll(filepath.Join(baseDir, "docs/plan"), 0755); err != nil {
					return err
				}
				return nil
			},
			wantFolders: []string{}, // No new folders created
			wantErr:     false,
		},
		{
			name:          "creates nested folders",
			setupFunc:     nil,
			wantFolders:   []string{"docs/plan"},
			wantErr:       false,
			expectedPerms: 0755,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			if tt.setupFunc != nil {
				if err := tt.setupFunc(tempDir); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			initializer := NewInitializer()
			created, err := initializer.createFolders()

			if (err != nil) != tt.wantErr {
				t.Errorf("createFolders() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(created) != len(tt.wantFolders) {
				t.Errorf("createFolders() created %d folders, want %d", len(created), len(tt.wantFolders))
			}

			// Verify docs/plan was created
			docsPath := filepath.Join(tempDir, "docs/plan")
			info, err := os.Stat(docsPath)
			if err != nil {
				t.Errorf("docs/plan does not exist: %v", err)
			} else if !info.IsDir() {
				t.Error("docs/plan is not a directory")
			} else if len(tt.wantFolders) > 0 {
				gotPerms := info.Mode().Perm()
				if gotPerms != tt.expectedPerms {
					t.Errorf("docs/plan permissions = %o, want %o", gotPerms, tt.expectedPerms)
				}
			}

			// Verify the retired prompt tree is NOT created.
			retiredPromptTree := "shark" + "-templates"
			if _, err := os.Stat(filepath.Join(tempDir, retiredPromptTree)); err == nil {
				t.Error("retired prompt tree should not be created by shark admin init")
			}
		})
	}
}

func TestCreateFoldersInvalidPath(t *testing.T) {
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

	initializer := NewInitializer()
	_, err = initializer.createFolders()

	if err == nil {
		t.Error("createFolders() expected error when path is blocked, got nil")
	}
}

func TestCreateFoldersAbsolutePaths(t *testing.T) {
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

	initializer := NewInitializer()
	created, err := initializer.createFolders()
	if err != nil {
		t.Fatalf("createFolders() failed: %v", err)
	}

	for _, path := range created {
		if !filepath.IsAbs(path) {
			t.Errorf("createFolders() returned relative path %s, want absolute", path)
		}
	}
}
