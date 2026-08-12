package services

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	sprint "github.com/jwwelbor/shark-task-manager/internal/repository/sprint"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockSprintAnalyticsRepository is a test double for SprintAnalyticsRepository.
// Uses function-field pattern per project mock conventions.
// Uses service-owned types (AnalyticsVelocityRow, etc.) so that tests do not
// import the repository package (T-E19-F04-011 tech-debt fix).
type MockSprintAnalyticsRepository struct {
	GetVelocityDataFunc           func(ctx context.Context, limit int) ([]AnalyticsVelocityRow, error)
	GetSprintAssignedEntitiesFunc func(ctx context.Context, sprintID int64) ([]AnalyticsAssignedEntity, error)
	GetCompletionEventsFunc       func(ctx context.Context, sprintID int64, start, end time.Time) ([]AnalyticsCompletionEvent, error)
	GetCycleTimeByPhaseFunc       func(ctx context.Context, sprintID int64) ([]AnalyticsPhaseTimeRow, error)
}

func (m *MockSprintAnalyticsRepository) GetVelocityData(ctx context.Context, limit int) ([]AnalyticsVelocityRow, error) {
	if m.GetVelocityDataFunc != nil {
		return m.GetVelocityDataFunc(ctx, limit)
	}
	return nil, fmt.Errorf("GetVelocityData not implemented in mock")
}

func (m *MockSprintAnalyticsRepository) GetSprintAssignedEntities(ctx context.Context, sprintID int64) ([]AnalyticsAssignedEntity, error) {
	if m.GetSprintAssignedEntitiesFunc != nil {
		return m.GetSprintAssignedEntitiesFunc(ctx, sprintID)
	}
	return nil, fmt.Errorf("GetSprintAssignedEntities not implemented in mock")
}

func (m *MockSprintAnalyticsRepository) GetCompletionEvents(ctx context.Context, sprintID int64, start, end time.Time) ([]AnalyticsCompletionEvent, error) {
	if m.GetCompletionEventsFunc != nil {
		return m.GetCompletionEventsFunc(ctx, sprintID, start, end)
	}
	return nil, fmt.Errorf("GetCompletionEvents not implemented in mock")
}

func (m *MockSprintAnalyticsRepository) GetCycleTimeByPhase(ctx context.Context, sprintID int64) ([]AnalyticsPhaseTimeRow, error) {
	if m.GetCycleTimeByPhaseFunc != nil {
		return m.GetCycleTimeByPhaseFunc(ctx, sprintID)
	}
	return nil, fmt.Errorf("GetCycleTimeByPhase not implemented in mock")
}

// --- TC-V-01: Velocity shows last 5 completed sprints oldest-first with correct Σ size ---

func TestGetVelocity_Happy(t *testing.T) {
	// Arrange: 5 sprints in order S001..S005, known sizes.
	rows := []AnalyticsVelocityRow{
		{SprintKey: "S001", SprintName: "Sprint 1", CompletedSize: 10, UnsizedCompleted: 0},
		{SprintKey: "S002", SprintName: "Sprint 2", CompletedSize: 15, UnsizedCompleted: 0},
		{SprintKey: "S003", SprintName: "Sprint 3", CompletedSize: 20, UnsizedCompleted: 0},
		{SprintKey: "S004", SprintName: "Sprint 4", CompletedSize: 12, UnsizedCompleted: 0},
		{SprintKey: "S005", SprintName: "Sprint 5", CompletedSize: 18, UnsizedCompleted: 0},
	}

	var capturedLimit int
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]AnalyticsVelocityRow, error) {
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
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]AnalyticsVelocityRow, error) {
			capturedLimit = limit
			// Return 5 rows regardless of limit to exercise counting
			return []AnalyticsVelocityRow{
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
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]AnalyticsVelocityRow, error) {
			assert.Equal(t, 1, limit)
			return []AnalyticsVelocityRow{
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
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]AnalyticsVelocityRow, error) {
			capturedLimit = limit
			return []AnalyticsVelocityRow{}, nil
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
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]AnalyticsVelocityRow, error) {
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
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]AnalyticsVelocityRow, error) {
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
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]AnalyticsVelocityRow, error) {
			return []AnalyticsVelocityRow{
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
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]AnalyticsVelocityRow, error) {
			return []AnalyticsVelocityRow{
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
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]AnalyticsVelocityRow, error) {
			return []AnalyticsVelocityRow{
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
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]AnalyticsVelocityRow, error) {
			return []AnalyticsVelocityRow{}, nil // 0 completed sprints
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
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]AnalyticsVelocityRow, error) {
			return []AnalyticsVelocityRow{
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
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]AnalyticsVelocityRow, error) {
			return []AnalyticsVelocityRow{
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
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]AnalyticsVelocityRow, error) {
			return []AnalyticsVelocityRow{
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
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]AnalyticsVelocityRow, error) {
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

// =============================================================================
// GetBurndown tests (TC-B-01 through TC-B-12)
// =============================================================================

// testTime is a fixed reference date for deterministic time assertions.
// All sprint scenarios in these tests are constructed relative to this base.
var testBase = time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)

// --- TC-B-01: No key → uses active sprint via sprintRepo.List ---
// Counter-factual: a buggy impl that hard-codes a sprint key when key="" returns wrong sprint.

func TestGetBurndown_ActiveSprint_UsedWhenKeyEmpty(t *testing.T) {
	// Arrange: active sprint spanning 3 days: day0, day1, day2
	startDate := testBase
	endDate := testBase.AddDate(0, 0, 2)
	activeSprint := &models.Sprint{
		ID:        42,
		Key:       "S010",
		Name:      "Sprint 10",
		Status:    "active",
		StartDate: startDate,
		EndDate:   endDate,
	}

	var listCalled bool
	sprintRepo := &MockSprintRepository{
		ListFunc: func(ctx context.Context, filters *sprint.SprintListFilters) ([]*models.Sprint, error) {
			listCalled = true
			// Verify the filter requests active sprints
			require.NotNil(t, filters)
			return []*models.Sprint{activeSprint}, nil
		},
	}
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetSprintAssignedEntitiesFunc: func(ctx context.Context, sprintID int64) ([]AnalyticsAssignedEntity, error) {
			assert.Equal(t, int64(42), sprintID)
			sz := 10
			return []AnalyticsAssignedEntity{
				{EntityType: "task", EntityID: 1, AssignedAt: startDate, RemovedAt: nil, Size: &sz},
			}, nil
		},
		GetCompletionEventsFunc: func(ctx context.Context, sprintID int64, start, end time.Time) ([]AnalyticsCompletionEvent, error) {
			return []AnalyticsCompletionEvent{}, nil
		},
	}

	// inject a fixed "today" = day2 so all days are past
	today := endDate
	svc := newSprintAnalyticsServiceWithClock(analyticsRepo, sprintRepo, func() time.Time { return today })

	// Act
	result, err := svc.GetBurndown(context.Background(), "")

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, listCalled, "sprintRepo.List must be called when key is empty")
	assert.Equal(t, "S010", result.SprintKey)
	assert.Greater(t, len(result.DataPoints), 0)
}

