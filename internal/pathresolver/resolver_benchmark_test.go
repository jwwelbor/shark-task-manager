package pathresolver

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// Benchmark PathResolver.ResolveEpicPath with default path
func BenchmarkPathResolver_ResolveEpicPath_Default(b *testing.B) {
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := resolver.ResolveEpicPath(ctx, "E01")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark PathResolver.ResolveEpicPath with custom folder path
func BenchmarkPathResolver_ResolveEpicPath_CustomFolder(b *testing.B) {
	ctx := context.Background()
	projectRoot := "/project"

	customPath := "docs/custom/epic-folder"
	mockEpicRepo := &MockEpicRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				ID:               1,
				Key:              "E01",
				Title:            "Test Epic",
				CustomFolderPath: &customPath,
			}, nil
		},
	}

	resolver := NewPathResolver(mockEpicRepo, nil, nil, projectRoot)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := resolver.ResolveEpicPath(ctx, "E01")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark PathResolver.ResolveEpicPath with explicit filename
func BenchmarkPathResolver_ResolveEpicPath_ExplicitFilename(b *testing.B) {
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := resolver.ResolveEpicPath(ctx, "E01")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark PathResolver.ResolveFeaturePath with default path
func BenchmarkPathResolver_ResolveFeaturePath_Default(b *testing.B) {
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := resolver.ResolveFeaturePath(ctx, "E01-F01")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark PathResolver.ResolveFeaturePath with inherited epic path
func BenchmarkPathResolver_ResolveFeaturePath_InheritedPath(b *testing.B) {
	ctx := context.Background()
	projectRoot := "/project"

	epicCustomPath := "docs/2025-q1"
	featureSlug := "user-auth"

	mockEpicRepo := &MockEpicRepository{
		GetByIDFunc: func(ctx context.Context, id int64) (*models.Epic, error) {
			return &models.Epic{
				ID:               1,
				Key:              "E01",
				Title:            "Q1 Epic",
				CustomFolderPath: &epicCustomPath,
			}, nil
		},
	}

	mockFeatureRepo := &MockFeatureRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{
				ID:     1,
				EpicID: 1,
				Key:    "E01-F01",
				Title:  "User Auth",
				Slug:   &featureSlug,
			}, nil
		},
	}

	resolver := NewPathResolver(mockEpicRepo, mockFeatureRepo, nil, projectRoot)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := resolver.ResolveFeaturePath(ctx, "E01-F01")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark PathResolver.ResolveTaskPath with default path
func BenchmarkPathResolver_ResolveTaskPath_Default(b *testing.B) {
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := resolver.ResolveTaskPath(ctx, "T-E01-F01-001")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark PathResolver.ResolveTaskPath with explicit filepath (early return optimization)
func BenchmarkPathResolver_ResolveTaskPath_ExplicitPath(b *testing.B) {
	ctx := context.Background()
	projectRoot := "/project"

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

	resolver := NewPathResolver(nil, nil, mockTaskRepo, projectRoot)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := resolver.ResolveTaskPath(ctx, "T-E01-F01-001")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark complex scenario: Multiple path resolutions in sequence
func BenchmarkPathResolver_ComplexScenario(b *testing.B) {
	ctx := context.Background()
	projectRoot := "/project"

	epicSlug := "test-epic"
	featureSlug := "test-feature"

	mockEpicRepo := &MockEpicRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				ID:    1,
				Key:   "E01",
				Title: "Test Epic",
				Slug:  &epicSlug,
			}, nil
		},
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate typical workflow: resolve epic, feature, and task paths
		_, err := resolver.ResolveEpicPath(ctx, "E01")
		if err != nil {
			b.Fatal(err)
		}

		_, err = resolver.ResolveFeaturePath(ctx, "E01-F01")
		if err != nil {
			b.Fatal(err)
		}

		_, err = resolver.ResolveTaskPath(ctx, "T-E01-F01-001")
		if err != nil {
			b.Fatal(err)
		}
	}
}
