package pathresolver

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
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
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1,
				Key:   "E01",
				Title: "Test Epic",
				Slug:  &slug},
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
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1,
				Key:      "E01",
				Title:    "Test Epic",
				FilePath: &filename},
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
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1,
				Key:   "E01",
				Title: "Test Epic",
				Slug:  &epicSlug},
			}, nil
		},
	}

	mockFeatureRepo := &MockFeatureRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{BaseEntity: models.BaseEntity{ID: 1,

				Key:   "E01-F01",
				Title: "Test Feature",
				Slug:  &featureSlug}, EpicID: 1,
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
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1,
				Key:   "E01",
				Title: "Test Epic"},
			}, nil
		},
	}

	mockFeatureRepo := &MockFeatureRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{BaseEntity: models.BaseEntity{ID: 1,

				Key:      "E01-F01",
				Title:    "Test Feature",
				FilePath: &filename}, EpicID: 1,
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
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1,
				Key:   "E01",
				Title: "Test Epic",
				Slug:  &epicSlug},
			}, nil
		},
	}

	mockFeatureRepo := &MockFeatureRepository{
		GetByIDFunc: func(ctx context.Context, id int64) (*models.Feature, error) {
			return &models.Feature{BaseEntity: models.BaseEntity{ID: 1,

				Key:   "E01-F01",
				Title: "Test Feature",
				Slug:  &featureSlug}, EpicID: 1,
			}, nil
		},
	}

	mockTaskRepo := &MockTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			emptyPath := ""
			return &models.Task{BaseEntity: models.BaseEntity{ID: 1,

				Key:      "T-E01-F01-001",
				Title:    "Test Task",
				FilePath: &emptyPath}, FeatureID: 1,
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
			return &models.Task{BaseEntity: models.BaseEntity{ID: 1,

				Key:      "T-E01-F01-001",
				Title:    "Test Task",
				FilePath: &taskPath}, FeatureID: 1,
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
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1,
				Key:      "E01",
				Title:    "Test Epic",
				FilePath: &explicitPath,
				Slug:     &slug},
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

// --- ResolveFeatureBaseDir tests ---

func TestResolveFeatureBaseDir_EpicWithFilePath(t *testing.T) {
	ctx := context.Background()

	epicPath := "docs/plan/E01-my-epic/epic.md"
	mockEpicRepo := &MockEpicRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1,
				Key:      "E01",
				Title:    "My Epic",
				FilePath: &epicPath},
			}, nil
		},
	}

	resolver := NewPathResolver(mockEpicRepo, nil, nil, "/project")
	dir, err := resolver.ResolveFeatureBaseDir(ctx, "E01")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "docs/plan/E01-my-epic"
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}
}

func TestResolveFeatureBaseDir_EpicWithCustomPath(t *testing.T) {
	ctx := context.Background()

	epicPath := "backend/docs/roadmap/epic.md"
	mockEpicRepo := &MockEpicRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1,
				Key:      "E01",
				Title:    "My Epic",
				FilePath: &epicPath},
			}, nil
		},
	}

	resolver := NewPathResolver(mockEpicRepo, nil, nil, "/project")
	dir, err := resolver.ResolveFeatureBaseDir(ctx, "E01")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "backend/docs/roadmap"
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}
}

func TestResolveFeatureBaseDir_EpicWithNoFilePath_ReturnsError(t *testing.T) {
	ctx := context.Background()

	mockEpicRepo := &MockEpicRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1,
				Key:   "E07",
				Title: "Epic Without Path"},
			}, nil
		},
	}

	resolver := NewPathResolver(mockEpicRepo, nil, nil, "/project")
	_, err := resolver.ResolveFeatureBaseDir(ctx, "E07")

	if err == nil {
		t.Fatal("expected error when epic has no file_path, got nil")
	}

	// Should mention the epic key
	if !strings.Contains(err.Error(), "E07") {
		t.Errorf("error should mention epic key E07, got: %v", err)
	}
	// Should suggest the fix command
	if !strings.Contains(err.Error(), "shark epic update") {
		t.Errorf("error should suggest 'shark epic update' command, got: %v", err)
	}
	// Should mention --file flag
	if !strings.Contains(err.Error(), "--file") {
		t.Errorf("error should mention --file flag, got: %v", err)
	}
}

func TestResolveFeatureBaseDir_EpicWithEmptyFilePath_ReturnsError(t *testing.T) {
	ctx := context.Background()

	emptyPath := ""
	mockEpicRepo := &MockEpicRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{BaseEntity: models.BaseEntity{ID: 1,
				Key:      "E07",
				Title:    "Epic With Empty Path",
				FilePath: &emptyPath},
			}, nil
		},
	}

	resolver := NewPathResolver(mockEpicRepo, nil, nil, "/project")
	_, err := resolver.ResolveFeatureBaseDir(ctx, "E07")

	if err == nil {
		t.Fatal("expected error when epic has empty file_path, got nil")
	}
	if !strings.Contains(err.Error(), "shark epic update") {
		t.Errorf("error should suggest fix command, got: %v", err)
	}
}

