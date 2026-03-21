package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Mock Repositories for CriteriaService
// ============================================================================

// MockCriteriaRepository implements CriteriaRepository interface for testing.
type MockCriteriaRepository struct {
	CreateFunc             func(ctx context.Context, criteria *models.TaskCriteria) error
	GetByIDFunc            func(ctx context.Context, id int64) (*models.TaskCriteria, error)
	GetByTaskIDFunc        func(ctx context.Context, taskID int64) ([]*models.TaskCriteria, error)
	UpdateFunc             func(ctx context.Context, criteria *models.TaskCriteria) error
	UpdateStatusFunc       func(ctx context.Context, id int64, status models.CriteriaStatus, notes *string) error
	DeleteFunc             func(ctx context.Context, id int64) error
	DeleteByTaskIDFunc     func(ctx context.Context, taskID int64) error
	GetSummaryByTaskIDFunc func(ctx context.Context, taskID int64) (*repository.CriteriaSummary, error)
}

func (m *MockCriteriaRepository) Create(ctx context.Context, criteria *models.TaskCriteria) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, criteria)
	}
	return fmt.Errorf("Create not implemented in mock")
}

func (m *MockCriteriaRepository) GetByID(ctx context.Context, id int64) (*models.TaskCriteria, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, fmt.Errorf("GetByID not implemented in mock")
}

func (m *MockCriteriaRepository) GetByTaskID(ctx context.Context, taskID int64) ([]*models.TaskCriteria, error) {
	if m.GetByTaskIDFunc != nil {
		return m.GetByTaskIDFunc(ctx, taskID)
	}
	return nil, fmt.Errorf("GetByTaskID not implemented in mock")
}

func (m *MockCriteriaRepository) Update(ctx context.Context, criteria *models.TaskCriteria) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, criteria)
	}
	return fmt.Errorf("Update not implemented in mock")
}

func (m *MockCriteriaRepository) UpdateStatus(ctx context.Context, id int64, status models.CriteriaStatus, notes *string) error {
	if m.UpdateStatusFunc != nil {
		return m.UpdateStatusFunc(ctx, id, status, notes)
	}
	return fmt.Errorf("UpdateStatus not implemented in mock")
}

func (m *MockCriteriaRepository) Delete(ctx context.Context, id int64) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return fmt.Errorf("Delete not implemented in mock")
}

func (m *MockCriteriaRepository) DeleteByTaskID(ctx context.Context, taskID int64) error {
	if m.DeleteByTaskIDFunc != nil {
		return m.DeleteByTaskIDFunc(ctx, taskID)
	}
	return fmt.Errorf("DeleteByTaskID not implemented in mock")
}

func (m *MockCriteriaRepository) GetSummaryByTaskID(ctx context.Context, taskID int64) (*repository.CriteriaSummary, error) {
	if m.GetSummaryByTaskIDFunc != nil {
		return m.GetSummaryByTaskIDFunc(ctx, taskID)
	}
	return nil, fmt.Errorf("GetSummaryByTaskID not implemented in mock")
}

// MockCriteriaTaskRepository implements CriteriaTaskRepository interface for testing.
type MockCriteriaTaskRepository struct {
	GetByKeyFunc      func(ctx context.Context, key string) (*models.Task, error)
	ListByFeatureFunc func(ctx context.Context, featureID int64) ([]*models.Task, error)
}

func (m *MockCriteriaTaskRepository) GetByKey(ctx context.Context, key string) (*models.Task, error) {
	if m.GetByKeyFunc != nil {
		return m.GetByKeyFunc(ctx, key)
	}
	return nil, fmt.Errorf("GetByKey not implemented in mock")
}

func (m *MockCriteriaTaskRepository) ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error) {
	if m.ListByFeatureFunc != nil {
		return m.ListByFeatureFunc(ctx, featureID)
	}
	return nil, fmt.Errorf("ListByFeature not implemented in mock")
}

// MockCriteriaFeatureRepository implements CriteriaFeatureRepository interface for testing.
type MockCriteriaFeatureRepository struct {
	GetByKeyFunc func(ctx context.Context, key string) (*models.Feature, error)
}

func (m *MockCriteriaFeatureRepository) GetByKey(ctx context.Context, key string) (*models.Feature, error) {
	if m.GetByKeyFunc != nil {
		return m.GetByKeyFunc(ctx, key)
	}
	return nil, fmt.Errorf("GetByKey not implemented in mock")
}

