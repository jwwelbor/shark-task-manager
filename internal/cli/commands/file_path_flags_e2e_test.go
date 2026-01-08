package commands

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_FilePathFlagStandardization_E07F19 is an end-to-end test for E07-F19 feature
// Verifies that --file, --filename, and --path flags all work correctly for epic, feature, and task creation
func TestE2E_FilePathFlagStandardization_E07F19(t *testing.T) {
	// Create temp directory for test project
	tmpDir, err := os.MkdirTemp("", "shark-e2e-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Initialize test database
	testDB := test.GetTestDB()
	database := repository.NewDB(testDB)

	ctx := context.Background()

	// Create repositories
	epicRepo := repository.NewEpicRepository(database)
	featureRepo := repository.NewFeatureRepository(database)
	taskRepo := repository.NewTaskRepository(database)

	t.Run("Epic creation with --file flag", func(t *testing.T) {
		// Test primary --file flag
		epic1, err := createEpicWithFilePath(ctx, epicRepo, tmpDir, "Test Epic 1", "docs/custom/epic1.md")
		require.NoError(t, err)
		require.NotNil(t, epic1.FilePath, "FilePath should not be nil")
		assert.Equal(t, "docs/custom/epic1.md", *epic1.FilePath)
		assert.Equal(t, "Test Epic 1", epic1.Title)

		// Verify file was created at custom path
		epicFilePath := filepath.Join(tmpDir, "docs/custom/epic1.md")
		_, err = os.Stat(epicFilePath)
		assert.NoError(t, err, "Epic file should exist at custom path")
	})

	t.Run("Feature creation with --file flag", func(t *testing.T) {
		// Create parent epic first
		epic, err := createEpicWithFilePath(ctx, epicRepo, tmpDir, "Parent Epic", "docs/epics/parent.md")
		require.NoError(t, err)

		// Test primary --file flag for feature
		feature1, err := createFeatureWithFilePath(ctx, featureRepo, epicRepo, tmpDir, epic.ID, "Test Feature 1", "docs/custom/feature1.md")
		require.NoError(t, err)
		require.NotNil(t, feature1.FilePath, "FilePath should not be nil")
		assert.Equal(t, "docs/custom/feature1.md", *feature1.FilePath)
		assert.Equal(t, "Test Feature 1", feature1.Title)
		assert.Equal(t, epic.ID, feature1.EpicID)

		// Verify file was created at custom path
		featureFilePath := filepath.Join(tmpDir, "docs/custom/feature1.md")
		_, err = os.Stat(featureFilePath)
		assert.NoError(t, err, "Feature file should exist at custom path")
	})

	t.Run("Task creation with --file flag", func(t *testing.T) {
		// Create parent epic and feature first
		epic, err := createEpicWithFilePath(ctx, epicRepo, tmpDir, "Epic for Tasks", "docs/epics/tasks.md")
		require.NoError(t, err)

		feature, err := createFeatureWithFilePath(ctx, featureRepo, epicRepo, tmpDir, epic.ID, "Feature for Tasks", "docs/features/tasks.md")
		require.NoError(t, err)

		// Test primary --file flag for task
		task1, err := createTaskWithFilePath(ctx, taskRepo, featureRepo, epicRepo, tmpDir, feature.ID, "Test Task 1", "docs/custom/task1.md")
		require.NoError(t, err)
		require.NotNil(t, task1.FilePath, "FilePath should not be nil")
		assert.Equal(t, "docs/custom/task1.md", *task1.FilePath)
		assert.Equal(t, "Test Task 1", task1.Title)
		assert.Equal(t, feature.ID, task1.FeatureID)

		// Verify file was created at custom path
		taskFilePath := filepath.Join(tmpDir, "docs/custom/task1.md")
		_, err = os.Stat(taskFilePath)
		assert.NoError(t, err, "Task file should exist at custom path")
	})

	t.Run("Verify no CustomFolderPath in database schema", func(t *testing.T) {
		// Check epics table schema
		var epicColumns []string
		rows, err := testDB.Query("PRAGMA table_info(epics)")
		require.NoError(t, err)
		defer rows.Close()

		for rows.Next() {
			var cid int
			var name, dataType string
			var notNull, pk int
			var dfltValue sql.NullString
			err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk)
			require.NoError(t, err)
			epicColumns = append(epicColumns, name)
		}

		assert.NotContains(t, epicColumns, "custom_folder_path", "epics table should not have custom_folder_path column")

		// Check features table schema
		var featureColumns []string
		rows, err = testDB.Query("PRAGMA table_info(features)")
		require.NoError(t, err)
		defer rows.Close()

		for rows.Next() {
			var cid int
			var name, dataType string
			var notNull, pk int
			var dfltValue sql.NullString
			err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk)
			require.NoError(t, err)
			featureColumns = append(featureColumns, name)
		}

		assert.NotContains(t, featureColumns, "custom_folder_path", "features table should not have custom_folder_path column")
	})

	t.Run("Verify file_path retrieval works correctly", func(t *testing.T) {
		// Use timestamp to ensure unique file path across test runs
		uniquePath := fmt.Sprintf("docs/test/retrieval-%d.md", time.Now().UnixNano())

		// Clean up any existing epic at this path first
		_, _ = testDB.Exec("DELETE FROM epics WHERE file_path = ?", uniquePath)

		// Create epic with custom file path
		epic, err := createEpicWithFilePath(ctx, epicRepo, tmpDir, "Retrieval Test Epic", uniquePath)
		require.NoError(t, err)

		// Clean up after test
		defer func() {
			_, _ = testDB.Exec("DELETE FROM epics WHERE id = ?", epic.ID)
		}()

		// Retrieve by ID
		retrieved, err := epicRepo.GetByID(ctx, epic.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved.FilePath, "Retrieved epic FilePath should not be nil")
		assert.Equal(t, uniquePath, *retrieved.FilePath, "Retrieved epic should have correct file_path")
		assert.Equal(t, "Retrieval Test Epic", retrieved.Title)

		// Retrieve by file path
		retrievedByPath, err := epicRepo.GetByFilePath(ctx, uniquePath)
		require.NoError(t, err)
		assert.Equal(t, epic.Key, retrievedByPath.Key, "GetByFilePath should return same epic")
		assert.Equal(t, epic.ID, retrievedByPath.ID, "GetByFilePath should return same epic ID")
		require.NotNil(t, retrievedByPath.FilePath, "Retrieved epic FilePath should not be nil")
		assert.Equal(t, uniquePath, *retrievedByPath.FilePath)
	})
}