func TestResolveFeatureBaseDir_EpicNotFound(t *testing.T) {
	ctx := context.Background()

	mockEpicRepo := &MockEpicRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, errors.New("epic not found")
		},
	}

	resolver := NewPathResolver(mockEpicRepo, nil, nil, "/project")
	_, err := resolver.ResolveFeatureBaseDir(ctx, "E99")

	if err == nil {
		t.Fatal("expected error for non-existent epic, got nil")
	}
}

// --- ResolveTaskBaseDir tests ---

func TestResolveTaskBaseDir_FeatureWithFilePath(t *testing.T) {
	ctx := context.Background()

	featurePath := "docs/plan/E01-my-epic/E01-F01-auth/feature.md"
	mockFeatureRepo := &MockFeatureRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{BaseEntity: models.BaseEntity{ID: 1,

				Key:      "E01-F01",
				Title:    "Auth Feature",
				FilePath: &featurePath}, EpicID: 1,
			}, nil
		},
	}

	resolver := NewPathResolver(nil, mockFeatureRepo, nil, "/project")
	dir, err := resolver.ResolveTaskBaseDir(ctx, "E01-F01")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "docs/plan/E01-my-epic/E01-F01-auth"
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}
}

func TestResolveTaskBaseDir_FeatureAtCustomPath(t *testing.T) {
	ctx := context.Background()

	featurePath := "backend/docs/features/auth-feature.md"
	mockFeatureRepo := &MockFeatureRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{BaseEntity: models.BaseEntity{ID: 1,

				Key:      "E01-F01",
				Title:    "Auth Feature",
				FilePath: &featurePath}, EpicID: 1,
			}, nil
		},
	}

	resolver := NewPathResolver(nil, mockFeatureRepo, nil, "/project")
	dir, err := resolver.ResolveTaskBaseDir(ctx, "E01-F01")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Task should go under the feature file's directory, NOT docs/plan/
	expected := "backend/docs/features"
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}
}

func TestResolveTaskBaseDir_StandaloneFeatureFile(t *testing.T) {
	ctx := context.Background()

	// Feature file stored directly in epic folder (no feature subfolder)
	featurePath := "docs/plan/E01-my-epic/F11-technical-architecture.md"
	mockFeatureRepo := &MockFeatureRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{BaseEntity: models.BaseEntity{ID: 1,

				Key:      "E01-F11",
				Title:    "Technical Architecture",
				FilePath: &featurePath}, EpicID: 1,
			}, nil
		},
	}

	resolver := NewPathResolver(nil, mockFeatureRepo, nil, "/project")
	dir, err := resolver.ResolveTaskBaseDir(ctx, "E01-F11")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Tasks should go under the feature file's directory
	expected := "docs/plan/E01-my-epic"
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}
}

func TestResolveTaskBaseDir_FeatureWithNoFilePath_ReturnsError(t *testing.T) {
	ctx := context.Background()

	mockFeatureRepo := &MockFeatureRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{BaseEntity: models.BaseEntity{ID: 1,

				Key:   "E07-F03",
				Title: "Feature Without Path"}, EpicID: 1,
			}, nil
		},
	}

	resolver := NewPathResolver(nil, mockFeatureRepo, nil, "/project")
	_, err := resolver.ResolveTaskBaseDir(ctx, "E07-F03")

	if err == nil {
		t.Fatal("expected error when feature has no file_path, got nil")
	}

	// Should mention the feature key
	if !strings.Contains(err.Error(), "E07-F03") {
		t.Errorf("error should mention feature key E07-F03, got: %v", err)
	}
	// Should suggest the fix command
	if !strings.Contains(err.Error(), "shark feature update") {
		t.Errorf("error should suggest 'shark feature update' command, got: %v", err)
	}
	// Should mention --file flag
	if !strings.Contains(err.Error(), "--file") {
		t.Errorf("error should mention --file flag, got: %v", err)
	}
}

func TestResolveTaskBaseDir_FeatureWithEmptyFilePath_ReturnsError(t *testing.T) {
	ctx := context.Background()

	emptyPath := ""
	mockFeatureRepo := &MockFeatureRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{BaseEntity: models.BaseEntity{ID: 1,

				Key:      "E07-F03",
				Title:    "Feature With Empty Path",
				FilePath: &emptyPath}, EpicID: 1,
			}, nil
		},
	}

	resolver := NewPathResolver(nil, mockFeatureRepo, nil, "/project")
	_, err := resolver.ResolveTaskBaseDir(ctx, "E07-F03")

	if err == nil {
		t.Fatal("expected error when feature has empty file_path, got nil")
	}
	if !strings.Contains(err.Error(), "shark feature update") {
		t.Errorf("error should suggest fix command, got: %v", err)
	}
}

func TestResolveTaskBaseDir_FeatureNotFound(t *testing.T) {
	ctx := context.Background()

	mockFeatureRepo := &MockFeatureRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
			return nil, errors.New("feature not found")
		},
	}

	resolver := NewPathResolver(nil, mockFeatureRepo, nil, "/project")
	_, err := resolver.ResolveTaskBaseDir(ctx, "E99-F99")

	if err == nil {
		t.Fatal("expected error for non-existent feature, got nil")
	}
}