// ============================================================================
// CriteriaService.ListCriteria (GetTaskCriteria) Tests
// ============================================================================

func TestCriteriaService_ListCriteria_Happy_Path(t *testing.T) {
	// Arrange
	taskKey := "E07-F01-001"
	taskID := int64(42)

	expectedCriteria := []*models.TaskCriteria{
		{
			ID:        1,
			TaskID:    taskID,
			Criterion: "User can log in with valid credentials",
			Status:    models.CriteriaStatusPending,
		},
		{
			ID:        2,
			TaskID:    taskID,
			Criterion: "Error message shown for invalid credentials",
			Status:    models.CriteriaStatusComplete,
		},
	}

	mockCriteriaRepo := &MockCriteriaRepository{
		GetByTaskIDFunc: func(ctx context.Context, id int64) ([]*models.TaskCriteria, error) {
			assert.Equal(t, taskID, id)
			return expectedCriteria, nil
		},
	}

	mockTaskRepo := &MockCriteriaTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			assert.Equal(t, taskKey, key)
			return &models.Task{BaseEntity: models.BaseEntity{ID: taskID,
				Key: taskKey},
			}, nil
		},
	}

	svc := NewCriteriaService(mockCriteriaRepo, mockTaskRepo, nil)

	// Act
	criteria, err := svc.ListCriteria(context.Background(), taskKey)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, criteria)
	assert.Len(t, criteria, 2)
	assert.Equal(t, "User can log in with valid credentials", criteria[0].Criterion)
	assert.Equal(t, models.CriteriaStatusPending, criteria[0].Status)
	assert.Equal(t, "Error message shown for invalid credentials", criteria[1].Criterion)
	assert.Equal(t, models.CriteriaStatusComplete, criteria[1].Status)
}

func TestCriteriaService_ListCriteria_Task_Not_Found(t *testing.T) {
	// Arrange
	taskKey := "E07-F01-999"

	mockCriteriaRepo := &MockCriteriaRepository{}

	mockTaskRepo := &MockCriteriaTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return nil, fmt.Errorf("task not found: %s", key)
		},
	}

	svc := NewCriteriaService(mockCriteriaRepo, mockTaskRepo, nil)

	// Act
	criteria, err := svc.ListCriteria(context.Background(), taskKey)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, criteria)
	assert.Contains(t, err.Error(), "E07-F01-999")
}

func TestCriteriaService_ListCriteria_Empty_Results(t *testing.T) {
	// Arrange: Task exists but has no criteria
	taskKey := "E07-F01-001"
	taskID := int64(42)

	mockCriteriaRepo := &MockCriteriaRepository{
		GetByTaskIDFunc: func(ctx context.Context, id int64) ([]*models.TaskCriteria, error) {
			return []*models.TaskCriteria{}, nil
		},
	}

	mockTaskRepo := &MockCriteriaTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{BaseEntity: models.BaseEntity{ID: taskID, Key: taskKey}}, nil
		},
	}

	svc := NewCriteriaService(mockCriteriaRepo, mockTaskRepo, nil)

	// Act
	criteria, err := svc.ListCriteria(context.Background(), taskKey)

	// Assert: Returns empty slice without error
	assert.NoError(t, err)
	assert.NotNil(t, criteria)
	assert.Len(t, criteria, 0)
}

func TestCriteriaService_ListCriteria_Repository_Error(t *testing.T) {
	// Arrange: Criteria repository fails
	taskKey := "E07-F01-001"
	taskID := int64(42)

	mockCriteriaRepo := &MockCriteriaRepository{
		GetByTaskIDFunc: func(ctx context.Context, id int64) ([]*models.TaskCriteria, error) {
			return nil, fmt.Errorf("database connection failed")
		},
	}

	mockTaskRepo := &MockCriteriaTaskRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Task, error) {
			return &models.Task{BaseEntity: models.BaseEntity{ID: taskID, Key: taskKey}}, nil
		},
	}

	svc := NewCriteriaService(mockCriteriaRepo, mockTaskRepo, nil)

	// Act
	criteria, err := svc.ListCriteria(context.Background(), taskKey)

	// Assert: Error propagated with business context
	assert.Error(t, err)
	assert.Nil(t, criteria)
	assert.Contains(t, err.Error(), taskKey)
}

// ============================================================================
// CriteriaService.GetFeatureCriteria Tests
// ============================================================================