// --- TC-B-02: Key provided → sprintRepo.GetByKey is called ---
// Counter-factual: a buggy impl that always calls List ignores the key.

func TestGetBurndown_KeyProvided_UsesGetByKey(t *testing.T) {
	startDate := testBase
	endDate := testBase.AddDate(0, 0, 1)
	s024 := &models.Sprint{
		ID:        24,
		Key:       "S024",
		Name:      "Sprint 24",
		Status:    "completed",
		StartDate: startDate,
		EndDate:   endDate,
	}

	var getByKeyCalled bool
	var listCalled bool
	sprintRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			getByKeyCalled = true
			assert.Equal(t, "S024", key)
			return s024, nil
		},
		ListFunc: func(ctx context.Context, filters *sprint.SprintListFilters) ([]*models.Sprint, error) {
			listCalled = true
			return nil, nil
		},
	}
	sz := 5
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetSprintAssignedEntitiesFunc: func(ctx context.Context, sprintID int64) ([]AnalyticsAssignedEntity, error) {
			return []AnalyticsAssignedEntity{
				{EntityType: "task", EntityID: 1, AssignedAt: startDate, RemovedAt: nil, Size: &sz},
			}, nil
		},
		GetCompletionEventsFunc: func(ctx context.Context, sprintID int64, start, end time.Time) ([]AnalyticsCompletionEvent, error) {
			return []AnalyticsCompletionEvent{}, nil
		},
	}

	today := endDate.Add(24 * time.Hour) // past sprint
	svc := newSprintAnalyticsServiceWithClock(analyticsRepo, sprintRepo, func() time.Time { return today })

	result, err := svc.GetBurndown(context.Background(), "S024")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, getByKeyCalled, "GetByKey must be called when key is provided")
	assert.False(t, listCalled, "List must NOT be called when key is provided")
	assert.Equal(t, "S024", result.SprintKey)
}

// --- TC-B-04: Planning sprint returns informational result (no data points, no error) ---
// Counter-factual: a buggy impl that returns err != nil for planning status causes CLI to exit 1.

func TestGetBurndown_PlanningStatus_InformationalNoError(t *testing.T) {
	planningSprint := &models.Sprint{
		ID:        1,
		Key:       "S001",
		Name:      "Sprint 1",
		Status:    "planning",
		StartDate: testBase,
		EndDate:   testBase.AddDate(0, 0, 13),
	}

	sprintRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return planningSprint, nil
		},
	}

	svc := newSprintAnalyticsServiceWithClock(&MockSprintAnalyticsRepository{}, sprintRepo, func() time.Time { return testBase })

	result, err := svc.GetBurndown(context.Background(), "S001")

	// Must NOT return error (informational, not error path)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.DataPoints, "planning sprint has no burndown data")
}

// --- TC-B-05: Ideal burndown is linear from total to 0 over 14 days ---
// Counter-factual: a buggy impl using integer division would return wrong ideal_remaining values.
// day0: ideal=35.0, day1=35×13/14=32.5, day2=35×12/14=30.0, day13=0.0

func TestGetBurndown_IdealLine_Linear14Days(t *testing.T) {
	startDate := testBase
	endDate := testBase.AddDate(0, 0, 13) // 14-day sprint (days 0-13)

	s := &models.Sprint{
		ID:        25,
		Key:       "S025",
		Name:      "Sprint 25",
		Status:    "completed",
		StartDate: startDate,
		EndDate:   endDate,
	}

	sz := 35
	sprintRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) { return s, nil },
	}
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetSprintAssignedEntitiesFunc: func(ctx context.Context, sprintID int64) ([]AnalyticsAssignedEntity, error) {
			return []AnalyticsAssignedEntity{
				{EntityType: "task", EntityID: 1, AssignedAt: startDate, RemovedAt: nil, Size: &sz},
			}, nil
		},
		GetCompletionEventsFunc: func(ctx context.Context, sprintID int64, start, end time.Time) ([]AnalyticsCompletionEvent, error) {
			return []AnalyticsCompletionEvent{}, nil
		},
	}

	today := endDate.Add(24 * time.Hour) // sprint is in the past, all days have actual
	svc := newSprintAnalyticsServiceWithClock(analyticsRepo, sprintRepo, func() time.Time { return today })

	result, err := svc.GetBurndown(context.Background(), "S025")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 35, result.TotalSize)

	// 14 days: index 0 = day0 ... index 13 = day13
	require.Len(t, result.DataPoints, 14)

	// day0: ideal = 35.0 (full sprint, no burn yet)
	assert.InDelta(t, 35.0, result.DataPoints[0].IdealRemaining, 0.001, "day0 ideal")
	// day1: ideal = 35 × (13/14)
	assert.InDelta(t, 35.0*13.0/14.0, result.DataPoints[1].IdealRemaining, 0.001, "day1 ideal")
	// day2: ideal = 35 × (12/14)
	assert.InDelta(t, 35.0*12.0/14.0, result.DataPoints[2].IdealRemaining, 0.001, "day2 ideal")
	// day13: ideal = 0.0
	assert.InDelta(t, 0.0, result.DataPoints[13].IdealRemaining, 0.001, "day13 ideal")
}

// --- TC-B-06: Ideal resets piecewise when entity added mid-sprint ---
// Counter-factual: a buggy impl that ignores mid-sprint adds keeps the original ideal line.
// Sprint 14 days, original_size=35. On day 3, entity size=7 added.
// day0: ideal=35.0, day3: new_total=42, days_remaining=11, ideal=42.0 (reset)

func TestGetBurndown_IdealLine_PiecewiseResetOnEntityAdd(t *testing.T) {
	startDate := testBase
	endDate := testBase.AddDate(0, 0, 13) // 14-day sprint

	s := &models.Sprint{
		ID:        26,
		Key:       "S026",
		Name:      "Sprint 26",
		Status:    "completed",
		StartDate: startDate,
		EndDate:   endDate,
	}

	day3 := startDate.AddDate(0, 0, 3)
	sz35 := 35
	sz7 := 7

	sprintRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) { return s, nil },
	}
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetSprintAssignedEntitiesFunc: func(ctx context.Context, sprintID int64) ([]AnalyticsAssignedEntity, error) {
			return []AnalyticsAssignedEntity{
				// Original entity assigned at start
				{EntityType: "task", EntityID: 1, AssignedAt: startDate, RemovedAt: nil, Size: &sz35},
				// New entity added on day 3
				{EntityType: "task", EntityID: 2, AssignedAt: day3, RemovedAt: nil, Size: &sz7},
			}, nil
		},
		GetCompletionEventsFunc: func(ctx context.Context, sprintID int64, start, end time.Time) ([]AnalyticsCompletionEvent, error) {
			return []AnalyticsCompletionEvent{}, nil
		},
	}

	today := endDate.Add(24 * time.Hour) // past sprint
	svc := newSprintAnalyticsServiceWithClock(analyticsRepo, sprintRepo, func() time.Time { return today })

	result, err := svc.GetBurndown(context.Background(), "S026")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.DataPoints, 14)

	// Before add (day0): ideal based on original 35
	assert.InDelta(t, 35.0, result.DataPoints[0].IdealRemaining, 0.001, "day0 ideal before add")
	// After piecewise reset on day3: new total=42, 11 days remaining (days 3..13)
	// ideal_remaining = 42.0 (start of new segment)
	assert.InDelta(t, 42.0, result.DataPoints[3].IdealRemaining, 0.001, "day3 ideal after piecewise reset")
	// day4: 42 × (10/11)
	assert.InDelta(t, 42.0*10.0/11.0, result.DataPoints[4].IdealRemaining, 0.001, "day4 ideal after reset")
}

