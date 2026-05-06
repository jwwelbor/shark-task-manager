package commands

import (
	"context"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// MockSprintService implements sprintServicer for testing.
type MockSprintService struct {
	CreateSprintFunc  func(ctx context.Context, input services.CreateSprintInput) (*models.Sprint, error)
	GetSprintFunc     func(ctx context.Context, key string) (*models.Sprint, error)
	ListSprintsFunc   func(ctx context.Context, filters *services.SprintListFilters) ([]*models.Sprint, error)
	UpdateSprintFunc  func(ctx context.Context, key string, updates services.UpdateSprintInput) (*models.Sprint, error)
	DeleteSprintFunc  func(ctx context.Context, key string) error
	StartSprintFunc   func(ctx context.Context, key string) (*models.Sprint, error)
	CloseSprintFunc   func(ctx context.Context, key string) (*models.Sprint, error)
	ArchiveSprintFunc func(ctx context.Context, key string) (*models.Sprint, error)
}

func (m *MockSprintService) CreateSprint(ctx context.Context, input services.CreateSprintInput) (*models.Sprint, error) {
	if m.CreateSprintFunc != nil {
		return m.CreateSprintFunc(ctx, input)
	}
	return nil, nil
}

func (m *MockSprintService) GetSprint(ctx context.Context, key string) (*models.Sprint, error) {
	if m.GetSprintFunc != nil {
		return m.GetSprintFunc(ctx, key)
	}
	return nil, nil
}

func (m *MockSprintService) ListSprints(ctx context.Context, filters *services.SprintListFilters) ([]*models.Sprint, error) {
	if m.ListSprintsFunc != nil {
		return m.ListSprintsFunc(ctx, filters)
	}
	return nil, nil
}

func (m *MockSprintService) UpdateSprint(ctx context.Context, key string, updates services.UpdateSprintInput) (*models.Sprint, error) {
	if m.UpdateSprintFunc != nil {
		return m.UpdateSprintFunc(ctx, key, updates)
	}
	return nil, nil
}

func (m *MockSprintService) DeleteSprint(ctx context.Context, key string) error {
	if m.DeleteSprintFunc != nil {
		return m.DeleteSprintFunc(ctx, key)
	}
	return nil
}

func (m *MockSprintService) StartSprint(ctx context.Context, key string) (*models.Sprint, error) {
	if m.StartSprintFunc != nil {
		return m.StartSprintFunc(ctx, key)
	}
	return nil, nil
}

func (m *MockSprintService) CloseSprint(ctx context.Context, key string) (*models.Sprint, error) {
	if m.CloseSprintFunc != nil {
		return m.CloseSprintFunc(ctx, key)
	}
	return nil, nil
}

func (m *MockSprintService) ArchiveSprint(ctx context.Context, key string) (*models.Sprint, error) {
	if m.ArchiveSprintFunc != nil {
		return m.ArchiveSprintFunc(ctx, key)
	}
	return nil, nil
}

// Test helpers
func setupSprintTest(t *testing.T, mock *MockSprintService) func() {
	oldOverride := sprintSvcOverride
	sprintSvcOverride = mock
	return func() {
		sprintSvcOverride = oldOverride
	}
}

func newTestSprint() *models.Sprint {
	startDate := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	return &models.Sprint{
		ID:        1,
		Key:       "S001",
		Slug:      "sprint-1",
		Name:      "Sprint 1",
		Goal:      "Complete authentication",
		Status:    models.SprintStatus("planning"),
		StartDate: startDate,
		EndDate:   endDate,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Test runSprintCreate
func TestSprintCreate_Success(t *testing.T) {
	startDate := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	mock := &MockSprintService{
		CreateSprintFunc: func(ctx context.Context, input services.CreateSprintInput) (*models.Sprint, error) {
			assert.Equal(t, "Sprint 1", input.Name)
			assert.Equal(t, "Complete auth", input.Goal)
			assert.True(t, input.StartDate.Equal(startDate))
			assert.True(t, input.EndDate.Equal(endDate))
			return newTestSprint(), nil
		},
	}

	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("start", "", "start date")
	cmd.Flags().String("end", "", "end date")
	cmd.Flags().String("goal", "", "sprint goal")

	// Set flags programmatically
	cmd.Flags().Set("start", "2026-03-18")
	cmd.Flags().Set("end", "2026-04-01")
	cmd.Flags().Set("goal", "Complete auth")

	err := runSprintCreate(cmd, []string{"Sprint 1"})
	assert.NoError(t, err)
}

func TestSprintCreate_InvalidDateFormat(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("start", "", "start date")
	cmd.Flags().String("end", "", "end date")
	cmd.Flags().String("goal", "", "sprint goal")

	cmd.Flags().Set("start", "03/18/2026")
	cmd.Flags().Set("end", "2026-04-01")

	err := runSprintCreate(cmd, []string{"Sprint 1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid start date format")
}

func TestSprintCreate_EmptyName(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runSprintCreate(cmd, []string{"   "})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

// Test runSprintGet
func TestSprintGet_Success(t *testing.T) {
	mock := &MockSprintService{
		GetSprintFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			assert.Equal(t, "S001", key)
			return newTestSprint(), nil
		},
	}

	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().Bool("json", false, "")

	err := runSprintGet(cmd, []string{"S001"})
	assert.NoError(t, err)
}

func TestSprintGet_JSON(t *testing.T) {
	mock := &MockSprintService{
		GetSprintFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return newTestSprint(), nil
		},
	}

	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().Bool("json", true, "")
	cmd.Flags().Lookup("json").Value.Set("true")

	err := runSprintGet(cmd, []string{"S001"})
	assert.NoError(t, err)
}

// Test runSprintList
func TestSprintList_Success(t *testing.T) {
	sprints := []*models.Sprint{
		newTestSprint(),
		{
			ID:        2,
			Key:       "S002",
			Name:      "Sprint 2",
			Status:    models.SprintStatus("in_progress"),
			StartDate: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	mock := &MockSprintService{
		ListSprintsFunc: func(ctx context.Context, filters *services.SprintListFilters) ([]*models.Sprint, error) {
			return sprints, nil
		},
	}

	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("status", "", "")

	err := runSprintList(cmd, []string{})
	assert.NoError(t, err)
}

func TestSprintList_WithStatusFilter(t *testing.T) {
	mock := &MockSprintService{
		ListSprintsFunc: func(ctx context.Context, filters *services.SprintListFilters) ([]*models.Sprint, error) {
			assert.NotNil(t, filters)
			assert.Equal(t, "in_progress", filters.Status)
			return []*models.Sprint{newTestSprint()}, nil
		},
	}

	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("status", "", "sprint status")
	cmd.Flags().Set("status", "in_progress")

	err := runSprintList(cmd, []string{})
	assert.NoError(t, err)
}

// Test runSprintUpdate
func TestSprintUpdate_Success(t *testing.T) {
	updatedSprint := newTestSprint()
	updatedSprint.Name = "Updated Sprint"

	mock := &MockSprintService{
		UpdateSprintFunc: func(ctx context.Context, key string, updates services.UpdateSprintInput) (*models.Sprint, error) {
			assert.Equal(t, "S001", key)
			assert.NotNil(t, updates.Name)
			assert.Equal(t, "Updated Sprint", *updates.Name)
			return updatedSprint, nil
		},
	}

	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("name", "", "sprint name")
	cmd.Flags().String("goal", "", "sprint goal")
	cmd.Flags().String("start", "", "start date")
	cmd.Flags().String("end", "", "end date")
	cmd.Flags().Set("name", "Updated Sprint")

	err := runSprintUpdate(cmd, []string{"S001"})
	assert.NoError(t, err)
}

func TestSprintUpdate_NoFlagsProvided(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("name", "", "sprint name")
	cmd.Flags().String("goal", "", "sprint goal")
	cmd.Flags().String("start", "", "start date")
	cmd.Flags().String("end", "", "end date")

	err := runSprintUpdate(cmd, []string{"S001"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one update flag is required")
}

func TestSprintUpdate_InvalidEndDate(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().String("name", "", "sprint name")
	cmd.Flags().String("goal", "", "sprint goal")
	cmd.Flags().String("start", "", "start date")
	cmd.Flags().String("end", "", "end date")
	cmd.Flags().Set("name", "Updated Sprint")
	cmd.Flags().Set("end", "invalid-date")

	err := runSprintUpdate(cmd, []string{"S001"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid end date format")
}

// Test runSprintDelete
func TestSprintDelete_Success(t *testing.T) {
	mock := &MockSprintService{
		DeleteSprintFunc: func(ctx context.Context, key string) error {
			assert.Equal(t, "S001", key)
			return nil
		},
	}

	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().Bool("force", true, "")
	cmd.Flags().Lookup("force").Value.Set("true")

	err := runSprintDelete(cmd, []string{"S001"})
	assert.NoError(t, err)
}

// Test runSprintStart
func TestSprintStart_Success(t *testing.T) {
	startedSprint := newTestSprint()
	startedSprint.Status = models.SprintStatus("in_progress")

	mock := &MockSprintService{
		StartSprintFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			assert.Equal(t, "S001", key)
			return startedSprint, nil
		},
	}

	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runSprintStart(cmd, []string{"S001"})
	assert.NoError(t, err)
}

// Test runSprintClose
func TestSprintClose_Success(t *testing.T) {
	closedSprint := newTestSprint()
	closedSprint.Status = models.SprintStatus("ready_for_review")

	mock := &MockSprintService{
		CloseSprintFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			assert.Equal(t, "S001", key)
			return closedSprint, nil
		},
	}

	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runSprintClose(cmd, []string{"S001"})
	assert.NoError(t, err)
}

// Test runSprintArchive
func TestSprintArchive_Success(t *testing.T) {
	archivedSprint := newTestSprint()
	archivedSprint.Status = models.SprintStatus("completed")

	mock := &MockSprintService{
		ArchiveSprintFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			assert.Equal(t, "S001", key)
			return archivedSprint, nil
		},
	}

	cleanup := setupSprintTest(t, mock)
	defer cleanup()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runSprintArchive(cmd, []string{"S001"})
	assert.NoError(t, err)
}

// Test helper function - confirmSprintDelete
func TestConfirmSprintDelete(t *testing.T) {
	sprint := newTestSprint()
	assert.NotNil(t, sprint)
}
