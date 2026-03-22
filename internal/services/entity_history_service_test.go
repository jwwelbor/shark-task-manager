package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockEntityHistoryQuerier implements EntityHistoryQuerier for testing.
type MockEntityHistoryQuerier struct {
	ListByEntityFunc func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityHistory, error)
}

func (m *MockEntityHistoryQuerier) ListByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityHistory, error) {
	if m.ListByEntityFunc != nil {
		return m.ListByEntityFunc(ctx, entityType, entityID)
	}
	return nil, fmt.Errorf("ListByEntity not implemented in mock")
}

// mockEntityForHistory is a simple models.Entity implementation for testing.
type mockEntityForHistory struct {
	id int64
}

func (m *mockEntityForHistory) GetID() int64                     { return m.id }
func (m *mockEntityForHistory) GetKey() string                   { return "" }
func (m *mockEntityForHistory) GetTitle() string                 { return "" }
func (m *mockEntityForHistory) GetSlug() string                  { return "" }
func (m *mockEntityForHistory) GetStatus() string                { return "" }
func (m *mockEntityForHistory) SetStatus(string)                 {}
func (m *mockEntityForHistory) GetEntityType() models.EntityType { return "" }
func (m *mockEntityForHistory) GetDescription() string           { return "" }
func (m *mockEntityForHistory) GetFilePath() string              { return "" }
func (m *mockEntityForHistory) GetContextData() *string          { return nil }
func (m *mockEntityForHistory) SetContextData(*string)           {}
func (m *mockEntityForHistory) GetCreatedAt() time.Time          { return time.Time{} }
func (m *mockEntityForHistory) GetUpdatedAt() time.Time          { return time.Time{} }
func (m *mockEntityForHistory) Validate() error                  { return nil }

// mockEntityRepoForHistory implements EntityRepository for testing.
type mockEntityRepoForHistory struct {
	getByKeyFunc func(ctx context.Context, key string) (models.Entity, error)
}

func (m *mockEntityRepoForHistory) GetByKey(ctx context.Context, key string) (models.Entity, error) {
	if m.getByKeyFunc != nil {
		return m.getByKeyFunc(ctx, key)
	}
	return nil, fmt.Errorf("GetByKey not implemented")
}

func (m *mockEntityRepoForHistory) GetByID(ctx context.Context, id int64) (models.Entity, error) {
	return nil, fmt.Errorf("GetByID not implemented")
}

func (m *mockEntityRepoForHistory) UpdateStatus(ctx context.Context, id int64, status string) error {
	return fmt.Errorf("UpdateStatus not implemented")
}

func (m *mockEntityRepoForHistory) Update(ctx context.Context, entity models.Entity) error {
	return fmt.Errorf("Update not implemented")
}

func (m *mockEntityRepoForHistory) GetContextData(ctx context.Context, id int64) (*string, error) {
	return nil, fmt.Errorf("GetContextData not implemented")
}

func (m *mockEntityRepoForHistory) UpdateContextData(ctx context.Context, id int64, data *string) error {
	return fmt.Errorf("UpdateContextData not implemented")
}

// Helper to build a registry with a single entity type for tests.
func buildRegistryWithType(entityType models.EntityType, repo EntityRepository) *EntityRegistry {
	reg := NewEntityRegistry()
	reg.Register(entityType, repo)
	return reg
}

func TestEntityHistoryService_GetHistory(t *testing.T) {
	// Arrange
	now := time.Now()
	expectedHistory := []*models.EntityHistory{
		{ID: 2, EntityType: models.EntityTypeFeature, EntityID: 42, FromStatus: strPtr("todo"), ToStatus: "in_progress", ChangedAt: now},
		{ID: 1, EntityType: models.EntityTypeFeature, EntityID: 42, FromStatus: nil, ToStatus: "todo", ChangedAt: now.Add(-time.Hour)},
	}

	mockRepo := &MockEntityHistoryQuerier{
		ListByEntityFunc: func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityHistory, error) {
			assert.Equal(t, models.EntityTypeFeature, entityType)
			assert.Equal(t, int64(42), entityID)
			return expectedHistory, nil
		},
	}

	entityRepo := &mockEntityRepoForHistory{
		getByKeyFunc: func(ctx context.Context, key string) (models.Entity, error) {
			assert.Equal(t, "E21-F07", key)
			return &mockEntityForHistory{id: 42}, nil
		},
	}

	registry := buildRegistryWithType(models.EntityTypeFeature, entityRepo)
	svc := NewEntityHistoryService(mockRepo, registry)

	// Act
	history, err := svc.GetHistory(context.Background(), models.EntityTypeFeature, "E21-F07")

	// Assert
	require.NoError(t, err)
	assert.Len(t, history, 2)
	assert.Equal(t, expectedHistory, history)
}