// --- TC-B-07: Actual remaining reconstructed from task_history completion events ---
// Counter-factual: a buggy impl that uses current status (not history) marks entity as
// completed on all days instead of only from day3 onward.
// Tasks: sizes 10, 15, 17. T001 (size=10) completed at end of day3.

func TestGetBurndown_ActualRemaining_FromCompletionEvents(t *testing.T) {
	startDate := testBase
	endDate := testBase.AddDate(0, 0, 6) // 7-day sprint

	s := &models.Sprint{
		ID:        27,
		Key:       "S027",
		Name:      "Sprint 27",
		Status:    "completed",
		StartDate: startDate,
		EndDate:   endDate,
	}

	sz10, sz15, sz17 := 10, 15, 17
	// T001 completes near end of day3
	completedAt := startDate.AddDate(0, 0, 3).Add(23 * time.Hour)

	sprintRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) { return s, nil },
	}
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetSprintAssignedEntitiesFunc: func(ctx context.Context, sprintID int64) ([]AnalyticsAssignedEntity, error) {
			return []AnalyticsAssignedEntity{
				{EntityType: "task", EntityID: 1, AssignedAt: startDate, RemovedAt: nil, Size: &sz10},
				{EntityType: "task", EntityID: 2, AssignedAt: startDate, RemovedAt: nil, Size: &sz15},
				{EntityType: "task", EntityID: 3, AssignedAt: startDate, RemovedAt: nil, Size: &sz17},
			}, nil
		},
		GetCompletionEventsFunc: func(ctx context.Context, sprintID int64, start, end time.Time) ([]AnalyticsCompletionEvent, error) {
			return []AnalyticsCompletionEvent{
				{EntityID: 1, EntityType: "task", NewStatus: "completed", Timestamp: completedAt},
			}, nil
		},
	}

	today := endDate.Add(24 * time.Hour) // past sprint
	svc := newSprintAnalyticsServiceWithClock(analyticsRepo, sprintRepo, func() time.Time { return today })

	result, err := svc.GetBurndown(context.Background(), "S027")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.DataPoints, 7)

	// day0..day2: all 3 tasks active → actual = 42
	for i := 0; i <= 2; i++ {
		require.NotNil(t, result.DataPoints[i].ActualRemaining, "day%d must have actual", i)
		assert.InDelta(t, 42.0, *result.DataPoints[i].ActualRemaining, 0.001, "day%d actual", i)
	}
	// day3 onward: T001 completed, actual = 32
	for i := 3; i <= 6; i++ {
		require.NotNil(t, result.DataPoints[i].ActualRemaining, "day%d must have actual", i)
		assert.InDelta(t, 32.0, *result.DataPoints[i].ActualRemaining, 0.001, "day%d actual after completion", i)
	}
}

func TestGetBurndown_NonTaskCompletionReducesActualRemaining(t *testing.T) {
	startDate := testBase
	endDate := testBase.AddDate(0, 0, 2)

	s := &models.Sprint{
		ID:        30,
		Key:       "S030",
		Name:      "Sprint 30",
		Status:    "completed",
		StartDate: startDate,
		EndDate:   endDate,
	}

	sz8, sz13 := 8, 13
	completedAt := startDate.AddDate(0, 0, 1).Add(2 * time.Hour)

	sprintRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) { return s, nil },
	}
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetSprintAssignedEntitiesFunc: func(ctx context.Context, sprintID int64) ([]AnalyticsAssignedEntity, error) {
			return []AnalyticsAssignedEntity{
				{EntityType: "bug", EntityID: 1, AssignedAt: startDate, RemovedAt: nil, Size: &sz8},
				{EntityType: "tech_debt", EntityID: 2, AssignedAt: startDate, RemovedAt: nil, Size: &sz13},
			}, nil
		},
		GetCompletionEventsFunc: func(ctx context.Context, sprintID int64, start, end time.Time) ([]AnalyticsCompletionEvent, error) {
			return []AnalyticsCompletionEvent{
				{EntityID: 1, EntityType: "bug", NewStatus: "resolved", Timestamp: completedAt},
			}, nil
		},
	}

	today := endDate.Add(24 * time.Hour)
	svc := newSprintAnalyticsServiceWithClock(analyticsRepo, sprintRepo, func() time.Time { return today })

	result, err := svc.GetBurndown(context.Background(), "S030")

	require.NoError(t, err)
	require.Len(t, result.DataPoints, 3)
	require.NotNil(t, result.DataPoints[0].ActualRemaining)
	require.NotNil(t, result.DataPoints[1].ActualRemaining)
	assert.InDelta(t, 21.0, *result.DataPoints[0].ActualRemaining, 0.001)
	assert.InDelta(t, 13.0, *result.DataPoints[1].ActualRemaining, 0.001)
}

// --- TC-B-09: UnsizedRemaining present in every data point ---
// Counter-factual: a buggy impl that initializes UnsizedRemaining=0 always fails day0 assertion.

func TestGetBurndown_UnsizedRemaining_InEveryDataPoint(t *testing.T) {
	startDate := testBase
	endDate := testBase.AddDate(0, 0, 2)

	s := &models.Sprint{
		ID:        28,
		Key:       "S028",
		Name:      "Sprint 28",
		Status:    "active",
		StartDate: startDate,
		EndDate:   endDate,
	}

	sz5 := 5
	sprintRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) { return s, nil },
	}
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetSprintAssignedEntitiesFunc: func(ctx context.Context, sprintID int64) ([]AnalyticsAssignedEntity, error) {
			return []AnalyticsAssignedEntity{
				// Sized entity
				{EntityType: "task", EntityID: 1, AssignedAt: startDate, RemovedAt: nil, Size: &sz5},
				// 2 unsized entities (Size == nil)
				{EntityType: "task", EntityID: 2, AssignedAt: startDate, RemovedAt: nil, Size: nil},
				{EntityType: "task", EntityID: 3, AssignedAt: startDate, RemovedAt: nil, Size: nil},
			}, nil
		},
		GetCompletionEventsFunc: func(ctx context.Context, sprintID int64, start, end time.Time) ([]AnalyticsCompletionEvent, error) {
			return []AnalyticsCompletionEvent{}, nil
		},
	}

	// today = day0 so day1 and day2 are future
	today := startDate
	svc := newSprintAnalyticsServiceWithClock(analyticsRepo, sprintRepo, func() time.Time { return today })

	result, err := svc.GetBurndown(context.Background(), "S028")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 2, result.UnsizedTotal, "UnsizedTotal must reflect 2 unsized entities")
	require.Greater(t, len(result.DataPoints), 0)

	// UnsizedRemaining must be 2 in every data point
	for i, dp := range result.DataPoints {
		assert.Equal(t, 2, dp.UnsizedRemaining, "data point %d must have UnsizedRemaining=2", i)
	}
}

