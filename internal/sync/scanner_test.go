package sync

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsTaskFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		// Valid task file patterns
		{
			name:     "valid task file with all digits",
			filename: "T-E04-F07-001.md",
			want:     true,
		},
		{
			name:     "valid task file with different numbers",
			filename: "T-E01-F02-003.md",
			want:     true,
		},
		{
			name:     "valid task file with high numbers",
			filename: "T-E99-F99-999.md",
			want:     true,
		},
		// Invalid patterns
		{
			name:     "missing T prefix",
			filename: "E04-F07-001.md",
			want:     false,
		},
		{
			name:     "wrong extension",
			filename: "T-E04-F07-001.txt",
			want:     false,
		},
		{
			name:     "single digit epic",
			filename: "T-E4-F07-001.md",
			want:     false,
		},
		{
			name:     "single digit feature",
			filename: "T-E04-F7-001.md",
			want:     false,
		},
		{
			name:     "single digit task number",
			filename: "T-E04-F07-1.md",
			want:     false,
		},
		{
			name:     "two digit task number",
			filename: "T-E04-F07-01.md",
			want:     false,
		},
		{
			name:     "lowercase letters",
			filename: "t-e04-f07-001.md",
			want:     false,
		},
		{
			name:     "no extension",
			filename: "T-E04-F07-001",
			want:     false,
		},
		{
			name:     "extra characters before",
			filename: "prefix-T-E04-F07-001.md",
			want:     false,
		},
		{
			name:     "extra characters after",
			filename: "T-E04-F07-001-suffix.md",
			want:     false,
		},
		{
			name:     "random markdown file",
			filename: "README.md",
			want:     false,
		},
	}

	scanner := NewFileScanner()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scanner.isTaskFile(tt.filename)
			if got != tt.want {
				t.Errorf("isTaskFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestInferEpicFeature(t *testing.T) {
	tests := []struct {
		name           string
		filePath       string
		wantEpicKey    string
		wantFeatureKey string
		wantErr        bool
	}{
		// Standard feature folder structure
		{
			name:           "standard structure with epic and feature folders",
			filePath:       "/project/docs/plan/E04-task-mgmt-cli-core/E04-F07-initialization-sync/T-E04-F07-001.md",
			wantEpicKey:    "E04",
			wantFeatureKey: "E04-F07",
			wantErr:        false,
		},
		{
			name:           "different epic and feature",
			filePath:       "/project/docs/plan/E01-user-auth/E01-F02-login/T-E01-F02-003.md",
			wantEpicKey:    "E01",
			wantFeatureKey: "E01-F02",
			wantErr:        false,
		},
		{
			name:           "nested in subfolder",
			filePath:       "/project/docs/plan/E04-epic/E04-F07-feature/tasks/T-E04-F07-001.md",
			wantEpicKey:    "E04",
			wantFeatureKey: "E04-F07",
			wantErr:        false,
		},
		// Legacy folder structure (docs/tasks/*)
		{
			name:           "legacy todo folder - extract from filename",
			filePath:       "/project/docs/tasks/todo/T-E04-F07-001.md",
			wantEpicKey:    "E04",
			wantFeatureKey: "E04-F07",
			wantErr:        false,
		},
		{
			name:           "legacy active folder - extract from filename",
			filePath:       "/project/docs/tasks/active/T-E01-F02-003.md",
			wantEpicKey:    "E01",
			wantFeatureKey: "E01-F02",
			wantErr:        false,
		},
		{
			name:           "legacy completed folder - extract from filename",
			filePath:       "/project/docs/tasks/completed/T-E04-F07-005.md",
			wantEpicKey:    "E04",
			wantFeatureKey: "E04-F07",
			wantErr:        false,
		},
		// Edge cases
		{
			name:           "root directory - extract from filename",
			filePath:       "/project/T-E04-F07-001.md",
			wantEpicKey:    "E04",
			wantFeatureKey: "E04-F07",
			wantErr:        false,
		},
		{
			name:           "relative path - extract from filename",
			filePath:       "docs/plan/E04-epic/E04-F07-feature/T-E04-F07-001.md",
			wantEpicKey:    "E04",
			wantFeatureKey: "E04-F07",
			wantErr:        false,
		},
	}

	scanner := NewFileScanner()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			epicKey, featureKey, err := scanner.inferEpicFeature(tt.filePath)

			if (err != nil) != tt.wantErr {
				t.Errorf("inferEpicFeature() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if epicKey != tt.wantEpicKey {
				t.Errorf("inferEpicFeature() epicKey = %v, want %v", epicKey, tt.wantEpicKey)
			}

			if featureKey != tt.wantFeatureKey {
				t.Errorf("inferEpicFeature() featureKey = %v, want %v", featureKey, tt.wantFeatureKey)
			}
		})
	}
}

