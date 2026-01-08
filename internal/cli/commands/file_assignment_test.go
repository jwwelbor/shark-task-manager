package commands

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// FileAssignmentEpicRepo provides minimal interface for file assignment testing
type FileAssignmentEpicRepo struct {
	GetByFilePathFunc func(ctx context.Context, filePath string) (*models.Epic, error)
	UpdateFunc        func(ctx context.Context, epic *models.Epic) error
}

func (m *FileAssignmentEpicRepo) GetByFilePath(ctx context.Context, filePath string) (*models.Epic, error) {
	if m.GetByFilePathFunc != nil {
		return m.GetByFilePathFunc(ctx, filePath)
	}
	return nil, nil
}

func (m *FileAssignmentEpicRepo) Update(ctx context.Context, epic *models.Epic) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, epic)
	}
	return nil
}

// FileAssignmentFeatureRepo provides minimal interface for file assignment testing
type FileAssignmentFeatureRepo struct {
	GetByFilePathFunc func(ctx context.Context, filePath string) (*models.Feature, error)
	UpdateFunc        func(ctx context.Context, feature *models.Feature) error
}

func (m *FileAssignmentFeatureRepo) GetByFilePath(ctx context.Context, filePath string) (*models.Feature, error) {
	if m.GetByFilePathFunc != nil {
		return m.GetByFilePathFunc(ctx, filePath)
	}
	return nil, nil
}

func (m *FileAssignmentFeatureRepo) Update(ctx context.Context, feature *models.Feature) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, feature)
	}
	return nil
}

// FileAssignmentTaskRepo provides minimal interface for file assignment testing
type FileAssignmentTaskRepo struct {
	GetByFilePathFunc func(ctx context.Context, filePath string) (*models.Task, error)
	UpdateFunc        func(ctx context.Context, task *models.Task) error
}

func (m *FileAssignmentTaskRepo) GetByFilePath(ctx context.Context, filePath string) (*models.Task, error) {
	if m.GetByFilePathFunc != nil {
		return m.GetByFilePathFunc(ctx, filePath)
	}
	return nil, nil
}

func (m *FileAssignmentTaskRepo) Update(ctx context.Context, task *models.Task) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, task)
	}
	return nil
}

// TestDetectFileCollision_NoCollision tests when no entity claims the file
func TestDetectFileCollision_NoCollision(t *testing.T) {
	ctx := context.Background()
	filePath := "docs/test/epic.md"

	// Mock repositories return nil (no collision)
	epicRepo := &FileAssignmentEpicRepo{
		GetByFilePathFunc: func(ctx context.Context, fp string) (*models.Epic, error) {
			return nil, nil
		},
	}
	featureRepo := &FileAssignmentFeatureRepo{
		GetByFilePathFunc: func(ctx context.Context, fp string) (*models.Feature, error) {
			return nil, nil
		},
	}
	taskRepo := &FileAssignmentTaskRepo{
		GetByFilePathFunc: func(ctx context.Context, fp string) (*models.Task, error) {
			return nil, nil
		},
	}

	// Execute
	collision, err := DetectFileCollision(ctx, filePath, epicRepo, featureRepo, taskRepo)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if collision != nil {
		t.Errorf("Expected no collision, got: %+v", collision)
	}
}

// TestDetectFileCollision_EpicClaimed tests when an epic claims the file
func TestDetectFileCollision_EpicClaimed(t *testing.T) {
	ctx := context.Background()
	filePath := "docs/test/epic.md"

	existingEpic := &models.Epic{
		ID:       1,
		Key:      "E01",
		Title:    "Existing Epic",
		FilePath: &filePath,
	}

	// Mock epic repository returns existing epic
	epicRepo := &FileAssignmentEpicRepo{
		GetByFilePathFunc: func(ctx context.Context, fp string) (*models.Epic, error) {
			if fp == filePath {
				return existingEpic, nil
			}
			return nil, nil
		},
	}
	featureRepo := &FileAssignmentFeatureRepo{
		GetByFilePathFunc: func(ctx context.Context, fp string) (*models.Feature, error) {
			return nil, nil
		},
	}
	taskRepo := &FileAssignmentTaskRepo{
		GetByFilePathFunc: func(ctx context.Context, fp string) (*models.Task, error) {
			return nil, nil
		},
	}

	// Execute
	collision, err := DetectFileCollision(ctx, filePath, epicRepo, featureRepo, taskRepo)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if collision == nil {
		t.Fatal("Expected collision, got nil")
	}
	if collision.FilePath != filePath {
		t.Errorf("Expected FilePath=%s, got: %s", filePath, collision.FilePath)
	}
	if collision.Epic == nil {
		t.Error("Expected Epic to be set, got nil")
	}
	if collision.Epic.Key != "E01" {
		t.Errorf("Expected Epic.Key=E01, got: %s", collision.Epic.Key)
	}
	if collision.Feature != nil {
		t.Errorf("Expected Feature to be nil, got: %+v", collision.Feature)
	}
	if collision.Task != nil {
		t.Errorf("Expected Task to be nil, got: %+v", collision.Task)
	}
}

