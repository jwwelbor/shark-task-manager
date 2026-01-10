package taskcreation

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateCustomFilename_ValidPaths tests successful validation of valid paths
func TestValidateCustomFilename_ValidPaths(t *testing.T) {
	tests := []struct {
		name          string
		filename      string
		projectRoot   string
		expectAbsEnd  string // Expected end of absolute path (platform-agnostic)
		expectRelPath string
	}{
		{
			name:          "simple_markdown_file",
			filename:      "test.md",
			projectRoot:   "/project",
			expectRelPath: "test.md",
			expectAbsEnd:  filepath.Join("project", "test.md"),
		},
		{
			name:          "markdown_in_subdirectory",
			filename:      "docs/plan/task.md",
			projectRoot:   "/project",
			expectRelPath: filepath.Join("docs", "plan", "task.md"),
			expectAbsEnd:  filepath.Join("project", "docs", "plan", "task.md"),
		},
		{
			name:          "relative_path_with_dot",
			filename:      "./docs/task.md",
			projectRoot:   "/project",
			expectRelPath: filepath.Join("docs", "task.md"),
			expectAbsEnd:  filepath.Join("project", "docs", "task.md"),
		},
		{
			name:          "nested_directories",
			filename:      "docs/plan/E01/E01-F01/task.md",
			projectRoot:   "/project",
			expectRelPath: filepath.Join("docs", "plan", "E01", "E01-F01", "task.md"),
			expectAbsEnd:  filepath.Join("project", "docs", "plan", "E01", "E01-F01", "task.md"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			absPath, relPath, err := ValidateCustomFilename(tt.filename, tt.projectRoot)

			require.NoError(t, err)
			assert.NotEmpty(t, absPath)
			assert.Equal(t, tt.expectRelPath, relPath)
			// Check that absolute path ends with expected path (platform-agnostic)
			assert.True(t, strings.HasSuffix(absPath, filepath.FromSlash(tt.expectAbsEnd)),
				"Expected absolute path to end with %s, got %s", tt.expectAbsEnd, absPath)
		})
	}
}

// TestValidateCustomFilename_InvalidPaths tests rejection of invalid paths
func TestValidateCustomFilename_InvalidPaths(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		projectRoot string
		expectError string
	}{
		{
			name:        "absolute_path_rejected",
			filename:    "/absolute/path.md",
			projectRoot: "/project",
			expectError: "absolute path",
		},
		{
			name:        "path_traversal_double_dot",
			filename:    "../outside.md",
			projectRoot: "/project",
			expectError: "..",
		},
		{
			name:        "path_traversal_in_middle",
			filename:    "docs/../../../outside.md",
			projectRoot: "/project",
			expectError: "..",
		},
		{
			name:        "wrong_extension_txt",
			filename:    "file.txt",
			projectRoot: "/project",
			expectError: "file extension",
		},
		{
			name:        "wrong_extension_none",
			filename:    "file",
			projectRoot: "/project",
			expectError: "file extension",
		},
		{
			name:        "empty_filename",
			filename:    "",
			projectRoot: "/project",
			expectError: "file extension",
		},
		{
			name:        "dot_only",
			filename:    ".",
			projectRoot: "/project",
			expectError: "file extension",
		},
		{
			name:        "double_dot_only",
			filename:    "..",
			projectRoot: "/project",
			expectError: "..",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			absPath, relPath, err := ValidateCustomFilename(tt.filename, tt.projectRoot)

			assert.Error(t, err)
			assert.Empty(t, absPath)
			assert.Empty(t, relPath)
			assert.Contains(t, err.Error(), tt.expectError)
		})
	}
}

// TestValidateCustomFilename_PathNormalization tests path normalization
func TestValidateCustomFilename_PathNormalization(t *testing.T) {
	tests := []struct {
		name          string
		filename      string
		expectRelPath string
	}{
		{
			name:          "forward_slashes_normalized",
			filename:      "docs/plan/task.md",
			expectRelPath: filepath.Join("docs", "plan", "task.md"),
		},
		{
			name:          "mixed_slashes_normalized",
			filename:      "./docs/plan/task.md",
			expectRelPath: filepath.Join("docs", "plan", "task.md"),
		},
		{
			name:          "leading_dot_slash_removed",
			filename:      "./task.md",
			expectRelPath: "task.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, relPath, err := ValidateCustomFilename(tt.filename, "/project")

			require.NoError(t, err)
			assert.Equal(t, tt.expectRelPath, relPath)
		})
	}
}

