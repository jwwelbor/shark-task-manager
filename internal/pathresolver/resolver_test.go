package pathresolver

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// Mock repositories for testing

type MockEpicRepository struct {
	GetByKeyFunc func(ctx context.Context, key string) (*models.Epic, error)
	GetByIDFunc  func(ctx context.Context, id int64) (*models.Epic, error)
}

func (m *MockEpicRepository) GetByKey(ctx context.Context, key string) (*models.Epic, error) {
	if m.GetByKeyFunc != nil {
		return m.GetByKeyFunc(ctx, key)
	}
	return nil, errors.New("not implemented")
}

func (m *MockEpicRepository) GetByID(ctx context.Context, id int64) (*models.Epic, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errors.New("not implemented")
}

type MockFeatureRepository struct {
	GetByKeyFunc func(ctx context.Context, key string) (*models.Feature, error)
	GetByIDFunc  func(ctx context.Context, id int64) (*models.Feature, error)
}

func (m *MockFeatureRepository) GetByKey(ctx context.Context, key string) (*models.Feature, error) {
	if m.GetByKeyFunc != nil {
		return m.GetByKeyFunc(ctx, key)
	}
	return nil, errors.New("not implemented")
}

func (m *MockFeatureRepository) GetByID(ctx context.Context, id int64) (*models.Feature, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errors.New("not implemented")
}

type MockTaskRepository struct {
	GetByKeyFunc func(ctx context.Context, key string) (*models.Task, error)
	GetByIDFunc  func(ctx context.Context, id int64) (*models.Task, error)
}

func (m *MockTaskRepository) GetByKey(ctx context.Context, key string) (*models.Task, error) {
	if m.GetByKeyFunc != nil {
		return m.GetByKeyFunc(ctx, key)
	}
	return nil, errors.New("not implemented")
}

func (m *MockTaskRepository) GetByID(ctx context.Context, id int64) (*models.Task, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errors.New("not implemented")
}

// TestResolveEpicPath_DefaultPath tests epic path resolution with default path
func TestResolveEpicPath_DefaultPath(t *testing.T) {
	ctx := context.Background()
	projectRoot := "/project"

	slug := "test-epic"
	mockEpicRepo := &MockEpicRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				ID:    1,
				Key:   "E01",
				Title: "Test Epic",
				Slug:  &slug,
			}, nil
		},
	}

	resolver := NewPathResolver(mockEpicRepo, nil, nil, projectRoot)
	path, err := resolver.ResolveEpicPath(ctx, "E01")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(projectRoot, "docs", "plan", "E01-test-epic", "epic.md")
	if path != expected {
		t.Errorf("expected path %s, got %s", expected, path)
	}
}

// TestResolveEpicPath_CustomFolderPath removed - custom_folder_path feature no longer supported

// TestResolveEpicPath_ExplicitFilename tests epic with explicit filename
func TestResolveEpicPath_ExplicitFilename(t *testing.T) {
	ctx := context.Background()
	projectRoot := "/project"

	filename := "docs/special/my-epic.md"
	mockEpicRepo := &MockEpicRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				ID:       1,
				Key:      "E01",
				Title:    "Test Epic",
				FilePath: &filename,
			}, nil
		},
	}

	resolver := NewPathResolver(mockEpicRepo, nil, nil, projectRoot)
	path, err := resolver.ResolveEpicPath(ctx, "E01")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(projectRoot, filename)
	if path != expected {
		t.Errorf("expected path %s, got %s", expected, path)
	}
}

// TestResolveEpicPath_NotFound tests error handling for non-existent epic
func TestResolveEpicPath_NotFound(t *testing.T) {
	ctx := context.Background()
	projectRoot := "/project"

	mockEpicRepo := &MockEpicRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, errors.New("epic not found")
		},
	}

	resolver := NewPathResolver(mockEpicRepo, nil, nil, projectRoot)
	_, err := resolver.ResolveEpicPath(ctx, "E99")

	if err == nil {
		t.Fatal("expected error for non-existent epic, got nil")
	}
}

// TestResolveFeaturePath_DefaultPath tests feature with default path
func TestResolveFeaturePath_DefaultPath(t *testing.T) {
	ctx := context.Background()
	projectRoot := "/project"

	epicSlug := "test-epic"
	featureSlug := "test-feature"

	mockEpicRepo := &MockEpicRepository{
		GetByIDFunc: func(ctx context.Context, id int64) (*models.Epic, error) {
			return &models.Epic{
				ID:    1,
				Key:   "E01",
				Title: "Test Epic",
				Slug:  &epicSlug,
			}, nil
		},
	}

	mockFeatureRepo := &MockFeatureRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{
				ID:     1,
				EpicID: 1,
				Key:    "E01-F01",
				Title:  "Test Feature",
				Slug:   &featureSlug,
			}, nil
		},
	}

	resolver := NewPathResolver(mockEpicRepo, mockFeatureRepo, nil, projectRoot)
	path, err := resolver.ResolveFeaturePath(ctx, "E01-F01")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(projectRoot, "docs", "plan", "E01-test-epic", "E01-F01-test-feature", "prd.md")
	if path != expected {
		t.Errorf("expected path %s, got %s", expected, path)
	}
}