func TestValidateFilePath(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create allowed subdirectories
	planDir := filepath.Join(tmpDir, "docs", "plan")
	tasksDir := filepath.Join(tmpDir, "docs", "tasks")
	templatesDir := filepath.Join(tmpDir, "shark-templates")

	os.MkdirAll(planDir, 0755)
	os.MkdirAll(tasksDir, 0755)
	os.MkdirAll(templatesDir, 0755)

	// Create a file outside allowed directories
	outsideDir := filepath.Join(tmpDir, "outside")
	os.MkdirAll(outsideDir, 0755)

	tests := []struct {
		name     string
		filePath string
		rootDir  string
		wantErr  bool
	}{
		{
			name:     "file in docs/plan is allowed",
			filePath: filepath.Join(planDir, "E04-epic", "T-E04-F07-001.md"),
			rootDir:  tmpDir,
			wantErr:  false,
		},
		{
			name:     "file in docs/tasks is allowed",
			filePath: filepath.Join(tasksDir, "todo", "T-E04-F07-001.md"),
			rootDir:  tmpDir,
			wantErr:  false,
		},
		{
			name:     "file in shark-templates is allowed",
			filePath: filepath.Join(templatesDir, "task-template.md"),
			rootDir:  tmpDir,
			wantErr:  false,
		},
		{
			name:     "file outside allowed directories is rejected",
			filePath: filepath.Join(outsideDir, "T-E04-F07-001.md"),
			rootDir:  tmpDir,
			wantErr:  true,
		},
		{
			name:     "path traversal attempt is rejected",
			filePath: filepath.Join(planDir, "..", "..", "..", "etc", "passwd"),
			rootDir:  tmpDir,
			wantErr:  true,
		},
		{
			name:     "absolute path to system file is rejected",
			filePath: "/etc/passwd",
			rootDir:  tmpDir,
			wantErr:  true,
		},
	}

	scanner := NewFileScanner()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := scanner.validateFilePath(tt.filePath, tt.rootDir)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateFilePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateFileSize(t *testing.T) {
	// Create temporary files with different sizes
	tmpDir := t.TempDir()

	// Create a small file (< 1MB)
	smallFile := filepath.Join(tmpDir, "small.md")
	smallContent := []byte("# Small file\n\nContent here")
	os.WriteFile(smallFile, smallContent, 0644)

	// Create a large file (> 1MB)
	largeFile := filepath.Join(tmpDir, "large.md")
	largeContent := make([]byte, 2*1024*1024) // 2MB
	for i := range largeContent {
		largeContent[i] = 'a'
	}
	os.WriteFile(largeFile, largeContent, 0644)

	tests := []struct {
		name     string
		filePath string
		wantErr  bool
	}{
		{
			name:     "small file is accepted",
			filePath: smallFile,
			wantErr:  false,
		},
		{
			name:     "large file is rejected",
			filePath: largeFile,
			wantErr:  true,
		},
		{
			name:     "non-existent file returns error",
			filePath: filepath.Join(tmpDir, "nonexistent.md"),
			wantErr:  true,
		},
	}

	scanner := NewFileScanner()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := scanner.validateFileSize(tt.filePath)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateFileSize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateFileIsRegular(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a regular file
	regularFile := filepath.Join(tmpDir, "regular.md")
	os.WriteFile(regularFile, []byte("content"), 0644)

	// Create a directory
	dirPath := filepath.Join(tmpDir, "directory")
	os.Mkdir(dirPath, 0755)

	// Create a symlink (only on Unix systems)
	symlinkPath := filepath.Join(tmpDir, "symlink.md")
	targetPath := filepath.Join(tmpDir, "target.md")
	os.WriteFile(targetPath, []byte("target"), 0644)
	_ = os.Symlink(targetPath, symlinkPath) // Ignore error on Windows

	tests := []struct {
		name     string
		filePath string
		wantErr  bool
	}{
		{
			name:     "regular file is accepted",
			filePath: regularFile,
			wantErr:  false,
		},
		{
			name:     "directory is rejected",
			filePath: dirPath,
			wantErr:  true,
		},
		{
			name:     "symlink is rejected",
			filePath: symlinkPath,
			wantErr:  true,
		},
		{
			name:     "non-existent file returns error",
			filePath: filepath.Join(tmpDir, "nonexistent.md"),
			wantErr:  true,
		},
	}

	scanner := NewFileScanner()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := scanner.validateFileIsRegular(tt.filePath)

			// Skip symlink test on Windows (not supported)
			if tt.name == "symlink is rejected" && os.Getenv("OS") == "Windows_NT" {
				t.Skip("Skipping symlink test on Windows")
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("validateFileIsRegular() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExtractKeyFromFilename(t *testing.T) {
	tests := []struct {
		name           string
		filename       string
		wantEpicKey    string
		wantFeatureKey string
	}{
		{
			name:           "standard task filename",
			filename:       "T-E04-F07-001.md",
			wantEpicKey:    "E04",
			wantFeatureKey: "E04-F07",
		},
		{
			name:           "different numbers",
			filename:       "T-E01-F02-003.md",
			wantEpicKey:    "E01",
			wantFeatureKey: "E01-F02",
		},
		{
			name:           "high numbers",
			filename:       "T-E99-F99-999.md",
			wantEpicKey:    "E99",
			wantFeatureKey: "E99-F99",
		},
	}

	scanner := NewFileScanner()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			epicKey, featureKey := scanner.extractKeyFromFilename(tt.filename)

			if epicKey != tt.wantEpicKey {
				t.Errorf("extractKeyFromFilename() epicKey = %v, want %v", epicKey, tt.wantEpicKey)
			}

			if featureKey != tt.wantFeatureKey {
				t.Errorf("extractKeyFromFilename() featureKey = %v, want %v", featureKey, tt.wantFeatureKey)
			}
		})
	}
}

func TestScan_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	scanner := NewFileScanner()
	files, err := scanner.Scan(tmpDir)

	if err != nil {
		t.Errorf("Scan() error = %v, want nil", err)
	}

	if len(files) != 0 {
		t.Errorf("Scan() returned %d files, want 0", len(files))
	}
}