// TestDetectFileCollision_FeatureClaimed tests when a feature claims the file
func TestDetectFileCollision_FeatureClaimed(t *testing.T) {
	ctx := context.Background()
	filePath := "docs/test/feature.md"

	existingFeature := &models.Feature{
		ID:       1,
		Key:      "E01-F01",
		Title:    "Existing Feature",
		FilePath: &filePath,
	}

	epicRepo := &FileAssignmentEpicRepo{
		GetByFilePathFunc: func(ctx context.Context, fp string) (*models.Epic, error) {
			return nil, nil
		},
	}
	featureRepo := &FileAssignmentFeatureRepo{
		GetByFilePathFunc: func(ctx context.Context, fp string) (*models.Feature, error) {
			if fp == filePath {
				return existingFeature, nil
			}
			return nil, nil
		},
	}
	taskRepo := &FileAssignmentTaskRepo{
		GetByFilePathFunc: func(ctx context.Context, fp string) (*models.Task, error) {
			return nil, nil
		},
	}

	// Execute
	collision, err := DetectFileCollision(ctx, filePath, epicRepo, featureRepo, taskRepo)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if collision == nil {
		t.Fatal("Expected collision, got nil")
	}
	if collision.FilePath != filePath {
		t.Errorf("Expected FilePath=%s, got: %s", filePath, collision.FilePath)
	}
	if collision.Epic != nil {
		t.Errorf("Expected Epic to be nil, got: %+v", collision.Epic)
	}
	if collision.Feature == nil {
		t.Error("Expected Feature to be set, got nil")
	}
	if collision.Feature.Key != "E01-F01" {
		t.Errorf("Expected Feature.Key=E01-F01, got: %s", collision.Feature.Key)
	}
	if collision.Task != nil {
		t.Errorf("Expected Task to be nil, got: %+v", collision.Task)
	}
}

// TestDetectFileCollision_TaskClaimed tests when a task claims the file
func TestDetectFileCollision_TaskClaimed(t *testing.T) {
	ctx := context.Background()
	filePath := "docs/test/task.md"

	existingTask := &models.Task{
		ID:       1,
		Key:      "T-E01-F01-001",
		Title:    "Existing Task",
		FilePath: &filePath,
	}

	epicRepo := &FileAssignmentEpicRepo{
		GetByFilePathFunc: func(ctx context.Context, fp string) (*models.Epic, error) {
			return nil, nil
		},
	}
	featureRepo := &FileAssignmentFeatureRepo{
		GetByFilePathFunc: func(ctx context.Context, fp string) (*models.Feature, error) {
			return nil, nil
		},
	}
	taskRepo := &FileAssignmentTaskRepo{
		GetByFilePathFunc: func(ctx context.Context, fp string) (*models.Task, error) {
			if fp == filePath {
				return existingTask, nil
			}
			return nil, nil
		},
	}

	// Execute
	collision, err := DetectFileCollision(ctx, filePath, epicRepo, featureRepo, taskRepo)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if collision == nil {
		t.Fatal("Expected collision, got nil")
	}
	if collision.FilePath != filePath {
		t.Errorf("Expected FilePath=%s, got: %s", filePath, collision.FilePath)
	}
	if collision.Epic != nil {
		t.Errorf("Expected Epic to be nil, got: %+v", collision.Epic)
	}
	if collision.Feature != nil {
		t.Errorf("Expected Feature to be nil, got: %+v", collision.Feature)
	}
	if collision.Task == nil {
		t.Error("Expected Task to be set, got nil")
	}
	if collision.Task.Key != "T-E01-F01-001" {
		t.Errorf("Expected Task.Key=T-E01-F01-001, got: %s", collision.Task.Key)
	}
}