// --- TC-B-12: Future days have ActualRemaining == nil; today's day has a value ---
// Counter-factual: a buggy impl that sets ActualRemaining=0.0 for future days would
// fail the nil check, and JSON would include spurious actual_remaining: 0.

func TestGetBurndown_FutureDays_NilActualRemaining(t *testing.T) {
	startDate := testBase
	endDate := testBase.AddDate(0, 0, 13) // 14-day sprint

	s := &models.Sprint{
		ID:        29,
		Key:       "S029",
		Name:      "Sprint 29",
		Status:    "active",
		StartDate: startDate,
		EndDate:   endDate,
	}

	sz10 := 10
	sprintRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) { return s, nil },
	}
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetSprintAssignedEntitiesFunc: func(ctx context.Context, sprintID int64) ([]AnalyticsAssignedEntity, error) {
			return []AnalyticsAssignedEntity{
				{EntityType: "task", EntityID: 1, AssignedAt: startDate, RemovedAt: nil, Size: &sz10},
			}, nil
		},
		GetCompletionEventsFunc: func(ctx context.Context, sprintID int64, start, end time.Time) ([]AnalyticsCompletionEvent, error) {
			return []AnalyticsCompletionEvent{}, nil
		},
	}

	// "today" = day 5 (inclusive boundary: days 0-5 get actual, days 6-13 are nil)
	today := startDate.AddDate(0, 0, 5)
	svc := newSprintAnalyticsServiceWithClock(analyticsRepo, sprintRepo, func() time.Time { return today })

	result, err := svc.GetBurndown(context.Background(), "S029")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.DataPoints, 14)

	// days 0..5: ActualRemaining must be non-nil
	for i := 0; i <= 5; i++ {
		assert.NotNil(t, result.DataPoints[i].ActualRemaining, "day%d must have actual", i)
	}
	// days 6..13: ActualRemaining must be nil
	for i := 6; i <= 13; i++ {
		assert.Nil(t, result.DataPoints[i].ActualRemaining, "day%d must be nil (future)", i)
	}
}

// --- TC-B-03: Valid statuses (active, closing, completed, archived) return data ---

func TestGetBurndown_ValidStatuses_ReturnData(t *testing.T) {
	startDate := testBase
	endDate := testBase.AddDate(0, 0, 1)
	sz5 := 5

	for _, status := range []string{"active", "closing", "completed", "archived"} {
		status := status
		t.Run(status, func(t *testing.T) {
			s := &models.Sprint{
				ID:        1,
				Key:       "S001",
				Name:      "Sprint 1",
				Status:    models.SprintStatus(status),
				StartDate: startDate,
				EndDate:   endDate,
			}
			sprintRepo := &MockSprintRepository{
				GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) { return s, nil },
			}
			analyticsRepo := &MockSprintAnalyticsRepository{
				GetSprintAssignedEntitiesFunc: func(ctx context.Context, sprintID int64) ([]AnalyticsAssignedEntity, error) {
					return []AnalyticsAssignedEntity{
						{EntityType: "task", EntityID: 1, AssignedAt: startDate, RemovedAt: nil, Size: &sz5},
					}, nil
				},
				GetCompletionEventsFunc: func(ctx context.Context, sprintID int64, start, end time.Time) ([]AnalyticsCompletionEvent, error) {
					return []AnalyticsCompletionEvent{}, nil
				},
			}
			today := endDate.Add(24 * time.Hour)
			svc := newSprintAnalyticsServiceWithClock(analyticsRepo, sprintRepo, func() time.Time { return today })

			result, err := svc.GetBurndown(context.Background(), "S001")

			require.NoError(t, err, "status=%s must not return error", status)
			require.NotNil(t, result, "status=%s must return result", status)
			assert.Greater(t, len(result.DataPoints), 0, "status=%s must have data points", status)
		})
	}
}

// =============================================================================
// GetSummary tests (TC-S-01 through TC-S-08)
// =============================================================================

// sprintHelper builds a minimal Sprint for summary tests.
func sprintHelper(key, status string, startDate, endDate time.Time) *models.Sprint {
	return &models.Sprint{
		ID:        42,
		Key:       key,
		Name:      "Sprint " + key,
		Status:    models.SprintStatus(status),
		StartDate: startDate,
		EndDate:   endDate,
	}
}

// --- TC-S-01: GetSummary for completed sprint returns non-nil result, no error.
// detailed=false → CycleTimeByPhase = nil.
// Counter-factual: a buggy impl that skips status validation returns data for planning sprint.
func TestGetSummary_CompletedSprint(t *testing.T) {
	startDate := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	sprintRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			assert.Equal(t, "S024", key)
			return sprintHelper("S024", "completed", startDate, endDate), nil
		},
	}
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetSprintAssignedEntitiesFunc: func(ctx context.Context, sprintID int64) ([]AnalyticsAssignedEntity, error) {
			assert.Equal(t, int64(42), sprintID)
			return []AnalyticsAssignedEntity{}, nil
		},
		GetCompletionEventsFunc: func(ctx context.Context, sprintID int64, start, end time.Time) ([]AnalyticsCompletionEvent, error) {
			return []AnalyticsCompletionEvent{}, nil
		},
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]AnalyticsVelocityRow, error) {
			return []AnalyticsVelocityRow{}, nil
		},
	}

	svc := NewSprintAnalyticsService(analyticsRepo, sprintRepo)
	result, err := svc.GetSummary(context.Background(), "S024", false)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "S024", result.SprintKey)
	assert.Nil(t, result.CycleTimeByPhase, "CycleTimeByPhase must be nil when detailed=false")
	assert.Nil(t, result.AddedMidSprintCount, "detailed pointer fields must be nil when detailed=false")
}

// --- TC-S-01 archived variant: archived sprint is also accepted.
func TestGetSummary_ArchivedSprintAccepted(t *testing.T) {
	startDate := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	sprintRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return sprintHelper("S024", "archived", startDate, endDate), nil
		},
	}
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetSprintAssignedEntitiesFunc: func(ctx context.Context, sprintID int64) ([]AnalyticsAssignedEntity, error) {
			return []AnalyticsAssignedEntity{}, nil
		},
		GetCompletionEventsFunc: func(ctx context.Context, sprintID int64, start, end time.Time) ([]AnalyticsCompletionEvent, error) {
			return []AnalyticsCompletionEvent{}, nil
		},
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]AnalyticsVelocityRow, error) {
			return []AnalyticsVelocityRow{}, nil
		},
	}

	svc := NewSprintAnalyticsService(analyticsRepo, sprintRepo)
	result, err := svc.GetSummary(context.Background(), "S024", false)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "S024", result.SprintKey)
}

