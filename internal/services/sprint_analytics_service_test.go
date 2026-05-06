package services

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/repository/sprint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockSprintAnalyticsRepository is a test double for SprintAnalyticsRepository.
// Uses function-field pattern per project mock conventions.
type MockSprintAnalyticsRepository struct {
	GetVelocityDataFunc           func(ctx context.Context, limit int) ([]sprint.VelocityRow, error)
	GetSprintAssignedEntitiesFunc func(ctx context.Context, sprintID int64) ([]sprint.AssignedEntity, error)
	GetCompletionEventsFunc       func(ctx context.Context, sprintID int64, start, end time.Time) ([]sprint.TaskCompletionEvent, error)
	GetCycleTimeByPhaseFunc       func(ctx context.Context, sprintID int64) ([]sprint.PhaseTimeRow, error)
}

func (m *MockSprintAnalyticsRepository) GetVelocityData(ctx context.Context, limit int) ([]sprint.VelocityRow, error) {
	if m.GetVelocityDataFunc != nil {
		return m.GetVelocityDataFunc(ctx, limit)
	}
	return nil, fmt.Errorf("GetVelocityData not implemented in mock")
}

func (m *MockSprintAnalyticsRepository) GetSprintAssignedEntities(ctx context.Context, sprintID int64) ([]sprint.AssignedEntity, error) {
	if m.GetSprintAssignedEntitiesFunc != nil {
		return m.GetSprintAssignedEntitiesFunc(ctx, sprintID)
	}
	return nil, fmt.Errorf("GetSprintAssignedEntities not implemented in mock")
}

func (m *MockSprintAnalyticsRepository) GetCompletionEvents(ctx context.Context, sprintID int64, start, end time.Time) ([]sprint.TaskCompletionEvent, error) {
	if m.GetCompletionEventsFunc != nil {
		return m.GetCompletionEventsFunc(ctx, sprintID, start, end)
	}
	return nil, fmt.Errorf("GetCompletionEvents not implemented in mock")
}

func (m *MockSprintAnalyticsRepository) GetCycleTimeByPhase(ctx context.Context, sprintID int64) ([]sprint.PhaseTimeRow, error) {
	if m.GetCycleTimeByPhaseFunc != nil {
		return m.GetCycleTimeByPhaseFunc(ctx, sprintID)
	}
	return nil, fmt.Errorf("GetCycleTimeByPhase not implemented in mock")
}

// --- TC-V-01: Velocity shows last 5 completed sprints oldest-first with correct Σ size ---

func TestGetVelocity_Happy(t *testing.T) {
	// Arrange: 5 sprints in order S001..S005, known sizes.
	rows := []sprint.VelocityRow{
		{SprintKey: "S001", SprintName: "Sprint 1", CompletedSize: 10, UnsizedCompleted: 0},
		{SprintKey: "S002", SprintName: "Sprint 2", CompletedSize: 15, UnsizedCompleted: 0},
		{SprintKey: "S003", SprintName: "Sprint 3", CompletedSize: 20, UnsizedCompleted: 0},
		{SprintKey: "S004", SprintName: "Sprint 4", CompletedSize: 12, UnsizedCompleted: 0},
		{SprintKey: "S005", SprintName: "Sprint 5", CompletedSize: 18, UnsizedCompleted: 0},
	}

	var capturedLimit int
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]sprint.VelocityRow, error) {
			capturedLimit = limit
			return rows, nil
		},
	}
	svc := NewSprintAnalyticsService(analyticsRepo, nil)

	// Act
	result, err := svc.GetVelocity(context.Background(), 5)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 5, capturedLimit, "repo must be called with limit=5")
	assert.Equal(t, 5, result.SprintCount)
	assert.Equal(t, 5, len(result.Sprints))
	assert.False(t, result.InsufficientData)
	// trailing average = (10+15+20+12+18)/5 = 75/5 = 15.0
	assert.InDelta(t, 15.0, result.TrailingAverage, 0.001)
	// Verify per-sprint data
	assert.Equal(t, "S001", result.Sprints[0].Key)
	assert.Equal(t, 10, result.Sprints[0].CompletedSize)
	assert.Equal(t, "S005", result.Sprints[4].Key)
	assert.Equal(t, 18, result.Sprints[4].CompletedSize)
}

// --- TC-V-02: Velocity respects default limit (limit passed to repo) ---

