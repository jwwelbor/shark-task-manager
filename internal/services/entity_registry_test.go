package services

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEntityRepository is a minimal EntityRepository implementation for registry tests.
type mockEntityRepository struct {
	entityType models.EntityType
}

func (m *mockEntityRepository) GetByKey(_ context.Context, _ string) (models.Entity, error) {
	return nil, nil
}
func (m *mockEntityRepository) GetByID(_ context.Context, _ int64) (models.Entity, error) {
	return nil, nil
}
func (m *mockEntityRepository) UpdateStatus(_ context.Context, _ int64, _ string) error {
	return nil
}
func (m *mockEntityRepository) UpdateStatusIfCurrent(_ context.Context, _ int64, _ string, _ string) (bool, error) {
	return true, nil
}
func (m *mockEntityRepository) Update(_ context.Context, _ models.Entity) error {
	return nil
}
func (m *mockEntityRepository) GetContextData(_ context.Context, _ int64) (*string, error) {
	return nil, nil
}
func (m *mockEntityRepository) UpdateContextData(_ context.Context, _ int64, _ *string) error {
	return nil
}

func TestEntityRegistry_RegisterAndGet(t *testing.T) {
	reg := NewEntityRegistry()

	epicRepo := &mockEntityRepository{entityType: models.EntityTypeEpic}
	taskRepo := &mockEntityRepository{entityType: models.EntityTypeTask}

	reg.Register(models.EntityTypeEpic, epicRepo)
	reg.Register(models.EntityTypeTask, taskRepo)

	// GetRepository returns the correct adapter for each type.
	got, err := reg.GetRepository(models.EntityTypeEpic)
	require.NoError(t, err)
	assert.Same(t, epicRepo, got)

	got, err = reg.GetRepository(models.EntityTypeTask)
	require.NoError(t, err)
	assert.Same(t, taskRepo, got)
}

func TestEntityRegistry_UnregisteredType(t *testing.T) {
	reg := NewEntityRegistry()

	repo, err := reg.GetRepository(models.EntityTypeFeature)
	assert.Nil(t, repo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no repository registered")
	assert.Contains(t, err.Error(), string(models.EntityTypeFeature))
}

func TestEntityRegistry_MustGetRepository_Panics(t *testing.T) {
	reg := NewEntityRegistry()

	assert.Panics(t, func() {
		reg.MustGetRepository(models.EntityTypeBug)
	})
}

func TestEntityRegistry_MustGetRepository_Success(t *testing.T) {
	reg := NewEntityRegistry()
	epicRepo := &mockEntityRepository{entityType: models.EntityTypeEpic}
	reg.Register(models.EntityTypeEpic, epicRepo)

	// Should not panic.
	got := reg.MustGetRepository(models.EntityTypeEpic)
	assert.Same(t, epicRepo, got)
}

func TestEntityRegistry_DuplicateRegistration_Panics(t *testing.T) {
	reg := NewEntityRegistry()
	repo1 := &mockEntityRepository{entityType: models.EntityTypeEpic}
	repo2 := &mockEntityRepository{entityType: models.EntityTypeEpic}

	reg.Register(models.EntityTypeEpic, repo1)

	assert.Panics(t, func() {
		reg.Register(models.EntityTypeEpic, repo2)
	})
}

func TestEntityRegistry_RegisteredTypes(t *testing.T) {
	t.Run("empty registry", func(t *testing.T) {
		reg := NewEntityRegistry()
		types := reg.RegisteredTypes()
		assert.Empty(t, types)
	})

	t.Run("all five types", func(t *testing.T) {
		reg := NewEntityRegistry()
		allTypes := []models.EntityType{
			models.EntityTypeEpic,
			models.EntityTypeFeature,
			models.EntityTypeTask,
			models.EntityTypeBug,
			models.EntityTypeChange,
		}
		for _, et := range allTypes {
			reg.Register(et, &mockEntityRepository{entityType: et})
		}

		types := reg.RegisteredTypes()
		assert.Len(t, types, 5)

		// Verify sorted alphabetically.
		for i := 1; i < len(types); i++ {
			assert.True(t, string(types[i-1]) < string(types[i]),
				"expected %s < %s", types[i-1], types[i])
		}

		// Verify all expected types are present.
		typeSet := make(map[models.EntityType]bool)
		for _, et := range types {
			typeSet[et] = true
		}
		for _, et := range allTypes {
			assert.True(t, typeSet[et], "missing entity type: %s", et)
		}
	})
}

func TestEntityRegistry_ConcurrentAccess(t *testing.T) {
	t.Run("concurrent reads", func(t *testing.T) {
		reg := NewEntityRegistry()

		// Pre-register all types.
		allTypes := []models.EntityType{
			models.EntityTypeEpic,
			models.EntityTypeFeature,
			models.EntityTypeTask,
			models.EntityTypeBug,
			models.EntityTypeChange,
		}
		for _, et := range allTypes {
			reg.Register(et, &mockEntityRepository{entityType: et})
		}

		// Concurrent reads should not race.
		var wg sync.WaitGroup
		const goroutines = 50

		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				et := allTypes[idx%len(allTypes)]

				repo, err := reg.GetRepository(et)
				assert.NoError(t, err)
				assert.NotNil(t, repo)

				types := reg.RegisteredTypes()
				assert.Len(t, types, 5)
			}(i)
		}

		wg.Wait()
	})

	t.Run("concurrent register and get", func(t *testing.T) {
		// Use unique entity types to avoid duplicate registration panics.
		// Register in goroutines while other goroutines read.
		const numTypes = 20
		var wg sync.WaitGroup
		reg := NewEntityRegistry()

		// Register goroutines — each registers a unique type.
		for i := 0; i < numTypes; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				et := models.EntityType(fmt.Sprintf("test_type_%d", idx))
				reg.Register(et, &mockEntityRepository{entityType: et})
			}(i)
		}

		// Concurrent reader goroutines that call GetRepository and RegisteredTypes.
		for i := 0; i < 30; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				et := models.EntityType(fmt.Sprintf("test_type_%d", idx%numTypes))
				// May or may not find the type depending on registration order.
				_, _ = reg.GetRepository(et)
				_ = reg.RegisteredTypes()
			}(i)
		}

		wg.Wait()

		// After all goroutines complete, all types should be registered.
		types := reg.RegisteredTypes()
		assert.Len(t, types, numTypes)
	})
}