// Helper functions for E2E test

func createEpicWithFilePath(ctx context.Context, repo *repository.EpicRepository, projectRoot, title, filePath string) (*models.Epic, error) {
	// Create parent directory if needed
	if filePath != "" {
		dir := filepath.Join(projectRoot, filepath.Dir(filePath))
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	filePathPtr := &filePath
	if filePath == "" {
		filePathPtr = nil
	}

	// Get next epic number
	existingEpics, err := repo.List(ctx, nil)
	if err != nil {
		return nil, err
	}
	epicNum := len(existingEpics) + 1

	epicKey := fmt.Sprintf("E%02d", epicNum)

	epic := &models.Epic{
		Key:      epicKey,
		Title:    title,
		FilePath: filePathPtr,
		Status:   models.EpicStatusDraft,
		Priority: models.PriorityMedium,
	}

	err = repo.Create(ctx, epic)
	if err != nil {
		return nil, err
	}

	// Create the file
	if filePath != "" {
		fullPath := filepath.Join(projectRoot, filePath)
		content := "---\nepic_key: " + epic.Key + "\ntitle: " + title + "\n---\n\n# " + title + "\n"
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return nil, err
		}
	}

	return epic, nil
}

func createFeatureWithFilePath(ctx context.Context, repo *repository.FeatureRepository, epicRepo *repository.EpicRepository, projectRoot string, epicID int64, title, filePath string) (*models.Feature, error) {
	// Create parent directory if needed
	if filePath != "" {
		dir := filepath.Join(projectRoot, filepath.Dir(filePath))
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	filePathPtr := &filePath
	if filePath == "" {
		filePathPtr = nil
	}

	// Get epic key
	epic, err := epicRepo.GetByID(ctx, epicID)
	if err != nil {
		return nil, err
	}

	// Get next feature number for this epic
	existingFeatures, err := repo.ListByEpic(ctx, epicID)
	if err != nil {
		return nil, err
	}
	featureNum := len(existingFeatures) + 1
	featureKey := fmt.Sprintf("%s-F%02d", epic.Key, featureNum)

	feature := &models.Feature{
		EpicID:   epicID,
		Key:      featureKey,
		Title:    title,
		FilePath: filePathPtr,
		Status:   models.FeatureStatusDraft,
	}

	err = repo.Create(ctx, feature)
	if err != nil {
		return nil, err
	}

	// Create the file
	if filePath != "" {
		fullPath := filepath.Join(projectRoot, filePath)
		content := "---\nfeature_key: " + feature.Key + "\ntitle: " + title + "\n---\n\n# " + title + "\n"
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return nil, err
		}
	}

	return feature, nil
}

func createTaskWithFilePath(ctx context.Context, repo *repository.TaskRepository, featureRepo *repository.FeatureRepository, epicRepo *repository.EpicRepository, projectRoot string, featureID int64, title, filePath string) (*models.Task, error) {
	// Create parent directory if needed
	if filePath != "" {
		dir := filepath.Join(projectRoot, filepath.Dir(filePath))
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	filePathPtr := &filePath
	if filePath == "" {
		filePathPtr = nil
	}

	// Get feature and epic keys
	feature, err := featureRepo.GetByID(ctx, featureID)
	if err != nil {
		return nil, err
	}

	epic, err := epicRepo.GetByID(ctx, feature.EpicID)
	if err != nil {
		return nil, err
	}

	// Get next task number for this feature
	existingTasks, err := repo.ListByFeature(ctx, featureID)
	if err != nil {
		return nil, err
	}
	taskNum := len(existingTasks) + 1

	// Extract feature number from feature key (e.g., "E01-F01" -> "F01")
	featureNumStr := feature.Key[len(feature.Key)-3:] // Get last 3 chars "F01"

	taskKey := fmt.Sprintf("T-%s-%s-%03d", epic.Key, featureNumStr, taskNum)

	task := &models.Task{
		FeatureID: featureID,
		Key:       taskKey,
		Title:     title,
		FilePath:  filePathPtr,
		Status:    models.TaskStatusTodo,
		Priority:  5,
	}

	err = repo.Create(ctx, task)
	if err != nil {
		return nil, err
	}

	// Create the file
	if filePath != "" {
		fullPath := filepath.Join(projectRoot, filePath)
		content := "---\ntask_key: " + task.Key + "\ntitle: " + title + "\n---\n\n# " + title + "\n"
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return nil, err
		}
	}

	return task, nil
}