// TestResolveFeaturePath_InheritedEpicPath removed - custom_folder_path feature no longer supported

// TestResolveFeaturePath_FeatureOverridePath removed - custom_folder_path feature no longer supported

// TestResolveFeaturePath_ExplicitFilename tests feature with explicit filename
func TestResolveFeaturePath_ExplicitFilename(t *testing.T) {
	ctx := context.Background()
	projectRoot := "/project"

	filename := "docs/custom/feature-spec.md"

	mockEpicRepo := &MockEpicRepository{
		GetByIDFunc: func(ctx context.Context, id int64) (*models.Epic, error) {
			return &models.Epic{
				ID:    1,
				Key:   "E01",
				Title: "Test Epic",
			}, nil
		},
	}

	mockFeatureRepo := &MockFeatureRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{
				ID:       1,
				EpicID:   1,
				Key:      "E01-F01",
				Title:    "Test Feature",
				FilePath: &filename,
			}, nil
		},
	}

	resolver := NewPathResolver(mockEpicRepo, mockFeatureRepo, nil, projectRoot)
	path, err := resolver.ResolveFeaturePath(ctx, "E01-F01")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(projectRoot, filename)
	if path != expected {
		t.Errorf("expected path %s, got %s", expected, path)
	}
}

// TestResolveTaskPath_DefaultPath tests task with default path
func TestResolveTaskPath_DefaultPath(t *testing.T) {
	ctx := context.Background()
	projectRoot := "/project"

	epicSlug := "test-epic"
	featureSlug := "test-feature"

	mockEpicRepo := &MockEpicRepository{
		GetByIDFunc: func(ctx context.Context, id int64) (*models.Epic, error) {
			return &models.Epic{
				ID:    1,
				Key:   "E01",
				Title: "Test Epic",
				Slug:  &epicSlug,
			}, nil
		},
	}

	mockFeatureRepo := &MockFeatureRepository{
		GetByIDFunc: func(ctx context.Context, id int64) (*models.Feature, error) {
			return &models.Feature{
				ID:     1,
				EpicID: 1,
				Key:    "E01-F01",
				Title:  "Test Feature",
				Slug:   &featureSlug,
			}, nil
		},
	}

	mockTaskRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			emptyPath := ""
			return &models.Task{
				ID:        1,
				FeatureID: 1,
				Key:       "T-E01-F01-001",
				Title:     "Test Task",
				FilePath:  &emptyPath,
			}, nil
		},
	}

	resolver := NewPathResolver(mockEpicRepo, mockFeatureRepo, mockTaskRepo, projectRoot)
	path, err := resolver.ResolveTaskPath(ctx, "T-E01-F01-001")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(projectRoot, "docs", "plan", "E01-test-epic", "E01-F01-test-feature", "tasks", "T-E01-F01-001.md")
	if path != expected {
		t.Errorf("expected path %s, got %s", expected, path)
	}
}

// TestResolveTaskPath_ExplicitFilePath tests task with explicit file path
func TestResolveTaskPath_ExplicitFilePath(t *testing.T) {
	ctx := context.Background()
	projectRoot := "/project"

	mockEpicRepo := &MockEpicRepository{}
	mockFeatureRepo := &MockFeatureRepository{}

	mockTaskRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			taskPath := "docs/custom/my-task.md"
			return &models.Task{
				ID:        1,
				FeatureID: 1,
				Key:       "T-E01-F01-001",
				Title:     "Test Task",
				FilePath:  &taskPath,
			}, nil
		},
	}

	resolver := NewPathResolver(mockEpicRepo, mockFeatureRepo, mockTaskRepo, projectRoot)
	path, err := resolver.ResolveTaskPath(ctx, "T-E01-F01-001")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(projectRoot, "docs/custom/my-task.md")
	if path != expected {
		t.Errorf("expected path %s, got %s", expected, path)
	}
}

// TestPathPrecedence tests that path precedence is correctly followed
func TestPathPrecedence_EpicWithAllOptions(t *testing.T) {
	ctx := context.Background()
	projectRoot := "/project"

	// Epic has both explicit filepath and slug - filepath should win
	explicitPath := "docs/explicit/epic.md"
	slug := "my-epic"

	mockEpicRepo := &MockEpicRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				ID:       1,
				Key:      "E01",
				Title:    "Test Epic",
				FilePath: &explicitPath,
				Slug:     &slug,
			}, nil
		},
	}

	resolver := NewPathResolver(mockEpicRepo, nil, nil, projectRoot)
	path, err := resolver.ResolveEpicPath(ctx, "E01")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should use explicit filepath (highest precedence)
	expected := filepath.Join(projectRoot, explicitPath)
	if path != expected {
		t.Errorf("expected explicit path %s, got %s", expected, path)
	}
}