func TestScan_NonexistentDirectory(t *testing.T) {
	scanner := NewFileScanner()
	_, err := scanner.Scan("/nonexistent/directory/path")

	if err == nil {
		t.Error("Scan() error = nil, want error for nonexistent directory")
	}
}

func TestScan_SingleFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a single task file
	taskFile := filepath.Join(tmpDir, "T-E04-F07-001.md")
	content := []byte("---\nkey: T-E04-F07-001\ntitle: Test Task\n---\n")
	os.WriteFile(taskFile, content, 0644)

	// Get file mod time for comparison
	info, _ := os.Stat(taskFile)
	expectedModTime := info.ModTime()

	scanner := NewFileScanner()
	files, err := scanner.Scan(tmpDir)

	if err != nil {
		t.Errorf("Scan() error = %v, want nil", err)
	}

	if len(files) != 1 {
		t.Fatalf("Scan() returned %d files, want 1", len(files))
	}

	file := files[0]

	// Verify all fields
	if file.FileName != "T-E04-F07-001.md" {
		t.Errorf("FileName = %v, want T-E04-F07-001.md", file.FileName)
	}

	if file.EpicKey != "E04" {
		t.Errorf("EpicKey = %v, want E04", file.EpicKey)
	}

	if file.FeatureKey != "E04-F07" {
		t.Errorf("FeatureKey = %v, want E04-F07", file.FeatureKey)
	}

	if file.FilePath != taskFile {
		t.Errorf("FilePath = %v, want %v", file.FilePath, taskFile)
	}

	// ModTime comparison (within 1 second tolerance)
	if file.ModifiedAt.Sub(expectedModTime) > time.Second {
		t.Errorf("ModifiedAt = %v, want %v", file.ModifiedAt, expectedModTime)
	}
}

