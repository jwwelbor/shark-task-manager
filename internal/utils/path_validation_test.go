package utils

import (
	"strings"
	"testing"
)

func TestValidateFolderPath(t *testing.T) {
	projectRoot := "/home/user/project"

	tests := []struct {
		name        string
		path        string
		projectRoot string
		wantErr     bool
		errType     error
		wantAbsPath string
		wantRelPath string
	}{
		{
			name:        "valid relative path",
			path:        "docs/custom",
			projectRoot: projectRoot,
			wantErr:     false,
			wantAbsPath: "/home/user/project/docs/custom",
			wantRelPath: "docs/custom",
		},
		{
			name:        "valid path with single level",
			path:        "custom",
			projectRoot: projectRoot,
			wantErr:     false,
			wantAbsPath: "/home/user/project/custom",
			wantRelPath: "custom",
		},
		{
			name:        "valid path with multiple levels",
			path:        "docs/roadmap/2025-q1",
			projectRoot: projectRoot,
			wantErr:     false,
			wantAbsPath: "/home/user/project/docs/roadmap/2025-q1",
			wantRelPath: "docs/roadmap/2025-q1",
		},
		{
			name:        "path with trailing slash normalized",
			path:        "docs/custom/",
			projectRoot: projectRoot,
			wantErr:     false,
			wantAbsPath: "/home/user/project/docs/custom",
			wantRelPath: "docs/custom",
		},
		{
			name:        "path with ./ normalized",
			path:        "./docs/custom",
			projectRoot: projectRoot,
			wantErr:     false,
			wantAbsPath: "/home/user/project/docs/custom",
			wantRelPath: "docs/custom",
		},
		{
			name:        "empty path error",
			path:        "",
			projectRoot: projectRoot,
			wantErr:     true,
			errType:     ErrEmptyPath,
		},
		{
			name:        "whitespace only path error",
			path:        "   ",
			projectRoot: projectRoot,
			wantErr:     true,
			errType:     ErrEmptyPath,
		},
		{
			name:        "absolute path error",
			path:        "/docs/custom",
			projectRoot: projectRoot,
			wantErr:     true,
			errType:     ErrAbsolutePath,
		},
		{
			name:        "path traversal error",
			path:        "docs/../etc",
			projectRoot: projectRoot,
			wantErr:     true,
			errType:     ErrPathTraversal,
		},
		{
			name:        "path traversal with .. error",
			path:        "docs/../../etc",
			projectRoot: projectRoot,
			wantErr:     true,
			errType:     ErrPathTraversal,
		},
		{
			name:        "path outside project error",
			path:        "../outside",
			projectRoot: projectRoot,
			wantErr:     true,
			errType:     ErrPathTraversal,
		},
		{
			name:        "path with hyphenated names",
			path:        "docs/roadmap/2025-q1/modules/oauth",
			projectRoot: projectRoot,
			wantErr:     false,
			wantAbsPath: "/home/user/project/docs/roadmap/2025-q1/modules/oauth",
			wantRelPath: "docs/roadmap/2025-q1/modules/oauth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			absPath, relPath, err := ValidateFolderPath(tt.path, tt.projectRoot)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFolderPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errType != nil {
				// Check if error message contains the expected error type
				if !strings.Contains(err.Error(), tt.errType.Error()) {
					t.Errorf("ValidateFolderPath() error = %v, want to contain %v", err, tt.errType)
				}
			}

			if !tt.wantErr {
				if absPath != tt.wantAbsPath {
					t.Errorf("ValidateFolderPath() absPath = %v, want %v", absPath, tt.wantAbsPath)
				}
				if relPath != tt.wantRelPath {
					t.Errorf("ValidateFolderPath() relPath = %v, want %v", relPath, tt.wantRelPath)
				}
			}
		})
	}
}

func TestValidateFolderPath_EdgeCases(t *testing.T) {
	projectRoot := "/project"

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "single dot normalized",
			path:    "./",
			wantErr: true, // Results in empty path after normalization
		},
		{
			name:    "multiple trailing slashes",
			path:    "docs///custom///",
			wantErr: false,
		},
		{
			name:    "consecutive dots in directory name",
			path:    "docs/2025..2026",
			wantErr: false, // Dots within directory name are OK (not ".." path component)
		},
		{
			name:    "single dot in path component",
			path:    "docs/2025.q1",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ValidateFolderPath(tt.path, projectRoot)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFolderPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