// TestValidateCustomFilename_CasePreservation tests that case is preserved
func TestValidateCustomFilename_CasePreservation(t *testing.T) {
	_, relPath, err := ValidateCustomFilename("Docs/Plan/MyTask.md", "/project")

	require.NoError(t, err)
	// Case should be preserved in the relative path
	assert.Contains(t, relPath, "Docs")
	assert.Contains(t, relPath, "Plan")
	assert.Contains(t, relPath, "MyTask")
}

// TestValidateCustomFilename_SpecialCharacters tests handling of special characters
func TestValidateCustomFilename_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		valid    bool
	}{
		{
			name:     "hyphenated_filename",
			filename: "my-task-name.md",
			valid:    true,
		},
		{
			name:     "underscored_filename",
			filename: "my_task_name.md",
			valid:    true,
		},
		{
			name:     "numbered_filename",
			filename: "task-001.md",
			valid:    true,
		},
		{
			name:     "numbers_in_path",
			filename: "E01/E01-F01/001.md",
			valid:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ValidateCustomFilename(tt.filename, "/project")

			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// TestValidateCustomFilename_DeepNesting tests deeply nested paths
func TestValidateCustomFilename_DeepNesting(t *testing.T) {
	// Deep nesting should be valid
	_, relPath, err := ValidateCustomFilename(
		"docs/plan/E01/E01-F01/E01-F01-sub/task.md",
		"/project",
	)

	assert.NoError(t, err)
	assert.NotEmpty(t, relPath)
}

// TestValidateCustomFilename_AbsPathResolution tests absolute path resolution
func TestValidateCustomFilename_AbsPathResolution(t *testing.T) {
	absPath, _, err := ValidateCustomFilename("docs/task.md", "/project")

	require.NoError(t, err)
	// Absolute path should be absolute
	assert.True(t, filepath.IsAbs(absPath))
	// Should contain the project root
	assert.True(t, strings.HasPrefix(absPath, "/project") || absPath[0] == filepath.Separator)
}

// TestValidateCustomFilename_ConsistentResults tests that same input gives consistent output
func TestValidateCustomFilename_ConsistentResults(t *testing.T) {
	filename := "docs/plan/task.md"
	projectRoot := "/project"

	// Call multiple times
	absPath1, relPath1, err1 := ValidateCustomFilename(filename, projectRoot)
	absPath2, relPath2, err2 := ValidateCustomFilename(filename, projectRoot)

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, relPath1, relPath2)
	assert.Equal(t, absPath1, absPath2)
}

// TestCreator_FileExistsAssignsFile tests that when a file exists and --file is provided, it assigns the file
func TestCreator_FileExistsAssignsFile(t *testing.T) {
	ctx := context.Background()

	// Setup: Create temp directory for test workspace
	tempDir := t.TempDir()

	// Create a test markdown file that already exists
	existingFilePath := filepath.Join(tempDir, "docs", "existing-task.md")
	err := os.MkdirAll(filepath.Dir(existingFilePath), 0755)
	require.NoError(t, err)
	existingContent := []byte("# Existing Task\n\nThis file already exists.")
	err = os.WriteFile(existingFilePath, existingContent, 0644)
	require.NoError(t, err)

	// Setup: Create test database and repositories
	database := test.GetTestDB()
	_, err = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E98-%'")
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E98-%'")
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E98'")
	require.NoError(t, err)

	db := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	historyRepo := repository.NewTaskHistoryRepository(db)

	// Create test epic
	epic := &models.Epic{
		Key:      "E98",
		Title:    "Test Epic for File Exists",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}
	err = epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	// Create test feature
	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E98-F98",
		Title:  "Test Feature for File Exists",
		Status: models.FeatureStatusDraft,
	}
	err = featureRepo.Create(ctx, feature)
	require.NoError(t, err)

	// Setup: Create Creator
	keygen := NewKeyGenerator(taskRepo, featureRepo)
	validator := NewValidator(epicRepo, featureRepo, taskRepo)
	loader := templates.NewLoader("")
	renderer := templates.NewRenderer(loader)
	creator := NewCreator(db, keygen, validator, renderer, taskRepo, historyRepo, epicRepo, featureRepo, tempDir, nil)

	// Act: Create a task with custom filename pointing to existing file (no --create flag)
	input := CreateTaskInput{
		EpicKey:    "E98",
		FeatureKey: "F98",
		Title:      "Test Task with Existing File",
		AgentType:  "general",
		Priority:   5,
		Filename:   "docs/existing-task.md",
		// Note: Create flag is not set (defaults to false)
	}

	result, err := creator.CreateTask(ctx, input)

	// Assert: Task created successfully
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Task)

	// Assert: Task should reference the existing file
	require.NotNil(t, result.Task.FilePath)
	assert.Equal(t, filepath.Join("docs", "existing-task.md"), *result.Task.FilePath)

	// Assert: File content should remain unchanged (not overwritten)
	actualContent, err := os.ReadFile(existingFilePath)
	require.NoError(t, err)
	assert.Equal(t, existingContent, actualContent, "Existing file should not be overwritten")

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", result.Task.ID)
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestCreator_FileDoesNotExistWithCreateFlagCreatesFile tests that when file doesn't exist and --create is provided, it creates the file
func TestCreator_FileDoesNotExistWithCreateFlagCreatesFile(t *testing.T) {
	ctx := context.Background()

	// Setup: Create temp directory for test workspace
	tempDir := t.TempDir()

	// Ensure the file does NOT exist initially
	newFilePath := filepath.Join(tempDir, "docs", "new-task.md")
	_, err := os.Stat(newFilePath)
	assert.True(t, os.IsNotExist(err), "File should not exist initially")

	// Setup: Create test database and repositories
	database := test.GetTestDB()
	_, err = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E97-%'")
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E97-%'")
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E97'")
	require.NoError(t, err)

	db := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	historyRepo := repository.NewTaskHistoryRepository(db)

	// Create test epic
	epic := &models.Epic{
		Key:      "E97",
		Title:    "Test Epic for File Creation",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}
	err = epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	// Create test feature
	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E97-F97",
		Title:  "Test Feature for File Creation",
		Status: models.FeatureStatusDraft,
	}
	err = featureRepo.Create(ctx, feature)
	require.NoError(t, err)

	// Setup: Create Creator
	keygen := NewKeyGenerator(taskRepo, featureRepo)
	validator := NewValidator(epicRepo, featureRepo, taskRepo)
	loader := templates.NewLoader("")
	renderer := templates.NewRenderer(loader)
	creator := NewCreator(db, keygen, validator, renderer, taskRepo, historyRepo, epicRepo, featureRepo, tempDir, nil)

	// Act: Create a task with custom filename that doesn't exist WITH --create flag
	input := CreateTaskInput{
		EpicKey:    "E97",
		FeatureKey: "F97",
		Title:      "Test Task with New File",
		AgentType:  "general",
		Priority:   5,
		Filename:   "docs/new-task.md",
		Create:     true, // This field needs to be added to CreateTaskInput
	}

	result, err := creator.CreateTask(ctx, input)

	// Assert: Task created successfully
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Task)

	// Assert: Task should reference the new file
	require.NotNil(t, result.Task.FilePath)
	assert.Equal(t, filepath.Join("docs", "new-task.md"), *result.Task.FilePath)

	// Assert: File should have been created
	_, err = os.Stat(newFilePath)
	assert.NoError(t, err, "File should have been created")

	// Assert: File should contain task content (not be empty)
	content, err := os.ReadFile(newFilePath)
	require.NoError(t, err)
	assert.NotEmpty(t, content, "Created file should have content")
	assert.Contains(t, string(content), "Test Task with New File", "File should contain task title")

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", result.Task.ID)
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestCreator_FileDoesNotExistWithoutCreateFlagFails tests that when file doesn't exist and --create is NOT provided, it fails
func TestCreator_FileDoesNotExistWithoutCreateFlagFails(t *testing.T) {
	ctx := context.Background()

	// Setup: Create temp directory for test workspace
	tempDir := t.TempDir()

	// Ensure the file does NOT exist initially
	nonExistentFilePath := filepath.Join(tempDir, "docs", "nonexistent-task.md")
	_, err := os.Stat(nonExistentFilePath)
	assert.True(t, os.IsNotExist(err), "File should not exist initially")

	// Setup: Create test database and repositories
	database := test.GetTestDB()
	_, err = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E96-%'")
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E96-%'")
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E96'")
	require.NoError(t, err)

	db := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	historyRepo := repository.NewTaskHistoryRepository(db)

	// Create test epic
	epic := &models.Epic{
		Key:      "E96",
		Title:    "Test Epic for File Not Found",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}
	err = epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	// Create test feature
	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E96-F96",
		Title:  "Test Feature for File Not Found",
		Status: models.FeatureStatusDraft,
	}
	err = featureRepo.Create(ctx, feature)
	require.NoError(t, err)

	// Setup: Create Creator
	keygen := NewKeyGenerator(taskRepo, featureRepo)
	validator := NewValidator(epicRepo, featureRepo, taskRepo)
	loader := templates.NewLoader("")
	renderer := templates.NewRenderer(loader)
	creator := NewCreator(db, keygen, validator, renderer, taskRepo, historyRepo, epicRepo, featureRepo, tempDir, nil)

	// Act: Create a task with custom filename that doesn't exist WITHOUT --create flag
	input := CreateTaskInput{
		EpicKey:    "E96",
		FeatureKey: "F96",
		Title:      "Test Task with Missing File",
		AgentType:  "general",
		Priority:   5,
		Filename:   "docs/nonexistent-task.md",
		Create:     false, // Explicitly set to false (or omit, since it should default to false)
	}

	result, err := creator.CreateTask(ctx, input)

	// Assert: Task creation should fail
	assert.Error(t, err, "Should fail when file doesn't exist and --create flag is not provided")
	assert.Nil(t, result, "Result should be nil on error")
	assert.Contains(t, err.Error(), "does not exist", "Error should mention file doesn't exist")

	// Assert: File should NOT have been created
	_, err = os.Stat(nonExistentFilePath)
	assert.True(t, os.IsNotExist(err), "File should not have been created")

	// Assert: Task should NOT have been created in database
	tasks, err := taskRepo.List(ctx)
	require.NoError(t, err)
	for _, task := range tasks {
		assert.NotEqual(t, "Test Task with Missing File", task.Title, "Task should not exist in database")
	}

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