// TestDetectFileCollision_RepositoryError tests handling of repository errors
func TestDetectFileCollision_RepositoryError(t *testing.T) {
	ctx := context.Background()
	filePath := "docs/test/epic.md"
	testErr := errors.New("database connection failed")

	tests := []struct {
		name        string
		epicRepo    *FileAssignmentEpicRepo
		featureRepo *FileAssignmentFeatureRepo
		taskRepo    *FileAssignmentTaskRepo
		wantErr     bool
	}{
		{
			name: "Epic repository error",
			epicRepo: &FileAssignmentEpicRepo{
				GetByFilePathFunc: func(ctx context.Context, fp string) (*models.Epic, error) {
					return nil, testErr
				},
			},
			featureRepo: &FileAssignmentFeatureRepo{},
			taskRepo:    &FileAssignmentTaskRepo{},
			wantErr:     true,
		},
		{
			name:     "Feature repository error",
			epicRepo: &FileAssignmentEpicRepo{},
			featureRepo: &FileAssignmentFeatureRepo{
				GetByFilePathFunc: func(ctx context.Context, fp string) (*models.Feature, error) {
					return nil, testErr
				},
			},
			taskRepo: &FileAssignmentTaskRepo{},
			wantErr:  true,
		},
		{
			name:        "Task repository error",
			epicRepo:    &FileAssignmentEpicRepo{},
			featureRepo: &FileAssignmentFeatureRepo{},
			taskRepo: &FileAssignmentTaskRepo{
				GetByFilePathFunc: func(ctx context.Context, fp string) (*models.Task, error) {
					return nil, testErr
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DetectFileCollision(ctx, filePath, tt.epicRepo, tt.featureRepo, tt.taskRepo)
			if (err != nil) != tt.wantErr {
				t.Errorf("Expected error=%v, got error=%v", tt.wantErr, err)
			}
			if err != nil && !strings.Contains(err.Error(), "failed to check") {
				t.Errorf("Expected error message to contain 'failed to check', got: %v", err)
			}
		})
	}
}

// TestHandleFileReassignment_NoCollision tests pass-through when no collision exists
func TestHandleFileReassignment_NoCollision(t *testing.T) {
	ctx := context.Background()

	epicRepo := &FileAssignmentEpicRepo{}
	featureRepo := &FileAssignmentFeatureRepo{}
	taskRepo := &FileAssignmentTaskRepo{}

	// No collision
	err := HandleFileReassignment(ctx, nil, false, epicRepo, featureRepo, taskRepo)

	if err != nil {
		t.Errorf("Expected no error for nil collision, got: %v", err)
	}
}

// TestHandleFileReassignment_WithoutForce tests error when collision exists and force=false
func TestHandleFileReassignment_WithoutForce(t *testing.T) {
	ctx := context.Background()
	filePath := "docs/test/epic.md"

	tests := []struct {
		name      string
		collision *FileCollision
		wantErr   string
	}{
		{
			name: "Epic collision without force",
			collision: &FileCollision{
				FilePath: filePath,
				Epic: &models.Epic{
					Key:   "E01",
					Title: "Existing Epic",
				},
			},
			wantErr: "already claimed by epic E01",
		},
		{
			name: "Feature collision without force",
			collision: &FileCollision{
				FilePath: filePath,
				Feature: &models.Feature{
					Key:   "E01-F01",
					Title: "Existing Feature",
				},
			},
			wantErr: "already claimed by feature E01-F01",
		},
		{
			name: "Task collision without force",
			collision: &FileCollision{
				FilePath: filePath,
				Task: &models.Task{
					Key:   "T-E01-F01-001",
					Title: "Existing Task",
				},
			},
			wantErr: "already claimed by task T-E01-F01-001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			epicRepo := &FileAssignmentEpicRepo{}
			featureRepo := &FileAssignmentFeatureRepo{}
			taskRepo := &FileAssignmentTaskRepo{}

			err := HandleFileReassignment(ctx, tt.collision, false, epicRepo, featureRepo, taskRepo)

			if err == nil {
				t.Fatal("Expected error when collision exists and force=false")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Expected error to contain %q, got: %v", tt.wantErr, err)
			}
			if !strings.Contains(err.Error(), "--force") {
				t.Errorf("Expected error to mention --force flag, got: %v", err)
			}
		})
	}
}