func TestCriteriaService_GetFeatureCriteria_Happy_Path(t *testing.T) {
	// Arrange
	featureKey := "E07-F01"
	featureID := int64(10)
	task1ID := int64(101)
	task2ID := int64(102)

	mockFeatureRepo := &MockCriteriaFeatureRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
			assert.Equal(t, featureKey, key)
			return &models.Feature{BaseEntity: models.BaseEntity{ID: featureID,
				Key: featureKey},
			}, nil
		},
	}

	mockTaskRepo := &MockCriteriaTaskRepository{
		ListByFeatureFunc: func(ctx context.Context, id int64) ([]*models.Task, error) {
			assert.Equal(t, featureID, id)
			return []*models.Task{
				{BaseEntity: models.BaseEntity{ID: task1ID, Key: "E07-F01-001"}},
				{BaseEntity: models.BaseEntity{ID: task2ID, Key: "E07-F01-002"}},
			}, nil
		},
	}

	mockCriteriaRepo := &MockCriteriaRepository{
		GetByTaskIDFunc: func(ctx context.Context, taskID int64) ([]*models.TaskCriteria, error) {
			if taskID == task1ID {
				return []*models.TaskCriteria{
					{ID: 1, TaskID: task1ID, Criterion: "Criterion A", Status: models.CriteriaStatusComplete},
					{ID: 2, TaskID: task1ID, Criterion: "Criterion B", Status: models.CriteriaStatusPending},
				}, nil
			}
			if taskID == task2ID {
				return []*models.TaskCriteria{
					{ID: 3, TaskID: task2ID, Criterion: "Criterion C", Status: models.CriteriaStatusComplete},
				}, nil
			}
			return []*models.TaskCriteria{}, nil
		},
		GetSummaryByTaskIDFunc: func(ctx context.Context, taskID int64) (*repository.CriteriaSummary, error) {
			if taskID == task1ID {
				return &repository.CriteriaSummary{
					TaskID:        task1ID,
					TotalCount:    2,
					PendingCount:  1,
					CompleteCount: 1,
					CompletionPct: 50.0,
				}, nil
			}
			if taskID == task2ID {
				return &repository.CriteriaSummary{
					TaskID:        task2ID,
					TotalCount:    1,
					CompleteCount: 1,
					CompletionPct: 100.0,
				}, nil
			}
			return &repository.CriteriaSummary{}, nil
		},
	}

	svc := NewCriteriaService(mockCriteriaRepo, mockTaskRepo, mockFeatureRepo)

	// Act
	result, err := svc.GetFeatureCriteria(context.Background(), featureKey)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, featureKey, result.FeatureKey)
	assert.Len(t, result.Tasks, 2)

	// Verify aggregate summary
	assert.Equal(t, 2, result.Summary.TotalTasks)
	assert.Equal(t, 3, result.Summary.TotalCriteria)
	assert.Equal(t, 2, result.Summary.CompleteCount)
	assert.Equal(t, 1, result.Summary.PendingCount)

	// Completion pct: 2 complete out of 3 total = 66.67%
	assert.InDelta(t, 66.67, result.Summary.CompletionPct, 0.01)
}

func TestCriteriaService_GetFeatureCriteria_Feature_Not_Found(t *testing.T) {
	// Arrange
	featureKey := "E07-F99"

	mockFeatureRepo := &MockCriteriaFeatureRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
			return nil, fmt.Errorf("feature not found: %s", key)
		},
	}

	mockTaskRepo := &MockCriteriaTaskRepository{}
	mockCriteriaRepo := &MockCriteriaRepository{}

	svc := NewCriteriaService(mockCriteriaRepo, mockTaskRepo, mockFeatureRepo)

	// Act
	result, err := svc.GetFeatureCriteria(context.Background(), featureKey)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "E07-F99")
}

func TestCriteriaService_GetFeatureCriteria_Nil_Feature_Repo(t *testing.T) {
	// Arrange: featureRepo is nil
	mockTaskRepo := &MockCriteriaTaskRepository{}
	mockCriteriaRepo := &MockCriteriaRepository{}

	svc := NewCriteriaService(mockCriteriaRepo, mockTaskRepo, nil)

	// Act
	result, err := svc.GetFeatureCriteria(context.Background(), "E07-F01")

	// Assert: Error because featureRepo is required for this operation
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "feature repository is required")
}