func TestGetSummary_UsesConfiguredDonePhase(t *testing.T) {
	startDate := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	sprintRepo := &MockSprintRepository{GetByKeyFunc: func(context.Context, string) (*models.Sprint, error) {
		return sprintHelper("S024", "wrapped", startDate, endDate), nil
	}}
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetSprintAssignedEntitiesFunc: func(context.Context, int64) ([]AnalyticsAssignedEntity, error) {
			return []AnalyticsAssignedEntity{}, nil
		},
		GetCompletionEventsFunc: func(context.Context, int64, time.Time, time.Time) ([]AnalyticsCompletionEvent, error) {
			return []AnalyticsCompletionEvent{}, nil
		},
		GetVelocityDataFunc: func(context.Context, int) ([]AnalyticsVelocityRow, error) { return []AnalyticsVelocityRow{}, nil },
	}
	sprintWorkflow := &config.WorkflowConfig{Steps: map[string]*config.Step{"wrapped": {Phase: "done"}, "archived": {Phase: "done", Terminal: true}}, StatusMetadata: map[string]config.StatusMetadata{"wrapped": {Phase: "done"}, "archived": {Phase: "done"}}}
	sprintWorkflow.DeriveLegacy()
	svc := NewSprintAnalyticsService(analyticsRepo, sprintRepo)
	svc.SetWorkflow(workflow.NewServiceFromMultiLevel(&config.MultiLevelWorkflow{Sprint: sprintWorkflow}))
	result, err := svc.GetSummary(context.Background(), "S024", false)
	require.NoError(t, err)
	assert.Equal(t, "S024", result.SprintKey)
}

// --- TC-S-02: Summary for planning or active sprint returns error (informational signal).
// Counter-factual: buggy impl returning nil error causes CLI to not show informational message.
func TestGetSummary_InvalidStatus_ReturnsError(t *testing.T) {
	startDate := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	for _, status := range []string{"planning", "active"} {
		status := status
		t.Run(status, func(t *testing.T) {
			callCount := 0
			sprintRepo := &MockSprintRepository{
				GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
					return sprintHelper("S024", status, startDate, endDate), nil
				},
			}
			analyticsRepo := &MockSprintAnalyticsRepository{
				GetSprintAssignedEntitiesFunc: func(ctx context.Context, sprintID int64) ([]AnalyticsAssignedEntity, error) {
					callCount++
					return nil, nil
				},
				GetCompletionEventsFunc: func(ctx context.Context, sprintID int64, start, end time.Time) ([]AnalyticsCompletionEvent, error) {
					callCount++
					return nil, nil
				},
			}

			svc := NewSprintAnalyticsService(analyticsRepo, sprintRepo)
			result, err := svc.GetSummary(context.Background(), "S024", false)

			require.Error(t, err, "non-completed/archived sprint must return an error signal")
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "completed or archived", "error must describe the restriction")
			assert.Equal(t, 0, callCount, "analytics repo must NOT be called for invalid status")
		})
	}
}

// --- TC-S-03: Base summary contains all required fields.
// Counter-factual: buggy impl omits VelocityDeltaPct → field-presence assertion fails.
func TestGetSummary_BaseFieldsComplete(t *testing.T) {
	startDate := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	size5 := 5
	assignedBefore := startDate.Add(-1 * time.Hour) // before start

	// 8 planned entities (size 5 each), 8 completed.
	entities := make([]AnalyticsAssignedEntity, 8)
	for i := range entities {
		entities[i] = AnalyticsAssignedEntity{
			EntityType: "task",
			EntityID:   int64(i + 1),
			AssignedAt: assignedBefore,
			Size:       &size5,
		}
	}
	completionEvents := make([]AnalyticsCompletionEvent, 8)
	for i := range completionEvents {
		completionEvents[i] = AnalyticsCompletionEvent{
			EntityID: int64(i + 1), EntityType: "task", NewStatus: "completed",
			Timestamp: endDate.Add(-2 * 24 * time.Hour),
		}
	}

	// Previous 5 sprints trailing average = 38.0
	prevRows := []AnalyticsVelocityRow{
		{SprintKey: "S019", CompletedSize: 36},
		{SprintKey: "S020", CompletedSize: 38},
		{SprintKey: "S021", CompletedSize: 40},
		{SprintKey: "S022", CompletedSize: 38},
		{SprintKey: "S023", CompletedSize: 38},
	}

	sprintRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return sprintHelper("S024", "completed", startDate, endDate), nil
		},
	}
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetSprintAssignedEntitiesFunc: func(ctx context.Context, sprintID int64) ([]AnalyticsAssignedEntity, error) {
			return entities, nil
		},
		GetCompletionEventsFunc: func(ctx context.Context, sprintID int64, start, end time.Time) ([]AnalyticsCompletionEvent, error) {
			return completionEvents, nil
		},
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]AnalyticsVelocityRow, error) {
			return prevRows, nil
		},
	}

	svc := NewSprintAnalyticsService(analyticsRepo, sprintRepo)
	result, err := svc.GetSummary(context.Background(), "S024", false)

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "S024", result.SprintKey)
	assert.Equal(t, "Sprint S024", result.SprintName)
	assert.Equal(t, 40, result.PlannedSize, "8 tasks × 5 = 40")
	assert.Equal(t, 40, result.CompletedSize, "8 completed × 5 = 40")
	assert.InDelta(t, 100.0, result.CompletionPctBySize, 0.01, "40/40 = 100%")
	assert.Equal(t, 8, result.PlannedCount)
	assert.Equal(t, 8, result.CompletedCount)
	assert.Equal(t, 40, result.VelocityThisSprint)
	// trailing avg = (36+38+40+38+38)/5 = 190/5 = 38.0
	assert.InDelta(t, 38.0, result.TrailingAvgVelocity, 0.01)
	// delta = 40 - 38 = 2
	assert.InDelta(t, 2.0, result.VelocityDelta, 0.01)
	// delta pct = 2/38 * 100 ≈ 5.26%
	assert.InDelta(t, 5.26, result.VelocityDeltaPct, 0.1)
}

