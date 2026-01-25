package view

import (
	"context"
	"errors"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli/scope"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// Mock repositories for testing
type MockEpicRepository struct {
	GetByKeyFunc func(ctx context.Context, key string) (*models.Epic, error)
}

func (m *MockEpicRepository) GetByKey(ctx context.Context, key string) (*models.Epic, error) {
	if m.GetByKeyFunc != nil {
		return m.GetByKeyFunc(ctx, key)
	}
	return nil, errors.New("not implemented")
}

type MockFeatureRepository struct {
	GetByKeyFunc func(ctx context.Context, key string) (*models.Feature, error)
}

func (m *MockFeatureRepository) GetByKey(ctx context.Context, key string) (*models.Feature, error) {
	if m.GetByKeyFunc != nil {
		return m.GetByKeyFunc(ctx, key)
	}
	return nil, errors.New("not implemented")
}

type MockTaskRepository struct {
	GetByKeyFunc func(ctx context.Context, key string) (*models.Task, error)
}

func (m *MockTaskRepository) GetByKey(ctx context.Context, key string) (*models.Task, error) {
	if m.GetByKeyFunc != nil {
		return m.GetByKeyFunc(ctx, key)
	}
	return nil, errors.New("not implemented")
}

func TestService_GetFilePath_Epic(t *testing.T) {
	filePath := "docs/plan/E01-epic-name/epic.md"
	mockEpicRepo := &MockEpicRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				Key:      "E01",
				FilePath: &filePath,
			}, nil
		},
	}

	service := NewService(mockEpicRepo, nil, nil)

	testScope := &scope.Scope{
		Type: scope.ScopeEpic,
		Key:  "E01",
	}

	resultPath, err := service.GetFilePath(context.Background(), testScope)
	if err != nil {
		t.Fatalf("GetFilePath() error = %v", err)
	}

	expectedPath := "docs/plan/E01-epic-name/epic.md"
	if resultPath != expectedPath {
		t.Errorf("GetFilePath() = %v, want %v", resultPath, expectedPath)
	}
}

func TestService_GetFilePath_Feature(t *testing.T) {
	filePath := "docs/plan/E01-epic-name/E01-F01-feature-name/feature.md"
	mockFeatureRepo := &MockFeatureRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{
				Key:      "E01-F01",
				FilePath: &filePath,
			}, nil
		},
	}

	service := NewService(nil, mockFeatureRepo, nil)

	testScope := &scope.Scope{
		Type: scope.ScopeFeature,
		Key:  "E01-F01",
	}

	resultPath, err := service.GetFilePath(context.Background(), testScope)
	if err != nil {
		t.Fatalf("GetFilePath() error = %v", err)
	}

	expectedPath := "docs/plan/E01-epic-name/E01-F01-feature-name/feature.md"
	if resultPath != expectedPath {
		t.Errorf("GetFilePath() = %v, want %v", resultPath, expectedPath)
	}
}

func TestService_GetFilePath_Task(t *testing.T) {
	filePath := "docs/plan/E01-epic-name/E01-F01-feature-name/T-E01-F01-001.md"
	mockTaskRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{
				Key:      "T-E01-F01-001",
				FilePath: &filePath,
			}, nil
		},
	}

	service := NewService(nil, nil, mockTaskRepo)

	testScope := &scope.Scope{
		Type: scope.ScopeTask,
		Key:  "T-E01-F01-001",
	}

	resultPath, err := service.GetFilePath(context.Background(), testScope)
	if err != nil {
		t.Fatalf("GetFilePath() error = %v", err)
	}

	expectedPath := "docs/plan/E01-epic-name/E01-F01-feature-name/T-E01-F01-001.md"
	if resultPath != expectedPath {
		t.Errorf("GetFilePath() = %v, want %v", resultPath, expectedPath)
	}
}

func TestService_GetFilePath_NotFound(t *testing.T) {
	mockEpicRepo := &MockEpicRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, errors.New("epic not found: E99")
		},
	}

	service := NewService(mockEpicRepo, nil, nil)

	testScope := &scope.Scope{
		Type: scope.ScopeEpic,
		Key:  "E99",
	}

	_, err := service.GetFilePath(context.Background(), testScope)
	if err == nil {
		t.Fatal("GetFilePath() expected error, got nil")
	}
}

func TestService_GetFilePath_EmptyFilePath(t *testing.T) {
	emptyPath := ""
	mockEpicRepo := &MockEpicRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				Key:      "E01",
				FilePath: &emptyPath, // Empty file path
			}, nil
		},
	}

	service := NewService(mockEpicRepo, nil, nil)

	testScope := &scope.Scope{
		Type: scope.ScopeEpic,
		Key:  "E01",
	}

	_, err := service.GetFilePath(context.Background(), testScope)
	if err == nil {
		t.Fatal("GetFilePath() expected error for empty file path, got nil")
	}
}

func TestService_GetFilePath_NilFilePath(t *testing.T) {
	mockEpicRepo := &MockEpicRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				Key:      "E01",
				FilePath: nil, // Nil file path
			}, nil
		},
	}

	service := NewService(mockEpicRepo, nil, nil)

	testScope := &scope.Scope{
		Type: scope.ScopeEpic,
		Key:  "E01",
	}

	_, err := service.GetFilePath(context.Background(), testScope)
	if err == nil {
		t.Fatal("GetFilePath() expected error for nil file path, got nil")
	}
}

func TestService_GetFilePath_UnknownScope(t *testing.T) {
	service := NewService(nil, nil, nil)

	testScope := &scope.Scope{
		Type: scope.ScopeType("unknown"),
		Key:  "INVALID",
	}

	_, err := service.GetFilePath(context.Background(), testScope)
	if err == nil {
		t.Fatal("GetFilePath() expected error for unknown scope type, got nil")
	}
}

func TestService_LaunchViewer(t *testing.T) {
	service := NewService(nil, nil, nil)

	// Test with "cat" viewer (should work on most systems)
	err := service.LaunchViewer(context.Background(), "/dev/null", "cat")
	if err != nil {
		t.Errorf("LaunchViewer() with cat error = %v", err)
	}
}

func TestService_LaunchViewer_FileNotFound(t *testing.T) {
	service := NewService(nil, nil, nil)

	// Test with non-existent file
	err := service.LaunchViewer(context.Background(), "/non/existent/file.md", "cat")
	if err == nil {
		t.Error("LaunchViewer() expected error for non-existent file, got nil")
	}
}

func TestService_LaunchViewer_InvalidCommand(t *testing.T) {
	service := NewService(nil, nil, nil)

	// Test with invalid viewer command
	err := service.LaunchViewer(context.Background(), "/dev/null", "nonexistent-viewer-command-12345")
	if err == nil {
		t.Error("LaunchViewer() expected error for invalid command, got nil")
	}
}