func TestGetVelocity_PassesLimitToRepo(t *testing.T) {
	// Arrange: verify the n argument is forwarded to GetVelocityData
	var capturedLimit int
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]sprint.VelocityRow, error) {
			capturedLimit = limit
			// Return 5 rows regardless of limit to exercise counting
			return []sprint.VelocityRow{
				{SprintKey: "S003", CompletedSize: 5},
				{SprintKey: "S004", CompletedSize: 10},
				{SprintKey: "S005", CompletedSize: 15},
				{SprintKey: "S006", CompletedSize: 8},
				{SprintKey: "S007", CompletedSize: 12},
			}, nil
		},
	}
	svc := NewSprintAnalyticsService(analyticsRepo, nil)

	result, err := svc.GetVelocity(context.Background(), 5)

	require.NoError(t, err)
	assert.Equal(t, 5, capturedLimit, "repo must receive limit=5")
	assert.Equal(t, 5, result.SprintCount)
}

// --- TC-V-03: --sprints=1 boundary (minimum valid) ---

func TestGetVelocity_MinimumLimit(t *testing.T) {
	// Arrange: N=1 is valid; repo returns 1 row
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]sprint.VelocityRow, error) {
			assert.Equal(t, 1, limit)
			return []sprint.VelocityRow{
				{SprintKey: "S001", SprintName: "Sprint 1", CompletedSize: 8},
			}, nil
		},
	}
	svc := NewSprintAnalyticsService(analyticsRepo, nil)

	result, err := svc.GetVelocity(context.Background(), 1)

	require.NoError(t, err)
	assert.InDelta(t, 8.0, result.TrailingAverage, 0.001)
	assert.Equal(t, 1, result.SprintCount)
}

// --- TC-V-03 (N=100): Maximum boundary valid ---

func TestGetVelocity_MaximumLimit(t *testing.T) {
	var capturedLimit int
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]sprint.VelocityRow, error) {
			capturedLimit = limit
			return []sprint.VelocityRow{}, nil
		},
	}
	svc := NewSprintAnalyticsService(analyticsRepo, nil)

	_, err := svc.GetVelocity(context.Background(), 100)

	require.NoError(t, err)
	assert.Equal(t, 100, capturedLimit)
}

// --- TC-V-04: N=0 returns validation error ---

func TestGetVelocity_LimitZeroReturnsError(t *testing.T) {
	// Arrange: repo must NOT be called when validation fails
	callCount := 0
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]sprint.VelocityRow, error) {
			callCount++
			return nil, nil
		},
	}
	svc := NewSprintAnalyticsService(analyticsRepo, nil)

	result, err := svc.GetVelocity(context.Background(), 0)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "sprints must be between 1 and 100")
	assert.Equal(t, 0, callCount, "repo must NOT be called on validation failure")
}

// --- TC-V-04: N=101 returns validation error ---

func TestGetVelocity_LimitTooLargeReturnsError(t *testing.T) {
	callCount := 0
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]sprint.VelocityRow, error) {
			callCount++
			return nil, nil
		},
	}
	svc := NewSprintAnalyticsService(analyticsRepo, nil)

	result, err := svc.GetVelocity(context.Background(), 101)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "sprints must be between 1 and 100")
	assert.Equal(t, 0, callCount, "repo must NOT be called on validation failure")
}

// --- TC-V-05 & TC-V-06: Unsized entities pass through as-is from repo ---

func TestGetVelocity_UnsizedEntitiesPassThrough(t *testing.T) {
	// The repo handles COALESCE and NULL tracking; service must faithfully
	// populate UnsizedCompleted from the VelocityRow without modification.
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]sprint.VelocityRow, error) {
			return []sprint.VelocityRow{
				{SprintKey: "S001", SprintName: "Sprint 1", CompletedSize: 10, UnsizedCompleted: 3},
			}, nil
		},
	}
	svc := NewSprintAnalyticsService(analyticsRepo, nil)

	result, err := svc.GetVelocity(context.Background(), 1)

	require.NoError(t, err)
	require.Len(t, result.Sprints, 1)
	assert.Equal(t, 10, result.Sprints[0].CompletedSize)
	assert.Equal(t, 3, result.Sprints[0].UnsizedCompleted)
	// Trailing average is based on CompletedSize only (not affected by unsized)
	assert.InDelta(t, 10.0, result.TrailingAverage, 0.001)
}

// --- TC-V-07: Zero-velocity sprint included in denominator ---
// Counter-factual: buggy impl divides by non-zero sprints only → returns 15.0 instead of 10.0