func TestGetSummary_CountsOnlyTerminalCompletionEvents(t *testing.T) {
	startDate := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	size5 := 5

	sprintRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return sprintHelper("S024", "completed", startDate, endDate), nil
		},
	}
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetSprintAssignedEntitiesFunc: func(ctx context.Context, sprintID int64) ([]AnalyticsAssignedEntity, error) {
			return []AnalyticsAssignedEntity{
				{EntityType: "task", EntityID: 1, AssignedAt: startDate, Size: &size5},
				{EntityType: "task", EntityID: 2, AssignedAt: startDate, Size: &size5},
			}, nil
		},
		GetCompletionEventsFunc: func(ctx context.Context, sprintID int64, start, end time.Time) ([]AnalyticsCompletionEvent, error) {
			return []AnalyticsCompletionEvent{
				{EntityID: 1, EntityType: "task", NewStatus: "in_progress", Timestamp: startDate.Add(24 * time.Hour)},
				{EntityID: 2, EntityType: "task", NewStatus: "completed", Timestamp: startDate.Add(48 * time.Hour)},
			}, nil
		},
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]AnalyticsVelocityRow, error) {
			return []AnalyticsVelocityRow{}, nil
		},
	}

	svc := NewSprintAnalyticsService(analyticsRepo, sprintRepo)
	result, err := svc.GetSummary(context.Background(), "S024", false)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.CompletedCount)
	assert.Equal(t, 5, result.CompletedSize)
	assert.Equal(t, 5, result.VelocityThisSprint)
	assert.InDelta(t, 50.0, result.CompletionPctBySize, 0.001)
}

func TestGetSummary_ReturnsVelocityDataError(t *testing.T) {
	startDate := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	velocityErr := errors.New("velocity backend unavailable")

	sprintRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return sprintHelper("S024", "completed", startDate, endDate), nil
		},
	}
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetSprintAssignedEntitiesFunc: func(ctx context.Context, sprintID int64) ([]AnalyticsAssignedEntity, error) {
			return []AnalyticsAssignedEntity{}, nil
		},
		GetCompletionEventsFunc: func(ctx context.Context, sprintID int64, start, end time.Time) ([]AnalyticsCompletionEvent, error) {
			return []AnalyticsCompletionEvent{}, nil
		},
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]AnalyticsVelocityRow, error) {
			return nil, velocityErr
		},
	}

	svc := NewSprintAnalyticsService(analyticsRepo, sprintRepo)
	result, err := svc.GetSummary(context.Background(), "S024", false)

	require.Error(t, err)
	assert.ErrorIs(t, err, velocityErr)
	assert.Nil(t, result)
}

// --- TC-S-04: planned_size=0 → CompletionPctBySize=0.0 (no divide-by-zero panic).
// Counter-factual: buggy impl panics on 0/0.
func TestGetSummary_ZeroPlannedSize_NoPanic(t *testing.T) {
	startDate := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	sprintRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return sprintHelper("S024", "completed", startDate, endDate), nil
		},
	}
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetSprintAssignedEntitiesFunc: func(ctx context.Context, sprintID int64) ([]AnalyticsAssignedEntity, error) {
			return []AnalyticsAssignedEntity{}, nil // empty sprint
		},
		GetCompletionEventsFunc: func(ctx context.Context, sprintID int64, start, end time.Time) ([]AnalyticsCompletionEvent, error) {
			return []AnalyticsCompletionEvent{}, nil
		},
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]AnalyticsVelocityRow, error) {
			return []AnalyticsVelocityRow{}, nil
		},
	}

	svc := NewSprintAnalyticsService(analyticsRepo, sprintRepo)

	assert.NotPanics(t, func() {
		result, err := svc.GetSummary(context.Background(), "S024", false)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 0.0, result.CompletionPctBySize, "0/0 must return 0.0, not panic")
	})
}

// --- TC-S-05: detailed=true with cycle-time data populates detailed fields.
// Counter-factual: buggy impl ignoring detailed=true returns nil for all detailed fields.
func TestGetSummary_DetailedWithCycleTimeData(t *testing.T) {
	startDate := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	size5 := 5
	assignedBefore := startDate.Add(-1 * time.Hour)
	midSprint := startDate.Add(3 * 24 * time.Hour) // day 3

	entities := []AnalyticsAssignedEntity{
		{EntityType: "task", EntityID: 1, AssignedAt: assignedBefore, Size: &size5},
		{EntityType: "task", EntityID: 2, AssignedAt: assignedBefore, Size: &size5},
		{EntityType: "task", EntityID: 3, AssignedAt: midSprint, Size: &size5}, // mid-sprint add
	}
	completionEvents := []AnalyticsCompletionEvent{
		{EntityID: 1, EntityType: "task", NewStatus: "completed", Timestamp: endDate.Add(-1 * time.Hour)},
		{EntityID: 2, EntityType: "task", NewStatus: "completed", Timestamp: endDate.Add(-1 * time.Hour)},
	}
	phaseRows := []AnalyticsPhaseTimeRow{
		{Phase: "in_progress", AverageDays: 3.5},
		{Phase: "ready_for_review", AverageDays: 1.2},
	}

	sprintRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return sprintHelper("S024", "completed", startDate, endDate), nil
		},
	}
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetSprintAssignedEntitiesFunc: func(ctx context.Context, sprintID int64) ([]AnalyticsAssignedEntity, error) {
			return entities, nil
		},
		GetCompletionEventsFunc: func(ctx context.Context, sprintID int64, start, end time.Time) ([]AnalyticsCompletionEvent, error) {
			return completionEvents, nil
		},
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]AnalyticsVelocityRow, error) {
			return []AnalyticsVelocityRow{}, nil
		},
		GetCycleTimeByPhaseFunc: func(ctx context.Context, sprintID int64) ([]AnalyticsPhaseTimeRow, error) {
			assert.Equal(t, int64(42), sprintID)
			return phaseRows, nil
		},
	}

	svc := NewSprintAnalyticsService(analyticsRepo, sprintRepo)
	result, err := svc.GetSummary(context.Background(), "S024", true)

	require.NoError(t, err)
	require.NotNil(t, result)

	// CycleTimeByPhase must be populated (non-nil)
	require.NotNil(t, result.CycleTimeByPhase, "CycleTimeByPhase must be non-nil when data available")
	require.Len(t, result.CycleTimeByPhase, 2)
	assert.Equal(t, "in_progress", result.CycleTimeByPhase[0].Phase)
	assert.InDelta(t, 3.5, result.CycleTimeByPhase[0].AverageDays, 0.001)

	// Mid-sprint add count = 1 (entity 3)
	require.NotNil(t, result.AddedMidSprintCount)
	assert.Equal(t, 1, *result.AddedMidSprintCount)

	// AvgCompletedSize: 2 completed entities each size 5 → 5.0
	require.NotNil(t, result.AvgCompletedSize)
	assert.InDelta(t, 5.0, *result.AvgCompletedSize, 0.001)

	// Carryover: entity 3 not completed and not removed
	require.NotNil(t, result.CarryoverEntities)
	assert.Len(t, result.CarryoverEntities, 1)
	assert.Equal(t, "task", result.CarryoverEntities[0].EntityType)
}