// TestCreator_UsesWorkflowConfigEntryStatus tests that new tasks use the first entry
// status from workflow config's special_statuses._start_ instead of hardcoded TaskStatusTodo
func TestCreator_UsesWorkflowConfigEntryStatus(t *testing.T) {
	ctx := context.Background()

	// Setup: Create temp directory for test workspace
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".sharkconfig.json")

	// Create workflow config with custom entry status
	workflowConfig := map[string]interface{}{
		"status_flow_version": "1.0",
		"special_statuses": map[string][]string{
			"_start_":    []string{"draft", "ready_for_development"},
			"_complete_": []string{"completed", "cancelled"},
		},
		"status_flow": map[string][]string{
			"draft":                 {"ready_for_refinement"},
			"ready_for_refinement":  {"in_refinement"},
			"in_refinement":         {"ready_for_development"},
			"ready_for_development": {"in_development"},
			"in_development":        {"completed"},
			"completed":             {},
		},
		"status_metadata": map[string]interface{}{
			"draft": map[string]interface{}{
				"color":       "gray",
				"description": "Initial draft status",
				"phase":       "planning",
			},
		},
	}

	configData, err := json.MarshalIndent(workflowConfig, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(configPath, configData, 0644)
	require.NoError(t, err)

	// Clear workflow cache to ensure fresh load
	config.ClearWorkflowCache()

	// Setup: Create test database and repositories
	database := test.GetTestDB()
	_, err = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E99-%'")
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E99-%'")
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E99'")
	require.NoError(t, err)

	db := repository.NewDB(database)
	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	historyRepo := repository.NewTaskHistoryRepository(db)

	// Create test epic
	epic := &models.Epic{
		Key:      "E99",
		Title:    "Test Epic for Workflow Config",
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}
	err = epicRepo.Create(ctx, epic)
	require.NoError(t, err)

	// Create test feature
	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E99-F99",
		Title:  "Test Feature for Workflow Config",
		Status: models.FeatureStatusDraft,
	}
	err = featureRepo.Create(ctx, feature)
	require.NoError(t, err)

	// Setup: Create Creator with config path
	keygen := NewKeyGenerator(taskRepo, featureRepo)
	validator := NewValidator(epicRepo, featureRepo, taskRepo)
	loader := templates.NewLoader("")
	renderer := templates.NewRenderer(loader)
	// Pass nil for workflowService - Creator will create one automatically from tempDir
	creator := NewCreator(db, keygen, validator, renderer, taskRepo, historyRepo, epicRepo, featureRepo, tempDir, nil)

	// Act: Create a new task
	input := CreateTaskInput{
		EpicKey:    "E99",
		FeatureKey: "F99",
		Title:      "Test Task for Workflow Config",
		AgentType:  "general",
		Priority:   5,
	}

	result, err := creator.CreateTask(ctx, input)

	// Assert: Task created successfully
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Task)

	// Assert: Task status should be "draft" (first entry status from workflow config)
	// NOT "todo" (hardcoded value)
	assert.Equal(t, "draft", string(result.Task.Status),
		"Task status should be 'draft' (first entry status from workflow config special_statuses._start_), not hardcoded 'todo'")

	// Cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", result.Task.ID)
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
	config.ClearWorkflowCache()
}