func TestScan_IgnoresNonTaskFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create various files, only one is a valid task file
	files := []string{
		"T-E04-F07-001.md",      // Valid - should be found
		"README.md",              // Invalid - not a task file
		"notes.txt",              // Invalid - wrong extension
		"T-E04-F07-001.txt",      // Invalid - wrong extension
		"E04-F07-001.md",         // Invalid - missing T prefix
		".hidden.md",             // Invalid - hidden file
	}

	for _, file := range files {
		path := filepath.Join(tmpDir, file)
		os.WriteFile(path, []byte("content"), 0644)
	}

	scanner := NewFileScanner()
	results, err := scanner.Scan(tmpDir)

	if err != nil {
		t.Errorf("Scan() error = %v, want nil", err)
	}

	if len(results) != 1 {
		t.Errorf("Scan() returned %d files, want 1", len(results))
	}

	if len(results) > 0 && results[0].FileName != "T-E04-F07-001.md" {
		t.Errorf("Found file %v, want T-E04-F07-001.md", results[0].FileName)
	}
}

func TestScan_RecursiveTraversal(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested directory structure
	planDir := filepath.Join(tmpDir, "docs", "plan", "E04-epic", "E04-F07-feature")
	os.MkdirAll(planDir, 0755)

	tasksDir := filepath.Join(tmpDir, "docs", "tasks", "todo")
	os.MkdirAll(tasksDir, 0755)

	// Create task files in different locations
	file1 := filepath.Join(planDir, "T-E04-F07-001.md")
	file2 := filepath.Join(planDir, "T-E04-F07-002.md")
	file3 := filepath.Join(tasksDir, "T-E01-F01-001.md")

	os.WriteFile(file1, []byte("task 1"), 0644)
	os.WriteFile(file2, []byte("task 2"), 0644)
	os.WriteFile(file3, []byte("task 3"), 0644)

	scanner := NewFileScanner()
	results, err := scanner.Scan(tmpDir)

	if err != nil {
		t.Errorf("Scan() error = %v, want nil", err)
	}

	if len(results) != 3 {
		t.Errorf("Scan() returned %d files, want 3", len(results))
	}

	// Verify files were found
	foundFiles := make(map[string]bool)
	for _, result := range results {
		foundFiles[result.FileName] = true
	}

	expectedFiles := []string{"T-E04-F07-001.md", "T-E04-F07-002.md", "T-E01-F01-001.md"}
	for _, expected := range expectedFiles {
		if !foundFiles[expected] {
			t.Errorf("Expected file %v not found", expected)
		}
	}
}

func TestScan_RejectsOversizedFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a task file that exceeds size limit
	largeFile := filepath.Join(tmpDir, "T-E04-F07-001.md")
	largeContent := make([]byte, 2*1024*1024) // 2MB (over 1MB limit)
	for i := range largeContent {
		largeContent[i] = 'a'
	}
	os.WriteFile(largeFile, largeContent, 0644)

	// Create a normal-sized file
	normalFile := filepath.Join(tmpDir, "T-E04-F07-002.md")
	os.WriteFile(normalFile, []byte("normal content"), 0644)

	scanner := NewFileScanner()
	results, err := scanner.Scan(tmpDir)

	if err != nil {
		t.Errorf("Scan() error = %v, want nil", err)
	}

	// Should only find the normal-sized file
	if len(results) != 1 {
		t.Errorf("Scan() returned %d files, want 1", len(results))
	}

	if len(results) > 0 && results[0].FileName != "T-E04-F07-002.md" {
		t.Errorf("Found file %v, want T-E04-F07-002.md", results[0].FileName)
	}
}