func TestGetVelocity_ZeroVelocitySprintInDenominator(t *testing.T) {
	// Arrange: 3 sprints where first has CompletedSize=0
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]sprint.VelocityRow, error) {
			return []sprint.VelocityRow{
				{SprintKey: "S001", SprintName: "Sprint 1", CompletedSize: 0},
				{SprintKey: "S002", SprintName: "Sprint 2", CompletedSize: 10},
				{SprintKey: "S003", SprintName: "Sprint 3", CompletedSize: 20},
			}, nil
		},
	}
	svc := NewSprintAnalyticsService(analyticsRepo, nil)

	result, err := svc.GetVelocity(context.Background(), 3)

	require.NoError(t, err)
	// (0+10+20)/3 = 10.0, NOT (10+20)/2 = 15.0
	assert.InDelta(t, 10.0, result.TrailingAverage, 0.001)
	assert.Equal(t, 3, result.SprintCount)
}

// --- TC-V-08: All-zero velocity sprints returns 0.0 without panic ---

func TestGetVelocity_AllZeroVelocity(t *testing.T) {
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]sprint.VelocityRow, error) {
			return []sprint.VelocityRow{
				{SprintKey: "S001", CompletedSize: 0},
				{SprintKey: "S002", CompletedSize: 0},
				{SprintKey: "S003", CompletedSize: 0},
			}, nil
		},
	}
	svc := NewSprintAnalyticsService(analyticsRepo, nil)

	result, err := svc.GetVelocity(context.Background(), 3)

	require.NoError(t, err)
	assert.InDelta(t, 0.0, result.TrailingAverage, 0.001)
	assert.False(t, result.InsufficientData, "data exists, just zero velocity — InsufficientData must be false")
	assert.Equal(t, 3, result.SprintCount)
}

// --- TC-V-09: InsufficientData=true when repo returns <3 rows; no error ---
// Sub-cases: 0, 1, 2 rows

func TestGetVelocity_InsufficientData_ZeroRows(t *testing.T) {
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]sprint.VelocityRow, error) {
			return []sprint.VelocityRow{}, nil // 0 completed sprints
		},
	}
	svc := NewSprintAnalyticsService(analyticsRepo, nil)

	result, err := svc.GetVelocity(context.Background(), 5)

	require.NoError(t, err, "insufficient data must NOT return an error")
	require.NotNil(t, result)
	assert.True(t, result.InsufficientData)
	assert.Equal(t, 0, result.SprintCount)
	assert.InDelta(t, 0.0, result.TrailingAverage, 0.001)
}

func TestGetVelocity_InsufficientData_OneRow(t *testing.T) {
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]sprint.VelocityRow, error) {
			return []sprint.VelocityRow{
				{SprintKey: "S001", CompletedSize: 8},
			}, nil
		},
	}
	svc := NewSprintAnalyticsService(analyticsRepo, nil)

	result, err := svc.GetVelocity(context.Background(), 5)

	require.NoError(t, err)
	assert.True(t, result.InsufficientData)
	assert.Equal(t, 1, result.SprintCount)
}

func TestGetVelocity_InsufficientData_TwoRows(t *testing.T) {
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]sprint.VelocityRow, error) {
			return []sprint.VelocityRow{
				{SprintKey: "S001", CompletedSize: 8},
				{SprintKey: "S002", CompletedSize: 12},
			}, nil
		},
	}
	svc := NewSprintAnalyticsService(analyticsRepo, nil)

	result, err := svc.GetVelocity(context.Background(), 5)

	require.NoError(t, err)
	assert.True(t, result.InsufficientData)
	assert.Equal(t, 2, result.SprintCount)
}

func TestGetVelocity_SufficientData_ThreeRows(t *testing.T) {
	// Exactly 3 rows → InsufficientData must be false (boundary)
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]sprint.VelocityRow, error) {
			return []sprint.VelocityRow{
				{SprintKey: "S001", CompletedSize: 5},
				{SprintKey: "S002", CompletedSize: 10},
				{SprintKey: "S003", CompletedSize: 15},
			}, nil
		},
	}
	svc := NewSprintAnalyticsService(analyticsRepo, nil)

	result, err := svc.GetVelocity(context.Background(), 5)

	require.NoError(t, err)
	assert.False(t, result.InsufficientData, "3 rows is sufficient")
	assert.Equal(t, 3, result.SprintCount)
}

// --- Repo error propagates as an error ---

func TestGetVelocity_RepoError(t *testing.T) {
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]sprint.VelocityRow, error) {
			return nil, errors.New("database connection failed")
		},
	}
	svc := NewSprintAnalyticsService(analyticsRepo, nil)

	result, err := svc.GetVelocity(context.Background(), 5)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "database connection failed")
}

// --- Constructor test: ensure NewSprintAnalyticsService wires correctly ---

func TestNewSprintAnalyticsService(t *testing.T) {
	analyticsRepo := &MockSprintAnalyticsRepository{}
	svc := NewSprintAnalyticsService(analyticsRepo, nil)
	assert.NotNil(t, svc)
}