// TestHandleFileReassignment_WithForce_Epic tests reassignment from epic when force=true
func TestHandleFileReassignment_WithForce_Epic(t *testing.T) {
	ctx := context.Background()
	filePath := "docs/test/epic.md"

	existingEpic := &models.Epic{
		ID:       1,
		Key:      "E01",
		Title:    "Existing Epic",
		FilePath: &filePath,
	}

	updateCalled := false
	epicRepo := &FileAssignmentEpicRepo{
		UpdateFunc: func(ctx context.Context, epic *models.Epic) error {
			updateCalled = true
			if epic.ID != existingEpic.ID {
				t.Errorf("Expected epic ID=%d, got: %d", existingEpic.ID, epic.ID)
			}
			if epic.FilePath != nil {
				t.Errorf("Expected FilePath to be nil after reassignment, got: %v", *epic.FilePath)
			}
			return nil
		},
	}
	featureRepo := &FileAssignmentFeatureRepo{}
	taskRepo := &FileAssignmentTaskRepo{}

	collision := &FileCollision{
		FilePath: filePath,
		Epic:     existingEpic,
	}

	// Execute with force=true
	err := HandleFileReassignment(ctx, collision, true, epicRepo, featureRepo, taskRepo)

	// Assert
	if err != nil {
		t.Errorf("Expected no error with force=true, got: %v", err)
	}
	if !updateCalled {
		t.Error("Expected epic Update to be called")
	}
}

// TestHandleFileReassignment_WithForce_Feature tests reassignment from feature when force=true
func TestHandleFileReassignment_WithForce_Feature(t *testing.T) {
	ctx := context.Background()
	filePath := "docs/test/feature.md"

	existingFeature := &models.Feature{
		ID:       1,
		Key:      "E01-F01",
		Title:    "Existing Feature",
		FilePath: &filePath,
	}

	updateCalled := false
	epicRepo := &FileAssignmentEpicRepo{}
	featureRepo := &FileAssignmentFeatureRepo{
		UpdateFunc: func(ctx context.Context, feature *models.Feature) error {
			updateCalled = true
			if feature.ID != existingFeature.ID {
				t.Errorf("Expected feature ID=%d, got: %d", existingFeature.ID, feature.ID)
			}
			if feature.FilePath != nil {
				t.Errorf("Expected FilePath to be nil after reassignment, got: %v", *feature.FilePath)
			}
			return nil
		},
	}
	taskRepo := &FileAssignmentTaskRepo{}

	collision := &FileCollision{
		FilePath: filePath,
		Feature:  existingFeature,
	}

	// Execute with force=true
	err := HandleFileReassignment(ctx, collision, true, epicRepo, featureRepo, taskRepo)

	// Assert
	if err != nil {
		t.Errorf("Expected no error with force=true, got: %v", err)
	}
	if !updateCalled {
		t.Error("Expected feature Update to be called")
	}
}

// TestHandleFileReassignment_WithForce_Task tests reassignment from task when force=true
func TestHandleFileReassignment_WithForce_Task(t *testing.T) {
	ctx := context.Background()
	filePath := "docs/test/task.md"

	existingTask := &models.Task{
		ID:       1,
		Key:      "T-E01-F01-001",
		Title:    "Existing Task",
		FilePath: &filePath,
	}

	updateCalled := false
	epicRepo := &FileAssignmentEpicRepo{}
	featureRepo := &FileAssignmentFeatureRepo{}
	taskRepo := &FileAssignmentTaskRepo{
		UpdateFunc: func(ctx context.Context, task *models.Task) error {
			updateCalled = true
			if task.ID != existingTask.ID {
				t.Errorf("Expected task ID=%d, got: %d", existingTask.ID, task.ID)
			}
			if task.FilePath != nil {
				t.Errorf("Expected FilePath to be nil after reassignment, got: %v", *task.FilePath)
			}
			return nil
		},
	}

	collision := &FileCollision{
		FilePath: filePath,
		Task:     existingTask,
	}

	// Execute with force=true
	err := HandleFileReassignment(ctx, collision, true, epicRepo, featureRepo, taskRepo)

	// Assert
	if err != nil {
		t.Errorf("Expected no error with force=true, got: %v", err)
	}
	if !updateCalled {
		t.Error("Expected task Update to be called")
	}
}