// --- TC-S-06: detailed=true with GetCycleTimeByPhase returning empty slice → CycleTimeByPhase=nil.
// Counter-factual: buggy impl stores [] instead of nil fails the nil check and JSON null test.
func TestGetSummary_DetailedNoCycleTimeData_CycleTimeNil(t *testing.T) {
	startDate := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	sprintRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return sprintHelper("S024", "completed", startDate, endDate), nil
		},
	}
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetSprintAssignedEntitiesFunc: func(ctx context.Context, sprintID int64) ([]AnalyticsAssignedEntity, error) {
			return []AnalyticsAssignedEntity{}, nil
		},
		GetCompletionEventsFunc: func(ctx context.Context, sprintID int64, start, end time.Time) ([]AnalyticsCompletionEvent, error) {
			return []AnalyticsCompletionEvent{}, nil
		},
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]AnalyticsVelocityRow, error) {
			return []AnalyticsVelocityRow{}, nil
		},
		GetCycleTimeByPhaseFunc: func(ctx context.Context, sprintID int64) ([]AnalyticsPhaseTimeRow, error) {
			// Empty slice (not error) — simulates work_sessions unavailable
			return []AnalyticsPhaseTimeRow{}, nil
		},
	}

	svc := NewSprintAnalyticsService(analyticsRepo, sprintRepo)

	assert.NotPanics(t, func() {
		result, err := svc.GetSummary(context.Background(), "S024", true)

		require.NoError(t, err, "empty cycle-time slice must NOT cause error")
		require.NotNil(t, result)

		// KEY: empty slice from repo → nil in DTO (TC-S-06, AC-S-3)
		assert.Nil(t, result.CycleTimeByPhase,
			"empty slice from GetCycleTimeByPhase must be stored as nil in DTO, not []PhaseTime{}")
	})
}

// --- TC-S-07: detailed=false → all detailed pointer fields are nil.
// Also verifies that GetCycleTimeByPhase is NOT called when detailed=false.
// Counter-factual: buggy impl calling GetCycleTimeByPhase when detailed=false wastes a round-trip.
func TestGetSummary_DetailedFalse_DetailedFieldsNil(t *testing.T) {
	startDate := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	cycleTimeCalled := false
	sprintRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return sprintHelper("S024", "completed", startDate, endDate), nil
		},
	}
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetSprintAssignedEntitiesFunc: func(ctx context.Context, sprintID int64) ([]AnalyticsAssignedEntity, error) {
			return []AnalyticsAssignedEntity{}, nil
		},
		GetCompletionEventsFunc: func(ctx context.Context, sprintID int64, start, end time.Time) ([]AnalyticsCompletionEvent, error) {
			return []AnalyticsCompletionEvent{}, nil
		},
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]AnalyticsVelocityRow, error) {
			return []AnalyticsVelocityRow{}, nil
		},
		GetCycleTimeByPhaseFunc: func(ctx context.Context, sprintID int64) ([]AnalyticsPhaseTimeRow, error) {
			cycleTimeCalled = true
			return nil, nil
		},
	}

	svc := NewSprintAnalyticsService(analyticsRepo, sprintRepo)
	result, err := svc.GetSummary(context.Background(), "S024", false)

	require.NoError(t, err)
	require.NotNil(t, result)

	// All detailed pointer fields must be nil when detailed=false (AC-S-4, TC-S-07)
	assert.Nil(t, result.AddedMidSprintCount)
	assert.Nil(t, result.AddedMidSprintSize)
	assert.Nil(t, result.RemovedMidSprintCount)
	assert.Nil(t, result.RemovedMidSprintSize)
	assert.Nil(t, result.CycleTimeByPhase)
	assert.Nil(t, result.AvgCompletedSize)
	assert.Nil(t, result.SizeBandDistribution)
	assert.Nil(t, result.CarryoverEntities)
	assert.False(t, cycleTimeCalled, "GetCycleTimeByPhase must NOT be called when detailed=false")
}

// --- Unsized entities counting (TC-S-03 extension).
func TestGetSummary_UnsizedCounting(t *testing.T) {
	startDate := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	size5 := 5
	assignedBefore := startDate.Add(-1 * time.Hour)

	// 3 planned: 2 sized, 1 unsized
	entities := []AnalyticsAssignedEntity{
		{EntityType: "task", EntityID: 1, AssignedAt: assignedBefore, Size: &size5},
		{EntityType: "task", EntityID: 2, AssignedAt: assignedBefore, Size: nil}, // unsized
		{EntityType: "task", EntityID: 3, AssignedAt: assignedBefore, Size: &size5},
	}
	// 2 completed: entity 1 (sized) and entity 2 (unsized)
	completionEvents := []AnalyticsCompletionEvent{
		{EntityID: 1, EntityType: "task", NewStatus: "completed", Timestamp: endDate.Add(-1 * time.Hour)},
		{EntityID: 2, EntityType: "task", NewStatus: "completed", Timestamp: endDate.Add(-1 * time.Hour)},
	}

	sprintRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return sprintHelper("S024", "completed", startDate, endDate), nil
		},
	}
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetSprintAssignedEntitiesFunc: func(ctx context.Context, sprintID int64) ([]AnalyticsAssignedEntity, error) {
			return entities, nil
		},
		GetCompletionEventsFunc: func(ctx context.Context, sprintID int64, start, end time.Time) ([]AnalyticsCompletionEvent, error) {
			return completionEvents, nil
		},
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]AnalyticsVelocityRow, error) {
			return []AnalyticsVelocityRow{}, nil
		},
	}

	svc := NewSprintAnalyticsService(analyticsRepo, sprintRepo)
	result, err := svc.GetSummary(context.Background(), "S024", false)

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 1, result.UnsizedPlanned, "1 unsized planned entity")
	assert.Equal(t, 1, result.UnsizedCompleted, "1 unsized completed entity")
	assert.Equal(t, 10, result.PlannedSize, "2 sized planned × 5 = 10")
	assert.Equal(t, 5, result.CompletedSize, "1 sized completed × 5 = 5")
	assert.InDelta(t, 50.0, result.CompletionPctBySize, 0.01, "5/10 = 50%")
}

// --- Sprint not found propagates error.
func TestGetSummary_SprintNotFound(t *testing.T) {
	sprintRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return nil, fmt.Errorf("sprint not found: %s", key)
		},
	}
	analyticsRepo := &MockSprintAnalyticsRepository{}

	svc := NewSprintAnalyticsService(analyticsRepo, sprintRepo)
	result, err := svc.GetSummary(context.Background(), "S999", false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "sprint not found")
}