func TestEntityHistoryService_GetHistory_EntityNotFound(t *testing.T) {
	// Arrange
	mockRepo := &MockEntityHistoryQuerier{}
	entityRepo := &mockEntityRepoForHistory{
		getByKeyFunc: func(ctx context.Context, key string) (models.Entity, error) {
			return nil, fmt.Errorf("task not found: E99-F99-999")
		},
	}
	registry := buildRegistryWithType(models.EntityTypeTask, entityRepo)
	svc := NewEntityHistoryService(mockRepo, registry)

	// Act
	history, err := svc.GetHistory(context.Background(), models.EntityTypeTask, "E99-F99-999")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, history)
	assert.Contains(t, err.Error(), "E99-F99-999")
	assert.Contains(t, err.Error(), "failed to get history")
}

func TestEntityHistoryService_GetHistory_UnregisteredType(t *testing.T) {
	// Arrange: empty registry
	mockRepo := &MockEntityHistoryQuerier{}
	registry := NewEntityRegistry()
	svc := NewEntityHistoryService(mockRepo, registry)

	// Act
	history, err := svc.GetHistory(context.Background(), models.EntityType("nonexistent"), "X01")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, history)
	assert.Contains(t, err.Error(), "no repository registered for entity type")
}

func TestEntityHistoryService_GetHistory_EmptyResult(t *testing.T) {
	// Arrange
	mockRepo := &MockEntityHistoryQuerier{
		ListByEntityFunc: func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityHistory, error) {
			return make([]*models.EntityHistory, 0), nil
		},
	}
	entityRepo := &mockEntityRepoForHistory{
		getByKeyFunc: func(ctx context.Context, key string) (models.Entity, error) {
			return &mockEntityForHistory{id: 10}, nil
		},
	}
	registry := buildRegistryWithType(models.EntityTypeFeature, entityRepo)
	svc := NewEntityHistoryService(mockRepo, registry)

	// Act
	history, err := svc.GetHistory(context.Background(), models.EntityTypeFeature, "E21-F07")

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, history)
	assert.Empty(t, history)
}

func TestEntityHistoryService_GetHistory_AllEntityTypes(t *testing.T) {
	entityTypes := []struct {
		entityType models.EntityType
		key        string
	}{
		{models.EntityTypeEpic, "E21"},
		{models.EntityTypeFeature, "E21-F07"},
		{models.EntityTypeTask, "E21-F08-001"},
		{models.EntityTypeBug, "B001"},
		{models.EntityTypeChange, "CC-001"},
	}

	for _, tc := range entityTypes {
		t.Run(string(tc.entityType), func(t *testing.T) {
			var calledEntityType models.EntityType
			var calledEntityID int64

			mockRepo := &MockEntityHistoryQuerier{
				ListByEntityFunc: func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityHistory, error) {
					calledEntityType = entityType
					calledEntityID = entityID
					return []*models.EntityHistory{
						{ID: 1, EntityType: entityType, EntityID: entityID, ToStatus: "in_progress"},
					}, nil
				},
			}

			entityRepo := &mockEntityRepoForHistory{
				getByKeyFunc: func(ctx context.Context, key string) (models.Entity, error) {
					assert.Equal(t, tc.key, key)
					return &mockEntityForHistory{id: 100}, nil
				},
			}

			registry := buildRegistryWithType(tc.entityType, entityRepo)
			svc := NewEntityHistoryService(mockRepo, registry)

			history, err := svc.GetHistory(context.Background(), tc.entityType, tc.key)

			require.NoError(t, err)
			assert.Len(t, history, 1)
			assert.Equal(t, tc.entityType, calledEntityType)
			assert.Equal(t, int64(100), calledEntityID)
		})
	}
}

func TestEntityHistoryService_GetHistory_RepoError(t *testing.T) {
	// Arrange: entity exists but history repo returns error
	mockRepo := &MockEntityHistoryQuerier{
		ListByEntityFunc: func(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityHistory, error) {
			return nil, fmt.Errorf("database connection failed")
		},
	}
	entityRepo := &mockEntityRepoForHistory{
		getByKeyFunc: func(ctx context.Context, key string) (models.Entity, error) {
			return &mockEntityForHistory{id: 42}, nil
		},
	}
	registry := buildRegistryWithType(models.EntityTypeFeature, entityRepo)
	svc := NewEntityHistoryService(mockRepo, registry)

	// Act
	history, err := svc.GetHistory(context.Background(), models.EntityTypeFeature, "E21-F07")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, history)
	assert.Contains(t, err.Error(), "database connection failed")
}

// strPtr returns a pointer to the given string.
func strPtr(s string) *string {
	return &s
}