// TestHandleFileReassignment_UpdateError tests handling of update errors
func TestHandleFileReassignment_UpdateError(t *testing.T) {
	ctx := context.Background()
	filePath := "docs/test/epic.md"
	testErr := errors.New("database update failed")

	existingEpic := &models.Epic{
		ID:       1,
		Key:      "E01",
		Title:    "Existing Epic",
		FilePath: &filePath,
	}

	epicRepo := &FileAssignmentEpicRepo{
		UpdateFunc: func(ctx context.Context, epic *models.Epic) error {
			return testErr
		},
	}
	featureRepo := &FileAssignmentFeatureRepo{}
	taskRepo := &FileAssignmentTaskRepo{}

	collision := &FileCollision{
		FilePath: filePath,
		Epic:     existingEpic,
	}

	err := HandleFileReassignment(ctx, collision, true, epicRepo, featureRepo, taskRepo)

	if err == nil {
		t.Fatal("Expected error when update fails")
	}
	if !strings.Contains(err.Error(), "failed to clear file path") {
		t.Errorf("Expected error to contain 'failed to clear file path', got: %v", err)
	}
}

// TestCreateBackupIfForce_NotForced tests that no backup is created when force=false
func TestCreateBackupIfForce_NotForced(t *testing.T) {
	backupPath, err := CreateBackupIfForce(false, "shark-tasks.db", "test-operation")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if backupPath != "" {
		t.Errorf("Expected empty backup path when force=false, got: %s", backupPath)
	}
}

// TestCreateBackupIfForce_Forced tests backup creation when force=true
func TestCreateBackupIfForce_Forced(t *testing.T) {
	// Create a temporary database file for testing
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "shark-tasks.db")

	// Create a dummy database file
	err := os.WriteFile(dbPath, []byte("test database content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Execute
	backupPath, err := CreateBackupIfForce(true, dbPath, "test-operation")

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if backupPath == "" {
		t.Fatal("Expected backup path to be set when force=true")
	}
	if !strings.HasSuffix(backupPath, ".backup") {
		t.Errorf("Expected backup path to end with .backup, got: %s", backupPath)
	}
	if !strings.Contains(backupPath, "test-operation") {
		t.Errorf("Expected backup path to contain operation name, got: %s", backupPath)
	}

	// Verify backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Errorf("Expected backup file to exist at: %s", backupPath)
	}

	// Verify backup content matches original
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Errorf("Failed to read backup file: %v", err)
	}
	if string(backupContent) != "test database content" {
		t.Errorf("Backup content doesn't match original")
	}
}

// TestCreateBackupIfForce_NonexistentDB tests error handling for missing database
func TestCreateBackupIfForce_NonexistentDB(t *testing.T) {
	dbPath := "/nonexistent/path/shark-tasks.db"

	backupPath, err := CreateBackupIfForce(true, dbPath, "test-operation")

	if err == nil {
		t.Fatal("Expected error when database doesn't exist")
	}
	if backupPath != "" {
		t.Errorf("Expected empty backup path on error, got: %s", backupPath)
	}
	if !strings.Contains(err.Error(), "failed to create backup") {
		t.Errorf("Expected error to contain 'failed to create backup', got: %v", err)
	}
}

// TestGetAbsoluteFilePath tests conversion of relative paths to absolute paths
func TestGetAbsoluteFilePath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantAbs bool
		wantErr bool
	}{
		{
			name:    "Relative path",
			input:   "docs/test/file.md",
			wantAbs: true,
			wantErr: false,
		},
		{
			name:    "Absolute path unchanged",
			input:   "/tmp/absolute/path.md",
			wantAbs: true,
			wantErr: false,
		},
		{
			name:    "Dot relative path",
			input:   "./file.md",
			wantAbs: true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetAbsoluteFilePath(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("Expected error=%v, got error=%v", tt.wantErr, err)
			}

			if !tt.wantErr {
				if !filepath.IsAbs(result) && tt.wantAbs {
					t.Errorf("Expected absolute path, got: %s", result)
				}
				// For absolute input, should return unchanged
				if filepath.IsAbs(tt.input) && result != tt.input {
					t.Errorf("Expected absolute path to be unchanged, got: %s", result)
				}
			}
		})
	}
}