// --- Size-band distribution populated correctly with detailed=true.
func TestGetSummary_SizeBandDistribution(t *testing.T) {
	startDate := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	size1 := 1 // XS
	size3 := 3 // M
	size5 := 5 // L
	assignedBefore := startDate.Add(-1 * time.Hour)

	entities := []AnalyticsAssignedEntity{
		{EntityType: "task", EntityID: 1, AssignedAt: assignedBefore, Size: &size1}, // XS
		{EntityType: "task", EntityID: 2, AssignedAt: assignedBefore, Size: &size3}, // M
		{EntityType: "task", EntityID: 3, AssignedAt: assignedBefore, Size: &size5}, // L
	}
	completionEvents := []AnalyticsCompletionEvent{
		{EntityID: 1, EntityType: "task", NewStatus: "completed", Timestamp: endDate.Add(-1 * time.Hour)},
		{EntityID: 2, EntityType: "task", NewStatus: "completed", Timestamp: endDate.Add(-1 * time.Hour)},
		{EntityID: 3, EntityType: "task", NewStatus: "completed", Timestamp: endDate.Add(-1 * time.Hour)},
	}

	sprintRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return sprintHelper("S024", "completed", startDate, endDate), nil
		},
	}
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetSprintAssignedEntitiesFunc: func(ctx context.Context, sprintID int64) ([]AnalyticsAssignedEntity, error) {
			return entities, nil
		},
		GetCompletionEventsFunc: func(ctx context.Context, sprintID int64, start, end time.Time) ([]AnalyticsCompletionEvent, error) {
			return completionEvents, nil
		},
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]AnalyticsVelocityRow, error) {
			return []AnalyticsVelocityRow{}, nil
		},
		GetCycleTimeByPhaseFunc: func(ctx context.Context, sprintID int64) ([]AnalyticsPhaseTimeRow, error) {
			return []AnalyticsPhaseTimeRow{}, nil
		},
	}

	svc := NewSprintAnalyticsService(analyticsRepo, sprintRepo)
	result, err := svc.GetSummary(context.Background(), "S024", true)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.SizeBandDistribution)

	// Find XS, M, L bands
	bandMap := map[string]int{}
	for _, b := range result.SizeBandDistribution {
		bandMap[b.Label] = b.Count
	}
	assert.Equal(t, 1, bandMap["XS"], "1 XS entity (size=1)")
	assert.Equal(t, 1, bandMap["M"], "1 M entity (size=3)")
	assert.Equal(t, 1, bandMap["L"], "1 L entity (size=5)")
	assert.Equal(t, 0, bandMap["S"], "no S entity")
}

// --- BUG-001 regression: trailing average must EXCLUDE the current sprint.
//
// When the repo returns 6 rows and the 6th row is the sprint being summarised
// (S024), the trailing average must be computed over only the 5 prior rows.
// Counter-factual: buggy impl passes velocityResult.TrailingAverage (avg of
// all 6 rows) instead of manually averaging the 5 prior rows.
func TestGetSummary_TrailingAvgExcludesCurrentSprint(t *testing.T) {
	startDate := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	size5 := 5
	assignedBefore := startDate.Add(-1 * time.Hour)

	// Sprint S024 completed 40 points (8 tasks × 5).
	entities := make([]AnalyticsAssignedEntity, 8)
	for i := range entities {
		entities[i] = AnalyticsAssignedEntity{
			EntityType: "task",
			EntityID:   int64(i + 1),
			AssignedAt: assignedBefore,
			Size:       &size5,
		}
	}
	completionEvents := make([]AnalyticsCompletionEvent, 8)
	for i := range completionEvents {
		completionEvents[i] = AnalyticsCompletionEvent{
			EntityID:   int64(i + 1),
			EntityType: "task",
			NewStatus:  "completed",
			Timestamp:  endDate.Add(-2 * 24 * time.Hour),
		}
	}

	// The repo returns 6 rows: 5 prior sprints + the current sprint S024.
	// Prior 5: (36+38+40+38+38) = 190 → average = 38.0
	// Including S024 (40): (190+40)/6 = 38.33… → wrong if used
	sixRows := []AnalyticsVelocityRow{
		{SprintKey: "S019", CompletedSize: 36},
		{SprintKey: "S020", CompletedSize: 38},
		{SprintKey: "S021", CompletedSize: 40},
		{SprintKey: "S022", CompletedSize: 38},
		{SprintKey: "S023", CompletedSize: 38},
		{SprintKey: "S024", CompletedSize: 40}, // current sprint — must be excluded
	}

	sprintRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return sprintHelper("S024", "completed", startDate, endDate), nil
		},
	}
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetSprintAssignedEntitiesFunc: func(ctx context.Context, sprintID int64) ([]AnalyticsAssignedEntity, error) {
			return entities, nil
		},
		GetCompletionEventsFunc: func(ctx context.Context, sprintID int64, start, end time.Time) ([]AnalyticsCompletionEvent, error) {
			return completionEvents, nil
		},
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]AnalyticsVelocityRow, error) {
			return sixRows, nil
		},
	}

	svc := NewSprintAnalyticsService(analyticsRepo, sprintRepo)
	result, err := svc.GetSummary(context.Background(), "S024", false)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Trailing average must be 38.0 (5 prior sprints only), NOT 38.33 (all 6).
	assert.InDelta(t, 38.0, result.TrailingAvgVelocity, 0.01,
		"trailing average must exclude the current sprint S024")
	// Velocity delta = 40 - 38 = 2
	assert.InDelta(t, 2.0, result.VelocityDelta, 0.01)
}

// --- BUG-002 regression: SizeBandDistribution is nil (not empty slice) when
// no recognized sizes are present.
//
// Counter-factual: buggy impl returns []SizeBand{} which serialises as JSON
// array [] instead of null.
func TestGetSummary_SizeBandDistribution_NilWhenEmpty(t *testing.T) {
	startDate := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	// Entity with unrecognised size (99 is not in sizeLabelMap).
	size99 := 99
	assignedBefore := startDate.Add(-1 * time.Hour)

	entities := []AnalyticsAssignedEntity{
		{EntityType: "task", EntityID: 1, AssignedAt: assignedBefore, Size: &size99},
	}
	completionEvents := []AnalyticsCompletionEvent{
		{EntityID: 1, EntityType: "task", NewStatus: "completed",
			Timestamp: endDate.Add(-1 * time.Hour)},
	}

	sprintRepo := &MockSprintRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Sprint, error) {
			return sprintHelper("S024", "completed", startDate, endDate), nil
		},
	}
	analyticsRepo := &MockSprintAnalyticsRepository{
		GetSprintAssignedEntitiesFunc: func(ctx context.Context, sprintID int64) ([]AnalyticsAssignedEntity, error) {
			return entities, nil
		},
		GetCompletionEventsFunc: func(ctx context.Context, sprintID int64, start, end time.Time) ([]AnalyticsCompletionEvent, error) {
			return completionEvents, nil
		},
		GetVelocityDataFunc: func(ctx context.Context, limit int) ([]AnalyticsVelocityRow, error) {
			return []AnalyticsVelocityRow{}, nil
		},
		GetCycleTimeByPhaseFunc: func(ctx context.Context, sprintID int64) ([]AnalyticsPhaseTimeRow, error) {
			return []AnalyticsPhaseTimeRow{}, nil
		},
	}

	svc := NewSprintAnalyticsService(analyticsRepo, sprintRepo)
	result, err := svc.GetSummary(context.Background(), "S024", true)

	require.NoError(t, err)
	require.NotNil(t, result)

	// SizeBandDistribution must be nil, not an empty slice, when no bands exist.
	assert.Nil(t, result.SizeBandDistribution,
		"SizeBandDistribution must be nil (not []) when no recognized sizes are present")
}