func TestCriteriaService_GetFeatureCriteria_No_Tasks(t *testing.T) {
	// Arrange: Feature exists but has no tasks
	featureKey := "E07-F01"
	featureID := int64(10)

	mockFeatureRepo := &MockCriteriaFeatureRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{BaseEntity: models.BaseEntity{ID: featureID, Key: featureKey}}, nil
		},
	}

	mockTaskRepo := &MockCriteriaTaskRepository{
		ListByFeatureFunc: func(ctx context.Context, id int64) ([]*models.Task, error) {
			return []*models.Task{}, nil
		},
	}

	mockCriteriaRepo := &MockCriteriaRepository{}

	svc := NewCriteriaService(mockCriteriaRepo, mockTaskRepo, mockFeatureRepo)

	// Act
	result, err := svc.GetFeatureCriteria(context.Background(), featureKey)

	// Assert: Returns result with empty tasks, no error
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, featureKey, result.FeatureKey)
	assert.Len(t, result.Tasks, 0)
	assert.Equal(t, 0, result.Summary.TotalTasks)
	assert.Equal(t, 0, result.Summary.TotalCriteria)
	assert.Equal(t, 0.0, result.Summary.CompletionPct)
}

func TestCriteriaService_GetFeatureCriteria_Completion_Pct_Calculation(t *testing.T) {
	// Arrange: Test completion pct calculation with NA criteria counted as complete
	featureKey := "E07-F01"
	featureID := int64(10)
	taskID := int64(101)

	mockFeatureRepo := &MockCriteriaFeatureRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{BaseEntity: models.BaseEntity{ID: featureID, Key: featureKey}}, nil
		},
	}

	mockTaskRepo := &MockCriteriaTaskRepository{
		ListByFeatureFunc: func(ctx context.Context, id int64) ([]*models.Task, error) {
			return []*models.Task{
				{BaseEntity: models.BaseEntity{ID: taskID, Key: "E07-F01-001"}},
			}, nil
		},
	}

	mockCriteriaRepo := &MockCriteriaRepository{
		GetByTaskIDFunc: func(ctx context.Context, id int64) ([]*models.TaskCriteria, error) {
			return []*models.TaskCriteria{
				{ID: 1, TaskID: taskID, Status: models.CriteriaStatusComplete},
				{ID: 2, TaskID: taskID, Status: models.CriteriaStatusNA},
				{ID: 3, TaskID: taskID, Status: models.CriteriaStatusPending},
				{ID: 4, TaskID: taskID, Status: models.CriteriaStatusFailed},
			}, nil
		},
		GetSummaryByTaskIDFunc: func(ctx context.Context, id int64) (*repository.CriteriaSummary, error) {
			return &repository.CriteriaSummary{
				TaskID:        taskID,
				TotalCount:    4,
				CompleteCount: 1,
				NACount:       1,
				PendingCount:  1,
				FailedCount:   1,
				CompletionPct: 50.0,
			}, nil
		},
	}

	svc := NewCriteriaService(mockCriteriaRepo, mockTaskRepo, mockFeatureRepo)

	// Act
	result, err := svc.GetFeatureCriteria(context.Background(), featureKey)

	// Assert: (complete + na) / total * 100 = (1+1)/4*100 = 50%
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.Summary.CompleteCount)
	assert.Equal(t, 1, result.Summary.NACount)
	assert.Equal(t, 4, result.Summary.TotalCriteria)
	assert.Equal(t, 50.0, result.Summary.CompletionPct)
}

// ============================================================================
// CriteriaService Constructor Tests
// ============================================================================

func TestNewCriteriaService_Panics_On_Nil_CriteriaRepo(t *testing.T) {
	mockTaskRepo := &MockCriteriaTaskRepository{}

	assert.Panics(t, func() {
		NewCriteriaService(nil, mockTaskRepo, nil)
	})
}

func TestNewCriteriaService_Panics_On_Nil_TaskRepo(t *testing.T) {
	mockCriteriaRepo := &MockCriteriaRepository{}

	assert.Panics(t, func() {
		NewCriteriaService(mockCriteriaRepo, nil, nil)
	})
}

func TestNewCriteriaService_Succeeds_With_Nil_FeatureRepo(t *testing.T) {
	// featureRepo is optional - constructor should succeed with nil
	mockCriteriaRepo := &MockCriteriaRepository{}
	mockTaskRepo := &MockCriteriaTaskRepository{}

	assert.NotPanics(t, func() {
		svc := NewCriteriaService(mockCriteriaRepo, mockTaskRepo, nil)
		assert.NotNil(t, svc)
	})
}
